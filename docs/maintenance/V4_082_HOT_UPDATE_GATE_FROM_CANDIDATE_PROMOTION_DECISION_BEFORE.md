# V4-082 Hot-Update Gate From Candidate Promotion Decision Before State

## Gap

V4-081 assessed the next safe implementation boundary after `CandidatePromotionDecisionRecord` and concluded that the next slice should add a missioncontrol helper that creates a prepared `HotUpdateGateRecord` from a durable candidate promotion decision.

Before V4-082, hot-update gate creation already existed, but it was candidate-pack driven:

```go
EnsureHotUpdateGateRecordFromCandidate(root, hotUpdateID, candidatePackID, createdBy, requestedAt)
```

That helper derived gate fields from the candidate runtime pack and current active pointer, but it did not use the durable candidate promotion decision as source authority. It did not load or cross-check the candidate promotion decision, linked candidate result, improvement run, improvement candidate, eval suite, promotion policy, or current promotion eligibility.

## Existing Authority Surfaces

The available upstream authority was:

- committed `CandidatePromotionDecisionRecord`
- linked `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- linked frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- baseline and candidate `RuntimePackRecord`
- current active runtime-pack pointer
- optional last-known-good runtime-pack pointer

The existing candidate-pack gate helper already had the gate-field derivation semantics to preserve:

- `candidate_pack_id` from the selected candidate pack
- `previous_active_pack_id` from the current active pointer
- `rollback_target_pack_id` from the candidate runtime pack
- target surfaces, surface classes, reload mode, and compatibility contract from the candidate pack
- prepared state with `decision=keep_staged`

## Required Boundary

V4-082 must add only a missioncontrol helper. It must not add a TaskState wrapper, direct operator command, deploy-lock implementation, canary execution, owner approval, phase advancement, pointer switch, reload/apply, outcome creation, promotion creation, rollback creation, rollback-apply creation, LKG mutation, active pointer mutation, `reload_generation` mutation, or V4-083 work.
