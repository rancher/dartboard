/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package logs

import (
	"fmt"
	"strings"
	"time"
)

// appSpec describes how to discover and pull logs from one Rancher-family app.
type appSpec struct {
	Label             string
	Namespaces        []string // ordered probe list; first namespace with pods wins
	Container         string
	AuditLogContainer string // empty when the app has no audit-log sidecar
}

// supportedApps mirrors the choices the continuous_profiling.sh script supports.
var supportedApps = map[string]appSpec{
	"rancher": {
		Label:             "app=rancher",
		Namespaces:        []string{"cattle-system"},
		Container:         "rancher",
		AuditLogContainer: "rancher-audit-log",
	},
	"cattle-cluster-agent": {
		Label:      "app=cattle-cluster-agent",
		Namespaces: []string{"cattle-system"},
		Container:  "cluster-register",
	},
	"fleet-controller": {
		Label:      "app=fleet-controller",
		Namespaces: []string{"cattle-fleet-system"},
		Container:  "fleet-controller",
	},
	"fleet-agent": {
		Label:      "app=fleet-agent",
		Namespaces: []string{"cattle-fleet-local-system", "cattle-fleet-system"},
		Container:  "fleet-agent",
	},
}

// Config holds the parsed flags that drive a single collect-logs run.
type Config struct {
	For      time.Duration
	Interval time.Duration
	Apps     []string // validated subset of supportedApps keys; preserves user order
}

// ParseConfig validates the raw CLI flag strings into a Config.
func ParseConfig(forStr, intervalStr, appsCSV string) (Config, error) {
	cfg := Config{}

	d, err := time.ParseDuration(forStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid --for: %w", err)
	}
	if d <= 0 {
		return cfg, fmt.Errorf("--for must be positive")
	}
	cfg.For = d

	iv, err := time.ParseDuration(intervalStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid --interval: %w", err)
	}
	if iv <= 0 {
		return cfg, fmt.Errorf("--interval must be positive")
	}
	cfg.Interval = iv

	if cfg.For < cfg.Interval {
		return cfg, fmt.Errorf("--for (%s) must be >= --interval (%s)", cfg.For, cfg.Interval)
	}

	parts := strings.Split(appsCSV, ",")
	cfg.Apps = make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, a := range parts {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := supportedApps[a]; !ok {
			return cfg, fmt.Errorf("invalid app %q (allowed: rancher, cattle-cluster-agent, fleet-controller, fleet-agent)", a)
		}
		if _, dup := seen[a]; dup {
			continue
		}
		seen[a] = struct{}{}
		cfg.Apps = append(cfg.Apps, a)
	}
	if len(cfg.Apps) == 0 {
		cfg.Apps = []string{"rancher"}
	}

	return cfg, nil
}
