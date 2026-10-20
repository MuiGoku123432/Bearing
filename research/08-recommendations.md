# Recommendations and implementation decisions

Research date: 20 October 2026.

## Keep the approved direction

Build Bearing as a Go CLI with SQLite and stdio MCP, alongside Gortex. Make the complete engineering standards part of the executable and the agent workflow. Optimize for useful continuation of real work and direct language.

Bearing should be a tool agents can reliably operate: explicit schemas, compact results, visible errors, durable IDs, bounded recall, correction history, and recoverable task state. A collection of Markdown prompts alone would not meet that requirement.

## Decision table

| Topic | Recommended decision | Status / reason |
| --- | --- | --- |
| Language and interfaces | Go executable, CLI and stdio MCP | User-approved direction |
| Durable store | SQLite | User-approved direction |
| Gortex relationship | Native companion for current code intelligence | User requirement; avoid duplicating the graph |
| Engineering policy | Embed canonical `ENGINEERING.md`; expose full content and policy identity; use it at start/resume/review | Explicitly confirmed by user during research |
| Memory extraction | Consuming agent proposes claims; Bearing validates/stores them | Proposal; avoids another model runtime |
| Memory authority | Keep source, scope, evidence, and replacement history | Proposal; persistence is not truth or instruction priority |
| Initial location | `.bearing/state.db` per checkout, Git ignored | Retain plan; test whether worktree separation fits real use |
| Concurrency | Short transactions, expected versions, durable request receipts | Proposal; needed for multiple clients and uncertain retries |
| Journal settings | Test WAL with FULL synchronization on local filesystems | Proposal; bundled SQLite must contain the WAL-reset fix |
| Retrieval | Scope and anchors first, lexical search next | Proposal; embeddings need a measured failure to justify them |
| MCP dependency | Official Go SDK | Candidate; exact package/version still needs confirmation |
| SQLite dependency | Prefer evaluating `modernc.org/sqlite`; compare `mattn/go-sqlite3` if necessary | Candidate; embedded engine and target builds unverified |
| Tool surface | Evaluate the current seven operations | Proposal, not a target to preserve regardless of evidence |
| Hooks | Add only for demonstrated missed-checkpoint problems | Deferred; host semantics differ |
| Cross-machine/worktree sharing | No implicit sharing in first version | Deferred; needs explicit scope and conflict semantics |

## Reconcile Gortex memory before adding more

Gortex already supports durable records, anchors, and supersession. The design should use those capabilities where active policy requires them and preserve the record's source.

Bearing's strongest differentiator is the task lifecycle: an atomic checkpoint, a known next action, concurrency behavior, and continuity across consumers. Implement and measure that before building elaborate memory extraction.

Existing Gortex claims should be referenced rather than copied into independently editable Bearing claims. If a requirement later calls for migration, define it explicitly. “Both use memory” is not sufficient justification for synchronization.

## How built-in engineering policy should work

Treat policy delivery as part of the product contract:

1. The binary includes the complete canonical file.
2. `rules` returns that content and a stable identity.
3. Start/resume and host entry points arrange for it to be loaded before coding.
4. Memory never overrides the rules or edits the embedded policy.
5. Completion preserves actual verification and a concise self-review outcome.
6. Evaluation checks the resulting behavior, including reuse and avoiding speculative abstractions.

A host-specific installer may produce thin entry points from the same maintained source. It must not replace existing project instructions wholesale.

Bearing can enforce input and state invariants directly. Engineering quality remains a combination of instructions, Gortex evidence, real checks, and review. Report agent assertions as assertions; do not imply that a policy version field certifies compliance.

## Build slices

These are implementation recommendations, not authorization to add new dependencies now.

### 1. Prove the local memory and checkpoint path

Confirm the concrete dependency proposal and Gortex coverage for implementation. Build only enough to:

- Return embedded engineering rules.
- Start a task and save a checkpoint.
- Save and correct a short memory with evidence and scope.
- Resume from a fresh CLI process.
- Demonstrate conflict, retry, and failed-write behavior.

Do not start with a generic framework or an entire generated package tree. Choose files when concrete responsibilities exist.

**Exit evidence:** known records survive reopen, correction hides the old claim from normal recall, and task/checkpoint writes are atomic.

### 2. Expose the same behavior through MCP

Use the supported SDK transport and the same concrete operations. Bind the process to one checkout. Implement clear errors, bounded output, and exact-ID expansion.

**Exit evidence:** CLI/MCP parity and clean stdio; a real consumer uses the tools correctly and receives the complete embedded rules.

### 3. Complete one real coding task with Gortex

Exercise a task that has an existing reusable implementation. Record relevant memory, inspect current evidence through Gortex, make the smallest change, run checks, and self-review.

**Exit evidence:** actual reuse and passing relevant checks, not just a filled-out task record.

### 4. Prove cross-consumer recovery

Run fresh-session handoffs and stale-evidence variants. Finish backup/restore and migration tests before relying on Bearing as the only copy of important work state.

**Exit evidence:** the next agent continues correctly; wrong-state evidence is flagged; acknowledged records survive the tested failures.

### 5. Measure and remove overhead

Compare against Gortex and Gortex with a plain handoff note. Inspect invalid tool calls, repeated investigation, missed facts, and total token/time cost.

**Exit evidence:** a documented benefit on the chosen tasks, or a narrower implementation that removes parts that did not help.

## Features to defer

| Feature | Reconsider only when |
| --- | --- |
| Embeddings or vector database | Measured semantic misses survive lexical/anchor improvements |
| Temporal knowledge graph | Real queries require relationships that simple scoped records cannot express |
| Autonomous summarization/consolidation agent | Manual structured capture fails enough to justify another model and its errors |
| Hosted service and remote MCP | Cross-machine access is an actual requirement |
| Automatic worktree sharing | Observed work patterns require it and scope/conflict semantics are defined |
| Multi-agent coordinator | Real tasks show value beyond existing host capabilities |
| Plugin framework | Several concrete integrations require a stable extension boundary |
| Full transcript ingestion | A demonstrated recovery case cannot be served by bounded checkpoints and evidence |
| Automatic deletion/decay | Retention needs and correction semantics are understood |

This is a decision record for current scope, not a permanent ban on these features.

## Questions the implementation experiments must settle

| Unknown | Cheapest useful evidence |
| --- | --- |
| Which driver release to pin? | Build candidate on target platforms; inspect dependency tree, embedded engine, FTS and backup behavior |
| Does per-checkout storage fit usage? | Exercise the user's normal worktree and branch workflow |
| Are seven tools easy to use? | Inspect tool selection and invalid-call patterns on a few real tasks |
| How should policy be delivered without duplication? | Inspect loaded instructions in fresh contexts on all three hosts |
| Does retrieval need semantics? | Known-answer paraphrase fixtures with measured misses |
| Are hooks necessary? | Count missed checkpoints during normal use |
| Does Bearing beat a handoff note? | Controlled interrupted-work comparisons |

Do not convert these into a questionnaire before the first useful experiment. Resolve what can be tested; ask the user only where a preference or approval genuinely changes the implementation.

## What this research changes in the plan

- Elevate built-in policy delivery and evidence-based review into explicit acceptance criteria.
- Make Gortex's existing memory part of the baseline and ownership design.
- Require a patched embedded SQLite engine before WAL use.
- Distinguish confirmed failed writes from writes whose responses were lost.
- Verify complete policy loading, skill duplication, and result duplication on each host.
- Evaluate against a simple handoff note, not just an agent without memory.
- Keep tokens and reliability claims unquantified until there are actual results.
