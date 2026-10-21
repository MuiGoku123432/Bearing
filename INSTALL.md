# Install Bearing

Bearing installs as a local executable plus a small integration in each Git checkout. Your coding client runs its model; Bearing stores tasks and memory; native Gortex provides source understanding and editing.

## Requirements

- Git and Go 1.26 or newer to build from source. The running binary does not need Go or a C compiler.
- Codex, Claude Code, or OpenCode installed separately.
- Native Gortex MCP connected to the selected client and covering the target checkout. Finding the `gortex` executable alone does not prove that connection.

The bootstrap script targets macOS and Linux. It builds the source in the checkout you run it from and may download the pinned Go modules. It does not fetch a release binary, invoke a model, change global client settings, edit shell profiles, or register repositories with Gortex. The installer has no new package dependencies.

## Install the binary and one project

From the Bearing checkout:

```sh
sh install.sh --repo /absolute/path/to/your/project --host codex
```

Use `--host claude` or `--host opencode` for another client. The default executable path is `~/.local/bin/bearing`. Set a different destination with `--bin-dir`:

```sh
sh install.sh --bin-dir "$HOME/tools/bin" --repo "$HOME/projects/my project" --host claude
```

The script builds with CGo disabled, checks that the new binary runs, then replaces the executable. If it replaces a different existing binary, it saves that binary beside the destination under `.bearing-previous.*` and prints the path. A failed build leaves the previous executable intact. Identical binaries are not replaced.

If the destination is not on PATH, add that directory to your shell's PATH or invoke the installed binary by its full path. The script prints the directory to add; it does not change your shell settings. Keep the binary at this location: project MCP settings use its absolute path.

To install only the executable:

```sh
sh install.sh
```

On Windows, build `bearing.exe` with Go, put it in a stable directory on PATH, and use the project command below. The shell bootstrap has not been tested on Windows.

## Connect a project

Preview changes first when working with an existing configuration:

```sh
bearing install --repo /path/to/project --host codex --dry-run
bearing install --repo /path/to/project --host codex
```

Run the command once for each desired client. A preview reads the project and renders temporary setup files but does not initialize or modify the checkout. Applying it initializes `.bearing/state.db` (the directory ignores itself), then installs these files:

| Client | Project MCP config | Instruction entry point | Workflow skill |
| --- | --- | --- | --- |
| Codex | `.codex/config.toml` | `AGENTS.md` | `.agents/skills/bearing/SKILL.md` |
| Claude Code | `.mcp.json` | `CLAUDE.md` | `.claude/skills/bearing/SKILL.md` |
| OpenCode | `opencode.json` | `AGENTS.md` | `.opencode/skills/bearing/SKILL.md` |

The skill holds the embedded WORKFLOW.md and tells the agent to call `bearing_resume` or `bearing_start` first; those return the full engineering policy, so the skill does not copy it. Existing project ENGINEERING.md files are not replaced. Instruction entry points receive one marked Bearing block, shared by clients that use AGENTS.md. Text outside that block stays verbatim. Installation supports the host's instruction mechanisms; it cannot force a model to obey them.

The Bearing entry in the project MCP config contains absolute paths to this machine's executable and checkout. Do not commit that entry for a team; it will not work on another machine. Each engineer runs `bearing install` in their own checkout. If the config file is already shared in Git, keep the Bearing change out of commits or merge it manually from `bearing setup`. The instruction block and skill contain no machine paths and can be shared.

JSON configuration retains unrelated values and other MCP servers. The installer updates Bearing's launch fields while preserving extra settings such as environment variables and timeouts. JSON layout may be reformatted. Codex configuration receives a marked TOML block; other text is preserved byte for byte. Reinstalling updates Bearing-owned content and saves backups before replacement. If the effective content is already current, no client files are rewritten or backed up again.

OpenCode's installed major version is detected automatically. When it is not installed yet, or when selecting a known deployment version explicitly:

```sh
bearing install --repo /path/to/project --host opencode --opencode-version 1
bearing install --repo /path/to/project --host opencode --opencode-version 2
```

