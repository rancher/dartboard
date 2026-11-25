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

	"github.com/rancher/dartboard/internal/logs"
	cli "github.com/urfave/cli/v3"
)

const (
	ArgApps = "apps"
	// ArgK3KFocusNamespace names a virtual cluster whose pod is always
	// collected, on top of the capped per-role selection.
	ArgK3KFocusNamespace = "k3k-focus-namespace"
)

func CollectLogs(ctx context.Context, cli *cli.Command) error {
	// Validate flags before touching OpenTofu or the filesystem, so a typo
	// fails fast instead of after a workspace refresh.
	cfg, err := logs.ParseConfig(
		cli.String(ArgDiagnosticsFor),
		cli.String(ArgDiagnosticsInterval),
		cli.String(ArgApps),
		cli.String(ArgK3KFocusNamespace),
	)
	if err != nil {
		return err
	}

	t, err := resolveTarget(ctx, cli)
	if err != nil {
		return err
	}

	outDir := outputDir(cli.String(ArgDiagnosticsOutput), "logs", t)

	return logs.Collect(t.Kubeconfig, t.DartFile, t.Workspace, cfg, outDir)
}
