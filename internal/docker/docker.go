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

package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Image is the mapping of docker images --format=json
type Image struct {
	Repository string
	Tag        string
}

// Images returns known docker images matching the image reference
func Images(image string) ([]string, error) {
	args := []string{"images", "--filter=reference=" + image, "--format=json"}
	log.Printf("Exec: docker %s\n", strings.Join(args, " "))

	cmd := exec.Command("docker", args...)

	var (
		outStream strings.Builder
		errStream strings.Builder
	)

	cmd.Stdout = &outStream

	cmd.Stderr = &errStream
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v", errStream.String())
	}

	lines := strings.Split(strings.TrimSpace(outStream.String()), "\n")

	var images []string

	for _, line := range lines {
		if line != "" {
			var img Image

			err := json.Unmarshal([]byte(line), &img)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling JSON output from docker images: %w", err)
			}

			images = append(images, img.Repository+":"+img.Tag)
		}
	}

	return images, nil
}
