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
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/moio/scalability-tests/internal/helm"
	"github.com/moio/scalability-tests/internal/recipe"
	"github.com/moio/scalability-tests/internal/tofu"
)

type chart struct {
	name      string
	namespace string
	path      string
}

func chartInstall(kubeConf string, chart chart, jsonVals string) error {
	var vals map[string]interface{} = nil
	var err error

	name := chart.name
	namespace := chart.namespace
	path := chart.path
	if !strings.HasPrefix(path, "http") {
		path = filepath.Join("charts", path)
	}

	log.Printf("Installing chart %q (%s)\n", namespace+"/"+name, path)

	if len(jsonVals) > 0 {
		if vals, err = jsonToMap(jsonVals); err != nil {
			return fmt.Errorf("chart %s vals:\n%s\n %w", name, jsonVals, err)
		}
	}

	if err = helm.Install(kubeConf, path, name, namespace, vals); err != nil {
		return fmt.Errorf("chart %s: %w", name, err)
	}
	return nil
}

func chartInstallGrafana(r *recipe.Recipe, cluster *tofu.Cluster) error {
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

func chartInstallCertManager(r *recipe.Recipe, cluster *tofu.Cluster) error {
	chartCertManager := chart{
		name:      "cert-manager",
		namespace: "cert-manager",
		path:      fmt.Sprintf("https://charts.jetstack.io/charts/cert-manager-v%s.tgz", r.ChartVariables.CertManagerVersion),
	}
	return chartInstall(cluster.Kubeconfig, chartCertManager, `{"installCRDs": true}`)
}

func chartInstallRancher(r *recipe.Recipe, rancherImageTag string, cluster *tofu.Cluster) error {
	rancherRepo := "https://releases.rancher.com/server-charts/"

	// one of "alpha", "latest" or "stable"
	if strings.Contains(r.ChartVariables.RancherVersion, "alpha") {
		rancherRepo += "alpha"
	} else {
		rancherRepo += "latest"
	}

	// or "prime"
	if r.ChartVariables.ForcePrimeRegistry {
		rancherRepo = "https://charts.rancher.com/server-charts/prime"
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
	chartVals := getRancherValsJSON(rancherImageTag, r.ChartVariables.AdminPassword, rancherClusterName, rancherClusterURL, r.ChartVariables.RancherReplicas)

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

	rancherSANs := ""
	if len(clusterAdd.Local.Name) > 0 {
		rancherSANs = fmt.Sprintf("%q", clusterAdd.Local.Name)
	}
	if len(clusterAdd.Public.Name) > 0 {
		if len(rancherSANs) > 0 {
			rancherSANs += ", "
		}
		rancherSANs += fmt.Sprintf("%q", clusterAdd.Public.Name)
	}

	chartVals := `{
		"sans": [` + rancherSANs + `],
		"ingressClassName": "` + cluster.IngressClassName + `"
	}`

	return chartInstall(cluster.Kubeconfig, chartRancherIngress, chartVals)
}

func chartInstallRancherMonitoring(r *recipe.Recipe, cluster *tofu.Cluster, noSchedToleration bool) error {
	rancherMinorVersion := strings.Join(strings.Split(r.ChartVariables.RancherVersion, ".")[0:2], ".")

	chartRancherMonitoringCRD := chart{
		name:      "rancher-monitoring-crd",
		namespace: "cattle-monitoring-system",
		path:      fmt.Sprintf("https://github.com/rancher/charts/raw/release-v%s/assets/rancher-monitoring-crd/rancher-monitoring-crd-%s.tgz", rancherMinorVersion, r.ChartVariables.RancherMonitoringVersion),
	}

	chartVals := `{
		"global": {
			"cattle": {
					"clusterId": "local",
					"clusterName": "local",
					"systemDefaultRegistry": ""
			}
		},
		"systemDefaultRegistry": ""
	}`
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

	nodeSelector := ""
	tolerations := ""
	if !noSchedToleration {
		nodeSelector = `{"monitoring": "true"}`
		tolerations = `[{"key": "monitoring", "operator": "Exists", "effect": "NoSchedule"}]`
	}

	chartVals = getRancherMonitoringValsJSON(nodeSelector, tolerations, mimirURL)

	return chartInstall(cluster.Kubeconfig, chartRancherMonitoring, chartVals)
}

func chartInstallCgroupsExporter(cluster *tofu.Cluster) error {
	return chartInstall(cluster.Kubeconfig, chart{"cgroups-exporter", "cattle-monitoring-system", "cgroups-exporter"}, "")
}

func getRancherMonitoringValsJSON(nodeSelector, tolerations, mimirURL string) string {

	monitoringRestrictions := ""
	if len(nodeSelector) > 0 {
		monitoringRestrictions += fmt.Sprintf("{%q: %s,\n", "nodeSelector", nodeSelector)
	} else {
		nodeSelector = `{}`
	}
	if len(tolerations) > 0 {
		monitoringRestrictions += fmt.Sprintf("%q: %s}", "tolerations", tolerations)
	} else {
		tolerations = `[]`
	}
	if len(monitoringRestrictions) == 0 {
		monitoringRestrictions = `{}`
	}

	remoteWrite := ""
	if len(mimirURL) > 0 {
		remoteWrite = `{
			"url": ` + fmt.Sprintf("%q", mimirURL) + `,
			"writeRelabelConfigs": [
				{
					"sourceLabels": ["__name__"],
					"regex": "(node_namespace_pod_container|node_cpu|node_load|node_memory|node_network_receive_bytes_total|container_network_receive_bytes_total|cgroups_).*",
					"action": "keep"
				}
			]
		}`
	}

	jsonVals := `{
		"alertmanager": {"enabled": "false"},
		"grafana": ` + monitoringRestrictions + `,
		"prometheus": {
			"prometheusSpec": {
				"evaluationInterval": "1m",
				"nodeSelector": ` + nodeSelector + `,
				"tolerations": ` + tolerations + `,
				"resources": {"limits": {"memory": "5000Mi"}},
				"retentionSize": "50GiB",
				"scrapeInterval": "1m",

				"additionalScrapeConfigs": [
					{
						"job_name": "node-cgroups-exporter",
						"honor_labels": false,
						"kubernetes_sd_configs": [{
							"role": "node"
						}],
						"scheme": "http",
						"relabel_configs": [
							{
								"action": "labelmap",
								"regex": "__meta_kubernetes_node_label_(.+)"
							},
							{
								"source_labels": ["__address__"],
								"action": "replace",
								"target_label": "__address__",
								"regex": "([^:;]+):(\\d+)",
								"replacement": "${1}:9753"
							},
							{
								"source_labels": ["__meta_kubernetes_node_name"],
								"action": "keep",
								"regex": ".*"
							},
							{
								"source_labels": ["__meta_kubernetes_node_name"],
								"action": "replace",
								"target_label": "node",
								"regex": "(.*)",
								"replacement": "${1}"
							}
						]
					}
				],

				"remoteWrite": [` + remoteWrite + `]
			}
		},
		"prometheus-adapter": ` + monitoringRestrictions + `,
		"kube-state-metrics": ` + monitoringRestrictions + `,
		"prometheusOperator": ` + monitoringRestrictions + `,
		"global": {
			"cattle": {
				"clusterId": "local",
				"clusterName": "local",
				"systemDefaultRegistry": ""
			}
		},
		"systemDefaultRegistry": ""
	}`

	return jsonVals
}

func jsonToMap(jsonVals string) (map[string]interface{}, error) {

	var mapVals map[string]interface{}
	err := json.Unmarshal([]byte(jsonVals), &mapVals)

	return mapVals, err
}

func getGrafanaValsJSON(r *recipe.Recipe, name, url, ingressClass string) string {
	return `{
		"datasources": {
			"datasources.yaml": {
				"apiVersion": 1,
				"datasources": [{
					"name": "mimir",
					"type": "prometheus",
					"url": "http://mimir.tester:9009/mimir/prometheus",
					"access": "proxy",
					"isDefault": true
				}]
			}
		},
		"dashboardProviders": {
			"dashboardproviders.yaml": {
				"apiVersion": 1,
				"providers": [{
					"name": "default",
					"folder": "",
					"type": "file",
					"disableDeletion": false,
					"editable": true,
					"options": {
						"path": "/var/lib/grafana/dashboards/default"
					}
				}]
			}
		},
		"dashboardsConfigMaps": { "default": "grafana-dashboards" },
		"ingress": {
			"enabled": true,
			"path": "/grafana",
			"hosts": [` + fmt.Sprintf("%q", name) + `],
			"ingressClassName": ` + fmt.Sprintf("%q", ingressClass) + `
		},
		"env": {
			"GF_SERVER_ROOT_URL": ` + fmt.Sprintf("\"%s/grafana\"", url) + `,
			"GF_SERVER_SERVE_FROM_SUB_PATH": "true"
		},
		"adminPassword": ` + fmt.Sprintf("%q", r.ChartVariables.AdminPassword) + `
	}`
}

func getRancherValsJSON(rancherImageTag, bootPwd, hostname, serverURL string, replicas int) string {
	return `
	{
		"bootstrapPassword": ` + fmt.Sprintf("%q", bootPwd) + `,
		"hostname": ` + fmt.Sprintf("%q", hostname) + `,
		"replicas": ` + fmt.Sprintf("%d", replicas) + `,
		"rancherImageTag": ` + fmt.Sprintf("%q", rancherImageTag) + `,
		"extraEnv": [{
				"name": "CATTLE_SERVER_URL",
				"value": ` + fmt.Sprintf("%q", serverURL) + `
		},
		{
				"name": "CATTLE_PROMETHEUS_METRICS",
				"value": "true"
		},
		{
				"name": "CATTLE_DEV_MODE",
				"value": "true"
		}],
		"livenessProbe": {
				"initialDelaySeconds": 30,
				"periodSeconds": 3600
		}
	}`
}
