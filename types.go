package bearing

// Repository is an observation, not proof that code or evidence is current.
type Repository struct {
	Path       string `json:"path"`
	Head       string `json:"head,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Dirty      bool   `json:"dirty"`
	CapturedAt string `json:"captured_at"`
}

type Criterion struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
}

type Task struct {
	ID                 string      `json:"id"`
	Kind               string      `json:"kind"`
	Outcome            string      `json:"outcome"`
	Acceptance         []Criterion `json:"acceptance"`
	InputDocuments     []string    `json:"input_documents,omitempty"`
	RelatedTaskID      string      `json:"related_task_id,omitempty"`
	Status             string      `json:"status"`
	Version            int         `json:"version"`
	LatestCheckpointID string      `json:"latest_checkpoint_id,omitempty"`
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
	CreatedContext     Repository  `json:"created_context"`
}

// Evidence is agent-reported; the checkpoint that stores it records Bearing's own checkout context.
type Evidence struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Description   string   `json:"description"`
	Source        string   `json:"source"`
	Outcome       string   `json:"outcome"`
	ExitCode      *int     `json:"exit_code,omitempty"`
	ObservedAt    string   `json:"observed_at,omitempty"`
	AcceptanceIDs []string `json:"acceptance_ids"`
	Blocking      bool     `json:"blocking,omitempty"`
}

// AcceptanceResult is kept so finishes saved before results were derived from verification stay readable.
type AcceptanceResult struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type ReviewCheck struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type ReviewInput struct {
	Summary    string                 `json:"summary"`
	Checks     map[string]ReviewCheck `json:"checks"`
	Unresolved []string               `json:"unresolved,omitempty"`
}

// Review is the saved review; Bearing records the policy it was checked against.
type Review struct {
	PolicyID string `json:"policy_id"`
	ReviewInput
}

type TaskUpdate struct {
	Outcome    string      `json:"outcome"`
	Acceptance []Criterion `json:"acceptance"`
	Reason     string      `json:"reason"`
	Source     string      `json:"source"`
}

type DocumentInput struct {
	Path    string   `json:"path"`
	Kind    string   `json:"kind"`
	Summary string   `json:"summary"`
	Sources []string `json:"sources,omitempty"`
}

type Document struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	CheckpointID string `json:"checkpoint_id"`
	DocumentInput
	Digest         string   `json:"digest"`
	Body           string   `json:"body,omitempty"`
	InputDocuments []string `json:"input_documents,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

type Checkpoint struct {
	ID                string             `json:"id"`
	TaskID            string             `json:"task_id"`
	Sequence          int                `json:"sequence"`
	Summary           string             `json:"summary"`
	NextAction        string             `json:"next_action,omitempty"`
	Steps             []string           `json:"steps,omitempty"`
	Blockers          []string           `json:"blockers,omitempty"`
	FailedAttempts    []string           `json:"failed_attempts,omitempty"`
	Evidence          []Evidence         `json:"evidence,omitempty"`
	MemoryRefs        []string           `json:"memory_refs,omitempty"`
	DocumentIDs       []string           `json:"document_ids,omitempty"`
	TaskUpdate        *TaskUpdate        `json:"task_update,omitempty"`
	PreviousTask      *Task              `json:"previous_task,omitempty"`
	AcceptanceResults []AcceptanceResult `json:"acceptance_results,omitempty"`
	Verification      []Evidence         `json:"verification,omitempty"`
	Review            *Review            `json:"review,omitempty"`
	Limitations       []string           `json:"limitations,omitempty"`
	Context           Repository         `json:"context"`
	CreatedAt         string             `json:"created_at"`
}

type StartInput struct {
	RequestID      string      `json:"request_id,omitempty"`
	Kind           string      `json:"kind,omitempty"`
	Outcome        string      `json:"outcome"`
	Acceptance     []Criterion `json:"acceptance"`
	InputDocuments []string    `json:"input_documents,omitempty"`
	RelatedTaskID  string      `json:"related_task_id,omitempty"`
}

type CheckpointInput struct {
	RequestID       string          `json:"request_id,omitempty"`
	TaskID          string          `json:"task_id"`
	ExpectedVersion int             `json:"expected_version"`
	Summary         string          `json:"summary"`
	NextAction      string          `json:"next_action,omitempty"`
	Status          string          `json:"status,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	Steps           []string        `json:"steps,omitempty"`
	Blockers        []string        `json:"blockers,omitempty"`
	FailedAttempts  []string        `json:"failed_attempts,omitempty"`
	Evidence        []Evidence      `json:"evidence,omitempty"`
	MemoryRefs      []string        `json:"memory_refs,omitempty"`
	Documents       []DocumentInput `json:"documents,omitempty"`
	TaskUpdate      *TaskUpdate     `json:"task_update,omitempty"`
}

type FinishInput struct {
	RequestID       string          `json:"request_id,omitempty"`
	TaskID          string          `json:"task_id"`
	ExpectedVersion int             `json:"expected_version"`
	Summary         string          `json:"summary"`
	Verification    []Evidence      `json:"verification"`
	Review          ReviewInput     `json:"review"`
	Documents       []DocumentInput `json:"documents,omitempty"`
	Limitations     []string        `json:"limitations,omitempty"`
}

type Mutation struct {
	RequestID    string   `json:"request_id"`
	ID           string   `json:"id"`
	Version      int      `json:"version"`
	CheckpointID string   `json:"checkpoint_id,omitempty"`
	DocumentIDs  []string `json:"document_ids,omitempty"`
	Status       string   `json:"status"`
	Replayed     bool     `json:"replayed"`
	PolicyID     string   `json:"policy_id"`
}

type Context struct {
	Policy     Policy      `json:"policy"`
	Repository Repository  `json:"repository"`
	Task       *Task       `json:"task,omitempty"`
	Checkpoint *Checkpoint `json:"checkpoint,omitempty"`
	Documents  []Document  `json:"documents,omitempty"`
	Memories   []Memory    `json:"memories,omitempty"`
	Candidates []Task      `json:"candidates,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
	Omitted    int         `json:"omitted"`
	Next       string      `json:"next"`
}

type StartResult struct {
	Mutation Mutation `json:"mutation"`
	Context  Context  `json:"context"`
}
