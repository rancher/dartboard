/*
Copyright © 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package subcommands

import (
	"fmt"
	"path/filepath"

	"github.com/rancher/dartboard/internal/docker"
	"github.com/rancher/dartboard/internal/k3d"
	"github.com/rancher/dartboard/internal/vendored"
	cli "github.com/urfave/cli/v2"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/tofu"
)

const (
	ArgDart        = "dart"
	ArgSkipApply   = "skip-apply"
	ArgSkipCharts  = "skip-charts"
	ArgSkipRefresh = "skip-refresh"
)

type clusterAddress struct {
	Name     string
	HTTPURL  string
	HTTPSURL string
}

type clusterAddresses struct {
	Local  clusterAddress
	Public clusterAddress
}

// prepare prepares tofu for execution and parses a dart file from the command line context
func prepare(cli *cli.Context) (*tofu.Tofu, *dart.Dart, error) {
	dartPath := cli.String(ArgDart)
	d, err := dart.Parse(dartPath)
	if err != nil {
		return nil, nil, err
	}
	tofuWorkspaceStatePath := fmt.Sprintf("%s/%s_config", d.TofuMainDirectory, d.TofuWorkspace)
	absPath, err := filepath.Abs(tofuWorkspaceStatePath)
	if err != nil {
		return nil, nil, err
	}
	d.TofuWorkspaceStatePath = absPath
	fmt.Printf("Using dart: %s\n", dartPath)
	fmt.Printf("OpenTofu main directory: %s\n", d.TofuMainDirectory)
	fmt.Printf("Using Tofu workspace: %s\n", d.TofuWorkspace)

	err = vendored.ExtractBinaries()
	if err != nil {
		return nil, nil, err
	}

	tf, err := tofu.New(d.TofuVariables, d.TofuMainDirectory, d.TofuWorkspace, d.TofuParallelism, true)
	if err != nil {
		return nil, nil, err
	}
	return tf, d, nil
}

// printAccessDetails prints to console addresses and kubeconfig file paths of a cluster for user convenience
func printAccessDetails(r *dart.Dart, name string, cluster tofu.Cluster, rancherURL string) {
	fmt.Printf("*** %s CLUSTER\n", name)
	if rancherURL != "" {
		fmt.Printf("    Rancher UI: %s (admin/%s)\n", rancherURL, r.ChartVariables.AdminPassword)
	}
	fmt.Println("    Kubernetes API:")
	fmt.Printf("export KUBECONFIG=%q\n", cluster.Kubeconfig)
	fmt.Printf("kubectl config use-context %q\n", cluster.Context)
	for node, command := range cluster.NodeAccessCommands {
		fmt.Printf("    Node %s: %q\n", node, command)
	}
	fmt.Println()
}

// getAppAddressFor returns local cluster address data, public cluster address data and an error
func getAppAddressFor(cluster tofu.Cluster) (clusterAddresses, error) {
	add := cluster.AppAddresses

	addresses := clusterAddresses{}

	// ignore error if we are not able to get the Rancher FQDN from the LoadBalancer
	loadBalancerName, _ := kubectl.GetRancherFQDNFromLoadBalancer(cluster.Kubeconfig)

	// addresses meant to be resolved from the machine running Tofu
	// use tunnel if available, otherwise public, otherwise go through the load balancer
	localNetworkName := add.Tunnel.Name
	if len(localNetworkName) == 0 {
		localNetworkName = add.Public.Name
		if len(localNetworkName) == 0 {
			localNetworkName = loadBalancerName
		}
	}
	localNetworkHTTPPort := add.Tunnel.HTTPPort
	if localNetworkHTTPPort == 0 {
		localNetworkHTTPPort = add.Public.HTTPPort
		if localNetworkHTTPPort == 0 {
			localNetworkHTTPPort = 80
		}
	}
	localNetworkHTTPSPort := add.Tunnel.HTTPSPort
	if localNetworkHTTPSPort == 0 {
		localNetworkHTTPSPort = add.Public.HTTPSPort
		if localNetworkHTTPSPort == 0 {
			localNetworkHTTPSPort = 443
		}
	}

	// addresses meant to be resolved from the network running clusters
	// use public if available, otherwise private if available, otherwise go through the load balancer
	clusterNetworkName := add.Public.Name
	if len(clusterNetworkName) == 0 {
		clusterNetworkName = add.Private.Name
		if len(clusterNetworkName) == 0 {
			clusterNetworkName = loadBalancerName
		}
	}
	clusterNetworkHTTPPort := add.Public.HTTPPort
	if clusterNetworkHTTPPort == 0 {
		clusterNetworkHTTPPort = add.Private.HTTPPort
		if clusterNetworkHTTPPort == 0 {
			clusterNetworkHTTPPort = 80
		}
	}
	clusterNetworkHTTPSPort := add.Public.HTTPSPort
	if clusterNetworkHTTPSPort == 0 {
		clusterNetworkHTTPSPort = add.Private.HTTPSPort
		if clusterNetworkHTTPSPort == 0 {
			clusterNetworkHTTPSPort = 443
		}
	}

	addresses.Local.Name = localNetworkName
	addresses.Local.HTTPURL = fmt.Sprintf("http://%s:%d", localNetworkName, localNetworkHTTPPort)
	addresses.Local.HTTPSURL = fmt.Sprintf("https://%s:%d", localNetworkName, localNetworkHTTPSPort)

	addresses.Public.Name = clusterNetworkName
	addresses.Public.HTTPURL = fmt.Sprintf("http://%s:%d", clusterNetworkName, clusterNetworkHTTPPort)
	addresses.Public.HTTPSURL = fmt.Sprintf("https://%s:%d", clusterNetworkName, clusterNetworkHTTPSPort)

	return addresses, nil
}

// importImageIntoK3d uses k3d import to import the specified image in the specified cluster, if such image
// is known by the docker installation. This is for testing custom Rancher images (built via make quick) locally
// in k3d
func importImageIntoK3d(tf *tofu.Tofu, image string, cluster tofu.Cluster) error {
	if tf.IsK3d() {
		images, err := docker.Images(image)
		if err != nil {
			return err
		}

		if len(images) > 0 {
			if err := k3d.ImageImport(cluster.Name, images[0]); err != nil {
				return err
			}
		}
	}
	return nil
}
