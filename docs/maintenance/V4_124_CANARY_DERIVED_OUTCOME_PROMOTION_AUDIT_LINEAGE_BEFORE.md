# V4-124 - Canary-Derived Outcome/Promotion Audit Lineage Before

## Scope

V4-124 starts from the V4-123 assessment and implements audit-lineage propagation only. The target gap is that canary-derived hot-update gates carry `canary_ref` and `approval_ref`, but downstream outcome and promotion records/read models do not preserve those refs directly.

This slice does not change command syntax, add commands, add TaskState wrappers, change outcome kinds, execute gates, advance phases, pointer-switch, reload/apply, create rollbacks, create rollback-apply records, mutate runtime pointers, mutate `reload_generation`, mutate canary or owner approval source records, broaden `CandidatePromotionDecisionRecord`, change rollback/LKG behavior, or implement V4-125.

## Before-State Gap From V4-123

V4-123 concluded that V4-122 already guards canary-derived execution at the appropriate pre-effect lifecycle points. After a gate becomes terminal, outcome and promotion creation should remain generic and should not re-run `AssessHotUpdateCanaryGateExecutionReadiness(...)`.

The remaining gap is audit parity:

- `HotUpdateGateRecord` has `canary_ref` and `approval_ref`.
- `hot_update_gate_identity` exposes those refs.
- `HotUpdateOutcomeRecord` does not have copied canary/approval refs.
- `PromotionRecord` does not have copied canary/approval refs.
- `hot_update_outcome_identity` and `promotion_identity` cannot show those refs without joining back to the gate.

## Expected Helper Shape

No new readiness helper is added in V4-124. The existing outcome and promotion helpers remain the only write paths:

- `CreateHotUpdateOutcomeFromTerminalGate(...)`
- `CreatePromotionFromSuccessfulHotUpdateOutcome(...)`

The implementation should extend those helpers and their existing linkage validators rather than introduce a new execution or authority path.

## Structured Fields To Add

`HotUpdateOutcomeRecord` should gain:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

`PromotionRecord` should gain:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

The corresponding operator status records should expose the same JSON fields.

## Non-Canary Behavior

Non-canary gates have empty `canary_ref` and `approval_ref`. Outcomes created from those gates should keep both fields empty, and promotions created from those outcomes should keep both fields empty.

Existing generic replay, duplicate detection, linkage checks, outcome kinds, promotion creation, rollback, rollback-apply, LKG, direct-command, and TaskState behavior should remain unchanged for non-canary gates.

## Canary No-Owner Branch

For a no-owner canary-derived terminal gate:

- outcome `canary_ref` should copy the linked gate `canary_ref`
- outcome `approval_ref` should remain empty
- promotion `canary_ref` should copy the linked outcome `canary_ref`
- promotion `approval_ref` should remain empty
- successful outcomes should remain `outcome_kind=hot_updated`
- failed outcomes should remain `outcome_kind=failed`

## Owner-Approved Branch

For an owner-approved canary-derived terminal gate:

- outcome `canary_ref` should copy the linked gate `canary_ref`
- outcome `approval_ref` should copy the linked gate `approval_ref`
- promotion `canary_ref` should copy the linked outcome `canary_ref`
- promotion `approval_ref` should copy the linked outcome `approval_ref`
- promotion creation should fail closed if outcome and gate lineage disagree

V4-124 does not treat owner approval as a substitute for canary evidence and does not re-authorize owner approval after terminal execution.

## Integration Points

Outcome creation should copy lineage while building the deterministic outcome record from a terminal gate.

Promotion creation should copy lineage while building the deterministic promotion record from a successful hot-update outcome.

Status/read-model composition should expose copied refs without repairing invalid records. Invalid records should remain visible as invalid statuses.

## Fail-Closed Behavior

Replay and duplicates should remain append-only:

- exact outcome replay returns unchanged and preserves bytes
- exact promotion replay returns unchanged and preserves bytes
- divergent duplicate outcomes with mismatched `canary_ref` or `approval_ref` fail closed
- divergent duplicate promotions with mismatched `canary_ref` or `approval_ref` fail closed
- promotion creation fails closed if linked outcome refs and linked gate refs disagree

## Source Records Cross-Checked

V4-124 should cross-check only the downstream lineage already linked by the terminal gate and outcome:

- `HotUpdateGateRecord`
- `HotUpdateOutcomeRecord`
- `PromotionRecord`
- linked runtime packs and existing candidate/run/result refs when present

It should not load canary satisfaction authority, owner approval decision, canary evidence, canary requirement, or fresh promotion eligibility for re-authorization.

## Invariants Preserved

V4-124 preserves:

- no command syntax change
- no new command
- no new TaskState wrapper
- no outcome kind change
- no canary readiness re-run from outcome or promotion creation
- rollback, rollback-apply, and LKG remain generic
- no active pointer mutation
- no LKG pointer mutation
- no `reload_generation` mutation
- no source record mutation
- no `CandidatePromotionDecisionRecord` broadening
- no V4-125 work
