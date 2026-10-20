package bearing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// GitInput exposes specific local operations, never arbitrary Git arguments.
type GitInput struct {
	Operation      string `json:"operation" jsonschema:"inspect, branch, switch, worktree, check-commit, or commit"`
	Branch         string `json:"branch,omitempty"`
	Base           string `json:"base,omitempty"`
	Path           string `json:"path,omitempty"`
	Message        string `json:"message,omitempty"`
	ExpectedHead   string `json:"expected_head,omitempty"`
	ExpectedBranch string `json:"expected_branch,omitempty" jsonschema:"Current branch name, or HEAD for a detached checkout"`
	ExpectedIndex  string `json:"expected_index,omitempty" jsonschema:"index_tree from inspect; required for commit"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

type GitWorktree struct {
	Path     string `json:"path"`
	Head     string `json:"head,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Detached bool   `json:"detached,omitempty"`
	Locked   bool   `json:"locked,omitempty"`
	Prunable bool   `json:"prunable,omitempty"`
}

type GitResult struct {
	Operation      string        `json:"operation"`
	Repository     Repository    `json:"repository"`
	Branches       []string      `json:"branches,omitempty"`
	Worktrees      []GitWorktree `json:"worktrees,omitempty"`
	IndexTree      string        `json:"index_tree,omitempty"`
	StagedSummary  string        `json:"staged_summary,omitempty"`
	Destination    *Repository   `json:"destination,omitempty"`
	StateCopied    bool          `json:"state_copied,omitempty"`
	RecoveryBackup string        `json:"recovery_backup,omitempty"`
	DryRun         bool          `json:"dry_run,omitempty"`
	Message        string        `json:"message"`
	Warnings       []string      `json:"warnings,omitempty"`
}

