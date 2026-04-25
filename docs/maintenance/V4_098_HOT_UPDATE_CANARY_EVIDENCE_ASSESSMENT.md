# V4-098 Hot-Update Canary Evidence Assessment

## Scope

V4-098 assesses the smallest safe durable surface for representing canary evidence after a `HotUpdateCanaryRequirementRecord` has been created, but before any canary execution automation.

This slice is assessment-only. It does not change Go code, tests, commands, TaskState wrappers, canary requirement records, canary evidence records, owner approval records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, active runtime-pack pointer, last-known-good pointer, `reload_generation`, or V4-099 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_094_HOT_UPDATE_CANARY_REQUIREMENT_PROPOSAL_ASSESSMENT.md`
- `docs/maintenance/V4_095_HOT_UPDATE_CANARY_REQUIREMENT_REGISTRY_AFTER.md`
- `docs/maintenance/V4_096_HOT_UPDATE_CANARY_REQUIREMENT_CONTROL_ENTRY_ASSESSMENT.md`
- `docs/maintenance/V4_097_HOT_UPDATE_CANARY_REQUIREMENT_CONTROL_ENTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol/candidate_result_registry.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- rollback, rollback-apply, last-known-good, status, and committed status snapshot patterns
- `internal/missioncontrol/hot_update_execution_safety_evidence.go` as the closest evidence/idempotence pattern

## Current Canary-Adjacent Surface

`PromotionPolicyRecord` can require canary and owner approval through:

- `requires_canary`
- `requires_owner_approval`
- `max_canary_duration`

`EvaluateCandidateResultPromotionEligibility(root, resultID)` derives:

- `eligible`
- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

`HotUpdateCanaryRequirementRecord` is now the durable policy fact for canary-required candidate results. It is sourced from a committed candidate result and fresh derived eligibility, and it remains in state `required`.

`HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>` creates or selects that requirement through TaskState, with audit action `hot_update_canary_requirement_create`. It does not create canary evidence, owner approval requests, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, pointer changes, or reload changes.

## Downstream Surfaces Are Not Evidence

`HotUpdateGateRecord` already has canary-adjacent fields and enums:

- `canary_ref`
- state `canarying`
- decision `apply_canary`
- reload mode `canary_reload`

Those fields are placeholders for a later gated path. Current gate creation from a candidate promotion decision remains eligible-only and explicitly rejects canary-required eligibility states.

`HotUpdateOutcomeRecord` has outcome kind `canary_applied`, but outcome creation currently derives from terminal reload/apply gate states and only creates `hot_updated` or `failed` outcomes. It is not a canary evidence producer.

`PromotionRecord` is downstream of a successful `hot_updated` outcome. It must not represent canary evidence or pending canary satisfaction.

Rollback, rollback-apply, and last-known-good recertification surfaces are recovery and post-activation surfaces. They must not be used to prove canary success.

## Decision

The next implementation should add a durable canary evidence registry and read-model skeleton.

It should not add a canary execution proposal record first. A proposal records intent, but the current durable gap after V4-097 is the absence of a record that can represent observed canary result facts against an existing requirement.

It should not add canary execution automation first. The spec requires canary evidence before canary-required promotion or hot-update can commit; execution before a durable evidence contract would produce opaque runtime behavior.

It should not add a separate canary requirement satisfaction record first. Satisfaction should be derived from immutable evidence records rather than mutating the original requirement or introducing a second mutable current-state ledger.

It should not change `CandidatePromotionDecisionRecord` to accept canary-required states. That record is intentionally eligible-only and currently feeds hot-update gate creation. Overloading it would risk creating gates for unsatisfied canary requirements.

It should not create hot-update gates for unsatisfied canary requirements, and it should not use owner approval as a substitute for canary evidence.

## Source Authority

Canary evidence should be authorized primarily by a committed `HotUpdateCanaryRequirementRecord`.

The evidence helper should load and validate:

- `HotUpdateCanaryRequirementRecord`
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

The helper should cross-check copied refs against the requirement and source records:

- `canary_requirement_id`
- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`

It should require the linked requirement to remain valid and in `state=required`. Because requirement loading already re-derives eligibility, stale eligibility away from `canary_required` or `canary_and_owner_approval_required` should fail closed.

## Evidence References

The evidence record should reference the canary requirement ID and copy stable source refs from that requirement.

It should not use a future hot-update gate as source authority. Evidence must exist before any later gate path can consume it.

Recommended references:

- primary authority: `canary_requirement_id`
- copied identity: `result_id`, `run_id`, `candidate_id`, `eval_suite_id`, `promotion_policy_id`, `baseline_pack_id`, `candidate_pack_id`

It may later be consumed by a hot-update gate through `canary_ref`, but the gate should point to evidence only after a passed evidence record exists.

## Before Or After Hot-Update Gate

Canary evidence should be created after a canary requirement exists and before a canary-required hot-update gate exists.

Reasoning:

- current promotion decisions and gate creation are eligible-only
- a canary-required result is not ready for gate creation until canary evidence satisfies policy
- storing evidence against the requirement keeps the authority chain independent of a future gate
- a future gate can copy the evidence ID into `canary_ref` without retroactively authorizing itself

## Ledger Shape

Canary evidence should be immutable append-only evidence, not a mutable current-state record.

Reasoning:

- a failed, blocked, expired, or passed canary observation is a historical fact
- retry attempts should not rewrite earlier evidence
- exact replay of the same observation should be byte-stable
- later satisfaction logic can derive "current" canary satisfaction from evidence records without mutating the requirement

Recommended V4-099 record:

```go
type HotUpdateCanaryEvidenceRecord struct {
    RecordVersion int
    CanaryEvidenceID string
    CanaryRequirementID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    EvidenceState string
    Passed bool
    Reason string
    ObservedAt time.Time
    CreatedAt time.Time
    CreatedBy string
}
```

Recommended storage path:

```text
runtime_packs/hot_update_canary_evidence/<canary_evidence_id>.json
```

Recommended deterministic ID helper:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

The compact timestamp should use a repo-consistent filename-safe UTC representation. The point is deterministic replay for the same requirement and observation time while still allowing later append-only attempts with distinct `observed_at`.

## Evidence States

Recommended V4-099 evidence states:

- `passed`
- `failed`
- `blocked`
- `expired`

`passed=true` should be valid only when `evidence_state=passed`. All other states should require `passed=false`.

State meanings:

- `passed`: a bounded canary observation completed successfully for the candidate pack represented by the requirement
- `failed`: the canary ran and observed a failing condition
- `blocked`: the canary could not run because a prerequisite, scope, budget, safety, or operator condition blocked it
- `expired`: a canary observation window expired without a valid pass

`waived` should not be added in V4-099. A waiver is an authority decision, not evidence. If a later slice needs canary waivers, it should add an explicit waiver/approval authority record with its own source authority and should not allow owner approval to silently substitute for required canary evidence.

## Idempotence And Duplicate Behavior

Recommended store behavior:

- first write stores the normalized evidence record and returns `changed=true`
- exact replay returns `changed=false`
- exact replay is byte-stable
- divergent duplicate for the same `canary_evidence_id` fails closed
- multiple evidence records for the same `canary_requirement_id` are allowed only when they have distinct deterministic evidence IDs
- list order is deterministic by filename

This preserves an append-only attempt ledger. V4-099 should not implement current satisfaction selection beyond exposing records in status.

## Read Model

V4-099 should add a read-only operator identity surface:

```text
hot_update_canary_evidence_identity
```

Minimum status fields:

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

The read model should surface:

- `not_configured` when no evidence records exist
- `configured` for valid evidence records
- `invalid` records without hiding other valid records

The read model must not mutate records, candidate results, requirements, promotion policies, runtime packs, gates, outcomes, promotions, rollbacks, rollback-apply records, pointers, or `reload_generation`.

## Owner Approval Relationship

For `canary_and_owner_approval_required`, owner approval should proceed only after a passed canary evidence record exists.

V4-099 should not implement owner approval. It should preserve enough evidence identity and source refs for a later owner-approval control surface to verify that the canary requirement has passed. Owner approval must not satisfy, waive, or replace required canary evidence.

## Later Promotion Or Gate Path

Passed canary evidence should enable a later assessment of the canary-satisfied promotion/gate path.

V4-099 should not:

- change existing eligible-only `CandidatePromotionDecisionRecord`
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- mutate candidate results to mark them eligible
- mutate promotion policies or runtime packs

A later slice should decide whether passed evidence produces a new canary-satisfied decision record type, a specialized gate helper that requires evidence, or another explicitly gated conversion path. That later path must remain fail-closed if evidence is absent, invalid, failed, blocked, expired, stale, or mismatched.

## Fail-Closed Requirements

V4-099 should fail closed for:

- missing or invalid `record_version`
- missing `canary_evidence_id`
- missing `canary_requirement_id`
- missing copied source refs
- invalid evidence state
- `passed=true` when state is not `passed`
- `passed=false` when state is `passed`
- missing reason
- zero `observed_at`
- zero `created_at`
- missing `created_by`
- deterministic evidence ID mismatch
- missing canary requirement
- invalid canary requirement
- requirement not in state `required`
- requirement derived eligibility changing away from canary-required
- copied refs not matching the requirement
- missing linked candidate result, run, candidate, eval suite, promotion policy, baseline pack, or candidate pack
- unfrozen eval suite
- divergent duplicate evidence ID

V4-099 should not decide that an `expired`, `failed`, or `blocked` record permanently prevents retry; it should preserve the facts and leave current satisfaction rules to a later slice.

## Recommended V4-099 Slice

Implement exactly one next slice:

```text
V4-099 — Hot-Update Canary Evidence Registry Skeleton
```

Scope:

- add `HotUpdateCanaryEvidenceRecord` or repo-consistent equivalent
- add deterministic evidence ID helper
- add normalize, validate, store, load, and list helpers
- add creation/storage helper scoped to committed `HotUpdateCanaryRequirementRecord`
- cross-check requirement and copied source refs
- add read-only `hot_update_canary_evidence_identity` status surface
- include the identity in committed mission status snapshots if consistent with existing status composition patterns
- add focused missioncontrol tests for validation, idempotence, duplicates, deterministic list order, linkage validation, and read-only status behavior
- add before/after maintenance docs

Do not add direct commands or TaskState wrappers in V4-099. The repo pattern should remain registry/read-model first, then a later control-entry slice.

## Non-Goals

V4-098 and the recommended V4-099 must not:

- execute canaries
- create canary execution automation
- create canary execution proposal records
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- change pointer-switch behavior
- change reload/apply behavior
- broaden promotion policy grammar
- claim V4 complete while canary and owner-approval branches remain unimplemented
