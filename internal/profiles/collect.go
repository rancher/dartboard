/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package profiles

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/sirupsen/logrus"
)

const (
	rancherNamespace = "cattle-system"
	rancherContainer = "rancher"
	rancherLabel     = "app=rancher"
	pprofPort        = 6060
)

// Collect repeatedly snapshots pprof profiles from every Rancher pod in
// cattle-system for cfg.For, with cfg.Interval between snapshots. Each snapshot
// goes into outDir/snapshot-<RFC3339>/<pod>-<type>.pprof. A manifest.json is
// written when collection ends (cleanly or via SIGINT).
//
// If Rancher pods are not yet present at startup, Collect retries discovery on
// a short interval until pods appear, the signal context fires, or cfg.For
// elapses. This makes it safe to start in the background before a fresh deploy
// brings up Rancher; if pods never appear, an empty manifest is still written
// and Collect returns nil.
func Collect(kubeconfig, dartFile, workspace string, cfg Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now().UTC()
	deadline := start.Add(cfg.For)

	var pods []string
	snapshots := 0
	defer func() {
		manifestErr := WriteManifest(outDir, Manifest{
			GeneratedAt: time.Now().UTC(),
			Start:       start,
			End:         time.Now().UTC(),
			For:         cfg.For,
			Interval:    cfg.Interval,
			CPUDuration: cfg.CPUDuration,
			Profiles:    cfg.Profiles,
			Pods:        pods,
			Snapshots:   snapshots,
			DartFile:    dartFile,
			Workspace:   workspace,
			Kubeconfig:  kubeconfig,
		})
		if manifestErr != nil {
			logrus.Warnf("failed to write manifest: %v", manifestErr)
		}
		logrus.Infof("profiles collected to %s (%d snapshots)", outDir, snapshots)
	}()

	const discoverInterval = 20 * time.Second
	for {
		p, derr := discoverRancherPods(kubeconfig)
		if derr == nil && len(p) > 0 {
			pods = p
			break
		}
		if derr != nil {
			logrus.Debugf("discover Rancher pods (will retry): %v", derr)
		} else {
			logrus.Infof("no Rancher pods yet in %s (label %s); retrying in %s",
				rancherNamespace, rancherLabel, discoverInterval)
		}
		select {
		case <-ctx.Done():
			logrus.Warnf("interrupted before any Rancher pods discovered; writing empty manifest")
			return nil
		case <-time.After(discoverInterval):
			if !time.Now().Before(deadline) {
				logrus.Warnf("deadline reached before any Rancher pods discovered; writing empty manifest")
				return nil
			}
		}
	}
	logrus.Infof("found %d Rancher pod(s): %s", len(pods), strings.Join(pods, ", "))
	logrus.Infof("collecting profiles every %s until %s (total %s)", cfg.Interval, deadline.Format(time.RFC3339), cfg.For)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Take the first snapshot immediately rather than waiting one full interval.
	if err := snapshot(ctx, kubeconfig, outDir, pods, cfg); err != nil {
		logrus.Warnf("snapshot failed: %v", err)
	} else {
		snapshots++
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Infof("interrupt received, stopping after %d snapshot(s)", snapshots)
			return nil
		case t := <-ticker.C:
			if !t.Before(deadline) {
				return nil
			}
			if err := snapshot(ctx, kubeconfig, outDir, pods, cfg); err != nil {
				logrus.Warnf("snapshot failed: %v", err)
				continue
			}
			snapshots++
		}
	}
}

// snapshot takes a single round of profiles for every pod in pods, fanning out
// one goroutine per (pod, profile) pair. It returns when every collection has
// finished or the context is cancelled.
func snapshot(ctx context.Context, kubeconfig, outDir string, pods []string, cfg Config) error {
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	snapDir := filepath.Join(outDir, "snapshot-"+ts)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return fmt.Errorf("create snapshot dir %s: %w", snapDir, err)
	}
	logrus.Infof("snapshot %s", ts)

	var wg sync.WaitGroup
	for _, pod := range pods {
		for _, profile := range cfg.Profiles {
			wg.Add(1)
			go func(pod, profile string) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				if err := fetchProfile(kubeconfig, snapDir, pod, profile, cfg.CPUDuration); err != nil {
					logrus.Warnf("fetch %s from %s: %v", profile, pod, err)
				}
			}(pod, profile)
		}
	}
	wg.Wait()
	return nil
}

// fetchProfile execs into one Rancher pod and curls the pprof endpoint,
// streaming the binary response into snapDir/<pod>-<profile>.pprof.
func fetchProfile(kubeconfig, snapDir, pod, profile string, cpuDuration time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d/debug/pprof/%s", pprofPort, profile)
	if profile == "profile" {
		url = fmt.Sprintf("%s?seconds=%d", url, int(cpuDuration.Seconds()))
	}

	outPath := filepath.Join(snapDir, fmt.Sprintf("%s-%s.pprof", pod, profile))
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer f.Close()

	return kubectl.Exec(kubeconfig, f,
		"exec", "-n", rancherNamespace, pod, "-c", rancherContainer,
		"--", "curl", "-s", url,
	)
}

// discoverRancherPods returns the names of all Rancher pods in cattle-system.
func discoverRancherPods(kubeconfig string) ([]string, error) {
	var buf bytes.Buffer
	err := kubectl.Exec(kubeconfig, &buf,
		"get", "pods", "-n", rancherNamespace, "-l", rancherLabel, "-o", "name",
	)
	if err != nil {
		return nil, err
	}
	var pods []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pods = append(pods, strings.TrimPrefix(line, "pod/"))
	}
	return pods, nil
}
