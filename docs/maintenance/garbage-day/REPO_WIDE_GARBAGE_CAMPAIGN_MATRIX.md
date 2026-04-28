# Repo-Wide Garbage Campaign Matrix

Controller branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

Source assessment: [REPO_WIDE_GARBAGE_CAMPAIGN_ASSESSMENT.md](./REPO_WIDE_GARBAGE_CAMPAIGN_ASSESSMENT.md)

This matrix tracks repo-wide cleanup candidates from the assessment. It is a maintenance controller, not product scope and not authorization for destructive actions.

## Status Counts

- DONE: 1
- PARTIAL: 0
- MISSING: 8
- BLOCKED: 0

## Matrix

| requirement_id | cleanup_target | current_evidence | status | gap_type | smallest_next_slice | suggested_tests | risk_if_skipped | can_implement_without_new_human_policy | last_slice_attempted | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GC5-000 | Checkpoint hygiene before new repo-wide treatment. | `GC5_TREAT_000_CHECKPOINT_HYGIENE_AFTER.md` records the checkpoint; this controller setup is committed before later treatment starts. | DONE | process | None. | `git status --short --branch`; `git diff --check`. | New cleanup work could mix with completed documentation slices and make review or rollback noisy. | yes | GC5-TREAT-000 | Controller setup is separated from later treatment work. |
| GC5-001 | Local ignored artifact inventory and prune decision. | Ignored local artifacts include `picobot` (`33M`), `internal/agent/memory/*.md` (`38` files, `336K`), `internal/agent/sessions/`, `missions/`, and `.codex`. | MISSING | housekeeping | Add a no-delete inventory note, or request explicit approval for exact local deletions. | `git status --ignored`; `du -sh .codex internal/agent/memory internal/agent/sessions missions picobot`. | Operator-local state could be deleted accidentally, or confusing local noise could remain untracked forever. | partial | none | Deletion requires explicit human approval. Inventory-only work does not. |
| GC5-002 | `internal/agent/loop_processdirect_test.go` is further split by command family. | File remains `10873` lines; hot-update and rollback command families dominate the middle of the file after the earlier inspect split. | MISSING | tests | Move one contiguous family, such as rollback apply command tests, into a same-package test file. | `/usr/local/go/bin/go test -count=1 ./internal/agent -run 'TestProcessDirectRollbackApply'`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; full suite. | Process-direct command review remains high-friction and broad diffs stay harder to audit. | yes | none | Test-only move; preserve names, assertions, command strings, and shared fixtures. |
| GC5-003 | `internal/agent/tools/taskstate_test.go` capability activation tests are split out. | `taskstate_test.go` remains `7307` lines; capability activation tests are contiguous near the end. | MISSING | tests | Move capability activation tests into a same-package file such as `taskstate_capability_exposure_test.go`. | Focused capability activation tests; `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`; full suite. | Protected capability gates remain harder to review and future TaskState changes stay noisier. | yes | none | Test-only move; exact assertions and helper behavior must be preserved. |
| GC5-004 | `cmd/picobot/main_runtime_bootstrap_test.go` is split by another bootstrap subfamily. | File remains `6510` lines after the store-root split; set-step, bootstrap, control-file, and durable runtime families coexist. | MISSING | tests | Move mission set-step/control-file tests into a same-package file. | `/usr/local/go/bin/go test -count=1 ./cmd/picobot -run 'TestMissionSetStep|TestApplyMissionStepControl|TestWatchMissionStepControl'`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`; full suite. | CLI runtime bootstrap reviews stay unnecessarily broad and fragile. | yes | none | Test-only move; helper placement must stay stable. |
| GC5-005 | `internal/missioncontrol/status.go` read-model loaders are split further. | `status.go` remains `3879` lines after the autonomy split; many V4 identity/status loaders are still clustered. | MISSING | structure | Move one cohesive identity/read-model family into a same-package file. | Focused status identity tests; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; full suite. | Operator-facing status JSON and read-model changes remain concentrated and harder to review. | yes | none | Same-package mechanical move only; no JSON, state, linkage, or loader behavior drift. |
| GC5-006 | Repeated phone capability source initializers are assessed before abstraction. | Similar source initializers exist across camera, contacts, location, microphone, SMS phone, Bluetooth/NFC, and broad app control. | MISSING | assessment | Write a read-only capability initializer duplication assessment with invariants and no code changes. | Existing capability package tests only if code changes are later selected. | A premature helper could flatten meaningful per-capability policy differences. | yes | none | Assessment first; no abstraction is authorized by the matrix row alone. |
| GC5-007 | Tool schema typing boundary is deferred until a real API decision exists. | Tool schemas and args still use `map[string]interface{}` / `interface{}` across tools, provider payloads, MCP, and tests. | MISSING | api-design | Record a typed tool-schema boundary decision or explicitly defer broad string/map churn. | Tool registry tests; provider tests; full suite if any API code changes. | Broad cleanup could destabilize tool/provider APIs for little immediate gain. | yes | none | Treat as design/controller work first, not mechanical replacement. |
| GC5-008 | Direct-write audit for runtime-adjacent non-test files. | Non-test `os.WriteFile` remains in onboarding, phone capability source creation, and exec project setup. | MISSING | audit | Audit each direct write as runtime state, config bootstrap, or safe local seed output before changing code. | Targeted tests per audited area; full suite if write paths change. | Blind replacement can alter permissions, bootstrap behavior, or fixture expectations. | yes | none | Audit first; no direct-write replacement is authorized yet. |

## Treatment Order

Recommended order:

1. Resolve `GC5-000`.
2. Pick either `GC5-003` or `GC5-004` for the first code-adjacent treatment because both are test-only, same-package slices.
3. Treat `GC5-001` separately from code work because deletion requires explicit approval.

## Guardrails

- Do not mix cleanup treatment with product behavior changes.
- Do not delete, archive, or move runtime/operator artifacts without explicit approval and a before/after note.
- Prefer test-only and same-package mechanical moves before behavior or API work.
- Preserve public method names, command names, JSON fields, record formats, and validation order unless a later task explicitly authorizes a behavior change.
- Every treatment slice must update this matrix and record validation evidence.
