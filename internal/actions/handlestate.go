package actions

import (
	"fmt"
	"os"

	"github.com/rancher/dartboard/internal/tofu"
	yaml "gopkg.in/yaml.v2"
)

// ClusterStatus holds the state of each cluster.
type ClusterStatus struct {
	Name         string `yaml:"name"`
	Created      bool   `yaml:"created"`
	Imported     bool   `yaml:"imported"`
	Provisioned  bool   `yaml:"provisioned"`
	tofu.Cluster `yaml:"clusterdata"`
}

const ClustersStateFile = "clusters_state.yaml"

// SaveClusterState persists the []ClusterStatus to a YAML file.
func SaveClusterState(filePath string, statuses []ClusterStatus) error {
	data, err := yaml.Marshal(statuses)
	if err != nil {
		return fmt.Errorf("failed to marshal Cluster state: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Cluster state file: %w", err)
	}
	return nil
}

// LoadClusterState reads the YAML state file and unmarshals into []ClusterStatus.
// If the file does not exist, it returns an empty [] without error.
func LoadClusterState(filePath string) ([]ClusterStatus, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Did not find existing Cluster state file at %s. Returning new empty []ClusterStatus\n", filePath)
		return []ClusterStatus{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to os.Stat Cluster state file at %s: %w", filePath, err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to os.ReadFile Cluster state file at %s: %w", filePath, err)
	}
	var statuses []ClusterStatus
	if err := yaml.Unmarshal(data, &statuses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return statuses, nil
}

func DestroyClusterState(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Did not find existing Cluster state file at %s.\n", filePath)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to os.Stat Cluster state file at %s during destroy: %w", filePath, err)
	}

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to os.Remove Cluster state file at %s: %w", filePath, err)
	}

	return nil
}

func FindClusterStatus(statuses []ClusterStatus, condition func(ClusterStatus) bool) *ClusterStatus {
	for _, s := range statuses {
		if condition(s) {
			return &s
		}
	}
	return nil
}

func FindClusterStatusByName(statuses []ClusterStatus, name string) *ClusterStatus {
	return FindClusterStatus(statuses, func(cs ClusterStatus) bool {
		return cs.Name == name
	})
}
