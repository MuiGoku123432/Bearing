package bearing

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
)

//go:embed ENGINEERING.md
var engineering string

//go:embed WORKFLOW.md
var workflow string

const Version = "0.1.0-dev"

type Policy struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	WorkflowID string `json:"workflow_id"`
	Workflow   string `json:"workflow,omitempty"`
}

func Rules() Policy {
	return Policy{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(engineering))), Text: engineering,
		WorkflowID: fmt.Sprintf("%x", sha256.Sum256([]byte(workflow))), Workflow: workflow}
}
