package scale

import (
	provV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/kubeconfig"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ConfigurationFileKey = "scaleInput"
	MemProfileFileName   = "mem-profile"
	CPUProfileFileName   = "cpu-profile"
	ProfileFileExtension = ".log"
	MemProfileCommand    = "curl -s http://localhost:6060/debug/pprof/heap -o " + MemProfileFileName
	CPUProfileCommand    = "curl -s http://localhost:6060/debug/pprof/profile -o " + CPUProfileFileName
	CustomTimeFormatCode = "2006-01-02_T15-04-05"
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

func loadScaleConfig() *Config {
	scaleConfig := new(Config)
	config.LoadConfig(ConfigurationFileKey, scaleConfig)
	return scaleConfig
}

func getAllRancherLogs(r *rancher.Client, clusterID string, podName string, since metav1.Time) (string, error) {
	podLogOptions := &corev1.PodLogOptions{
		Container:  "rancher",
		Timestamps: true,
		SinceTime:  &since,
	}
	return kubeconfig.GetPodLogsWithOpts(r, clusterID, podName, "cattle-system", podLogOptions)
}

func generateRancherMemProfile(r restclient.Config, podName string) (*kubeconfig.LogStreamer, error) {
	cmd := memProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, podName, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing memory profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func getRancherMemProfile(r restclient.Config, k clientcmd.ClientConfig, podName string, dest string) error {
	_, err := generateRancherMemProfile(r, podName)
	if err != nil {
		return err
	}

	err = kubeconfig.CopyFileFromPod(&r, k, podName, "cattle-system", MemProfileFileName+ProfileFileExtension, dest)
	if err != nil {
		log.Warnf("error copying file (%s) from pod (%s) to dest (%s): %v", MemProfileFileName+ProfileFileExtension, podName, dest, err)
		return err
	}
	return nil
}

func generateRancherCPUProfile(r restclient.Config, podName string) (*kubeconfig.LogStreamer, error) {
	cmd := cpuProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, podName, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing cpu profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func getRancherCPUProfile(r restclient.Config, k clientcmd.ClientConfig, podName string, dest string) error {
	_, err := generateRancherCPUProfile(r, podName)
	if err != nil {
		return err
	}
	err = kubeconfig.CopyFileFromPod(&r, k, podName, "cattle-system", CPUProfileFileName+ProfileFileExtension, dest)
	if err != nil {
		log.Warnf("error copying file (%s) from pod (%s) to dest (%s): %v", CPUProfileFileName+ProfileFileExtension, podName, dest, err)
		return err
	}
	return nil
}
