package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bearing "github.com/MuiGoku123432/Bearing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPProcess(t *testing.T) {
	if os.Getenv("BEARING_TEST_SERVER") != "1" {
		return
	}
	if err := run(context.Background(), []string{"mcp", "--repo", os.Getenv("BEARING_TEST_REPO")}, os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestStdioMCPAndCLIShareState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git: %s %v", out, err)
	}
	store, err := bearing.Init(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMCPProcess$")
	cmd.Env = append(os.Environ(), "BEARING_TEST_SERVER=1", "BEARING_TEST_REPO="+root)
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	client := mcp.NewClient(&mcp.Implementation{Name: "bearing-integration-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v %s", err, diagnostics.String())
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 7 {
		t.Fatalf("tools: %+v %v", tools, err)
	}
	for _, tool := range tools.Tools {
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("missing schema: %s", tool.Name)
		}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_rules", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatalf("rules: %+v %v", result, err)
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "Rule of three") {
		t.Fatal("engineering policy absent")
	}
	arguments := map[string]any{"request_id": "mcp-start", "outcome": "Carry this task between clients", "acceptance": []map[string]any{{"description": "CLI can resume the task"}}}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_start", Arguments: arguments})
	if err != nil || result.IsError {
		t.Fatalf("start: %+v %v", result, err)
	}
	var envelope toolOutput[bearing.StartResult]
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Result == nil {
		t.Fatalf("missing structured start result: %+v", envelope)
	}
	started := *envelope.Result
	var output, errors bytes.Buffer
	if err := run(ctx, []string{"resume", "--repo", root, "--json", started.Mutation.ID}, strings.NewReader(""), &output, &errors); err != nil {
		t.Fatal(err)
	}
	var resumed bearing.Context
	if err := json.Unmarshal(output.Bytes(), &resumed); err != nil {
		t.Fatal(err)
	}
	if resumed.Task.ID != started.Mutation.ID || resumed.Task.Outcome != "Carry this task between clients" {
		t.Fatal("CLI/MCP state diverged")
	}
	cp := bearing.CheckpointInput{RequestID: "cli-cp", TaskID: started.Mutation.ID, ExpectedVersion: 1, Summary: "CLI saved the handoff", NextAction: "Continue through MCP"}
	data, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := run(ctx, []string{"checkpoint", "--repo", root, "--json", "--input", "-"}, bytes.NewReader(data), &output, &errors); err != nil {
		t.Fatal(err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_resume", Arguments: map[string]any{"task_id": started.Mutation.ID}})
	if err != nil || result.IsError {
		t.Fatalf("resume: %+v %v", result, err)
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "Continue through MCP") {
		t.Fatal("MCP did not see CLI checkpoint")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_checkpoint", Arguments: cp})
	if err != nil || result.IsError {
		t.Fatalf("cross-transport receipt: %+v %v", result, err)
	}
	cp.RequestID = "stale-mcp"
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_checkpoint", Arguments: cp})
	if err != nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "conflict") {
		t.Fatalf("structured conflict missing: %+v %v", result, err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_remember", Arguments: bearing.RememberInput{RequestID: "mcp-memory", Kind: "lesson", Claim: "The CLI and MCP share checkpoints", Basis: "observed", Source: "This integration test"}})
	if err != nil || result.IsError {
		t.Fatalf("remember: %+v %v", result, err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_recall", Arguments: bearing.RecallInput{Query: "checkpoints"}})
	if err != nil || result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "The CLI and MCP share checkpoints") {
		t.Fatalf("recall: %+v %v", result, err)
	}
	checks := map[string]bearing.ReviewCheck{}
	for _, name := range []string{"single_use", "reuse", "necessity", "conventions", "minimality"} {
		checks[name] = bearing.ReviewCheck{Status: "pass"}
	}
	finish := bearing.FinishInput{RequestID: "mcp-finish", TaskID: started.Mutation.ID, ExpectedVersion: 2, Summary: "Verified the transport handoff", AcceptanceResults: []bearing.AcceptanceResult{{ID: "a1", Status: "pass", EvidenceIDs: []string{"v1"}}}, Verification: []bearing.Evidence{{ID: "v1", Kind: "inspection", Description: "CLI resumed the task saved through MCP", Source: "TestStdioMCPAndCLIShareState", Outcome: "pass", ObservedAt: resumed.Repository.CapturedAt, Context: resumed.Repository, AcceptanceIDs: []string{"a1"}}}, Review: bearing.Review{PolicyID: bearing.Rules().ID, Summary: "Integration fixture review", Checks: checks}}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_finish", Arguments: finish})
	if err != nil || result.IsError {
		t.Fatalf("finish: %+v %v", result, err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "bearing_start", Arguments: map[string]any{"request_id": "bad", "outcome": "bad", "acceptance": []any{}, "repo": "/tmp/elsewhere"}})
	if err == nil && !result.IsError {
		t.Fatal("tool accepted a repository override")
	}
}

func TestCLIInputErrorsAndSetup(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git: %s %v", out, err)
	}
	store, err := bearing.Init(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	var output, errors bytes.Buffer
	err = run(ctx, []string{"start", "--repo", root, "--input", "-", "--json"}, strings.NewReader("{\"unknown\":true}"), &output, &errors)
	if err == nil || exitCode(err) != 2 || !json.Valid(output.Bytes()) {
		t.Fatalf("CLI error envelope: %s %v", output.String(), err)
	}
	for _, host := range []string{"codex", "claude", "opencode"} {
		destination := filepath.Join(t.TempDir(), host)
		if _, err := renderSetup(ctx, root, host, destination); err != nil {
			t.Fatal(err)
		}
		if _, err := renderSetup(ctx, root, host, destination); err == nil {
			t.Fatal("setup overwrote an existing directory")
		}
		policy, err := os.ReadFile(filepath.Join(destination, "ENGINEERING.md"))
		if err != nil || string(policy) != bearing.Rules().Text {
			t.Fatal("setup policy differs from embedded policy")
		}
		files, err := os.ReadDir(destination)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(destination, file.Name()))
			if err != nil || !json.Valid(data) {
				t.Fatalf("invalid setup JSON: %s %v", file.Name(), err)
			}
		}
	}
}
