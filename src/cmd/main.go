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
	"strconv"
	"strings"

	"github.com/moio/scalability-tests/pkg/docker"
	"github.com/moio/scalability-tests/pkg/k3d"
	"github.com/moio/scalability-tests/pkg/kubectl"
	"github.com/moio/scalability-tests/pkg/tofu"
	"github.com/urfave/cli/v2"
)

const (
	argTofuDir                       = "tf-dir"
	argTofuVarFile                   = "tf-var-file"
	argTofuParallelism               = "tf-parallelism"
	argTofuSkip                      = "tf-skip"
	argChartDir                      = "chart-dir"
	argChartRancherReplicas          = "rancher-replicas"
	argChartSkipDownstreamMonitoring = "skip-downstream-monitoring"
	argLoadConfigMapCnt              = "config-maps"
	argLoadProjectCnt                = "projects"
	argLoadRoleCnt                   = "roles"
	argLoadSecretCnt                 = "secrets"
	argLoadUserCnt                   = "users"
	baseDir                          = ""
	adminPassword                    = "adminadminadmin"
)

var (
	chartDir                 string
	chartRancherReplicas     int
	skipDownstreamMonitoring bool = true
)

func main() {
	app := &cli.App{
		Usage:     "test Rancher at scale",
		Copyright: "(c) 2024 SUSE LLC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    argTofuDir,
				Value:   filepath.Join(baseDir, "tofu", "main", "k3d"),
				Usage:   "tofu working directory",
				EnvVars: []string{"TOFU_WORK_DIR"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "setup",
				Usage:       "Deploys the test environment",
				Description: "prepares the test environment deploying the clusters and installing the required charts",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    argTofuVarFile,
						Value:   "",
						Usage:   "tofu variable definition file",
						EnvVars: []string{"TOFU_VAR_FILE"},
					},
					&cli.IntFlag{
						Name:  argTofuParallelism,
						Value: 10,
						Usage: "tofu 'parallelism': number of concurrent threads",
					},
					&cli.BoolFlag{
						Name:  argTofuSkip,
						Value: false,
						Usage: "skip tofu apply, start from current tofu state",
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
						Value:       1,
						Destination: &chartRancherReplicas,
					},
					&cli.BoolFlag{
						Name:        argChartSkipDownstreamMonitoring,
						Value:       false,
						Usage:       "skip installing rancher-monitoring chart on downstream clusters",
						Destination: &skipDownstreamMonitoring,
					},
				},
				Action: actionCmdSetup,
			},
			{
				Name:        "get-access",
				Usage:       "Retrieves information to access the deployed clusters",
				Description: "print out links and access information for the deployed clusters",
				Action:      actionCmdGetAccess,
			},
			{
				Name:        "teardown",
				Aliases:     []string{"destroy"},
				Usage:       "Tears down the test environment (all the clusters)",
				Description: "destroy all the provisioned clusters",
				Action:      actionCmdDestroy,
			},
			{
				Name:        "load",
				Usage:       "Creates K8s resources on upstream and downstream clusters",
				Description: "Loads ConfigMaps and Secrets on all the deployed K8s cluster; Roles, Users and Projects on the Rancher cluster",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        argChartDir,
						Value:       filepath.Join(baseDir, "charts"),
						Usage:       "charts directory",
						Destination: &chartDir,
					},
					&cli.IntFlag{
						Name:    argLoadConfigMapCnt,
						Value:   1000,
						Usage:   "number of ConfigMap resources to create",
						EnvVars: []string{"SCLI_CONFIGMAP_COUNT"},
					},
					&cli.IntFlag{
						Name:    argLoadSecretCnt,
						Value:   1000,
						Usage:   "number of Secret resources to create",
						EnvVars: []string{"SCLI_SECRET_COUNT"},
					},
					&cli.IntFlag{
						Name:    argLoadRoleCnt,
						Value:   10,
						Usage:   "number of Role resources to create",
						EnvVars: []string{"SCLI_ROLE_COUNT"},
					},
					&cli.IntFlag{
						Name:    argLoadUserCnt,
						Value:   5,
						Usage:   "number of User resources to create",
						EnvVars: []string{"SCLI_USER_COUNT"},
					},
					&cli.IntFlag{
						Name:    argLoadProjectCnt,
						Value:   20,
						Usage:   "number of Project resources to create",
						EnvVars: []string{"SCLI_PROJECT_COUNT"},
					},
				},
				Action: actionCmdLoad,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func actionCmdDestroy(cCtx *cli.Context) error {
	tf := new(tofu.Tofu)

	if err := tf.Init(cCtx.String(argTofuDir), true); err != nil {
		return err
	}

	return tf.Destroy(cCtx.String(argTofuVarFile))
}

