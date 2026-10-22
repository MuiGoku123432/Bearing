# Onboarding a repository into Bearing

Research date: 22 October 2026. Question: what should an agent do, once, to bring an existing repository into Bearing so that later sessions start with useful, correct context?

Sources: the [source ledger](SOURCES.md#project-onboarding), IDs O01, O03, and O04, and Bearing's own contract and code.

## Findings

1. **Installation is the user's step, not the agent's.** `bearing install` writes host config, the instruction entry, and the skill, then the host must restart and approve the server [CONTRACT.md, INSTALL.md]. An agent can only onboard after the Bearing tools are available. The first useful check is a working `bearing_resume`.
2. **Gortex coverage is required but is not Bearing's to change.** INSTALL.md says Bearing does not register repositories with Gortex, and that an executable on PATH does not prove coverage. `gortex track <path>` adds the repository to Gortex's global config [O04]. In the spike S1 trial, the untracked repository forced every reviewer to fall back to file reads [research 11]. **Implication:** check coverage, and if it is missing, ask the user before running `gortex track`.
3. **Persistent memory should hold only what the code cannot tell you.** Claude Code's auto memory skips anything derivable from the codebase, such as architecture, file paths, or fixes. Its `/doctor` trim removes directory layouts, dependency lists, and architecture overviews, and keeps pitfalls, rationale, and conventions that differ from defaults [O01]. Instruction files work best when short, concrete, and verifiable, for example "Run `npm test` before committing" [O01]. Bearing already has Gortex for code facts, and resume recalls only three memories within 6,000 bytes [context.go]. **Implication:** save a handful of high-value, non-derivable claims: the verified build and test commands, the trunk branch, protected areas, and team decisions. Do not save a codebase summary.
4. **Existing instruction files are the best starting evidence.** Claude Code's `/init` reads existing instruction and rules files from other tools and builds on them rather than overwriting [O01]. Bearing install appends to `AGENTS.md`/`CLAUDE.md` and preserves the team's content. **Implication:** read them, then save their durable rules as memories with `basis: observed` and the file as `source`, instead of rewriting them.
5. **A claimed command must be run before it is saved.** Bearing records provenance, not confidence. `observed` should mean the agent saw it work [CLI_EXAMPLES.md]. A build or test command copied from a README may be stale.
6. **Bearing memory has the right shape already.** Kinds `decision`, `constraint`, `lesson`, and `preference`; basis `user`, `observed`, or `inferred`; scope `project` or `checkout`; anchors on paths or symbols; supersede and retract [CONTRACT.md]. Recall the existing memories before saving, to avoid duplicates. No new code or table is needed.
7. **Tracker setup belongs to planning.** The tracker profile is a planning memory [PLANNING_SPEC.md §5]. Onboarding should only offer it when the user plans to groom a backlog.
8. **The skill is the right place.** The workflow is already delivered through the installed skill and `bearing rules` [CONTRACT.md]. One more WORKFLOW.md section reaches all three hosts without new install surface. The Agent Skills format loads only the name and description at startup and the body on activation, and the description should name the tasks it covers [O03], so the skill description should mention onboarding. The body stays well under the recommended 5,000 tokens.

## Recommendation

Add an "Onboard" section to WORKFLOW.md and mention onboarding in the skill description. The steps:

1. Confirm Bearing works: `bearing_resume`. If the tools are missing, tell the user to run `bearing install` and restart.
2. Check Gortex coverage. If it is missing, ask before running `gortex track`.
3. Recall existing project memories.
4. Read the instruction files, README, and build files. Run the build and test commands.
5. Ask only the questions that change future work: trunk branch, areas not to touch, required checks, tracker.
6. Save at most about ten memories, non-derivable only. Use `observed` with a source for what was seen working, and `user` for what the user said.
7. Resume again to confirm the memories surface, then report what was saved and what is missing.

Never save secrets, tokens, connection strings, customer data, or production records.

No task kind, table, or code path is needed. A test confirms that the installed skill contains the section.

## Not chosen

- **Generate a project summary memory.** Gortex derives it fresh, while a saved copy goes stale and uses recall budget [O01].
- **Run `gortex track` automatically.** It changes global config outside Bearing's boundary [INSTALL.md].
- **A separate onboarding skill or `onboard` task kind.** It adds install surface for a one-time procedure. The research → spec → plan → build kinds and plain memory writes cover it.
