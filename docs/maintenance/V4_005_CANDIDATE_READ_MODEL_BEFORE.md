# V4-005 Candidate Read-Model Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `807d14322c00990a7718e1374438298f4ac583dd`

## Tags At HEAD

- `frank-v4-002-improvement-candidate-skeleton`

## Ahead/Behind Upstream

- `402 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_improvement_candidate_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_005_CANDIDATE_READ_MODEL_AFTER.md`

## Exact Read-Model Surface Chosen

- Primary surface: `missioncontrol.OperatorStatusSummary`
- Persisted/read-only status adapter path:
  - `internal/agent/tools` operator status readout post-processing
- Store-backed committed snapshot path:
  - `missioncontrol.BuildCommittedMissionStatusSnapshot`

Chosen shape:

- add one read-only `improvement_candidate_identity` block to `OperatorStatusSummary`
- expose zero-or-more committed candidate records deterministically because V4-002 introduced candidate records but no single active-candidate pointer

## Exact Output Fields Planned

- `improvement_candidate_identity.state`
- `improvement_candidate_identity.candidates[]`
- per candidate:
  - `state`
  - `candidate_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `source_workspace_ref`
  - `source_summary`
  - `validation_basis_refs`
  - `hot_update_id`
  - `created_at`
  - `created_by`
  - `error`

Planned state values:

- `configured`
- `not_configured`
- `invalid`

## Exact Non-Goals

- no improvement execution
- no evaluator framework
- no hot-update apply or reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no new inspect step/control surface beyond the chosen status read-model hook
- no mutation of runtime-pack, hot-update, or candidate storage contracts
- no cleanup outside this slice
