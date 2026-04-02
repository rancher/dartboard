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
	"time"

	"github.com/rancher/dartboard/internal/summarize/collectprofiles"
	"github.com/rancher/dartboard/internal/summarize/countresources"
	"github.com/rancher/dartboard/internal/summarize/exportmetrics"
	"github.com/urfave/cli/v2"
)

func handleTimeInputs(cli *cli.Context) (int64, int64, int64, error) {
	// Handle time inputs
	startTimeStr := cli.String("start-time")
	endTimeStr := cli.String("end-time")
	step := cli.Int("step")
	var offsetSeconds int64
	if step == 0 {
		offsetSeconds = 3600 // 1 hour default
	} else if step > 7200 {
		offsetSeconds = 7200 // 2 hours (requires increased memory on mimirtool)
	} else {
		offsetSeconds = int64(step)
	}

	var from, to time.Time
	var err error

	if startTimeStr != "" {
		from, err = time.Parse(exportmetrics.PromTimeFormat, startTimeStr)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid start time format: %w", err)
		}
	}

	if endTimeStr != "" {
		to, err = time.Parse(exportmetrics.PromTimeFormat, endTimeStr)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid end time format: %w", err)
		}
	}

	var fromSeconds, toSeconds int64
	// Set inputs based on provided arguments
	if startTimeStr != "" && endTimeStr != "" {
		// Both To and From provided
		fromSeconds = from.Unix()
		toSeconds = to.Unix()
	} else if startTimeStr != "" && endTimeStr == "" {
		// Only From provided, To is set to current time
		fromSeconds = from.Unix()
		toSeconds = time.Now().Unix()
	} else if startTimeStr == "" && endTimeStr != "" {
		// Only To provided, From is set to 1 hour before current time
		toSeconds = to.Unix()
		fromSeconds = to.Add(-1 * time.Hour).Unix()
	} else {
		// Last hour
		toSeconds = time.Now().Unix()
		fromSeconds = time.Now().Add(-1 * time.Hour).Unix()
	}
	return fromSeconds, toSeconds, offsetSeconds, nil
}

func Summarize(cli *cli.Context) error {
	tf, _, err := prepare(cli)
	if err != nil {
		return err
	}

	clusters, err := tf.OutputClusters(cli.Context)
	if err != nil {
		return err
	}

	// Select upstream cluster configuration
	upstream := clusters["upstream"]

	// Read flags; allow shorthand via Aliases set in main.go
	metrics := cli.Bool("metrics")
	counts := cli.Bool("counts")
	profiles := cli.Bool("profiles")
	allFlag := cli.Bool("all")

	// If --all provided, enable everything
	if allFlag {
		metrics, counts, profiles = true, true, true
	}

	// If user didn't specify any of these flags, default to all
	if !(cli.IsSet("metrics") || cli.IsSet("counts") || cli.IsSet("profiles") || cli.IsSet("all")) {
		metrics, counts, profiles = true, true, true
	}

	// Create top-level summary directory for this run so all tool outputs aggregate there
	summaryDir := fmt.Sprintf("summarize-results-%s", time.Now().Format("2006-01-02"))
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create summary directory %s: %w", summaryDir, err)
	}

	// Change working directory to summaryDir so tools output files there
	originalWd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %w", err)
	}
	if err := os.Chdir(summaryDir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", summaryDir, err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to restore working directory: %v\n", err)
		}
	}()

	ctx := cli.Context

	// Run Tools
	if profiles {
		fmt.Println(">>> Running collect-profile...")
		// Defaults match original flags
		cfg := collectprofile.Config{
			App:      "rancher",
			Profiles: "goroutine,heap,profile",
			Duration: 30,
			LogLevel: "debug",
		}
		if err := collectprofile.Run(ctx, cfg); err != nil {
			fmt.Printf("Error running collect-profile: %v\n", err)
		}
	}

	if counts {
		fmt.Println(">>> Running resource-counts...")
		cfg := countresources.Config{
			Kubeconfig: upstream.Kubeconfig,
		}
		if err := countresources.Run(ctx, cfg); err != nil {
			fmt.Printf("Error running resource-counts: %v\n", err)
		}
	}

	if metrics {
		fmt.Println(">>> Running export-metrics...")
		cfg := exportmetrics.Config{
			Kubeconfig: upstream.Kubeconfig,
			Selector:   cli.String("query"),
		}

		from, to, offset, err := handleTimeInputs(cli)
		if err != nil {
			return err
		}
		cfg.FromSeconds = from
		cfg.ToSeconds = to
		cfg.OffsetSeconds = offset

		if err := exportmetrics.Run(ctx, cfg); err != nil {
			fmt.Printf("Error running export-metrics: %v\n", err)
		}
	}

	return nil
}