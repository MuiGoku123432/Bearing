# Bearing build plan

Version 0.1 plan · 21 October 2026 · Developer preview implemented; live-agent acceptance and comparative evaluation remain

## Current implementation

The engineer-environment installer is now implemented: `install.sh` builds the native executable, and `bearing install` applies per-project client integration with preview, backups, and repeat-install checks. It reuses the existing setup renderer and adds no dependencies. See [INSTALL.md](INSTALL.md) for usage and supported configuration formats, and [IMPLEMENTATION.md](IMPLEMENTATION.md) for verification limits.

The Go executable, SQLite persistence, eight typed MCP tools, CLI, engineering policy, research/spec/build workflows, setup generation, and recovery commands are implemented. The eighth tool, `bearing_git`, was added after this plan's original scope at the user's request. It works on branches in the main checkout: Conventional Branch names from the trunk, no commits to the trunk, and staged-only Conventional Commits. Bearing refuses linked worktrees. See [README.md](README.md) for build and use, [CLI_EXAMPLES.md](CLI_EXAMPLES.md) and [GIT_WORKFLOW.md](GIT_WORKFLOW.md) for concrete requests, and [IMPLEMENTATION.md](IMPLEMENTATION.md) for verification and remaining release gates.

The deterministic tests cover cross-process CLI writes, MCP subprocess ↔ CLI handoff, retries, rollback, scoped memory, document lineage, completion validation, and restore. These establish software behavior. They do not replace the fresh Codex/Claude/OpenCode agent runs or token comparison in steps 3, 4, and 6.

Research and specification are first-class task kinds. Agents author Markdown through their normal tools, save explicit document snapshots with checkpoints or finish, and pass immutable document IDs into the next task. Research → spec → build is available when useful; it is not a mandatory sequence for small changes.

Build a local coding-agent tool for Codex, Claude Code, and OpenCode. Bearing supplies built-in engineering policy, useful task context, durable decisions, progress recovery, and an honest completion record. Agents use Gortex to understand and change the code they are working on.

This document defines the build sequence and completion criteria. [CONTRACT.md](CONTRACT.md) defines the v0.1 behavior and data. [DESIGN.md](DESIGN.md) explains the product boundaries. The [research](research/README.md) supplies the rationale. [ENGINEERING.md](ENGINEERING.md) remains the unchanged, canonical policy.

## The first complete user experience

> Start a feature in Codex. Bearing supplies the engineering rules and relevant project decisions. The agent finds existing behavior through Gortex, makes a small change, runs checks, and saves the unresolved work. Open a fresh Claude Code or OpenCode session in the same checkout. The new agent recovers the next action, revalidates code-dependent evidence, finishes the change, and records actual verification and self-review.

This is the acceptance story for v0.1. Database persistence alone does not complete the product.

## What makes this a coding-agent tool

| Capability | Concrete behavior |
| --- | --- |
| Built-in engineering policy | Full canonical text embedded in the binary, identified by its content hash, returned at task start/resume |
| Useful task context | One response combines outcome, acceptance criteria, latest progress, relevant memories, unresolved failures, and next action |
| Gortex cooperation | The workflow directs native code discovery, reuse, impact, edits, and post-change checks under the active Gortex policy |
| Practical execution guidance | One maintained workflow tells agents when to investigate, change, verify, checkpoint, and finish |
| Durable progress | Explicit task versions, atomic checkpoints, and safe repeated write requests |
| Honest completion | Completion requires acceptance results, verification evidence, and the five engineering self-review outcomes |
| Cross-client continuity | All three hosts use the same executable and checkout database |
| Efficient context | Bounded recall, exact-ID expansion, visible omissions, and no automatic transcript ingestion |
| Usable interfaces | Typed MCP tools, readable CLI output, machine-readable JSON, and actionable errors |

