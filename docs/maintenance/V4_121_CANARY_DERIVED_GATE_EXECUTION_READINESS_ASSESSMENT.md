# V4-121 - Canary-Derived Hot-Update Gate Execution Readiness Assessment

## Scope

V4-121 assesses whether prepared hot-update gates created by:

```text
HOT_UPDATE_CANARY_GATE_CREATE <job_id> <canary_satisfaction_authority_id> [owner_approval_decision_id]
```

need additional lifecycle guards before phase advance, pointer switch, reload/apply, outcome creation, promotion creation, rollback, rollback-apply, or last-known-good recertification.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, hot-update gates, active runtime-pack pointers, last-known-good pointers, `reload_generation`, outcomes, promotions, rollbacks, rollback-apply records, canary satisfaction authorities, owner approval decisions, source records, or V4-122 implementation.

## Live State Inspected

Starting state:

- branch: `frank-v4-121-canary-derived-gate-execution-readiness-assessment`
- HEAD: `7340a58ab3d749a0b4d6ab2bbf2b428fdaedb906`
- tag at HEAD: `frank-v4-120-canary-gate-lifecycle-checkpoint`
- worktree: clean before this memo
- baseline validator: `/usr/local/go/bin/go test -count=1 ./...`

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_116_CANARY_OWNER_APPROVAL_GATE_AUTHORITY_PATH_ASSESSMENT.md`
- `docs/maintenance/V4_117_HOT_UPDATE_GATE_FROM_CANARY_SATISFACTION_AUTHORITY_AFTER.md`
- `docs/maintenance/V4_119_HOT_UPDATE_CANARY_GATE_CONTROL_ENTRY_AFTER.md`
- `docs/maintenance/V4_120_CANARY_GATE_LIFECYCLE_CHECKPOINT.md`

Code paths inspected:

- `HotUpdateGateRecord`
- `CreateHotUpdateGateFromCanarySatisfactionAuthority(...)`
- `AdvanceHotUpdateGatePhase(...)`
- `ExecuteHotUpdateGatePointerSwitch(...)`
- `ExecuteHotUpdateGateReloadApply(...)`
- `AssessHotUpdateExecutionReadiness(...)`
- `TaskState.AdvanceHotUpdateGatePhase(...)`
- `TaskState.ExecuteHotUpdateGatePointerSwitch(...)`
- `TaskState.ExecuteHotUpdateGateReloadApply(...)`
- direct commands `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, `HOT_UPDATE_GATE_RELOAD`
- `CreateHotUpdateOutcomeFromTerminalGate(...)`
- `CreatePromotionFromSuccessfulHotUpdateOutcome(...)`
- rollback and rollback-apply registries
- `RecertifyLastKnownGoodFromPromotion(...)`
- canary satisfaction authority, owner approval decision, canary satisfaction, canary evidence, and canary requirement registries
- status/read-model behavior for `hot_update_gate_identity`

## Current Prepared Canary Gate Path

V4-117 added the separate canary gate helper:

```go
CreateHotUpdateGateFromCanarySatisfactionAuthority(
    root string,
    canarySatisfactionAuthorityID string,
    ownerApprovalDecisionID string,
    createdBy string,
    createdAt time.Time,
) (HotUpdateGateRecord, bool, error)
```

It creates a normal prepared `HotUpdateGateRecord` with:

```text
state=prepared
decision=keep_staged
canary_ref=<canary_satisfaction_authority_id>
approval_ref=<owner_approval_decision_id only for owner-required granted branch>
```

At creation time, it revalidates source linkage, fresh canary satisfaction, fresh promotion eligibility, active pointer baseline, rollback target, optional LKG pointer validity, and owner approval decision linkage. It does not use `CandidatePromotionDecisionRecord`, and it preserves the eligible-only promotion-decision path.

The current helper already contains useful unexported canary validation building blocks:

- `validateHotUpdateGateCanaryOwnerApprovalBranch(...)`
- `validateHotUpdateGateCanaryAuthorityFreshness(...)`
- `validateHotUpdateGateCanaryAuthorityRuntimeReadiness(...)`
- `validateHotUpdateGateCanaryAuthoritySource(...)`
- `validateHotUpdateGateOwnerApprovalDecisionMatchesAuthority(...)`

Those checks are creation-time only today.

## Existing Generic Lifecycle Behavior

`AdvanceHotUpdateGatePhase(...)` advances any valid gate through:

```text
prepared -> validated -> staged
```

It checks gate state adjacency and runtime-pack linkage, but it does not branch on `canary_ref` or `approval_ref`.

`ExecuteHotUpdateGatePointerSwitch(...)` requires `state=staged`, validates execution linkage, checks that the active pointer still references `previous_active_pack_id`, switches the active pointer to `candidate_pack_id`, writes `update_record_ref=hot_update:<hot_update_id>`, increments `reload_generation`, and moves the gate to `reloading`. It does not revalidate canary satisfaction authority or owner approval authority.

`ExecuteHotUpdateGateReloadApply(...)` requires the active pointer to already reference the candidate pack with the expected hot-update record ref, performs restart-style convergence, and moves the gate to `reload_apply_succeeded`, `reload_apply_failed`, or recovery states. It does not revalidate `canary_ref` or `approval_ref`.

