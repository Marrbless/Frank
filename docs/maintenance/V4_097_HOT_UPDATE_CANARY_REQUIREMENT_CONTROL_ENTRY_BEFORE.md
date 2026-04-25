# V4-097 Hot-Update Canary Requirement Control Entry Before

## Before-State Gap

V4-096 assessed the control entry but intentionally did not add it. The durable V4-095 registry could create `HotUpdateCanaryRequirementRecord` only through missioncontrol code:

```go
CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (HotUpdateCanaryRequirementRecord, bool, error)
```

There was no direct operator command and no TaskState wrapper to bind canary requirement creation to the active or persisted mission job context.

## Required Command Shape

V4-097 must add exactly this direct command:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
```

The command uses `result_id` as the source authority. The canary requirement ID remains derived:

```text
hot-update-canary-requirement-<result_id>
```

## Required TaskState Wrapper

The wrapper must expose the existing missioncontrol helper through TaskState:

```go
func (s *TaskState) CreateHotUpdateCanaryRequirementFromCandidateResult(jobID string, resultID string) (missioncontrol.HotUpdateCanaryRequirementRecord, bool, error)
```

It must resolve the mission store root, validate the active or persisted job context, reject mismatched `job_id`, derive `created_by="operator"`, derive `created_at` from the TaskState timestamp path, and call the missioncontrol helper.

For replay stability, it must first derive the deterministic requirement ID and, when the existing record loads successfully, reuse that record's `created_at`. Any existing deterministic record that cannot load or validate must fail closed.

## Expected Responses

Created path:

```text
Created hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

Selected/idempotent path:

```text
Selected hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

Malformed command arguments must return an empty response plus:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE requires job_id and result_id
```

## Required Failure Behavior

The command must fail closed for missing mission store root, missing active and persisted runtime context, wrong `job_id`, missing or invalid candidate result, missing linked run/candidate/eval suite/promotion policy/runtime packs, unfrozen eval suite, non-canary eligibility states, divergent duplicates, stale eligibility on replay, invalid existing deterministic records, and second requirements for the same result under a different ID.

Accepted derived eligibility states are only:

- `canary_required`
- `canary_and_owner_approval_required`

Rejected states include:

- `eligible`
- `owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

## Read-Model Expectation

After a successful create/select, existing `STATUS <job_id>` should surface the record through `hot_update_canary_requirement_identity`. The status path must read records only and must not mutate source records or runtime state.

## Invariants To Preserve

This slice must not execute canaries, create canary evidence, request owner approval, create owner approval proposal records, create candidate promotion decisions for canary-required states, create hot-update gates for canary-required states, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate candidate results, mutate promotion policies, mutate runtime packs, mutate the active runtime-pack pointer, mutate the last-known-good pointer, mutate `reload_generation`, change reload/apply behavior, add TaskState surfaces beyond the wrapper, broaden promotion policy grammar, or implement V4-098.
