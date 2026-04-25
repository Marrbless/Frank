# V4-086 Hot-Update Deploy-Lock And Quiesce Enforcement Assessment

## Facts

V4-085 completed the controlled path from an eligible candidate result to a durable candidate promotion decision and then to a prepared hot-update gate through `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`.

The current hot-update execution surface is broader than the candidate-decision gate creation path:

- `CreateHotUpdateGateFromCandidatePromotionDecision` creates a prepared gate only.
- `AdvanceHotUpdateGatePhase` advances metadata states through `prepared`, `validated`, and `staged`.
- `ExecuteHotUpdateGatePointerSwitch` mutates the active runtime-pack pointer, sets `previous_active_pack_id`, increments `reload_generation`, and moves the gate to `reloading`.
- `ExecuteHotUpdateGateReloadApply` performs reload/apply convergence from `reloading` or `reload_apply_recovery_needed` and records succeeded, failed, or recovery-needed gate state.
- `ResolveHotUpdateGateTerminalFailure` records terminal failure from recovery-needed state.
- `CreateHotUpdateOutcomeFromTerminalGate` creates an outcome ledger record from a terminal gate.
- `CreatePromotionFromSuccessfulHotUpdateOutcome` creates a promotion ledger record from a successful outcome.
- `RecertifyLastKnownGoodFromPromotion` mutates the last-known-good pointer after a successful promotion.

The spec requires hot update apply only at a valid quiesce point: no tool call executing, no governed external side effect in flight, the active step between atomic actions, durable state persisted, active live job safety policy permitting reload, and rollback target available. It also defines rejection codes `E_ACTIVE_JOB_DEPLOY_LOCK` and `E_RELOAD_QUIESCE_FAILED`.

The repo already has those rejection code constants, but no current helper enforces them on hot-update pointer switch or reload/apply.

## Current Runtime Surface

The current durable runtime state can identify active job occupancy, but it does not yet identify an unsafe live job or a valid quiesce point.

Existing durable surfaces:

- `ActiveJobRecord` records the globally active job, its state, active step, attempt, lease holder, lease expiry, and activation sequence.
- `HoldsGlobalActiveJobOccupancy` treats `running`, `waiting_user`, and `paused` as globally occupied states.
- `JobRuntimeRecord` records execution plane, execution host, mission family, active step, waiting/paused reasons, timestamps, and V4 job metadata.
- `RuntimeControlRecord` records the active step, execution plane, execution host, mission family, allowed tools, authority, and declared surfaces.
- `ExecutionContext` and the existing guard path can evaluate tool authority and governed external targets for a specific tool call.

Missing durable surfaces:

- no explicit deploy-lock record
- no explicit unsafe-live-job bit
- no durable "tool call in flight" marker
- no durable "governed external side effect in flight" marker
- no durable quiesce readiness record
- no helper that says an active live job is safe for hot reload
- no read model combining active job, runtime record, control record, active step, and quiesce status for hot-update execution

Therefore the repo can currently detect "there is an active occupied job", but it cannot distinguish safe paused/quiesced live work from unsafe live work with enough precision to enforce the spec without first defining the read model.

## Transition Classification

Prepared gate creation should not be blocked by deploy-lock or quiesce checks.

It is metadata creation. It validates promotion authority, active baseline consistency, and rollback target existence, but it does not switch active packs, run reload/apply, execute canaries, create outcomes, create promotions, mutate last-known-good, or change `reload_generation`.

Phase advancement to `validated` or `staged` should not be the first enforcement point.

The current phase helper writes gate metadata only. A staged gate is closer to execution, but staging does not mutate runtime pointers or apply code. Blocking staging before a durable quiesce surface exists would make metadata bookkeeping depend on transient runtime conditions and would not protect the actual pointer-switch/reload paths unless those paths are also guarded.

Phase advancement to `executing` or `executed` is not a current live helper surface. The live phase helper only accepts `prepared`, `validated`, and `staged`.

Pointer switch is execution-sensitive and must be guarded before first mutation.

