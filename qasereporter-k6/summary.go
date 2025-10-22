package main

import (
	"context"
	"encoding/json"
	"os"

	v1 "github.com/qase-tms/qase-go/qase-api-client"
	"github.com/rancher/dartboard/internal/qase"
	"github.com/sirupsen/logrus"
)

const (
	k6SummaryJsonFileEnvVar = "K6_SUMMARY_JSON_FILE"
	k6SummaryHtmlFileEnvVar = "K6_SUMMARY_HTML_FILE"
)

var (
	k6SummaryJsonFile = os.Getenv(k6SummaryJsonFileEnvVar)
	k6SummaryHtmlFile = os.Getenv(k6SummaryHtmlFileEnvVar)
)

// K6Summary represents the structure of the k6 summary JSON output.
type K6Summary struct {
	RootGroup K6Group                    `json:"root_group"`
	Metrics   map[string]K6SummaryMetric `json:"metrics"`
}

// K6Group represents a group of checks in the k6 summary.
type K6Group struct {
	Groups []K6Group `json:"groups"`
	Checks []K6Check `json:"checks"`
}

// K6SummaryMetric represents a metric in the k6 summary, which may contain thresholds.
type K6SummaryMetric struct {
	Thresholds map[string]K6SummaryThreshold `json:"thresholds,omitempty"`
}

// K6SummaryThreshold represents a threshold with its pass/fail status.
type K6SummaryThreshold struct {
	OK bool `json:"ok"`
}

func reportSummary() {
	logrus.Info("Running k6 QASE reporter")

	if k6SummaryJsonFile == "" {
		logrus.Fatalf("Missing required environment variable: %s", k6SummaryJsonFileEnvVar)
	}

	// Read and parse the k6 summary JSON
	k6SummaryJsonData, err := os.ReadFile(k6SummaryJsonFile)
	if err != nil {
		logrus.Fatalf("Failed to read k6 summary JSON file %s: %v", k6SummaryJsonFile, err)
	}

	logrus.Info("Parsing k6 summary JSON for results.")
	checks, thresholds, overallPass := parseK6SummaryJson(k6SummaryJsonData)

	// Build the comment for Qase
	comment := buildQaseComment(thresholds, checks, "")

	// Report to Qase
	status := qase.StatusPassed
	if !overallPass {
		status = qase.StatusFailed
	}

	// Prepare attachments if the HTML report exists
	var attachments []string
	if k6SummaryHtmlFile != "" {
		if _, err := os.Stat(k6SummaryHtmlFile); err == nil {
			logrus.Infof("Found HTML report at %s, preparing for upload.", k6SummaryHtmlFile)
			attachments = append(attachments, k6SummaryHtmlFile)
		} else {
			logrus.Warnf("HTML report file specified but not found at %s.", k6SummaryHtmlFile)
		}
	}

	logrus.Infof("Reporting to Qase: Project=%s, Run=%d, Case=%d, Status=%s", projectID, runID, testCaseID, status)

	// Upload attachments and get their hashes
	var attachmentHashes []string
	if len(attachments) > 0 {
		var files []*os.File
		for _, filePath := range attachments {
			file, err := os.Open(filePath)
			if err != nil {
				logrus.Fatalf("Failed to open attachment file %s: %v", filePath, err)
			}
			defer func(f *os.File, path string) {
				if err := f.Close(); err != nil {
					logrus.Warnf("Failed to close attachment file %s: %v", path, err)
				}
			}(file, filePath)
			files = append(files, file)
			hashes, err := qaseClient.UploadAttachments(context.Background(), files)
			if err != nil {
				logrus.Fatalf("Failed to upload attachments to Qase: %v", err)
			}
			attachmentHashes = append(attachmentHashes, hashes...)
		}
	}

	resultBody := v1.NewResultCreate(status)
	resultBody.SetCaseId(testCaseID)
	resultBody.SetComment(comment)
	resultBody.SetAttachments(attachmentHashes)

	err = qaseClient.CreateTestResultV1(context.Background(), projectID, runID, *resultBody)
	if err != nil {
		logrus.Fatalf("Failed to create Qase result: %v", err)
	}

	logrus.Info("Successfully reported k6 results to Qase.")
}

// parseK6SummaryJson processes the k6 summary JSON to extract checks, thresholds, and determine the overall status.
func parseK6SummaryJson(jsonData []byte) ([]K6Check, []K6Threshold, bool) {
	var summary K6Summary
	if err := json.Unmarshal(jsonData, &summary); err != nil {
		logrus.Fatalf("Failed to unmarshal k6 summary JSON: %v", err)
	}

	var checks []K6Check
	var thresholds []K6Threshold
	overallPass := true

	// Recursively extract all checks from the root group and its subgroups.
	var collectChecks func(g K6Group)
	collectChecks = func(g K6Group) {
		for _, ch := range g.Checks {
			checks = append(checks, ch)
			if ch.Fails > 0 {
				overallPass = false
			}
		}
		for _, subGroup := range g.Groups {
			collectChecks(subGroup)
		}
	}
	collectChecks(summary.RootGroup)

	// Extract all thresholds from the metrics map.
	for metricName, metric := range summary.Metrics {
		for thresholdName, threshold := range metric.Thresholds {
			thresholds = append(thresholds, K6Threshold{Name: thresholdName, Metric: metricName, Pass: threshold.OK})
			if !threshold.OK {
				overallPass = false
			}
		}
	}

	return checks, thresholds, overallPass
}
