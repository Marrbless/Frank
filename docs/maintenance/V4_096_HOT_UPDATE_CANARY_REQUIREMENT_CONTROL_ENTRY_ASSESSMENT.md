# V4-096 Hot-Update Canary Requirement Control Entry Assessment

## Scope

V4-096 assesses the smallest safe TaskState/direct-command surface for creating `HotUpdateCanaryRequirementRecord` through the existing operator control path.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, canary requirement records, canary evidence, owner approval records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, active runtime-pack pointer, last-known-good pointer, `reload_generation`, or V4-097 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_094_HOT_UPDATE_CANARY_REQUIREMENT_PROPOSAL_ASSESSMENT.md`
- `docs/maintenance/V4_095_HOT_UPDATE_CANARY_REQUIREMENT_REGISTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/status_hot_update_canary_requirement_identity_test.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol/audit.go`
- existing direct command tests in `internal/agent/loop_processdirect_test.go`
- candidate result and promotion eligibility helpers
- hot-update gate, outcome, promotion, execution-ready, and rollback-apply control patterns

## Current Canary Requirement Surface

V4-095 added:

```go
CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (HotUpdateCanaryRequirementRecord, bool, error)
```

The helper creates or selects a deterministic canary requirement record with ID:

```text
hot-update-canary-requirement-<result_id>
```

Records are stored under:

```text
runtime_packs/hot_update_canary_requirements/<canary_requirement_id>.json
```

The helper derives authority from the committed candidate result and `EvaluateCandidateResultPromotionEligibility(root, resultID)`. It accepts only:

- `canary_required`
- `canary_and_owner_approval_required`

It rejects:

- `eligible`
- `owner_approval_required`
- `rejected`
- `unsupported_policy`
- `invalid`

For `canary_and_owner_approval_required`, the record is still one canary requirement record and sets:

```text
owner_approval_required=true
```

No owner approval request is created by V4-095.

## Existing Control Patterns

Existing direct commands are parsed in `internal/agent/loop.go` with case-insensitive snake-case command regexes. Hot-update commands currently include:

- `HOT_UPDATE_GATE_RECORD <job_id> <hot_update_id> <candidate_pack_id>`
- `HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>`
- `HOT_UPDATE_GATE_PHASE <job_id> <hot_update_id> <phase>`
- `HOT_UPDATE_GATE_EXECUTE <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>`
- `HOT_UPDATE_GATE_FAIL <job_id> <hot_update_id> [reason...]`
- `HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]`
- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`
- `HOT_UPDATE_PROMOTION_CREATE <job_id> <outcome_id>`
- `HOT_UPDATE_LKG_RECERTIFY <job_id> <promotion_id>`

The direct command path delegates to TaskState wrappers. The wrappers consistently:

- resolve and validate the mission store root
- validate active or persisted job context
- reject a supplied `job_id` that does not match the active or persisted job
- derive timestamps from `taskStateTransitionTimestamp(taskStateNowUTC())`
- use `operator` as the actor for operator-created records
- emit a runtime-control audit event with a lowercase snake-case action name
- return the missioncontrol `changed` flag, or a record plus changed flag when response formatting needs record details

Failure behavior is consistent: direct command handlers return an empty response plus the error.

## Decision

Canary requirement creation should now be exposed as a direct operator command in the next slice.

Reasoning:

- V4-095 already added the durable missioncontrol registry and read model.
- The helper is idempotent, deterministic, and fail-closed.
- The operator needs a control entry to intentionally materialize the canary requirement fact before later canary evidence or execution slices.
- This remains pre-execution ledger creation, not canary execution.

The command should use candidate `result_id` as the source authority, not `canary_requirement_id`.

Reasoning:

- the requirement is derived from a committed `CandidateResultRecord` and freshly derived eligibility
- `canary_requirement_id` is deterministic from `result_id`
- accepting a caller-supplied requirement ID would add redundant authority and create mismatch risk
- hot-update gate and promotion decision IDs are not source authority for canary-required results

The command should require `job_id` even though the missioncontrol helper is store-root-only.

Reasoning:

