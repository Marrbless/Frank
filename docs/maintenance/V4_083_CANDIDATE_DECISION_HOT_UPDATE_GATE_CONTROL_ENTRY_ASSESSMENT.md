# V4-083 Candidate Decision Hot-Update Gate Control Entry Assessment

## Facts

V4-082 added the missioncontrol-only helper:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

The helper creates a prepared hot-update gate from a committed `CandidatePromotionDecisionRecord`. It derives the deterministic hot-update ID as:

```text
hot-update-<promotion_decision_id>
```

The helper loads and cross-checks the committed promotion decision, linked candidate result, current derived promotion eligibility, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, active runtime-pack pointer, and last-known-good pointer when present. It requires a selected and eligible decision, still-eligible derived result state, matching decision/result authority, an active pointer whose active pack equals the decision baseline pack, and a candidate rollback target that is present and loadable as a runtime pack.

Existing hot-update direct commands use the `HOT_UPDATE_GATE_*` namespace for gate lifecycle control:

```text
HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>
HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>
HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>
HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id>
```

Existing direct command handlers return a non-empty operator response on success or idempotent selection, and return an empty response with an error on failure.

Existing TaskState hot-update wrappers validate the mission store root, require an active or persisted mission-control job context, reject a job ID that does not match the active/persisted job, derive timestamps through the TaskState clock path, use `operator` as the actor for operator commands, and emit audit actions with lowercase names matching the command domain.

## Analysis

The smallest safe V4-084 implementation should expose the V4-082 helper through the existing direct command path and a TaskState wrapper. It should not introduce a new storage helper, a new missioncontrol behavior, or an execution-side hot-update transition.

Recommended V4-084 command:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
```

This name keeps the command in the established `HOT_UPDATE_GATE_*` family and describes the source authority. The alternative `HOT_UPDATE_DECISION_GATE_CREATE` would introduce a new `HOT_UPDATE_DECISION_*` namespace that does not match the current gate lifecycle command layout.

The direct command should call a TaskState wrapper rather than calling missioncontrol directly. The wrapper should own job validation, store-root resolution, timestamp derivation, actor derivation, audit emission, and idempotent replay semantics, matching the existing hot-update gate/outcome/promotion/LKG control pattern.

The wrapper should call:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, "operator", createdAt)
```

where `createdAt` comes from the existing TaskState transition timestamp path. The wrapper should derive the deterministic hot-update ID for response and replay handling as `hot-update-<promotion_decision_id>`.

Because V4-082 stores the gate preparation timestamp in the gate record, command replay with a fresh TaskState timestamp can otherwise look like a divergent duplicate. V4-084 should preserve stable replay by reusing the existing gate `prepared_at` timestamp when a deterministic gate already exists for `hot-update-<promotion_decision_id>`, then rerunning the V4-082 helper and letting the helper perform the authority checks. This mirrors the existing timestamp-reuse pattern used by deterministic outcome, promotion, and LKG wrappers.

The audit action should be:

```text
hot_update_gate_from_decision
```

## Required V4-084 Behavior

`HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>` should:

1. Parse exactly two arguments after the command name.
2. Validate the active or persisted job context and reject the command when the supplied `job_id` does not match.
3. Resolve and validate the mission store root through the existing TaskState path.
4. Derive `created_at` from the TaskState clock path.
5. Use `operator` as `created_by`.
6. Invoke `CreateHotUpdateGateFromCandidatePromotionDecision`.
7. Emit audit action `hot_update_gate_from_decision` on success or idempotent selection.
8. Return an empty response and error on failure, consistent with existing direct commands.

Success response:

```text
Created hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

Idempotent response:

```text
Selected hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

## Failure Behavior

V4-084 should fail closed with an empty direct-command response and a non-nil error for:

- missing or malformed arguments
- missing promotion decision
- decision state other than `selected_for_promotion`
- decision eligibility state other than `eligible`
- derived promotion eligibility no longer eligible
- missing linked candidate result, improvement run, improvement candidate, eval suite, promotion policy, baseline pack, or candidate pack
- mismatched decision/result/run/candidate/eval-suite/policy linkage
- missing active runtime-pack pointer
- active pointer whose active pack no longer equals the decision baseline pack
- candidate pack with no rollback target
- rollback target that does not load as a runtime pack
- divergent duplicate hot-update gate
- existing deterministic gate with a different candidate pack or incompatible decision authority
- command `job_id` that does not match the active or persisted job context

## V4-084 Tests

V4-084 should add focused tests for:

- direct command parses `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`
- direct command rejects missing or extra arguments
- command invokes the TaskState wrapper and creates a prepared gate for an eligible selected decision
- created response includes `job`, `promotion_decision`, and deterministic `hot_update`
- idempotent replay returns the selected response
- idempotent replay is byte-stable by reusing the existing gate `prepared_at` timestamp
- wrong `job_id` fails closed with an empty response
- missing decision fails closed with an empty response
- non-selected decision fails closed with an empty response
- stale derived eligibility fails closed with an empty response
- missing linked records fail closed with an empty response
- stale active pointer fails closed with an empty response
- missing candidate rollback target fails closed with an empty response
- missing rollback target runtime pack fails closed with an empty response
- mismatched decision/result/run/candidate/eval-suite/policy authority fails closed with an empty response
- divergent duplicate gate fails closed with an empty response
- existing deterministic gate with a different candidate pack fails closed with an empty response
- audit action `hot_update_gate_from_decision` is emitted on created and selected paths
- the wrapper does not mutate the promotion decision, candidate result, improvement run, improvement candidate, eval suite, promotion policy, runtime pack records, active pointer, last-known-good pointer, or reload generation
- the wrapper does not create hot-update outcomes, promotions, rollback records, rollback-apply records, or LKG records

## Non-Goals Preserved

V4-084 must not:

- add new missioncontrol storage behavior
- mutate the active runtime-pack pointer
- mutate the last-known-good pointer
- mutate `reload_generation`
- create `HotUpdateOutcomeRecord`
- create `PromotionRecord`
- create rollback records
- create rollback-apply records
- create or mutate LKG records
- advance hot-update phases
- execute a pointer switch
- run reload/apply behavior
- run canaries
- request owner approval
- implement deploy-lock or unsafe-live-job blocking
- implement V4-085 or later behavior

## Recommendation

Implement V4-084 as one direct operator command and one TaskState wrapper around the existing V4-082 helper:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
```

The command should create or select only a prepared hot-update gate whose authority is the committed promotion decision and its cross-checked linked records. All execution, phase advancement, pointer mutation, outcome creation, promotion creation, rollback creation, LKG mutation, deploy-lock enforcement, canary execution, and approval behavior should remain deferred to later slices.
