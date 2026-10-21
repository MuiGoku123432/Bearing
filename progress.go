package bearing

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

func (s *Store) Checkpoint(ctx context.Context, in CheckpointInput) (Mutation, error) {
	return s.saveProgress(ctx, in, nil)
}

func (s *Store) Finish(ctx context.Context, in FinishInput) (Mutation, error) {
	return s.saveProgress(ctx, CheckpointInput{RequestID: in.RequestID, TaskID: in.TaskID, ExpectedVersion: in.ExpectedVersion, Summary: in.Summary, Documents: in.Documents}, &in)
}

// saveProgress keeps the checkpoint, document versions, task state, and retry receipt atomic.
func (s *Store) saveProgress(ctx context.Context, in CheckpointInput, finish *FinishInput) (Mutation, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	operation := "checkpoint"
	var input any = in
	if finish != nil {
		operation, input = "finish", finish
	}
	if in.TaskUpdate != nil {
		update := *in.TaskUpdate
		update.Acceptance = append([]Criterion(nil), update.Acceptance...)
		in.TaskUpdate = &update
	}
	var repo Repository
	var docs []Document
	return s.write(ctx, in.RequestID, operation, input, func() error {
		if len(in.Summary) > 2000 || len(in.NextAction) > 1000 {
			return fail("invalid_input", "Keep summaries within 2000 bytes and next actions within 1000 bytes.")
		}
		if strings.TrimSpace(in.Summary) == "" {
			return fail("invalid_input", "Provide a checkpoint summary.")
		}
		if in.Status == "" {
			in.Status = "active"
		}
		if finish == nil {
			if err := inputSize(in, 16384); err != nil {
				return err
			}
			if in.Status != "cancelled" && strings.TrimSpace(in.NextAction) == "" {
				return fail("invalid_input", "Save one concrete next action for unfinished work.")
			}
		}
		if in.Status != "active" && in.Status != "blocked" && in.Status != "cancelled" {
			return fail("invalid_input", "Checkpoint status must be active, blocked, or cancelled.")
		}
		if in.Status == "cancelled" && strings.TrimSpace(in.Reason) == "" {
			return fail("invalid_input", "Cancellation needs a reason.")
		}
		if in.Status == "blocked" && len(in.Blockers) == 0 {
			return fail("invalid_input", "A blocked task needs a blocker.")
		}
		if in.TaskUpdate != nil {
			if err := inputSize(in.TaskUpdate, 16000); err != nil {
				return err
			}
			if len(in.TaskUpdate.Outcome) > 2000 {
				return fail("invalid_input", "Keep the task outcome within 2000 bytes.")
			}
			if strings.TrimSpace(in.TaskUpdate.Outcome) == "" || strings.TrimSpace(in.TaskUpdate.Reason) == "" || strings.TrimSpace(in.TaskUpdate.Source) == "" {
				return fail("invalid_input", "Changing requirements needs an outcome, reason, and source.")
			}
			if err := validateAcceptance(in.TaskUpdate.Acceptance); err != nil {
				return err
			}
		}
		var err error
		repo, err = ResolveRepository(ctx, s.Root)
		if err != nil {
			return err
		}
		docs, err = s.captureDocuments(in.Documents)
		return err
	}, func(tx *sql.Tx) (Mutation, error) {
		task, err := taskInTransaction(ctx, tx, in.TaskID, in.ExpectedVersion)
		if err != nil {
			return Mutation{}, err
		}
		cp := Checkpoint{ID: NewID(), TaskID: task.ID, Sequence: task.Version, Summary: in.Summary, NextAction: in.NextAction, Steps: in.Steps, Blockers: in.Blockers, FailedAttempts: in.FailedAttempts, Evidence: in.Evidence, MemoryRefs: in.MemoryRefs, TaskUpdate: in.TaskUpdate, Context: repo, CreatedAt: repo.CapturedAt}
		if in.Reason != "" {
			cp.Summary += "\nReason: " + in.Reason
		}
		if in.TaskUpdate != nil {
			old := task
			cp.PreviousTask = &old
			task.Outcome, task.Acceptance = in.TaskUpdate.Outcome, in.TaskUpdate.Acceptance
		}
		for _, id := range in.MemoryRefs {
			var exists int
			if err := tx.QueryRowContext(ctx, "SELECT 1 FROM memories WHERE id = ?", id).Scan(&exists); err != nil {
				return Mutation{}, fail("invalid_input", "Checkpoint references an unknown memory.")
			}
		}
		for i := range docs {
			docs[i].TaskID, docs[i].CheckpointID = task.ID, cp.ID
			docs[i].InputDocuments, docs[i].CreatedAt = task.InputDocuments, cp.CreatedAt
			cp.DocumentIDs = append(cp.DocumentIDs, docs[i].ID)
		}
		if finish != nil {
			if err := validateFinish(task, *finish); err != nil {
				return Mutation{}, err
			}
			if task.Kind != "build" {
				hasDocument := false
				for _, doc := range docs {
					if doc.Kind == task.Kind {
						hasDocument = true
					}
				}
				if !hasDocument {
					return Mutation{}, fail("invalid_input", "Finishing research or a spec requires saving its final Markdown document in this request.")
				}
			}
			cp.Verification, cp.Review, cp.Limitations = finish.Verification, &Review{PolicyID: Rules().ID, ReviewInput: finish.Review}, finish.Limitations
			in.Status = "completed"
		}
		data, err := json.Marshal(cp)
		if err != nil {
			return Mutation{}, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO checkpoints(id,task_id,sequence,data) VALUES(?,?,?,?)", cp.ID, task.ID, cp.Sequence, string(data)); err != nil {
			return Mutation{}, err
		}
		for _, doc := range docs {
			data, err := json.Marshal(doc)
			if err != nil {
				return Mutation{}, err
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO documents(id,task_id,checkpoint_id,path,data) VALUES(?,?,?,?,?)", doc.ID, task.ID, cp.ID, doc.Path, string(data)); err != nil {
				return Mutation{}, err
			}
		}
		task.Version++
		task.Status, task.LatestCheckpointID, task.UpdatedAt = in.Status, cp.ID, cp.CreatedAt
		data, err = json.Marshal(task)
		if err != nil {
			return Mutation{}, err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE tasks SET status=?,version=?,data=? WHERE id=?", task.Status, task.Version, string(data), task.ID); err != nil {
			return Mutation{}, err
		}
		return Mutation{ID: task.ID, Version: task.Version, CheckpointID: cp.ID, DocumentIDs: cp.DocumentIDs, Status: task.Status}, nil
	})
}

