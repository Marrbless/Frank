# V4-082 Hot-Update Gate From Candidate Promotion Decision After State

## Helper

V4-082 adds the missioncontrol-only helper:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

It creates a prepared `HotUpdateGateRecord` from a committed `CandidatePromotionDecisionRecord`.

## Deterministic Identity

The helper derives the hot-update ID from the durable decision ID:

```text
hot-update-<promotion_decision_id>
```

For example:

```text
hot-update-candidate-promotion-decision-result-eligible
```

The caller does not supply the hot-update ID.

## Source Authority

The helper loads and cross-checks:

- committed `CandidatePromotionDecisionRecord`
- linked `CandidateResultRecord`
- re-derived `CandidatePromotionEligibilityStatus`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- linked frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- active runtime-pack pointer
- last-known-good runtime-pack pointer when present

The helper requires:

- decision `selected_for_promotion`
- eligibility state `eligible`
- current derived eligibility still `eligible`
- decision fields to match the candidate result and derived eligibility
- run, candidate, and eval-suite linkage to match the decision
- active pointer to exist
- active pointer `active_pack_id` to equal the decision baseline pack
- candidate pack `rollback_target_pack_id` to be present
- rollback target pack to load successfully

Rollback compatibility preserves the existing gate semantics in this slice: the rollback target must be declared on the candidate pack and load as a runtime pack. Stricter last-known-good policy checks remain deferred.

## Gate Derivation

Gate fields continue to use the existing candidate-pack derivation semantics:

- `candidate_pack_id` comes from the decision candidate pack
- `previous_active_pack_id` comes from the current active pointer
- `rollback_target_pack_id` comes from the candidate runtime pack
- target surfaces, surface classes, reload mode, and compatibility contract come from the candidate pack
- the created gate starts in `prepared`
- the created gate uses `decision=keep_staged`

The helper does not copy candidate/run/result/policy refs into the gate because the current `HotUpdateGateRecord` schema has no fields for those refs. The durable authority remains the deterministic hot-update ID derived from the promotion decision plus the cross-checked source chain.

## Replay And Duplicates

Exact replay with the same deterministic gate returns `changed=false` and leaves the gate bytes stable.

A divergent existing gate with the same deterministic hot-update ID fails closed. An existing same-ID gate with a different candidate pack also fails closed.

## Tests

V4-082 adds focused missioncontrol coverage proving:

- eligible promotion decision creates a prepared gate
- deterministic hot-update ID is `hot-update-<promotion_decision_id>`
- candidate pack comes from the decision
- previous active pack comes from the current active pointer
- missing or stale active pointer fails closed
- missing rollback target field fails closed
- missing rollback target runtime pack fails closed
- stale decision/result authority fails closed
- derived eligibility changing away from `eligible` fails closed
- exact replay is byte-stable and returns `changed=false`
- divergent duplicate gates fail closed
- same deterministic gate ID with different candidate pack fails closed
- source records, active pointer, last-known-good pointer, and `reload_generation` are not mutated
- no hot-update outcomes, promotions, rollbacks, or rollback-apply records are created

## Invariants Preserved

V4-082 does not add TaskState wrappers, direct operator commands, deploy-lock implementation, unsafe-live-job blocking, canary execution, owner approval, gate phase advancement, pointer switch, reload/apply, `HotUpdateOutcomeRecord`, `PromotionRecord`, rollback records, rollback-apply records, LKG records, active pointer mutation, last-known-good pointer mutation, `reload_generation` mutation, candidate promotion decision mutation, candidate result mutation, improvement run mutation, improvement candidate mutation, eval-suite mutation, promotion policy mutation, runtime-pack mutation, or V4-083 work.
