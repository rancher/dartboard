/*
Copyright © 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package subcommands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

func Summarize(cli *cli.Context) error {
	tf, _, err := prepare(cli)
	if err != nil {
		return err
	}

	clusters, err := tf.OutputClusters(cli.Context)
	if err != nil {
		return err
	}

	// Refresh k6 files
	upstream := clusters["upstream"]

	//TODO: Logic to target defined cluster from darts/aws.yaml

	// Run individual build scripts and ensure each resulting binary is executable
	type buildItem struct{
		script string
		binary string
	}

	builds := []buildItem{
		{"./summarize-tools/collect-profile/build_collect_profile.sh", "./summarize-tools/collect-profile/collect-profile"},
		{"./summarize-tools/resource-counts/build_cr.sh", "./summarize-tools/resource-counts/cr"},
		{"./summarize-tools/export-metrics/build_export_metrics.sh", "./summarize-tools/export-metrics/export-metrics"},
	}

	runScript := func(script string) error {
		cmd := exec.Command("bash", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	for _, b := range builds {
		if err := runScript(b.script); err != nil {
			return fmt.Errorf("failed to run build script %s: %w", b.script, err)
		}

		if err := os.Chmod(b.binary, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission on %s: %w", b.binary, err)
		}

		// Ensure each built binary is removed after use. Capture path for defer.
		bin := b.binary
		defer func(p string) {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: failed to delete %s: %v\n", p, err)
			}
		}(bin)
	}

	// Create top-level summary directory for this run so all tool outputs aggregate there
	summaryDir := fmt.Sprintf("summarize-results-%s", time.Now().Format("2006-01-02"))
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create summary directory %s: %w", summaryDir, err)
	}

	// Run built tools: collect-profile, cr, then export-metrics — each runs with CWD set to summaryDir
	runBins := []string{
		"./summarize-tools/collect-profile/collect-profile",
		"./summarize-tools/resource-counts/cr",
		"./summarize-tools/export-metrics/export-metrics",
	}

	// Resolve run binary paths relative to the repo root so setting `cmd.Dir` doesn't break exec path lookup
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %w", err)
	}

	for _, bin := range runBins {
		binPath := bin
		if !filepath.IsAbs(binPath) {
			binPath = filepath.Join(repoRoot, binPath)
		}

		cmd := exec.Command(binPath, upstream.Kubeconfig)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = summaryDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run %s: %w", binPath, err)
		}
	}

	return nil
}
