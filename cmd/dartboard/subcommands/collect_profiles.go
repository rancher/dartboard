/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package subcommands

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rancher/dartboard/internal/profiles"
	cli "github.com/urfave/cli/v3"
)

const (
	ArgDiagnosticsFor      = "for"
	ArgDiagnosticsInterval = "interval"
	ArgDiagnosticsOutput   = "output"
	ArgProfiles            = "profiles"
	ArgCPUDuration         = "cpu-duration"
)

func CollectProfiles(ctx context.Context, cli *cli.Command) error {
	tf, d, err := prepare(ctx, cli)
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

	cfg, err := profiles.ParseConfig(cli.String(ArgDiagnosticsFor), cli.String(ArgDiagnosticsInterval), cli.String(ArgProfiles), cli.String(ArgCPUDuration))
	if err != nil {
		return err
	}

	outDir := resolveProfilesOutputDir(cli, d.TofuWorkspace)

	return profiles.Collect(upstream.Kubeconfig, cli.String(ArgDart), d.TofuWorkspace, cfg, outDir)
}

func resolveProfilesOutputDir(cli *cli.Command, workspace string) string {
	if s := cli.String(ArgDiagnosticsOutput); s != "" {
		return s
	}

	suffix := time.Now().UTC().Format("20060102-150405")
	if workspace == "" {
		return filepath.Join(".", "profiles-"+suffix)
	}

	return filepath.Join(".", "profiles-"+workspace+"-"+suffix)
}
