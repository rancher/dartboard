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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/moio/scalability-tests/internal/dart"
	"github.com/moio/scalability-tests/internal/docker"
	"github.com/moio/scalability-tests/internal/k3d"
	"github.com/moio/scalability-tests/internal/kubectl"
	"github.com/moio/scalability-tests/internal/tofu"
	"github.com/urfave/cli/v2"
)

const (
	argDart      = "dart"
	argSkipApply = "skip-apply"
)

func main() {
	app := &cli.App{
		Usage:     "setup and test Rancher (at scale if needed)",
		Copyright: "(c) 2024 SUSE LLC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    argDart,
				Aliases: []string{"d"},
				Value:   filepath.Join("darts", "k3d_full.yaml"),
				Usage:   "dart to use",
				EnvVars: []string{"DART"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "apply",
				Usage:       "Deploys Kubernetes clusters for testing",
				Description: "runs `tofu apply` to prepare infrastructure and Kubernetes clusters for tests",
				Action:      cmdApply,
			},
			{
				Name:        "deploy",
				Usage:       "Deploys Rancher and other charts on top of clusters",
				Description: "prepares the test environment installing all required charts",
				Action:      cmdDeploy,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  argSkipApply,
						Value: false,
						Usage: "skip `tofu apply`, assume apply was already called",
					},
				},
			},
			{
				Name:        "load",
				Usage:       "Creates K8s resources on upstream and downstream clusters",
				Description: "Loads ConfigMaps and Secrets on all the deployed K8s cluster; Roles, Users and Projects on the Rancher cluster",
				Action:      cmdLoad,
			},
			{
				Name:        "get-access",
				Usage:       "Retrieves information to access the deployed clusters",
				Description: "print out links and access information for the deployed clusters",
				Action:      cmdGetAccess,
			},
			{
				Name:        "destroy",
				Usage:       "Tears down the test environment (all the clusters)",
				Description: "runs `tofu destroy` to destroy all the provisioned clusters",
				Action:      cmdDestroy,
			},
			{
				Name:        "redeploy",
				Usage:       "Tears down the test environment (all the clusters) and redeploys them from scratch",
				Description: "runs `tofu destroy` and then deploys all the provisioned clusters",
				Action:      cmdRedeploy,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func cmdApply(cli *cli.Context) error {
	tf, _, err := prepare(cli)
	if err != nil {
		return err
	}

	if err = tofuVersionPrint(cli.Context, tf); err != nil {
		return err
	}
	if err = tf.Apply(cli.Context); err != nil {
		return err
	}

	return cmdGetAccess(cli)
}

func cmdDeploy(cli *cli.Context) error {
	// Tofu
	tf, r, err := prepare(cli)
	if err != nil {
		return err
	}

	if !cli.Bool(argSkipApply) {
		if err = tofuVersionPrint(cli.Context, tf); err != nil {
			return err
		}
		if err = tf.Apply(cli.Context); err != nil {
			return err
		}
	}

	clusters, err := tf.OutputClusters(cli.Context)
	if err != nil {
		return err
	}

	// Helm charts
	tester := clusters["tester"]

	if err = chartInstall(tester.Kubeconfig, chart{"k6-files", "tester", "k6-files"}, ""); err != nil {
		return err
	}
	if err = chartInstall(tester.Kubeconfig, chart{"mimir", "tester", "mimir"}, ""); err != nil {
		return err
	}
	if err = chartInstall(tester.Kubeconfig, chart{"grafana-dashboards", "tester", "grafana-dashboards"}, ""); err != nil {
		return err
	}
	if err = chartInstallGrafana(r, &tester); err != nil {
		return err
	}

	upstream := clusters["upstream"]
	rancherVersion := r.ChartVariables.RancherVersion
	rancherImageTag := "v" + rancherVersion
	if r.ChartVariables.RancherImageTagOverride != "" {
		rancherImageTag = r.ChartVariables.RancherImageTagOverride
		err = importImageIntoK3d(tf, "rancher/rancher:"+rancherImageTag, upstream)
		if err != nil {
			return err
		}
	}

	if err = chartInstallCertManager(r, &upstream); err != nil {
		return err
	}
	if err = chartInstallRancher(r, rancherImageTag, &upstream); err != nil {
		return err
	}
	if err = chartInstallRancherIngress(&upstream); err != nil {
		return err
	}
	if err = chartInstallRancherMonitoring(r, &upstream, tf.IsK3d()); err != nil {
		return err
	}
	if err = chartInstallCgroupsExporter(&upstream); err != nil {
		return err
	}

	// Import downstream clusters
	// Wait Rancher Deployment to be complete, or importing downstream clusters may fail
	if err = kubectl.WaitRancher(upstream.Kubeconfig); err != nil {
		return err
	}
	if err = importDownstreamClusters(r, rancherImageTag, tf, clusters); err != nil {
		return err
	}

	return cmdGetAccess(cli)
}

func cmdLoad(cli *cli.Context) error {
	tf, r, err := prepare(cli)
	if err != nil {
		return err
	}

	clusters, err := tf.OutputClusters(cli.Context)
	if err != nil {
		return err
	}

	// Refresh k6 files
	tester := clusters["tester"]
	if err := chartInstall(tester.Kubeconfig, chart{"k6-files", "tester", "k6-files"}, ""); err != nil {
		return err
	}

	// Create ConfigMaps and Secrets
	cliTester := &kubectl.Client{}
	if err := cliTester.Init(tester.Kubeconfig); err != nil {
		return err
	}

	// Create ConfigMaps and Secrets on Rancher and all the downstream clusters
	for clusterName, clusterData := range clusters {
		// NOTE: we may change the condition with 'cluster == "tester"', but better to stay on the safe side
		if clusterName != "upstream" && !strings.HasPrefix(clusterName, "downstream") {
			continue
		}
		if err := loadConfigMapAndSecrets(r, cliTester, clusterName, clusterData); err != nil {
			return err
		}
	}

	// Create Users and Roles
	if err := loadRolesAndUsers(r, cliTester, "upstream", clusters["upstream"]); err != nil {
		return err
	}
	// Create Projects
	if err := loadProjects(r, cliTester, "upstream", clusters["upstream"]); err != nil {
		return err
	}
	return nil
}

func cmdGetAccess(cli *cli.Context) error {
	tf, r, err := prepare(cli)
	if err != nil {
		return err
	}

	clusters, err := tf.OutputClusters(cli.Context)
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

func cmdDestroy(cli *cli.Context) error {
	tf, _, err := prepare(cli)
	if err != nil {
		return err
	}

	return tf.Destroy(cli.Context)
}

func cmdRedeploy(c *cli.Context) error {
	err := cmdDestroy(c)
	if err != nil {
		return err
	}

	return cmdDeploy(c)
}

func prepare(cli *cli.Context) (*tofu.Tofu, *dart.Dart, error) {
	rp := cli.String(argDart)
	r, err := dart.Parse(rp)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("Using dart: %s\n", rp)
	fmt.Printf("Terraform main directory: %s\n", r.TofuMainDirectory)

	tf, err := tofu.New(cli.Context, r.TofuVariables, r.TofuMainDirectory, r.TofuParallelism, true)
	if err != nil {
		return nil, nil, err
	}
	return tf, r, nil
}

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

func importDownstreamClusters(r *dart.Dart, rancherImageTag string, tf *tofu.Tofu, clusters map[string]tofu.Cluster) error {

	log.Print("Import downstream clusters")

	if err := importDownstreamClustersRancherSetup(r, clusters); err != nil {
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
		go importDownstreamClusterDo(r, rancherImageTag, tf, clusters, clusterName, clustersChan, errorChan)
	}

	for {
		if clustersCount == 0 {
			return nil
		}
		select {
		case err := <-errorChan:
			return err
		case completed := <-clustersChan:
			log.Printf("Cluster %q imported successfully.\n", completed)
			clustersCount--
		}
	}
}

func importDownstreamClusterDo(r *dart.Dart, rancherImageTag string, tf *tofu.Tofu, clusters map[string]tofu.Cluster, clusterName string, ch chan<- string, errCh chan<- error) {
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
	if r.ChartVariables.RancherImageTagOverride != "" {
		err = importImageIntoK3d(tf, "rancher/rancher-agent:"+rancherImageTag, downstream)
		if err != nil {
			errCh <- fmt.Errorf("%s downstream k3d image import failed: %w", clusterName, err)
			return
		}
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
	if r.ChartVariables.DownstreamRancherMonitoring {
		if err := chartInstallRancherMonitoring(r, &downstream, true); err != nil {
			errCh <- fmt.Errorf("downstream monitoring installation on cluster %s failed: %w", clusterName, err)
			return
		}
	}
	ch <- clusterName
}

func importDownstreamClustersRancherSetup(r *dart.Dart, clusters map[string]tofu.Cluster) error {
	cliTester := kubectl.Client{}
	tester := clusters["tester"]
	upstream := clusters["upstream"]
	upstreamAdd, err := getAppAddressFor(upstream)
	if err != nil {
		return err
	}

	if err = cliTester.Init(tester.Kubeconfig); err != nil {
		return err
	}

	downstreamClusters := []string{}
	for clusterName := range clusters {
		if strings.HasPrefix(clusterName, "downstream") {
			downstreamClusters = append(downstreamClusters, clusterName)
		}
	}
	if len(downstreamClusters) == 0 {
		return nil
	}
	importedClusterNames := strings.Join(downstreamClusters, ",")

	envVars := map[string]string{
		"BASE_URL":               upstreamAdd.Public.HTTPSURL,
		"BOOTSTRAP_PASSWORD":     "admin",
		"PASSWORD":               r.ChartVariables.AdminPassword,
		"IMPORTED_CLUSTER_NAMES": importedClusterNames,
	}

	if err = cliTester.K6run("rancher-setup", "k6/rancher_setup.js", envVars, nil, true, false); err != nil {
		return err
	}
	return nil
}

func importClustersDownstreamGetYAML(clusters map[string]tofu.Cluster, name string, yamlFile *os.File) (clusterId string, err error) {
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

func loadConfigMapAndSecrets(r *dart.Dart, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	configMapCount := strconv.Itoa(r.TestVariables.TestConfigMaps)
	secretCount := strconv.Itoa(r.TestVariables.TestSecrets)

	envVars := map[string]string{
		"BASE_URL":         clusterData.PrivateKubernetesAPIURL,
		"KUBECONFIG":       clusterData.Kubeconfig,
		"CONTEXT":          clusterData.Context,
		"CONFIG_MAP_COUNT": configMapCount,
		"SECRET_COUNT":     secretCount,
	}
	tags := map[string]string{
		"cluster":    clusterName,
		"test":       "create_k8s_resources.js",
		"ConfigMaps": configMapCount,
		"Secrets":    secretCount,
	}

	log.Printf("Load resources on cluster %q (#ConfigMaps: %s, #Secrets: %s)\n", clusterName, configMapCount, secretCount)
	if err := cli.K6run("create-k8s-resources", "k6/create_k8s_resources.js", envVars, tags, true, false); err != nil {
		return fmt.Errorf("failed loading ConfigMaps and Secrets on cluster %q: %w", clusterName, err)
	}
	return nil
}

func loadRolesAndUsers(r *dart.Dart, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	roleCount := strconv.Itoa(r.TestVariables.TestRoles)
	userCount := strconv.Itoa(r.TestVariables.TestUsers)
	clusterAdd, err := getAppAddressFor(clusterData)
	if err != nil {
		return fmt.Errorf("failed loading Roles and Users on cluster %q: %w", clusterName, err)
	}
	envVars := map[string]string{
		"BASE_URL":   clusterAdd.Public.HTTPSURL,
		"USERNAME":   "admin",
		"PASSWORD":   r.ChartVariables.AdminPassword,
		"ROLE_COUNT": roleCount,
		"USER_COUNT": userCount,
	}
	tags := map[string]string{
		"cluster": clusterName,
		"test":    "create_roles_users.mjs",
		"Roles":   roleCount,
		"Users":   userCount,
	}

	log.Printf("Load resources on cluster %q (#Roles: %s, #Users: %s)\n", clusterName, roleCount, userCount)

	if err := cli.K6run("create-roles-users", "k6/create_roles_users.js", envVars, tags, true, false); err != nil {
		return fmt.Errorf("failed loading Roles and Users on cluster %q: %w", clusterName, err)
	}
	return nil
}

func loadProjects(r *dart.Dart, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	projectCount := strconv.Itoa(r.TestVariables.TestProjects)
	clusterAdd, err := getAppAddressFor(clusterData)
	if err != nil {
		return fmt.Errorf("failed loading Projects on cluster %q: %w", clusterName, err)
	}
	envVars := map[string]string{
		"BASE_URL":      clusterAdd.Public.HTTPSURL,
		"USERNAME":      "admin",
		"PASSWORD":      r.ChartVariables.AdminPassword,
		"PROJECT_COUNT": projectCount,
	}
	tags := map[string]string{
		"cluster":  clusterName,
		"test":     "create_projects.mjs",
		"Projects": projectCount,
	}

	log.Printf("Load resources on cluster %q (#Projects: %s)\n", clusterName, projectCount)

	if err := cli.K6run("create-projects", "k6/create_projects.js", envVars, tags, true, false); err != nil {
		return fmt.Errorf("failed loading Projects on cluster %q: %w", clusterName, err)
	}
	return nil
}

func tofuVersionPrint(ctx context.Context, tofu *tofu.Tofu) error {
	ver, providers, err := tofu.Version(ctx)
	if err != nil {
		return err
	}

	log.Printf("OpenTofu version: %s", ver)
	log.Printf("provider list:")
	for prov, ver := range providers {
		log.Printf("- %s (%s)", prov, ver)
	}
	return nil
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
			err = k3d.ImageImport(cluster, images[0])
			if err != nil {
				return err
			}
		}
	}
	return nil
}
