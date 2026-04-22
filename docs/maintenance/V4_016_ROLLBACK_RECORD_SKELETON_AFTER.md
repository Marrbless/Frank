## V4-016 Rollback Record Skeleton After

### git diff --stat

```text
```

Tracked diff output is empty because this slice currently consists of untracked new files.

### git diff --numstat

```text
```

Tracked diff output is empty because this slice currently consists of untracked new files.

### Files Changed

- `docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_BEFORE.md`
- `docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_AFTER.md`
- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_registry_test.go`

### Exact Records / Helpers Added

- `RollbackRef`
- `RollbackRecord`
- `ErrRollbackRecordNotFound`
- `StoreRollbacksDir`
- `StoreRollbackPath`
- `NormalizeRollbackRef`
- `NormalizeRollbackRecord`
- `RollbackPromotionRef`
- `RollbackHotUpdateGateRef`
- `RollbackHotUpdateOutcomeRef`
- `RollbackFromPackRef`
- `RollbackTargetPackRef`
- `RollbackLastKnownGoodPackRef`
- `ValidateRollbackRef`
- `ValidateRollbackRecord`
- `StoreRollbackRecord`
- `LoadRollbackRecord`
- `ListRollbackRecords`
- immutable replay rule:
  - exact replay by `rollback_id` is idempotent
  - divergent duplicate by the same `rollback_id` is rejected
- linkage validation added for:
  - source and target runtime packs
  - optional last-known-good runtime pack
  - optional promotion record
  - optional hot-update gate
  - optional hot-update outcome

### Exact Tests Added

- `TestRollbackRecordRoundTripAndList`
- `TestRollbackReplayIsIdempotentAndImmutable`
- `TestRollbackValidationFailsClosed`
- `TestRollbackRejectsMissingAndMismatchedLinkedRefs`
- `TestLoadRollbackRecordNotFound`
- helper fixtures:
  - `validRollbackRecord`
  - `storeRollbackFixtures`

### Validation Commands And Results

- `gofmt -w internal/missioncontrol/rollback_registry.go internal/missioncontrol/rollback_registry_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation:

```text
## frank-v4-016-rollback-record-skeleton
?? docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_AFTER.md
?? docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_BEFORE.md
?? internal/missioncontrol/rollback_registry.go
?? internal/missioncontrol/rollback_registry_test.go
```

### Deferred Next V4 Candidates

- rollback read-model / inspect exposure
- rollback control-surface work that consumes the durable records without implementing apply behavior in this slice
- later promotion or rollback orchestration that reuses the immutable storage contract introduced here
