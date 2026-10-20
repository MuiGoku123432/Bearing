package bearing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupRestoreAndExport(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	started := testStart(t, s, "build")
	in := CheckpointInput{RequestID: "saved-receipt", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "Ready for handoff", NextAction: "Verify"}
	cp, err := s.Checkpoint(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "bearing.db")
	if err := s.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err := s.Backup(ctx, backup); err == nil {
		t.Fatal("backup overwrote existing file")
	}
	destination := testRepo(t)
	if err := Restore(ctx, destination, backup); err != nil {
		t.Fatal(err)
	}
	if err := Restore(ctx, destination, backup); err == nil {
		t.Fatal("restore replaced existing state")
	}
	restored, err := Open(ctx, destination)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	resumed, err := restored.Resume(ctx, started.Mutation.ID)
	if err != nil || resumed.Checkpoint.ID != cp.CheckpointID {
		t.Fatalf("restored checkpoint: %+v %v", resumed, err)
	}
	if !strings.Contains(strings.Join(resumed.Warnings, " "), "Checkout or commit changed") {
		t.Fatal("restored checkout mismatch not reported")
	}
	retry, err := restored.Checkpoint(ctx, in)
	if err != nil || !retry.Replayed {
		t.Fatalf("receipt lost on restore: %v", err)
	}
	export := filepath.Join(t.TempDir(), "state.json")
	if err := restored.Export(ctx, export); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(export)
	if err != nil {
		t.Fatal(err)
	}
	var contents map[string]json.RawMessage
	if err := json.Unmarshal(data, &contents); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents["checkpoints"]), "Ready for handoff") {
		t.Fatal("checkpoint absent from export")
	}
	if err := Restore(ctx, testRepo(t), export); err == nil {
		t.Fatal("JSON export accepted as a physical backup")
	}
}
