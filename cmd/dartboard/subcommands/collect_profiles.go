/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package subcommands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rancher/dartboard/internal/profiles"
	cli "github.com/urfave/cli/v2"
)

const (
	ArgDiagnosticsFor      = "for"
	ArgDiagnosticsInterval = "interval"
	ArgDiagnosticsOutput   = "output"
	ArgProfiles            = "profiles"
	ArgCPUDuration         = "cpu-duration"
)

func CollectProfiles(c *cli.Context) error {
	tf, d, err := prepare(c)
	if err != nil {
		return err
	}

	clusters, _, err := tf.ParseOutputs()
	if err != nil {
		return fmt.Errorf("parse tofu outputs: %w", err)
	}

	upstream, ok := clusters["upstream"]
	if !ok || upstream.Kubeconfig == "" {
		return fmt.Errorf("upstream cluster kubeconfig is empty; cannot reach Rancher pods")
	}

	cfg, err := profiles.ParseConfig(c.String(ArgDiagnosticsFor), c.String(ArgDiagnosticsInterval), c.String(ArgProfiles), c.String(ArgCPUDuration))
	if err != nil {
		return err
	}

	outDir := resolveProfilesOutputDir(c, d.TofuWorkspace)

	return profiles.Collect(upstream.Kubeconfig, c.String(ArgDart), d.TofuWorkspace, cfg, outDir)
}

func resolveProfilesOutputDir(c *cli.Context, workspace string) string {
	if s := c.String(ArgDiagnosticsOutput); s != "" {
		return s
	}
	suffix := time.Now().UTC().Format("20060102-150405")
	if workspace == "" {
		return filepath.Join(".", "profiles-"+suffix)
	}
	return filepath.Join(".", "profiles-"+workspace+"-"+suffix)
}
