package bearing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreRejectsInconsistentRecords(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	if err := os.WriteFile(filepath.Join(s.Root, "spec.md"), []byte("# Captured requirement"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := s.Checkpoint(ctx, CheckpointInput{RequestID: "cp", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Saved work", NextAction: "Verify", Documents: []DocumentInput{{Path: "spec.md", Kind: "spec", Summary: "Requirement"}}})
	if err != nil {
		t.Fatal(err)
	}
	for name, corrupt := range map[string]string{
		"document_tampering":     `UPDATE documents SET data=json_set(data,'$.body','Changed content')`,
		"invalid_receipt":        `UPDATE write_receipts SET outcome='{'`,
		"broken_latest_pointer":  `UPDATE tasks SET data=json_set(data,'$.latest_checkpoint_id','missing')`,
		"version_disagreement":   `UPDATE tasks SET data=json_set(data,'$.version',20)`,
		"missing_input_document": `UPDATE tasks SET data=json_set(data,'$.input_documents',json('["missing"]'))`,
		"invalid_json":           `UPDATE checkpoints SET data='{'`,
		"missing_checkpoint":     `DELETE FROM checkpoints`,
	} {
		t.Run(name, func(t *testing.T) {
			backup := filepath.Join(t.TempDir(), "backup.db")
			if err := s.Backup(ctx, backup); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", backup)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(corrupt); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			destination := testRepo(t)
			if err := Restore(ctx, destination, backup); err == nil {
				t.Fatal("inconsistent backup restored")
			}
			if _, err := os.Stat(filepath.Join(destination, ".bearing", "state.db")); !os.IsNotExist(err) {
				t.Fatal("failed restore published a database")
			}
		})
	}
}

func TestSQLiteFullRollsBackCheckpoint(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	var pages int
	if err := s.DB.QueryRow("PRAGMA page_count").Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(fmt.Sprintf("PRAGMA max_page_count=%d", pages)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "large.md"), []byte(strings.Repeat("Research finding.\n", 12000)), 0600); err != nil {
		t.Fatal(err)
	}
	in := CheckpointInput{RequestID: "full-retry", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Capture findings", NextAction: "Review", Documents: []DocumentInput{{Path: "large.md", Kind: "research", Summary: "Research findings"}}}
	_, err := s.Checkpoint(ctx, in)
	if err == nil || !strings.Contains(err.Error(), "full") {
		t.Fatalf("expected SQLite full error: %v", err)
	}
	resumed, err := s.Resume(ctx, started.Mutation.ID)
	if err != nil || resumed.Task.Version != 1 || resumed.Checkpoint != nil {
		t.Fatalf("partial checkpoint after full database: %+v %v", resumed.Task, err)
	}
	if _, err := s.DB.Exec("PRAGMA max_page_count=100000"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(ctx, in); err != nil {
		t.Fatalf("failed write left a receipt behind: %v", err)
	}
}
