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

func TestGitBranchesFollowConventionsAndKeepChanges(t *testing.T) {
	ctx := context.Background()
	s := gitTestStore(t)
	state, err := s.Git(ctx, GitInput{Operation: "inspect"})
	if err != nil || state.Trunk == "" || state.Trunk != state.Repository.Branch {
		t.Fatalf("trunk: %+v %v", state, err)
	}
	for _, name := range []string{"memory", "feat/Memory", "feat/new--memory", "refactor/memory", "--force"} {
		if _, err := s.Git(ctx, GitInput{Operation: "branch", Branch: name}); ErrorDetails(err).Code != "invalid_input" {
			t.Fatalf("unconventional branch %q accepted: %v", name, err)
		}
	}
	in := GitInput{Operation: "branch", Branch: "feat/memory", DryRun: true}
	if _, err := s.Git(ctx, in); err != nil {
		t.Fatal(err)
	}
	if got := testGit(t, s.Root, "branch", "--show-current"); got != state.Trunk {
		t.Fatal("dry run switched branches")
	}
	// Work started on the trunk moves onto the new branch.
	if err := os.WriteFile(filepath.Join(s.Root, "user-work.txt"), []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	in.DryRun = false
	created, err := s.Git(ctx, in)
	if err != nil || created.Repository.Branch != "feat/memory" || len(created.Warnings) != 0 {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("existing branch recreated: %v", err)
	}
	if _, err := s.Git(ctx, GitInput{Operation: "switch", Branch: state.Trunk}); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("dirty switch: %v", err)
	}
	testGit(t, s.Root, "add", "user-work.txt")
	testGit(t, s.Root, "commit", "-qm", "feat: keep user work")
	head := testGit(t, s.Root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(s.Root, "more-work.txt"), []byte("keep me too"), 0600); err != nil {
		t.Fatal(err)
	}
	// New branches start from the trunk, so dirty work elsewhere needs an explicit base.
	if _, err := s.Git(ctx, GitInput{Operation: "branch", Branch: "fix/header"}); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("dirty work moved to another base: %v", err)
	}
	stacked, err := s.Git(ctx, GitInput{Operation: "branch", Branch: "fix/header", Base: "HEAD"})
	if err != nil || stacked.Repository.Head != head {
		t.Fatalf("stacked branch: %+v %v", stacked, err)
	}
	if data, _ := os.ReadFile(filepath.Join(s.Root, "more-work.txt")); string(data) != "keep me too" {
		t.Fatal("user changes lost")
	}
	if err := os.Remove(filepath.Join(s.Root, "more-work.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Git(ctx, GitInput{Operation: "switch", Branch: "fix/missing"}); ErrorDetails(err).Code != "invalid_input" {
		t.Fatalf("missing branch: %v", err)
	}
	if _, err := s.Git(ctx, GitInput{Operation: "switch", Branch: state.Trunk}); err != nil {
		t.Fatal(err)
	}
	mergeHead := testGit(t, s.Root, "rev-parse", "--git-path", "MERGE_HEAD")
	if !filepath.IsAbs(mergeHead) {
		mergeHead = filepath.Join(s.Root, mergeHead)
	}
	if err := os.WriteFile(mergeHead, []byte(state.Repository.Head), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Git(ctx, GitInput{Operation: "branch", Branch: "fix/blocked-by-merge"}); ErrorDetails(err).Code != "conflict" {
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
	in := GitInput{Operation: "commit", ExpectedIndex: state.IndexTree, Message: "feat(memory): retain checkpoints"}
	if _, err := s.Git(ctx, in); ErrorDetails(err).Code != "conflict" {
		t.Fatalf("commit to trunk accepted: %v", err)
	}
	if _, err := s.Git(ctx, GitInput{Operation: "branch", Branch: "feat/checkpoints"}); err != nil {
		t.Fatal(err)
	}
	in.ExpectedIndex = "stale"
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
	if _, err := s.Git(ctx, in); err == nil {
		t.Fatal("blind retry accepted")
	}
	if testGit(t, s.Root, "rev-list", "--count", "HEAD") != "2" {
		t.Fatal("duplicate commit")
	}
}

func TestGitRefusesRedirectedEnvironment(t *testing.T) {
	s := gitTestStore(t)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "another-index"))
	if _, err := s.Git(context.Background(), GitInput{Operation: "inspect"}); ErrorDetails(err).Code != "invalid_input" {
		t.Fatalf("redirect accepted: %v", err)
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
