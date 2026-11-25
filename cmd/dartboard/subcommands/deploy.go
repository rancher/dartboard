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
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/helm"
	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/tofu"
	"github.com/rancher/shepherd/clients/rancher"
	shepherdclusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/session"
	cli "github.com/urfave/cli/v3"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/rancher/dartboard/internal/actions"
)

type chart struct {
	name      string
	namespace string
	path      string
}

const (
	// Chart names
	chartNameK6Files              = "k6-files"
	chartNameMimir                = "mimir"
	chartNameGrafanaDashboards    = "grafana-dashboards"
	chartNameGrafana              = "grafana"
	chartNameCertManager          = "cert-manager"
	chartNameRancher              = "rancher"
	chartNameRancherIngress       = "rancher-ingress"
	chartNameRancherMonitoringCRD = "rancher-monitoring-crd"
	chartNameRancherMonitoring    = "rancher-monitoring"
	chartNameKubePrometheusStack  = "kube-prometheus-stack"
	chartNameMonitoringDashboards = "rancher-monitoring-dashboards"
	chartNameCgroupsExporter      = "cgroups-exporter"

	// Chart namespaces
	nsDefault                = "default"
	nsTester                 = "tester"
	nsCertManager            = "cert-manager"
	nsCattleSystem           = "cattle-system"
	nsCattleMonitoringSystem = "cattle-monitoring-system"
)

func Deploy(ctx context.Context, cli *cli.Command) error {
	// Tofu
	tf, r, err := prepare(ctx, cli)
	if err != nil {
		return err
	}

	if err = applyTofuChanges(cli, tf); err != nil {
		return err
	}

	clusters, custom_clusters, err := tf.ParseOutputs()
	if err != nil {
		return err
	}

	// Helm charts
	tester := clusters["tester"]

	testerDeployed := len(tester.Kubeconfig) > 0
	if testerDeployed && !cli.Bool(ArgSkipCharts) {
		if err = installTesterCharts(tester, r); err != nil {
			return err
		}
	}

	// The Mimir remote_write endpoint is resolved once, from the tester
	// cluster's own address (reachable from any other cluster's network),
	// and reused for both upstream and downstream monitoring installs. When
	// there is no tester cluster, this is left empty and remote_write is
	// simply disabled in the monitoring chart values.
	var testerMimirURL string

	if testerDeployed {
		testerAdd, err := getAppAddressFor(tester)
		if err != nil {
			return err
		}

		testerMimirURL = testerAdd.Public.HTTPURL + "/mimir/api/v1/push"
	}

	upstream := clusters["upstream"]

	upstreamDeployed := upstream.Kubeconfig != ""
	if !upstreamDeployed && (len(custom_clusters) > 0 || len(r.ClusterTemplates) > 0) {
		return fmt.Errorf("an upstream Rancher cluster is required for custom or provisioned downstream clusters")
	}
	// upstreamPreExisting is true only when the dart declares a top-level,
	// pre-existing `upstream_cluster` passthrough (see injectPreExistingUpstream).
	// It must NOT be used to decide whether upstream charts should be
	// installed: a dart that lets OpenTofu create a fresh upstream cluster via
	// `tofu_variables.upstream_cluster` never sets this field, yet still has a
	// real, deployed upstream cluster that needs Rancher and monitoring.
	upstreamPreExisting := r.UpstreamCluster != nil
	if !upstreamDeployed && upstreamPreExisting {
		return fmt.Errorf("upstream cluster output is empty")
	}
	// Keep the resolved output available to registration helpers regardless of
	// whether OpenTofu created it or passed through an existing cluster.
	r.UpstreamCluster = &upstream
	rancherVersion := r.ChartVariables.RancherVersion

	rancherImageTag := "v" + rancherVersion
	if upstreamDeployed && r.ChartVariables.RancherImageTagOverride != "" {
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

	if !cli.Bool(ArgSkipCharts) && upstreamDeployed {
		if err = installUpstreamCharts(r, rancherImageTag, &upstream, testerMimirURL); err != nil {
			return err
		}
	}

	// Install rancher-monitoring directly (via Helm) on every downstream
	// cluster's kubeconfig. This is independent of whether an upstream/Rancher
	// cluster exists at all: downstream clusters already have their own
	// kubeconfig from OpenTofu's output, so monitoring can be installed on
	// them standalone, without importing them into Rancher first.
	if !cli.Bool(ArgSkipCharts) && r.ChartVariables.DownstreamRancherMonitoring {
		if err = installDownstreamMonitoring(r, clusters, testerMimirURL); err != nil {
			return err
		}
	}

	// Setup rancher client
	if upstreamDeployed {
		rancherClient, err := setupRancher(r, upstream, clusters, custom_clusters)
		if err != nil {
			return err
		}

		if !cli.Bool(ArgSkipCharts) && r.ChartVariables.DownstreamRancherMonitoring {
			if err = installRancherDownstreamMonitoring(r, upstream, rancherClient, custom_clusters, testerMimirURL); err != nil {
				return err
			}
		}
	}

	return GetAccess(ctx, cli)
}

// setupRancher connects to the upstream Rancher and imports, registers or
// provisions the downstream clusters through it.
func setupRancher(r *dart.Dart, upstream tofu.Cluster, clusters map[string]tofu.Cluster, customClusters []tofu.CustomCluster) (*rancher.Client, error) {
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return nil, err
	}

	rancherSession := session.NewSession()
	rancherSession.CleanupEnabled = false

	logrus.Info("Setting up Rancher Client's Config")

	rancherHost := strings.Split(upstreamAdd.Local.HTTPSURL, "://")[1]
	rancherConfig := actions.NewRancherConfig(rancherHost, "", r.ChartVariables.AdminPassword, true)

	logrus.Info("Setting up Rancher Client")

	rancherClient, err := actions.SetupRancherClient(&rancherConfig, r.ChartVariables.AdminPassword, rancherSession)
	if err != nil {
		return nil, err
	}

	if len(clusters) > 0 {
		if err = importDownstreamClusters(r, clusters, rancherClient, &rancherConfig); err != nil {
			return nil, err
		}
	}

	logrus.Debugf("\nBEFORE CUSTOM CLUSTER LOGIC\n")

	if len(customClusters) > 0 {
		logrus.Debugf("\nIN CUSTOM CLUSTER LOGIC\n")

		if err := actions.RegisterCustomClusters(r, customClusters, rancherClient, &rancherConfig); err != nil {
			return nil, err
		}
	}

	if len(r.ClusterTemplates) > 0 {
		if err = setupHarvesterAndProvision(r, rancherClient); err != nil {
			return nil, err
		}

		logrus.Info("Provisioning Downstream Clusters")

		if err = actions.ProvisionDownstreamClusters(r, r.ClusterTemplates, rancherClient); err != nil {
			return nil, err
		}
	}

	return rancherClient, nil
}

