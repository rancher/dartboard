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
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/helm"
	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/tofu"
	"github.com/urfave/cli/v2"
)

type chart struct {
	name      string
	namespace string
	path      string
}

func Deploy(cli *cli.Context) error {
	// Tofu
	tf, r, err := prepare(cli)
	if err != nil {
		return err
	}

	if !cli.Bool(ArgSkipApply) {
		if err = tf.PrintVersion(cli.Context); err != nil {
			return err
		}
		if err = tf.Apply(cli.Context); err != nil {
			return err
		}
	}

	clusters, err := tf.OutputClusters(cli.Context)
	if err != nil {
		return err
	}

	// Helm charts
	tester := clusters["tester"]

	if err = chartInstall(tester.Kubeconfig, chart{"k6-files", "tester", "k6-files"}, nil); err != nil {
		return err
	}
	if err = chartInstall(tester.Kubeconfig, chart{"mimir", "tester", "mimir"}, nil); err != nil {
		return err
	}
	if err = chartInstall(tester.Kubeconfig, chart{"grafana-dashboards", "tester", "grafana-dashboards"}, nil); err != nil {
		return err
	}
	if err = chartInstallGrafana(r, &tester); err != nil {
		return err
	}

	upstream := clusters["upstream"]
	rancherVersion := r.ChartVariables.RancherVersion
	rancherImageTag := "v" + rancherVersion
	if r.ChartVariables.RancherImageTagOverride != "" {
		rancherImageTag = r.ChartVariables.RancherImageTagOverride
		image := "rancher/rancher"
		if r.ChartVariables.RancherImageOverride != "" {
			image = r.ChartVariables.RancherImageOverride
		}
		err = importImageIntoK3d(tf, image+":"+rancherImageTag, upstream)
		if err != nil {
			return err
		}
	}

	if err = chartInstallCertManager(r, &upstream); err != nil {
		return err
	}
	if err = chartInstallRancher(r, rancherImageTag, &upstream); err != nil {
		return err
	}
	if err = chartInstallRancherIngress(&upstream); err != nil {
		return err
	}
	if err = chartInstallCgroupsExporter(&upstream); err != nil {
		return err
	}

	// Wait for Rancher deployments to be complete, or subsequent steps may fail
	if err = kubectl.WaitRancher(upstream.Kubeconfig); err != nil {
		return err
	}
	if err = chartInstallRancherMonitoring(r, &upstream); err != nil {
		return err
	}
	if err = importDownstreamClusters(r, rancherImageTag, tf, clusters); err != nil {
		return err
	}

	return GetAccess(cli)
}

func chartInstall(kubeConf string, chart chart, vals map[string]any) error {
	var err error

	name := chart.name
	namespace := chart.namespace
	path := chart.path
	if !strings.HasPrefix(path, "http") {
		path = filepath.Join("charts", path)
	}

	log.Printf("Installing chart %q (%s)\n", namespace+"/"+name, path)

	if err = helm.Install(kubeConf, path, name, namespace, vals); err != nil {
		return fmt.Errorf("chart %s: %w", name, err)
	}
	return nil
}

func chartInstallGrafana(r *dart.Dart, cluster *tofu.Cluster) error {
	chartGrafana := chart{
		name:      "grafana",
		namespace: "tester",
		path:      fmt.Sprintf("https://github.com/grafana/helm-charts/releases/download/grafana-%[1]s/grafana-%[1]s.tgz", r.ChartVariables.TesterGrafanaVersion),
	}

	clusterAdd, err := getAppAddressFor(*cluster)
	if err != nil {
		return fmt.Errorf("chart %s: %w", chartGrafana.name, err)
	}

	grafanaName := clusterAdd.Local.Name
	grafanaURL := clusterAdd.Local.HTTPURL
	chartVals := getGrafanaValsJSON(r, grafanaName, grafanaURL, cluster.IngressClassName)

	return chartInstall(cluster.Kubeconfig, chartGrafana, chartVals)
}