It changes the active runtime-pack pointer and increments `reload_generation`. This is the first existing transition that can make the candidate pack active for the runtime. It should require the deploy-lock/quiesce guard before performing a new switch.

Reload/apply is execution-sensitive and must be guarded before first attempt and before retry from recovery-needed.

It performs convergence while the candidate pack is active. A retry from `reload_apply_recovery_needed` is a new execution attempt, not pure metadata replay, so it must be blocked when active live work is unsafe or quiesce cannot be established.

Terminal failure is a recovery metadata operation and should not be blocked.

An operator must be able to resolve recovery-needed state to terminal failure even when a deploy lock exists. Blocking failure resolution would make it harder to stop unsafe rollout state.

Outcome creation is a ledger operation and should not be blocked.

It derives an immutable outcome from a terminal gate. It does not execute reload/apply or mutate active pack state.

Promotion creation is a ledger operation and should not be blocked by deploy-lock/quiesce.

It derives a promotion from a successful hot-update outcome. It should still preserve its existing source-record checks, but deploy-lock/quiesce is about execution safety, not post-terminal ledger creation.

LKG recertification mutates the last-known-good pointer, but it is not a hot-update execution transition.

It should not be the first deploy-lock/quiesce enforcement point. If later policy wants a separate lock for LKG mutation, that should be a separate slice, not bundled into hot-update execution safety.

## Enforcement Placement

Direct command parsing should not own deploy-lock or quiesce semantics.

The direct command layer currently parses arguments, calls a TaskState method, and returns an empty response plus error on failure. It should keep that role. Once enforcement exists, direct commands should surface `E_ACTIVE_JOB_DEPLOY_LOCK` and `E_RELOAD_QUIESCE_FAILED` the same way other hot-update wrapper failures are surfaced: empty response with the returned error.

TaskState is the first practical enforcement layer for operator commands.

The TaskState wrappers already validate active or persisted job context, resolve the mission store root, derive timestamps, emit audit actions, and call missioncontrol helpers. They also have access to the active execution context or persisted runtime control record. That makes TaskState the right layer to apply the guard before `ExecuteHotUpdateGatePointerSwitch` and `ExecuteHotUpdateGateReloadApply` in the direct command path.

Missioncontrol should own a shared read-only guard helper before command enforcement is added.

The actual deploy-lock/quiesce decision should not be duplicated in multiple TaskState methods. A missioncontrol read-only helper should assemble or consume the active job/runtime/control state and return a typed assessment with:

- whether a first execution mutation is allowed
- whether the block is `E_ACTIVE_JOB_DEPLOY_LOCK` or `E_RELOAD_QUIESCE_FAILED`
- the active job id, state, execution plane, mission family, and active step considered
- whether a quiesce surface was absent, unknown, or explicitly not ready
- whether the transition is an idempotent replay that should be allowed

The helper must not write records, advance gates, mutate pointers, run reload/apply, create outcomes, create promotions, create rollback records, or mutate LKG.

Missioncontrol execution helpers may later call the same guard if their signatures are expanded or if a durable root-only guard is introduced. V4-087 should not force that broader storage/API change before the read model is defined.

## Replay Semantics

Idempotent replay must be evaluated before applying a new lock failure to an already-completed transition.

For pointer switch:

- if the gate is already `reloading` and the active pointer already references the deterministic hot update and candidate pack, replay should continue to return `changed=false`
- a deploy lock that appears after the pointer switch must not make exact replay fail
- a new pointer switch from `staged` must be blocked when the guard says active live work is unsafe or quiesce is unavailable

For reload/apply:

- if the gate is already `reload_apply_succeeded`, replay should continue to return `changed=false`
- a deploy lock that appears after success must not make exact replay fail
- a retry from `reload_apply_recovery_needed` is a new execution attempt and must be guarded

For terminal failure, outcome creation, promotion creation, and LKG recertification, deploy-lock/quiesce should not change current idempotent replay behavior in V4-087.

## V4-087 Recommendation

Recommend exactly one next slice:

```text
V4-087 - Hot-Update Execution Readiness Guard Skeleton
```

Scope:

Add a read-only missioncontrol guard/read-model helper and focused tests. Do not yet wire it into direct command enforcement.

