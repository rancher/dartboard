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
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/rancher/dartboard/cmd/dartboard/subcommands"
	cli "github.com/urfave/cli/v3"
)

func appCommands() []*cli.Command {
	return []*cli.Command{
		applyCommand(),
		deployCommand(),
		loadCommand(),
		getAccessCommand(),
		destroyCommand(),
		reapplyCommand(),
		redeployCommand(),
		collectMetricsCommand(),
		summarizeCommand(),
		collectProfilesCommand(),
		collectLogsCommand(),
	}
}

func applyCommand() *cli.Command {
	return &cli.Command{
		Name:        "apply",
		Usage:       "Runs `tofu apply`",
		Description: "runs `tofu apply` to prepare infrastructure and Kubernetes clusters for tests",
		Action:      subcommands.Apply,
	}
}

func deployCommand() *cli.Command {
	return &cli.Command{
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
	}
}

func loadCommand() *cli.Command {
	return &cli.Command{
		Name:        "load",
		Usage:       "Creates K8s resources on upstream and downstream clusters",
		Description: "Loads ConfigMaps and Secrets on all the deployed K8s cluster; Roles, Users and Projects on the Rancher cluster",
		Action:      subcommands.Load,
	}
}

func getAccessCommand() *cli.Command {
	return &cli.Command{
		Name:        "get-access",
		Usage:       "Retrieves information to access the deployed clusters",
		Description: "print out links and access information for the deployed clusters",
		Action:      subcommands.GetAccess,
	}
}

func destroyCommand() *cli.Command {
	return &cli.Command{
		Name:        "destroy",
		Usage:       "Tears down the test environment (all the clusters)",
		Description: "runs `tofu destroy` to destroy all the provisioned clusters",
		Action:      subcommands.Destroy,
	}
}

func reapplyCommand() *cli.Command {
	return &cli.Command{
		Name:        "reapply",
		Usage:       "Tears down the test environment (all the clusters) and re-runs `tofu apply`",
		Description: "runs `tofu destroy` and then `tofu apply`",
		Action:      subcommands.Reapply,
	}
}

func redeployCommand() *cli.Command {
	return &cli.Command{
		Name:        "redeploy",
		Usage:       "Tears down the test environment (all the clusters) and redeploys them from scratch",
		Description: "runs `tofu destroy` and then deploys all the provisioned clusters",
		Action:      subcommands.Redeploy,
	}
}

func collectMetricsCommand() *cli.Command {
	return &cli.Command{
		Name:        "collect-metrics",
		Usage:       "Collect scaling metrics from a cluster's Prometheus",
		Description: "port-forwards to the cluster's Prometheus and exports curated range queries as CSV and JSON",
		Action:      subcommands.CollectMetrics,
		Flags: append(subcommands.TargetFlags(),
			&cli.StringFlag{Name: subcommands.ArgMetricsStart, Usage: "RFC3339 start (default: end minus last)"},
			&cli.StringFlag{Name: subcommands.ArgMetricsEnd, Usage: "RFC3339 end (default: now)"},
			&cli.StringFlag{Name: subcommands.ArgMetricsLast, Value: "1h", Usage: "lookback when start is omitted"},
			&cli.StringFlag{Name: subcommands.ArgMetricsStep, Value: "30s", Usage: "Prometheus query_range step"},
			&cli.StringFlag{Name: subcommands.ArgMetricsOutput, Usage: "output directory"},
			&cli.StringFlag{
				Name:  subcommands.ArgPreset,
				Value: "rancher",
				Usage: "curated query set to collect: \"rancher\" or \"k3k\"",
			},
			&cli.StringFlag{
				Name:  subcommands.ArgNamespaces,
				Usage: "namespace regex for per-pod queries (default: preset-specific)",
			},
			&cli.StringFlag{Name: subcommands.ArgK3KRunPrefix, Usage: "k3k bulk run name prefix (requires k3k-start-index and k3k-count)"},
			&cli.IntFlag{Name: subcommands.ArgK3KStartIndex, Usage: "first numeric index in the k3k bulk run"},
			&cli.IntFlag{Name: subcommands.ArgK3KCount, Usage: "number of clusters in the k3k bulk run"},
			&cli.BoolFlag{Name: subcommands.ArgOperational, Usage: "include optional health, lifecycle, placement, and controller-runtime metrics"},
			&cli.StringFlag{
				Name:  subcommands.ArgPrometheusNamespace,
				Value: subcommands.DefaultPrometheusNamespace,
				Usage: "namespace hosting Prometheus",
			},
			&cli.StringFlag{
				Name:  subcommands.ArgPrometheusService,
				Usage: "Prometheus port-forward target, e.g. svc/my-prometheus (default: detect rancher-monitoring or kube-prometheus-stack)",
			},
			&cli.StringFlag{
				Name: subcommands.ArgK3KFocusNamespace,
				Usage: "k3k preset only: give this virtual cluster's namespace its own " +
					"unbounded per-pod series, so it is never crowded out of the bounded ones",
			},
		),
	}
}