// rancherDownstreamClusterNames lists the Rancher names of the custom clusters
// (mirroring RegisterCustomClustersInBatches naming) followed by the provisioned
// clusters' real names as recorded in the cluster state.
func rancherDownstreamClusterNames(customClusters []tofu.CustomCluster, provisioned []string) []string {
	names := []string{}

	for _, c := range customClusters {
		for i := range c.ClusterCount {
			names = append(names, fmt.Sprintf("%s-%d", c.Name, i))
		}
	}

	return append(names, provisioned...)
}

// installRancherDownstreamMonitoring installs rancher-monitoring on custom and
// provisioned clusters, using kubeconfigs generated through Rancher. Both
// creation paths already wait for the clusters to be ready/active.
func installRancherDownstreamMonitoring(r *dart.Dart, upstream tofu.Cluster, rancherClient *rancher.Client, customClusters []tofu.CustomCluster, mimirURL string) error {
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return err
	}

	statuses, err := actions.LoadClusterState(fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, actions.ClustersStateFile))
	if err != nil {
		return err
	}

	for name, cs := range statuses {
		if cs != nil && cs.Provisioned && cs.RancherName == "" {
			logrus.Warnf("Cluster %q was provisioned before its Rancher name was recorded: no monitoring will be installed on it", name)
		}
	}

	names := rancherDownstreamClusterNames(customClusters, actions.ProvisionedRancherNames(statuses))

	for _, name := range names {
		id, err := shepherdclusters.GetClusterIDByName(rancherClient, name)
		if err == nil && id == "" {
			err = fmt.Errorf("cluster %s not found in Rancher", name)
		}

		if err != nil {
			return fmt.Errorf("cluster %s: %w", name, err)
		}

		if err = installMonitoringViaRancher(r, rancherClient, id, name, mimirURL, upstreamAdd.Public.HTTPSURL, upstreamAdd.Local.HTTPSURL); err != nil {
			return fmt.Errorf("cluster %s: %w", name, err)
		}
	}

	return nil
}

// installMonitoringViaRancher generates the cluster's kubeconfig through Rancher
// into a temporary file and installs rancher-monitoring with it. Rancher embeds
// its public URL in the kubeconfig, so it is rewritten to the locally reachable one.
func installMonitoringViaRancher(r *dart.Dart, rancherClient *rancher.Client, id, name, mimirURL, publicURL, localURL string) error {
	cluster, err := rancherClient.Management.Cluster.ByID(id)
	if err != nil {
		return fmt.Errorf("getting cluster by ID %s: %w", id, err)
	}

	output, err := rancherClient.Management.Cluster.ActionGenerateKubeconfig(cluster)
	if err != nil {
		return fmt.Errorf("generating kubeconfig for cluster %s: %w", id, err)
	}

	localKubeconfig, err := localizeKubeconfig(output.Config, publicURL, localURL)
	if err != nil {
		return fmt.Errorf("cluster %s: %w", id, err)
	}

	path, err := writeValuesFile(localKubeconfig)
	if path != "" {
		defer os.Remove(path)
	}

	if err != nil {
		return err
	}

	logrus.Infof("Installing rancher-monitoring on Rancher cluster %q (%s)", name, id)

	return chartInstallRancherMonitoring(r, &tofu.Cluster{Name: name, Kubeconfig: path}, id, name, mimirURL)
}

