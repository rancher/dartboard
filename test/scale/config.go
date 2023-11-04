package scale

import (
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/kubeconfig"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ConfigurationFileKey = "scaleInput"
	MemProfileFileName   = "mem-profile.log"
	CPUProfileFileName   = "cpu-profile.log"
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

type Config struct {
	ClusterTarget int    `json:"clusterTarget" yaml:"clusterTarget"`
	BatchSize     int    `json:"batchSize" yaml:"batchSize"`
	OutputDir     string `json:"outputDir" yaml:"outputDir"`
}

func ScaleConfig() *Config {
	scaleConfig := new(Config)
	config.LoadConfig(ConfigurationFileKey, scaleConfig)
	return scaleConfig
}

func getAllRancherLogs(r *rancher.Client, c string, p string, t metav1.Time) (string, error) {
	podLogOptions := &corev1.PodLogOptions{
		Container:  "rancher",
		Timestamps: true,
		SinceTime:  &t,
	}
	return kubeconfig.GetPodLogsWithOpts(r, c, p, "cattle-system", podLogOptions)
}

func generateRancherMemProfile(r restclient.Config, p string) (*kubeconfig.LogStreamer, error) {
	cmd := memProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, p, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing memory profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func getRancherMemProfile(r restclient.Config, k clientcmd.ClientConfig, p string, dest string) (bool, error) {
	_, err := generateRancherMemProfile(r, p)
	if err != nil {
		return false, err
	}

	err = kubeconfig.CopyFileFromPod(&r, k, p, "cattle-system", MemProfileFileName, dest)
	if err != nil {
		log.Warnf("error copying file (%s) from pod (%s) to dest (%s): %v", MemProfileFileName, p, dest, err)
		return false, err
	}
	return true, nil
}

func generateRancherCPUProfile(r restclient.Config, p string) (*kubeconfig.LogStreamer, error) {
	cmd := cpuProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, p, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing cpu profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func getRancherCPUProfile(r restclient.Config, k clientcmd.ClientConfig, p string, dest string) (bool, error) {
	_, err := generateRancherCPUProfile(r, p)
	if err != nil {
		return false, err
	}
	err = kubeconfig.CopyFileFromPod(&r, k, p, "cattle-system", CPUProfileFileName, dest)
	if err != nil {
		log.Warnf("error copying file (%s) from pod (%s) to dest (%s): %v", CPUProfileFileName, p, dest, err)
		return false, err
	}
	return true, nil
}
