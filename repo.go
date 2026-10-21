package bearing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ResolveRepository(ctx context.Context, path string) (Repository, error) {
	if path == "" {
		path = "."
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Repository{}, err
	}
	out, err := exec.CommandContext(ctx, "git", "-C", absolute, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return Repository{}, fail("invalid_input", "Choose a directory inside a Git working tree.")
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return Repository{}, err
	}
	r := Repository{Path: root, CapturedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	out, err = exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "HEAD").Output()
	if err == nil {
		r.Head = strings.TrimSpace(string(out))
	}
	out, err = exec.CommandContext(ctx, "git", "-C", root, "symbolic-ref", "--quiet", "--short", "HEAD").Output()
	if err == nil {
		r.Branch = strings.TrimSpace(string(out))
	}
	out, err = exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain=v1", "--untracked-files=normal").Output()
	if err != nil {
		return Repository{}, fail("io_error", "Could not inspect the working tree.")
	}
	r.Dirty = len(out) != 0
	return r, nil
}

// workingTree hashes HEAD plus every non-ignored working file, untracked files
// included. A temporary index under the self-ignored state directory leaves the
// user's staging area untouched; Git stores blobs for files it has not seen.
func workingTree(ctx context.Context, repo Repository) (string, error) {
	dir, err := os.MkdirTemp(filepath.Join(repo.Path, ".bearing"), "tree-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	index := filepath.Join(dir, "index")
	steps := [][]string{{"add", "--all"}, {"write-tree"}}
	if repo.Head != "" {
		// A copy of the user's index lets read-tree --reset keep file stat data for
		// unchanged entries, so add does not rehash every file (about 20x faster).
		if path, err := runGit(ctx, repo.Path, "rev-parse", "--path-format=absolute", "--git-path", "index"); err == nil {
			if data, err := os.ReadFile(strings.TrimSpace(string(path))); err == nil {
				if err := os.WriteFile(index, data, 0600); err != nil {
					return "", err
				}
			}
		}
		steps = append([][]string{{"read-tree", "--reset", repo.Head}}, steps...)
	}
	var out []byte
	for _, args := range steps {
		command := exec.CommandContext(ctx, "git", append([]string{"-C", repo.Path}, args...)...)
		command.Env = append(os.Environ(), "GIT_INDEX_FILE="+index)
		if out, err = command.Output(); err != nil {
			return "", fail("io_error", "Could not hash the working tree: git "+args[0]+" failed.")
		}
	}
	return strings.TrimSpace(string(out)), nil
}

func statePath(ctx context.Context, root string) (string, error) {
	// Removing a linked worktree deletes its ignored files, including Bearing state.
	dirs, err := runGit(ctx, root, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if gitDir, commonDir, _ := strings.Cut(strings.TrimSpace(string(dirs)), "\n"); gitDir != commonDir {
		return "", fail("invalid_input", "Bearing runs in the main checkout, not a linked worktree, because removing a worktree deletes its Bearing state. Use branches in the main checkout.")
	}
	dir := filepath.Join(root, ".bearing")
	for _, path := range []string{dir, filepath.Join(dir, "state.db")} {
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fail("invalid_input", "Bearing state must not be a symbolic link.")
		}
	}
	return filepath.Join(dir, "state.db"), nil
}
