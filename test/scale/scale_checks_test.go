package scale

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/git-ival/dartboard/test/utils/helpers"

	"github.com/git-ival/dartboard/test/utils/ranchermonitoring"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring/dashboards/rancherclusternodes"

	gapi "github.com/grafana/grafana-api-golang-client"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/kubeapi/configmaps"
	"github.com/rancher/shepherd/extensions/kubeapi/workloads/deployments"
	"github.com/rancher/shepherd/extensions/kubeconfig"
	"github.com/rancher/shepherd/extensions/provisioning"
	"github.com/rancher/shepherd/extensions/provisioninginput"
	"github.com/rancher/shepherd/extensions/rke1/nodetemplates"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ScaleChecksTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	k8sVersion         string
	rke1Provider       *provisioning.RKE1Provider
	provider           *provisioning.Provider
	provisioningConfig *provisioninginput.Config
	clusterConfig      *clusters.ClusterConfig
	nodeTemplate       *nodetemplates.NodeTemplate
	scaleConfig        *Config
	clientConfig       *clientcmd.ClientConfig
	restConfig         *rest.Config
	clusterID          string
	outputPath         string
	promV1             *promv1.API
	gapiClient         *gapi.Client
	dashboards         []string
}

func (s *ScaleChecksTestSuite) TearDownSuite() {
	log.Info("Cleaning up Scale Test Suite")
	s.session.Cleanup()
}

func (s *ScaleChecksTestSuite) SetupSuite() {
	// Force standard logger to parse newlines without quoting (print actual newline instead of "\n")
	log.StandardLogger().SetFormatter(&log.TextFormatter{
		DisableQuote: true,
	})

	testSession := session.NewSession()
	s.session = testSession

	scaleConfig := LoadScaleConfig()
	s.scaleConfig = scaleConfig

	s.dashboards = []string{}
	s.dashboards = append(s.dashboards, rancherclusternodes.UID)
	s.dashboards = append(s.dashboards, ranchermonitoring.CustomKubernetesAPIServerUIDs()...)
	s.dashboards = append(s.dashboards, ranchermonitoring.CustomRancherPerformanceDebuggingUIDs()...)

	rancherConfig := new(rancher.Config)
	config.LoadConfig(rancher.ConfigurationFileKey, rancherConfig)

	client, err := rancher.NewClient(rancherConfig.AdminToken, s.session)
	require.NoError(s.T(), err)

	s.client = client

	s.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, s.provisioningConfig)

	s.k8sVersion, s.rke1Provider, s.provider, s.clusterConfig, s.nodeTemplate = helpers.
		SetupProvisioningReqs(s.T(), s.session, s.client, s.provisioningConfig)

	clusterID, err := clusters.GetClusterIDByName(s.client, rancherConfig.ClusterName)
	require.NoError(s.T(), err)
	s.clusterID = clusterID

	kubeConfig, err := kubeconfig.GetKubeconfig(s.client, "local")
	require.NoError(s.T(), err)
	s.clientConfig = kubeConfig
	s.restConfig, err = (*kubeConfig).ClientConfig()
	require.NoError(s.T(), err)

	formattedTimestamp := time.Now().Format(time.DateOnly)
	s.scaleConfig.OutputDir = fmt.Sprintf("%s/%s-%s", s.scaleConfig.OutputDir, strings.Split(s.client.RancherConfig.Host, ".")[0], formattedTimestamp)

	err = os.MkdirAll(s.scaleConfig.OutputDir+"/"+PreScaleInOutputDir+"/", 0755)
	require.NoError(s.T(), err)
	err = os.MkdirAll(s.scaleConfig.OutputDir+"/"+HalftimeOutputDir+"/", 0755)
	require.NoError(s.T(), err)
	err = os.MkdirAll(s.scaleConfig.OutputDir+"/"+PostScaleInOutputDir+"/", 0755)
	require.NoError(s.T(), err)

	s.promV1, s.gapiClient, err = ranchermonitoring.SetupClients(s.client.RancherConfig.Host, s.client.RancherConfig.AdminToken, s.client.RancherConfig.AdminPassword)
	require.NoError(s.T(), err)
	err = helpers.CreateCustomMonitoringDashboards(s.T(), s.session, s.client, s.scaleConfig.ConfigMapsDir)
	require.NoError(s.T(), err)

	// Enable as much data as possible through ingress, needed for taking Grafana Dashboard Snapshots
	ingressConfigMapPatch := `{"data":{"proxy-body-size":"0"}}`
	patchedIngressConfigMap, err := configmaps.PatchConfigMap(s.client, "local", ranchermonitoring.IngressConfigMapName, "ingress-nginx", ingressConfigMapPatch, types.MergePatchType)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "0", patchedIngressConfigMap.Data["proxy-body-size"])
	nginxPath, err := ranchermonitoring.GrafanaNginxYAML("../utils/ranchermonitoring/files/grafana-nginx/")
	require.NoError(s.T(), err)
	grafanaConfigMapYAML, err := os.ReadFile(nginxPath)
	require.NoError(s.T(), err)
	patchedGrafanaConfigMap, err := configmaps.PatchConfigMapFromYAML(s.client, "local", ranchermonitoring.GrafanaConfigMapName, ranchermonitoring.RancherMonitoringNamespace, grafanaConfigMapYAML, types.MergePatchType)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "0", patchedGrafanaConfigMap.Data["proxy-body-size"])
	_, err = deployments.RestartDeployment(s.client, "local", ranchermonitoring.GrafanaDeploymentName, ranchermonitoring.RancherMonitoringNamespace)
	require.NoError(s.T(), err)
	s.T().Log("SETUP COMPLETE")
}

