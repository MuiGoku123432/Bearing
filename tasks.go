package bearing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func validateAcceptance(criteria []Criterion) error {
	if len(criteria) == 0 || len(criteria) > 30 {
		return fail("invalid_input", "Provide one to thirty acceptance criteria.")
	}
	seen := map[string]bool{}
	for i := range criteria {
		if criteria[i].ID == "" {
			criteria[i].ID = fmt.Sprintf("a%d", i+1)
		}
		if seen[criteria[i].ID] || strings.TrimSpace(criteria[i].Description) == "" {
			return fail("invalid_input", "Acceptance criteria need unique IDs and clear descriptions.")
		}
		seen[criteria[i].ID] = true
	}
	return nil
}

func (s *Store) Start(ctx context.Context, in StartInput) (StartResult, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	in.Acceptance = append([]Criterion(nil), in.Acceptance...)
	var repo Repository
	mutation, err := s.write(ctx, in.RequestID, "start", in, func() error {
		if err := inputSize(in, 16000); err != nil {
			return err
		}
		if len(in.Outcome) > 2000 {
			return fail("invalid_input", "Keep the task outcome within 2000 bytes.")
		}
		if strings.TrimSpace(in.Outcome) == "" {
			return fail("invalid_input", "Describe the intended outcome.")
		}
		if in.Kind == "" {
			in.Kind = "build"
		}
		if in.Kind != "build" && in.Kind != "research" && in.Kind != "spec" {
			return fail("invalid_input", "Task kind must be build, research, or spec.")
		}
		if len(in.InputDocuments) > 16 {
			return fail("invalid_input", "Link at most sixteen input documents.")
		}
		if err := validateAcceptance(in.Acceptance); err != nil {
			return err
		}
		var err error
		repo, err = ResolveRepository(ctx, s.Root)
		if err != nil {
			return err
		}
		// The start tree is the baseline for test files changed before finish.
		repo.Tree, err = workingTree(ctx, repo)
		return err
	}, func(tx *sql.Tx) (Mutation, error) {
		for _, id := range in.InputDocuments {
			var exists int
			if err := tx.QueryRowContext(ctx, "SELECT 1 FROM documents WHERE id = ?", id).Scan(&exists); err != nil {
				return Mutation{}, fail("invalid_input", "An input document does not exist in this checkout.")
			}
		}
		if in.RelatedTaskID != "" {
			var exists int
			if err := tx.QueryRowContext(ctx, "SELECT 1 FROM tasks WHERE id = ?", in.RelatedTaskID).Scan(&exists); err != nil {
				return Mutation{}, fail("invalid_input", "Related task not found.")
			}
		}
		task := Task{ID: NewID(), Kind: in.Kind, Outcome: in.Outcome, Acceptance: in.Acceptance, InputDocuments: in.InputDocuments, RelatedTaskID: in.RelatedTaskID, Status: "active", Version: 1, CreatedAt: repo.CapturedAt, UpdatedAt: repo.CapturedAt, CreatedContext: repo}
		data, err := json.Marshal(task)
		if err != nil {
			return Mutation{}, err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO tasks(id,status,version,data) VALUES(?,?,?,?)", task.ID, task.Status, task.Version, string(data))
		return Mutation{ID: task.ID, Version: 1, Status: task.Status}, err
	})
	if err != nil {
		return StartResult{}, err
	}
	result := StartResult{Mutation: mutation}
	result.Context, err = s.Resume(ctx, mutation.ID)
	if err != nil {
		result.Context = Context{Policy: Rules(), Warnings: []string{"Task saved. Context could not be loaded; resume using the returned task ID."}}
	}
	return result, nil
}
