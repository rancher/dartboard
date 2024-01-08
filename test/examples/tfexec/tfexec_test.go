package examples

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TfApplyDestroy struct {
	suite.Suite
	session *session.Session
	client  *tfexec.Terraform
	config  *Config
}

func (s *TfApplyDestroy) TearDownTest() {
	log.Info("TEARING DOWN")
	s.TfDestroy()
	_, err := os.Stat(s.config.PlanFilePath)
	if err == nil {
		err := os.Remove(s.config.PlanFilePath)
		require.NoError(s.T(), err)
	}
}

func (s *TfApplyDestroy) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession
	s.config = new(Config)
	config.LoadConfig(ConfigurationFileKey, s.config)

	client, err := tfexec.NewTerraform(s.config.WorkingDir, s.config.ExecPath)
	require.NoError(s.T(), err)
	log.Info("Successfully created tfexec instance")

	err = client.Init(context.Background())
	require.NoError(s.T(), err)
	s.client = client
	log.Info("Successfully ran `terraform init` on module at: ", s.client.WorkingDir())

	s.TfSetupWorkspace()
}

func (s *TfApplyDestroy) TfPlanJSON() {
	if s.config.PlanFilePath == "" {
		s.config.PlanFilePath = s.config.PlanOpts.OutDir + "/tfexec_plan_" + time.Now().Format(time.RFC3339) + ".tfplan"
	}
	outOpt := tfexec.PlanOption(tfexec.Out(s.config.PlanFilePath))
	varFileOpt := tfexec.PlanOption(tfexec.VarFile(s.config.VarFilePath))
	_, err := s.client.PlanJSON(context.Background(), nil, varFileOpt, outOpt)
	require.NoError(s.T(), err)
	log.Info("Successfully ran `terraform plan -json -var-file=" + s.config.VarFilePath + " -out=" + s.config.PlanFilePath + "`")
}

func (s *TfApplyDestroy) TfApplyPlanJSON() {
	planFileOpt := tfexec.ApplyOption(tfexec.DirOrPlan(s.config.PlanFilePath))
	err := s.client.ApplyJSON(context.Background(), os.Stdout, planFileOpt)
	require.NoError(s.T(), err)
	log.Info("Successfully ran `terraform apply -json " + s.config.PlanFilePath)
	// state, err := s.client.Show(context.Background())
	// require.NoError(s.T(), err)
	// out, err := json.Marshal(state.Values.RootModule.Resources)
	// require.NoError(s.T(), err)
	// log.Info("TF State: " + string(out))
}

func (s *TfApplyDestroy) TfOutput() {
	outputs, err := s.client.Output(context.Background())
	require.NoError(s.T(), err)
	outputsJSON, err := json.Marshal(outputs)
	require.NoError(s.T(), err)
	log.Info("Output: " + string(outputsJSON))
}

func (s *TfApplyDestroy) TfWorkspaceExists(ctx context.Context, workspace string) bool {
	wsList, _, err := s.client.WorkspaceList(context.Background())
	require.NoError(s.T(), err)
	for _, str := range wsList {
		if str == workspace {
			return true
		}
	}
	return false
}

func (s *TfApplyDestroy) TfSetupWorkspace() {
	var err error
	if s.config.WorkspaceName != "" {
		if s.TfWorkspaceExists(context.Background(), s.config.WorkspaceName) {
			err = s.client.WorkspaceSelect(context.Background(), s.config.WorkspaceName)
		} else {
			err = s.client.WorkspaceNew(context.Background(), s.config.WorkspaceName)
		}
	} else {
		s.config.WorkspaceName = "dartboard-" + namegen.RandStringLower(5)
		err = s.client.WorkspaceNew(context.Background(), s.config.WorkspaceName)
	}
	require.NoError(s.T(), err)
	log.Info("Successfully setup workspace: " + s.config.WorkspaceName)
}

func (s *TfApplyDestroy) TfDestroy() {
	varFileOpt := tfexec.DestroyOption(tfexec.VarFile(s.config.VarFilePath))
	err := s.client.DestroyJSON(context.Background(), os.Stdout, varFileOpt)
	require.NoError(s.T(), err)
	log.Info("Successfully ran `terraform destroy -json -var-file=" + s.config.VarFilePath + "`")
}

func (s *TfApplyDestroy) TestTfPlanJSON() {
	s.TfPlanJSON()
}

func (s *TfApplyDestroy) TestTfApplyPlanJSON() {
	s.Run("Apply plan with planfile", func() {
		s.TfPlanJSON()
		s.TfApplyPlanJSON()
	})
}

func (s *TfApplyDestroy) TestTfOutput() {
	_, err := os.Stat(s.config.PlanFilePath)
	if err != nil {
		s.TfPlanJSON()
	}
	s.TfOutput()
}

func TestTfApplyDestroy(t *testing.T) {
	suite.Run(t, new(TfApplyDestroy))
}
