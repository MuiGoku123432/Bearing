package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"

	bearing "github.com/MuiGoku123432/Bearing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const usage = `Bearing keeps coding work moving across sessions.

  bearing init [--repo PATH]
  bearing start|checkpoint|finish|remember --input FILE|- [--json]
  bearing resume [--repo PATH] [TASK_ID]
  bearing recall --input FILE|- [--json]
  bearing status [--json]
  bearing git --input FILE|- [--repo PATH] [--json]
  bearing history [--before SEQUENCE] TASK_ID [--json]
  bearing rules [--json]
  bearing setup --host codex|claude|opencode --out NEW_DIRECTORY
  bearing install --host codex|claude|opencode [--repo PATH] [--dry-run]
                  [--opencode-version 1|2]
  bearing mcp --repo PATH
  bearing export|backup --out NEW_FILE
  bearing restore --input BACKUP --repo PATH
  bearing version

Use --repo PATH on any repository command. Put flags before positional IDs.
Mutation input is a JSON object; --input - reads stdin. Missing request IDs
are generated and printed to stderr before the write so retries are possible.
`

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	switch bearing.ErrorDetails(err).Code {
	case "invalid_input", "not_initialized", "not_found":
		return 2
	case "conflict", "request_id_conflict":
		return 3
	case "busy", "outcome_unknown":
		return 4
	default:
		return 1
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (resultErr error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		_, err := io.WriteString(stdout, usage)
		return err
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "Git checkout")
	input := flags.String("input", "", "JSON input file, or - for stdin")
	out := flags.String("out", "", "new output path")
	host := flags.String("host", "", "codex, claude, or opencode")
	dryRun := flags.Bool("dry-run", false, "preview installation without changing the checkout")
	opencodeVersion := flags.String("opencode-version", "", "OpenCode major version; otherwise detect installed version")
	before := flags.Int("before", 0, "history sequence cursor")
	asJSON := flags.Bool("json", false, "JSON output")
	defer func() {
		if resultErr == nil {
			return
		}
		detail := bearing.ErrorDetails(resultErr)
		if *asJSON && command != "mcp" {
			_ = json.NewEncoder(stdout).Encode(map[string]any{"error": detail})
		} else {
			fmt.Fprintln(stderr, detail.Message)
		}
	}()
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &bearing.Error{Code: "invalid_input", Message: err.Error()}
	}
	if flags.NArg() > 1 || (flags.NArg() > 0 && command != "resume" && command != "history") {
		return &bearing.Error{Code: "invalid_input", Message: "Unexpected argument. Put flags before the task ID."}
	}
	var value any
	switch command {
	case "version":
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			return err
		}
		defer db.Close()
		var engine string
		if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&engine); err != nil {
			return err
		}
		value = map[string]string{"version": bearing.Version(), "sqlite_version": engine, "go_version": runtime.Version(), "policy_id": bearing.Rules().ID, "workflow_id": bearing.Rules().WorkflowID, "planning_id": bearing.Rules().PlanningID}
	case "rules":
		value = bearing.Rules()
	case "setup":
		var err error
		value, err = renderSetup(ctx, *repo, *host, *out)
		if err != nil {
			return err
		}
	case "install":
		var err error
		value, err = installSetup(ctx, *repo, *host, *opencodeVersion, *dryRun)
		if err != nil {
			return err
		}
	case "restore":
		if err := bearing.Restore(ctx, *repo, *input); err != nil {
			return err
		}
		value = "Restored the saved store. Resume a task to review its checkout context."
	default:
		known := map[string]bool{"init": true, "mcp": true, "git": true, "start": true, "resume": true, "status": true, "history": true, "recall": true, "remember": true, "checkpoint": true, "finish": true, "export": true, "backup": true}
		if !known[command] {
			return &bearing.Error{Code: "invalid_input", Message: "Unknown command. Run bearing help."}
		}
		var store *bearing.Store
		var err error
		if command == "init" {
			store, err = bearing.Init(ctx, *repo)
		} else {
			store, err = bearing.Open(ctx, *repo)
		}
		if err != nil {
			return err
		}
		defer store.Close()
		switch command {
		case "mcp":
			return newMCPServer(store).Run(ctx, &mcp.StdioTransport{})
		case "init":
			value = map[string]string{"repository": store.Root, "sqlite_version": store.SQLiteVersion, "status": "ready"}
		case "resume":
			value, err = store.Resume(ctx, flags.Arg(0))
		case "status":
			value, err = store.Status(ctx)
		case "history":
			value, err = store.History(ctx, flags.Arg(0), *before)
		case "export":
			err = store.Export(ctx, *out)
			value = "Exported durable state to " + *out
		case "backup":
			err = store.Backup(ctx, *out)
			value = "Saved a consistent backup to " + *out
		default:
			value, err = runInput(ctx, store, command, *input, stdin, stderr)
		}
		if err != nil {
			return err
		}
	}
	if *asJSON {
		return json.NewEncoder(stdout).Encode(value)
	}
	return printHuman(stdout, value)
}

func runInput(ctx context.Context, store *bearing.Store, command, path string, stdin io.Reader, stderr io.Writer) (any, error) {
	if path == "" {
		return nil, &bearing.Error{Code: "invalid_input", Message: "Supply --input FILE or --input -."}
	}
	reader := stdin
	if path != "-" {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader = file
	}
	data, err := io.ReadAll(io.LimitReader(reader, 65537))
	if err != nil {
		return nil, err
	}
	var start bearing.StartInput
	var checkpoint bearing.CheckpointInput
	var finish bearing.FinishInput
	var memory bearing.RememberInput
	var recall bearing.RecallInput
	var gitInput bearing.GitInput
	var target any
	var requestID *string
	switch command {
	case "start":
		target, requestID = &start, &start.RequestID
	case "checkpoint":
		target, requestID = &checkpoint, &checkpoint.RequestID
	case "finish":
		target, requestID = &finish, &finish.RequestID
	case "remember":
		target, requestID = &memory, &memory.RequestID
	case "recall":
		target = &recall
	case "git":
		target = &gitInput
	}
	if err := bearing.DecodeInput(data, target); err != nil {
		return nil, err
	}
	if requestID != nil && strings.TrimSpace(*requestID) == "" {
		*requestID = bearing.NewID()
		fmt.Fprintln(stderr, "Request ID:", *requestID)
	}
	switch command {
	case "start":
		return store.Start(ctx, start)
	case "checkpoint":
		return store.Checkpoint(ctx, checkpoint)
	case "finish":
		return store.Finish(ctx, finish)
	case "remember":
		return store.Remember(ctx, memory)
	case "git":
		return store.Git(ctx, gitInput)
	default:
		return store.Recall(ctx, recall)
	}
}
