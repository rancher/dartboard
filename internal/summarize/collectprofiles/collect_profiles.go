package collectprofile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	App          string
	Profiles     string
	Duration     int
	LogLevel     string
	Namespace    string
	Container    string
	Prefix       string
	BlobURL      string
	BlobToken    string
	MainFilename string
}

func Run(ctx context.Context, cfg Config) error {
	// Validate App choice
	validApps := map[string]bool{
		"rancher":              true,
		"cattle-cluster-agent": true,
		"fleet-controller":     true,
		"fleet-agent":          true,
	}
	if !validApps[cfg.App] {
		return fmt.Errorf("invalid app: %s. Supported: rancher, cattle-cluster-agent, fleet-controller, fleet-agent", cfg.App)
	}

	// Setup defaults
	if cfg.Prefix == "" {
		cfg.Prefix = "rancher"
	}
	if cfg.MainFilename == "" {
		cfg.MainFilename = fmt.Sprintf("profiles-%s.tar", time.Now().Format("2006-01-02_15_04"))
	}
	if cfg.BlobURL == "" {
		cfg.BlobURL = os.Getenv("BLOB_URL")
	}
	if cfg.BlobToken == "" {
		cfg.BlobToken = os.Getenv("BLOB_TOKEN")
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "debug"
	}

	// Set timezone to UTC
	os.Setenv("TZ", "UTC")

	var portForwardCmd *exec.Cmd
	var err error

	// Ensure cleanup runs on exit
	defer func() {
		cleanup(cfg, portForwardCmd)
	}()

	// Prepare Environment
	switch cfg.App {
	case "rancher":
		cfg.Container = "rancher"
		cfg.Namespace = "cattle-system"
		setRancherLogLevel(cfg, cfg.LogLevel)
	case "cattle-cluster-agent":
		cfg.Container = "cluster-register"
		cfg.Namespace = "cattle-system"
	case "fleet-controller":
		cfg.Container = "fleet-controller"
		cfg.Namespace = "cattle-fleet-system"
		portForwardCmd, err = startPortForward(cfg, 60601, 6060)
		if err != nil {
			return err
		}
	case "fleet-agent":
		cfg.Container = "fleet-agent"
		// Check for local system namespace first
		if err := exec.Command("kubectl", "get", "namespace", "cattle-fleet-local-system").Run(); err == nil {
			cfg.Namespace = "cattle-fleet-local-system"
		} else {
			cfg.Namespace = "cattle-fleet-system"
		}
		portForwardCmd, err = startPortForward(cfg, 60601, 6060)
		if err != nil {
			return err
		}
	}

	return collect(cfg)
}

