# Codex, Claude Code, and OpenCode

Research date: 20 October 2026. Documentation support is not the same as a successful integration test.

## Shared state, different hosts

The three consumers should connect to the same Bearing database when they operate in the same checkout. They also connect directly to Gortex. Bearing does not own their model loop, permissions, compaction, or tool execution.

The first handoff should require only a saved task and a fresh session in another consumer. It should not require exporting a conversation or keeping the original agent alive.

Versions observed during the planning research:

| Component | Observed version | Verification still needed |
| --- | --- | --- |
| Codex | `0.163.0` | Actual tool discovery, repository binding, rules loading, errors |
| Claude Code | `2.1.263` | Same, including its memory and instruction-loading behavior |
| OpenCode | `1.18.33` | Same; distinguish 1.x documentation and configuration from v2 |
| Gortex | `0.64.5+3310720` | Exact checkout selection and policy behavior in a disposable tracked project |

These are an observation record, not minimum supported versions or a claim about the latest releases. Recheck before implementation.

## Instructions and skills

| Consumer | Documented mechanism | Recommended use |
| --- | --- | --- |
| Codex | Hierarchical `AGENTS.md`; repository skills under `.agents/skills` | Small workflow entry point; complete canonical engineering rules; procedure details on demand |
| Claude Code | `CLAUDE.md`, imports, and `.claude/skills` | Thin host entry point referencing the same maintained policy |
| OpenCode | `AGENTS.md`, explicit instruction files, compatible skill locations | Load canonical policy through its actual supported mechanism |

Sources: [Codex instructions](https://learn.chatgpt.com/docs/agent-configuration/agents-md), [Codex skills](https://learn.chatgpt.com/docs/build-skills), [Claude memory and instructions](https://code.claude.com/docs/en/memory), [Claude skills](https://code.claude.com/docs/en/skills), [OpenCode rules](https://opencode.ai/docs/rules/), [OpenCode skills](https://opencode.ai/docs/skills/).

Important differences:

- Codex documents that duplicate skill names are not merged. Avoid installing the same Bearing skill into every possible discovery directory.
- Claude's imports organize instructions but still load their content. They are not a token-saving mechanism.
- OpenCode uses `CLAUDE.md` as a fallback when `AGENTS.md` is absent. A plain file reference inside `AGENTS.md` is not automatic import expansion.

**Recommendation:** maintain one canonical policy and one workflow source. Package only the thin host-specific entry points needed for reliable loading. Do not maintain three independently edited policies. Test actual loaded content; file presence alone does not establish that the agent received it.

A short entry point can tell the agent where the policy lives, but the mandatory engineering rules must actually be loaded before work. On-demand workflow detail is appropriate; making mandatory rules optional is not.

The research folder itself is reference material. Do not automatically load every research document at session start.

## Native memory and Bearing

Claude's documented auto memory is repository-scoped, shared across worktrees, and loads a bounded initial portion of its memory index. That differs from Bearing's proposed per-checkout store. [Claude memory behavior](https://code.claude.com/docs/en/memory)

Do not treat host auto memory as a reliable transport between products. Do not disable or overwrite it by default either. Bearing records should remain identified by Bearing IDs and source references. If a host keeps a reminder, use a pointer to the authoritative record rather than a second editable copy of its claim.

A real handoff test starts from a fresh receiving session with no shared conversation. Otherwise the test may appear to prove durable memory while merely reusing the original prompt.

## MCP setup

All three document local MCP server configuration. Codex exposes command, arguments, environment, and working directory for stdio servers. Claude and OpenCode have their own configuration formats and scopes. [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude MCP](https://code.claude.com/docs/en/mcp), [OpenCode MCP](https://opencode.ai/docs/mcp-servers/)

Use an explicit absolute repository argument when starting Bearing. Treat startup working directory as a convenience, not the sole protection against opening the wrong database.

Configuration installation should:

1. Show the exact files and entries it intends to change.
2. Preserve unrelated servers and settings.
3. Avoid duplicating an existing Bearing entry.
4. Leave Gortex's existing connection and daemon management alone.
5. Verify launch and a read operation before claiming setup succeeded.

Exact generated snippets should wait for the chosen CLI flags and tested consumer versions. Publishing plausible-looking but untested configuration is worse than recording the integration contract.

## Hooks are an optional improvement

Hooks do not provide a portable persistence contract:

| Host | Documented behavior | Design consequence |
| --- | --- | --- |
| Codex | `PreCompact` ignores plain stdout; JSON follows hook output rules | Printing a checkpoint does not automatically inject it into resumed context |
| Claude Code | Has pre/post-compaction events with event-specific output behavior | A similarly named event is not a Codex-compatible adapter |
| OpenCode | Documents `experimental.session.compacting` and context injection; replacing its prompt is a separate operation | Experimental plugin behavior must be version-tested |

Sources: [Codex hooks](https://learn.chatgpt.com/docs/hooks), [Claude hooks](https://code.claude.com/docs/en/hooks), [OpenCode plugins](https://opencode.ai/docs/plugins/).

Start with explicit checkpointing at useful work boundaries. A crash, forced termination, or abrupt session closure may prevent a lifecycle hook from running. A hook cannot reconstruct facts the agent never saved.

If missed checkpoints become a measured problem, add one narrowly scoped host adapter at a time. It should be bounded, idempotent, and clear about failure. Do not replace the host's compaction prompt as a first integration step.

## Context cost and caching

Measure the complete additional context: instruction entry point, loaded rules, skill metadata, tool definitions, memory responses, and repeated writes. Count the host's native overhead separately so Bearing is not credited or blamed for unrelated tooling.

OpenAI's caching documentation describes prefix reuse that depends on matching content and request settings. Stable text is helpful, but cache behavior belongs to the host and model. [OpenAI prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching)

**Recommendation:** keep rule text and tool definitions stable; place dynamic task state in tool results. Do not promise cache hits or inflate instructions to chase a provider-specific threshold. Report uncached input, cached input, output, wall time, and successful outcomes separately when available.

Cached tokens can still occupy context. Fewer bytes returned by Bearing do not automatically mean lower total cost if the agent needs more calls or repeats its investigation.

## Handoff acceptance

Exercise all six directions: Codex to Claude, Claude to Codex, Codex to OpenCode, OpenCode to Codex, Claude to OpenCode, and OpenCode to Claude.

For each direction:

1. Save a decision, an unfinished task, a failed attempt, and a specific next action.
2. Close the sender session.
3. Open a fresh receiver bound to the same checkout.
4. Confirm it retrieves the correct task and original record IDs.
5. Change a relevant source file before resume in a separate variant.
6. Confirm the receiver revalidates the code evidence through Gortex.
7. Complete the work and inspect the actual result.

Test subdirectory launch, paths with spaces, two simultaneous consumers, a wrong repository, missing executable, denied write access, and unsupported protocol/schema versions.

The first release can claim compatibility only for tested versions and cases. Documentation discovery is the starting point for that matrix.
