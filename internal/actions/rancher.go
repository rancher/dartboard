package actions

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
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
)

const fleetNamespace = "fleet-default"

var maxWorkers = runtime.GOMAXPROCS(0) * 2

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
	adminUser := &management.User{
		Username: "admin",
		Password: "adminadminadmin",
	}
	fmt.Printf("Rancher Config:\nHost: %s\nAdminPassword: %s\nAdminToken: %s\nInsecure: %t\n", rancherConfig.Host, rancherConfig.AdminPassword, rancherConfig.AdminToken, *rancherConfig.Insecure)
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

	// Load or initialize map[string]*ClusterStatus
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
			statuses[clusterName] = &newClusterStatus
			clusterStatus = statuses[clusterName]
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

func importCluster(r *dart.Dart, cluster tofu.Cluster, statePath string, statuses map[string]*ClusterStatus, rancherClient *rancher.Client,
	rancherConfig *rancher.Config, updates chan<- stateUpdate) (skipped bool, err error) {
	stateMutex.Lock()
	clusterStatus := FindOrCreateStatusByName(statuses, cluster.Name)
	clusterStatus.Cluster = cluster
	stateMutex.Unlock()
	updates <- stateUpdate{}

	fmt.Printf("Found existing ClusterStatus object for Cluster with name %s.\n", cluster.Name)
	if clusterStatus.Imported {
		fmt.Printf("Cluster %s has already been imported, skipping...\n", clusterStatus.Name)
		return true, nil
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
			return false, fmt.Errorf("error while creating Steve Cluster with Name %v:\n%w", importCluster.Name, err)
		}
		stateMutex.Lock()
		clusterStatus.Created = true
		// err = SaveClusterState(statePath, *statuses)
		stateMutex.Unlock()
		updates <- stateUpdate{}
		if err != nil {
			return false, fmt.Errorf("error during ClusterStatus save after Cluster creation: %w", err)
		}
		fmt.Printf("Cluster named %s was created.\n", importCluster.Name)
	}

	updatedCluster := new(provv1.Cluster)
	err = BackoffWait(30, func() (finished bool, err error) {
		updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
		if err != nil {
			return false, fmt.Errorf("error while getting Cluster by Name %v in Namespace %v:\n%w", importCluster.Name, importCluster.Namespace, err)
		}

		if updatedCluster.Status.ClusterName != "" {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return false, err
	}

	restConfig, err := GetRESTConfigFromPath(cluster.Kubeconfig)
	if err != nil {
		return false, err
	}
	// Apply client-side rate limiting
	restConfig.QPS = 50
	restConfig.Burst = 100

	fmt.Printf("Importing Cluster: %s\n", updatedCluster.Status.ClusterName)
	err = shepherdclusters.ImportCluster(rancherClient, updatedCluster, restConfig)
	if err != nil {
		return false, fmt.Errorf("error while creating Job for importing Cluster %v:\n%w", updatedCluster.Name, err)
	}

	err = BackoffWait(100, func() (finished bool, err error) {
		updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
		if err != nil {
			return false, fmt.Errorf("error while getting Cluster by Name %v in Namespace %v:\n%w", importCluster.Name, importCluster.Namespace, err)
		}

		return updatedCluster.Status.Ready, nil
	})
	if err != nil {
		return false, err
	}
	stateMutex.Lock()
	clusterStatus.Imported = true
	// err = SaveClusterState(statePath, *statuses)
	stateMutex.Unlock()
	updates <- stateUpdate{}
	if err != nil {
		return false, err
	}
	fmt.Printf("Cluster named %s was imported.\n", updatedCluster.Name)

	podErrors := pods.StatusPodsWithTimeout(rancherClient, updatedCluster.Status.ClusterName, shepherddefaults.OneMinuteTimeout)
	if len(podErrors) > 0 {
		errorStrings := make([]string, len(podErrors))
		for i, e := range podErrors {
			errorStrings[i] = e.Error()
		}
		return false, fmt.Errorf("error while checking Status of Pods in Cluster %v:\n%s", updatedCluster.Status.ClusterName, strings.Join(errorStrings, "\n"))
	}

	return false, nil
}

func ImportClustersInBatches(r *dart.Dart, clusters []tofu.Cluster, batchSize int, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		return err
	}

	// channel for all write requests
	updates := make(chan stateUpdate, len(clusters)*2)
	defer close(updates)

	// start writer goroutine
	go func() {
		for range updates {
			stateMutex.Lock()
			if err := SaveClusterState(clusterStatePath, statuses); err != nil {
				fmt.Printf("error saving state: %v", err)
			}
			stateMutex.Unlock()
		}
	}()

	// Enqueue clusters in batches and collect results
	for i := 0; i < len(clusters); i += batchSize {
		jobs := make(chan tofu.Cluster, len(clusters))
		results := make(chan struct {
			skipped bool
			err     error
		}, len(clusters))

		// Spawn N workers per batch
		var wg sync.WaitGroup
		for w := 0; w < maxWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done() // decrement WaitGroup counter by one after each goroutine is done
				for c := range jobs {
					skipped, err := importCluster(r, c, clusterStatePath, statuses, rancherClient, rancherConfig, updates)
					results <- struct {
						skipped bool
						err     error
					}{skipped, err}
				}
			}()
		}

		j := min(i+batchSize, len(clusters))
		batch := clusters[i:j]

		// Send this batch as jobs
		for _, c := range batch {
			jobs <- c
		}
		close(jobs)

		// Reset skip count for this batch
		numSkipped := 0
		sleepAfter := false
		// Collect batch results
		for range batch {
			res := <-results

			if res.err != nil {
				return fmt.Errorf("import error: %w", res.err)
			}
			if res.skipped {
				numSkipped++
			}
			// Decide whether to sleep before propagating error
			sleepAfter = numSkipped < len(batch)/2
		}

		// After finishing this batch:
		if sleepAfter {
			// If fewer than half were skipped, sleep briefly
			fmt.Printf("Batch done: %d/%d skipped; sleeping before next batch.\n", numSkipped, len(batch))
			time.Sleep(shepherddefaults.TwoMinuteTimeout)
		} else {
			// Otherwise, go straight into the next batch
			fmt.Printf("Batch done: %d/%d skipped; continuing without sleep.\n", numSkipped, len(batch))
		}

		// Wait for all of the batch's workers to be Done and close the batch's results channel
		wg.Wait()
		close(results)
	}

	return nil
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