func validateFinish(task Task, in FinishInput) error {
	if strings.TrimSpace(in.Review.Summary) == "" {
		return fail("invalid_input", "Finish needs a review summary.")
	}
	if len(in.Review.Unresolved) != 0 {
		return fail("invalid_input", "The review lists unresolved findings. Resolve them before finishing.")
	}
	for _, name := range []string{"single_use", "reuse", "necessity", "conventions", "minimality"} {
		check, exists := in.Review.Checks[name]
		if !exists || (check.Status != "pass" && check.Status != "not_applicable") || (check.Status == "not_applicable" && strings.TrimSpace(check.Reason) == "") {
			return fail("invalid_input", "Review check "+name+" must pass, or be not_applicable with a reason.")
		}
	}
	// A criterion passes when at least one passing verification lists it.
	passed := map[string]bool{}
	for _, criterion := range task.Acceptance {
		passed[criterion.ID] = false
	}
	seen := map[string]bool{}
	for _, item := range in.Verification {
		if item.ID == "" {
			return fail("invalid_input", "Every verification item needs an ID.")
		}
		if seen[item.ID] {
			return fail("invalid_input", "Verification ID "+item.ID+" is used more than once.")
		}
		seen[item.ID] = true
		if item.Source == "" || item.Description == "" {
			return fail("invalid_input", "Verification "+item.ID+" needs a description and a source.")
		}
		if item.Outcome != "pass" && item.Outcome != "fail" && item.Outcome != "not_run" {
			return fail("invalid_input", "Verification "+item.ID+" outcome must be pass, fail, or not_run.")
		}
		if item.ExitCode != nil && ((*item.ExitCode == 0) != (item.Outcome == "pass")) {
			return fail("invalid_input", "Verification "+item.ID+" outcome disagrees with its exit code.")
		}
		if item.Blocking && item.Outcome != "pass" {
			return fail("invalid_input", "Blocking verification "+item.ID+" has not passed.")
		}
		for _, id := range item.AcceptanceIDs {
			done, known := passed[id]
			if !known {
				return fail("invalid_input", "Verification "+item.ID+" names unknown acceptance criterion "+id+".")
			}
			passed[id] = done || item.Outcome == "pass"
		}
	}
	for _, criterion := range task.Acceptance {
		if !passed[criterion.ID] {
			return fail("invalid_input", "Acceptance criterion "+criterion.ID+" has no passing verification that lists it in acceptance_ids.")
		}
	}
	return nil
}