// localizeKubeconfig points every cluster served at publicURL to localURL. When
// the hostnames differ, TLSServerName keeps verification against the public
// host, which is the one the served certificate is issued for.
func localizeKubeconfig(cfg, publicURL, localURL string) (string, error) {
	if publicURL == localURL {
		return cfg, nil
	}

	kc, err := clientcmd.Load([]byte(cfg))
	if err != nil {
		return "", fmt.Errorf("parsing kubeconfig: %w", err)
	}

	pub, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("parsing public URL: %w", err)
	}

	loc, err := url.Parse(localURL)
	if err != nil {
		return "", fmt.Errorf("parsing local URL: %w", err)
	}

	matched := false

	for _, c := range kc.Clusters {
		if !strings.HasPrefix(c.Server, publicURL) {
			continue
		}

		matched = true
		c.Server = localURL + strings.TrimPrefix(c.Server, publicURL)

		if pub.Hostname() != loc.Hostname() {
			c.TLSServerName = pub.Hostname()
		}
	}

	if !matched {
		return "", fmt.Errorf("no cluster in kubeconfig uses Rancher URL %s", publicURL)
	}

	out, err := clientcmd.Write(*kc)
	if err != nil {
		return "", fmt.Errorf("writing kubeconfig: %w", err)
	}

	return string(out), nil
}

// applyTofuChanges applies or outputs Terraform/Tofu changes
func applyTofuChanges(cli *cli.Command, tf *tofu.Tofu) error {
	skipRefresh := cli.Bool(ArgSkipRefresh)

	if !cli.Bool(ArgSkipApply) {
		if err := tf.PrintVersion(); err != nil {
			return err
		}

		return tf.Apply(skipRefresh)
	}

	return tf.Output(nil, false)
}

// installTesterCharts installs required charts on the tester cluster
func installTesterCharts(tester tofu.Cluster, r *dart.Dart) error {
	if err := chartInstall(tester.Kubeconfig, chart{chartNameK6Files, nsTester, chartNameK6Files}, nil); err != nil {
		return err
	}

	if err := chartInstall(tester.Kubeconfig, chart{chartNameMimir, nsTester, chartNameMimir}, nil); err != nil {
		return err
	}

	if err := chartInstall(tester.Kubeconfig, chart{chartNameGrafanaDashboards, nsTester, chartNameGrafanaDashboards}, nil); err != nil {
		return err
	}

	return chartInstallGrafana(r, &tester)
}

// installUpstreamCharts installs Rancher and related charts on the upstream cluster
func installUpstreamCharts(r *dart.Dart, rancherImageTag string, upstream *tofu.Cluster, mimirURL string) error {
	if err := chartInstallCertManager(r, upstream); err != nil {
		return err
	}

	if err := chartInstallRancher(r, rancherImageTag, upstream); err != nil {
		return err
	}

	if err := chartInstallRancherIngress(upstream); err != nil {
		return err
	}

	if err := chartInstallCgroupsExporter(upstream); err != nil {
		return err
	}

	// Wait for Rancher deployments to be complete, or subsequent steps may fail
	if err := kubectl.WaitRancher(upstream.Kubeconfig); err != nil {
		return err
	}

	// "local" is the Rancher convention for the management (upstream) cluster's
	// own identity in the monitoring stack.
	if err := chartInstallRancherMonitoring(r, upstream, "local", "local", mimirURL); err != nil {
		return err
	}

	if err := updateMonitoringProject(upstream); err != nil {
		return err
	}

	return nil
}

// installDownstreamMonitoring installs rancher-monitoring directly, via Helm,
// on every downstream cluster's kubeconfig. Unlike upstream monitoring, this
// does not require Rancher or an upstream cluster to exist at all: it is
// gated solely on chart_variables.downstream_rancher_monitoring.
func installDownstreamMonitoring(r *dart.Dart, clusters map[string]tofu.Cluster, mimirURL string) error {
	downstreamClusters := []tofu.Cluster{}

	for k, v := range clusters {
		if strings.HasPrefix(k, "downstream") {
			v.Name = k
			downstreamClusters = append(downstreamClusters, v)
		}
	}

	SortItemsNaturally(downstreamClusters, func(c tofu.Cluster) string { return c.Name })

	for i := range downstreamClusters {
		cluster := downstreamClusters[i]

		if cluster.Kubeconfig == "" {
			logrus.Warnf("Downstream cluster %q has no kubeconfig output, skipping monitoring install", cluster.Name)

			continue
		}

		logrus.Infof("Installing rancher-monitoring on downstream cluster %q", cluster.Name)

		// Use the downstream cluster's own name as its monitoring identity,
		// since there is no Rancher "local" management cluster convention to
		// piggyback on, and each downstream cluster's metrics should be
		// distinguishable from one another (and from upstream) in Mimir/Grafana.
		if err := chartInstallRancherMonitoring(r, &cluster, cluster.Name, cluster.Name, mimirURL); err != nil {
			return fmt.Errorf("downstream cluster %s: %w", cluster.Name, err)
		}
	}

	return nil
}

