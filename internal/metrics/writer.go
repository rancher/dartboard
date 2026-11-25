/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SeriesStats holds the aggregates we emit per series in summary.json.
type SeriesStats struct {
	ChangePercent *float64 `json:"change_percent,omitempty"`
	SlopePerHour  *float64 `json:"slope_per_hour,omitempty"`
	Count         int      `json:"count"`
	Min           float64  `json:"min"`
	P50           float64  `json:"p50"`
	P95           float64  `json:"p95"`
	P99           float64  `json:"p99"`
	Max           float64  `json:"max"`
	Mean          float64  `json:"mean"`
	First         float64  `json:"first"`
	Last          float64  `json:"last"`
	Change        float64  `json:"change"`
}

// QuerySummary maps a stable label string ("k1=v1,k2=v2") to its stats.
type QuerySummary map[string]SeriesStats

// Manifest captures the run metadata written to manifest.json.
type Manifest struct {
	GeneratedAt    time.Time     `json:"generated_at"`
	Start          time.Time     `json:"start"`
	End            time.Time     `json:"end"`
	Namespaces     string        `json:"namespaces,omitempty"`
	DartFile       string        `json:"dart_file,omitempty"`
	Workspace      string        `json:"tofu_workspace,omitempty"`
	Cluster        string        `json:"cluster,omitempty"`
	Preset         string        `json:"preset,omitempty"`
	PromURL        string        `json:"prom_url"`
	FocusNamespace string        `json:"focus_namespace,omitempty"`
	Step           time.Duration `json:"step_seconds"`
	Queries        int           `json:"queries"`
	Operational    bool          `json:"operational,omitempty"`
}

// WriteSeriesCSV writes one CSV per query: <outDir>/<group>/<name>.csv with
// columns timestamp,<sorted-label-keys>,value. Returns per-series stats.
func WriteSeriesCSV(outDir string, q Query, series []Series) (QuerySummary, error) {
	if len(series) == 0 {
		if err := os.Remove(filepath.Join(outDir, q.Group, q.Name+".csv")); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove stale empty CSV for %s: %w", q.Name, err)
		}

		return QuerySummary{}, nil
	}

	groupDir := filepath.Join(outDir, q.Group)
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", groupDir, err)
	}

	csvPath := filepath.Join(groupDir, q.Name+".csv")

	labelKeys := collectLabelKeys(series)

	f, err := os.Create(csvPath)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", csvPath, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)

	header := append([]string{"timestamp"}, labelKeys...)

	header = append(header, "value")
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	summary := QuerySummary{}

	for _, s := range series {
		labelVals := make([]string, len(labelKeys))
		for i, k := range labelKeys {
			labelVals[i] = s.Labels[k]
		}

		for _, sample := range s.Samples {
			row := append([]string{sample.Timestamp.UTC().Format(time.RFC3339)}, labelVals...)

			row = append(row, strconv.FormatFloat(sample.Value, 'f', -1, 64))
			if err := w.Write(row); err != nil {
				return nil, fmt.Errorf("write row: %w", err)
			}
		}

		summary[labelKey(s.Labels)] = computeStats(s.Samples)
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}

	return summary, nil
}

// WriteSummary writes the aggregate summary across all queries to summary.json.
func WriteSummary(outDir string, all map[string]QuerySummary) error {
	path := filepath.Join(outDir, "summary.json")

	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, body, 0o644)
}

// WriteManifest writes the run-level manifest.
func WriteManifest(outDir string, m Manifest) error {
	path := filepath.Join(outDir, "manifest.json")

	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, body, 0o644)
}

// WriteQueryCatalog records the exact descriptions, units, and PromQL used by
// this collection. It includes empty-result queries so missing data remains
// distinguishable from a query that was never selected.
func WriteQueryCatalog(outDir string, queries []Query) error {
	body, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(outDir, "queries.json"), body, 0o644); err != nil {
		return err
	}

	var markdown strings.Builder
	markdown.WriteString("# Prometheus Queries\n\n")

	for _, query := range queries {
		fmt.Fprintf(&markdown, "## %s\n\n%s\n\n- **Group:** `%s`\n- **Unit:** `%s`\n\n```promql\n%s\n```\n\n",
			query.Name, query.Description, query.Group, query.Unit, query.PromQL)
	}

	return os.WriteFile(filepath.Join(outDir, "queries.md"), []byte(markdown.String()), 0o644)
}

func collectLabelKeys(series []Series) []string {
	seen := map[string]struct{}{}

	for _, s := range series {
		for k := range s.Labels {
			seen[k] = struct{}{}
		}
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func labelKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	out := ""

	for i, k := range keys {
		if i > 0 {
			out += ","
		}

		out += k + "=" + labels[k]
	}

	if out == "" {
		return "aggregate"
	}

	return out
}

func computeStats(samples []Sample) SeriesStats {
	if len(samples) == 0 {
		return SeriesStats{}
	}

	finite := make([]Sample, 0, len(samples))

	for _, sample := range samples {
		if !math.IsNaN(sample.Value) && !math.IsInf(sample.Value, 0) {
			finite = append(finite, sample)
		}
	}

	if len(finite) == 0 {
		return SeriesStats{}
	}

	sort.SliceStable(finite, func(i, j int) bool { return finite[i].Timestamp.Before(finite[j].Timestamp) })

	values := make([]float64, 0, len(finite))
	sum := 0.0

	for _, s := range finite {
		values = append(values, s.Value)
		sum += s.Value
	}

	first := finite[0].Value
	last := finite[len(finite)-1].Value
	change := last - first

	var changePercent *float64

	if first != 0 {
		value := change / math.Abs(first) * 100
		changePercent = &value
	}

	slope := linearSlopePerHour(finite)

	sort.Float64s(values)

	return SeriesStats{
		Count: len(values), Min: values[0], Max: values[len(values)-1], Mean: sum / float64(len(values)),
		P50: percentile(values, 0.50), P95: percentile(values, 0.95), P99: percentile(values, 0.99),
		First: first, Last: last, Change: change, ChangePercent: changePercent, SlopePerHour: slope,
	}
}

func linearSlopePerHour(samples []Sample) *float64 {
	if len(samples) < 2 {
		return nil
	}

	origin := samples[0].Timestamp

	var sumX, sumY float64

	for _, sample := range samples {
		sumX += sample.Timestamp.Sub(origin).Hours()
		sumY += sample.Value
	}

	meanX := sumX / float64(len(samples))
	meanY := sumY / float64(len(samples))

	var numerator, denominator float64

	for _, sample := range samples {
		x := sample.Timestamp.Sub(origin).Hours()
		numerator += (x - meanX) * (sample.Value - meanY)
		denominator += (x - meanX) * (x - meanX)
	}

	if denominator == 0 {
		return nil
	}

	value := numerator / denominator

	return &value
}

// percentile expects values sorted ascending; uses nearest-rank.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}

	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}