`AssessHotUpdateExecutionReadiness(...)` currently guards execution-sensitive transitions only for live-runtime deploy-lock/quiesce concerns. It classifies `pointer_switch` and `reload_apply` as execution-sensitive, but it does not inspect canary-derived gate authority.

The direct commands route through TaskState:

- `HOT_UPDATE_GATE_PHASE` -> `TaskState.AdvanceHotUpdateGatePhase(...)`
- `HOT_UPDATE_GATE_EXECUTE` -> `TaskState.ExecuteHotUpdateGatePointerSwitch(...)`
- `HOT_UPDATE_GATE_RELOAD` -> `TaskState.ExecuteHotUpdateGateReloadApply(...)`

TaskState validates job context and calls `AssessHotUpdateExecutionReadiness(...)` before execute/reload, but not before phase. Because the shared readiness assessment is generic, the direct commands are not currently protected from stale canary or owner-approval authority.

## Risk: Phase Advancement Without Canary Revalidation

`HOT_UPDATE_GATE_PHASE` should not move a canary-derived gate from `prepared` to `validated` or `staged` without revalidating canary authority.

Reason: `staged` is the immediate precursor to pointer switch. Allowing a canary-derived gate to become staged after the selected evidence, canary satisfaction authority, owner approval decision, promotion eligibility, candidate pack, baseline pack, or active pointer context has drifted creates a misleading execution-ready record.

Recommended fail-closed behavior for canary-derived phase advancement:

- load the gate and detect canary-derived lineage by non-empty `canary_ref`
- require a valid `HotUpdateCanarySatisfactionAuthorityRecord`
- require the authority candidate pack, baseline pack, result, run, candidate, eval suite, promotion policy, requirement, and selected evidence to still match
- require fresh `AssessHotUpdateCanarySatisfaction(...)` to still derive the expected satisfaction state
- require fresh `EvaluateCandidateResultPromotionEligibility(...)` to still derive either `canary_required` or `canary_and_owner_approval_required`
- require owner-required gates to have a non-empty valid `approval_ref`
- require owner approval decision `decision=granted` and matching copied refs
- reject `decision=rejected`
- preserve ordinary eligible-only gates when `canary_ref` is empty

## Risk: Pointer Switch Without Canary Revalidation

`HOT_UPDATE_GATE_EXECUTE` / pointer switch must revalidate canary and owner approval authority before mutating the active runtime-pack pointer and incrementing `reload_generation`.

Pointer switch is the highest-risk transition in this path because it mutates the active runtime pack and records a new reload generation. A canary-derived gate that was valid at creation time may become invalid before execution if evidence is superseded, source records are corrupted, promotion eligibility changes, the owner approval decision is missing or replaced, or the active/baseline context no longer matches.

Recommended behavior:

- run the canary-specific guard immediately before pointer-switch mutation
- guard only gates with non-empty `canary_ref`
- require fresh canary satisfaction
- require fresh promotion eligibility
- require owner approval `decision=granted` for owner-required gates
- reject missing, invalid, stale, mismatched, or rejected owner approval decisions
- reject missing, invalid, stale, or mismatched canary satisfaction authorities
- do not mutate any source record while checking readiness

Owner approval is not a substitute for passed canary evidence. The owner-required branch must still prove the current canary satisfaction state is `waiting_owner_approval`, which itself depends on passed canary evidence.

## Risk: Reload/Apply Without Canary Revalidation

`HOT_UPDATE_GATE_RELOAD` should revalidate canary and owner approval authority before reload/apply.

Even after pointer switch, reload/apply is still an execution-sensitive transition. If the authority chain is found invalid before reload/apply, the system should fail closed into the existing recovery/rollback path rather than completing apply under stale authority.

The reload/apply guard should use the same canary-specific readiness helper as pointer switch. It should not duplicate validation in TaskState or command parsing.

## Risk: Generic Outcome And Promotion Consumption

Outcome creation should remain generic after guarded execution.

Reason: `CreateHotUpdateOutcomeFromTerminalGate(...)` is a ledger transition from a terminal gate state. It should be able to record both success and failure after execution. Blocking outcome creation because a canary source record drifted after terminal execution would risk losing the durable outcome of an already-attempted update.

Promotion creation can remain generic for the next guarded-execution slice, provided canary-derived gates cannot reach `reload_apply_succeeded` through the governed command path without passing the canary execution guard. `CreatePromotionFromSuccessfulHotUpdateOutcome(...)` already requires a successful hot-update outcome and gate/pack linkage.

That said, the current `PromotionRecord` path does not surface canary-specific authority fields from the gate. A later slice may still assess whether promotion records should copy `canary_ref` and `approval_ref` for audit parity with the V4 spec's minimum promotion fields. That is not required before blocking unsafe execution.

## Rollback, Rollback-Apply, And LKG

Rollback and rollback-apply should remain generic recovery paths.

Reason: once a candidate pack has become active or partially applied, rollback authority should be based on restoring previous active or last-known-good packs safely. It should not be blocked by stale canary evidence or a now-invalid owner approval decision. Canary policy can explain why the update should not have executed, but it should not prevent recovery.