// importDownstreamClusters imports all downstream clusters into Rancher
func importDownstreamClusters(r *dart.Dart, clusters map[string]tofu.Cluster, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
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
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	logrus.Infof("Import Clusters: %s", string(jsonBytes))

	logrus.Info("Importing Downstream Clusters")

	return actions.ImportDownstreamClusters(r, downstreamClusters, rancherClient, rancherConfig)
}

// setupHarvesterAndProvision sets up Harvester client if needed
func setupHarvesterAndProvision(r *dart.Dart, rancherClient *rancher.Client) error {
	if !strings.Contains(r.TofuMainDirectory, "harvester") {
		return nil
	}

	logrus.Info("Parsing Harvester's Kubeconfig")

	var (
		kubeconfig *actions.Kubeconfig
		err        error
	)

	if len(r.TofuVariables["kubeconfig"].(string)) > 0 {
		kubeconfig, err = actions.ParseKubeconfig(r.TofuVariables["kubeconfig"].(string))
		if err != nil {
			return fmt.Errorf("error while parsing kubeconfig at %v: %w", r.TofuVariables["kubeconfig"].(string), err)
		}
	}

	logrus.Info("Setting up Harvester Client's Config")

	harvesterHost := strings.Split(kubeconfig.Clusters[0].Cluster.Server, "://")[1]
	harvesterConfig := actions.NewHarvesterConfig(harvesterHost, kubeconfig.Users[0].User.Token, "", true)

	logrus.Info("Setting up Harvester Client")

	harvesterClient, err := actions.NewHarvesterImportClient(rancherClient, &harvesterConfig)
	if err != nil {
		return fmt.Errorf("error while setting up HarvesterImportClient with config %v: %w", harvesterConfig, err)
	}

	logrus.Info("Importing Harvester Cluster into Rancher for provisioning")

	return harvesterClient.ImportCluster()
}

func chartInstall(kubeConf string, chart chart, vals map[string]any, extraArgs ...string) error {
	var err error

	name := chart.name
	namespace := chart.namespace
	path := chart.path

	// Pull from local `charts/` dir if not using remote chart
	if !strings.HasPrefix(path, "http") && !strings.HasPrefix(path, "oci") {
		path = filepath.Join("charts", path)
	}

	logrus.Infof("Installing chart %q (%s)", namespace+"/"+name, path)

	if err = helm.Install(kubeConf, path, name, namespace, vals, extraArgs...); err != nil {
		return fmt.Errorf("chart %s: %w", name, err)
	}

	return nil
}

func chartInstallGrafana(r *dart.Dart, cluster *tofu.Cluster) error {
	chartGrafana := chart{
		name:      chartNameGrafana,
		namespace: nsTester,
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
		name:      chartNameCertManager,
		namespace: nsCertManager,
		path:      fmt.Sprintf("https://charts.jetstack.io/charts/cert-manager-v%s.tgz", r.ChartVariables.CertManagerVersion),
	}
	chartValues := map[string]any{"installCRDs": true}

	var extraArgs []string

	if certManagerSupportsOCI(r.ChartVariables.CertManagerVersion) {
		chartCertManager.path = "oci://quay.io/jetstack/charts/cert-manager"

		extraArgs = []string{"--version=v" + r.ChartVariables.CertManagerVersion}
		if certManagerSupportsCRDsValue(r.ChartVariables.CertManagerVersion) {
			chartValues = map[string]any{"crds": map[string]any{"enabled": true}}
		}
	}

	return chartInstall(cluster.Kubeconfig, chartCertManager, chartValues, extraArgs...)
}

func certManagerSupportsOCI(version string) bool {
	major, minor, ok := certManagerMajorMinor(version)
	return ok && (major > 1 || (major == 1 && minor >= 12))
}

func certManagerSupportsCRDsValue(version string) bool {
	major, minor, ok := certManagerMajorMinor(version)
	return ok && (major > 1 || (major == 1 && minor >= 15))
}

func certManagerMajorMinor(version string) (int, int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0, false
	}

	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])

	return major, minor, majorErr == nil && minorErr == nil
}

func chartInstallRancher(r *dart.Dart, rancherImageTag string, cluster *tofu.Cluster) error {
	var rancherRepo string

	if r.ChartVariables.RancherChartRepoOverride != "" {
		rancherRepo = r.ChartVariables.RancherChartRepoOverride
	} else {
		baseRepo := "https://releases.rancher.com/server-charts/"

		// otherwise, if one of "alpha", or "latest"
		if strings.Contains(r.ChartVariables.RancherVersion, "alpha") {
			rancherRepo = baseRepo + "alpha/rancher-"
		} else {
			rancherRepo = baseRepo + "latest/rancher-"
		}

		// "prime"
		if r.ChartVariables.ForcePrimeRegistry {
			rancherRepo = "https://charts.rancher.com/server-charts/prime/rancher-"
		}
	}

	chartRancher := chart{
		name:      chartNameRancher,
		namespace: nsCattleSystem,
		path:      rancherRepo + r.ChartVariables.RancherVersion + ".tgz",
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

	var extraArgs []string

	if r.ChartVariables.RancherValues != "" {
		p, err := writeValuesFile(r.ChartVariables.RancherValues)
		if err != nil {
			return fmt.Errorf("writing extra values file: %w", err)
		}
		defer os.Remove(p)

		extraArgs = append(extraArgs, "-f", p)
	}

	logrus.Debug("RANCHER CHART VALS:")

	for key, value := range chartVals {
		logrus.Debugf("\t%s = %v", key, value)
	}

	return chartInstall(cluster.Kubeconfig, chartRancher, chartVals, extraArgs...)
}

func writeValuesFile(content string) (string, error) {
	p, err := os.CreateTemp("", "values-*.yaml")
	if err != nil {
		return "", err
	}

	_, err = io.WriteString(p, content)
	if cerr := p.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		os.Remove(p.Name())
		return "", err
	}

	return p.Name(), nil
}

