package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
	apisV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/tests/actions/clusters"
	rancherclusters "github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/registries"
	"github.com/rancher/tests/actions/reports"
	"github.com/sirupsen/logrus"

	"github.com/rancher/shepherd/clients/rancher"
	mgmtv3 "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	shepherdclusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/defaults"
	shepherddefaults "github.com/rancher/shepherd/extensions/defaults"
	stevetypes "github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/rancher/shepherd/extensions/etcdsnapshot"
	"github.com/rancher/shepherd/extensions/kubeconfig"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/tokenregistration"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	shepherdnodes "github.com/rancher/shepherd/pkg/nodes"
	"github.com/rancher/shepherd/pkg/wait"

	"github.com/rancher/tests/actions/psact"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	psactRancherPrivileged string = "rancher-privileged"
)

// ConvertConfigToClusterConfig converts the ClusterConfig from (user) input to a rancher/tests ClusterConfig
func ConvertConfigToClusterConfig(config *dart.ClusterConfig) *rancherclusters.ClusterConfig {
	var newConfig rancherclusters.ClusterConfig
	for i := range config.MachinePools {
		newConfig.MachinePools[i].Pools = config.MachinePools[i].Pools
		newConfig.MachinePools[i].MachinePoolConfig = config.MachinePools[i].MachinePoolConfig.MachinePoolConfig
	}
	newConfig.Providers = &[]string{config.Provider}
	newConfig.PSACT = psactRancherPrivileged
	return &newConfig
}

// CreateK3SRKE2Cluster is a "helper" functions that takes a rancher client, and the rke2 cluster config as parameters.
// This function registers a delete cluster function with a wait.WatchWait to ensure the cluster is removed cleanly
func CreateK3SRKE2Cluster(client *rancher.Client, config *rancher.Config, cluster *apisV1.Cluster) (*v1.SteveAPIObject, error) {
	clusterObj, err := client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).Create(cluster)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	err = kwait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 2*time.Minute, true, func(_ context.Context) (done bool, err error) {
		client, err = client.ReLoginForConfig(config)
		if err != nil {
			return false, err
		}

		_, err = client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).ByID(clusterObj.ID)
		if err != nil {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	client.Session.RegisterCleanupFunc(func() error {
		adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
		if err != nil {
			return err
		}

		provKubeClient, err := adminClient.GetKubeAPIProvisioningClient()
		if err != nil {
			return err
		}

		watchInterface, err := provKubeClient.Clusters(clusterObj.ObjectMeta.Namespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + clusterObj.ObjectMeta.Name,
			TimeoutSeconds: &shepherddefaults.WatchTimeoutSeconds,
		})

		if err != nil {
			return err
		}

		client, err = client.ReLogin()
		if err != nil {
			return err
		}

		err = client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).Delete(clusterObj)
		if err != nil {
			return err
		}

		return wait.WatchWait(watchInterface, func(event watch.Event) (ready bool, err error) {
			cluster := event.Object.(*apisV1.Cluster)
			if event.Type == watch.Error {
				return false, fmt.Errorf("there was an error deleting cluster %s: %w", cluster.Name, err)
			} else if event.Type == watch.Deleted {
				return true, nil
			} else if cluster == nil {
				return true, nil
			}
			return false, nil
		})
	})

	return clusterObj, nil
}

// createRegistrationCommand is a helper for rke2/k3s custom clusters to create the registration command with advanced options configured per node
func createRegistrationCommand(command, publicIP, privateIP string, machinePool apisV1.RKEMachinePool) string {
	if len(publicIP) > 0 {
		command += fmt.Sprintf(" --address %s", publicIP)
	}
	if len(privateIP) > 0 {
		command += fmt.Sprintf(" --internal-address %s", privateIP)
	}
	for labelKey, labelValue := range machinePool.Labels {
		command += fmt.Sprintf(" --label %s=%s", labelKey, labelValue)
	}
	for _, taint := range machinePool.Taints {
		command += fmt.Sprintf(" --taints %s=%s:%s", taint.Key, taint.Value, taint.Effect)
	}
	return command
}

