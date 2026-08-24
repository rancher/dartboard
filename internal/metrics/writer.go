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
	"time"
)

// SeriesStats holds the aggregates we emit per series in summary.json.
type SeriesStats struct {
	Count int     `json:"count"`
	Min   float64 `json:"min"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
}

// QuerySummary maps a stable label string ("k1=v1,k2=v2") to its stats.
type QuerySummary map[string]SeriesStats

// Manifest captures the run metadata written to manifest.json.
type Manifest struct {
	GeneratedAt time.Time     `json:"generated_at"`
	PromURL     string        `json:"prom_url"`
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	Step        time.Duration `json:"step_seconds"`
	DartFile    string        `json:"dart_file,omitempty"`
	Workspace   string        `json:"tofu_workspace,omitempty"`
	Queries     int           `json:"queries"`
}

// WriteSeriesCSV writes one CSV per query: <outDir>/<group>/<name>.csv with
// columns timestamp,<sorted-label-keys>,value. Returns per-series stats.
func WriteSeriesCSV(outDir string, q Query, series []Series) (QuerySummary, error) {
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
		return "(no_labels)"
	}
	return out
}

func computeStats(samples []Sample) SeriesStats {
	if len(samples) == 0 {
		return SeriesStats{}
	}
	values := make([]float64, 0, len(samples))
	sum := 0.0
	for _, s := range samples {
		if math.IsNaN(s.Value) || math.IsInf(s.Value, 0) {
			continue
		}
		values = append(values, s.Value)
		sum += s.Value
	}
	if len(values) == 0 {
		return SeriesStats{}
	}
	sort.Float64s(values)
	return SeriesStats{
		Count: len(values),
		Min:   values[0],
		Max:   values[len(values)-1],
		Mean:  sum / float64(len(values)),
		P50:   percentile(values, 0.50),
		P95:   percentile(values, 0.95),
		P99:   percentile(values, 0.99),
	}
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
