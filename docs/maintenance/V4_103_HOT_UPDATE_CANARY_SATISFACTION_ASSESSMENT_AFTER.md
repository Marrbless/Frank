# V4-103 Hot-Update Canary Satisfaction Assessment After

## Implemented Surface

V4-103 adds the read-only missioncontrol helper:

```go
AssessHotUpdateCanarySatisfaction(root, canaryRequirementID string) (HotUpdateCanarySatisfactionAssessment, error)
```

It loads a committed `HotUpdateCanaryRequirementRecord`, scans committed `HotUpdateCanaryEvidenceRecord` records for that requirement, validates source linkage, and reports whether the requirement is satisfied. It does not write records.

## Assessment Shape

The assessment fields are:

- `state`
- `canary_requirement_id`
- `selected_canary_evidence_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `eligibility_state`
- `owner_approval_required`
- `satisfaction_state`
- `evidence_state`
- `passed`
- `observed_at`
- `reason`
- `error`

Satisfaction states are:

- `not_satisfied`
- `satisfied`
- `waiting_owner_approval`
- `failed`
- `blocked`
- `expired`
- `invalid`

## Helper Behavior

The helper validates the requirement and cross-checks:

- committed candidate result
- linked improvement run
- linked improvement candidate
- frozen eval suite
- referenced promotion policy
- baseline runtime pack
- candidate runtime pack
- freshly derived candidate promotion eligibility

Fresh eligibility must remain `canary_required` or `canary_and_owner_approval_required`.

If no valid evidence exists for a valid requirement, the helper returns `not_satisfied`. If matching evidence exists but all matching evidence is invalid, it returns `invalid`. Missing or invalid requirements are `invalid`.

## Evidence Selection

The helper considers only evidence records whose `canary_requirement_id` matches the requirement. Each matching evidence record must validate and must cross-check against the requirement and source records.

Selection is deterministic:

- choose the newest valid evidence by `observed_at`
- break exact timestamp ties by `canary_evidence_id`

Invalid evidence does not hide other valid evidence. If a valid evidence record exists, the helper selects from valid evidence only. The status/read-model identity still surfaces invalid matching evidence beside the selected valid assessment.

## Owner Approval Split

For selected `passed` evidence:

- `satisfied` means `owner_approval_required=false`
- `waiting_owner_approval` means `owner_approval_required=true`

Owner approval is not requested or created in this slice. Passed canary evidence is not treated as a substitute for owner approval.

For selected non-passed evidence:

- `failed` maps from evidence state `failed`
- `blocked` maps from evidence state `blocked`
- `expired` maps from evidence state `expired`

## Status And Read Model

Mission status now includes:

```json
"hot_update_canary_satisfaction_identity": { ... }
```

The identity surfaces:

- `configured` when at least one requirement assessment exists and no invalid matching requirement/evidence state is surfaced
- `not_configured` when no canary requirement records exist
- `invalid` when an invalid requirement or matching invalid evidence is surfaced

Committed mission status snapshots compose the satisfaction identity after canary evidence identity and before candidate promotion decision identity.

The read model is read-only. It does not mutate canary requirements, canary evidence, candidate results, promotion policies, runtime packs, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, active runtime-pack pointers, last-known-good pointers, or `reload_generation`.

## Invariants Preserved

V4-103 preserves these non-goals:

- no durable canary-satisfied decision or proposal record
- no direct command
- no TaskState wrapper
- no candidate promotion decision for canary-required states
- no broadening of `CandidatePromotionDecisionRecord`
- no hot-update gate for canary-required states
- no owner approval request
- no owner approval proposal record
- no canary execution
- no canary evidence creation
- no outcome, promotion, rollback, rollback-apply, or last-known-good creation
- no candidate result mutation
- no canary requirement mutation
- no canary evidence mutation
- no promotion policy mutation
- no runtime pack mutation
- no active runtime-pack pointer mutation
- no last-known-good pointer mutation
- no `reload_generation` mutation
- no pointer-switch behavior change
- no reload/apply behavior change
- no V4-104 work
