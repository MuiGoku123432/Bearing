# Workflow, engineering standards, and clear language

Research date: 20 October 2026. The user-defined [engineering standards](../ENGINEERING.md) are requirements. Research informs how Bearing delivers them; it does not vote them away.

## Engineering policy is built into the product

The user explicitly confirmed this requirement during research: `ENGINEERING.md` must be part of Bearing itself and shape how the agents and platform work.

Embed the canonical file in the Go executable. Keep its complete text available through `bearing rules` and the corresponding MCP operation. Identify the embedded policy by version or content hash so a consumer can detect a changed policy. Do not store an editable second policy in the memory database.

Start/resume behavior and host instructions must ensure the agent receives the applicable policy before coding. A policy ID alone is not the policy; if the content has not been loaded in the current context, retrieve it. After a context reset, do not assume an earlier session's acknowledgement means the rules are still available.

Use the rules in three concrete ways:

| Mechanism | Bearing's responsibility | Limit |
| --- | --- | --- |
| Built-in guidance | Supply the full rules, clear tool descriptions, and a short work/review procedure | Loading instructions does not prove compliance |
| Direct validation | Reject malformed writes, stale updates, wrong scope, unsupported schemas, and incomplete required records | Storage validation cannot assess the design of a code change |
| Evidence at completion | Preserve relevant verification and the outcome of the five required self-review checks | Agent-reported review is attributed evidence, not independent certification |

Completion should capture one concise review result with references or findings where needed. It should not demand five essays or manufacture a passing result. An unresolved violation remains unfinished work.

Host-specific hooks could later guard selected tool actions if supported and justified by failures. They are not a universal enforcement mechanism, and Bearing must not claim it can control every shell or tool call in all three products.

This makes the standards an operational product feature: distributed consistently, used during work, represented in completion evidence, and exercised by acceptance tests.

## A workflow should make the next action easier

Use a short loop:

**Recover the task → understand the relevant system → make a small change → verify it → preserve what matters.**

A one-line correction does not need a planning ceremony. A multi-session change needs a recoverable outcome, known constraints, and a useful next action. The amount of saved state should follow the work.

