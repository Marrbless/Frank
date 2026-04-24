# V4-064 Frank V4 Frozen Spec Gap Assessment

## Facts

- Live branch for this slice: `frank-v4-064-v4-spec-gap-assessment`.
- Starting point was `926bc59c3f6916c6a7d68d21fe893c89aa21387c`, tagged `frank-v4-063-hot-update-operator-runbook`.
- Baseline `/usr/local/go/bin/go test -count=1 ./...` passed before this assessment branch was created.
- Frozen spec read: `docs/FRANK_V4_SPEC.md`.
- This slice is docs-only. It does not change Go code, tests, commands, runtime pointers, hot-update gates, outcomes, promotions, rollbacks, candidates, eval suites, or improvement runs.

## Assessment Summary

The V4 hot-update lane is now a substantial implemented subsection, but it is not the whole Frank V4 frozen spec. Current code has meaningful foundations for runtime pack records, active and last-known-good pointers, improvement candidates, eval suites, improvement runs, candidate scorecards, hot-update gate operation, hot-update outcomes, hot-update promotions, LKG recertification, rollback ledgers, and operator status/readout surfaces.

The largest remaining gap is controlled self-improvement enforcement. The frozen spec requires every job/proposal to declare and enforce `execution_plane`, `execution_host`, `mission_family`, and target surfaces before improvement-family work, runtime-source boundaries, topology changes, policy gates, canary requirements, and hot-update calls can be safely composed. Current `Job` and `Step` records do not carry those V4 execution-plane fields. `ImprovementRunRecord` carries similar fields, but that is too late in the lifecycle to enforce the global job/proposal boundary.

## Gap Matrix

