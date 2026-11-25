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
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rancher/dartboard/internal/kubectl"
	"github.com/sirupsen/logrus"
)

// podRef identifies one pod. Pods for an app can span namespaces, since k3k
// creates a namespace per virtual cluster, so the namespace travels with the pod
// rather than being a property of the app.
type podRef struct {
	Namespace string
	Name      string
}

// String renders a pod as "namespace/name", the form recorded in the manifest.
func (p podRef) String() string { return p.Namespace + "/" + p.Name }

// discovered records the pods selected for one app.
//
// Selection latches: pods are added until the app's cap is full and are then
// never swapped, dropped or re-ordered. Once full, the app stops listing pods
// altogether, which both keeps a pod's log history continuous for the whole run
// and avoids repeatedly listing every pod on a host running thousands of them.
type discovered struct {
	App  string
	Pods []podRef
	Spec appSpec
	full bool
}

// atCapacity reports whether this app has latched its full complement of pods.
// An uncapped app is never at capacity, so it keeps picking up new pods.
func (d *discovered) atCapacity() bool {
	return d.Spec.MaxPods > 0 && len(d.Pods) >= d.Spec.MaxPods
}

// needsFocus reports whether the focus namespace is still unrepresented for this
// app. The focus pod is additive rather than occupying one of the capped slots,
// so it is still picked up when it appears after the cap is already full.
func (d *discovered) needsFocus(focusNamespace string) bool {
	return focusNamespace != "" && !d.hasNamespace(focusNamespace)
}

// hasPod reports whether a pod is already selected.
func (d *discovered) hasPod(p podRef) bool {
	for _, existing := range d.Pods {
		if existing == p {
			return true
		}
	}

	return false
}

// hasNamespace reports whether any selected pod already comes from a namespace.
func (d *discovered) hasNamespace(ns string) bool {
	for _, existing := range d.Pods {
		if existing.Namespace == ns {
			return true
		}
	}

	return false
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
//
// Capped apps keep latching newly seen pods before each snapshot until their cap
// is full, and then stop discovering entirely. Starting before the workload
// exists therefore still ends up tracking a full complement, without the pod set
// churning mid-run or a large pod list being fetched on every snapshot.
func Collect(kubeconfig, dartFile, workspace string, cfg Config, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now().UTC()
	deadline := start.Add(cfg.For)

	live := newDiscovered(cfg.Apps)
	snapshots := 0

	defer func() {
		pods := map[string][]string{}

		for _, d := range live {
			names := make([]string, 0, len(d.Pods))
			for _, p := range d.Pods {
				names = append(names, p.String())
			}

			pods[d.App] = names
		}

		manifestErr := WriteManifest(outDir, Manifest{
			GeneratedAt:    time.Now().UTC(),
			Start:          start,
			End:            time.Now().UTC(),
			For:            cfg.For,
			Interval:       cfg.Interval,
			Apps:           cfg.Apps,
			FocusNamespace: cfg.FocusNamespace,
			Pods:           pods,
			Snapshots:      snapshots,
			DartFile:       dartFile,
			Workspace:      workspace,
			Kubeconfig:     kubeconfig,
		})
		if manifestErr != nil {
			logrus.Warnf("failed to write manifest: %v", manifestErr)
		}

		logrus.Infof("logs collected to %s (%d snapshots)", outDir, snapshots)
	}()

	const discoverInterval = 20 * time.Second

	for {
		reconcile(kubeconfig, live, cfg.FocusNamespace)

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

	logSelection(live)
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
			// Top up any app that has not latched its full complement yet. Apps
			// that are done are skipped entirely, so a settled run does no pod
			// listing at all.
			before := podCount(live)
			reconcile(kubeconfig, live, cfg.FocusNamespace)

			if podCount(live) != before {
				logSelection(live)
			}

			if err := snapshot(ctx, kubeconfig, outDir, live, cfg.Interval); err != nil {
				logrus.Warnf("snapshot failed: %v", err)
				continue
			}

			snapshots++
		}
	}
}

// newDiscovered builds the initial, empty selection state for each app.
func newDiscovered(apps []string) []discovered {
	out := make([]discovered, 0, len(apps))

	for _, app := range apps {
		spec, ok := supportedApps[app]
		if !ok {
			continue // ParseConfig should have caught this
		}

		out = append(out, discovered{App: app, Spec: spec})
	}

	return out
}

// podCount totals the pods selected across every app.
func podCount(live []discovered) int {
	n := 0
	for i := range live {
		n += len(live[i].Pods)
	}

	return n
}

