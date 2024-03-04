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
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/moio/scalability-tests/pkg/kubectl"
	"github.com/moio/scalability-tests/pkg/terraform"
	"github.com/urfave/cli/v2"
)

const (
	argTerraformDir         = "tf-dir"
	argTerraformVarFile     = "tf-var-file"
	argTerraformParallelism = "tf-parallelism"
	argTerraformSkip        = "tf-skip"
	argChartDir             = "chart-dir"
	argChartRancherReplicas = "rancher-replicas"
	baseDir                 = ""
	adminPassword           = "adminadminadmin"
)

var (
	tfProvider           = "k3d"
	chartDir             string
	chartRancherReplicas int
)

func main() {
	app := &cli.App{
		Usage:     "test Rancher at scale",
		Copyright: "(c) 2024 SUSE LLC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    argTerraformDir,
				Value:   filepath.Join(baseDir, "terraform", "main", "k3d"),
				Usage:   "terraform working directory",
				EnvVars: []string{"TERRAFORM_WORK_DIR"},
				Action: func(cCtx *cli.Context, tfDir string) error {
					if len(tfDir) > 0 {
						tfProvider = filepath.Base(tfDir)
					}
					return nil
				},
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "setup",
				Usage:       "Deploys the test environment",
				Description: "prepares the test environment deploying the clusters and installing the required charts",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    argTerraformVarFile,
						Value:   "",
						Usage:   "terraform variable definition file",
						EnvVars: []string{"TERRAFORM_VAR_FILE"},
					},
					&cli.IntFlag{
						Name:  argTerraformParallelism,
						Value: 10,
						Usage: "terraform 'parallelism': number of concurrent threads",
					},
					&cli.BoolFlag{
						Name:  argTerraformSkip,
						Value: false,
						Usage: "skip terraform apply, start from current terraform state",
					},
					&cli.StringFlag{
						Name:        argChartDir,
						Value:       filepath.Join(baseDir, "charts"),
						Usage:       "charts directory",
						Destination: &chartDir,
					},
					&cli.IntFlag{
						Name:        argChartRancherReplicas,
						Usage:       "number of Rancher replicas",
						DefaultText: "1 for k3d tf provider, otherwise 3",
						Destination: &chartRancherReplicas,
					},
				},
				Action: actionCmdSetup,
			},
			{
				Name:        "state",
				Usage:       "Retrieves information of the deployed clusters",
				Description: "print out the state of the provisioned clusters",
				Action:      actionCmdState,
			},
			{
				Name:        "teardown",
				Aliases:     []string{"destroy"},
				Usage:       "Tears down the test environment (all the clusters)",
				Description: "destroy all the provisioned clusters",
				Action:      actionCmdDestroy,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func actionCmdState(cCtx *cli.Context) error {
	tf := new(terraform.Terraform)

	if err := tf.Init(cCtx.String(argTerraformDir), false); err != nil {
		return err
	}

	clusters, err := tf.OutputClustersJson()
	if err != nil {
		return err
	}

	fmt.Println(clusters)
	return nil
}

func actionCmdDestroy(cCtx *cli.Context) error {
	tf := new(terraform.Terraform)

	if err := tf.Init(cCtx.String(argTerraformDir), true); err != nil {
		return err
	}

	return tf.Destroy(cCtx.String(argTerraformVarFile))
}

func actionCmdSetup(cCtx *cli.Context) error {
	// Terraform
	tf := new(terraform.Terraform)
	tf.Threads = cCtx.Int(argTerraformParallelism)

	if err := tf.Init(cCtx.String(argTerraformDir), true); err != nil {
		return err
	}
	terraformVersionPrint(tf)

	doTerraformApply := !cCtx.Bool(argTerraformSkip)
	if doTerraformApply {
		if err := tf.Apply(cCtx.String(argTerraformVarFile)); err != nil {
			return err
		}
	}

	clusters, err := tf.OutputClusters()
	if err != nil {
		return err
	}

	// Helm charts
	tester := clusters["tester"]

	if err := chartInstallMimir(&tester); err != nil {
		return err
	}
	if err := chartInstallK6Files(&tester); err != nil {
		return err
	}
	if err := chartInstallGrafanaDashboard(&tester); err != nil {
		return err
	}
	if err := chartInstallGrafana(&tester); err != nil {
		return err
	}

	upstream := clusters["upstream"]
	// TODO: implement "importImage" function

	if err := chartInstallCertManager(&upstream); err != nil {
		return err
	}
	if !cCtx.IsSet(argChartRancherReplicas) {
		chartRancherReplicas = 3
		if isProviderK3d() {
			chartRancherReplicas = 1
		}
	}
	if err := chartInstallRancher(&upstream, int(chartRancherReplicas)); err != nil {
		return err
	}
	if err := chartInstallRancherIngress(&upstream); err != nil {
		return err
	}
	if err := chartInstallRancherMonitoring(&upstream, isProviderK3d()); err != nil {
		return err
	}
	if err := chartInstallCgroupsExporter(&upstream); err != nil {
		return err
	}

	// Import downstream clusters
	// Wait Rancher Deployment to be complete, or importing downstream clusters may fail
	if err := kubectl.WaitRancher(upstream.Kubeconfig); err != nil {
		return err
	}
	if err := importDownstreamClusters(clusters); err != nil {
		return err
	}

	return nil
}

func importDownstreamClusters(clusters map[string]terraform.Cluster) error {

	log.Print("Import downstream clusters")

	if err := importDownstreamClustersRancherSetup(clusters); err != nil {
		return err
	}

	buffer := 10
	clustersChan := make(chan string, buffer)
	errorChan := make(chan error)
	clustersCount := 0

	for clusterName := range clusters {
		if !strings.HasPrefix(clusterName, "downstream") {
			continue
		}
		clustersCount++
		go importDownstreamClusterDo(clusters, clusterName, clustersChan, errorChan)
	}

	for {
		select {
		case err := <-errorChan:
			return err
		case completed := <-clustersChan:
			log.Printf("Cluster %q imported successfully.\n", completed)
			clustersCount--
			if clustersCount == 0 {
				return nil
			}
		}
	}
}

func importDownstreamClusterDo(clusters map[string]terraform.Cluster, clusterName string, ch chan<- string, errCh chan<- error) {
	log.Print("Import cluster " + clusterName)
	yamlFile, err := os.CreateTemp("", "scli-"+clusterName+"-*.yaml")
	if err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	defer os.Remove(yamlFile.Name())
	defer yamlFile.Close()

	clusterId, err := importClustersDownstreamGetYAML(clusters, clusterName, yamlFile)
	if err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	downstream, ok := clusters[clusterName]
	if !ok {
		err := fmt.Errorf("error: cannot find access data for cluster %q", clusterName)
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	if err := kubectl.Apply(downstream.Kubeconfig, yamlFile.Name()); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}

	if err := kubectl.WaitForReadyCondition(clusters["upstream"].Kubeconfig,
		"clusters.management.cattle.io", clusterId, "", 10); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	if err := kubectl.WaitForReadyCondition(clusters["upstream"].Kubeconfig,
		"cluster.fleet.cattle.io", clusterName, "fleet-default", 10); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	if err := chartInstallRancherMonitoring(&downstream, true); err != nil {
		errCh <- fmt.Errorf("%s import failed: %w", clusterName, err)
		return
	}
	ch <- clusterName
}

func importDownstreamClustersRancherSetup(clusters map[string]terraform.Cluster) error {
	cliTester := kubectl.Client{}
	tester := clusters["tester"]
	upstream := clusters["upstream"]
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return err
	}

	if err := cliTester.Init(tester.Kubeconfig); err != nil {
		return err
	}

	downstreamClusters := []string{}
	for clusterName := range clusters {
		if strings.HasPrefix(clusterName, "downstream") {
			downstreamClusters = append(downstreamClusters, clusterName)
		}
	}
	importedClusterNames := strings.Join(downstreamClusters, ",")

	envVars := map[string]string{
		"BASE_URL":               upstreamAdd.Public.HTTPSURL,
		"BOOTSTRAP_PASSWORD":     "admin",
		"PASSWORD":               adminPassword,
		"IMPORTED_CLUSTER_NAMES": importedClusterNames,
	}

	if err := cliTester.K6run(envVars, nil, "k6/rancher_setup.js", true, false); err != nil {
		return err
	}
	return nil
}

