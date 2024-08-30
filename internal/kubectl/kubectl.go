/*
Copyright © 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubectl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	k6Image          = "grafana/k6:0.46.0"
	K6Name           = "k6"
	K6Namespace      = "tester"
	K6KubeSecretName = "kube"
	mimirURL         = "http://mimir.tester:9009/mimir"
)

type Client struct {
	kubeconfig string
	config     *rest.Config
	clientset  *kubernetes.Clientset
	dynclient  *dynamic.DynamicClient
}

func Exec(kubepath string, output io.Writer, args ...string) error {
	fullArgs := append([]string{"--kubeconfig=" + kubepath}, args...)
	log.Printf("Exec: kubectl %s\n", strings.Join(fullArgs, " "))
	cmd := exec.Command("kubectl", fullArgs...)

	var errStream strings.Builder
	cmd.Stdout = output
	cmd.Stderr = &errStream

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v", errStream.String())
	}
	return nil
}

func Apply(kubePath, filePath string) error {
	return Exec(kubePath, log.Writer(), "apply", "-f", filePath)
}

func WaitRancher(kubePath string) error {
	err := WaitForReadyCondition(kubePath, "deployment", "rancher", "cattle-system", "available", 60)
	if err != nil {
		return err
	}
	err = WaitForReadyCondition(kubePath, "deployment", "rancher-webhook", "cattle-system", "available", 60)
	if err != nil {
		return err
	}
	err = WaitForReadyCondition(kubePath, "deployment", "fleet-controller", "cattle-fleet-system", "available", 60)
	return err
}

func WaitForReadyCondition(kubePath, resource, name, namespace string, condition string, minutes int) error {
	var err error
	args := []string{"wait", resource, name}

	if len(namespace) > 0 {
		args = append(args, "--namespace", namespace)
	}
	args = append(args, "--for", fmt.Sprintf("condition=%s=true", condition), fmt.Sprintf("--timeout=%dm", minutes))

	maxRetries := minutes * 30
	for i := 1; i < maxRetries; i++ {
		err = Exec(kubePath, log.Writer(), args...)
		if err == nil {
			return nil
		}
		// Check if by chance the resource is not yet available
		if strings.Contains(err.Error(), fmt.Sprintf("%q not found", name)) {
			log.Printf("resource %s/%s not available yet, retry %d/%d\n", namespace, name, i, maxRetries)
			time.Sleep(2 * time.Second)
		} else {
			return err
		}
	}

	return err
}

func GetRancherFQDNFromLoadBalancer(kubePath string) (string, error) {
	output := new(bytes.Buffer)
	if err := Exec(kubePath, output, "get", "services", "--all-namespaces",
		"-o", "jsonpath={.items[0].status.loadBalancer.ingress[0]}"); err != nil {
		return "", fmt.Errorf("failed to fetch loadBalancer data: %w", err)
	}

	ingress := map[string]string{}
	if err := json.Unmarshal(output.Bytes(), &ingress); err != nil {
		return "", fmt.Errorf("cannot unmarshal ingress data: %w\n%s", err, output.String())
	}

	if ip, ok := ingress["ip"]; ok {
		return ip + ".sslip.io", nil
	}
	if hostname, ok := ingress["hostname"]; ok {
		return hostname, nil
	}

	return "", nil
}

func (cl *Client) Init(kubePath string) error {
	var err error
	cl.kubeconfig = kubePath
	if cl.config, err = clientcmd.BuildConfigFromFlags("", kubePath); err != nil {
		return err
	}
	if cl.clientset, err = kubernetes.NewForConfig(cl.config); err != nil {
		return err
	}
	if cl.dynclient, err = dynamic.NewForConfig(cl.config); err != nil {
		return err
	}
	return nil
}

func (cl *Client) GetStatus(group, ver, res, name, namespace string) (map[string]interface{}, error) {
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  ver,
		Resource: res,
	}

	get, err := cl.dynclient.Resource(resource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// expect status as map[string]interface{} or error
	status, ok := get.UnstructuredContent()["status"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error accessing '%s/%s' %s: 'Status' format not supported", namespace, name, res)
	}

	return status, nil
}

func fillK6TestFilesVols(vol *[]v1.Volume, volMount *[]v1.VolumeMount) {
	*vol = append(*vol,
		v1.Volume{
			Name: "k6-test-files",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: "k6-test-files"},
				},
			},
		},
		v1.Volume{
			Name: "k6-lib-files",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: "k6-lib-files"},
				},
			},
		},
	)

	*volMount = append(*volMount,
		v1.VolumeMount{Name: "k6-test-files", MountPath: "/k6"},
		v1.VolumeMount{Name: "k6-lib-files", MountPath: "/k6/lib"},
	)
}

func (cl *Client) K6run(name, testPath string, envVars, tags map[string]string, printLogs, record bool) error {

	podVolumes := []v1.Volume{}
	podVolumeMounts := []v1.VolumeMount{}
	paramEnvVars := []string{}
	for key, val := range envVars {
		if key == "KUBECONFIG" {
			Exec(cl.kubeconfig, nil, "--namespace="+K6Namespace, "delete", "secret", K6KubeSecretName, "--ignore-not-found")
			Exec(cl.kubeconfig, nil, "--namespace="+K6Namespace, "create", "secret", "generic", K6KubeSecretName,
				"--from-file=config="+val)
			val = "/kube/config"
			podVolumeMounts = []v1.VolumeMount{{Name: K6KubeSecretName, MountPath: "/kube"}}
			podVolumes = []v1.Volume{{Name: K6KubeSecretName,
				VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: K6KubeSecretName}}}}
		}
		paramEnvVars = append(paramEnvVars, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	fillK6TestFilesVols(&podVolumes, &podVolumeMounts)

	paramTags := []string{}
	for key, val := range tags {
		paramTags = append(paramTags, "--tag", fmt.Sprintf("%s=%s", key, val))
	}

	args := append([]string{"run"}, paramEnvVars...)
	args = append(args, paramTags...)
	args = append(args, testPath)
	if record {
		args = append(args, "-o", "experimental-prometheus-rw")
	}

	podName := fmt.Sprintf("%s-%s", K6Name, name)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:       K6Name,
					Image:      k6Image,
					Stdin:      true,
					TTY:        true,
					Args:       args,
					WorkingDir: "/",
					Env: []v1.EnvVar{
						{Name: "K6_PROMETHEUS_RW_SERVER_URL", Value: mimirURL + "/api/v1/push"},
						{Name: "K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM", Value: "true"},
						{Name: "K6_PROMETHEUS_RW_STALE_MARKERS", Value: "true"},
					},
					VolumeMounts: podVolumeMounts,
				},
			},
			Volumes: podVolumes,
		},
	}

	podCli := cl.clientset.CoreV1().Pods(K6Namespace)
	err := podCli.Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		// Don't fail, let's just log a warning
		log.Printf("WARN: k6 pod deletion failed: %s\n", err.Error())
	}

	_, err = podCli.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {

		return err
	}

	// Check the status of the Pod and wait it to be started (so that we can print logs if needed)
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second, 120*time.Second, false, cl.isPodRunningOrSuccessful(podName, K6Namespace))

	if printLogs {
		req := podCli.GetLogs(podName, &v1.PodLogOptions{Follow: true})
		stream, err := req.Stream(context.TODO())
		if err != nil {
			return fmt.Errorf("error retrieving logs: %w", err)
		}
		defer stream.Close()

		for {
			buf := make([]byte, 2048)
			numBytes, err := stream.Read(buf)
			if numBytes > 0 {
				message := string(buf[:numBytes])
				fmt.Print(message)
				continue
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("error retrieving logs: %w", err)
			}
		}
	}
	// Check if the Pod was in error state before printing the logs.
	// If not, there is a chance it was running but still not ended, so we have to double check it is completed *and* successful.
	if err == nil {
		err = wait.PollUntilContextTimeout(context.TODO(), time.Second, 120*time.Second, false, cl.isPodSuccessful(podName, K6Namespace))
	}

	if err != nil {
		return fmt.Errorf("k6 pod not ready: %w", err)
	}

	return nil
}

func (cl *Client) isPodRunningOrSuccessful(name, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		podCli := cl.clientset.CoreV1().Pods(namespace)
		pod, err := podCli.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case v1.PodRunning, v1.PodSucceeded:
			return true, nil
		case v1.PodPending:
			return false, nil
		default:
			// Pod failed
			return false, fmt.Errorf("pod failed")
		}
	}
}

func (cl *Client) isPodSuccessful(name, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		podCli := cl.clientset.CoreV1().Pods(namespace)
		pod, err := podCli.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case v1.PodSucceeded:
			return true, nil
		case v1.PodPending, v1.PodRunning:
			return false, nil
		default:
			// Pod failed
			return false, fmt.Errorf("pod failed")
		}
	}
}
