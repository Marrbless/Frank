## V4-014 Promotion Record Skeleton After

### git diff --stat

```text
```

Tracked diff output is empty because this slice currently consists of untracked new files.

### git diff --numstat

```text
```

Tracked diff output is empty because this slice currently consists of untracked new files.

### Files Changed

- `docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_BEFORE.md`
- `docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_AFTER.md`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/promotion_registry_test.go`

### Exact Records / Helpers Added

- `PromotionRef`
- `PromotionRecord`
- `ErrPromotionRecordNotFound`
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
- immutable replay rule:
  - exact replay by `promotion_id` is idempotent
  - divergent duplicate by the same `promotion_id` is rejected
- linkage validation added for:
  - promoted runtime pack
  - previous active runtime pack
  - optional last-known-good runtime pack and basis
  - hot-update gate
  - optional hot-update outcome
  - optional improvement candidate
  - optional improvement run
  - optional candidate result

### Exact Tests Added

- `TestPromotionRecordRoundTripAndList`
- `TestPromotionReplayIsIdempotentAndImmutable`
- `TestPromotionValidationFailsClosed`
- `TestPromotionRejectsMissingAndMismatchedLinkedRefs`
- `TestLoadPromotionRecordNotFound`
- helper fixtures:
  - `validPromotionRecord`
  - `storePromotionFixtures`

### Validation Commands And Results

- `gofmt -w internal/missioncontrol/promotion_registry.go internal/missioncontrol/promotion_registry_test.go`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation:

```text
## frank-v4-014-promotion-record-skeleton
?? docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_AFTER.md
?? docs/maintenance/V4_014_PROMOTION_RECORD_SKELETON_BEFORE.md
?? internal/missioncontrol/promotion_registry.go
?? internal/missioncontrol/promotion_registry_test.go
```

### Deferred Next V4 Candidates

- promotion read-model / inspect exposure
- rollback record skeleton
- promotion or rollback control surfaces that consume the durable records without changing this slice’s storage contract
