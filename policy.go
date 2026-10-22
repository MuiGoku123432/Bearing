package bearing

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"runtime/debug"
)

//go:embed ENGINEERING.md
var engineering string

//go:embed WORKFLOW.md
var workflow string

//go:embed PLANNING.md
var planning string

// version is set at link time for builds without a module version:
// -ldflags "-X github.com/MuiGoku123432/Bearing.version=v0.1.0"
var version string

// Version reports the module version recorded by go install or a Git-stamped build,
// then the -ldflags value, then the development version.
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	if version != "" {
		return version
	}
	return "0.1.0-dev"
}

type Policy struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	WorkflowID string `json:"workflow_id"`
	Workflow   string `json:"workflow,omitempty"`
	PlanningID string `json:"planning_id"`
	Planning   string `json:"planning,omitempty"`
}

func Rules() Policy {
	return Policy{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(engineering))), Text: engineering,
		WorkflowID: fmt.Sprintf("%x", sha256.Sum256([]byte(workflow))), Workflow: workflow,
		PlanningID: fmt.Sprintf("%x", sha256.Sum256([]byte(planning))), Planning: planning}
}
