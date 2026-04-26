# V4-102 Canary-Passed Promotion/Gate Path Assessment

## Scope

V4-102 assesses the smallest safe authority path after a canary-required candidate result has a passed `HotUpdateCanaryEvidenceRecord`.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, canary requirements, canary evidence, candidate promotion decisions, hot-update gates, owner approval records, outcomes, promotions, rollbacks, rollback-apply records, runtime-pack pointers, last-known-good pointers, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-103 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_095_HOT_UPDATE_CANARY_REQUIREMENT_REGISTRY_AFTER.md`
- `docs/maintenance/V4_097_HOT_UPDATE_CANARY_REQUIREMENT_CONTROL_ENTRY_AFTER.md`
- `docs/maintenance/V4_099_HOT_UPDATE_CANARY_EVIDENCE_REGISTRY_AFTER.md`
- `docs/maintenance/V4_101_HOT_UPDATE_CANARY_EVIDENCE_CONTROL_ENTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/candidate_result_registry.go`
- `internal/missioncontrol/candidate_promotion_decision_registry.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/hot_update_gate_registry.go`
- `internal/missioncontrol/hot_update_outcome_registry.go`
- `internal/missioncontrol/promotion_registry.go`
- `internal/missioncontrol/approval.go`
- `internal/missioncontrol/store_records.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- direct-command and status tests adjacent to canary requirement, canary evidence, promotion decision, and hot-update gate paths

## Current Authority Chain

`EvaluateCandidateResultPromotionEligibility(...)` derives these relevant states from the committed candidate result and promotion policy:

- `eligible`
- `canary_required`
- `owner_approval_required`
- `canary_and_owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

The derived state is policy-based. A result whose policy still requires canary remains `canary_required` or `canary_and_owner_approval_required` after passed evidence exists. Passed evidence is a separate committed fact; it does not currently rewrite the candidate result, promotion policy, or derived eligibility status.

`CandidatePromotionDecisionRecord` is deliberately strict:

- `eligibility_state` must be `eligible`
- `decision` must be `selected_for_promotion`
- creation calls `EvaluateCandidateResultPromotionEligibility(...)`
- creation rejects `canary_required`, `owner_approval_required`, and `canary_and_owner_approval_required`
- linkage validation re-runs eligibility and fails if the result is no longer `eligible`

`CreateHotUpdateGateFromCandidatePromotionDecision(...)` is also strict:

- it loads a committed `CandidatePromotionDecisionRecord`
- it requires `decision=selected_for_promotion`
- it requires `eligibility_state=eligible`
- it re-runs candidate-result eligibility and requires `eligible`
- it cross-checks the candidate result, run, candidate, eval suite, promotion policy, baseline pack, candidate pack, active pointer, rollback target, and optional last-known-good pointer
- it builds a normal hot-update gate with `state=prepared` and `decision=keep_staged`

The current gate record shape already has canary-adjacent fields:

- `canary_ref`
- `approval_ref`
- state `canarying`
- decision `apply_canary`
- reload mode `canary_reload`

Those fields exist in the generic gate schema, but the current promotion-decision-to-gate helper does not populate `canary_ref` or `approval_ref`, and the existing gate execution path is still based on a selected eligible promotion decision.

## Current Canary Requirement And Evidence Surface

`HotUpdateCanaryRequirementRecord` is created only for derived states:

- `canary_required`
- `canary_and_owner_approval_required`

It stores stable source refs copied from the candidate result and derived eligibility:

- `result_id`
- `run_id`
- `candidate_id`
- `eval_suite_id`
- `promotion_policy_id`
- `baseline_pack_id`
- `candidate_pack_id`

It also stores:

- `required_by_policy=true`
- `owner_approval_required=true` only for `canary_and_owner_approval_required`
- `state=required`

`HotUpdateCanaryEvidenceRecord` is append-only under:

```text
runtime_packs/hot_update_canary_evidence/<canary_evidence_id>.json
```

The deterministic ID is:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

Only `evidence_state=passed` stores `passed=true`. `failed`, `blocked`, and `expired` evidence records are durable observations, but they do not satisfy the canary requirement.

Evidence creation loads and validates the canary requirement and cross-checks:

- candidate result
- improvement run
- improvement candidate
- frozen eval suite
- promotion policy
- baseline runtime pack
- candidate runtime pack
- freshly derived candidate promotion eligibility

The derived state must remain `canary_required` or `canary_and_owner_approval_required`.

## Direct Command Surface

The current direct commands are:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
```

Both commands use TaskState wrappers, validate active or persisted job context, use `created_by=operator`, emit runtime-control audit events, and return created/selected responses. Neither command creates promotion decisions, hot-update gates, owner approval requests, outcomes, promotions, rollbacks, rollback-apply records, pointer mutations, or reload/apply effects.

## Assessment

Passed canary evidence should not make a canary-required result look like a normal `eligible` result inside `CandidatePromotionDecisionRecord`.

Reasoning:

- `CandidatePromotionDecisionRecord` has a clear existing contract: only derived `eligible` results are selected for promotion.
- Both creation and linkage validation re-run `EvaluateCandidateResultPromotionEligibility(...)`.
- A policy requiring canary still derives `canary_required` after evidence exists; changing that would require broadening the eligibility grammar or mutating source records.
- Reusing the normal decision type for canary-satisfied results would erase the distinction between "policy did not require canary" and "policy required canary and passed evidence satisfied it."

Passed canary evidence should not feed hot-update gate creation directly in the next slice.

Reasoning:

