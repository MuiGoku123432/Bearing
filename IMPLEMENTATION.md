# Implementation and verification

20 October 2026 · `0.1.0-dev` · macOS arm64

Bearing now has a working local developer preview. The native executable, shared domain operations, durable state, research/spec documents, embedded engineering policy, eight MCP tools, setup bundles, and recovery commands are implemented. The complete release acceptance story still needs fresh model-driven sessions and comparative evaluation.

## Implemented

- One Go binary using the standard library, `modernc.org/sqlite v1.60.1`, and `github.com/modelcontextprotocol/go-sdk v1.8.0`.
- Go module minimum 1.26.0; built and tested with Go 1.27.1. A build with `CGO_ENABLED=0` succeeds. Its SQLite engine reports **3.53.4**.
- Checkout-bound SQLite with WAL, foreign keys, FULL synchronization, schema version checks, bounded lock waits, transactional writes, optimistic versions, and durable request receipts.
- Research, spec, and build tasks. Explicit Markdown snapshots include bodies, digests, provenance, and lineage. Saved documents survive later checkpoints and missing working files.
- Attributed memory, project/checkout applicability, path/symbol anchors, FTS5 ranking, supersession/retraction, and bounded responses.
- Completion checks for acceptance evidence, current known checkout/HEAD, and all five engineering review outcomes.
- Exact embedded ENGINEERING.md, one maintained WORKFLOW.md, and host setup artifacts carrying those instructions.
- Shared CLI/MCP operations; all eight MCP handlers have typed input/output schemas and structured errors with text fallback.
- Explicit Git branch creation/switching, worktree creation with saved-state copying, and staged-only conventional commits. CLI and MCP share precondition checks, dry runs, bounded output, and native hook/signing behavior; no dependencies or database schema were added.
- Logical export, consistent physical backup, and restore into an empty destination.
- Source installer `install.sh` and `bearing install`: user-local binary, per-project MCP settings and workflow skill, preview mode, backups, repeat-install checks, and error recovery. [INSTALL.md](INSTALL.md) documents supported formats and manual-merge cases.

There is no model runtime, Gortex proxy, second source index, ORM, CLI framework, agent coordinator, or background memory ingestion.

## Checks run

| Check | Result |
| --- | --- |
| `CGO_ENABLED=0 go build -o bin/bearing ./cmd/bearing` | Pass |
| `go test ./...` | Pass during implementation |
| `go test -race ./...` | Pass after production changes |
| Expanded real-stdio test of all eight MCP handlers under `-race` | Pass |
| `go vet ./...` | Pass |
| `git diff --check` | Pass |
| Repository initialization and binary version report | SQLite 3.53.4; embedded policy/workflow hashes returned |

Twelve test files cover restart recovery, retry digest conflicts, same-request replay, separate-connection and separate-process writers, bounded SQLite contention, failure injection with full rollback, SQLite-full simulation with a page limit, corrupt JSON relationship rejection, document digest verification on restore, missing documents on retry, task requirement history, completion rejection, scope filtering, literal FTS queries, response budgets, Unicode, document lineage, path boundaries, worktree isolation, backup/restore, and CLI error/setup behavior. Installer tests cover all three clients (including both OpenCode layouts), dry runs, existing settings and instruction preservation, custom environment/timeouts, backup contents, permissions, repeated installs, paths with spaces, malformed and duplicate-key JSON, conflicting formats, custom skills, symlink refusal, and rollback/retry after a forced skill-directory write failure.

The MCP integration test launches a separate server process. It lists the eight schemas, exercises every handler, creates a real Git branch and staged conventional commit, passes inspected Git and task state between MCP and CLI, checks cross-interface receipt replay, and rejects a per-call repository override. This is a protocol and persistence test, not a model behavior evaluation.

Git tests cover stale HEAD/branch/index rejection, dry runs, dirty checkout protection, staged-only commits, native hook rejection, message validation, redirected-environment refusal, occupied/nested worktree destinations, and saved research/spec lineage across worktrees. They verify that later task versions and checkout-scoped memory remain independent. External Git writers can still race preflight checks; uncertain/partial outcomes require inspection as documented in [GIT_WORKFLOW.md](GIT_WORKFLOW.md).

## Consumer checks