func chartInstallRancherIngress(cluster *tofu.Cluster) error {
	chartRancherIngress := chart{
		name:      chartNameRancherIngress,
		namespace: nsDefault,
		path:      chartNameRancherIngress,
	}

	clusterAdd, err := getAppAddressFor(*cluster)
	if err != nil {
		return fmt.Errorf("chart %s: %w", chartRancherIngress.name, err)
	}

	var sans []string
	// Add the local address as a SAN if it's different from the public address, which can occur when using port forwarding or with certain k3d configurations.
	// This ensures that the TLS certificate will be valid for both the public and local addresses.
	if len(clusterAdd.Local.Name) > 0 && clusterAdd.Local.Name != clusterAdd.Public.Name {
		sans = append(sans, clusterAdd.Local.Name)
	}

	// If there are no additional SANs to configure, uninstall any existing
	// rancher-ingress release to avoid leaving behind an invalid Ingress
	// manifest, then return without installing.
	if len(sans) == 0 {
		logrus.Infof("No additional SANs needed, uninstalling chart %q if present", chartRancherIngress.namespace+"/"+chartRancherIngress.name)

		if err := helm.UninstallIfPresent(cluster.Kubeconfig, chartRancherIngress.name, chartRancherIngress.namespace); err != nil {
			return fmt.Errorf("chart %s: uninstall: %w", chartRancherIngress.name, err)
		}

		return nil
	}

	chartVals := map[string]any{
		"sans":             sans,
		"ingressClassName": cluster.IngressClassName,
	}

	return chartInstall(cluster.Kubeconfig, chartRancherIngress, chartVals)
}

// chartInstallRancherMonitoring installs the rancher-monitoring-crd and
// rancher-monitoring charts on the given cluster's kubeconfig. clusterID and
// clusterName identify the cluster in the monitoring stack's global.cattle
// values: pass "local" for the upstream/management cluster (matching
// Rancher's own convention), or the cluster's own name for a standalone
// downstream cluster. mimirURL is the (optional) Mimir remote_write push
// endpoint; pass an empty string to disable remote_write, e.g. when no tester
// cluster is deployed.
func chartInstallRancherMonitoring(r *dart.Dart, cluster *tofu.Cluster, clusterID, clusterName, mimirURL string) error {
	if useLegacyRancherMonitoring(r) {
		major, minor := parseRancherMajorMinor(r.ChartVariables.RancherVersion)
		rancherMinorVersion := fmt.Sprintf("%d.%d", major, minor)

		return chartInstallLegacyRancherMonitoring(r, cluster, rancherMinorVersion, clusterID, clusterName, mimirURL)
	}

	return chartInstallKubePrometheusStack(r, cluster, clusterID, clusterName, mimirURL)
}

// useLegacyRancherMonitoring reports whether the legacy rancher-monitoring
// chart (Rancher < 2.15) is installed instead of kube-prometheus-stack.
func useLegacyRancherMonitoring(r *dart.Dart) bool {
	major, minor := parseRancherMajorMinor(r.ChartVariables.RancherVersion)

	return !r.ChartVariables.ForceKubePrometheusStack && (major < 2 || (major == 2 && minor < 15))
}

// parseRancherMajorMinor extracts the major and minor numbers from a version
// such as "2.9.1" or "v2.15.0-alpha1". Missing or non-numeric parts are 0.
func parseRancherMajorMinor(version string) (major, minor int) {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	major, _ = strconv.Atoi(parts[0])

	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}

	return major, minor
}

