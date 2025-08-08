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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/rancher/dartboard/internal/vendored"
)

const (
	k6Image          = "grafana/k6:1.0.0"
	K6Namespace      = "tester"
	K6KubeSecretName = "kube"
	mimirURL         = "http://mimir.tester:9009/mimir"
)

type FileEntry struct {
	RelPath string
	Key     string
}

var (
	cacheOnce     sync.Once
	cachedEntries []FileEntry
	cacheErr      error
)

func collectFileEntries(root string, exts map[string]bool) ([]FileEntry, error) {
	var entries []FileEntry
	visited := make(map[string]bool) // Track visited paths to prevent infinite loops

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}

			// Check if we've already visited this resolved path to prevent infinite loops
			if visited[resolvedPath] {
				return nil
			}
			visited[resolvedPath] = true

			// Get info about the resolved path
			resolvedInfo, err := os.Stat(resolvedPath)
			if err != nil {
				return nil
			}

			// If the symlink points to a directory, walk it recursively
			if resolvedInfo.IsDir() {
				return filepath.Walk(resolvedPath, func(subPath string, subInfo os.FileInfo, subErr error) error {
					if subErr != nil || subInfo.IsDir() {
						return subErr
					}

					// Handle nested symlinks by resolving them
					actualPath := subPath
					if subInfo.Mode()&os.ModeSymlink != 0 {
						if resolved, resolveErr := filepath.EvalSymlinks(subPath); resolveErr == nil {
							if !visited[resolved] {
								visited[resolved] = true
								actualPath = resolved
								if fileInfo, statErr := os.Stat(resolved); statErr == nil && !fileInfo.IsDir() {
									subInfo = fileInfo
								} else {
									return subErr
								}
							} else {
								return nil // Skip already visited
							}
						} else {
							return nil // Skip broken symlinks
						}
					}

					ext := filepath.Ext(actualPath)
					if !exts[ext] {
						return nil
					}

					// Calculate relative path from original root
					// First get the relative path of the symlink from root
					symlinkRel, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}

					// Then get the relative path of the file from the resolved symlink target
					fileRel, err := filepath.Rel(resolvedPath, actualPath)
					if err != nil {
						return err
					}

					// Combine them to get the final relative path
					var finalRel string
					if symlinkRel == "." {
						finalRel = fileRel
					} else {
						finalRel = filepath.Join(symlinkRel, fileRel)
					}

					key := strings.ReplaceAll(finalRel, string(os.PathSeparator), "__")
					entries = append(entries, FileEntry{RelPath: finalRel, Key: key})
					return nil
				})
			}
			// Else, it's a file symlink, treat it as a regular file
			ext := filepath.Ext(resolvedPath)
			if !exts[ext] {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			key := strings.ReplaceAll(rel, string(os.PathSeparator), "__")
			entries = append(entries, FileEntry{RelPath: rel, Key: key})

			return nil
		}

		// Skip directories (non-symlink)
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !exts[ext] {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		key := strings.ReplaceAll(rel, string(os.PathSeparator), "__")
		entries = append(entries, FileEntry{RelPath: rel, Key: key})
		return nil
	})

	return entries, err
}

func getCachedEntries(root string, exts map[string]bool) ([]FileEntry, error) {
	cacheOnce.Do(func() {
		cachedEntries, cacheErr = collectFileEntries(root, exts)
	})
	return cachedEntries, cacheErr
}

func Exec(kubepath string, output io.Writer, args ...string) error {
	fullArgs := append([]string{"--kubeconfig=" + kubepath}, args...)
	cmd := vendored.Command("kubectl", fullArgs...)

	var errStream strings.Builder
	cmd.Stderr = &errStream
	cmd.Stdin = os.Stdin

	if output != nil {
		cmd.Stdout = output
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error while running kubectl with params %v: %v", fullArgs, errStream.String())
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
	ingress := map[string]string{}
	err := Get(kubePath, "services", "", "", ".items[0].status.loadBalancer.ingress[0]", &ingress)
	if err != nil {
		return "", err
	}
	if ip, ok := ingress["ip"]; ok {
		return ip + ".sslip.io", nil
	}
	if hostname, ok := ingress["hostname"]; ok {
		return hostname, nil
	}

	return "", nil
}

func Get(kubePath string, kind string, name string, namespace string, jsonpath string, out any) error {
	output := new(bytes.Buffer)
	args := []string{
		"get",
		kind,
	}
	if name != "" {
		args = append(args, name)
	}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	} else {
		args = append(args, "--all-namespaces")
	}
	args = append(args, "-o", fmt.Sprintf("jsonpath={%s}", jsonpath))

	if err := Exec(kubePath, output, args...); err != nil {
		return fmt.Errorf("failed to kubectl get %v: %w", name, err)
	}

	if err := json.Unmarshal(output.Bytes(), out); err != nil {
		return fmt.Errorf("cannot unmarshal kubectl data for %v: %w\n%s", name, err, output.String())
	}

	return nil
}

