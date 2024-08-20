package scale

import (
	provV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/shepherd/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ConfigurationFileKey = "scaleInput"
	MemProfileFileName   = "mem-profile"
	CPUProfileFileName   = "cpu-profile"
	ProfileFileExtension = ".log"
	MemProfileCommand    = "curl -s http://localhost:6060/debug/pprof/heap -o " + MemProfileFileName
	CPUProfileCommand    = "curl -s http://localhost:6060/debug/pprof/profile -o " + CPUProfileFileName
	CustomTimeFormatCode = "2006-01-02_T15-04-05"
	PreScaleInOutputDir  = "pre-scale"
	PreUpgradeOutputDir  = "pre-upgrade"
	HalftimeOutputDir    = "halftime"
	PostScaleInOutputDir = "post-scale"
	PostUpgradeOutputDir = "post-upgrade"
)

func memProfileCommand() []string {
	return []string{
		"/bin/sh",
		"-c",
		MemProfileCommand,
	}
}

func cpuProfileCommand() []string {
	return []string{
		"/bin/sh",
		"-c",
		CPUProfileCommand,
	}
}

func configMapGVR() schema.GroupVersionResource {
	return corev1.SchemeGroupVersion.WithResource("configmaps")
}

func v1ClusterGVR() schema.GroupVersionResource {
	return provV1.SchemeGroupVersion.WithResource("clusters")
}

type Config struct {
	ClusterTarget int    `json:"clusterTarget" yaml:"clusterTarget"`
	BatchSize     int    `json:"batchSize" yaml:"batchSize"`
	BatchTimeout  int    `json:"batchTimeout" yaml:"batchTimeout"`
	OutputDir     string `json:"outputDir" yaml:"outputDir"`
	ConfigMapsDir string `json:"configMapsDir" yaml:"configMapsDir"`
}

func LoadScaleConfig() *Config {
	scaleConfig := new(Config)
	config.LoadConfig(ConfigurationFileKey, scaleConfig)
	return scaleConfig
}