| Spec area | Requirement summary | Current repo evidence | Status | Suggested next slice |
| --- | --- | --- | --- | --- |
| Execution plane requirement on jobs | Every job must declare `execution_plane`; missing or incompatible planes reject. | `internal/missioncontrol/types.go` `Job` has `SpecVersion`, `State`, `MaxAuthority`, `AllowedTools`, and `Plan`, but no execution-plane fields. `ValidatePlan` in `internal/missioncontrol/validate.go` does not check execution plane. | missing | V4-065: add execution-plane field/storage/read-model/validation skeleton. |
| Execution host requirement on jobs | Every job must declare `execution_host`; phone is valid for improvement families only inside `improvement_workspace`. | `Job` lacks `execution_host`; `ImprovementRunRecord` has `ExecutionHost`, but that does not protect job admission. | missing | Include host enum/validation in V4-065 skeleton. |
| Mission family requirement on jobs | Every job must declare `mission_family`; live, improvement, and hot-update families must be routed to compatible planes. | `Job` lacks `mission_family`. `ImprovementRunRecord` has `MissionFamily`, but no global job-level enforcement exists. | missing | Include mission-family field and compatibility validation in V4-065 skeleton. |
| Improvement-family desktop/lab enforcement | Frozen spec no longer requires desktop-only lab. It requires improvement-family work to run in `improvement_workspace`; `execution_host=phone` is valid there. | No job-level plane enforcement. `ImprovementRunRecord` stores `ExecutionPlane`, `ExecutionHost`, and `MissionFamily`, and requires them non-empty. | partial | V4-065 should enforce `improvement_workspace` for improvement-family jobs before any lab implementation. |
| Mission proposal object fields | Proposal/control objects must carry plane, host, family, target surfaces, and hot-update permission intent. | Existing `Job`/`Step` plan schema has authority, tools, capability refs, external targets, and system action metadata, but lacks V4 proposal fields. | missing | Start with execution-plane fields, then add target-surface/hot-update permission skeleton later. |
| Runtime pack registry | Runtime packs must record pack identity, channel, pack refs, mutable/immutable surfaces, surface classes, compatibility contract, and rollback target. | `RuntimePackRecord` in `runtime_pack_registry.go` includes channel, prompt/skill/manifest/extension/policy refs, surfaces, compatibility contract, and optional rollback target. It lacks an explicit `created_by`. | partial | Later polish pack schema only after job-plane enforcement. |
| Active runtime pack pointer | Active channel must be explicit and controlled. | `ActiveRuntimePackPointer` and `StoreActiveRuntimePackPointerPath` exist; hot-update pointer switch mutates active pointer only through `ExecuteHotUpdateGatePointerSwitch`. | complete | No immediate V4-065 work. |
| Last-known-good pointer | LKG channel must be explicit and recertifiable. | `LastKnownGoodRuntimePackPointer` exists; V4-058/V4-060 helper/control recertifies from promotion with guarded active pointer and current LKG checks. | complete | Deferred polish around status metadata consistency only. |
| Candidate channel | Candidate packs live in improvement workspace until staged, hot-updated, promoted, rejected, or discarded. | `ImprovementCandidateRecord` and `CandidateRuntimePackPointer` exist; candidate pointer is linked to hot-update gate. No complete candidate lifecycle policy exists. | partial | Later candidate lifecycle/policy slice. |
| Canary channel | Canary is required when policy requires it and must be represented separately. | `HotUpdateGateRecord` has `CanaryRef`, `HotUpdateGateStateCanarying`, and `HotUpdateOutcomeKindCanaryApplied`, but no canary policy/evidence enforcement. | partial | Later canary policy/evidence slice after plane enforcement. |
| Prompt pack concept | Runtime pack must compose prompt-pack refs and treat prompt updates as governed pack content. | `RuntimePackRecord.PromptPackRef` is required, but no first-class prompt-pack registry was found. | partial | Later prompt-pack registry or artifact contract slice. |
| Skill pack concept | Runtime pack must compose skill-pack refs and protect topology-modifying skill changes. | `RuntimePackRecord.SkillPackRef` is required, but no first-class skill-pack registry or topology gate was found. | partial | Later topology/skill-pack guard slice. |
| Eval suite immutability | Eval suite, evaluator, rubric, train corpus, holdout corpus, and policy refs must be immutable for the run. | `EvalSuiteRecord` requires rubric/train/holdout/evaluator refs and `FrozenForRun=true`. Records are stored as JSON, but `StoreEvalSuiteRecord` overwrites rather than fail-closing divergent duplicate writes. Promotion policy ref is not present. | partial | Later immutable eval write/idempotence hardening slice. |
| Improvement run ledger | Improvement runs must track baseline, mutation, train, holdout, candidate, hot-update, promotion, rejection, rollback states. | `ImprovementRunRecord` has V4-like states and required baseline/candidate/eval refs. Store rejects divergent same-id rewrites. It does not drive execution or enforce plane/family globally. | partial | Defer until after job-plane skeleton. |
| Candidate result / scorecard | Candidate result must capture baseline/train/holdout/resource/compatibility/regression data and decision. | `CandidateResultRecord` has baseline/train/holdout, complexity, compatibility, resource scores, regression flags, decision, notes, and linkage checks. | partial | Later promotion policy evaluation slice. |
| Promotion policy | Promotion requires deterministic policy checks, including holdout and canary when required. | Promotion records/helpers exist for hot-update success, but no promotion policy object or evaluator is present. | missing | Later policy registry/evaluator slice. |
| Baseline-before-mutation rule | A baseline must be captured before candidate mutation. | Candidate, eval, run, and result records carry `BaselinePackID`; linkage checks ensure consistency. No job/proposal-level admission rule enforces baseline before mutation starts. | partial | Later improvement-run admission/control slice after execution-plane skeleton. |
| Train vs holdout separation | Train and holdout evidence must remain separated and holdout must not be used during mutation. | `EvalSuiteRecord` has separate `TrainCorpusRef` and `HoldoutCorpusRef`; `CandidateResultRecord` has separate train and holdout scores. No execution guard prevents holdout access during mutation. | partial | Later eval access-policy slice. |
| Canary requirement when policy requires it | If policy requires canary, promotion/hot-update rejects without canary evidence. | Canary fields and enums exist, but no policy-required canary enforcement was found. | missing | Later canary policy guard slice. |
| Promotion requires rollback target | Promotion/hot-update must have a rollback target. | `RuntimePackRecord` has optional `RollbackTargetPackID`; `HotUpdateGateRecord` requires `RollbackTargetPackID` and validates it. `PromotionRecord` stores previous active pack. | partial | Later promotion-policy guard can require rollback targets before outcome/promotion creation outside hot-update helper assumptions. |
| Rollback determinism | Rollback must target known prior/LKG packs and be deterministic. | `RollbackRecord` validates from/target packs and links to promotion/gate/outcome. `rollback_apply_registry.go` and rollback direct commands exist from prior V4 slices. | partial | Later rollback integration polish after controlled self-improvement flow. |
| Append-only improvement ledger | Improvement events should be append-only with deterministic idempotence and divergent duplicate rejection. | Many registries are immutable-by-id or divergent-rejecting (`ImprovementRunRecord`, `CandidateResultRecord`, outcomes, promotions, rollbacks). Some stores, such as runtime pack and eval suite records, still use direct atomic overwrite semantics. No single append-only improvement event ledger exists. | partial | Later append-only event ledger slice if required after field enforcement. |
| Rejection codes | V4 requires machine-readable codes such as `E_EXECUTION_PLANE_REQUIRED`, `E_IMPROVEMENT_WORKSPACE_REQUIRED`, `E_HOT_UPDATE_GATE_REQUIRED`, `E_BASELINE_REQUIRED`, `E_HOLDOUT_REQUIRED`, `E_CANARY_REQUIRED`, `E_TOPOLOGY_CHANGE_DISABLED`, and `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`. | Existing rejection codes in `validate.go`, `runtime.go`, and tests are mostly V2/V3-style strings such as `invalid_runtime_state`, `invalid_step_type`, and capability/campaign codes. V4 `E_*` constants are not implemented. | missing | Rejection-code skeleton is a candidate, but should follow execution-plane skeleton so codes attach to real validation. |
| No live phone self-editing | Live runtime must not mutate itself; improvement work must be isolated. | There is no job-level `execution_plane` or live-vs-workspace guard. Hot-update pointer mutation is controlled, but generic live self-editing is not enforced by V4 plane fields. | missing | V4-065 field/validation skeleton. |
| No runtime-source autonomous deployment | Runtime-source changes must not auto-apply/deploy. | No dedicated runtime-source mutation policy was found. Existing hot-update lane handles runtime packs, not source patch deployment policy. | missing | Later source-patch policy slice after plane/family skeleton. |
| Source patch proposal artifact-only rule | `propose_source_patch` may generate artifact proposals only, not apply them. | No `propose_source_patch` command or policy surface found in current code search. | missing | Later source-patch artifact-only validator. |
| Topology mode disabled by default | Add/remove/split/merge skill-pack operations must reject unless topology mode is explicitly enabled. | No topology-mode flag or topology rejection code found. | missing | Later topology mode disabled-by-default guard. |
| Deploy lock / unsafe live job promotion blocking | Hot update should block promotion when unsafe live jobs are active or deploy lock cannot be acquired. | Store writer lease exists in maintenance docs, but no hot-update deploy-lock/unsafe-live-job blocker was found. Hot-update pointer switch validates pack linkage and gate state. | missing | Later deploy-lock/readiness guard. |
| Operator inspect/status for packs | Operators must inspect active, candidate, LKG, pack identity, and gate state. | `status.go` exposes `runtime_pack_identity`, `improvement_candidate_identity`, `hot_update_gate_identity`, outcomes, promotions, rollbacks, and LKG. | partial | Later read-model polish for pack channels and LKG metadata. |
| Operator inspect/status for improvement runs | Operators must inspect improvement runs and candidate scorecards. | `status.go` exposes `improvement_run_identity` and `candidate_result_identity`, but improvement run status omits execution plane, host, family, objective, target type/ref, surface class, state, decision, and stop reason even though the record stores them. | partial | Later read-model polish after V4-065. |
| Hot-update gate storage/read-model/control | Gate creation, status, phase progression, pointer switch, reload/apply, recovery, retry, terminal failure, outcome, promotion, and LKG recertification should be governed. | `hot_update_gate_registry.go`, `hot_update_outcome_registry.go`, `promotion_registry.go`, `runtime_pack_registry.go`, `taskstate.go`, `loop.go`, and `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` implement direct operator flow through V4-063. | complete for current operational lane; partial versus full spec states | No V4-065 work. Broader smoke/canary/commit state semantics remain deferred. |
| Hot-update promotion/rollback-like ledger behavior | Successful hot-update outcome can create promotion; LKG recertifies from promotion; rollback records can link back to promotion/gate/outcome. | V4-050, V4-054, V4-058 helpers and V4-052, V4-056, V4-060 direct commands exist. `RollbackRecord` linkage validates promotion/gate/outcome refs. | partial | Later rollback integration polish if required. |
| Operator command/runbook surfaces | Operators need direct commands and concise runbook for completed hot-update lifecycle. | `loop.go` parses `HOT_UPDATE_GATE_RECORD`, `HOT_UPDATE_GATE_PHASE`, `HOT_UPDATE_GATE_EXECUTE`, `HOT_UPDATE_GATE_RELOAD`, `HOT_UPDATE_GATE_FAIL`, `HOT_UPDATE_OUTCOME_CREATE`, `HOT_UPDATE_PROMOTION_CREATE`, `HOT_UPDATE_LKG_RECERTIFY`, and `STATUS`. `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` documents the sequence. | complete for hot-update lane | No V4-065 work. |

