package scale

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/git-ival/dartboard/test/utils/grafanautils"
	"github.com/git-ival/dartboard/test/utils/imageutils"

	"github.com/git-ival/dartboard/test/utils/ranchermonitoring"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring/dashboards/rancherclusternodes"

	gapi "github.com/grafana/grafana-api-golang-client"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	mgmtV3 "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/kubernetesversions"
	"github.com/rancher/rancher/tests/framework/extensions/kubeconfig"
	"github.com/rancher/rancher/tests/framework/extensions/kubectl"
	"github.com/rancher/rancher/tests/framework/extensions/provisioning"
	"github.com/rancher/rancher/tests/framework/extensions/provisioninginput"
	"github.com/rancher/rancher/tests/framework/extensions/rke1/nodetemplates"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
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

	scaleConfig := loadScaleConfig()
	s.scaleConfig = scaleConfig
	require.Condition(s.T(), func() bool {
		return s.scaleConfig.BatchSize%5 == 0
	}, "BatchSize must be a multiple of 5")

	s.dashboards = []string{}
	s.dashboards = append(s.dashboards, rancherclusternodes.UID)
	s.dashboards = append(s.dashboards, ranchermonitoring.CustomKubernetesAPIServerUIDs()...)
	s.dashboards = append(s.dashboards, ranchermonitoring.CustomRancherPerformanceDebuggingUIDs()...)

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

	formattedTimestamp := time.Now().Format(time.DateOnly)
	s.outputPath = fmt.Sprintf("%s/%s-%s", s.scaleConfig.OutputDir, strings.Split(s.client.RancherConfig.Host, ".")[0], formattedTimestamp)
	err = os.MkdirAll(s.outputPath+"/", 0755)
	require.NoError(s.T(), err)
	s.promV1, s.gapiClient, err = ranchermonitoring.SetupClients(s.client.RancherConfig.Host, s.client.RancherConfig.AdminToken, s.client.RancherConfig.AdminPassword)
	require.NoError(s.T(), err)
	err = s.createCustomDashboards()
	require.NoError(s.T(), err)
}

func (s *ScaleChecksTestSuite) TestRKE1BatchScale() {
	var scaleInStartTime, scaleInEndTime, scaleInHalfTime metav1.Time
	var i, j int

	rawBatches := float64((s.scaleConfig.ClusterTarget / s.scaleConfig.BatchSize)) + float64(0.5)
	numBatches := int(math.Floor(rawBatches)) + 1
	halfComplete := int(float64(numBatches-1) / float64(2))
	log.Info("Number of Batches to provision: ", (numBatches - 1))
	log.Info("Number of Batches until half completed: ", halfComplete)
	scaleInStartTime = metav1.NewTime(time.Now()).Rfc3339Copy()
	var clusterObject *mgmtV3.Cluster
	var err error
	numClusters := 0
	/// Create clusters
	for i = 1; i < numBatches; i++ {
		j = 1
		// timeoutMinutes := time.Duration(s.scaleConfig.BatchTimeout) * time.Minute
		// verifyTimeout := int64(timeoutMinutes.Seconds())
		for end := time.Now().Add(time.Minute * time.Duration(s.scaleConfig.BatchTimeout)); ; {
			log.Info("Provisioning RKE1 cluster")
			clusterObject, err = provisioning.CreateProvisioningRKE1Cluster(s.client, s.rke1Provider, s.clusterConfig, s.nodeTemplate)
			if err != nil {
				log.Errorf("Failed to provision Cluster: %v", err)
			}
			require.NoError(s.T(), err)
			// Get first cluster's provisioning time
			if i == 1 && j == 1 {
				err = clusters.WaitForActiveRKE1Cluster(s.client, clusterObject.ID)
				if err != nil {
					log.Errorf("Cluster with Name (%s) and ID (%s) was not ready within the %d minute timeout: %v", clusterObject.Name, clusterObject.ID, 30, err)
					continue
				}
				err = s.logProvisioningTime(clusterObject, numClusters)
				if err != nil {
					log.Errorf("Failed to log provisioning time for Cluster (%v): %v", clusterObject, err)
					continue
				}
			}
			// Wait for final cluster to be ready (all clusters in batch should have plenty of time to be ready by now)
			// Collect provisioning time of this final batch cluster
			if j == s.scaleConfig.BatchSize {
				err = clusters.WaitForActiveRKE1Cluster(s.client, clusterObject.ID)
				if err != nil {
					log.Errorf("Cluster with Name (%s) and ID (%s) was not ready within the %d minute timeout: %v", clusterObject.Name, clusterObject.ID, 30, err)
					continue
				}
				err = s.logProvisioningTime(clusterObject, numClusters)
				if err != nil {
					log.Errorf("Failed to log provisioning time for Cluster (%v): %v", clusterObject, err)
					continue
				}
				break
			}
			if j%5 == 0 {
				if time.Now().After(end) {
					break
				}
			}
			time.Sleep(24 * time.Second)
			j++
			numClusters = numClusters + 1
		}
		if i == halfComplete {
			scaleInHalfTime = metav1.NewTime(time.Now()).Rfc3339Copy()
			s.collectMetricsAndArtifacts(numClusters, scaleInStartTime, scaleInHalfTime)
		}
	}
	scaleInEndTime = metav1.NewTime(time.Now()).Rfc3339Copy()
	s.collectMetricsAndArtifacts(numClusters, scaleInStartTime, scaleInEndTime)
}

