# CLI and MCP interface

Research date: 20 October 2026. Commands and schemas below are proposed contracts.

## One executable, one set of behavior

Use the Go executable for human commands and a stdio MCP server. Both interfaces should invoke the same concrete operations and validation rules. The command parser and protocol adapter have different responsibilities; neither should implement a second memory store.

This requirement justifies sharing the actual operation implementation. It does not justify a generic repository interface, service container, transport framework, or a utility layer. Apply the user's rule of three to incidental helper extraction; do not duplicate the product's central behavior merely to satisfy a numerical reading of that rule.

## Protocol and SDK selection

The [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) is the preferred proposal. Its documented compatibility table includes protocol `2026-07-28` in SDK `v1.7.0+` and older protocol versions. Pin a release after checking the three actual consumers; this research does not establish that each installed client supports the newest version.

The versioned [2026-07-28 tool specification](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) differs from the [2025-11-25 specification](https://modelcontextprotocol.io/specification/2025-11-25/server/tools), including request metadata and response structure. Delegate wire behavior to the SDK and test compatibility. Do not hand-build a protocol from an old tutorial.

Use stdio first. The existing clients launch the executable locally. A remote HTTP server would introduce deployment, identity, authentication, and network availability concerns without satisfying a current requirement.

## Seven operations are a hypothesis

Keep the current plan's small tool set for initial evaluation:

| Operation | What it must do | What it must not imply |
| --- | --- | --- |
| `start` | Create a task with a concrete outcome | A new multi-stage process is mandatory |
| `resume` | Return selected task, latest checkpoint, relevant memories, and scope warnings | Prior observations are still current |
| `remember` | Save, supersede, or retract one claim with evidence | The claim becomes a higher-priority instruction |
| `recall` | Retrieve bounded records or expand selected IDs | Search results are exhaustive |
| `checkpoint` | Atomically save progress and advance task version | Source files or external systems are rolled back |
| `finish` | Save outcome and verification evidence and close the task | Model assertions independently prove success |
| `rules` | Return the canonical policy and its version | A short memory summary can replace the policy |

MCP names can use the `bearing_` prefix. CLI subcommands use the short names. Keep maintenance operations such as backup and restore out of the always-advertised model tool list.

Tool design should follow real user actions and be evaluated with agents, rather than exposing every internal table as an endpoint. This is consistent with [Anthropic's tool design guidance](https://www.anthropic.com/research/writing-tools-for-agents). It does not prove that seven is the best count.

If `finish` and `checkpoint` are repeatedly confused, improve the descriptions or compare a combined operation. If a generic operation with many modes produces invalid calls, prefer clear verbs. Measure errors and context cost before changing the interface.

## Contracts that prevent ambiguity

Common mutation inputs should include an application request key. Updates include an expected record/task version. Let the tool derive its bound checkout instead of asking the agent to restate it.

Use stable IDs; titles are display text. Reject a checkpoint targeting a missing task. When several unfinished tasks match, return short choices with IDs rather than silently continuing the most recent one.

`resume` should combine the data ordinarily needed together. That avoids requiring a task lookup, checkpoint lookup, memory lookup, and scope lookup as four round trips. It should not return the whole database.

`recall` should support expansion by ID. Without an expansion path, excerpts either hide useful evidence or become full records by default. This can remain one tool with two explicit, mutually exclusive input modes.

Do not add options because they might be useful later. Define the minimal inputs from actual acceptance scenarios.

## Errors and uncertain outcomes

| Case | Machine behavior | Human message |
| --- | --- | --- |
| Writer contention | Retryable busy error with bounded wait | “Progress was not saved. Another write is still running. Retry with the same request ID.” |
| Version conflict | Conflict plus current version | “This task changed in another client. Reload it before saving.” |
| Lost response | Outcome may be unknown until request-key lookup/retry | “The result could not be confirmed. Retry with the same request ID.” |
| Invalid scope | Validation error | “This memory needs a project or checkout scope.” |
| Newer database schema | Refuse mutation | “This database was created by a newer Bearing version. Update Bearing to open it.” |
| No task selected | Short candidate list or explicit empty state | “Choose one of these unfinished tasks.” |

Do not label a request “not saved” when commit may have succeeded. Distinguish a confirmed rollback from a lost response. Do not encourage a new request key for an uncertain retry.

The MCP tools specification distinguishes protocol errors from tool execution failures and provides `isError` for failed tool execution. Structured results must match a declared output schema. Annotation hints are not authorization. [MCP tool contract](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)

## Output and context cost

Human CLI output should begin with the result and next useful action. JSON mode should provide a stable machine-readable object and a nonzero exit status on failure. Diagnostics go to stderr. MCP stdout must contain protocol traffic only, including during startup errors.

Include IDs, versions, evidence labels, scope, and truncation fields. Avoid decorative wrappers and repeated instructions in every result.

The older MCP specification recommends a serialized JSON text block alongside structured output for compatibility. The best output shape therefore needs client testing; a short text sentence alone may be insufficient for an older consumer. [MCP structured content](https://modelcontextprotocol.io/specification/2025-11-25/server/tools#structured-content)

Test whether each host renders or feeds both representations to the model. A valid server response does not establish that token duplication is absent. Keep compatibility and measured context cost separate.

## Process lifecycle

An MCP process holds its repository binding and database connection. It should handle normal EOF, cancellation, and termination cleanly. It must not infer persistent task state from the connection lifetime: restarting the server does not mean starting a new task.

Set bounded database operations and return meaningful errors. Avoid a background polling worker in the first version. There is no need for the server to watch the repository, run Gortex, launch models, or execute verification commands.

A checkpoint records what happened; the agent and its consumer perform the work. This boundary keeps permissions and execution in the host where the user already controls them.

## Interface acceptance

Run the same success and failure cases through CLI and MCP and compare persisted results, not formatting. Cover discovery, tool schemas, clean stdout, malformed inputs, cancellation, repeated writes, conflict reporting, and reopen after restart.

Then connect each real consumer and confirm that it can select the right tool, interpret a failure, expand a record, and resume a task. A synthetic MCP client proves protocol behavior; it does not prove host usability.
