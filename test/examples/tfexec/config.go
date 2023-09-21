package examples

const (
	ConfigurationFileKey = "tfexecInput"
)

type PlanOptions struct {
	OutDir string `json:"outDir" yaml:"outDir"`
}

type Config struct {
	WorkspaceName string      `json:"workspaceName" yaml:"workspaceName"`
	WorkingDir    string      `json:"workingDir" yaml:"workingDir"`
	ExecPath      string      `json:"execPath" yaml:"execPath"`
	VarFilePath   string      `json:"varFilePath" yaml:"varFilePath"`
	PlanFilePath  string      `json:"planFilePath" yaml:"planFilePath"`
	PlanOpts      PlanOptions `json:"planOpts" yaml:"planOpts"`
}
