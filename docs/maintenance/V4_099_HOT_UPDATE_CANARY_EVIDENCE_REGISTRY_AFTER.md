# V4-099 Hot-Update Canary Evidence Registry After State

## Record Shape

V4-099 adds `HotUpdateCanaryEvidenceRecord`:

- `record_version`
- `canary_evidence_id`
- `canary_requirement_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `evidence_state`
- `passed`
- `reason`
- `observed_at`
- `created_at`
- `created_by`

Valid evidence states are:

- `passed`
- `failed`
- `blocked`
- `expired`

`passed=true` is valid only when `evidence_state=passed`. All non-passed states require `passed=false`.

## Storage Path And Deterministic ID

Records are stored under:

```text
runtime_packs/hot_update_canary_evidence/<canary_evidence_id>.json
```

The deterministic ID helper is:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

The implementation uses a filename-safe compact UTC timestamp with nanosecond precision:

```text
YYYYMMDDThhmmssnnnnnnnnnZ
```

Validation rejects any record whose `canary_evidence_id` does not match the deterministic ID for its `canary_requirement_id` and `observed_at`.

## Validation Behavior

Validation rejects:

- missing or invalid `record_version`
- missing `canary_evidence_id`
- missing `canary_requirement_id`
- missing `result_id`
- missing `run_id`
- missing `candidate_id`
- missing `eval_suite_id`
- missing `promotion_policy_id`
- missing `baseline_pack_id`
- missing `candidate_pack_id`
- invalid `evidence_state`
- `passed=true` when `evidence_state` is not `passed`
- `passed=false` when `evidence_state=passed`
- missing `reason`
- zero `observed_at`
- zero `created_at`
- missing `created_by`
- deterministic evidence ID mismatch

Normalization trims string fields and normalizes timestamps to UTC before validation and storage.

## Source Authority Records

Creation and linkage validation rely on committed source authority:

- `HotUpdateCanaryRequirementRecord`
- `CandidateResultRecord`
- `ImprovementRunRecord`
- `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

The helper requires the linked canary requirement to remain valid and in:

```text
state=required
```

The helper requires the freshly derived promotion eligibility to remain:

- `canary_required`, or
- `canary_and_owner_approval_required`

The helper copies stable refs from the requirement/source records. It does not accept caller-supplied result, run, candidate, eval-suite, policy, or runtime-pack refs.

## Creation Helper Behavior

V4-099 adds:

```go
CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID string, state HotUpdateCanaryEvidenceState, observedAt time.Time, createdBy string, createdAt time.Time, reason string) (HotUpdateCanaryEvidenceRecord, bool, error)
```

The helper:

- validates the store root
- normalizes and validates the canary requirement ID
- validates the requested evidence state
- rejects zero `observed_at`
- rejects missing `created_by`
- rejects zero `created_at`
- rejects missing `reason`
- loads the committed canary requirement
- requires requirement `state=required`
- cross-checks candidate result, run, candidate, frozen eval suite, promotion policy, baseline pack, candidate pack, and freshly derived eligibility
- copies source refs from the requirement
- sets `passed=true` only for `evidence_state=passed`
- stores or selects the normalized evidence record

## Idempotence And Duplicate Behavior

- first write stores the normalized evidence record and returns `changed=true`
- exact replay returns `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `canary_evidence_id` fails closed
- multiple evidence records for the same `canary_requirement_id` are allowed only when they have distinct deterministic evidence IDs
- list order is deterministic by file name

V4-099 intentionally does not implement current satisfaction selection. Later slices must decide how passed evidence is consumed by owner approval, promotion, or hot-update gate paths.

## Status / Read Model

V4-099 adds the read-only operator identity surface:

```text
hot_update_canary_evidence_identity
```

Minimum status fields are surfaced:

- `state`
- `canary_evidence_id`
- `canary_requirement_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`
- `evidence_state`
- `passed`
- `reason`
- `observed_at`
- `created_at`
- `created_by`
- `error`

The read model surfaces:

- `not_configured` when no evidence records exist
- `configured` for valid evidence records
- `invalid` records without hiding other valid records

`BuildCommittedMissionStatusSnapshot` includes the identity through the existing status composition path. The read model does not mutate records.

## Invariants Preserved

V4-099 does not:

- execute canaries
- create canary execution automation
- create canary execution proposal records
- add a direct command
- add a TaskState wrapper
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate canary requirements
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- change pointer-switch behavior
- change reload/apply behavior
- implement V4-100
