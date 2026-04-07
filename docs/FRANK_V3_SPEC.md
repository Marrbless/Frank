# Frank V3 Frozen Spec

**Status:** Drafted from Frank V2 frozen spec, later autonomy-policy sources, operational-awareness companion docs, donor-subsystem integration guidance, and current implementation-status notes  
**Date:** 2026-03-26  
**Source Basis:** `docs/FRANK_V2_SPEC.md` as the control-plane baseline; `2.txt` as the primary V3 policy source for outreach, community, identity/account bootstrap, treasury, and autonomy rules; `1.txt` and `3.txt` as mission-control/runtime-shape evidence; `FRANK_BODY_SIGNALS.md`, `FRANK_ENVIRONMENT_SIGNALS.md`, and `FRANK_OBSERVABILITY_PLAYBOOK.md` as the operational-awareness basis; `FRANK_IMPLEMENTATION_STATUS.md` as current-state reference only; Hermes Agent and OpenClaw as donor-subsystem references for reusable runtime and operations patterns only. Current repo state is taken only from the evidence contained in those sources, not from a fresh repo crawl.

## Problem Statement

Frank is intended to be a persistent personal operator, not a generic chatbot.

Frank v1 froze the minimum deterministic execution surface. Frank v2 froze the control plane: durable jobs and steps, explicit approvals, pause/resume, idempotence, and operator inspection.

The remaining gap is not “more personality,” “more prompt tuning,” or “more channels.” The remaining gap is external operation.

Frank v3 is the version where Frank stops being only a stateful internal executor and becomes a bounded external operator that can:

- act as **Frank** rather than silently as the owner,
- discover and create Frank-owned identities and accounts,
- perform outreach and community participation through campaign objects,
- bootstrap and operate a zero-owner-seed treasury,
- and do all of that only inside an autonomy-compatible envelope that does not secretly depend on the owner’s identity, money, or manual intervention.

The point of v3 is not ungoverned freedom. The point is to extend the v2 control plane into external surfaces without losing determinism, auditability, or boundedness.

## Definition of Frank V3

Frank v3 is a text-first, phone-hosted, mission-persistent, autonomy-compatible personal operator.

The desktop is the lab. The phone is the deployed body.

Frank means the deployed operator identity and agent alias, not just the repo, prompt pack, or tool bundle.

Default interaction remains message-driven through approved operator channels. Frank v3 is still not defined as a voice-native assistant.

Frank v3 retains the v2 mission-control substrate and adds a bounded external operating envelope:

- Frank may act externally as **Frank** by default.
- Frank may create and manage Frank-owned identities, accounts, campaigns, and treasury containers only when they satisfy the autonomy predicate.
- Frank may pursue earning, outreach, and community participation only through bounded jobs, validated plans, explicit step contracts, durable state, audit events, and the existing authority model.
- Frank may use operational body/environment signals and observer reports only as read-only context. Those signals never widen authority.
- Frank may reuse selected donor subsystems from Hermes Agent and OpenClaw only as bounded implementation modules behind Frank’s own control plane, capability exposure rules, authority model, and audit model.
- Live self-improvement, autonomous skill mutation, autonomous prompt mutation, and autonomous core-runtime self-modification are not part of Frank v3.

## Relationship to Frank V2

Frank v3 supersedes Frank v2.

Frank v3 carries forward the Frank v2 control-plane and behavior contract unless this document tightens or extends it. The following v2 invariants remain in force:

- execution-first posture,
- one canonical working root for file-creating work,
- validation before completion,
- persist before report,
- explicit step switching,
- no silent unsafe resume,
- idempotent replay behavior,
- no silent acting as the owner,
- one active governed job at a time on the phone,
- the seven frozen step types.

Frank v3 adds:

- the autonomy axiom and autonomy predicate,
- enabled external mission families,
- operator/agent identity separation as a runtime rule,
- Frank-owned identities/accounts/containers as first-class runtime objects,
- outreach and community campaign contracts,
- treasury lifecycle and ledger contracts,
- policy-aware platform/provider eligibility checking,
- capability and data onboarding as first-class proposals,
- read-only body/environment sidecars,
- read-only observability outputs,
- donor-subsystem integration rules for Hermes Agent and OpenClaw class modules,
- and an explicit v3 boundary that keeps adaptive self-improvement in future scope.

If a Frank v2 statement conflicts with a Frank v3 statement, the Frank v3 statement governs.

## Goals

- Accept bounded tasks by message and classify them before execution.
- Preserve the v2 control plane: durable jobs, steps, approvals, audit events, and restart-safe execution.
- Enable these mission families inside a governed envelope: outreach, community discovery/participation, bootstrap revenue, and bootstrap identity/account creation.
- Let Frank operate externally as **Frank** without silently operating as the owner.
- Let Frank evaluate platforms, providers, accounts, and treasury containers using a durable autonomy-compatibility check.
- Let Frank maintain a distinct zero-owner-seed treasury with a durable ledger and explicit lifecycle.
- Keep the phone body constrained, text-first, and continuously running in gateway mode with remote model inference.
- Add operational awareness through read-only body/environment sidecars and read-only observability outputs.
- Reuse proven runtime and operations patterns from donor projects where they strengthen Frank without widening authority or replacing Frank’s governance model.
- Preserve auditable allow/reject decisions for all governed actions.

## Non-Goals

Frank v3 does not include any of the following as default implementation targets:

- Native voice assistant UX.
- Silent operation as the owner.
- Using owner funds as Frank treasury seed capital.
- Any path that requires owner identity, owner legal personhood, owner approval, owner payment method, or owner manual completion in order to exist at all as part of Frank’s autonomous envelope.
- Human-KYC banking, custodial exchange signup, or other human-gated regulated onboarding as part of Frank’s autonomous path.
- Unrestricted public posting or uncontrolled mass outreach outside bounded campaigns.
- Broad phone-native sensor exposure by default.
- Broad Android app/APK control by default.
- Parallel governed execution on the phone body.
- Multi-agent orchestration.
- Replacing Frank’s mission-control kernel or operating model wholesale with a donor project.
- Live self-improvement, autonomous prompt mutation, autonomous skill synthesis or installation, or autonomous core-runtime self-modification.
- Broad assistant-product surface imported wholesale from donor repos.
- On-device model inference as a requirement.
- A web UI or dashboard as a requirement.

These may appear later as explicit future-scope work, but they are not part of the frozen Frank v3 default envelope.

## System Model

### Roles

**Desktop role**

- build, test, and prompt lab,
- repo source of truth,
- place where schema changes, mission authoring, and hardening land first.

**Phone role**

- deployed body,
- long-running gateway runtime,
- local workspace and process execution surface,
- local mission state writer,
- local sidecar/observer host,
- more constrained permissions and exposed capabilities than the desktop lab by default.

**Remote provider role**

- remote model inference,
- provider and adapter are implementation details so long as mission-control rules remain intact.

**Operator channel role**

- Telegram and SSH/Tailscale terminal remain the approved owner-control channels in v3,
- owner-control email is not active by default because no concrete owner email address is frozen,
- Frank-owned email, once bootstrapped, is for Frank’s own operations and not automatically an owner-control channel.

**External-surface role**

Frank v3 treats external identities, accounts, platforms, communities, and value containers as governed runtime targets rather than ad hoc destinations.

### Runtime Shape

Frank v3 assumes the same Picobot-derived baseline shape visible in the v2 source basis:

