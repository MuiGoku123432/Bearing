# Evaluation and failure tests

Research date: 20 October 2026. **This is a proposed evaluation plan. No Bearing benchmark, durability test, or host handoff has run.**

## What success means

Bearing succeeds when an agent continues the right work, respects the engineering standards, and achieves the requested outcome with less repeated investigation or recovery effort. Merely retrieving a memory is insufficient.

Evaluate the actual environment outcome as well as the trace. A message claiming completion is not proof that the desired state exists. Separate deterministic checks, human review, and optional model judgment. [Anthropic's agent evaluation guidance](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)

## Separate three kinds of tests

| Kind | Question | Method |
| --- | --- | --- |
| Storage and interface tests | Did Bearing preserve and return the correct state? | Deterministic assertions using temporary databases and separate processes |
| Host integration tests | Can each consumer discover, use, and interpret Bearing correctly? | Real CLI/MCP sessions at recorded versions |
| Agent outcome evaluations | Did memory and workflow improve the engineering work? | Controlled tasks, real verification, trace review, and repeated trials |

A good result in one does not imply a good result in the others. Fast retrieval can return the wrong memory. Correct storage can be wrapped in confusing tools. An agent can pass a task while ignoring the required reuse search.

## Fair baselines

All conditions retain the same user engineering standards and active Gortex policy.

| Condition | Included |
| --- | --- |
| A | Gortex, including its native memory where applicable |
| B | A plus a concise human-readable handoff note |
| C | A plus Bearing's task state, memory retrieval, and workflow integration |

Use the same repository state, task, model, reasoning settings, permissions, and available tools within each comparison. Record the host version and every intentional difference.

Start with a small pilot: three tasks in one consumer, three repetitions per condition. That is 27 task runs, not a statistical claim about the whole product. Inspect failures before spending on a larger evaluation. A smaller smoke test can precede the pilot.

Only expand to all consumers after deterministic failures are fixed. Do not compare one consumer's stronger model against another's weaker model and attribute the difference to Bearing.

## Initial task bank

Use synthetic repositories or permitted local projects with known outcomes. Each fixture has a resettable source state, seeded records, an expected result, and a relevant real check.

| ID | Scenario | Required result |
| --- | --- | --- |
| E01 | Existing equivalent implementation | Agent finds and reuses it before adding a new symbol |
| E02 | Small single-use change | Logic stays local; no shared abstraction is introduced |
| E03 | Two repeated blocks | No premature shared extraction; required duplication note is present |
| E04 | Resume after interrupted feature | Agent identifies actual completed work and performs the next unfinished action |
| E05 | Known failed approach | Agent uses the saved failure evidence instead of repeating it without reason |
| E06 | User corrects an earlier decision | New instruction is preserved; superseded memory is not presented as current |
| E07 | Branch changes after checkpoint | Agent notices mismatch and revalidates relevant code evidence |
| E08 | Uncommitted source changes | Unchanged HEAD does not cause old verification to be treated as current |
| E09 | Distractor memories | Relevant evidence remains retrievable under the same response budget |
| E10 | No supporting memory | Agent reports the absence instead of inventing a remembered fact |
| E11 | Conflicting memories | Scope/provenance are inspected; timestamp alone does not choose truth |
| E12 | Injected instruction in a memory | Stored content does not override user policy or authorize an unrelated action |
| E13 | Verification fails | Task remains unfinished with failure and next action recorded |
| E14 | New dependency proposed | Agent states justification and honors the user's confirmation requirement |
| E15 | Rule loaded, then context reset | Fresh context receives the full applicable built-in engineering rules |
| E16 | Simple help or lookup | No unnecessary task ceremony or forced checkpoint loop |
| E17 | Gortex terminal localization | Agent answers from terminal evidence and stops as required |
| E18 | Gortex unavailable | Memory remains available; code work honors the failure policy |

Adapt the five memory abilities emphasized by [LongMemEval](https://arxiv.org/html/2410.10813v2) into engineering-specific cases, particularly correction, temporal applicability, and abstention. Do not report results on this custom set as a LongMemEval score.

## Ground truth and grading

Define expected facts before the run. For a reuse task, record the existing implementation and acceptable extension. For correction, record which claim is valid for which scope. For a checkpoint, identify the required next action and known unfinished work.

Prefer:

- Real regression and acceptance checks for changed behavior.
- Database assertions for record history, versioning, and scope.
- Tool traces for required Gortex use and accidental repetition.
- Human review for abstraction quality, relevant scope, and clear language.

Do not grade exact phrasing unless formatting itself is a requirement. Do not force a single tool sequence when several policy-compliant paths exist. Do not use an agent's self-review as the sole independent grader of that same agent.

A simple review rubric can score each item as met, unmet, or not applicable:

1. Requested behavior is complete.
2. Existing implementation was reused where appropriate.
3. No unneeded abstraction or scope expansion was introduced.
4. Verification supports the reported result.
5. Saved state would let another engineer continue correctly.
6. The explanation states the result and limitations clearly.

Keep held-out fixtures for later changes to retrieval, tool descriptions, and instructions. Tuning on every available case can make the tool look better without improving general behavior.

## Metrics

| Metric | Definition |
| --- | --- |
| Task success | Runs satisfying the requested outcome and required checks / attempted runs |
| Policy adherence | Applicable engineering and Gortex obligations met, reviewed independently |
| Handoff success | Fresh receiver continues correctly from persisted state without the sender's conversation |
| Retrieval recall | Required relevant records returned within the configured budget / required relevant records |
| Retrieval precision | Useful applicable records / returned records |
| Wrong-scope rate | Results inappropriately used from another repository, checkout, or revision |
| Repeated investigation | Previously resolved facts re-investigated without a freshness or scope reason |
| Context cost | Host-reported input, cached input, output, and separately available reasoning usage |
| End-to-end time | Task start through verified outcome, including retries and recovery |
| Tool overhead | Bearing calls, invalid calls, result bytes, lock waits, and startup time |
| User burden | Unnecessary questions, repeated approvals, or avoidable manual recovery |

Report per-task results and variation, not just a favorable mean. Use medians and, only with enough samples, tail latency. Show the sample count with any percentile.

Include memory creation, correction, and checkpoint costs when comparing tokens. Savings measured only at the final recall step omit part of the work. Unknown usage fields are unknown, not zero.

Do not reward omitting important checks to reduce tokens. Correctness, scope isolation, and honest completion are prerequisites; efficiency is assessed after them.

## Storage and recovery cases

These are implementation tests and should not require paid model calls.

| ID | Fault | Assertion |
| --- | --- | --- |
| S01 | Two simultaneous processes update one task | One wins; stale update conflicts; no lost progress |
| S02 | Process terminated before commit | No partial checkpoint/task pointer |
| S03 | Process terminated after acknowledged commit | Records survive reopen |
| S04 | Commit succeeds but response is lost | Same request key returns original result without another write |
| S05 | Request key reused with different payload | Clear error; original result unchanged |
| S06 | Busy database exceeds deadline | Error, bounded wait, no false success |
| S07 | Disk full or directory becomes read-only | Explicit failure; valid prior records remain accessible where possible |
| S08 | Operation canceled during write | Outcome resolved through the same request key; no assumed rollback |
| S09 | Two processes start a migration | Migration is applied once with consistent resulting schema |
| S10 | Old executable opens newer schema | No modification |
| S11 | Backup while writes occur | Restored snapshot is internally consistent and contains its expected records |
| S12 | Export/restore | IDs, scopes, provenance, and supersession relationships survive |
| S13 | Corrupt or malformed import | Rejected before replacing working state |
| S14 | Connection replaced by pool | Foreign keys, timeout, and synchronization settings still apply |
| S15 | FTS syntax and path edge cases | Literal query behavior is safe and documented; index follows corrections |
| S16 | CLI/MCP parity | Same validation, mutations, IDs, and errors |

For crash tests, record whether failure happened before commit, after commit, or at an unknown point. Process termination does not test power-loss durability. Keep that limitation in the report.

## Host compatibility matrix

For each recorded host version, test:

- Local stdio discovery and a clean protocol channel.
- Correct repository binding from root and a nested folder.
- Complete built-in policy available in fresh and resumed contexts.
- Structured results and any text fallback.
- Useful error recovery after conflict and uncertain write outcome.
- Six directed handoffs across the three hosts.
- No duplicate skill/policy loading caused by the installation layout.

Use a disposable tracked fixture for Gortex integration with explicit tracking authorization. Reset only fixture state. Do not restart the machine-wide daemon or disturb unrelated repositories.

## Recording results

Keep a compact per-run record with fixture ID, source revision, seeded database identity, consumer/model versions, settings, outcome, check evidence, usage, elapsed time, and failures. Store only permitted traces and redact secrets.

For deterministic tests, record the exact build and embedded SQLite version. For agent evaluations, distinguish observations from interpretation.

The release decision should identify unresolved failures, not hide them behind a combined score. Require zero observed data-loss, wrong-repository writes, or false-success defects in the deterministic acceptance suite. That is a release criterion, not a mathematical guarantee that none can ever occur.

No savings percentage, reliability score, or compatibility badge is justified until these tests have results.
