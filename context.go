package bearing

import (
	"context"
	"encoding/json"
)

func (s *Store) Status(ctx context.Context) ([]Task, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	rows, err := s.DB.QueryContext(ctx, "SELECT data FROM tasks WHERE status IN ('active','blocked') ORDER BY json_extract(data,'$.updated_at') DESC,id LIMIT 21")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []Task{}
	for rows.Next() {
		var data string
		var task Task
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) Resume(ctx context.Context, id string) (Context, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	out := Context{Policy: Rules()}
	out.Policy.Workflow = ""
	var err error
	out.Repository, err = ResolveRepository(ctx, s.Root)
	if err != nil {
		return out, err
	}
	if id == "" {
		candidates, err := s.Status(ctx)
		if err != nil {
			return out, err
		}
		if len(candidates) == 0 {
			out.Next = "Start a task with an outcome and acceptance criteria."
			return out, nil
		}
		if len(candidates) != 1 {
			out.Next = "Choose a task ID and resume it. Bearing does not guess between active tasks."
			if len(candidates) > 10 {
				out.Omitted += len(candidates) - 10
				candidates = candidates[:10]
			}
			for i := range candidates {
				candidates[i].Acceptance, candidates[i].InputDocuments = nil, nil
				candidates[i].CreatedContext = Repository{}
				if text := []rune(candidates[i].Outcome); len(text) > 100 {
					candidates[i].Outcome = string(text[:100]) + "…"
				}
			}
			out.Candidates = candidates
			return out, nil
		}
		id = candidates[0].ID
	}
	var task Task
	if err := s.loadJSON(ctx, "SELECT data FROM tasks WHERE id=?", id, &task); err != nil {
		return out, err
	}
	out.Task = &task
	context := task.CreatedContext
	docIDs := append([]string{}, task.InputDocuments...)
	if task.LatestCheckpointID != "" {
		var cp Checkpoint
		if err := s.loadJSON(ctx, "SELECT data FROM checkpoints WHERE id=?", task.LatestCheckpointID, &cp); err != nil {
			return out, err
		}
		context = cp.Context
		docIDs = append(docIDs, cp.DocumentIDs...)
		out.Next = cp.NextAction
		data, _ := json.Marshal(cp)
		if len(data) > 14000 {
			cp.Evidence, cp.Verification, cp.PreviousTask, cp.TaskUpdate = nil, nil, nil, nil
			cp.Steps, cp.FailedAttempts, cp.AcceptanceResults = nil, nil, nil
			out.Omitted++
			out.Warnings = append(out.Warnings, "Checkpoint details omitted from the context budget; use history for the full record.")
		}
		out.Checkpoint = &cp
	}
	if context.Head != out.Repository.Head || context.Path != out.Repository.Path {
		out.Warnings = append(out.Warnings, "Checkout or commit changed since this task was saved. Revalidate code and verification evidence.")
	}
	if context.Dirty || out.Repository.Dirty {
		out.Warnings = append(out.Warnings, "Working files have uncommitted changes. Saved evidence is not proof of their current state.")
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT d.id FROM documents d JOIN checkpoints c ON c.id=d.checkpoint_id WHERE d.task_id=? AND NOT EXISTS(SELECT 1 FROM documents newer JOIN checkpoints nc ON nc.id=newer.checkpoint_id WHERE newer.task_id=d.task_id AND newer.path=d.path AND nc.sequence>c.sequence) ORDER BY c.sequence DESC,d.id LIMIT 17", task.ID)
	if err != nil {
		return out, err
	}
	count := 0
	for rows.Next() {
		var docID string
		if err := rows.Scan(&docID); err != nil {
			rows.Close()
			return out, err
		}
		count++
		if count <= 16 {
			docIDs = append(docIDs, docID)
		} else {
			out.Omitted++
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, docID := range docIDs {
		if seen[docID] {
			continue
		}
		seen[docID] = true
		doc, err := s.document(ctx, docID)
		if err != nil {
			return out, err
		}
		doc.Body = ""
		out.Documents = append(out.Documents, doc)
		if warning := s.documentWarning(doc); warning != "" {
			out.Warnings = append(out.Warnings, warning)
		}
	}
	memories, err := s.Recall(ctx, RecallInput{Query: task.Outcome, Limit: 3, BudgetBytes: 6000})
	if err != nil {
		return out, err
	}
	out.Memories, out.Omitted = memories.Memories, out.Omitted+memories.Omitted
	out.Warnings = append(out.Warnings, memories.Warnings...)
	if out.Next == "" {
		out.Next = "Use the saved requirements and native Gortex tools to choose the next concrete action."
	}
	if task.Status == "completed" || task.Status == "cancelled" {
		out.Next = "This task is closed. Start a related task if more work is needed."
	}
	if len(out.Warnings) > 10 {
		out.Omitted += len(out.Warnings) - 10
		out.Warnings = out.Warnings[:10]
	}
	for {
		data, err := json.Marshal(out)
		if err != nil {
			return out, err
		}
		if len(data) <= 31800 {
			break
		}
		out.Omitted++
		if len(out.Memories) > 0 {
			out.Memories = out.Memories[:len(out.Memories)-1]
			continue
		}
		if len(out.Documents) > 0 {
			out.Documents = out.Documents[:len(out.Documents)-1]
			continue
		}
		if out.Checkpoint != nil && (len(out.Checkpoint.Blockers) > 0 || out.Checkpoint.Review != nil || out.Checkpoint.Evidence != nil || out.Checkpoint.TaskUpdate != nil || out.Checkpoint.Steps != nil || out.Checkpoint.Limitations != nil || out.Checkpoint.Verification != nil || out.Checkpoint.FailedAttempts != nil) {
			cp := out.Checkpoint
			out.Checkpoint = &Checkpoint{ID: cp.ID, TaskID: cp.TaskID, Sequence: cp.Sequence, Summary: cp.Summary, NextAction: cp.NextAction, Context: cp.Context, CreatedAt: cp.CreatedAt}
			continue
		}
		return out, fail("invalid_input", "Saved context exceeds the response budget. Read task history and documents explicitly.")
	}
	if out.Omitted > 0 {
		out.Warnings = append(out.Warnings, "Context was shortened. Full checkpoints are available through history; saved documents through recall.")
	}
	return out, nil
}

func (s *Store) History(ctx context.Context, taskID string, before int) ([]Checkpoint, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	var task Task
	if err := s.loadJSON(ctx, "SELECT data FROM tasks WHERE id=?", taskID, &task); err != nil {
		return nil, err
	}
	if before == 0 {
		before = task.Version + 1
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT data FROM checkpoints WHERE task_id=? AND sequence<? ORDER BY sequence DESC LIMIT 10", taskID, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Checkpoint{}
	for rows.Next() {
		var data string
		var cp Checkpoint
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(data), &cp); err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, rows.Err()
}
