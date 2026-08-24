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
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Collect runs every query in Catalog() against promURL over [start, end] at
// step resolution and writes CSVs + summary.json + manifest.json under outDir.
// dartFile and workspace are recorded in the manifest for traceability.
func Collect(promURL, dartFile, workspace string, start, end time.Time, step time.Duration, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", outDir, err)
	}

	if dartFile != "" {
		body, err := os.ReadFile(dartFile)
		if err != nil {
			return fmt.Errorf("read dart %s: %w", dartFile, err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "dart.yaml"), body, 0o644); err != nil {
			return fmt.Errorf("snapshot dart: %w", err)
		}
	}

	client := NewClient(promURL)
	queries := Catalog()
	all := make(map[string]QuerySummary, len(queries))

	for _, q := range queries {
		logrus.Infof("collecting %s/%s", q.Group, q.Name)
		series, err := client.QueryRange(q.PromQL, start, end, step)
		if err != nil {
			logrus.Warnf("query %s failed: %v", q.Name, err)
			continue
		}
		summary, err := WriteSeriesCSV(outDir, q, series)
		if err != nil {
			return fmt.Errorf("write %s csv: %w", q.Name, err)
		}
		all[q.Name] = summary
	}

	if err := WriteSummary(outDir, all); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	if err := WriteManifest(outDir, Manifest{
		GeneratedAt: time.Now().UTC(),
		PromURL:     promURL,
		Start:       start.UTC(),
		End:         end.UTC(),
		Step:        step,
		DartFile:    dartFile,
		Workspace:   workspace,
		Queries:     len(queries),
	}); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	logrus.Infof("metrics collected to %s", outDir)
	return nil
}
