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

package main

import (
	"fmt"

	"github.com/moio/scalability-tests/pkg/kubectl"
	"github.com/moio/scalability-tests/pkg/tofu"
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
