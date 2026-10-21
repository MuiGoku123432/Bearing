# Bearing

Bearing helps coding agents keep their direction across sessions. It carries the task, engineering rules, useful decisions, research, specs, and next action between Codex, Claude Code, and OpenCode.

It is a local Go CLI and stdio MCP server. Gortex supplies code understanding and source changes; the host agent supplies the model and runs verification. Bearing stores durable state in SQLite and checks the evidence submitted when a task finishes.

**Status:** working developer preview, tested on macOS arm64. Live-agent handoff evaluations and measured token comparisons remain before a v0.1 release. See [implementation status](IMPLEMENTATION.md).

## Build and try it

Requires Git and Go 1.26 or newer. No C compiler is required.

```sh
CGO_ENABLED=0 go build -o bin/bearing ./cmd/bearing
./bin/bearing version
./bin/bearing init --repo /absolute/path/to/checkout
./bin/bearing rules
```

`init` creates `.bearing/state.db` in the Git checkout. The `.bearing/` directory ignores itself, so tracked files such as `.gitignore` stay unchanged. Bearing runs in the main checkout, not in linked worktrees. State stays local; Bearing does not send documents to a model or external service.

Start a task with a JSON file or stdin:

```sh
./bin/bearing start --repo /absolute/path/to/checkout --input - --json <<'JSON'
{
  "request_id": "implement-retry-1",
  "outcome": "Fix retrying a saved checkpoint",
  "acceptance": [
    {"id": "a1", "description": "An identical retry returns the original checkpoint without another write"}
  ]
}
JSON
```

Save the returned task ID and version. Use `bearing checkpoint` after useful progress, then `bearing resume TASK_ID` in a fresh session. Put flags before positional IDs. [CLI examples](CLI_EXAMPLES.md) show research, specs, memory, checkpoint, and completion requests.

## Install and connect your coding agent

From this checkout, install the binary and connect one project:

```sh
sh install.sh --repo /absolute/path/to/checkout --host codex
```

Choose `claude` or `opencode` for another client. The script builds with CGo disabled and installs into `~/.local/bin`; `--bin-dir PATH` selects another directory. It reports any PATH change you need to make. It does not edit your shell profile.

After installing the binary, connect more projects or clients:

```sh
bearing install --repo /absolute/path/to/checkout --host claude --dry-run
bearing install --repo /absolute/path/to/checkout --host claude
bearing install --repo /absolute/path/to/checkout --host opencode
```

Installation initializes the local store, adds the project MCP settings, and installs the workflow and embedded engineering policy. Existing team instructions and other servers stay in place. Changed files are backed up under `.bearing/install-backups/`; repeating an unchanged installation leaves client files alone. Restart the client and approve its project MCP settings if prompted. Native Gortex must already be connected and cover the checkout.

See [installation and recovery](INSTALL.md) for configuration limits, OpenCode version selection, updates, and removal. For a manual merge, generate a setup bundle:

```sh
./bin/bearing setup --repo /absolute/path/to/checkout --host codex --out /tmp/bearing-codex-setup
./bin/bearing setup --repo /absolute/path/to/checkout --host claude --out /tmp/bearing-claude-setup
./bin/bearing setup --repo /absolute/path/to/checkout --host opencode --out /tmp/bearing-opencode-setup
```

Each destination must be new. `APPLY.md` explains how to merge the configuration fragment, add the instruction entry point, and install the workflow skill. The bundle contains the full engineering policy. Setup does not change existing client configuration. Keep Gortex configured as its own native MCP server and follow its active repository policy.

The server command is `bearing mcp --repo /absolute/path/to/checkout`. It exposes eight tools:

| Tool | Purpose |
| --- | --- |
| `bearing_start` | Start a research, spec, or build task with acceptance criteria |
| `bearing_resume` | Recover requirements, rules, progress, documents, and relevant memory |
| `bearing_remember` | Save, supersede, or retract an attributed durable claim |
| `bearing_recall` | Search memory or retrieve exact saved document versions |
| `bearing_checkpoint` | Save progress and document snapshots atomically |
| `bearing_finish` | Check acceptance evidence and engineering self-review before completing |
| `bearing_git` | Inspect Git state, create and switch branches, and create conventional commits |
| `bearing_rules` | Read the embedded engineering policy and workflow |

