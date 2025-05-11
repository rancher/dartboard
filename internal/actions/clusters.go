package actions

import (
	"context"
	"fmt"
	"time"

	apisV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"

	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	shepherdclusters "github.com/rancher/shepherd/extensions/clusters"
	shepherddefaults "github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/pkg/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

// CreateK3SRKE2Cluster is a "helper" functions that takes a rancher client, and the rke2 cluster config as parameters.
// This function registers a delete cluster function with a wait.WatchWait to ensure the cluster is removed cleanly
func CreateK3SRKE2Cluster(client *rancher.Client, config *rancher.Config, rke2Cluster *apisV1.Cluster) (*v1.SteveAPIObject, error) {
	cluster, err := client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).Create(rke2Cluster)
	if err != nil {
		return nil, err
	}

	err = kwait.Poll(500*time.Millisecond, 2*time.Minute, func() (done bool, err error) {
		client, err = client.ReLoginForConfig(config)
		if err != nil {
			return false, err
		}

		_, err = client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).ByID(cluster.ID)
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

		watchInterface, err := provKubeClient.Clusters(cluster.ObjectMeta.Namespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + cluster.ObjectMeta.Name,
			TimeoutSeconds: &shepherddefaults.WatchTimeoutSeconds,
		})

		if err != nil {
			return err
		}

		client, err = client.ReLogin()
		if err != nil {
			return err
		}

		err = client.Steve.SteveType(shepherdclusters.ProvisioningSteveResourceType).Delete(cluster)
		if err != nil {
			return err
		}

		return wait.WatchWait(watchInterface, func(event watch.Event) (ready bool, err error) {
			cluster := event.Object.(*apisV1.Cluster)
			if event.Type == watch.Error {
				return false, fmt.Errorf("there was an error deleting cluster")
			} else if event.Type == watch.Deleted {
				return true, nil
			} else if cluster == nil {
				return true, nil
			}
			return false, nil
		})
	})

	return cluster, nil
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
