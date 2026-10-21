package bearing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testFinish(t *testing.T, s *Store, taskID string, version int) FinishInput {
	t.Helper()
	checks := map[string]ReviewCheck{}
	for _, name := range []string{"single_use", "reuse", "necessity", "conventions", "minimality"} {
		checks[name] = ReviewCheck{Status: "pass"}
	}
	zero := 0
	return FinishInput{RequestID: NewID(), TaskID: taskID, ExpectedVersion: version, Summary: "Verified the intended result", Verification: []Evidence{{ID: "v1", Kind: "command", Description: "Run the relevant acceptance check", Source: "Local test run", Command: "go test ./...", Outcome: "pass", ExitCode: &zero, AcceptanceIDs: []string{"a1"}, Blocking: true}}, Review: ReviewInput{Summary: "Reviewed the smallest change and existing code", Checks: checks}}
}

func TestResearchSpecBuildLineage(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	research := testStart(t, s, "research")
	if err := os.WriteFile(filepath.Join(s.Root, "research.md"), []byte("# Findings\nUse a single durable store.\nSource: https://sqlite.org/atomiccommit.html\n"), 0600); err != nil {
		t.Fatal(err)
	}
	finish := testFinish(t, s, research.Mutation.ID, 1)
	finish.Documents = []DocumentInput{{Path: "research.md", Kind: "research", Summary: "SQLite transaction findings", Sources: []string{"https://sqlite.org/atomiccommit.html"}}}
	done, err := s.Finish(ctx, finish)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := s.Start(ctx, StartInput{RequestID: NewID(), Kind: "spec", Outcome: "Specify durable writes", Acceptance: []Criterion{{Description: "Define verifiable behavior"}}, InputDocuments: done.DocumentIDs, RelatedTaskID: research.Mutation.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "spec.md"), []byte("# Spec\nA failed write saves nothing.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	finish = testFinish(t, s, spec.Mutation.ID, 1)
	finish.Documents = []DocumentInput{{Path: "spec.md", Kind: "spec", Summary: "Atomic writes"}}
	specDone, err := s.Finish(ctx, finish)
	if err != nil {
		t.Fatal(err)
	}
	build, err := s.Start(ctx, StartInput{RequestID: NewID(), Outcome: "Implement atomic writes", Acceptance: []Criterion{{Description: "Failure rolls back"}}, InputDocuments: specDone.DocumentIDs})
	if err != nil {
		t.Fatal(err)
	}
	if build.Context.Documents[0].InputDocuments[0] != done.DocumentIDs[0] {
		t.Fatal("research lineage was lost")
	}
	if err := os.WriteFile(filepath.Join(s.Root, "spec.md"), []byte("# Different requirements"), 0600); err != nil {
		t.Fatal(err)
	}
	resumed, err := s.Resume(ctx, build.Mutation.ID)
	if err != nil || !strings.Contains(strings.Join(resumed.Warnings, " "), "changed") {
		t.Fatalf("changed spec not reported: %+v %v", resumed.Warnings, err)
	}
	if err := os.Remove(filepath.Join(s.Root, "spec.md")); err != nil {
		t.Fatal(err)
	}
	recalled, err := s.Recall(ctx, RecallInput{DocumentIDs: specDone.DocumentIDs})
	if err != nil || !strings.Contains(recalled.Documents[0].Body, "A failed write saves nothing") {
		t.Fatalf("document snapshot lost: %+v %v", recalled, err)
	}
}

func TestFinishCannotSkipEvidenceOrReview(t *testing.T) {
	// Each failure names the criterion, evidence, or review check at fault.
	for scenario, named := range map[string]string{"unlinked_criterion": "criterion a1", "unknown_criterion": "criterion a9", "only_not_run": "criterion a1", "failed_check": "Blocking verification v1", "duplicate_id": "ID v1", "command_without_text": "Command verification v1", "command_without_exit_code": "Command verification v1", "unreviewed": "check reuse", "unresolved": "unresolved", "exit_mismatch": "Verification v1", "research_without_document": "document"} {
		t.Run(scenario, func(t *testing.T) {
			s := testStore(t)
			kind := "build"
			if scenario == "research_without_document" {
				kind = "research"
			}
			started := testStart(t, s, kind)
			in := testFinish(t, s, started.Mutation.ID, 1)
			switch scenario {
			case "unlinked_criterion":
				in.Verification[0].AcceptanceIDs = nil
			case "unknown_criterion":
				in.Verification[0].AcceptanceIDs = []string{"a1", "a9"}
			case "only_not_run":
				in.Verification[0].Outcome, in.Verification[0].ExitCode, in.Verification[0].Blocking = "not_run", nil, false
			case "failed_check":
				one := 1
				in.Verification[0].Outcome, in.Verification[0].ExitCode = "fail", &one
			case "duplicate_id":
				in.Verification = append(in.Verification, in.Verification[0])
			case "command_without_text":
				in.Verification[0].Command = " "
			case "command_without_exit_code":
				in.Verification[0].ExitCode = nil
			case "unreviewed":
				delete(in.Review.Checks, "reuse")
			case "unresolved":
				in.Review.Unresolved = []string{"known defect"}
			case "exit_mismatch":
				one := 1
				in.Verification[0].ExitCode = &one
			}
			_, err := s.Finish(context.Background(), in)
			if err == nil || !strings.Contains(err.Error(), named) {
				t.Fatalf("finish error should name %q: %v", named, err)
			}
			resumed, err := s.Resume(context.Background(), started.Mutation.ID)
			if err != nil || resumed.Task.Status != "active" || resumed.Task.Version != 1 {
				t.Fatalf("failed finish mutated state: %+v %v", resumed.Task, err)
			}
		})
	}
}

func TestRequirementHistoryAndExplicitTaskSelection(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	testStart(t, s, "build")
	result, err := s.Resume(ctx, "")
	if err != nil || result.Task != nil || len(result.Candidates) != 2 {
		t.Fatalf("task guessed: %+v %v", result, err)
	}
	cp, err := s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "User clarified scope", NextAction: "Verify new requirement", TaskUpdate: &TaskUpdate{Outcome: "New outcome", Acceptance: []Criterion{{Description: "New criterion"}}, Reason: "User correction", Source: "User message"}})
	if err != nil {
		t.Fatal(err)
	}
	history, err := s.History(ctx, started.Mutation.ID, 0)
	if err != nil || history[0].PreviousTask.Outcome != "Make durable progress" {
		t.Fatalf("requirement history: %v", err)
	}
	in := testFinish(t, s, started.Mutation.ID, cp.Version)
	completed, err := s.Finish(ctx, in)
	if err != nil || completed.Status != "completed" {
		t.Fatalf("valid finish: %+v %v", completed, err)
	}
	final, err := s.History(ctx, started.Mutation.ID, 0)
	if err != nil || final[0].Context.Path != s.Root || final[0].Context.CapturedAt == "" {
		t.Fatalf("finish did not record Bearing's checkout context: %v", err)
	}
	if final[0].Review.PolicyID != Rules().ID {
		t.Fatalf("finish did not record the policy ID: %q", final[0].Review.PolicyID)
	}
	// Finishes saved before results were derived kept explicit acceptance_results.
	if _, err := s.DB.Exec(`UPDATE checkpoints SET data=json_set(data,'$.acceptance_results',json('[{"id":"a1","status":"pass","evidence_ids":["v1"]}]')) WHERE id=?`, completed.CheckpointID); err != nil {
		t.Fatal(err)
	}
	legacy, err := s.History(ctx, started.Mutation.ID, 0)
	if err != nil || len(legacy[0].AcceptanceResults) != 1 || legacy[0].Review.PolicyID != Rules().ID {
		t.Fatalf("older finish record is unreadable: %+v %v", legacy[0], err)
	}
	replayed, err := s.Finish(ctx, in)
	if err != nil || !replayed.Replayed {
		t.Fatalf("finish retry: %v", err)
	}
	if _, err := s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: completed.Version, Summary: "Reopen", NextAction: "Verify"}); err == nil {
		t.Fatal("completed task changed")
	}
}

