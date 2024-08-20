package helpers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/shepherd/pkg/config"

	"github.com/git-ival/dartboard/test/utils/grafanautils"
	"github.com/git-ival/dartboard/test/utils/imageutils"
	"github.com/git-ival/dartboard/test/utils/ranchermonitoring"
	"github.com/git-ival/dartboard/test/utils/rancherprofiling"
	gapi "github.com/grafana/grafana-api-golang-client"
	provV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	mgmtV3 "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/kubeconfig"
	"github.com/rancher/shepherd/extensions/kubectl"
	"github.com/rancher/shepherd/extensions/provisioning"
	"github.com/rancher/shepherd/extensions/provisioninginput"
	"github.com/rancher/shepherd/extensions/rke1/nodetemplates"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

func ConfigMapGVR() schema.GroupVersionResource {
	return corev1.SchemeGroupVersion.WithResource("configmaps")
}

func V1ClusterGVR() schema.GroupVersionResource {
	return provV1.SchemeGroupVersion.WithResource("clusters")
}

type ScreenshotParams struct {
	URL           string
	ImageFilePath string
	WindowSize    [2]int
	Selector      string
	Timeout       int
	Cookies       []string
}

type GenericProvider interface {
	provisioning.RKE1Provider | provisioning.Provider
}

// endpoints for the different pprof visualizations
// selectors have been commented out as they can sometimes fail to be found by chromedp
func PprofEndpoints() map[string]ScreenshotParams {
	return map[string]ScreenshotParams{
		"graph": {
			URL: "http://" + rancherprofiling.BasePprofAddress + "/ui/",
			// Selector: "div#graph",
		},
		"top": {
			URL: "http://" + rancherprofiling.BasePprofAddress + "/ui/top",
			// Selector: "table#toptable",
		},
		"flame": { // Not able to programmatically retrieve screenshot of this via browser, have to resort to html
			URL: "http://" + rancherprofiling.BasePprofAddress + "/ui/flamegraph",
			// Selector: "div#stack-chart",
		},
		"peek": {
			URL: "http://" + rancherprofiling.BasePprofAddress + "/ui/peek",
			// Selector: "div#content",
		},
		"source": {
			URL: "http://" + rancherprofiling.BasePprofAddress + "/ui/source",
			// Selector: "div#content",
		},
	}
}

func GetAllRancherLogs(r *rancher.Client, clusterID string, podName string, since metav1.Time) (string, error) {
	podLogOptions := &corev1.PodLogOptions{
		Container:  "rancher",
		Timestamps: true,
		SinceTime:  &since,
	}
	log.Infof("Collecting Rancher logs since: %s", since.String())
	return kubeconfig.GetPodLogsWithOpts(r, clusterID, podName, "cattle-system", "", podLogOptions)
}

