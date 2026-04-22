# V4-022 Rollback-Apply Phase Progression After

## `git diff --stat`

```text
 internal/missioncontrol/rollback_apply_registry.go |  94 +++++++++
 .../missioncontrol/rollback_apply_registry_test.go | 223 +++++++++++++++++++++
 2 files changed, 317 insertions(+)
```

Note: tracked `git diff --stat` output does not include new untracked files created in this slice.

## `git diff --numstat`

```text
94	0	internal/missioncontrol/rollback_apply_registry.go
223	0	internal/missioncontrol/rollback_apply_registry_test.go
```

Note: tracked `git diff --numstat` output does not include new untracked files created in this slice.

## Files Changed

- `docs/maintenance/V4_022_ROLLBACK_APPLY_PHASE_PROGRESSION_BEFORE.md`
- `docs/maintenance/V4_022_ROLLBACK_APPLY_PHASE_PROGRESSION_AFTER.md`
- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`

## Exact Helpers / Records Updated

- `RollbackApplyRecord`
  - added `phase_updated_at`
  - added `phase_updated_by`
- `NormalizeRollbackApplyRecord`
  - now backfills missing phase-transition metadata from create metadata for legacy records
- `ValidateRollbackApplyRecord`
  - now validates required phase-transition metadata and ordering against `created_at`
- `CreateRollbackApplyRecordFromRollback`
  - now seeds `phase_updated_at` / `phase_updated_by` on initial `recorded` creation
- `AdvanceRollbackApplyPhase(root, applyID, nextPhase, updatedBy, updatedAt)`
  - new durable non-executing phase progression helper
  - supports adjacent progression only:
    - `recorded -> validated`
    - `validated -> ready_to_apply`
  - same-phase replay is idempotent
  - skipped and regressive transitions fail closed
  - `activation_state` remains unchanged
- `rollbackApplyPhaseOrder`
  - new bounded phase ordering helper

## Exact Tests Added

- `TestLoadRollbackApplyRecordBackfillsLegacyPhaseTransitionMetadata`
- `TestAdvanceRollbackApplyPhaseValidProgressionAndPreservesActiveRuntimePackPointer`
- `TestAdvanceRollbackApplyPhaseRejectsInvalidTransition`
- `TestAdvanceRollbackApplyPhaseIsIdempotentForSamePhase`

## Existing Tests Updated

- `TestCreateRollbackApplyRecordFromCommittedRollback`
- `TestRollbackApplyRecordRejectsMissingOrInvalidRollbackRefs`
- `TestRollbackApplyReplayIsIdempotentAndImmutable`
  - updated to assert the new phase-transition metadata contract

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/rollback_apply_registry.go internal/missioncontrol/rollback_apply_registry_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation and report writing:

```text
## frank-v4-022-rollback-apply-phase-progression
 M internal/missioncontrol/rollback_apply_registry.go
 M internal/missioncontrol/rollback_apply_registry_test.go
?? docs/maintenance/V4_022_ROLLBACK_APPLY_PHASE_PROGRESSION_AFTER.md
?? docs/maintenance/V4_022_ROLLBACK_APPLY_PHASE_PROGRESSION_BEFORE.md
```

## Explicit No-Execution Statement

- No activation mutation was implemented.
- No rollback apply behavior or reload behavior was implemented.
- No active runtime-pack pointer mutation, promotion behavior changes, evaluator execution, scoring behavior, or autonomy changes were implemented.
- This slice only adds durable non-executing phase progression metadata and validation over existing rollback-apply records.

## Deferred Next V4 Candidates

- optional non-mutating control-surface extension for advancing rollback-apply phases if a later slice needs operator entry instead of direct helper use
- execution orchestration that consumes `ready_to_apply` rollback-apply records in a separate slice
- any read-model expansion for phase-transition metadata only if later operator surfaces need it explicitly
