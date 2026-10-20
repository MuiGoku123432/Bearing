# Bearing requests

Use `bearing COMMAND --input FILE --json` or `--input -` for stdin. MCP tool arguments use the same operation fields. Replace example IDs and context with values from the current task. Never invent verification results.

## Research and spec inputs

Start a research task:

```json
{
  "request_id": "research-storage-1",
  "kind": "research",
  "outcome": "Choose the simplest storage approach for durable task checkpoints",
  "acceptance": [
    {"id": "a1", "description": "Compare viable options using primary sources and recommend one with its tradeoffs"}
  ]
}
```

The agent writes `research/storage.md` with findings, source links, uncertainties, and a recommendation. Capture it with `checkpoint`:

```json
{
  "request_id": "research-storage-checkpoint-1",
  "task_id": "RETURNED_TASK_ID",
  "expected_version": 1,
  "summary": "Compared the storage options and recorded the sources",
  "next_action": "Review the recommendation against the acceptance criterion",
  "documents": [
    {
      "path": "research/storage.md",
      "kind": "research",
      "summary": "Storage options and durability tradeoffs",
      "sources": ["https://www.sqlite.org/atomiccommit.html"]
    }
  ]
}
```

Use the returned document ID to start a spec:

```json
{
  "request_id": "spec-storage-1",
  "kind": "spec",
  "outcome": "Specify checkpoint and retry behavior",
  "acceptance": [
    {"id": "a1", "description": "Define success, retry, conflict, and failure cases with verification steps"}
  ],
  "input_documents": ["RETURNED_RESEARCH_DOCUMENT_ID"],
  "related_task_id": "RETURNED_RESEARCH_TASK_ID"
}
```

Write and capture a spec the same way, using `kind: "spec"` on its document. Start implementation with `kind: "build"` and the spec's document ID in `input_documents`. The resulting record keeps the research → spec → implementation lineage. Tasks do not have to wait on a formal phase gate; use the review appropriate to the work.

## Read saved document content

```json
{
  "document_ids": ["SAVED_DOCUMENT_ID"],
  "budget_bytes": 300000
}
```

Run this through `recall`. The larger explicit budget retrieves most large documents. JSON escaping counts toward this budget; use the logical export to inspect a body that cannot fit. The default budget is 12,000 bytes; omitted bodies are reported, while metadata remains visible. Stored bodies survive missing or changed working files.

## Save and correct durable memory

```json
{
  "request_id": "remember-storage-choice-1",
  "kind": "decision",
  "claim": "Use SQLite transactions to save the checkpoint and its request receipt together",
  "basis": "user",
  "source": "Approved implementation plan",
  "scope": "project",
  "anchors": [{"kind": "path", "value": "store.go"}]
}
```

Valid kinds are `decision`, `constraint`, `lesson`, and `preference`. Basis is `user`, `observed`, or `inferred`; it describes provenance rather than assigning an artificial confidence percentage. Scope is `project` or `checkout` within this store. Store facts about current source code in Gortex.

To replace a claim, add `action: "supersede"`, the old `id`, its `expected_version`, and a `reason`, alongside the replacement fields. The response contains the new memory ID. To retract without replacement:

```json
{
  "request_id": "retract-outdated-rule-1",
  "action": "retract",
  "id": "MEMORY_ID",
  "expected_version": 1,
  "reason": "The user explicitly withdrew this constraint"
}
```

Recall supports `query`, `anchors`, `limit`, and `budget_bytes`. Use `ids` to inspect a particular memory, including one that was superseded or retracted.

## Update requirements or stop a task

Use `checkpoint` with a current `expected_version`. An attributed requirement change includes:

```json
{
  "request_id": "scope-change-1",
  "task_id": "TASK_ID",
  "expected_version": 2,
  "summary": "Applied the user's scope clarification",
  "next_action": "Implement and verify the clarified requirement",
  "task_update": {
    "outcome": "Implement the clarified behavior",
    "acceptance": [{"id": "a1", "description": "The clarified behavior passes its acceptance check"}],
    "reason": "The original requirement was ambiguous",
    "source": "User clarification in this session"
  }
}
```

The checkpoint keeps the previous outcome and criteria. Use `status: "blocked"` with `blockers` when work cannot proceed; `status: "cancelled"` requires a `reason`. A completed or cancelled task stays closed. Start a related task for additional work.

## Finish with real verification

First run the relevant checks and review the actual diff under ENGINEERING.md. Use current repository context from `resume`; a known checkout or commit mismatch is rejected. Dirty files require the agent to revalidate the evidence.

```json
{
  "request_id": "finish-verified-task-1",
  "task_id": "TASK_ID",
  "expected_version": 3,
  "summary": "Implemented and verified the requested behavior",
  "acceptance_results": [
    {"id": "a1", "status": "pass", "evidence_ids": ["v1"]}
  ],
  "verification": [
    {
      "id": "v1",
      "kind": "test",
      "description": "Describe the acceptance check actually performed",
      "source": "The actual command, output reference, or observed result",
      "outcome": "pass",
      "exit_code": 0,
      "observed_at": "ACTUAL_OBSERVATION_TIME",
      "context": {
        "path": "CANONICAL_CHECKOUT_PATH_FROM_RESUME",
        "head": "CURRENT_COMMIT_FROM_RESUME_OR_OMIT_IF_UNBORN",
        "dirty": true,
        "captured_at": "ACTUAL_CONTEXT_TIME"
      },
      "acceptance_ids": ["a1"],
      "blocking": true
    }
  ],
  "review": {
    "policy_id": "CURRENT_POLICY_ID_FROM_RULES_OR_RESUME",
    "summary": "Describe the actual diff review and Gortex reuse check",
    "checks": {
      "single_use": {"status": "pass"},
      "reuse": {"status": "pass"},
      "necessity": {"status": "pass"},
      "conventions": {"status": "pass"},
      "minimality": {"status": "pass"}
    },
    "unresolved": []
  },
  "limitations": []
}
```

For research/spec tasks include the final Markdown in `documents` in this finish request. Manual evidence can omit `exit_code`. `not_applicable` is allowed on a review check only with a reason; acceptance criteria must pass. Finish validates the submitted evidence structure and known context, not the truth of an agent's claims.

## Retry and inspect

Reuse the identical request and request ID after an uncertain result. A saved retry returns `replayed: true` with the original IDs/version; changing the payload with the same ID is a conflict. A version conflict means another writer advanced the record: resume, reconcile, and use a new request ID for the new action.

`bearing history TASK_ID` returns up to ten checkpoints, newest first. Continue with `bearing history --before OLDEST_RETURNED_SEQUENCE TASK_ID`. `bearing status` lists up to twenty-one unfinished tasks. JSON exit codes: 2 for invalid input/missing state or record, 3 for conflict, 4 for busy/uncertain write outcome, 1 for other failures.
