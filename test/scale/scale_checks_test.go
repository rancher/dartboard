package scale

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promconfig "github.com/prometheus/common/config"
	"github.com/rancher/norman/types"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/kubernetesversions"
	"github.com/rancher/rancher/tests/framework/extensions/kubeconfig"
	"github.com/rancher/rancher/tests/framework/extensions/provisioning"
	"github.com/rancher/rancher/tests/framework/extensions/provisioninginput"
	"github.com/rancher/rancher/tests/framework/extensions/rke1/nodetemplates"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ScaleChecksTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	rke1Provider       provisioning.RKE1Provider
	nodeTemplateConfig *nodetemplates.NodeTemplate
	provisioningConfig *provisioninginput.Config
	clusterConfig      *clusters.ClusterConfig
	nodeTemplate       *nodetemplates.NodeTemplate
	scaleConfig        *Config
	clientConfig       *clientcmd.ClientConfig
	restClient         *restclient.Config
	clusterID          string
	outputPath         string
	promV1             promv1.API
}

func (s *ScaleChecksTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *ScaleChecksTestSuite) SetupSuite() {
	// Force standard logger to parse newlines without quoting (print actual newline instead of "\n")
	log.StandardLogger().SetFormatter(&log.TextFormatter{
		DisableQuote: true,
	})

	testSession := session.NewSession()
	s.session = testSession

	scaleConfig := ScaleConfig()
	s.scaleConfig = scaleConfig

	rancherConfig := new(rancher.Config)
	config.LoadConfig(rancher.ConfigurationFileKey, rancherConfig)

	s.nodeTemplateConfig = new(nodetemplates.NodeTemplate)
	config.LoadConfig(nodetemplates.NodeTemplateConfigurationFileKey, s.nodeTemplateConfig)

	s.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, s.provisioningConfig)

	client, err := rancher.NewClient(rancherConfig.AdminToken, s.session)
	require.NoError(s.T(), err)

	s.client = client

	clusterID, err := clusters.GetClusterIDByName(s.client, rancherConfig.ClusterName)
	require.NoError(s.T(), err)
	s.clusterID = clusterID

	kubeConfig, err := kubeconfig.GetKubeconfig(s.client, "local")
	require.NoError(s.T(), err)
	s.clientConfig = kubeConfig
	s.restClient, err = (*kubeConfig).ClientConfig()
	require.NoError(s.T(), err)

	s.provisioningConfig.RKE1KubernetesVersions, err = kubernetesversions.Default(
		s.client, clusters.RKE1ClusterType.String(), s.provisioningConfig.RKE1KubernetesVersions)
	require.NoError(s.T(), err)

	awsProviderName := string(provisioninginput.AWSProviderName)
	s.rke1Provider = provisioning.CreateRKE1Provider(awsProviderName)
	s.clusterConfig = clusters.ConvertConfigToClusterConfig(s.provisioningConfig)
	s.clusterConfig.KubernetesVersion = s.provisioningConfig.RKE1KubernetesVersions[0]
	s.nodeTemplate, err = s.rke1Provider.NodeTemplateFunc(s.client)
	require.NoError(s.T(), err)

	formattedTimestamp := time.Now().Format("2006-01-02")
	s.outputPath = fmt.Sprintf("%s/%s-%s", s.scaleConfig.OutputDir, strings.Split(s.client.RancherConfig.Host, ".")[0], formattedTimestamp)
	err = os.MkdirAll(s.outputPath+"/", 0755)
	require.NoError(s.T(), err)

	promSecret := promconfig.Secret("N7q-*fs+Ut&Wb_Y")
	promConfig := promapi.Config{
		Address:      "https://" + s.client.RancherConfig.Host + "/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-prometheus:9090/proxy/",
		RoundTripper: promconfig.NewBasicAuthRoundTripper("admin", promSecret, "", "", promapi.DefaultRoundTripper),
	}
	promClient, err := promapi.NewClient(promConfig)
	require.NoError(s.T(), err)
	promV1 := promv1.NewAPI(promClient)
	s.promV1 = promV1
}