func chartInstallLegacyRancherMonitoring(r *dart.Dart, cluster *tofu.Cluster, rancherMinorVersion, clusterID, clusterName, mimirURL string) error {
	const chartPrefix = "https://github.com/rancher/charts/raw/release-v"

	chartPath := fmt.Sprintf("%s%s", chartPrefix, rancherMinorVersion)

	if len(r.ChartVariables.RancherAppsRepoOverride) > 0 {
		chartPath = r.ChartVariables.RancherAppsRepoOverride
	}

	chartRancherMonitoringCRDRoute := "assets/rancher-monitoring-crd/rancher-monitoring-crd"
	chartRancherMonitoringCRD := chart{
		name:      chartNameRancherMonitoringCRD,
		namespace: nsCattleMonitoringSystem,
		path:      fmt.Sprintf("%s/%s-%s.tgz", chartPath, chartRancherMonitoringCRDRoute, r.ChartVariables.RancherMonitoringVersion),
	}

	chartVals := map[string]any{
		"global":                getMonitoringGlobalVals(clusterID, clusterName, !r.ChartVariables.EnableIPv6),
		"systemDefaultRegistry": "",
	}

	err := chartInstall(cluster.Kubeconfig, chartRancherMonitoringCRD, chartVals)
	if err != nil {
		return err
	}

	chartRancherMonitoringRoute := "assets/rancher-monitoring/rancher-monitoring"
	chartRancherMonitoring := chart{
		name:      chartNameRancherMonitoring,
		namespace: nsCattleMonitoringSystem,
		path:      fmt.Sprintf("%s/%s-%s.tgz", chartPath, chartRancherMonitoringRoute, r.ChartVariables.RancherMonitoringVersion),
	}

	chartVals = getRancherMonitoringValsJSON(cluster.ReserveNodeForMonitoring, mimirURL, clusterID, clusterName, !r.ChartVariables.EnableIPv6)

	return chartInstall(cluster.Kubeconfig, chartRancherMonitoring, chartVals)
}

func chartInstallKubePrometheusStack(r *dart.Dart, cluster *tofu.Cluster, clusterID, clusterName, mimirURL string) error {
	if err := helm.UninstallIfPresent(cluster.Kubeconfig, chartNameRancherMonitoring, nsCattleMonitoringSystem); err != nil {
		return fmt.Errorf("uninstall legacy %s: %w", chartNameRancherMonitoring, err)
	}

	kubePrometheusStack := chart{
		name:      chartNameKubePrometheusStack,
		namespace: nsCattleMonitoringSystem,
		path:      "oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack",
	}

	distro, err := kubectl.GetK8sDistro(cluster.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes distro: %w", err)
	}

	stackValues := getKubePrometheusStackVals(cluster.ReserveNodeForMonitoring, strings.Contains(distro, "k3s"), r.ChartVariables.EnableIPv6, r.ChartVariables.GrafanaAdminPassword, mimirURL, clusterID, clusterName)
	if err := chartInstall(cluster.Kubeconfig, kubePrometheusStack, stackValues, "--version="+r.ChartVariables.KubePrometheusStackVersion); err != nil {
		return err
	}

	chartPath := "https://github.com/rancher/charts/raw/refs/heads/release-v2.15"
	if r.ChartVariables.RancherAppsRepoOverride != "" {
		chartPath = r.ChartVariables.RancherAppsRepoOverride
	}

	dashboards := chart{
		name:      chartNameMonitoringDashboards,
		namespace: nsCattleMonitoringSystem,
		path:      monitoringDashboardsChartPath(chartPath, r.ChartVariables.MonitoringDashboardsVersion),
	}

	dashboardValues := getMonitoringDashboardsValues(cluster.ReserveNodeForMonitoring, r.ChartVariables.EnableIPv6, clusterID, clusterName)
	if err := chartInstall(cluster.Kubeconfig, dashboards, dashboardValues); err != nil {
		return err
	}

	fmt.Printf("Grafana admin password (admin): %s\n", kubePrometheusStackGrafanaPasswordCommand())

	return nil
}

func kubePrometheusStackGrafanaPasswordCommand() string {
	return "kubectl -n cattle-monitoring-system get secret kube-prometheus-stack-grafana -o jsonpath='{.data.admin-password}' | openssl base64 -d -A; echo"
}

func getMonitoringDashboardsValues(reserveNodeForMonitoring, enableIPv6 bool, clusterID, clusterName string) map[string]any {
	return map[string]any{
		"global":            map[string]any{"clusterId": clusterID, "clusterName": clusterName, "systemDefaultRegistry": "", "disableProxyIPv6": !enableIPv6, "cattle": map[string]any{"clusterId": clusterID, "clusterName": clusterName, "systemDefaultRegistry": ""}},
		"monitoringProxy":   monitoringSchedulingValues(reserveNodeForMonitoring),
		"alertmanagerProxy": map[string]any{"enabled": false},
	}
}

func monitoringDashboardsChartPath(chartPath, version string) string {
	return fmt.Sprintf("%s/assets/rancher-monitoring-dashboards/rancher-monitoring-dashboards-%s.tgz", chartPath, version)
}

// getMonitoringGlobalVals builds the `global` values shared by the
// rancher-monitoring-crd and rancher-monitoring charts.
//
// disableProxyIPv6 maps to the chart's global.disableProxyIPv6, which controls
// whether the nginx sidecars fronting Grafana (:8080) and Prometheus (:8081)
// also bind `[::]`. On a node with IPv6 disabled in the kernel that bind fails
// with EAFNOSUPPORT and nginx exits, so the sidecar (and therefore the pod)
// never becomes ready.
func getMonitoringGlobalVals(clusterID, clusterName string, disableProxyIPv6 bool) map[string]any {
	return map[string]any{
		"cattle": map[string]any{
			"clusterId":             clusterID,
			"clusterName":           clusterName,
			"systemDefaultRegistry": "",
		},
		"disableProxyIPv6": disableProxyIPv6,
	}
}

