package countresources

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Kubeconfig string
}

func Run(ctx context.Context, cfg Config) error {
	// Set the KUBECONFIG environment variable for the current process
	if cfg.Kubeconfig != "" {
		os.Setenv("KUBECONFIG", cfg.Kubeconfig)
	}

	// Prepare Directory Paths
	now := time.Now()

	// Parent directory: counts-MM-DD
	parentDirDate := now.Format("01-02")
	parentDir := fmt.Sprintf("counts-%s", parentDirDate)

	// Output directory: cr-outputs-MM-DD-HH-MM
	subDirDate := now.Format("01-02-15-04")
	outputDir := filepath.Join(parentDir, fmt.Sprintf("cr-outputs-%s", subDirDate))

	// Create directories
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Prepare Output Filename
	baseName := filepath.Base(cfg.Kubeconfig)
	if baseName == "." || baseName == "" {
		baseName = "kubeconfig"
	}
	// Remove the file extension (e.g., .yaml)
	cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Timestamp: MM-DD-HH-MM-SS
	fileDate := now.Format("01-02-15-04-05")
	outputFilename := fmt.Sprintf("%s-%s.txt", cleanName, fileDate)
	outputPath := filepath.Join(outputDir, outputFilename)

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	fmt.Printf("Using Kubeconfig: %s\n", cfg.Kubeconfig)
	fmt.Printf("Writing report to: %s\n", outputPath)

	// Get List of Resources
	cmd := exec.CommandContext(ctx, "kubectl", "api-resources", "--no-headers", "-o", "wide")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get api-resources: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		resource := fields[0]

		// Count Resources
		countCmd := exec.CommandContext(ctx, "kubectl", "get", resource, "-A", "--no-headers", "--ignore-not-found")
		countOutput, err := countCmd.Output()

		var count int
		if err == nil {
			// Count non-empty lines
			lines := bytes.Split(countOutput, []byte{'\n'})
			for _, l := range lines {
				if len(bytes.TrimSpace(l)) > 0 {
					count++
				}
			}
		}

		// Write to File
		lineOutput := fmt.Sprintf(" %s : %d\n", resource, count)
		if _, err := outFile.WriteString(lineOutput); err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading api-resources output: %w", err)
	}

	return nil
}