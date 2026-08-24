/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package logs

import (
	"bytes"
	"context"
	"fmt"
	"math"
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

// discovered records the live pods for one app at the moment discovery succeeded.
type discovered struct {
	App       string
	Spec      appSpec
	Namespace string
	Pods      []string
}

// Collect repeatedly snapshots logs (current, previous, describe, events, audit)
// from every selected Rancher-family pod for cfg.For, with cfg.Interval between
// snapshots. Each snapshot lands under outDir/snapshot-<RFC3339>/<app>/.
// A manifest.json is written when collection ends (cleanly or via SIGINT/SIGTERM).
//
// If pods are not yet present at startup, Collect retries discovery on a short
// interval until at least one app has pods, the signal context fires, or cfg.For
// elapses. This makes it safe to start in the background before a fresh deploy.
// If no app's pods ever appear, an empty manifest is still written and Collect
// returns nil.
func Collect(kubeconfig, dartFile, workspace string, cfg Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now().UTC()
	deadline := start.Add(cfg.For)

	var live []discovered
	snapshots := 0
	defer func() {
		pods := map[string][]string{}
		for _, d := range live {
			pods[d.App] = d.Pods
		}
		manifestErr := WriteManifest(outDir, Manifest{
			GeneratedAt: time.Now().UTC(),
			Start:       start,
			End:         time.Now().UTC(),
			For:         cfg.For,
			Interval:    cfg.Interval,
			Apps:        cfg.Apps,
			Pods:        pods,
			Snapshots:   snapshots,
			DartFile:    dartFile,
			Workspace:   workspace,
			Kubeconfig:  kubeconfig,
		})
		if manifestErr != nil {
			logrus.Warnf("failed to write manifest: %v", manifestErr)
		}
		logrus.Infof("logs collected to %s (%d snapshots)", outDir, snapshots)
	}()

	const discoverInterval = 20 * time.Second
	for {
		live = discoverAll(kubeconfig, cfg.Apps)
		if anyPods(live) {
			break
		}
		logrus.Infof("no pods yet for any selected app (%s); retrying in %s",
			strings.Join(cfg.Apps, ","), discoverInterval)
		select {
		case <-ctx.Done():
			logrus.Warnf("interrupted before any pods discovered; writing empty manifest")
			return nil
		case <-time.After(discoverInterval):
			if !time.Now().Before(deadline) {
				logrus.Warnf("deadline reached before any pods discovered; writing empty manifest")
				return nil
			}
		}
	}

	for _, d := range live {
		if len(d.Pods) == 0 {
			logrus.Warnf("app %s: no pods found; will be skipped this run", d.App)
			continue
		}
		logrus.Infof("app %s: found %d pod(s) in %s: %s", d.App, len(d.Pods), d.Namespace, strings.Join(d.Pods, ", "))
	}
	logrus.Infof("collecting logs every %s until %s (total %s)", cfg.Interval, deadline.Format(time.RFC3339), cfg.For)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// First snapshot uses 2x interval as --since so we don't lose the seconds
	// between Collect start and the first tick.
	if err := snapshot(ctx, kubeconfig, outDir, live, cfg.Interval*2); err != nil {
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
			if err := snapshot(ctx, kubeconfig, outDir, live, cfg.Interval); err != nil {
				logrus.Warnf("snapshot failed: %v", err)
				continue
			}
			snapshots++
		}
	}
}

// snapshot takes a single round of log artifacts for every (app, pod) pair,
// fanning out one goroutine per pod. It returns when every collection has
// finished or the context is cancelled.
func snapshot(ctx context.Context, kubeconfig, outDir string, live []discovered, since time.Duration) error {
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	snapDir := filepath.Join(outDir, "snapshot-"+ts)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return fmt.Errorf("create snapshot dir %s: %w", snapDir, err)
	}
	logrus.Infof("snapshot %s", ts)

	sinceArg := fmt.Sprintf("%ds", int(math.Ceil(since.Seconds())))

	var wg sync.WaitGroup
	for _, d := range live {
		if len(d.Pods) == 0 {
			continue
		}
		appDir := filepath.Join(snapDir, d.App)
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			logrus.Warnf("create app dir %s: %v", appDir, err)
			continue
		}
		for _, pod := range d.Pods {
			wg.Add(1)
			go func(d discovered, pod string) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				collectPod(kubeconfig, appDir, d, pod, sinceArg)
			}(d, pod)
		}
	}
	wg.Wait()
	return nil
}

