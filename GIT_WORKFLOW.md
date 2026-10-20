# Git workflow

Bearing can create and switch branches, create worktrees with a copy of saved task state, and commit reviewed staged changes. The CLI and `bearing_git` MCP tool use the same requests. Git must be installed and the repository must already have an initial commit for these writes.

For the CLI, save a request as JSON and run:

```sh
bearing git --repo /absolute/checkout --input git-request.json --json
```

Or pass JSON on standard input with `--input -`. MCP accepts the same JSON object directly. The MCP server stays bound to the checkout it started in.

## Inspect first

```json
{"operation":"inspect"}
```

The response includes `repository.head`, `repository.branch`, `repository.dirty`, local `branches`, `worktrees`, a staged summary, and `index_tree` when the index has no conflicts. A detached checkout omits the branch; use the literal `HEAD` as its expected branch.

Use the observed values in write requests. The examples below use placeholders; replace them with the actual response values. Inspection summarizes staged files; review their diff through the host's Git tools before committing. Branch and worktree lists are bounded at 100 entries, with warnings when omitted. Other Git output is bounded at 64 KiB.

## Create or switch a branch

```json
{
  "operation":"branch",
  "branch":"feat/task-checkpoints",
  "expected_head":"<observed HEAD>",
  "expected_branch":"main",
  "dry_run":true
}
```

Remove `dry_run` or set it to `false` to create and switch to that branch. `base` optionally selects a different starting commit or ref; it defaults to the inspected HEAD. An existing branch is a conflict. Use `operation: "switch"` with the target `branch` to switch to an existing local branch. Branch creation and switching require a clean checkout. Saved Bearing state stays in this checkout across branch changes.

## Create a worktree

Checkpoint progress and save the selected research/spec documents before creating a worktree:

```json
{
  "operation":"worktree",
  "branch":"feat/task-checkpoints",
  "path":"/absolute/new-checkout",
  "expected_head":"<observed HEAD>",
  "expected_branch":"main"
}
```

This creates a new branch and worktree, then restores a consistent SQLite snapshot into its `.bearing/state.db`. The destination must be new, outside the source checkout, with an existing parent directory. The source branch stays unchanged. Dirty files remain in the source checkout; their saved document snapshots still travel in the database.

Task IDs, checkpoints, memory, request receipts, and document lineage are preserved. Future updates are independent: this is a copy, not shared live state. Resume the same task ID in the destination and review checkout/HEAD warnings. Project memories remain available; memories scoped to the old checkout do not apply by default.

Reinstall the client for the new path, for example:

```sh
bearing install --repo /absolute/new-checkout --host codex
```

Choose `claude` or `opencode` for those clients. Existing tracked client settings may still point to the old checkout until reinstalled. Gortex remains separately configured; use its active checkout/worktree view. Bearing does not register a second graph.

A worktree response can succeed at creating the checkout but report `state_copied: false` with warnings. Inspect it before continuing. `recovery_backup`, when present, names the consistent SQLite backup retained in the source checkout. If the destination has no database, restore it with:

```sh
bearing restore --input /absolute/retained-backup.db --repo /absolute/new-checkout
```

Do not overwrite a database that already contains new work. After confirming recovery, the retained backup can be removed explicitly.

## Commit reviewed changes

Stage only the intended files using the host's Git tools. Inspect again and copy its `index_tree` into `expected_index`:

```json
{
  "operation":"commit",
  "message":"feat(memory): preserve research across worktrees\n\nCopy saved documents when creating a worktree.",
  "expected_head":"<observed HEAD>",
  "expected_branch":"feat/task-checkpoints",
  "expected_index":"<observed index_tree>"
}
```

The commit uses only already-staged content. It rejects a changed HEAD, branch, or staged tree; detached HEAD; empty staged changes; and an invalid message. Native Git hooks and signing still run. A hook can rewrite the message or change content: Bearing reports warnings if the resulting commit differs from the reviewed request. Inspect any warning before proceeding.

Validate a message without committing:

```json
{"operation":"check-commit","message":"fix(cli): preserve unrelated settings"}
```

Messages follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/): a lowercase type, optional scope, optional `!`, then `: ` and a description. `feat`, `fix`, `docs`, `test`, `refactor`, and `chore` are common choices; custom lowercase types are allowed. A body needs a blank line after the subject. Breaking changes use `!` or a nonempty `BREAKING CHANGE:` / `BREAKING-CHANGE:` footer. The validator checks format, not whether a semantic-version claim matches the code.

## Recovery and boundaries

Git operations require the authorization appropriate to the agent host and user request. There is no automatic staging, push, stash, reset, branch deletion, or worktree removal.

Git writes do not use SQLite request IDs. On `outcome_unknown`, inspect the actual repository before retrying. A timeout or failed hook does not prove that no action occurred. Commands have a 60-second deadline, so interactive signing or long hooks may require direct Git use. Dry runs create no branches, commits, or worktrees; inspection can write Git tree objects.

Bearing checks preconditions before execution and checks the result afterward. Its `.bearing/git.lock` prevents concurrent Bearing writes in one checkout, but unrelated Git processes can still race. Avoid other Git writers during the operation. A crash may leave the lock file behind; remove it only after confirming the original operation has stopped and inspecting the repository. Existing merges, rebases, cherry-picks, reverts, or bisects block writes until resolved. Environment overrides that redirect Git's repository or index are rejected.

After a successful change, checkpoint its result. Refresh completion evidence against the current HEAD before finishing a task. See the native [Git worktree](https://git-scm.com/docs/git-worktree) and [Git commit](https://git-scm.com/docs/git-commit) documentation for Git's own semantics.
