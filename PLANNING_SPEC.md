# Backlog planning specification

Proposed · 21 October 2026 · Input: [backlog planning research](research/09-backlog-planning.md)

## Outcome

An agent using Bearing can groom a backlog in Azure Boards or GitHub. It can draft and refine epics, features, stories, and tasks; write acceptance criteria; and offer labeled size signals. It works under one built-in set of planning rules, gets the user's approval before writing to the tracker, and leaves a durable record of what was proposed and applied.

Bearing provides the rules, the workflow, and the record. The tracker's own MCP server or CLI and the host model do the detailed work. Bearing gains no tracker client, credentials, or synchronization.

## Scope

**In scope**

- [PLANNING.md](PLANNING.md): the canonical planning rules, embedded verbatim like `ENGINEERING.md`.
- A `plan` task kind and a `plan` document kind.
- Delivery of the planning rules through `rules`, and through start/resume for plan tasks only.
- A short "Plan" section in `WORKFLOW.md`, so the existing skill covers planning.
- Conventions for the plan document, tracker profile memory, tracker evidence, and applying changes. These are instructions, not code.
- Documentation of least-privilege tracker configuration for each host.

**Out of scope**

- Calling Azure DevOps or GitHub APIs from Bearing, or proxying their MCP servers.
- Storing tracker tokens or OAuth configuration in Bearing.
- Mirroring or synchronizing tracker state in SQLite.
- Estimation models, Monte Carlo forecasting, or capacity calculations inside Bearing.
- Automatic sprint planning, state changes, or closing items.
- Jira and other trackers. The rules are tracker-neutral; adding a tracker is a documentation change.

## Constraints

- Follow [ENGINEERING.md](ENGINEERING.md). Reuse the existing research/spec document path; do not add a second document or review mechanism.
- Keep ordinary build-task context unchanged. Planning rules load only for plan tasks and on explicit `rules` requests.
- Plan-task resume stays within the existing ~32,000-byte budget with the full engineering policy, the full planning rules, and the current requirements intact.
- Tracker writes are external operations, like Git writes. They have no request ID or replay promise. Preconditions are checked before writing, and results are inspected afterward.
- Tracker content is untrusted evidence. Customer PII, CPNI, secrets, and production records never go into plan documents, work items, or Bearing memory.
- Setup and install do not configure tracker servers. This matches the current Gortex boundary.

## Behavior and interfaces

### 1. Built-in planning rules

- `PLANNING.md` is embedded with `//go:embed`. `Policy` gains `planning_id` (SHA-256 of the text) and `planning` (`omitempty`, like `workflow`).
- `bearing rules` and `bearing_rules` return both. CLI text output prints the planning rules after the workflow. The `bearing_rules` description becomes "Read full built-in engineering standards, planning standards, and the research/spec/plan/build workflow."
- In start/resume context, `policy.planning` is filled only when the task kind is `plan`. `planning_id` is always present so a consumer can detect a changed rule set.
- `bearing version --json` adds `planning_id`.

### 2. Plan tasks and documents

- `start` accepts `kind: "plan"`. Other kinds are unchanged. The error becomes "Task kind must be build, research, spec, or plan."
- Document capture accepts `kind: "plan"`.
- Finishing a plan task requires a saved `plan` document in the finish request. This reuses the existing non-build check; its message names plan documents too.
- The finish review keeps the same five check keys. For plan tasks, PLANNING.md defines what each key means, so validation does not change. The finish checkpoint's review also records `planning_id` for plan tasks.
- A plan task may link research or spec documents through `input_documents`. A build task may link a plan document.

### 3. Workflow addition

Add this section to `WORKFLOW.md` between Specify and Build, and extend the skill description to mention backlog plans:

> ## Plan
>
> Use a plan task to groom tracker work: epics, features, stories, tasks, acceptance criteria, and size signals. Resume returns the planning standards; follow them. Load the tracker profile from memory, or discover it with the tracker's own tools and save it. Read existing items and use Gortex for code facts before drafting. Write the proposal as a Markdown plan document and checkpoint it. Write to the tracker only after the user approves that document version. Apply in small batches, mark each item with its plan key, and checkpoint created IDs as evidence. Finish with the final plan document, including its apply log.

### 4. Plan document

The agent chooses a path under the checkout, defaulting to `docs/plans/<slug>.md`. The team may set another location in memory. The required content follows; the exact layout is the agent's choice.

