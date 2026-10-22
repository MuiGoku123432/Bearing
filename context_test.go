package bearing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextRetainsDocumentsAndHonorsBudget(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "spec")
	if err := os.WriteFile(filepath.Join(s.Root, "spec.md"), []byte("# Durable requirements"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Saved spec", NextAction: "Review findings", Documents: []DocumentInput{{Path: "spec.md", Kind: "spec", Summary: "Requirements"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: first.Version, Summary: "Reviewed findings", NextAction: "Finish the spec", FailedAttempts: []string{strings.Repeat("Attempt failed. ", 1000)}})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := s.Resume(ctx, started.Mutation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed.Documents) != 1 || resumed.Documents[0].ID != first.DocumentIDs[0] {
		t.Fatal("a later checkpoint hid the saved document")
	}
	data, err := json.Marshal(resumed)
	if err != nil || len(data) > 32000 {
		t.Fatalf("context budget exceeded: %d %v", len(data), err)
	}
	if resumed.Omitted == 0 || resumed.Policy.Text != Rules().Text || resumed.Next != "Finish the spec" {
		t.Fatal("context trimming lost required guidance")
	}
	if strings.Contains(strings.Join(resumed.Warnings, " "), "No memories matched") {
		t.Fatal("resume repeated the explicit-recall hint")
	}
	history, err := s.History(ctx, started.Mutation.ID, 0)
	if err != nil || len(history[0].FailedAttempts[0]) < 10000 {
		t.Fatal("trimming destroyed the durable record")
	}
}

func TestUnicodeTaskCandidates(t *testing.T) {
	s := testStore(t)
	for i := 0; i < 2; i++ {
		_, err := s.Start(context.Background(), StartInput{RequestID: NewID(), Outcome: strings.Repeat("🧭", 80), Acceptance: []Criterion{{Description: "Unicode survives"}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := s.Resume(context.Background(), "")
	if err != nil || len(result.Candidates) != 2 {
		t.Fatalf("candidate selection: %v", err)
	}
}

func TestPlanContextKeepsPolicyWithinBudget(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	outcome := strings.Repeat("o", 2000)
	acceptance := []Criterion{}
	for i := 0; i < 30; i++ {
		acceptance = append(acceptance, Criterion{Description: strings.Repeat("c", 400)})
	}
	started, err := s.Start(ctx, StartInput{RequestID: NewID(), Kind: "plan", Outcome: outcome, Acceptance: acceptance})
	if err != nil {
		t.Fatal(err)
	}
	// Fill the checkpoint to just under the 16384-byte input limit.
	cp := CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Drafted the plan", NextAction: "Ask the user to approve the plan"}
	for data, _ := json.Marshal(cp); len(data) < 16000; data, _ = json.Marshal(cp) {
		cp.Steps = append(cp.Steps, strings.Repeat("s", 300))
	}
	if _, err := s.Checkpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}
	resumed, err := s.Resume(ctx, started.Mutation.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(resumed)
	if err != nil || len(data) > 32000 {
		t.Fatalf("plan context budget exceeded: %d %v", len(data), err)
	}
	if resumed.Policy.Text != Rules().Text || resumed.Policy.Planning != Rules().Planning || resumed.Task.Outcome != outcome || len(resumed.Task.Acceptance) != 30 {
		t.Fatal("plan context trimming lost policy, planning standards, or requirements")
	}
}
