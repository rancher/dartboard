/*
Copyright © 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Collect runs every query selected by opts against promURL over [start, end]
// at step resolution and writes CSVs + summary.json + manifest.json under
// outDir. dartFile and workspace are recorded in the manifest for traceability
// and may be empty when the run targeted a kubeconfig directly.
func Collect(ctx context.Context, promURL, dartFile, workspace string, start, end time.Time, step time.Duration, outDir string, opts CatalogOptions) error {
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

	queries := Catalog(opts)
	if err := WriteQueryCatalog(outDir, queries); err != nil {
		return fmt.Errorf("write query catalog: %w", err)
	}

	all := make(map[string]QuerySummary, len(queries))

	var collectionErr error

	for _, q := range queries {
		logrus.Infof("collecting %s/%s", q.Group, q.Name)

		series, err := client.QueryRangeContext(ctx, q.PromQL, start, end, step)
		if err != nil {
			if opts.preset() == PresetK3K && isRequiredK3KQuery(q.Name) {
				collectionErr = fmt.Errorf("required k3k query %s failed: %w", q.Name, err)
				break
			}

			logrus.Warnf("query %s failed: %v", q.Name, err)

			continue
		}

		if opts.preset() == PresetK3K && isRequiredK3KQuery(q.Name) && len(series) == 0 {
			collectionErr = fmt.Errorf("required k3k query %s returned no series for namespace scope %q", q.Name, opts.namespaces())
			break
		}

		summary, err := WriteSeriesCSV(outDir, q, series)
		if err != nil {
			return fmt.Errorf("write %s csv: %w", q.Name, err)
		}

		if len(summary) > 0 {
			all[q.Name] = summary
		}
	}

	if err := WriteSummary(outDir, all); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	if err := WriteManifest(outDir, Manifest{
		GeneratedAt:    time.Now().UTC(),
		PromURL:        promURL,
		Start:          start.UTC(),
		End:            end.UTC(),
		Step:           step,
		DartFile:       dartFile,
		Workspace:      workspace,
		Preset:         string(opts.preset()),
		Namespaces:     opts.namespaces(),
		Operational:    opts.Operational,
		FocusNamespace: opts.FocusNamespace,
		Queries:        len(queries),
	}); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	if collectionErr != nil {
		return collectionErr
	}

	logrus.Infof("metrics collected to %s", outDir)

	return nil
}

func isRequiredK3KQuery(name string) bool {
	switch name {
	case "k3k_server_cpu_per_active_pod_millicores",
		"k3k_server_memory_per_active_pod_mb":
		return true
	default:
		return false
	}
}