Bearing's executable enforces its data and state contracts. The host agent follows the engineering procedure, calls Gortex, and runs checks. Behavioral evaluations verify that this combination works. Bearing does not claim independent proof of engineering quality from an agent's submitted review.

## Scope for v0.1

**Include:**

- One Go executable, local CLI, and stdio MCP.
- Embedded engineering rules and one embedded coding workflow.
- Git-checkout binding and `.bearing/state.db`, excluded from version control.
- Tasks, checkpoints, memory correction/retraction, bounded recall, and write receipts.
- Eight agent operations: `start`, `resume`, `remember`, `recall`, `checkpoint`, `finish`, `git`, `rules`.
- Human maintenance commands for initialization, status/history, setup output, export, backup, and restore into an empty destination.
- Tested setup material for Codex, Claude Code, and OpenCode.
- Storage failure tests, host handoffs, and a small comparative evaluation.

**Defer:**

- A separate model runtime, model routing, or autonomous subagent coordinator.
- Embeddings, a second code index, and a general knowledge graph.
- Remote MCP hosting and cross-machine synchronization.
- Linked worktree support; Bearing works on branches in the main checkout.
- Automatic transcript extraction, compaction hooks, background watchers, and a plugin framework.
- Automatic editing of arbitrary host configuration files.

Bearing refuses linked worktrees, because removing a worktree deletes its ignored state; work happens on branches in the main checkout. A branch change retains the database and triggers applicability checks. The first development/acceptance platform is macOS arm64, matching the current workspace. Claim other platform support only after running the relevant binary and tests there.

## Dependencies and readiness

| Item | Decision | Reason / prerequisite |
| --- | --- | --- |
| Go standard library | Use for command parsing, JSON, SQL access, embedding, hashing, paths, and tests | No CLI framework, ORM, or orchestration dependency is needed |
| `modernc.org/sqlite` | v1.60.1, approved and pinned | CGo-free SQLite 3.53.4 verified in the built binary; FTS5 and physical snapshot tests pass |
| `github.com/modelcontextprotocol/go-sdk` | v1.8.0, approved and pinned | Maintained protocol implementation, stdio transport, and generated input/output schemas |
| Gortex | Existing external companion | Source work must have graph coverage and follow its active policy |

The user approved implementation from this plan, confirmed the native Go SQLite choice, and explicitly authorized tracking Bearing with Gortex. The two planned packages are now installed and pinned. No CLI framework, ORM, model runtime, or additional architecture layer was added. The MCP adapter uses one concrete-purpose generic response envelope across all eight typed SDK handlers so their success/error output schemas remain precise; it is transport code, not a general framework.

At the start of implementation, resolve compatible stable releases, record exact versions and the Go toolchain, and pin them in `go.mod`/`go.sum` after confirmation. Verify the built binary's SQLite version contains the WAL-reset fix described in the [storage research](research/02-go-sqlite-storage.md). Do not guess versions from an unpinned documentation page.

This workspace is tracked by Gortex as `bearing`, under the user's explicit authorization. Native Gortex is used for source discovery, editing, and change checks. Disposable test repositories are not registered. Gortex currently has no Go LSP provider for formatting, so the standard Go formatter is used for formatting only.

These are the only identified pre-build approvals. Do not repeatedly ask to approve the already-selected architecture or ordinary implementation details.

## Code organization

There is no existing application source layout to preserve. Start with a small root Go package and one executable. Create files as their step requires them:

| Location | Responsibility |
| --- | --- |
| `go.mod`, `go.sum` | Pinned module and dependencies |
| `policy.go` | Embed root `ENGINEERING.md` and the maintained workflow; return policy identity |
| `WORKFLOW.md` | Concise coding procedure consumed by agents; authored during implementation |
| `store.go` | Open/initialize SQLite, connection settings, and migrations |
| `repo.go` | Resolve and bind a Git checkout; collect limited Git context |
| `tasks.go` | Task/checkpoint operations and completion validation |
| `memory.go` | Save/correct/retract and retrieve scoped memories |
| `cmd/bearing/main.go`, `cli.go` | Process entry and CLI parsing/rendering |
| `cmd/bearing/mcp.go` | MCP schemas and adaptation to the same operations |
| `cmd/bearing/setup.go` | Render host-specific setup artifacts from maintained sources |
| `recovery.go` | Consistent export, backup, and restore |
| Adjacent `*_test.go` files | Tests for the behavior in those files |
| `testdata/` | Small fixtures used by tests or host acceptance runs |

This is a responsibility map, not a scaffold to generate in advance. Split files only along actual seams if they become hard to understand. Do not create generic `manager`, `provider`, `repository`, `utils`, or transport-abstraction packages.

Both interfaces call the same concrete domain operations. Do not duplicate the implementation for CLI and MCP. Apply the rule of three to incidental helpers and query Gortex before adding symbols.

## Build steps

Complete these in order. Each step ends in working behavior and relevant checks, not another planning phase.

### 1. Executable, embedded policy, and a durable task

**Implement**

- Confirm dependencies and Gortex coverage; establish the module.
- Build `version`, `rules`, `init`, `start`, `checkpoint`, `resume`, and `status` through the CLI.
- Embed ENGINEERING.md verbatim and report its SHA-256 identity.
- Bind commands to the Git top-level checkout; reject missing, bare, or unwritable working directories.
- Initialize the database explicitly. Add only the missing `.bearing/` ignore entry without disturbing existing ignore rules.
- Store task acceptance criteria and append-only checkpoints. Return the full policy in start/resume context.
- Add expected-version updates and durable request receipts from the first write implementation.

**Proof**

- Embedded rules exactly equal the source file.
- A new process resumes the same task, checkpoint ID, version, and next action.
- Concurrent updates cannot silently overwrite progress.
- Retrying a committed request returns its original mutation outcome.
- Invalid repository and failed writes produce actionable errors.

**Deliverable:** an engineer can stop the CLI process and recover a real unfinished task.

### 2. Useful memory and coding completion

**Implement**

- Add `remember`, `recall`, `finish`, and checkpoint history.
- Implement project versus checkout applicability, provenance, explicit supersession, and retraction.
- Add exact anchor lookup and FTS5 lexical retrieval with literal-query handling.
- Return bounded context with omission information and exact-ID expansion.
- Add acceptance-result, verification, and engineering-review validation to `finish`.
- Preserve user-directed outcome changes as attributed, versioned task updates.
- Detect known checkout/revision mismatches; conservatively flag dirty-state evidence for revalidation.

**Proof**

- A correction replaces the old claim in normal recall while preserving history.
- Wrong-scope results cannot silently become current project facts.
- Relevant memories survive distractors within the configured response budget.
- Missing acceptance evidence or an unresolved review finding prevents completion.
- Historical failed attempts remain readable after a later successful verification.
- A changed user request updates the current criteria while preserving their previous definition.
- Parameterized SQL and literal FTS queries handle paths, punctuation, and Unicode.

**Deliverable:** the tool can preserve useful engineering knowledge and distinguish progress from completion.

### 3. MCP and one working coding-agent integration

**Implement**

- Add `bearing mcp --repo <absolute-checkout>` using the confirmed SDK.
- Expose the agent operations with exact input/output schemas and clear descriptions.
- Keep MCP stdout free of startup text and diagnostics.
- Author and embed WORKFLOW.md: orient, inspect through Gortex, make the next useful change, verify, preserve.
- Render the first setup bundle for Codex, including MCP configuration, a minimal instruction entry point, and the workflow skill.
- Use the existing native Gortex connection; do not add a Gortex proxy inside Bearing.

**Proof**

