package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/moio/scalability-tests/pkg/terraform"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/storage/driver"
)

const (
	rancherVersion   = "2.7.9"
	rancherChart     = "https://releases.rancher.com/server-charts/latest/rancher-" + rancherVersion + ".tgz"
	rancherImageTag  = "v" + rancherVersion
	certManagerChart = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
	grafanaChart     = "https://github.com/grafana/helm-charts/releases/download/grafana-6.56.5/grafana-6.56.5.tgz"
	baseDir          = ".."
	adminPassword    = "adminadminadmin"
)

type clusterAddress struct {
	Name     string
	HTTPURL  string
	HTTPSURL string
}

type clusterAddresses struct {
	Local  clusterAddress
	Public clusterAddress
}

func main() {
	// Step 1: Terraform
	tf := new(terraform.Terraform)

	if err := tf.Init(terraformDir()); err != nil {
		log.Fatal(err)
	}
	terraformVersionPrint(tf)

	if err := tf.Apply(os.Getenv("TERRAFORM_VAR_FILE")); err != nil {
		log.Fatal(err)
	}

	clusters, err := tf.OutputClusters()

	// Step 3: Helm charts
	// tester cluster

	tester := clusters["tester"]

	// TODO: deal with the type assertion failure instead of triggering a panic
	kubePath := tester.Kubeconfig

	if err := installChart(kubePath, filepath.Join(baseDir, "charts", "mimir"), "mimir", "tester", nil); err != nil {
		log.Fatal(err)
	}
	if err := installChart(kubePath, filepath.Join(baseDir, "charts", "k6-files"), "k6-files", "tester", nil); err != nil {
		log.Fatal(err)
	}
	if err := installChart(kubePath, filepath.Join(baseDir, "charts", "grafana-dashboards"), "grafana-dashboards", "tester", nil); err != nil {
		log.Fatal(err)
	}

	testerAdd, err := getAppAddressFor(tester)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("testerAdd:\n%+v\n", testerAdd)
	grafanaName := testerAdd.Local.Name
	grafanaURL := testerAdd.Local.HTTPURL

	grafanaValues := `{
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
			"hosts": [` + fmt.Sprintf("%q", grafanaName) + `],
			"ingressClassName": ` + fmt.Sprintf("%q", tester.IngressClassName) + `
		},
		"env": {
			"GF_SERVER_ROOT_URL": ` + fmt.Sprintf("\"%s/grafana\"", grafanaURL) + `,
			"GF_SERVER_SERVE_FROM_SUB_PATH": "true"
		},
		"adminPassword": ` + fmt.Sprintf("%q", adminPassword) + `
	}`

	fmt.Println(grafanaValues)
	var grafanaValuesMap map[string]interface{}
	err = json.Unmarshal([]byte(grafanaValues), &grafanaValuesMap)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("grafanaMap:\n%+v\n", grafanaValuesMap)

	// TODO: why do we download the grafana chart and have the others locally?
	err = installChart(kubePath, filepath.Join(baseDir, "charts", "grafana"), "grafana", "tester", grafanaValuesMap)
	if err != nil {
		fmt.Println(err)
	}
}

