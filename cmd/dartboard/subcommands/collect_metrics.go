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
	"strings"
	"time"

	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/metrics"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"
)

const (
	ArgMetricsStart  = "start"
	ArgMetricsEnd    = "end"
	ArgMetricsLast   = "last"
	ArgMetricsStep   = "step"
	ArgMetricsOutput = "output"

	ArgPreset              = "preset"
	ArgNamespaces          = "namespaces"
	ArgK3KRunPrefix        = "k3k-run-prefix"
	ArgK3KStartIndex       = "k3k-start-index"
	ArgK3KCount            = "k3k-count"
	ArgOperational         = "operational"
	ArgPrometheusNamespace = "prometheus-namespace"
	ArgPrometheusService   = "prometheus-service"

	// DefaultPrometheusNamespace is where both rancher-monitoring and
	// kube-prometheus-stack are installed by dartboard deploy. Without
	// --prometheus-service the Prometheus service is detected in it.
	DefaultPrometheusNamespace = "cattle-monitoring-system"

	prometheusPort = 9090
)

func CollectMetrics(ctx context.Context, cli *cli.Command) error {
	// Validate flags before touching OpenTofu or the filesystem, so a typo
	// fails fast instead of after a workspace refresh.
	catalogOpts, err := resolveCatalogOptions(cli)
	if err != nil {
		return err
	}

	start, end, step, err := resolveWindow(cli)
	if err != nil {
		return err
	}

	t, err := resolveTarget(ctx, cli)
	if err != nil {
		return err
	}

	outDir := outputDir(cli.String(ArgMetricsOutput), "metrics", t)

	promNamespace := cli.String(ArgPrometheusNamespace)

	promService, err := prometheusServiceTarget(ctx, t.Kubeconfig, promNamespace, cli.String(ArgPrometheusService))
	if err != nil {
		return err
	}

	logrus.Infof("port-forwarding to %s/%s in cluster %q", promNamespace, promService, t.Name)

	localPort, stop, err := kubectl.PortForward(ctx, t.Kubeconfig, promNamespace, promService, prometheusPort)
	if err != nil {
		return fmt.Errorf("port-forward to Prometheus in cluster %q: %w", t.Name, err)
	}
	defer stop()

	promURL := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	logrus.Infof("collecting %s metrics from %s for window %s..%s (step %s)",
		catalogOpts.Preset, promURL, start.Format(time.RFC3339), end.Format(time.RFC3339), step)

	return metrics.Collect(ctx, promURL, t.DartFile, t.Workspace, start, end, step, outDir, catalogOpts)
}

// resolveCatalogOptions turns the --preset and --namespaces flags into the
// options that select which PromQL catalog is collected.
func resolveCatalogOptions(cli *cli.Command) (metrics.CatalogOptions, error) {
	preset, err := metrics.ParsePreset(cli.String(ArgPreset))
	if err != nil {
		return metrics.CatalogOptions{}, err
	}

	opts := metrics.CatalogOptions{
		Preset:         preset,
		Namespaces:     cli.String(ArgNamespaces),
		FocusNamespace: cli.String(ArgK3KFocusNamespace),
		K3KRunPrefix:   cli.String(ArgK3KRunPrefix),
		K3KStartIndex:  cli.Int(ArgK3KStartIndex),
		K3KCount:       cli.Int(ArgK3KCount),
		Operational:    cli.Bool(ArgOperational),
	}
	if err := opts.Validate(); err != nil {
		return metrics.CatalogOptions{}, err
	}

	return opts, nil
}

// prometheusServiceTarget returns the port-forward target for Prometheus: the
// explicit --prometheus-service value, or else whichever of rancher-monitoring
// or kube-prometheus-stack is installed in the namespace.
func prometheusServiceTarget(ctx context.Context, kubeconfig, namespace, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	name, err := kubectl.DetectPrometheusService(ctx, kubeconfig, namespace)
	if err != nil {
		return "", err
	}

	return "svc/" + name, nil
}

// prometheusServiceName strips the resource-type prefix from a port-forward
// target such as "svc/rancher-monitoring-prometheus", yielding the bare service
// name that in-cluster clients dial.
func prometheusServiceName(target string) string {
	if _, name, found := strings.Cut(target, "/"); found {
		return name
	}

	return target
}

func resolveWindow(cli *cli.Command) (time.Time, time.Time, time.Duration, error) {
	step, err := time.ParseDuration(cli.String(ArgMetricsStep))
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s: %w", ArgMetricsStep, err)
	}

	var end time.Time
	if s := cli.String(ArgMetricsEnd); s != "" {
		end, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s (want RFC3339): %w", ArgMetricsEnd, err)
		}
	} else {
		end = time.Now().UTC()
	}

	var start time.Time
	if s := cli.String(ArgMetricsStart); s != "" {
		start, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s (want RFC3339): %w", ArgMetricsStart, err)
		}
	} else {
		last, err := time.ParseDuration(cli.String(ArgMetricsLast))
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s: %w", ArgMetricsLast, err)
		}

		start = end.Add(-last)
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("--%s must be after --%s", ArgMetricsEnd, ArgMetricsStart)
	}

	return start, end, step, nil
}
