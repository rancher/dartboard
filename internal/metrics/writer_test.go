/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestComputeStatsTrend(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)

	stats := computeStats([]Sample{
		{Timestamp: start.Add(2 * time.Hour), Value: 30},
		{Timestamp: start, Value: 10},
		{Timestamp: start.Add(time.Hour), Value: 20},
	})
	if stats.First != 10 || stats.Last != 30 || stats.Change != 20 {
		t.Fatalf("unexpected trend endpoints: %+v", stats)
	}

	if stats.ChangePercent == nil || *stats.ChangePercent != 200 {
		t.Fatalf("unexpected change percent: %+v", stats.ChangePercent)
	}

	if stats.SlopePerHour == nil || *stats.SlopePerHour != 10 {
		t.Fatalf("unexpected slope: %+v", stats.SlopePerHour)
	}
}

func TestWriteSeriesCSVSkipsEmptyResults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	stale := filepath.Join(dir, "k3k", "empty.csv")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(stale, []byte("timestamp,value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteSeriesCSV(dir, Query{Name: "empty", Group: "k3k"}, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("empty query created a CSV: %v", err)
	}
}

func TestLabelKeyNamesUnlabelledSeriesAggregate(t *testing.T) {
	t.Parallel()

	if got := labelKey(nil); got != "aggregate" {
		t.Fatalf("labelKey(nil) = %q, want aggregate", got)
	}
}

func TestWriteQueryCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	queries := []Query{{
		Name: "pod_cpu", Description: "CPU used by selected pods.", Group: "cpu",
		Unit: "cores", PromQL: `sum(rate(container_cpu_usage_seconds_total[5m]))`,
	}}
	if err := WriteQueryCatalog(dir, queries); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "queries.json"))
	if err != nil {
		t.Fatal(err)
	}

	var decoded []Query
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded) != 1 || decoded[0] != queries[0] {
		t.Fatalf("decoded queries = %#v, want %#v", decoded, queries)
	}

	markdown, err := os.ReadFile(filepath.Join(dir, "queries.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"## pod_cpu", "CPU used by selected pods.", "**Unit:** `cores`", "```promql", queries[0].PromQL} {
		if !strings.Contains(string(markdown), expected) {
			t.Errorf("queries.md does not contain %q", expected)
		}
	}
}
