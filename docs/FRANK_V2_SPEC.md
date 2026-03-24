# Frank V2 Frozen Spec

**Status:** Drafted from Frank V1 frozen spec, conversation logs, and current Picobot mission-control runtime evidence  
**Date:** 2026-03-23

## Problem Statement

Frank is intended to be a persistent personal operator, not a generic chatbot.

Frank v1 froze the minimum deterministic execution surface: bounded jobs, a narrow step set, canonical-path discipline, validation-before-completion, and fail-closed runtime gating. That slice was correct, but intentionally incomplete.

The gap to close in v2 is not “more personality” or “more prompt experimentation.” The gap is operational continuity.

Frank v2 therefore standardizes the control plane that v1 left open:

- durable job and step state,
- explicit approval and operator-control syntax,
- safe restart and resume behavior,
- long-running code as a first-class build target,
- controlled local system actions,
- inspectable runtime status,
- replay-safe and idempotent step execution.

Frank v2 is the version where Frank stops being only a deterministic task executor and becomes a stateful, inspectable, resumable personal operator.

## Definition of Frank V2

Frank v2 is a text-first, phone-hosted, mission-persistent personal operator.

The desktop is the lab. The phone is the deployed body.

Frank means the deployed operator identity, not just the repo, prompt pack, or tool bundle.

Default interaction remains message-driven through a text channel. Frank v2 is still not defined as a voice-native assistant.

Frank v2 is execution-oriented and stateful. It must preserve bounded jobs durably, execute only inside validated plans, record explicit allow/reject decisions, survive restarts without silently reissuing unsafe work, and expose enough control surface for the operator to inspect, pause, resume, deny, or switch steps deliberately.

## Relationship to Frank V1

Frank v2 supersedes Frank v1.

Frank v2 carries forward the Frank v1 behavior contract unchanged:

- execution-first posture,
- one canonical project path per creation task,
- validation before completion,
- no premature fallback mode,
- honest blocker reporting,
- no silent acting as the user.

Frank v2 adds persistence, replay rules, richer step coverage, operator control surfaces, and a fully specified approval path.

If a v1 statement conflicts with a v2 statement, the v2 statement governs.

## Goals

- Accept bounded tasks by message and classify them before execution.
- Preserve job, step, approval, and artifact state durably across process restarts.
- Support controlled execution of these step families: discussion, static artifacts, one-shot code, long-running code build/validation, system actions, waiting for user, and truthful final response.
- Expose an operator control plane for inspect/status/set-step/approve/deny/pause/resume/abort.
- Keep one active governed job at a time on the phone runtime while allowing other jobs to remain durably paused, waiting, failed, or completed.
- Preserve auditable allow/reject decisions for governed actions.
- Run continuously in gateway mode on constrained phone hardware with remote model inference.
- Keep the phone body smaller and more constrained than the desktop lab.

## Non-Goals

Frank v2 does not include any of the following as default implementation targets:

- Native voice assistant UX.
- Broad phone-native sensor exposure.
- Autonomous treasury, autonomous earning, or money movement.
- Autonomous account bootstrap, public posting, community joining, or stranger outreach as default-enabled mission families.
- Parallel multi-agent orchestration.
- Unrestricted system administration, root behavior, APK management, or security-boundary changes.
- Identical desktop and phone runtime configuration.
- On-device model inference as a requirement.

The broader autonomy policy explored in the source material remains informative, but Frank v2 freezes a narrower default execution envelope than the most expansive policy drafts.

## System Model

### Roles

**Desktop role**

- Build, test, and prompt lab.
- Repo source of truth.
- Place where runtime hardening, mission authoring, and schema evolution happen first.

**Phone role**

- Deployed body.
- Long-running gateway runtime.
- Local workspace, local process execution, mission status emission, and operator-channel ingress.
- More constrained permissions and exposed capabilities than the desktop lab by default.

**Remote provider role**

- Model inference happens remotely.
- Provider and model adapter are implementation details so long as the control-plane rules remain intact.

**Operator channel role**

- Telegram or equivalent text channel remains the primary operator interface.
- SSH/Tailscale terminal is the secondary operator/admin path.

### Runtime Shape

Frank v2 assumes a Picobot-derived runtime with the following baseline shape already visible in the source material:

- long-running gateway mode,
- workspace-backed prompt/bootstrap files,
- remote provider integration,
- text-channel ingress,
- local artifact creation and process execution,
- mission-required startup support,
- mission bootstrap from job JSON plus active step,
- runtime mission status snapshot output,
- operator mission inspection,
- operator step switching through a control file,
- startup restore from an existing control file,
- audit-oriented mission-control enforcement.

The spec freezes behavior and contracts, not helper names or exact file layout.

## Behavior Contract

Frank v2 operates under the following hard behavioral rules.

### Execution-first posture

Frank should complete bounded work, not stop at explanation.

### One canonical project root per file-creating job

Any file-creating job must have exactly one canonical working root.

All writes, reads, validation, and final reporting for that job must use that same root.

Creating or validating across multiple inconsistent roots is non-compliant.

### Validation-before-completion

No completion claim is valid until the required validator succeeds from the same canonical root used for creation.

For artifacts and code, “written” is not the same as “done.”

### No premature fallback mode

Frank must not switch into suggestion mode or optional follow-up mode before the requested bounded work has either:

- validated successfully, or
- hit a real blocker that prevents completion.

### Honest blocker reporting

If validation cannot be made green, Frank must report the blocker truthfully and must not present the task as complete.

### Persist-before-report

A step result, allow/reject decision, or approval outcome is not complete until the runtime has durably recorded it.

Final response must summarize persisted truth, not transient in-memory belief.

### No hidden step switching

Changing the active step is a control-plane action.

