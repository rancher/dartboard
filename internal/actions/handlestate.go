package actions

import (
	"fmt"
	"os"
	"sync"

	"github.com/rancher/dartboard/internal/dart"
	"github.com/rancher/dartboard/internal/tofu"
	yaml "gopkg.in/yaml.v2"
)

// ClusterStatus holds the state of each cluster.
type ClusterStatus struct {
	Name        string `yaml:"name"`
	Created     bool   `yaml:"created"`
	Imported    bool   `yaml:"imported"`
	Provisioned bool   `yaml:"provisioned"`
	// Only one of the following should be included
	tofu.Cluster         `yaml:"cluster,omitempty"`          //For Imported Clusters
	dart.ClusterTemplate `yaml:"cluster_template,omitempty"` //For Provisioned Clusters
}

const ClustersStateFile = "clusters_state.yaml"

// Mutex to sync map[string]*ClusterStatus mutations and file writes
var stateMutex sync.Mutex

// stateUpdate is a simple "signaling" struct for the writer goroutine to persist state
type stateUpdate struct{}

// SaveClusterState persists the map[string]*ClusterStatus to a YAML file.
func SaveClusterState(filePath string, statuses map[string]*ClusterStatus) error {
	data, err := yaml.Marshal(statuses)
	if err != nil {
		return fmt.Errorf("failed to marshal Cluster state: %w", err)
	}

	// stateMutex.Lock()
	// defer stateMutex.Unlock()

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Cluster state file: %w", err)
	}
	return nil
}

// LoadClusterState reads the YAML state file and unmarshals into map[string]*ClusterStatus.
// If the file does not exist, it returns an empty [] without error.
func LoadClusterState(filePath string) (map[string]*ClusterStatus, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Did not find existing Cluster state file at %s.\n Creating new Cluster state file and returning new empty map[string]*ClusterStatus\n", filePath)
		if err := SaveClusterState(filePath, map[string]*ClusterStatus{}); err != nil {
			return nil, fmt.Errorf("failed init Cluster state file: %w", err)
		}
		return map[string]*ClusterStatus{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to os.Stat Cluster state file at %s: %w", filePath, err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to os.ReadFile Cluster state file at %s: %w", filePath, err)
	}
	var statuses map[string]*ClusterStatus
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

func FindClusterStatus(statuses map[string]*ClusterStatus, condition func(*ClusterStatus) bool) *ClusterStatus {
	for key := range statuses {
		if condition(statuses[key]) {
			return statuses[key]
		}
	}
	return nil
}

func FindClusterStatusByName(statuses map[string]*ClusterStatus, name string) *ClusterStatus {
	return FindClusterStatus(statuses, func(cs *ClusterStatus) bool {
		return cs.Name == name
	})
}

func FindOrCreateStatusByName(statuses map[string]*ClusterStatus, name string) *ClusterStatus {
	clusterStatus := FindClusterStatusByName(statuses, name)
	if clusterStatus == nil {
		fmt.Printf("Did not find existing ClusterStatus object for Cluster with name %s.\n", name)

		newClusterStatus := ClusterStatus{
			Name:        name,
			Created:     false,
			Imported:    false,
			Provisioned: false,
		}
		statuses[name] = &newClusterStatus
		fmt.Println("Created new ClusterStatus object. ClusterStatus. Cluster must be set appropriately.")

		return statuses[name]
	}

	fmt.Printf("Found ClusterStatus object named %s. ClusterStatus. Cluster must be set appropriately.\n", name)
	return clusterStatus
}
