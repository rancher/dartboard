/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package profiles

import (
	"fmt"
	"strings"
	"time"
)

// Allowed pprof endpoints exposed by Rancher at http://localhost:6060/debug/pprof/.
var allowedProfileTypes = map[string]struct{}{
	"goroutine":    {},
	"heap":         {},
	"threadcreate": {},
	"block":        {},
	"mutex":        {},
	"profile":      {},
}

// Config holds the parsed flags that drive a single collect-profiles run.
type Config struct {
	For         time.Duration
	Interval    time.Duration
	Profiles    []string
	CPUDuration time.Duration
}

// ParseConfig validates the raw CLI flag strings into a Config.
func ParseConfig(forStr, intervalStr, profilesCSV, cpuDurationStr string) (Config, error) {
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

	cpuD, err := time.ParseDuration(cpuDurationStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid --cpu-duration: %w", err)
	}
	if cpuD <= 0 {
		return cfg, fmt.Errorf("--cpu-duration must be positive")
	}
	cfg.CPUDuration = cpuD

	parts := strings.Split(profilesCSV, ",")
	cfg.Profiles = make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := allowedProfileTypes[p]; !ok {
			return cfg, fmt.Errorf("invalid profile type %q (allowed: goroutine, heap, threadcreate, block, mutex, profile)", p)
		}
		cfg.Profiles = append(cfg.Profiles, p)
	}
	if len(cfg.Profiles) == 0 {
		return cfg, fmt.Errorf("--profiles must contain at least one type")
	}

	return cfg, nil
}
