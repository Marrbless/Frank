# V4-019 Rollback Apply Skeleton After

## git diff --stat

```text
```

Tracked `git diff --stat` output is empty because this slice currently consists of untracked new files.

## git diff --numstat

```text
```

Tracked `git diff --numstat` output is empty because this slice currently consists of untracked new files.

## Files Changed

- `docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_BEFORE.md`
- `docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`

## Exact Records / Helpers Added

- `RollbackApplyRef`
- `RollbackApplyPhase`
  - `recorded`
  - `validated`
  - `ready_to_apply`
- `RollbackApplyActivationState`
  - `unchanged`
- `RollbackApplyRecord`
- `ErrRollbackApplyRecordNotFound`
- `StoreRollbackAppliesDir`
- `StoreRollbackApplyPath`
- `NormalizeRollbackApplyRef`
- `NormalizeRollbackApplyRecord`
- `RollbackApplyRollbackRef`
- `ValidateRollbackApplyRef`
- `ValidateRollbackApplyRecord`
- `StoreRollbackApplyRecord`
- `CreateRollbackApplyRecordFromRollback`
- `LoadRollbackApplyRecord`
- `ListRollbackApplyRecords`
- `validateRollbackApplyLinkage`
- `validateRollbackApplyIdentifierField`
- immutable replay rule:
  - exact replay by `apply_id` is idempotent
  - divergent duplicate by the same `apply_id` is rejected
- explicit non-activation semantics:
  - every stored rollback-apply record must keep `activation_state=unchanged`
  - linkage authority stays with the committed rollback record referenced by `rollback_id`

## Exact Tests Added

- `TestCreateRollbackApplyRecordFromCommittedRollback`
- `TestRollbackApplyRecordRejectsMissingOrInvalidRollbackRefs`
- `TestRollbackApplyReplayIsIdempotentAndImmutable`
- `TestLoadRollbackApplyRecordNotFound`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation and report writing:

```text
## frank-v4-019-rollback-apply-skeleton
?? docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_AFTER.md
?? docs/maintenance/V4_019_ROLLBACK_APPLY_SKELETON_BEFORE.md
?? internal/missioncontrol/rollback_apply_registry.go
?? internal/missioncontrol/rollback_apply_registry_test.go
```

## Explicit No-Activation Statement

- No active runtime-pack pointer mutation was implemented.
- No rollback apply, reload, or runtime activation behavior was implemented.
- The new surface only records rollback-apply workflow intent/state linked to an existing committed rollback record.

## Deferred Next V4 Candidates

- non-mutating control-surface entry for rollback-apply creation if a later slice needs operator-triggered workflow records
- rollback-apply read-model exposure if later inspection needs more than the durable registry
- later execution orchestration that consumes rollback-apply and rollback records to mutate activation or reload behavior in a separate slice
