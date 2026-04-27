# V4-122 Canary-Derived Gate Execution Readiness Guard - After

## Scope

V4-122 adds a shared, read-only `missioncontrol` guard for canary-derived hot-update gates and wires it into the lifecycle points that can move a prepared canary gate toward runtime effects.

The slice preserves the normal eligible-only path and keeps command syntax, TaskState wrappers, outcome creation, promotion creation, rollback, rollback-apply, LKG recertification, and `CandidatePromotionDecisionRecord` unchanged.

## Helper Shape

`internal/missioncontrol/hot_update_gate_registry.go` now exposes:

```go
AssessHotUpdateCanaryGateExecutionReadiness(root string, hotUpdateID string) (HotUpdateCanaryGateExecutionReadinessAssessment, error)
```

The structured assessment reports state, hot update ID, canary and approval refs, authority and decision IDs, result/run/candidate/eval suite/promotion policy refs, baseline and candidate packs, expected eligibility state, satisfaction state, owner approval requirement, readiness, reason, and error text.

## Non-Canary Gate Behavior

When `HotUpdateGateRecord.canary_ref` is empty, the helper returns ready with state `not_applicable`. It does not require canary authority, evidence, requirement, or owner approval records. Existing eligible-only gate creation and lifecycle behavior remains unchanged.

## No-Owner Canary Gate Behavior

For canary-derived gates without owner approval, the helper revalidates the full authority chain:

- `gate.canary_ref` equals the canary satisfaction authority ID.
- `gate.approval_ref` is empty.
- Authority remains `authorized`.
- `owner_approval_required=false`.
- Fresh canary satisfaction remains `satisfied`.
- Fresh promotion eligibility remains `canary_required`.
- Selected canary evidence still loads, is selected by the fresh assessment, and is passed.
- Gate source refs still match authority refs.
- Candidate pack, rollback target, active pointer, and present LKG pointer remain valid.

## Owner-Approved Canary Gate Behavior

For owner-required canary gates, the helper requires:

- `gate.canary_ref` equals the canary satisfaction authority ID.
- `gate.approval_ref` is non-empty.
- Authority remains `waiting_owner_approval`.
- `owner_approval_required=true`.
- Fresh canary satisfaction remains `waiting_owner_approval`.
- Fresh promotion eligibility remains `canary_and_owner_approval_required`.
- Owner approval decision loads, matches the canary authority copied refs, and has `decision=granted`.

Missing, rejected, stale, or mismatched owner approval decisions fail closed.

## Integration Points

`AdvanceHotUpdateGatePhase(...)` calls the guard for canary-derived gates before writing transitions to `validated` or `staged`.

`ExecuteHotUpdateGatePointerSwitch(...)` calls the guard after loading the gate and before active runtime-pack pointer mutation.

`ExecuteHotUpdateGateReloadApply(...)` calls the guard before reload/apply convergence writes. The guard is state-aware: phase and pointer-switch states require the active pointer to still be the baseline, while reload/apply states require the pointer already switched to the candidate by the same hot-update record.

Direct commands `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, and `HOT_UPDATE_GATE_RELOAD` inherit the guard through the existing TaskState and missioncontrol call paths. No direct command syntax changed.

## Source Records Loaded And Cross-Checked

The guard loads and cross-checks:

- `HotUpdateGateRecord`
- `HotUpdateCanarySatisfactionAuthorityRecord`
- `HotUpdateCanaryRequirementRecord`
- selected `HotUpdateCanaryEvidenceRecord`
- fresh `AssessHotUpdateCanarySatisfaction(...)`
- committed `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline `RuntimePackRecord`
- candidate `RuntimePackRecord`
- active runtime-pack pointer
- candidate rollback target runtime pack
- present LKG pointer, when present
- owner approval decision record, when required

## Fail-Closed Behavior

The guard rejects missing or invalid roots, missing or invalid hot update IDs, missing gates, missing or invalid canary authorities, authority/gate source mismatch, stale satisfaction, stale promotion eligibility, selected evidence missing or no longer passed, missing rollback targets, missing rollback target packs, active pointer drift, invalid present LKG pointer, missing owner approval decision, rejected owner approval decisions, and mismatched owner approval decisions.

Readiness checks do not mutate source records, repair invalid records, create records, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate active pointer, mutate LKG, or increment `reload_generation`.

## Invariants Preserved

The normal eligible-only gate path remains generic. `CandidatePromotionDecisionRecord` remains strictly eligible-only. `CreateHotUpdateGateFromCandidatePromotionDecision(...)` is unchanged. Owner approval is not treated as a substitute for passed canary evidence. Rejected owner approval decisions do not pass the guard. Outcome and promotion consumption remain generic after guarded execution. Rollback and LKG behavior remain generic and unchanged. V4-123 work was not implemented.
