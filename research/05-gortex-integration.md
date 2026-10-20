# Working alongside Gortex

Research date: 20 October 2026. Local evidence: installed Gortex guide, active session policy, and live capability schemas. Observed daemon version: `0.64.5+3310720`.

## The central finding

Gortex already exposes persistent memory concepts. Bearing's value cannot simply be “give the agent memory.” Its proposed contribution is a consistent task/checkpoint contract, explicit recovery behavior, a small shared interface across consumers, and the user's engineering workflow.

Prove that added value against a baseline that already uses Gortex's memory. Comparing Bearing against an artificially memoryless setup would overstate the benefit.

## What was actually observed

| Surface | Observed capability | What it does not prove |
| --- | --- | --- |
| `remember.memory` | Typed memories, file/symbol anchors, confidence/importance metadata, supersession links, workspace/global scope | Crash durability, transaction isolation, or reliable automated truth assessment |
| `recall.surface` | Task and anchor-based retrieval, result/excerpt bounds, superseded-record controls | Semantic recall quality on Bearing tasks |
| `remember.notebook` | Persistent Markdown notebook entries under `.gortex/notebook` | A task/checkpoint concurrency contract |
| `recall.distill` | Summarization of saved session notes with limits | Recovery of unsaved conversation content |
| Guide: token controls | Output limits, selected fields, body compression, conditional reads, truncation metadata | A measured reduction in Bearing's end-to-end token usage |
| Guide: views | Checkout overlays, explicit view selection, exact/fresh requirements, immutable ref views | Permission to edit through a fallback or wrong checkout |

These are metadata observations. Bearing's directory is not currently tracked; graph operations there are unavailable. No tracking, memory import, daemon restart, or mutation of another repository was performed for this research.

The guide contains older names in places. Treat mounted tool schemas as the callable interface. For example, the engineering rules' `smart_context / semantic search` wording expresses a reuse obligation, not a guarantee that a method with that exact name exists.

## Ownership without competing copies

| Information or action | Owner |
| --- | --- |
| Current source, symbols, callers, dependencies, and code reuse | Gortex |
| Source impact, guarded edits, and graph checks | Gortex under the active policy |
| Bearing task IDs, checkpoint versions, write receipts, and recovery | Bearing |
| Claims originally saved in Gortex | Gortex, referenced from Bearing where useful |
| Claims originally saved in Bearing | Bearing, with attributed evidence |
| Running tests, models, permissions, and sessions | Consumer and project environment |

There is an important policy interaction: the active Gortex rules can require immediately saving durable decisions with `remember`. Bearing's workflow must not silently override that instruction.

**Recommendation:** when policy requires a Gortex memory, retain that authoritative record and let the Bearing checkpoint link to it. Avoid creating a second independently editable claim. When Bearing owns the memory, keep its ID and provenance explicit. If deployment policy requires both systems to write, document the designated authoritative record and test correction behavior before claiming the two stay consistent.

Do not build bidirectional synchronization in the first version. Similar features do not imply identical record semantics. Import/export or migration would require a separate, explicit design.

## The normal coding path

1. Use Bearing to recover the task, constraints, next step, and evidence references.
2. Establish the current checkout and applicable rules.
3. Use Gortex's task flow for diagnosis or modification.
4. Search for equivalent symbols before creating anything new.
5. Follow the active impact/edit/post-change contract.
6. Run relevant real checks in the consumer environment.
7. Save the outcome and any new useful knowledge according to ownership and policy.

The current Gortex policy specifies impact before mutation, signature verification when changing a signature, Gortex editing/refactoring, then change detection and the associated test/guard/contract queries. Bearing should preserve that sequence rather than add a second wrapper around each step.

A graph tool returning test targets is not evidence that the tests passed. A saved checkpoint must distinguish proposed checks, executed checks, and their results.

## Localization has a different stopping rule

Pure discovery requests use Gortex's localization flow. Once it returns terminal `answer_ready`, the active contract requires answering from that evidence and stopping.

Do not require a Bearing write after that terminal response. Also avoid starting localization as the first step of a coding task that will need continued diagnosis and edits; the task flow is the appropriate route.

This is a concrete reason to avoid a rigid “every request runs all workflow stages” design.

## Freshness and checkout routing

The local guide documents automatic checkout overlays and explicit view selection. The selected overlay plus its designated primary is not a union of every branch. A response with `exact:false` means the requested view was not served; fallback reads do not authorize edits.

Bearing should store references sufficient to request the correct evidence again:

- Repository-relative path and optional symbol ID.
- Task checkout and observed revision, when available.
- Whether uncommitted work existed.
- What question the evidence answered.

Do not assume a stored symbol ID remains resolvable after refactoring or reindexing. A failed reference requires fresh localization in the appropriate task flow, not a cached source body treated as current.

For changes that require exact/fresh evidence, use Gortex's documented `require_exact`, `require_fresh`, and bounded `wait_deadline` controls where supported. Detect a mismatch explicitly. Do not use re-tracking or daemon restarts as a refresh strategy.

## Token efficiency through Gortex

Use bounded results, relevant fields, and compressed bodies when they still answer the question. Expand only the source needed for the next decision. Respect truncation and freshness metadata.

Conditional reads can avoid returning unchanged content when the active API exposes the relevant mechanism. A cache key must include the selected view and evidence identity; “same path” is not enough across branches.

Bearing should preserve references and the decision derived from evidence, not full source dumps. A duplicate source cache would create another freshness problem and dilute the purpose of Gortex.

## Failure behavior

| Condition | Bearing behavior | Coding behavior |
| --- | --- | --- |
| Repository not tracked | Its own memory can remain available | Report unavailable graph coverage; follow active policy |
| Configured Gortex tools missing | Do not pretend code evidence was checked | Report integration failure and stop the affected code work |
| Requested view not exact | Preserve task/checkpoint | No edits based on substituted evidence |
| Graph warming or incomplete result | State uncertainty | Do not equate zero results with no equivalent implementation |
| Gortex unavailable after earlier progress | Save permitted task state if useful | Do not bypass the policy with shell source searches |

Automatic startup must not take over the machine-wide daemon. Explicit tracking is a user action, and ordinary linked-worktree overlays should not be registered as new logical repositories without the appropriate request.

## What still needs testing

Use a disposable project with authorized tracking. Demonstrate reuse, exact checkout selection, stale-reference handling, policy-compliant edits, verification, and cross-client continuation.

Compare three practical setups:

1. Gortex with the engineering standards.
2. Gortex with the standards and a short manual handoff note.
3. Gortex with the standards and Bearing.

If the third does not materially improve continuity or reduce repeated work, simplify Bearing's scope. SQLite and MCP are approved choices; they do not by themselves establish product value.
