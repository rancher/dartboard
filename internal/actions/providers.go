package actions

import (
	"fmt"

	"github.com/rancher/shepherd/extensions/cloudcredentials/aws"
	"github.com/rancher/shepherd/extensions/cloudcredentials/azure"
	"github.com/rancher/shepherd/extensions/cloudcredentials/harvester"
	"github.com/rancher/tests/actions/machinepools"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/provisioninginput"
)

// CreateProvider returns all machine and cloud credential
// configs in the form of a Provider struct. Accepts a
// string of the name of the provider.
func CreateProvider(name string) provisioning.Provider {
	var provider provisioning.Provider
	switch name {
	case provisioninginput.AWSProviderName.String():
		provider = provisioning.Provider{
			Name:                               provisioninginput.AWSProviderName,
			MachineConfigPoolResourceSteveType: machinepools.AWSPoolType,
			MachinePoolFunc:                    machinepools.NewAWSMachineConfig,
			CloudCredFunc:                      aws.CreateAWSCloudCredentials,
			Roles:                              machinepools.GetAWSMachineRoles(),
		}
		return provider
	case provisioninginput.AzureProviderName.String():
		provider = provisioning.Provider{
			Name:                               provisioninginput.AzureProviderName,
			MachineConfigPoolResourceSteveType: machinepools.AzurePoolType,
			MachinePoolFunc:                    machinepools.NewAzureMachineConfig,
			CloudCredFunc:                      azure.CreateAzureCloudCredentials,
			Roles:                              machinepools.GetAzureMachineRoles(),
		}
		return provider
	case provisioninginput.HarvesterProviderName.String():
		provider = provisioning.Provider{
			Name:                               provisioninginput.HarvesterProviderName,
			MachineConfigPoolResourceSteveType: machinepools.HarvesterPoolType,
			MachinePoolFunc:                    machinepools.NewHarvesterMachineConfig,
			CloudCredFunc:                      harvester.CreateHarvesterCloudCredentials,
			Roles:                              machinepools.GetHarvesterMachineRoles(),
		}
		return provider
	}
	panic(fmt.Sprintf("Provider:%v not found", name))
	// Unreachable, but makes golangci-lint (govet) analyzer happy
	return provider
}
