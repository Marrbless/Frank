# V4-003 Hot-Update Gate Skeleton After

## git diff --stat

```text

```

`git diff --stat` is empty because this slice currently consists of new untracked files rather than modifications to tracked files.

## git diff --numstat

```text

```

`git diff --numstat` is empty for the same reason: all implementation files added in this slice are new and untracked.

## Files Changed

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_BEFORE.md`
- `docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_AFTER.md`

## Exact Records / Helpers Added

Records and bounded enums:

- `HotUpdateReloadMode`
- `HotUpdateGateState`
- `HotUpdateGateDecision`
- `HotUpdateGateRef`
- `CandidateRuntimePackPointer`
- `HotUpdateGateRecord`

Not-found sentinels:

- `ErrHotUpdateGateRecordNotFound`
- `ErrCandidateRuntimePackPointerNotFound`

Storage and validation helpers:

- `StoreHotUpdateGatesDir`
- `StoreHotUpdateGatePath`
- `StoreCandidateRuntimePackPointerPath`
- `NormalizeHotUpdateGateRef`
- `NormalizeCandidateRuntimePackPointer`
- `NormalizeHotUpdateGateRecord`
- `ValidateHotUpdateGateRef`
- `ValidateCandidateRuntimePackPointer`
- `ValidateHotUpdateGateRecord`
- `StoreHotUpdateGateRecord`
- `LoadHotUpdateGateRecord`
- `ListHotUpdateGateRecords`
- `StoreCandidateRuntimePackPointer`
- `LoadCandidateRuntimePackPointer`
- `ResolveCandidateRuntimePackRecord`

Private/local helpers:

- `loadHotUpdateGateRecordFile`
- `normalizeHotUpdateStrings`
- `validateHotUpdateIdentifierField`
- `isValidHotUpdateReloadMode`
- `isValidHotUpdateGateState`
- `isValidHotUpdateGateDecision`

## Exact Tests Added

- `TestHotUpdateGateRecordRoundTripAndList`
- `TestCandidateRuntimePackPointerRoundTripAndResolve`
- `TestHotUpdateGateReplayIsIdempotent`
- `TestHotUpdateGateValidationFailsClosed`
- `TestHotUpdateGateRejectsMissingRefs`
- `TestLoadHotUpdateGateAndCandidatePointerNotFound`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - final:

```text
## frank-v4-001-pack-registry-foundation
?? docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_AFTER.md
?? docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_BEFORE.md
?? internal/missioncontrol/hot_update_gate_registry.go
?? internal/missioncontrol/hot_update_gate_registry_test.go
```

## Deferred Next V4 Candidates

- hot-update gate read-model / inspect exposure for staged candidate, state, decision, and reload mode
- hot-update policy record skeleton linked from the gate envelope
- append-only hot-update record / ledger skeleton separate from the gate envelope
- promotion and rollback durable records, still without apply/reload behavior
