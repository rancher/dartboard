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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
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

type Cluster struct {
	AppAddresses            ClusterAppAddresses `json:"app_addresses"`
	Context                 string              `json:"context"`
	IngressClassName        string              `json:"ingress_class_name"`
	Kubeconfig              string              `json:"kubeconfig"`
	NodeAccessCommands      map[string]string   `json:"node_access_commands"`
	PrivateKubernetesAPIURL string              `json:"private_kubernetes_api_url"`
}

type Tofu struct {
	tf        *tfexec.Terraform
	dir       string
	threads   int
	variables []*tfexec.VarOption
}

// converts a golang map[string]interface{} into map[string]cty.Value
func convertMapToCty(m map[string]interface{}) map[string]cty.Value {
	attributes := map[string]cty.Value{}
	for k, v := range m {
		attributes[k] = convertValueToHCL(v)
	}
	return attributes
}

// converts golang types into HCL format utilizing the hclwrite and cty packages
// reference: https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclwrite#example-package-GenerateFromScratch
func convertValueToHCL(value interface{}) cty.Value {
	switch v := value.(type) {
	case string:
		return cty.StringVal(value.(string))
	case bool:
		return cty.BoolVal(v)
	case int, int32, int64:
		// explicitly convert to int64 if needed
		vInt64, ok := v.(int64)
		if !ok {
			vInt32, ok := v.(int32)
			if !ok {
				vInt64 = int64(v.(int))
			} else {
				vInt64 = int64(vInt32)
			}
		}
		return cty.NumberIntVal(vInt64)
	case float32, float64:
		// explicitly convert to float64 if needed
		vFloat64, ok := v.(float64)
		if !ok {
			vFloat64 = float64(v.(float32))
		}
		return cty.NumberFloatVal(vFloat64)
		// maps and objects in HCL are equivalent to cty.ObjectVals
	case map[string]interface{}:
		return cty.ObjectVal(convertMapToCty(v))
		// if we have a list/slice, loop through it and make recursive calls
	case []interface{}:
		var elements []cty.Value
		for _, sliceVal := range v {
			elements = append(elements, convertValueToHCL(sliceVal))
		}
		return cty.ListVal(elements)
	default:
		return cty.UnknownAsNull(cty.StringVal(""))
	}
}

func appendVariables(variables []*tfexec.VarOption, variableMap map[string]any) []*tfexec.VarOption {
	for k, v := range variableMap {
		f := hclwrite.NewEmptyFile()
		body := f.Body()
		ctyVal := convertValueToHCL(v)
		body.SetAttributeValue(k, ctyVal)
		assignment := strings.ReplaceAll(string(f.Bytes()), " ", "")
		variables = append(variables, tfexec.Var(assignment))
	}

	return variables
}

func New(ctx context.Context, variableMap map[string]any, dir string, parallelism int, verbose bool) (*Tofu, error) {
	tfBinary, err := exec.LookPath("tofu")
	if err != nil {
		return nil, fmt.Errorf("error: tofu not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(dir, tfBinary)
	if err != nil {
		return nil, fmt.Errorf("tfexec.NewTerraform error: %w", err)
	}

	if verbose {
		tf.SetStdout(os.Stdout)
	}

	if err = tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return nil, fmt.Errorf("error: tofu Init: %w", err)
	}

	variables := appendVariables(nil, variableMap)

	return &Tofu{
		tf:        tf,
		dir:       dir,
		threads:   parallelism,
		variables: variables,
	}, nil
}

func (t *Tofu) Apply(ctx context.Context) error {
	options := []tfexec.ApplyOption{tfexec.Parallelism(t.threads)}
	for _, variable := range t.variables {
		options = append(options, variable)
	}

	if err := t.tf.Apply(ctx, options...); err != nil {
		return fmt.Errorf("error: tofu apply failed: %w", err)
	}
	return nil
}

func (t *Tofu) Destroy(ctx context.Context) error {
	options := []tfexec.DestroyOption{tfexec.Parallelism(t.threads)}
	for _, variable := range t.variables {
		options = append(options, variable)
	}

	if err := t.tf.Destroy(ctx, options...); err != nil {
		return fmt.Errorf("error: tofu destroy failed: %w", err)
	}

	return nil
}

func (t *Tofu) OutputClustersJson(ctx context.Context) (string, error) {
	tfOutput, err := t.tf.Output(ctx)
	if err != nil {
		return "", fmt.Errorf("error: tofu OutputClustersJson: %w", err)
	}

	if clusters, ok := tfOutput["clusters"]; ok {
		return string(clusters.Value), nil
	}

	return "", fmt.Errorf("error: tofu OutputClustersJson: no cluster data")
}

func (t *Tofu) OutputClusters(ctx context.Context) (map[string]Cluster, error) {
	tfOutput, err := t.tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("error: tofu OutputClusters: %w", err)
	}

	clusters := map[string]Cluster{}
	if err := json.Unmarshal(tfOutput["clusters"].Value, &clusters); err != nil {
		return nil, fmt.Errorf("error: tofu OutputClusters: %w", err)
	}

	return clusters, nil

}

// Version queries Tofu version and the provider list.
// It returns the version as a string, the provider list as a map of strings
// and any error encountered.
func (t *Tofu) Version(ctx context.Context) (version string, providers map[string]string, err error) {
	tfVer, tfProv, err := t.tf.Version(ctx, false)
	if err != nil {
		err = fmt.Errorf("error: tofu GetVersion: %w", err)
		return
	}

	version = tfVer.String()
	providers = make(map[string]string)
	for prov, ver := range tfProv {
		providers[prov] = ver.String()
	}

	return
}

// IsK3d determines if the current backend is k3d
func (t *Tofu) IsK3d() bool {
	_, f := filepath.Split(t.dir)
	return f == "k3d"
}
