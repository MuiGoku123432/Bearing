package bearing

import (
	"context"
	"strings"
	"testing"
)

func TestScopedMemoryLifecycle(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	var saved []Mutation
	for _, anchor := range []string{"api.go", "ui.go"} {
		out, err := s.Remember(ctx, RememberInput{RequestID: NewID(), Kind: "constraint", Claim: "Use SQLite transactions for durable progress", Basis: "user", Source: "User instruction", Scope: "project", Anchors: []Anchor{{Kind: "path", Value: anchor}}})
		if err != nil {
			t.Fatal(err)
		}
		saved = append(saved, out)
	}
	result, err := s.Recall(ctx, RecallInput{Query: "SQLite OR nonsense*", Anchors: []Anchor{{Kind: "path", Value: "api.go"}}})
	if err != nil || len(result.Memories) != 1 || result.Memories[0].ID != saved[0].ID {
		t.Fatalf("anchor scope leaked: %+v %v", result, err)
	}
	other := testStore(t)
	empty, err := other.Recall(ctx, RecallInput{Query: "SQLite"})
	if err != nil || len(empty.Memories) != 0 {
		t.Fatalf("repository scope leaked: %+v %v", empty, err)
	}
	replacement, err := s.Remember(ctx, RememberInput{RequestID: NewID(), Action: "supersede", ID: saved[0].ID, ExpectedVersion: 1, Kind: "decision", Claim: "Use SQLite immediate transactions", Basis: "observed", Source: "Concurrency experiment", Reason: "New evidence", Anchors: []Anchor{{Kind: "path", Value: "api.go"}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = s.Recall(ctx, RecallInput{Query: "SQLite", Anchors: []Anchor{{Kind: "path", Value: "api.go"}}})
	if err != nil || len(result.Memories) != 1 || result.Memories[0].Supersedes != saved[0].ID {
		t.Fatalf("old memory still active: %+v %v", result, err)
	}
	old, err := s.Recall(ctx, RecallInput{IDs: []string{saved[0].ID}})
	if err != nil || old.Memories[0].Status != "superseded" {
		t.Fatalf("memory audit lost: %+v %v", old, err)
	}
	_, err = s.Remember(ctx, RememberInput{RequestID: NewID(), Action: "retract", ID: replacement.ID, ExpectedVersion: 1, Reason: "No longer relevant"})
	if err != nil {
		t.Fatal(err)
	}
	result, err = s.Recall(ctx, RecallInput{Query: "SQLite", Anchors: []Anchor{{Kind: "path", Value: "api.go"}}})
	if err != nil || len(result.Memories) != 0 {
		t.Fatalf("retracted memory recalled: %+v %v", result, err)
	}
}

func TestRecallExplicitIDsBeyondLimit(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	var ids []string
	for i := 0; i < 8; i++ {
		out, err := s.Remember(ctx, RememberInput{RequestID: NewID(), Kind: "decision", Claim: "Keep explicit lookups exact", Basis: "user", Source: "Test"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, out.ID)
	}
	result, err := s.Recall(ctx, RecallInput{IDs: append(ids, ids[0])})
	if err != nil || len(result.Memories) != 5 || result.Omitted != 3 {
		t.Fatalf("existing IDs reported missing: count=%d omitted=%d err=%v", len(result.Memories), result.Omitted, err)
	}
	if _, err := s.Recall(ctx, RecallInput{IDs: []string{ids[0], "missing"}}); ErrorDetails(err).Code != "not_found" {
		t.Fatalf("missing ID not reported: %v", err)
	}
}

func TestRecallBudgetAndLiteralFTS(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	for i := 0; i < 6; i++ {
		_, err := s.Remember(ctx, RememberInput{RequestID: NewID(), Kind: "lesson", Claim: "Checkpoint " + strings.Repeat("useful context ", 150), Basis: "observed", Source: "Test observation"})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := s.Recall(ctx, RecallInput{Query: "checkpoint", BudgetBytes: 3000, Limit: 5})
	if err != nil || len(result.Memories) != 1 || result.Omitted == 0 {
		t.Fatalf("budget not applied: count=%d omitted=%d err=%v", len(result.Memories), result.Omitted, err)
	}
	for _, query := range []string{"\"", "NEAR(()", "checkpoint' OR 1=1 --", "x:y*"} {
		if _, err := s.Recall(ctx, RecallInput{Query: query}); err != nil {
			t.Fatalf("plain query interpreted as syntax: %q %v", query, err)
		}
	}
}
