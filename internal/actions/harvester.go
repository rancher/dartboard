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
	clusterID       string
	harvesterClient *harvester.Client
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

func (h *HarvesterImportClient) ImportCluster() error {
	harvesterInRancherID, err := harvesteraction.RegisterHarvesterWithRancher(h.client, h.harvesterClient)
	if err != nil {
		return fmt.Errorf("error while registering Harvester cluster with Rancher: %v", err)
	}
	logrus.Info(harvesterInRancherID)

	h.clusterID = harvesterInRancherID

	// cluster, err := h.client.Management.Cluster.ByID(harvesterInRancherID)
	// if err != nil {
	// 	return fmt.Errorf("error while getting Harvester's Rancher Cluster ID: %v", err)
	// }

	// kubeConfig, err := h.client.Management.Cluster.ActionGenerateKubeconfig(cluster)
	// if err != nil {
	// 	return fmt.Errorf("error while generating Harvester's Rancher Cluster kubeconfig: %v", err)
	// }

	// var harvesterCredentialConfig cloudcredentials.HarvesterCredentialConfig

	// harvesterCredentialConfig.ClusterID = harvesterInRancherID
	// harvesterCredentialConfig.ClusterType = "imported"
	// harvesterCredentialConfig.KubeconfigContent = kubeConfig.Config

	return nil
}
