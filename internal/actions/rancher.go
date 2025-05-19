package actions

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
	yaml "gopkg.in/yaml.v2"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	shepherdclusters "github.com/rancher/shepherd/extensions/clusters"
	shepherddefaults "github.com/rancher/shepherd/extensions/defaults"
	shepherdtokens "github.com/rancher/shepherd/extensions/token"
	"github.com/rancher/shepherd/pkg/session"
	shepherdwait "github.com/rancher/shepherd/pkg/wait"

	"github.com/rancher/tests/actions/pipeline"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/reports"

	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	adminUser := &management.User{
		Username: "admin",
		Password: bootstrapPassword,
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

func ProvisionDownstreamClusters(r *dart.Dart, templates []dart.ClusterTemplate, rancherClient *rancher.Client) error {
	if r.ClusterBatchSize <= 0 {
		panic("ClusterBatchSize must be > 0")
	}
	for _, template := range r.ClusterTemplates {
		err := ProvisionClustersInBatches(r, template, rancherClient)
		if err != nil {
			return err
		}
	}
	return nil
}

// Provisions clusters in "batches" where r.ClusterBatchSize is the maximum # of clusters to provision before sleeping for a short period and continuing
// This will continue to provision clusters until template.ClusterCount # of Clusters have been provisioned
func ProvisionClustersInBatches(r *dart.Dart, template dart.ClusterTemplate, rancherClient *rancher.Client) error {
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		return err
	}

	batchNum := 0
	// Create batches of clusters from the template
	for i := 0; i < template.ClusterCount; i += r.ClusterBatchSize {
		// Create a batch of templates with unique names
		batchTemplates := make([]dart.ClusterTemplate, 0, r.ClusterBatchSize)
		j := min(i+r.ClusterBatchSize, template.ClusterCount)

		// Generate the name for each instance of the template and add the template instances to the batchTemplates slice
		for k := i; k < j; k++ {
			templateCopy := template
			templateCopy.SetGeneratedName(fmt.Sprintf("%d-%d", batchNum, k-i))
			batchTemplates = append(batchTemplates, templateCopy)
		}

		// Create and run a batch runner for this batch of templates
		batchRunner := NewSequencedBatchRunner[dart.ClusterTemplate](len(batchTemplates))
		err := batchRunner.Run(batchTemplates, statuses, clusterStatePath, rancherClient, nil)
		if err != nil {
			return err
		}

		batchNum++
	}

	return nil
}

func provisionClusterWithRunner[J JobDataTypes](br *SequencedBatchRunner[J], template dart.ClusterTemplate,
	statuses map[string]*ClusterStatus, rancherClient *rancher.Client) (skipped bool, err error) {

	clusterName := template.GeneratedName()

	stateMutex.Lock()
	cs := FindOrCreateStatusByName(statuses, clusterName)
	// cs.ClusterTemplate = template
	stateMutex.Unlock()

	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageNew, Completed: time.Now()}
	br.seqCh <- struct{}{}

	if cs.Provisioned {
		fmt.Printf("Cluster %s has already been provisioned, skipping...\n", cs.Name)
		return true, nil
	}
	fmt.Printf("Continuing with cluster provisioning...\n")

	// switch {
	// case strings.Contains(template.DistroVersio"k3s"):
	// 	template.DistroVersion = []string{template.DistroVersion}
	// case strings.Contains(template.DistroVersio"rke2"):
	// 	template.DistroVersion = []string{template.DistroVersion}
	// default:
	// 	return false, fmt.Errorf("error while parsing kubernetes version for version %v", template.DistroVersion)
	// }

	nodeProvider := CreateProvider(template.ClusterConfig.Provider)
	templateClusterConfig := ConvertConfigToClusterConfig(template.ClusterConfig)

	// Create the cluster
	clusterObject, err := provisioning.CreateProvisioningCluster(rancherClient, nodeProvider, templateClusterConfig, nil)
	reports.TimeoutClusterReport(clusterObject, err)
	if err != nil {
		return false, fmt.Errorf("error while provisioning cluster with ClusterConfig %v:\n%v", templateClusterConfig, err)
	}

	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageCreated, Completed: time.Now()}
	br.seqCh <- struct{}{}
	fmt.Printf("Cluster named %s was created.\n", clusterName)

	// Wait for the cluster to be ready
	fiveMinuteTimeout := int64(shepherddefaults.FiveMinuteTimeout)
	listOpts := metav1.ListOptions{
		FieldSelector:  "metadata.name=" + clusterObject.ID,
		TimeoutSeconds: &fiveMinuteTimeout,
	}
	watchInterface, err := rancherClient.GetManagementWatchInterface(management.ClusterType, listOpts)
	if err != nil {
		return false, fmt.Errorf("error while getting Management Watch Interface with Cluster %v and ListOptions %v:\n%v", clusterObject.ID, listOpts, err)
	}

	checkFunc := shepherdclusters.IsProvisioningClusterReady
	err = shepherdwait.WatchWait(watchInterface, checkFunc)
	reports.TimeoutClusterReport(clusterObject, err)
	if err != nil {
		return false, fmt.Errorf("error while waiting for Provisioned Cluster to be Ready %v:\n%v", clusterObject.ID, err)
	}

	cs.Provisioned = true
	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageProvisioned, Completed: time.Now()}
	br.seqCh <- struct{}{}
	fmt.Printf("Cluster named %s was provisioned.\n", clusterName)

	return false, nil
}