- CLI and MCP produce equivalent persistent effects and error codes.
- A real Codex session receives full policy and chooses the correct tools.
- The agent completes a reuse fixture with native Gortex evidence and real checks.
- A pure Gortex localization request still obeys terminal `answer_ready`.
- Process restart, protocol errors, cancellation, and replay have correct behavior.

**Deliverable:** a usable coding-agent tool in one real consumer. This is the first working product milestone.

### 4. Cross-consumer setup and handoff

**Implement**

- Render Claude Code and OpenCode setup bundles from the same maintained policy/workflow.
- Include exact target paths and configuration merge instructions; leave unrelated host settings intact.
- Verify startup from the repository root, a subdirectory, and a path containing spaces.
- Return concise candidate tasks when resume is ambiguous.

**Proof**

- Test all six directed handoffs among the three consumers with fresh receiving sessions.
- Receiving agents recover the same records without the sender's conversation.
- Repeat with a branch change and with uncommitted edits.
- Verify policy reload after context reset and check duplicate skill/result loading.
- Record exact host/model versions and actual limitations.

**Deliverable:** the acceptance story works across Codex, Claude Code, and OpenCode.

### 5. Recovery and release hardening

**Implement**

- Consistent logical export and physical backup.
- Restore into an empty destination, with version, integrity, and relationship validation.
- Migration locking and rejection of unsupported newer schemas.
- Bounded operation deadlines and reliable behavior after connection recreation.
- Build/version output that reports the embedded SQLite engine and policy identity.

**Proof**

- Run the relevant [S01–S16 storage cases](research/07-evaluation.md), including separate processes and lost responses.
- Verify restored record counts, IDs, scope, supersession, and task pointers.
- Keep the prior state intact after corrupt input or failed restore.
- Confirm stdout/exit behavior for human and JSON commands.

**Deliverable:** a release candidate with tested recovery, not just a successful demonstration.

### 6. Measure the benefit and publish the supported scope

**Execute**

- Run the proposed three-task pilot against Gortex, Gortex plus a handoff note, and Bearing plus Gortex.
- Keep engineering rules and relevant Gortex memory in every condition.
- Record correctness, policy adherence, continuation quality, repeated investigation, total token usage, and elapsed time.
- Review the actual diffs and explanations. Remove redundant guidance or operations when evidence supports it.
- Write a short usage guide with install/setup steps, examples, supported versions, and limitations.
- Build the tested release binary and record reproducible build instructions.

**Proof**

- No unresolved data-loss, wrong-repository-write, or false-completion defect in deterministic acceptance tests.
- All three consumers complete the handoff acceptance cases.
- Agent outcomes meet the engineering rules.
- Any token/time claims have measured, reproducible evidence.

**Deliverable:** v0.1 is ready for an explicit release decision. Repository publishing is a separate action, not implied by this planning request.

## Verification discipline

During implementation, run formatting and the relevant tests for each changed behavior. Run `go test ./...` and `go vet ./...` once the executable and package exist. Run the race detector on the supported platform for concurrency-related changes; process tests remain necessary because Go race detection does not establish SQLite interprocess correctness.

Use the active Gortex change contract before and after source mutation. Inspect the diff against all five engineering self-review checks. Fix defects before claiming the step complete.

Do not write tests that only mirror trivial wiring, and do not rerun broad checks without a change or unresolved concern. Host acceptance runs are needed for integration claims; a mock MCP client is insufficient.

## Ready-to-build checklist

- Product behavior and interfaces: [CONTRACT.md](CONTRACT.md).
- Implementation order and proof for each step: above.
- Canonical engineering policy: [ENGINEERING.md](ENGINEERING.md).
- Dependency proposal: two packages, with justification above.
- Dependency confirmation and explicit Gortex tracking authorization: received.
- Implemented scope and verification: [IMPLEMENTATION.md](IMPLEMENTATION.md).
- Remaining acceptance: fresh receiving-agent sessions across all three clients and the controlled comparative pilot. Do not claim measured token savings or a released v0.1 before those checks.