func chartInstallCertManager(r *dart.Dart, cluster *tofu.Cluster) error {
	chartCertManager := chart{
		name:      "cert-manager",
		namespace: "cert-manager",
		path:      fmt.Sprintf("https://charts.jetstack.io/charts/cert-manager-v%s.tgz", r.ChartVariables.CertManagerVersion),
	}
	return chartInstall(cluster.Kubeconfig, chartCertManager, map[string]any{"installCRDs": true})
}

func chartInstallRancher(r *dart.Dart, rancherImageTag string, cluster *tofu.Cluster) error {
	var rancherRepo string

	if r.ChartVariables.RancherChartRepoOverride != "" {
		rancherRepo = r.ChartVariables.RancherChartRepoOverride
	} else {
		// "prime"
		if r.ChartVariables.ForcePrimeRegistry {
			rancherRepo = "https://charts.rancher.com/server-charts/prime"
		}

		baseRepo := "https://releases.rancher.com/server-charts/"

		// otherwise, if one of "alpha", or "latest"
		if strings.Contains(r.ChartVariables.RancherVersion, "alpha") {
			rancherRepo = baseRepo + "alpha"
		} else {
			rancherRepo = baseRepo + "latest"
		}
	}

	chartRancher := chart{
		name:      "rancher",
		namespace: "cattle-system",
		path:      rancherRepo + "/rancher-" + r.ChartVariables.RancherVersion + ".tgz",
	}

	clusterAdd, err := getAppAddressFor(*cluster)
	if err != nil {
		return fmt.Errorf("chart %s: %w", chartRancher.name, err)
	}
	rancherClusterName := clusterAdd.Public.Name
	rancherClusterURL := clusterAdd.Public.HTTPSURL

	var extraEnv []map[string]any
	extraEnv = []map[string]any{
		{
			"name":  "CATTLE_SERVER_URL",
			"value": rancherClusterURL,
		},
		{
			"name":  "CATTLE_PROMETHEUS_METRICS",
			"value": "true",
		},
		{
			"name":  "CATTLE_DEV_MODE",
			"value": "true",
		},
	}
	extraEnv = append(extraEnv, r.ChartVariables.ExtraEnvironmentVariables...)

	chartVals := getRancherValsJSON(r.ChartVariables.RancherImageOverride, rancherImageTag, r.ChartVariables.AdminPassword, rancherClusterName, extraEnv, r.ChartVariables.RancherReplicas)

	fmt.Printf("\n\nRANCHER CHART VALS:\n")

	for key, value := range chartVals {
		fmt.Printf("\t%s = %v\n", key, value)
	}

	return chartInstall(cluster.Kubeconfig, chartRancher, chartVals)
}

func chartInstallRancherIngress(cluster *tofu.Cluster) error {
	chartRancherIngress := chart{
		name:      "rancher-ingress",
		namespace: "default",
		path:      "rancher-ingress",
	}

	clusterAdd, err := getAppAddressFor(*cluster)
	if err != nil {
		return fmt.Errorf("chart %s: %w", chartRancherIngress.name, err)
	}

	var sans []string
	if len(clusterAdd.Local.Name) > 0 {
		sans = append(sans, clusterAdd.Local.Name)
	}
	if len(clusterAdd.Public.Name) > 0 {
		sans = append(sans, clusterAdd.Public.Name)
	}

	chartVals := map[string]any{
		"sans":             sans,
		"ingressClassName": cluster.IngressClassName,
	}

	return chartInstall(cluster.Kubeconfig, chartRancherIngress, chartVals)
}