func ImportDownstreamClusters(r *dart.Dart, clusters []tofu.Cluster, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	if r.ClusterBatchSize <= 0 {
		panic("ClusterBatchSize must be > 0")
	}

	if len(clusters) == 0 {
		fmt.Printf("No importable Clusters were provided.\n")
	}

	err := ImportClustersInBatches(r, clusters, rancherClient, rancherConfig)
	if err != nil {
		return err
	}

	return nil
}

func ImportClustersInBatches(r *dart.Dart, clusters []tofu.Cluster, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		return err
	}

	// Enqueue clusters in batches and collect results
	for i := 0; i < len(clusters); i += r.ClusterBatchSize {
		j := min(i+r.ClusterBatchSize, len(clusters))
		batch := clusters[i:j]

		batchRunner := NewSequencedBatchRunner[tofu.Cluster](len(batch))
		err := batchRunner.Run(batch, statuses, clusterStatePath, rancherClient, rancherConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func importClusterWithRunner[J JobDataTypes](br *SequencedBatchRunner[J], cluster tofu.Cluster,
	statuses map[string]*ClusterStatus, rancherClient *rancher.Client, rancherConfig *rancher.Config,
) (skipped bool, err error) {
	stateMutex.Lock()
	cs := FindOrCreateStatusByName(statuses, cluster.Name)
	stateMutex.Unlock()
	<-br.seqCh
	br.Updates <- stateUpdate{Name: cluster.Name, Stage: StageNew, Completed: time.Now()}
	br.seqCh <- struct{}{}

	fmt.Printf("Found existing ClusterStatus object for Cluster with name %s.\n", cluster.Name)
	if cs.Imported {
		fmt.Printf("Cluster %s has already been imported, skipping...\n", cs.Name)
		return true, nil
	}
	fmt.Printf("Continuing with cluster creation...\n")

	importCluster := provv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: fleetNamespace,
		},
	}
	if !cs.Created {
		if _, err = CreateK3SRKE2Cluster(rancherClient, rancherConfig, &importCluster); err != nil {
			return false, fmt.Errorf("error while creating Steve Cluster with Name %s:\n%w", importCluster.Name, err)
		}
		// sequence the Created event
		<-br.seqCh
		br.Updates <- stateUpdate{Name: cluster.Name, Stage: StageCreated, Completed: time.Now()}
		br.seqCh <- struct{}{}
		fmt.Printf("Cluster named %s was created.\n", importCluster.Name)
	}

	updatedCluster := new(provv1.Cluster)
	err = BackoffWait(30, func() (finished bool, err error) {
		updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
		if err != nil {
			return false, fmt.Errorf("error while getting Cluster by Name %s in Namespace %s:\n%w", importCluster.Name, importCluster.Namespace, err)
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

	fmt.Printf("Importing Cluster, ID:%s Name:%s\n", updatedCluster.Status.ClusterName, updatedCluster.Name)
	err = shepherdclusters.ImportCluster(rancherClient, updatedCluster, restConfig)
	if err != nil {
		return false, fmt.Errorf("error while creating Job for importing Cluster %s:\n%w", updatedCluster.Name, err)
	}

	err = BackoffWait(100, func() (finished bool, err error) {
		updatedCluster, _, err = shepherdclusters.GetProvisioningClusterByName(rancherClient, importCluster.Name, importCluster.Namespace)
		if err != nil {
			return false, fmt.Errorf("error while getting Cluster by Name %s in Namespace %s:\n%w", importCluster.Name, importCluster.Namespace, err)
		}

		return updatedCluster.Status.Ready, nil
	})
	if err != nil {
		return false, err
	}

	cs.Imported = true
	<-br.seqCh
	br.Updates <- stateUpdate{Name: cluster.Name, Stage: StageImported, Completed: time.Now()}
	br.seqCh <- struct{}{}
	fmt.Printf("Cluster named %s was imported.\n", updatedCluster.Name)

	podErrors := StatusPodsWithTimeout(rancherClient, updatedCluster.Status.ClusterName, shepherddefaults.OneMinuteTimeout)
	if len(podErrors) > 0 {
		errorStrings := make([]string, len(podErrors))
		for i, e := range podErrors {
			errorStrings[i] = e.Error()
		}
		return false, fmt.Errorf("error while checking Status of Pods in Cluster %s:\n%s", updatedCluster.Status.ClusterName, strings.Join(errorStrings, "\n"))
	}

	return false, nil
}

func RegisterCustomClusters(r *dart.Dart, templates []tofu.CustomCluster,
	rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	if r.ClusterBatchSize <= 0 {
		panic("ClusterBatchSize must be > 0")
	}

	if len(templates) == 0 {
		fmt.Printf("No cluster templates were provided.\n")
	}

	for _, template := range templates {
		yamlData, err := yaml.Marshal(template)
		if err != nil {
			log.Fatalf("Error marshaling YAML: %v", err)
		}
		fmt.Printf("\ntofu.CustomCluster: %s\n", string(yamlData))
	}

	for _, template := range templates {
		err := RegisterCustomClustersInBatches(r, template, rancherClient, rancherConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func RegisterCustomClustersInBatches(r *dart.Dart, template tofu.CustomCluster, rancherClient *rancher.Client, rancherConfig *rancher.Config) error {
	clusterStatePath := fmt.Sprintf("%s/%s", r.TofuWorkspaceStatePath, ClustersStateFile)
	statuses, err := LoadClusterState(clusterStatePath)
	if err != nil {
		return err
	}

	batchNum := 0
	// Enqueue clusters in batches and collect results
	for i := 0; i < template.ClusterCount; i += r.ClusterBatchSize {
		batchTemplates := make([]tofu.CustomCluster, 0, r.ClusterBatchSize)
		j := min(i+r.ClusterBatchSize, template.ClusterCount)

		// var nts []dart.NodeTemplate
		// Generate the name for each instance of the template (each cluster) and add the template instances to the batchTemplates slice
		// Build []NodeTemplate[dart.ProviderConfig] as well, so we know how many nodes to create + with what configurations
		for k := i; k < j; k++ {
			// templateCopy := template
			// templateCopy.SetGeneratedName(fmt.Sprintf("%d-%d", batchNum, k-i))
			// switch {
			// case strings.Contains(templateCopy.ClusterConfig.Provider, dart.HarvesterProvider):
			// 	nts, err = dart.BuildNodeTemplates[dart.HarvesterNodeConfig](&templateCopy, k)
			// case strings.Contains(templateCopy.ClusterConfig.Provider, dart.AWSProvider):
			// 	panic("AWS Custom Cluster provider flow not yet supported")
			// case strings.Contains(templateCopy.ClusterConfig.Provider, dart.AzureProvider):
			// 	panic("Azure Custom Cluster provider flow not yet supported")
			// case strings.Contains(templateCopy.ClusterConfig.Provider, dart.K3DProvider):
			// 	panic("K3D Custom Cluster provider flow not yet supported")
			// }
			// if err != nil {
			// 	return err
			// }
			batchTemplates = append(batchTemplates, template)
		}

		batchRunner := NewSequencedBatchRunner[tofu.CustomCluster](len(batchTemplates))
		err := batchRunner.Run(batchTemplates, statuses, clusterStatePath, rancherClient, rancherConfig)
		if err != nil {
			return err
		}

		batchNum++
	}

	return nil
}

func registerCustomClusterWithRunner[J JobDataTypes](br *SequencedBatchRunner[J],
	template tofu.CustomCluster, statuses map[string]*ClusterStatus,
	rancherClient *rancher.Client, rancherConfig *rancher.Config) (skipped bool, err error) {

	clusterName := template.Name
	stateMutex.Lock()
	cs := FindOrCreateStatusByName(statuses, clusterName)
	stateMutex.Unlock()

	// gather # nodes for this cluster
	// for _, nt := range nts {
	// 	harvesterNodeVars := nt.NodeModuleVariables.(dart.HarvesterNodeConfig)
	// 	switch {
	// 	case strings.Contains(template.ClusterConfig.Provider, dart.HarvesterProvider):
	// 		var diskSizes []string
	// 		for _, disk := range harvesterNodeVars.Disks {
	// 			diskSizes = append(diskSizes, fmt.Sprintf("%dGi", disk.Size))
	// 		}
	// 		// var sshKeys []harvester.VMSSHKey
	// 		// for _, sshKey := range harvesterNodeVars.SSHSharedPublicKeys {
	// 		// 	sshKeys = append(sshKeys, harvester.VMSSHKey{
	// 		// 		Name:      sshKey.Name,
	// 		// 		Namespace: sshKey.Namespace,
	// 		// 	})
	// 		// }
	// 		vmInput := harvester.VMInput{
	// 			Count:       nt.NodeCount,
	// 			Name:        nt.NamePrefix,
	// 			Namespace:   harvesterNodeVars.Namespace,
	// 			Description: "",
	// 			Image: harvester.VMImage{
	// 				ID:        "",
	// 				Name:      harvesterNodeVars.ImageName,
	// 				Namespace: harvesterNodeVars.ImageNamespace,
	// 			},
	// 			CPUs:      harvesterNodeVars.CPU,
	// 			Memory:    harvesterNodeVars.Memory,
	// 			DiskSizes: diskSizes,
	// 			User: harvester.VMUser{
	// 				Name: "opensuse",
	// 			},
	// 			Network: harvester.VMNetworkInput{
	// 				Name:      "vlan2179-public",
	// 				Namespace: "harvester-public",
	// 			},
	// 			SSHKey: harvester.VMSSHKey{
	// 				Name:      "ival-ssh-key",
	// 				Namespace: "bullseye",
	// 			},
	// 		}
	// 		err = harvester.CreateVMs(h, k, &vmInput)
	// 	case strings.Contains(template.ClusterConfig.Provider, dart.AWSProvider):
	// 		panic("AWS Custom Cluster provider flow not yet supported")
	// 	case strings.Contains(template.ClusterConfig.Provider, dart.AzureProvider):
	// 		panic("Azure Custom Cluster provider flow not yet supported")
	// 	case strings.Contains(template.ClusterConfig.Provider, dart.K3DProvider):
	// 		panic("K3D Custom Cluster provider flow not yet supported")
	// 	}
	// }

	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageNew, Completed: time.Now()}
	br.seqCh <- struct{}{}

	if cs.Registered {
		fmt.Printf("Cluster %s has already been registered, skipping...\n", cs.Name)
		return true, nil
	}
	fmt.Printf("Continuing with cluster registration...\n")

	provCluster := &provv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "provisioning.cattle.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: fleetNamespace,
		},
		Spec: provv1.ClusterSpec{
			KubernetesVersion: template.DistroVersion,
			DefaultPodSecurityAdmissionConfigurationTemplateName: psactRancherPrivileged,
			RKEConfig: &provv1.RKEConfig{},
		},
	}
	var clusterResp *v1.SteveAPIObject
	if !cs.Created {
		fmt.Printf("Creating Cluster object for %s\n", cs.Name)
		clusterResp, err = CreateK3SRKE2Cluster(rancherClient, rancherConfig, provCluster)
		if err != nil {
			return false, err
		}
	}
	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageCreated, Completed: time.Now()}
	br.seqCh <- struct{}{}
	fmt.Printf("Cluster named %s was created.\n", provCluster.Name)

	var machinePools []provv1.RKEMachinePool
	for _, pool := range template.MachinePools {
		newPool := provv1.RKEMachinePool{
			EtcdRole:         pool.MachinePoolConfig.Etcd,
			ControlPlaneRole: pool.MachinePoolConfig.ControlPlane,
			WorkerRole:       pool.MachinePoolConfig.Worker,
			Quantity:         &pool.MachinePoolConfig.Quantity,
		}
		machinePools = append(machinePools, newPool)
	}
	provCluster.Spec.RKEConfig.MachinePools = machinePools

	// vmiMap, vms, vmData, err := harvester.ListVMs(h, "bullseye")
	// var tofuNodes []tofu.Node
	// for i, data := range vmData {
	// 	tofuNodes = append(tofuNodes, tofu.Node{
	// 		Name:            data.Name,
	// 		PublicIP:        vmiMap[data.Name].Status.Interfaces[0].IPs[0],
	// 		PublicHostName:  vms[i].Spec.Template.Spec.Hostname,
	// 		PrivateIP:       vmiMap[data.Name].Status.Interfaces[0].IPs[1],
	// 		PrivateHostName: "",
	// 		SSHUser:         "opensuse",
	// 		SSHKeyPath:      "~/.ssh/jenkins-elliptic-validation.pem",
	// 	})
	// }

	clusterObject, err := RegisterCustomCluster(rancherClient, clusterResp, provCluster, template.Nodes)
	reports.TimeoutClusterReport(clusterObject, err)
	if err != nil {
		return false, err
	}

	err = VerifyCluster(rancherClient, rancherConfig, clusterObject)
	if err != nil {
		return false, err
	}
	<-br.seqCh
	br.Updates <- stateUpdate{Name: clusterName, Stage: StageRegistered, Completed: time.Now()}
	br.seqCh <- struct{}{}
	fmt.Printf("Cluster named %s was registered.\n", clusterName)

	return false, nil
}