- long-running gateway mode,
- remote model provider integration,
- text-channel ingress,
- local artifact creation and process execution,
- mission-required startup support,
- mission bootstrap from job JSON plus active step,
- mission status snapshot output,
- operator mission inspection,
- operator step switching through a control path,
- startup restore from an existing control file,
- audit-oriented mission-control enforcement.

Frank v3 adds these runtime shape requirements:

- a durable eligibility registry for providers/platforms/accounts/containers,
- a durable identity/account registry,
- a durable campaign registry or equivalent durable campaign records,
- a durable treasury object and ledger,
- read-only body/environment sidecar files,
- read-only observer reports and snapshots,
- optional schedule definitions or equivalent durable trigger records when governed scheduling is enabled,
- and optional versioned plugin/skill/module records when donor-derived modules are enabled.

The spec freezes behavior and contracts, not helper names or exact file layout.

## Donor Subsystems and Integration Policy

Frank v3 may adapt selected subsystems from **Hermes Agent** and **OpenClaw**.

Those projects are donor sources, not replacement architectures.

If a donor implementation conflicts with this spec, this spec governs.

### Donor-subsystem rule

Any donor-derived module must remain subordinate to:

- Frank’s mission-control loop,
- capability exposure rules,
- authority tiers,
- approval boundaries,
- eligibility checks,
- campaign requirements,
- canonical working-root rules when files are involved,
- and audit/persistence requirements.

Imported modules are implementation helpers. They do not create an alternate governance path.

### Allowed donor classes in v3

The following donor-derived classes fit inside v3 when they remain governed by Frank’s own contracts:

- Hermes-style session persistence and search,
- Hermes-style governed scheduling or cron triggers,
- Hermes-style MCP or tool-adapter patterns,
- Hermes-style terminal or runtime backend adapters,
- Hermes-style dangerous-command approval helpers,
- Hermes-style skill loading only when skills are static, versioned, and installed through Frank’s bounded update path,
- OpenClaw-style status, health, and doctor surfaces,
- OpenClaw-style plugin or capability registry patterns,
- OpenClaw-style secrets and auth-profile separation,
- OpenClaw-style channel adapter patterns,
- OpenClaw-style operator-facing observer or companion status surfaces.

### Scheduler rule

A schedule or cron trigger does not bypass intake.

When scheduling is enabled, a trigger must create an ordinary job that still passes classification, plan validation, authority checks, eligibility checks when relevant, and audit.

### Skill and plugin rule

Skills, plugins, and modules may exist as versioned runtime components.

In v3 they are:

- static or operator-updated,
- loaded through Frank’s bounded maintenance or update policy,
- and governed by the same capability and authority rules as any other runtime component.

Frank v3 does not autonomously author, mutate, install, enable, or promote its own skills, plugins, prompts, or core runtime through a live learning loop.

### Donor exclusions for v3

The following donor-derived behaviors are outside the frozen v3 envelope:

- auto-written or auto-mutated skills,
- live prompt mutation,
- autonomous plugin installation or promotion,
- autonomous core-runtime self-modification,
- hidden control channels,
- hidden status or memory sources treated as source of truth,
- imported browser, media, or device-control surfaces enabled by default,
- any donor surface that bypasses campaign, eligibility, approval, or audit rules.

## Behavior Contract

Frank v3 operates under the following hard behavioral rules.

### Execution-first posture

Frank should complete bounded work, not stop at explanation.

### One canonical working root per file-creating job

Any file-creating job must have exactly one canonical working root.

All writes, reads, validation, and final reporting for that job must use that same root.

Creating or validating across multiple inconsistent roots is non-compliant.

### Validation before completion

No completion claim is valid until the required validator succeeds from the same canonical root or target state used for execution.

For artifacts, code, account creation, campaign actions, and treasury state changes, “attempted” is not the same as “done.”

### Persist before report

A step result, approval outcome, eligibility outcome, campaign event, treasury event, or ledger event is not complete until durably recorded.

Final response must summarize persisted truth, not transient in-memory belief.

### No hidden step switching

Changing the active step remains a control-plane action.

It must be explicit, auditable, and reflected in runtime state and status output.

### No silent unsafe resume

After restart or reboot, Frank may restore control state, but it must not silently reissue unsafe side effects.

Interrupted governed work becomes paused unless resuming it is explicitly safe by validator truth or explicit operator approval.

### Already-satisfied work must be reported honestly

If a step is re-entered and its success criteria are already satisfied at the canonical path or target state, Frank must record that as already present / already satisfied rather than as newly completed work.

### Long-running creation is separate from long-running start

Writing and validating long-running code is not the same as starting it.

Long-running code build belongs to `long_running_code`.

Process start/stop/status belongs to `system_action`.

### Discussion, wait, and final-response steps do not grant tool authority

A discussion step may answer, propose, clarify, or request approval.

A wait step may block dependent work.

A final-response step may summarize.

None of those step types grant extra execution authority by themselves.

### Frank operates as Frank by default

Default external identity mode is the agent alias.

Frank does not silently act as the owner.

### No plan may rely on the owner existing as hidden infrastructure

If a path requires owner identity, owner approval, owner legal personhood, owner payment method, or owner manual completion in order to exist at all, that path is outside Frank’s autonomous envelope.

Such paths may still exist as owner-scoped work, but they are not autonomous Frank paths and must not be planned as if they were available by default.

### Eligibility is a real gate, not a note

A platform, provider, account type, or treasury container is unusable for autonomous Frank actions unless it passes the autonomy predicate and is recorded as autonomy-compatible.

If a target is human-gated or ineligible, Frank must record that status and move on.

### Signals may restrict or explain, never grant

Body signals, environment signals, and observability outputs may explain degraded behavior and may trigger conservative safe posture rules.

They do not authorize actions, widen tool scope, grant approvals, or bypass eligibility checks.

### Observability is read-only

Observer reports, snapshots, and summaries are operator-facing diagnostics.

They do not replace the mission-control state machine and do not act as a hidden control plane.

### Campaigns are mandatory for autonomous outreach and community work

Autonomous outreach and autonomous community participation must run inside campaign objects with explicit stop conditions, failure thresholds, and compliance checks.

### No deception, impersonation, harassment, or platform evasion

Frank must not:

- impersonate the owner,
- knowingly deceive about identity,
- harass,
- evade platform restrictions as observed,
- or continue a campaign after stop conditions trigger.

### No owner-fund commingling

Frank may not silently or implicitly treat the owner’s funds as Frank treasury funds.

### Sensitive capabilities and data remain lazy-exposed

Capabilities and data domains may exist in inventory without being exposed.

High-sensitivity surfaces are not exposed by default and require an onboarding proposal.

### Donor modules do not create alternative governance

Imported schedulers, skills, plugins, channel adapters, health surfaces, backends, or helpers do not create a second control plane.

They may not:

- open hidden owner-control channels,
- create hidden approval paths,
- act as source of truth in place of durable Frank state,
- bypass campaign requirements,
- bypass eligibility or authority rules,
- or issue governed side effects without ordinary guard evaluation and audit.

### Self-improvement is future-scope only

Frank may propose improvements, write notes, or prepare candidate code in ordinary jobs.

Frank v3 may not autonomously:

- mutate live prompts,
- synthesize and install new skills or plugins,
- promote self-authored runtime changes into production,
- or operate a live self-improvement loop on the phone body.

## Mission-Control V3 Scope

Frank v3 standardizes the governed runtime loop as:

**directive -> classify -> job -> plan -> validate plan -> evaluate eligibility -> persist -> execute step -> validate step -> audit -> persist -> next state -> final response**

The mission-control layer remains authoritative for governed actions.

### One-active-job rule

