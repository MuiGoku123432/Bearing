# Plan: session-review — multi-angle review before work is finished

Tracker: none yet (candidate: GitHub `MuiGoku123432/Bearing`) · Status: draft, awaiting approval · Rules: proposed [PLANNING.md](../../PLANNING.md) · Research: [10-session-review](../../research/10-session-review.md)

## Goal and scope

**Outcome:** before an agent finishes a task or hands off unreviewed changes, several reviewers each check the work from one angle. The lead agent confirms their findings against evidence and combines them into one review. Blocking findings are fixed or remain on record, so the task cannot finish with them hidden.

**Who benefits:** developers using Bearing with Codex, Claude Code, or OpenCode. They get fewer missed defects and policy violations reaching their diff, without having to ask for a review each time.

**Success signal:** in the S1 trial, the panel finds confirmed issues that a single self-review misses, at a cost the team accepts. After rollout, panel reviews appear in finish records for larger changes, and few findings are dismissed as false positives.

**In scope:** the review procedure, scaling rules, reviewer angles, the combining step, read-only reviewer agent definitions for the three hosts, and an optional finish record.

**Out of scope:** Bearing running models or orchestrating agents itself, posting pull request comments, automatic fix loops, and reviews triggered by session-end hooks. The research shows those hooks cannot host a review.

## Items

| Key | Level | Title | Parent | Depends on | Size signal (agent, unconfirmed) | Tracker ID |
| --- | --- | --- | --- | --- | --- | --- |
| F1 | Feature | Multi-angle review before work is finished | — | — | — | |
| S1 | Spike | Does a reviewer panel catch more than self-review, and at what cost? | F1 | — | Time-boxed: 3 days | |
| S2 | Story | Agent runs a scaled review panel before finishing | F1 | S1 | Largest here; mostly workflow and skill text plus setup output | |
| S3 | Story | Reviewers are installed as read-only host agents | F1 | S2 | Similar to the existing skill install in `cmd/bearing/install.go` | |
| S4 | Story | Finish records which review angles ran | F1 | S2 | Small: one optional field, validation, and a test | |

Size signals are the agent's relative guesses for discussion. The developers size these items.

### S1 — Spike: does a reviewer panel catch more than self-review, and at what cost?

Why: the research supports distinct angles but contains no evidence for LLM code review on Bearing tasks. Multi-agent work costs much more than single-agent work. Measure before building.

Method: set up a disposable repository with six tasks, each seeded with one known problem, plus one clean control:
- a logic bug
- a missing test for a new branch
- a duplicate of an existing helper (reuse violation)
- a string-concatenated SQL query
- an acceptance criterion the change does not meet
- an unneeded abstraction

Run each task in Claude Code and Codex twice: once with the current single self-review, and once with a prompt-only panel skill written for the spike.

Acceptance criteria (questions to answer):
- How many seeded problems did each approach find and confirm? How many false positives did each raise?
- What were the total tokens and wall time per review for each approach and host?
- Did the combining step drop real findings or keep false ones?
- Did Codex reviewers edit files despite instructions not to?
- Recommendation: build S2 as written, change the angles or the scaling rule, or stop.

Notes: use no customer data. Record host and model versions. Save the results as a Bearing research document.

### S2 — Agent runs a scaled review panel before finishing

Why: review should happen at the point where it can still change the outcome — before `finish` or a handoff — and scale with the risk of the change.

Acceptance criteria:
- Given a task whose diff exceeds the size threshold, touches security-sensitive paths, or adds a dependency, when the agent prepares to finish, then it runs one reviewer per angle: requirements, correctness and tests, engineering policy, and security and data. Each reviewer starts from `bearing_resume` output and the actual diff.
- Given a small, low-risk diff, when the agent prepares to finish, then it performs only the existing single self-review.
- Given the reviewers' findings, when the lead agent combines them, then duplicates are merged and each finding is confirmed against code, recorded test output, or Gortex evidence. Unconfirmed findings are dropped with a one-line reason in the review summary.
- Given a confirmed blocking finding that has not been fixed, when the agent calls `finish`, then the finding is listed in `review.unresolved` and Bearing rejects the finish. This is existing behavior.
- Given a session that ends with unreviewed changes, when the agent writes its handoff checkpoint, then the next action names the pending review.
- The review procedure ships in the Bearing workflow and skill for all three hosts, and `bearing rules` returns it.

Notes: the size threshold is set from S1 data. Reviewers read recorded verification output and do not run tests themselves; the lead agent already must run them.

### S3 — Reviewers are installed as read-only host agents

Why: a reviewer that can edit files is no longer independent. Hosts differ in whether they can enforce read-only access.

Acceptance criteria:
- Given `bearing install` or `bearing setup` for Claude Code, then a single `bearing-reviewer` agent definition is written to `.claude/agents/` with read-only tools. The angle is supplied when the agent is spawned.
- Given OpenCode, then the definition is written to `.opencode/agents/` with `permission.edit: deny`.
- Given Codex, then `.codex/agents/bearing-reviewer.toml` is written with review-only instructions and the shell tool disabled. INSTALL.md states that Codex cannot enforce read-only access for each agent separately.
- Given existing agent files or settings, when install runs again unchanged, then nothing changes, and other files are preserved or backed up as installation already does.

### S4 — Finish records which review angles ran

Why: later sessions and reviewers can see whether a change had a panel review, and S1-style measurements can continue on real work.

Acceptance criteria:
- Given a finish request with an optional `review.panel` list of entries `{angle, reviewer, confirmed, dropped}`, then Bearing validates its shape and stores it with the review. `bearing history` shows it.
- Given a finish request without `review.panel`, then behavior is unchanged.
- The CONTRACT.md wording makes clear that the panel record is agent-reported evidence, not certification.

Notes: build only if S1 recommends the panel.

## Deferred variations

- **Stop-hook reminder** (Claude Code and Codex): a reminder when a turn ends with unreviewed changes. Deferred until agents are observed skipping the workflow step. Stop fires every turn, so it needs loop and cost guards.
- **Session-end hooks:** rejected. They cannot keep the session open (Claude Code) or time out within three seconds (Codex).
- **Reviewers from a different model family:** this has the strongest evidence for diverse reviewers, but it depends on host support and organizational model policy. Revisit after S1.
- **Posting findings as pull request comments**, as the Claude Code plugin does: a separate feature after Bearing creates pull requests.
- **Turning should-fix findings into tracker items:** connects to the planning work; wait until `plan` tasks exist.
- **Panel review of plan tasks:** apply the same pattern to backlog plans later.

## Apply log

Not applied. No tracker is connected for Bearing yet. After approval, this document should be saved as a `plan` document snapshot once that kind exists ([PLANNING_SPEC.md](../../PLANNING_SPEC.md)) and its items created in the chosen tracker.
