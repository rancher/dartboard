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

	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/rancher/dartboard/internal/metrics"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

const (
	ArgStart  = "start"
	ArgEnd    = "end"
	ArgLast   = "last"
	ArgStep   = "step"
	ArgOutput = "output"

	prometheusNamespace = "cattle-monitoring-system"
	prometheusService   = "svc/rancher-monitoring-prometheus"
	prometheusPort      = 9090
)

func CollectMetrics(c *cli.Context) error {
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
		return fmt.Errorf("upstream cluster kubeconfig is empty; cannot reach Prometheus")
	}

	start, end, step, err := resolveWindow(c)
	if err != nil {
		return err
	}
	outDir := resolveOutputDir(c, d.TofuWorkspace)

	logrus.Infof("port-forwarding to %s/%s in upstream cluster", prometheusNamespace, prometheusService)
	localPort, stop, err := kubectl.PortForward(upstream.Kubeconfig, prometheusNamespace, prometheusService, prometheusPort)
	if err != nil {
		return fmt.Errorf("port-forward to upstream Prometheus: %w", err)
	}
	defer stop()

	promURL := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	logrus.Infof("collecting metrics from %s for window %s..%s (step %s)", promURL, start.Format(time.RFC3339), end.Format(time.RFC3339), step)

	return metrics.Collect(promURL, c.String(ArgDart), d.TofuWorkspace, start, end, step, outDir)
}

func resolveWindow(c *cli.Context) (time.Time, time.Time, time.Duration, error) {
	step, err := time.ParseDuration(c.String(ArgStep))
	if err != nil {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s: %w", ArgStep, err)
	}

	var end time.Time
	if s := c.String(ArgEnd); s != "" {
		end, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s (want RFC3339): %w", ArgEnd, err)
		}
	} else {
		end = time.Now().UTC()
	}

	var start time.Time
	if s := c.String(ArgStart); s != "" {
		start, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s (want RFC3339): %w", ArgStart, err)
		}
	} else {
		last, err := time.ParseDuration(c.String(ArgLast))
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --%s: %w", ArgLast, err)
		}
		start = end.Add(-last)
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("--%s must be after --%s", ArgEnd, ArgStart)
	}
	return start, end, step, nil
}

func resolveOutputDir(c *cli.Context, workspace string) string {
	if s := c.String(ArgOutput); s != "" {
		return s
	}
	suffix := time.Now().UTC().Format("20060102-150405")
	if workspace == "" {
		return filepath.Join(".", "metrics-"+suffix)
	}
	return filepath.Join(".", "metrics-"+workspace+"-"+suffix)
}