## Existing Implemented Foundations

- Runtime pack active pointer: implemented by `ActiveRuntimePackPointer`, `StoreActiveRuntimePackPointer`, `LoadActiveRuntimePackPointer`, status `runtime_pack_identity.active`, and hot-update pointer switch guards.
- Last-known-good pointer: implemented by `LastKnownGoodRuntimePackPointer`, `StoreLastKnownGoodRuntimePackPointer`, `LoadLastKnownGoodRuntimePackPointer`, `RecertifyLastKnownGoodFromPromotion`, and status `runtime_pack_identity.last_known_good`.
- Hot-update promotion/rollback-like ledger behavior: implemented as separate records for hot-update gates, outcomes, promotions, LKG recertification from promotion, rollbacks, and rollback-apply records. This gives deterministic linkage but is not a full autonomous promotion policy engine.
- Operator command/runbook surfaces: direct commands and `STATUS` cover the completed hot-update lane, and `docs/HOT_UPDATE_OPERATOR_RUNBOOK.md` gives the current operator sequence.

## Missing Or Incomplete Controlled Self-Improvement Areas

- Job/proposal-level `execution_plane`, `execution_host`, `mission_family`, and target-surface enforcement.
- Full improvement-family admission control before mutation/evaluation begins.
- Runtime-source mutation prohibition and source-patch artifact-only policy.
- Topology mode flag and disabled-by-default rejection path.
- Promotion policy object/evaluator, including baseline, train/holdout, canary, compatibility, and rollback-target checks.
- Machine-readable V4 `E_*` rejection-code constants tied to the new validators.
- Deploy lock and unsafe live-job promotion blocking.
- First-class prompt-pack and skill-pack registries beyond refs on runtime packs.
- Append-only improvement event ledger, if the existing record-per-entity ledgers are insufficient for audit requirements.
- Read-model polish for improvement run plane/host/family/objective/state/decision and candidate/canary channel status.

