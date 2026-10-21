# Git workflow

Bearing works on branches in one checkout. It can create and switch branches and commit reviewed staged changes. It does not create or use linked worktrees: removing a worktree deletes its ignored files, including Bearing state, so Bearing refuses to initialize or open one. The CLI and the `bearing_git` MCP tool take the same requests. Git must be installed and the repository needs an initial commit.

```sh
bearing git --repo /absolute/checkout --input git-request.json --json
```

Or pass JSON on standard input with `--input -`. MCP accepts the same JSON object. The MCP server stays bound to the checkout it started in.

## Branching practices Bearing applies

| Practice | What Bearing does |
| --- | --- |
| Named by purpose | New branches follow [Conventional Branch 1.1.0](https://conventional-branch.github.io/): `type/description` with types `feature`, `feat`, `bugfix`, `fix`, `hotfix`, `release`, `chore`, or an AI agent prefix (`ai`, `claude`, `codex`, `copilot`, `cursor`). Descriptions use lowercase letters, digits, single hyphens, and dots, for example `feat/add-login` or `release/v1.2.0`. |
| Start from the trunk | A new branch starts from the trunk unless you give `base`. The trunk is the remote's default branch when known, otherwise `main` or `master`. Bearing never fetches; it warns when the trunk is behind its upstream as of the last fetch. |
| No commits to the trunk | Bearing refuses to commit on `main`, `master`, `develop`, or the detected trunk. Create a branch first. |
| Keep work safe | Work started before branching moves to the new branch when it starts at the current commit. Bearing refuses to switch with uncommitted changes and never stashes, resets, or discards them. |
| Commit what was reviewed | A commit uses only already-staged changes and must match the staged tree you inspected. |
| Readable history | Commit messages follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). |
| No remote or history changes | No push, force, branch deletion, rebase, or automatic staging. |

Existing branches with other names can still be switched to and committed on; the naming rule applies when Bearing creates a branch.

## Inspect

```json
{"operation":"inspect"}
```

The response includes `repository.branch`, `repository.head`, `repository.dirty`, `trunk`, local `branches` (up to 100), a staged summary, and `index_tree` when the index has no conflicts. Review the staged diff through the host's Git tools before committing.

## Create or switch a branch

```json
{"operation":"branch", "branch":"feat/task-checkpoints"}
```

This creates the branch from the trunk and switches to it. Add `"dry_run": true` to preview. Use `base` to start elsewhere, for example `"base":"HEAD"` to stack on the current branch or carry uncommitted work from a non-trunk branch. An existing branch name is a conflict.

```json
{"operation":"switch", "branch":"fix/header-bug"}
```

Switching requires a clean checkout. Saved Bearing state stays with the checkout across branch changes.

## Commit reviewed changes

Stage only the intended files with the host's Git tools. Inspect, review the staged diff, and copy `index_tree` into `expected_index`:

```json
{
  "operation":"commit",
  "message":"feat(memory): keep corrections in recall\n\nShow the replacement claim and keep the old one auditable.",
  "expected_index":"<index_tree from inspect>"
}
```

The commit is refused when the staged tree changed, nothing is staged, HEAD is detached, the branch is a trunk, or the message is not a Conventional Commit: a lowercase type, optional scope, optional `!`, then `: ` and a description. A body needs a blank line after the subject. Breaking changes use `!` or a nonempty `BREAKING CHANGE:` footer. Native Git hooks and signing still run; Bearing warns if a hook changed the result.

## Recovery and boundaries

Git operations need the authorization appropriate to the host and the user's request. Git writes have no request IDs. On `outcome_unknown`, inspect the repository before retrying: a timeout or failed hook does not prove nothing changed. Commands have a 60-second deadline, so interactive signing or long hooks may need direct Git use. Dry runs create no branch or commit; inspection can write Git tree objects.

`.bearing/git.lock` prevents concurrent Bearing writes in one checkout; unrelated Git processes can still race. If a crash leaves the lock behind, remove it only after confirming the operation stopped. An unfinished merge, rebase, cherry-pick, revert, or bisect blocks writes. Environment variables that redirect Git's repository or index are rejected.

After a branch change or commit, checkpoint the result.
