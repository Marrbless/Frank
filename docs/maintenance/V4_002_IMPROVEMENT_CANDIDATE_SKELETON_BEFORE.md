# V4-002 Improvement Candidate Skeleton Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `842869287369b08414cd1cbbc4a278347ea8c590`

## Tags At HEAD

- `frank-v4-003-hot-update-gate-skeleton`

## Ahead/Behind Upstream

- `401 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/improvement_candidate_registry.go`
- `internal/missioncontrol/improvement_candidate_registry_test.go`
- `docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_BEFORE.md`
- `docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_AFTER.md`

## Exact Record / Storage Shapes Planned

### `ImprovementCandidateRef`

- same-package durable ref wrapper with canonical `candidate_id`

### `ImprovementCandidateRecord`

- bounded improvement-workspace candidate record with:
  - `record_version`
  - `candidate_id`
  - `baseline_pack_id`
  - `candidate_pack_id`
  - `source_workspace_ref`
  - `source_summary`
  - `validation_basis_refs`
  - `hot_update_id`
  - `created_at`
  - `created_by`

### Planned Storage / Validation Helpers

- candidate directory and path helpers under the existing V4 `runtime_packs` mission-store surface
- normalize / validate helpers for candidate ref and candidate record
- store / load / list helpers for candidate records
- linkage validation against:
  - `RuntimePackRecord` for `baseline_pack_id` and `candidate_pack_id`
  - `HotUpdateGateRecord` for optional `hot_update_id`
- lineage validation where applicable:
  - if the candidate runtime pack declares `parent_pack_id`, it must match `baseline_pack_id`
  - if a linked hot-update gate exists, its `candidate_pack_id` must match the candidate runtime pack and its `previous_active_pack_id` must match `baseline_pack_id`

## Exact Non-Goals

- no improvement execution
- no evaluator or scoring framework
- no apply or reload behavior
- no promotion workflow
- no rollback workflow
- no autonomy changes
- no provider or channel behavior changes
- no operator read-model exposure in this slice
- no mutation of active-pack / last-known-good / hot-update registry contracts outside candidate linkage checks
- no cleanup outside this slice
