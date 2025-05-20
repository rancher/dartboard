package actions

import (
	"fmt"

	"github.com/rancher/dartboard/internal/tofu"
	"github.com/rancher/shepherd/clients/rancher"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubeconfig struct {
	APIVersion     string
	Kind           string
	Clusters       []Cluster
	Users          []User
	Contexts       []Context
	CurrentContext string
}

type Cluster struct {
	Name    string
	Cluster ClusterInfo
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
	kubeconfigBytes, err := GetKubeconfigBytes(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(kubeconfigBytes, &kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling kubeconfig YAML for %s: %w", kubeconfigBytes, err)
	}

	return &kubeconfig, nil
}

func GetKubeconfigBytes(kubeconfigPath string) ([]byte, error) {
	kubeconfigBytes, err := tofu.ReadBytesFromPath(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	return kubeconfigBytes, err
}

// RESTConfigFromKubeConfig is a convenience method to give back a restconfig from your kubeconfig bytes.
// For programmatic access, this is what you want 80% of the time
func GetRESTConfigFromBytes(kubeconfig []byte) (*rest.Config, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error while getting Rest Config from kubeconfig bytes: %w", err)
	}
	return restConfig, nil
}

func GetRESTConfigFromPath(kubeconfigPath string) (*rest.Config, error) {
	clusterKubeconfig, err := GetKubeconfigBytes(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	restConfig, err := GetRESTConfigFromBytes(clusterKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error getting REST Config for Kubeconfig at %s:\n%w", kubeconfigPath, err)
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
