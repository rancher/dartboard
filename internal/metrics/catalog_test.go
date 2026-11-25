/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"regexp"
	"strings"
	"testing"
)

func TestK3KPerActivePodQueries(t *testing.T) {
	t.Parallel()

	queries := k3kRoleQueries(K3KNamespaces)

	byName := make(map[string]Query, len(queries))
	for _, query := range queries {
		byName[query.Name] = query
	}

	tests := []struct {
		name     string
		unit     string
		contains []string
	}{
		{
			name:     "k3k_server_cpu_per_active_pod_millicores",
			unit:     "millicores",
			contains: []string{"1000 * sum", "count", "container_memory_working_set_bytes", k3kServerContainers},
		},
		{
			name:     "k3k_server_memory_per_active_pod_mb",
			unit:     "MB",
			contains: []string{"container_memory_working_set_bytes", "count", "/ 1000000", k3kServerContainers},
		},
		{
			name:     "k3k_kubelet_cpu_per_active_pod_millicores",
			unit:     "millicores",
			contains: []string{"1000 * sum", "count", "container_memory_working_set_bytes", k3kKubeletContainers},
		},
		{
			name:     "k3k_kubelet_memory_per_active_pod_mb",
			unit:     "MB",
			contains: []string{"container_memory_working_set_bytes", "count", "/ 1000000", k3kKubeletContainers},
		},
		{
			name:     "k3k_agent_cpu_per_active_pod_millicores",
			unit:     "millicores",
			contains: []string{"1000 * sum", "count", "container_memory_working_set_bytes", `container=~"k3k-agent"`},
		},
		{
			name:     "k3k_agent_memory_per_active_pod_mb",
			unit:     "MB",
			contains: []string{"container_memory_working_set_bytes", "count", "/ 1000000", `container=~"k3k-agent"`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, ok := byName[test.name]
			if !ok {
				t.Fatalf("query %q not found", test.name)
			}

			if query.Unit != test.unit {
				t.Fatalf("unit = %q, want %q", query.Unit, test.unit)
			}

			for _, fragment := range test.contains {
				if !strings.Contains(query.PromQL, fragment) {
					t.Errorf("PromQL %q does not contain %q", query.PromQL, fragment)
				}
			}
		})
	}
}