func SetupProvisioningReqs(t *testing.T, ts *session.Session, client *rancher.Client,
	provisioningConfig *provisioninginput.Config) (string, *provisioning.RKE1Provider, *provisioning.Provider, *clusters.ClusterConfig, *nodetemplates.NodeTemplate) {
	k8sVersions := append(provisioningConfig.RKE1KubernetesVersions, provisioningConfig.RKE2KubernetesVersions...)
	k8sVersions = append(k8sVersions, provisioningConfig.K3SKubernetesVersions...)
	require.Condition(t, func() bool { return len(k8sVersions) == 1 })
	k8sVersion := k8sVersions[0]

	var err error
	var clusterConfig *clusters.ClusterConfig
	var nodeTemplate *nodetemplates.NodeTemplate
	var rke1Provider *provisioning.RKE1Provider
	var provider *provisioning.Provider

	switch {
	case strings.Contains(k8sVersion, "rancher"): //RKE1
		// only used if provisioning RKE1 clusters
		nodeTemplate = new(nodetemplates.NodeTemplate)
		config.LoadConfig(nodetemplates.NodeTemplateConfigurationFileKey, nodeTemplate)
		tempProvider := provisioning.CreateRKE1Provider(provisioninginput.AWSProviderName.String())
		rke1Provider = &tempProvider
		nodeTemplate, err = rke1Provider.NodeTemplateFunc(client)
		require.NoError(t, err)
		provisioningConfig.RKE1KubernetesVersions, err = kubernetesversions.Default(
			client, clusters.RKE1ClusterType.String(), provisioningConfig.RKE1KubernetesVersions)
		require.NoError(t, err)
		clusterConfig = clusters.ConvertConfigToClusterConfig(provisioningConfig)
		clusterConfig.KubernetesVersion = provisioningConfig.RKE1KubernetesVersions[0]
	case strings.Contains(k8sVersion, "rke2"):
		tempProvider := provisioning.CreateProvider(provisioninginput.AWSProviderName.String())
		provider = &tempProvider
		provisioningConfig.RKE2KubernetesVersions, err = kubernetesversions.Default(
			client, clusters.RKE2ClusterType.String(), provisioningConfig.RKE2KubernetesVersions)
		require.NoError(t, err)
		clusterConfig = clusters.ConvertConfigToClusterConfig(provisioningConfig)
		clusterConfig.KubernetesVersion = provisioningConfig.RKE2KubernetesVersions[0]
	case strings.Contains(k8sVersion, "k3s"):
		t.Log("Setting up k3s config")
		tempProvider := provisioning.CreateProvider(provisioninginput.AWSProviderName.String())
		provider = &tempProvider
		provisioningConfig.K3SKubernetesVersions, err = kubernetesversions.Default(
			client, clusters.K3SClusterType.String(), provisioningConfig.K3SKubernetesVersions)
		require.NoError(t, err)
		clusterConfig = clusters.ConvertConfigToClusterConfig(provisioningConfig)
		clusterConfig.KubernetesVersion = provisioningConfig.K3SKubernetesVersions[0]
		t.Log("Finished setting up k3s config")
	}
	return k8sVersion, rke1Provider, provider, clusterConfig, nodeTemplate
}

func BatchScaleUpClusters(t *testing.T, ts *session.Session, client *rancher.Client, batchSize int, timeout time.Duration,
	rke1Provider *provisioning.RKE1Provider, provider *provisioning.Provider, k8sVersion string, clusterConfig *clusters.ClusterConfig, nodeTemplate *nodetemplates.NodeTemplate) {
	numClusters := 0
	batchStart := time.Now()
	timer := time.NewTimer(timeout)
	for i := 1; i <= batchSize; i++ {
		select {
		case <-timer.C:
			logrus.Info("Timeout reached")
			return
		default:
		}

		switch {
		case strings.Contains(k8sVersion, "rancher"): //RKE1
			if rke1Provider == nil {
				log.Errorf("Got RKE1 Kubernetes version, but RKE1Provider is nil!")
				panic(rke1Provider)
			}
			t.Logf("Provisioning %s cluster #%d", k8sVersion, numClusters+1)
			provisioningStart := time.Now()
			rke1ClusterObject, err := provisioning.CreateProvisioningRKE1Cluster(client, *rke1Provider, clusterConfig, nodeTemplate)
			if err != nil {
				log.Errorf("Failed to provision Cluster: %v", err)
			} else {
				// Wait for final cluster to be ready (all clusters in batch should have plenty of time to be ready by now)
				// Collect provisioning time of this final batch cluster
				if i == batchSize {
					timePassed := time.Since(provisioningStart)
					remainingTime := int(timeout.Minutes() - timePassed.Minutes())
					t.Logf("Provisioning time until forced timeout %dm", remainingTime)
					err = clusters.WaitForActiveRKE1ClusterWithTimeout(client, rke1ClusterObject.ID, remainingTime)
					if err != nil {
						log.Errorf("Cluster with Name (%s) and ID (%s) was not ready within the %d minute timeout: %v", rke1ClusterObject.Name, rke1ClusterObject.ID, 30, err)
					}
					t.Logf("Batch %d complete! Cluster became ready after %v!", i, time.Since(provisioningStart))
				}
			}
		case strings.Contains(k8sVersion, "rke2"), strings.Contains(k8sVersion, "k3s"):
			if provider == nil {
				log.Errorf("Got RKE2 or K3s Kubernetes version, but Provider is nil!")
				panic(provider)
			}
			t.Logf("Provisioning %s cluster #%d", k8sVersion, numClusters+1)
			provisioningStart := time.Now()
			clusterObject, err := provisioning.CreateProvisioningCluster(client, *provider, clusterConfig, nil)
			if err != nil {
				log.Errorf("Failed to provision Cluster: %v", err)
			} else {
				// Wait for final cluster to be ready (all clusters in batch should have plenty of time to be ready by now)
				// Collect provisioning time of this final batch cluster
				if i == batchSize {
					steveID := fmt.Sprintf("%s/%s", clusterObject.Namespace, clusterObject.Name)
					timePassed := time.Since(batchStart)
					remainingTime := int64(timeout.Minutes() - timePassed.Minutes())
					remainingSeconds := int64(remainingTime * 60)
					t.Logf("Provisioning time until forced timeout: %dm", remainingTime)
					err = clusters.WatchAndWaitForClusterWithTimeout(client, steveID, &remainingSeconds)
					if err != nil {
						log.Errorf("Cluster with Name (%s) and ID (%s) was not ready within the %d minute timeout: %v", clusterObject.Name, clusterObject.ID, 30, err)
					}
					t.Logf("Batch %d complete! Cluster became ready after %v!", i, time.Since(provisioningStart))
				}
			}
		}
		time.Sleep(24 * time.Second)
		numClusters = numClusters + 1
	}
}

