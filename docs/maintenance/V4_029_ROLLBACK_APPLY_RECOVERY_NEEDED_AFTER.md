# V4-029 Rollback Apply Recovery-Needed After

## git diff --stat

```text
 internal/missioncontrol/rollback_apply_registry.go | 138 ++++++---
 .../missioncontrol/rollback_apply_registry_test.go | 307 +++++++++++++++++++++
 2 files changed, 412 insertions(+), 33 deletions(-)
```

## git diff --numstat

```text
105	33	internal/missioncontrol/rollback_apply_registry.go
307	0	internal/missioncontrol/rollback_apply_registry_test.go
```

## Files changed

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_BEFORE.md`
- `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_AFTER.md`

## Exact phase/helper changes added

- Added rollback-apply durable phase `reload_apply_recovery_needed`.
- Added `ReconcileRollbackApplyRecoveryNeeded(root, applyID, updatedBy, updatedAt)` as the bounded normalization helper for persisted `reload_apply_in_progress`.
- Added shared execution-linkage validation so recovery normalization and reload/apply execution both validate the same committed rollback and active-pointer linkage.
- Preserved explicit recovery-needed semantics:
  - normalize `reload_apply_in_progress -> reload_apply_recovery_needed` when linkage is coherent
  - fail closed on missing rollback or invalid active-pointer linkage
  - treat exact replay on `reload_apply_recovery_needed` as idempotent

## Exact tests added

- `TestReconcileRollbackApplyRecoveryNeededNormalizesInProgressWithoutMutatingPointerState`
- `TestReconcileRollbackApplyRecoveryNeededRejectsInvalidLinkageWithoutPointerMutation`
- `TestReconcileRollbackApplyRecoveryNeededReplayIsIdempotent`
- `storeRollbackApplyReloadInProgressFixture`

## Validation commands and results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go` : passed
- `git diff --check` : passed
- `go test -count=1 ./internal/missioncontrol` : passed
- `go test -count=1 ./internal/agent` : passed
- `go test -count=1 ./internal/agent/tools` : passed
- `go test -count=1 ./...` : passed
- `git status --short --branch` : `## frank-v4-029-rollback-apply-recovery-needed`, modified `internal/missioncontrol/rollback_apply_registry.go`, modified `internal/missioncontrol/rollback_apply_registry_test.go`, untracked `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_AFTER.md`, untracked `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_BEFORE.md`

## Explicit non-implementation statements

- No pointer mutation was implemented in this slice.
- No reload/apply retry was implemented in this slice.
- No `last_known_good_pointer.json` mutation was implemented in this slice.

## Deferred next V4 candidates

- add a tiny operator/control or inspect hook for explicit recovery normalization if later checkpoints require it
- define explicit operator-driven retry or terminal-failure handling from `reload_apply_recovery_needed`
- expose recovery-needed or execution-error details through read-only status only if a later slice explicitly needs that visibility
