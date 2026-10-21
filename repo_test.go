package bearing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryBindingAndWorktreeIsolation(t *testing.T) {
	ctx := context.Background()
	root := testRepo(t)
	subdir := filepath.Join(root, "folder with spaces")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Init(ctx, subdir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Root != root {
		t.Fatalf("bound to subdirectory: %s", s.Root)
	}
	started := testStart(t, s, "build")
	for _, args := range [][]string{
		{"-C", root, "-c", "core.hooksPath=/dev/null", "-c", "user.name=Bearing Test", "-c", "user.email=bearing@example.invalid", "commit", "--allow-empty", "-qm", "fixture"},
		{"-C", root, "worktree", "add", "--detach", filepath.Join(t.TempDir(), "linked checkout"), "HEAD"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git fixture: %s %v", out, err)
		}
		if args[2] != "worktree" {
			continue
		}
		other, err := Init(ctx, args[5])
		if err != nil {
			t.Fatal(err)
		}
		defer other.Close()
		if _, err := other.Resume(ctx, started.Mutation.ID); err == nil {
			t.Fatal("worktrees share task state")
		}
	}
	bare := filepath.Join(t.TempDir(), "bare.git")
	if out, err := exec.Command("git", "init", "--bare", "-q", bare).CombinedOutput(); err != nil {
		t.Fatalf("bare fixture: %s %v", out, err)
	}
	if _, err := ResolveRepository(ctx, bare); err == nil {
		t.Fatal("bare repository accepted")
	}
	if _, err := ResolveRepository(ctx, t.TempDir()); err == nil {
		t.Fatal("non-Git directory accepted")
	}
}

func TestInitRefusesSymlinkedStateAndIgnore(t *testing.T) {
	for _, name := range []string{".bearing", filepath.Join(".bearing", ".gitignore")} {
		root := testRepo(t)
		outside := t.TempDir()
		if name != ".bearing" {
			outside = filepath.Join(outside, "ignore")
			if err := os.WriteFile(outside, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(root, ".bearing"), 0700); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(outside, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		if s, err := Init(context.Background(), root); err == nil {
			s.Close()
			t.Fatalf("followed symlink: %s", name)
		}
		if data, err := os.ReadFile(outside); name != ".bearing" && (err != nil || string(data) != "preserve") {
			t.Fatalf("wrote through symlink: %q %v", data, err)
		}
	}
}

func TestBusyWriterIsBoundedAndRetryable(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	other, err := Open(ctx, s.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	begin := time.Now()
	_, err = other.Checkpoint(ctx, CheckpointInput{RequestID: "busy", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Contended write", NextAction: "Verify"})
	if err == nil || ErrorDetails(err).Code != "busy" || !ErrorDetails(err).Retryable {
		t.Fatalf("lock error: %v", err)
	}
	if time.Since(begin) > 4*time.Second {
		t.Fatal("writer exceeded bounded lock wait")
	}
}
