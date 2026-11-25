/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Query is one entry in the curated PromQL catalog.
type Query struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Group       string `json:"group"` // cpu | memory | disk | network | pod | k3k
	PromQL      string `json:"promql"`
	Unit        string `json:"unit"`
}

// Preset names a curated set of queries tailored to one kind of workload.
type Preset string

const (
	// PresetRancher focuses on the Rancher and Fleet control plane.
	PresetRancher Preset = "rancher"
	// PresetK3K focuses on a k3k host cluster: the k3k controller plus the
	// per-virtual-cluster server and agent workloads it manages.
	PresetK3K Preset = "k3k"
)

// RancherNamespaces matches the namespaces hosting Rancher components and
// fleet so the catalog focuses on the workload we want to size for.
const RancherNamespaces = `cattle-.*|fleet-.*`

// K3KNamespaces matches the k3k controller namespace plus the k3k-<name>
// namespaces the controller creates, one per virtual cluster.
const K3KNamespaces = `k3k-.*`

// k3k names server and shared-kubelet containers after the cluster. Virtual
// agent pods are named k3k-<cluster>-agent-<replicaset>-<suffix>, while their
// container name is k3k-agent. Matching on container labels avoids a
// kube-state-metrics join because cAdvisor already carries the label.
//
// The suffix is lost only when the generated name reaches 64 characters and k3k
// replaces the tail with a hash, which needs a virtual cluster name of roughly
// fifty characters; well beyond the <prefix>-NNN scheme bulk runs use.
const (
	k3kServerContainers     = `.*-server`
	k3kKubeletContainers    = `.*-kubelet`
	k3kAgentContainers      = `k3k-agent`
	k3kControllerContainers = `k3k`
)

// k3kTopPods bounds how many per-pod series the role-scoped queries return. It
// matches the per-role pod cap in internal/logs, so the metrics and the logs
// describe a comparable slice of the fleet. The unbounded per-role aggregates
// alongside them still describe every pod.
const k3kTopPods = 5

// CatalogOptions selects which queries Catalog returns.
type CatalogOptions struct {
	// Preset picks the curated query set. Empty means PresetRancher.
	Preset Preset
	// Namespaces overrides the preset's default namespace regex. It is
	// interpolated into PromQL `namespace=~"..."` matchers.
	Namespaces string
	// FocusNamespace, when set, adds unbounded series for one virtual cluster's
	// pods, so a cluster of interest cannot be crowded out of the topk-bounded
	// per-pod queries by busier neighbours.
	FocusNamespace string
	K3KRunPrefix   string
	K3KStartIndex  int
	K3KCount       int
	Operational    bool
}

func (o CatalogOptions) Validate() error {
	hasRunSelector := o.K3KRunPrefix != "" || o.K3KStartIndex != 0 || o.K3KCount != 0
	if !hasRunSelector {
		return nil
	}

	if o.preset() != PresetK3K {
		return fmt.Errorf("k3k run selection requires --preset k3k")
	}

	if o.Namespaces != "" {
		return fmt.Errorf("--namespaces conflicts with k3k run selection")
	}

	if o.K3KRunPrefix == "" || o.K3KStartIndex < 1 || o.K3KCount < 1 {
		return fmt.Errorf("k3k run selection requires a prefix, start index >= 1, and count >= 1")
	}

	return nil
}

// ParsePreset validates a preset name from the command line. An empty string
// selects PresetRancher, preserving the historical default.
func ParsePreset(s string) (Preset, error) {
	switch Preset(s) {
	case "":
		return PresetRancher, nil
	case PresetRancher:
		return PresetRancher, nil
	case PresetK3K:
		return PresetK3K, nil
	default:
		return "", fmt.Errorf("invalid --preset %q (allowed: %s, %s)", s, PresetRancher, PresetK3K)
	}
}

// preset returns the effective preset, treating the zero value as
// PresetRancher so a zero CatalogOptions keeps the historical behaviour.
func (o CatalogOptions) preset() Preset {
	if o.Preset == "" {
		return PresetRancher
	}

	return o.Preset
}

