# Backlog planning with Azure Boards and GitHub

Research date: 21 October 2026. Question: how should Bearing help agents groom a backlog — create and refine epics, features, stories, and tasks, write acceptance criteria, and handle estimates — in Azure Boards and GitHub, while the trackers' own tools and the host model do the detailed work?

Sources are listed in the [source ledger](SOURCES.md#backlog-planning-practice) under IDs B01–B13 and T01–T10.

## Main findings

1. **Refinement is ongoing work, and sizing belongs to the developers.** The 2020 Scrum Guide defines refinement as breaking down and further defining backlog items "into smaller more precise items," adding "a description, order, and size." It states that "the Developers who will be doing the work are responsible for the sizing." Sprint planning often decomposes selected items "into smaller work items of one day or less." [B01]
2. **A story is a conversation token with a test, not a document.** Card, Conversation, Confirmation: the card identifies the requirement; the details come from conversation; acceptance tests confirm it. Jeffries reports that examples, preferably executable, work better than long specification documents. [B02]
3. **INVEST is a quick check, and different letters matter at different times.** Independent, Negotiable, Valuable, Estimable, Small, Testable. Wake suggests a spike when a story cannot be estimated. [B03] Humanizing Work notes that near-term items most need to be small, estimable, and testable, while far-off items most need value and negotiability. Their heuristic is that six to ten stories should fit in a sprint. [B04] SMART (Specific, Measurable, Achievable, Relevant, Time-boxed) is Wake's companion check for tasks. [B03]
4. **Splitting should produce vertical slices.** Splitting by architectural layer fails the Independent and Valuable checks. The Humanizing Work flowchart tries nine patterns in order: workflow steps, operations, business-rule variations, data variations, data entry methods, major effort, simple/complex, defer performance, and break out a spike (last). Its underlying approach is to find the core complexity and reduce the variations to one. Prefer splits that let low-value parts be deferred and that give similar sizes. In complex work, find one or two learning slices instead of listing every story. [B04]
5. **Use a Definition of Ready as guidance, not a gate.** Cohn does not recommend a Definition of Ready for most teams. Where a team uses one, he advises favoring guidelines over rules and avoiding any requirement that work be "100 percent" done before a story enters a sprint, because such rules lead to a sequential, stage-gate process. [B05] The Definition of Done is a required Scrum commitment. [B01]
6. **Estimates are weak forecasts and easily anchored.** The 2020 Scrum Guide uses "size" and says only what has already happened can inform forward-looking decisions. [B01] Jeffries, who may have originated story points, calls using them to predict "when we'll be done" "at best a weak idea." He also calls comparing teams by velocity harmful, and he prefers slicing stories small over estimating them. [B06] In controlled studies, an initial number strongly shifts experts' effort estimates. Training reduces this anchoring effect but does not remove it. [B07] Practitioners recommend throughput-based Monte Carlo forecasting, but the sources found make the case without controlled comparisons. [B08] **Implication:** an AI-suggested number shown before the team sizes an item acts as an anchor.
7. **LLMs evaluate story quality reasonably well but estimate points poorly out of the box.** ChatGPT's assessments using the 13-criterion QUS framework aligned well with human reviewers; a "best of three" run improved stability. [B09] In a small study (GPT-4, 30 test issues from each of three TAWOS Jira projects), zero-shot story-point estimates were worse than simply predicting the mean or median. Optimized example selection cut the mean absolute error by 59% on average. [B10] Fine-tuned models report better accuracy on research datasets, but that does not transfer to a team's own scale. [B11]
8. **The common failure is too much plausible work.** Humanizing Work reports clients with "backlogs full of AI slop": detailed epics and stories that nobody chose or fully understands. Their recommendation is to use AI to process evidence, enumerate variations, and improve tests, while the product owner selects and ranks what enters the backlog. A reasonable-sounding plan is not necessarily a correct one. [B12] They have also published Gherkin guidance for AI-assisted acceptance scenarios. [B13]

## Tracker models

### Azure Boards

Each project uses one of four base processes, which cannot be changed after the project is created. Inherited processes can add fields, states, and types. [T01]

| Process | Portfolio | Requirement | Task | Size field on requirement |
| --- | --- | --- | --- | --- |
| Basic | Epic | Issue | Task (Remaining Work) | Not confirmed in these sources |
| Agile | Epic, Feature | User Story | Task (Original Estimate, Remaining, Completed Work) | Story Points |
| Scrum | Epic, Feature | Product Backlog Item | Task (Remaining Work) | Effort |
| CMMI | Epic, Feature | Requirement | Task (Original Estimate, Remaining, Completed Work) | Size |

Velocity can sum Story Points (user stories), Effort (product backlog items), Size (requirements), or count items. [T02] Customized processes can rename or add fields. Discover the actual fields with `wit_work_item get_type` instead of assuming.

### GitHub

Planning structure on GitHub is assembled from several features:

- **Issue types** are defined per organization, for example Epic, Feature, Task, or Bug. Personal repositories have none, so labels are the fallback.
- **Sub-issues** form the hierarchy. Blocked-by/blocking relationships record dependencies. `gh` 2.94.0 and later sets type, parent, and dependencies directly with `--type`, `--parent`, `--blocked-by`, and `--blocking`. [T03]
- **Issue fields** became generally available on 2 July 2026. Every organization gets `Priority`, `Effort`, `Start date`, and `Target date`, and the GitHub MCP server can read and set them. [T04]
- **Projects** add iteration and custom fields.

GitHub Issues also shows the rationale and confidence behind agent changes and can hold them as suggestions. GitHub states that these approvals are "a workflow convenience, not a security control." [T05]

## Existing agent tooling

| Need | Azure DevOps | GitHub |
| --- | --- | --- |
| Server | Microsoft's [remote MCP server](https://learn.microsoft.com/en-us/azure/devops/mcp-server/remote-mcp-server?view=azure-devops) (recommended; Entra ID) or local `@azure-devops/mcp` (`stdio`; PAT or Entra) [T06][T07] | [GitHub MCP server](https://github.com/github/github-mcp-server) remote or local; `gh` CLI [T08][T03] |
| Read items | `wit_work_item` (`get`, `get_batch`, `get_type`, `list_for_iteration`), `wit_backlog`, `search_workitem` | `issue_read` (`get`, `get_sub_issues`, `get_parent`), `list_issues`, `search_issues`, `list_issue_types`, `list_issue_fields` |
| Create/update | `wit_work_item_write` (`create`, `update`, `update_batch`, `add_child`) | `issue_write` (`create`/`update` with `type`, `parent_issue_number`, `issue_fields`) |
| Hierarchy and links | `wit_work_item_link_write` (`link`, `link_to_pull_request`) | `sub_issue_write` (`add`, `remove`, `reprioritize`); `gh issue edit --add-sub-issue` |
| Iterations and capacity | `work` (`list_team_iterations`, `get_team_capacity`) | `projects_write` (`create_iteration_field`, `update_project_items`); the `projects` toolset is not in the default set |
| Limit the surface | `X-MCP-Toolsets: wit,work`, `X-MCP-Readonly: true`; local `-d core work work-items` | `--toolsets issues,projects`, `--read-only`, `X-MCP-Tools`, `X-MCP-Lockdown` |

Host limits matter. The remote Azure DevOps server requires an Entra-backed organization. Claude Code and Cursor need a custom Entra app registration. Codex and Claude Desktop cannot authenticate to the remote server and must use the local one. [T06] Microsoft recently renamed and consolidated its tools; agents should read the live schemas instead of relying on names in instructions. [T07]

## Security observations

- **Issue text is an injection channel.** Invariant Labs showed a malicious public issue steering an agent into leaking private repository data through the GitHub MCP server. They describe the root cause as architectural, not a server bug, and recommend one repository per session and least-privilege tokens. [T09]
- **GitHub's mitigations help but are partial.** The GitHub MCP server now sanitizes hidden Unicode, HTML, and code-fence content by default. Lockdown mode filters public-repository content from authors without push access. [T10] Sanitization also means hidden markers in issue bodies may not survive reads, so plan-tracking markers should be visible text.
- **Use least privilege.** Microsoft advises granting only the OAuth scopes the client needs, using read scopes for read-only tools. Scopes never exceed the user's own permissions. [T06]
- **Keep regulated data out of work items.** Plans built from support tickets or customer evidence must not copy customer PII, CPNI, or credentials into work items, plan documents, or Bearing memory. This is an organizational requirement, not a tracker feature.

## Implications for Bearing

| Topic | Recommendation |
| --- | --- |
| Product boundary | Bearing supplies planning rules, a workflow, and durable records. It does not call tracker APIs, store tracker credentials, or synchronize tracker state. The host agent uses the tracker's own MCP server or CLI, just as it uses Gortex for code. |
| Rule delivery | Embed one canonical planning rule set and deliver it the same way as `ENGINEERING.md`: through `rules`, start/resume for planning tasks, and an on-demand skill. Do not load it into ordinary build tasks. |
| Proposal before writes | Draft changes in a reviewed Markdown plan document saved as a Bearing document snapshot. Write to the tracker only after the user approves that version. |
| Estimates | Developers own the official size fields. The agent may offer a labeled, relative size signal with its reasons, after or apart from the team's estimate. Forecasts come only from the team's history. |
| Duplicates and retries | Tracker writes are external operations, like Git writes. Search before creating, mark items with a visible plan key, and inspect after an uncertain result before retrying. |
| Restraint | By default, propose the smallest set of items that serves the next decision. Keep deferred variations in the plan document, not the tracker. |
| Tracker profile | Save the organization/project, process, team, area and iteration paths, type names, and field reference names as attributed Bearing memory so later sessions do not have to rediscover them. |

## Unknowns

- No live planning session was run against Azure Boards or GitHub for this research. Tool names and parameters come from current documentation and can change.
- Whether each host reliably discovers a second Bearing skill and avoids loading it twice needs testing in Codex, Claude Code, and OpenCode.
- The quality of agent-drafted stories on Uniti teams' own backlogs, and whether the restraint rules prevent excess items, needs evaluation with real product owners.
- Behavior with customized Azure processes and GitHub organizations without issue types or fields needs testing.
- No controlled evidence was found that throughput forecasting beats point-based forecasting. Both remain team choices.
