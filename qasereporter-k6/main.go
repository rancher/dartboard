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

	qase_config "github.com/qase-tms/qase-go/pkg/qase-go/config"
	v1 "github.com/qase-tms/qase-go/qase-api-client"
	"github.com/rancher/dartboard/internal/qase"
	"github.com/sirupsen/logrus"
)

const (
	k6MetricsOutputFileEnvVar = "K6_OUTPUT_FILE"
)

var (
	projectID           = os.Getenv(qase_config.QaseTestOpsProjectEnvVar)
	runIDStr            = os.Getenv(qase_config.QaseTestOpsRunIDEnvVar)
	runName             = os.Getenv(qase.TestRunNameEnvVar)
	testCaseName        = os.Getenv(qase.TestCaseNameEnvVar)
	k6MetricsOutputFile = os.Getenv(k6MetricsOutputFileEnvVar)
)

var qaseClient *qase.CustomUnifiedClient
var (
	runID      int64
	testCaseID int64
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

// K6Metric represents a 'Metric' line from the k6 metrics JSON.
type K6Metric struct {
	K6Line

	Data K6Data[K6MetricData] `json:"data"` // Data associated with the Metric, contains lots of stuff
}

// K6Point represents a 'Point' line from the k6 metrics JSON.
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
	Time   time.Time         `json:"time"`
	Tags   map[string]string `json:"tags"`
	Value  float64           `json:"value"`
	Passes int64             `json:"passes"`
	Fails  int64             `json:"fails"`
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

	granularReporting := flag.Bool("granular", false, "Enable granular reporting of all Metric and Point lines from k6 JSON output.")
	// The -runID flag allows overriding the test case ID.
	runIDOverride := flag.String("runID", "", "Qase test run ID to report results against.")
	flag.Parse()

	if runIDStr == "" && runName == "" {
		logrus.Fatalf("Missing required environment variables for reporting: both %s, and %s",
			qase_config.QaseTestOpsRunIDEnvVar, qase.TestRunNameEnvVar)
	}

	// Use the provided case ID flag, otherwise default to the run ID.
	if *runIDOverride != "" {
		runIDStr = *runIDOverride
	}

	qaseClient = qase.SetupQaseClient()

	parsedRunID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		if runName != "" {
			logrus.Infof("%s not found or invalid, creating new test run with name: %s", qase_config.QaseTestOpsRunIDEnvVar, runName)

			createdRunID, err := qaseClient.CreateTestRun(context.Background(), runName, projectID)
			if err != nil {
				logrus.Fatalf("Failed to create Qase test run: %v", err)
			}

			runID = createdRunID
			logrus.Infof("Successfully created new Qase test run with ID: %d", runID)
		} else {
			logrus.Fatalf("Invalid QASE_RUN_ID: %v", err)
		}
	} else {
		runID = parsedRunID

		resp, err := qaseClient.GetTestRun(context.Background(), projectID, runID)
		if err != nil {
			logrus.Fatalf("Failed to get Qase test run while fetching run title (runID: %v): %v", int32(runID), err)
		}

		runName = *resp.Title
	}

	if testCaseName != "" {
		logrus.Infof("Fetching Qase test case by title: %s", testCaseName)

		testCase, err := qaseClient.GetTestCaseByTitle(context.Background(), projectID, testCaseName)
		if err != nil {
			logrus.Fatalf("Failed to get Qase test case by title: %v", err)
		}

		testCaseID = *testCase.Id
	} else {
		// Fallback or error if no test case name is provided
		logrus.Fatalf("%s environment variable not set.", qase.TestCaseNameEnvVar)
	}

	// Get the full test case details to check for parameters
	testCaseDetails, err := qaseClient.GetTestCase(context.Background(), projectID, testCaseID)
	if err != nil {
		logrus.Fatalf("Failed to get full details for Qase test case ID %d: %v", testCaseID, err)
	}

	params := getAndValidateTestCaseParameters(testCaseDetails.Parameters)

	if !*granularReporting {
		reportSummary(params)
	} else {
		reportMetrics(params)
	}
}

