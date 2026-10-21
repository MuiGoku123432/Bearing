package bearing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testRepo(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	cmd := exec.Command("git", "init", "-q", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Init(context.Background(), testRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testStart(t *testing.T, s *Store, kind string) StartResult {
	t.Helper()
	out, err := s.Start(context.Background(), StartInput{RequestID: NewID(), Kind: kind, Outcome: "Make durable progress", Acceptance: []Criterion{{Description: "The intended result is verified"}}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRestartRetryAndConflict(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	input := StartInput{RequestID: "start-once", Outcome: "Keep requirements", Acceptance: []Criterion{{Description: "Survives restart"}}}
	started, err := s.Start(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Acceptance[0].ID != "" {
		t.Fatal("Start mutated caller input; retries would get a different digest")
	}
	root := s.Root
	s.Close()
	s, err = Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	retried, err := s.Start(ctx, input)
	if err != nil || !retried.Mutation.Replayed || retried.Mutation.ID != started.Mutation.ID {
		t.Fatalf("retry: %+v %v", retried, err)
	}
	input.Outcome = "Different write"
	if _, err := s.Start(ctx, input); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("changed retry accepted: %v", err)
	}
	cp, err := s.Checkpoint(ctx, CheckpointInput{RequestID: "cp", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Saved useful work", NextAction: "Verify the result"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Checkpoint(ctx, CheckpointInput{RequestID: "stale", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Overwrite", NextAction: "Verify"})
	if ErrorDetails(err).Code != "conflict" || ErrorDetails(err).CurrentVersion != 2 {
		t.Fatalf("stale update: %v", err)
	}
	resumed, err := s.Resume(ctx, "")
	if err != nil || resumed.Checkpoint.ID != cp.CheckpointID || resumed.Next != "Verify the result" {
		t.Fatalf("resume: %+v %v", resumed, err)
	}
	history, err := s.History(ctx, started.Mutation.ID, 0)
	if err != nil || len(history) != 1 {
		t.Fatalf("history: %v %v", history, err)
	}
}

func TestCompetingConnections(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	other, err := Open(ctx, s.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	gate := make(chan struct{})
	errors := make(chan error, 2)
	var group sync.WaitGroup
	for _, store := range []*Store{s, other} {
		group.Add(1)
		go func(store *Store) {
			defer group.Done()
			<-gate
			_, err := store.Checkpoint(ctx, CheckpointInput{RequestID: NewID(), TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Writer progress", NextAction: "Verify"})
			errors <- err
		}(store)
	}
	close(gate)
	group.Wait()
	close(errors)
	success, conflict := 0, 0
	for err := range errors {
		if err == nil {
			success++
		} else if ErrorDetails(err).Code == "conflict" {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success %d conflict %d", success, conflict)
	}
}

func TestFailureRollsBackWholeCheckpoint(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	if _, err := s.DB.Exec("CREATE TRIGGER fail_document BEFORE INSERT ON documents BEGIN SELECT RAISE(ABORT,'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "spec.md"), []byte("# Spec\nKeep the requirement."), 0600); err != nil {
		t.Fatal(err)
	}
	in := CheckpointInput{RequestID: "atomic", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Save with document", NextAction: "Review spec", Documents: []DocumentInput{{Path: "spec.md", Kind: "spec", Summary: "Requirement"}}}
	if _, err := s.Checkpoint(ctx, in); err == nil {
		t.Fatal("injected write failure did not fail")
	}
	for _, table := range []string{"checkpoints", "documents"} {
		var count int
		if err := s.DB.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial %s: %d %v", table, count, err)
		}
	}
	resumed, err := s.Resume(ctx, started.Mutation.ID)
	if err != nil || resumed.Task.Version != 1 {
		t.Fatalf("task advanced after rollback: %v", err)
	}
	if _, err := s.DB.Exec("DROP TRIGGER fail_document"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(ctx, in); err != nil {
		t.Fatalf("rolled back receipt prevented retry: %v", err)
	}
	if err := os.Remove(filepath.Join(s.Root, "spec.md")); err != nil {
		t.Fatal(err)
	}
	replay, err := s.Checkpoint(ctx, in)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay unnecessarily reopened missing document: %v", err)
	}
}

func TestInitializationAndStrictInputs(t *testing.T) {
	ctx := context.Background()
	root := testRepo(t)
	if _, err := Open(ctx, root); ErrorDetails(err).Code != "not_initialized" {
		t.Fatalf("open created state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".bearing")); !os.IsNotExist(err) {
		t.Fatal("read created state directory")
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Init(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Init(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil || string(data) != "existing" {
		t.Fatalf("tracked ignore rules changed: %q %v", data, err)
	}
	status, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil || string(status) != "?? .gitignore\n" {
		t.Fatalf("state is visible to Git: %q %v", status, err)
	}
	for _, input := range []string{"null", "[]", "{\"unknown\":true}", "{} {}", strings.Repeat(" ", 65537)} {
		var target StartInput
		if err := DecodeInput([]byte(input), &target); err == nil {
			t.Fatalf("invalid JSON accepted: %.30s", input)
		}
	}
	if _, err := s.DB.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if newer, err := Open(ctx, root); err == nil {
		newer.Close()
		t.Fatal("newer schema accepted")
	}
}
