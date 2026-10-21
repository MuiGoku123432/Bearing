package bearing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func testGit(t *testing.T, path string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", path}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitTestStore(t *testing.T) *Store {
	t.Helper()
	s := testStore(t)
	for _, args := range [][]string{
		{"config", "user.name", "Bearing Test"}, {"config", "user.email", "bearing@example.invalid"},
		{"config", "core.hooksPath", "/dev/null"}, {"config", "commit.gpgsign", "false"},
		{"commit", "--allow-empty", "-qm", "chore: initialize fixture"},
	} {
		testGit(t, s.Root, args...)
	}
	return s
}

func TestGitBranchesPreserveChangesAndCheckContext(t *testing.T) {
	ctx := context.Background()
	s := gitTestStore(t)
	state, err := s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	in := GitInput{Operation: "branch", Branch: "feat/memory", ExpectedHead: state.Repository.Head, ExpectedBranch: state.Repository.Branch, DryRun: true}
	if _, err := s.Git(ctx, in); err != nil {
		t.Fatal(err)
	}
	if got := testGit(t, s.Root, "branch", "--show-current"); got != state.Repository.Branch {
		t.Fatal("dry run switched branches")
	}
	in.ExpectedBranch = "wrong"
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("stale branch: %v", err)
	}
	in.ExpectedBranch = state.Repository.Branch
	if err := os.WriteFile(filepath.Join(s.Root, "user-work.txt"), []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	in.DryRun = false
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("dirty branch switched: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(s.Root, "user-work.txt"))
	if string(data) != "keep me" {
		t.Fatal("user changes lost")
	}
	if err := os.Remove(filepath.Join(s.Root, "user-work.txt")); err != nil {
		t.Fatal(err)
	}
	created, err := s.Git(ctx, in)
	if err != nil || created.Repository.Branch != in.Branch {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("stale retry: %v", err)
	}
	_, err = s.Git(ctx, GitInput{Operation: "switch", Branch: state.Repository.Branch, ExpectedHead: created.Repository.Head, ExpectedBranch: created.Repository.Branch})
	if err != nil {
		t.Fatal(err)
	}
	in.Branch = "--force"
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "invalid_input" {
		t.Fatalf("option branch accepted: %v", err)
	}
	mergeHead := testGit(t, s.Root, "rev-parse", "--git-path", "MERGE_HEAD")
	if !filepath.IsAbs(mergeHead) {
		mergeHead = filepath.Join(s.Root, mergeHead)
	}
	if err := os.WriteFile(mergeHead, []byte(state.Repository.Head), 0600); err != nil {
		t.Fatal(err)
	}
	in.Branch = "blocked-by-merge"
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("merge state ignored: %v", err)
	}
}

func TestGitCommitUsesReviewedIndexAndHooks(t *testing.T) {
	ctx := context.Background()
	s := gitTestStore(t)
	if err := os.WriteFile(filepath.Join(s.Root, "staged.txt"), []byte("reviewed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "unstaged.txt"), []byte("user work"), 0600); err != nil {
		t.Fatal(err)
	}
	testGit(t, s.Root, "add", "staged.txt")
	state, err := s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil || !strings.Contains(state.StagedSummary, "staged.txt") {
		t.Fatalf("inspect: %+v %v", state, err)
	}
	in := GitInput{Operation: "commit", ExpectedHead: state.Repository.Head, ExpectedBranch: state.Repository.Branch, ExpectedIndex: "stale", Message: "feat(memory): retain checkpoints"}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("stale index accepted: %v", err)
	}
	in.ExpectedIndex = state.IndexTree
	in.DryRun = true
	if _, err := s.Git(ctx, in); err != nil {
		t.Fatal(err)
	}
	if testGit(t, s.Root, "rev-parse", "HEAD") != state.Repository.Head {
		t.Fatal("dry run committed")
	}
	hooks := filepath.Join(s.Root, ".bearing", "hooks")
	if err := os.Mkdir(hooks, 0700); err != nil {
		t.Fatal(err)
	}
	testGit(t, s.Root, "config", "core.hooksPath", hooks)
	hook := filepath.Join(hooks, "commit-msg")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nprintf ran > .bearing/hook-ran\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	in.DryRun = false
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "outcome_unknown" {
		t.Fatalf("hook failure bypassed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, ".bearing", "hook-ran")); err != nil {
		t.Fatal("hook did not run")
	}
	if testGit(t, s.Root, "rev-parse", "HEAD") != state.Repository.Head {
		t.Fatal("failed hook advanced HEAD")
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	result, err := s.Git(ctx, in)
	if err != nil || len(result.Warnings) != 0 || result.Repository.Head == state.Repository.Head {
		t.Fatalf("commit: %+v %v", result, err)
	}
	if testGit(t, s.Root, "show", "HEAD:staged.txt") != "reviewed" {
		t.Fatal("wrong staged content")
	}
	if strings.Contains(testGit(t, s.Root, "ls-tree", "--name-only", "HEAD"), "unstaged.txt") {
		t.Fatal("automatically staged user work")
	}
	if testGit(t, s.Root, "log", "-1", "--format=%s") != in.Message {
		t.Fatal("commit message changed")
	}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("blind retry accepted: %v", err)
	}
	if testGit(t, s.Root, "rev-list", "--count", "HEAD") != "2" {
		t.Fatal("duplicate commit")
	}
}

func TestConventionalCommitValidation(t *testing.T) {
	for _, message := range []string{"feat: add memory", "fix(sqlite): retry writes", "feat(api)!: change contract\n\nBREAKING CHANGE: use new inputs", "custom: useful action\n\nBody text.\n\nRefs: #12"} {
		if err := validateCommitMessage(message); err != nil {
			t.Fatalf("valid %q: %v", message, err)
		}
	}
	for _, message := range []string{"", "fix bug", "Feat: uppercase", "feat: ", "feat( ): empty scope", "fix: change\nno blank separator", "feat: change\n\nBREAKING CHANGE:", "fix: bad\x00message"} {
		if err := validateCommitMessage(message); err == nil {
			t.Fatalf("invalid accepted: %q", message)
		}
	}
	var out gitOutput
	data := []byte(strings.Repeat("x", 100000))
	if n, err := out.Write(data); err != nil || n != len(data) || out.buffer.Len() != 65536 || !out.truncated {
		t.Fatal("Git output was not bounded and drained")
	}
}
