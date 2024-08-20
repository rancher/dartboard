package rancherprofiling

import (
	"os/exec"

	"github.com/rancher/shepherd/extensions/kubeconfig"
	log "github.com/sirupsen/logrus"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ProfileFileExtension  = ".log"
	MemProfileFileName    = "mem-profile"
	CPUProfileFileName    = "cpu-profile"
	BasePprofAddress      = "localhost:6060"
	MemProfileCurlCommand = "curl -s http://localhost:6060/debug/pprof/heap -o " + MemProfileFileName + ProfileFileExtension
	CPUProfileCurlCommand = "curl -s http://localhost:6060/debug/pprof/profile -o " + CPUProfileFileName + ProfileFileExtension
	CustomTimeFormatCode  = "2006-01-02_T15-04-05"
)

func ProfileEndpoints() map[string]string {
	return map[string]string{"graph": "/ui/", "top": "/ui/top", "flame": "/ui/flamegraph", "peek": "/ui/peek", "source": "/ui/source"}
}

func MemProfileCommand() []string {
	return []string{
		"/bin/sh",
		"-c",
		MemProfileCurlCommand,
	}
}

func CPUProfileCommand() []string {
	return []string{
		"/bin/sh",
		"-c",
		CPUProfileCurlCommand,
	}
}

func GenerateRancherMemProfile(r restclient.Config, podName string) (*kubeconfig.LogStreamer, error) {
	cmd := MemProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, podName, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing memory profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func GetRancherMemProfile(r restclient.Config, k clientcmd.ClientConfig, podName string, dest string) error {
	_, err := GenerateRancherMemProfile(r, podName)
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

func GenerateRancherCPUProfile(r restclient.Config, podName string) (*kubeconfig.LogStreamer, error) {
	cmd := CPUProfileCommand()
	streamer, err := kubeconfig.KubectlExec(&r, podName, "cattle-system", cmd)
	if err != nil {
		log.Warnf("error executing cpu profile command (%s): %v", cmd, err)
	}
	return streamer, err
}

func GetRancherCPUProfile(r restclient.Config, k clientcmd.ClientConfig, podName string, dest string) error {
	_, err := GenerateRancherCPUProfile(r, podName)
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

func StartServeProfile(filePath string) *exec.Cmd {
	// Use `go tool pprof` command-line tool to serve  profiles stored locally
	cmd := exec.Command("go", "tool", "pprof", "-http="+BasePprofAddress, "-no_browser", filePath)
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error serving profile: %v", err)
		return cmd
	}

	// Provide instructions to access the profile in the browser
	log.Infof("Profile served successfully at: http://%s", BasePprofAddress)
	return cmd
}
