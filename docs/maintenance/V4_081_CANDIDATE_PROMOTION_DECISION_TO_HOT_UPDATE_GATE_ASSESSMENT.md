# V4-081 Candidate Promotion Decision To Hot-Update Gate Assessment

## Facts

- This is a docs-only assessment slice after V4-080.
- V4-080 added `CandidatePromotionDecisionRecord` under `runtime_packs/candidate_promotion_decisions/<promotion_decision_id>.json`.
- The deterministic promotion decision ID is `candidate-promotion-decision-<result_id>`.
- The only current candidate promotion decision is `selected_for_promotion`, and the record is created only from `promotion_eligibility.state == "eligible"`.
- Existing hot-update gate creation is already implemented, but it is candidate-pack driven:

```go
EnsureHotUpdateGateRecordFromCandidate(root, hotUpdateID, candidatePackID, createdBy, requestedAt)
```

- Existing direct operator gate creation is:

```text
HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>
```

- Existing gate creation derives:
  - `previous_active_pack_id` from the current active runtime-pack pointer
  - `rollback_target_pack_id` from the candidate `RuntimePackRecord`
  - `target_surfaces`, `surface_classes`, `reload_mode`, and `compatibility_contract_ref` from the candidate pack
- Existing gate creation does not load a `CandidatePromotionDecisionRecord`, `CandidateResultRecord`, `ImprovementRunRecord`, `EvalSuiteRecord`, or `PromotionPolicyRecord`.
- Existing `HotUpdateGateRecord` has no fields for `promotion_decision_id`, `result_id`, `run_id`, `candidate_id`, `eval_suite_id`, or `promotion_policy_id`.
- Existing `HotUpdateOutcomeRecord` and `PromotionRecord` can carry candidate/run/result refs, but they are downstream of gate execution and successful terminal outcome creation.
- Existing deploy-lock / unsafe-live-job blocking is not implemented as a concrete helper surface. The frozen spec requires hot-update execution to respect quiesce/deploy-lock rules, but current gate creation only validates pack linkage and active pointer presence.

## Analysis

`CandidatePromotionDecisionRecord` is necessary authority for decision-driven gate creation, but it is not sufficient authority by itself. A future helper must treat the decision ID as an entrypoint, then reload and cross-check the committed record chain before writing a gate.

Required source records:

- committed `CandidatePromotionDecisionRecord`
- linked `CandidateResultRecord`
- linked `ImprovementRunRecord`
- linked `ImprovementCandidateRecord`
- linked frozen `EvalSuiteRecord`
- referenced `PromotionPolicyRecord`
- candidate `RuntimePackRecord`
- baseline `RuntimePackRecord`
- current active runtime-pack pointer
- current last-known-good runtime-pack pointer when present, as rollback/readiness evidence

The helper should re-run or reuse the same fail-closed eligibility authority that V4-080 used. A stale decision must not become a gate if the linked result no longer derives `eligible`, if any copied linkage differs, or if the candidate pack can no longer load.

The hot-update ID should be deterministic from the durable decision:

```text
hot-update-<promotion_decision_id>
```

For example:

```text
hot-update-candidate-promotion-decision-result-eligible
```

The candidate pack for the gate should be `CandidatePromotionDecisionRecord.candidate_pack_id`, cross-checked against the linked candidate, run, result, and eval suite. The previous active pack should be the current active runtime-pack pointer's `active_pack_id`, but gate creation should fail closed unless that active pack still equals the decision baseline pack. If the active pack has moved since the decision, the promotion evidence is stale for direct gate creation.

The rollback target should preserve existing gate semantics: use the candidate runtime pack's `rollback_target_pack_id`, require it to load, and require it to be consistent with the decision baseline/current active pack unless a later explicit rollback policy broadens this. If a last-known-good pointer exists, V4-082 should load it and fail closed on unrelated rollback evidence rather than silently inventing a rollback basis.

V4-082 should create a hot-update gate directly from a candidate promotion decision through a missioncontrol helper first. It should not add a new hot-update gate proposal record first because `CandidatePromotionDecisionRecord` is already the durable upstream selection/proposal artifact. It should not require a direct operator command first because the repo pattern is helper first, then TaskState/direct command in a later slice. It should not require canary or owner-approval surfaces first because V4-080 decisions are only created for `eligible` results; canary-required and owner-approval-required results are rejected before decision creation.

Deploy-lock / unsafe-live-job blocking should not be a prerequisite for V4-082 gate creation because the new helper should only create a prepared gate and must not mutate the active pointer. Deploy-lock and quiesce checks should be required before pointer switch, phase advancement into execution-sensitive states, or any future autonomous apply path. V4-082 should still document that gate creation is not permission to execute.

## Decision

Recommended V4-082 slice:

**V4-082 - Create Hot-Update Gate From Candidate Promotion Decision Helper**

Add one missioncontrol helper, likely:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

