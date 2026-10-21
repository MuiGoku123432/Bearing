# Source ledger

Reviewed on 20 October 2026. This ledger covers the primary sources used in this research. Links point to the supporting documents, not search result pages.

## How to read the evidence

- **Specification/reference:** documents an API or behavior; the selected version and implementation still matter.
- **Research paper:** reports experiments under particular conditions; results do not automatically transfer to Bearing.
- **Engineering report:** first-party experience, useful for design hypotheses; not independent proof of comparative performance.
- **Local observation:** installed Gortex metadata or guidance inspected in this workspace; not an end-to-end test.

Rolling documentation can change after this date. Pin dependency versions and record tested host versions during implementation. Paper versions are retained in links where a specific HTML version was inspected.

The recommendations in this folder are original design conclusions drawn from the user's requirements and this evidence. They are not presented as vendor requirements. Numerical performance claims from memory vendors were not adopted as Bearing targets.

## Memory, context, and behavior research

| ID | Primary source | Inspection and use |
| --- | --- | --- |
| P01 | [LongMemEval](https://arxiv.org/html/2410.10813v2) | Paper abstract and selected HTML material; memory abilities and evaluation decomposition |
| P02 | [LoCoMo](https://arxiv.org/abs/2402.17753) | Abstract; long conversational memory evaluation scope |
| P03 | [MemGPT](https://arxiv.org/html/2310.08560v2) | Abstract and memory-system passages; selective movement between context and external storage |
| P04 | [Mem0](https://arxiv.org/abs/2504.19413) | Abstract; extraction/consolidation/retrieval approach, without adopting reported savings |
| P05 | [Zep](https://arxiv.org/html/2501.13956v1) | Temporal extraction and graph sections; recorded time versus validity and evidence relationships |
| P06 | [Lost in the Middle](https://arxiv.org/abs/2307.03172) | Abstract; context-position sensitivity in the tested models/tasks |
| P07 | [Evaluating AGENTS.md](https://arxiv.org/html/2602.11988v1) | Abstract, setup, and reported findings; counterevidence to assuming more instructions are better |
| P08 | [Memory Injection Attacks / MINJA](https://arxiv.org/abs/2503.03704v5) | Revised abstract; why persistent content must retain a trust boundary |

These papers are not a product ranking. Their tasks, models, baselines, and measurement methods differ. Abstract-only inspection is sufficient for the limited conceptual claims made here, not for reproducing their methods.

## Agent engineering reports

| ID | Primary source | Use |
| --- | --- | --- |
| E01 | [Anthropic: long-running agent harnesses](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) | Progress, incremental work, and recovery failure patterns |
| E02 | [Anthropic: writing effective tools](https://www.anthropic.com/research/writing-tools-for-agents) | Clear operations and evaluation of tool usability |
| E03 | [Anthropic: agent evaluations](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents) | Environment outcome versus agent claims; different kinds of graders |
| E04 | [GSD architecture](https://github.com/gsd-build/get-shit-done/blob/main/docs/ARCHITECTURE.md) | Persistent planning state and orchestration reference; no GSD installation proposed |

## Storage and Go references

| ID | Primary source | Use |
| --- | --- | --- |
| D01 | [SQLite isolation](https://sqlite.org/isolation.html) | Writer serialization and stale read-to-write upgrades |
| D02 | [SQLite WAL](https://sqlite.org/wal.html) | Local coordination, journal behavior, checkpoint limits, WAL-reset bug |
| D03 | [SQLite synchronization settings](https://sqlite.org/pragma.html#pragma_synchronous) | Distinguishing process recovery, consistency, and power-loss durability |
| D04 | [SQLite foreign keys](https://sqlite.org/foreignkeys.html) | Per-connection enforcement |
| D05 | [SQLite backup API](https://sqlite.org/backup.html) | Consistent snapshots |
| D06 | [SQLite FTS5](https://sqlite.org/fts5.html) | Search syntax, ranking order, snippets, index consistency |
| D07 | [SQLite 3.51.3 release](https://sqlite.org/releaselog/3_51_3.html) | Confirms the WAL-reset fix in that release |
| D08 | [Go connection management](https://go.dev/doc/database/manage-connections) | Pool behavior and connection-limit pitfalls |
| D09 | [Go transactions](https://go.dev/doc/database/execute-transactions) | Transaction APIs and keeping related operations on the transaction |
| D10 | [modernc.org/sqlite package documentation](https://pkg.go.dev/modernc.org/sqlite) | Driver candidate, CGo-free build, dependency caveat |
| D11 | [mattn/go-sqlite3 repository](https://github.com/mattn/go-sqlite3) | Alternative driver and CGo requirement |
| D12 | [Git worktrees](https://git-scm.com/docs/git-worktree) | Shared versus per-worktree Git state |

No selected driver's embedded engine or platform matrix was verified by building it. System SQLite and Go module versions cannot substitute for that check.

## MCP references

| ID | Primary source | Use |
| --- | --- | --- |
| M01 | [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) | SDK candidate and documented protocol compatibility |
| M02 | [MCP tools, 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) | Older tool/result contract and structured-content text compatibility |
| M03 | [MCP tools, 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) | Current versioned tool contract, metadata, errors, and schema behavior |

## Consumer references

| ID | Primary source | Use |
| --- | --- | --- |
| C01 | [Codex AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md) | Instruction discovery |
| C02 | [Codex skills](https://learn.chatgpt.com/docs/build-skills) | Skill locations and duplicate-name behavior |
| C03 | [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli) | Local server command and repository binding |
| C04 | [Codex hooks](https://learn.chatgpt.com/docs/hooks) | Compaction event output behavior |
| C05 | [OpenAI prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching) | Prefix matching and why caching must be measured in the host |
| C06 | [Claude memory and instructions](https://code.claude.com/docs/en/memory) | Imports, auto-memory scope, and startup loading |
| C07 | [Claude skills](https://code.claude.com/docs/en/skills) | Host-specific skill delivery |
| C08 | [Claude MCP](https://code.claude.com/docs/en/mcp) | Local MCP setup |
| C09 | [Claude hooks](https://code.claude.com/docs/en/hooks) | Event-specific lifecycle integration |
| C10 | [OpenCode rules](https://opencode.ai/docs/rules/) | AGENTS/CLAUDE precedence and explicit instruction loading |
| C11 | [OpenCode skills](https://opencode.ai/docs/skills/) | Compatible skill locations |
| C12 | [OpenCode MCP](https://opencode.ai/docs/mcp-servers/) | Local MCP server setup |
| C13 | [OpenCode plugins](https://opencode.ai/docs/plugins/) | Experimental compaction integration |

Use the installed client's documentation/version contract when generating configuration. In particular, newer OpenCode v2 documentation should not be assumed to describe the observed 1.x installation.

## Backlog planning practice

Reviewed on 21 October 2026 for [backlog planning](09-backlog-planning.md).

| ID | Primary source | Inspection and use |
| --- | --- | --- |
| B01 | [The 2020 Scrum Guide](https://scrumguides.org/scrum-guide.html) | Full text; refinement, sizing ownership, Definition of Done, empirical forecasting |
| B02 | [Jeffries: Card, Conversation, Confirmation](https://ronjeffries.com/xprog/articles/expcardconversationconfirmation) | Full text; stories as tokens confirmed by acceptance tests |
| B03 | [Wake: INVEST in Good Stories, and SMART Tasks](https://xp123.com/invest-in-good-stories-and-smart-tasks/) | Full text; story and task quality checks, spikes |
| B04 | [Humanizing Work Guide to Splitting User Stories](https://www.humanizingwork.com/the-humanizing-work-guide-to-splitting-user-stories/) | Full text; vertical slices, nine patterns, meta-pattern, split evaluation, Cynefin |
| B05 | [Cohn: The Dangers of a Definition of Ready](https://mountaingoatsoftware.com/blog/the-dangers-of-a-definition-of-ready) | Full text; guidelines over rules, stage-gate risk |
| B06 | [Jeffries: Story Points Revisited](https://ronjeffries.com/articles/019-01ff/story-points/Index.html) | Full text; origin and misuse of points, slicing instead |
| B07 | [Shepperd, Mair, Jørgensen: de-biasing anchoring](https://arxiv.org/abs/1804.03919v1) | Abstract-level; anchoring effect and partial mitigation |
| B08 | [ProKanban: story points with Monte Carlo](https://prokanban.org/blog/story-points-with-mcs) | Practitioner argument; no controlled comparison |
| B09 | [Ronanki et al.: ChatGPT for user story quality evaluation](https://arxiv.org/abs/2306.12132) | Abstract-level; QUS alignment, best-of-three |
| B10 | [Search-based optimisation of LLM shots for story points](https://arxiv.org/abs/2403.08430) | Full text; small GPT-4 study, zero-shot below mean/median baselines |
| B11 | [Story point estimation using transformer-based agents](https://eceasst.org/index.php/eceasst/article/view/2685) | Abstract-level; dataset accuracy claims, not adopted as targets |
| B12 | [Humanizing Work: "Claude, Make Me a Backlog"](https://www.humanizingwork.com/claude-make-me-a-backlog/) | Full text; AI-generated backlog failure mode and recommended uses |
| B13 | [Better Gherkin with AI](https://github.com/HumanizingWork/better-gherkin-with-ai) | Referenced from B12; not inspected |

## Tracker references

| ID | Primary source | Inspection and use |
| --- | --- | --- |
| T01 | [Azure Boards default processes](https://learn.microsoft.com/en-us/azure/devops/boards/work-items/guidance/choose-process?view=azure-devops) | Full text; types, hierarchy, task estimate fields, process immutability |
| T02 | [Azure DevOps team velocity](https://learn.microsoft.com/en-us/azure/devops/report/dashboards/team-velocity?view=azure-devops) | Search excerpts; Story Points, Effort, Size in velocity |
| T03 | [GitHub CLI: sub-issues, types, and dependencies](https://github.blog/changelog/2026-06-10-manage-sub-issues-types-and-dependencies-from-github-cli/) | Full text; `gh` 2.94.0 flags |
| T04 | [GitHub issue fields GA](https://github.blog/changelog/2026-07-02-issue-fields-are-now-generally-available) | Full text; default fields and MCP access |
| T05 | [GitHub agent automation controls](https://github.blog/changelog/2026-07-23-agent-automation-controls-in-github-issues-in-public-preview/) | Full text; suggestions, confidence, rationale; not a security control |
| T06 | [Remote Azure DevOps MCP Server](https://learn.microsoft.com/en-us/azure/devops/mcp-server/remote-mcp-server?view=azure-devops) | Full text; tools, toolset/read-only headers, auth, host support, OAuth scopes |
| T07 | [microsoft/azure-devops-mcp](https://github.com/microsoft/azure-devops-mcp) and [TOOLSET.md](https://github.com/microsoft/azure-devops-mcp/blob/main/docs/TOOLSET.md) | README and toolset; local domains, consolidation warning |
| T08 | [github/github-mcp-server](https://github.com/github/github-mcp-server) | README sections on toolsets, read-only, issue, sub-issue, and project tools |
| T09 | [Invariant Labs: GitHub MCP exploited](https://invariantlabs.ai/blog/mcp-github-vulnerability) | Search excerpts; toxic agent flow and mitigations |
| T10 | [GitHub MCP tool configuration and Lockdown mode](https://github.blog/changelog/2025-12-10-the-github-mcp-server-adds-support-for-tool-specific-configuration-and-more) | Full text; sanitization, lockdown, `X-MCP-Tools` |

Tracker tool names and parameters were read from documentation on this date; no live tracker calls were made.

## Session review

Reviewed on 21 October 2026 for [multi-angle session review](10-session-review.md).

| ID | Primary source | Inspection and use |
| --- | --- | --- |
| R01 | [Basili et al.: perspective-based reading](https://www.cs.umd.edu/users/basili/publications/technical/T136.pdf) and [replication](https://core.ac.uk/works/31885195) | Search excerpts; perspectives versus ad hoc reading of requirements |
| R02 | [Bacchelli and Bird: Expectations, Outcomes, and Challenges of Modern Code Review](https://www.microsoft.com/en-us/research/?p=164195) | Abstract-level; defects versus understanding and knowledge transfer |
| R03 | [Huang et al.: LLMs Cannot Self-Correct Reasoning Yet](https://arxiv.org/abs/2310.01798) | Abstract; intrinsic self-correction without external feedback |
| R04 | [Smit et al.: Should we be going MAD?](https://proceedings.mlr.press/v235/smit24a.html) | Search excerpts; debate versus self-consistency and ensembling |
| R05 | [Zhang et al.: Stop Overvaluing Multi-Agent Debate](https://arxiv.org/abs/2502.08788v3) | Abstract; 5 methods, 9 benchmarks, model heterogeneity |
| R06 | [Wataoka et al.: Self-Preference Bias in LLM-as-a-Judge](https://arxiv.org/abs/2410.21819) | Abstract-level; self-preference linked to perplexity |
| R07 | [Anthropic: How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) | Full text; token cost, effort scaling, delegation guidance |
| R08 | [Cognition: Don't Build Multi-Agents](https://cognition.com/blog/dont-build-multi-agents) | Full text; shared context and conflicting decisions |
| R09 | [Claude Code `code-review` plugin command](https://github.com/anthropics/claude-code/blob/main/plugins/code-review/commands/code-review.md) | Full text; parallel reviewers, validation, high-signal filter |

## Host review support

| ID | Primary source | Inspection and use |
| --- | --- | --- |
| H01 | [Claude Code hooks](https://code.claude.com/docs/en/hooks) | Lifecycle and decision-control sections; `Stop` versus `SessionEnd` |
| H02 | [Claude Code subagents](https://code.claude.com/docs/en/sub-agents) | Overview and configuration sections; own context, tools, locations |
| H03 | [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents) | Full text; triggering, inheritance, custom agents, no per-agent sandbox |
| H04 | [Codex hooks](https://learn.chatgpt.com/docs/hooks) | Full text; `Stop` continuation, `SessionEnd` timeout and advisory output |
| H05 | [OpenCode agents](https://opencode.ai/docs/agents/) | Full text; subagents, permissions, model per agent |
| H06 | [OpenCode plugins](https://opencode.ai/docs/plugins/) | Search excerpts; `session.idle` event |

## Session review trial

Run on 21 October 2026 for the [session review trial](11-session-review-trial.md). These are local observations, not published sources.

| ID | Source | Inspection and use |
| --- | --- | --- |
| X01 | Trial repository `~/repos/mine/GoApps/bearing-review-trial` (synthetic data, 7 task branches, Bearing tasks with recorded `go test` output) | Seeded defects and clean control under review |
| X02 | Trial kit `~/repos/mine/GoApps/bearing-review-trial-kit`: `make_task.py`, `tasks/*.json`, `bearing-review-panel/SKILL.md`, `run.sh` | Exact task content, panel skill, and run procedure |
| X03 | Raw run outputs `results.tsv` and `runs/*-claude-*.json` in the kit | Findings, `modelUsage` tokens, `total_cost_usd`, permission denials, wall time per cell |
| X04 | Claude Code 2.1.299 (`claude --version`), model `claude-opus-5-5`, `claude -p --output-format json` | Host and model for all 14 measured cells |
| X05 | `git status --porcelain` after each cell | Whether any review edited files |

Codex cells were not run; Codex CLI was not installed on the trial machine. The raw outputs live outside this repository and may be removed with the disposable trial.

## Local evidence and user requirements

| Evidence | Observation |
| --- | --- |
| User's engineering standards | Preserved verbatim in [ENGINEERING.md](../ENGINEERING.md) |
| User's product clarification | Standards must be built into Bearing and shape agent/platform behavior |
| Existing plan | [DESIGN.md](../DESIGN.md) records approved technology and product direction |
| Gortex live capability schemas | `remember.memory`, `remember.notebook`, `recall.surface`, `recall.distill` |
| Installed Gortex guide | General guide plus `tokens` and `views` topics |
| Active Gortex session policy | Tracked-source workflow, terminal localization, mutation checks, failure behavior |
| Workspace coverage | Bearing directory is untracked; graph operations unavailable here |

Local observations were gathered without changing Gortex tracking or saving experimental records into other projects. They establish exposed features and instructions, not storage guarantees or retrieval performance.