func chartInstallRancherMonitoring(r *dart.Dart, cluster *tofu.Cluster) error {
	rancherMinorVersion := strings.Join(strings.Split(r.ChartVariables.RancherVersion, ".")[0:2], ".")

	chartRancherMonitoringCRD := chart{
		name:      "rancher-monitoring-crd",
		namespace: "cattle-monitoring-system",
		path:      fmt.Sprintf("https://github.com/rancher/charts/raw/release-v%s/assets/rancher-monitoring-crd/rancher-monitoring-crd-%s.tgz", rancherMinorVersion, r.ChartVariables.RancherMonitoringVersion),
	}

	chartVals := map[string]any{
		"global": map[string]any{
			"cattle": map[string]any{
				"clusterId":             "local",
				"clusterName":           "local",
				"systemDefaultRegistry": "",
			},
		},
		"systemDefaultRegistry": "",
	}
	err := chartInstall(cluster.Kubeconfig, chartRancherMonitoringCRD, chartVals)
	if err != nil {
		return err
	}

	chartRancherMonitoring := chart{
		name:      "rancher-monitoring",
		namespace: "cattle-monitoring-system",
		path:      fmt.Sprintf("https://github.com/rancher/charts/raw/release-v%s/assets/rancher-monitoring/rancher-monitoring-%s.tgz", rancherMinorVersion, r.ChartVariables.RancherMonitoringVersion),
	}

	clusterAdd, err := getAppAddressFor(*cluster)
	if err != nil {
		return fmt.Errorf("chart %s: %w", chartRancherMonitoring.name, err)
	}
	mimirURL := clusterAdd.Public.HTTPURL + "/mimir/api/v1/push"

	chartVals = getRancherMonitoringValsJSON(cluster.ReserveNodeForMonitoring, mimirURL)

	return chartInstall(cluster.Kubeconfig, chartRancherMonitoring, chartVals)
}

func chartInstallCgroupsExporter(cluster *tofu.Cluster) error {
	return chartInstall(cluster.Kubeconfig, chart{"cgroups-exporter", "cattle-monitoring-system", "cgroups-exporter"}, nil)
}

func getRancherMonitoringValsJSON(reserveNodeForMonitoring bool, mimirURL string) map[string]any {

	nodeSelector := map[string]any{}
	tolerations := []any{}
	monitoringRestrictions := map[string]any{}
	if reserveNodeForMonitoring {
		nodeSelector["monitoring"] = "true"
		tolerations = append(tolerations, map[string]any{"key": "monitoring", "operator": "Exists", "effect": "NoSchedule"})
		monitoringRestrictions["nodeSelector"] = nodeSelector
		monitoringRestrictions["tolerations"] = tolerations
	}

	remoteWrite := []any{}
	if len(mimirURL) > 0 {
		remoteWrite = append(remoteWrite, map[string]any{
			"url": mimirURL,
			"writeRelabelConfigs": []any{
				map[string]any{
					"sourceLabels": []any{"__name__"},
					"regex":        "(node_namespace_pod_container|node_cpu|node_load|node_memory|node_network_receive_bytes_total|container_network_receive_bytes_total|cgroups_).*",
					"action":       "keep",
				},
			},
		})
	}

	return map[string]any{
		"alertmanager": map[string]any{"enabled": false},
		"grafana":      monitoringRestrictions,
		"prometheus": map[string]any{
			"prometheusSpec": map[string]any{
				"evaluationInterval": "1m",
				"nodeSelector":       nodeSelector,
				"tolerations":        tolerations,
				"resources":          map[string]any{"limits": map[string]any{"memory": "10000Mi"}},
				"retentionSize":      "50GiB",
				"scrapeInterval":     "1m",

				"additionalScrapeConfigs": []any{
					map[string]any{
						"job_name":     "node-cgroups-exporter",
						"honor_labels": false,
						"kubernetes_sd_configs": []any{map[string]any{
							"role": "node",
						}},
						"scheme": "http",
						"relabel_configs": []any{
							map[string]any{
								"action": "labelmap",
								"regex":  "__meta_kubernetes_node_label_(.+)",
							},
							map[string]any{
								"source_labels": []any{"__address__"},
								"action":        "replace",
								"target_label":  "__address__",
								"regex":         "([^:;]+):(\\d+)",
								"replacement":   "${1}:9753",
							},
							map[string]any{
								"source_labels": []any{"__meta_kubernetes_node_name"},
								"action":        "keep",
								"regex":         ".*",
							},
							map[string]any{
								"source_labels": []any{"__meta_kubernetes_node_name"},
								"action":        "replace",
								"target_label":  "node",
								"regex":         "(.*)",
								"replacement":   "${1}",
							},
						},
					},
				},

				"remoteWrite": remoteWrite,
			},
		},
		"prometheus-adapter": monitoringRestrictions,
		"kube-state-metrics": monitoringRestrictions,
		"prometheusOperator": monitoringRestrictions,
		"global": map[string]any{
			"cattle": map[string]any{
				"clusterId":             "local",
				"clusterName":           "local",
				"systemDefaultRegistry": "",
			},
		},
		"systemDefaultRegistry": "",
	}
}

