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

package helm

import (
	"log"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

// Timeout for chart installation, default is 1 minute but it could be not enough sometimes...
const timeout = 3 * time.Minute

func Install(kubecfg, chartLocation, releaseName, namespace string, vals map[string]interface{}) error {
	settings := cli.New()
	settings.KubeConfig = kubecfg
	settings.SetNamespace(namespace)

	var chartPath string
	var chart *chart.Chart
	var err error

	actionConfig := new(action.Configuration)

	// TODO: use logger to provide debug logs
	var logger = func(format string, v ...interface{}) {}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger); err != nil {
		return err
	}

	// Check if the chart is already installed...
	hCli := action.NewHistory(actionConfig)
	hCli.Max = 1
	// ...if not, install it...
	if _, err = hCli.Run(releaseName); err == driver.ErrReleaseNotFound {
		if err == driver.ErrReleaseNotFound {
			installAction := action.NewInstall(actionConfig)
			installAction.CreateNamespace = true
			installAction.ReleaseName = releaseName
			installAction.Namespace = namespace
			installAction.Timeout = timeout
			if chartPath, err = installAction.LocateChart(chartLocation, settings); err != nil {
				return err
			}
			if chart, err = loader.Load(chartPath); err != nil {
				return err
			}
			r, err := installAction.Run(chart, vals)
			return printNotes(r, err)
		}
		return err
	}

	// ...otherwise do an upgrade.
	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Install = true
	upgradeAction.Namespace = namespace
	upgradeAction.Timeout = timeout
	if chartPath, err = upgradeAction.LocateChart(chartLocation, settings); err != nil {
		return err
	}
	if chart, err = loader.Load(chartPath); err != nil {
		return err
	}
	r, err := upgradeAction.Run(releaseName, chart, vals)
	return printNotes(r, err)
}

// printNotes prints the release notes to the log unless there was an error
func printNotes(r *release.Release, err error) error {
	if err != nil {
		return err
	}
	if r.Info.Notes != "" {
		log.Printf(r.Info.Notes)
	} else {
		log.Printf("No notes for release %s", r.Name)
	}
	return nil
}