LKG recertification can remain generic for the next slice. `RecertifyLastKnownGoodFromPromotion(...)` depends on a promotion, successful outcome, and active pointer state. If guarded execution and generic promotion are accepted, LKG recertification does not need canary-specific policy before V4-122.

## Candidate Guard Options

### Option A: Direct Command Guard

Add canary-specific checks in `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, and `HOT_UPDATE_GATE_RELOAD` parsing.

This is not recommended. It duplicates policy at the command edge and leaves TaskState and missioncontrol callers with weaker semantics.

### Option B: TaskState Guard

Add canary-specific checks in `TaskState.AdvanceHotUpdateGatePhase(...)`, `TaskState.ExecuteHotUpdateGatePointerSwitch(...)`, and `TaskState.ExecuteHotUpdateGateReloadApply(...)`.

This protects the current operator command path, but it still duplicates checks across wrappers and leaves lower-level missioncontrol execution helpers callable without canary validation.

### Option C: Extend Generic Execution Readiness Only

Extend `AssessHotUpdateExecutionReadiness(...)` to call a canary guard when `gate.canary_ref != ""`.

This is useful for `HOT_UPDATE_GATE_EXECUTE` and `HOT_UPDATE_GATE_RELOAD`, because those wrappers already call readiness. It does not cover `HOT_UPDATE_GATE_PHASE`, because phase transitions are currently metadata-class and not execution-sensitive.

### Option D: Shared Missioncontrol Canary Readiness Helper

Add a read-only missioncontrol helper and call it from lifecycle points that can make a canary-derived gate look executable or actually execute it.

Recommended helper shape:

```go
AssessHotUpdateCanaryGateExecutionReadiness(
    root string,
    hotUpdateID string,
) (HotUpdateCanaryGateExecutionReadinessAssessment, error)
```

Alternative lower-level shape:

```go
ValidateHotUpdateCanaryGateAuthorityForExecution(root string, gate HotUpdateGateRecord) error
```

The assessment-returning helper is preferable because it can expose structured reasons in status or readiness surfaces later without changing validation logic again.

The helper should be read-only. It should return ready/not-ready for ordinary non-canary gates without requiring canary records, preserving the normal eligible-only gate path.

## Recommended V4-122 Slice

V4-122 should be:

```text
V4-122 - Canary-Derived Hot-Update Gate Execution Readiness Guard
```

Exact scope:

- add `HotUpdateCanaryGateExecutionReadinessAssessment`
- add `AssessHotUpdateCanaryGateExecutionReadiness(root string, hotUpdateID string) (HotUpdateCanaryGateExecutionReadinessAssessment, error)`
- reuse or factor the existing canary creation-time validation checks from `hot_update_gate_registry.go`
- call the helper from `AdvanceHotUpdateGatePhase(...)` when `canary_ref` is non-empty and the target phase is `validated` or `staged`
- call the helper from `ExecuteHotUpdateGatePointerSwitch(...)` before active pointer mutation when `canary_ref` is non-empty
- call the helper from `ExecuteHotUpdateGateReloadApply(...)` before reload/apply convergence when `canary_ref` is non-empty
- preserve existing `AssessHotUpdateExecutionReadiness(...)` deploy-lock behavior for TaskState execute/reload
- add missioncontrol and direct-command tests proving stale/invalid canary authority, missing/mismatched/rejected owner approval, stale canary satisfaction, stale promotion eligibility, and normal eligible-only gates behave correctly

V4-122 should not change outcome creation, promotion creation, rollback, rollback-apply, LKG recertification, `CandidatePromotionDecisionRecord`, `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, command syntax, or source record mutation.

## Explicit Non-Goals

V4-121 does not:

- change Go code or tests
- add commands
- add TaskState wrappers
- execute gates
- advance gate phase
- pointer-switch
- reload/apply
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- mutate canary satisfaction authority
- mutate owner approval decision
- mutate source records
- broaden `CandidatePromotionDecisionRecord`
- create candidate promotion decisions for canary-required states
- change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`
- implement V4-122

## Fail-Closed Requirements

For canary-derived gates, lifecycle readiness must fail closed when:

- `canary_ref` is missing from a canary gate or points to a missing authority
- the canary satisfaction authority is invalid
- the authority no longer matches requirement, selected evidence, result, run, candidate, eval suite, promotion policy, baseline pack, or candidate pack
- fresh `AssessHotUpdateCanarySatisfaction(...)` is not configured
- fresh canary satisfaction is not the expected state
- selected canary evidence is no longer valid or no longer selected
- fresh promotion eligibility is not `canary_required` for no-owner gates
- fresh promotion eligibility is not `canary_and_owner_approval_required` for owner-required gates
- owner-required gates have empty, missing, invalid, stale, or mismatched `approval_ref`
- owner approval decision is anything other than `granted`
- a rejected owner approval decision is supplied or discovered
- active pointer, rollback target, or present LKG pointer is invalid

Readiness checks must not mutate source records. They must preserve the normal eligible-only path when `canary_ref` is empty.