// RegisterCustomCluster registers a non-rke1 cluster using a 3rd party client for its nodes
func RegisterCustomCluster(client *rancher.Client, config *rancher.Config, steveObject *v1.SteveAPIObject, cluster *apisV1.Cluster, nodes []tofu.Node) (*v1.SteveAPIObject, error) {
	quantityPerPool := []int32{}
	rolesPerPool := []string{}
	for _, pool := range cluster.Spec.RKEConfig.MachinePools {
		var finalRoleCommand string
		if pool.ControlPlaneRole {
			finalRoleCommand += " --controlplane"
		}
		if pool.EtcdRole {
			finalRoleCommand += " --etcd"
		}
		if pool.WorkerRole {
			finalRoleCommand += " --worker"
		}

		quantityPerPool = append(quantityPerPool, *pool.Quantity)
		rolesPerPool = append(rolesPerPool, finalRoleCommand)
	}

	customCluster, err := client.Steve.SteveType(etcdsnapshot.ProvisioningSteveResouceType).ByID(steveObject.ID)
	if err != nil {
		return nil, err
	}

	clusterStatus := &apisV1.ClusterStatus{}
	err = v1.ConvertToK8sType(customCluster.Status, clusterStatus)
	if err != nil {
		return nil, err
	}

	token, err := tokenregistration.GetRegistrationToken(client, clusterStatus.ClusterName)
	if err != nil {
		return nil, err
	}

	kubeProvisioningClient, err := client.GetKubeAPIProvisioningClient()
	if err != nil {
		return nil, err
	}

	result, err := kubeProvisioningClient.Clusters(cluster.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + cluster.Name,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		return nil, err
	}

	checkFunc := shepherdclusters.IsProvisioningClusterReady
	var command string
	totalNodesObserved := 0
	for poolIndex, poolRole := range rolesPerPool {
		for nodeIndex := range int(quantityPerPool[poolIndex]) {
			node := nodes[totalNodesObserved+nodeIndex]

			logrus.Infof("Execute Registration Command for node named %s, ID %s", node.NodeName, node.NodeID)
			logrus.Infof("Linux pool detected, using bash...")

			command = fmt.Sprintf("%s %s", token.InsecureNodeCommand, poolRole)
			command = createRegistrationCommand(command, node.PublicIPAddress, node.PrivateIPAddress, cluster.Spec.RKEConfig.MachinePools[poolIndex])
			logrus.Infof("Node command: %s", command)

			nodeSSHKey, err := tofu.ReadBytesFromPath(node.SSHKeyPath)
			if err != nil {
				return nil, fmt.Errorf("error getting node's SSH Key from %s: %w", node.SSHKeyPath, err)
			}
			shepherdNode := shepherdnodes.Node{
				NodeID:           node.NodeID,
				PublicIPAddress:  node.PublicIPAddress,
				PrivateIPAddress: node.PrivateIPAddress,
				SSHUser:          node.SSHUser,
				SSHKey:           nodeSSHKey,
			}
			output, err := shepherdNode.ExecuteCommand(command)
			if err != nil {
				return nil, err
			}
			logrus.Info(output)
		}
		totalNodesObserved += int(quantityPerPool[poolIndex])
	}

	err = wait.WatchWait(result, checkFunc)
	if err != nil {
		return nil, err
	}

	registeredCluster, err := client.Steve.SteveType(stevetypes.Provisioning).ByID(cluster.Namespace + "/" + cluster.Name)
	return registeredCluster, err
}

// VerifyClusterCreated confirms that the cluster resource exists
func VerifyClusterCreated(client *rancher.Client, name, namespace string) (bool, error) {
	obj, _, err := shepherdclusters.GetProvisioningClusterByName(client, name, namespace)
	if err != nil {
		return false, fmt.Errorf("API error verifying creation of %s: %w", name, err)
	}
	return obj != nil, nil
}

// VerifyClusterImported confirms that the cluster resource is in Ready state
func VerifyClusterImported(client *rancher.Client, name, namespace string) (bool, error) {
	obj, _, err := shepherdclusters.GetProvisioningClusterByName(client, name, namespace)
	if err != nil {
		return false, fmt.Errorf("error getting Cluster by verifying import of %s: %w", name, err)
	}
	// In case the Cluster object was not successfully created in the first place
	if obj == nil {
		return false, nil
	}
	return obj.Status.Ready, nil
}