Rationale:

- The live repo lacks a concrete deploy-lock or quiesce readiness surface.
- Enforcing now would require inventing implicit semantics from coarse active-job occupancy alone.
- A read-only guard skeleton creates one shared place to encode conservative default behavior, replay exceptions, and typed rejection codes before pointer-switch/reload wrappers depend on it.
- This is the smallest implementation that advances deploy-lock/quiesce safely without widening into canary, approval, promotion, rollback, or pointer mutation work.

Suggested helper shape, subject to live naming conventions:

```go
AssessHotUpdateExecutionReadiness(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReadinessAssessment, error)
```

The input should include the transition being assessed, the hot-update id, the command job id, current/persisted runtime context when available, and the timestamp only if needed for stale lease/quiesce calculations. The assessment should be read-only and serializable enough for tests to assert reason codes.

Initial conservative semantics:

- prepared gate creation is outside the guard
- `validated` and `staged` phase advancement is outside the guard
- first pointer switch from `staged` requires readiness
- first reload/apply from `reloading` requires readiness
- reload/apply retry from `reload_apply_recovery_needed` requires readiness
- exact pointer-switch replay after `reloading` is allowed
- exact reload/apply replay after success is allowed
- absent active occupied job is not deploy-locked
- active occupied hot-update control job for the same command is not itself an unsafe live job
- active occupied live-runtime job with no explicit quiesce/readiness proof is assessed as not ready or deploy-locked
- absent quiesce surface is reported distinctly from an explicit failed quiesce once such a surface exists

V4-088 should then wire the guard into `TaskState.ExecuteHotUpdateGatePointerSwitch` and `TaskState.ExecuteHotUpdateGateReloadApply`, preserving direct command empty-response-on-error behavior.

## Required V4-087 Tests

The guard skeleton should have missioncontrol tests proving:

- it is read-only and leaves hot-update gates, active runtime-pack pointer, last-known-good pointer, runtime records, and `reload_generation` unchanged
- prepared gate creation is classified as non-execution-sensitive
- phase advancement to `validated` and `staged` is classified as non-execution-sensitive
- pointer switch from `staged` is execution-sensitive
- reload/apply from `reloading` is execution-sensitive
- reload/apply retry from `reload_apply_recovery_needed` is execution-sensitive
- terminal failure resolution is classified as metadata/recovery and not blocked by deploy-lock/quiesce
- outcome creation is classified as ledger-only and not blocked by deploy-lock/quiesce
- promotion creation is classified as ledger-only and not blocked by deploy-lock/quiesce
- LKG recertification is outside hot-update execution readiness enforcement
- absent active occupied job allows execution readiness to pass
- active occupied same hot-update control job does not count as unsafe live work by itself
- active occupied live-runtime job without an explicit quiesce surface returns a blocking assessment
- blocking assessment carries `E_ACTIVE_JOB_DEPLOY_LOCK` when unsafe live work is the reason
- blocking assessment carries `E_RELOAD_QUIESCE_FAILED` when quiesce is the reason
- exact pointer-switch replay after the pointer already switched is allowed even if an active occupied job appears later
- exact reload/apply replay after success is allowed even if an active occupied job appears later
- reload/apply retry from recovery-needed is blocked when readiness is not proven

Before any pointer-switch or reload path is considered safe for autonomous use, later enforcement tests must also prove the direct command wrappers return an empty response plus error on deploy-lock/quiesce failures and preserve idempotent replay responses after already-applied transitions.

## Deferred

V4-086 does not recommend implementing any of these in V4-087:

- blocking prepared gate creation
- wiring direct command enforcement before the guard/read-model skeleton exists
- changing missioncontrol storage behavior beyond read-only guard types
- creating deploy-lock records
- creating quiesce records
- mutating the active runtime-pack pointer
- mutating the last-known-good pointer
- mutating `reload_generation`
- creating hot-update gates, outcomes, promotions, rollback records, rollback-apply records, canary records, approval records, or LKG records
- executing canaries
- requesting owner approval
- changing pointer-switch or reload/apply behavior
- implementing V4 handover
