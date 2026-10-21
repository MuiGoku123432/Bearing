package bearing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryBinding(t *testing.T) {
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

func TestInitRefusesLinkedWorktree(t *testing.T) {
	s := gitTestStore(t)
	linked := filepath.Join(t.TempDir(), "linked")
	testGit(t, s.Root, "worktree", "add", "-q", "-b", "feat/linked", linked)
	if store, err := Init(context.Background(), linked); ErrorDetails(err).Code != "invalid_input" {
		if store != nil {
			store.Close()
		}
		t.Fatalf("linked worktree accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(linked, ".bearing")); !os.IsNotExist(err) {
		t.Fatal("state created in a linked worktree")
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
