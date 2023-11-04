package examples

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	tfClient "github.com/rancher/rancher/tests/framework/clients/tfexec"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TfClientApplyDestroy struct {
	suite.Suite
	session *session.Session
	client  *tfClient.Client
}

func (s *TfClientApplyDestroy) TearDownSuite() {
	s.client.DestroyJSON(os.Stdout)
	_, err := os.Stat(s.client.TerraformConfig.PlanFilePath)
	if err == nil {
		err := os.Remove(s.client.TerraformConfig.PlanFilePath)
		require.NoError(s.T(), err)
	}
}

func (s *TfClientApplyDestroy) SetupSuite() {
	// Force standard logger to parse newlines without quoting (print actual newline instead of "\n")
	log.StandardLogger().SetFormatter(&log.TextFormatter{
		DisableQuote: true,
	})

	testSession := session.NewSession()
	s.session = testSession

	client, err := tfClient.NewClient()
	require.NoError(s.T(), err)
	log.Info("Successfully created tfexec instance")

	err = client.InitTerraform()
	require.NoError(s.T(), err)
	s.client = client
	log.Info("Successfully ran `terraform init` on module at: ", s.client.WorkingDir())

	s.client.SetupWorkspace()
	log.Info("Successfully setup workspace: ", s.client.TerraformConfig.WorkspaceName)
}

func (s *TfClientApplyDestroy) TestTfApplyPlanJSON() {
	require.NoError(s.T(), s.client.PlanJSON(os.Stdout))
	require.NoError(s.T(), s.client.ApplyPlanJSON(os.Stdout))
	outputs, err := s.client.Output()
	require.NoError(s.T(), err)
	// Pretty print output
	b, err := json.MarshalIndent(outputs, "", "  ")
	if err == nil {
		fmt.Println(string(b))
		log.Info("Outputs: \n", string(b), "\n")
	}
}

func TestTfClientApplyDestroy(t *testing.T) {
	suite.Run(t, new(TfClientApplyDestroy))
}
