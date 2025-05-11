package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	shepherdclusters "github.com/rancher/shepherd/extensions/clusters"
	shepherddefaults "github.com/rancher/shepherd/extensions/defaults"
	shepherdtokens "github.com/rancher/shepherd/extensions/token"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/rancher/shepherd/pkg/session"
	shepherdwait "github.com/rancher/shepherd/pkg/wait"

	clusteractions "github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/pipeline"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/reports"

	"github.com/sirupsen/logrus"

	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const fleetNamespace = "fleet-default"

func NewRancherConfig(host, adminToken, adminPassword string, insecure bool) rancher.Config {
	defaultBool := false
	return rancher.Config{
		Host:          host,
		AdminToken:    adminToken,
		AdminPassword: adminPassword,
		Insecure:      &insecure,
		Cleanup:       &defaultBool,
	}
}

func SetupRancherClient(rancherConfig *rancher.Config, bootstrapPassword string, session *session.Session) (*rancher.Client, error) {
	// adminToken, err := pipeline.CreateAdminToken(bootstrapPassword, rancherConfig)

	adminUser := &management.User{
		Username: "admin",
		Password: "adminadminadmin",
	}
	logrus.Printf("Rancher Config:\nHost: %v\nAdminPassword: %v\nAdminToken: %v\nInsecure: %v", rancherConfig.Host, rancherConfig.AdminPassword, rancherConfig.AdminToken, *rancherConfig.Insecure)
	adminToken, err := shepherdtokens.GenerateUserToken(adminUser, rancherConfig.Host)

	if err != nil {
		return nil, fmt.Errorf("error while creating Admin Token with config %v:\n%v", &rancherConfig, err)
	}
	rancherConfig.AdminToken = adminToken.Token

	client, err := rancher.NewClientForConfig(rancherConfig.AdminToken, rancherConfig, session)
	if err != nil {
		return nil, fmt.Errorf("error while setting up Rancher client with config %v:\n%v", rancherConfig, err)
	}

	err = pipeline.PostRancherInstall(client, rancherConfig.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("error during post- rancher install: %v", err)
	}

	client, err = rancher.NewClientForConfig(rancherConfig.AdminToken, rancherConfig, session)
	if err != nil {
		return nil, fmt.Errorf("error during post- rancher install on re-login: %v", err)
	}

	return client, err
}

// Provisions clusters in "batches" where batchSize is the maximum # of clusters to provision before sleeping for a short period and continuing
// This will continue to provision clusters until template.ClusterCount # of Clusters have been provisioned
func ProvisionClustersInBatches(r *dart.Dart, template dart.ClusterTemplate, batchSize int, rancherClient *rancher.Client) error {
	batchNum := 0
	for i := 0; i < template.ClusterCount; i += batchSize {
		// Number of clusters that can "fit" in this batch, if we have surpassed
		// template.ClusterCount # of Clusters then just provision the leftovers
		j := min(i+batchSize, template.ClusterCount)
		numInBatch := j - i
		sleepAfterBatch, err := ProvisionDownstreamClusterBatch(r, template, numInBatch, batchNum, rancherClient)
		if err != nil {
			return fmt.Errorf("error while provisioning clusters in batches: %v", err)
		}
		batchNum += 1
		fmt.Printf("Finished provisioning Cluster batch.\n")
		if sleepAfterBatch {
			fmt.Printf("Sleeping between batches.\n")
			time.Sleep(shepherddefaults.TwoMinuteTimeout)
		}
	}

	return nil
}

