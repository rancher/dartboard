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
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rancher/dartboard/internal/vendored"
	cli "github.com/urfave/cli/v3"
)

const (
	// ArgCluster selects which cluster from the OpenTofu `clusters` output a
	// diagnostics subcommand acts on.
	ArgCluster = "cluster"
	// ArgKubeconfig points a diagnostics subcommand at an arbitrary cluster,
	// bypassing OpenTofu entirely.
	ArgKubeconfig = "kubeconfig"

	// DefaultClusterName preserves the historical behaviour of the collect-*
	// and summarize subcommands, which always acted on the Rancher cluster.
	DefaultClusterName = "upstream"
)

// target identifies the cluster a diagnostics subcommand collects from.
//
// It is produced either from the OpenTofu `clusters` output (via --cluster) or
// straight from a kubeconfig path (via --kubeconfig). In the latter case there
// is no dart and no workspace, so DartFile and Workspace are empty and the
// collectors simply omit those fields from their manifests.
type target struct {
	// Name labels the target in logs, output directory names and manifests.
	Name string
	// Kubeconfig is an existing path to a kubeconfig file for the target.
	Kubeconfig string
	// Context is the kubeconfig context, when known. Generated kubeconfigs set
	// their own current-context, so this is informational only.
	Context string
	// DartFile is the dart the target was resolved from, empty for --kubeconfig.
	DartFile string
	// Workspace is the OpenTofu workspace, empty for --kubeconfig.
	Workspace string
}

// TargetFlags returns the cluster-selection flags shared by every subcommand
// that collects diagnostics from a single cluster.
func TargetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  ArgCluster,
			Value: DefaultClusterName,
			Usage: "cluster to collect from, as named in the OpenTofu clusters output " +
				"(e.g. \"upstream\", \"tester\", \"downstream-0-0\")",
		},
		&cli.StringFlag{
			Name: ArgKubeconfig,
			Usage: "path to a kubeconfig for a cluster not managed by this dart; " +
				"bypasses OpenTofu and is mutually exclusive with --" + ArgCluster,
		},
	}
}

// resolveTarget determines which cluster to collect from.
//
// When --kubeconfig is given the dart and OpenTofu state are not consulted at
// all, which is what makes it possible to point the collectors at a
// pre-existing cluster (for instance a k3k host installed outside dartboard).
// Otherwise the dart is parsed, OpenTofu outputs are read and --cluster selects
// an entry from the resulting `clusters` map.
func resolveTarget(ctx context.Context, c *cli.Command) (target, error) {
	if kubeconfig := c.String(ArgKubeconfig); kubeconfig != "" {
		if c.IsSet(ArgCluster) {
			return target{}, fmt.Errorf("--%s and --%s are mutually exclusive", ArgCluster, ArgKubeconfig)
		}

		return targetFromKubeconfig(kubeconfig)
	}

	return targetFromTofu(ctx, c)
}

// targetFromKubeconfig builds a target from a user-supplied kubeconfig path.
// The vendored binaries are still extracted because the collectors shell out to
// the vendored kubectl.
func targetFromKubeconfig(kubeconfig string) (target, error) {
	path, err := expandPath(kubeconfig)
	if err != nil {
		return target{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return target{}, fmt.Errorf("--%s %s: %w", ArgKubeconfig, kubeconfig, err)
	}

	if info.IsDir() {
		return target{}, fmt.Errorf("--%s %s is a directory, want a kubeconfig file", ArgKubeconfig, kubeconfig)
	}

	// Absolute, because some collectors chdir into their output directory
	// before shelling out to kubectl. The OpenTofu path is already absolute.
	path, err = filepath.Abs(path)
	if err != nil {
		return target{}, fmt.Errorf("--%s %s: %w", ArgKubeconfig, kubeconfig, err)
	}

	if err := vendored.ExtractBinaries(); err != nil {
		return target{}, err
	}

	fmt.Printf("Using kubeconfig: %s\n", path)

	return target{
		Name:       nameFromKubeconfigPath(path),
		Kubeconfig: path,
	}, nil
}

// targetFromTofu resolves --cluster against the OpenTofu `clusters` output.
func targetFromTofu(ctx context.Context, c *cli.Command) (target, error) {
	tf, d, err := prepare(ctx, c)
	if err != nil {
		return target{}, err
	}

	clusters, _, err := tf.ParseOutputs()
	if err != nil {
		return target{}, fmt.Errorf("parse tofu outputs: %w", err)
	}

	name := c.String(ArgCluster)
	if name == "" {
		name = DefaultClusterName
	}

	cluster, ok := clusters[name]
	if !ok {
		return target{}, fmt.Errorf("cluster %q not found in tofu outputs (available: %s); "+
			"use --%s to point at a cluster this dart does not manage",
			name, strings.Join(clusterNames(clusters), ", "), ArgKubeconfig)
	}

	if cluster.Kubeconfig == "" {
		return target{}, fmt.Errorf("cluster %q has an empty kubeconfig in the tofu outputs", name)
	}

	return target{
		Name:       name,
		Kubeconfig: cluster.Kubeconfig,
		Context:    cluster.Context,
		DartFile:   c.String(ArgDart),
		Workspace:  d.TofuWorkspace,
	}, nil
}

// clusterNames returns the sorted keys of a clusters map, for error messages.
func clusterNames[T any](clusters map[string]T) []string {
	names := make([]string, 0, len(clusters))
	for name := range clusters {
		names = append(names, name)
	}

	sort.Strings(names)

	if len(names) == 0 {
		return []string{"none"}
	}

	return names
}

// nameFromKubeconfigPath derives a short, filesystem-safe target name from a
// kubeconfig path, so output directories stay distinguishable across runs.
func nameFromKubeconfigPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	if base == "" || base == "." || base == string(filepath.Separator) {
		return "kubeconfig"
	}

	return base
}

// expandPath resolves a leading "~" to the current user's home directory.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("cannot expand %q: %w", path, err)
	}

	return filepath.Join(usr.HomeDir, strings.TrimPrefix(path, "~")), nil
}

// outputDir returns the explicit --output value when set, otherwise a
// timestamped directory name scoped to the workspace and target.
func outputDir(explicit, kind string, t target) string {
	if explicit != "" {
		return explicit
	}

	parts := []string{kind}
	if t.Workspace != "" {
		parts = append(parts, t.Workspace)
	}

	if t.Name != "" {
		parts = append(parts, t.Name)
	}

	parts = append(parts, time.Now().UTC().Format("20060102-150405"))

	return filepath.Join(".", strings.Join(parts, "-"))
}