func TestFinishBindsEvidenceToWorkingTree(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(s.Root, path)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.Root, path), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("app.go", "package app\n")
	unchanged, changed := testStart(t, s, "build"), testStart(t, s, "build")
	for _, task := range []StartResult{unchanged, changed} {
		in := testFinish(t, s, task.Mutation.ID, 1)
		if _, err := s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: task.Mutation.ID, ExpectedVersion: 1, Summary: "Tests pass", NextAction: "Finish", Evidence: in.Verification}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Finish(ctx, testFinish(t, s, unchanged.Mutation.ID, 2)); err != nil {
		t.Fatalf("evidence from the same working tree was rejected: %v", err)
	}
	write("app.go", "package app\n\n// Edited after the passing check.\n")
	write("app_test.go", "package app\n")
	write("tests/smoke.sh", "exit 0\n")
	write("docs/test.md", "Not a test path.\n")
	in := testFinish(t, s, changed.Mutation.ID, 2)
	if _, err := s.Finish(ctx, in); err == nil || !strings.Contains(err.Error(), "v1") || !strings.Contains(err.Error(), "checkpoint 1") {
		t.Fatalf("stale evidence accepted or not named: %v", err)
	}
	in.RequestID, in.Verification[0].ID = NewID(), "v2"
	done, err := s.Finish(ctx, in)
	if err != nil {
		t.Fatalf("rerun evidence rejected: %v", err)
	}
	if strings.Join(done.ChangedTestFiles, ",") != "app_test.go,tests/smoke.sh" {
		t.Fatalf("changed test files: %v", done.ChangedTestFiles)
	}
	history, err := s.History(ctx, changed.Mutation.ID, 0)
	if err != nil || history[0].Context.Tree == "" || history[0].Context.Tree == history[1].Context.Tree {
		t.Fatalf("checkpoint and finish trees not recorded: %v", err)
	}
}

func TestDocumentPathBoundaries(t *testing.T) {
	s := testStore(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("Private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.Root, "escape.md")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{outside, "../outside.md", "escape.md", "code.go"} {
		if _, err := s.captureDocuments([]DocumentInput{{Path: path, Kind: "spec", Summary: "Invalid"}}); err == nil {
			t.Fatalf("unsafe document path accepted: %s", path)
		}
	}
}
