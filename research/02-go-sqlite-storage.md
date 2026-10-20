# Go, SQLite, and recovery

Research date: 20 October 2026. This is a storage recommendation and test plan. No driver, database, migration, or recovery experiment has been installed or run.

## Recommended starting point

Use one local database per checkout, short transactions, explicit conflict detection, and a small Go implementation. Several CLI invocations and MCP processes may access the same file. A process-local mutex cannot protect against the other processes.

SQLite permits concurrent readers but serializes writes. A read transaction that tries to upgrade after another writer commits can fail with `SQLITE_BUSY_SNAPSHOT`; retrying only its final statement does not repair its stale snapshot. [SQLite isolation](https://sqlite.org/isolation.html)

Bearing writes are small and infrequent compared with a typical database workload. This favors a direct implementation. There is no demonstrated need for a daemon, job queue, ORM, event-sourcing framework, or database service.

## Driver choice

| Candidate | Documented property | Bearing implication |
| --- | --- | --- |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | SQLite driver without CGo; documentation warns that its `modernc.org/libc` dependency version must match appropriately. | Preferred candidate for portable releases, subject to a pinned-version build and behavior check. |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | Uses CGo and requires a C compiler to build. | Credible alternative if packaging and driver behavior justify the toolchain requirement. |

Neither is selected merely because it is popular. Compare the exact versions on required operating systems, embedded SQLite version, binary size, startup time, context cancellation, FTS availability, backup support, and maintenance burden.

The package page fetched for modernc showed `v1.60.1` and explicitly said it was not the latest module version. This research does not establish the best release to pin. Do not turn that observation into a dependency recommendation without checking the module and bundled engine.

**Dependency gate:** the user has approved Go and SQLite, but concrete packages still need confirmation under the engineering rules. First prepare the smallest candidate module and dependency justification for review; adding the dependency is a later implementation action.

## A current SQLite release requirement

SQLite documents a rare WAL-reset corruption bug affecting concurrent writes/checkpoints on multiple connections. It is fixed in SQLite **3.51.3 and later**, with backports **3.44.6** and **3.50.7**. This matters because Bearing will have multiple client processes. [SQLite WAL bug details](https://sqlite.org/wal.html), [3.51.3 release notes](https://sqlite.org/releaselog/3_51_3.html)

Require a bundled engine containing that fix before adopting WAL. Verify `sqlite_version()` from the actual built binary; a Go module version or the system `sqlite3` version is insufficient. This is a compatibility requirement, not a claim that the bug was reproduced here.

## Journal mode and the meaning of “saved”

**Proposal:** start with WAL and `synchronous=FULL` on supported local filesystems. Verify the actual settings after opening the database.

In WAL mode, FULL synchronizes each transaction commit. NORMAL preserves consistency but may lose a recently committed transaction after power loss or an operating-system crash. Process termination is a different failure case. Durability also depends on the filesystem and storage honoring synchronization. [SQLite synchronization settings](https://sqlite.org/pragma.html#pragma_synchronous)

WAL uses additional files and shared-memory coordination, requires participating processes on the same host, and has reader/checkpoint tradeoffs. A long-lived read can delay checkpoint progress. [SQLite WAL overview](https://sqlite.org/wal.html)

Therefore:

- Promise acknowledged writes survive a Bearing process restart after the tests prove that behavior.
- Describe power-loss durability as the configured SQLite guarantee under its storage assumptions.
- Support local filesystem storage initially; do not claim live network-drive or cloud-folder synchronization.
- Keep read transactions short and close query rows promptly.
- Do not force a full WAL checkpoint on every Bearing task checkpoint. They are different operations.

If measurements show ordinary rollback journaling fits the workload better, compare it explicitly. Do not change synchronization settings merely to improve a benchmark.

## Connection setup

Go's `sql.DB` is a pool. Limiting its open connections can cause deadlock if code holds a transaction and then waits for another connection from the same pool. [Go connection management](https://go.dev/doc/database/manage-connections)

**Proposal:** begin with one open connection per Bearing process. This simplifies connection-local settings and transaction reasoning; it does not eliminate interprocess contention. Revisit only if measured reads require it.

Set connection-local behavior whenever a physical connection is created or recreated, using the chosen driver's supported mechanism:

| Setting | Intended purpose |
| --- | --- |
| Foreign keys enabled | Prevent broken record relationships |
| Bounded busy timeout | Wait briefly for ordinary writer contention |
| Appropriate synchronization | Match the durability promise |
| Context deadline/cancellation | Bound the whole operation, including retries |

Foreign-key enforcement must be explicitly enabled for each connection and cannot be changed effectively inside a transaction. [SQLite foreign keys](https://sqlite.org/foreignkeys.html)

Keep all work belonging to a transaction on its `sql.Tx`. Do not mix database-handle calls into it, and do not manually issue transaction-control SQL through a pooled handle. [Go transactions](https://go.dev/doc/database/execute-transactions)

If an immediate write transaction is needed, verify the driver's transaction-mode support. Use its supported mechanism on the correct connection; do not create an imitation transaction helper because SQLite's SQL syntax looks simple.

## Atomic operations and lost responses

Three domain records are enough: memories, tasks, and checkpoints. Supporting metadata and request receipts exist only to implement actual consistency requirements.

| Operation | Atomic change |
| --- | --- |
| Save a checkpoint | Insert checkpoint; update task's latest checkpoint and version; record request result |
| Supersede memory | Insert replacement; mark previous record superseded; maintain retrieval index; record result |
| Finish task | Record final checkpoint and verification evidence; change status; record result |

Use optimistic concurrency: an update supplies the version it read. A different current version produces a conflict, not silent overwriting. Return the current ID/version so the caller can reload and reconcile.

Use a durable request key for mutations. Store its operation, normalized payload digest, and result in the same transaction as the change:

1. Same key and same payload returns the original result.
2. Same key and different payload returns an error.
3. A new key is a new operation.

The application request key is different from an MCP JSON-RPC message ID. A connection can disappear after commit but before delivery. Cancellation also does not prove the write failed. Retrying with the same application key resolves that uncertainty without duplicates.

Do not retry validation errors or version conflicts automatically. Retry recognized lock contention within a bounded deadline, repeating the entire transaction if its snapshot is no longer valid. Keep external calls, model requests, and source inspection outside write transactions.

## Minimal constraints worth enforcing

Use database constraints for facts the database can know: unique request keys, valid references, nonempty required fields, and consistent replacement links. Use explicit application validation for scope and allowed transitions.

The database cannot know whether a test really passed or whether a Gortex search was adequate. Store attributed evidence rather than a trusted-looking boolean called “senior reviewed.”

Keep checkpoints append-only in ordinary use. An agent can write a corrected checkpoint and advance the task pointer; it should not rewrite the history another client used to continue.

## Location and identity

Keep `.bearing/state.db` and its journal files out of Git. The ignore rule must cover the whole runtime state directory, including exports if placed there.

Resolve the repository root using Git, then bind the MCP process to that explicit path. Do not accept arbitrary repository overrides on individual tool calls. Avoid deriving identity solely from the remote URL: two unrelated local states can have the same remote.

Linked Git worktrees have their own working directories and per-worktree state while sharing other repository data. [Git worktree documentation](https://git-scm.com/docs/git-worktree)

The proposed per-checkout database intentionally does not solve worktree sharing. Explain that restriction. A later shared store would need an explicit repository identity, checkout identity, and task applicability model; moving a file into `.git` does not solve those problems.

## Backup, restore, and migrations

Use a SQLite-consistent snapshot mechanism for physical backups. The SQLite online backup API produces a consistent snapshot even while the source is being used; implementation must handle progress and interruption. [SQLite backup API](https://sqlite.org/backup.html)

Do not copy only the main database file while WAL writes are active. Choose the driver's supported backup path or a carefully tested SQLite-native alternative.

Keep these user concepts separate:

- **Backup:** a consistent recoverable database snapshot.
- **Export:** versioned logical records for inspection or transfer.
- **Restore:** validation followed by replacement of state in an explicitly selected destination.

First restore into a fresh temporary destination and validate references, schema version, counts, and integrity. Replacement of an existing live database requires exclusive operational control and explicit user intent; it is not a normal agent memory tool.

Maintain a schema version. Apply supported migrations transactionally and reject a newer schema without modifying it. Concurrent startup must not run the same migration twice. An old binary must not “repair” a new database into its preferred format.

## Required proof before release

Test real separate processes, not only goroutines:

- Two writers conflict without lost updates.
- A write acknowledged before forced process termination remains readable.
- Termination before commit leaves no partial task/checkpoint pair.
- A retry after a lost response returns the original IDs.
- Disk-full, read-only, busy, and cancellation errors never report success.
- Backup/restore preserves history and correction relationships.
- An older binary rejects a newer schema without altering it.

These tests establish bounded claims. A process-kill test is not a simulated power failure; a passing integrity check is not proof that all expected records exist.