func CreateCustomMonitoringDashboards(t *testing.T, ts *session.Session, client *rancher.Client, configMapsDir string) error {
	files, err := os.ReadDir(configMapsDir)
	require.NoError(t, err)

	for _, file := range files {
		if !file.IsDir() {
			f, err := os.ReadFile(configMapsDir + "/" + file.Name())
			require.NoError(t, err)
			dashboardYAML, err := yaml.YAMLToJSON(f)
			require.NoError(t, err)
			_, err = kubectl.CreateUnstructured(ts, client, dashboardYAML, "local", "cattle-dashboards", ConfigMapGVR())
			if k8sErrors.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
				logrus.Infof("configmap already exists for %v, skipping", file)
				continue
			}
			require.NoError(t, err)
		}
	}
	return nil
}

func WriteMonitoringSnapshotsToPNGs(t *testing.T, ts *session.Session, client *rancher.Client, gapi *gapi.Client, from time.Time, to time.Time, outputPath, clusterID, configMapsDir, prefix, suffix string, dashboardUIDs []string) ([]string, error) {
	var snapshotURLs []string
	var err error
	for _, d := range dashboardUIDs {
		// Ensure ConfigMaps for each Dashboard exist
		configMapPath := configMapsDir + "/" + d
		// In case not all dashboards have custom configmaps in local dir, get from cluster
		if _, err := os.Stat(configMapPath); errors.Is(err, os.ErrExist) {
			_, err := kubectl.GetUnstructured(ts, client, d, clusterID, "cattle-dashboards", ConfigMapGVR())
			require.NoError(t, err)
		}
		snapshotResponse, err := grafanautils.GetDashboardSnapshot(gapi, from, to, d, 9000, false)
		if err != nil {
			log.Warnf("Failed to retrieve dashboard snapshot for %v using time range from %v to %v. Skipping dashboard.", d, from, to)
			continue
		}
		var cookies []string
		imageutils.HTTPCookiesToSlice(gapi.Cookies(), &cookies)
		snapshotURL := "https://" + client.RancherConfig.Host + ranchermonitoring.GrafanaSnapshotRoute + snapshotResponse.Key
		snapshotURLs = append(snapshotURLs, snapshotURL)
		filePath := fmt.Sprintf("%s/%s%s%s.png", outputPath, prefix, d, suffix)
		err = imageutils.URLScreenshotToPNG(snapshotURL, filePath, ranchermonitoring.PanelContentSelector, nil, 60, cookies...)
		if err != nil {
			log.Warnf("Failed to write snapshotURL (%s) to file (%s): %v", snapshotURL, filePath, err)
		}
	}
	return snapshotURLs, err
}

