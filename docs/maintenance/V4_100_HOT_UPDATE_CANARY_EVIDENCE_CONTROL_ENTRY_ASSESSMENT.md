# V4-100 Hot-Update Canary Evidence Control Entry Assessment

## Scope

V4-100 assesses the smallest safe TaskState/direct-command surface for creating `HotUpdateCanaryEvidenceRecord` through the existing operator control path.

This slice is docs-only. It does not change Go code, tests, commands, TaskState wrappers, canary requirements, canary evidence, owner approval records, candidate promotion decisions, hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, active runtime-pack pointer, last-known-good pointer, `reload_generation`, pointer-switch behavior, reload/apply behavior, or V4-101 implementation.

## Live State Inspected

Docs inspected:

- `docs/FRANK_V4_SPEC.md`
- `docs/maintenance/V4_098_HOT_UPDATE_CANARY_EVIDENCE_ASSESSMENT.md`
- `docs/maintenance/V4_099_HOT_UPDATE_CANARY_EVIDENCE_REGISTRY_AFTER.md`

Code surfaces inspected:

- `internal/missioncontrol/hot_update_canary_evidence_registry.go`
- `internal/missioncontrol/hot_update_canary_requirement_registry.go`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/missioncontrol/audit.go`
- direct command tests in `internal/agent/loop_processdirect_test.go`
- hot-update gate, outcome, promotion, rollback, rollback-apply, and last-known-good control patterns

## Current Canary Evidence Surface

V4-099 added:

```go
CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID string, state HotUpdateCanaryEvidenceState, observedAt time.Time, createdBy string, createdAt time.Time, reason string) (HotUpdateCanaryEvidenceRecord, bool, error)
```

The helper creates or selects append-only canary evidence records under:

```text
runtime_packs/hot_update_canary_evidence/<canary_evidence_id>.json
```

The deterministic evidence ID is:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

The helper accepts only these evidence states:

- `passed`
- `failed`
- `blocked`
- `expired`

`passed` is derived from `evidence_state`; it is not caller-supplied. The helper stores `passed=true` only when `evidence_state=passed`.

The helper loads a committed `HotUpdateCanaryRequirementRecord`, requires it to remain `state=required`, and revalidates the source chain: candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, and freshly derived promotion eligibility. Derived eligibility must still be `canary_required` or `canary_and_owner_approval_required`.

The status/read-model surface is:

```text
hot_update_canary_evidence_identity
```

It is included in committed mission status snapshots and surfaces configured, not-configured, and invalid records without mutating the store.

## Existing Control Patterns

Hot-update direct commands are parsed in `internal/agent/loop.go` as uppercase or lowercase snake-case command names with fixed positional arguments and optional trailing reason text for commands that need free-form explanation.

The closest existing command is:

```text
HOT_UPDATE_CANARY_REQUIREMENT_CREATE <job_id> <result_id>
```

Its TaskState wrapper:

- validates the mission store root
- validates active execution context when present
- otherwise validates persisted runtime state and persisted runtime control context
- rejects `job_id` mismatches
- derives `created_by="operator"`
- derives `created_at` from `taskStateTransitionTimestamp(taskStateNowUTC())`
- reuses an existing deterministic record's `created_at` to make exact replay byte-stable
- emits audit action `hot_update_canary_requirement_create` on success, idempotent selection, and failure
- returns the missioncontrol record and changed flag

Neighboring create commands return empty direct-command response plus error on failure. Successful deterministic creates use "Created ..." when `changed=true` and "Selected ..." when `changed=false`.

## Decision

Canary evidence creation should be exposed as a direct operator command in the next slice.

Reasoning:

- V4-099 added the durable registry, validation, idempotence, and read model.
- The helper is scoped to an existing canary requirement and does not create gates, owner approvals, outcomes, promotions, rollbacks, or pointer mutations.
- Operators need a deliberate way to record manual or externally observed canary evidence before later slices can consume passed evidence.

The command should use `canary_requirement_id` as the source authority.

Reasoning:

- V4-099 made `HotUpdateCanaryRequirementRecord` the authority for evidence creation.
- `result_id` is upstream of the requirement and would make the control entry redo requirement selection semantics.
- `canary_evidence_id` is derived from `canary_requirement_id` and `observed_at`; accepting it directly would add mismatch risk and redundant authority.
- hot-update gate IDs are not evidence authority because evidence must exist before any canary-required gate can be created.

The command should accept `observed_at` as a required argument.

Reasoning:

- The V4-099 deterministic evidence ID includes `observed_at`.
- If the wrapper derived `observed_at` from TaskState time, exact replay of the same direct command would produce a different evidence ID and append a second evidence record.
- Required `observed_at` preserves append-only attempt semantics while allowing exact replay to select the same record.
- The timestamp should be parsed as RFC3339/RFC3339Nano and normalized to UTC by the missioncontrol helper.

`created_at` should still be derived from the TaskState timestamp path. On exact replay, the wrapper should load the existing deterministic evidence record and reuse its `created_at` so the missioncontrol helper sees an exact normalized record.

## Recommended V4-101 Command

Recommend exactly this command:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
```

