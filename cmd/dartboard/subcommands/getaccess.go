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
	"context"
	"fmt"
	"strings"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
	cli "github.com/urfave/cli/v3"
)

const (
	grafanaMonitoringNamespace   = "cattle-monitoring-system"
	grafanaMonitoringServiceName = "rancher-monitoring-grafana"
	grafanaKPSServiceName        = "kube-prometheus-stack-grafana"
)

func GetAccess(ctx context.Context, cli *cli.Command) error {
	tf, r, err := prepare(ctx, cli)
	if err != nil {
		return err
	}

	clusters, _, err := tf.ParseOutputs()
	if err != nil {
		return err
	}

	upstream := clusters["upstream"]
	tester := clusters["tester"]

	downstreams := []tofu.Cluster{}

	for k, v := range clusters {
		if strings.HasPrefix(k, "downstream") {
			v.Name = k
			downstreams = append(downstreams, v)
		}
	}

	SortItemsNaturally(downstreams, func(c tofu.Cluster) string { return c.Name })

	// A standalone deploy has no upstream; looking it up would print a bogus
	// Rancher URL (and could query whatever KUBECONFIG is in the environment).
	rancherURL := ""

	if upstream.Kubeconfig != "" {
		upstreamAddresses, err := getAppAddressFor(upstream)
		if err == nil {
			rancherURL = upstreamAddresses.Local.HTTPSURL
		} else {
			fmt.Printf("Error getting application addresses for cluster upstream: %v\n", err)
		}
	}

	fmt.Println("\n\n\n*** ACCESS DETAILS")
	fmt.Println()

	printAccessDetails(r, "UPSTREAM", upstream, rancherURL)
	printRancherMonitoringGrafanaAccess(r, upstream)

	for _, downstream := range downstreams {
		printAccessDetails(r, strings.ToUpper(downstream.Name), downstream, "")

		if r.ChartVariables.DownstreamRancherMonitoring {
			printRancherMonitoringGrafanaAccess(r, downstream)
		}
	}

	printAccessDetails(r, "TESTER", tester, "")
	printTesterGrafanaAccess(tester)

	return nil
}

// printRancherMonitoringGrafanaAccess prints how to reach the rancher-monitoring
// Grafana instance installed on cluster, if any. Unlike the tester's Grafana,
// this one has no ingress of its own -- it is normally only reachable by
// proxying through the Rancher UI -- so a kubectl port-forward is the only
// portable way to reach it directly, with or without an upstream/Rancher
// cluster. It prints nothing for a cluster without a kubeconfig; it does not
// check that monitoring is actually installed there.
func printRancherMonitoringGrafanaAccess(r *dart.Dart, cluster tofu.Cluster) {
	if cluster.Kubeconfig == "" {
		return
	}

	service := grafanaKPSServiceName
	if useLegacyRancherMonitoring(r) {
		service = grafanaMonitoringServiceName
	}

	fmt.Println("    Grafana Dashboard (rancher-monitoring, this cluster's own metrics):")
	fmt.Printf("        kubectl --kubeconfig %q port-forward -n %s svc/%s 3000:80\n",
		cluster.Kubeconfig, grafanaMonitoringNamespace, service)
	fmt.Printf("        then open http://localhost:3000; retrieve credentials from the %s secret if needed\n", service)
	fmt.Println()
}

// printTesterGrafanaAccess prints how to reach the tester cluster's Grafana,
// which aggregates metrics remote_written from every cluster's
// rancher-monitoring into Mimir. Unlike rancher-monitoring's own Grafana, this
// one has its own ingress, so it needs no
// port-forward.
func printTesterGrafanaAccess(tester tofu.Cluster) {
	if tester.Kubeconfig == "" {
		return
	}

	testerAdd, err := getAppAddressFor(tester)
	if err != nil {
		return
	}

	fmt.Println("    Grafana Dashboard (aggregated metrics from all clusters, via Mimir):")
	fmt.Printf("        %s/grafana\n", testerAdd.Local.HTTPURL)
	fmt.Println()
}