func collect(cfg Config) error {
	tmpDir, err := os.MkdirTemp("", "rancher-profile-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	techo("Created " + tmpDir)
	defer func() {
		techo("Removing " + tmpDir)
		os.RemoveAll(tmpDir)
	}()

	// Timestamps
	appendToFile(filepath.Join(tmpDir, "timestamps.txt"), "Start: "+time.Now().Format(time.RFC3339)+"\n")

	// Global Cluster Stats
	shellExecToFile(filepath.Join(tmpDir, "top-pods.txt"), "kubectl", "top", "pods", "-A")
	shellExecToFile(filepath.Join(tmpDir, "top-nodes.txt"), "kubectl", "top", "nodes")

	// Collect App Specific Data
	if cfg.App == "rancher" || cfg.App == "cattle-cluster-agent" {
		collectRancherLogic(cfg, tmpDir)
	} else {
		collectFleetLogic(cfg, tmpDir)
	}

	// Final Global Stats
	techo("Getting leases")
	shellExecToFile(filepath.Join(tmpDir, "leases.txt"), "kubectl", "get", "leases", "-n", "kube-system")

	techo("Getting pod details")
	shellExecToFile(filepath.Join(tmpDir, "pods-wide.txt"), "kubectl", "get", "pods", "-A", "-o", "wide")

	appendToFile(filepath.Join(tmpDir, "timestamps.txt"), "End:   "+time.Now().Format(time.RFC3339)+"\n")

	// Create Tarball
	tarName := fmt.Sprintf("%s-profiles-%s.tar.xz", cfg.Prefix, time.Now().Format("2006-01-02_15_04"))
	tarPath := filepath.Join(os.TempDir(), tarName)
	techo("Creating tarball " + tarName)

	// Using exec for tar to handle XZ compression easily without external go libs
	cmd := exec.Command("tar", "cfJ", tarPath, "--directory", tmpDir, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		techo("Error creating tarball: " + string(out))
	} else {
		handleUploadOrAppend(cfg, tarPath, tarName)
	}

	return nil
}

func collectRancherLogic(cfg Config, tmpDir string) {
	pods := getPodNames(cfg)
	profiles := strings.Split(cfg.Profiles, ",")

	for _, pod := range pods {
		if pod == "" {
			continue
		}
		// Profiles
		for _, profile := range profiles {
			profile = strings.TrimSpace(profile)
			techo(fmt.Sprintf("Getting %s profile for %s", profile, pod))
			url := fmt.Sprintf("http://localhost:6060/debug/pprof/%s", profile)
			if profile == "profile" {
				url += fmt.Sprintf("?seconds=%d", cfg.Duration)
			}

			// For Rancher/Agent we curl FROM INSIDE the pod
			outFile := filepath.Join(tmpDir, fmt.Sprintf("%s-%s-%s", pod, profile, time.Now().Format("2006-01-02T15_04_05")))
			execKubectlCurl(cfg, pod, url, outFile)
		}

		collectCommonLogsAndEvents(cfg, tmpDir, pod)

		// Specific Rancher items
		if cfg.App == "rancher" {
			techo("Getting rancher-audit-logs for " + pod)
			shellExecToFile(filepath.Join(tmpDir, pod+"-audit.log"), "kubectl", "logs", "--since=5m", "-n", cfg.Namespace, pod, "-c", "rancher-audit-log")

			techo("Getting metrics for Rancher")
			// Complex bash command inside exec needs careful wrapping
			metricsCmd := `curl -s -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" -k https://127.0.0.1/metrics`
			outFile := filepath.Join(tmpDir, pod+"-metrics.txt")

			cmd := exec.Command("kubectl", "exec", "-n", cfg.Namespace, pod, "-c", cfg.Container, "--", "bash", "-c", metricsCmd)
			output, _ := cmd.CombinedOutput()
			if err := os.WriteFile(outFile, output, 0644); err != nil {
				techo(fmt.Sprintf("Failed to write metrics to %s: %v", outFile, err))
			}
		}
	}
}

func collectFleetLogic(cfg Config, tmpDir string) {
	pods := getPodNames(cfg)
	if len(pods) == 0 {
		techo("No pods found for " + cfg.App)
		return
	}
	// Fleet usually targets one leader, we take the first one found
	pod := pods[0]
	profiles := strings.Split(cfg.Profiles, ",")

	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		techo(fmt.Sprintf("Getting %s profile for %s", profile, pod))

		url := fmt.Sprintf("http://localhost:60601/debug/pprof/%s", profile)
		if profile == "profile" {
			url += fmt.Sprintf("?seconds=%d", cfg.Duration)
		}

		outFile := filepath.Join(tmpDir, fmt.Sprintf("%s-%s-%s", pod, profile, time.Now().Format("2006-01-02T15_04_05")))
		// For Fleet we curl LOCALHOST via port-forward
		execLocalCurl(url, outFile)
	}

	collectCommonLogsAndEvents(cfg, tmpDir, pod)
}

func collectCommonLogsAndEvents(cfg Config, tmpDir string, pod string) {
	techo("Getting logs for " + pod)
	shellExecToFile(filepath.Join(tmpDir, pod+".log"), "kubectl", "logs", "--since=5m", "-n", cfg.Namespace, pod, "-c", cfg.Container)

	techo("Getting previous logs for " + pod)
	shellExecToFile(filepath.Join(tmpDir, pod+"-previous.log"), "kubectl", "logs", "-n", cfg.Namespace, pod, "-c", cfg.Container, "--previous=true")

	techo("Getting events for " + pod)
	shellExecToFile(filepath.Join(tmpDir, pod+"-events.txt"), "kubectl", "get", "event", "--namespace", cfg.Namespace, "--field-selector", "involvedObject.name="+pod)

	techo("Getting describe for " + pod)
	shellExecToFile(filepath.Join(tmpDir, pod+"-describe.txt"), "kubectl", "describe", "pod", pod, "-n", cfg.Namespace)
}

// Helpers

