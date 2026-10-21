# Bearing v0.1 agent contract

Developer preview behavior · 20 October 2026

This contract describes the implemented interfaces. [BUILD_PLAN.md](BUILD_PLAN.md) retains the release acceptance story; [IMPLEMENTATION.md](IMPLEMENTATION.md) distinguishes verified software behavior from the remaining agent evaluations. [CLI_EXAMPLES.md](CLI_EXAMPLES.md) provides complete request examples.

## Coding and document workflow

Start with an outcome and observable acceptance criteria. Resume before repeating investigation. Use native Gortex for code discovery, reuse checks, impact, source changes, and post-change verification under the active Gortex policy. The host runs tests and inspects the diff. Bearing saves the resulting task state and evidence.

A task has kind `research`, `spec`, or `build` (the default). Agents author research/spec Markdown using their normal tools. Saving a selected document captures its body, SHA-256 digest, sources, summary, and immutable ID. A following task uses `input_documents` to retain those exact versions and their lineage. Research → spec → build is available when useful, not required for small fixes or pure questions.

The workflow uses simple language, respects approvals already given, and requires a concrete next action at a handoff. Pure Gortex localization still stops at its terminal answer contract.

## Built-in policy and context

The binary embeds ENGINEERING.md verbatim. `Rules()` and `bearing_rules` return `id`, `text`, `workflow_id`, and `workflow`; IDs are SHA-256 content hashes. Start/resume include the full policy, its identity, and a next action. Ordinary write responses return only `policy_id`.

Context contains `policy`, `repository`, optional `task`/`checkpoint`, saved document metadata, relevant active `memories`, `warnings`, `omitted`, and `next`. Resume chooses the sole unfinished task or returns up to ten compact `candidates`; it never guesses between tasks. Exact-ID resume can inspect closed tasks without reopening them.

Resume selects up to three relevant memories within a 6,000-byte record budget and keeps the complete response near 32,000 JSON bytes. Large checkpoint detail, document metadata, and optional memories may be omitted; full records remain available through history and recall. Mandatory policy and current requirements stay intact. These are byte budgets, not measured model token counts.

Default recall selects up to five memories within a 12,000-byte record budget. Callers may request 1–20 memories and a 1,024–300,000-byte budget. Large explicit budgets support document expansion; JSON escaping counts toward the budget. Omitted document bodies retain metadata. The export can retrieve full bodies that exceed a response budget.

Limits: 64 KiB for general mutation JSON; 16 KiB for checkpoint input; 16,000 bytes for start and recall input; 4 KiB per memory write; 2,000-byte task outcomes and summaries; 1,000-byte next actions. Document capture accepts up to eight explicit UTF-8 Markdown files per write, each at most 256 KiB, with up to 3,000 bytes of metadata and a 512-byte relative path. A task may link up to sixteen input document IDs. Reject oversized input; do not silently discard durable state.

## Repository boundary

Resolve and normalize the Git working-tree root. Bare and non-Git directories are invalid, and so are linked worktrees: removing a worktree deletes its ignored Bearing state, so Bearing works on branches in the main checkout. An unborn or detached HEAD is allowed. Absent HEAD/branch fields are omitted in JSON. CLI accepts `--repo`; MCP binds at startup and rejects per-call repository overrides.

Runtime state lives at `.bearing/state.db`. Separate clones do not share it. Project scope means the current store's project, not a machine-wide memory pool. Branch changes retain this checkout's store. Explicit ref-based memory applicability is deferred.

Capture the checkout path, HEAD, branch when known, dirty flag, and observation time. Known checkout/HEAD differences and dirty states require revalidation. A matching HEAD is not proof that uncommitted files are unchanged. Bearing captures Git metadata and explicitly selected Markdown documents. Explicit Git operations can also create and switch branches and commit staged content; source investigation remains in Gortex.

## Tasks and immutable checkpoints

