# Multi-angle review at the end of a session

Research date: 21 October 2026. Question: should Bearing provide a reviewer skill that runs at the end of each session, has several agents review the work from different angles, and then combines their results into one review? If so, how should it work in Codex, Claude Code, and OpenCode?

Sources are listed in the [source ledger](SOURCES.md#session-review) under IDs R01–R09 and H01–H06.

## Main findings

1. **Reviewing from assigned perspectives finds more defects than unstructured review.** In perspective-based reading, each reviewer reads from one viewpoint, such as a tester or a designer. In NASA/Goddard experiments and a university replication, it raised individual and team defect detection compared with ad hoc reading. Results by defect class were less consistent. [R01] This is the strongest evidence for giving reviewers distinct angles instead of running the same reviewer several times.
2. **Review finds fewer defects than people expect, and understanding the change is the bottleneck.** At Microsoft, developers' main reason for review was finding defects, but the reviews mostly produced knowledge transfer, team awareness, and alternative solutions. Understanding the change was the key difficulty. [R02] For an agent reviewer, this means the task, requirements, and decisions must be supplied, not just the diff.
3. **More agents debating does not reliably beat simpler methods.** Multi-agent debate often fails to outperform a single agent using chain-of-thought or self-consistency, even with more compute. [R04][R05] Reviewers from different model families were the one change that consistently helped. [R05] Without outside feedback, models generally cannot correct their own reasoning and sometimes get worse. [R03] **Implication:** use independent reviewers with separate angles, then check each finding against evidence such as code, tests, or Gortex results. Do not let reviewers argue until they agree.
4. **A reviewer favors work that resembles its own.** GPT-4 showed significant self-preference, linked to the familiarity (low perplexity) of the text. [R06] A reviewer with a fresh context is better than the authoring thread reviewing itself; a different model family is better still where the host allows it.
5. **A production multi-agent reviewer already exists and shows a workable shape.** Anthropic's Claude Code `code-review` plugin works in stages:
   - It checks first whether a review is needed at all.
   - Four reviewers run in parallel: two check instruction compliance and two look for bugs and security problems in the changed code.
   - For each bug or security finding, a separate agent confirms that the issue is real.
   - Findings that cannot be confirmed are dropped.

   Its reviewers report only "high signal" issues and list known false positives, such as pre-existing issues, nitpicks, and anything a linter catches. [R09]
6. **Multi-agent work is expensive and only pays for valuable tasks.** Anthropic reports that multi-agent systems use about 15 times the tokens of a chat, compared with about 4 times for a single agent. They pay off when the task value justifies the cost. Early versions over-spawned agents for simple queries, so Anthropic added explicit rules that scale effort to the task. Each subagent needs an objective, an output format, and boundaries. [R07]
7. **Parallel agents fail when their decisions conflict; read-only review avoids most of this.** Cognition argues against parallel agents that each make decisions without shared context. [R08] Reviewers that only read and report do not make conflicting changes. The two remaining risks are missing context and duplicate findings. Bearing's resume context addresses the first; the combining step addresses the second.

## What the hosts support

| Capability | Claude Code | Codex | OpenCode |
| --- | --- | --- | --- |
| Reviewer subagents | Own context, prompt, and tools per agent; can be denied Write/Edit; defined in `.claude/agents/` [H02] | Custom agents in `.codex/agents/*.toml`; delegation happens when the user, `AGENTS.md`, or a skill asks for it [H03] | Subagents in `.opencode/agents/` with `permission.edit: deny` [H05] |
| Read-only enforcement | Per agent through its tool list | **None per agent:** subagents inherit the parent's sandbox; instructions can ask them not to edit but don't enforce it [H03] | Per agent through permissions [H05] |
| Model per reviewer | `model` setting per agent | `model` and reasoning effort per agent [H03] | `model` per agent [H05] |
| End-of-turn hook | `Stop`: can block and continue the conversation [H01] | `Stop`: `decision: "block"` starts a continuation prompt; `stop_hook_active` prevents loops [H04] | Plugin `session.idle` event [H06] |
| End-of-session hook | `SessionEnd`: no decision control; side effects only [H01] | `SessionEnd`: advisory, 1-second default and 3-second maximum timeout [H04] | No equivalent confirmed |

**"End of session" cannot be a session-end hook.** A session-end hook cannot run a multi-agent review: it cannot keep the session open (Claude Code), or it times out within three seconds (Codex). Sessions also end by crash, closed window, or idle timeout. Stop hooks fire at the end of every turn, not once per session, so a Stop-hook review would need careful guarding against repeat runs and cost. The reliable trigger already exists in Bearing's workflow: the review step before `bearing_finish`, plus the checkpoint before a handoff.

## Implications for Bearing

| Topic | Recommendation |
| --- | --- |
| When to run | Before `finish` on any task with a diff, and before a handoff checkpoint when the session leaves unreviewed changes. A Stop-hook reminder is an optional later addition, justified only if agents skip the step. |
| How much to run | Scale to the change. A small, low-risk diff gets the existing single self-review. Larger diffs, security-sensitive paths, or new dependencies get the panel. The rule must be explicit; agents do not judge effort well. [R07] |
| Angles | Each reviewer gets one angle: requirements (do the acceptance criteria and verification evidence hold?), correctness and tests, engineering policy (the five checks, with reuse confirmed through Gortex), and security and data (OWASP Top 10, secrets, customer data, and CPNI). Handoff quality — checkpoint accuracy, next action, and memory worth saving — stays with the lead agent. |
| Shared context | Every reviewer starts from `bearing_resume` output: task, acceptance criteria, policy, decisions, and input documents. It then reads the actual diff, with no summary in between. |
| Combining | The lead agent removes duplicates and confirms each finding against code, tests, or Gortex evidence. It drops findings it cannot confirm, with a reason, and classifies the rest as blocking, should-fix, or note. Blocking findings are fixed, or recorded in `review.unresolved`, which Bearing already rejects at finish. |
| Independence | Reviewers start with fresh context. They use a different model family where the host and the organization permit it. Reviewers never edit files. |
| Record | Finish keeps one concise review summary. If an optional record of each reviewer's angle and finding counts is added, it is attributed evidence, not certification. |
| Boundary | Bearing ships instructions and host agent definitions. It does not run models, orchestrate agents, or execute tests. The host does that, as with the rest of the workflow. |

## Unknowns

- No experiment compared a reviewer panel with a single self-review on Bearing tasks. Whether the panel catches enough extra real issues to justify its extra cost is the first thing to measure. The cost is unmeasured; roughly one review's tokens per reviewer plus confirmation is an estimate, not data.
- Codex cannot make a reviewer agent read-only. Whether instruction-only restraint holds in practice needs testing.
- Discovery of project agent files and skills differs by host and version; installation needs live checks.
- The perspective-based reading evidence comes from human inspection of requirements documents. Its transfer to LLM reviewers of code is a hypothesis.
- Using a different model family may conflict with organizational model policy. The default should work with a single model family.