| Consumer | Installed version | Observed result |
| --- | --- | --- |
| Codex CLI | 0.163.0 | Parsed the generated command/repository configuration through a temporary command-line override |
| Claude Code | 2.1.263 | Connected to Bearing using isolated local test settings; project `.mcp.json` correctly requests normal project approval |
| OpenCode | 1.18.33 | Connected to Bearing from an isolated project using its generated V1 configuration |

No model session was launched by these checks and no real client configuration was overwritten. They establish configuration/startup behavior, not correct agent tool choice or all six directed handoffs. OpenCode V2 configuration is also rendered from current documentation; it has not been tested against a V2 binary.

Configuration references checked during implementation: [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude Code MCP](https://code.claude.com/docs/en/mcp), [OpenCode V1](https://docs.opencode.ai/docs/mcp-servers/), and [OpenCode V2](https://opencode.ai/v2/docs/mcp-servers).

The ready-to-review setup bundles for this checkout are under `.bearing/setup/codex`, `.bearing/setup/claude`, and `.bearing/setup/opencode`. Each has an `APPLY.md`. They use this checkout's `bin/bearing` executable. These local artifacts and the database are ignored by Git.

## Installer verification

The source bootstrap was run into a temporary bin directory and Git checkout whose paths contain spaces. It built and launched the CGo-free executable, initialized state, and installed all three project integrations. A second run left the binary and client files unchanged. At the installer checkpoint, the installed binary completed an MCP handshake and exposed the then-current seven tools. The current source adds `bearing_git` as the eighth tool; its transport is covered by the expanded integration test. OpenCode 1.18.33 connected through the installed project configuration; Claude Code found the installed server and requested its normal project approval. Existing team instructions were retained and backed up.

Codex project discovery could not be verified in this sandbox: the CLI's `mcp get` did not read the project entry, and its app-server reported that persisted project trust is required. The generated launch format was previously parsed through command-line overrides. Users must open and trust their project, then check `/mcp`; no live Codex connection is claimed by this installer check. OpenCode V2 has format/merge tests, but no installed V2 runtime test. No model runs or global configuration changes were made.

The installer reuses `renderSetup` and adds no dependencies. Its config and filesystem helpers stay private to the executable package; no reusable installer framework was introduced. Gortex's installer contract check also reports broad impact through CLI dispatch; the relevant race tests pass. Its suggested parameter-object abstraction was unnecessary for this direct command.

## Gortex and engineering review

The user explicitly authorized tracking this repository; its Gortex name is `bearing`. Source discovery and manual edits used native Gortex. Its change checks identify test coverage for all eight reviewed core operation targets. No custom Gortex guard rules are configured. The change contract reports a broad-impact warning for the initial implementation; it is not an approval or a correctness certificate. Relevant race/process tests pass.

Gortex reported no Go LSP provider for formatting; Go's standard formatter was used for formatting only. Generated dependency checksums and binaries came from Go tooling.

Self-review against ENGINEERING.md:

1. Single-use logic stays in its owning file. The one MCP response envelope is needed by eight typed SDK handlers; it is not a general abstraction layer.
2. CLI and MCP share one implementation of each operation. Gortex reuse queries found no prior application implementation; policy, repository resolution, receipt handling, and shared SQL reads are reused.
3. No speculative plugin, manager, provider, ORM, or orchestration layer was introduced.
4. Modules follow the root-package/executable layout and stay under the 300-line guideline after formatting. Tests remain adjacent to the behavior they exercise.
5. The scope covers the current durable workflow and its operational checks. Code quality is reviewed alongside behavior; a short result or passing agent-submitted review alone is not proof.

## Remaining release gates

- Run fresh receiving-agent sessions for all six Codex/Claude/OpenCode directions, using native Gortex and the generated workflow. Include branch changes, dirty files, context resets, and policy reload.
- Run the planned controlled pilot comparing Gortex alone, Gortex plus a handoff note, and Bearing plus Gortex. Measure complete-session tokens/time, correctness, repeated investigation, and continuation quality. **No token-saving claim has been established.**
- Complete the remaining storage fault matrix before release, including interruption at commit, actual filesystem I/O exhaustion beyond the tested SQLite-full simulation, and connection replacement. Current tests cover specific failure modes, not every item in the research matrix.
- Keep supported-platform claims limited to macOS arm64 until other platforms are built and run.

The live-agent pilot is execution work, not more product planning. The current binary and setup material make those runs concrete. The initial preview and installer were committed and pushed to `main` as `9730eb2`. No release has been published.
