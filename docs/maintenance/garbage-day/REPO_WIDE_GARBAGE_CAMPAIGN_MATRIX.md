# Repo-Wide Garbage Campaign Matrix

Controller branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`

Source assessment: [REPO_WIDE_GARBAGE_CAMPAIGN_ASSESSMENT.md](./REPO_WIDE_GARBAGE_CAMPAIGN_ASSESSMENT.md)

This matrix tracks repo-wide cleanup candidates from the assessment. It is a maintenance controller, not product scope and not authorization for destructive actions.

## Status Counts

- DONE: 9
- PARTIAL: 0
- MISSING: 0
- BLOCKED: 0

## Matrix

| requirement_id | cleanup_target | current_evidence | status | gap_type | smallest_next_slice | suggested_tests | risk_if_skipped | can_implement_without_new_human_policy | last_slice_attempted | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GC5-000 | Checkpoint hygiene before new repo-wide treatment. | `GC5_TREAT_000_CHECKPOINT_HYGIENE_AFTER.md` records the checkpoint; this controller setup is committed before later treatment starts. | DONE | process | None. | `git status --short --branch`; `git diff --check`. | New cleanup work could mix with completed documentation slices and make review or rollback noisy. | yes | GC5-TREAT-000 | Controller setup is separated from later treatment work. |
| GC5-001 | Local ignored artifact inventory and prune decision. | `GC5_TREAT_001_LOCAL_ARTIFACT_INVENTORY_AFTER.md` records ignored local artifacts and confirms no deletion was performed. | DONE | housekeeping | None. | `git status --ignored`; `du -sh .codex internal/agent/memory internal/agent/sessions missions picobot`. | Operator-local state could be deleted accidentally, or confusing local noise could remain untracked forever. | yes | GC5-TREAT-001 | Deletion still requires explicit human approval naming exact paths. |
| GC5-002 | `internal/agent/loop_processdirect_test.go` is further split by command family. | `loop_processdirect_rollback_apply_test.go` now owns the contiguous `ROLLBACK_APPLY_*` process-direct command tests; the omnibus dropped to `9847` lines. | DONE | tests | None. | `/usr/local/go/bin/go test -count=1 ./internal/agent -run 'TestProcessDirectRollbackApply'`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; full suite. | Process-direct command review remains high-friction and broad diffs stay harder to audit. | yes | GC5-TREAT-002 | Test-only move; names, assertions, command strings, and shared fixtures preserved. |
| GC5-003 | `internal/agent/tools/taskstate_test.go` capability activation tests are split out. | `taskstate_capability_activation_test.go` now owns the contiguous capability activation tests; `taskstate_test.go` dropped to `6013` lines. | DONE | tests | None. | Focused capability activation tests; `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`; full suite. | Protected capability gates remain harder to review and future TaskState changes stay noisier. | yes | GC5-TREAT-003 | Test-only move; exact assertions and helper behavior preserved. |
| GC5-004 | `cmd/picobot/main_runtime_bootstrap_test.go` is split by another bootstrap subfamily. | `main_runtime_bootstrap_set_step_test.go` now owns set-step, control-file, watcher, and operator set-step tests; the bootstrap omnibus dropped to `4407` lines. | DONE | tests | None. | `/usr/local/go/bin/go test -count=1 ./cmd/picobot -run 'TestMissionSetStep|TestApplyMissionStepControl|TestWatchMissionStepControl'`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`; full suite. | CLI runtime bootstrap reviews stay unnecessarily broad and fragile. | yes | GC5-TREAT-004 | Test-only move; helper placement stayed stable. |
| GC5-005 | `internal/missioncontrol/status.go` read-model loaders are split further. | `status_rollback.go` now owns rollback and rollback-apply identity/status read-model loaders; `status.go` dropped to `3634` lines. | DONE | structure | None. | Focused rollback identity tests; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; full suite. | Operator-facing status JSON and read-model changes remain concentrated and harder to review. | yes | GC5-TREAT-005 | Same-package mechanical move only; no JSON, state, linkage, or loader behavior drift. |
| GC5-006 | Repeated phone capability source initializers are assessed before abstraction. | `GC5_TREAT_006_PHONE_CAPABILITY_INITIALIZER_ASSESSMENT.md` records the repeated initializer shape, per-capability differences, and helper boundary risks. | DONE | assessment | None. | Existing capability package tests only if code changes are later selected. | A premature helper could flatten meaningful per-capability policy differences. | yes | GC5-TREAT-006 | Assessment complete; no abstraction was introduced. |
| GC5-007 | Tool schema typing boundary is deferred until a real API decision exists. | `GC5_TREAT_007_TOOL_SCHEMA_TYPING_BOUNDARY_DECISION.md` records the generic JSON boundary across tools, providers, and MCP, and defers broad typing. | DONE | api-design | None. | Tool registry tests; provider tests; full suite if any API code changes. | Broad cleanup could destabilize tool/provider APIs for little immediate gain. | yes | GC5-TREAT-007 | Design/controller row complete; no API churn was introduced. |
| GC5-008 | Direct-write audit for runtime-adjacent non-test files. | `GC5_TREAT_008_DIRECT_WRITE_AUDIT.md` classifies all non-test direct writes under `cmd` and `internal`; no direct replacement is authorized. | DONE | audit | None. | Targeted tests per audited area; full suite if write paths change. | Blind replacement can alter permissions, bootstrap behavior, or fixture expectations. | yes | GC5-TREAT-008 | Mission-store records already use atomic JSON writes; audited direct writes are config/bootstrap/local seed outputs. |

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