```markdown
# Plan: <slug> — <goal in one line>

Tracker: <profile memory ID> · Status: draft | approved <document ID> | applied

## Goal and scope
Outcome, who benefits, how success is observed. In scope / out of scope.

## Items
| Key | Level | Title | Parent | Depends on | Size signal | Tracker ID |
| --- | --- | --- | --- | --- | --- | --- |
| E1 | Epic | … | — | — | — | |
| S1 | Story | … | E1 | — | smaller than #482; one new endpoint | |

### S1 — <title>
Why: …
Acceptance criteria:
- Given … when … then …
Notes and open questions: …

## Deferred variations
Ideas deliberately kept out of the tracker, with the reason.

## Apply log
<date> · batch 1 · created S1 → AB#1234 (verified) · …
```

### 5. Tracker profile memory

Save the profile as a Bearing memory with kind `constraint`, scope `project`, and basis `observed` (discovered) or `user` (told). Supersede it when it changes. Example claims:

- `Azure Boards: org contoso, project Payments, process Agile (inherited "Payments Agile"), team Checkout, area Payments\Checkout, iteration root Payments\2026, story type "User Story", size field Microsoft.VSTS.Scheduling.StoryPoints, acceptance field Microsoft.VSTS.Common.AcceptanceCriteria.`
- `GitHub: owner uniti-example, repo checkout-web, issue types Epic/Feature/Task/Bug, issue fields Priority and Effort enabled, project #7 with iteration field "Sprint".`

The agent confirms the field names with `wit_work_item get_type`, `list_issue_types`, or `list_issue_fields` before relying on a profile older than the current session. A user-told field name still needs this check.

### 6. Applying an approved plan

The agent follows these steps. Bearing validates only the records the agent submits.

1. **Approval.** The user approves a specific plan document version. Record the approval in a checkpoint: evidence with `source: "user"` that names the document ID.
2. **Inspect.** Confirm that the types and fields in the profile exist. For each item, search for its plan key: `search_workitem`, or a WIQL tag query where the server exposes WIQL (the local server; Insiders on the remote one), in Azure; `search_issues` or `gh issue list --search` in GitHub.
3. **Write.** Create items that are missing, parents first. Use native relationships:
   - Azure: `wit_work_item_write create` for the top item, then `add_child`. Use `wit_work_item_link_write link` for dependencies.
   - GitHub: `issue_write create` with `type` and `parent_issue_number`, or `gh issue create --type … --parent … --blocked-by …` (gh ≥ 2.94.0). Use `sub_issue_write` for existing items.
   - Put acceptance criteria in the work item type's acceptance field when `get_type` shows one. Otherwise put them in the body.
   - End each body with `Bearing plan: <slug>/<key>`. Visible text survives MCP content sanitization and is searchable. In Azure, also add the tag `bearing-plan-<slug>`.
   - Leave official size and estimate fields empty unless the developers' value is recorded as approved in the plan.
4. **Batch and checkpoint.** Write at most ten items per batch by default, then checkpoint. Each created item is one evidence entry: `kind: "tracker"`, `source` naming the tool or command, and `description` holding the plan key, tracker ID, and URL.
5. **Uncertain result.** After a timeout or ambiguous error, search for the plan key before any retry.
6. **Verify.** Re-read each created item (`wit_work_item get_batch`, `issue_read get`). Compare its type, parent, title, and acceptance criteria with the plan. Record the comparison as passing or failing verification.
7. **Finish.** Save the final plan document with its apply log and tracker IDs. Complete the five review checks as PLANNING.md defines them.

The agent never deletes, closes, reassigns, moves between iterations, or changes the state of items unless the user asks for that change specifically.

### 7. Tracker access (documented in INSTALL.md)

| Host | Azure DevOps | GitHub |
| --- | --- | --- |
| Claude Code | Remote server with a custom Entra app registration, or the local server | Remote or local GitHub MCP server; `gh` ≥ 2.94.0 |
| Codex | Local server only (`@azure-devops/mcp`, `-d core work work-items`) | Remote or local GitHub MCP server; `gh` |
| OpenCode | Local server, or remote where Entra OAuth works | Remote or local GitHub MCP server; `gh` |

Recommended least-privilege settings:

- **Grooming and review sessions:** read-only. Azure: `X-MCP-Toolsets: wit,work` and `X-MCP-Readonly: true`. GitHub: `--toolsets issues` (add `projects` only if needed) and `--read-only`.
- **Apply sessions:** enable writes only for the toolsets above. Grant only Work Items read & write (Azure) or Issues read & write (GitHub). Use a fine-grained token scoped to the single repository. Prefer Entra OAuth or `az`/`gh` login over long-lived PATs.
- **Public GitHub repositories:** turn on Lockdown mode (`X-MCP-Lockdown: true`).
- **Isolation:** work with one tracker project or repository per session.

