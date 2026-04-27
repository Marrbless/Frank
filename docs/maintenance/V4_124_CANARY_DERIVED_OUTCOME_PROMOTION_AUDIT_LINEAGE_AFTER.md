# V4-124 - Canary-Derived Outcome/Promotion Audit Lineage After

## Scope

V4-124 propagates immutable canary-derived audit lineage from terminal hot-update gates into downstream outcome and promotion records/read models.

The implementation is limited to missioncontrol record fields, linkage validation, read-model fields, focused tests, and this maintenance note. It does not change command syntax, add commands, add TaskState wrappers, alter execution, alter outcome kinds, alter rollback, alter rollback-apply, alter LKG recertification, mutate source records, broaden `CandidatePromotionDecisionRecord`, or implement V4-125.

## Record Field Additions

`HotUpdateOutcomeRecord` now includes:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

`PromotionRecord` now includes:

```go
CanaryRef   string `json:"canary_ref,omitempty"`
ApprovalRef string `json:"approval_ref,omitempty"`
```

Both records normalize the new fields, validate non-empty refs with the existing canary satisfaction authority and owner approval decision ref validators, and preserve empty refs for non-canary records.

## Outcome Lineage Copy Behavior

`CreateHotUpdateOutcomeFromTerminalGate(...)` copies `canary_ref` and `approval_ref` from the linked `HotUpdateGateRecord` into the new outcome record.

Non-canary gates produce outcomes with empty refs. No-owner canary gates produce outcomes with `canary_ref` populated and `approval_ref` empty. Owner-approved canary gates produce outcomes with both refs populated.

Successful canary-derived outcomes remain `outcome_kind=hot_updated`. Failed canary-derived outcomes remain `outcome_kind=failed`. V4-124 does not use `canary_applied`.

Outcome creation does not call `AssessHotUpdateCanaryGateExecutionReadiness(...)` and does not re-authorize source canary records after terminal execution.

## Promotion Lineage Copy Behavior

`CreatePromotionFromSuccessfulHotUpdateOutcome(...)` copies `canary_ref` and `approval_ref` from the linked outcome into the new promotion record.

Promotion linkage now verifies:

- promotion refs match the linked gate refs
- linked outcome refs match the linked gate refs
- promotion refs match the linked outcome refs

This preserves outcome as the source of truth for promotion lineage while failing closed if the outcome/gate chain diverges.

Promotion creation does not call `AssessHotUpdateCanaryGateExecutionReadiness(...)` and does not re-authorize source canary records after a successful terminal outcome.

## Status And Read Models

`OperatorHotUpdateOutcomeStatus` now exposes:

- `canary_ref`
- `approval_ref`

`OperatorPromotionStatus` now exposes:

- `canary_ref`
- `approval_ref`

The read models remain read-only. They normalize and validate records for status reporting, mark invalid records as invalid, and do not repair records or hide valid records.

## Replay And Divergent Duplicates

Exact outcome replay remains byte-stable and returns `changed=false`.

Exact promotion replay remains byte-stable and returns `changed=false`.

Divergent duplicate outcomes with mismatched `canary_ref` or `approval_ref` fail closed through the existing load/linkage path.

Divergent duplicate promotions with mismatched `canary_ref` or `approval_ref` fail closed through the existing load/linkage path.

Promotion creation also fails closed when linked outcome refs and linked gate refs disagree.

## Non-Canary Behavior

Normal eligible-only gates continue to produce outcomes and promotions with empty `canary_ref` and `approval_ref`.

Existing generic outcome creation, promotion creation, replay, duplicate detection, candidate/run/result linkage, direct-command routing, and TaskState routing remain unchanged for non-canary records.

## Canary No-Owner Behavior

No-owner canary-derived gates now preserve lineage downstream:

- gate `canary_ref` copies into outcome `canary_ref`
- outcome `canary_ref` copies into promotion `canary_ref`
- `approval_ref` stays empty in gate, outcome, and promotion

The implementation does not require owner approval for this branch and does not add any post-terminal canary authority recheck.

## Owner-Approved Behavior

Owner-approved canary-derived gates now preserve both refs downstream:

- gate `canary_ref` and `approval_ref` copy into outcome
- outcome `canary_ref` and `approval_ref` copy into promotion
- rejected or stale owner approval decisions are still guarded before execution by V4-122, not rechecked during outcome or promotion creation

After terminal execution, the downstream records preserve lineage instead of re-authorizing the original decision.

## Rollback, Rollback-Apply, And LKG

Rollback and rollback-apply remain generic recovery flows.

LKG recertification remains generic and derives audit lineage through the promotion basis rather than storing new canary fields in LKG records.

V4-124 adds no canary-specific rollback, rollback-apply, or LKG policy. Stale canary evidence or owner approval drift after execution must not block recovery or downstream accounting.

## Invariants Preserved

V4-124 preserves:

- no command syntax change
- no new command
- no new TaskState wrapper
- no gate execution, phase advancement, pointer switch, reload/apply, rollback, rollback-apply, or LKG behavior change
- no active runtime-pack pointer mutation outside existing execution code
- no last-known-good pointer mutation
- no `reload_generation` mutation outside existing execution code
- no canary satisfaction authority mutation
- no owner approval request or decision mutation
- no canary evidence or requirement mutation
- no source record mutation from readiness or lineage checks
- no `CandidatePromotionDecisionRecord` broadening
- no `CreateHotUpdateGateFromCandidatePromotionDecision(...)` change
- no V4-125 work

## Validation Coverage

Focused tests cover:

- non-canary outcome and promotion refs remain empty
- no-owner canary outcome and promotion copy `canary_ref`
- owner-approved canary outcome and promotion copy both refs
- failed canary-derived outcome remains `failed`
- successful canary-derived outcome remains `hot_updated`
- outcome and promotion creation do not re-authorize canary source records after terminal execution
- exact replay remains byte-stable
- divergent duplicate lineage fails closed
- outcome/gate lineage mismatch blocks promotion
- outcome and promotion status surfaces expose copied refs
- source records, active pointer, LKG pointer, and `reload_generation` remain unchanged by outcome/promotion creation
- no rollback, rollback-apply, or candidate promotion decision records are created by the new lineage propagation