// Provisions batchSize number of Clusters
func ProvisionDownstreamClusterBatch(r *dart.Dart, template dart.ClusterTemplate, batchSize int, batchNum int, rancherClient *rancher.Client) (sleepAfterBatch bool, err error) {
	iter := 0
	numSkippedClusters := 0
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)

	// Load or initialize []ClusterStatus
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		logrus.Fatalf("error loading Cluster state: %v\n", err)
	}

	for i := range batchSize {
		clusterName := fmt.Sprintf("%s-%d-%d", template.NamePrefix, batchNum, i)
		clusterStatus := FindClusterStatusByName(statuses, clusterName)
		if clusterStatus == nil {
			newClusterStatus := ClusterStatus{
				Name:            clusterName,
				Created:         false,
				Imported:        false,
				ClusterTemplate: template,
			}
			statuses = append(statuses, newClusterStatus)
			clusterStatus = &statuses[len(statuses)-1]
			fmt.Printf("Did not find existing ClusterStatus object for Cluster with name %s.\n", clusterName)
		} else {
			fmt.Printf("Found existing ClusterStatus object for Cluster with name %s.\n", clusterName)
			if clusterStatus.Provisioned {
				fmt.Printf("Cluster %s has already been provisioned, skipping...\n", clusterStatus.Name)
				numSkippedClusters += 1
				continue
			}
		}
		fmt.Printf("Continuing with cluster provisioning...\n")

		switch {
		case strings.Contains(template.DistroVersion, "k3s"):
			template.Config.K3SKubernetesVersions = []string{template.DistroVersion}
		case strings.Contains(template.DistroVersion, "rke2"):
			template.Config.RKE2KubernetesVersions = []string{template.DistroVersion}
		default:
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while parsing kubernetes version for version %v", template.DistroVersion)
		}
		nodeProvider := CreateProvider(template.Config.Providers[0])

		templateClusterConfig := clusteractions.ConvertConfigToClusterConfig(template.Config)
		templateClusterConfig.CNI = template.Config.CNIs[0]

		// Uncomment below when we can set the clusterName directly
		// clusterName := fmt.Sprintf("downstream-prov-%d-%d", batchNum, iter)
		// TODO: Replace usage of rancher/tests/ functions with our own actions
		clusterObject, err := provisioning.CreateProvisioningCluster(rancherClient, nodeProvider, templateClusterConfig, nil)
		reports.TimeoutClusterReport(clusterObject, err)
		if err != nil {
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while provisioning cluster with ClusterConfig %v:\n%v", templateClusterConfig, err)
		}
		clusterStatus.Provisioned = true
		err = SaveClusterState(clusterStatePath, statuses)
		if err != nil {
			return (numSkippedClusters < batchSize/2), err
		}

		fiveMinuteTimeout := int64(shepherddefaults.FiveMinuteTimeout)
		listOpts := metav1.ListOptions{
			FieldSelector:  "metadata.name=" + clusterObject.ID,
			TimeoutSeconds: &fiveMinuteTimeout,
		}
		watchInterface, err := rancherClient.GetManagementWatchInterface(management.ClusterType, listOpts)
		if err != nil {
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while getting Management Watch Interface with Cluster %v and ListOptions %v:\n%v", clusterObject.ID, listOpts, err)
		}

		checkFunc := shepherdclusters.IsProvisioningClusterReady
		err = shepherdwait.WatchWait(watchInterface, checkFunc)
		reports.TimeoutClusterReport(clusterObject, err)
		if err != nil {
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while waiting for Provisioned Cluster to be Ready %v:\n%v", clusterObject.ID, err)
		}

		iter += 1
	}
	return (numSkippedClusters < batchSize/2), nil
}