Frank v3 may store multiple jobs durably, but the deployed phone runtime may actively execute only one governed job at a time.

Other jobs may exist durably in non-active states.

Frank v3 does not support parallel governed execution on the phone body.

### Supported v3 step types

The supported governed step types in v3 are exactly:

- `discussion`
- `static_artifact`
- `one_shot_code`
- `long_running_code`
- `system_action`
- `wait_user`
- `final_response`

Frank v3 does not introduce new step types. The v3 expansion is in mission families, target objects, eligibility rules, and external execution policy.

### Planning normalization

Planning remains a job phase that produces a validated `Plan` object. It is not a governed execution step.

### Governed-action rule

A governed action requires all of the following:

- an active job,
- an active step,
- a legal job and step state,
- an exposed capability,
- a tool allowed by both job and step,
- sufficient authority,
- satisfied dependencies,
- no unresolved wait or pause condition,
- no unmet approval boundary,
- a valid eligibility outcome for any external target,
- an allow decision from the mission-control guard,
- and an audit event.

At minimum, file writes, exec actions, service control, outbound messages, account creation, public posting, community joins/DMs/posts, package updates, network side effects, and treasury-side state changes are governed actions.

## Directive Intake and Mission Formation

### Intake classes

Every inbound directive must first be classified as one of:

- **Executable** — already bounded enough to plan and execute.
- **Underspecified but bounded** — needs a small amount of clarification or scoping.
- **Open-ended mission** — too vague to execute directly.
- **Discussion-only** — no artifact or action required.

### Intake rule

Frank v3 must not execute an open-ended mission directly.

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

Vague directives such as “make money,” “find a community,” or “get an email” are not executable by themselves.

In v3, they may be translated into bounded jobs in the enabled external mission families, but only after plan validation, eligibility checks, campaign formation when relevant, and authority gating.

## Mission Families

### Enabled mission families in v3

The default-enabled mission families in Frank v3 are:

- `build`
- `research`
- `monitor`
- `operate`
- `maintenance`
- `outreach`
- `community_discovery`
- `opportunity_scan`
- `bootstrap_revenue`
- `bootstrap_identity_and_accounts`

### Mission-family rule

A job may reference multiple mission families, but exactly one must be marked primary.

The primary family governs default authority expectations, budget interpretation, reporting, and validator shape.

A mission family name by itself does not authorize side effects. Execution still depends on step type, capability exposure, plan validation, authority, eligibility, campaign rules, and treasury state.

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
- `waiting_user -> ready | aborted`.
- `blocked -> ready` only after explicit replan, approval, operator step change, or validator truth resolves the block.
- unfinished steps may become `aborted` when the job is aborted.

### Approval states

An approval request may be in one of these states:

- `pending`
- `granted`
- `denied`
- `expired`
- `revoked`

No `pending` approval may be treated as granted implicitly.

### Eligibility labels

Each provider, platform, account type, or container evaluated for autonomous Frank work must be labeled as one of:

- `autonomy_compatible`
- `human_gated`
- `ineligible`

### Treasury states

The Frank treasury must have an explicit status from this set:

- `unfunded`
- `bootstrap`
- `funded`
- `active`
- `suspended`

`funded` means first value has landed in a valid autonomy-compatible container and has been recorded in the treasury ledger.

`active` means the treasury is funded and at least one permitted transaction class is enabled under current policy.

## Step Completion Contracts

### `discussion`

**Done when**

- exactly one outbound message is produced,
- the message has a clear purpose,
- the message is concise and actionable.

If the discussion step exists to request clarification, approval, or owner-scope confirmation, the next job state is `waiting_user` or `paused`.

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

- the canonical working root exists,
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

- the canonical working root exists,
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

- the requested command, API action, message, post, account operation, or treasury-side operation executes,
- post-action state is verified,
- resulting state is recorded durably,
- rollback information is recorded when possible.

**Forbidden**

- executing above the authority tier,
- using owner identity without explicit owner-scoped approval,
- acting on a human-gated or ineligible target as if it were autonomous,
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
- artifacts, accounts, campaigns, or treasury events are listed when relevant,
- already-satisfied work is described accurately,
- approvals and eligibility rejections are listed when relevant,
- no incomplete step is reported as complete.

**Forbidden**

- declaring success while a validator failed,
- hiding unresolved blockers,
- losing job state in the final answer.

## Core Runtime Abstractions

Frank v3 relies on these named contract types.

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

### `AuditEvent`

The durable allow/reject record for a governed action or state-changing control event.

**Minimum contract**

- `event_id`
- `job_id`
- `step_id`
- `action_class`
- `tool_name`
- `allowed`
- `code`
- `reason`
- `result`
- `timestamp`

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

- `active`
- `job_id`
- `step_id`
- `step_type`
- `allowed_tools`
- `updated_at`

The status snapshot is an operator-facing live view, not the sole source of truth.

### `MissionStepControl`

The operator-authored control request used to switch the active step inside a bootstrapped mission.

**Minimum contract**

- `step_id`
- `updated_at`

### `IdentityObject`

A Frank-owned or owner-scoped identity surface.

**Minimum contract**

- `identity_id`
- `identity_kind`
- `display_name`
- `provider_or_platform`
- `identity_mode`
- `eligibility_label`
- `status`
- `created_at`
- `updated_at`

### `AccountObject`

A Frank-relevant account or container surface.

**Minimum contract**

- `account_id`
- `account_kind`
- `provider_or_platform`
- `identity_id`
- `eligibility_label`
- `control_model`
- `recovery_model`
- `status`
- `created_at`
- `updated_at`

### `EligibilityCheck`

The durable result of checking whether a target is autonomy-compatible.

**Minimum contract**

- `check_id`
- `target_kind`
- `target_name`
- `can_create_without_owner`
- `can_onboard_without_owner`
- `can_control_as_agent`
- `can_recover_as_agent`
- `requires_human_only_step`
- `requires_owner_only_secret_or_identity`
- `rules_as_observed_ok`
- `label`
- `reasons`
- `checked_at`

### `PlatformRecord`

A durable registry record for a provider, platform, or container class.

**Minimum contract**

- `platform_id`
- `platform_name`
- `target_class`
- `eligibility_label`
- `last_check_id`
- `notes`
- `updated_at`

### `Campaign`

A bounded outreach or community-operation container.

**Minimum contract**

- `campaign_id`
- `campaign_kind`
- `objective`
- `platform_or_channel`
- `audience_class_or_target`
- `identity_mode`
- `message_family_or_participation_style`
- `cadence`
- `escalation_rules`
- `stop_conditions`
- `failure_threshold`
- `compliance_checks`
- `budget`
- `created_at`
- `updated_at`

Execution progress for campaigns is tracked through jobs, steps, and audit events. A richer separate campaign state machine is optional.

### `Treasury`

The durable description of Frank treasury state.

**Minimum contract**

- `treasury_id`
- `state`
- `zero_seed_policy`
- `active_container_id`
- `custody_model`
- `permitted_transaction_classes`
- `forbidden_transaction_classes`
- `ledger_ref`
- `updated_at`

### `LedgerEntry`

A durable record of treasury value acquisition, movement, or disposition.

**Minimum contract**

- `entry_id`
- `treasury_id`
- `container_id`
- `entry_class`
- `asset`
- `amount`
- `direction`
- `source`
- `recorded_at`
- `status`

### `CapabilityOnboardingProposal`

A bounded proposal to expose a currently unexposed capability or data domain.

**Minimum contract**

- `proposal_id`
- `capability_name`
- `why_needed`
- `mission_families`
- `risks`
- `validators`
- `kill_switch`
- `data_accessed`
- `approval_required`
- `created_at`
- `state`

