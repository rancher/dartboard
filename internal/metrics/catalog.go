/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

// Query is one entry in the curated PromQL catalog.
type Query struct {
	Name        string
	Description string
	Group       string // cpu | memory | disk | network | pod
	PromQL      string
	Unit        string
}

// rancherNamespaces matches the namespaces hosting Rancher components and
// fleet so the catalog focuses on the workload we want to size for.
const rancherNamespaces = `cattle-.*|fleet-.*`

// Catalog returns the list of queries we collect for VM/machine sizing
// analysis. Names are file-safe (suitable for use as CSV filenames).
func Catalog() []Query {
	return []Query{
		// CPU
		{
			Name:        "node_cpu_utilization",
			Description: "Per-node CPU utilization ratio (1 - idle).",
			Group:       "cpu",
			Unit:        "ratio",
			PromQL:      `1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))`,
		},
		{
			Name:        "node_load1",
			Description: "Per-node 1-minute load average.",
			Group:       "cpu",
			Unit:        "load",
			PromQL:      `node_load1`,
		},
		{
			Name:        "rancher_pod_cpu",
			Description: "Per-Rancher-pod CPU usage in cores.",
			Group:       "cpu",
			Unit:        "cores",
			PromQL:      `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace=~"` + rancherNamespaces + `", container!="", container!="POD"}[5m]))`,
		},

		// Memory
		{
			Name:        "node_mem_used_bytes",
			Description: "Per-node memory used (MemTotal - MemAvailable).",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      `node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes`,
		},
		{
			Name:        "rancher_pod_workingset",
			Description: "Per-Rancher-pod working set memory.",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      `sum by (namespace, pod) (container_memory_working_set_bytes{namespace=~"` + rancherNamespaces + `", container!="", container!="POD"})`,
		},
		{
			Name:        "rancher_pod_rss",
			Description: "Per-Rancher-pod RSS memory.",
			Group:       "memory",
			Unit:        "bytes",
			PromQL:      `sum by (namespace, pod) (container_memory_rss{namespace=~"` + rancherNamespaces + `", container!="", container!="POD"})`,
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
		{
			Name:        "node_net_errors",
			Description: "Per-node network errors+drops (rx+tx).",
			Group:       "network",
			Unit:        "events/sec",
			PromQL:      `sum by (instance, device) (rate(node_network_receive_errs_total[5m]) + rate(node_network_receive_drop_total[5m]) + rate(node_network_transmit_errs_total[5m]) + rate(node_network_transmit_drop_total[5m]))`,
		},

		// Pod resource headroom (kube-state-metrics)
		{
			Name:        "pod_resource_requests",
			Description: "Per-Rancher-pod resource requests (CPU + memory).",
			Group:       "pod",
			Unit:        "mixed",
			PromQL:      `kube_pod_container_resource_requests{namespace=~"` + rancherNamespaces + `"}`,
		},
		{
			Name:        "pod_resource_limits",
			Description: "Per-Rancher-pod resource limits (CPU + memory).",
			Group:       "pod",
			Unit:        "mixed",
			PromQL:      `kube_pod_container_resource_limits{namespace=~"` + rancherNamespaces + `"}`,
		},
	}
}
