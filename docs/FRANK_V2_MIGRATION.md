# Frank v1 to v2 Migration

## Purpose

This guide is for moving from the currently deployed Frank v1 baseline toward the frozen v2 target in [`docs/FRANK_V2_SPEC.md`](./FRANK_V2_SPEC.md).

Keep one distinction explicit the whole time:

- the frozen v2 spec is the target contract
- the current runtime is only a partial implementation of that target

Do not call the runtime "v2" just because the spec exists.

## Current baseline now

### Deployed v1 baseline

Assuming the currently deployed phone build is `7564a58` / `7564a589`, the deployed baseline is the mission-control runtime around commit `7564a58`:

- mission-required startup gating
- mission bootstrap from `--mission-file` + `--mission-step`
- mission status snapshot JSON via `--mission-status-file`
- step control file support via `--mission-step-control-file`
- operator CLI for:
  - `picobot mission status`
  - `picobot mission inspect`
  - `picobot mission assert`
  - `picobot mission assert-step`
  - `picobot mission set-step`
- guard audit events for allow/reject decisions
- reboot resume blocked unless `--mission-resume-approved` is provided

### Desktop-ahead items

Relative to the deployed phone baseline:

- `4047d8a` is test-only acceptance coverage ahead of the deployed baseline
- the desktop repo may also be ahead in documentation-only commits, including the frozen v2 spec and operator/migration docs

That means the repo may be ahead in documentation and tests without changing deployed runtime behavior.

## Current runtime vs frozen v2 target

### Already implemented and usable now

These v2-adjacent contracts already exist in the current runtime:

- fail-closed `--mission-required` execution
- mission bootstrap from job JSON plus active step
- mission status snapshot output
- step switching through a control file
- fresh status confirmation for `mission set-step`
- startup restore from an existing step control file
- runtime state carrying `running`, `waiting_user`, `paused`, `failed`, and terminal states
- reboot resume requiring explicit approval
- audited allow/reject tool-guard decisions
- step validation for:
  - `discussion`
  - `static_artifact`
  - `one_shot_code`
  - `final_response`

### Frozen v2 contracts not yet implemented

The frozen spec requires more than the current runtime provides. Gaps include:

- the full seven-step v2 set is not implemented
  - missing runtime step types: `long_running_code`, `system_action`, `wait_user`
- durable multi-record control plane does not exist yet
  - no durable job store
  - no durable step-state store
  - no durable approval record store
  - no durable artifact registry
- text-channel operator syntax is not implemented as runtime contract
  - `APPROVE`
  - `DENY`
  - `PAUSE`
  - `RESUME`
  - `ABORT`
  - `STATUS`
  - `SET_STEP`
- one-active-job rule is not implemented as a durable multi-job system
  - current runtime is effectively one bootstrapped mission, not a persisted job set
- replay-safe and idempotent step execution are not fully implemented as durable behavior contracts
- approval ambiguity handling, channel scope binding, and approval scopes are not implemented
- budget ceilings, retention defaults, mandatory notifications, and packaging rules are not implemented
- v2 rejection-code mapping is not implemented
  - current runtime uses internal codes like `tool_not_allowed` and `resume_approval_required`, not the frozen `E_*` surface

## Workstreams

### Workstream 1: Freeze the real v1 baseline in engineering terms

Goal:

- treat `7564a58` as the deployed runtime baseline
- treat `4047d8a` as extra evidence only
- treat `b407905de` as the spec/documentation checkpoint

What to do:

- keep deployment notes anchored to `7564a58`
- use current CLI/runtime behavior as the migration starting point
- do not rewrite the current baseline into imagined v2 semantics

Exit condition:

- everyone can answer "what is deployed now?" without referring to the v2 spec as if it were already live

### Workstream 2: Durable runtime state

Goal:

- move from snapshot-assisted runtime memory to a real durable control plane

What exists now:

- status snapshot may carry embedded runtime state
- startup can resume same-job non-terminal runtime from that snapshot

What remains:

- durable record model for job, step, approval, audit, and artifact state
- crash-safe write rules
- conservative recovery when records disagree

Why this comes first:

- most v2 semantics depend on durable state
- operator approval, replay safety, pause/resume, and multi-job handling are not credible without it

### Workstream 3: Step-model expansion to the frozen v2 set

