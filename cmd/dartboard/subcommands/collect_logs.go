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

	"github.com/rancher/dartboard/internal/logs"
	cli "github.com/urfave/cli/v3"
)

const (
	ArgApps = "apps"
)

func CollectLogs(ctx context.Context, cli *cli.Command) error {
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

	cfg, err := logs.ParseConfig(cli.String(ArgDiagnosticsFor), cli.String(ArgDiagnosticsInterval), cli.String(ArgApps))
	if err != nil {
		return err
	}

	outDir := resolveLogsOutputDir(cli, d.TofuWorkspace)

	return logs.Collect(upstream.Kubeconfig, cli.String(ArgDart), d.TofuWorkspace, cfg, outDir)
}

func resolveLogsOutputDir(cli *cli.Command, workspace string) string {
	if s := cli.String(ArgDiagnosticsOutput); s != "" {
		return s
	}

	suffix := time.Now().UTC().Format("20060102-150405")
	if workspace == "" {
		return filepath.Join(".", "logs-"+suffix)
	}

	return filepath.Join(".", "logs-"+workspace+"-"+suffix)
}