func getGrafanaValsJSON(r *dart.Dart, name, url, ingressClass string) map[string]any {
	return map[string]any{
		"datasources": map[string]any{
			"datasources.yaml": map[string]any{
				"apiVersion": 1,
				"datasources": []any{map[string]any{
					"name":      "mimir",
					"type":      "prometheus",
					"url":       "http://mimir.tester:9009/mimir/prometheus",
					"access":    "proxy",
					"isDefault": true,
				}},
			},
		},
		"dashboardProviders": map[string]any{
			"dashboardproviders.yaml": map[string]any{
				"apiVersion": 1,
				"providers": []any{map[string]any{
					"name":            "default",
					"folder":          "",
					"type":            "file",
					"disableDeletion": false,
					"editable":        true,
					"options": map[string]any{
						"path": "/var/lib/grafana/dashboards/default",
					},
				}},
			},
		},
		"dashboardsConfigMaps": map[string]any{"default": "grafana-dashboards"},
		"ingress": map[string]any{
			"enabled":          true,
			"path":             "/grafana",
			"hosts":            []string{name},
			"ingressClassName": ingressClass,
		},
		"env": map[string]any{
			"GF_SERVER_ROOT_URL":            url + "/grafana",
			"GF_SERVER_SERVE_FROM_SUB_PATH": true,
		},
		"adminPassword": r.ChartVariables.AdminPassword,
	}
}

func getRancherValsJSON(rancherImageOverride, rancherImageTag, bootPwd, hostname string, extraEnv []map[string]any, replicas int) map[string]any {
	result := map[string]any{
		"bootstrapPassword": bootPwd,
		"hostname":          hostname,
		"replicas":          replicas,
		"rancherImageTag":   rancherImageTag,
		"extraEnv":          extraEnv,
		"livenessProbe": map[string]any{
			"initialDelaySeconds": 30,
			"periodSeconds":       3600,
		},
	}

	if rancherImageOverride != "" {
		result["rancherImage"] = rancherImageOverride
	}

	return result
}

func importDownstreamClusters(r *dart.Dart, rancherImageTag string, tf *tofu.Tofu, clusters map[string]tofu.Cluster) error {

	log.Print("Import downstream clusters")

	if err := importDownstreamClustersRancherSetup(r, clusters); err != nil {
		return err
	}

	buffer := 10
	clustersChan := make(chan string, buffer)
	errorChan := make(chan error)
	clustersCount := 0

	for clusterName := range clusters {
		if !strings.HasPrefix(clusterName, "downstream") {
			continue
		}
		clustersCount++
		go importDownstreamClusterDo(r, rancherImageTag, tf, clusters, clusterName, clustersChan, errorChan)
	}

	for {
		if clustersCount == 0 {
			return nil
		}
		select {
		case err := <-errorChan:
			return err
		case completed := <-clustersChan:
			log.Printf("Cluster %q imported successfully.\n", completed)
			clustersCount--
		}
	}
}