Goal:

- implement only the step types named in the frozen spec

What exists now:

- `discussion`
- `static_artifact`
- `one_shot_code`
- `final_response`

What remains:

- `long_running_code`
- `system_action`
- `wait_user`

Constraints:

- no new mission families
- no new step types outside the frozen spec
- keep planning as a job phase, not a step type

### Workstream 4: Operator control plane

Goal:

- upgrade from file/CLI-only control to the frozen operator contract without breaking the current surfaces

What exists now:

- `picobot mission status`
- `picobot mission inspect`
- `picobot mission set-step`
- helper assertions for local verification

What remains:

- durable `APPROVE` / `DENY`
- durable `PAUSE` / `RESUME` / `ABORT`
- `STATUS <job_id>` and `SET_STEP <job_id> <step_id>` semantics as operator contract
- approval scope rules
- ambiguity handling for plain-language yes/no replies

Implementation note:

- preserve the current file/CLI controls during migration
- add the v2 control plane around them rather than removing the proven v1 path first

### Workstream 5: Validation, idempotence, and truthful resume

Goal:

- make restart and re-entry behavior match the frozen v2 contract

What exists now:

- step completion validators for the current v1 step types
- resume approval requirement after reboot
- fresh status confirmation for step switch

What remains:

- already-satisfied detection as durable runtime truth
- replay-safe step re-entry rules by step family
- long-running build versus start separation
- post-state verification for `system_action`

### Workstream 6: Deployment hardening

Goal:

- roll out v2 safely on the phone without losing the ability to fall back to v1 behavior

What remains:

- durable storage deployment path
- migration of snapshot-only state into durable state
- rollback rules for partially migrated phone bodies
- operator runbook updates for approval, pause, resume, and abort flows

## Recommended implementation order

1. Lock the deployed baseline and acceptance evidence around `7564a58`, with `4047d8a` used only as extra test coverage.
2. Build the durable state layer for jobs, steps, approvals, audits, and artifacts.
3. Implement the v2 job/step state machine on top of durable state before adding new operator syntax.
4. Add the missing frozen v2 step types: `wait_user`, `long_running_code`, `system_action`.
5. Add durable approval records and operator actions: approve, deny, pause, resume, abort.
6. Add operator-channel syntax and ambiguity-resolution rules.
7. Add idempotence, replay-safe restart behavior, and already-satisfied handling.
8. Add budgets, notifications, and retention rules that the frozen spec requires.
9. Run the acceptance gates and only then mark the runtime as v2-ready.

## Deployment and rollback

### Deployment considerations

- deploy from a commit that contains runtime code, not just the v2 spec doc
- keep the existing v1 CLI/file control path working during rollout
- preserve the current `--mission-required`, `--mission-file`, `--mission-step`, `--mission-status-file`, and `--mission-step-control-file` startup path until the v2 durable control plane is proven
- gate reboot resume behavior conservatively; do not weaken `--mission-resume-approved`

### Rollback considerations

- if durable v2 state is introduced, define a one-way migration boundary before deploying
- keep a rollback path that can still boot the phone into the known v1 mission-required posture
- do not require newly introduced v2 records for basic safe startup unless the migration completed cleanly
- if rollback would strand the phone between snapshot-only v1 and durable v2 state, the migration design is incomplete

## Acceptance gates before calling v2 ready

Do not call the runtime v2-ready until all of these are true:

- durable job, step, approval, audit, and artifact records exist
- all seven frozen v2 step types are implemented and validated
- one-active-job rule is enforced over durable state
- `APPROVE`, `DENY`, `PAUSE`, `RESUME`, `ABORT`, `STATUS`, and `SET_STEP` exist as operator contracts
- restart pauses unsafe work and does not silently reissue side effects
- already-satisfied work is reported truthfully
- long-running build and start are separate contracts
- `system_action` verifies post-state before completion
- approval ambiguity does not bind
- the runtime passes the relevant acceptance criteria in [`docs/FRANK_V2_SPEC.md`](./FRANK_V2_SPEC.md)

## Practical reading of the migration

Use this shorthand:

- v1 now = deployed narrow mission-control runtime
- v2 spec = frozen target contract
- v2 runtime = not real until the runtime matches the spec

That distinction keeps the migration honest and prevents scope drift beyond the frozen spec.