func TestK3KRunNamespaces(t *testing.T) {
	t.Parallel()

	selector := strings.ReplaceAll(k3kRunNamespaces("bulk.test", 98, 6), `\\`, `\`)
	if strings.Contains(selector, "?:") {
		t.Fatalf("selector contains unsupported RE2 non-capturing group: %s", selector)
	}

	pattern := regexp.MustCompile("^(?:" + selector + ")$")
	for _, namespace := range []string{"k3k-system", "k3k-bulk.test-098", "k3k-bulk.test-099", "k3k-bulk.test-100", "k3k-bulk.test-103"} {
		if !pattern.MatchString(namespace) {
			t.Errorf("selector does not match %q", namespace)
		}
	}

	for _, namespace := range []string{"k3k-bulkXtest-098", "k3k-bulk.test-097", "k3k-bulk.test-104", "k3k-other-100"} {
		if pattern.MatchString(namespace) {
			t.Errorf("selector unexpectedly matches %q", namespace)
		}
	}
}

func TestK3KCatalogCapsRawPodResources(t *testing.T) {
	t.Parallel()

	queries := Catalog(CatalogOptions{Preset: PresetK3K})
	resourceQueries := 0

	for _, query := range queries {
		if strings.HasPrefix(query.Name, "pod_") && (strings.HasSuffix(query.Name, "_requests") || strings.HasSuffix(query.Name, "_limits")) {
			resourceQueries++

			if !strings.HasPrefix(query.PromQL, "topk(10, ") {
				t.Errorf("%s is not capped: %s", query.Name, query.PromQL)
			}
		}
	}

	if resourceQueries != 4 {
		t.Fatalf("found %d resource request/limit queries, want 4", resourceQueries)
	}
}

func TestCatalogOptionsValidateK3KRun(t *testing.T) {
	t.Parallel()

	valid := CatalogOptions{Preset: PresetK3K, K3KRunPrefix: "bulk", K3KStartIndex: 1, K3KCount: 10}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid selector rejected: %v", err)
	}

	invalid := valid

	invalid.Namespaces = K3KNamespaces
	if err := invalid.Validate(); err == nil {
		t.Fatal("namespace override and run selector should conflict")
	}
}

func TestK3KLargeRunNamespaces(t *testing.T) {
	t.Parallel()

	selector := strings.ReplaceAll(k3kRunNamespaces("scale", 1, 5000), `\\`, `\`)
	if len(selector) > 1000 {
		t.Fatalf("selector is not compact: %d bytes", len(selector))
	}

	pattern := regexp.MustCompile("^(?:" + selector + ")$")
	for _, namespace := range []string{"k3k-scale-001", "k3k-scale-999", "k3k-scale-1000", "k3k-scale-5000", "k3k-system"} {
		if !pattern.MatchString(namespace) {
			t.Errorf("selector does not match %q", namespace)
		}
	}

	for _, namespace := range []string{"k3k-scale-000", "k3k-scale-5001", "k3k-other-001"} {
		if pattern.MatchString(namespace) {
			t.Errorf("selector unexpectedly matches %q", namespace)
		}
	}
}

func TestK3KSharedSingleClusterCatalogScope(t *testing.T) {
	t.Parallel()

	opts := CatalogOptions{
		Preset:        PresetK3K,
		K3KRunPrefix:  "k3k-shared",
		K3KStartIndex: 1,
		K3KCount:      1,
	}

	scope := opts.namespaces()
	if scope != `(k3k-system|k3k-k3k-shared-(001))` {
		t.Fatalf("scope = %q", scope)
	}

	if strings.Contains(scope, "?:") {
		t.Fatalf("scope contains unsupported non-capturing group: %s", scope)
	}

	for _, query := range Catalog(opts) {
		if strings.Contains(query.PromQL, "namespace=~") && !strings.Contains(query.PromQL, `namespace=~"`+scope+`"`) {
			t.Errorf("query %s does not use exact run scope: %s", query.Name, query.PromQL)
		}
	}
}

func TestK3KRunNamespacesSmallAndWidthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		matches []string
		rejects []string
		start   int
		count   int
	}{
		{[]string{"k3k-run-001"}, []string{"k3k-run-000", "k3k-run-002"}, 1, 1},
		{[]string{"k3k-run-001", "k3k-run-010"}, []string{"k3k-run-011"}, 1, 10},
		{[]string{"k3k-run-098", "k3k-run-099"}, []string{"k3k-run-100"}, 98, 2},
		{[]string{"k3k-run-999", "k3k-run-1000"}, []string{"k3k-run-998", "k3k-run-1001"}, 999, 2},
	}
	for _, test := range tests {
		selector := strings.ReplaceAll(k3kRunNamespaces("run", test.start, test.count), `\\`, `\`)

		pattern := regexp.MustCompile("^(?:" + selector + ")$")
		for _, namespace := range test.matches {
			if !pattern.MatchString(namespace) {
				t.Errorf("range %d+%d does not match %q (selector %s)", test.start, test.count, namespace, selector)
			}
		}

		for _, namespace := range test.rejects {
			if pattern.MatchString(namespace) {
				t.Errorf("range %d+%d unexpectedly matches %q (selector %s)", test.start, test.count, namespace, selector)
			}
		}
	}
}

func TestK3KDefaultCatalogIsResourceOnly(t *testing.T) {
	t.Parallel()

	for _, query := range Catalog(CatalogOptions{Preset: PresetK3K}) {
		if query.Group == "operational" {
			t.Errorf("default catalog contains operational query %s", query.Name)
		}

		if strings.Contains(query.Name, "coredns") || strings.Contains(query.Name, "reconcile") || strings.Contains(query.Name, "workqueue") || strings.Contains(query.Name, "restart") || strings.Contains(query.Name, "phase") {
			t.Errorf("default catalog contains expected-empty/non-resource query %s", query.Name)
		}
	}
}

func TestK3KOperationalCatalogIsOptIn(t *testing.T) {
	t.Parallel()

	queries := Catalog(CatalogOptions{Preset: PresetK3K, Operational: true})

	found := map[string]bool{}
	for _, query := range queries {
		found[query.Name] = true
	}

	for _, name := range []string{"node_load1", "node_net_errors", "k3k_vcluster_count", "k3k_pod_restarts", "k3k_controller_reconcile_rate"} {
		if !found[name] {
			t.Errorf("operational catalog missing %s", name)
		}
	}
}

func TestOnlyServerRoleIsRequired(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"k3k_server_cpu_per_active_pod_millicores",
		"k3k_server_memory_per_active_pod_mb",
	} {
		if !isRequiredK3KQuery(name) {
			t.Errorf("server query %s should be required", name)
		}
	}

	for _, name := range []string{
		"k3k_kubelet_cpu_per_active_pod_millicores",
		"k3k_kubelet_memory_per_active_pod_mb",
		"k3k_agent_cpu_per_active_pod_millicores",
		"k3k_agent_memory_per_active_pod_mb",
		"k3k_controller_cpu_per_active_pod_millicores",
	} {
		if isRequiredK3KQuery(name) {
			t.Errorf("optional role query %s should not be required", name)
		}
	}
}

func TestVirtualAgentContainerSelector(t *testing.T) {
	t.Parallel()

	if !regexp.MustCompile(`^k3k-agent$`).MatchString("k3k-agent") {
		t.Fatal("virtual agent container selector does not match k3k-agent")
	}

	if !regexp.MustCompile(`^k3k-k3k-shared-001-agent-[^-]+-[^-]+$`).MatchString("k3k-k3k-shared-001-agent-766549fb6-lxrrj") {
		t.Fatal("expected virtual agent pod naming convention did not match")
	}
}