func actionCmdSetup(cCtx *cli.Context) error {
	// Tofu
	tf := new(tofu.Tofu)
	tf.Threads = cCtx.Int(argTofuParallelism)

	if err := tf.Init(cCtx.String(argTofuDir), true); err != nil {
		return err
	}
	tofuVersionPrint(tf)

	apply := !cCtx.Bool(argTofuSkip)
	if apply {
		if err := tf.Apply(cCtx.String(argTofuVarFile)); err != nil {
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
	err = importImageIntoK3d(tf, "rancher/rancher:"+rancherImageTag, upstream)
	if err != nil {
		return err
	}

	if err := chartInstallCertManager(&upstream); err != nil {
		return err
	}
	if err := chartInstallRancher(&upstream, int(chartRancherReplicas)); err != nil {
		return err
	}
	if err := chartInstallRancherIngress(&upstream); err != nil {
		return err
	}
	if err := chartInstallRancherMonitoring(&upstream, tf.IsK3d()); err != nil {
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
	if err := importDownstreamClusters(tf, clusters); err != nil {
		return err
	}

	return actionCmdGetAccess(cCtx)
}

func actionCmdLoad(cCtx *cli.Context) error {
	tf := new(tofu.Tofu)

	if err := tf.Init(cCtx.String(argTofuDir), false); err != nil {
		return err
	}

	clusters, err := tf.OutputClusters()
	if err != nil {
		return err
	}

	// Refresh k6 files
	tester := clusters["tester"]
	if err := chartInstallK6Files(&tester); err != nil {
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
		if err := loadConfigMapAndSecrets(cCtx, cliTester, clusterName, clusterData); err != nil {
			return err
		}
	}

	// Create Users and Roles
	if err := loadRolesAndUsers(cCtx, cliTester, "upstream", clusters["upstream"]); err != nil {
		return err
	}
	// Create Projects
	if err := loadProjects(cCtx, cliTester, "upstream", clusters["upstream"]); err != nil {
		return err
	}
	return nil
}

func actionCmdGetAccess(cCtx *cli.Context) error {
	tf := new(tofu.Tofu)

	if err := tf.Init(cCtx.String(argTofuDir), false); err != nil {
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

	printAccessDetails("UPSTREAM", upstream, rancherURL)
	for name, downstream := range downstreams {
		printAccessDetails(strings.ToUpper(name), downstream, "")
	}
	printAccessDetails("TESTER", tester, "")

	return nil
}

func printAccessDetails(name string, cluster tofu.Cluster, rancherURL string) {
	fmt.Printf("*** %s CLUSTER\n", name)
	if rancherURL != "" {
		fmt.Printf("    Rancher UI: %s (admin/%s)\n", rancherURL, adminPassword)
	}
	fmt.Println("    Kubernetes API:")
	fmt.Printf("export KUBECONFIG=%q\n", cluster.Kubeconfig)
	fmt.Printf("kubectl config use-context %q\n", cluster.Context)
	for node, command := range cluster.NodeAccessCommands {
		fmt.Printf("    Node %s: %q\n", node, command)
	}
	fmt.Println()
}

func importDownstreamClusters(tf *tofu.Tofu, clusters map[string]tofu.Cluster) error {

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
		go importDownstreamClusterDo(tf, clusters, clusterName, clustersChan, errorChan)
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

func importDownstreamClusterDo(tf *tofu.Tofu, clusters map[string]tofu.Cluster, clusterName string, ch chan<- string, errCh chan<- error) {
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
	err = importImageIntoK3d(tf, "rancher/rancher-agent:"+rancherImageTag, downstream)
	if err != nil {
		errCh <- fmt.Errorf("%s downstream k3d image import failed: %w", clusterName, err)
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
	if !skipDownstreamMonitoring {
		if err := chartInstallRancherMonitoring(&downstream, true); err != nil {
			errCh <- fmt.Errorf("downstream monitoring installation on cluster %s failed: %w", clusterName, err)
			return
		}
	}
	ch <- clusterName
}

func importDownstreamClustersRancherSetup(clusters map[string]tofu.Cluster) error {
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
	if len(downstreamClusters) == 0 {
		return nil
	}
	importedClusterNames := strings.Join(downstreamClusters, ",")

	envVars := map[string]string{
		"BASE_URL":               upstreamAdd.Public.HTTPSURL,
		"BOOTSTRAP_PASSWORD":     "admin",
		"PASSWORD":               adminPassword,
		"IMPORTED_CLUSTER_NAMES": importedClusterNames,
	}

	if err := cliTester.K6run("rancher-setup", "k6/rancher_setup.js", envVars, nil, true, false); err != nil {
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

func loadConfigMapAndSecrets(cCtx *cli.Context, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	configMapCount := strconv.Itoa(cCtx.Int(argLoadConfigMapCnt))
	secretCount := strconv.Itoa(cCtx.Int(argLoadSecretCnt))

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

func loadRolesAndUsers(cCtx *cli.Context, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	roleCount := strconv.Itoa(cCtx.Int(argLoadRoleCnt))
	userCount := strconv.Itoa(cCtx.Int(argLoadUserCnt))
	clusterAdd, err := getAppAddressFor(clusterData)
	if err != nil {
		return fmt.Errorf("failed loading Roles and Users on cluster %q: %w", clusterName, err)
	}
	envVars := map[string]string{
		"BASE_URL":   clusterAdd.Public.HTTPSURL,
		"USERNAME":   "admin",
		"PASSWORD":   adminPassword,
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

func loadProjects(cCtx *cli.Context, cli *kubectl.Client, clusterName string, clusterData tofu.Cluster) error {
	projectCount := strconv.Itoa(cCtx.Int(argLoadProjectCnt))
	clusterAdd, err := getAppAddressFor(clusterData)
	if err != nil {
		return fmt.Errorf("failed loading Projects on cluster %q: %w", clusterName, err)
	}
	envVars := map[string]string{
		"BASE_URL":      clusterAdd.Public.HTTPSURL,
		"USERNAME":      "admin",
		"PASSWORD":      adminPassword,
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

func tofuVersionPrint(tofu *tofu.Tofu) error {
	ver, providers, err := tofu.Version()
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
