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
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/helm"
	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/tofu"
	"github.com/rancher/shepherd/pkg/session"
	cli "github.com/urfave/cli/v2"

	"github.com/sirupsen/logrus"

	"github.com/rancher/dartboard/internal/actions"
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
		if err = tf.PrintVersion(); err != nil {
			return err
		}
		if err = tf.Apply(); err != nil {
			return err
		}
	} else {
		if err = tf.Output(nil, false); err != nil {
			return err
		}
	}

	clusters, custom_clusters, err := tf.ParseOutputs()
	if err != nil {
		return err
	}

	// Helm charts
	tester := clusters["tester"]
	if len(tester.Kubeconfig) > 0 && !cli.Bool(ArgSkipCharts) {
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

	if !cli.Bool(ArgSkipCharts) {
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
	}

	// Setup rancher client
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return err
	}

	rancherSession := session.NewSession()
	rancherSession.CleanupEnabled = false
	log.Printf("Setting up Rancher Client's Config")
	rancherHost := strings.Split(upstreamAdd.Public.HTTPSURL, "://")[1]
	rancherConfig := actions.NewRancherConfig(rancherHost, "", r.ChartVariables.AdminPassword, true)
	log.Printf("Setting up Rancher Client")
	rancherClient, err := actions.SetupRancherClient(&rancherConfig, r.ChartVariables.AdminPassword, rancherSession)
	if err != nil {
		return err
	}

	// Get all downstream cluster info
	downstreamClusters := []tofu.Cluster{}
	for k, v := range clusters {
		if strings.HasPrefix(k, "downstream") {
			v.Name = k
			downstreamClusters = append(downstreamClusters, v)
		}
	}
	SortItemsNaturally(downstreamClusters, func(c tofu.Cluster) string { return c.Name })

	jsonBytes, err := json.MarshalIndent(downstreamClusters, "", "    ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}
	fmt.Println("Import Clusters:\n", string(jsonBytes))

	log.Printf("Importing Downstream Clusters")
	if err = actions.ImportDownstreamClusters(r, downstreamClusters, rancherClient, &rancherConfig); err != nil {
		return err
	}

	logrus.Debugf("\nBEFORE CUSTOM CLUSTER LOGIC\n")
	if len(custom_clusters) > 0 {
		logrus.Debugf("\nIN CUSTOM CLUSTER LOGIC\n")
		// Get all custom cluster info
		if err := actions.RegisterCustomClusters(r, custom_clusters, rancherClient, &rancherConfig); err != nil {
			return err
		}
	}

	if len(r.ClusterTemplates) > 0 {
		// If we have provision Cluster Templates then setup Harvester Client + import Harvester Cluster into Rancher
		if strings.Contains(r.TofuMainDirectory, "harvester") {
			log.Printf("Parsing Harvester's Kubeconfig")
			var kubeconfig *actions.Kubeconfig
			if len(r.TofuVariables["kubeconfig"].(string)) > 0 {
				kubeconfig, err = actions.ParseKubeconfig(r.TofuVariables["kubeconfig"].(string))
				if err != nil {
					return fmt.Errorf("error while parsing kubeconfig at %v: %v", r.TofuVariables["kubeconfig"].(string), err)
				}
			}

			var harvesterClient *actions.HarvesterImportClient
			log.Printf("Setting up Harvester Client's Config")
			harvesterHost := strings.Split(kubeconfig.Clusters[0].Cluster.Server, "://")[1]
			harvesterConfig := actions.NewHarvesterConfig(harvesterHost, kubeconfig.Users[0].User.Token, "", true)

			log.Printf("Setting up Harvester Client")
			harvesterClient, err = actions.NewHarvesterImportClient(rancherClient, &harvesterConfig)
			if err != nil {
				return fmt.Errorf("error while setting up HarvesterImportClient with config %v: %v", harvesterConfig, err)
			}

			log.Printf("Importing Harvester Cluster into Rancher for provisioning")
			err = harvesterClient.ImportCluster()
			if err != nil {
				return fmt.Errorf("error while importing Harvester cluster into Rancher %v: %v", harvesterConfig, err)
			}
		}

		log.Printf("Provisioning Downstream Clusters")
		if err = actions.ProvisionDownstreamClusters(r, r.ClusterTemplates, rancherClient); err != nil {
			return err
		}
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
		baseRepo := "https://releases.rancher.com/server-charts/"

		// otherwise, if one of "alpha", or "latest"
		if strings.Contains(r.ChartVariables.RancherVersion, "alpha") {
			rancherRepo = baseRepo + "alpha"
		} else {
			rancherRepo = baseRepo + "latest"
		}

		// "prime"
		if r.ChartVariables.ForcePrimeRegistry {
			rancherRepo = "https://charts.rancher.com/server-charts/prime"
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

// naturalCompare compares strings a and b in "natural" alphanumeric order
func naturalCompare(a, b string) bool {
	var tokenRegex = regexp.MustCompile(`\d+|\D+`)
	// split into tokens of numbers
	aTokens := tokenRegex.FindAllString(a, -1)
	bTokens := tokenRegex.FindAllString(b, -1)
	for i := 0; i < len(aTokens) && i < len(bTokens); i++ {
		aTok, bTok := aTokens[i], bTokens[i]
		// If both tokens are numeric, compare as integers
		if aNum, errA := strconv.Atoi(aTok); errA == nil {
			if bNum, errB := strconv.Atoi(bTok); errB == nil {
				if aNum != bNum {
					return aNum < bNum
				}
				continue // numbers are equal, move to next token
			}
		}
		// Fallback to default lexicographic compare
		if aTok != bTok {
			return aTok < bTok
		}
	}
	// If all shared tokens are equal, the shorter string is less
	return len(aTokens) < len(bTokens)
}

// SortItemsNaturally ia a generic function that sorts a slice of a given type
// by "Name" (any provided string) using natural order
func SortItemsNaturally[T any](items []T, getName func(T) string) {
	sort.Slice(items, func(i, j int) bool {
		return naturalCompare(getName(items[i]), getName(items[j]))
	})
}
