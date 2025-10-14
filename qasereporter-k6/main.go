package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rancher/tests/actions/qase"
	"github.com/sirupsen/logrus"
	upstream "go.qase.io/client"
)

const (
	k6OutputFileEnvVar  = "K6_OUTPUT_FILE"
	k6SummaryFileEnvVar = "K6_SUMMARY_FILE"
)

// see https://grafana.com/docs/k6/latest/results-output/real-time/json/#json-format for specifics on the JSON formatting to match

// K6Line represents a generic line from the k6 JSON output.
type K6Line struct {
	Type   string `json:"type"`   // "Metric" or "Point", Metric means the line is declaring a metric, and Point is an actual data point (sample) for a metric.
	Metric string `json:"metric"` // Name of the metric
}

type K6Data[T K6MetricData | K6PointData] struct {
	Value T
}

// K6Metric represents a 'Metric' line from the k6 JSON output.
type K6Metric struct {
	K6Line
	Data K6Data[K6MetricData] `json:"data"` // Data associated with the Metric, contains lots of stuff
}

// K6Point represents a 'Point' line from the k6 JSON output.
type K6Point struct {
	K6Line
	Data K6Data[K6PointData] `json:"data"` // Data associated with the Point, contains lots of stuff
}

// K6MetricData represents the 'data' field for a 'Metric' type metric.
// We define it for completeness, though we don't currently use it.
type K6MetricData struct {
	Type       string   `json:"type"`       // The metric type (“gauge”, “rate”, “counter” or “trend”)
	Contains   string   `json:"contains"`   // Information on the type of data collected (can e.g. be “time” for timing metrics)
	Tainted    *bool    `json:"tainted"`    // Has this metric caused any threshold(s) to fail? Use a pointer to handle 'null'
	Thresholds []string `json:"thresholds"` // Any and all Thresholds attached to this metric
	SubMetrics []string `json:"submetrics"` // Any and all metrics derived from this metric as a result of adding a threshold using tags
}

// K6PointData represents the 'data' field for a 'Point' type metric.
type K6PointData struct {
	Time   time.Time         `json:"time"`   // Timestamp when the sample was collected
	Value  float64           `json:"value"`  // The actual data sample; time values are in milliseconds
	Tags   map[string]string `json:"tags"`   // Map with tagname-tagvalue pairs that can be used when filtering results data
	Passes int64             `json:"passes"` // Specific to 'checks' metric
	Fails  int64             `json:"fails"`  // Specific to 'checks' metric
}

// K6Threshold represents a threshold metric with its pass/fail status
// derived from `K6MetricData.Tainted`.
type K6Threshold struct {
	Name   string
	Metric string
	Pass   bool
}

// K6Check represents a check metric with its pass/fail counts.
type K6Check struct {
	Name   string
	Passes int64
	Fails  int64
}

func main() {
	logrus.Info("Running k6 QASE reporter")

	projectID := os.Getenv(qase.ProjectIDEnvVar)
	runIDStr := os.Getenv(qase.TestRunEnvVar)
	k6OutputFile := os.Getenv(k6OutputFileEnvVar)
	k6SummaryFile := os.Getenv(k6SummaryFileEnvVar)

	granularParsing := flag.Bool("granular", false, "Enable granular parsing of all Metric and Point lines from k6 JSON output.")
	flag.Parse()

	if projectID == "" || runIDStr == "" || k6OutputFile == "" {
		logrus.Fatalf("Missing required environment variables: %s, %s, %s, %s",
			qase.ProjectIDEnvVar, qase.TestRunEnvVar, k6OutputFileEnvVar)
	}

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		logrus.Fatalf("Invalid QASE_RUN_ID: %v", err)
	}

	testCaseID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		logrus.Fatalf("Invalid QASE_TEST_CASE_ID: %v", err)
	}

	// Read and parse the k6 JSON output
	k6JsonData, err := os.ReadFile(k6OutputFile)
	if err != nil {
		logrus.Fatalf("Failed to read k6 output file %s: %v", k6OutputFile, err)
	}

	var thresholds []K6Threshold
	var checks []K6Check
	var overallPass bool
	var summary string

	if *granularParsing {
		logrus.Info("Performing granular parsing of k6 JSON output.")
		checks, thresholds, overallPass = granularParseK6JsonOutput(k6JsonData)
	} else {
		// Default behavior: Use the summary for overall status and parse Point data for details.
		logrus.Info("Parsing k6 summary and Point data.")
		var parseErr error
		checks, thresholds, parseErr = parseK6JsonOutput(k6JsonData)
		if parseErr != nil {
			logrus.Warnf("Error parsing k6 JSON Point data: %v", parseErr)
		}

		// Read the human-readable summary to determine overall status
		summaryBytes, readErr := os.ReadFile(k6SummaryFile)
		if readErr != nil {
			logrus.Fatalf("Could not read k6 summary file %s: %v. Cannot determine test status.", k6SummaryFile, readErr)
		}
		// A failed threshold will have an '✗' in the summary.
		overallPass = !strings.Contains(string(summaryBytes), "✗")
		summary = string(summaryBytes)
	}

	// Build the comment for Qase
	comment := buildQaseComment(thresholds, checks, summary)

	// Report to Qase
	status := "passed"
	if !overallPass {
		status = "failed"
	}

	logrus.Infof("Reporting to Qase: Project=%s, Run=%d, Case=%d, Status=%s", projectID, runID, testCaseID, status)
	qaseService := qase.SetupQaseClient()
	resultBody := upstream.ResultCreate{
		CaseId:  testCaseID,
		Status:  status,
		Comment: comment,
	}

	_, _, err = qaseService.Client.ResultsApi.CreateResult(context.TODO(), resultBody, projectID, runID)
	if err != nil {
		logrus.Fatalf("Failed to create Qase result: %v", err)
	}

	logrus.Info("Successfully reported k6 results to Qase.")
}