Every tool publishes input and output schemas. MCP results contain either `result` or `error`, with matching structured content and text fallback. CLI `--json` returns the operation result directly, or an `error` envelope. Neither interface silently switches repositories.

## Branches and conventional commits

Bearing works on branches in one checkout and does not use linked worktrees: removing a worktree deletes its ignored Bearing state, so Bearing refuses to run in one. Use `bearing git --input git-request.json --json` or `bearing_git`.

- New branches follow [Conventional Branch](https://conventional-branch.github.io/) names such as `feat/add-login` and start from the trunk (the remote default branch, or `main`/`master`).
- Work started on the trunk moves to the new branch. Bearing never commits to `main`, `master`, `develop`, or the trunk.
- A commit uses only changes already staged by the host and must match the `index_tree` from inspection. Messages follow Conventional Commits. Git hooks and signing remain active.
- No push, force, deletion, stash, or automatic staging.

See [Git requests and recovery](GIT_WORKFLOW.md) for examples and dry runs.

## Research → spec → build

Use the same task loop for each kind. A small fix can start directly as a build task.

1. Start `kind: "research"`. The agent investigates primary sources and writes concise Markdown findings with sources, uncertainty, and a recommendation.
2. Save the Markdown through `documents` in a checkpoint or finish request. Bearing stores its body, SHA-256 digest, sources, and an immutable document ID.
3. Start `kind: "spec"` with those IDs in `input_documents`. The agent turns the findings into scope, behavior, constraints, and verifiable acceptance criteria.
4. Start `kind: "build"` with the saved spec IDs. Research lineage remains attached. The agent uses Gortex to inspect and change code, verifies the requirements, and checkpoints progress.

Research and spec tasks must save their final document when finishing. Resume returns document metadata; recall returns the stored body on request. If a file changes or disappears, Bearing warns and keeps the captured version available. It does not silently replace accepted requirements.

## Engineering rules are built in

[ENGINEERING.md](ENGINEERING.md) is embedded verbatim and identified by SHA-256. Start and resume return the full policy. Setup installs the same text in the skill. The [workflow](WORKFLOW.md) keeps investigation, implementation, verification, and handoff direct.

Finish requires passing acceptance evidence and the five self-review checks: single-use code, reuse, necessity, conventions, and minimality. Evidence records its source and outcome; Bearing records the checkout context itself. Bearing validates these records; the agent and its host remain responsible for actually performing the work.

## Durable state and context

- SQLite uses WAL, foreign keys, full synchronous writes, and bounded lock waits. Tasks, checkpoints, document snapshots, memory corrections, and request receipts use transactions.
- Task and memory writes accept request IDs; Bearing generates one when it is omitted and returns it. Updates require current versions. Retry identical uncertain database requests with the same ID. Use a new ID for a new action. Git writes use state preconditions instead; inspect after an uncertain Git result before retrying.
- Memory is explicitly saved with a claim, source, basis, scope, and optional path/symbol anchors. Corrected and retracted claims remain auditable but leave normal recall.
- Recall filters applicability before FTS5 ranking. English stemming matches word forms such as `retry` and `retrying`. Default memory recall returns at most five items within a 12,000-byte record budget. Resume is bounded to roughly 32,000 JSON bytes including full policy, with visible omissions.
- These are byte budgets, not measured model token counts. No embedding service, transcript crawler, second code index, or background model loop is included.
- Restoring into a new checkout preserves provenance and warns about checkout changes. Project-scoped claims can be reconsidered there; old checkout-scoped claims are excluded from ordinary recall.

## Recovery

```sh
bearing export --out /safe/path/state.json
bearing backup --out /safe/path/state.db
bearing restore --repo /new/checkout --input /safe/path/state.db
```

Export is readable JSON containing tasks, checkpoints, document bodies, and memories. Backup is a consistent SQLite snapshot including retry receipts and indexes. Restore accepts a physical backup, checks its schema and integrity, and refuses to replace an existing database. Choose new output files; commands do not overwrite backups or exports. State and backups may contain private project information, so choose their destinations deliberately.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
CGO_ENABLED=0 go build -o bin/bearing ./cmd/bearing
```

The application uses the standard library plus `modernc.org/sqlite` and the official MCP Go SDK. See [BUILD_PLAN.md](BUILD_PLAN.md), [CONTRACT.md](CONTRACT.md), [DESIGN.md](DESIGN.md), and the [research](research/README.md).
