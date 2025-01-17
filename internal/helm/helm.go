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

package helm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/dartboard/internal/vendored"
)

func Install(kubecfg, chartLocation, releaseName, namespace string, vals map[string]any, extraArgs ...string) error {
	args := []string{
		"--kubeconfig=" + kubecfg,
		"upgrade",
		"--install",
		"--namespace=" + namespace,
		releaseName,
		chartLocation,
		"--create-namespace",
	}
	if vals != nil {
		valueString := ""
		for k, v := range vals {
			jsonVal, err := json.Marshal(v)
			if err != nil {
				return err
			}
			valueString += k + "=" + string(jsonVal) + ","
		}
		args = append(args, "--set-json="+valueString)
	}
	args = append(args, extraArgs...)

	cmd := vendored.Command("helm", args...)
	var errStream strings.Builder
	cmd.Stdout = os.Stdout
	cmd.Stderr = &errStream
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v", errStream.String())
	}

	return nil
}
