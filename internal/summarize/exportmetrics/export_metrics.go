package exportmetrics

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func Run(ctx context.Context, cfg Config) error {
	// Ensure kubeconfig is set for subprocesses
	if cfg.Kubeconfig != "" {
		os.Setenv("KUBECONFIG", cfg.Kubeconfig)
	}

	// Set defaults if not provided
	if cfg.Selector == "" {
		cfg.Selector = `{__name__!=""}`
	}
	if cfg.OffsetSeconds == 0 {
		cfg.OffsetSeconds = 3600
	}
	if cfg.ToSeconds == 0 {
		cfg.ToSeconds = time.Now().Unix()
	}
	if cfg.FromSeconds == 0 {
		cfg.FromSeconds = time.Now().Add(-1 * time.Hour).Unix()
	}

	// Logic for limiting offset based on Prometheus memory
	if cfg.OffsetSeconds > 7200 {
		cfg.OffsetSeconds = 7200
		out, err := exec.CommandContext(ctx, "kubectl", "get", "statefulsets", "-n", Namespace, "prometheus-rancher-monitoring-prometheus", "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.memory}").Output()
		if err == nil {
			memStr := strings.TrimSuffix(string(out), "Mi")
			if mem, err := strconv.Atoi(memStr); err == nil && mem < 3001 {
				cfg.OffsetSeconds = 3600
			}
		}
	}

	fmt.Printf("Starting export-metrics...\n\n")
	fmt.Printf(" Prometheus query: %s\n", cfg.Selector)
	fmt.Printf(" Query start: %s\n", time.Unix(cfg.FromSeconds, 0).UTC().Format(PromTimeFormat))
	fmt.Printf(" Query end:   %s\n", time.Unix(cfg.ToSeconds, 0).UTC().Format(PromTimeFormat))

	if cfg.OffsetSeconds > 3600 {
		fmt.Printf(" OFFSET: %d\n\n", cfg.OffsetSeconds)
	} else {
		fmt.Printf("\n")
	}

	// 1. Confirm access
	if err := runCmd(ctx, "kubectl", "get", "all", "-A"); err != nil {
		return fmt.Errorf("failed to access cluster: %w", err)
	}
	fmt.Println(" - Confirm kubeconfig access \033[32mPASS\033[0m")

	// 2. Cleanup old pod
	runCmd(ctx, "kubectl", "delete", "pod", "-n", Namespace, PodName)

	// 3. Apply mimirtool
	yamlPath := "../summarize-tools/export-metrics/mimirtool.yaml"

	if err := runCmd(ctx, "kubectl", "apply", "-f", yamlPath); err != nil {
		return fmt.Errorf("failed to apply mimirtool.yaml: %w", err)
	}
	
	// Wait for pod to be ready
	fmt.Println("Waiting for mimirtool pod to be ready...")
	time.Sleep(10 * time.Second)
	if err := runCmd(ctx, "kubectl", "wait", "--for=condition=Ready", "pod", "-n", Namespace, PodName, "--timeout=60s"); err != nil {
		return fmt.Errorf("mimirtool pod not ready: %w", err)
	}
	fmt.Println(" - Confirm mimirtool pod is running \033[32mPASS\033[0m")

	// 4. Setup local directory
	ts1 := time.Now().Format("2006-01-02")
	kubeName := "cluster"
	if cfg.Kubeconfig != "" {
		kubeName = strings.Split(filepath.Base(cfg.Kubeconfig), ".")[0]
	}
	exportDir := fmt.Sprintf("metrics-%s-%s", kubeName, ts1)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory %q: %w", exportDir, err)
	}
	
	// 5. Loop through time ranges
	currentTo := cfg.ToSeconds
	for currentTo > cfg.FromSeconds {
		offset := cfg.OffsetSeconds
		if (currentTo - cfg.FromSeconds) < offset {
			offset = currentTo - cfg.FromSeconds
		}

		rangeStart := currentTo - offset
		fromStr := time.Unix(rangeStart, 0).UTC().Format(PromTimeFormat)
		toStr := time.Unix(currentTo, 0).UTC().Format(PromTimeFormat)
		ts2 := time.Unix(rangeStart, 0).UTC().Format(FilenameTimeFormat)

		fmt.Printf("Exporting range: %s to %s\n", fromStr, toStr)

		var err error
		for attempt := 1; attempt <= 3; attempt++ {
			if attempt > 1 {
				fmt.Printf(" - Retrying... (Attempt %d/3)\n", attempt)
				// Cleanup before retry
				runCmd(ctx, "kubectl", "exec", "-n", Namespace, PodName, "--", "rm", "-rf", "prometheus-export")
				time.Sleep(2 * time.Second)
			}

			// Remote Read
			err = runCmd(ctx, "kubectl", "exec", "-n", Namespace, PodName, "--", "mimirtool", "remote-read", "export",
				"--tsdb-path", "./prometheus-export",
				"--address", "http://rancher-monitoring-prometheus:9090",
				"--remote-read-path", "/api/v1/read",
				"--to="+toStr, "--from="+fromStr, "--selector", cfg.Selector)
			if err != nil {
				fmt.Printf(" - Remote read failed: %v\n", err)
				continue
			}

			// Tar in pod
			err = runCmd(ctx, "kubectl", "exec", "-n", Namespace, PodName, "--", "tar", "zcf", "/tmp/prometheus-export.tar.gz", "./prometheus-export")
			if err != nil {
				fmt.Printf(" - Tar failed: %v\n", err)
				continue
			}

			// Copy locally
			localTar := filepath.Join(exportDir, fmt.Sprintf("prometheus-export-%s.tar.gz", ts2))
			err = runCmd(ctx, "kubectl", "-n", Namespace, "cp", PodName+":/tmp/prometheus-export.tar.gz", localTar)
			if err != nil {
				fmt.Printf(" - Copy failed: %v\n", err)
				continue
			}

			// Success
			err = nil
			break
		}

		if err != nil {
			fmt.Printf("Failed to export range %s to %s after 3 attempts.\n", fromStr, toStr)
		}

		// Cleanup pod files
		runCmd(ctx, "kubectl", "exec", "-n", Namespace, PodName, "--", "rm", "-rf", "prometheus-export")

		currentTo -= offset
		// Small pause between exports to reduce load and avoid potential issues with rapid queries
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	// 6. Cleanup
	runCmd(ctx, "kubectl", "delete", "pod", "-n", Namespace, PodName)

	finalPath, _ := filepath.Abs(exportDir)
	fmt.Printf("\n\033[32mMetrics export complete!\033[0m\n")
	fmt.Printf("Metrics saved to: %s\n", finalPath)
	
	return nil
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}