package qase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/qase-tms/qase-go/pkg/qase-go/clients"
	"github.com/qase-tms/qase-go/pkg/qase-go/config"
	api_v1_client "github.com/qase-tms/qase-go/qase-api-client"
	api_v2_client "github.com/qase-tms/qase-go/qase-api-v2-client"
	"github.com/sirupsen/logrus"
)

const (
	TestRunNameEnvVar  = "QASE_TEST_RUN_NAME"
	TestCaseNameEnvVar = "QASE_TEST_CASE_NAME"
)

// CustomUnifiedClient combines V1 and V2 clients for our specific needs.
type CustomUnifiedClient struct {
	V1Client *clients.V1Client
	V2Client *clients.V2Client
	Config   *config.Config
}

// NewCustomUnifiedClient creates a new client that encapsulates V1 and V2 clients.
func NewCustomUnifiedClient(cfg *config.Config) (*CustomUnifiedClient, error) {
	// BaseURL is set by the underling APIClients
	clientConfig := clients.ClientConfig{
		APIToken: cfg.TestOps.API.Token,
		Debug:    cfg.Debug,
	}

	v1Client, err := clients.NewV1Client(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create v1 client: %w", err)
	}

	v2Client, err := clients.NewV2Client(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create v2 client: %w", err)
	}

	return &CustomUnifiedClient{
		V1Client: v1Client,
		V2Client: v2Client,
		Config:   cfg,
	}, nil
}

// SetupQaseClient creates a new Qase client using the qase-go package.
func SetupQaseClient() *CustomUnifiedClient {
	token := os.Getenv(config.QaseTestOpsAPITokenEnvVar)
	projectCode := os.Getenv(config.QaseTestOpsProjectEnvVar)
	if token == "" {
		logrus.Fatalf("%s environment variable not set", config.QaseTestOpsAPITokenEnvVar)
	}
	if projectCode == "" {
		logrus.Fatalf("%s environment variable not set", config.QaseTestOpsProjectEnvVar)
	}

	var err error
	var cfg *config.Config

	cfgBuilder := config.NewConfigBuilder().LoadFromEnvironment()
	cfg, err = cfgBuilder.Build()
	if err != nil {
		logrus.Fatalf("Failed to build Qase config from environment variables: %v", err)
	}
	if cfg.Mode == "" {
		cfg.Mode = config.MODE_TESTOPS
	}
	if cfg.Fallback == "" {
		cfg.Fallback = config.MODE_REPORT
	}
	if cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	qaseClient, err := NewCustomUnifiedClient(cfg)
	if err != nil {
		logrus.Fatalf("Failed to create Qase client: %v", err)
	}

	logrus.Debugf("QASE Config: %v", cfg)

	return qaseClient
}

// CreateTestRun creates a new Qase test run using the V1 client.
func (c *CustomUnifiedClient) CreateTestRun(ctx context.Context, testRunName, projectCode string) (int64, error) {
	logrus.Debugf("Creating test run with name \"%s\" in project %s", testRunName, projectCode)

	runCreate := api_v1_client.NewRunCreate(testRunName)
	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	resp, res, err := c.V1Client.GetAPIClient().RunsAPI.CreateRun(authCtx, projectCode).RunCreate(*runCreate).Execute()
	logResponseBody(res, "CreateTestRun")
	if err != nil {
		return 0, fmt.Errorf("failed to create test run: %w", err)
	}
	runID := *resp.Result.Id
	c.Config.TestOps.Run.ID = &runID
	return runID, nil
}

// GetTestRun retrieves a Qase test run by its ID.
func (c *CustomUnifiedClient) GetTestRun(ctx context.Context, projectCode string, runID int64) (*api_v1_client.Run, error) {
	logrus.Debugf("Getting test run with ID %d in project %s", runID, projectCode)

	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	resp, res, err := c.V1Client.GetAPIClient().RunsAPI.GetRun(authCtx, projectCode, int32(runID)).Execute()
	logResponseBody(res, "GetTestRun")
	if err != nil {
		return nil, fmt.Errorf("failed to get test run: %w", err)
	}

	return resp.Result, nil
}

// GetTestCase retrieves a Qase test case by its ID.
func (c *CustomUnifiedClient) GetTestCase(ctx context.Context, projectCode string, caseID int64) (*api_v1_client.TestCase, error) {
	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})
	resp, res, err := c.V1Client.GetAPIClient().CasesAPI.GetCase(authCtx, projectCode, int32(caseID)).Execute()
	logResponseBody(res, "GetTestCase")
	return resp.Result, err
}

