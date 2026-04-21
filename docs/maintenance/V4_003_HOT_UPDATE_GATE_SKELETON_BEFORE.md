# V4-003 Hot-Update Gate Skeleton Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `42ba01d5735d501c00686d55d5aaadc6db5fa48e`

## Tags At HEAD

- `frank-v4-004-pack-identity-read-model`

## Ahead/Behind Upstream

- `400 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_gate_registry_test.go`
- `docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_BEFORE.md`
- `docs/maintenance/V4_003_HOT_UPDATE_GATE_SKELETON_AFTER.md`

## Exact Record / Storage Shapes Planned

- `HotUpdateGateRef`
- `CandidateRuntimePackPointer`
- `HotUpdateGateRecord`
- bounded hot-update state validation using spec-aligned envelope states
- bounded hot-update decision validation using spec-aligned decision values
- durable helpers for:
  - gate path / directory resolution
  - candidate pointer path
  - store / load gate record
  - list gate records
  - store / load candidate pointer
  - resolve candidate pack record

## Exact Non-Goals

- no apply or reload behavior
- no promotion workflow
- no rollback workflow
- no improvement workspace execution
- no autonomy changes
- no provider or channel behavior changes
- no operator read-model exposure in this slice
- no mutation of active-pack / last-known-good registry contracts
- no cleanup outside this slice
