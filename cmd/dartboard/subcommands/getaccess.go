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
	"strings"

	"github.com/rancher/dartboard/internal/tofu"
	cli "github.com/urfave/cli/v2"
)

func GetAccess(cli *cli.Context) error {
	tf, r, err := prepare(cli)
	if err != nil {
		return err
	}

	clusters, err := tf.OutputClusters()
	if err != nil {
		return err
	}

	upstream := clusters["upstream"]
	tester := clusters["tester"]

	downstreams := make(map[string]tofu.Cluster)
	for k, v := range clusters {
		if strings.HasPrefix(k, "downstream") {
			downstreams[k] = v
		}
	}

	upstreamAddresses, err := getAppAddressFor(upstream)
	rancherURL := ""
	if err == nil {
		rancherURL = upstreamAddresses.Local.HTTPSURL
	} else {
		fmt.Printf("Error getting application addresses for cluster upstream: %v\n", err)
	}

	fmt.Println("\n\n\n*** ACCESS DETAILS")
	fmt.Println()

	printAccessDetails(r, "UPSTREAM", upstream, rancherURL)
	for name, downstream := range downstreams {
		printAccessDetails(r, strings.ToUpper(name), downstream, "")
	}
	printAccessDetails(r, "TESTER", tester, "")

	return nil
}