// CompleteTestRun completes a Qase test run using the V1 client.
func (c *CustomUnifiedClient) CompleteTestRun(ctx context.Context, projectCode string, runID int64) error {
	logrus.Debugf("Completing test run ID: %d", runID)

	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	if c.Config.TestOps.Run.ID != nil {
		runID := *c.Config.TestOps.Run.ID
		logrus.Debugf("Completing test run ID: %d", runID)
		_, res, err := c.V1Client.GetAPIClient().RunsAPI.CompleteRun(authCtx, c.Config.TestOps.Project, int32(runID)).Execute()
		logResponseBody(res, "CompleteTestRun")
		if err != nil {
			return fmt.Errorf("failed to complete test run: %w", err)
		}
	}
	return nil
}

// UploadAttachments uploads files to Qase using the V1 client.
func (c *CustomUnifiedClient) UploadAttachments(ctx context.Context, files []*os.File) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}

	projectCode := c.Config.TestOps.Project

	var hashes []string
	for _, file := range files {
		logrus.Debugf("Uploading attachment: %s", file.Name())
		hash, err := c.V1Client.UploadAttachment(ctx, projectCode, []*os.File{file})
		if err != nil {
			logrus.Warnf("Failed to upload attachment %s: %v", file.Name(), err)
			continue
		}
		if hash != "" {
			hashes = append(hashes, hash)
		}
	}

	if len(hashes) == 0 && len(files) > 0 {
		return nil, fmt.Errorf("failed to upload any attachments")
	}

	logrus.Infof("Successfully uploaded %d out of %d attachments.", len(hashes), len(files))
	return hashes, nil
}

// GetTestCaseByTitle finds a Qase test case by its title using the V1 client.
func (c *CustomUnifiedClient) GetTestCaseByTitle(ctx context.Context, projectCode, title string) (*api_v1_client.TestCase, error) {
	logrus.Debugf("Getting test case with title \"%s\" in project %s", title, projectCode)

	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	limit := int32(100)
	offset := int32(0)
	var matchingCase *api_v1_client.TestCase

	for {
		resp, res, err := c.V1Client.GetAPIClient().CasesAPI.GetCases(authCtx, projectCode).
			Search(title).
			Limit(limit).
			Offset(offset).
			Execute()
		logResponseBody(res, "GetTestCaseByTitle")
		if err != nil {
			return nil, fmt.Errorf("failed to get test cases: %w", err)
		}

		for i := range resp.Result.Entities {
			entity := resp.Result.Entities[i]
			if entity.Title != nil && *entity.Title == title {
				matchingCase = &entity
				break
			}
		}

		if matchingCase != nil || len(resp.Result.Entities) < int(limit) {
			break
		}

		offset += limit
	}

	if matchingCase == nil {
		return nil, fmt.Errorf("test case with title \"%s\" not found in project %s", title, projectCode)
	}

	return matchingCase, nil
}

// CreateTestResultV1 creates a test result using the V1 API.
func (c *CustomUnifiedClient) CreateTestResultV1(ctx context.Context, projectCode string, runID int64, result api_v1_client.ResultCreate) error {
	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})
	_, res, err := c.V1Client.GetAPIClient().ResultsAPI.CreateResult(authCtx, projectCode, int32(runID)).ResultCreate(result).Execute()
	logResponseBody(res, "CreateTestResultV1")
	if err != nil || !strings.Contains(strings.ToLower(res.Status), "ok") {
		return fmt.Errorf("failed to create v1 test result or did not receive 'OK; response: %w", err)
	}
	return nil
}

// CreateTestResultV2 creates a test result using the V2 API.
func (c *CustomUnifiedClient) CreateTestResultV2(ctx context.Context, projectCode string, runID int64, result api_v2_client.ResultCreate) error {
	res, err := c.V2Client.GetAPIClient().ResultsAPI.CreateResultV2(ctx, projectCode, runID).ResultCreate(result).Execute()
	logResponseBody(res, "CreateTestResultV2")
	if err != nil {
		return fmt.Errorf("failed to create v2 test result: %w", err)
	}
	return nil
}

// logResponseBody reads, logs, and restores an *http.Response.Body for debugging.
func logResponseBody(res *http.Response, context string) {
	if res == nil || res.Body == nil {
		return
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Warnf("Failed to read response body for %s: %v", context, err)
		return
	}

	res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	logrus.Debugf("%s response body: %s", context, string(bodyBytes))
}
