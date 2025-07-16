package actions

import (
	"context"
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	shepherdpods "github.com/rancher/shepherd/extensions/workloads/pods"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	PodResourceSteveType = "pod"
)

// StatusPods is a helper function that uses the steve client to list pods on a namespace for a specific cluster
// and return the statuses in a list of strings
func StatusPodsWithTimeout(client *rancher.Client, clusterID string, timeout time.Duration) []error {
	downstreamClient, err := client.Steve.ProxyDownstream(clusterID)
	if err != nil {
		return []error{err}
	}

	var podErrors []error

	steveClient := downstreamClient.SteveType(PodResourceSteveType)
	ctx := context.Background()
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(_ context.Context) (done bool, err error) {
		// emptying pod errors every time we poll so that we don't return stale errors
		podErrors = []error{}

		pods, err := steveClient.List(nil)
		if err != nil {
			// not returning the error in this case, as it could cause a false positive if we start polling too early.
			return false, nil
		}

		for _, pod := range pods.Data {
			isReady, err := shepherdpods.IsPodReady(&pod)
			if !isReady {
				// not returning the error in this case, as it could cause a false positive if we start polling too early.
				return false, nil
			}

			if err != nil {
				podErrors = append(podErrors, err)
			}
		}
		return true, nil
	})

	if err != nil {
		podErrors = append(podErrors, err)
	}

	return podErrors
}