func chartInstallCgroupsExporter(cluster *tofu.Cluster) error {
	var b strings.Builder
	if err := kubectl.Exec(cluster.Kubeconfig, &b, "get", "nodes", "-o", "jsonpath={.items[*].status.nodeInfo.osImage}"); err != nil {
		return fmt.Errorf("failed to get node os images: %w", err)
	}

	vals := map[string]any{}

	osImages := strings.ToLower(b.String())
	if strings.Contains(osImages, "suse linux micro") || strings.Contains(osImages, "opensuse leap micro") {
		vals["mountHostSys"] = false

		logrus.Infof("Disabling mountHostSys for cgroups-exporter due to detected OS: %s", b.String())
	}

	return chartInstall(cluster.Kubeconfig, chart{chartNameCgroupsExporter, nsCattleMonitoringSystem, chartNameCgroupsExporter}, vals)
}

func getRancherMonitoringValsJSON(reserveNodeForMonitoring bool, mimirURL, clusterID, clusterName string, disableProxyIPv6 bool) map[string]any {
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

				"additionalScrapeConfigs": getCgroupsScrapeConfig(),

				"remoteWrite": remoteWrite,

				"externalLabels": monitoringExternalLabels(clusterID, clusterName),
			},
		},
		"prometheus-adapter": monitoringRestrictions,
		"prometheus-node-exporter": map[string]any{
			"nodeSelector": map[string]any{"kubernetes.io/os": "linux"},
			"tolerations":  []any{map[string]any{"operator": "Exists"}},
		},
		"kube-state-metrics":    monitoringRestrictions,
		"prometheusOperator":    monitoringRestrictions,
		"global":                getMonitoringGlobalVals(clusterID, clusterName, disableProxyIPv6),
		"systemDefaultRegistry": "",
	}
}