// perPodLimit caps the role-blind per-pod queries. A k3k host runs a pod per
// virtual cluster in the matched namespaces, so leaving those unbounded would
// put thousands of series into a single CSV; Rancher runs stay uncapped, which
// preserves the original behaviour exactly.
func (o CatalogOptions) perPodLimit() int {
	if o.preset() == PresetK3K {
		// Both roles, so twice the per-role bound.
		return 2 * k3kTopPods
	}

	return 0
}

// topk bounds a query to the n highest series, or returns it unchanged when n
// is zero.
func topk(n int, query string) string {
	if n <= 0 {
		return query
	}

	return fmt.Sprintf("topk(%d, %s)", n, query)
}

// namespaces returns the namespace regex to interpolate into the catalog's
// PromQL, honouring an explicit override before the preset default.
func (o CatalogOptions) namespaces() string {
	if o.Namespaces != "" {
		return o.Namespaces
	}

	if o.preset() == PresetK3K {
		if o.K3KRunPrefix != "" {
			return k3kRunNamespaces(o.K3KRunPrefix, o.K3KStartIndex, o.K3KCount)
		}

		return K3KNamespaces
	}

	return RancherNamespaces
}

func k3kRunNamespaces(prefix string, start, count int) string {
	end := start + count - 1
	patterns := make([]string, 0)

	for width := max(3, len(strconv.Itoa(start))); width <= max(3, len(strconv.Itoa(end))); width++ {
		low := start
		if width > 3 {
			low = max(low, pow10(width-1))
		}

		high := min(end, pow10(width)-1)
		if low <= high {
			patterns = append(patterns, fixedWidthRange(fmt.Sprintf("%0*d", width, low), fmt.Sprintf("%0*d", width, high))...)
		}
	}

	escapedPrefix := strings.ReplaceAll(regexp.QuoteMeta(prefix), `\`, `\\`)
	// Prometheus label regexes use RE2, which does not support non-capturing
	// groups. Capturing groups are equivalent here and work across versions.
	return `(k3k-system|k3k-` + escapedPrefix + `-(` + strings.Join(patterns, "|") + `))`
}

func fixedWidthRange(low, high string) []string {
	if low == high {
		return []string{low}
	}

	common := 0
	for common < len(low) && low[common] == high[common] {
		common++
	}

	if common > 0 {
		out := fixedWidthRange(low[common:], high[common:])
		for i := range out {
			out[i] = low[:common] + out[i]
		}

		return out
	}

	out := make([]string, 0)
	firstLow := int(low[0] - '0')

	firstHigh := int(high[0] - '0')
	for digit := firstLow; digit <= firstHigh; digit++ {
		suffixLow := strings.Repeat("0", len(low)-1)

		suffixHigh := strings.Repeat("9", len(low)-1)
		if digit == firstLow {
			suffixLow = low[1:]
		}

		if digit == firstHigh {
			suffixHigh = high[1:]
		}

		if suffixLow == suffixHigh {
			out = append(out, strconv.Itoa(digit)+suffixLow)
			continue
		}

		if suffixLow == strings.Repeat("0", len(suffixLow)) && suffixHigh == strings.Repeat("9", len(suffixHigh)) {
			out = append(out, strconv.Itoa(digit)+`[0-9]{`+strconv.Itoa(len(suffixLow))+`}`)
			continue
		}

		for _, suffix := range fixedWidthRange(suffixLow, suffixHigh) {
			out = append(out, strconv.Itoa(digit)+suffix)
		}
	}

	return out
}

func pow10(power int) int {
	value := 1
	for range power {
		value *= 10
	}

	return value
}

// Catalog returns the list of queries we collect for VM/machine sizing
// analysis. Names are file-safe (suitable for use as CSV filenames).
func Catalog(opts CatalogOptions) []Query {
	queries := append(nodeResourceQueries(), workloadResourceQueries(opts.namespaces(), opts.perPodLimit())...)

	if opts.preset() == PresetK3K {
		queries = append(queries, k3kRoleQueries(opts.namespaces())...)

		if opts.FocusNamespace != "" {
			queries = append(queries, k3kFocusResourceQueries(opts.FocusNamespace)...)
		}

		if opts.Operational {
			queries = append(queries, nodeOperationalQueries()...)

			queries = append(queries, k3kOperationalQueries(opts.namespaces())...)
			if opts.FocusNamespace != "" {
				queries = append(queries, k3kFocusOperationalQueries(opts.FocusNamespace)...)
			}
		}
	}

	if opts.Operational && opts.preset() != PresetK3K {
		queries = append(queries, nodeOperationalQueries()...)
	}

	return queries
}

// nodeResourceQueries are host-level and independent of the workload under test, so
// they are collected for every preset.
func nodeResourceQueries() []Query {
	return []Query{
		// CPU
		{
			Name:        "node_cpu_utilization",
			Description: "Per-node CPU utilization ratio (1 - idle).",
			Group:       "cpu",
			Unit:        "ratio",
			PromQL:      `1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))`,
		},
		// Memory
		{
			Name:        "node_mem_used_bytes",
			Description: "Per-node memory used (MemTotal - MemAvailable).",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      `node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes`,
		},

		// Disk
		{
			Name:        "node_disk_throughput",
			Description: "Per-node disk read+write throughput.",
			Group:       "disk",
			Unit:        "bytes/sec",
			PromQL:      `sum by (instance, device) (rate(node_disk_read_bytes_total[5m]) + rate(node_disk_written_bytes_total[5m]))`,
		},
		{
			Name:        "node_disk_iops",
			Description: "Per-node disk IOPS (reads + writes).",
			Group:       "disk",
			Unit:        "iops",
			PromQL:      `sum by (instance, device) (rate(node_disk_reads_completed_total[5m]) + rate(node_disk_writes_completed_total[5m]))`,
		},
		{
			Name:        "node_filesystem_utilization",
			Description: "Per-node filesystem utilization ratio (1 - free/size).",
			Group:       "disk",
			Unit:        "ratio",
			PromQL:      `1 - node_filesystem_avail_bytes{fstype!~"tmpfs|overlay|squashfs"} / node_filesystem_size_bytes{fstype!~"tmpfs|overlay|squashfs"}`,
		},

		// Network
		{
			Name:        "node_net_throughput",
			Description: "Per-node network rx+tx bytes/sec.",
			Group:       "network",
			Unit:        "bytes/sec",
			PromQL:      `sum by (instance, device) (rate(node_network_receive_bytes_total{device!~"lo|veth.*|docker.*|cni.*"}[5m]) + rate(node_network_transmit_bytes_total{device!~"lo|veth.*|docker.*|cni.*"}[5m]))`,
		},
	}
}

func nodeOperationalQueries() []Query {
	return []Query{
		{Name: "node_load1", Description: "Per-node 1-minute load average.", Group: "operational", Unit: "load", PromQL: `node_load1`},
		{Name: "node_net_errors", Description: "Per-node network errors+drops (rx+tx).", Group: "operational", Unit: "events/sec", PromQL: `sum by (instance, device) (rate(node_network_receive_errs_total[5m]) + rate(node_network_receive_drop_total[5m]) + rate(node_network_transmit_errs_total[5m]) + rate(node_network_transmit_drop_total[5m]))`},
	}
}

// workloadResourceQueries scope per-pod usage and resource headroom to the namespaces
// hosting the workload under test. Query names are kept stable across presets
// so downstream tooling (CSV filenames, summary.json keys) does not have to
// branch on which preset produced a run.
// limit is the number of per-pod series to keep, or 0 to keep all of them. A
// k3k host runs thousands of pods in the matched namespaces, so those runs cap
// the per-pod queries; Rancher runs have few enough pods to keep every series.
func workloadResourceQueries(namespaces string, limit int) []Query {
	return []Query{
		{
			Name:        "rancher_pod_cpu",
			Description: "Per-pod CPU usage in cores, for the selected namespaces.",
			Group:       "cpu",
			Unit:        "cores",
			PromQL:      topk(limit, `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace=~"`+namespaces+`", container!="", container!="POD"}[5m]))`),
		},
		{
			Name:        "rancher_pod_workingset",
			Description: "Per-pod working set memory, for the selected namespaces.",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      topk(limit, `sum by (namespace, pod) (container_memory_working_set_bytes{namespace=~"`+namespaces+`", container!="", container!="POD"})`),
		},
		{
			Name:        "rancher_pod_rss",
			Description: "Per-pod RSS memory, for the selected namespaces.",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      topk(limit, `sum by (namespace, pod) (container_memory_rss{namespace=~"`+namespaces+`", container!="", container!="POD"})`),
		},

		{Name: "pod_cpu_requests", Description: "Largest per-pod CPU requests.", Group: "cpu", Unit: "cores", PromQL: topk(limit, `sum by (namespace, pod) (kube_pod_container_resource_requests{namespace=~"`+namespaces+`", resource="cpu", unit="core"})`)},
		{Name: "pod_cpu_limits", Description: "Largest per-pod CPU limits.", Group: "cpu", Unit: "cores", PromQL: topk(limit, `sum by (namespace, pod) (kube_pod_container_resource_limits{namespace=~"`+namespaces+`", resource="cpu", unit="core"})`)},
		{Name: "pod_memory_requests", Description: "Largest per-pod memory requests.", Group: "memory", Unit: "bytes", PromQL: topk(limit, `sum by (namespace, pod) (kube_pod_container_resource_requests{namespace=~"`+namespaces+`", resource="memory", unit="byte"})`)},
		{Name: "pod_memory_limits", Description: "Largest per-pod memory limits.", Group: "memory", Unit: "bytes", PromQL: topk(limit, `sum by (namespace, pod) (kube_pod_container_resource_limits{namespace=~"`+namespaces+`", resource="memory", unit="byte"})`)},
	}
}

// k3kOperationalQueries capture how a k3k host cluster behaves as virtual cluster count
// grows: how many vclusters exist, whether their pods stay Running, how much
// they restart, how densely they pack onto nodes, how much local-path storage
// they consume, and how hard the k3k controller itself is working.
//
// The controller reconcile and workqueue series come from the k3k controller's
// controller-runtime metrics endpoint, which is only present in Prometheus when
// the k3k-monitoring chart (charts/k3k-monitoring) has been installed to scrape
// it. The remaining series come from kube-state-metrics and the kubelet, so
// they work against any k3k host with rancher-monitoring installed.
func k3kOperationalQueries(namespaces string) []Query {
	return []Query{
		{
			Name:        "k3k_vcluster_count",
			Description: "Number of k3k virtual cluster namespaces observed on the host.",
			Group:       "k3k",
			Unit:        "count",
			PromQL:      `count(count by (namespace) (kube_pod_info{namespace=~"` + namespaces + `", namespace!="k3k-system"}))`,
		},
		{
			Name:        "k3k_pod_phase",
			Description: "Pod count by namespace and phase across k3k namespaces.",
			Group:       "k3k",
			Unit:        "count",
			PromQL:      `sum by (namespace, phase) (kube_pod_status_phase{namespace=~"` + namespaces + `"})`,
		},
		{
			Name:        "k3k_pod_restarts",
			Description: "Cumulative container restarts across k3k namespaces.",
			Group:       "k3k",
			Unit:        "count",
			PromQL:      `sum by (namespace, pod) (kube_pod_container_status_restarts_total{namespace=~"` + namespaces + `"})`,
		},
		{
			Name:        "k3k_pods_per_node",
			Description: "k3k pods scheduled per node, to spot packing limits.",
			Group:       "k3k",
			Unit:        "count",
			PromQL:      `count by (node) (kube_pod_info{namespace=~"` + namespaces + `"})`,
		},
		{
			Name:        "k3k_volume_used_bytes",
			Description: "Persistent volume bytes used by k3k virtual clusters when persistent storage is enabled.",
			Group:       "operational",
			Unit:        "bytes",
			PromQL:      `sum by (namespace, persistentvolumeclaim) (kubelet_volume_stats_used_bytes{namespace=~"` + namespaces + `"})`,
		},
		{
			Name:        "k3k_controller_reconcile_rate",
			Description: "k3k controller reconciliations per second by controller and result.",
			Group:       "k3k",
			Unit:        "ops/sec",
			PromQL:      `sum by (controller, result) (rate(controller_runtime_reconcile_total{namespace="k3k-system"}[5m]))`,
		},
		{
			Name:        "k3k_controller_reconcile_errors",
			Description: "k3k controller reconciliation errors per second by controller.",
			Group:       "k3k",
			Unit:        "ops/sec",
			PromQL:      `sum by (controller) (rate(controller_runtime_reconcile_errors_total{namespace="k3k-system"}[5m]))`,
		},
		{
			Name:        "k3k_controller_workqueue_depth",
			Description: "k3k controller workqueue depth, the primary backlog signal.",
			Group:       "k3k",
			Unit:        "count",
			PromQL:      `sum by (name) (workqueue_depth{namespace="k3k-system"})`,
		},
	}
}

// k3kRoleQueries split the virtual cluster workload into server, shared-mode
// kubelet, virtual-mode agent, and controller roles. Kubelet and agent roles
// are optional because their pods depend on cluster mode and agent count.
//
// Each role gets both a bounded per-pod view and unbounded fleet aggregates.
// The per-pod queries are the ones that would otherwise explode: one series per
// virtual cluster, thousands of them on a loaded host. The aggregates are a
// handful of series no matter how large the fleet, so they still describe every
// pod and are what the totals should be read from.
func k3kRoleQueries(namespaces string) []Query {
	roles := []struct {
		name       string
		label      string
		containers string
	}{
		{"server", "server", k3kServerContainers},
		{"kubelet", "shared-mode kubelet", k3kKubeletContainers},
		{"agent", "virtual-mode agent", k3kAgentContainers},
		{"controller", "controller", k3kControllerContainers},
	}

	var out []Query

	for _, r := range roles {
		cpu := `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace=~"` +
			namespaces + `", container=~"` + r.containers + `"}[5m]))`
		mem := `sum by (namespace, pod) (container_memory_working_set_bytes{namespace=~"` +
			namespaces + `", container=~"` + r.containers + `"})`

		out = append(out,
			Query{
				Name:        "k3k_" + r.name + "_cpu_per_active_pod_millicores",
				Description: "Average CPU per active k3k " + r.label + " pod, in millicores.",
				Group:       "k3k",
				Unit:        "millicores",
				PromQL:      `1000 * sum (` + cpu + `) / count (` + mem + `)`,
			},
			Query{
				Name:        "k3k_" + r.name + "_memory_per_active_pod_mb",
				Description: "Average working set memory per active k3k " + r.label + " pod, in decimal MB.",
				Group:       "k3k",
				Unit:        "MB",
				PromQL:      `sum (` + mem + `) / count (` + mem + `) / 1000000`,
			},
			Query{
				Name:        "k3k_" + r.name + "_pod_cpu",
				Description: "Busiest k3k " + r.label + " pods by CPU, in cores.",
				Group:       "k3k",
				Unit:        "cores",
				PromQL:      topk(k3kTopPods, cpu),
			},
			Query{
				Name:        "k3k_" + r.name + "_pod_workingset",
				Description: "Largest k3k " + r.label + " pods by working set memory.",
				Group:       "k3k",
				Unit:        "bytes",
				PromQL:      topk(k3kTopPods, mem),
			},
			Query{
				Name:        "k3k_" + r.name + "_cpu_total",
				Description: "Total CPU used by all k3k " + r.label + " pods, in cores.",
				Group:       "k3k",
				Unit:        "cores",
				PromQL:      `sum (` + cpu + `)`,
			},
			Query{
				Name:        "k3k_" + r.name + "_cpu_max",
				Description: "CPU of the single busiest k3k " + r.label + " pod, in cores.",
				Group:       "k3k",
				Unit:        "cores",
				PromQL:      `max (` + cpu + `)`,
			},
			Query{
				Name:        "k3k_" + r.name + "_workingset_total",
				Description: "Total working set memory across all k3k " + r.label + " pods.",
				Group:       "k3k",
				Unit:        "bytes",
				PromQL:      `sum (` + mem + `)`,
			},
			Query{
				Name:        "k3k_" + r.name + "_workingset_max",
				Description: "Working set memory of the largest k3k " + r.label + " pod.",
				Group:       "k3k",
				Unit:        "bytes",
				PromQL:      `max (` + mem + `)`,
			},
		)
	}

	return out
}