// getAppAddressFor returns local cluster address data, public cluster address data and an error
func getAppAddressFor(cluster terraform.Cluster) (clusterAddresses, error) {
	add := cluster.AppAddresses

	addresses := clusterAddresses{}

	// addresses meant to be resolved from the machine running Terraform
	// use tunnel if available, otherwise public, otherwise go through the load balancer
	localNetworkName := add.Tunnel.Name
	if len(localNetworkName) == 0 {
		localNetworkName = add.Public.Name
		if len(localNetworkName) == 0 {
			// TODO: retrieve address from the LoadBalancer
			return addresses, fmt.Errorf("getAppAddressFor: cannot find cluster local name")
		}
	}
	localNetworkHTTPPort := add.Tunnel.HTTPPort
	if localNetworkHTTPPort == 0 {
		localNetworkHTTPPort = add.Public.HTTPPort
		if localNetworkHTTPPort == 0 {
			localNetworkHTTPPort = 80
		}
	}
	localNetworkHTTPSPort := add.Tunnel.HTTPSPort
	if localNetworkHTTPSPort == 0 {
		localNetworkHTTPSPort = add.Public.HTTPSPort
		if localNetworkHTTPSPort == 0 {
			localNetworkHTTPSPort = 443
		}
	}

	// addresses meant to be resolved from the network running clusters
	// use public if available, otherwise private if available, otherwise go through the load balancer
	clusterNetworkName := add.Public.Name
	if len(clusterNetworkName) == 0 {
		clusterNetworkName = add.Private.Name
		if len(clusterNetworkName) == 0 {
			// TODO: retrieve address from the LoadBalancer
			return addresses, fmt.Errorf("getAppAddressFor: cannot find cluster network name")
		}
	}
	clusterNetworkHTTPPort := add.Public.HTTPPort
	if clusterNetworkHTTPPort == 0 {
		clusterNetworkHTTPPort = add.Private.HTTPPort
		if clusterNetworkHTTPPort == 0 {
			clusterNetworkHTTPPort = 80
		}
	}
	clusterNetworkHTTPSPort := add.Public.HTTPSPort
	if clusterNetworkHTTPSPort == 0 {
		clusterNetworkHTTPSPort = add.Private.HTTPSPort
		if clusterNetworkHTTPSPort == 0 {
			clusterNetworkHTTPSPort = 443
		}
	}

	addresses.Local.Name = localNetworkName
	addresses.Local.HTTPURL = fmt.Sprintf("http://%s:%d", localNetworkName, localNetworkHTTPPort)
	addresses.Local.HTTPSURL = fmt.Sprintf("https://%s:%d", localNetworkName, localNetworkHTTPSPort)

	addresses.Public.Name = clusterNetworkName
	addresses.Public.HTTPURL = fmt.Sprintf("http://%s:%d", clusterNetworkName, clusterNetworkHTTPPort)
	addresses.Public.HTTPSURL = fmt.Sprintf("https://%s:%d", clusterNetworkName, clusterNetworkHTTPSPort)

	return addresses, nil
}

// installChart updates the chart if it is already installed otherwise installs it
func installChart(kubeconfig, chartPath, releaseName, namespace string, vals map[string]interface{}) error {
	settings := cli.New()
	settings.KubeConfig = kubeconfig

	actionConfig := new(action.Configuration)

	// var logger = log.Printf
	var logger = func(format string, v ...interface{}) {}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logger); err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	// first check if the chart is already installed
	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	if _, err := histClient.Run(releaseName); err == driver.ErrReleaseNotFound {
		if err == driver.ErrReleaseNotFound {
			installAction := action.NewInstall(actionConfig)
			installAction.CreateNamespace = true
			installAction.ReleaseName = releaseName
			installAction.Namespace = namespace

			release, err := installAction.Run(chart, vals)
			if err != nil {
				return err
			}

			log.Printf("Helm chart %s installed successfully: %s/%s", chartPath, release.Namespace, release.Name)
			return nil
		}
		return err
	}

	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Install = true
	release, err := upgradeAction.Run(releaseName, chart, vals)
	if err != nil {
		return err
	}

	log.Printf("Helm chart %s upgraded successfully: %s/%s", chartPath, release.Namespace, release.Name)
	return nil
}

func terraformDir() string {
	defaultDir := os.Getenv("TERRAFORM_WORK_DIR")
	if len(defaultDir) == 0 {
		defaultDir = filepath.Join("..", "terraform", "main", "k3d")
	}
	return defaultDir
}

func terraformVersionPrint(tf *terraform.Terraform) error {
	ver, providers, err := tf.Version()
	if err != nil {
		return err
	}

	log.Printf("Terraform version: %s", ver)
	log.Printf("provider list:")
	for prov, ver := range providers {
		log.Printf("- %s (%s)", prov, ver)
	}
	return nil
}