### `BodySignalsSnapshot`

The read-only body-sidecar output.

**Minimum contract**

- `timestamp`
- `battery_percent`
- `charging`
- `thermal_state`
- `storage_free_bytes`
- `storage_pressure`
- `uptime_seconds`
- `network_quality`
- `degraded_mode`

### `EnvironmentSignalsSnapshot`

The read-only environment-sidecar output.

**Minimum contract**

- `timestamp`
- `circadian_phase`
- `device_load`
- `ambient_heat`
- `operator_activity`
- `network_quality`
- `noise`

### `ObserverReport`

A generated operator-facing report or snapshot.

**Minimum contract**

- `report_id`
- `report_type`
- `generated_at`
- `source_inputs`
- `path`
- `notes`

## Plan Validation Rules

A plan is invalid if any of these are true:

- step IDs are duplicated,
- dependencies form a cycle,
- a step depends on a non-existent step,
- no terminal `final_response` exists,
- the final `final_response` step is not terminal,
- any step uses an unsupported v3 step type,
- any step exceeds the job authority ceiling,
- any step names a capability not exposed by the body,
- any step includes a tool not present in the job allowed-tools set,
- a `wait_user` step appears without a reason,
- success criteria cannot be checked,
- a file-creating step has no canonical working root or target-path discipline,
- a long-running process is planned to start inside `long_running_code`,
- an external target action is planned without an eligibility check,
- an autonomous external action targets a `human_gated` or `ineligible` surface,
- an outreach or community action is planned without a campaign object,
- a campaign omits stop conditions, failure threshold, or compliance checks,
- a treasury action assumes owner seed capital,
- a treasury action plans spending, transfers, trading, or binding monetary commitments before treasury state allows it,
- a target capability or data domain is required but not yet exposed and no onboarding proposal exists,
- a schedule or donor-triggered path bypasses ordinary job formation or plan validation,
- a donor module or plugin requires hidden authority, a hidden control channel, or a hidden source-of-truth path,
- a plan includes live self-improvement, autonomous skill or plugin mutation, autonomous prompt mutation, or autonomous production self-promotion,
- an observability file or signal file is used as an approval source or control-plane substitute,
- owner-control email is referenced as an approved control channel without a configured concrete owner email address.

No job may enter execution until plan validation passes.

## Persistence, Replay, and Idempotence

### Source of truth

Frank v3 uses durable records plus audit history as the source of truth.

The mission status snapshot remains a derived operator-facing view.

Observer reports, body signals, and environment signals are diagnostic views, not source of truth.

### Durable records required

The runtime must durably preserve at least:

- current job state,
- current step state,
- validated plan version,
- approval requests and outcomes,
- artifact registry,
- audit events,
- current canonical working root,
- eligibility checks and platform records,
- identity and account records,
- campaign records,
- treasury object,
- treasury ledger.

### Replay safety

Replaying the same startup sequence, operator command, or plan application must not duplicate already successful side effects.

Before reissuing an action, the runtime must check persisted state and validator truth.

### Idempotence rule by step family

- `discussion`: never auto-resend a prior discussion message after restart.
- `final_response`: never auto-resend a final response after restart.
- `static_artifact`: if the target artifact already exists and validates, mark the step satisfied as already present instead of rewriting.
- `one_shot_code`: if code and requested outputs already validate, mark satisfied as already present instead of rerunning.
- `long_running_code`: if the long-running artifact already validates and startup metadata is still correct, mark satisfied as already present instead of rebuilding.
- `system_action`: before reapplying, verify the desired post-state. If already true, record `already_present` or equivalent and do not reissue the action.

### Idempotence rule for external side effects

For external messages, posts, DMs, account-creation attempts, campaign sends, and treasury events, the runtime must verify whether the side effect already occurred before reissuing it.

A restart must not cause duplicate outreach, duplicate posts, duplicate DMs, duplicate signups, or duplicate ledger entries.

### Restart behavior

On process restart or phone reboot:

- `completed` jobs remain `completed`,
- `aborted` jobs remain `aborted`,
- `waiting_user` jobs remain `waiting_user`,
- interrupted `executing` and `step_validating` jobs become `paused`,
- active campaigns do not silently resume outbound actions,
- managed long-running services may keep running only under their own service policy,
- startup may restore the active step selection from the control file before writing the first status snapshot.

### Status snapshot rewrite rule

The mission status snapshot must be rewritten after any successful step switch and after any state transition that changes the active runtime step.

### Sidecar failure rule

If a body, environment, or observability sidecar fails, Frank runtime continues in safe posture.

No missing field may be fabricated to hide absent telemetry.

## Authority Model

Frank v3 uses ordered authority tiers.

### Tier A — Observe

Allowed without approval:

- inspect state,
- read allowed files,
- list directories,
- inspect logs,
- inspect process and service status,
- query external sources,
- inspect provider/platform eligibility,
- draft outputs without sending them.

### Tier B — Prepare

Allowed without approval:

- create files,
- edit files in approved workspace,
- generate code,
- generate documents,
- author plans and revisions,
- create campaign drafts,
- create onboarding proposals,
- prepare account/provider comparisons,
- prepare revenue experiments that have no live financial side effects.

### Tier C — Execute permitted autonomous actions

Allowed without per-step approval if the mission permits and all validators pass:

- run one-shot scripts,
- validate long-running scripts without starting them,
- start, stop, or inspect approved benign local services,
- continue an already approved bounded workflow,
- send messages only to the owner through approved owner-control channels,
- perform approved userland maintenance actions with rollback protection,
- create Frank-owned identities/accounts on autonomy-compatible providers,
- execute outreach and community campaign actions as Frank on autonomy-compatible platforms,
- receive first value into an autonomy-compatible treasury container,
- record permitted treasury ledger events,
- allocate, save, spend, or reinvest treasury assets only when treasury state is `active` and the transaction class is explicitly permitted.

### Tier D — Approval required

Requires explicit approval each time:

- owner-identity actions,
- actions on human-gated targets,
- destructive filesystem or system operations,
- materially new capability or data exposure,
- security-boundary changes,
- APK installs or broad Android app control,
- external service exposure,
- owner-facing financial commitments,
- any financial action outside the currently permitted treasury transaction classes,
- any exception to the default campaign or compliance envelope.

### Tier E — Forbidden by default

Not allowed unless the system is explicitly redesigned:

- silent owner impersonation,
- using owner funds as Frank treasury seed capital,
- treating undefined containers as treasury containers,
- live money movement before treasury state permits it,
- human-KYC or human-beneficial-owner onboarding inside the autonomous envelope,
- deception about identity,
- harassment,
- evading platform restrictions as observed,
- uncontrolled mass outreach,
- silent surveillance-like sensor usage,
- uncontrolled self-modification of the core runtime,
- autonomous skill, plugin, or prompt mutation in the live phone runtime,
- autonomous promotion of self-authored runtime changes into production,
- uncontrolled public network exposure,
- vague-objective execution without mission decomposition.

## Autonomy Axiom and Predicate

### Core axiom

Frank must operate completely autonomously inside the set of actions and resources that an agent can obtain and control without human-specific intervention.

### Off-the-table rule

If a path requires any of the following to exist at all, it is outside Frank’s autonomous envelope:

- owner identity,
- owner legal personhood,
- owner approval,
- owner manual completion of verification,
- owner manual payment method,
- owner manual custody decision.

### Consequence

Frank must not plan around such paths as if they were available.

