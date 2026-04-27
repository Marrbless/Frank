# V4-126 Canary Gate Operator Runbook And Help Text After

## Scope

V4-126 updates operator-facing documentation for the completed canary-required hot-update path. The implementation is docs/runbook only.

## Runbook Updated

`docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` now documents both hot-update paths:

- the existing eligible-only path
- the canary-required path through promotion audit lineage

The runbook explicitly states that the canary-required path is complete through promotion audit lineage, while the eligible-only path remains separate. It also states that `CandidatePromotionDecisionRecord` remains strictly eligible-only and that `CreateHotUpdateGateFromCandidatePromotionDecision(...)` remains unchanged.

## Canary Branches Documented

The no-owner-approval branch is documented as:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=satisfied
-> canary satisfaction authority=authorized
-> prepared canary gate with canary_ref and empty approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref
-> promotion with copied canary_ref
```

The owner-approved branch is documented as:

```text
candidate result + policy
-> canary requirement
-> passed canary evidence
-> canary satisfaction=waiting_owner_approval
-> canary satisfaction authority=waiting_owner_approval
-> owner approval request=requested
-> owner approval decision=granted
-> prepared canary gate with canary_ref and approval_ref
-> guarded phase/execute/reload
-> outcome with copied canary_ref and approval_ref
-> promotion with copied canary_ref and approval_ref
```

Rejected owner approval is documented as terminal rejection authority that does not authorize `HOT_UPDATE_CANARY_GATE_CREATE` and must not authorize execution. Natural-language aliases such as `yes`, `no`, `approve`, `deny`, `approved`, and `denied` remain intentionally unbound.

## Command Sequences Documented

The runbook now includes exact operator sequences for:

- no-owner canary branch
- owner-approved canary branch
- rejected owner approval decision

The canary sequences use the live command names from the direct-command parser and continue through phase, execute, reload, outcome, and promotion creation. The existing eligible-only sequence remains documented separately.

## Status And Read Models

The runbook now points operators to `STATUS <job_id>` and lists the status sections to inspect as the canary path progresses:

- `hot_update_canary_requirement_identity`
- `hot_update_canary_evidence_identity`
- `hot_update_canary_satisfaction_identity`
- `hot_update_canary_satisfaction_authority_identity`
- `hot_update_owner_approval_request_identity`
- `hot_update_owner_approval_decision_identity`
- `hot_update_gate_identity`
- `hot_update_outcome_identity`
- `promotion_identity`
- `rollback_identity`
- `rollback_apply_identity`
- `runtime_pack_identity.active`
- `runtime_pack_identity.last_known_good`

The runbook states that canary-derived gates must show `canary_ref`, owner-approved gates must also show `approval_ref`, and downstream outcome/promotion identities must show the copied lineage refs.

## Guard Behavior

The runbook documents that canary-derived gates are guarded before:

- phase advancement to `validated` or `staged`
- pointer switch
- reload/apply

Guard failures are documented as fail-closed and expected to preserve runtime/source state.

## Outcome And Promotion Behavior

The runbook documents that outcome creation does not re-run canary readiness after terminal execution and that promotion creation does not re-run canary readiness after a successful terminal outcome.

It also documents the preserved outcome kinds:

- successful canary-derived outcomes remain `outcome_kind = hot_updated`
- failed canary-derived outcomes remain `outcome_kind = failed`
- `canary_applied` is not used by this path

Outcome and promotion audit-lineage propagation is documented as copied `canary_ref` and `approval_ref`.

## Generic Recovery And LKG

The runbook documents that rollback and rollback-apply remain generic recovery flows and that LKG recertification remains generic after promotion. It also states that rollback/LKG should not be blocked by stale canary evidence or owner-approval drift after execution.

## Help Text Decision

No central direct-command help command or docs-generated command summary exists for the hot-update direct commands. V4-126 therefore did not change any code help surface, command parser, CLI behavior, or tests.

## Invariants Preserved

V4-126 preserves:

- no command syntax change
- no new command
- no new TaskState wrapper
- no missioncontrol authority or schema change
- no runtime behavior change
- no rollback, rollback-apply, or LKG behavior change
- no pointer mutation
- no `reload_generation` mutation
- no natural-language owner approval binding
- no `CandidatePromotionDecisionRecord` broadening
- no change to `CreateHotUpdateGateFromCandidatePromotionDecision(...)`
- no V4-127 work
