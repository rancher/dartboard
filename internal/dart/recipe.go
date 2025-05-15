package dart

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/rancher/tests/actions/provisioninginput"
)

// Dart is a "recipe" that encodes all parameters for a test run
type Dart struct {
	TofuMainDirectory      string            `yaml:"tofu_main_directory"`
	TofuWorkspace          string            `yaml:"tofu_workspace"`
	TofuParallelism        int               `yaml:"tofu_parallelism"`
	TofuVariables          map[string]any    `yaml:"tofu_variables"`
	ChartVariables         ChartVariables    `yaml:"chart_variables"`
	ClusterBatchSize       int               `yaml:"cluster_batch_size"`
	ClusterTemplates       []ClusterTemplate `yaml:"cluster_template"`
	TestVariables          TestVariables     `yaml:"test_variables"`
	TofuWorkspaceStatePath string            `yaml:"-"` // omit from YAML output
}

type ClusterTemplate struct {
	generatedName string
	NamePrefix    string                    `yaml:"name_prefix"`
	Config        *provisioninginput.Config `yaml:"provisioning_config"`
	DistroVersion string                    `yaml:"distro_version"`
	ClusterCount  int                       `yaml:"cluster_count"`
}

type ChartVariables struct {
	RancherReplicas             int              `yaml:"rancher_replicas"`
	DownstreamRancherMonitoring bool             `yaml:"downstream_rancher_monitoring"`
	AdminPassword               string           `yaml:"admin_password"`
	RancherVersion              string           `yaml:"rancher_version"`
	ForcePrimeRegistry          bool             `yaml:"force_prime_registry"`
	RancherChartRepoOverride    string           `yaml:"rancher_chart_repo_override"`
	RancherImageOverride        string           `yaml:"rancher_image_override"`
	RancherImageTagOverride     string           `yaml:"rancher_image_tag_override"`
	RancherMonitoringVersion    string           `yaml:"rancher_monitoring_version"`
	CertManagerVersion          string           `yaml:"cert_manager_version"`
	TesterGrafanaVersion        string           `yaml:"tester_grafana_version"`
	RancherValues               string           `yaml:"rancher_values"`
	ExtraEnvironmentVariables   []map[string]any `yaml:"extra_environment_variables"`
}

type TestVariables struct {
	TestConfigMaps int `yaml:"test_config_maps"`
	TestSecrets    int `yaml:"test_secrets"`
	TestRoles      int `yaml:"test_roles"`
	TestUsers      int `yaml:"test_users"`
	TestProjects   int `yaml:"test_projects"`
}

var defaultDart = Dart{
	TofuParallelism: 10,
	TofuVariables:   map[string]any{},
	ChartVariables: ChartVariables{
		RancherReplicas:             1,
		DownstreamRancherMonitoring: false,
		AdminPassword:               "adminadminadmin",
		RancherVersion:              "2.9.1",
		RancherMonitoringVersion:    "104.1.0+up57.0.3",
		CertManagerVersion:          "1.8.0",
		TesterGrafanaVersion:        "6.56.5",
	},
	TestVariables: TestVariables{
		TestConfigMaps: 2000,
		TestSecrets:    2000,
		TestRoles:      20,
		TestUsers:      10,
		TestProjects:   20,
	},
}

func Parse(path string) (*Dart, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dart file: %w", err)
	}
	result := defaultDart
	err = yaml.Unmarshal(bytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dart file: %w", err)
	}
	tofuVars, err := yaml.Marshal(result.TofuVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recipe's tofu variables: %w", err)
	}
	log.Printf("\nTofu variables: \n%v\n", string(tofuVars))

	result.ChartVariables.RancherVersion = normalizeVersion(result.ChartVariables.RancherVersion)
	result.ChartVariables.RancherMonitoringVersion = normalizeVersion(result.ChartVariables.RancherMonitoringVersion)
	result.ChartVariables.CertManagerVersion = normalizeVersion(result.ChartVariables.CertManagerVersion)
	result.ChartVariables.TesterGrafanaVersion = normalizeVersion(result.ChartVariables.TesterGrafanaVersion)
	result.ChartVariables.ForcePrimeRegistry = result.ChartVariables.ForcePrimeRegistry || needsPrime(result.ChartVariables.RancherVersion)

	return &result, nil
}

// normalizeVersion tolerates versions with an initial spurious v
func normalizeVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// needsPrime returns true if the Rancher version is known to require use of the Prime registry
func needsPrime(version string) bool {
	versionSplits := regexp.MustCompile("[.-]").Split(version, -1)
	major, _ := strconv.Atoi(versionSplits[0])
	minor, _ := strconv.Atoi(versionSplits[1])
	patch, _ := strconv.Atoi(versionSplits[2])
	return (major == 2 && minor == 7 && patch >= 11) ||
		(major == 2 && minor == 8 && patch >= 6)
}

func (ct *ClusterTemplate) SetGeneratedName(suffix string) {
	ct.generatedName = fmt.Sprintf("%s-%s", ct.NamePrefix, suffix)
}

func (ct *ClusterTemplate) GeneratedName() string {
	return ct.generatedName
}