// logSelection reports what is currently being tracked, per app.
func logSelection(live []discovered) {
	for i := range live {
		d := &live[i]
		if len(d.Pods) == 0 {
			logrus.Warnf("app %s: no pods found yet", d.App)
			continue
		}

		names := make([]string, 0, len(d.Pods))
		for _, p := range d.Pods {
			names = append(names, p.String())
		}

		state := "latching"
		if d.full {
			state = "final"
		}

		logrus.Infof("app %s: tracking %d pod(s) [%s]: %s",
			d.App, len(d.Pods), state, strings.Join(names, ", "))
	}
}

// maxConcurrentPods bounds how many pods are collected at once. Each pod costs
// four or five kubectl invocations, so an unbounded fan-out over a large
// selection would spawn hundreds of processes per snapshot.
const maxConcurrentPods = 8

// snapshot takes a single round of log artifacts for every (app, pod) pair,
// with at most maxConcurrentPods in flight. It returns when every collection has
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

	sem := make(chan struct{}, maxConcurrentPods)

	for i := range live {
		d := live[i]
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

			go func(d discovered, pod podRef) {
				defer wg.Done()

				if ctx.Err() != nil {
					return
				}

				sem <- struct{}{}

				defer func() { <-sem }()

				collectPod(kubeconfig, appDir, d, pod, sinceArg)
			}(d, pod)
		}
	}

	wg.Wait()

	return nil
}

// podFileBase returns the filename stem for a pod's artifacts. Apps discovered
// across namespaces get the namespace folded in, since two virtual clusters can
// otherwise produce identically named pods in one snapshot directory.
func podFileBase(spec appSpec, pod podRef) string {
	if spec.AllNamespaces {
		return pod.Namespace + "_" + pod.Name
	}

	return pod.Name
}

// containerArgs selects which container(s) to read. k3k names containers after
// the virtual cluster, so those apps leave Container empty and read them all.
func containerArgs(spec appSpec) []string {
	if spec.Container != "" {
		return []string{"-c", spec.Container}
	}

	return []string{"--all-containers=true"}
}

