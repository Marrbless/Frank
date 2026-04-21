# V4-004 Pack Identity Read-Model Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `c9623c2155472337975a4392510175c9717d4df0`

## Tags At HEAD

- `frank-v4-001-pack-registry-foundation`

## Ahead/Behind Upstream

- `399 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_runtime_pack_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Chosen

- Primary surface: `missioncontrol.OperatorStatusSummary`
- Store-backed committed status path:
  - `missioncontrol.BuildCommittedMissionStatusSnapshot`
- Store-backed operator status readout path:
  - `internal/agent/tools` status readout post-processing for mission-store-backed operator status JSON

## Exact Non-Goals

- no hot-update apply or reload behavior
- no promotion workflow
- no rollback workflow
- no improvement workspace execution
- no autonomy changes
- no provider/channel behavior changes
- no inspect-step/runtime-control surface expansion beyond the chosen status read-model
- no cleanup outside this slice
