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
	K6ResultsDir     = "/tmp/k6-results"
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

// collectFileEntries walks the directory tree at `root`, follows symlinks,
// and returns FileEntry objects for files whose extensions are present in `exts`.
//
// Behavior:
//   - Uses a single DFS traversal algorithm that follows symlinks safely
//   - Tracks visited *resolved absolute paths* to avoid infinite cycles
//   - Produces virtual RelPath values that reflect the path under `root`,
//     even when the actual file content lives outside the tree via a symlink
//   - Logs (with log.Printf) broken symlinks instead of silently swallowing them
//
// Parameters:
//   - root: path to the directory to walk (may be relative). Must exist.
//   - exts: map of allowed extensions (include the leading dot: ".js")
//
// Returns:
//   - []FileEntry: slice of discovered file entries
//   - error: non-nil on fatal errors (e.g., missing root); non-fatal issues are logged
//
// NOTE:
//   - Does NOT handle collisions. If a file at "/dir1/file.js" exists and a file
//     named "dir1__file.js" exists, then the file that gets parsed last will overwrite
//     any entries matching the flattened key (dir1__file.js in this case)
func collectFileEntries(root string, exts map[string]bool) ([]FileEntry, error) {
	// Get valid path to root and ensure it exists
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", root)
	}

	// Resolve root's real path
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		log.Printf("warning: cannot eval symlinks for root %q: %v (continuing with root as-is)", root, err)
		resolvedRoot = root
	}

	type stackEntry struct {
		virtualRel string // path relative to root in virtual namespace ("" for root)
		realPath   string // the actual filesystem path to read (resolved)
	}

	visited := map[string]bool{} // tracks visited resolved real paths to avoid cycles
	var out []FileEntry
	stack := []stackEntry{{virtualRel: "", realPath: resolvedRoot}}

	for len(stack) > 0 {
		// pop from stack
		n := len(stack) - 1
		cur := stack[n]
		stack = stack[:n]

		absReal, err := filepath.Abs(cur.realPath)
		if err != nil {
			log.Printf("warning: cannot abs resolved path %q: %v", cur.realPath, err)
			continue
		}
		// get absolute path via EvalSymlinks if possible
		absRealResolved := absReal
		if rp, err := filepath.EvalSymlinks(absReal); err == nil {
			absRealResolved = rp
		}
		if visited[absRealResolved] {
			continue
		}
		visited[absRealResolved] = true

		entries, err := os.ReadDir(cur.realPath)
		if err != nil {
			log.Printf("warning: failed to read dir %q: %v", cur.realPath, err)
			continue
		}

		for _, de := range entries {
			name := de.Name()
			virtualRel := filepath.Join(cur.virtualRel, name)
			entryRealPath := filepath.Join(cur.realPath, name)

			isSymlink := (de.Type() & os.ModeSymlink) != 0

			if isSymlink {
				target, err := filepath.EvalSymlinks(entryRealPath)
				if err != nil {
					log.Printf("warning: broken symlink or cannot resolve %q: %v", entryRealPath, err)
					continue
				}
				stat, err := os.Stat(target)
				if err != nil {
					log.Printf("warning: cannot stat symlink target %q: %v", target, err)
					continue
				}
				if stat.IsDir() {
					// push directory to stack to preserve virtualRel
					stack = append(stack, stackEntry{virtualRel: virtualRel, realPath: target})
					continue
				}
			} else {
				if de.IsDir() {
					stack = append(stack, stackEntry{virtualRel: virtualRel, realPath: entryRealPath})
					continue
				}
			}

			// Now handle a file (resolved if it was a symlink)
			ext := filepath.Ext(name)
			if !exts[ext] {
				continue
			}

			rel := filepath.Clean(virtualRel)
			// On Unix, filepath.Clean might return ".",
			// and if rel is empty, file is at current-level
			if rel == "." || rel == "" {
				rel = name
			}

			// Make the flattened key for the ConfigMap
			flat := strings.ReplaceAll(rel, string(os.PathSeparator), "__")

			out = append(out, FileEntry{
				RelPath: rel,
				Key:     flat,
			})
		}
	}

	return out, nil
}

// getCachedEntries returns a cached list of FileEntry objects for all files
// under the given root directory that match the provided file extensions in `exts`.
//
// The directory walk is performed only once per process invocation, even if
// getCachedEntries is called multiple times, using sync.Once to guard the scan.
// Ex: Running `dartboard load` results in a single caching process, consecutive
//
//	`dartboard load` commands will also results in a single caching process.
//
// The cache is in-memory only! Once the process exits, the cache is discarded.
//
// Parameters:
//   - root: absolute or relative path to the root directory to scan
//   - exts: a map of allowed file extensions (include the leading dot: ".js")
//
// Returns:
//   - []FileEntry: a slice of matching file entries
//   - error: any I/O or filesystem errors encountered during the scan
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
	err := WaitForReadyCondition(kubePath, "deployment", "rancher", "cattle-system", "available", 20)
	if err != nil {
		return err
	}
	err = WaitForReadyCondition(kubePath, "deployment", "rancher-webhook", "cattle-system", "available", 3)
	if err != nil {
		return err
	}
	err = WaitForReadyCondition(kubePath, "deployment", "fleet-controller", "cattle-fleet-system", "available", 5)
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
		map[string]any{"name": "k6-results", "hostPath": map[string]string{"path": K6ResultsDir, "type": "DirectoryOrCreate"}},
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
		volumeMounts = append(volumeMounts, map[string]string{"mountPath": K6ResultsDir, "name": "k6-results"})
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
