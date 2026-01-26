package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// 1. Determine Kubeconfig Path
	// Priority: 1. Command Line Arg, 2. Environment Variable
	kubeconfigPath := os.Getenv("KUBECONFIG")

	if len(os.Args) > 1 {
		kubeconfigPath = os.Args[1]
	}

	if kubeconfigPath == "" {
		log.Fatal("Error: Kubeconfig not found. Please set KUBECONFIG env var or pass as argument.")
	}

	// Set the KUBECONFIG environment variable for the current process
	// This ensures the subsequent 'kubectl' commands use the correct config
	os.Setenv("KUBECONFIG", kubeconfigPath)

	// 2. Prepare Directory Paths
	now := time.Now()

	// Parent directory: counts-MM-DD
	parentDirDate := now.Format("01-02")
	parentDir := fmt.Sprintf("counts-%s", parentDirDate)

	// Output directory: cr-outputs-MM-DD-HH-MM
	subDirDate := now.Format("01-02-15-04")
	outputDir := filepath.Join(parentDir, fmt.Sprintf("cr-outputs-%s", subDirDate))

	// Create directories
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// 3. Prepare Output Filename
	// Extract just the filename, removing the directory path
	baseName := filepath.Base(kubeconfigPath)
	// Remove the file extension (e.g., .yaml)
	cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Timestamp: MM-DD-HH-MM-SS
	fileDate := now.Format("01-02-15-04-05")
	outputFilename := fmt.Sprintf("%s-%s.txt", cleanName, fileDate)
	outputPath := filepath.Join(outputDir, outputFilename)

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	fmt.Printf("Using Kubeconfig: %s\n", kubeconfigPath)
	fmt.Printf("Writing report to: %s\n", outputPath)

	// 4. Get List of Resources
	// Equivalent to: kubectl api-resources -o wide | grep -v "NAME" | awk '{ print $1 }'
	cmd := exec.Command("kubectl", "api-resources", "--no-headers", "-o", "wide")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to get api-resources: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		resource := fields[0]

		// 5. Count Resources
		// Equivalent to: kubectl get "$resource" -A | grep -v ... | wc -l
		countCmd := exec.Command("kubectl", "get", resource, "-A", "--no-headers", "--ignore-not-found")
		countOutput, err := countCmd.Output()

		var count int
		if err != nil {
			// If error occurs (e.g., permission denied), assume 0
			count = 0
		} else {
			// Count non-empty lines
			lines := bytes.Split(countOutput, []byte{'\n'})
			for _, l := range lines {
				if len(bytes.TrimSpace(l)) > 0 {
					count++
				}
			}
		}

		// 6. Write to File
		// Format: " resource : count"
		lineOutput := fmt.Sprintf(" %s : %d\n", resource, count)
		if _, err := outFile.WriteString(lineOutput); err != nil {
			log.Printf("Error writing to file: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading api-resources output: %v", err)
	}
}