## V4-065 Recommendation

Pick exactly one next slice: **execution-plane field/storage/read-model/validation skeleton**.

This should add the smallest job/proposal-level skeleton for:

- `execution_plane`
- `execution_host`
- `mission_family`
- validation that these fields are present for V4 jobs
- compatibility validation for improvement-family jobs requiring `execution_plane=improvement_workspace`
- compatibility validation for live external jobs requiring `execution_plane=live_runtime`
- compatibility validation for hot-update families requiring `execution_plane=hot_update_gate` unless represented as a governed internal gate call
- read-model/inspect exposure sufficient for operators to see the declared plane/host/family

Why this comes first:

- The frozen spec makes execution-plane declarations the root boundary between live runtime, improvement workspace, and hot-update gate.
- Improvement-family lab enforcement, source patch restrictions, topology gating, deploy locks, and V4 rejection codes all need a concrete field surface to attach to.
- Adding a full adaptive lab before this boundary exists would create workflows whose safety depends on convention rather than validated state.
- Rejection-code constants alone would be premature without validators that can emit them deterministically.
- Improvement-run ledger expansion alone is too late in the lifecycle because an unsafe job could already have been admitted before a run record exists.

## Non-Goals Preserved

- No Go code changed.
- No tests changed.
- No commands added.
- No TaskState wrappers added.
- No `active_pointer.json` mutation.
- No `last_known_good_pointer.json` mutation.
- No `reload_generation` mutation.
- No runtime packs, candidates, eval suites, improvement runs, outcomes, promotions, rollbacks, or gates created or mutated.
- No `execution_plane`, improvement-family enforcement, or rejection-code implementation started.
- No V4-065 work started.