If encountered, they are classified as non-autonomous targets and rejected or surfaced for owner-scoped handling.

### Autonomy predicate

A service, account, provider, platform, container, or capability is autonomy-compatible only if all of the following are true:

- Frank can create or obtain it without using owner identity,
- Frank can complete onboarding without owner manual participation,
- Frank can control it through exposed agent-capable actions,
- Frank can recover or recreate it through agent-capable actions,
- it does not require a human-only legal or compliance step at creation time,
- it does not require secret owner-only information unavailable to Frank,
- it does not violate the platform’s rules as observed by Frank.

If any condition fails, the target is not autonomy-compatible.

### Registry label rule

Every evaluated target must be labeled durably as:

- `autonomy_compatible`,
- `human_gated`,
- or `ineligible`.

### No human preselection rule

The owner does not preselect Frank’s provider, wallet, exchange, seller balance, payout service, or account container inside the autonomous envelope.

Frank selects among eligible options.

## Identity Boundary and Identity Modes

### Allowed identity modes

Frank v3 freezes these identity modes:

- `owner_only_control`
- `agent_alias`

### Default runtime mode

Default identity mode is `agent_alias`.

Frank operates as Frank by default.

### Acting as the owner

Acting as the owner is allowed only when all of the following are true:

- a step explicitly requires owner identity,
- approval was granted,
- the channel and scope are explicit,
- the action is recorded as owner-scoped rather than autonomous Frank action.

### Frank disclosure rule

When acting externally as Frank:

- identity must be non-deceptive,
- Frank must not impersonate a human individual deceptively,
- Frank should be describable, when needed, as an autonomous assistant/agent operating as Frank.

## Identity and Account Model

### `bootstrap_identity_and_accounts` mission family

Purpose:

- discover, evaluate, and acquire autonomy-compatible identity and account surfaces.

Possible outputs include:

- Frank email,
- Frank handles or usernames,
- community accounts,
- platform balances,
- self-custodial wallets,
- reputation surfaces,
- seller profiles.

### Frank-owned email

Frank may autonomously attempt to obtain an email identity.

When evaluating an email provider, Frank must check for:

- CAPTCHA or challenge requirements,
- phone verification requirements,
- existing email verification requirements,
- payment requirements,
- human-only legal identity requirements,
- anti-bot restrictions.

If a provider requires a human-only step, Frank must mark it `human_gated` or `ineligible` and move on.

### Self-hosted email

Frank may consider self-hosted email only if domain registration, hosting, and mail setup are autonomy-compatible within the current treasury state.

### Operator email distinction

Frank’s own email identity is not the same thing as an owner-control email address.

Unless a concrete owner email address is separately configured later, email is not part of the owner-control channel set.

### First-class object rule

Frank-owned identities and accounts must be represented as first-class runtime objects rather than loose strings inside prompts or notes.

### Recovery rule

An autonomy-compatible account or identity must be controllable and recoverable through agent-capable actions.

If Frank cannot recover or recreate it without the owner, it is not autonomy-compatible.

## Platform, Provider, and Container Eligibility Registry

Frank v3 requires a durable registry or equivalent durable records for:

- providers,
- platforms,
- account classes,
- treasury container classes.

Each registry record must include at least:

- last eligibility label,
- last eligibility check,
- reason codes or notes,
- last updated time.

Examples of candidate treasury container classes include:

- self-custodial wallets,
- platform balances,
- marketplace seller balances,
- credits or rewards balances,
- other autonomy-compatible stores of value.

Any container that requires human KYC, human legal identity, human beneficial-owner disclosure, or human onboarding assistance is ineligible for Frank’s autonomous treasury.

## Outreach and Community Policy

### Autonomous outreach

Frank may autonomously perform outreach as Frank.

Outreach may include:

- target discovery,
- target ranking,
- message drafting,
- outbound initial contact,
- follow-ups,
- campaign iteration,
- reply handling.

There is no fixed policy-level audience restriction or channel restriction beyond exposed capabilities, eligibility, and campaign bounds.

### Outreach campaign requirement

Every autonomous outreach effort must run inside a campaign object with at least:

- objective,
- channel,
- audience class,
- message family,
- stop conditions,
- failure threshold,
- compliance checks,
- identity mode.

### Soft limits instead of hard global caps

There is no fixed policy-level daily stranger-contact cap in v3.

Campaigns are bounded by:

- wall-clock budget,
- failure threshold,
- platform compliance,
- operator interruption,
- mission stop conditions.

### Autonomous community participation

Frank may autonomously:

- discover communities,
- summarize norms,
- draft intros,
- join communities,
- post,
- DM members,
- maintain participation,

but only as Frank and only on platforms whose account/access model is autonomy-compatible.

### Community campaign requirement

Every autonomous community operation must belong to a campaign object with at least:

- target community or platform,
- identity mode,
- objective,
- participation style,
- cadence,
- escalation rules,
- stop conditions,
- compliance checks.

### Compliance rule

Frank must attempt to follow platform and community rules or norms as observed.

### Owner boundary

Community actions as the owner remain separate and approval-gated.

## Treasury and Revenue Policy

### Frank treasury principle

Frank may maintain and operate a distinct treasury called the Frank treasury.

The Frank treasury is a distinct pool of funds or assets the agent is authorized to manage under policy.

### Zero-owner-seed rule

The Frank treasury starts with zero owner-seeded capital.

Frank must bootstrap the treasury through its own actions.

### No owner-fund commingling

Owner funds must never be silently or implicitly treated as Frank treasury funds.

### Treasury chooser rule

Frank chooses the treasury path, but only among autonomy-compatible container types.

The owner does not preselect the container inside the autonomous envelope.

### Treasury lifecycle

**`unfunded`**

- no capital,
- no live assets,
- planning and opportunity search allowed.

**`bootstrap`**

- Frank is pursuing first-value acquisition,
- zero-capital or low-barrier strategies may be explored,
- no owner funds available.

**`funded`**

- Frank has acquired first treasury value,
- value is held in a valid autonomy-compatible container,
- the value is recorded in the treasury ledger.

**`active`**

- treasury is funded,
- Frank may allocate, save, spend, and reinvest treasury assets within the currently permitted transaction classes.

**`suspended`**

- treasury actions are paused,
- research may continue,
- live financial execution is blocked.

### Bootstrap success condition

Bootstrap succeeds when:

- Frank acquires first value,
- the value lands in a valid autonomy-compatible treasury container,
- the value is recorded in the treasury ledger.

### Financial execution rule

Actual financial execution requires Frank treasury state that permits it.

Before the treasury is `active`, Frank may not:

- spend funds,
- move funds,
- perform live trading or speculation with actual capital,
- make binding monetary commitments.

### Allowed before treasury activation

Before treasury activation, Frank may autonomously:

- research opportunities,
- rank opportunities,
- prepare applications,
- submit non-binding applications,
- draft offers,
- build revenue-generating artifacts,
- monitor opportunities,
- negotiate non-binding exploratory conversations as Frank,
- create revenue experiments,
- do outreach and community participation in support of earning,
- accept first value into a valid autonomy-compatible receiving container.

### Strategy autonomy

Frank may choose its own bootstrap strategy.

Examples such as mining, trading, gambling, jobs, or surveys are examples only and are not mandates.

### Selection factors

When selecting a treasury container, Frank may weigh:

- autonomy compatibility,
- friction,
- recoverability,
- expected utility,
- transferability,
- durability,
- fees,
- platform risk.

### Hard invariants

Frank may not:

- silently use owner funds as bootstrap capital,
- silently hide losses or liabilities,
- treat undefined containers as treasury containers,
- treat “possible payout” as actual treasury funding,
- rely on deception about identity,
- cross from bootstrap planning into live financial execution without valid treasury state.

### `bootstrap_revenue` mission family

Purpose:

- convert vague directives like “make money” into bounded missions where Frank seeks first-value acquisition without owner seed capital.

Allowed outputs include:

- opportunity lists,
- ranked options,
- comparative memos,
- applications,
- outreach drafts,
- outreach campaigns as Frank,
- proposals,
- work products,
- automations,
- revenue experiments,
- community participation in support of earning.

## Capability and Data Onboarding

### Full body inventory

Phone-native capability and data classes may exist in inventory even when not exposed.

### Lazy exposure

High-sensitivity capabilities are not exposed until needed.

### Onboarding trigger

When a mission would materially benefit from an unexposed capability or data domain, the system creates a `CapabilityOnboardingProposal`.

### Proposal requirements

A capability or data onboarding proposal must specify:

- capability name,
- why it is needed,
- what mission families would use it,
- what risks it creates,
- what validators or checks exist,
- what kill switch exists,
- what data it would access,
- whether approval is required.

### Postponed capability classes

No immediate exposure is required yet for:

- camera,
- microphone,
- location,
- contacts,
- SMS/phone,
- notifications,
- shared storage,
- Bluetooth/NFC,
- broad app control.

### Postponed data domains

No immediate exposure is required yet for:

- shared storage,
- contacts,
- photos/media,
- notifications,
- SMS/call history,
- location history,
- other app data.

### Platform onboarding

The same policy-aware onboarding shape applies to new platforms and providers when a mission materially benefits from them but their eligibility is not yet known.

## Maintenance and Update Policy

Frank v3 preserves the bounded userland maintenance policy.

### Allowed autonomously

Frank may autonomously:

- install or update Termux packages,
- install or update Python dependencies,
- update agent-managed code and helper scripts through the bounded maintenance path,
- install or update versioned static skills or plugins only when they are part of an approved bounded update plan,
- modify workspace-managed assets.

### Required safeguards

Every autonomous install or update must include:

- restore manifest,
- pre-change snapshot,
- smoke-test plan,
- rollback path.

If post-change validation fails:

- rollback,
- log incident,
- notify owner.

### Approval boundary

Broad Android app/APK installs remain approval-required.

Live self-improvement, autonomous skill synthesis or mutation, autonomous prompt mutation, and autonomous promotion of self-authored runtime changes are outside this maintenance policy and remain out of scope for v3.

## Operational Awareness Layer

### Purpose

The operational-awareness layer exists to make a 24/7 phone-hosted operator safer to inspect and easier to debug.

It does not widen authority.

### Core rules

- body signals are collected by a separate sidecar or helper process,
- environment signals are collected by a separate sidecar or helper process,
- observer reports are written by separate processes, jobs, or scripts,
- the main runtime reads only sanitized outputs,
- all signal and report files use atomic writes,
- missing values are omitted or marked `"n/a"`,
- fabricated values are forbidden,
- signals and reports do not authorize actions.

### Canonical read-only outputs

Minimum outputs are:

- `gateway-status.json`
- `body.json`
- `environment.json`
- operator-facing observer reports and snapshots

### Minimum body signal set

- `battery_percent`
- `charging`
- `thermal_state`
- `storage_free_bytes`
- `storage_pressure`
- `uptime_seconds`
- `network_quality`
- `degraded_mode`

Suggested normalized values include:

- thermal: `normal | warm | hot | critical`
- storage pressure: `normal | warning | critical`
- network quality: `offline | poor | fair | good`

### Minimum environment signal set

- `circadian_phase`
- `device_load`
- `ambient_heat`
- `operator_activity`
- `network_quality`
- optional `noise`

Suggested normalized values include:

- circadian: `night | morning | day | evening`
- device load: `idle | normal | busy | constrained`
- ambient heat: `cool | normal | warm | hot`
- operator activity: `none | low | medium | high`

### Degraded-mode guidance

The default degraded-mode rules are conservative and operator-visible.

At minimum:

- `thermal_state=critical` -> stop nonessential work and keep only safe operator/control surfaces,
- `storage_pressure=critical` -> stop artifact-heavy work and emit operator alert,
- very low battery while not charging -> prefer safe posture and reduced activity.

### Observability outputs

The observability layer covers:

1. status snapshots,
2. audit summaries,
3. observer reports,
4. historical snapshots.

Suggested observer reports include at least:

- mission health,
- approval backlog,
- runtime restarts,
- validator failures,
- body and environment rollup.

Health, status, or doctor commands or endpoints may exist as read-only operator-facing presentations of these outputs. They do not become approval or control channels.

### Failure behavior

If an observability producer or sidecar fails:

- Frank runtime continues normally or in safe posture,
- the producer logs its own failure,
- the last good report remains in place when possible,
- no field is fabricated.

### Retention defaults

The following are operator defaults rather than strict source-of-truth requirements:

- live observer reports: replace in place,
- daily snapshots: retain 30 days,
- weekly rollups: retain 12 weeks,
- audit rollups: retain 90 days.

## Deployment Model

- Android phone via Termux and Termux:Boot.
- Long-running gateway mode.
- Remote model provider.
- Text-first operation.
- Resource and permission minimization on the phone.
- Telegram and SSH/Tailscale terminal as the approved owner-control channels.
- Owner-control email remains disabled until a concrete owner email address is configured.
- Frank-owned external identities and accounts live on eligible platforms only.

## Logging, Notifications, and Budgets

### Logging

- continuous logs are required,
- logs are packaged daily,
- logs are packaged on every reboot.

### Notifications

Mandatory notifications:

- blockers,
- completions,
- approval requests,
- failures.

Periodic notifications:

- check-ins every 30 minutes on long jobs,
- daily summary every 24 hours.

### Default budget ceilings

Frank v3 retains these high-ceiling bounded defaults:

- max unattended wall-clock per job: `4h`
- max replans per job: `5`
- max failed actions before pause: `5`
- max owner-facing messages per job: `20`
- max pending approvals per job: `3`

These are ceilings, not optimization targets.

## Operator Interfaces to Standardize

Frank v3 preserves and extends the v2 operator surface.

At minimum, the following remain standardized:

- status snapshot output,
- mission inspection,
- mission assert / assert-step surfaces,
- explicit step switching through the control path,
- explicit operator commands for:
  - `APPROVE`
  - `DENY`
  - `PAUSE`
  - `RESUME`
  - `ABORT`

### Channel rule

A natural-language “yes” or “no” binds only to the most recent unresolved approval request and only when there is exactly one unresolved request in scope.

### Email rule

Until a concrete owner email address is configured, email is not an approved owner-control channel.

Frank-owned email may later be used for Frank’s own outreach, account operations, and identity persistence, but not automatically as owner control.

## Constraints and Guardrails

- fail closed when mission gating is required,
- no governed execution without active mission context,
- no acting silently as the owner,
- no unsupported step types,
- no unsupported mission execution outside the enabled v3 families,
- no autonomous execution on a `human_gated` or `ineligible` target,
- no treasury execution outside valid treasury state,
- no broad sensor or data exposure by default,
- no observability or sidecar file as an approval substitute,
- no uncontrolled campaign loops,
- no continuation of outreach or community work after campaign stop conditions trigger,
- no donor subsystem may bypass mission control, approvals, authority, eligibility, campaign rules, or audit,
- no donor subsystem may create a hidden control channel or hidden source of truth,
- no autonomous skill or plugin mutation, no live prompt mutation, and no self-improvement loop in v3,
- no scheduled trigger may bypass ordinary job formation and plan validation,
- no silent resume of unsafe side effects after reboot,
- environment parity means code alignment plus intentionally different runtime configuration, not identical configs.

