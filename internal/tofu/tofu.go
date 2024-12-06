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

package tofu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rancher/dartboard/internal/tofu/format"
	"github.com/rancher/dartboard/internal/vendored"
)

type ClusterAddress struct {
	HTTPPort  uint   `json:"http_port"`
	HTTPSPort uint   `json:"https_port"`
	Name      string `json:"name"`
}

type ClusterAppAddresses struct {
	Private ClusterAddress `json:"private"`
	Public  ClusterAddress `json:"public"`
	Tunnel  ClusterAddress `json:"tunnel"`
}

type Addresses struct {
	Public  string `json:"public"`
	Private string `json:"private"`
	Tunnel  string `json:"tunnel"`
}

type Cluster struct {
	AppAddresses             ClusterAppAddresses `json:"app_addresses"`
	Context                  string              `json:"context"`
	IngressClassName         string              `json:"ingress_class_name"`
	Kubeconfig               string              `json:"kubeconfig"`
	NodeAccessCommands       map[string]string   `json:"node_access_commands"`
	KubernetesAddresses      Addresses           `json:"kubernetes_addresses"`
	ReserveNodeForMonitoring bool                `json:"reserve_node_for_monitoring"`
}

type Clusters struct {
	Value map[string]Cluster `json:"value"`
}

type Output struct {
	Clusters Clusters `json:"clusters"`
}

type Tofu struct {
	dir       string
	workspace string
	threads   int
	verbose   bool
	variables []string
}

func New(ctx context.Context, variableMap map[string]interface{}, dir string, ws string, parallelism int, verbose bool) (*Tofu, error) {
	var variables []string
	for k, v := range variableMap {
		variable := fmt.Sprintf("%s=%s", k, format.ConvertValueToHCL(v, false))
		variables = append(variables, variable)
	}

	t := &Tofu{
		dir:       dir,
		workspace: ws,
		threads:   parallelism,
		verbose:   verbose,
		variables: variables,
	}

	if err := t.exec(nil, "init", "-upgrade"); err != nil {
		return nil, err
	}

	return t, nil
}

// exec runs Tofu with the correct chdir parameter
func (t *Tofu) exec(output io.Writer, args ...string) error {
	fullArgs := append([]string{"-chdir=" + t.dir}, args...)
	cmd := vendored.Command("tofu", fullArgs...)

	var errStream strings.Builder
	cmd.Stderr = &errStream
	cmd.Stdin = os.Stdin

	if t.verbose {
		cmd.Stdout = os.Stdout
	}

	if output != nil {
		cmd.Stdout = output
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error while running tofu: %v", errStream.String())
	}
	return nil
}

func (t *Tofu) handleWorkspace(ctx context.Context) error {
	if !(len(t.workspace) > 0) {
		t.workspace = "default"
	}

	wsExists, err := t.workspaceExists(ctx)
	if err != nil {
		return err
	}

	if wsExists {
		log.Printf("Found existing tofu workspace: %s", t.workspace)
		return t.selectWorkspace(ctx)
	}

	log.Printf("Creating new tofu workspace: %s", t.workspace)
	if err = t.newWorkspace(ctx); err != nil {
		return err
	}

	return t.selectWorkspace(ctx)
}

func (t *Tofu) workspaceExists(ctx context.Context) (bool, error) {
	args := []string{"workspace", "list"}

	var out bytes.Buffer
	var err error

	if err = t.exec(&out, args...); err != nil {
		return false, fmt.Errorf("failed to list workspaces: %v", err)
	}

	wsExists := bytes.Contains(out.Bytes(), []byte(t.workspace))

	return wsExists, err
}

func (t *Tofu) selectWorkspace(ctx context.Context) error {
	args := []string{"workspace", "select", t.workspace}

	return t.exec(nil, args...)
}

func (t *Tofu) newWorkspace(ctx context.Context) error {
	args := []string{"workspace", "new", t.workspace}

	return t.exec(nil, args...)
}

func (t *Tofu) Apply(ctx context.Context) error {
	t.handleWorkspace(ctx)

	args := t.commonArgs("apply")

	return t.exec(nil, args...)
}

func (t *Tofu) Destroy(ctx context.Context) error {
	t.handleWorkspace(ctx)

	args := t.commonArgs("destroy")

	return t.exec(nil, args...)
}

// commonArgs formats arguments common to multiple commands
func (t *Tofu) commonArgs(command string) []string {
	args := []string{command, "-parallelism", strconv.Itoa(t.threads), "-auto-approve"}

	for _, variable := range t.variables {
		args = append(args, "-var", variable)
	}
	return args
}

func (t *Tofu) OutputClusters(ctx context.Context) (map[string]Cluster, error) {
	t.handleWorkspace(ctx)

	buffer := new(bytes.Buffer)
	if err := t.exec(buffer, "output", "-json"); err != nil {
		return nil, err
	}

	output := &Output{}
	if err := json.Unmarshal(buffer.Bytes(), output); err != nil {
		return nil, fmt.Errorf("error: tofu OutputClusters: %w", err)
	}

	return output.Clusters.Value, nil
}

// PrintVersion prints the Tofu version information
func (t *Tofu) PrintVersion(ctx context.Context) error {
	return t.exec(log.Writer(), "version")
}

// IsK3d determines if the current backend is k3d
func (t *Tofu) IsK3d() bool {
	_, f := filepath.Split(t.dir)
	return f == "k3d"
}
