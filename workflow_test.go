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
	return FinishInput{RequestID: NewID(), TaskID: taskID, ExpectedVersion: version, Summary: "Verified the intended result", AcceptanceResults: []AcceptanceResult{{ID: "a1", Status: "pass", EvidenceIDs: []string{"v1"}}}, Verification: []Evidence{{ID: "v1", Kind: "test", Description: "Run the relevant acceptance check", Source: "go test ./...", Outcome: "pass", ExitCode: &zero, AcceptanceIDs: []string{"a1"}, Blocking: true}}, Review: Review{PolicyID: Rules().ID, Summary: "Reviewed the smallest change and existing code", Checks: checks}}
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
	for _, scenario := range []string{"no_results", "bad_evidence", "failed_check", "unreviewed", "unresolved", "old_policy", "exit_mismatch", "research_without_document"} {
		t.Run(scenario, func(t *testing.T) {
			s := testStore(t)
			kind := "build"
			if scenario == "research_without_document" {
				kind = "research"
			}
			started := testStart(t, s, kind)
			in := testFinish(t, s, started.Mutation.ID, 1)
			switch scenario {
			case "no_results":
				in.AcceptanceResults = nil
			case "bad_evidence":
				in.AcceptanceResults[0].EvidenceIDs = []string{"missing"}
			case "failed_check":
				in.Verification[0].Outcome = "fail"
			case "unreviewed":
				delete(in.Review.Checks, "reuse")
			case "unresolved":
				in.Review.Unresolved = []string{"known defect"}
			case "old_policy":
				in.Review.PolicyID = "old"
			case "exit_mismatch":
				one := 1
				in.Verification[0].ExitCode = &one
			}
			if _, err := s.Finish(context.Background(), in); err == nil {
				t.Fatal("incomplete task was marked completed")
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
	replayed, err := s.Finish(ctx, in)
	if err != nil || !replayed.Replayed {
		t.Fatalf("finish retry: %v", err)
	}
	if _, err := s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: completed.Version, Summary: "Reopen", NextAction: "Verify"}); err == nil {
		t.Fatal("completed task changed")
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
