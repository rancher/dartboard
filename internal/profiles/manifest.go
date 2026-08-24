/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package profiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Manifest captures the run metadata written to manifest.json.
type Manifest struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	For         time.Duration `json:"for_seconds"`
	Interval    time.Duration `json:"interval_seconds"`
	CPUDuration time.Duration `json:"cpu_duration_seconds"`
	Profiles    []string      `json:"profiles"`
	Pods        []string      `json:"pods"`
	Snapshots   int           `json:"snapshots"`
	DartFile    string        `json:"dart_file,omitempty"`
	Workspace   string        `json:"tofu_workspace,omitempty"`
	Kubeconfig  string        `json:"kubeconfig,omitempty"`
}

// WriteManifest writes the run-level manifest.
func WriteManifest(outDir string, m Manifest) error {
	path := filepath.Join(outDir, "manifest.json")
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
