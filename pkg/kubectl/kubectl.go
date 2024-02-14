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
	"context"
	"fmt"
	"io"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	k6Image     = "grafana/k6:0.46.0"
	K6Name      = "k6"
	K6Namespace = "tester"
	mimirURL    = "http://mimir.tester:9009/mimir"
)

type Client struct {
	kubeconfig string
	config     *rest.Config
	clientset  *kubernetes.Clientset
	dynclient  *dynamic.DynamicClient
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

func (cl *Client) K6run(envVars, tags map[string]string, testPath string, printLogs, record bool) error {

	paramEnvVars := []string{}
	for key, val := range envVars {
		paramEnvVars = append(paramEnvVars, "-e", fmt.Sprintf("%s=%s", key, val))
	}

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

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: K6Name,
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
					VolumeMounts: []v1.VolumeMount{
						{Name: "k6-test-files", MountPath: "/k6"},
						{Name: "k6-lib-files", MountPath: "/k6-lib"},
						// TODO: add & manage KUBECONFIG env variable
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "k6-test-files",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: "k6-test-files"},
						},
					},
				},
				{
					Name: "k6-lib-files",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: "k6-lib-files"},
						},
					},
				},
			},
		},
	}

	podCli := cl.clientset.CoreV1().Pods(K6Namespace)
	_, err := podCli.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Check the status of the Pod and wait it to be started (so that we can print logs if needed)
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second, 120*time.Second, false, cl.isPodRunningOrSuccessful(K6Name, K6Namespace))

	if printLogs {
		req := podCli.GetLogs(K6Name, &v1.PodLogOptions{Follow: true})
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
		err = wait.PollUntilContextTimeout(context.TODO(), time.Second, 120*time.Second, false, cl.isPodSuccessful(K6Name, K6Namespace))
	}

	if err != nil {
		return fmt.Errorf("k6 pod not ready: %w", err)
	}

	err = podCli.Delete(context.TODO(), K6Name, metav1.DeleteOptions{})
	if err != nil {
		// Don't fail, let's just log a warning
		log.Printf("WARN: k6 pod deletion failed: %s\n", err.Error())
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