- all existing direct hot-update control entries are bound to an active or persisted mission job
- TaskState owns mission store root resolution, timestamp derivation, actor derivation, and audit emission
- `job_id` prevents a stale or unrelated operator command from writing to the store without matching the current mission context

## Recommended V4-097 Command

Recommend exactly this command:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
```

Rationale:

- it stays in the existing `HOT_UPDATE_*` operator-control namespace
- it names the created ledger object directly
- it uses `CREATE`, matching `HOT_UPDATE_OUTCOME_CREATE` and `HOT_UPDATE_PROMOTION_CREATE`
- it does not imply canary execution, canary evidence, gate creation, or owner approval
- it consumes the authoritative `result_id`, not a derived ID

Rejected alternatives:

- `HOT_UPDATE_CANARY_CREATE`: ambiguous with canary execution
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE`: belongs to a later evidence slice
- `HOT_UPDATE_CANARY_REQUIREMENT_RECORD`: less aligned with recent `*_CREATE` commands for derived deterministic ledger creation
- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <canary_requirement_id>`: uses a derived identity instead of source authority

## Recommended TaskState Wrapper

Add this wrapper in V4-097:

```go
func (s *TaskState) CreateHotUpdateCanaryRequirementFromCandidateResult(jobID string, resultID string) (missioncontrol.HotUpdateCanaryRequirementRecord, bool, error)
```

The wrapper should:

1. Return zero record, `false`, `nil` when `s == nil`, matching local wrapper conventions.
2. Derive `now` from `taskStateTransitionTimestamp(taskStateNowUTC())`.
3. Clone execution context, runtime control, runtime state, and mission store root under lock.
4. Build the audit execution context from active execution context or persisted runtime/control context.
5. Validate the mission store root.
6. Validate active execution context when present: job, step, and runtime must exist.
7. Reject `job_id` that does not match the active execution context job.
8. Otherwise validate persisted runtime state and persisted runtime control context.
9. Reject `job_id` that does not match persisted runtime state or persisted runtime control.
10. Derive `created_by="operator"`.
11. Derive the deterministic requirement ID using `missioncontrol.HotUpdateCanaryRequirementIDFromResult(resultID)`.
12. If an existing deterministic requirement loads successfully, reuse its `created_at` for replay stability.
13. If loading returns anything other than not-found, fail closed.
14. Call `missioncontrol.CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, "operator", createdAt)`.
15. Emit audit action `hot_update_canary_requirement_create` on success, idempotent selection, and failure.
16. Return the record and changed flag from the missioncontrol helper.

The existing-created-at reuse is required because the missioncontrol helper treats a different `created_at` as a divergent duplicate. Without that reuse, an exact direct-command replay with a fresh TaskState timestamp would fail instead of selecting the existing deterministic record.

## Direct Command Responses

On `changed=true`:

```text
Created hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

On `changed=false`:

```text
Selected hot-update canary requirement job=<job_id> result=<result_id> canary_requirement=<canary_requirement_id> owner_approval_required=<bool>.
```

The response should include `owner_approval_required=<bool>` on both paths.

Reasoning:

- `canary_and_owner_approval_required` creates only a canary requirement in this slice
- surfacing the flag makes the remaining owner-approval branch visible without creating an approval request
- callers do not need to infer combined-state behavior from status JSON after the command

On failure:

- return empty direct-command response plus error
- do not emit a success acknowledgement
- do not create or mutate unrelated records

