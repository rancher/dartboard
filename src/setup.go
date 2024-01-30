package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/moio/scalability-tests/pkg/terraform"
)

const (
	baseDir       = ".."
	adminPassword = "adminadminadmin"
)

func main() {
	// Step 1: Terraform
	tf := new(terraform.Terraform)

	if err := tf.Init(terraformDir()); err != nil {
		log.Fatal(err)
	}
	terraformVersionPrint(tf)

	if err := tf.Apply(os.Getenv("TERRAFORM_VAR_FILE")); err != nil {
		log.Fatal(err)
	}

	clusters, err := tf.OutputClusters()
	if err != nil {
		log.Fatal(err)
	}

	// Step 3: Helm charts
	// tester cluster

	tester := clusters["tester"]
	fmt.Printf("\n%+v\n\n", tester)
	// TODO: deal with the type assertion failure instead of triggering a panic

	if err := chartInstallMimir(&tester); err != nil {
		log.Fatal(err)
	}
	if err := chartInstallK6Files(&tester); err != nil {
		log.Fatal(err)
	}
	if err := chartInstallGrafanaDashboard(&tester); err != nil {
		log.Fatal(err)
	}
	if err := chartInstallGrafana(&tester); err != nil {
		log.Fatal(err)
	}

	// upstream cluster
	upstream := clusters["upstream"]

	// TODO: implement "importImage" function

	if err := chartInstallCertManager(&upstream); err != nil {
		log.Fatal(err)
	}
	if err := chartInstallRancher(&upstream); err != nil {
		log.Fatal(err)
	}

	// rancher-ingress
	if err := chartInstallRancherIngress(&upstream); err != nil {
		log.Fatal(err)
	}

	// rancher-monitoring
	if err := chartInstallRancherMonitoring(&upstream); err != nil {
		log.Fatal(err)
	}

	// cgroups-exporter
	if err := chartInstallCgroupsExporter(&upstream); err != nil {
		log.Fatal(err)
	}
	// rancherURL := upstreamAdd.Local.HTTPSURL
}

func terraformDir() string {
	defaultDir := os.Getenv("TERRAFORM_WORK_DIR")
	if len(defaultDir) == 0 {
		defaultDir = filepath.Join("..", "terraform", "main", "k3d")
	}
	return defaultDir
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
