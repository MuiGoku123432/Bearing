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