func GetStatus(kubepath, kind, name, namespace string) (map[string]any, error) {
	out := map[string]any{}
	err := Get(kubepath, kind, name, namespace, ".status", &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func K6run(kubeconfig, testPath string, envVars, tags map[string]string, printLogs bool, localBaseURL string, record bool) error {
	// gather file entries
	root := "./charts/k6-files/test-files"
	exts := map[string]bool{".js": true, ".mjs": true, ".sh": true, ".env": true}
	entries, err := getCachedEntries(root, exts)
	if err != nil {
		log.Fatal(err)
	}
	relTestPath := testPath
	// get rel path to test file
	for _, e := range entries {
		if strings.Contains(e.RelPath, testPath) {
			relTestPath = e.RelPath
			break
		}
	}

	// print what we are about to do
	quotedArgs := []string{"run"}
	for k, v := range envVars {
		if k == "BASE_URL" {
			v = localBaseURL
		}
		quotedArgs = append(quotedArgs, "-e", shellescape.Quote(fmt.Sprintf("%s=%s", k, v)))
	}
	quotedArgs = append(quotedArgs, shellescape.Quote(relTestPath))
	log.Printf("Running equivalent of:\n./bin/k6 %s\n", strings.Join(quotedArgs, " "))

	// if a kubeconfig is specified, upload it as secret to later mount it
	if path, ok := envVars["KUBECONFIG"]; ok {
		err := Exec(kubeconfig, nil, "--namespace="+K6Namespace, "delete", "secret", K6KubeSecretName, "--ignore-not-found")
		if err != nil {
			return err
		}
		err = Exec(kubeconfig, nil, "--namespace="+K6Namespace, "create", "secret", "generic", K6KubeSecretName,
			"--from-file=config="+path)
		if err != nil {
			return err
		}
	}

	// prepare k6 commandline
	args := []string{"run"}
	// ensure we get the complete summary
	args = append(args, "--summary-mode=full")
	for k, v := range envVars {
		// substitute kubeconfig file path with path to secret
		if k == "KUBECONFIG" {
			v = "/kube/config"
		}
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range tags {
		args = append(args, "--tag", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, relTestPath)
	if record {
		args = append(args, "-o", "experimental-prometheus-rw")
	}

	// prepare volumes and volume mounts
	volumes := []any{
		map[string]any{"name": "k6-test-files", "configMap": map[string]string{"name": "k6-test-files"}},
	}

	volumeMounts := []any{}
	for _, e := range entries {
		volumeMounts = append(volumeMounts, map[string]any{
			"name":      "k6-test-files",
			"mountPath": e.RelPath,
			"subPath":   e.Key,
		})
	}
	if _, ok := envVars["KUBECONFIG"]; ok {
		volumes = append(volumes, map[string]any{"name": K6KubeSecretName, "secret": map[string]string{"secretName": "kube"}})
		volumeMounts = append(volumeMounts, map[string]string{"mountPath": "/kube", "name": K6KubeSecretName})
	}

	// prepare pod override map
	override := map[string]any{
		"apiVersion": "v1",
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name":       "k6",
					"image":      k6Image,
					"stdin":      true,
					"tty":        true,
					"args":       args,
					"workingDir": "/",
					"env": []any{
						map[string]any{"name": "K6_PROMETHEUS_RW_SERVER_URL", "value": mimirURL + "/api/v1/push"},
						map[string]any{"name": "K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM", "value": "true"},
						map[string]any{"name": "K6_PROMETHEUS_RW_STALE_MARKERS", "value": "true"},
					},
					"volumeMounts": volumeMounts,
				},
			},
			"volumes": volumes,
		},
	}
	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return err
	}

	var output *os.File
	if printLogs {
		output = os.Stdout
	}

	err = Exec(kubeconfig, output, "run", "k6", "--image="+k6Image, "--namespace=tester", "--rm", "--stdin", "--restart=Never", "--overrides="+string(overrideJSON))
	if err != nil {
		return err
	}

	return nil
}
