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

package k3d

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/rancher/dartboard/internal/tofu"
)

func ImageImport(cluster tofu.Cluster, image string) error {
	args := []string{"image", "import", "--cluster", strings.Replace(cluster.Context, "k3d-", "", -1), image}
	log.Printf("Exec: docker %s\n", strings.Join(args, " "))

	cmd := exec.Command("k3d", args...)
	var errStream strings.Builder
	cmd.Stdout = os.Stdout
	cmd.Stderr = &errStream
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v", errStream.String())
	}

	return nil
}
