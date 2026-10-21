# Changelog

Binary versions follow semver. `policy_id` and `workflow_id` are SHA-256 hashes of the embedded ENGINEERING.md and WORKFLOW.md; `bearing version` prints all three.

## 0.1.0 · unreleased

| Item | Value |
| --- | --- |
| Version | 0.1.0 |
| Schema version | 2 |
| `policy_id` | `3445b7e150a4a234752997aff50f8c2608907621168f6b417c0196dcd2e41a20` |
| `workflow_id` | `17113f2b8f9fade3c9bfeb690c0d64708e0a74eafaf9b0abed6dc308231c791d` |
| Installed skill text | Changed: the skill carries the workflow only. Rerun `bearing install` |

### Breaking

- `finish` no longer accepts `acceptance_results` or `review.policy_id`. A criterion passes when a passing verification lists it in `acceptance_ids`; Bearing records the policy ID itself.

### Added

- Start, checkpoint, and finish record a working-tree hash. Finish rejects passing evidence reused from a different tree and lists test files changed since the task started.
- Command verification requires `command` and `exit_code`.
- Known-answer recall fixtures report hit@3 (11 of 20 at this version).
- Automatic store backup to `.bearing/backups/` before a schema migration.
- The install manifest records the binary version, `policy_id`, `workflow_id`, and the SHA-256 of every written file; reinstalling reports files the user edited.
- CI runs `go vet`, `go test`, and govulncheck with the Go version in go.mod.

### Changed

- Server instructions tell agents to call `bearing_resume` or `bearing_start` first and follow the policy it returns. The policy is no longer copied into the skill.
- Finish errors name the failing criterion, evidence ID, or review check.
- Start, remember, checkpoint, and finish declare `destructiveHint: false`.
- Empty recall results warn and suggest anchors or explicit IDs.
- `bearing version` reads the module version from build info, then an `-ldflags` value, then `0.1.0-dev`.
- The newer-schema refusal names both schema versions.