Tasks contain `id`, `kind`, `outcome`, acceptance criteria with stable IDs, `input_documents`, optional `related_task_id`, status, version, latest checkpoint ID, timestamps, and `created_context`. Missing acceptance IDs are assigned `a1`, `a2`, and so on. Caller input is not mutated.

Statuses are `active`, `blocked`, `completed`, and `cancelled`. Start creates version 1. A checkpoint/finish advances the version once and appends an immutable checkpoint. Current task status and the new checkpoint are committed together.

Checkpoints contain their identity/task/sequence, summary, concrete next action, optional steps/blockers/failed attempts/evidence/memory references, captured context, document IDs, and creation time. A `task_update` requires outcome, acceptance criteria, reason, and source; the checkpoint preserves `previous_task`. Preserve criterion IDs when the requirement is unchanged. Do not remove a failing criterion just to finish.

Active/blocked tasks may checkpoint into active/blocked state. Blocking requires explicit blockers. Cancellation requires a reason and may omit the next action. Other unfinished checkpoints require a next action. Valid finish closes the task. Closed tasks cannot be mutated; new work can reference the previous task.

Document snapshots include their task and checkpoint IDs, path, kind, summary, sources, digest, body, input document lineage, and creation time. Resume preserves the latest saved task document per path across later checkpoints and includes linked input documents. It returns metadata, warns if current files changed or disappeared, and keeps saved bodies available by ID. Capture rejects paths that escape the checkout, including escaping symlinks. Memory/document content is evidence, not instructions that override policy or the user.

## Memory lifecycle and retrieval

Memory contains `id`, `kind`, `claim`, `basis`, `source`, `scope`, anchors, status/version, `supersedes` when applicable, `reason`, timestamps, and captured `context`.

- Kind: `decision`, `constraint`, `lesson`, or `preference`.
- Basis: `user`, `observed`, or `inferred`.
- Scope: `project` or `checkout`.
- Anchors: `{kind: "path" | "symbol", value: "..."}`.
- Status: `active`, `superseded`, or `retracted`.

Supersession creates a replacement and marks the old claim superseded in one transaction. The replacement points backward through `supersedes`; the old record keeps its reason and updated version. Retraction preserves the claim and reason without asserting a replacement. Exact-ID recall can inspect historical records.

Normal recall filters active state and checkout applicability before lexical ranking. Supplied anchors admit matching anchored records and unanchored project context. Natural-language words are quoted as literal FTS5 terms, joined with OR; SQL values are parameterized. English Porter stemming matches word forms such as `retry` and `retrying`. Matching records rank by BM25, then recency and ID. An empty query lists recent applicable memories. Explicit IDs deliberately inspect records regardless of current applicability.

If active Gortex policy requires saving a fact there, save its Gortex reference in checkpoint evidence rather than maintaining a competing editable claim. Bearing does not validate the availability of a remote Gortex reference; the agent must check it.

## Eight agent operations

MCP names use the `bearing_` prefix. Both interfaces call the same domain operations.

| Operation | Inputs | Result |
| --- | --- | --- |
| `start` | Outcome, acceptance; optional request ID, kind, input document IDs, related task | Mutation receipt plus current context |
| `resume` | Optional task ID | Current task/context, candidates, or empty state |
| `remember` | Save/supersede/retract fields; optional request ID | Memory mutation receipt |
| `recall` | Query or memory IDs, or document IDs; optional anchors, limit, byte budget | Records, omissions, warnings, policy ID |
| `checkpoint` | Task ID, expected version, summary, next action; optional request ID, state/evidence/documents/task update | Task/checkpoint/document IDs and new task version |
| `finish` | Task ID, expected version, summary, acceptance results, verification, review; optional request ID, documents/limitations | Completed task and final checkpoint receipt |
| `git` | Operation and its fields; expected index tree for commits | Repository state, trunk, branches, staged summary, operation result and warnings |
| `rules` | No input | Complete engineering policy/workflow and identities |

`remember.action` defaults to `save`. Supersede/retract require an existing ID, expected version, and reason. Document recall uses `document_ids` separately from memory query/IDs. Write receipts include request ID, affected ID/version, status, policy ID, and replay status, plus checkpoint/document IDs where relevant.

