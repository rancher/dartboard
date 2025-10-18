package qase

import (
	"context"
	"fmt"
	"os"

	qase_config "github.com/qase-tms/qase-go/pkg/qase-go/config"
	"github.com/sirupsen/logrus"
)

const (
	TestRunNameEnvVar  = "QASE_TEST_RUN_NAME"
	TestCaseNameEnvVar = "QASE_TEST_CASE_NAME"
)

type Service struct {
	Client      *CustomUnifiedClient
	RunID       int64
	ProjectCode string
}

// SetupQaseClient creates a new Qase client using the qase-go package.
func SetupQaseClient() *Service {
	token := os.Getenv(qase_config.QaseTestOpsAPITokenEnvVar)
	projectCode := os.Getenv(qase_config.QaseTestOpsProjectEnvVar)
	if token == "" {
		logrus.Fatal("QASE_AUTOMATION_TOKEN environment variable not set")
	}
	if projectCode == "" {
		logrus.Fatal("QASE_PROJECT_ID environment variable not set")
	}
	cfgBuilder := qase_config.NewConfigBuilder().LoadFromEnvironment()
	cfg, err := cfgBuilder.Build()
	if err != nil {
		logrus.Fatalf("Failed to build Qase config from environment variables: %v", err)
	}

	qaseClient, err := NewCustomUnifiedClient(cfg)
	if err != nil {
		logrus.Fatalf("Failed to create Qase client: %v", err)
	}

	return &Service{
		Client:      qaseClient,
		RunID:       *cfg.TestOps.Run.ID,
		ProjectCode: projectCode,
	}
}

// CreateTestRun creates a new Qase test run.
func (q *Service) CreateTestRun(testRunName string, projectID string) (int64, error) {
	if q.RunID != 0 {
		return q.RunID, nil
	}

	runID, err := q.Client.CreateTestRun(context.Background(), testRunName, projectID)
	if err != nil {
		return 0, err
	}
	q.RunID = runID
	return q.RunID, nil
}

// CompleteTestRun completes the test run if it was started.
func (q *Service) CompleteTestRun() error {
	if q.RunID != 0 {
		logrus.Debugf("Completing test run ID: %d", q.RunID)
		if err := q.Client.CompleteTestRun(context.Background(), q.ProjectCode, q.RunID); err != nil {
			return err
		}
		q.RunID = 0
	}
	return nil
}

// UploadAttachments uploads files to Qase and returns their hashes.
func (q *Service) UploadAttachments(files []*os.File) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}

	hashes, err := q.Client.UploadAttachments(context.Background(), q.ProjectCode, files)
	if err != nil {
		return nil, err // The client method already handles logging, so we just return the error.
	}
	if len(hashes) == 0 && len(files) > 0 {
		return nil, fmt.Errorf("failed to upload any attachments")
	}

	logrus.Infof("Successfully uploaded %d out of %d attachments.", len(hashes), len(files))
	return hashes, nil
}
