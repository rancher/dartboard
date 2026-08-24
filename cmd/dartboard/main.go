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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rancher/dartboard/cmd/dartboard/subcommands"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:     "setup and test Rancher (at scale if needed)",
		Copyright: "(c) 2024 SUSE LLC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    subcommands.ArgDart,
				Aliases: []string{"d"},
				Value:   filepath.Join("darts", "k3d.yaml"),
				Usage:   "dart to use",
				EnvVars: []string{"DART"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "apply",
				Usage:       "Runs `tofu apply`",
				Description: "runs `tofu apply` to prepare infrastructure and Kubernetes clusters for tests",
				Action:      subcommands.Apply,
			},
			{
				Name:        "deploy",
				Usage:       "Deploys Rancher and other charts on top of clusters",
				Description: "prepares the test environment installing all required charts",
				Action:      subcommands.Deploy,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        subcommands.ArgSkipApply,
						Value:       false,
						Usage:       "skip 'tofu apply', assume apply was already called",
						DefaultText: "false",
					},
					&cli.BoolFlag{
						Name:        subcommands.ArgSkipCharts,
						Value:       false,
						Usage:       "skip 'helm install' for all charts, assume charts have already been installed for upstream and tester clusters",
						DefaultText: "false",
					},
					&cli.BoolFlag{
						Name:        subcommands.ArgSkipRefresh,
						Value:       false,
						Usage:       "skip refresh phase for tofu resources, assume resources are refreshed and up-to-date",
						DefaultText: "false",
					},
				},
			},
			{
				Name:        "load",
				Usage:       "Creates K8s resources on upstream and downstream clusters",
				Description: "Loads ConfigMaps and Secrets on all the deployed K8s cluster; Roles, Users and Projects on the Rancher cluster",
				Action:      subcommands.Load,
			},
			{
				Name:        "get-access",
				Usage:       "Retrieves information to access the deployed clusters",
				Description: "print out links and access information for the deployed clusters",
				Action:      subcommands.GetAccess,
			},
			{
				Name:        "destroy",
				Usage:       "Tears down the test environment (all the clusters)",
				Description: "runs `tofu destroy` to destroy all the provisioned clusters",
				Action:      subcommands.Destroy,
			},
			{
				Name:        "reapply",
				Usage:       "Tears down the test environment (all the clusters) and re-runs `tofu apply`",
				Description: "runs `tofu destroy` and then `tofu apply`",
				Action:      subcommands.Reapply,
			},
			{
				Name:        "redeploy",
				Usage:       "Tears down the test environment (all the clusters) and redeploys them from scratch",
				Description: "runs `tofu destroy` and then deploys all the provisioned clusters",
				Action:      subcommands.Redeploy,
			},
			{
				Name:        "collect-metrics",
				Usage:       "Collects scaling-relevant metrics (CPU/mem/disk/net) from upstream Rancher's Prometheus",
				Description: "port-forwards to the upstream cluster's rancher-monitoring Prometheus, runs a curated PromQL catalog over a time window, and exports CSV per series + summary.json",
				Action:      subcommands.CollectMetrics,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  subcommands.ArgStart,
						Usage: "RFC3339 start of metrics window (default: end - last)",
					},
					&cli.StringFlag{
						Name:  subcommands.ArgEnd,
						Usage: "RFC3339 end of metrics window (default: now)",
					},
					&cli.StringFlag{
						Name:  subcommands.ArgLast,
						Value: "1h",
						Usage: "Duration to look back from --end when --start is not given",
					},
					&cli.StringFlag{
						Name:  subcommands.ArgStep,
						Value: "30s",
						Usage: "Sample step for query_range",
					},
					&cli.StringFlag{
						Name:  subcommands.ArgOutput,
						Usage: "Output directory (default: ./metrics-{workspace}-{timestamp}/)",
					},
				},
			},
		},
	}

	subcmd := subcommandFromArgs(os.Args)
	start := time.Now()
	err := app.Run(os.Args)
	elapsed := time.Since(start).Round(time.Second)

	prefix := "dartboard"
	if subcmd != "" {
		prefix = "dartboard " + subcmd
	}

	if err != nil {
		fmt.Printf("%s exited with error after %s: %v\n", prefix, elapsed, err)
		os.Exit(1)
	}
	fmt.Printf("%s exited successfully (took %s)\n", prefix, elapsed)
}

// subcommandFromArgs returns the first non-flag token in args[1:], skipping the
// global -d/--dart flag and its value. Returns "" when no subcommand is present.
func subcommandFromArgs(args []string) string {
	i := 1
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-d" || a == "--dart":
			i += 2
		case strings.HasPrefix(a, "--dart=") || strings.HasPrefix(a, "-d="):
			i++
		case strings.HasPrefix(a, "-"):
			i++
		default:
			return a
		}
	}
	return ""
}
