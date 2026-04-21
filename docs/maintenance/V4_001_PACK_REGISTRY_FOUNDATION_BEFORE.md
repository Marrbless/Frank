# V4-001 Pack Registry Foundation Before

## Branch

- `frank-v4-001-pack-registry-foundation`

## HEAD

- `1cf96941dc5fa527fc6a9a567f18dc6f4daba063`

## Tags At HEAD

- none

## Ahead/Behind Upstream

- `398 0`

## git status --short --branch

```text
## frank-v4-001-pack-registry-foundation
```

## Baseline `go test -count=1 ./...` Result

- passed

## Exact Files Planned

- `internal/missioncontrol/runtime_pack_registry.go`
- `internal/missioncontrol/runtime_pack_registry_test.go`
- `docs/maintenance/V4_001_PACK_REGISTRY_FOUNDATION_BEFORE.md`
- `docs/maintenance/V4_001_PACK_REGISTRY_FOUNDATION_AFTER.md`

## Exact Record/Pointer Shapes Planned

### `RuntimePackRef`

- same-package durable ref wrapper with canonical `pack_id`

### `RuntimePackRecord`

- bounded V4 runtime-pack identity record with:
  - `record_version`
  - `pack_id`
  - `parent_pack_id`
  - `created_at`
  - `channel`
  - `prompt_pack_ref`
  - `skill_pack_ref`
  - `manifest_ref`
  - `extension_pack_ref`
  - `policy_ref`
  - `source_summary`
  - `mutable_surfaces`
  - `immutable_surfaces`
  - `surface_classes`
  - `compatibility_contract_ref`
  - `rollback_target_pack_id`

### `ActiveRuntimePackPointer`

- durable active-pack pointer with:
  - `record_version`
  - `active_pack_id`
  - `previous_active_pack_id`
  - `last_known_good_pack_id`
  - `updated_at`
  - `updated_by`
  - `update_record_ref`
  - `reload_generation`

### `LastKnownGoodRuntimePackPointer`

- durable last-known-good pointer with:
  - `record_version`
  - `pack_id`
  - `basis`
  - `verified_at`
  - `verified_by`
  - `rollback_record_ref`

## Exact Non-Goals

- no improvement workspace execution
- no evaluator or scoring framework
- no hot-update apply/reload workflow
- no promotion workflow
- no rollback workflow or rollback state machine
- no background autonomy changes
- no provider/channel behavior changes
- no cleanup outside this bounded registry/pointer slice