// k3kFocusResourceQueries pin one virtual cluster's resource usage, unbounded.
//
// The role queries above are topk-bounded, so a specific cluster is only visible
// while it is among the busiest. These make the focus cluster's server and
// kubelet observable for the whole run regardless of what its neighbours do,
// which is what makes a single cluster's behaviour followable at scale.
func k3kFocusResourceQueries(namespace string) []Query {
	return []Query{
		{
			Name:        "k3k_focus_server_cpu",
			Description: "CPU in cores for the focus virtual cluster's server pods.",
			Group:       "k3k",
			Unit:        "cores",
			PromQL: `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace="` +
				namespace + `", container=~"` + k3kServerContainers + `"}[5m]))`,
		},
		{
			Name:        "k3k_focus_server_workingset",
			Description: "Working set memory for the focus virtual cluster's server pods.",
			Group:       "k3k",
			Unit:        "bytes",
			PromQL: `sum by (namespace, pod) (container_memory_working_set_bytes{namespace="` +
				namespace + `", container=~"` + k3kServerContainers + `"})`,
		},
		{
			Name:        "k3k_focus_kubelet_cpu",
			Description: "CPU in cores for the focus virtual cluster's shared-mode kubelet pods.",
			Group:       "k3k",
			Unit:        "cores",
			PromQL: `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace="` +
				namespace + `", container=~"` + k3kKubeletContainers + `"}[5m]))`,
		},
		{
			Name:        "k3k_focus_kubelet_workingset",
			Description: "Working set memory for the focus virtual cluster's shared-mode kubelet pods.",
			Group:       "k3k",
			Unit:        "bytes",
			PromQL: `sum by (namespace, pod) (container_memory_working_set_bytes{namespace="` +
				namespace + `", container=~"` + k3kKubeletContainers + `"})`,
		},
		{
			Name:        "k3k_focus_agent_cpu",
			Description: "CPU in cores for the focus virtual cluster's virtual-mode agent pods.",
			Group:       "k3k",
			Unit:        "cores",
			PromQL: `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace="` +
				namespace + `", container=~"` + k3kAgentContainers + `"}[5m]))`,
		},
		{
			Name:        "k3k_focus_agent_workingset",
			Description: "Working set memory for the focus virtual cluster's virtual-mode agent pods.",
			Group:       "k3k",
			Unit:        "bytes",
			PromQL: `sum by (namespace, pod) (container_memory_working_set_bytes{namespace="` +
				namespace + `", container=~"` + k3kAgentContainers + `"})`,
		},
	}
}

func k3kFocusOperationalQueries(namespace string) []Query {
	return []Query{{
		Name: "k3k_focus_pod_restarts", Description: "Container restarts for the focus virtual cluster's pods.",
		Group: "operational", Unit: "count", PromQL: `sum by (namespace, pod) (kube_pod_container_status_restarts_total{namespace="` + namespace + `"})`,
	}}
}
