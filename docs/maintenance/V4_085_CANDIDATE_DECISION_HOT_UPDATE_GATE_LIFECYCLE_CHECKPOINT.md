# V4-085 Candidate Decision Hot-Update Gate Lifecycle Checkpoint

## Facts

V4 now has a controlled path from V4 job admission metadata through an operator-created prepared hot-update gate:

- V4-065 added execution-plane, execution-host, and mission-family metadata. V4 jobs now distinguish `live_runtime`, `improvement_workspace`, and `hot_update_gate`.
- V4-066 added the V4 rejection code skeleton, including codes for execution-plane admission, immutable evals, mutation scope, topology, promotion policy, canary, approval, deploy lock, quiesce, rollback, and active-pack mutation failures.
- V4-067 admitted improvement-family jobs only in `improvement_workspace` on compatible hosts.
- V4-068 added target, mutable, and immutable surface declarations for improvement-family jobs.
- V4-069 constrained source-patch work to `source_patch_artifact` admission instead of direct runtime-source mutation.
- V4-070 made topology improvement disabled by default unless `topology_mode_enabled=true`.
- V4-071 added the durable `PromotionPolicyRecord` registry.
- V4-072 required improvement-family jobs to reference a promotion policy and optionally validated that policy against the mission store.
- V4-073 added required baseline, train, and holdout evidence refs for improvement-family jobs, with train/holdout separation.
- V4-074 hardened eval-suite immutability by making `eval_suite_id` writes idempotent only for exact replay and fail-closed for divergent duplicates.
- V4-075 hardened improvement-run linkage to frozen eval suites, candidates, baseline packs, and candidate packs.
- V4-076 added `promotion_policy_id` to candidate results and kept candidate result linkage store-aware.
- V4-078 added `EvaluateCandidateResultPromotionEligibility`, deriving promotion eligibility from committed result and policy records.
- V4-080 added `CandidatePromotionDecisionRecord` and `CreateCandidatePromotionDecisionFromEligibleResult`, creating a durable decision only for derived `eligible` candidate results.
- V4-082 added `CreateHotUpdateGateFromCandidatePromotionDecision`, creating a prepared `HotUpdateGateRecord` from a committed selected decision after rechecking the full source chain and current active pointer.
- V4-084 exposed that helper through `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>` with TaskState job validation, operator actor derivation, audit action `hot_update_gate_from_decision`, and byte-stable idempotent replay.

## Complete

The core controlled self-improvement-to-prepared-hot-update-gate path is complete.

A V4 improvement candidate can now be represented through admitted improvement-family metadata, immutable eval-suite/run/result/policy records, derived eligibility, a durable selected promotion decision, and a prepared hot-update gate created by direct operator command.

The gate creation path is governed and fail-closed:

- it does not accept caller-supplied candidate/run/result/policy authority
- it re-derives current promotion eligibility
- it requires the active runtime-pack pointer to still match the decision baseline pack
- it derives deterministic hot-update ID `hot-update-<promotion_decision_id>`
- it creates only a prepared gate
- it does not mutate active pack, last-known-good, or reload generation
- it does not create outcomes, promotions, rollback records, rollback-apply records, canary records, approval records, or LKG records

## Still Deferred

Automatic execution or pointer switch remains blocked by core safety work, not by the decision-to-gate path itself.

The main blockers are:

- deploy-lock and unsafe-live-job enforcement before execution-sensitive hot-update transitions
- explicit quiesce detection before applying or switching packs
- policy branches for `canary_required`, `owner_approval_required`, and `canary_and_owner_approval_required`
- canary proposal/evidence records
- owner approval request/proposal records for promotion and hot-update policy
- final V4 handover once the remaining core gates are either implemented or explicitly deferred

Existing hot-update phase, pointer switch, reload/apply, outcome, promotion, rollback, and LKG machinery exists, but V4-085 does not certify it as automatically safe for the new candidate-decision path without deploy-lock/quiesce enforcement.

## Core Versus Later Polish

V4 core:

- execution-plane and family admission boundaries
- mutation-surface declaration
- immutable eval-suite/run/result/policy linkage
- promotion eligibility derived from committed records
- durable candidate promotion decision
- prepared hot-update gate from that decision
- operator-visible control entry
- deploy-lock and quiesce enforcement before pointer switch or reload/apply
- canary and owner-approval gates when policy requires them

Later polish:

- richer human-facing status text for every intermediate record
- broader policy grammar beyond the narrow supported evaluator rules
- prompt-pack, skill-pack, manifest, and extension registries beyond current runtime-pack refs
- deeper candidate/result/gate backrefs in hot-update gate schema
- topology mutation implementation
- source-patch application/deployment automation
- final UX smoothing for command aliases

## Next Track Decision

The next safest implementation track is deploy-lock / unsafe-live-job / quiesce enforcement.

Reasoning:

- The prepared gate path is now complete, so the next real safety boundary is execution.
- The frozen spec explicitly rejects silent update during active unsafe live work.
- Deploy-lock/quiesce applies to every hot-update execution path, while canary and owner approval apply only to policy-selected branches.
- Candidate decisions currently require derived state `eligible`; policies requiring canary or owner approval are intentionally not converted into selected decisions yet.
- Adding canary or approval records before deploy-lock/quiesce would create more proposal surface while leaving the universal execution blocker unresolved.
- A final V4 handover before deploy-lock/quiesce would prematurely treat a core safety invariant as deferred polish.

## Recommended V4-086 Slice

Recommend exactly one next slice:

```text
V4-086 — Hot-Update Deploy-Lock And Quiesce Enforcement Assessment
```

Scope should be docs-only assessment unless live code reveals an already obvious tiny helper. It should inspect:

- existing active/persisted job runtime state
- runtime control context
- hot-update gate phase, pointer switch, and reload/apply helpers
- existing rejection codes `E_ACTIVE_JOB_DEPLOY_LOCK` and `E_RELOAD_QUIESCE_FAILED`
- where deploy-lock/quiesce should be checked: phase advance, pointer switch, reload/apply, direct command wrapper, or shared missioncontrol helper
- whether checks should block all execution-sensitive transitions or only pointer switch/reload
- how to preserve idempotent replay
- how to avoid mutating active runtime-pack pointers during assessment

The recommended V4-086 output should pick one smallest implementation slice for deploy-lock/quiesce enforcement and list exact tests.

## Must Not Widen

The next work must not silently widen into:

- canary execution
- owner approval request creation
- promotion creation
- outcome creation
- rollback creation
- rollback-apply creation
- LKG mutation
- deploy execution during unsafe live work
- active pointer mutation outside existing governed pointer-switch helpers
- reload/apply behavior changes not required for deploy-lock/quiesce assessment
- topology mutation
- source-patch deployment
- V4 handover claims before core blockers are resolved or explicitly deferred

## Checkpoint Answer

The controlled path from a committed eligible candidate result to a prepared hot-update gate is complete. What remains before automatic execution or pointer switch is safe is the runtime safety boundary: deploy-lock, unsafe-live-job, and quiesce enforcement first; canary and owner-approval policy branches after that; final handover only after those core decisions are grounded.
