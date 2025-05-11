package actions

import (
	"fmt"
	"os"

	"github.com/rancher/shepherd/clients/rancher"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubeconfig struct {
	APIVersion     string
	Kind           string
	Clusters       []ClusterInfo
	Users          []User
	Contexts       []Context
	CurrentContext string
}

type ClusterInfo struct {
	Server                   string
	CertificateAuthorityData string
}

type User struct {
	Name string
	User UserData
}

type UserData struct {
	Token string
}

type Context struct {
	Name    string
	Context ContextData
}

type ContextData struct {
	User    string
	Cluster string
}

func ParseKubeconfig(kubeconfigPath string) (*Kubeconfig, error) {
	kubeconfig := Kubeconfig{}
	if _, err := os.Stat(kubeconfigPath); err == nil {
		kubeconfigBytes, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("error reading kubeconfig file at %v: %v", kubeconfigPath, err)
		}

		err = yaml.Unmarshal(kubeconfigBytes, &kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling kubeconfig YAML for %v: %v", kubeconfigBytes, err)
		}
	} else {
		return nil, fmt.Errorf("error could not find kubeconfig at %v: %v", kubeconfigPath, err)
	}
	return &kubeconfig, nil
}

func GetKubeconfigBytes(kubeconfigPath string) ([]byte, error) {
	var kubeconfigBytes []byte
	if _, err := os.Stat(kubeconfigPath); err == nil {
		kubeconfigBytes, err = os.ReadFile(kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("error reading kubeconfig file at %v: %v", kubeconfigPath, err)
		}
	} else {
		return nil, fmt.Errorf("error could not find kubeconfig at %v: %v", kubeconfigPath, err)
	}

	return kubeconfigBytes, nil
}

// RESTConfigFromKubeConfig is a convenience method to give back a restconfig from your kubeconfig bytes.
// For programmatic access, this is what you want 80% of the time
func GetRESTConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error while getting Rest Config from kubeconfig bytes: %v", err)
	}
	return restConfig, nil
}

func GetRESTConfigForClusterID(rancherClient *rancher.Client, id string) (*rest.Config, error) {
	cluster, err := rancherClient.Management.Cluster.ByID(id)
	if err != nil {
		return nil, fmt.Errorf("error while getting Cluster by ID %s: %v", id, err)
	}
	output, err := rancherClient.Management.Cluster.ActionGenerateKubeconfig(cluster)
	if err != nil {
		return nil, fmt.Errorf("error while generating Kubeconfig for Cluster with ID %s: %v", id, err)
	}

	configBytes := []byte(output.Config)

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error while getting Rest Config for Cluster with ID %s: %v", id, err)
	}

	return restConfig, nil
}

func GetLocalClusterRESTConfig(rancherClient *rancher.Client) (*rest.Config, error) {
	return GetRESTConfigForClusterID(rancherClient, "local")
}
