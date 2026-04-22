## V4-016 Rollback Record Skeleton Before

- branch: `frank-v4-016-rollback-record-skeleton`
- HEAD: `c70fbf4d8856846f03fec37713ed7bf2aa79469f`
- tags at HEAD: `frank-v4-015-promotion-read-model`
- ahead/behind `upstream/main`: `414 0`
- `git status --short --branch`:
  - `## frank-v4-016-rollback-record-skeleton`
- baseline `go test -count=1 ./...` result:
  - `PASS`
  - package summary included `ok github.com/local/picobot/internal/missioncontrol`

### Exact Files Planned

- `internal/missioncontrol/rollback_registry.go`
- `internal/missioncontrol/rollback_registry_test.go`
- `docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_BEFORE.md`
- `docs/maintenance/V4_016_ROLLBACK_RECORD_SKELETON_AFTER.md`

### Exact Record / Storage Shapes Planned

- immutable per-record JSON storage under `runtime_packs/rollbacks/`
- file path shape: `runtime_packs/rollbacks/<rollback_id>.json`
- replay contract:
  - exact replay of the same normalized record is idempotent
  - divergent duplicate by the same `rollback_id` is rejected
- planned primary types:
  - `RollbackRef`
  - `RollbackRecord`
- planned helper surface:
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
- planned `RollbackRecord` fields:
  - `record_version`
  - `rollback_id`
  - `promotion_id`
  - `hot_update_id`
  - `outcome_id`
  - `from_pack_id`
  - `target_pack_id`
  - `last_known_good_pack_id`
  - `reason`
  - `notes`
  - `rollback_at`
  - `created_at`
  - `created_by`
- planned linkage rules:
  - `from_pack_id` and `target_pack_id` must resolve to committed runtime packs
  - optional `last_known_good_pack_id` must resolve if present
  - optional `promotion_id` must resolve and stay consistent with `from_pack_id`, `hot_update_id`, and optional `last_known_good_pack_id`
  - optional `hot_update_id` must resolve and stay consistent with `from_pack_id` and `target_pack_id`
  - optional `outcome_id` must resolve and stay consistent with `hot_update_id` and `from_pack_id`

### Exact Non-Goals

- no apply or reload behavior
- no rollback execution behavior
- no promotion behavior changes
- no evaluator execution
- no scoring behavior
- no autonomy changes
- no provider or channel behavior changes
- no dependency changes
- no inspect or read-model exposure in this slice
- no cleanup outside the bounded rollback-record storage slice