func (s *ScaleChecksTestSuite) logProvisioningTime(cluster *mgmtV3.Cluster, numClusters int) error {
	createdTime, err := time.Parse(time.RFC3339, cluster.Created)
	if err != nil {
		return err
	}
	var readyTime time.Time
	v1Cluster, _, err := clusters.GetProvisioningClusterByName(s.client, cluster.ID, "fleet-default")
	if err != nil {
		return err
	}
	for _, condition := range v1Cluster.Status.Conditions {
		if condition.Type == "Ready" {
			readyTime, err = time.Parse(time.RFC3339, condition.LastUpdateTime)
			break
		}
	}
	if err != nil {
		return err
	}
	filePath := s.outputPath + "/provisioning-times.log"
	provisioningTimeDiff := readyTime.Sub(createdTime)
	text := fmt.Sprintf("%d Clusters: %s\n", numClusters, provisioningTimeDiff)
	s.writeLogsToFile(text, filePath)
	log.Infof("Provisioning took: %s", provisioningTimeDiff)
	return nil
}

func (s *ScaleChecksTestSuite) writeSnapshotsToPNGs(from time.Time, to time.Time, suffix string) []string {
	var snapshotURLs []string
	for _, d := range s.dashboards {
		// Ensure ConfigMaps for each Dashboard exist
		configMapPath := s.scaleConfig.ConfigMapsDir + "/" + d
		// In case not all dashboards have custom configmaps in local dir, get from cluster
		if _, err := os.Stat(configMapPath); errors.Is(err, os.ErrExist) {
			_, err := kubectl.GetUnstructured(s.session, s.client, d, s.clusterID, "cattle-dashboards", configMapGVR())
			require.NoError(s.T(), err)
		}
		snapshotResponse, err := grafanautils.GetDashboardSnapshot(s.gapiClient, from, to, d, 9000, false)
		require.NoError(s.T(), err)
		var cookies []string
		imageutils.HTTPCookiesToSlice(s.gapiClient.Cookies(), &cookies)
		snapshotURL := "https://" + s.client.RancherConfig.Host + ranchermonitoring.GrafanaSnapshotRoute + snapshotResponse.Key
		snapshotURLs = append(snapshotURLs, snapshotURL)
		filePath := s.outputPath + "/" + d + suffix + ".png"
		err = imageutils.URLScreenshotToPNG(snapshotURL, filePath, ranchermonitoring.PanelContentSelector, cookies...)
		if err != nil {
			log.Warnf("Failed to write snapshotURL (%s) to file (%s): %v", snapshotURL, filePath, err)
		}
	}
	return snapshotURLs
}

