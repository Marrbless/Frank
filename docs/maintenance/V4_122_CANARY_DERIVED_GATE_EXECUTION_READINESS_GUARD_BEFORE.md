# V4-122 Canary-Derived Gate Execution Readiness Guard - Before

## Scope

This slice starts from V4-121, which identified that canary-derived hot-update gates could be created from a satisfied canary authority but then advanced through later lifecycle points using only generic gate checks.

The target change is a shared, read-only `missioncontrol` readiness helper for canary-derived gates and wiring at the smallest execution-adjacent lifecycle points:

- `AdvanceHotUpdateGatePhase(...)` for validated and staged transitions.
- `ExecuteHotUpdateGatePointerSwitch(...)` before active pointer mutation.
- `ExecuteHotUpdateGateReloadApply(...)` before reload/apply convergence writes.

## Before-State Gap From V4-121

V4-121 found that `HOT_UPDATE_CANARY_GATE_CREATE` already validates the canary satisfaction authority, owner approval branch, fresh canary satisfaction, fresh promotion eligibility, source record linkage, active pointer, rollback target, and present LKG pointer before creating a prepared canary-derived gate.

After the prepared gate exists, later lifecycle calls do not revalidate the canary authority chain. A stale failed canary evidence record, changed promotion eligibility, missing authority, rejected owner approval decision, active pointer drift, missing rollback target, or invalid present LKG pointer can therefore be missed by phase, execute, or reload/apply paths unless guarded again.

## Helper Shape

The intended helper is:

```go
type HotUpdateCanaryGateExecutionReadinessAssessment struct {
    State string
    HotUpdateID string
    CanaryRef string
    ApprovalRef string
    CanarySatisfactionAuthorityID string
    OwnerApprovalDecisionID string
    ResultID string
    RunID string
    CandidateID string
    EvalSuiteID string
    PromotionPolicyID string
    BaselinePackID string
    CandidatePackID string
    ExpectedEligibilityState string
    SatisfactionState HotUpdateCanarySatisfactionState
    OwnerApprovalRequired bool
    Ready bool
    Reason string
    Error string
}
```

The intended entry point is:

```go
AssessHotUpdateCanaryGateExecutionReadiness(root string, hotUpdateID string) (HotUpdateCanaryGateExecutionReadinessAssessment, error)
```

## Expected Behavior

Non-canary gates with empty `canary_ref` should return a ready `not_applicable` assessment and should not require canary registries.

No-owner canary gates should require:

- `gate.canary_ref` equals the authority ID.
- `gate.approval_ref` is empty.
- Authority state is `authorized`.
- `owner_approval_required=false`.
- Fresh satisfaction remains `satisfied`.
- Fresh eligibility remains `canary_required`.
- Selected evidence still loads, is selected, and passed.
- Active pointer still matches the baseline before phase and pointer-switch execution.
- Candidate pack, rollback target, and present LKG pointer remain loadable.

Owner-approved canary gates should require:

- `gate.canary_ref` equals the authority ID.
- `gate.approval_ref` is non-empty.
- Authority state is `waiting_owner_approval`.
- `owner_approval_required=true`.
- Fresh satisfaction remains `waiting_owner_approval`.
- Fresh eligibility remains `canary_and_owner_approval_required`.
- The owner approval decision loads, matches the authority refs, and has `decision=granted`.
- Rejected, stale, missing, or mismatched owner approval decisions fail closed.

## Source Records Loaded And Cross-Checked

The guard should load and cross-check the hot-update gate, canary satisfaction authority, canary requirement, selected canary evidence, fresh canary satisfaction assessment, candidate result, improvement run, improvement candidate, eval suite, promotion policy, baseline runtime pack, candidate runtime pack, active runtime-pack pointer, candidate rollback target runtime pack, present LKG pointer, and owner approval decision when required.

## Non-Goals

This slice must not change command syntax, add commands, add TaskState wrappers, create outcomes, create promotions, create rollbacks, create rollback-apply records, mutate source records from readiness checks, mutate LKG, broaden `CandidatePromotionDecisionRecord`, change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, or implement V4-123.