func LogV1ClusterProvisioningTime(t *testing.T, ts *session.Session, client *rancher.Client, cluster *mgmtV3.Cluster, numClusters *int, outputPath, clusterID, namespace, configMapsDir, suffix string) (time.Duration, error) {
	createdTime, err := time.Parse(time.RFC3339, cluster.Created)
	if err != nil {
		return time.Duration(0), err
	}
	var readyTime time.Time
	v1Cluster, _, err := clusters.GetProvisioningClusterByName(client, cluster.ID, "fleet-default")
	if err != nil {
		return time.Duration(0), err
	}
	for _, condition := range v1Cluster.Status.Conditions {
		if condition.Type == "Ready" {
			readyTime, err = time.Parse(time.RFC3339, condition.LastUpdateTime)
			log.Infof("Cluster Created time is: %s", createdTime.Format(time.RFC3339))
			log.Infof("Cluster Ready time is: %s", condition.LastUpdateTime)
			break
		}
	}
	if err != nil {
		return time.Duration(0), err
	}
	filePath := outputPath + "/provisioning-times.log"
	provisioningTimeDiff := readyTime.Sub(createdTime)
	if numClusters != nil {
		text := fmt.Sprintf("%d Clusters: %s\n", numClusters, provisioningTimeDiff)
		WriteStringToFile(text, filePath)
	}
	log.Infof("Provisioning took: %s", provisioningTimeDiff)
	return provisioningTimeDiff, nil
}

func WriteSnapshotURLsToFiles(t *testing.T, ts *session.Session, client *rancher.Client, gapi *gapi.Client, from time.Time, to time.Time, outputPath, clusterID, configMapsDir, prefix, suffix string, dashboardUIDs []string) {
	snapshotURLs, err := WriteMonitoringSnapshotsToPNGs(t, ts, client, gapi, from, to, outputPath, clusterID, configMapsDir, prefix, suffix, dashboardUIDs)
	if err != nil {
		log.Infof("error writing snapshots to PNG: %v", err)
	}
	filename := prefix + "snapshots" + suffix + ".txt"
	f, err := os.Create(outputPath + "/" + filename)
	if err != nil {
		log.Infof("error creating file with path (%s): %v", outputPath+"/"+filename, err)
	}
	for _, url := range snapshotURLs {
		_, err = f.WriteString(url + "\n")
		if err != nil {
			log.Infof("error writing bytes to file (%s): %v", outputPath+"/"+filename, err)
		}
	}
}

func WriteStringToFile(s string, dest string) {
	f, err := os.OpenFile(dest,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warnf("error creating file (%s): %v", dest, err)
	} else {
		log.Info("Writing to file at: ", dest)
		_, err = f.WriteString(s)
		if err != nil {
			log.Warnf("error writing to file (%s): %v", dest, err)
		}
		err := f.Close()
		if err != nil {
			log.Warnf("error closing file (%s): %v", f.Name(), err)
		}
	}
}

func CollectProfileScreenshots(profilePath, imagePath string) error {
	var err error
	profCmd := rancherprofiling.StartServeProfile(profilePath)
	defer func() {
		if err := profCmd.Process.Kill(); err != nil {
			log.Errorf("CollectProfileScreenshots: Error killing command (%s): %v", profCmd.Args, err)
		} else {
			log.Infof("CollectProfileScreenshots: Successfully killed command (%s)", profCmd.Args)
		}
	}()

	endpoints := PprofEndpoints()
	for k, endpoint := range endpoints {
		endpoint.ImageFilePath = fmt.Sprintf("%s-%s", imagePath, k)
		endpoint.WindowSize = [2]int{2560, 1440}
		endpoint.Timeout = 30

		_, err = imageutils.GetURLWithRetry(endpoint.URL, 5)
		if err != nil {
			log.Errorf("CollectProfileScreenshots: Failed to get URL (%s), skipping URL: %v", endpoint.URL, err)
			continue
		}

		if k == "flame" {
			err = imageutils.LocalHTMLFromURL(endpoint.URL, endpoint.ImageFilePath+".html")
			if err != nil {
				log.Errorf("CollectProfileScreenshots: Failed to get flamegraph html: %v", err)
			}
		} else {
			err = imageutils.URLScreenshotToPNG(endpoint.URL, endpoint.ImageFilePath+".png", endpoint.Selector, &endpoint.WindowSize, 25, "")
			if err != nil {
				log.Errorf("CollectProfileScreenshots: Failed to get image of url (%s): %v", endpoint.URL, err)
			}
		}
	}
	return err
}