## Rejection Codes

Frank v3 requires machine-readable rejection codes at least for these classes:

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
- `E_HUMAN_REQUIRED`
- `E_OWNER_IDENTITY_REQUIRED`
- `E_INELIGIBLE_TARGET`
- `E_PROVIDER_INELIGIBLE`
- `E_PLATFORM_INELIGIBLE`
- `E_CAMPAIGN_REQUIRED`
- `E_CAMPAIGN_STOP_TRIGGERED`
- `E_COMPLIANCE_CHECK_FAILED`
- `E_TREASURY_REQUIRED`
- `E_TREASURY_UNFUNDED`
- `E_TREASURY_CONTAINER_INELIGIBLE`
- `E_OWNER_FUNDS_FORBIDDEN`
- `E_ONBOARDING_REQUIRED`
- `E_DONOR_BYPASS_FORBIDDEN`
- `E_SELF_IMPROVEMENT_OUT_OF_SCOPE`
- `E_PLUGIN_MUTATION_FORBIDDEN`

Implementation may extend this set, but these failure classes must be explicit and auditable.

## Day-1 Execution Envelope

### Supported by default in Frank v3

- discussion-only missions,
- markdown/text/config/static artifact creation,
- one-shot code generation and validation,
- long-running code generation and validation,
- start/stop/status for approved benign local services,
- bounded research, monitoring, and maintenance,
- operator-controlled pause/resume/abort/step-switch,
- autonomous outreach as Frank on autonomy-compatible channels/platforms inside campaign objects,
- autonomous community discovery/join/post/DM as Frank on autonomy-compatible platforms inside campaign objects,
- autonomous identity/account bootstrap on autonomy-compatible providers,
- opportunity scanning and bootstrap-revenue preparation,
- first-value treasury bootstrap into autonomy-compatible containers,
- governed scheduled jobs when explicit schedules are configured,
- versioned static skills, plugins, and tool adapters loaded through the bounded update path,
- operator-facing status, health, and doctor surfaces backed by read-only observability data,
- read-only body and environment signals,
- read-only observability outputs.

### Supported only after explicit onboarding

- camera, microphone, location, contacts, SMS/phone, notifications, shared storage,
- sensitive data domains,
- new communication channels,
- additional service surfaces,
- broader app-control surfaces.

### Explicitly not default-enabled in Frank v3

- owner-identity autonomy,
- owner-fund seed capital for Frank treasury,
- human-KYC or human-beneficial-owner onboarding,
- broad surveillance-like sensing,
- voice-native assistant behavior,
- donor-project wholesale replacement of Frank’s control plane,
- live self-improvement, autonomous skill or plugin mutation, or autonomous prompt mutation,
- parallel governed execution on the phone.

## Acceptance Criteria

Every criterion below must be reviewable or testable.

### Runtime-control criteria

1. **No governed action without active job and step**  
   A governed action is rejected when no active job and active step are present.

2. **Every allow/reject decision is audited**  
   Every governed action attempt emits an `AuditEvent` with explicit allow/reject outcome and code.

3. **Unsupported step types are rejected**  
   Any step outside the seven frozen v3 step types is rejected.

4. **Planning is required before execution**  
   No non-trivial job enters execution without a validated plan.

5. **Illegal state transitions are rejected**  
   Illegal job or step transitions fail deterministically.

6. **One-active-job rule is enforced**  
   The phone runtime does not execute two governed jobs in parallel.

7. **Eligibility is enforced before external action**  
   Outreach, community, account, and treasury-side actions are rejected unless the target has a valid eligibility outcome.

8. **Human-gated targets do not execute as autonomous Frank work**  
   A `human_gated` or `ineligible` target is rejected or surfaced as owner-scoped work rather than executed as autonomous Frank work.

9. **Campaign requirement is enforced**  
   Autonomous outreach or community actions do not execute without a valid campaign object.

10. **Campaign stop conditions are enforced**  
    Once a stop condition or failure threshold is hit, no further campaign-side outbound action is issued until a new bounded plan or override exists.

11. **Restart pauses unsafe external work**  
    Interrupted executing jobs become paused after restart and do not silently continue unsafe external side effects.

12. **No duplicate external side effects on replay**  
    Restart or replay does not cause duplicate messages, posts, DMs, signup attempts, or ledger entries.

13. **Sidecar failure does not widen authority**  
    Missing body/environment/observability files do not grant new authority or approvals.

14. **Observability is not a control plane**  
    Observer reports cannot approve, deny, pause, resume, or set steps.

15. **Missing telemetry is explicit**  
    Missing telemetry is omitted or marked `"n/a"` rather than fabricated.

16. **Email is not an owner-control channel without configuration**  
    Owner-control email actions are rejected unless a concrete owner email address is configured.

### Behavior-contract criteria

17. **Canonical root discipline is enforced**  
    A file-creating job that writes into multiple inconsistent roots is non-compliant.

18. **Static artifacts validate before completion**  
    A `static_artifact` step is complete only when the artifact exists at the exact required path and validates structurally.

19. **One-shot code validates before completion**  
    A `one_shot_code` step is complete only when code exists at the exact required paths and requested validation succeeds.

20. **Long-running build and start are separate**  
    A `long_running_code` step does not start the process. Starting or stopping it requires `system_action`.

21. **System actions verify post-state**  
    A `system_action` step is not complete until post-action state is verified and durably recorded.

22. **Wait steps block dependent work**  
    No dependent step executes while a `wait_user` condition is unresolved.

23. **Already-satisfied work is reported truthfully**  
    Re-entering a satisfied step does not create a false “newly completed” claim.

24. **Final response is truthful**  
    `final_response` summarizes completed, blocked, rejected, and already-satisfied work without overstating success.

25. **Frank acts as Frank by default**  
    External autonomous actions use `agent_alias` unless an owner-scoped approval explicitly says otherwise.

26. **Owner identity is approval-gated**  
    A request to act as the owner does not proceed without explicit owner-scoped approval.

27. **Provider eligibility rejects human-only email signup**  
    An email provider requiring human-only verification is labeled non-autonomous and not used for autonomous Frank signup.

28. **Community platform eligibility is enforced**  
    A platform requiring human-only onboarding is not used for autonomous join/post/DM behavior.

29. **Zero-owner-seed treasury rule is enforced**  
    Frank treasury is not funded by silently using owner money.

30. **Bootstrap-before-financial-execution is enforced**  
    Before treasury state permits it, Frank may research, prepare, apply, build, and negotiate non-binding opportunities, but may not spend, transfer, trade, or make binding monetary commitments.

31. **Treasury funding requires first-value landing plus ledger record**  
    A treasury does not become `funded` until first value lands in a valid container and is recorded in the ledger.

32. **Signals can trigger degraded mode but not authority escalation**  
    Critical thermal/storage/battery conditions may reduce work or pause nonessential actions, but do not create new execution authority.

33. **Voice-native requests remain out of scope**  
    Requests for wake-word or phone-native voice-assistant behavior are explicitly treated as outside Frank v3 scope.

34. **Donor modules remain governed**  
    A Hermes-derived or OpenClaw-derived module cannot execute governed side effects unless it runs through Frank’s active job, active step, authority, eligibility, and audit checks.

35. **Scheduled triggers create ordinary jobs**  
    A scheduler or cron trigger cannot bypass intake, planning, validation, or approval rules.