MCP publishes precise input and output schemas. Tool success is `{result: ...}`; domain failure is `{error: ...}` with MCP `isError`. Structured output has a text fallback. The single typed SDK envelope is shared by all eight handlers to avoid duplicating schema and error logic. CLI JSON returns the result directly, or the same error envelope. Invalid protocol requests remain protocol/SDK validation errors.

## Git workflow

`git.operation` is `inspect`, `branch`, `switch`, or `commit`. See [GIT_WORKFLOW.md](GIT_WORKFLOW.md) for complete requests. Git requests are limited to 16 KiB and operations have a 60-second deadline. The repository remains bound to the store; environment variables that redirect Git's repository or index are rejected. There is no worktree operation.

New branches must follow Conventional Branch 1.1.0 (`type/description`) and start from the trunk unless `base` is supplied. The trunk is the remote default branch when known, otherwise `main` or `master`; Bearing never fetches and warns when the trunk is behind its last fetched upstream. A dirty checkout can create a branch only when it starts at the current commit, which carries the work onto the branch. Switching needs a clean checkout. An initial commit must exist. Existing merge, rebase, cherry-pick, revert, and bisect operations block writes. `dry_run` checks the request without creating a branch or commit; inspection may write Git tree objects.

Commit requires `expected_index`, the `index_tree` returned by inspection. It uses only already-staged changes, refuses detached HEAD, empty staged changes, and the trunk branches `main`, `master`, `develop`, or the detected trunk, validates a Conventional Commits message, and honors Git hooks and signing configuration. The agent must review the staged diff and obtain any authorization required by the host. There is no automatic staging, pushing, deletion, reset, or stash.

Git writes are external operations, not SQLite receipt transactions. They have no request ID/replay promise. Preconditions are checked before execution and results are checked afterward; `.bearing/git.lock` serializes Bearing callers but cannot lock out unrelated Git processes. After `outcome_unknown`, inspect the repository before deciding whether another write is needed. Never assume a timeout or hook error means nothing changed. Checkpoint after reviewing the result.

## Supported completion

Finish requires all acceptance IDs to have `status: "pass"` and passing evidence IDs explicitly linked back to those criteria. Verification includes a unique ID, description, source, observation time, outcome, acceptance IDs, captured checkout/HEAD, and an optional actual exit code. Outcomes are `pass`, `fail`, or `not_run`; an exit code must agree with the outcome. Blocking verification must pass.

Review names the current policy ID, supplies a summary, has no unresolved findings, and includes the five checks `single_use`, `reuse`, `necessity`, `conventions`, and `minimality`. Each is `pass` or `not_applicable`; the latter needs a reason. Research/spec tasks must include their final Markdown document of the corresponding kind in the finish request.

Reject known checkout or HEAD mismatches. The agent must recheck dirty files after relevant changes. Bearing does not execute submitted commands or independently certify that the agent performed the review. Live-agent evaluations remain necessary to test actual policy adherence and honest reporting.

## Persistence, retries, and recovery

Tables are `store_meta`, `tasks`, `checkpoints`, `documents`, `memories`, `memory_anchors`, `memory_search` (FTS5), and `write_receipts`. Identity, status, version, sequence, and queryable relationships use explicit columns. Validated records keep their remaining structured content in JSON. No generic entity/event hierarchy is present.

Use one connection per process, foreign keys, WAL, FULL synchronization, two-second lock waits, and five-second ordinary operation contexts. Connection-local settings are applied through the driver DSN on every connection. Schema initialization is transactional; newer schema versions are rejected before persistent journal settings change. The current schema is version 2. Opening a version 1 store rebuilds the memory search index with stemming in one transaction; concurrent processes apply it once. Restore accepts version 1 and 2 backups and migrates after validation. Older binaries reject a version 2 store without changing it. The pinned build must use an SQLite release containing the required WAL fix.