func CollectRancherMetricsAndArtifacts(t *testing.T, ts *session.Session, client *rancher.Client, gapi *gapi.Client,
	restClient *restclient.Config, clientConfig *clientcmd.ClientConfig, outputPath, prefix, clusterID, configMapsDir string,
	numClusters int, start metav1.Time, end metav1.Time, dashboardUIDs []string) {
	log.Infof("Collecting metrics and other artifacts at %d Clusters", numClusters)
	podNames, err := kubeconfig.GetPodNames(client, clusterID, "cattle-system", &metav1.ListOptions{
		LabelSelector: "app=rancher",
		FieldSelector: "status.phase=Running",
	})
	require.NoError(t, err)
	log.Info("Pod Names: ", podNames)
	clustersSuffix := fmt.Sprintf("-%d-clusters", numClusters)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	// Create a channel to communicate completion of goroutines
	done := make(chan string, len(podNames))
	pprofBaseDir := outputPath + "/pprof/"

	for _, podName := range podNames {
		err = os.MkdirAll(pprofBaseDir, 0755)
		if err != nil {
			log.Errorf("Failed to create pprof file directory: %v", err)
		}

		memProfilePath := fmt.Sprintf("%s%s%s-%s%s", pprofBaseDir, prefix, rancherprofiling.MemProfileFileName, podName, clustersSuffix+rancherprofiling.ProfileFileExtension)
		log.Infof("Getting mem profile for pod %s, storing at %s", podName, memProfilePath)
		err = rancherprofiling.GetRancherMemProfile(*restClient, *clientConfig, podName, memProfilePath)
		if err != nil {
			log.Errorf("Failed to get Rancher memory profile: %v", err)
			continue
		}
		memProfileImagePath := fmt.Sprintf("%s%s%s-%s%s", pprofBaseDir, prefix, rancherprofiling.MemProfileFileName, podName, clustersSuffix)
		log.Infof("Getting mem profile screenshots for %s, storing at %s", podName, pprofBaseDir)
		CollectProfileScreenshots(memProfilePath, memProfileImagePath)

		cpuProfilePath := fmt.Sprintf("%s%s%s-%s%s", pprofBaseDir, prefix, rancherprofiling.CPUProfileFileName, podName, clustersSuffix+rancherprofiling.ProfileFileExtension)
		log.Infof("Getting cpu profile for pod %s, storing at %s", podName, cpuProfilePath)
		err = rancherprofiling.GetRancherCPUProfile(*restClient, *clientConfig, podName, cpuProfilePath)
		if err != nil {
			log.Errorf("Failed to get Rancher CPU profile: %v", err)
			continue
		}
		cpuProfileImagePath := fmt.Sprintf("%s%s%s-%s%s", pprofBaseDir, prefix, rancherprofiling.CPUProfileFileName, podName, clustersSuffix)
		log.Infof("Getting cpu profile screenshots for %s, storing at %s", podName, pprofBaseDir)
		CollectProfileScreenshots(cpuProfilePath, cpuProfileImagePath)

		// Get rancher pod logs
		logFileDest := fmt.Sprintf("%s/%s%s%s-%s", outputPath, prefix, podName, clustersSuffix, start.String()+".log")
		podLogOptions := &corev1.PodLogOptions{
			Container: "rancher",
			SinceTime: &start,
		}
		go func(ctx context.Context, podName, logFileDest string, podLogOptions *corev1.PodLogOptions) {
			defer func() { done <- podName }() // Signal completion of goroutine
			log.Info("Getting pod logs")
			_, err := kubeconfig.GetPodLogsWithContext(ctx, client, "local", podName, "cattle-system", "", logFileDest, true, podLogOptions)
			if err != nil {
				log.Warnf("error getting pod logs for pod (%s): %v", podName, err)
			}
		}(ctx, podName, logFileDest, podLogOptions)
	}

	// Wait for all goroutines to complete or timeout
	for range podNames {
		select {
		case podName := <-done:
			log.Infof("Completed getting pod logs for %s", podName)
		case <-ctx.Done():
			log.Warn("Timeout waiting for podlog goroutines to complete")
			return
		}
	}
	_, newGAPI, err := ranchermonitoring.SetupClients(client.RancherConfig.Host, client.RancherConfig.AdminToken, client.RancherConfig.AdminPassword)
	if err != nil {
		log.Errorf("Could not re-setup prometheus and/or grafana clients: %v", err)
	}
	WriteMonitoringSnapshotsToPNGs(t, ts, client, newGAPI, start.Time, end.Time, outputPath, clusterID, configMapsDir, prefix, clustersSuffix, dashboardUIDs)
}
