package actions

import (
	"fmt"

	"github.com/rancher/shepherd/clients/harvester"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/sirupsen/logrus"

	"github.com/rancher/shepherd/pkg/session"
	harvesteraction "github.com/rancher/tests/interoperability/harvester"
)

type HarvesterImportClient struct {
	client          *rancher.Client
	session         *session.Session
	harvesterClient *harvester.Client
	clusterID       string
}

func NewHarvesterConfig(host, adminToken, adminPassword string, insecure bool) harvester.Config {
	defaultBool := false

	return harvester.Config{
		Host:          host,
		AdminToken:    adminToken,
		AdminPassword: adminPassword,
		Insecure:      &insecure,
		Cleanup:       &defaultBool,
	}
}

// Function to import the Harvester client into the Rancher cluster
func NewHarvesterImportClient(rancherClient *rancher.Client, harvesterConfig *harvester.Config) (*HarvesterImportClient, error) {
	h := HarvesterImportClient{
		client:  rancherClient,
		session: session.NewSession(),
	}

	harvesterClient, err := harvester.NewClientForConfig(harvesterConfig.AdminToken, harvesterConfig, h.session)
	if err != nil {
		return nil, fmt.Errorf("error while setting up Harvester client: %v", err)
	}

	h.harvesterClient = harvesterClient

	h.session.RegisterCleanupFunc(func() error {
		return harvesteraction.ResetHarvesterRegistration(h.harvesterClient)
	})

	return &h, nil
}

// ImportCluster imports the Harvester cluster into Rancher and sets up the necessary credentials for it.
// Note: This function currently only registers the Harvester cluster with Rancher. Additional steps may
// be needed to fully set up the cluster in Rancher, such as generating a kubeconfig and creating cloud credentials.
func (h *HarvesterImportClient) ImportCluster() error {
	harvesterInRancherID, err := harvesteraction.RegisterHarvesterWithRancher(h.client, h.harvesterClient)
	if err != nil {
		return fmt.Errorf("error while registering Harvester cluster with Rancher: %v", err)
	}

	logrus.Info(harvesterInRancherID)

	h.clusterID = harvesterInRancherID

	return nil
}