// parseK6JsonOutput processes the raw JSON output from k6 to extract structured
// information about checks and thresholds.
func parseK6JsonOutput(jsonData []byte) ([]K6Check, []K6Threshold, error) {
	var checks []K6Check
	var thresholds []K6Threshold

	lines := strings.SplitSeq(string(jsonData), "\n")
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var k6Line K6Line
		if err := json.Unmarshal([]byte(line), &k6Line); err != nil {
			logrus.Warnf("Failed to unmarshal k6 line: %v", err)
			continue
		}

		// We only care about 'Point' types which contain the actual data samples.
		if k6Line.Type != "Point" {
			continue
		}

		var point K6Point
		if err := json.Unmarshal([]byte(line), &point); err != nil {
			logrus.Warnf("Failed to unmarshal k6 Point data: %v", err)
			continue
		}

		switch k6Line.Metric {
		case "checks":
			check := K6Check{
				Name:   point.Data.Value.Tags["check"],
				Passes: point.Data.Value.Passes,
				Fails:  point.Data.Value.Fails,
			}
			checks = append(checks, check)
		case "thresholds":
			threshold := K6Threshold{
				Name:   point.Data.Value.Tags["threshold"],
				Metric: point.Data.Value.Tags["metric"],
				Pass:   point.Data.Value.Value == 1.0, //TODO: Replace this dummy with actual check vs the threshold
			}
			thresholds = append(thresholds, threshold)
		}
	}
	return checks, thresholds, nil
}

// granularParseK6JsonOutput processes the raw JSON output from k6 by inspecting every
// Metric and Point line to determine which Thresholds and Checks passed or failed.
func granularParseK6JsonOutput(jsonData []byte) ([]K6Check, []K6Threshold, bool) {
	var checks []K6Check
	var thresholds []K6Threshold
	overallPass := true

	// The k6 JSON output is a stream of JSON objects, one per line.
	lines := strings.SplitSeq(string(jsonData), "\n")
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// First, unmarshal into a generic map to check the 'type' field.
		var genericLine map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &genericLine); err != nil {
			logrus.Warnf("Failed to unmarshal k6 metric line: %v", err)
			continue
		}

		var lineType string
		if err := json.Unmarshal(genericLine["type"], &lineType); err != nil {
			continue
		}

		switch lineType {
		case "Metric":
			var metric K6Metric
			if err := json.Unmarshal([]byte(line), &metric); err != nil {
				logrus.Warnf("Failed to unmarshal k6 Metric line: %v", err)
				continue
			}
			// If a metric has thresholds and is tainted, it means a threshold failed.
			if len(metric.Data.Value.Thresholds) > 0 && metric.Data.Value.Tainted != nil {
				isTainted := *metric.Data.Value.Tainted
				if isTainted {
					overallPass = false
				}
				// Note: This only tells us if the *metric* is tainted, not which specific threshold failed.
				// The 'thresholds' Point metric is more precise for individual status.
				for _, t := range metric.Data.Value.Thresholds {
					thresholds = append(thresholds, K6Threshold{Name: t, Metric: metric.Metric, Pass: !isTainted})
				}
			}
		case "Point":
			var point K6Point
			if err := json.Unmarshal([]byte(line), &point); err != nil {
				logrus.Warnf("Failed to unmarshal k6 Point line: %v", err)
				continue
			}

			if point.Metric == "checks" {
				check := K6Check{Name: point.Data.Value.Tags["check"], Passes: point.Data.Value.Passes, Fails: point.Data.Value.Fails}
				checks = append(checks, check)
				if check.Fails > 0 {
					overallPass = false
				}
			}
		}
	}
	return checks, thresholds, overallPass
}

func buildQaseComment(thresholds []K6Threshold, checks []K6Check, summary string) string {
	var builder strings.Builder

	builder.WriteString("### k6 Test Results\n\n")
	builder.WriteString("#### Thresholds\n")
	builder.WriteString("| Status | Threshold | Metric |\n")
	builder.WriteString("|---|---|---|\n")

	for _, t := range thresholds {
		statusIcon := "❌"
		if t.Pass {
			statusIcon = "✅"
		}
		builder.WriteString(fmt.Sprintf("| %s | `%s` | `%s` |\n", statusIcon, t.Name, t.Metric))
	}
	if len(thresholds) == 0 {
		builder.WriteString("| N/A | No thresholds defined | N/A |\n")
	}

	builder.WriteString("\n#### Checks\n")
	builder.WriteString("| Status | Check | Passes | Fails |\n")
	builder.WriteString("|---|---|---|---|\n")
	for _, c := range checks {
		statusIcon := "❌"
		if c.Fails == 0 {
			statusIcon = "✅"
		}
		builder.WriteString(fmt.Sprintf("| %s | `%s` | %d | %d |\n", statusIcon, c.Name, c.Passes, c.Fails))
	}
	if len(checks) == 0 {
		builder.WriteString("| N/A | No checks defined | N/A | N/A |\n")
	}

	if summary != "" {
		builder.WriteString("\n<details>\n<summary>Full k6 Summary</summary>\n\n")
		builder.WriteString("```\n")
		builder.WriteString(summary)
		builder.WriteString("\n```\n")
		builder.WriteString("</details>\n")
	}

	return builder.String()
}