func (s *ScaleChecksTestSuite) TestAWSBatchScale() {
	clusterObjects := []*management.Cluster{}
	// var clusterObject *management.Cluster
	listOptions := &metav1.ListOptions{
		LabelSelector: "app=rancher",
		FieldSelector: "status.phase=Running",
	}
	rawBatches := float64((s.scaleConfig.ClusterTarget / s.scaleConfig.BatchSize)) + float64(0.5)
	numBatches := int(math.Floor(rawBatches))
	halfComplete := int(float64(numBatches) / float64(2))
	log.Info("numBatches: ", numBatches)
	log.Info("halfComplete: ", halfComplete)
	scaleInStartTime := metav1.NewTime(time.Now()).Rfc3339Copy()
	var scaleInHalfTime metav1.Time
	var numClusters string
	var podNames []string
	// Create clusters
	for i := 0; i < numBatches; i++ {
		for j := 0; j < s.scaleConfig.BatchSize; j++ {
			// clusterObject, err := provisioning.CreateProvisioningRKE1Cluster(s.client, s.rke1Provider, s.clusterConfig, s.nodeTemplate)
			// require.NoError(s.T(), err)
			// clusterObjects = append(clusterObjects, clusterObject)
		}
		if i == halfComplete {
			scaleInHalfTime = metav1.NewTime(time.Now()).Rfc3339Copy()
			// collect metrics
			podNames, err := kubeconfig.GetPodNames(s.client, s.clusterID, "cattle-system", listOptions)
			require.NoError(s.T(), err)
			log.Info("POD NAMES: ", podNames)
			numClusters = strconv.Itoa(halfComplete*s.scaleConfig.BatchSize) + "-clusters"
			for _, podName := range podNames {
				log.Info("Getting mem profile for: ", podName)
				memProfileDest := fmt.Sprintf("%s/%s-%s-%s", s.outputPath, podName, numClusters, MemProfileFileName)
				getRancherMemProfile(*s.restClient, *s.clientConfig, podName, memProfileDest)
				log.Info("Getting cpu profile for: ", podName)
				cpuProfileDest := fmt.Sprintf("%s/%s-%s-%s", s.outputPath, podName, numClusters, CPUProfileFileName)
				getRancherCPUProfile(*s.restClient, *s.clientConfig, podName, cpuProfileDest)
				log.Info("Getting rancher logs for: ", podName)
				logs, err := getAllRancherLogs(s.client, s.clusterID, podName, scaleInStartTime)
				if err != nil {
					log.Warnf("error getting pod logs for pod (%s): %v", podName, err)
				}
				logFileDest := fmt.Sprintf("%s/%s-%s%s", s.outputPath, "rancher_logs", numClusters, ".txt")
				s.writeLogsToFile(logs, logFileDest)
			}
		}
	}
	clusterCollection, err := s.client.Management.Cluster.List(&types.ListOpts{})
	if err != nil {
		log.Warnf("error getting management clusters (%v): %v", clusterCollection, err)
	}
	clusterList := clusterCollection.Data
	numClusters = strconv.Itoa(len(clusterList)-1) + "-clusters"
	for _, podName := range podNames {
		logs, err := getAllRancherLogs(s.client, s.clusterID, podName, scaleInHalfTime)
		if err != nil {
			log.Warnf("error getting pod logs for pod (%s): %v", podName, err)
		}
		logFileDest := fmt.Sprintf("%s/%s-%s%s", s.outputPath, "rancher_logs", numClusters, ".txt")
		s.writeLogsToFile(logs, logFileDest)
	}
	s.session.RegisterCleanupFunc(func() error {
		var err error
		for _, cluster := range clusterObjects {
			err = s.client.Management.Cluster.Delete(cluster)
			require.NoError(s.T(), err)
		}
		return err
	})
	// timeoutSeconds := int64(defaults.FifteenMinuteTimeout.Seconds())
	scaleInEndTime := metav1.NewTime(time.Now()).Rfc3339Copy()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := promv1.Range{
		Start: scaleInStartTime.Time,
		End:   scaleInEndTime.Time,
		Step:  time.Minute,
	}
	query := "sum(irate(apiserver_request_total{}[5m])) by (code, resource, verb)"
	result, warnings, err := s.promV1.QueryRange(ctx, query, r, promv1.WithTimeout(5*time.Second))
	if err != nil {
		log.Warnf("Error querying Prometheus for (%s): %v\n", query, err)
	}
	if len(warnings) > 0 {
		log.Warnf("Warnings querying Prometheus for (%s): %v\n", query, warnings)
	}
	log.Infof("Prometheus query result: \n%v\n", result)
	// provisioning.VerifyRKE1ClusterWithTimeout(s.T(), s.client, s.clusterConfig, clusterObject, timeoutSeconds)
}

func (s *ScaleChecksTestSuite) writeLogsToFile(logs string, dest string) {
	f, err := os.OpenFile(dest,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warnf("error creating log file (%s): %v", dest, err)
	} else {
		log.Info("Writing rancher logs to file at: ", dest)
		_, err = f.WriteString(logs)
		if err != nil {
			log.Warnf("error writing to log file (%s): %v", dest, err)
		}
		err := f.Close()
		if err != nil {
			log.Warnf("error closing log file (%s): %v", f.Name(), err)
		}
	}
}

func TestScaleChecks(t *testing.T) {
	suite.Run(t, new(ScaleChecksTestSuite))
}
