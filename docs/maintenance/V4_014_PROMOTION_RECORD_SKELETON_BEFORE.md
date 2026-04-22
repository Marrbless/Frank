## V4-014 Promotion Record Skeleton Before

- branch: `frank-v4-014-promotion-record-skeleton`
- HEAD: `3d16c75173849e45ed786d17294c3f5df518c264`
- tags at HEAD: `frank-v4-013-hot-update-outcome-read-model`
- ahead/behind `upstream/main`: `412 0`
- `git status --short --branch`:
  - `## frank-v4-014-promotion-record-skeleton`
- baseline `go test -count=1 ./...` result:
  - `PASS`
  - package summary included `ok github.com/local/picobot/internal/missioncontrol`

### Exact Files Planned

- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/promotion_registry_test.go`
- `docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_BEFORE.md`
- `docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_AFTER.md`

### Exact Record / Storage Shapes Planned

- immutable per-record JSON storage under `runtime_packs/promotions/`
- file path shape: `runtime_packs/promotions/<promotion_id>.json`
- replay contract:
  - exact replay of the same normalized record is idempotent
  - divergent duplicate by the same `promotion_id` is rejected
- planned primary types:
  - `PromotionRef`
  - `PromotionRecord`
- planned helper surface:
  - `StorePromotionsDir`
  - `StorePromotionPath`
  - `NormalizePromotionRef`
  - `NormalizePromotionRecord`
  - `PromotionPromotedPackRef`
  - `PromotionPreviousActivePackRef`
  - `PromotionLastKnownGoodPackRef`
  - `PromotionHotUpdateGateRef`
  - `PromotionHotUpdateOutcomeRef`
  - `PromotionImprovementCandidateRef`
  - `PromotionImprovementRunRef`
  - `PromotionCandidateResultRef`
  - `ValidatePromotionRef`
  - `ValidatePromotionRecord`
  - `StorePromotionRecord`
  - `LoadPromotionRecord`
  - `ListPromotionRecords`
- planned `PromotionRecord` fields:
  - `record_version`
  - `promotion_id`
  - `promoted_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `last_known_good_basis`
  - `hot_update_id`
  - `outcome_id`
  - `candidate_id`
  - `run_id`
  - `candidate_result_id`
  - `reason`
  - `notes`
  - `promoted_at`
  - `created_at`
  - `created_by`

### Exact Non-Goals

- no apply or reload behavior
- no active-pointer mutation behavior
- no rollback workflow
- no promotion execution workflow
- no evaluator execution
- no scoring calculation behavior
- no autonomy changes
- no provider or channel behavior changes
- no dependency changes
- no inspect or read-model exposure unless forced by implementation, which is not planned for this slice
- no cleanup outside the bounded promotion-record storage slice