func (s *ScaleChecksTestSuite) writeSnapshotsToFiles(from time.Time, to time.Time, suffix string) {
	snapshotURLs := s.writeSnapshotsToPNGs(from, to, suffix)
	filename := "snapshots" + suffix + ".txt"
	f, err := os.Create(s.outputPath + "/" + filename)
	if err != nil {
		log.Infof("error creating file with path (%s): %v", s.outputPath+"/"+filename, err)
	}
	for _, url := range snapshotURLs {
		_, err = f.WriteString(url + "\n")
		if err != nil {
			log.Infof("error writing bytes to file (%s): %v", s.outputPath+"/"+filename, err)
		}
	}
}

func (s *ScaleChecksTestSuite) writeLogsToFile(logs string, dest string) {
	f, err := os.OpenFile(dest,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warnf("error creating file (%s): %v", dest, err)
	} else {
		log.Info("Writing to file at: ", dest)
		_, err = f.WriteString(logs)
		if err != nil {
			log.Warnf("error writing to file (%s): %v", dest, err)
		}
		err := f.Close()
		if err != nil {
			log.Warnf("error closing file (%s): %v", f.Name(), err)
		}
	}
}

func (s *ScaleChecksTestSuite) collectMetricsAndArtifacts(numClusters int, start metav1.Time, end metav1.Time) {
	log.Infof("Collecting metrics and other artifacts at %d Clusters", numClusters)
	podNames, err := kubeconfig.GetPodNames(s.client, s.clusterID, "cattle-system", &metav1.ListOptions{
		LabelSelector: "app=rancher",
		FieldSelector: "status.phase=Running",
	})
	require.NoError(s.T(), err)
	log.Info("Pod Names: ", podNames)
	clustersSuffix := fmt.Sprintf("-%d-clusters", numClusters)
	for _, podName := range podNames {
		log.Info("Getting mem profile for: ", podName)
		memProfileDest := fmt.Sprintf("%s/%s-%s%s", s.outputPath, podName, MemProfileFileName, clustersSuffix+ProfileFileExtension)
		err = getRancherMemProfile(*s.restClient, *s.clientConfig, podName, memProfileDest)
		if err != nil {
			log.Errorf("Failed to get Rancher memory profile: %v", err)
			continue
		}
		log.Info("Getting cpu profile for: ", podName)
		cpuProfileDest := fmt.Sprintf("%s/%s-%s%s", s.outputPath, podName, CPUProfileFileName, clustersSuffix+ProfileFileExtension)
		err = getRancherCPUProfile(*s.restClient, *s.clientConfig, podName, cpuProfileDest)
		if err != nil {
			log.Errorf("Failed to get Rancher CPU profile: %v", err)
			continue
		}
		log.Info("Getting rancher logs for: ", podName)
		logs, err := getAllRancherLogs(s.client, s.clusterID, podName, start)
		if err != nil {
			log.Warnf("error getting pod logs for pod (%s): %v", podName, err)
		}
		logFileDest := fmt.Sprintf("%s/%s%s", s.outputPath, "rancher_logs", clustersSuffix+".txt")
		s.writeLogsToFile(logs, logFileDest)
		queryStartTime := start.Time.Add(-(time.Hour * 1))
		s.writeSnapshotsToPNGs(queryStartTime, end.Time, clustersSuffix)
	}
}

func (s *ScaleChecksTestSuite) createCustomDashboards() error {
	files, err := os.ReadDir(s.scaleConfig.ConfigMapsDir)
	require.NoError(s.T(), err)

	for _, file := range files {
		if !file.IsDir() {
			f, err := os.ReadFile(s.scaleConfig.ConfigMapsDir + "/" + file.Name())
			require.NoError(s.T(), err)
			dashboardYAML, err := yaml.YAMLToJSON(f)
			require.NoError(s.T(), err)
			_, err = kubectl.CreateUnstructured(s.session, s.client, dashboardYAML, s.clusterID, "cattle-dashboards", configMapGVR())
			if k8sErrors.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
				continue
			}
			require.NoError(s.T(), err)
		}
	}
	return nil
}

func TestScaleChecks(t *testing.T) {
	suite.Run(t, new(ScaleChecksTestSuite))
}
