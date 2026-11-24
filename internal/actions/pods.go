package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	steveclient "github.com/rancher/shepherd/clients/rancher/v1"
	shepherdpods "github.com/rancher/shepherd/extensions/workloads/pods"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	PodResourceSteveType = "pod"
)

// StatusPods is a helper function that uses the steve client to list pods on a namespace for a specific cluster
// and return the statuses in a list of strings
func StatusPodsWithTimeout(client *rancher.Client, clusterID string, timeout time.Duration) []error {
	var (
		podErrors []error
		lastErr   error
		steveType *steveclient.SteveClient
	)

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, timeout, true, func(_ context.Context) (done bool, err error) {
		// emptying pod errors every time we poll so that we don't return stale errors
		podErrors = []error{}

		// The downstream proxy can return 500 (no tunnel session yet) right after import, so retry it.
		if steveType == nil {
			downstreamClient, err := client.Steve.ProxyDownstream(clusterID)
			if err != nil {
				lastErr = err
				return false, nil
			}

			steveType = downstreamClient.SteveType(PodResourceSteveType)
		}

		pods, err := steveType.List(nil)
		if err != nil {
			// not returning the error in this case, as it could cause a false positive if we start polling too early.
			lastErr = err
			// drop the cached client so the next poll re-fetches schemas (pod schema may not have been available yet).
			steveType = nil

			return false, nil
		}

		lastErr = nil

		for _, pod := range pods.Data {
			isReady, err := shepherdpods.IsPodReady(&pod)
			if !isReady {
				// not returning the error in this case, as it could cause a false positive if we start polling too early.
				lastErr = fmt.Errorf("pod %s/%s not ready", pod.Namespace, pod.Name)
				if err != nil {
					lastErr = fmt.Errorf("%w: %v", lastErr, err)
				}

				return false, nil
			}

			if err != nil {
				podErrors = append(podErrors, err)
			}
		}

		return true, nil
	})
	if err != nil {
		if lastErr != nil {
			err = fmt.Errorf("%w (last error: %v)", err, lastErr)
		}

		podErrors = append(podErrors, err)
	}

	return podErrors
}