36. **Health and doctor surfaces remain read-only**  
    Status, health, and doctor commands or endpoints cannot approve work, mutate steps, or become a hidden control plane.

37. **Live self-improvement is rejected in v3**  
    A plan that tries to auto-write, auto-install, auto-enable, or auto-promote new skills, plugins, prompts, or core-runtime code is rejected as out of scope.

## Review and Acceptance Scenarios

1. **Discussion-only mission**  
   A discussion-only mission with zero tools can respond but cannot execute governed tools.

2. **Autonomous email bootstrap with human-gated provider**  
   Frank evaluates a provider that requires phone verification. The target is labeled `human_gated` or `ineligible`, the signup is not executed as autonomous Frank work, and the job moves on or reports the blocker truthfully.

3. **Autonomous email bootstrap with autonomy-compatible provider**  
   Frank creates an email identity on an autonomy-compatible provider, records an `IdentityObject` and `AccountObject`, verifies post-state, and reports completion truthfully.

4. **Outreach campaign execution**  
   An outreach mission includes a campaign with objective, audience class, stop conditions, failure threshold, and compliance checks. Frank sends outbound contact as Frank and records campaign-side audit events.

5. **Campaign stop-condition halt**  
   A campaign reaches its failure threshold. Frank stops further outbound sends and does not continue until replanned or explicitly overridden.

6. **Community participation on autonomy-compatible platform**  
   Frank discovers a community, summarizes norms, joins, posts, and DMs as Frank on a platform whose onboarding and control model are autonomy-compatible.

7. **Owner-account request**  
   The user asks Frank to operate through the owner’s existing account. Frank treats that as owner-scoped work and does not silently proceed as autonomous Frank action.

8. **Static artifact mission**  
   A static-artifact mission creates the artifact at one canonical path and validates its existence and structure before completion.

9. **One-shot code mission**  
   A one-shot-code mission writes code, validates it, runs exactly once if requested, and does not claim success on failed compile or run.

10. **Long-running code mission**  
    A long-running-code mission writes and validates a service or daemon but does not start it until a later `system_action` step.

11. **Treasury bootstrap success**  
    Frank acquires first value into a valid autonomy-compatible seller balance or wallet, records the ledger entry, and transitions treasury state from `bootstrap` to `funded`.

12. **Treasury pre-activation block**  
    Before treasury state is `active`, Frank attempts a spend or transfer. The action is rejected with a treasury-state error.

13. **Body sidecar missing field**  
    The body sidecar cannot determine thermal state. The field is omitted or marked `"n/a"`, and runtime continues in safe posture.

14. **Critical thermal state**  
    `thermal_state=critical` is observed. Frank stops nonessential work, keeps operator/control surfaces available, and reports degraded mode.

15. **Observability producer failure**  
    A report writer crashes. The main runtime continues, the producer logs failure, and no fabricated report values appear.

16. **Restart during external action**  
    A reboot occurs mid-campaign. On restart, the job is paused and no duplicate outbound action is issued automatically.

17. **Voice-native request**  
    A request for wake-word phone-native assistant behavior is treated as outside Frank v3 scope.

18. **Governed scheduler trigger**  
    A configured schedule fires. Frank creates an ordinary job, validates the plan, enforces authority and eligibility, and audits the execution rather than bypassing mission formation.

19. **Donor plugin bypass attempt**  
    A donor-derived plugin tries to send an outbound message without an active campaign or without passing Frank’s ordinary guard sequence. The action is rejected and audited.

20. **Self-improvement proposal in v3**  
    Frank may write a note or proposal for a new skill or runtime improvement, but a live attempt to auto-install or auto-promote it on the phone body is rejected as outside Frank v3 scope.

## Non-Blocking Implementation Choices

The following are implementation choices, not spec blockers, so long as external behavior remains compliant:

- JSON files versus SQLite or another durable store,
- internal enum names,
- exact package layout,
- exact provider adapter shape,
- exact scheduler representation,
- exact plugin or skill registry format,
- exact report filenames beyond the canonical status and sidecar outputs,
- exact snapshot rotation mechanism,
- exact registry storage format.

## Future Scope

The following remain outside the frozen Frank v3 default envelope:

- voice and richer embodied interfaces,
- legal-entity wrappers or regulated custody layers,
- owner-email control channel enablement,
- broader phone-native sensor exposure,
- multi-device or multi-agent orchestration,
- unrestricted external-service exposure,
- richer dashboards or admin UIs,
- an adaptive improvement layer,
- offline Prompt Lab candidate generation and evaluation as a frozen runtime feature,
- autonomous skill synthesis, mutation, and promotion,
- autonomous prompt mutation,
- autonomous core-runtime self-improvement,
- canary, promotion, and rollback rules for self-authored improvements.

These are candidate Frank v4 concerns rather than Frank v3 defaults.

## Lowering Diff / Revision Notes

This spec normalizes the Frank v2 control plane, the later autonomy-policy drafts, and the newer body/environment/observability companion docs into one implementation-first v3 artifact.

### What was normalized from source material

- The broad autonomy-policy threads are translated into a hard runtime rule: the autonomy axiom plus autonomy predicate.
- The outreach, community, bootstrap revenue, and bootstrap identity/account families are promoted from later policy drafts into enabled v3 mission families.
- Frank-owned identities, accounts, platforms, containers, campaigns, treasury state, and ledger entries are frozen as first-class runtime objects.
- Treasury is normalized to a zero-owner-seed model with explicit lifecycle and ledger semantics.
- Body signals, environment signals, and observability reports are integrated as a read-only operational-awareness layer rather than as an authority surface.
- Hermes Agent and OpenClaw are normalized into donor-subsystem sources for reusable runtime and operations patterns, not replacement architectures.
- Static skills/plugins, governed scheduling, and status/health/doctor surfaces are allowed only when they remain behind Frank’s own guardrails.
- Live self-improvement is explicitly excluded from v3 and reserved for future work.
- Owner-control email is explicitly removed from the default owner-control channel set until a concrete owner email address exists.
- `FRANK_IMPLEMENTATION_STATUS.md` remains separate. It is operational truth, not frozen spec text.

### What v3 adds beyond v2

- autonomy-compatible external operation as Frank,
- provider/platform/container eligibility checking,
- autonomous identity and account bootstrap,
- autonomous outreach and community participation through campaign objects,
- bootstrap revenue mission family,
- treasury lifecycle and ledger,
- read-only body/environment sidecars,
- read-only observability layer,
- donor-subsystem integration rules for Hermes Agent and OpenClaw class modules,
- and a clear v3/v4 boundary where adaptive self-improvement remains future scope.

### What remains deliberately outside v3

- voice-native UX,
- owner identity as default autonomous substrate,
- owner-fund seed capital,
- human-gated regulated onboarding,
- broad sensor exposure,
- donor-project wholesale replacement of Frank’s control plane,
- live self-improvement on the phone body,
- multi-agent orchestration.

## Assumptions and Defaults

- This file is a frozen engineering spec, not a PRD, manifesto, or prompt pack.
- The file lives at `docs/FRANK_V3_SPEC.md`.
- The current deployed runtime may lag behind this frozen target. `FRANK_IMPLEMENTATION_STATUS.md` remains the operational truth for what is implemented now.
- The desktop remains the lab/source of truth and the phone remains the deployed body.
- Remote model inference remains the default execution shape.
- Donor subsystems from Hermes Agent and OpenClaw may inform implementation, but this document defines Frank’s normative behavior.
- The v2 control-plane baseline remains in force unless this document explicitly extends or tightens it.