func importDownstreamClusterDo(r *dart.Dart, rancherImageTag string, tf *tofu.Tofu, clusters map[string]tofu.Cluster, clusterName string, ch chan<- string, errCh chan<- error) {
	log.Print("Import cluster " + clusterName)
	yamlFile, err := os.CreateTemp("", "scli-"+clusterName+"-*.yaml")
	if err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	defer os.Remove(yamlFile.Name())
	defer yamlFile.Close()

	clusterID, err := importClustersDownstreamGetYAML(clusters, clusterName, yamlFile)
	if err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	downstream, ok := clusters[clusterName]
	if !ok {
		err := fmt.Errorf("error: cannot find access data for cluster %q", clusterName)
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	if r.ChartVariables.RancherImageTagOverride != "" {
		err = importImageIntoK3d(tf, "rancher/rancher-agent:"+rancherImageTag, downstream)
		if err != nil {
			errCh <- fmt.Errorf("%s downstream k3d image import failed: %w", clusterName, err)
			return
		}
	}

	if err := kubectl.Apply(downstream.Kubeconfig, yamlFile.Name()); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	if err := kubectl.WaitForReadyCondition(clusters["upstream"].Kubeconfig,
		"clusters.management.cattle.io", clusterID, "", "ready", 10); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	if err := kubectl.WaitForReadyCondition(clusters["upstream"].Kubeconfig,
		"cluster.fleet.cattle.io", clusterName, "fleet-default", "ready", 10); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	err = kubectl.WaitForReadyCondition(downstream.Kubeconfig, "deployment", "rancher-webhook", "cattle-system", "available", 60)
	if err != nil {
		errCh <- fmt.Errorf("%s waiting for rancher-webhook failed: %w", clusterName, err)
		return
	}
	if r.ChartVariables.DownstreamRancherMonitoring {
		if err := chartInstallRancherMonitoring(r, &downstream); err != nil {
			errCh <- fmt.Errorf("downstream monitoring installation on cluster %s failed: %w", clusterName, err)
			return
		}
	}
	ch <- clusterName
}

func importDownstreamClustersRancherSetup(r *dart.Dart, clusters map[string]tofu.Cluster) error {
	tester := clusters["tester"]
	upstream := clusters["upstream"]
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return err
	}

	downstreamClusters := []string{}
	for clusterName := range clusters {
		if strings.HasPrefix(clusterName, "downstream") {
			downstreamClusters = append(downstreamClusters, clusterName)
		}
	}
	importedClusterNames := strings.Join(downstreamClusters, ",")

	envVars := map[string]string{
		"BASE_URL":               upstreamAdd.Public.HTTPSURL,
		"BOOTSTRAP_PASSWORD":     "admin",
		"PASSWORD":               r.ChartVariables.AdminPassword,
		"IMPORTED_CLUSTER_NAMES": importedClusterNames,
	}

	if err = kubectl.K6run(tester.Kubeconfig, "k6/rancher_setup.js", envVars, nil, true, upstreamAdd.Local.HTTPSURL, false); err != nil {
		return err
	}
	return nil
}

func importClustersDownstreamGetYAML(clusters map[string]tofu.Cluster, name string, yamlFile *os.File) (clusterID string, err error) {
	var status map[string]interface{}

	upstream := clusters["upstream"]
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return
	}

	if status, err = kubectl.GetStatus(upstream.Kubeconfig, "clusters.provisioning.cattle.io", name, "fleet-default"); err != nil {
		return
	}
	clusterID, ok := status["clusterName"].(string)
	if !ok {
		err = fmt.Errorf("error accessing fleet-default/%s clusters: no valid 'clusterName' in 'Status'", name)
		return
	}

	if status, err = kubectl.GetStatus(upstream.Kubeconfig, "clusterregistrationtokens.management.cattle.io", "default-token", clusterID); err != nil {
		return
	}
	token, ok := status["token"].(string)
	if !ok {
		err = fmt.Errorf("error accessing %s/default-token clusterregistrationtokens: no valid 'token' in 'Status'", clusterID)
		return
	}

	url := fmt.Sprintf("%s/v3/import/%s_%s.yaml", upstreamAdd.Local.HTTPSURL, token, clusterID)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(yamlFile, resp.Body)
	if err != nil {
		return
	}
	if err = yamlFile.Sync(); err != nil {
		return
	}

	return
}