func importClustersDownstreamGetYAML(clusters map[string]terraform.Cluster, name string, yamlFile *os.File) (clusterId string, err error) {
	var status map[string]interface{}

	upstream := clusters["upstream"]
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return
	}

	cliUpstream := kubectl.Client{}
	if err = cliUpstream.Init(upstream.Kubeconfig); err != nil {
		return
	}
	namespace := "fleet-default"
	resource := "clusters"
	if status, err = cliUpstream.GetStatus("provisioning.cattle.io", "v1", resource, name, namespace); err != nil {
		return
	}
	clusterId, ok := status["clusterName"].(string)
	if !ok {
		err = fmt.Errorf("error accessing %s/%s %s: no valid 'clusterName' in 'Status'", namespace, name, resource)
		return
	}

	name = "default-token"
	namespace = clusterId
	resource = "clusterregistrationtokens"
	if status, err = cliUpstream.GetStatus("management.cattle.io", "v3", resource, name, namespace); err != nil {
		return
	}
	token, ok := status["token"].(string)
	if !ok {
		err = fmt.Errorf("error accessing %s/%s %s: no valid 'token' in 'Status'", namespace, name, resource)
		return
	}

	url := fmt.Sprintf("%s/v3/import/%s_%s.yaml", upstreamAdd.Local.HTTPSURL, token, clusterId)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(yamlFile, resp.Body)
	if err != nil {
		return
	}
	if err = yamlFile.Sync(); err != nil {
		return
	}

	return
}

func isProviderK3d() bool {
	return tfProvider == "k3d"
}

func terraformVersionPrint(tf *terraform.Terraform) error {
	ver, providers, err := tf.Version()
	if err != nil {
		return err
	}

	log.Printf("Terraform version: %s", ver)
	log.Printf("provider list:")
	for prov, ver := range providers {
		log.Printf("- %s (%s)", prov, ver)
	}
	return nil
}