// VerifyCluster validates that a non-rke1 cluster and its resources are in a good state, matching a given config.
func VerifyCluster(client *rancher.Client, cluster *v1.SteveAPIObject) error {
	client, err := client.ReLogin()
	if err != nil {
		return err
	}

	adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
	if err != nil {
		return err
	}

	kubeProvisioningClient, err := adminClient.GetKubeAPIProvisioningClient()
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	watchInterface, err := kubeProvisioningClient.Clusters(cluster.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + cluster.Name,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	checkFunc := shepherdclusters.IsProvisioningClusterReady
	err = wait.WatchWait(watchInterface, checkFunc)
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	clusterToken, err := clusters.CheckServiceAccountTokenSecret(client, cluster.Name)
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}
	if !clusterToken {
		return fmt.Errorf("serviceAccountTokenSecret does not exist in this cluster: %s", cluster.Name)
	}

	err = nodestat.AllMachineReady(client, cluster.ID, defaults.ThirtyMinuteTimeout)
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	status := &apisV1.ClusterStatus{}
	err = v1.ConvertToK8sType(cluster.Status, status)
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	clusterSpec := &apisV1.ClusterSpec{}
	err = v1.ConvertToK8sType(cluster.Spec, clusterSpec)
	reports.TimeoutClusterReport(cluster, err)
	if err != nil {
		return err
	}

	if clusterSpec.DefaultPodSecurityAdmissionConfigurationTemplateName != "" && len(clusterSpec.DefaultPodSecurityAdmissionConfigurationTemplateName) > 0 {

		err := psact.CreateNginxDeployment(client, status.ClusterName, clusterSpec.DefaultPodSecurityAdmissionConfigurationTemplateName)
		reports.TimeoutClusterReport(cluster, err)
		if err != nil {
			return err
		}
	}

	if clusterSpec.RKEConfig.Registries != nil {
		for registryName := range clusterSpec.RKEConfig.Registries.Configs {
			havePrefix, err := registries.CheckAllClusterPodsForRegistryPrefix(client, status.ClusterName, registryName)
			reports.TimeoutClusterReport(cluster, err)
			if !havePrefix {
				return fmt.Errorf("found cluster (%s) pods that do not have the expected registry prefix %s: %w", status.ClusterName, registryName, err)
			}
			if err != nil {
				return err
			}
		}
	}

	if clusterSpec.LocalClusterAuthEndpoint.Enabled {
		mgmtClusterObject, err := adminClient.Management.Cluster.ByID(status.ClusterName)
		reports.TimeoutClusterReport(cluster, err)
		if err != nil {
			return err
		}
		err = VerifyACE(adminClient, mgmtClusterObject)
		if err != nil {
			return err
		}
	}

	podErrors := pods.StatusPods(client, status.ClusterName)
	if len(podErrors) > 0 {
		errorStrings := make([]string, len(podErrors))
		for i, e := range podErrors {
			errorStrings[i] = e.Error()
		}
		return fmt.Errorf("encountered pod errors: %s", strings.Join(errorStrings, ";"))
	}
	return nil
}

func VerifyACE(client *rancher.Client, cluster *mgmtv3.Cluster) error {
	client, err := client.ReLogin()
	if err != nil {
		return err
	}

	kubeConfig, err := kubeconfig.GetKubeconfig(client, cluster.ID)
	if err != nil {
		return err
	}

	original, err := client.SwitchContext(cluster.Name, kubeConfig)
	if err != nil {
		return err
	}

	originalResp, err := original.Resource(corev1.SchemeGroupVersion.WithResource("pods")).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range originalResp.Items {
		fmt.Printf("Pod %s", pod.GetName())
	}

	// each control plane has a context. For ACE, we should check these contexts
	contexts, err := kubeconfig.GetContexts(kubeConfig)
	if err != nil {
		return err
	}
	var contextNames []string
	for context := range contexts {
		if strings.Contains(context, "pool") {
			contextNames = append(contextNames, context)
		}
	}

	for _, contextName := range contextNames {
		dynamic, err := client.SwitchContext(contextName, kubeConfig)
		if err != nil {
			return err
		}
		resp, err := dynamic.Resource(corev1.SchemeGroupVersion.WithResource("pods")).Namespace("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Switched Context to %v", contextName)
		for _, pod := range resp.Items {
			fmt.Printf("Pod %v", pod.GetName())
		}
	}
	return nil
}
