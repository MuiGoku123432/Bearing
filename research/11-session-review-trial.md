# Session review trial (spike S1)

Trial date: 21 October 2026. Question: does a four-angle reviewer panel catch more than the current single self-review on Bearing tasks, and at what cost? This is spike S1 from the [session-review plan](../docs/plans/session-review.md); background is in [research 10](10-session-review.md). Sources and local observations are listed under [Session review trial](SOURCES.md#session-review-trial).

**Status:** Claude Code half complete. The Codex half was not run: Codex is not installed on the trial machine. Questions 2 and 4 are answered for Claude Code only.

## Result in brief

- On these seven tasks, the panel found **no seeded defect that the single self-review missed**. Both found 6 of 6, and both cleared the clean control.
- The panel cost **3.1 times as much** ($1.57 vs $0.50 per review) and took **2.3 times as long** (85 s vs 37 s).
- The panel's extra findings were mostly real but minor: test gaps and edge cases. None was blocking, and none changed a verdict.
- No reviewer edited files, and no blocking finding was a false positive.
- **Recommendation: do not build S2 as written.** Run a harder trial first, described under "What would change this". Stop if that trial also shows no difference.

## Method

**Repository.** `~/repos/mine/GoApps/bearing-review-trial` is a disposable, stdlib-only Go module with synthetic data. `main` holds accounts, discounts, and `NormalizeName`. Each task is a branch with one commit. Each branch has a Bearing build task, created with `bearing start`, and one checkpoint holding the real `go test -count=1 -v ./...` output. Tests pass on every branch, so each defect is one that tests miss. Code, commit messages, and task text do not hint at the defects.

**Kit.** `~/repos/mine/GoApps/bearing-review-trial-kit` sits outside the trial repository, so reviewers cannot read the answer key. It contains:

- `make_task.py`, which builds a task branch
- `tasks/*.json`, the task definitions
- `task_ids.json`
- `bearing-review-panel/SKILL.md`, the panel skill (copied into the trial repository's `.claude/skills` and `.agents/skills`)
- `run.sh`, which runs one cell
- `results.tsv` and `runs/`, the raw outputs

**Answer key.**

| Key | Branch | Bearing task | Seeded problem | Where |
| --- | --- | --- | --- | --- |
| t1 | task/free-shipping | 80c6ace4… | Logic bug: `>` where criterion says "5000 or more" | shipping.go:7 |
| t2 | task/gold-tier | 2d460e1e… | New `gold` branch has no test | order.go:8 |
| t3 | task/normalize-new-accounts | 93d80302… | `cleanName` duplicates existing `NormalizeName` | create.go:10 |
| t4 | task/search-accounts | 048ca3f4… | String-concatenated SQL | search.go:7 |
| t5 | task/export-csv | 037c2224… | Criterion a2 (quote commas/quotes) not met | export.go:13 |
| t6 | task/sales-tax | 2691ba48… | Strategy interface, registry, and factory for one rate | tax.go:3-27 |
| c0 | task/display-name | ab011249… | None (clean control) | — |

**Modes.**

- *Single:* one prompt asks the agent to resume the task and perform Bearing's existing pre-finish self-review: the acceptance criteria plus the five engineering checks.
- *Panel:* the prompt invokes the `bearing-review-panel` skill. The skill runs four read-only subagents, each starting from the same `bearing_resume` output and diff. The angles are requirements, correctness and tests, engineering policy, and security and data. The lead then merges duplicates, confirms each finding against code, recorded output, or Gortex, and drops unconfirmed findings with a reason.

In both modes the agent must not edit files or call `bearing_finish`.

**Host.** Claude Code 2.1.299, model `claude-opus-5-5`, run as `claude -p --output-format json`.

- The allowlist permits only read tools, Bearing resume and recall, Gortex, subagents, skills, and `git diff/log/show`.
- An edit would have appeared as a permission denial or as a dirty tree. `run.sh` checks the tree with `git status --porcelain` after every cell.
- Tokens and cost come from `modelUsage` and `total_cost_usd`, which include subagents. Wall time is measured by `run.sh`.
- Bearing is the version built from `feat/planning-tasks`.

## Run sheet and results

Scoring:

- **Found:** the seeded problem is reported at the right location with a blocking or high severity.
- **FP:** a finding that is wrong on the facts.
- **Debatable:** the facts are right but the issue is questionable.

| Task | Mode | Found seeded | Severity | Other confirmed findings | FP | Debatable | Wall s | In+cache tokens | Out tokens | Cost $ | Edited files |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| t1 | single | yes | high | boundary not tested (high) | 0 | test-file placement, unnamed 799 | 38 | 396,326 | 2,440 | 0.59 | no |
| t1 | panel | yes | blocking | boundary not tested | 0 | test-file placement, unnamed 799 | 88 | 849,831 | 12,061 | 1.55 | no |
| t2 | single | yes | blocking | — | 0 | — | 40 | 278,915 | 1,864 | 0.39 | no |
| t2 | panel | yes | blocking | — | 0 | — | 58 | 668,993 | 8,759 | 1.39 | no |
| t3 | single | yes | blocking | not minimal (same cause) | 0 | file placement | 33 | 312,464 | 2,689 | 0.48 | no |
| t3 | panel | yes | blocking | single-case test, ID unchecked, empty name | 0 | — | 83 | 1,048,932 | 14,656 | 1.75 | no |
| t4 | single | yes | critical | no test for a1, LIKE wildcards unescaped | 0 | file placement | 49 | 366,308 | 2,940 | 0.50 | no |
| t4 | panel | yes | blocking | no test for a1, LIKE wildcards unescaped | 0 | file placement | 77 | 718,261 | 11,909 | 1.46 | no |
| t5 | single | yes | blocking | a2 evidence overclaimed, CSV formula injection | 0 | — | 32 | 445,369 | 2,740 | 0.54 | no |
| t5 | panel | yes | blocking | a2 evidence overclaimed, CSV formula injection, weak assertion | 0 | — | 78 | 813,361 | 12,677 | 1.51 | no |
| t6 | single | yes | high | unused exported API and race, convention, negative rounding | 0 | — | 28 | 255,887 | 2,329 | 0.45 | no |
| t6 | panel | yes | blocking | mutable registry, convention, negative rounding, zero test | 0 | — | 119 | 1,229,246 | 17,040 | 1.79 | no |
| c0 | single | n/a (verdict ready) | — | — | 0 | file placement ×2 | 38 | 484,334 | 3,211 | 0.54 | no |
| c0 | panel | n/a (verdict ready) | — | outcome mentions listings that do not exist (note) | 0 | test-file placement | 92 | 911,345 | 12,579 | 1.56 | no |
| t1–c0 | codex single/panel | **not run** | | | | | | | | | |

Totals across 7 tasks:

| Mode | Seeded found | Control cleared | Blocking FPs | Wall time | In+cache tokens | Out tokens | Cost |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Single | 6/6 | yes | 0 | 258 s (avg 37) | 2,539,603 (avg 362,800) | 18,213 (avg 2,602) | $3.49 (avg $0.50) |
| Panel | 6/6 | yes | 0 | 595 s (avg 85) | 6,239,969 (avg 891,424) | 89,681 (avg 12,812) | $11.02 (avg $1.57) |

Most input tokens are prompt-cache reads, which is why cost rises less than tokens.

## Answers to the S1 questions

1. **Seeded problems found and false positives.** Single: 6/6 found, 0 wrong findings. Panel: 6/6 found, 0 wrong findings. Both raised a few debatable convention notes, mostly that new tests went in a new file while `main` had one test file. Neither raised a blocking false positive, and both said the clean control was ready to finish.
2. **Tokens and wall time (Claude Code).** Per review, the panel used 2.5× the input tokens, 4.9× the output tokens, 2.3× the wall time, and 3.1× the cost. Codex was not measured.
3. **Combining step.** It dropped two findings and merged one: the source-file placement in t1, the "duplicate implementation" framing in t5 (the repository had no CSV helper), and a duplicate in t4. All three decisions were correct. No real finding was lost. Debatable notes about test-file placement survived because they are factually true. Inside the panel, reviewers often flagged issues outside their own angle; for example, the security and policy reviewers also reported the t1 boundary bug. The merge step absorbed this, so it raised cost without changing results.
4. **Did Codex reviewers edit files?** Not measured. In Claude Code, no run changed the tree. The 17 permission denials were all reads or `go test` attempts outside the allowlist; none was an edit attempt.
5. **Recommendation:** see the next section.

## Recommendation

**Do not build S2 as written.** On this evidence the panel adds cost without adding detection.

The trial has a ceiling effect. Each task holds one defect in a 10–30 line diff, and the single reviewer is a strong model with the full policy. A single review found everything, so these tasks cannot show a panel advantage even if one exists. The panel's distinct value was a slightly longer tail of minor test and edge-case notes. That is useful, but it does not justify three times the cost on every larger change.

Choose between:

- **Stop** (recommended if no harder trial is wanted): keep the single self-review and close F1. A user who wants a panel can still ask for one.
- **Adjust and retest before S2**: run S1b using the changes below. Build S2 only if the panel finds confirmed blocking issues that single review misses, on at least a few tasks, at an acceptable cost.

### What would change this

- Larger diffs (200+ lines) with two or three defects each, including subtle ones: a concurrency bug, an authorization check missing on one path, or a criterion met only in the happy path.
- At least one reviewer from a different model family. Research 10 found this to be the most consistently helpful variation [R05].
- Fewer angles. Requirements plus correctness and tests cover most findings here. Security could run only when the diff touches input handling, SQL, auth, or secrets.
- The Codex half, to answer question 4 and to compare a host that cannot enforce read-only access per agent [H03].

## Limitations

- One host, one model, one run per cell. There is no variance estimate; a second run could differ.
- Gortex did not track the trial repository, and tracking is a user decision. Reviewers checked reuse by reading files, which is reliable at this repository size. The reuse defect (t3) was still found in both modes.
- Reviewers could not run `go test`, by design; they used the recorded output. Some reviews said so.
- The seeded defects are textbook cases. Real sessions have larger, ambiguous diffs.
- Token counts and costs are the host's own report at list price.

## How to run the Codex half

On a machine with Codex CLI installed and signed in:

1. Run `bearing install --host codex --repo ~/repos/mine/GoApps/bearing-review-trial`. Already done here; repeat only if paths differ.
2. Confirm `.agents/skills/bearing-review-panel/SKILL.md` exists in the trial repository.
3. From the kit directory, run:

   ```sh
   for k in t1 t2 t3 t4 t5 t6 c0; do for m in single panel; do sh run.sh $k codex $m || break 2; done; done
   ```

4. `run.sh` uses `codex exec --json --sandbox workspace-write`, so an edit attempt is observable rather than blocked. If a run edits files, the script stops. Record the edit, then restore the trial repository yourself before continuing. Token usage is in the `runs/*-codex-*.jsonl` events.
5. Score the outputs against the answer key above, and add the rows to the run sheet.

## Panel skill used

The full text is in the kit at `bearing-review-panel/SKILL.md`. In summary:

- The lead calls `bearing_resume`, takes `git diff main...HEAD`, and gives four read-only subagents the same input, verbatim.
- Each subagent returns findings with `severity`, `location`, `claim`, and `evidence`, or reports no findings.
- The lead merges duplicates, confirms each finding against code, recorded output, or Gortex, and drops the rest with a reason.
- The report uses fixed sections: CONFIRMED, DROPPED, REVIEWERS, and VERDICT.
- If subagents are unavailable, the lead stops instead of simulating them.