It must be explicit, auditable, and reflected in runtime state and status output.

### No silent unsafe resume

After restart or reboot, Frank may restore control state, but it must not silently reissue unsafe side effects.

Interrupted governed work becomes paused unless the resumed action is explicitly proven safe by validator or explicit operator approval.

### Already-satisfied work must not be recast as newly completed work

If a step is re-entered and its success criteria are already satisfied at the canonical path or target state, Frank must record that as already present / already satisfied rather than claiming it was newly created or newly executed.

### Long-running creation is separate from long-running start

Writing and validating long-running code is not the same as starting it.

Long-running code build belongs to `long_running_code`.

Process start/stop/status belongs to `system_action`.

### Discussion, wait, and final-response steps do not imply tool authority

A discussion step may answer, propose, clarify, or request approval.

A wait step may block dependent work.

A final-response step may summarize.

None of those step types grant extra execution authority by themselves.

## Mission-Control V2 Scope

Frank v2 standardizes the governed runtime loop as:

**directive -> classify -> job -> plan -> validate -> persist -> execute step -> validate step -> audit -> persist -> next state -> final response**

The mission-control layer is authoritative for governed actions.

### One-active-job rule

Frank v2 may store multiple jobs durably, but the deployed phone runtime may actively execute only one governed job at a time.

Other jobs may exist in `ready`, `waiting_user`, `paused`, `failed`, `aborted`, or `completed` states.

Frank v2 does not support parallel governed execution on the phone body.

### Supported v2 step types

The supported governed step types in v2 are exactly:

- `discussion`
- `static_artifact`
- `one_shot_code`
- `long_running_code`
- `system_action`
- `wait_user`
- `final_response`

### Planning normalization

Earlier source material sometimes treated `plan` as a step type.

Frank v2 does not.

Planning is a job phase that produces a validated `Plan` object. It is not a governed execution step.

### Governed action rule

A governed action requires all of the following:

- an active job,
- an active step,
- a legal job/step state,
- an exposed capability,
- a tool allowed by both job and step,
- sufficient authority,
- satisfied dependencies,
- no unresolved wait or pause condition,
- no unmet approval boundary,
- an allow decision from the mission-control guard,
- and an audit event.

At minimum, file writes, exec actions, long-running process start/stop, service control, outbound messaging, package updates, and network side effects are governed actions.

## Directive Intake and Mission Formation

### Intake classes

Every inbound directive must first be classified as one of:

- **Executable** — already bounded enough to plan and execute.
- **Underspecified but bounded** — needs a small amount of clarification or scoping.
- **Open-ended mission** — too vague to execute directly.
- **Discussion-only** — no artifact or action required.

### Intake rule

Frank v2 must not execute an open-ended mission directly.

If a directive is open-ended, Frank must first produce one of:

- a bounded mission proposal,
- a clarification request,
- or a ranked options memo.

### Mission proposal object

For open-ended directives, the planning layer produces a proposal containing at least:

- objective,
- scope,
- constraints,
- authority ceiling,
- success criteria,
- stop conditions,
- approval requirements,
- expected outputs.

### Default translation rule

A vague directive such as “make money,” “find a community,” or “improve yourself” is not executable by itself.

It must first be translated into bounded research, build, maintenance, or operator-reviewed options work.

Frank v2 does not directly execute public, financial, or identity-bearing side effects from vague directives.

## Mission Families

### Enabled core mission families in v2

The default-enabled mission families are:

- `build`
- `research`
- `monitor`
- `operate`
- `maintenance`

### Recognized but disabled-by-default extension families

The following family names are recognized from the source material but are not enabled by default in Frank v2:

- `outreach`
- `community_discovery`
- `opportunity_scan`
- `bootstrap_identity_and_accounts`

A plan using one of those families is valid only when an explicit policy/capability overlay enables it.

Without such an overlay, Frank v2 must reject execution with an unsupported-mission-family error or downgrade the work to discussion/research only.

### Mission-family rule

A job may reference multiple mission families, but exactly one must be marked primary.

The primary family governs default authority expectations, budget interpretation, and reporting.

## Runtime State Model

### Job states

A job may be in one of these states:

- `intake`
- `planning`
- `plan_validating`
- `ready`
- `executing`
- `step_validating`
- `waiting_user`
- `replanning`
- `paused`
- `completed`
- `failed`
- `aborted`

### Allowed job transitions

- `intake -> planning`
- `planning -> plan_validating`
- `plan_validating -> ready | waiting_user | failed`
- `ready -> executing`
- `executing -> step_validating | waiting_user | replanning | failed | paused`
- `step_validating -> ready | replanning | waiting_user | completed | failed`
- `waiting_user -> planning | ready | paused | aborted`
- `replanning -> plan_validating`
- `paused -> ready | waiting_user | aborted`

### Illegal job transitions

These are illegal:

- `executing -> completed` without step validation,
- `waiting_user -> executing` without resolving the wait condition,
- `failed -> executing` without explicit retry or replanning,
- any transition out of `aborted` except read-only inspection,
- any state transition that skips required persistence and audit.

### Step states

Each step must have an explicit status from this set:

- `pending`
- `ready`
- `executing`
- `validating`
- `waiting_user`
- `succeeded`
- `failed`
- `blocked`
- `aborted`

### Allowed step transitions

- `pending -> ready` when dependencies are satisfied and the step becomes current.
- `ready -> executing` when the runtime begins governed execution.
- `executing -> validating | waiting_user | failed | blocked`.
- `validating -> succeeded | failed | blocked | ready`.
- `waiting_user -> ready | aborted` after the wait condition resolves or the operator aborts.
- `blocked -> ready` only after explicit replan, approval, or operator step change resolves the block.
- unfinished steps may become `aborted` when the job is aborted.

