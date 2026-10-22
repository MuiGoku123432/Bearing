# Backlog planning: setup and use

Use Bearing to have an agent groom Azure Boards or GitHub work: epics, features, stories, tasks, and acceptance criteria. The agent drafts a plan, you approve it, then the agent creates the items.

Bearing does not connect to Azure or GitHub itself. The agent uses the tracker's own MCP server or CLI, which you set up once.

## 1. One-time setup

### Bearing

```sh
sh install.sh                                   # builds and installs bearing to ~/.local/bin
bearing install --host claude --repo /path/to/your/repo   # or codex / opencode
```

Restart your agent and approve the Bearing server if asked.

### Azure DevOps (pick one)

`@azure-devops/mcp` is an npm package, not a shell command. Run it through `npx` (needs Node.js 20+).

Claude Code:

```sh
claude mcp add azure-devops -- npx -y @azure-devops/mcp YOUR_ORG -d core work work-items
```

Codex (`~/.codex/config.toml`):

```toml
[mcp_servers.azure-devops]
command = "npx"
args = ["-y", "@azure-devops/mcp", "YOUR_ORG", "-d", "core", "work", "work-items"]
```

The first tool call opens a browser to sign in. Your account needs only **Work Items read & write** on the project.

### GitHub (pick one)

- **`gh` CLI (simplest):** install `gh` 2.94.0 or later, then run `gh auth login`.
- **GitHub MCP server:** add the remote server in your agent, using `--toolsets issues`. Add `--read-only` for grooming-only sessions.

Use a fine-grained token scoped to the one repository, with **Issues read & write** only.

## 2. Use it

Tell the agent, in plain words:

> Start a Bearing plan task. Groom the next slices of <feature> in <Azure project / GitHub repo> into stories with acceptance criteria. Don't write to the tracker until I approve.

The agent then:

1. **Learns the tracker.** It finds the item types and fields and saves them as a Bearing memory, so this step runs only once.
2. **Drafts a plan** in `docs/plans/<slug>.md`. The plan has a table of items, acceptance criteria for each, and a "Deferred variations" list for ideas left out.
3. **Stops and asks you to approve.** Read the plan. Ask for changes, or say "approved".
4. **Creates the items**, at most 10 per batch, parents first. Each item body ends with `Bearing plan: <slug>/<key>`. In Azure, items also get the tag `bearing-plan-<slug>`.
5. **Re-reads each item** to check it, records the tracker IDs in the plan's apply log, and finishes the task.

Size and estimate fields stay empty unless the plan records who approved the value.

## 3. Safe-use rules

- Use **read-only** tracker access for grooming, and enable writes only for the apply session.
- Work in **one project or repository per session**.
- Never put customer data, secrets, or production records in plans or work items.
- The agent never deletes, closes, reassigns, or moves items unless you ask for that change specifically.
- If an apply is interrupted, just ask the agent to continue. It searches for the plan key first, so it does not create duplicates.

## 4. Troubleshooting

| Problem | Fix |
| --- | --- |
| `no such file or directory: @azure-devops/mcp` | Run it through `npx -y @azure-devops/mcp ...`, or add it to your agent as shown above |
| `gh`: "As an Enterprise Managed User, you cannot access this content" | `gh` is signed in with your work account. Run `gh auth status`, then `gh auth switch` (or `gh auth login`) to use the account that owns the repo |
| Agent doesn't follow the planning rules | Make sure it started a task with `kind: "plan"`. Only plan tasks receive the rules on resume |
| Finish is rejected | A plan task must save its final plan document with `kind: "plan"` in the finish request |

More detail: [INSTALL.md](../INSTALL.md#tracker-access), [PLANNING.md](../PLANNING.md), [CLI examples](../CLI_EXAMPLES.md#plan-tracker-work).
