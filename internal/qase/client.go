package qase

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qase-tms/qase-go/pkg/qase-go/clients"
	"github.com/qase-tms/qase-go/pkg/qase-go/config"
	api_v1_client "github.com/qase-tms/qase-go/qase-api-client"
	api_v2_client "github.com/qase-tms/qase-go/qase-api-v2-client"
	"github.com/sirupsen/logrus"
)

// CustomUnifiedClient combines V1 and V2 clients for our specific needs.
type CustomUnifiedClient struct {
	V1Client *clients.V1Client
	V2Client *clients.V2Client
	Config   *config.Config
}

// NewCustomUnifiedClient creates a new client that encapsulates V1 and V2 clients.
func NewCustomUnifiedClient(cfg *config.Config) (*CustomUnifiedClient, error) {
	clientConfig := clients.ClientConfig{
		APIToken: cfg.TestOps.API.Token,
		BaseURL:  "https://api.qase.io/v1",
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

// CreateTestRun creates a new Qase test run using the V1 client.
func (c *CustomUnifiedClient) CreateTestRun(ctx context.Context, testRunName, projectCode string) (int64, error) {
	logrus.Debugf("Creating test run with name \"%s\" in project %s", testRunName, projectCode)

	runCreate := api_v1_client.NewRunCreate(testRunName)
	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	resp, _, err := c.V1Client.GetAPIClient().RunsAPI.CreateRun(authCtx, projectCode).RunCreate(*runCreate).Execute()
	if err != nil {
		return 0, fmt.Errorf("failed to create test run: %w", err)
	}
	return *resp.Result.Id, nil
}

// CompleteTestRun completes a Qase test run using the V1 client.
func (c *CustomUnifiedClient) CompleteTestRun(ctx context.Context, projectCode string, runID int64) error {
	logrus.Debugf("Completing test run ID: %d", runID)

	authCtx := context.WithValue(ctx, api_v1_client.ContextAPIKeys, map[string]api_v1_client.APIKey{
		"TokenAuth": {Key: c.Config.TestOps.API.Token},
	})

	_, _, err := c.V1Client.GetAPIClient().RunsAPI.CompleteRun(authCtx, projectCode, int32(runID)).Execute()
	if err != nil {
		return fmt.Errorf("failed to complete test run: %w", err)
	}
	return nil
}

// UploadAttachments uploads files to Qase using the V1 client.
func (c *CustomUnifiedClient) UploadAttachments(ctx context.Context, projectCode string, files []*os.File) ([]string, error) {
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
		resp, _, err := c.V1Client.GetAPIClient().CasesAPI.GetCases(authCtx, projectCode).
			Search(title).
			Limit(limit).
			Offset(offset).
			Execute()
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
	_, r, err := c.V1Client.GetAPIClient().ResultsAPI.CreateResult(authCtx, projectCode, int32(runID)).ResultCreate(result).Execute()
	if err != nil || !strings.Contains(strings.ToLower(r.Status), "ok") {
		return fmt.Errorf("failed to create v1 test result or did not receive 'OK; response: %w", err)
	}
	return nil
}

// CreateTestResultV2 creates a test result using the V2 API.
func (c *CustomUnifiedClient) CreateTestResultV2(ctx context.Context, projectCode string, runID int64, result api_v2_client.ResultCreate) error {
	_, err := c.V2Client.GetAPIClient().ResultsAPI.CreateResultV2(ctx, projectCode, runID).ResultCreate(result).Execute()
	if err != nil {
		return fmt.Errorf("failed to create v2 test result: %w", err)
	}
	return nil
}