`observed_at` must parse as RFC3339/RFC3339Nano. The command should allow all V4-099 evidence states:

- `passed`
- `failed`
- `blocked`
- `expired`

`passed` must not be a command argument. It is inferred from `evidence_state` by `CreateHotUpdateCanaryEvidenceFromRequirement(...)`.

`reason` should be optional in the direct command but must become non-empty before calling missioncontrol. If omitted, use a deterministic default such as:

```text
operator recorded hot-update canary evidence <evidence_state>
```

Reasons are free-form trailing text so operators can provide context without changing command arity.

Rejected alternatives:

- `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <result_id> ...`: uses upstream source authority and bypasses the requirement ID chosen by V4-099.
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_evidence_id> ...`: accepts a derived ID and risks mismatch with `canary_requirement_id` and `observed_at`.
- `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> [reason...]`: cannot make exact replay byte-stable because `observed_at` would be derived from current TaskState time.
- Adding `passed=<bool>`: duplicates a field that the registry already derives and validates from `evidence_state`.

## Recommended TaskState Wrapper

Add this wrapper in V4-101:

```go
func (s *TaskState) CreateHotUpdateCanaryEvidenceFromRequirement(jobID string, canaryRequirementID string, state missioncontrol.HotUpdateCanaryEvidenceState, observedAt time.Time, reason string) (missioncontrol.HotUpdateCanaryEvidenceRecord, bool, error)
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
10. Validate and normalize `observed_at`; reject zero time.
11. Validate `evidence_state`; reject anything outside `passed`, `failed`, `blocked`, `expired`.
12. Trim `reason`; if empty, derive the deterministic default reason.
13. Use `created_by="operator"`.
14. Derive the deterministic evidence ID with `missioncontrol.HotUpdateCanaryEvidenceIDFromRequirementObservedAt(canaryRequirementID, observedAt)`.
15. If an existing deterministic evidence record loads successfully, reuse its `created_at` for replay stability.
16. If loading returns anything other than not-found, fail closed.
17. Call `missioncontrol.CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID, state, observedAt, "operator", createdAt, reason)`.
18. Emit audit action `hot_update_canary_evidence_create` on success, idempotent selection, and failure.
19. Return the record and changed flag from the missioncontrol helper.

The existing-created-at reuse is required because the missioncontrol helper treats a different `created_at` as a divergent duplicate for the same deterministic evidence ID.

## Direct Command Responses

On `changed=true`:

```text
Created hot-update canary evidence job=<job_id> canary_requirement=<canary_requirement_id> canary_evidence=<canary_evidence_id> evidence_state=<state> passed=<bool>.
```

On `changed=false`:

```text
Selected hot-update canary evidence job=<job_id> canary_requirement=<canary_requirement_id> canary_evidence=<canary_evidence_id> evidence_state=<state> passed=<bool>.
```