Version 1 uses `mcp.bearing`; version 2 uses `mcp.servers.bearing`. The installer does not migrate the rest of an OpenCode configuration between versions. The V1 connection has been tested locally; V2 output follows its documented format and automated merge tests.

## Check the connection

Restart the selected client. Codex must trust the project before loading its project configuration. Claude Code asks you to approve project MCP servers. Follow the client's ordinary trust prompts; Bearing does not bypass them. Review any existing host overrides or disabled-server settings if Bearing does not appear.

In Codex, open the trusted project and use `/mcp` to inspect its servers. `codex mcp get` in the tested CLI version does not establish whether project settings are loaded. For the other clients, inspect status from the target checkout:

```sh
claude mcp list
opencode mcp list
```

Use the command for your selected client. Confirm that both Bearing and Gortex are available. Ask the agent to resume or start work using Bearing. Check that Gortex reports coverage for the checkout before changing source. A CLI check such as `bearing status --repo /path/to/project` verifies the local store but does not verify the client's MCP connection.

## When the installer stops

The installer validates all intended client files before initializing the store. It refuses symlinks, non-regular files, files larger than 1 MiB, malformed JSON, duplicate JSON keys, damaged Bearing markers, and a custom skill occupying the target path.

Automatic merging deliberately supports a limited set of configuration formats:

- Existing `opencode.jsonc` requires a manual merge, including JSONC that contains comments or trailing commas. The installer does not create a competing `opencode.json` beside it.
- Codex configurations with an unmanaged Bearing entry, inline or dotted `mcp_servers` assignments, Unicode escapes, or multiline strings require a manual merge. It avoids guessing at TOML structure without a parser dependency.
- Remote Bearing server definitions and conflicting OpenCode layouts require a manual merge.

Use the existing render-only command in those cases:

```sh
bearing setup --repo /path/to/project --host opencode --out /tmp/new-bearing-setup
```

Choose a new output directory and follow its APPLY.md. The bundle contains config fragments, instructions, a skill, and both policy documents. It does not change the project or client settings. OpenCode bundles include both V1 and V2 fragments.

## Updates and recovery

Run the bootstrap again from the desired source checkout to update the executable. Re-run `bearing install` for each project/client to refresh the embedded workflow and launch settings. If the executable path changed, those project settings must be updated.

Before changing client files, the installer writes exact original bytes and an INSTALL.json manifest under `.bearing/install-backups/<id>/`. The manifest lists newly created and updated paths, the Bearing version, `policy_id`, `workflow_id`, and the path and SHA-256 of every file it wrote; backups mirror updated paths. On a repeat install, a file whose content no longer matches the hash Bearing last wrote is reported as edited by you. It is still backed up before Bearing updates its own entry. Backups are private files and may contain private settings. Keep them local.

Ordinary write errors trigger rollback of completed client-file replacements. Initialization and newly created directories may remain. The installer is not a transaction across all files: a process kill or machine failure can leave a partial installation. To recover, stop the installer, inspect INSTALL.json, restore updated files from the matching backup paths, and remove newly created files only if they still contain the installer's content. Restore the intended file permissions too; backup copies are stored with private permissions. A stale `.bearing/install.lock` may be removed only after confirming no installer is still running, then the install can be retried.

To remove the integration manually, remove the Bearing MCP entry, its marked instruction block, and its installed skill for that client. Leave other MCP servers and instructions in place. Remove the installed executable only after other projects no longer use it. The `.bearing` database contains durable work; keep it or back it up before deleting it.

## Tracker access

Plan tasks groom Azure Boards or GitHub work through the tracker's own MCP server or CLI. Bearing does not call trackers, store their tokens, or configure their servers; set them up separately.

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

## Configuration references

Verified 2026-10-20 against [Codex project configuration](https://developers.openai.com/codex/config-basic/), [Claude Code MCP setup and project approvals](https://code.claude.com/docs/en/mcp), [OpenCode configuration](https://docs.opencode.ai/docs/config/), and [OpenCode V2 MCP configuration](https://opencode.ai/v2/docs/mcp-servers).
