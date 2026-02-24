package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PromTimeFormat     = "2006-01-02T15:04:05Z"
	FilenameTimeFormat = "2006-01-02T15-04-05"
	Namespace          = "cattle-monitoring-system"
	PodName            = "mimirtool"
)

type Config struct {
	Kubeconfig    string
	Selector      string
	FromSeconds   int64
	ToSeconds     int64
	OffsetSeconds int64
}

func main() {
	config := parseArgs(os.Args[1:])
	// Ensure kubeconfig is set for subprocesses: prefer parsed arg, else environment
	if config.Kubeconfig != "" {
		os.Setenv("KUBECONFIG", config.Kubeconfig)
	} else if env := os.Getenv("KUBECONFIG"); env != "" {
		config.Kubeconfig = env
	} else {
		log.Fatalf("Kubeconfig not provided: pass a kubeconfig file or set KUBECONFIG env var")
	}

	fmt.Printf("Starting export-metrics script...\n\n")
	fmt.Printf(" Prometheus query: %s\n", config.Selector)
	fmt.Printf(" Query start: %s\n", time.Unix(config.FromSeconds, 0).UTC().Format(PromTimeFormat))
	fmt.Printf(" Query end:   %s\n", time.Unix(config.ToSeconds, 0).UTC().Format(PromTimeFormat))

	if config.OffsetSeconds > 3600 {
		fmt.Printf(" OFFSET: %d\n\n", config.OffsetSeconds)
	} else {
		fmt.Printf("\n")
	}

	// 1. Confirm access
	if err := runCmd("kubectl", "get", "all", "-A"); err != nil {
		log.Fatalf("Failed to access cluster: %v", err)
	}
	fmt.Println(" - Confirm kubeconfig access \033[32mPASS\033[0m")

	// 2. Cleanup old pod
	runCmd("kubectl", "delete", "pod", "-n", Namespace, PodName)

	// 3. Apply mimirtool
	// Prefer mimirtool.yaml located next to the executable; fall back to repo-relative path
	yamlPath := ""
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "mimirtool.yaml")
		if _, err := os.Stat(candidate); err == nil {
			yamlPath = candidate
		}
	}
	if yamlPath == "" {
		// fallback: repo-relative path (when running from repo root)
		candidate := filepath.Join("summarize-tools", "export-metrics", "mimirtool.yaml")
		if _, err := os.Stat(candidate); err == nil {
			yamlPath = candidate
		}
	}
	if yamlPath == "" {
		log.Fatalf("mimirtool.yaml not found next to the executable or in summarize-tools/export-metrics")
	}

	if err := runCmd("kubectl", "apply", "-f", yamlPath); err != nil {
		log.Fatalf("Failed to apply mimirtool.yaml: %v", err)
	}
	time.Sleep(10 * time.Second)

	// 4. Verify pod running
	if err := runCmd("kubectl", "exec", "-n", Namespace, PodName, "--", "ls"); err != nil {
		log.Fatalf("Mimirtool pod not ready: %v", err)
	}
	fmt.Println(" - Confirm mimirtool pod is running \033[32mPASS\033[0m")

	// 5. Setup local directory
	ts1 := time.Now().Format("2006-01-02")
	kubeName := strings.Split(filepath.Base(config.Kubeconfig), ".")[0]
	exportDir := fmt.Sprintf("metrics-%s-%s", kubeName, ts1)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Fatalf("failed to create export directory %q: %v", exportDir, err)
	}
	if err := os.Chdir(exportDir); err != nil {
		log.Fatalf("failed to change to export directory %q: %v", exportDir, err)
	}

	// 6. Loop through time ranges
	currentTo := config.ToSeconds
	for currentTo > config.FromSeconds {
		offset := config.OffsetSeconds
		if (currentTo - config.FromSeconds) < offset {
			offset = currentTo - config.FromSeconds
		}

		rangeStart := currentTo - offset
		fromStr := time.Unix(rangeStart, 0).UTC().Format(PromTimeFormat)
		toStr := time.Unix(currentTo, 0).UTC().Format(PromTimeFormat)
		ts2 := time.Unix(rangeStart, 0).UTC().Format(FilenameTimeFormat)

		fmt.Printf("Exporting range: %s to %s\n", fromStr, toStr)

		// Remote Read
		runCmd("kubectl", "exec", "-n", Namespace, PodName, "--", "mimirtool", "remote-read", "export",
			"--tsdb-path", "./prometheus-export",
			"--address", "http://rancher-monitoring-prometheus:9090",
			"--remote-read-path", "/api/v1/read",
			"--to="+toStr, "--from="+fromStr, "--selector", config.Selector)

		// Tar in pod
		runCmd("kubectl", "exec", "-n", Namespace, PodName, "--", "tar", "zcf", "/tmp/prometheus-export.tar.gz", "./prometheus-export")

		// Copy locally
		localTar := fmt.Sprintf("prometheus-export-%s.tar.gz", ts2)
		runCmd("kubectl", "-n", Namespace, "cp", PodName+":/tmp/prometheus-export.tar.gz", localTar)

		// Cleanup pod files
		runCmd("kubectl", "exec", "-n", Namespace, PodName, "--", "rm", "-rf", "prometheus-export")

		// Extract and merge
		runCmd("tar", "xf", localTar)
		if _, err := os.Stat("prometheus-export"); err == nil {
			if err := os.RemoveAll("prometheus-export/wal"); err != nil {
				log.Printf("failed to remove wal directory %q: %v", "prometheus-export/wal", err)
			}

			files, err := os.ReadDir("prometheus-export")
			if err != nil {
				log.Printf("failed to read directory %q: %v", "prometheus-export", err)
			} else if len(files) == 0 {
				fmt.Println(" - No blocks to copy")
				if err := os.Remove(localTar); err != nil {
					log.Printf("failed to remove local tar file %q: %v", localTar, err)
				}
			} else {
				// Copy blocks to parent (exportDir)
				for _, f := range files {
					runCmd("cp", "-R", filepath.Join("prometheus-export", f.Name()), "./")
				}
			}
			if err := os.RemoveAll("prometheus-export"); err != nil {
				log.Printf("failed to remove directory %q: %v", "prometheus-export", err)
			}
		}

		currentTo -= offset
		time.Sleep(5 * time.Second)
	}

	// 7. Cleanup
	runCmd("kubectl", "delete", "pod", "-n", Namespace, PodName)

	finalPath, _ := os.Getwd()
	fmt.Printf("\n\033[32mMetrics export complete!\033[0m\n")
	fmt.Printf("View metrics locally via Docker:\n\n")
	fmt.Printf("docker run --rm -u %d -ti -p 9090:9090 -v %s:/prometheus rancher/mirrored-prometheus-prometheus:v2.42.0 --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1y --config.file=/dev/null\n\n", os.Getuid(), finalPath)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// Suppress stdout for clean UI unless it's a specific check
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseArgs(args []string) Config {
	// Defaults
	conf := Config{
		Selector:      `{__name__!=""}`,
		OffsetSeconds: 3600,
		ToSeconds:     time.Now().Unix(),
		FromSeconds:   time.Now().Add(-1 * time.Hour).Unix(),
		Kubeconfig:    os.Getenv("KUBECONFIG"),
	}

	kubeRegex := regexp.MustCompile(`.*\.ya?ml`)
	selectorRegex := regexp.MustCompile(`\{.*\}`)
	offsetRegex := regexp.MustCompile(`^[0-9]+$`)
	dateRegex := regexp.MustCompile(`.*T.*Z`)

	var foundDates []string

	for _, arg := range args {
		switch {
		case kubeRegex.MatchString(arg):
			conf.Kubeconfig = arg
			os.Setenv("KUBECONFIG", arg)
		case selectorRegex.MatchString(arg):
			conf.Selector = arg
		case offsetRegex.MatchString(arg):
			if val, err := strconv.ParseInt(arg, 10, 64); err == nil {
				conf.OffsetSeconds = val
			}
		case dateRegex.MatchString(arg):
			foundDates = append(foundDates, arg)
		}
	}

	// Handle dates
	if len(foundDates) >= 1 {
		t, err := time.Parse(PromTimeFormat, foundDates[0])
		if err != nil {
			log.Printf("failed to parse start time %q with format %q: %v", foundDates[0], PromTimeFormat, err)
		} else {
			conf.FromSeconds = t.Unix()
		}
	}
	if len(foundDates) >= 2 {
		t, err := time.Parse(PromTimeFormat, foundDates[1])
		if err != nil {
			log.Printf("failed to parse end time %q with format %q: %v", foundDates[1], PromTimeFormat, err)
		} else {
			conf.ToSeconds = t.Unix()
		}
	}

	// Normalize order
	if conf.FromSeconds > conf.ToSeconds {
		conf.FromSeconds, conf.ToSeconds = conf.ToSeconds, conf.FromSeconds
	}

	// Logic for limiting offset based on Prometheus memory
	if conf.OffsetSeconds > 7200 {
		conf.OffsetSeconds = 7200
		out, err := exec.Command("kubectl", "get", "statefulsets", "-n", Namespace, "prometheus-rancher-monitoring-prometheus", "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.memory}").Output()
		if err == nil {
			memStr := strings.TrimSuffix(string(out), "Mi")
			if mem, err := strconv.Atoi(memStr); err == nil && mem < 3001 {
				conf.OffsetSeconds = 3600
			}
		}
	}

	return conf
}