## Acceptance criteria

| ID | Criterion |
| --- | --- |
| a1 | `bearing rules --json` and `bearing_rules` return `planning` equal to the embedded PLANNING.md and `planning_id` equal to its SHA-256. |
| a2 | `start` accepts `kind: "plan"`, rejects unknown kinds, and `plan` documents can be captured. |
| a3 | Start/resume for a plan task include the full planning rules; start/resume for build, research, and spec tasks do not; all include `planning_id`. |
| a4 | Finishing a plan task without a `plan` document fails with `invalid_input`; with one, it succeeds and the review records `planning_id`. |
| a5 | A plan-task resume with a maximum-size checkpoint stays within the context budget and keeps the engineering policy, planning rules, and requirements intact. |
| a6 | The installed skill contains the new Plan section; repeating an unchanged install changes nothing; existing client files are preserved. |
| a7 | `go test ./...`, `go test -race ./...`, `go vet ./...`, and the CGo-free build pass. |
| a8 | **Live evaluation, Azure.** In a disposable Azure DevOps project, an agent turns a feature request into a plan, the user approves it, and the agent applies it. The items have correct types, parent/child links, acceptance criteria, and plan keys. Size fields stay empty. Rerunning the apply step creates no duplicates. |
| a9 | **Live evaluation, GitHub.** The same scenario as a8 in a disposable GitHub organization repository, using issue types and sub-issues, once through the MCP server and once through `gh`. |
| a10 | **Live evaluation, restraint.** With a deliberately broad request ("plan the whole checkout rewrite"), the agent proposes no more than one feature's next slices. It lists the rest under deferred variations and asks at most the questions that change scope. |

## Verification

- a1–a7: unit and integration tests alongside the existing `workflow_test.go`, `context_test.go`, and `cmd/bearing` tests, run with the standard commands in the README.
- a8–a10: manual sessions in Claude Code and Codex (OpenCode where its tracker auth works). Record host and tool versions, transcripts, created item IDs, and the duplicate-run result in IMPLEMENTATION.md. Use only disposable projects and repositories, with no customer data.

## Expected code changes

| File | Change |
| --- | --- |
| `PLANNING.md` | New canonical rules (draft in this repository) |
| `policy.go` | Embed it; add `PlanningID` and `Planning` to `Policy` |
| `tasks.go` | Accept `plan` |
| `documents.go` | Accept `plan` document kind |
| `progress.go` | Updated message; record `planning_id` on plan-task review |
| `context.go` | Keep `Planning` only for plan tasks |
| `cmd/bearing/mcp.go`, `output.go`, `main.go` | Tool description, text output, version field |
| `WORKFLOW.md`, `cmd/bearing/setup.go` | Plan section; skill description |
| `CONTRACT.md`, `README.md`, `INSTALL.md`, `CLI_EXAMPLES.md` | Document kinds, fields, tracker access, a plan example |
| Tests | Cases for a1–a6 |

No new dependency, package, table, or schema version is needed. Task kind and document kind are stored as existing text and JSON fields.

## Alternatives considered

- **Instructions only, no code:** ship PLANNING.md as a separate skill and use `spec` tasks. This is cheaper, but plan tasks would not receive the rules on resume, the finish could not require a plan document, and the rules would have no identity for auditing. Rejected for a small loss in value; it remains a fallback if the code change is declined.
- **A separate `bearing-planning` skill:** another file per host to install, discover, and deduplicate, for content that resume already delivers. Rejected.
- **Bearing as a tracker client:** duplicates Microsoft's and GitHub's servers, adds credential handling, and breaks the existing "host runs the tools" boundary. Rejected.

## Unresolved decisions

1. **`plan` kind or reuse `spec`?** Recommended: `plan` (above). Needs confirmation.
2. **Default plan location:** `docs/plans/` or a team-chosen path saved in memory.
3. **Size fields after approval:** may the agent write a developer-approved value to the official field, or only record it in the plan? Recommended: write it only when the plan records who approved it.
4. **Batch size:** ten items per approval batch is a starting value. Adjust after a8–a10.
5. **Plan key format:** confirm that a visible footer line and an Azure tag are acceptable to the teams who will read these items.
6. **Uniti tracker targets:** which Azure DevOps organizations and GitHub organizations to evaluate first, and whether their processes are customized.