func ProvisionDownstreamClusters(r *dart.Dart, templates []dart.ClusterTemplate, batchSize int, rancherClient *rancher.Client) error {
	if batchSize <= 0 {
		panic("ClusterBatchSize must be > 0")
	}
	for _, template := range templates {
		err := ProvisionClustersInBatches(r, template, batchSize, rancherClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func ImportClustersInBatches(r *dart.Dart, clusters []tofu.Cluster, batchSize int, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	batchNum := 0
	numClusters := len(clusters)
	for i := 0; i < numClusters; i += batchSize {
		// Number of clusters that can "fit" in this batch, if we have surpassed
		// len(clusters) # of Clusters then just provision the leftovers
		j := min(i+batchSize, numClusters)
		batch := clusters[i:j]
		numInBatch := len(batch)
		fmt.Printf("Number of Clusters in batch: %d\n", numInBatch)
		sleepAfterBatch, err := ImportDownstreamClusterBatch(r, batch, numInBatch, batchNum, rancherClient, rancherConfig)
		if err != nil {
			return fmt.Errorf("error while importing clusters in batches: %v", err)
		}
		batchNum += 1
		fmt.Printf("Finished importing Cluster batch.\n")
		if sleepAfterBatch {
			fmt.Printf("Sleeping between batches.\n")
			time.Sleep(shepherddefaults.TwoMinuteTimeout)
		}
	}

	return nil
}

// Import Clusters in batches matching batchSize, tracking the state of each in a ClusterStatus struct which gets saved in a statefile within the Tofu module's "_config" dir
// returns a bool which determines whether or not to sleep after this batch (will NOT sleep if less than half of the clusters in the batch were skipped)
func ImportDownstreamClusterBatch(r *dart.Dart, clusters []tofu.Cluster, batchSize int, batchNum int, rancherClient *rancher.Client, rancherConfig *rancher.Config) (sleepAfterBatch bool, err error) {
	iter := 0
	numSkippedClusters := 0
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)

	// Load or initialize []ClusterStatus
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		logrus.Fatalf("error loading Cluster state: %v\n", err)
	}

	for _, cluster := range clusters {
		clusterStatus := FindClusterStatusByName(statuses, cluster.Name)
		if clusterStatus == nil {
			newClusterStatus := ClusterStatus{
				Name:     cluster.Name,
				Created:  false,
				Imported: false,
				Cluster:  cluster,
			}
			statuses = append(statuses, newClusterStatus)
			clusterStatus = &statuses[len(statuses)-1]
			fmt.Printf("Did not find existing ClusterStatus object for Cluster with name %s.\n", cluster.Name)
		} else {
			fmt.Printf("Found existing ClusterStatus object for Cluster with name %s.\n", cluster.Name)
			if clusterStatus.Imported {
				fmt.Printf("Cluster %s has already been imported, skipping...\n", clusterStatus.Name)
				numSkippedClusters += 1
				continue
			}
		}
		fmt.Printf("Continuing with cluster creation...\n")

		importCluster := provv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name,
				Namespace: fleetNamespace,
			},
		}

		if !clusterStatus.Created {
			_, err = CreateK3SRKE2Cluster(rancherClient, rancherConfig, &importCluster)
			if err != nil {
				return (numSkippedClusters < batchSize/2), fmt.Errorf("error while creating Steve Cluster with Name %v:\n%v", importCluster.Name, err)
			}
			clusterStatus.Created = true
			err = SaveClusterState(clusterStatePath, statuses)
			if err != nil {
				return (numSkippedClusters < batchSize/2), err
			}
		}

		backoff := wait.Backoff{
			Duration: 1 * time.Second,
			Factor:   1.1,
			Jitter:   0.1,
			Steps:    30,
		}

		updatedCluster := new(provv1.Cluster)
		err = wait.ExponentialBackoff(backoff, func() (finished bool, err error) {
			updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
			if err != nil {
				return false, fmt.Errorf("error while getting Cluster by Name %v in Namespace %v:\n%v", importCluster.Name, importCluster.Namespace, err)
			}

			if updatedCluster.Status.ClusterName != "" {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			return (numSkippedClusters < batchSize/2), err
		}

		clusterKubeconfig, err := GetKubeconfigBytes(cluster.Kubeconfig)
		if err != nil {
			return (numSkippedClusters < batchSize/2), err
		}
		restConfig, err := GetRESTConfigFromBytes(clusterKubeconfig)
		if err != nil {
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error getting REST Config for Import Cluster %v:\n%v", importCluster.Name, err)
		}

		fmt.Printf("Importing Cluster: %s", updatedCluster.Status.ClusterName)
		err = shepherdclusters.ImportCluster(rancherClient, updatedCluster, restConfig)
		if err != nil {
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while creating Job for importing Cluster %v:\n%v", updatedCluster.Name, err)
		}

		backoff = wait.Backoff{
			Duration: 1 * time.Second,
			Factor:   1.1,
			Jitter:   0.1,
			Steps:    100,
		}

		err = wait.ExponentialBackoff(backoff, func() (finished bool, err error) {
			updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
			if err != nil {
				return false, fmt.Errorf("error while getting Cluster by Name %v in Namespace %v:\n%v", importCluster.Name, importCluster.Namespace, err)
			}

			if updatedCluster.Status.Ready {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			return (numSkippedClusters < batchSize/2), err
		}
		clusterStatus.Imported = true
		err = SaveClusterState(clusterStatePath, statuses)
		if err != nil {
			return (numSkippedClusters < batchSize/2), err
		}

		podErrors := pods.StatusPodsWithTimeout(rancherClient, updatedCluster.Status.ClusterName, 3*shepherddefaults.TenSecondTimeout)
		if len(podErrors) > 0 {
			var errorStrings []string
			for _, e := range podErrors {
				errorStrings = append(errorStrings, e.Error())
			}
			return (numSkippedClusters < batchSize/2), fmt.Errorf("error while checking Status of Pods in Cluster %v:\n%s", updatedCluster.Status.ClusterName, strings.Join(errorStrings, "\n"))
		}

		iter += 1
	}

	return (numSkippedClusters < batchSize/2), nil
}

func ImportDownstreamClusters(r *dart.Dart, clusters []tofu.Cluster, batchSize int, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	if batchSize <= 0 {
		panic("ClusterBatchSize must be > 0")
	}

	if len(clusters) == 0 {
		panic("No importable Clusters were provided")
	}

	err := ImportClustersInBatches(r, clusters, batchSize, rancherClient, rancherConfig)
	if err != nil {
		return err
	}

	return nil
}