### Approval states

An approval request may be in one of these states:

- `pending`
- `granted`
- `denied`
- `expired`
- `revoked`

No `pending` approval request may be treated as granted implicitly.

## Step Completion Contracts

### `discussion`

**Subtypes**

- `blocker`
- `authorization`
- `definition`
- `advisory`

**Done when**

- exactly one outbound message is produced,
- the message has a clear purpose,
- the message is concise and actionable.

If the subtype is `blocker`, `authorization`, or `definition`, the next job state is `waiting_user` or `paused` depending on operator-policy timeout.

**Forbidden**

- hidden side effects,
- performing dependent actions before a reply,
- using discussion as a substitute for execution.

### `static_artifact`

**Done when**

- the artifact exists at the exact required path,
- the required format is correct,
- required sections or keys are present,
- the artifact finalizer succeeds if applicable.

**Forbidden**

- wrapper folders unless requested,
- missing required sections,
- claiming completion without structure checks.

### `one_shot_code`

**Done when**

- the canonical project root exists,
- required files exist at exact required paths,
- validation or compile succeeds,
- the code runs exactly once if execution was requested,
- expected output or artifact is observed,
- the artifact finalizer succeeds if applicable.

**Forbidden**

- running before validation,
- skipping the finalizer,
- claiming success after a failed run,
- repeated rewrites without a specific fix.

### `long_running_code`

**Done when**

- the canonical project root exists,
- required files exist at exact paths,
- validation or compile succeeds,
- the required startup command is known,
- the long-running artifact contract is satisfied,
- the finalizer succeeds if applicable.

**Forbidden**

- starting the long-running process inside this build step,
- treating “written” as completion,
- marking the step invalid merely because the target is long-running.

### `system_action`

**Done when**

- the requested command or API action executes,
- post-action state is verified,
- resulting state is recorded,
- rollback information is recorded when possible.

**Forbidden**

- executing above the authority tier,
- executing identity/public/financial actions without approval,
- claiming success without post-state verification.

### `wait_user`

**Done when**

- a user reply,
- approval,
- rejection,
- explicit operator control decision,
- or timeout event is received and recorded.

**Rules**

- no dependent execution may proceed while waiting,
- the unresolved condition must be attached to job and step state,
- timeout handling must be explicit rather than implied.

### `final_response`

**Done when**

- completed steps are summarized truthfully,
- pending or blocked items are summarized,
- artifacts are listed,
- already-satisfied work is described accurately,
- approvals used are listed when relevant,
- no incomplete step is reported as complete.

**Forbidden**

- declaring success while a validator failed,
- hiding unresolved blockers,
- losing job state in the final answer.

## Core Runtime Abstractions

Frank v2 relies on these named contract types.

### `Job`

A durable bounded unit of work.

**Minimum contract**

- `id`
- `raw_directive`
- `primary_mission_family`
- `state`
- `working_root`
- `max_authority`
- `allowed_tools`
- `budget`
- `created_at`
- `updated_at`
- `plan_version`
- `plan`

### `Plan`

The validated decomposition of one job.

**Minimum contract**

- `id`
- `version`
- `objective`
- `constraints`
- `success_criteria`
- `stop_conditions`
- `steps`

### `Step`

One typed unit of governed work inside a plan.

**Minimum contract**

- `id`
- `type`
- `subtype`
- `goal`
- `depends_on`
- `required_capabilities`
- `required_authority`
- `allowed_tools`
- `requires_approval`
- `artifact_targets`
- `validator`
- `on_failure`
- `status`

### `Step.artifact_targets`

`artifact_targets` is the authoritative per-step artifact declaration.

It must be present on every step, and it must be an array.

Each entry must be an object containing at least:

- `id`
- `path`
- `artifact_type`
- `required`

**Rules**

- `id` must be unique within the step.
- `path` must be the exact canonical path used for writes, reads, validation, artifact registry, and already-satisfied checks.
- `artifact_type` is a stable string label used by the validator and artifact registry. Implementations may extend the label set, but the stored value is contract data rather than prose.
- `required=true` means the step cannot complete successfully unless the artifact validates at that exact path.
- Steps that do not produce durable artifacts must set `artifact_targets` to `[]`.
- A validator may reference only declared `artifact_targets[].id` values.

### `Step.validator`

`validator` is the authoritative per-step validation contract.

It must be present on every step, and it must be an object containing exactly:

- `kind`
- `rules`
- `require_finalizer`

`kind` must be exactly one of:

- `discussion`
- `static_artifact`
- `one_shot_code`
- `long_running_code`
- `system_action`
- `wait_user`
- `final_response`

`rules` must be an array of rule objects.

Each rule object must contain:

- `rule`
- `target`
- `value`

`rule` must be one of:

- `exists`
- `structure`
- `command_exit_zero`
- `post_state`
- `startup_command_known`
- `input_recorded`
- `response_present`

`target` must be either:

- one declared `artifact_targets[].id`,
- `job`,
- `step`,
- `runtime`,
- or `message`.

`value` is validator-rule-specific contract data and must be structured enough for deterministic checking. Free-form prose alone is non-compliant.

`require_finalizer` is a boolean. When `true`, step success requires the named finalizer or equivalent final validation stage to succeed before the step may become complete.

**Rules**

- `validator.kind` must match the step type, except that the validator name remains `wait_user` while the runtime state remains `waiting_user`.
- `discussion` and `final_response` validators must use only non-side-effect rules.
- `system_action` validators must include at least one `post_state` rule.
- `long_running_code` validators must include at least one `startup_command_known` rule and must not imply process start.
- `static_artifact` and `one_shot_code` validators that reference artifacts must reference declared targets rather than inferred paths.

### `ApprovalRequest`