// collectPod writes the four (or five, when audit) artifacts for one pod.
// Errors per-artifact are logged but never fatal — a missing previous container
// is the common case.
func collectPod(kubeconfig, appDir string, d discovered, pod, sinceArg string) {
	// Current logs
	writeKubectl(kubeconfig, filepath.Join(appDir, pod+".log"),
		"logs", "-n", d.Namespace, pod, "-c", d.Spec.Container, "--since="+sinceArg,
	)

	// Previous logs (often absent — kubectl exits non-zero, that's fine)
	writeKubectl(kubeconfig, filepath.Join(appDir, pod+"-previous.log"),
		"logs", "-n", d.Namespace, pod, "-c", d.Spec.Container, "--previous=true",
	)

	// Describe
	writeKubectl(kubeconfig, filepath.Join(appDir, pod+"-describe.txt"),
		"describe", "pod", "-n", d.Namespace, pod,
	)

	// Events for this pod
	writeKubectl(kubeconfig, filepath.Join(appDir, pod+"-events.txt"),
		"get", "events", "-n", d.Namespace, "--field-selector=involvedObject.name="+pod,
	)

	// Audit-log sidecar (rancher only)
	if d.Spec.AuditLogContainer != "" {
		writeKubectl(kubeconfig, filepath.Join(appDir, pod+"-audit.log"),
			"logs", "-n", d.Namespace, pod, "-c", d.Spec.AuditLogContainer, "--since="+sinceArg,
		)
	}
}

// writeKubectl runs kubectl with the given args and streams stdout into outPath.
// Any error is logged at debug level — most callers tolerate failure (e.g. no
// previous container, no audit sidecar on a partially-up pod).
func writeKubectl(kubeconfig, outPath string, args ...string) {
	f, err := os.Create(outPath)
	if err != nil {
		logrus.Warnf("create %s: %v", outPath, err)
		return
	}
	defer f.Close()
	if err := kubectl.Exec(kubeconfig, f, args...); err != nil {
		logrus.Debugf("kubectl %s: %v", strings.Join(args, " "), err)
	}
}

// discoverAll runs pod discovery for every requested app. The returned slice has
// one entry per app in the order cfg.Apps was given; an app with no pods still
// appears (with an empty Pods slice) so the manifest can record the attempt.
func discoverAll(kubeconfig string, apps []string) []discovered {
	out := make([]discovered, 0, len(apps))
	for _, app := range apps {
		spec, ok := supportedApps[app]
		if !ok {
			continue // ParseConfig should have caught this
		}
		ns, pods := discoverPodsInNamespaces(kubeconfig, spec)
		out = append(out, discovered{App: app, Spec: spec, Namespace: ns, Pods: pods})
	}
	return out
}

// discoverPodsInNamespaces probes each candidate namespace in order and returns
// the first one where the label selector matches at least one pod. When none
// match, it returns the last-tried namespace and a nil slice.
func discoverPodsInNamespaces(kubeconfig string, spec appSpec) (string, []string) {
	var lastNS string
	for _, ns := range spec.Namespaces {
		lastNS = ns
		pods, err := listPods(kubeconfig, ns, spec.Label)
		if err != nil {
			logrus.Debugf("list pods %s -l %s: %v", ns, spec.Label, err)
			continue
		}
		if len(pods) > 0 {
			return ns, pods
		}
	}
	return lastNS, nil
}

// listPods returns the names of all pods matching label in namespace.
func listPods(kubeconfig, namespace, label string) ([]string, error) {
	var buf bytes.Buffer
	err := kubectl.Exec(kubeconfig, &buf,
		"get", "pods", "-n", namespace, "-l", label, "-o", "name",
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

// anyPods reports whether any app has at least one pod.
func anyPods(live []discovered) bool {
	for _, d := range live {
		if len(d.Pods) > 0 {
			return true
		}
	}
	return false
}