func reportMetrics(params map[string]string) {
	logrus.Info("Granular reporting enabled.")
	// Granular reporting requires the metrics output file.
	if projectID == "" || k6MetricsOutputFile == "" {
		logrus.Fatalf("Missing required environment variables for granular reporting: one of %s, or %s",
			qase_config.QaseTestOpsProjectEnvVar, k6MetricsOutputFileEnvVar)
	}

	// Read and parse the k6 metrics JSON
	k6MetricsJsonData, err := os.ReadFile(k6MetricsOutputFile)
	if err != nil {
		logrus.Fatalf("Failed to read k6 metrics file %s: %v", k6MetricsOutputFile, err)
	}

	logrus.Info("Performing granular reporting of k6 metrics JSON.")

	checks, thresholds, overallPass := granularParseK6MetricsJson(k6MetricsJsonData)

	// Build the comment for Qase
	// NOTE: The granular parser does not have access to the text summary.
	summary := "Full text summary not available in granular reporting mode."
	comment := buildQaseComment(thresholds, checks, summary)

	// Report to Qase
	status := qase.StatusPassed
	if !overallPass {
		status = qase.StatusFailed
	}

	logrus.Infof("Reporting to Qase: Project=%s, Run=%d, Case=%d, Status=%s", projectID, runID, testCaseID, status)
	resultBody := v1.NewResultCreate(status)
	resultBody.SetCaseId(testCaseID)
	resultBody.SetComment(comment)

	if len(params) > 0 {
		resultBody.SetParam(params)
	}

	err = qaseClient.CreateTestResultV1(context.Background(), projectID, runID, *resultBody)
	if err != nil {
		logrus.Fatalf("Failed to create Qase result: %v", err)
	}

	logrus.Info("Successfully reported k6 results to Qase.")
}

// getAndValidateTestCaseParameters checks if a test case has parameters and validates that corresponding environment variables are set.
func getAndValidateTestCaseParameters(testCaseParameters []v1.TestCaseParameter) map[string]string {
	if len(testCaseParameters) == 0 {
		logrus.Info("Test case has no parameters, skipping validation.")
		return nil
	}

	logrus.Infof("Test case has %d parameter(s), validating against environment variables...", len(testCaseParameters))

	parametersMap := make(map[string]string)

	for _, parameter := range testCaseParameters {
		var items []v1.ParameterSingle
		if parameter.TestCaseParameterSingle != nil {
			items = append(items, parameter.TestCaseParameterSingle.Item)
		} else if parameter.TestCaseParameterGroup != nil {
			items = append(items, parameter.TestCaseParameterGroup.Items...)
		} else {
			logrus.Warnf("Skipping unknown or malformed test case parameter.")
			continue
		}

		for _, item := range items {
			parameterTitle := item.Title
			parameterValue, isSet := os.LookupEnv(parameterTitle)

			if !isSet {
				logrus.Fatalf("Validation failed: Test case parameter '%s' is not set as an environment variable.", parameterTitle)
			}

			logrus.Debugf("Found environment variable for parameter '%s'", parameterTitle)
			parametersMap[parameterTitle] = parameterValue
		}
	}

	return parametersMap
}

// granularParseK6MetricsJson processes the raw metrics JSON from k6 by inspecting every
// Metric and Point line to determine which Thresholds and Checks passed or failed.
func granularParseK6MetricsJson(jsonData []byte) ([]K6Check, []K6Threshold, bool) {
	var (
		checks     []K6Check
		thresholds []K6Threshold
	)

	overallPass := true

	// The k6 metrics JSON is a stream of JSON objects, one per line.
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
	builder.WriteString("###### Thresholds\n")
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

	builder.WriteString("\n###### Checks\n")
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
