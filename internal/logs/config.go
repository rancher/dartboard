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
	"sort"
	"strings"
	"time"
)

// maxK3KPodsPerRole caps how many k3k virtual-cluster pods are tracked per role.
//
// Deliberately a constant rather than a flag: at the scale k3k is exercised at,
// thousands of vclusters, the useful signal is a small stable sample of pod logs
// plus the fleet-wide metrics, not more log files. Raising this multiplies the
// per-snapshot kubectl work by four artifacts per pod.
const maxK3KPodsPerRole = 5

// appSpec describes how to discover and pull logs from one app.
type appSpec struct {
	Label              string
	Container          string
	AuditLogContainer  string
	NamespacePrefix    string
	Namespaces         []string
	MaxPods            int
	AllNamespaces      bool
	OnePodPerNamespace bool
	SupportsFocus      bool
}

// supportedApps mirrors the choices the continuous_profiling.sh script
// supports, plus the k3k controller.
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
	// k3k is the k3k controller, which reconciles Cluster resources into the
	// per-virtual-cluster server and agent workloads. The chart names both the
	// Deployment and its container after the chart itself and labels pods with
	// the standard app.kubernetes.io selectors.
	"k3k": {
		Label:      "app.kubernetes.io/name=k3k",
		Namespaces: []string{"k3k-system"},
		Container:  "k3k",
	},
	// k3k-server and k3k-agent are the per-virtual-cluster workloads the k3k
	// controller creates, one set per cluster in its own k3k-<name> namespace.
	//
	// Container is left empty on both: k3k derives container names from the
	// virtual cluster name, so there is no fixed name to ask for.
	//
	// In shared mode the agent pod is the k3k-kubelet (labelled mode=shared);
	// in virtual mode it is a k3s agent (mode=virtual). Both are matched, since
	// a given cluster is only ever one or the other.
	"k3k-server": {
		Label:              "role=server",
		AllNamespaces:      true,
		NamespacePrefix:    "k3k-",
		OnePodPerNamespace: true,
		MaxPods:            maxK3KPodsPerRole,
		SupportsFocus:      true,
	},
	"k3k-agent": {
		Label:              "type=agent",
		AllNamespaces:      true,
		NamespacePrefix:    "k3k-",
		OnePodPerNamespace: true,
		MaxPods:            maxK3KPodsPerRole,
		SupportsFocus:      true,
	},
}

// appNames returns the supported app names in a stable order, for error
// messages.
func appNames() []string {
	names := make([]string, 0, len(supportedApps))
	for name := range supportedApps {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// Config holds the parsed flags that drive a single collect-logs run.
type Config struct {
	FocusNamespace string
	Apps           []string
	For            time.Duration
	Interval       time.Duration
}

// ParseConfig validates the raw CLI flag strings into a Config.
func ParseConfig(forStr, intervalStr, appsCSV, focusNamespace string) (Config, error) {
	cfg := Config{FocusNamespace: strings.TrimSpace(focusNamespace)}

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
			return cfg, fmt.Errorf("invalid app %q (allowed: %s)", a, strings.Join(appNames(), ", "))
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
