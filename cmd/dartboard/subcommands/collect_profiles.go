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

// CollectProfiles snapshots pprof profiles from the Rancher pods of the
// selected cluster.
//
// Note that this only works for Rancher: k3k does not expose a pprof endpoint
// (its controller builds a controller-runtime manager without setting
// PprofBindAddress), so pointing this at a k3k host cluster will find no pods
// and produce an empty manifest.
func CollectProfiles(ctx context.Context, cli *cli.Command) error {
	// Validate flags before touching OpenTofu or the filesystem, so a typo
	// fails fast instead of after a workspace refresh.
	cfg, err := profiles.ParseConfig(cli.String(ArgDiagnosticsFor), cli.String(ArgDiagnosticsInterval), cli.String(ArgProfiles), cli.String(ArgCPUDuration))
	if err != nil {
		return err
	}

	t, err := resolveTarget(ctx, cli)
	if err != nil {
		return err
	}

	outDir := outputDir(cli.String(ArgDiagnosticsOutput), "profiles", t)

	return profiles.Collect(t.Kubeconfig, t.DartFile, t.Workspace, cfg, outDir)
}