func (s *ScaleChecksTestSuite) TestRKE1BatchScale() {
	var scaleInEndTime, scaleInHalfTime metav1.Time
	var i int

	rawBatches := float64((s.scaleConfig.ClusterTarget / s.scaleConfig.BatchSize)) + float64(0.5)
	numBatches := int(math.Floor(rawBatches)) + 1
	halfComplete := int(float64(numBatches-1) / float64(2))
	log.Info("Number of Batches to provision: ", (numBatches - 1))
	log.Info("Number of Batches until half completed: ", halfComplete)
	numClusters := 1
	batchTimeout := time.Minute * time.Duration(s.scaleConfig.BatchTimeout)
	scaleInStartTime := metav1.NewTime(time.Now().Add(-3 * time.Hour)).Rfc3339Copy()
	var allClusters int
	helpers.CollectRancherMetricsAndArtifacts(s.T(), s.session, s.client, s.gapiClient, s.restConfig, s.clientConfig, s.scaleConfig.OutputDir+"/"+HalftimeOutputDir,
		"", "local", s.scaleConfig.ConfigMapsDir, numClusters, scaleInStartTime, metav1.NewTime(time.Now()).Rfc3339Copy(), s.dashboards)
	/// Create clusters
	for i = 1; i < numBatches; i++ {
		mgmtClusters, err := s.client.Management.Cluster.ListAll(nil)
		require.NoError(s.T(), err)
		if err == nil {
			allClusters = len(mgmtClusters.Data) - 1
			if allClusters == s.scaleConfig.ClusterTarget {
				break
			}
		}
		helpers.BatchScaleUpClusters(s.T(), s.session, s.client, s.scaleConfig.BatchSize, batchTimeout, s.rke1Provider, s.provider, s.k8sVersion, s.clusterConfig, s.nodeTemplate)
		if i == halfComplete {
			scaleInHalfTime = metav1.NewTime(time.Now()).Rfc3339Copy()
			helpers.CollectRancherMetricsAndArtifacts(s.T(), s.session, s.client, s.gapiClient, s.restConfig, s.clientConfig, s.scaleConfig.OutputDir+"/"+HalftimeOutputDir,
				"", "local", s.scaleConfig.ConfigMapsDir, numClusters, scaleInStartTime, scaleInHalfTime, s.dashboards)
		}
	}
	scaleInEndTime = metav1.NewTime(time.Now()).Rfc3339Copy()
	helpers.CollectRancherMetricsAndArtifacts(s.T(), s.session, s.client, s.gapiClient, s.restConfig, s.clientConfig, s.scaleConfig.OutputDir+"/"+PostScaleInOutputDir,
		"", "local", s.scaleConfig.ConfigMapsDir, numClusters, scaleInHalfTime, scaleInEndTime, s.dashboards)
}

func TestScaleChecks(t *testing.T) {
	suite.Run(t, new(ScaleChecksTestSuite))
}
