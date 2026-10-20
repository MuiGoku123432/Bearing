package bearing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitWorktreeCarriesSavedDocumentsAndIsolatesState(t *testing.T) {
	ctx := context.Background()
	s := gitTestStore(t)
	research := testStart(t, s, "research")
	for name, body := range map[string]string{"research.md": "# Research\nKeep the source finding.", "spec.md": "# Spec\nImplement the accepted requirement."} {
		if err := os.WriteFile(filepath.Join(s.Root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.Checkpoint(ctx, CheckpointInput{RequestID: "save-docs", TaskID: research.Mutation.ID, ExpectedVersion: 1, Summary: "Saved findings and spec", NextAction: "Build in a worktree", Documents: []DocumentInput{{Path: "research.md", Kind: "research", Summary: "Finding"}, {Path: "spec.md", Kind: "spec", Summary: "Accepted requirement"}}})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := s.Resume(ctx, research.Mutation.ID)
	if err != nil || len(saved.Documents) != 2 {
		t.Fatalf("saved documents: %+v %v", saved, err)
	}
	ids := []string{saved.Documents[0].ID, saved.Documents[1].ID}
	build, err := s.Start(ctx, StartInput{RequestID: "build-from-docs", Outcome: "Implement the saved spec", InputDocuments: ids, Acceptance: []Criterion{{Description: "Preserve the saved lineage"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remember(ctx, RememberInput{RequestID: "portable-memory", Kind: "decision", Claim: "Preserve portable design", Basis: "user", Source: "Test request", Scope: "project"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remember(ctx, RememberInput{RequestID: "local-memory", Kind: "lesson", Claim: "Portable checkout observation", Basis: "observed", Source: "Test source checkout", Scope: "checkout"}); err != nil {
		t.Fatal(err)
	}
	state, err := s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "worktree with spaces")
	in := GitInput{Operation: "worktree", Branch: "feat/worktree", Path: path, ExpectedHead: state.Repository.Head, ExpectedBranch: state.Repository.Branch, DryRun: true}
	if _, err := s.Git(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("dry run created worktree")
	}
	in.DryRun = false
	result, err := s.Git(ctx, in)
	if err != nil || !result.StateCopied || result.Destination == nil || result.Destination.Path != path || result.RecoveryBackup != "" {
		t.Fatalf("worktree: %+v %v", result, err)
	}
	if testGit(t, s.Root, "branch", "--show-current") != state.Repository.Branch {
		t.Fatal("source branch changed")
	}
	if _, err := os.Stat(filepath.Join(path, "spec.md")); !os.IsNotExist(err) {
		t.Fatal("copied unsaved source files")
	}
	other, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	resumed, err := other.Resume(ctx, build.Mutation.ID)
	if err != nil || len(resumed.Task.InputDocuments) != 2 || len(resumed.Warnings) == 0 {
		t.Fatalf("resume lineage/context: %+v %v", resumed, err)
	}
	documents, err := other.Recall(ctx, RecallInput{DocumentIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(documents)
	if !strings.Contains(string(data), "Keep the source finding") || !strings.Contains(string(data), "accepted requirement") {
		t.Fatal("document snapshot bodies lost")
	}
	memories, err := other.Recall(ctx, RecallInput{Query: "portable"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(memories)
	if !strings.Contains(string(data), "Preserve portable design") || strings.Contains(string(data), "Portable checkout observation") {
		t.Fatal("memory checkout filtering changed")
	}
	if _, err := other.Checkpoint(ctx, CheckpointInput{RequestID: "destination-progress", TaskID: build.Mutation.ID, ExpectedVersion: 1, Summary: "Worktree made progress", NextAction: "Verify"}); err != nil {
		t.Fatal(err)
	}
	source, err := s.Resume(ctx, build.Mutation.ID)
	if err != nil || source.Task.Version != 1 {
		t.Fatal("worktree changes leaked into source store")
	}
	state, err = s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil || len(state.Worktrees) != 2 {
		t.Fatalf("worktree listing: %+v %v", state, err)
	}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("repeated worktree creation accepted: %v", err)
	}
}

func TestGitRefusesRedirectedEnvironmentAndOccupiedDestination(t *testing.T) {
	ctx := context.Background()
	s := gitTestStore(t)
	state, err := s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "another-index"))
	if _, err := s.Git(ctx, GitInput{Operation: "inspect"}); ErrorDetails(err).Code != "invalid_input" {
		t.Fatalf("redirect accepted: %v", err)
	}
	t.Setenv("GIT_INDEX_FILE", "")
	in := GitInput{Operation: "worktree", Branch: "feat/occupied", Path: t.TempDir(), ExpectedHead: state.Repository.Head, ExpectedBranch: state.Repository.Branch}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("occupied destination accepted: %v", err)
	}
	in.Path = filepath.Join(s.Root, "nested")
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "invalid_input" {
		t.Fatalf("nested destination accepted: %v", err)
	}
	if testGit(t, s.Root, "branch", "--list", in.Branch) != "" {
		t.Fatal("failed request created a branch")
	}
}
