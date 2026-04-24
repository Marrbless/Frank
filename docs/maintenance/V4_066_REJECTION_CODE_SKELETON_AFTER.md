# V4-066 Rejection Code Skeleton After

## V4 Rejection Constants Added

Frozen-spec constants:

- `E_EXECUTION_PLANE_REQUIRED`
- `E_EXECUTION_HOST_REQUIRED`
- `E_IMPROVEMENT_WORKSPACE_REQUIRED`
- `E_HOT_UPDATE_GATE_REQUIRED`
- `E_BASELINE_REQUIRED`
- `E_HOLDOUT_REQUIRED`
- `E_SMOKE_CHECK_REQUIRED`
- `E_EVAL_IMMUTABLE`
- `E_MUTATION_SCOPE_VIOLATION`
- `E_SURFACE_CLASS_REQUIRED`
- `E_FORBIDDEN_SURFACE_CHANGE`
- `E_TOPOLOGY_CHANGE_DISABLED`
- `E_PROMOTION_POLICY_REQUIRED`
- `E_HOT_UPDATE_POLICY_REQUIRED`
- `E_CANARY_REQUIRED`
- `E_PROMOTION_APPROVAL_REQUIRED`
- `E_HOT_UPDATE_APPROVAL_REQUIRED`
- `E_ACTIVE_JOB_DEPLOY_LOCK`
- `E_PACK_NOT_FOUND`
- `E_LAST_KNOWN_GOOD_REQUIRED`
- `E_CANARY_FAILED`
- `E_SMOKE_CHECK_FAILED`
- `E_ROLLBACK_REQUIRED`
- `E_PROMOTION_ALREADY_APPLIED`
- `E_HOT_UPDATE_ALREADY_APPLIED`
- `E_RELOAD_MODE_UNSUPPORTED`
- `E_RELOAD_QUIESCE_FAILED`
- `E_EXTENSION_COMPATIBILITY_REQUIRED`
- `E_EXTENSION_PERMISSION_WIDENING`
- `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`
- `E_POLICY_MUTATION_FORBIDDEN`
- `E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN`
- `E_AUTONOMY_ENVELOPE_REQUIRED`
- `E_STANDING_DIRECTIVE_REQUIRED`
- `E_AUTONOMY_BUDGET_EXCEEDED`
- `E_NO_ELIGIBLE_AUTONOMOUS_ACTION`
- `E_AUTONOMY_PAUSED`
- `E_EXTERNAL_ACTION_LIMIT_REACHED`
- `E_REPEATED_FAILURE_PAUSE`
- `E_PACKAGE_AUTHORITY_GRANT_FORBIDDEN`

Slice-specific execution metadata constants:

- `E_LAB_ONLY_FAMILY`
- `E_MISSION_FAMILY_REQUIRED`
- `E_EXECUTION_PLANE_UNKNOWN`
- `E_EXECUTION_HOST_UNKNOWN`
- `E_MISSION_FAMILY_UNKNOWN`
- `E_EXECUTION_PLANE_INCOMPATIBLE`
- `E_LIVE_PHONE_SELF_EDIT_FORBIDDEN`

## V4-065 Validation Mappings Changed

- Missing `execution_plane`: `E_EXECUTION_PLANE_REQUIRED`
- Missing `execution_host`: `E_EXECUTION_HOST_REQUIRED`
- Missing `mission_family`: `E_MISSION_FAMILY_REQUIRED`
- Unknown `execution_plane`: `E_EXECUTION_PLANE_UNKNOWN`
- Unknown `execution_host`: `E_EXECUTION_HOST_UNKNOWN`
- Unknown `mission_family`: `E_MISSION_FAMILY_UNKNOWN`
- Improvement-family job outside `improvement_workspace`: `E_LAB_ONLY_FAMILY`
- Live-family or hot-update-family job on the wrong plane: `E_EXECUTION_PLANE_INCOMPATIBLE`

Human-readable messages were preserved.

## Compatibility Behavior

- V4 codes are only attached to the existing `spec_version=frank_v4` execution metadata validators.
- Pre-V4 jobs still do not require execution metadata.
- Existing internal validation constants remain present for backward compatibility.
- No new read-model fields or commands were added.

## Invariants Preserved

- No adaptive lab was implemented.
- No improvement records, candidates, eval suites, outcomes, promotions, rollbacks, or gates were created or mutated.
- No runtime-pack pointers were mutated.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No hot-update behavior change.
- No new commands.
- No TaskState wrappers.
- No topology, canary, deploy-lock, source-patch, promotion-policy, prompt-pack registry, or skill-pack registry enforcement.
- No V4-067 work.

## Validation

- `/usr/local/go/bin/gofmt -w internal/missioncontrol/types.go internal/missioncontrol/types_test.go internal/missioncontrol/validate.go internal/missioncontrol/validate_test.go`
  - passed
- `git diff --check`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
  - passed
- `/usr/local/go/bin/go test -count=1 ./cmd/picobot`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`
  - passed
- `/usr/local/go/bin/go test -count=1 ./internal/agent`
  - passed
- `/usr/local/go/bin/go test -count=1 ./...`
  - passed
