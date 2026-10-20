# Durable memory and useful retrieval

Research date: 20 October 2026. Recommendations are design proposals, not implemented behavior. See [the source ledger](SOURCES.md) for evidence and limits.

## What Bearing should remember

Bearing's useful unit of memory is a short claim with a reason to trust it and a clear scope. Conversation storage alone does not provide that. A transcript contains abandoned ideas, temporary observations, instructions, and corrections mixed together.

Separate three things:

| Information | Example | Treatment |
| --- | --- | --- |
| Durable project knowledge | “Keep the payment calculation in integer cents because the public API accepts cents.” | Memory with a source and applicability |
| Work in progress | “Parsing is fixed; the regression test still fails; inspect the rounding boundary next.” | Task checkpoint |
| Current source behavior | “This function calls the decimal converter.” | Retrieve through Gortex and validate against the active checkout |

The engineering standards remain canonical policy in [ENGINEERING.md](../ENGINEERING.md). A remembered summary must never replace them.

**Recommendation:** let the consuming agent propose a structured memory. Bearing validates, stores, retrieves, and tracks its history. Do not put another model call inside the storage service. This keeps persistence deterministic and avoids paying a second model to reinterpret every message.

## What the research establishes

| Work | Useful finding | Limit on applying it to Bearing |
| --- | --- | --- |
| [LongMemEval](https://arxiv.org/html/2410.10813v2) | Separates extraction, reasoning across sessions, temporal reasoning, updates, and abstention. It also separates indexing, retrieval, and reading. | Conversation question answering does not establish software engineering success. |
| [LoCoMo](https://arxiv.org/abs/2402.17753) | Evaluates long conversations with temporal and cross-session demands. | Remembering dialogue differs from verifying changing source code. |
| [MemGPT](https://arxiv.org/html/2310.08560v2) | Explores moving information between limited context and external memory. | Bearing can use selective loading without adopting its agent runtime. |
| [Mem0](https://arxiv.org/abs/2504.19413) | Studies extraction, consolidation, and retrieval of compact memories. | Reported improvements are results for the authors' setup, not promised Bearing savings. |
| [Zep](https://arxiv.org/html/2501.13956v1) | Distinguishes when information was recorded from when it was valid; preserves relationships to evidence. | A temporal graph and model-driven contradiction resolution add costs not yet justified here. |

These sources support treating memory as a lifecycle, not an append-only pile. They do not establish that Bearing needs embeddings, a knowledge graph, autonomous consolidation, or a separate memory agent.

## Proposed record contract

Use the fields needed to answer “what is this, why should I use it, and does it apply here?”

| Field group | Minimum purpose |
| --- | --- |
| Identity | Stable record ID; record version for conflicting updates |
| Claim | One understandable decision, constraint, lesson, or correction |
| Basis | User statement, observed result, or inference; reference to its source |
| Applicability | Project-wide, checkout-specific, or tied to a revision or task |
| Anchors | Relevant repository-relative paths; optional Gortex symbol references |
| Observation context | Commit when available, dirty-state indicator, time observed |
| Lifecycle | Active, superseded, or retracted; replacement link and reason |

“Verified” must mean a recorded observation, not a model's confidence adjective. Store the kind of observation and its limits. A passing test supports the tested behavior at the recorded state; it does not prove the entire system correct.

Do not require every memory to have a code anchor. A user's naming preference is valid without a symbol. Conversely, an implementation claim without evidence should remain a hypothesis in a checkpoint.

Use recorded time initially. Add separate validity dates only for facts that actually need them. Bearing's first version does not need a general temporal database to distinguish “saved yesterday” from “verified at commit X.”

## Scope before relevance

Apply scope restrictions before ranking. A perfectly matching record from another repository is still the wrong result.

The proposed database location, `.bearing/state.db`, gives clients in the same checkout shared local memory. It does not share data across separate clones or worktrees. That limitation must be explicit in setup and help.

Within one checkout:

1. Project-wide decisions may remain relevant across branch changes.
2. Checkout observations carry branch/revision context and may need revalidation.
3. Task checkpoints identify the task and code state they refer to.
4. Missing or ambiguous scope must be visible; do not silently treat it as global.

A branch name is a label, not a stable snapshot. The same name can point to new code; uncommitted edits can change behavior without changing the commit. Use both Git context and fresh Gortex evidence.

On a mismatch, return the checkpoint as useful historical context with `needs_revalidation`. Do not discard the next action, and do not present old verification as current.

## Correction and contradiction

Example:

1. Memory A says “The integration test requires a network connection.”
2. A later verified change makes the test use local fixtures.
3. Save memory B with the new evidence and supersede A in one transaction.
4. Normal recall returns B. History can still explain A and why it changed.

A newer timestamp alone is insufficient to invalidate an older claim. Two observations can refer to different branches or environments. If the agent cannot establish a replacement, preserve the disagreement and flag it. Do not use last-write-wins as semantic truth.

Retraction means “do not use this claim.” It is different from replacing it with a new claim. Retraction is also different from erasure: retaining history still retains sensitive content. A future erase command must explain its treatment of backups and exports.

Repeated successful use may be a useful ranking signal later. It should not automatically make a claim authoritative: repeated retrieval can reinforce an initial mistake.

## Retrieval that fits the task

Start with deterministic retrieval:

1. Select applicable active memories and the requested task.
2. Prefer explicit record IDs, task links, and exact path/symbol anchors.
3. Search the remaining short claims and titles by text.
4. Return the most relevant bounded set with evidence, scope, and IDs.
5. Allow expansion of selected records through the same recall operation.

Prefer a stable ordering to an elaborate weighted scoring model. A reasonable first ordering is explicit matches, anchored matches, then lexical matches with deterministic ties. Recency breaks ties; it must not defeat scope or a known correction.

[SQLite FTS5](https://sqlite.org/fts5.html) supplies lexical retrieval, ranking, and snippets. Its search expression has a grammar independent of SQL parameters. Its BM25 implementation ranks better matches with lower numeric scores. External-content indexes need synchronization with their source rows.

**Recommendation:** begin with ordinary indexed scope/anchor queries and a small FTS5 index only if the driver supports it in the tested builds. Treat user search text as literal terms by default, escaping FTS syntax. Test paths, underscores, dotted identifiers, hyphens, Unicode, empty queries, and malformed input. Do not add an embedding dependency to hide basic tokenization problems.

A result should say why it matched and whether anything was omitted. The following is an illustrative contract, not a finalized schema:

```json
{
  "records": [{
    "id": "mem-42",
    "claim": "Use integer cents at the API boundary.",
    "basis": "observed",
    "evidence": "API contract test at recorded revision",
    "scope": "project",
    "freshness": "needs_revalidation",
    "matched_by": "path"
  }],
  "truncated": true,
  "omitted_count": 7
}
```

An empty result means no qualifying record was found. It does not prove that the repository has no constraint or equivalent implementation.

## Context budgets

Optimize useful evidence per response, not the number of stored records. Keep the current outcome, next action, unresolved failure, and applicable constraints ahead of optional historical detail. Never silently truncate the engineering policy.

The current plan's 1,000-token guidance and 1,500-token memory targets are starting hypotheses. Measure them with each consumer. Without the model's tokenizer, a byte or character count is not an exact token count. Use a hard byte limit to bound output and label any token estimate.

If a response is too small to include necessary context, return a visible truncation marker and a way to expand it. Do not compress contradictory facts into a falsely confident summary.

Repeated full-history summarization is a poor default for this product: each rewrite can lose qualifiers and costs model work. Prefer a current checkpoint with links to selected prior evidence. Save new durable facts at the time they become useful, not only when the context window fills.

## When to consider embeddings

First create retrieval cases whose correct memories are known. Compare exact anchors plus lexical retrieval against a small semantic candidate implementation only if paraphrase misses remain material.

An embedding proposal needs to demonstrate improved task outcomes after including embedding generation, model availability, index maintenance, stale-vector handling, latency, and package size. Better nearest-neighbor scores alone do not establish value.

Unresolved measurements: retrieval recall under noise, false matches between branches, useful context per byte, and recovery accuracy after a correction. See [the evaluation plan](07-evaluation.md).