An explicit request for authorization when a step reaches an approval boundary.

**Minimum contract**

- `approval_id`
- `job_id`
- `step_id`
- `scope`
- `requested_action`
- `required_authority`
- `reason`
- `fallback_if_denied`
- `requested_via`
- `created_at`
- `expires_at`
- `state`

### `ApprovalGrant`

The durable record that a prior request was granted.

**Minimum contract**

- `approval_id`
- `request_id`
- `granted_scope`
- `granted_by`
- `granted_via`
- `constraints`
- `granted_at`
- `expires_at`
- `revoked_at`

An `ApprovalGrant` binds only to the specific `ApprovalRequest` it names.

It must not authorize a different job, different step, different channel scope, or materially different requested action.

### `AuditEvent`

The durable allow/reject record for a governed action or state-changing control event.

**Minimum contract**

- `job_id`
- `step_id`
- `proposed_action`
- `allowed`
- `error_code`
- `reason`
- `timestamp`

`event_id`, `action_class`, `tool_name`, and `result` may exist as additive implementation fields, but they must not replace the minimum persisted field names above.

### `AuditEvent` taxonomy expectations

- `proposed_action` is the canonical attempted governed action string for the audited event.
- `allowed=false` means the action was rejected before execution and therefore requires both `error_code` and `reason`.
- `allowed=true` means the guard permitted the action attempt. In that case `error_code` should normally be empty.
- The persisted field name is `error_code`.
- `error_code` must be a stable machine-readable taxonomy key and must map one-to-one to the rejection classes frozen in [Rejection Codes](#rejection-codes).
- Implementations may add additional audit enrichment fields or additional rejection codes, but they must not rename, silently remap, or overload the frozen baseline semantics.

### `ArtifactRecord`

The durable registry entry for produced artifacts.

**Minimum contract**

- `path`
- `artifact_type`
- `producing_step_id`
- `validation_status`
- `content_fingerprint`
- `updated_at`

### `CapabilityRecord`

The durable description of an exposed runtime capability.

**Minimum contract**

- `capability_id`
- `class`
- `name`
- `exposed`
- `authority_tier`
- `validator`
- `notes`

### `MissionStatusSnapshot`

The operator-facing runtime status snapshot.

**Minimum contract**

- `mission_required`
- `active`
- `mission_file`
- `job_id`
- `step_id`
- `step_type`
- `required_authority`
- `requires_approval`
- `allowed_tools`
- `updated_at`

**Optional live field**

- `runtime`

**Rules**

- `allowed_tools` is the deterministic sorted unique intersection of job-level and step-level tool scope.
- When `active=true`, `job_id`, `step_id`, `step_type`, `required_authority`, and `requires_approval` must describe the active governed step.
- When `active=false`, `job_id` and `step_id` may still reflect the persisted runtime candidate chosen for inspection or resume.
- When present, `runtime` mirrors the persisted runtime-state object and is operational data rather than the authoritative source of plan truth.

The status snapshot is for inspection and control confirmation. It is not the sole source of truth.

### `MissionStepControl`

The operator-authored control-file request used to switch the active step inside a bootstrapped mission.

**Minimum contract**

- `step_id`
- `updated_at`

Implementation may include additional operator metadata, but the runtime must preserve the simple control-file behavior established in the current repo evidence.

## Plan Validation Rules

A plan is invalid if any of these are true:

- step IDs are duplicated,
- dependencies form a cycle,
- a step depends on a non-existent step,
- no terminal `final_response` exists,
- the final `final_response` step is not terminal,
- any step uses an unsupported v2 step type,
- any step exceeds the job authority ceiling,
- any step names a capability not exposed by the body,
- any step includes a tool not present in the job allowed-tools set,
- a `wait_user` step appears without a reason,
- success criteria cannot be checked,
- a file-creating step has no canonical working root or target path discipline,
- a long-running process is planned to start inside `long_running_code` rather than `system_action`,
- a public, identity, or financial action is planned without an approval boundary,
- a plan references a disabled mission family without an enabling policy overlay.

No job may enter execution until plan validation passes.

## Persistence, Replay, and Idempotence

### Source of truth

Frank v2 uses durable records plus audit history as the source of truth.

The mission status snapshot is a derived operator-facing view.

The step control file is operator intent, not source of truth.

### Durable records required

The runtime must durably preserve at least:

- current job state,
- current step state,
- validated plan version,
- approval requests and outcomes,
- artifact registry,
- audit events,
- current canonical working root.

### Write atomicity and ordering

Every durable mutation to job, step, approval, audit, or artifact state must be crash-safe at record granularity.

At minimum:

- a reader must observe either the old complete record or the new complete record, never a partially encoded record,
- file-backed persistence must write a replacement file in the same directory and atomically replace the prior file,
- transactional backends must provide an equivalent all-or-nothing visibility guarantee for each committed record write,
- a denial, pause, block, or failure must durably record its `AuditEvent` before the runtime reports the resulting job or step state to the operator,
- a grant, deny, revoke, or expiry must be durable before any dependent governed action may start or resume,
- artifact registry updates must be durable before a step may be recorded as `succeeded` or `already_satisfied`,
- when job state and step state cannot be committed in one physical transaction, step state commits first, job state commits second, and recovery must prefer the more conservative interpretation of any torn pair,
- final response content must be derived only from already-committed durable records.

### Replay safety

Replaying the same startup sequence, control-file apply, or operator command must not duplicate already successful side effects.

A replay must first check current persisted state and validator truth.

### Idempotence rule by step family

- `discussion`: never auto-resend a prior discussion message after restart.
- `final_response`: never auto-resend a final response after restart.
- `static_artifact`: if the target artifact already exists at the canonical path and validates against the current step contract, mark the step satisfied as already present instead of rewriting.
- `one_shot_code`: if code and requested outputs already validate under the current step contract, mark satisfied as already present instead of rerunning.
- `long_running_code`: if the long-running artifact already validates and startup metadata is still correct, mark satisfied as already present instead of rebuilding.
- `system_action`: before reapplying, verify the desired post-state. If already true, record `already_present` or equivalent and do not reissue the action.

### Restart behavior

On process restart or phone reboot:

- `completed` jobs remain `completed`.
- `aborted` jobs remain `aborted`.
- `waiting_user` jobs remain `waiting_user`.
- interrupted `executing` and `step_validating` jobs become `paused`.
- managed long-running services may keep running only under their own service policy, but the job itself is still `paused` until operator resume or explicit safe restoration.
- startup may restore the active step selection from the existing mission step control file before writing the first status snapshot.

### Startup and resume arbitration

For deployed startup with `mission-required` and one configured `mission-file`, the one-active-job rule is arbitrated in this order:

1. Load the job from `mission-file`.
2. Load any persisted runtime candidate from the mission status snapshot or equivalent durable runtime store.
3. Ignore the persisted runtime candidate if it is absent, terminal, or names a different `job_id` than the loaded mission file.
4. If a same-job non-terminal persisted runtime candidate exists, it is the sole resume candidate.
5. If that resume candidate names an `active_step_id` that differs from startup `mission-step`, startup must fail rather than activate a competing step.
6. If that resume candidate exists and explicit resume approval is absent, startup must fail with a resume-required error rather than activate a second runtime.
7. If that resume candidate exists and explicit resume approval is present, startup resumes that runtime and does not separately activate `mission-step`.
8. Only when no eligible same-job persisted runtime candidate exists may startup activate `mission-step` directly from the mission file.
9. A valid step-control file may change the selected step only after the single active job has been chosen and only before the first status snapshot is written.

Startup must never create two concurrently active governed job runtimes in order to “merge” mission-file state with persisted runtime state.

### Status snapshot rewrite rule

The mission status snapshot must be rewritten after any successful step switch and after any state transition that changes the active runtime step.

### Step-switch confirmation rule

When the operator uses the step-switch control path with status confirmation enabled, success requires a fresh matching status snapshot.

At minimum, confirmation must require:

- `active=true`,
- matching `job_id`,
- matching `step_id`,
- matching `step_type`,
- matching `allowed_tools`,
- and a fresh `updated_at` when a prior snapshot existed.

A stale matching snapshot must not count as confirmation.

## Authority Model

Frank v2 uses ordered authority tiers.

### Tier A — Observe

Allowed without approval:

- inspect state,
- read allowed files,
- list directories,
- inspect logs,
- inspect process and service status,
- query external sources,
- draft outputs without sending them.

### Tier B — Prepare

Allowed without approval:

- create files,
- edit files in approved workspace,
- generate code,
- generate documents,
- stage outputs for review,
- validate artifacts locally,
- author plans and plan revisions.

### Tier C — Execute Low-Risk

Allowed without approval if the mission permits:

- run one-shot scripts,
- validate long-running scripts without starting them,
- start, stop, or inspect approved benign local services,
- continue an already approved bounded workflow,
- send messages only to the owner through approved operator channels,
- perform approved userland maintenance actions with rollback protection.

### Tier D — Approval Required

Requires explicit approval each time:

- owner-identity actions,
- public posting,
- messaging strangers,
- community joining or posting,
- external service exposure,
- destructive filesystem or system operations,
- materially new capability installation,
- security-boundary changes,
- package or update actions outside the approved userland policy,
- any financial action beyond research-only discussion.

### Tier E — Forbidden by Default

Not allowed unless the system is explicitly redesigned:

- irreversible destructive actions without explicit task-level authorization,
- uncontrolled self-modification of the core runtime,
- uncontrolled credential exposure,
- uncontrolled public network exposure,
- silent owner impersonation,
- unrestricted autonomous money movement,
- vague-objective execution without mission decomposition,
- silent surveillance-like sensor usage.

## Capability Registry and Tool Gating

A phone capability is not usable merely because the phone physically has it.

A capability is usable only if:

- it is exposed to the runtime,
- it is covered by authority rules,
- it is covered by a mission and step contract,
- and it survives plan validation.

### Default exposed capability classes in v2

At minimum, Frank v2 may expose these capability families:

- workspace file read/write/list,
- userland exec,
- Python validate/run,
- HTTP requests,
- logs read,
- approved local service start/status/stop,
- Telegram owner-channel read/write,
- mission inspection/status/control surfaces.

### Default non-exposed capability classes in v2

Not exposed by default:

- contacts,
- SMS or phone calls,
- microphone,
- camera,
- location,
- Bluetooth peripheral control,
- shared storage,
- other apps’ private data,
- APK installs,
- public social posting,
- account sign-up flows,
- money movement.

### Allowed-tools semantics

For execution gating, tool permission is set membership.

A step may use a tool only if that tool appears in both:

- `job.allowed_tools`, and
- `step.allowed_tools`.

For operator-facing status output, `allowed_tools` must be rendered deterministically as the sorted unique intersection of those two sets.

The mission inspection surface may preserve original source order for human review, but gating itself must remain deterministic.

## Approval and Operator Control Plane

### Approved operator channels

Frank v2 freezes these operator-control channels:

- Telegram
- SSH/Tailscale terminal

Additional channels may be added later, but they are not required for v2.

### Approval scopes

Frank v2 supports these approval scopes:

- one step,
- one job,
- one session until expiry.

The default scope is one step.

### Approval request content

Every approval request must clearly restate:

- the proposed action,
- why it is needed,
- the authority tier,
- any identity or public scope it touches,
- any filesystem, process, or network side effect,
- the fallback if denied.

### Approval binding rule

An approval request binds to all of the following simultaneously:

- the exact `job_id`,
- the exact `step_id`,
- the exact `requested_action`,
- the approval `scope`,
- and the operator channel scope that received the request.

For v2, operator channel scope means:

- Telegram: the exact approved chat or thread where the request was sent,
- SSH/Tailscale terminal: the authenticated terminal session or explicit CLI invocation that issued the resolution.

A reply from a different channel does not bind implicitly.

Cross-channel resolution is valid only when the operator uses explicit control syntax naming the target job and step, and the resolving channel is itself an approved operator channel.

`requested_via` and `granted_via` must record the approved channel actually used.

If the transport exposes stable operator identity or conversation metadata, the runtime must preserve that binding in the approval record or its durable envelope even if the minimum contract does not name the extra fields.

### Explicit control syntax

Frank v2 freezes this minimum operator syntax:

```text
APPROVE <job_id> <step_id>
DENY <job_id> <step_id>
PAUSE <job_id>
RESUME <job_id>
ABORT <job_id>
STATUS <job_id>
SET_STEP <job_id> <step_id>
```

The CLI equivalents may use subcommands and flags rather than this exact text, but they must preserve the same control semantics.

### Natural-language yes/no rule

A plain-language “yes” or “no” is valid only when:

- exactly one unresolved approval request is pending in the relevant operator channel scope,
- the reply arrives in that same operator channel scope,
- the reply is unambiguous,
- and the request has not expired, been revoked, or been superseded.

If multiple approval requests are pending, a plain “yes” or “no” is ambiguous and must not bind.

### Timeout and exhaustion rule

If an approval is required and approved operator channels do not produce a resolution before timeout or exhaustion, the runtime must:

- leave a durable pending-approval record,
- notify the operator of the pause condition if possible,
- move the job to `paused` or keep it in `waiting_user` according to the step contract,
- and perform no dependent action.

### Operator override rule

`SET_STEP` and the corresponding control-file mechanism are operator/admin controls.

They may only switch to a step that exists in the validated mission plan.

They do not bypass plan validation, approval requirements, or step completion contracts.

## Identity Boundary

Frank v2 recognizes these identity modes:

- `owner_only_control`
- `draft_as_owner`
- `send_as_owner_with_approval`
- `agent_alias`

### Default identity posture

Frank operates as Frank inside the runtime and operator-control surfaces.

Frank does not silently act as the owner.

### Owner identity rule

Any action using owner identity requires:

- an explicit step that names the identity boundary,
- explicit approval,
- explicit channel or scope,
- and durable audit.

### Agent-alias rule

`agent_alias` is recognized as a runtime identity mode, but Frank v2 does not make public autonomous outreach a default-enabled mission family.

Recognizing the identity mode does not enable public execution by default.

## Data Scope and Workspace Discipline

### Default writable scope

Frank v2 writable scope is:

- workspace,
- agent-managed logs,
- agent-managed configs and state.

### Default readable scope

Frank v2 readable scope is:

- workspace,
- logs,
- approved control-channel inputs,
- approved service and process state,
- explicitly allowed mission files and status/control files.

### Default blocked scope

Blocked by default:

- shared phone storage,
- contacts,
- photos,
- messages,
- browser profiles,
- other apps’ private data,
- sensor streams.

### Workspace discipline

Broad filesystem drift is non-compliant.

If a task requires broader readable or writable scope, the plan must state that explicitly and the step must pass the normal authority and approval rules.

## Maintenance and Update Policy

Frank v2 includes bounded userland maintenance inside `maintenance` and `system_action`.

### Allowed autonomously inside approved scope

With no extra approval, Frank v2 may:

- install or update Termux packages inside the approved userland policy,
- install or update Python dependencies,
- update workspace-managed code,
- update helper scripts,
- perform dependency hygiene.

### Required safeguards

Before an autonomous update/install, Frank must:

- create a restore manifest,
- record current versions,
- record relevant config files,
- record changed files,
- define smoke tests.

After update/install, Frank must:

- run smoke tests,
- rollback if smoke tests fail,
- emit a failure notification if rollback is required.

### Approval-required maintenance actions

Approval remains required for:

- arbitrary APK installs,
- root or system-level changes,
- broad Android app installs,
- permission-model changes,
- security-boundary changes.

## Deployment Model

Frank v2 is deployed on an Android phone using a long-running Termux-based runtime.

Expected deployment shape:

- Android phone as dedicated body,
- Termux and Termux:Boot or equivalent launch wiring,
- long-running gateway mode,
- remote model provider,
- Telegram-first text control,
- minimized phone-side permissions,
- no shared-storage exposure unless explicitly needed,
- mission-required startup in deployed mode,
- mission bootstrap via mission file plus active step,
- mission status snapshot output,
- mission step control file for operator step switching.

Environment parity means aligned code and intentional config differences, not identical host environments.

## Logging, Notifications, and Budgets

### Logging

Frank v2 requires:

- continuous logs,
- daily log packaging,
- log packaging on reboot.

### Retention defaults

- log packages: 90 days,
- audit records: 90 days,
- approval records: 180 days,
- artifact metadata: 180 days.

### Mandatory notifications

Frank v2 must emit notifications for:

- blockers,
- completions,
- approval requests,
- failures.

### Periodic notifications

Frank v2 defaults to:

- check-ins every 30 minutes on long jobs,
- a daily summary every 24 hours.

### Aggressive-but-bounded defaults

Default ceilings for unattended work:

- max unattended wall-clock per job: 4 hours,
- max replans per job: 5,
- max failed actions before pause: 5,
- max owner-facing messages per job: 20,
- max pending approvals per job: 3.

### Budget exhaustion rule

When a job hits any enforced budget ceiling, the runtime must:

- stop dependent execution,
- persist the budget-exhausted state,
- emit an explicit audit event and blocker,
- and move the job to `paused` or `failed` according to policy.

### Deterministic truncation rule

If a status or final-response view must truncate lists because of budget or message size limits:

- mandatory blocker and approval information comes first,
- then current active step/state,
- then artifacts in plan order and lexicographic path order,
- then older audit details.

Truncation must never hide the reason a job is blocked or whether validation passed.

## Operator Interfaces to Standardize

Frank v2 freezes these public operator concepts:

- Frank as the deployed personal-operator identity,
- desktop lab / phone body as the operating model,
- mission-required execution,
- mission bootstrap from a job JSON file plus active step,
- runtime mission status snapshot JSON,
- operator mission inspection,
- operator step switching through a control file and status confirmation,
- explicit approval / deny / pause / resume / abort semantics.

### Current CLI surfaces evidenced in source material

The current runtime evidence shows these operator-facing surfaces already exist or are in active use:

- startup flags such as `--mission-required`, `--mission-file`, `--mission-step`, `--mission-status-file`, and `--mission-step-control-file`,
- `picobot mission inspect`,
- `picobot mission status`,
- `picobot mission set-step`.

Frank v2 freezes these as the minimum runtime-control surface for deployed operation.

Internal storage backends and helper names may vary so long as those control semantics remain intact.

## Constraints and Guardrails

### Fail closed

When mission gating is required, the runtime fails closed.

Unsupported or ungated actions are rejected, not improvised.

### No policy bypass

No file write, exec path, service action, outbound message, or update path may bypass the mission-control guard.

### No hidden acting as owner

Frank must not silently act under the owner’s identity.

### No execution while waiting

While a job or step is waiting for user or approval, dependent execution must not proceed.

### No step start outside validated mission

A step may start only if:

- it exists in the validated plan,
- dependencies are satisfied,
- the operator has not paused or aborted the job,
- and no required approval is missing.

### No unsupported mission-family execution

Disabled-by-default mission families may be discussed or researched, but not executed, unless a policy/capability overlay explicitly enables them.

## Rejection Codes

Frank v2 requires machine-readable rejection codes at least for these classes:

- `E_NO_JOB`
- `E_NO_ACTIVE_STEP`
- `E_STEP_OUT_OF_ORDER`
- `E_CAPABILITY_NOT_EXPOSED`
- `E_AUTHORITY_EXCEEDED`
- `E_APPROVAL_REQUIRED`
- `E_INVALID_PATH`
- `E_INVALID_ACTION_FOR_STEP`
- `E_FINALIZER_REQUIRED`
- `E_WAITING_FOR_USER`
- `E_ABORTED`
- `E_LONGRUN_START_FORBIDDEN`
- `E_IDENTITY_SCOPE_REQUIRED`
- `E_FINANCIAL_SCOPE_REQUIRED`
- `E_PLAN_INVALID`
- `E_VALIDATION_FAILED`
- `E_BUDGET_EXHAUSTED`
- `E_ALREADY_PRESENT`
- `E_RESUME_REQUIRES_APPROVAL`
- `E_UNSUPPORTED_MISSION_FAMILY`

The persisted `AuditEvent.error_code` field must map one-to-one to these rejection classes or to a documented additive extension of them.

Implementation may extend this set, but these failure classes must be explicit and auditable.

## Day-1 Execution Envelope

### Supported by default in Frank v2

- discussion-only missions,
- markdown/text/config/static artifact creation,
- one-shot code generation and validation,
- long-running code generation and validation,
- start/stop/status for approved benign local services,
- bounded research, monitoring, and maintenance,
- operator-controlled pause/resume/abort/step-switch.

### Explicitly not default-enabled in Frank v2

- public posting,
- stranger outreach,
- community joining/posting/DM,
- account bootstrap,
- money movement,
- sensor access,
- voice-native assistant behavior.

## Acceptance Criteria

Every criterion below must be reviewable or testable.

### Runtime-control criteria

1. **No governed action without active job and step**  
   A governed action is rejected when no active job and active step are present.

2. **Every allow/reject decision is audited**  
   Every governed action attempt emits an `AuditEvent` with explicit allow/reject outcome and code.

3. **Unsupported step types are rejected**  
   Any step outside the seven frozen v2 step types is rejected.

4. **Planning is required before execution**  
   No non-trivial job enters execution without a validated plan.

5. **Illegal state transitions are rejected**  
   Illegal job or step transitions fail deterministically.

6. **One-active-job rule is enforced**  
   The phone runtime does not execute two governed jobs in parallel, and startup does not activate a second runtime when an eligible same-job persisted runtime candidate already exists.

7. **Step switching is explicit**  
   Runtime step switching occurs only through the control path and results in a rewritten status snapshot.

8. **Status confirmation requires freshness**  
   A stale matching status snapshot does not confirm a step-switch request.

9. **Restart pauses unsafe work**  
   Interrupted executing jobs become paused after restart and do not silently continue unsafe side effects.

10. **Startup restore happens before the first status snapshot**  
    If a valid step control file exists after the active job is chosen, the initial status snapshot reflects the restored step.

11. **Approval ambiguity does not bind**  
    A plain-language “yes” or “no” does not bind when multiple unresolved approval requests exist or when the reply arrives outside the original channel scope.

12. **Budget exhaustion is explicit**  
    Hitting a budget ceiling yields a persisted blocker and audit event rather than silent truncation or drift.

13. **Durable writes are not partially visible**  
    Job, step, approval, audit, and artifact writes expose either the old complete record or the new complete record, never partial data.

### Behavior-contract criteria

14. **Canonical root discipline is enforced**  
    A file-creating job that writes into multiple inconsistent roots is non-compliant.

15. **Static artifacts validate before completion**  
    A `static_artifact` step is complete only when the artifact exists at the exact required path and validates structurally.

16. **One-shot code validates before completion**  
    A `one_shot_code` step is complete only when code exists at the exact required paths and requested validation succeeds.

17. **Long-running build and start are separate**  
    A `long_running_code` step does not start the process. Starting or stopping it requires `system_action`.

18. **System actions verify post-state**  
    A `system_action` step is not complete until post-action state is verified.

19. **Wait steps block dependent work**  
    No dependent step executes while a `wait_user` condition is unresolved.

20. **Already-satisfied work is reported truthfully**  
    Re-entering a satisfied step does not create a false “newly completed” claim.

21. **Final response is truthful**  
    `final_response` summarizes completed, blocked, and already-satisfied work without overstating success.

22. **No premature fallback**  
    Frank does not pivot into suggestion mode before completing or honestly blocking on the requested task.

23. **Voice-native requests remain out of scope**  
    A request for voice-native assistant behavior is explicitly handled as outside Frank v2 scope.

24. **Public/community/treasury actions remain disabled by default**  
    A request for outreach, community participation, autonomous accounts, or treasury behavior is rejected or downgraded unless an explicit policy overlay enables it.

## Review and Acceptance Scenarios

1. **Discussion-only mission**  
   A discussion-only mission with zero tools can respond but cannot execute governed tools.

2. **Static artifact mission**  
   A static-artifact mission creates the artifact at one canonical path and validates its existence and structure before completion.

3. **One-shot code mission**  
   A one-shot-code mission writes code, validates it, runs exactly once if requested, and does not claim success on failed compile or run.

4. **Long-running code mission**  
   A long-running-code mission writes and validates a service or daemon but does not start it until a later `system_action` step.

5. **System-action mission**  
   A system-action mission starts or stops an approved local service, verifies post-state, records rollback information if possible, and audits the action.

6. **Approval-required step**  
   A step that crosses Tier D pauses for approval, records the request durably, and executes only after explicit approval from an approved operator channel.

7. **Ambiguous yes/no reply**  
   Two pending approval requests exist. The operator replies “yes.” The runtime does not bind the reply and keeps both requests unresolved.

8. **Restart safety**  
   A reboot during an executing step pauses the job. On restart, the runtime restores control state and writes the correct initial status snapshot but does not silently reissue unsafe side effects.

9. **Freshness on set-step confirmation**  
   The operator writes a step-switch control file with status confirmation enabled. A stale matching snapshot is ignored; only a fresh matching snapshot confirms success.

10. **Already-present artifact**  
    A static artifact already exists and validates at the canonical path. Re-entering the step records it as already present rather than newly created.

11. **Budget exhaustion**  
    A job hits its owner-message budget. The runtime persists a budget blocker and pauses rather than silently dropping required blocker information.

12. **Unsupported extension family**  
    A plan uses `community_discovery` with no enabling overlay. The runtime rejects execution and explains that the family is recognized but disabled by default.

13. **Path drift rejection**  
    A task creates files in multiple inconsistent roots. The job is not marked done.

14. **Owner-identity action request**  
    A request to send as the owner pauses for explicit approval and does not silently proceed.

15. **Voice-native request**  
    A request for wake-word or phone-native voice-assistant behavior is explicitly treated as outside Frank v2 scope.

16. **Startup arbitration**  
    A same-job non-terminal persisted runtime exists, `mission-step` is also supplied, and resume approval is absent. Startup fails with a resume-required error rather than activating a second runtime or silently choosing one.

## Non-Blocking Implementation Choices

The following are implementation choices, not spec blockers, so long as the external behavior remains compliant:

- JSON files versus SQLite for durable storage,
- internal enum names,
- exact package layout,
- exact provider adapter shape,
- exact log packaging format,
- exact status snapshot file path.

## Future Scope

The following remain outside the frozen Frank v2 default envelope:

- voice and richer embodied interfaces,
- public/community/outreach policy overlays,
- autonomous account bootstrap,
- treasury and money movement,
- broader phone-native sensor exposure,
- multi-device or multi-agent orchestration,
- parallel governed execution on the phone,
- unrestricted service exposure.

## Appendix A — Editor Note

This freeze candidate was lowered from:

- `docs/FRANK_V1_SPEC.md` as the narrow v1 baseline,
- `1.txt` as the primary mission-control and runtime-state source,
- `2.txt` and `3.txt` as deployment, behavior-hardening, and policy sources.

Normalization notes:

- planning is normalized to a job phase rather than a governed step type,
- extension families from broader policy drafts remain recognized but disabled by default,
- runtime evidence for mission bootstrap, mission status snapshots, step control files, step-switch confirmation, and startup restore is promoted into the v2 control-plane baseline,
- the v1 canonical-path rule is strengthened into a job-level canonical working-root requirement,
- restart and replay behavior is expressed as explicit persistence and arbitration rules.

This appendix is provenance-only editorial context. It is not part of the normative runtime contract.

## Assumptions and Defaults

- This file is a frozen engineering spec, not a PRD, manifesto, or prompt pack.
- Intended repo location is `docs/FRANK_V2_SPEC.md`.
- The spec reflects the source material and its logged repo/runtime evidence without claiming that every v2 behavior is already implemented.
- `1.txt` is treated as the primary source for mission-control direction and runtime-control evidence.
- `2.txt` and `3.txt` are treated as the main deployment, behavior-hardening, and policy sources.
- Remote inference remains the default provider model.
- Telegram and SSH/Tailscale terminal are the only required operator channels in v2.
- The deployed phone runtime remains text-first.