func (s *Store) Git(ctx context.Context, in GitInput) (GitResult, error) {
	result := GitResult{Operation: in.Operation, DryRun: in.DryRun}
	if err := inputSize(in, 16384); err != nil {
		return result, err
	}
	// These variables override -C and can redirect operations to another checkout.
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE"} {
		if os.Getenv(key) != "" {
			return result, fail("invalid_input", "Unset "+key+" before using Bearing Git operations.")
		}
	}
	if in.Operation != "inspect" && in.Operation != "branch" && in.Operation != "switch" && in.Operation != "worktree" && in.Operation != "commit" && in.Operation != "check-commit" {
		return result, fail("invalid_input", "Choose inspect, branch, switch, worktree, check-commit, or commit.")
	}
	usesBranch := in.Operation == "branch" || in.Operation == "switch" || in.Operation == "worktree"
	usesMessage := in.Operation == "commit" || in.Operation == "check-commit"
	if (!usesBranch && in.Branch != "") || (!usesMessage && in.Message != "") ||
		(in.Operation != "branch" && in.Operation != "worktree" && in.Base != "") ||
		(in.Operation != "worktree" && in.Path != "") || (in.Operation != "commit" && in.ExpectedIndex != "") {
		return result, fail("invalid_input", "This request contains fields for a different Git operation.")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	repository, err := ResolveRepository(ctx, s.Root)
	if err != nil {
		return result, err
	}
	result.Repository = repository
	if in.Operation == "check-commit" {
		if err := validateCommitMessage(in.Message); err != nil {
			return result, err
		}
		result.Message = "The message follows Bearing's Conventional Commits format."
		return result, nil
	}
	if in.Operation == "inspect" {
		return s.inspectGit(ctx, result)
	}
	branch := repository.Branch
	if branch == "" {
		branch = "HEAD"
	}
	if in.ExpectedHead == "" || in.ExpectedBranch == "" || in.ExpectedHead != repository.Head || in.ExpectedBranch != branch {
		return result, fail("conflict", "Inspect Git first and supply its current expected_head and expected_branch. An initial commit must exist.")
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return result, err
	}
	defer root.Close()
	lock, err := root.OpenFile(".bearing/git.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, fail("busy", "Another Bearing Git operation may be running. Inspect .bearing/git.lock before retrying.")
	}
	lock.Close()
	defer root.Remove(".bearing/git.lock")
	if err := checkGitState(ctx, s.Root, in); err != nil {
		return result, err
	}
	if in.Operation == "commit" {
		return s.commitGit(ctx, in, result)
	}
	if in.Branch == "" || strings.HasPrefix(in.Branch, "-") {
		return result, fail("invalid_input", "Supply an explicit branch name.")
	}
	if _, err := runGit(ctx, s.Root, "check-ref-format", "refs/heads/"+in.Branch); err != nil {
		return result, fail("invalid_input", "The branch name is not a valid Git ref.")
	}
	base := in.Base
	if base == "" {
		base = in.ExpectedHead
	}
	if in.Operation == "switch" {
		base = "refs/heads/" + in.Branch
	}
	resolved, err := runGit(ctx, s.Root, "rev-parse", "--verify", "--end-of-options", base+"^{commit}")
	if err != nil {
		return result, fail("invalid_input", "The requested base or local branch does not resolve to a commit.")
	}
	base = strings.TrimSpace(string(resolved))
	if in.Operation != "switch" {
		if _, err := runGit(ctx, s.Root, "show-ref", "--verify", "--quiet", "refs/heads/"+in.Branch); err == nil {
			return result, fail("conflict", "The branch already exists. Choose another name or use switch.")
		}
	}
	if in.Operation == "worktree" {
		return s.createGitWorktree(ctx, in, result, base)
	}
	if repository.Dirty {
		return result, fail("conflict", "Save, commit, or move your changes before switching branches. Bearing does not stash or discard them.")
	}
	result.Message = "Switch to branch " + in.Branch
	if in.Operation == "branch" {
		result.Message = "Create and switch to branch " + in.Branch + " at " + base
	}
	if in.DryRun {
		return result, nil
	}
	args := []string{"switch", "--no-guess", in.Branch}
	if in.Operation == "branch" {
		args = []string{"switch", "--create", in.Branch, base}
	}
	if _, err := runGit(ctx, s.Root, args...); err != nil {
		return result, fail("outcome_unknown", "Git did not confirm the branch change. Inspect before retrying: "+err.Error())
	}
	current, err := ResolveRepository(ctx, s.Root)
	if err != nil {
		return result, fail("outcome_unknown", "The branch command succeeded but its final state could not be read. Inspect before retrying.")
	}
	result.Repository = current
	if current.Branch != in.Branch || current.Head != base {
		result.Warnings = append(result.Warnings, "The final branch or HEAD differs from the requested result. Another Git process or hook may have changed it; inspect before proceeding.")
	}
	return result, nil
}

func checkGitState(ctx context.Context, path string, in GitInput) error {
	repository, err := ResolveRepository(ctx, path)
	if err != nil {
		return err
	}
	branch := repository.Branch
	if branch == "" {
		branch = "HEAD"
	}
	if repository.Head != in.ExpectedHead || branch != in.ExpectedBranch {
		return fail("conflict", "The branch or HEAD changed since inspection. Inspect again before writing.")
	}
	if (in.Operation == "branch" || in.Operation == "switch") && repository.Dirty {
		return fail("conflict", "The checkout has uncommitted changes. Bearing will not stash or discard them to switch branches.")
	}
	for _, marker := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply", "sequencer", "BISECT_LOG"} {
		data, err := runGit(ctx, path, "rev-parse", "--path-format=absolute", "--git-path", marker)
		if err != nil {
			return err
		}
		if _, err := os.Stat(strings.TrimSpace(string(data))); err == nil {
			return fail("conflict", "Finish or abort the existing Git operation before using Bearing Git mutations.")
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect Git operation: %w", err)
		}
	}
	return nil
}