The response should include `passed=<bool>` because later operator steps need to distinguish evidence that satisfies the canary branch from failed, blocked, or expired evidence.

On failure:

- return empty direct-command response plus error
- emit rejected audit action `hot_update_canary_evidence_create`
- do not create or mutate unrelated records

Malformed commands with missing or extra required arguments should reject with:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE requires job_id, canary_requirement_id, evidence_state, observed_at, and optional reason
```

## Replay And Duplicate Behavior

Exact replay should be byte-stable when all command arguments match, including `observed_at` and reason/default reason.

The V4-101 wrapper should:

- derive the same deterministic evidence ID from `canary_requirement_id` and `observed_at`
- load the existing deterministic record when present
- reuse existing `created_at`
- call the missioncontrol helper with the same normalized fields
- return selected/idempotent response when the record is identical

Divergent duplicates must fail closed. Examples:

- same `canary_requirement_id` and `observed_at`, different evidence state
- same `canary_requirement_id` and `observed_at`, different reason
- existing deterministic evidence record that fails validation or linkage checks
- stale eligibility that changes away from canary-required between record creation and replay

Multiple evidence records for the same requirement remain allowed only when `observed_at` differs and therefore the deterministic evidence ID differs.

## Owner Approval Relationship

For requirements whose underlying eligibility is `canary_and_owner_approval_required`, passed evidence should be represented as:

```text
evidence_state=passed passed=true
```

The command should not create owner approval requests or proposals. It should only make the passed canary evidence visible in `hot_update_canary_evidence_identity`. A later owner-approval slice can verify passed evidence before requesting approval. Owner approval must not replace required canary evidence.

## Fail-Closed Requirements

V4-101 should fail closed for:

- missing or malformed command arguments
- invalid `observed_at`
- invalid evidence state
- missing mission store root
- missing active execution context and missing persisted runtime context
- wrong `job_id`
- missing canary requirement
- invalid canary requirement
- canary requirement not in `state=required`
- missing candidate result
- missing linked improvement run
- missing linked improvement candidate
- missing or unfrozen eval suite
- missing promotion policy
- missing baseline runtime pack
- missing candidate runtime pack
- stale derived eligibility away from `canary_required` or `canary_and_owner_approval_required`
- divergent duplicate evidence ID
- existing deterministic evidence record that does not validate or load

V4-101 must not treat failed, blocked, or expired evidence as satisfying the canary requirement.

## Recommended V4-101 Slice

Implement exactly one next slice:

```text
V4-101 — Hot-Update Canary Evidence Control Entry
```

Scope:

- add direct parser for `HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]`
- add malformed-command rejection with the exact argument error
- add TaskState wrapper around `missioncontrol.CreateHotUpdateCanaryEvidenceFromRequirement(...)`
- derive `created_by="operator"`
- derive `created_at` from TaskState time and reuse existing `created_at` for exact replay
- parse and validate RFC3339/RFC3339Nano `observed_at`
- infer `passed` from evidence state
- emit audit action `hot_update_canary_evidence_create`
- return created/selected response including `passed=<bool>`
- add focused tests for created, selected, replay byte stability, divergent duplicate rejection, wrong job rejection, stale/missing requirement/source failures, audit behavior, status surfacing, and no unrelated mutations
- add before/after maintenance docs

## Non-Goals

V4-100 and the recommended V4-101 must not:

- execute canaries
- create canary execution automation
- create canary execution proposal records
- request owner approval
- create owner approval proposal records
- create candidate promotion decisions for canary-required states
- create hot-update gates for canary-required states
- create outcomes
- create promotions
- create rollbacks
- create rollback-apply records
- mutate candidate results
- mutate canary requirements
- mutate promotion policies
- mutate runtime packs
- mutate active runtime-pack pointer
- mutate last-known-good pointer
- mutate `reload_generation`
- change pointer-switch behavior
- change reload/apply behavior
- implement V4-102 or later work