// collectPod writes the four (or five, when audit) artifacts for one pod.
// Errors per-artifact are logged but never fatal — a missing previous container
// is the common case.
func collectPod(kubeconfig, appDir string, d discovered, pod podRef, sinceArg string) {
	base := podFileBase(d.Spec, pod)
	container := containerArgs(d.Spec)

	// Current logs
	writeKubectl(kubeconfig, filepath.Join(appDir, base+".log"),
		append([]string{"logs", "-n", pod.Namespace, pod.Name, "--since=" + sinceArg}, container...)...,
	)

	// Previous logs (often absent — kubectl exits non-zero, that's fine)
	writeKubectl(kubeconfig, filepath.Join(appDir, base+"-previous.log"),
		append([]string{"logs", "-n", pod.Namespace, pod.Name, "--previous=true"}, container...)...,
	)

	// Describe
	writeKubectl(kubeconfig, filepath.Join(appDir, base+"-describe.txt"),
		"describe", "pod", "-n", pod.Namespace, pod.Name,
	)

	// Events for this pod
	writeKubectl(kubeconfig, filepath.Join(appDir, base+"-events.txt"),
		"get", "events", "-n", pod.Namespace, "--field-selector=involvedObject.name="+pod.Name,
	)

	// Audit-log sidecar (rancher only)
	if d.Spec.AuditLogContainer != "" {
		writeKubectl(kubeconfig, filepath.Join(appDir, base+"-audit.log"),
			"logs", "-n", pod.Namespace, pod.Name, "-c", d.Spec.AuditLogContainer, "--since="+sinceArg,
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

// reconcile tops up each app's selection with pods it has not latched yet.
//
// An app that has latched everything it is going to track is skipped, so a
// settled run issues no pod listings at all. The one exception is an app still
// missing the focus namespace: that is chased with a single-namespace lookup,
// which costs the same regardless of how many pods the host is running.
func reconcile(kubeconfig string, live []discovered, focusNamespace string) {
	for i := range live {
		d := &live[i]

		if !d.full {
			candidates, err := discoverPods(kubeconfig, d.Spec)
			if err != nil {
				logrus.Debugf("app %s: discover pods: %v", d.App, err)
			} else {
				latch(d, candidates, focusNamespace)
			}
		}

		if d.Spec.SupportsFocus && d.needsFocus(focusNamespace) {
			latchFocus(kubeconfig, d, focusNamespace)
		}
	}
}

// latch appends previously unseen pods, in candidate order, until the app's cap
// is reached. Selected pods are never removed or re-ordered, so a pod's log
// history stays continuous for the whole run.
func latch(d *discovered, candidates []podRef, focusNamespace string) {
	for _, c := range candidates {
		if d.atCapacity() {
			break
		}

		if d.hasPod(c) {
			continue
		}

		// One pod per namespace keeps a single virtual cluster's DaemonSet from
		// filling every slot; see appSpec.OnePodPerNamespace.
		if d.Spec.OnePodPerNamespace && d.hasNamespace(c.Namespace) {
			continue
		}

		d.Pods = append(d.Pods, c)
	}

	markFull(d, focusNamespace)
}

// latchFocus adds the focus namespace's pod on top of the capped selection, so
// a specific virtual cluster is covered even when it appeared after the cap
// filled up. It queries only that namespace, so it is cheap to retry.
func latchFocus(kubeconfig string, d *discovered, focusNamespace string) {
	pods, err := listPodsInNamespace(kubeconfig, focusNamespace, d.Spec.Label)
	if err != nil {
		logrus.Debugf("app %s: list focus namespace %s: %v", d.App, focusNamespace, err)
		return
	}

	if len(pods) == 0 {
		return
	}

	d.Pods = append(d.Pods, pods[0])
	logrus.Infof("app %s: added focus pod %s", d.App, pods[0])

	markFull(d, focusNamespace)
}

// markFull latches an app once it has everything it is going to track.
//
// Uncapped apps (the Rancher-family ones) settle as soon as they find any pods,
// which reproduces the original discover-once behaviour exactly.
func markFull(d *discovered, focusNamespace string) {
	if d.Spec.SupportsFocus && d.needsFocus(focusNamespace) {
		return
	}

	if d.Spec.MaxPods == 0 {
		d.full = len(d.Pods) > 0
		return
	}

	d.full = d.atCapacity()
}

// discoverPods lists the candidate pods for an app, in a stable order.
func discoverPods(kubeconfig string, spec appSpec) ([]podRef, error) {
	if spec.AllNamespaces {
		return listPodsAllNamespaces(kubeconfig, spec.Label, spec.NamespacePrefix)
	}

	return discoverPodsInNamespaces(kubeconfig, spec), nil
}

// discoverPodsInNamespaces probes each candidate namespace in order and returns
// the pods from the first one where the label selector matches.
func discoverPodsInNamespaces(kubeconfig string, spec appSpec) []podRef {
	for _, ns := range spec.Namespaces {
		pods, err := listPodsInNamespace(kubeconfig, ns, spec.Label)
		if err != nil {
			logrus.Debugf("list pods %s -l %s: %v", ns, spec.Label, err)
			continue
		}

		if len(pods) > 0 {
			return pods
		}
	}

	return nil
}

// listPodsInNamespace returns all pods matching label in one namespace, sorted
// by name.
func listPodsInNamespace(kubeconfig, namespace, label string) ([]podRef, error) {
	var buf bytes.Buffer

	err := kubectl.Exec(kubeconfig, &buf,
		"get", "pods", "-n", namespace, "-l", label, "-o", "name",
	)
	if err != nil {
		return nil, err
	}

	var pods []podRef

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		pods = append(pods, podRef{Namespace: namespace, Name: strings.TrimPrefix(line, "pod/")})
	}

	sortPods(pods)

	return pods, nil
}

// listPodsAllNamespaces returns every pod matching label whose namespace starts
// with prefix, sorted by namespace then name. k3k creates one namespace per
// virtual cluster at runtime, so the set cannot be enumerated up front.
func listPodsAllNamespaces(kubeconfig, label, prefix string) ([]podRef, error) {
	var buf bytes.Buffer

	err := kubectl.Exec(kubeconfig, &buf,
		"get", "pods", "--all-namespaces", "-l", label,
		"-o", "custom-columns=:metadata.namespace,:metadata.name", "--no-headers",
	)
	if err != nil {
		return nil, err
	}

	var pods []podRef

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		if prefix != "" && !strings.HasPrefix(fields[0], prefix) {
			continue
		}

		pods = append(pods, podRef{Namespace: fields[0], Name: fields[1]})
	}

	sortPods(pods)

	return pods, nil
}

// sortPods orders pods by namespace then name, so selection is deterministic
// and reproducible across runs rather than dependent on API listing order.
func sortPods(pods []podRef) {
	sort.Slice(pods, func(i, j int) bool {
		if pods[i].Namespace != pods[j].Namespace {
			return pods[i].Namespace < pods[j].Namespace
		}

		return pods[i].Name < pods[j].Name
	})
}

// anyPods reports whether any app has at least one pod.
func anyPods(live []discovered) bool {
	for i := range live {
		if len(live[i].Pods) > 0 {
			return true
		}
	}

	return false
}
