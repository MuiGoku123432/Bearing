package bearing

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// GitInput exposes specific local operations, never arbitrary Git arguments.
type GitInput struct {
	Operation     string `json:"operation" jsonschema:"inspect, branch, switch, or commit"`
	Branch        string `json:"branch,omitempty" jsonschema:"New Conventional Branch name such as feat/add-login for branch; an existing local branch for switch"`
	Base          string `json:"base,omitempty" jsonschema:"Start point for branch; defaults to the trunk branch"`
	Message       string `json:"message,omitempty" jsonschema:"Conventional Commits message for commit"`
	ExpectedIndex string `json:"expected_index,omitempty" jsonschema:"index_tree from inspect; required for commit"`
	DryRun        bool   `json:"dry_run,omitempty"`
}

type GitResult struct {
	Operation     string     `json:"operation"`
	Repository    Repository `json:"repository"`
	Trunk         string     `json:"trunk,omitempty"`
	Branches      []string   `json:"branches,omitempty"`
	IndexTree     string     `json:"index_tree,omitempty"`
	StagedSummary string     `json:"staged_summary,omitempty"`
	DryRun        bool       `json:"dry_run,omitempty"`
	Message       string     `json:"message"`
	Warnings      []string   `json:"warnings,omitempty"`
}

// Conventional Branch 1.1.0 types and description grammar: https://conventional-branch.github.io/
var branchName = regexp.MustCompile(`^(feature|feat|bugfix|fix|hotfix|release|chore|ai|claude|codex|copilot|cursor)/[a-z0-9]+(\.[a-z0-9]+)*(-[a-z0-9]+(\.[a-z0-9]+)*)*$`)

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
	if in.Operation != "inspect" && in.Operation != "branch" && in.Operation != "switch" && in.Operation != "commit" {
		return result, fail("invalid_input", "Choose inspect, branch, switch, or commit.")
	}
	if (in.Branch != "" && in.Operation != "branch" && in.Operation != "switch") || (in.Base != "" && in.Operation != "branch") ||
		((in.Message != "" || in.ExpectedIndex != "") && in.Operation != "commit") {
		return result, fail("invalid_input", "This request contains fields for a different Git operation.")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	repository, err := ResolveRepository(ctx, s.Root)
	if err != nil {
		return result, err
	}
	result.Repository = repository
	if in.Operation == "inspect" {
		return s.inspectGit(ctx, result)
	}
	if repository.Head == "" {
		return result, fail("conflict", "Create the initial commit before using Bearing Git operations.")
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
	for _, marker := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply", "sequencer", "BISECT_LOG"} {
		data, err := runGit(ctx, s.Root, "rev-parse", "--path-format=absolute", "--git-path", marker)
		if err != nil {
			return result, err
		}
		if _, err := os.Stat(strings.TrimSpace(string(data))); err == nil {
			return result, fail("conflict", "Finish or abort the existing Git operation before using Bearing Git writes.")
		} else if !os.IsNotExist(err) {
			return result, fmt.Errorf("inspect Git operation: %w", err)
		}
	}
	if in.Operation == "commit" {
		return s.commitGit(ctx, in, result)
	}
	if strings.HasPrefix(in.Branch, "-") {
		return result, fail("invalid_input", "Supply an explicit branch name.")
	}
	if _, err := runGit(ctx, s.Root, "check-ref-format", "refs/heads/"+in.Branch); err != nil || in.Branch == "" {
		return result, fail("invalid_input", "Supply a valid branch name.")
	}
	_, missing := runGit(ctx, s.Root, "show-ref", "--verify", "--quiet", "refs/heads/"+in.Branch)
	args, start := []string{"switch", "--no-guess", in.Branch}, ""
	result.Message = "Switch to branch " + in.Branch
	if in.Operation == "switch" {
		if missing != nil {
			return result, fail("invalid_input", "That local branch does not exist. Use branch to create it.")
		}
		if repository.Dirty {
			return result, fail("conflict", "Commit your changes before switching branches. Bearing does not stash or discard them.")
		}
	} else {
		if !branchName.MatchString(in.Branch) {
			return result, fail("invalid_input", "Name new branches type/description, for example feat/add-login or fix/header-bug. Types: feature, feat, bugfix, fix, hotfix, release, chore, ai, claude, codex, copilot, cursor. Use lowercase letters, digits, single hyphens, and dots.")
		}
		if missing == nil {
			return result, fail("conflict", "The branch already exists. Choose another name or use switch.")
		}
		base := in.Base
		if base == "" {
			if base = trunk(ctx, s.Root); base == "" {
				return result, fail("invalid_input", "No main or master branch was found. Supply base.")
			}
			// Bearing never fetches, so this compares with the last fetched upstream state.
			if count, err := runGit(ctx, s.Root, "rev-list", "--count", base+".."+base+"@{upstream}"); err == nil && strings.TrimSpace(string(count)) != "0" {
				result.Warnings = append(result.Warnings, base+" is behind its upstream as of the last fetch. Pull it first if the new branch needs that work.")
			}
		}
		resolved, err := runGit(ctx, s.Root, "rev-parse", "--verify", "--end-of-options", base+"^{commit}")
		if err != nil {
			return result, fail("invalid_input", "The base does not resolve to a commit.")
		}
		start = strings.TrimSpace(string(resolved))
		// Starting at the current commit carries uncommitted work safely onto the new branch.
		if repository.Dirty && start != repository.Head {
			return result, fail("conflict", "Commit your changes first, or create the branch with base HEAD to carry them onto the new branch.")
		}
		args = []string{"switch", "--create", in.Branch, start}
		result.Message = "Create and switch to branch " + in.Branch + " from " + base
	}
	if in.DryRun {
		return result, nil
	}
	if _, err := runGit(ctx, s.Root, args...); err != nil {
		return result, fail("outcome_unknown", "Git did not confirm the branch change. Inspect before retrying: "+err.Error())
	}
	current, err := ResolveRepository(ctx, s.Root)
	if err != nil {
		return result, fail("outcome_unknown", "The branch command succeeded but its final state could not be read. Inspect before retrying.")
	}
	result.Repository = current
	if current.Branch != in.Branch || (start != "" && current.Head != start) {
		result.Warnings = append(result.Warnings, "The final branch or HEAD differs from the requested result. Another Git process or hook may have changed it; inspect before proceeding.")
	}
	return result, nil
}

// trunk returns the branch new work starts from: the remote default branch when known, else main or master.
func trunk(ctx context.Context, root string) string {
	candidates := []string{"main", "master"}
	if out, err := runGit(ctx, root, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		candidates = append([]string{strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")}, candidates...)
	}
	for _, name := range candidates {
		if _, err := runGit(ctx, root, "show-ref", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name
		}
	}
	return ""
}
