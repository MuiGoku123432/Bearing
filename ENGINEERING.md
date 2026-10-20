# Engineering Standards — follow these on every task

## Core principle
Write the simplest code that solves the current task. You are optimizing
for a senior engineer reading this diff, not for hypothetical future
requirements.

## Abstraction rules (IMPORTANT)
- **Single-use code stays inline.** If logic is used in exactly one place,
  do NOT extract it into a shared util, helper, hook, or component. Local
  and inline is correct for single-use code. A private function within the
  same file is fine if it aids readability — but never move it to a shared
  location.
- **Rule of three:** only extract shared code when the SAME logic appears
  in 3+ places. Two occurrences = leave duplicated with a `// note:` comment.
- **No speculative generality.** No abstract base classes, generic type
  params, config options, plugin systems, or "manager"/"factory"/"provider"
  layers unless the current task explicitly requires them.
- **YAGNI:** implement only what this task needs. No future-proofing,
  no unused parameters, no "while I'm here" additions.

## Reuse rules
- Before creating ANY new function, type, component, or util, query Gortex
  (smart_context / semantic search) to check whether an equivalent already
  exists. Extend or reuse existing code before writing new code.
- Never create a second implementation of something that already exists.
  If the existing one is close but not exact, extend it or refactor it —
  don't fork it.

## File & component structure
- One responsibility per file. If a file mixes concerns (e.g. data
  fetching + rendering + formatting), split it — but only along real
  seams, not into fragments for its own sake.
- Target < 150 lines per component / < 300 per module. Exceeding this is
  a signal to split, not a hard rule — never split into pieces that can't
  be understood alone.
- Follow the existing folder conventions in this repo. Look at how
  neighboring code is organized and match it. Never invent a new
  structure or pattern when one already exists here.

## Dependencies & patterns
- No new dependencies, design patterns, or architectural layers without
  stating the justification in the plan and getting confirmation.
- Prefer boring, direct code: early returns over nesting, plain functions
  over classes where possible, standard library over clever one-liners.

## Self-review before finishing any task
Before declaring a task done, review your own diff and confirm:
1. Nothing was extracted/shared that is only used once
2. Nothing duplicates an existing symbol (verify via Gortex)
3. No abstraction exists that the current task didn't require
4. Files follow existing repo conventions and stay single-purpose
5. The diff is the SMALLEST reasonable change that completes the task
If any check fails, fix it before finishing.