- `HotUpdateGateRecord` has `canary_ref` and `approval_ref`, but the current helper path is grounded in eligible candidate promotion decisions.
- Direct gate creation from evidence would need to decide owner-approval behavior, canary-ref semantics, approval-ref semantics, active pointer checks, rollback target checks, and duplicate gate IDs in one step.
- For `canary_and_owner_approval_required`, a passed canary alone is insufficient; owner approval is still required before progression.
- Creating a gate before representing satisfaction and approval authority would risk bypassing the existing decision contract.

Owner approval must not substitute for canary evidence. For `canary_and_owner_approval_required`, passed canary evidence should produce a read-only state equivalent to "canary satisfied; waiting for owner approval" until a later owner-approval authority record exists.

## Recommended Next Slice

Recommend exactly one V4-103 implementation slice:

```text
V4-103 — Hot-Update Canary Satisfaction Assessment Helper
```

V4-103 should add a missioncontrol read-only helper and status/read-model surface before adding any direct command:

```go
AssessHotUpdateCanarySatisfaction(root, canaryRequirementID string) (HotUpdateCanarySatisfactionAssessment, error)
```

The assessment should not create records. It should compute whether a committed canary requirement has a valid passed evidence record and whether progression is blocked by owner approval.

Recommended assessment states:

- `not_satisfied`
- `satisfied`
- `waiting_owner_approval`
- `failed`
- `blocked`
- `expired`
- `invalid`

State interpretation:

- `satisfied`: requirement is valid, latest selected evidence is `passed`, and `owner_approval_required=false`
- `waiting_owner_approval`: requirement is valid, latest selected evidence is `passed`, and `owner_approval_required=true`
- `not_satisfied`: no valid evidence exists for the requirement
- `failed`: latest selected valid evidence is `failed`
- `blocked`: latest selected valid evidence is `blocked`
- `expired`: latest selected valid evidence is `expired`
- `invalid`: requirement, evidence, source linkage, or derived eligibility is invalid

The selected evidence should be deterministic. Prefer the newest valid evidence by `observed_at`, with filename or `canary_evidence_id` as a stable tie-breaker. The helper should surface invalid records without hiding other records in the status/read model, matching existing identity patterns.

## Source Records To Cross-Check

The read-only assessment should load and cross-check:

- `HotUpdateCanaryRequirementRecord`
- all `HotUpdateCanaryEvidenceRecord` records for that requirement
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- freshly derived `CandidatePromotionEligibilityStatus`

The freshly derived eligibility must remain:

- `canary_required`, or
- `canary_and_owner_approval_required`

The helper should verify copied refs match across requirement, evidence, candidate result, and derived eligibility. It should not load or mutate active runtime-pack pointers, last-known-good pointers, hot-update gates, outcomes, promotions, rollbacks, or rollback-apply records.

## Record And ID Decisions

V4-103 should not add a durable authority record yet.

Reasoning:

- The current missing piece is selecting and explaining satisfaction from existing append-only evidence.
- A durable canary-satisfied decision/proposal record may still be needed later, but its shape depends on owner-approval integration.
- A read-only helper can be safely consumed by later owner-approval and gate slices without forcing a premature record contract.

If a later durable record is needed, it should not be `CandidatePromotionDecisionRecord`. It should be a canary-specific authority record, likely deterministic from the requirement and selected passed evidence:

```text
hot-update-canary-satisfaction-<canary_requirement_id>-<canary_evidence_id>
```

That later record should be immutable once created, exact-replay idempotent, and divergent-duplicate fail-closed. V4-103 should defer that durable record until the read-only assessment proves the state machine and owner-approval split.

## Status / Read Model Recommendation

V4-103 should add a read-only status surface, for example:

```text
hot_update_canary_satisfaction_identity
```

Minimum fields:

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

The read model should surface configured, not-configured, and invalid states without mutating records. It should be included in committed mission status snapshots if consistent with existing status composition patterns.

## Deferred Paths

After V4-103, the grounded follow-up path should be:

1. Add owner-approval proposal/request authority for `waiting_owner_approval` if needed.
2. Add a durable canary-satisfied authority record or gate helper only after owner-approval semantics are represented.
3. Add a specialized hot-update gate helper that consumes the canary-satisfaction authority and populates `canary_ref`, and `approval_ref` when owner approval is required.

Do not create a hot-update gate directly from raw canary evidence until the assessment and owner-approval branches are explicit.

## Fail-Closed Requirements

Future canary-satisfaction logic must fail closed for:

- missing store root
- missing canary requirement
- invalid canary requirement
- canary requirement not `state=required`
- no valid evidence records
- latest selected evidence not `passed`
- stale derived eligibility away from `canary_required` or `canary_and_owner_approval_required`
- evidence whose refs do not match the requirement
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- duplicate or divergent future satisfaction authority records, if such a record is later added
- any attempt to treat owner approval as a substitute for canary evidence
- any attempt to treat passed canary evidence as sufficient when owner approval is still required

## Non-Goals

V4-102 and the recommended V4-103 must not:

- change Go code in V4-102
- change tests in V4-102
- add commands in V4-102
- add TaskState wrappers in V4-102
- create canary-satisfied decisions in V4-102
- create candidate promotion decisions for canary-required states
- broaden the existing eligible-only `CandidatePromotionDecisionRecord` contract
- create hot-update gates for canary-required states
- request owner approval
- create owner approval proposal records
- execute canaries
- create canary evidence
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate canary requirements
- mutate canary evidence
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- change pointer-switch behavior
- change reload/apply behavior
- implement V4-103 in V4-102