func getPodNames(cfg Config) []string {
	out, err := exec.Command("kubectl", "-n", cfg.Namespace, "get", "pods", "-l", "app="+cfg.App, "--no-headers", "-o", "custom-columns=name:.metadata.name").Output()
	if err != nil {
		techo("Error getting pods: " + err.Error())
		return []string{}
	}
	lines := strings.Split(string(out), "\n")
	var pods []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			pods = append(pods, strings.TrimSpace(l))
		}
	}
	return pods
}

func execKubectlCurl(cfg Config, pod, url, outFile string) {
	// kubectl exec -n NS pod -c container -- curl -s URL
	cmd := exec.Command("kubectl", "exec", "-n", cfg.Namespace, pod, "-c", cfg.Container, "--", "curl", "-s", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		techo("Error curling " + url + ": " + err.Error())
	}
	if err := os.WriteFile(outFile, out, 0644); err != nil {
		techo("Error writing to " + outFile + ": " + err.Error())
	}
}

func execLocalCurl(url, outFile string) {
	resp, err := http.Get(url)
	if err != nil {
		techo("Error fetching " + url + ": " + err.Error())
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(outFile)
	if err != nil {
		techo("Error creating " + outFile + ": " + err.Error())
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		techo("Error writing response body to " + outFile + ": " + err.Error())
	}
}

func shellExecToFile(filename string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	out, _ := cmd.CombinedOutput() // Ignore error, just write what we got
	if err := os.WriteFile(filename, out, 0644); err != nil {
		techo("Error writing to " + filename + ": " + err.Error())
	}
}

func appendToFile(filename string, text string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(text)
}

func startPortForward(cfg Config, localPort, remotePort int) (*exec.Cmd, error) {
	pods := getPodNames(cfg)
	if len(pods) == 0 {
		techo("No pods found to port-forward")
		return nil, fmt.Errorf("no pods found")
	}
	pod := pods[0]

	cmdStr := fmt.Sprintf("%d:%d", localPort, remotePort)
	cmd := exec.Command("kubectl", "port-forward", "-n", cfg.Namespace, pod, cmdStr)

	err := cmd.Start()
	if err != nil {
		techo("Failed to start port-forward: " + err.Error())
		return nil, err
	}

	// Give it a moment to establish
	time.Sleep(2 * time.Second)
	techo(fmt.Sprintf("Port forward started for %s %s", pod, cmdStr))
	return cmd, nil
}

func setRancherLogLevel(cfg Config, level string) {
	pods := getPodNames(cfg)
	for _, pod := range pods {
		techo(fmt.Sprintf("Setting %s logging to %s", pod, level))
		exec.Command("kubectl", "--namespace", "cattle-system", "exec", pod, "-c", "rancher", "--", "loglevel", "--set", level).Run()
	}
}

func handleUploadOrAppend(cfg Config, srcPath, srcName string) {
	if cfg.BlobURL != "" {
		techo("Uploading " + srcName)
		// Use curl for upload to avoid complex Go http client setup for SAS tokens in a single file
		fullURL := fmt.Sprintf("%s/%s?%s", cfg.BlobURL, srcName, cfg.BlobToken)
		cmd := exec.Command("curl", "-H", "x-ms-blob-type: BlockBlob", "--upload-file", srcPath, fullURL)
		if out, err := cmd.CombinedOutput(); err != nil {
			techo("Upload failed: " + string(out))
		}
	} else {
		// Note: 'tar rf' appends. 'tar' needs to be available.
		techo("Appending to " + cfg.MainFilename)
		exec.Command("tar", "rf", cfg.MainFilename, srcPath).Run()
		if err := os.Remove(srcPath); err != nil {
			techo("Failed to remove " + srcPath + ": " + err.Error())
		}
	}
}

func cleanup(cfg Config, portForwardCmd *exec.Cmd) {
	if cfg.App == "rancher" {
		techo("Resetting Rancher log level to info")
		setRancherLogLevel(cfg, "info")
	} else if (cfg.App == "fleet-controller" || cfg.App == "fleet-agent") && portForwardCmd != nil {
		if err := portForwardCmd.Process.Kill(); err != nil {
			techo("Error killing port-forward: " + err.Error())
		}
		if err := portForwardCmd.Wait(); err != nil {
			techo("Error waiting for port-forward to exit: " + err.Error())
		}
		techo("Killing port-forward")
	}
}

func techo(msg string) {
	fmt.Printf("%s: %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}