func summarizeCommand() *cli.Command {
	return &cli.Command{
		Name:        "summarize",
		Usage:       "Summarize the current deployment by capturing metrics, profiles, and resource counts",
		Description: "runs `export-metrics`, `collect-profile`, and `resource-counts` against the deployed clusters",
		Action:      subcommands.Summarize,
		Flags: append(subcommands.TargetFlags(),
			&cli.BoolFlag{
				Name:    "metrics",
				Aliases: []string{"m"},
				Value:   false,
				Usage:   "only include metrics in summary",
			},
			&cli.StringFlag{
				Name:  "query",
				Value: `{__name__!=""}`,
				Usage: "prometheus expression for metrics query, defaults to all metrics",
			},
			&cli.StringFlag{
				Name:  "start-time",
				Value: "",
				Usage: "start time for metrics export (RFC3339), defaults to 1hr ago",
			},
			&cli.StringFlag{
				Name:  "end-time",
				Value: "",
				Usage: "end time for metrics export (RFC3339), defaults to current time",
			},
			&cli.IntFlag{
				Name:  "step",
				Value: 0,
				Usage: "step/offset in seconds for metrics export, defaults to 3600 (1hr), max 7200 (2hrs)",
			},
			&cli.BoolFlag{
				Name:    "counts",
				Aliases: []string{"c"},
				Value:   false,
				Usage:   "only include current resource counts in summary",
			},
			&cli.BoolFlag{
				Name:    "profiles",
				Aliases: []string{"p"},
				Value:   false,
				Usage:   "only include current profiles in summary",
			},
			&cli.StringFlag{
				Name:  subcommands.ArgPrometheusNamespace,
				Value: subcommands.DefaultPrometheusNamespace,
				Usage: "namespace hosting Prometheus",
			},
			&cli.StringFlag{
				Name:  subcommands.ArgPrometheusService,
				Usage: "Prometheus service to remote-read from (default: detect rancher-monitoring or kube-prometheus-stack)",
			},
		),
	}
}

func collectProfilesCommand() *cli.Command {
	return &cli.Command{
		Name: "collect-profiles", Usage: "Collect Rancher pprof profiles over time", Action: subcommands.CollectProfiles,
		Flags: append(subcommands.TargetFlags(),
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsFor, Value: "10m"},
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsInterval, Value: "120s"},
			&cli.StringFlag{Name: subcommands.ArgProfiles, Value: "goroutine,heap,profile"},
			&cli.StringFlag{Name: subcommands.ArgCPUDuration, Value: "30s"},
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsOutput},
		),
	}
}

func collectLogsCommand() *cli.Command {
	return &cli.Command{
		Name: "collect-logs", Usage: "Collect Rancher-family or k3k logs over time", Action: subcommands.CollectLogs,
		Flags: append(subcommands.TargetFlags(),
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsFor, Value: "10m"},
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsInterval, Value: "60s"},
			&cli.StringFlag{
				Name:  subcommands.ArgApps,
				Value: "rancher",
				Usage: "comma-separated apps to collect: rancher, cattle-cluster-agent, " +
					"fleet-controller, fleet-agent, k3k, k3k-server, k3k-agent",
			},
			&cli.StringFlag{
				Name: subcommands.ArgK3KFocusNamespace,
				Usage: "always collect this virtual cluster's namespace, in addition to the " +
					"first 5 pods per k3k role",
			},
			&cli.StringFlag{Name: subcommands.ArgDiagnosticsOutput},
		),
	}
}

func main() {
	app := &cli.Command{
		Usage:     "setup and test Rancher (at scale if needed)",
		Copyright: "(c) 2024 SUSE LLC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    subcommands.ArgDart,
				Aliases: []string{"d"},
				Value:   filepath.Join("darts", "k3d.yaml"),
				Usage:   "dart to use",
				Sources: cli.EnvVars("DART"),
			},
		},
		Commands: appCommands(),
	}

	ctx := context.Background()
	if err := app.Run(ctx, os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			log.Print(err)
			os.Exit(exitErr.ExitCode())
		}

		log.Fatal(err)
	}
}