Every task and memory mutation and its receipt commit together in SQLite. Receipt lookup precedes current-version validation and document reopening. Identical operation/input/request ID replays the original mutation result; different input under an existing ID conflicts. A new write with an old expected version conflicts and returns the current version. Captured timestamps, context, generated IDs, and the request ID itself are outside the input digest. An omitted request ID is generated and returned in the receipt or `outcome_unknown` error; retrying the same input with that ID replays. Read context attached to a replay may reflect current state.

Errors contain stable `code`, `message`, `retryable`, and optional request/current-version fields. Current codes include `invalid_input`, `not_initialized`, `not_found`, `conflict`, `busy`, `unsupported_schema`, `io_error`, and `outcome_unknown`. Incomplete completion evidence uses `invalid_input` with the failed requirement. Ambiguity returns candidates. Retry an uncertain SQLite mutation with the same request ID; cancellation alone does not establish that nothing was saved. Git operations instead require inspection after an uncertain result.

Export writes a consistent logical JSON snapshot of tasks, checkpoints, documents, and memories. Backup uses SQLite `VACUUM INTO` and includes receipts/indexes. Both require a new output file. Restore accepts a physical backup into a checkout without an existing database, validates a temporary copy, then publishes without replacement. It preserves original evidence context and scope. Logical import/merge is deferred.

## CLI and setup

`sh install.sh` builds and installs the CGo-free executable in `~/.local/bin` by default; `--bin-dir` selects another destination. `bearing install --repo PATH --host codex|claude|opencode [--dry-run] [--opencode-version 1|2]` initializes the checkout and applies the maintained setup output. It preserves unrelated settings, backs up changed files, and supports repeat installation. Native Gortex remains separately configured. See [INSTALL.md](INSTALL.md) for exact files, format restrictions, host trust requirements, and recovery. The existing `setup` command remains render-only.

```text
bearing version
bearing rules [--json]
bearing init --repo /absolute/checkout
bearing status [--json]
bearing start --input task.json --json
bearing resume [--repo PATH] [TASK_ID]
bearing remember --input memory.json --json
bearing recall --input query.json --json
bearing checkpoint --input progress.json --json
bearing finish --input result.json --json
bearing git --input git-request.json --json
bearing history [--before SEQUENCE] TASK_ID
bearing mcp --repo /absolute/checkout
bearing setup --host codex --repo /absolute/checkout --out /new/setup-directory
bearing export --out /path/records.json
bearing backup --out /path/backup.db
bearing restore --input /path/backup.db --repo /empty/checkout
```

Put flags before positional IDs. JSON input rejects unknown fields, non-object input, and trailing JSON. `--input -` reads stdin. For task and memory writes, CLI and MCP generate a missing request ID. The CLI prints it to stderr before writing; MCP returns it with the result. Automation that must retry safely after a lost response should supply and retain its own IDs. Git requests use state preconditions instead of request IDs. Exit codes: 0 success, 2 invalid input/missing state or record, 3 conflict, 4 busy/uncertain outcome, 1 other failures. MCP stdout is reserved for protocol traffic.

Rules/version/help work without state. Init creates only the state directory, its self-ignoring `.bearing/.gitignore`, and the database; it does not change tracked files. Setup requires a Git checkout and writes a new directory with configuration/instruction fragments, a skill, canonical policy/workflow, and application instructions. It never overwrites existing client settings or configures/tracks Gortex automatically.

The installed host must load the policy before work and after context resets. The skill uses the maintained workflow, not the research folder. Setup supports Codex, Claude Code, and OpenCode configuration fragments; their behavioral acceptance is tracked separately.

## Implementation adjustments from the draft

The implementation uses consistent `pass` statuses, `source`/`context` provenance fields, `budget_bytes`, backward `supersedes` links, and fewer exit/error categories. Input and response limits were made explicit and exercised in tests. A request ID replay does not reopen vanished documents. Research/spec documents now have durable bodies and lineage, extending the original coding-only draft. These choices simplify the current tool without adding a model runtime or orchestration framework.