Malformed command handling should reject missing or extra arguments with:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE requires job_id and result_id
```

## Audit

Use audit action:

```text
hot_update_canary_requirement_create
```

Emit it through the existing TaskState runtime-control audit path. Successful and idempotent paths should be allowed/applied. Failure paths should be rejected and should carry `missioncontrol.ValidationError` codes when the wrapper creates those errors for job-context failures. Helper errors can continue to use the existing invalid-runtime fallback used by neighboring wrappers.

## Failure Behavior

V4-097 must fail closed with empty response plus error for:

- missing or malformed command arguments
- missing mission store root
- missing active execution context and missing persisted runtime context
- supplied `job_id` not matching active job
- supplied `job_id` not matching persisted runtime state
- supplied `job_id` not matching persisted runtime control
- missing candidate result
- invalid candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- derived eligibility `eligible`
- derived eligibility `owner_approval_required`
- derived eligibility `rejected`
- derived eligibility `unsupported_policy`
- derived eligibility `invalid`
- derived eligibility changing away from canary-required during replay
- divergent duplicate requirement
- existing deterministic requirement that does not validate or load
- another requirement for the same result under a different ID

Wrong job ID should use the same TaskState behavior as neighboring hot-update wrappers:

- return a `missioncontrol.ValidationError`
- code `RejectionCodeStepValidationFailed`
- message `operator command does not match the active job`
- empty direct-command response
- rejected audit event for `hot_update_canary_requirement_create`

## Canary Plus Owner Approval

For `canary_and_owner_approval_required`, V4-097 should:

- create or select the canary requirement
- return `owner_approval_required=true`
- expose the same value through `hot_update_canary_requirement_identity`
- not create an owner approval request
- not create an owner approval proposal record
- not create a candidate promotion decision
- not create a hot-update gate

Owner approval remains a later dedicated surface. It must not be used as a substitute for required canary evidence.

## Read Model Expectations

After successful command execution, existing status should be sufficient:

- `STATUS <job_id>` should include `hot_update_canary_requirement_identity`
- configured records should expose `canary_requirement_id`, `result_id`, refs, `eligibility_state`, `required_by_policy`, `owner_approval_required`, `requirement_state`, reason, actor, and timestamp
- invalid records should remain visible without hiding valid records

V4-097 should not add a separate status command unless implementation proves the existing status read model is insufficient.

## Required V4-097 Tests

Direct command tests should prove:

- `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>` parses and creates a requirement for `canary_required`
- created response includes job, result, deterministic canary requirement ID, and `owner_approval_required=false`
- command creates a requirement for `canary_and_owner_approval_required`
- combined-state response includes `owner_approval_required=true`
- exact replay returns the selected response
- exact replay is byte-stable by reusing existing `created_at`
- malformed command with missing or extra args returns empty response plus argument error
- wrong `job_id` returns empty response plus TaskState job mismatch error
- missing candidate result fails closed
- eligible result fails closed
- owner-approval-only result fails closed
- rejected result fails closed
- unsupported-policy result fails closed
- invalid result fails closed
- missing linked records fail closed
- divergent duplicate fails closed
- stale eligibility on replay fails closed
- audit action `hot_update_canary_requirement_create` is emitted on created and selected paths
- failure emits rejected audit action `hot_update_canary_requirement_create`
- `STATUS <job_id>` surfaces the created requirement in `hot_update_canary_requirement_identity`
- source records are not mutated
- no candidate promotion decision is created
- no hot-update gate is created
- no canary evidence is created
- no owner approval request/proposal is created
- no outcome, promotion, rollback, rollback-apply, active pointer, last-known-good pointer, or `reload_generation` mutation occurs

TaskState-focused tests should be added only if direct command tests do not already lock job validation, audit action, timestamp reuse, and changed-flag propagation.

## Non-Goals For V4-097

V4-097 must not:

- execute canaries
- create canary evidence
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- broaden promotion policy grammar
- add canary execution readiness logic
- implement owner approval control entries
- implement V4-098 or later work

## Recommendation

Implement exactly one next slice:

```text
V4-097 — Hot-Update Canary Requirement Control Entry
```

Scope:

- add direct command `HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>`
- add TaskState wrapper `CreateHotUpdateCanaryRequirementFromCandidateResult`
- call the existing V4-095 missioncontrol helper
- derive `created_by=operator`
- derive `created_at` from the TaskState timestamp path
- reuse existing requirement `created_at` on deterministic replay
- emit audit action `hot_update_canary_requirement_create`
- return created/selected responses with `owner_approval_required=<bool>`
- add focused direct command tests and status/read-only side-effect assertions

Do not implement canary evidence, canary execution, owner approval, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, pointer mutation, reload/apply changes, or any V4-098 work.
