package dart

import (
	"errors"
	"fmt"

	"github.com/rancher/tests/actions/machinepools"
	yaml "gopkg.in/yaml.v3"
)

var (
	ErrNoConfig        = errors.New("no NodeConfig set, must have 1")
	ErrMultipleConfigs = errors.New("multiple NodeConfigs set, can only have 1")
	ErrInvalidCPU      = errors.New("cpu must be > 0")
	ErrInvalidMemory   = errors.New("memory must be > 0")
	ErrInvalidString   = errors.New("string must not be empty")
)

type AnyNodeConfig interface {
	ProviderConfig
	HarvesterNodeConfig | AWSNodeConfig | AzureNodeConfig | K3DNodeConfig
}

type ProviderConfig interface {
	ProviderName() string
	Validate() error
}

type NodeConfig struct {
	Harvester     *HarvesterNodeConfig `yaml:"harvester,omitempty" json:"harvester,omitempty"`
	AWS           *AWSNodeConfig       `json:"aws,omitempty" yaml:"aws,omitempty"`
	Azure         *AzureNodeConfig     `json:"azure,omitempty" yaml:"azure,omitempty"`
	K3DNodeConfig *K3DNodeConfig       `json:"k3d_node_config,omitempty" yaml:"k3d_node_config,omitempty"`
}

type AWSNodeConfig struct{}
type AzureNodeConfig struct{}
type K3DNodeConfig struct{}

type HarvesterNodeConfig struct {
	CPU                 int                 `json:"cpu" yaml:"cpu"`
	Memory              int                 `json:"memory" yaml:"memory"`
	Disks               []HarvesterDisk     `json:"disks" yaml:"disks"`
	ImageName           string              `json:"image_name" yaml:"image_name"`
	ImageNamespace      string              `json:"image_namespace" yaml:"image_namespace"`
	Namespace           string              `json:"namespace" yaml:"namespace"`
	Tags                map[string]string   `json:"tags" yaml:"tags"`
	Password            string              `json:"password" yaml:"password"`
	SSHSharedPublicKeys SSHSharedPublicKeys `json:"ssh_shared_public_keys" yaml:"ssh_shared_public_keys"`
	EFI                 bool                `json:"efi" yaml:"efi"`
	SecureBoot          bool                `json:"secure_boot" yaml:"secure_boot"`
}

type SSHSharedPublicKeys struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

type HarvesterDisk struct {
	Name string `json:"name" yaml:"name"`
	Size int    `json:"size" yaml:"size"`
	Type string `json:"type" yaml:"type"`
	Bus  string `json:"bus" yaml:"bus"`
}

// Used for injection into Dart.TofuVariables["node_templates"]
type NodeTemplate[N AnyNodeConfig] struct {
	NodeCount           int    `yaml:"node_count"`
	NamePrefix          string `yaml:"name_prefix"`
	NodeModuleVariables N      `yaml:"node_module_variables"`
}

type ClusterConfig struct {
	MachinePools []MachinePools `yaml:"machine_pools"`
	Provider     string         `yaml:"provider"`
}

type MachinePools struct {
	machinepools.Pools
	MachinePoolConfig MachinePoolConfig `yaml:"machinePoolConfig,omitempty" default:"[]"`
}

type MachinePoolConfig struct {
	machinepools.MachinePoolConfig
	ControlPlane bool       `json:"controlplane,omitempty" yaml:"controlplane,omitempty"`
	Etcd         bool       `json:"etcd,omitempty" yaml:"etcd,omitempty"`
	Worker       bool       `json:"worker,omitempty" yaml:"worker,omitempty"`
	Windows      bool       `json:"windows,omitempty" yaml:"windows,omitempty"`
	Quantity     int32      `json:"quantity" yaml:"quantity"`
	NodeConfig   NodeConfig `yaml:"node_config"`
}

func (h HarvesterNodeConfig) ProviderName() string { return "harvester" }
func (h HarvesterNodeConfig) Validate() error {
	// Currently these are the only *required* fields, by default
	if h.CPU <= 0 {
		return fmt.Errorf("error while validating harvester: %w", ErrInvalidCPU)
	}
	if h.Memory <= 0 {
		return fmt.Errorf("error while validating harvester: %w", ErrInvalidMemory)
	}
	if h.Password == "" {
		return fmt.Errorf("error while validating harvester: password %w", ErrInvalidString)
	}
	return nil
}
func (h AWSNodeConfig) ProviderName() string { return "aws" }
func (h AWSNodeConfig) Validate() error {
	panic("not yet implemented")
}
func (h AzureNodeConfig) ProviderName() string { return "azure" }
func (h AzureNodeConfig) Validate() error {
	panic("not yet implemented")
}
func (h K3DNodeConfig) ProviderName() string { return "k3d" }
func (h K3DNodeConfig) Validate() error {
	panic("not yet implemented")
}

// ToMap converts a given parameter to a valid map
func ToMap(a any) (map[string]interface{}, error) {
	bytes, err := yaml.Marshal(a)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func buildNodeTemplate[N AnyNodeConfig](config N, count int, prefix string) (map[string]any, error) {
	nt := NodeTemplate[N]{
		NodeCount:           count,
		NamePrefix:          prefix,
		NodeModuleVariables: config,
	}

	if err := nt.NodeModuleVariables.Validate(); err != nil {
		return nil, fmt.Errorf("error during %s ProviderConfig validation: %w", config.ProviderName(), err)
	}

	result, err := ToMap(nt)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetActiveConfig returns the single non‑nil ProviderConfig inside nc
// If exactly one is set, it returns that, otherwise an error.
func (nc *NodeConfig) GetActiveConfig() (ProviderConfig, error) {
	var found ProviderConfig
	var count int

	if nc.Harvester != nil {
		found = *nc.Harvester
		count++
	}
	if nc.AWS != nil {
		found = *nc.AWS
		count++
	}
	if nc.Azure != nil {
		found = *nc.Azure
		count++
	}
	if nc.K3DNodeConfig != nil {
		found = *nc.K3DNodeConfig
		count++
	}

	switch count {
	case 0:
		return nil, fmt.Errorf("error: %w", ErrNoConfig)
	case 1:
		return found, found.Validate()
	default:
		return nil, fmt.Errorf("error: %w", ErrMultipleConfigs)
	}
}
