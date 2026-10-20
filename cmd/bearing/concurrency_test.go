package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	bearing "github.com/MuiGoku123432/Bearing"
)

func TestCLIWriterProcess(t *testing.T) {
	if os.Getenv("BEARING_TEST_WRITER") != "1" {
		return
	}
	err := run(context.Background(), []string{"checkpoint", "--repo", os.Getenv("BEARING_TEST_REPO"), "--input", "-", "--json"}, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		os.Exit(exitCode(err))
	}
	os.Exit(0)
}

func TestConcurrentCLIProcesses(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git: %s %v", out, err)
	}
	store, err := bearing.Init(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	started, err := store.Start(ctx, bearing.StartInput{RequestID: "start", Outcome: "Serialize writes across clients", Acceptance: []bearing.Criterion{{Description: "One writer wins"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, sameRequest := range []bool{false, true} {
		resumed, err := store.Resume(ctx, started.Mutation.ID)
		if err != nil {
			t.Fatal(err)
		}
		var group sync.WaitGroup
		codes := make(chan int, 2)
		for i := 0; i < 2; i++ {
			id := bearing.NewID()
			if sameRequest {
				id = "shared-retry"
			}
			input, err := json.Marshal(bearing.CheckpointInput{RequestID: id, TaskID: started.Mutation.ID, ExpectedVersion: resumed.Task.Version, Summary: "Concurrent progress", NextAction: "Verify"})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestCLIWriterProcess$")
			cmd.Env = append(os.Environ(), "BEARING_TEST_WRITER=1", "BEARING_TEST_REPO="+root)
			cmd.Stdin = strings.NewReader(string(input))
			group.Add(1)
			go func() {
				defer group.Done()
				err := cmd.Run()
				code := 0
				var exit *exec.ExitError
				if errors.As(err, &exit) {
					code = exit.ExitCode()
				} else if err != nil {
					code = -1
				}
				codes <- code
			}()
		}
		group.Wait()
		close(codes)
		success, conflicts := 0, 0
		for code := range codes {
			if code == 0 {
				success++
			} else if code == 3 {
				conflicts++
			} else {
				t.Fatalf("unexpected writer exit: %d", code)
			}
		}
		if sameRequest && success != 2 {
			t.Fatal("identical retry did not return the original success")
		}
		if !sameRequest && (success != 1 || conflicts != 1) {
			t.Fatalf("writer race: %d success, %d conflicts", success, conflicts)
		}
		after, err := store.Resume(ctx, started.Mutation.ID)
		if err != nil || after.Task.Version != resumed.Task.Version+1 {
			t.Fatal("concurrent requests applied more than one checkpoint")
		}
	}
}