Long-running agent guidance describes failures caused by attempting too much at once, leaving unclear partial work, and mistaking partial completion for success. Progress records and incremental verification help address those failures in the reported setting. [Anthropic's long-running agent guidance](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)

**Bearing recommendation:** keep the recovery mechanisms, without imposing a dedicated initializer agent or fixed milestone process.

## What to borrow from GSD

[GSD's architecture](https://github.com/gsd-build/get-shit-done/blob/main/docs/ARCHITECTURE.md) provides a reference for persistent planning state and bounded execution context. It also describes a broader orchestration system than Bearing currently needs.

| Idea | Bearing treatment |
| --- | --- |
| Work is resumable across sessions | Keep |
| Steps have observable outcomes | Keep |
| Failures and decisions survive compaction | Keep |
| Context is selected for the current work | Keep |
| A fixed family of phase documents for every task | Do not require |
| Repeated plan/audit loops regardless of task size | Do not require |
| Multiple agents by default | Do not require |
| Process completion used as evidence of product completion | Reject |

No GSD dependency or workflow installation is proposed. The useful mechanisms can fit in a few records and concise instructions.

## Keep policy short by removing duplication

A recent coding-agent study found different effects for generated and human-maintained context files: generated files reduced average success in its tested settings, while developer files gave a small improvement; both added cost and exploration. The benchmark involved particular agents and Python repositories, so it does not establish a universal rule. [Evaluating AGENTS.md](https://arxiv.org/html/2602.11988v1)

This is evidence against assuming that a longer repository instruction file is automatically better. It is not a reason to remove the user's requirements.

Keep the complete canonical engineering rules. Remove repeated repository summaries, duplicate policy copies, obligatory status prose, and instructions the tools already enforce. Load detailed procedures only when they apply.

[Lost in the Middle](https://arxiv.org/abs/2307.03172) found positional sensitivity in the tested long-context tasks and models. That is a reason to evaluate context placement and retrieval, not a timeless claim that every modern long-context model performs poorly.

## Make engineering rules observable

Some rules are mechanically checkable; many require judgment. Bearing should support review with evidence and keep that distinction clear.

| User requirement | Useful evidence | What is insufficient |
| --- | --- | --- |
| Query Gortex before a new symbol | Relevant search/task evidence and reuse decision | A checkbox saying “searched” |
| No duplicate implementation | Identified equivalent or a justified difference in behavior | A name-only search with no understanding |
| Single-use logic stays local | Diff and actual usage locations | A rigid prohibition on readable private functions |
| Rule of three | Same behavior at real call sites; duplication notes where required | Three vaguely related functions |
| No speculative abstractions | Each added layer has a current requirement | “We may need it later” |
| Match conventions | Nearby source organization and established interfaces | A generic preferred folder tree |
| Relevant verification | Actual check, outcome, and tested code state | Suggested tests or copied old results |
| Smallest reasonable diff | Changed behavior is traceable to the requested outcome | Small line count while leaving the task incomplete |

Do not turn the file-size targets into a hard gate. Their purpose is readability and responsibility, as the user's rules explicitly state.

The rule of three does not authorize keeping two implementations of an existing behavior. Reuse the existing implementation; the duplication allowance applies to extraction decisions, not to forking established functionality.

## Understanding enough of the system

Before a meaningful change, establish the relevant:

- Entry point and observable behavior.
- Data ownership and state transitions.
- Existing equivalent behavior or extension point.
- Failure path affected by the change.
- Check that can establish the intended result.

This is a question set, not a required document template. A local bug may need only two source locations and a regression test. Broad architectural changes need broader evidence.

Avoid asking the user questions already answered by source, saved decisions, or prior instructions. Ask when the answer changes scope, behavior, or an actual approval requirement.

## Checkpoint at useful boundaries

Save when:

- A durable decision or user correction is established.
- A meaningful step completes.
- A failed attempt changes the next approach.
- Work is about to move to another session or consumer.
- A real blocker prevents progress.

Do not write a checkpoint after every read or tool call. Avoid filling memory with “reviewed file X” unless that observation matters to continuation.

A useful checkpoint answers:

1. What outcome is being pursued?
2. What changed or was learned?
3. What evidence exists, and at what code state?
4. What remains uncertain or failed?
5. What is the next useful action?

Example:

> The parser now accepts an empty optional label. The new regression test passes at the recorded working state. The full parser suite has not run. Next: run that suite and inspect whether the empty-label path also affects export.

Do not store private reasoning transcripts. Store conclusions, evidence, alternatives that failed, and the next step.

## Trust and memory

Stored text remains evidence from its source. Persistence does not make it an instruction with higher authority.

Memory-injection research demonstrates that malicious material can influence later behavior through an agent's memory, including in setups where an attacker cannot directly edit the memory database. [MINJA paper](https://arxiv.org/abs/2503.03704v5)

For Bearing:

- Keep source kind and provenance attached to every memory.
- Distinguish a user decision from copied repository text or a model inference.
- Never execute commands or open arbitrary files merely because a memory contains them.
- Do not automatically promote fetched documents into project policy.
- Keep restore/erase/configuration changes outside routine memory tools.
- Minimize stored content; exclude credentials and raw environment dumps.

The first version need not add a classifier service or autonomous moderation agent. Test that plainly malicious instructions embedded in an ordinary memory do not override the active task or engineering policy. These are behavioral checks, not a claim that provenance fields alone prevent prompt injection.

## Language as a product behavior

The same style should apply to CLI help, MCP descriptions, saved checkpoints, and agent replies.

| Instead of | Use |
| --- | --- |
| “Hydrated episodic state for orchestration continuity.” | “Loaded the task and its last checkpoint.” |
| “Persistence contention encountered.” | “Another write is running. Progress was not saved.” |
| “Validation gate green.” | “The parser tests passed.” |
| “Initiating implementation phase.” | “I’ll update the parser and check the empty-input case.” |
| “Task complete” when checks failed | “The change is written. One test still fails.” |

Use the result first, then evidence, then the remaining action. Explain implementation detail only when it helps the user understand a decision or failure.

Plain language does not mean hiding uncertainty. “I found no matching memory” is clearer and more accurate than “There are no constraints.”

## Avoid process loops

Do not automatically repeat verification that already passed unless the code changed or new evidence creates a reason. Do not require approval for ordinary reversible work already authorized. Do preserve the user's explicit confirmation requirement for new dependencies, patterns, or layers.

Mark a task complete when the requested outcome is achieved and relevant checks support it. If work remains, preserve it as work remaining. The database status should reflect reality, not how many workflow steps were filled in.