The prompt-suggested shape included `jobID` and caller-supplied `hotUpdateID`, but live repo conventions argue against both at the missioncontrol layer. Existing missioncontrol helpers do not take `jobID`; TaskState wrappers enforce active-job matching later. The hot-update ID should be deterministic from the promotion decision, not operator supplied.

The helper should:

- load the committed candidate promotion decision
- require `decision == selected_for_promotion`
- require `eligibility_state == eligible`
- load the linked candidate result and re-check all decision fields against it
- derive current promotion eligibility and require it still matches the decision
- load the linked run, candidate, eval suite, promotion policy, baseline pack, and candidate pack
- require the active runtime-pack pointer to exist
- require `active_pointer.active_pack_id == baseline_pack_id`
- derive `hot_update_id = hot-update-<promotion_decision_id>`
- derive the gate candidate pack from `candidate_pack_id`
- derive the rollback target from the candidate pack's `rollback_target_pack_id`
- require the rollback target to load and be compatible with the baseline/current active pack
- copy gate fields from the candidate pack using existing gate derivation semantics
- create a prepared `HotUpdateGateRecord` with `decision=keep_staged`
- make exact replay idempotent
- fail closed on divergent duplicate hot-update IDs
- fail closed if another gate already references the same decision-derived candidate/result authority in an incompatible way

The helper should not mutate active runtime-pack pointers, last-known-good pointers, `reload_generation`, candidate/result/run/policy records, outcomes, promotions, rollback records, or LKG records.

## Required Answers

Is `CandidatePromotionDecisionRecord` sufficient authority to create a hot-update gate?

No, not alone. It is the durable entrypoint. The helper must reload and cross-check the decision's linked result, run, candidate, eval suite, promotion policy, baseline pack, candidate pack, current active pointer, and rollback evidence.

What source records must be loaded and cross-checked?

Load the promotion decision, candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, active runtime-pack pointer, and current last-known-good pointer when present.

What hot-update ID should be deterministic?

`hot-update-<promotion_decision_id>`.

What candidate pack should become the hot-update candidate pack?

`CandidatePromotionDecisionRecord.candidate_pack_id`, after cross-checking it against the candidate result, improvement run, improvement candidate, eval suite, and loaded candidate runtime pack.

What previous active pack / rollback target should be used?

Use the current active pointer's `active_pack_id` as `previous_active_pack_id`, and fail closed unless it still equals the decision baseline pack. Use the candidate runtime pack's `rollback_target_pack_id` as `rollback_target_pack_id`, requiring it to load and remain compatible with the baseline/current-active rollback chain.

Should gate creation be a missioncontrol helper first, or a direct operator command first?

Missioncontrol helper first. A TaskState wrapper and direct operator command can follow after the storage semantics are proven.

Should deploy-lock / unsafe-live-job blocking be implemented before gate creation?

No. V4-082 should create only a prepared gate and must not execute or switch pointers. Deploy-lock / unsafe-live-job blocking belongs before execution-sensitive transition or pointer switch work.

Should the helper mutate active runtime-pack pointers?

No.

Should it create `HotUpdateOutcomeRecord`, `PromotionRecord`, rollback records, or LKG records?

No.

## Required V4-082 Tests

V4-082 should require focused missioncontrol tests for:

- eligible promotion decision creates a prepared hot-update gate
- deterministic `hot_update_id == hot-update-<promotion_decision_id>`
- gate `candidate_pack_id` comes from the decision candidate pack
- gate `previous_active_pack_id` comes from the current active pointer
- stale active pointer, where active pack no longer equals decision baseline, fails closed
- rollback target missing from candidate pack fails closed
- rollback target missing as a runtime pack fails closed
- mismatched decision/result/run/candidate/eval-suite/policy linkage fails closed
- derived promotion eligibility changing away from `eligible` fails closed
- exact replay returns `changed=false` and is byte-stable
- divergent duplicate hot-update gate fails closed
- existing gate for the same deterministic hot-update ID with a different candidate pack fails closed
- helper does not mutate candidate promotion decision, candidate result, run, candidate, eval suite, promotion policy, runtime pack records, active pointer, last-known-good pointer, or `reload_generation`
- helper does not create hot-update outcomes, promotions, rollback records, rollback apply records, or LKG records

V4-082 should not add direct operator commands, TaskState wrappers, deploy-lock implementation, canary execution, owner approval, gate phase advancement, pointer switch, reload/apply, outcome creation, promotion creation, rollback creation, or LKG recertification.

## Invariants Preserved

This V4-081 assessment does not change Go code, tests, commands, TaskState wrappers, candidate promotion decisions, candidate results, improvement runs, eval suites, promotion policies, runtime packs, hot-update gates, hot-update outcomes, promotions, rollbacks, active runtime-pack pointer, last-known-good pointer, `reload_generation`, canary evidence, owner approval state, deploy-lock state, eval execution, candidate scoring, or V4-082 implementation.
