package bearing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) createGitWorktree(ctx context.Context, in GitInput, result GitResult, base string) (GitResult, error) {
	if !filepath.IsAbs(in.Path) {
		return result, fail("invalid_input", "Supply an absolute path for the new worktree, beneath an existing directory.")
	}
	path := filepath.Clean(in.Path)
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return result, fail("invalid_input", "The worktree parent directory must already exist.")
	}
	path = filepath.Join(parent, filepath.Base(path))
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return result, fail("conflict", "The worktree destination exists or cannot be inspected. Choose a new path.")
	}
	relative, err := filepath.Rel(s.Root, path)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return result, fail("invalid_input", "Put the new worktree outside the current checkout.")
	}
	result.Destination = &Repository{Path: path, Head: base, Branch: in.Branch}
	result.Message = "Create worktree " + path + " on " + in.Branch + "; copy saved Bearing state into its own database."
	if result.Repository.Dirty {
		result.Warnings = append(result.Warnings, "Uncommitted source files stay in the current checkout. Only saved Bearing state and files in the selected Git commit are carried over.")
	}
	if in.DryRun {
		return result, nil
	}
	backup := filepath.Join(s.Root, ".bearing", "worktree-"+NewID()+".db")
	if err := s.Backup(ctx, backup); err != nil {
		return result, err
	}
	if err := checkGitState(ctx, s.Root, in); err != nil {
		os.Remove(backup)
		return result, err
	}
	if _, err := runGit(ctx, s.Root, "worktree", "add", "-b", in.Branch, "--", path, base); err != nil {
		return result, fail("outcome_unknown", "Git did not confirm worktree creation. Inspect branches/worktrees before retrying. Saved Bearing state is at "+backup+": "+err.Error())
	}
	result.RecoveryBackup = backup
	created, err := ResolveRepository(ctx, path)
	if err != nil || created.Path != path || created.Head != base || created.Branch != in.Branch {
		result.Warnings = append(result.Warnings, "The new checkout could not be verified. Bearing state was not restored. Inspect it and use the recovery backup explicitly.")
		return result, nil
	}
	result.Destination = &created
	if err := Restore(ctx, path, backup); err != nil {
		result.Warnings = append(result.Warnings, "Git created the worktree, but Bearing state restoration did not complete: "+err.Error()+". Inspect the destination; the recovery backup has been kept.")
		return result, nil
	}
	result.StateCopied = true
	if err := os.Remove(backup); err != nil {
		result.Warnings = append(result.Warnings, "State was copied, but its temporary recovery backup could not be removed.")
	} else {
		result.RecoveryBackup = ""
	}
	if destination, err := ResolveRepository(ctx, path); err == nil {
		result.Destination = &destination
	}
	result.Message = "Created worktree with an independent copy of saved Bearing state. Resume the same task ID there and review checkout warnings. Run bearing install for the new path and client; confirm native Gortex coverage."
	result.Warnings = append(result.Warnings, "Future state changes do not synchronize between worktrees. Existing client configuration may still point at the original checkout until reinstalled.")
	return result, nil
}