func getKubePrometheusStackVals(reserveNodeForMonitoring, k3sServer, enableIPv6 bool, grafanaAdminPassword, mimirURL, clusterID, clusterName string) map[string]any {
	nodeSelector := map[string]any{}
	tolerations := []any{}

	if reserveNodeForMonitoring {
		nodeSelector["monitoring"] = "true"

		tolerations = append(tolerations, map[string]any{"key": "monitoring", "operator": "Exists", "effect": "NoSchedule"})
	}

	remoteWrite := []any{}
	if mimirURL != "" {
		remoteWrite = append(remoteWrite, map[string]any{
			"url": mimirURL,
			"writeRelabelConfigs": []any{map[string]any{
				"sourceLabels": []any{"__name__"},
				"regex":        "(node_namespace_pod_container|node_cpu|node_load|node_memory|node_network_receive_bytes_total|container_network_receive_bytes_total|cgroups_).*",
				"action":       "keep",
			}},
		})
	}

	serviceIPDualStack := map[string]any{
		"enabled":        enableIPv6,
		"ipFamilies":     []any{"IPv4"},
		"ipFamilyPolicy": "SingleStack",
	}
	if enableIPv6 {
		serviceIPDualStack["ipFamilies"] = []any{"IPv6", "IPv4"}
		serviceIPDualStack["ipFamilyPolicy"] = "PreferDualStack"
	}

	values := map[string]any{
		"alertmanager": map[string]any{"enabled": false, "service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"grafana": map[string]any{
			"nodeSelector": nodeSelector,
			"tolerations":  tolerations,
			"persistence":  map[string]any{"enabled": false},
			"grafana.ini": map[string]any{
				"security":       map[string]any{"allow_embedding": true},
				"auth":           map[string]any{"disable_login_form": false},
				"auth.anonymous": map[string]any{"enabled": true, "org_role": "Viewer"},
				"dashboards":     map[string]any{"default_home_dashboard_path": "/tmp/dashboards/rancher-default-home.json"},
				"users":          map[string]any{"auto_assign_org_role": "Viewer"},
			},
		},
		"prometheus": map[string]any{"prometheusSpec": map[string]any{
			"evaluationInterval": "1m",
			"scrapeInterval":     "1m",
			"nodeSelector":       nodeSelector,
			"tolerations":        tolerations,
			"resources":          map[string]any{"limits": map[string]any{"memory": "10000Mi"}},
			"retentionSize":      "50GiB",
			"serviceMonitorSelectorNilUsesHelmValues": false,
			"podMonitorSelectorNilUsesHelmValues":     false,
			"additionalScrapeConfigs":                 getCgroupsScrapeConfig(),
			"remoteWrite":                             remoteWrite,
			"externalLabels":                          monitoringExternalLabels(clusterID, clusterName),
		}, "service": map[string]any{"ipDualStack": serviceIPDualStack}, "servicePerReplica": map[string]any{"ipDualStack": serviceIPDualStack}, "thanosService": map[string]any{"ipDualStack": serviceIPDualStack}},
		"kubeControllerManager": map[string]any{"enabled": false, "service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"coreDns":               map[string]any{"service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"kubeDns":               map[string]any{"service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"kubeEtcd":              map[string]any{"enabled": false, "service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"kubeScheduler":         map[string]any{"enabled": false, "service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"kubeProxy":             map[string]any{"enabled": false, "service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"prometheusOperator": map[string]any{
			"nodeSelector": nodeSelector,
			"tolerations":  tolerations,
			"service":      map[string]any{"ipDualStack": serviceIPDualStack},
			"admissionWebhooks": map[string]any{
				"patch": map[string]any{"nodeSelector": nodeSelector, "tolerations": tolerations},
				"deployment": map[string]any{
					"nodeSelector": nodeSelector,
					"tolerations":  tolerations,
					"service":      map[string]any{"ipDualStack": serviceIPDualStack},
				},
			},
		},
		"prometheus-node-exporter": map[string]any{
			"hostRootFsMount": map[string]any{"enabled": false},
			"nodeSelector":    map[string]any{"kubernetes.io/os": "linux"},
			"tolerations":     []any{map[string]any{"operator": "Exists"}},
			"service":         map[string]any{"ipDualStack": serviceIPDualStack},
		},
		"kube-state-metrics": map[string]any{
			"nodeSelector": nodeSelector,
			"tolerations":  tolerations,
			"service":      map[string]any{"ipDualStack": serviceIPDualStack},
		},
		"thanosRuler": map[string]any{"service": map[string]any{"ipDualStack": serviceIPDualStack}},
		"global": map[string]any{
			"cattle": map[string]any{
				"clusterId":             clusterID,
				"clusterName":           clusterName,
				"systemDefaultRegistry": "",
			},
		},
		"systemDefaultRegistry": "",
		"k3sServer":             k3sServer,
	}
	if grafanaAdminPassword != "" {
		values["grafana"].(map[string]any)["adminPassword"] = grafanaAdminPassword
	}

	return values
}

// monitoringExternalLabels are attached to everything Prometheus remote-writes,
// so series from different clusters stay distinguishable in a shared Mimir.
// Both charts take them at prometheus.prometheusSpec.externalLabels.
func monitoringExternalLabels(clusterID, clusterName string) map[string]any {
	return map[string]any{"cluster_id": clusterID, "cluster_name": clusterName}
}

func monitoringSchedulingValues(reserveNodeForMonitoring bool) map[string]any {
	if !reserveNodeForMonitoring {
		return map[string]any{}
	}

	return map[string]any{
		"nodeSelector": map[string]any{"monitoring": "true"},
		"tolerations":  []any{map[string]any{"key": "monitoring", "operator": "Exists", "effect": "NoSchedule"}},
	}
}

func getCgroupsScrapeConfig() []any {
	return []any{map[string]any{
		"job_name":              "node-cgroups-exporter",
		"honor_labels":          false,
		"kubernetes_sd_configs": []any{map[string]any{"role": "node"}},
		"scheme":                "http",
		"relabel_configs": []any{
			map[string]any{"action": "labelmap", "regex": "__meta_kubernetes_node_label_(.+)"},
			map[string]any{"source_labels": []any{"__address__"}, "action": "replace", "target_label": "__address__", "regex": "([^:;]+):(\\d+)", "replacement": "${1}:9753"},
			map[string]any{"source_labels": []any{"__meta_kubernetes_node_name"}, "action": "keep", "regex": ".*"},
			map[string]any{"source_labels": []any{"__meta_kubernetes_node_name"}, "action": "replace", "target_label": "node", "regex": "(.*)", "replacement": "${1}"},
		},
	}}
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
		"image":             map[string]any{"tag": rancherImageTag},
		"extraEnv":          extraEnv,
		"livenessProbe": map[string]any{
			"initialDelaySeconds": 30,
			"periodSeconds":       3600,
		},
		"fullnameOverride": "rancher",
		"nameOverride":     "rancher",
	}

	if rancherImageOverride != "" {
		result["rancherImage"] = rancherImageOverride
	}

	return result
}

// naturalCompare compares strings a and b in "natural" alphanumeric order
func naturalCompare(a, b string) bool {
	tokenRegex := regexp.MustCompile(`\d+|\D+`)
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

func updateMonitoringProject(cluster *tofu.Cluster) error {
	var b strings.Builder

	args := []string{
		"get", "namespace", nsCattleSystem,
		"-o", "jsonpath={.metadata.annotations.field\\.cattle\\.io/projectId}",
	}
	if err := kubectl.Exec(cluster.Kubeconfig, &b, args...); err != nil {
		return fmt.Errorf("failed to read projectId from cattle-system: %w", err)
	}

	projID := strings.TrimSpace(b.String())
	if projID == "" {
		return fmt.Errorf("no projectId found")
	}

	if err := kubectl.Exec(cluster.Kubeconfig, os.Stdout,
		"annotate", "namespace", nsCattleMonitoringSystem,
		"field.cattle.io/projectId="+projID, "--overwrite"); err != nil {
		return fmt.Errorf("failed to annotate cattle-monitoring-system: %w", err)
	}

	return nil
}
