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
	"log"
	"os"
	"path/filepath"

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
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
