# Frank V4 Frozen Spec — Phone-Resident Autonomous Hot-Update Runtime

**Status:** Revised V4 product target, replacing the earlier desktop-only adaptive-lab framing  
**Date:** 2026-04-19  
**Source Basis:** Original `FRANK_V4_SPEC.md`; Frank V4 handover; Frank V3 runtime/control-plane baseline as described by the handover; Pi agent / `pi-mono` architecture as donor inspiration for hot-reloadable extensions, skills, prompt templates, packages, sessions, and local agent customization.

---

## Revision Summary

This revision changes the V4 target from:

```text
desktop-only adaptive improvement lab + phone deployed body
```

to:

```text
phone-resident autonomous Frank runtime with an isolated on-phone improvement workspace and transactional hot-update gate
```

The desktop is no longer a permanent architectural requirement. It may remain a development host, build host, test host, or emergency recovery host, but the intended finished V4 deployment is phone-only.

The safety boundary is no longer a physical desktop/phone boundary. The safety boundary is now:

```text
active live runtime plane != isolated improvement workspace != transactional hot-update gate
```

Frank V4 may improve itself on the phone and may hot-update reloadable runtime surfaces, Pi-agent style, when those updates pass deterministic gates and remain inside the allowed hot-update surface.

Frank V4 still must not silently mutate authority, treasury, approval, campaign, or mission-control policy surfaces. Those are not reloadable pack content by default.

---

## Problem Statement

Frank v1 froze the minimum deterministic execution surface. Frank v2 froze the control plane. Frank v3 froze bounded external operation: identity/account bootstrap, outreach/community work, treasury bootstrap, capability onboarding, and read-only operational awareness.

The remaining gap is not merely controlled self-improvement in a desktop lab. The remaining gap is **phone-resident autonomous self-improvement**.

Frank needs to become a 24/7/365 phone-hosted personal operator that can:

- keep running without a desktop dependency,
- maintain durable mission state,
- pursue standing directives,
- improve its own prompt packs, skill packs, routing metadata, runtime extensions, and local procedures,
- apply safe updates without a full manual redeploy,
- roll back quickly when an update regresses,
- and preserve the V3 operator contract while becoming more adaptive.

Pi agent demonstrates the useful shape: an agent can be made powerful by making prompts, skills, extensions, tools, and packages reloadable while the user keeps working. Frank should adopt that hot-update posture, but with Frank-specific durability, replay safety, authority gates, rollback, and mission-control accountability.

The point of V4 is therefore not:

```text
Frank rewrites arbitrary live files whenever it wants.
```

The point of V4 is:

```text
Frank can continuously improve and hot-update allowed runtime surfaces from inside a phone-resident improvement workspace through a transactional promotion/reload/rollback mechanism.
```

---

## Definition of Frank V4

Frank V4 is a text-first, phone-hosted, mission-persistent, autonomy-compatible personal operator with:

- an always-on live runtime plane,
- an isolated improvement workspace that may run on the same phone,
- a Pi-like reloadable pack surface,
- transactional hot-update semantics,
- append-only improvement and hot-update records,
- and deterministic rollback to last-known-good packs.

Frank means the deployed operator identity and agent alias, not merely the repo, prompt bundle, skill pack, or phone process.

Frank V4 retains the V3 mission-control substrate and external-operation envelope. It adds three governed layers:

1. **Live runtime plane** — executes governed live jobs using the current active runtime pack.
2. **Improvement workspace** — creates and evaluates candidate packs, skills, prompts, routing changes, and reloadable runtime extensions.
3. **Hot-update gate** — atomically stages, checks, applies, commits, or rolls back reloadable updates.

Frank V4 therefore has four distinct capabilities:

- Frank may act externally as **Frank** inside the V3 autonomy-compatible envelope.
- Frank may operate 24/7/365 from durable standing directives.
- Frank may improve selected parts of its own runtime surface inside the improvement workspace.
- Frank may hot-update allowed runtime surfaces without a manual full redeploy when hot-update policy permits it.

Frank V4 is still not a voice-native assistant by default, still not a broad sensor platform by default, and still not an uncontrolled self-editing control plane.

---

## Relationship to Frank V3

Frank V4 supersedes Frank V3.

Frank V4 carries forward the Frank V3 control plane, behavior contract, autonomy predicate, identity boundary, campaign rules, treasury rules, capability-onboarding model, and read-only operational-awareness model unless this document tightens or extends them.

The following V3 invariants remain in force:

- execution-first posture,
- one canonical working root for file-creating work,
- validation before completion,
- persist before report,
- explicit step switching,
- no silent unsafe resume,
- idempotent replay behavior,
- no silent acting as the owner,
- one active governed live job at a time,
- frozen V3 step taxonomy,
- no autonomous execution on `human_gated` or `ineligible` targets,
- no owner-fund commingling,
- signals may restrict or explain, never grant,
- observability is read-only,
- Telegram owner-control onboarding remains the only accepted provider-onboarding lane unless explicitly reopened.

Frank V4 adds:

- phone-resident improvement workspace,
- runtime-pack versioning and hot-update promotion,
- Pi-like reloadable prompts, skills, manifests, extensions, and local packages,
- immutable eval-suite contracts per improvement attempt,
- train / holdout / canary separation for candidate evaluation,
- promotion, hot-update, and rollback records,
- new improvement and hot-update mission families,
- mutable-target rules for prompt packs, skills, routing metadata, and reloadable extension packs,
- explicit rejection of uncontrolled active-pack mutation,
- explicit rejection of evaluator mutation during a run,
- append-only improvement and hot-update ledgers,
- standing directives and autonomy budgets for continuous operation.

If a Frank V3 statement conflicts with a Frank V4 statement, the V4 statement governs.

---

## Core Product Decision

Frank V4 final deployment is phone-only.

The desktop is allowed but non-normative:

- development host,
- repo editing host,
- optional build host,
- optional test host,
- optional recovery host,
- optional remote inspector.

The phone is normative:

- deployed body,
- durable runtime state host,
- live mission-control host,
- improvement workspace host,
- hot-update gate host,
- last-known-good recovery host.

The permanent boundary is not:

```text
desktop = lab
phone = body
```

The permanent boundary is:

```text
live runtime plane = currently acting Frank
improvement workspace = candidate-building Frank
hot-update gate = only path from candidate to active
```

---

## Goals

- Preserve the full Frank V3 control plane and external-operation envelope.
- Make final V4 deployable on the phone only.
- Let Frank run continuously from durable standing directives.
- Add a phone-resident improvement workspace that can generate, evaluate, keep, discard, hot-update, promote, and roll back candidate runtime packs.
- Add Pi-like hot update for reloadable runtime surfaces.
- Improve prompt packs, skills, manifests, routing metadata, and allowed runtime extensions without arbitrary in-place mutation of active state.
- Keep evaluation concrete: baseline first, immutable evaluator per run, train/holdout separation, explicit pass/fail/keep/discard/promote/rollback decisions.
- Preserve auditable allow/reject decisions for all governed actions.
- Preserve a deterministic last-known-good path through versioned promotion and rollback.
- Allow low-risk autonomous hot updates where policy explicitly permits them.
- Require approval or hard rejection for updates that touch external authority, identity, finance, policy, or core control-plane surfaces.

---

## Non-Goals

Frank V4 does not include any of the following as default implementation targets:

- arbitrary active-pack file edits without a hot-update record,
- runtime-source self-rewrites of the deployed control plane,
- evaluator self-mutation during an improvement run,
- automatic mutation of treasury policy, autonomy predicate, authority tiers, campaign guardrails, or approval syntax,
- uncontrolled auto-promotion of high-risk executable/tooling changes,
- silent update during an active unsafe live job,
- parallel active runtime packs for live external work,
- a requirement for on-device model inference,
- native voice assistant UX,
- broad phone-native sensor exposure by default,
- multi-agent orchestration as a requirement,
- unrestricted external posting or spending,
- replacement of explicit V3 policy with opaque learned behavior.

These may appear later as explicit future-scope work, but they are not part of the default Frank V4 envelope.

---

## System Model

### Roles

#### Phone role

The phone is the finished V4 host.

It provides:

- deployed body,
- long-running gateway runtime,
- local workspace and process execution surface,
- local mission state writer,
- local sidecar/observer host where enabled,
- host for the active runtime pack,
- host for the isolated improvement workspace,
- host for hot-update staging, commit, and rollback.

The phone may be resource constrained. Resource constraints must be handled with budgets, scheduling, and bounded work units rather than by requiring a permanent desktop lab.

#### Desktop role

The desktop is optional.

It may provide:

- development and implementation work,
- heavy test execution,
- repo-level source editing,
- recovery workflows,
- inspection and backup,
- optional remote build/eval acceleration.

The desktop is not the permanent source of truth for runtime autonomy after V4 final deployment.

#### Live runtime plane

The live runtime plane is the currently acting Frank.

It:

- executes live governed jobs,
- consumes exactly one active runtime pack at a time,
- performs external actions only inside the V3/V4 authority envelope,
- persists mission state before reporting,
- may quiesce between steps for hot-update checks,
- must be able to roll back to last-known-good.

#### Improvement workspace

The improvement workspace is the candidate-building plane.

It:

- may run on the phone,
- may optionally run on the desktop during development,
- creates candidate prompt packs, skill packs, routing metadata, and reloadable extension packs,
- runs baseline/train/holdout/canary evals,
- emits candidate results,
- cannot directly become active without the hot-update gate.

#### Hot-update gate

The hot-update gate is the only valid transition path from candidate content to active runtime content.

It:

- validates candidate scope,
- checks mutable and forbidden surfaces,
- enforces promotion policy,
- checks compatibility and resource budgets,
- stages updates,
- quiesces the live runtime if required,
- reloads or restarts the necessary runtime components,
- commits active pack pointers,
- records update evidence,
- rolls back on failure.

#### Remote provider role

Remote model inference remains allowed for live runtime and improvement runs.

Provider choice is an implementation detail so long as:

- mission-control rules remain intact,
- eval immutability rules remain intact,
- hot-update rules remain intact,
- provider onboarding does not re-open beyond explicit accepted scope.

#### Operator channel role

Telegram and SSH/Tailscale terminal remain approved owner-control channels.

Promotion, hot-update, rollback, pause, resume, and abort commands are bound to the same operator-control surface.

Frank-owned email remains for Frank’s own operations, not automatic owner control.

---

## Runtime Shape

Frank V4 has two governed execution planes and one governed transition gate.

### Plane A — `live_runtime`

- default host: `phone`,
- purpose: live governed work,
- pack source: one committed active runtime pack,
- side effects: real artifacts, approved local services, bounded external actions inside the V3/V4 envelope,
- update behavior: may only consume candidate content through the hot-update gate.

### Plane B — `improvement_workspace`

- default host: `phone`,
- optional host during development: `desktop_dev`,
- purpose: search for better runtime packs and local procedures,
- pack source: active baseline pack plus candidate mutations,
- side effects: candidate artifacts, eval traces, scores, promotion proposals, rollback bundles.

### Gate C — `hot_update_gate`

- default host: `phone`,
- purpose: transition candidate content into active runtime content,
- side effects: staged update records, active pack pointer changes, hot reloads, process restarts where allowed, rollback records.

Every job must declare:

- `execution_plane`,
- `execution_host`,
- `mission_family`,
- `target_surfaces`,
- and whether it may invoke the hot-update gate.

Improvement-family jobs may execute on the phone only when `execution_plane=improvement_workspace`.

Live external jobs may execute only when `execution_plane=live_runtime`.

Hot-update operations may execute only through `execution_plane=hot_update_gate` or an explicit internal gate call from a governed live or improvement job.

---

## Pi-Agent Donor Policy

Pi-agent behavior is admitted as donor architecture for V4.

The admitted Pi-inspired ideas are:

- skills as self-contained capability packages,
- progressive disclosure of skill descriptions before full skill loading,
- prompt templates as reloadable runtime guidance,
- extensions as reloadable behavior modules,
- local packages as bundles of prompts, skills, themes, tools, and extension code,
- session/event logs as durable local interaction state,
- explicit reload commands and watcher-driven reloads,
- model/provider abstraction as a harness detail.

The rejected Pi-inspired ideas are:

- treating arbitrary write access as authorization,
- letting a package grant itself external authority,
- making policy surfaces hot-reloadable by default,
- installing or updating external action tools without Frank authority checks,
- letting model-generated code bypass mission-control state,
- treating “hot reload succeeded” as equivalent to “safe to act externally.”

A Pi package or Pi-style package imported into Frank becomes candidate pack content. It does not become active runtime authority by being copied into a directory.

---

## Behavior Contract

### Execution-first posture remains in force

Frank V4 remains execution-first. Improvement and hot-update work do not loosen the requirement to validate before completion, persist before report, and surface blockers honestly.

### The live runtime consumes active packs plus staged hot updates only

The live runtime may load:

- the current active runtime pack,
- a staged hot-update candidate during a governed reload operation,
- a bounded canary pack during an explicit canary,
- or the last-known-good pack during rollback.

The live runtime may not treat uncommitted workspace files as active runtime truth.

### Hot update is allowed

Frank V4 supports Pi-like hot update for approved reloadable surfaces.

Hot update means:

```text
prepare candidate -> validate scope -> stage -> quiesce if needed -> reload allowed components -> smoke check -> commit pointer -> record evidence
```

Hot update does not mean:

```text
modify arbitrary active files while tools are running and hope the process survives
```

### Active-pack mutation is transactional, not ad hoc

Frank may update the active runtime without a full manual redeploy only through a durable `HotUpdateEnvelope` and committed `HotUpdateRecord`.

The old active pack remains addressable as rollback target until replaced by a newer last-known-good policy.

### Evaluator immutability is mandatory during a run

For any single improvement run, the following are immutable:

- evaluator code,
- rubric,
- train corpus,
- holdout corpus,
- promotion policy used for that run,
- baseline pack being compared against.

A candidate may not win by editing its own test.

### Baseline-first is mandatory

Every improvement run records a baseline result before candidate mutation is marked eligible.

No candidate may be marked improved without a baseline result for the exact same target and eval suite.

### Train and holdout are distinct

The improvement workspace must record:

- train results,
- holdout results,
- and a separate promotion or hot-update decision.

A train-only win is not promotable unless the target surface is explicitly classified as non-evaluative cosmetic content and the policy says holdout is not required.

### One bounded mutation unit at a time

Within a single improvement attempt, Frank may mutate only the declared target unit for the declared phase.

Examples:

- one prompt pack,
- one target skill,
- one routing manifest entry,
- one reloadable extension pack,
- or one topology action when topology mode is enabled.

“Mutate everything and hot reload it” is non-compliant.

### Results are append-only and decision-complete

Every attempt must record one explicit outcome:

- `keep`,
- `discard`,
- `blocked`,
- `crash`,
- `hot_updated`,
- `promoted`,
- or `rolled_back`.

Historical result records are append-only. Corrections are written as new records, not silent rewrites.

### Promotion and hot update require explicit gates

No candidate becomes active runtime content until it has:

- baseline evidence when required,
- candidate train evidence when required,
- candidate holdout evidence when required,
- compatibility evidence,
- a hot-update or promotion decision,
- and a rollback target.

### Rollback is first-class

Every hot-updated or promoted pack must have an explicit last-known-good predecessor and deterministic rollback path.

### Improvement may not silently relax policy

An improvement candidate may not autonomously change:

- mission-control rules,
- authority tiers,
- approval syntax,
- autonomy predicate semantics,
- campaign guardrails,
- treasury lifecycle semantics,
- owner identity rules,
- provider onboarding rules,
- capability-exposure policy,
- observability/sidecar authority.

Those surfaces are frozen by default in V4 and are outside the autonomous hot-update surface.

### Signals may restrict or explain, never grant

The V3 rule remains in force. Body/environment/observability signals may be used to explain degraded runtime behavior or to block nonessential work, but never to grant promotion, approval, authority, or hot-update eligibility.

---

## Hot-Update Surface Policy

Frank V4 classifies update surfaces by risk.

### Class 0 — ephemeral runtime context

Examples:

- non-authoritative summaries,
- caches,
- temporary notes,
- non-policy memory extracts.

Behavior:

- may refresh live without pack promotion,
- may not override durable control-plane truth,
- must be discardable.

### Class 1 — reloadable guidance surfaces

Examples:

- prompt-pack text,
- skill descriptions,
- `SKILL.md` procedural guidance,
- routing descriptions,
- artifact-format templates,
- local non-executable reference docs.

Behavior:

- eligible for autonomous hot update when policy allows,
- requires candidate record and rollback target,
- requires smoke check and compatibility check,
- may reload without process restart.

### Class 2 — reloadable local helper surfaces

Examples:

- helper scripts inside a skill pack,
- local parser scripts,
- local artifact generators,
- non-networked analysis tools,
- deterministic eval helpers.

Behavior:

- may be hot-updated autonomously only if sandboxed and scoped,
- requires test evidence,
- requires output-shape checks,
- requires no new external side effects,
- may require process or tool-runner restart.

### Class 3 — reloadable runtime extension surfaces

Examples:

- tool adapters,
- command handlers,
- event hooks,
- context injectors,
- UI renderers,
- package loader entries.

Behavior:

- may be hot-reloaded like Pi extensions only through `RuntimeExtensionPack`,
- requires extension compatibility contract,
- requires before/after tool gating compatibility,
- requires canary for action-producing tools,
- requires approval for new external-action tools or widened permissions,
- may not grant itself capability exposure.

### Class 4 — governed external authority surfaces

Examples:

- provider onboarding,
- account identity,
- owner-control channel authority,
- external posting authority,
- financial execution authority,
- campaign stop conditions,
- outreach eligibility rules.

Behavior:

- not autonomously hot-updatable,
- may be changed only through explicit governed lanes and approvals,
- cannot be changed by pack content alone.

### Class 5 — frozen control-plane and policy surfaces

Examples:

- mission-control core code,
- task-state mutation ownership,
- authority tiers,
- approval syntax,
- treasury policy,
- autonomy predicate,
- replay/idempotence semantics,
- active-pack pointer mutation rules,
- promotion/rollback gate rules.

Behavior:

- forbidden for autonomous hot update in V4,
- may be proposed as source patches or design artifacts,
- requires external implementation/review path outside autonomous V4 hot update.

---

## Runtime Pack Model

### Pack channels

Frank V4 recognizes these channels:

- `active`,
- `staged_hot`,
- `candidate`,
- `canary`,
- `last_known_good`,
- `rejected`,
- `quarantined`.

Exactly one pack may be `active` for live external work at a time.

### Pack mutation rule

A pack is immutable after creation.

A hot update does not edit the active pack in place. It creates a new candidate pack, stages it, validates it, then atomically updates the active pointer if eligible.

### Runtime pack minimum fields

`RuntimePack` minimum fields:

- `pack_id`,
- `parent_pack_id`,
- `created_at`,
- `created_by`,
- `channel`,
- `prompt_pack_ref`,
- `skill_pack_ref`,
- `manifest_ref`,
- `extension_pack_ref`,
- `policy_ref`,
- `source_summary`,
- `mutable_surfaces`,
- `immutable_surfaces`,
- `compatibility_contract_ref`,
- `rollback_target_pack_id`.

### Active pack pointer

`ActivePackPointer` minimum fields:

- `active_pack_id`,
- `previous_active_pack_id`,
- `last_known_good_pack_id`,
- `updated_at`,
- `updated_by`,
- `update_record_ref`,
- `reload_generation`.

### Last-known-good pointer

`LastKnownGoodPackPointer` minimum fields:

- `pack_id`,
- `basis`,
- `verified_at`,
- `verified_by`,
- `rollback_record_ref`.

---

## Hot-Update Model

### HotUpdateEnvelope

Every active runtime update must be represented by a `HotUpdateEnvelope`.

Minimum fields:

- `hot_update_id`,
- `objective`,
- `candidate_pack_id`,
- `previous_active_pack_id`,
- `rollback_target_pack_id`,
- `target_surfaces`,
- `surface_classes`,
- `reload_mode`,
- `compatibility_contract_ref`,
- `eval_evidence_refs`,
- `smoke_check_refs`,
- `canary_ref`,
- `approval_ref`,
- `budget_ref`,
- `prepared_at`,
- `state`,
- `decision`,
- `failure_reason`.

### Reload modes

Frank V4 supports these reload modes:

- `soft_reload` — rebuilds prompt/skill/routing context without restarting the whole process.
- `skill_reload` — refreshes discovered skill metadata and full skill content cache.
- `extension_reload` — quiesces tool/extension runner, unloads old extension set, loads new extension set, rebuilds tool catalog.
- `pack_reload` — activates a full new runtime pack pointer and refreshes all reloadable pack components.
- `canary_reload` — temporarily routes bounded canary work to the candidate pack.
- `process_restart_hot_swap` — restarts the runtime process using the staged pack and resumes from durable state.
- `cold_restart_required` — not a V4 autonomous hot update; requires explicit operator-controlled deployment/restart.

### Hot update states

`HotUpdateEnvelope.state` values:

- `draft`,
- `prepared`,
- `validated`,
- `staged`,
- `quiescing`,
- `reloading`,
- `smoke_testing`,
- `canarying`,
- `committed`,
- `rolled_back`,
- `rejected`,
- `failed`,
- `aborted`.

### Hot update decision values

`HotUpdateEnvelope.decision` values:

- `keep_staged`,
- `discard`,
- `block`,
- `apply_hot_update`,
- `apply_canary`,
- `require_approval`,
- `require_cold_restart`,
- `rollback`.

### Quiesce rule

Hot update may apply only at a quiesce point.

A quiesce point is valid when:

- no tool call is currently executing,
- no governed external side effect is in flight,
- the active step is between atomic actions,
- durable state is persisted,
- active live job safety policy permits reload,
- and rollback target is available.

If quiesce cannot be reached, the hot update remains staged or is rejected with a deploy-lock rejection code.

### Commit rule

A hot update is committed only when:

- active pointer update is durable,
- reload generation increments,
- smoke checks pass,
- rollback target is durable,
- and `HotUpdateRecord` is appended.

### Rollback rule

Rollback may occur:

- automatically when reload fails before commit,
- automatically when smoke checks fail after reload,
- automatically when canary violates policy,
- by operator command,
- by repeated crash detection,
- by explicit mission-control rejection.

Rollback restores the previous active pack pointer or last-known-good pointer and records `RollbackRecord`.

---

## Autonomous Hot-Promotion Policy

Frank V4 may autonomously hot-update low-risk surfaces.

Autonomous hot update is allowed only when all of these are true:

- target surfaces are Class 0, Class 1, or allowed Class 2,
- candidate does not add external authority,
- candidate does not widen tool permissions,
- candidate does not mutate policy/control-plane/treasury/approval surfaces,
- baseline and evaluation requirements for the policy class are satisfied,
- rollback target exists,
- active live job state permits quiesce,
- update budget permits it,
- no repeated-failure pause is active.

Approval is required when any of these are true:

- Class 3 update adds or changes an action-producing tool,
- Class 3 update changes event hooks that can block/allow external side effects,
- candidate affects owner identity or provider identity,
- candidate affects outreach/campaign execution behavior beyond wording/procedure improvements,
- candidate affects spending, transfers, treasury lifecycle, or revenue receipt handling,
- canary would perform real external side effects,
- update requires `process_restart_hot_swap` while unsafe live work is active,
- update would change capability exposure state.

Autonomous hot update is forbidden when any of these are true:

- candidate changes authority tiers,
- candidate changes approval syntax,
- candidate changes autonomy predicate semantics,
- candidate changes treasury policy,
- candidate changes mission-control core mutation ownership,
- candidate changes replay/idempotence semantics,
- candidate changes active-pack pointer mutation rules,
- candidate attempts to edit evaluator/rubric/train/holdout surfaces during the same run.

---

## Continuous Autonomy Model

Frank V4 is intended to run continuously.

### StandingDirective

A `StandingDirective` defines durable long-running intent.

Minimum fields:

- `standing_directive_id`,
- `objective`,
- `allowed_mission_families`,
- `allowed_execution_planes`,
- `allowed_execution_hosts`,
- `autonomy_envelope_ref`,
- `budget_ref`,
- `schedule`,
- `success_criteria`,
- `stop_conditions`,
- `owner_pause_state`,
- `created_at`,
- `updated_at`.

### AutonomyEnvelope

An `AutonomyEnvelope` defines what Frank may do without asking.

Minimum fields:

- `autonomy_envelope_id`,
- `mode`,
- `allowed_live_families`,
- `allowed_improvement_families`,
- `allowed_hot_update_classes`,
- `approval_required_for`,
- `forbidden_surfaces`,
- `max_risk_class`,
- `created_at`.

### AutonomyBudget

An `AutonomyBudget` bounds continuous work.

Minimum fields:

- `budget_id`,
- `max_external_actions_per_day`,
- `max_hot_updates_per_day`,
- `max_candidate_mutations_per_day`,
- `max_api_spend_per_day`,
- `max_runtime_minutes_per_cycle`,
- `max_failed_attempts_before_pause`,
- `quiet_hours`,
- `reset_window`,
- `ledger_refs`.

### WakeCycleRecord

A `WakeCycleRecord` records each autonomous tick.

Minimum fields:

- `wake_cycle_id`,
- `started_at`,
- `completed_at`,
- `trigger`,
- `selected_directive_id`,
- `selected_job_id`,
- `decision`,
- `budget_debits`,
- `blocked_reasons`,
- `next_wake_at`.

### Autonomous job selection order

Default priority order:

1. owner command,
2. unresolved blocker / approval,
3. safety / rollback / failure handling,
4. active job continuation,
5. active hot-update completion or rollback,
6. standing directive with due schedule,
7. maintenance task,
8. improvement workspace task,
9. idle heartbeat.

Frank may not invent a new authority path because no eligible autonomous action exists.

---

## Mission-Control V4 Scope

### One-active-live-job rule remains in force

The live runtime executes at most one governed live job at a time.

Improvement-workspace tasks may run concurrently only when they cannot interfere with live runtime resource budgets, active job safety, or hot-update quiesce rules.

### Supported V4 step types

Frank V4 keeps the same frozen step taxonomy as V3:

- `discussion`,
- `static_artifact`,
- `one_shot_code`,
- `long_running_code`,
- `system_action`,
- `wait_user`,
- `final_response`.

V4 adds new mission families, pack contracts, and hot-update gates, not a broader step taxonomy.

### Planning normalization

Every directive still normalizes to:

```text
directive -> job -> plan -> validate -> execute step -> audit -> final response
```

Improvement families follow the same loop, but execute in `improvement_workspace` and emit candidate-pack and eval artifacts.

Hot-update families follow:

```text
candidate -> envelope -> validate -> stage -> quiesce -> reload -> smoke/canary -> commit or rollback -> audit
```

### Governed-action rule

All governed actions require:

- an active job,
- an active step,
- a valid execution plane,
- a valid execution host,
- a valid target surface,
- a valid authority outcome,
- and all required approvals.

---

## Directive Intake and Mission Formation

### Intake classes

Frank V4 recognizes these directive classes:

- live execution request,
- live research / monitoring / maintenance request,
- external-operation request,
- standing autonomy request,
- improvement request,
- hot-update request,
- promotion / rollback request,
- unsupported or ambiguous request.

### Intake rule

Open-ended directives such as:

- “improve yourself,”
- “be more like Pi,”
- “hot update yourself,”
- “run autonomously forever,”
- “get better at outreach,”
- “earn at least $1 better than before,”

must first be translated into a bounded mission proposal unless an existing standing directive already supplies the required scope.

No open-ended improvement loop may start without:

- explicit target surface,
- explicit execution plane,
- explicit host,
- explicit eval or smoke-check suite,
- explicit success criteria,
- explicit stop conditions,
- explicit mutation phase,
- explicit hot-update or promotion policy.

### Mission proposal object

Every V4 mission proposal must include at least:

- `job_id`,
- `objective`,
- `execution_plane`,
- `execution_host`,
- `mission_family`,
- `constraints`,
- `success_criteria`,
- `stop_conditions`,
- `target_surfaces`,
- `surface_classes`,
- `immutable_surfaces`,
- `approval_requirements`,
- `expected_outputs`,
- `hot_update_allowed`.

### Default translation rule

If a directive requests self-improvement, Frank proposes or resumes a bounded improvement mission instead of treating the request as permission to edit live active files directly.

If a directive requests Pi-like hot update, Frank uses a hot-update envelope over allowed reloadable surfaces instead of requiring a permanent desktop lab.

---

## Mission Families

### Enabled live families carried forward from V3

Frank V4 preserves these live families from V3:

- `build`,
- `research`,
- `monitor`,
- `operate`,
- `maintenance`,
- `outreach`,
- `community_discovery`,
- `opportunity_scan`,
- `bootstrap_revenue`,
- `bootstrap_identity_and_accounts`.

### New autonomy families in V4

Frank V4 adds:

- `continuous_autonomy_tick`,
- `standing_directive_review`,
- `autonomous_mission_proposal`,
- `autonomy_budget_report`,
- `autonomy_pause`,
- `autonomy_resume`.

### New improvement families in V4

Frank V4 adds:

- `improve_promptpack`,
- `improve_skills`,
- `improve_routing_manifest`,
- `improve_runtime_extension`,
- `evaluate_candidate`,
- `promote_candidate`,
- `rollback_candidate`,
- `improve_topology`,
- `propose_source_patch`.

### New hot-update families in V4

Frank V4 adds:

- `prepare_hot_update`,
- `validate_hot_update`,
- `stage_hot_update`,
- `apply_hot_update`,
- `smoke_test_hot_update`,
- `canary_hot_update`,
- `commit_hot_update`,
- `rollback_hot_update`.

### Family rules

- Improvement families execute in `improvement_workspace`, not `live_runtime`.
- Improvement families may execute on `phone` or `desktop_dev`; final V4 target is `phone`.
- Hot-update families execute through `hot_update_gate`.
- `propose_source_patch` may generate a source patch as an artifact, but may not autonomously apply or deploy runtime-source changes in V4.
- `improve_topology` is feature-gated. If topology mode is disabled, add/remove/split/merge skill-pack operations are rejected.
- Live jobs may consume only active, canary, or hot-update-staged packs through the loader.

---

## Runtime State Model

### Job states

Frank V4 retains the V3 job states:

- `draft`,
- `planned`,
- `ready`,
- `executing`,
- `paused`,
- `blocked`,
- `succeeded`,
- `failed`,
- `aborted`.

### Step states

Frank V4 retains the V3 step states:

- `pending`,
- `ready`,
- `running`,
- `succeeded`,
- `failed`,
- `blocked`,
- `waiting`,
- `aborted`.

### Approval states

Frank V4 retains the V3 approval states:

- `not_required`,
- `requested`,
- `granted`,
- `denied`,
- `expired`,
- `superseded`.

### Improvement-run states

Every improvement run has one of these states:

- `queued`,
- `baselining`,
- `mutating`,
- `evaluating_train`,
- `evaluating_holdout`,
- `candidate_ready`,
- `staged_for_hot_update`,
- `canarying`,
- `hot_updated`,
- `promoted`,
- `rejected`,
- `rolled_back`,
- `failed`,
- `aborted`.

### Candidate-pack states

Every candidate pack has one of these states:

- `draft`,
- `evaluated_train`,
- `evaluated_holdout`,
- `staged_hot`,
- `canary_ready`,
- `canarying`,
- `hot_updated`,
- `promoted`,
- `rejected`,
- `rolled_back`,
- `quarantined`.

### Hot-update states

Every hot-update attempt has one of these states:

- `draft`,
- `prepared`,
- `validated`,
- `staged`,
- `quiescing`,
- `reloading`,
- `smoke_testing`,
- `canarying`,
- `committed`,
- `rolled_back`,
- `rejected`,
- `failed`,
- `aborted`.

---

## Step Completion Contracts

Frank V4 preserves the V3 completion contracts for all seven step types.

Improvement-specific and hot-update-specific requirements are additional, not replacements.

### For `improve_promptpack`, `improve_skills`, `improve_routing_manifest`, and `improve_runtime_extension`

A step is not complete until:

- baseline result exists when required,
- candidate artifact exists,
- required train and holdout evaluations or smoke checks have completed,
- and an explicit keep/discard/block/crash/stage decision has been recorded.

### For `evaluate_candidate`

A step is not complete until:

- candidate train results are recorded when required,
- candidate holdout results are recorded when required,
- evaluator version is recorded,
- rubric version is recorded,
- surface class is recorded,
- regression flags are recorded.

### For `prepare_hot_update`

A step is not complete until:

- candidate pack exists,
- previous active pack is recorded,
- rollback target exists,
- target surfaces and surface classes are recorded,
- reload mode is selected,
- hot-update policy is linked.

### For `validate_hot_update`

A step is not complete until:

- mutable-surface checks pass,
- forbidden-surface checks pass,
- compatibility checks pass,
- budget checks pass,
- deploy-lock checks pass,
- approval requirements are resolved or recorded.

### For `apply_hot_update`

A step is not complete until:

- valid quiesce point is reached or deploy lock is recorded,
- staged candidate is loaded through the correct reload mode,
- smoke check result is recorded,
- active pointer is either committed or explicitly unchanged,
- failure path rolls back or remains staged.

### For `commit_hot_update`

A step is not complete until:

- active-pack pointer update is durable,
- reload generation increments,
- last-known-good pointer is preserved or updated according to policy,
- `HotUpdateRecord` is appended.

### For `rollback_hot_update` and `rollback_candidate`

A step is not complete until:

- active-pack pointer is reset or verified already correct,
- rollback target is verified,
- rollback event is durably recorded,
- quarantined candidate state is recorded when required.

---

## Core Runtime Abstractions

Frank V4 carries forward the V3 runtime abstractions, including:

- `Job`,
- `Plan`,
- `Step`,
- `ApprovalRequest`,
- `ApprovalGrant`,
- `AuditEvent`,
- `ArtifactRecord`,
- `CapabilityRecord`,
- `MissionStatusSnapshot`,
- `MissionStepControl`,
- `IdentityObject`,
- `AccountObject`,
- `EligibilityCheck`,
- `PlatformRecord`,
- `Campaign`,
- `Treasury`,
- `LedgerEntry`,
- `CapabilityOnboardingProposal`,
- `BodySignalsSnapshot`,
- `EnvironmentSignalsSnapshot`,
- `ObserverReport`.

Frank V4 adds the following public concepts.

### ExecutionPlane

Required on every job.

Minimum values:

- `live_runtime`,
- `improvement_workspace`,
- `hot_update_gate`.

### ExecutionHost

Required on every job.

Minimum values:

- `phone`,
- `desktop_dev`,
- `remote_provider`.

### RuntimePack

The versioned bundle the live runtime consumes.

Minimum fields:

- `pack_id`,
- `parent_pack_id`,
- `created_at`,
- `channel`,
- `prompt_pack_ref`,
- `skill_pack_ref`,
- `manifest_ref`,
- `extension_pack_ref`,
- `policy_ref`,
- `source_summary`,
- `mutable_surfaces`,
- `immutable_surfaces`,
- `surface_classes`,
- `compatibility_contract_ref`,
- `rollback_target_pack_id`.

### PromptPack

The versioned prompt/rules bundle for runtime behavior.

Minimum fields:

- `prompt_pack_id`,
- `files`,
- `parent_prompt_pack_id`,
- `change_summary`,
- `created_by`,
- `surface_class`,
- `hot_reloadable`.

### SkillPack

The versioned skill bundle used for routing and procedure execution.

Minimum fields:

- `skill_pack_id`,
- `manifest_ref`,
- `skills`,
- `parent_skill_pack_id`,
- `change_summary`,
- `created_by`,
- `surface_class`,
- `hot_reloadable`.

### RuntimeExtensionPack

The versioned bundle of reloadable tool/event/command/UI adapters.

Minimum fields:

- `extension_pack_id`,
- `extensions`,
- `parent_extension_pack_id`,
- `change_summary`,
- `created_by`,
- `declared_tools`,
- `declared_events`,
- `declared_permissions`,
- `external_side_effects`,
- `compatibility_contract_ref`,
- `hot_reloadable`,
- `approval_required`.

### EvalSuite

The immutable evaluator surface for an improvement run.

Minimum fields:

- `eval_suite_id`,
- `rubric_ref`,
- `train_corpus_ref`,
- `holdout_corpus_ref`,
- `evaluator_ref`,
- `negative_case_count`,
- `boundary_case_count`,
- `frozen_for_run`.

### ImprovementRun

The durable record of one bounded improvement search.

Minimum fields:

- `run_id`,
- `objective`,
- `execution_plane`,
- `execution_host`,
- `mission_family`,
- `target_type`,
- `target_ref`,
- `surface_class`,
- `baseline_pack_id`,
- `candidate_pack_id`,
- `eval_suite_id`,
- `state`,
- `decision`,
- `created_at`,
- `completed_at`,
- `stop_reason`.

### ImprovementTarget

The explicitly bounded mutable surface for a run.

Minimum values:

- `prompt_pack`,
- `skill`,
- `routing_manifest_entry`,
- `runtime_extension`,
- `skill_topology`,
- `source_patch_artifact`.

### CandidateResult

The scored result for one candidate.

Minimum fields:

- `candidate_pack_id`,
- `baseline_score`,
- `train_score`,
- `holdout_score`,
- `complexity_score`,
- `compatibility_score`,
- `resource_score`,
- `regression_flags`,
- `decision`,
- `notes`.

### PromotionPolicy

The rule set that determines whether a candidate may be promoted or hot-updated.

Minimum fields:

- `promotion_policy_id`,
- `requires_holdout_pass`,
- `requires_canary`,
- `requires_owner_approval`,
- `allows_autonomous_hot_update`,
- `allowed_surface_classes`,
- `epsilon_rule`,
- `regression_rule`,
- `compatibility_rule`,
- `resource_rule`,
- `max_canary_duration`,
- `forbidden_surface_changes`.

### HotUpdatePolicy

The rule set for applying hot updates.

Minimum fields:

- `hot_update_policy_id`,
- `allowed_reload_modes`,
- `allowed_surface_classes`,
- `quiesce_required`,
- `smoke_check_required`,
- `canary_required`,
- `approval_required`,
- `max_updates_per_window`,
- `deploy_lock_rule`,
- `rollback_rule`,
- `forbidden_surface_changes`.

### HotUpdateRecord

The durable record of a hot-update attempt.

Minimum fields:

- `hot_update_id`,
- `candidate_pack_id`,
- `previous_active_pack_id`,
- `new_active_pack_id`,
- `rollback_target_pack_id`,
- `reload_mode`,
- `approval_ref`,
- `canary_ref`,
- `smoke_check_ref`,
- `committed_at`,
- `committed_by`,
- `reload_generation`,
- `state`,
- `decision`.

### PromotionRecord

The durable record of a promotion attempt.

Minimum fields:

- `promotion_id`,
- `candidate_pack_id`,
- `previous_active_pack_id`,
- `new_active_pack_id`,
- `approval_ref`,
- `canary_ref`,
- `promoted_at`,
- `promoted_by`.

### RollbackRecord

The durable record of a rollback.

Minimum fields:

- `rollback_id`,
- `from_pack_id`,
- `to_pack_id`,
- `trigger`,
- `rolled_back_at`,
- `rolled_back_by`,
- `reload_generation_before`,
- `reload_generation_after`.

### ImprovementLedgerEntry

The append-only experiment ledger entry.

Minimum fields:

- `ledger_entry_id`,
- `run_id`,
- `target_ref`,
- `baseline_pack_id`,
- `candidate_pack_id`,
- `status`,
- `summary`,
- `created_at`.

### HotUpdateLedgerEntry

The append-only hot-update ledger entry.

Minimum fields:

- `ledger_entry_id`,
- `hot_update_id`,
- `candidate_pack_id`,
- `previous_active_pack_id`,
- `new_active_pack_id`,
- `reload_mode`,
- `status`,
- `summary`,
- `created_at`.

---

## Plan Validation Rules

All V3 plan-validation rules remain in force.

Frank V4 adds these rules.

1. **Execution plane is mandatory**  
   Every job must declare `execution_plane`.

2. **Execution host is mandatory**  
   Every job must declare `execution_host`.

3. **Improvement families require improvement workspace**  
   Any improvement family with `execution_plane != improvement_workspace` is invalid.

4. **Phone is valid for improvement workspace**  
   `execution_host=phone` is valid for improvement families when the job runs in `improvement_workspace`.

5. **Hot-update families require hot-update gate**  
   Any hot-update family outside `hot_update_gate` is invalid unless it is an internal governed gate call from a validated plan.

6. **Mutable surfaces must be explicit**  
   Every improvement or hot-update plan must enumerate exactly which files, skills, manifest entries, extension modules, or topology actions are mutable.

7. **Surface classes must be explicit**  
   Every mutable target must declare its hot-update surface class.

8. **Immutable surfaces must be explicit**  
   Every improvement plan must enumerate evaluator, rubric, train corpus, holdout corpus, promotion policy, hot-update policy, and baseline pack as immutable for the run.

9. **Baseline is required when policy requires evaluation**  
   No candidate mutation may be marked eligible until the baseline run definition exists.

10. **Holdout is required for promotion when policy requires it**  
   A candidate without required holdout evaluation is not promotable.

11. **Smoke check is required for hot update**  
   No hot update may commit without a smoke check result unless policy explicitly marks the target as Class 0 ephemeral context.

12. **Canary requirement is policy-controlled**  
   If hot-update or promotion policy requires canary, commit is invalid without canary scope and evidence.

13. **Forbidden surfaces are out of scope**  
   Plans that mutate evaluator, control plane, authority model, treasury policy, approval syntax, autonomy predicate, or active-pack gate rules are invalid in V4.

14. **Topology mode is explicit**  
   Add/remove/split/merge skill operations require `improve_topology` and an enabled topology flag.

15. **Source patches are artifact-only**  
   `propose_source_patch` may produce a patch artifact, but any plan that applies or deploys runtime-source mutations autonomously is invalid in V4.

16. **Hot update requires rollback target**  
   Any hot-update plan without explicit rollback target is invalid.

17. **Hot update requires quiesce or restart policy**  
   A plan must state how the runtime reaches a safe reload point.

18. **Extension update requires compatibility contract**  
   A runtime extension update is invalid without declared tools, events, permissions, and compatibility contract.

19. **External authority may not be granted by package content**  
   A package declaring a tool or skill does not grant capability exposure, provider authority, spending authority, owner identity authority, or campaign authority.

20. **No active unsafe deploy**  
   Hot update is blocked when the current live job or step has an unsafe deploy lock.

---

## Persistence, Replay, and Idempotence

### Source of truth

Frank V4 has these durable truth surfaces:

- live runtime state for governed jobs and steps,
- pack registry,
- active pack pointer,
- last-known-good pointer,
- improvement ledger,
- hot-update ledger,
- autonomy governor registry,
- rollback records.

### Durable records required

At minimum, the following must be durable:

- job and step state,
- approvals and grants,
- audit events,
- standing directives,
- autonomy budgets,
- wake-cycle records,
- active pack pointer,
- last-known-good pack pointer,
- candidate pack records,
- eval-suite records,
- improvement runs,
- hot-update envelopes,
- hot-update records,
- promotion records,
- rollback records,
- append-only improvement ledger entries,
- append-only hot-update ledger entries.

### Replay safety

Replay may reconstruct state, but must not:

- re-commit the same hot update twice,
- re-promote the same candidate twice,
- emit duplicate live external side effects,
- silently lose the last-known-good pointer,
- fabricate a successful holdout/canary result,
- fabricate a successful smoke check,
- or widen authority based on package content.

### Idempotence rules for pack operations

- Re-reading a candidate pack is idempotent.
- Re-evaluating a candidate pack creates a new evaluation record or dedupes by `attempt_id`; it does not overwrite old results.
- Re-applying the same hot-update record is invalid after commit and returns `E_HOT_UPDATE_ALREADY_APPLIED`.
- Re-applying rollback to the already-active last-known-good pack is a no-op with explicit `already_present` outcome.
- Re-staging an identical candidate for the same hot-update envelope returns `already_present`.
- Replaying a failed hot update may only retry if retry policy permits and a new attempt id is created.

### Restart behavior

- A phone restart reloads only the committed active pack unless recovery mode chooses last-known-good.
- An improvement-workspace restart may resume an improvement run only from durable run state.
- A crash during hot update before commit fails closed to previous active pack.
- A crash after active pointer commit must either complete smoke check or roll back according to recovery policy.
- A repeated crash on the same pack quarantines that pack and rolls back to last-known-good.

---

## Authority Model

Frank V4 preserves V3 authority tiers and tightens hot-update behavior.

### Tier A — Observe

Allowed without special improvement approval:

- read candidate metadata,
- read eval suites,
- inspect scores,
- inspect active and last-known-good pack IDs,
- inspect ledger entries,
- inspect hot-update envelope state,
- inspect reload generation.

### Tier B — Prepare

Allowed as bounded preparation:

- create candidate pack artifacts in improvement workspace,
- stage eval runs,
- stage hot-update envelopes,
- generate source patch artifacts without applying them,
- generate rollback bundles,
- run compatibility checks.

### Tier C — Execute permitted autonomous improvement and hot-update actions

Allowed autonomously when scoped to permitted mutable surfaces:

- run baseline evals,
- mutate a prompt pack,
- mutate one target skill,
- mutate manifest metadata for the target skill,
- mutate allowed Class 2 helper scripts,
- run train and holdout evaluation,
- run smoke checks,
- discard candidates,
- keep candidate artifacts,
- stage a canary candidate,
- hot-update Class 1 surfaces when policy permits,
- hot-update allowed Class 2 surfaces when sandbox, eval, and smoke checks pass.

### Tier D — Approval required

Approval is required for:

- hot-updating Class 3 extension packs that add or alter action-producing tools,
- promoting a candidate that affects live external operation behavior,
- running a canary that has real external side effects,
- widening topology scope beyond one target skill,
- any owner-identity action,
- any live financial execution already gated in V3,
- any capability exposure change,
- any provider onboarding change,
- any process-restart hot swap during unsafe or high-value work.

### Tier E — Forbidden by default

Forbidden in V4:

- mutating evaluator during a run,
- mutating runtime-source control-plane code autonomously,
- mutating authority tiers or approval syntax autonomously,
- mutating autonomy predicate or treasury policy autonomously,
- self-deploying core control-plane source changes through hot update,
- treating copied files as active without a hot-update record,
- installing unreviewed packages that declare external tools and treating them as authorized.

---

## Improvement Workspace

### Purpose

The improvement workspace exists to improve Frank’s packs, skill surfaces, routing surfaces, and allowed extension surfaces through bounded, evaluated experiments.

It is not a second uncontrolled live operator body.

### Execution boundary

- The workspace may run on the phone.
- The workspace may run on desktop during development.
- The workspace operates on disposable or versioned working roots.
- The live runtime consumes outputs only through the hot-update gate.
- The workspace cannot directly overwrite the active pack pointer.

### Workspace layout guidance

A reference phone-resident layout:

```text
frank/
  live/
    active_pack_pointer.json
    last_known_good_pointer.json
    runtime_state/
    logs/
  workspace/
    candidate_packs/
    promptpacks/
    skillpacks/
    extensionpacks/
    manifests/
    evals/
    runs/
    scratch/
  hot_updates/
    envelopes/
    staged/
    committed/
    rollback/
    ledger.jsonl
  autonomy/
    standing_directives/
    budgets/
    wake_cycles.jsonl
```

This layout is guidance only. The behavior contract is normative.

### Improvement phases

Frank V4 supports these phases.

#### Phase A — prompt-pack improvement

Mutable surfaces:

- runtime instruction files,
- routing descriptions,
- bounded behavioral guidance files,
- artifact templates.

Default hot-update class:

- Class 1.

#### Phase B — skill improvement

Mutable surfaces:

- one target skill directory,
- its metadata,
- related manifest entry for that target,
- local helper scripts when allowed.

Default hot-update class:

- Class 1 for docs/instructions,
- Class 2 for helper scripts.

#### Phase C — routing-manifest improvement

Mutable surfaces:

- skill descriptions,
- routing metadata,
- model/context loading hints,
- progressive-disclosure manifests.

Default hot-update class:

- Class 1.

#### Phase D — runtime-extension improvement

Mutable surfaces:

- reloadable local extension modules,
- local command handlers,
- context injectors,
- renderers,
- event hooks,
- tool wrappers.

Default hot-update class:

- Class 3.

Class 3 updates require compatibility contract and may require approval.

#### Phase E — topology improvement

Mutable surfaces:

- add/remove/split/merge skill-pack operations,
- manifest topology changes.

Phase E is feature-gated and disabled by default.

#### Phase F — source patch proposal

Mutable surfaces:

- patch artifacts only.

Generated source patches are review artifacts in V4, not autonomously deployed runtime-source changes.

### Immutable surfaces per run

The following are frozen for a single run:

- evaluator,
- rubric,
- train corpus,
- holdout corpus,
- promotion policy,
- hot-update policy,
- baseline pack,
- previous ledger entries,
- upstream reference material.

Historical result records are append-only. New findings create new entries.

### Evaluation design requirements

Every eval suite must include:

- positive cases,
- negative cases,
- boundary cases,
- deterministic or bounded graders where feasible,
- artifact-shape checks where relevant,
- guardrail checks where relevant,
- resource-budget checks where relevant,
- rollback/smoke checks for hot-updateable surfaces.

### Improvement decision rule

A candidate may be kept when one of these is true:

- holdout score improves without introducing a disallowed regression,
- holdout ties within epsilon, complexity is lower, and no regression flag is raised,
- target is Class 0 or low-risk Class 1 and smoke/shape checks pass under a policy that does not require holdout.

Train-only wins are insufficient for general promotion.

### Hot-update pipeline

The minimum hot-update path is:

```text
baseline -> candidate mutation -> train eval -> holdout eval -> prepare envelope -> validate -> stage -> quiesce -> reload -> smoke check -> commit -> rollback ready
```

For low-risk Class 1 targets, policy may reduce this to:

```text
candidate mutation -> shape check -> smoke check -> prepare envelope -> validate -> stage -> reload -> commit -> rollback ready
```

### Canary rules

Canary is bounded and explicit.

- Canary scope must declare which jobs and surfaces are allowed.
- A canary may use replayed internal jobs, benign artifact tasks, or explicitly approved low-risk live tasks.
- A canary may not silently issue new irreversible external actions unless the canary plan and approval explicitly permit that scope.
- A canary must record which pack handled each canary action.

### Rollback rules

- Every committed hot update has an explicit predecessor.
- Rollback may be triggered by regression, crash, invalid deployment, smoke failure, canary failure, or operator command.
- Rollback returns the active-pack pointer to last-known-good or previous active pack and records the trigger.
- A rolled-back candidate is quarantined unless policy says it may be revised.

---

## Donor Subsystems and Package Policy

Frank V4 may operate over versioned static skills, manifests, prompts, extensions, and routing surfaces imported or adapted from donor projects.

Pi-style donor packages are allowed as candidate content when they are represented as versioned pack content.

The improvement workspace may optimize admitted donor surfaces only when they are represented as versioned pack content.

The hot-update gate may activate admitted donor surfaces only when:

- surface class is declared,
- package content is resolved and pinned,
- compatibility contract exists,
- required eval/smoke/canary evidence exists,
- required approval exists,
- rollback target exists.

The improvement workspace and hot-update gate may not use donor modules to bypass:

- mission control,
- approvals,
- identity boundaries,
- campaign requirements,
- treasury rules,
- capability exposure,
- provider onboarding rules,
- or pack activation rules.

---

## Deployment Model

- Android phone via Termux and Termux:Boot remains the intended deployed body.
- Final V4 target is phone-only.
- The phone hosts both the live runtime and improvement workspace.
- The desktop may remain as a development or recovery host, but not a permanent runtime requirement.
- Remote model provider remains allowed for both live runtime and improvement workspace.
- Text-first operation remains default.
- Resource and permission minimization on the phone remains required.
- The phone carries exactly one active runtime pack for live work at a time.
- Candidate packs live in the improvement workspace until staged, hot-updated, promoted, rejected, or discarded.
- Hot update changes active state only by committed pointer update, not raw file copy.

---

## Logging, Notifications, and Budgets

### Logging

Frank V4 requires:

- continuous live runtime logs,
- continuous improvement-workspace logs,
- append-only improvement ledger entries,
- append-only hot-update ledger entries,
- promotion and rollback records,
- daily packaging of live logs,
- durable storage of evaluation summaries,
- reload generation records,
- crash/recovery records.

### Notifications

Mandatory notifications include:

- blockers,
- completions,
- approval requests,
- failures,
- repeated-failure pauses,
- promotion proposals,
- hot-update proposals when approval is required,
- committed hot updates,
- promotions,
- rollbacks,
- canary failures,
- quarantined packs.

### Default budget ceilings

Frank V4 retains V3 live-job ceilings and adds these V4 ceilings by default:

- max candidate mutations per run: `20`,
- max topology operations per run: `1`,
- max concurrent improvement runs on phone: `1`,
- max queued improvement runs: `3`,
- max canary duration per candidate: `24h`,
- max autonomous hot updates per day: `5`,
- max Class 3 autonomous hot updates per day: `0`,
- max automatic discard streak before pause: `10`,
- max repeated crashes before rollback: `2`,
- max failed hot-update attempts before pause: `3`.

These are ceilings, not optimization targets.

---

## Operator Interfaces to Standardize

Frank V4 preserves the V3 operator surface and adds autonomy, pack, and hot-update controls.

At minimum, the following remain standardized:

- status snapshot output,
- mission inspection,
- mission assert / assert-step surfaces,
- explicit step switching through the control path,
- explicit operator commands for:
  - `APPROVE`,
  - `DENY`,
  - `PAUSE`,
  - `RESUME`,
  - `ABORT`.

Frank V4 adds these operator concepts:

- inspect active pack,
- inspect last-known-good pack,
- inspect candidate pack,
- inspect staged hot update,
- inspect reload generation,
- inspect improvement run,
- inspect standing directive,
- inspect autonomy budget,
- request hot update,
- approve hot update,
- reject hot update,
- trigger rollback,
- pause autonomous hot updates,
- resume autonomous hot updates.

### Channel rule

A natural-language “yes” or “no” binds only to the most recent unresolved approval request and only when there is exactly one unresolved request in scope.

### Pack-control rule

Pack activation, hot update, promotion, and rollback changes are valid only through the governed control surface.

A file-copy into the active pack directory is not a valid hot-update event.

### Pi-like reload command

Frank V4 should expose a Pi-like reload command, but the command must invoke the hot-update gate.

Recommended command forms:

```text
/reload
/reload skills
/reload prompts
/reload extensions
/hot-update inspect
/hot-update stage <candidate>
/hot-update apply <candidate>
/hot-update rollback
```

`/reload` may refresh Class 0 or already-committed active content directly. It may not activate uncommitted candidate content.

---

## Constraints and Guardrails

- fail closed when mission gating is required,
- no governed execution without active mission context,
- no ad hoc active-pack mutation,
- no unsupported step types,
- no improvement-family execution inside `live_runtime`,
- improvement-family execution on phone is allowed only inside `improvement_workspace`,
- no autonomous mutation of frozen policy surfaces,
- no hot update without rollback target,
- no hot update without smoke check unless Class 0 policy permits,
- no promotion without required baseline, holdout, and rollback target,
- no silent canary with irreversible external actions,
- no replacement of explicit policy with opaque learned behavior,
- no duplicate hot-update side effects on replay,
- no broad sensor or data exposure by default,
- no observability or sidecar file as an approval substitute,
- no continuation of outreach or community work after campaign stop conditions trigger,
- no environment-parity requirement for identical configs across dev host and phone,
- no package content as authority grant,
- no extension hot reload that widens external authority without approval.

---

## Rejection Codes

Frank V4 requires machine-readable rejection codes at least for these additional classes beyond V3:

- `E_EXECUTION_PLANE_REQUIRED`,
- `E_EXECUTION_HOST_REQUIRED`,
- `E_IMPROVEMENT_WORKSPACE_REQUIRED`,
- `E_HOT_UPDATE_GATE_REQUIRED`,
- `E_BASELINE_REQUIRED`,
- `E_HOLDOUT_REQUIRED`,
- `E_SMOKE_CHECK_REQUIRED`,
- `E_EVAL_IMMUTABLE`,
- `E_MUTATION_SCOPE_VIOLATION`,
- `E_SURFACE_CLASS_REQUIRED`,
- `E_FORBIDDEN_SURFACE_CHANGE`,
- `E_TOPOLOGY_CHANGE_DISABLED`,
- `E_PROMOTION_POLICY_REQUIRED`,
- `E_HOT_UPDATE_POLICY_REQUIRED`,
- `E_CANARY_REQUIRED`,
- `E_PROMOTION_APPROVAL_REQUIRED`,
- `E_HOT_UPDATE_APPROVAL_REQUIRED`,
- `E_ACTIVE_JOB_DEPLOY_LOCK`,
- `E_PACK_NOT_FOUND`,
- `E_LAST_KNOWN_GOOD_REQUIRED`,
- `E_CANARY_FAILED`,
- `E_SMOKE_CHECK_FAILED`,
- `E_ROLLBACK_REQUIRED`,
- `E_PROMOTION_ALREADY_APPLIED`,
- `E_HOT_UPDATE_ALREADY_APPLIED`,
- `E_RELOAD_MODE_UNSUPPORTED`,
- `E_RELOAD_QUIESCE_FAILED`,
- `E_EXTENSION_COMPATIBILITY_REQUIRED`,
- `E_EXTENSION_PERMISSION_WIDENING`,
- `E_RUNTIME_SOURCE_MUTATION_FORBIDDEN`,
- `E_POLICY_MUTATION_FORBIDDEN`,
- `E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN`,
- `E_AUTONOMY_ENVELOPE_REQUIRED`,
- `E_STANDING_DIRECTIVE_REQUIRED`,
- `E_AUTONOMY_BUDGET_EXCEEDED`,
- `E_NO_ELIGIBLE_AUTONOMOUS_ACTION`,
- `E_AUTONOMY_PAUSED`,
- `E_EXTERNAL_ACTION_LIMIT_REACHED`,
- `E_REPEATED_FAILURE_PAUSE`,
- `E_PACKAGE_AUTHORITY_GRANT_FORBIDDEN`.

Implementation may extend this set, but these failure classes must be explicit and auditable.

---

## Day-1 Execution Envelope

### Supported by default in Frank V4

- full Frank V3 live execution envelope,
- phone-resident improvement workspace,
- prompt-pack improvement,
- single-skill improvement,
- routing-manifest improvement,
- candidate evaluation with train and holdout suites,
- source-patch proposal artifacts,
- candidate-pack staging,
- Class 1 autonomous hot update with rollback,
- Class 2 autonomous hot update only for sandboxed local helper surfaces,
- bounded canary when required by policy,
- explicit approval path for higher-risk hot updates,
- rollback to last-known-good.

### Supported only after explicit onboarding or flag enablement

- topology improvement,
- broader donor-surface optimization,
- Class 3 runtime-extension hot update,
- canaries with real external side effects,
- additional model backends,
- broader skill-manifest rewrites,
- process-restart hot swap during non-idle live runtime,
- package installation from network sources.

### Explicitly not default-enabled in Frank V4

- evaluator self-mutation within a run,
- autonomous runtime-source deployment,
- autonomous mutation of authority / approval / treasury / autonomy policy,
- silent high-risk auto-promotion,
- multi-phone parallel deployment,
- voice-native assistant behavior,
- broad surveillance-like sensing,
- package-declared authority grants.

---

## Acceptance Criteria

Every criterion below must be reviewable or testable.

### Runtime-control criteria

1. **Execution plane is enforced**  
   A job without `execution_plane` is rejected.

2. **Execution host is enforced**  
   A job without `execution_host` is rejected.

3. **Improvement families require improvement workspace**  
   Any improvement-family job targeting `live_runtime` is rejected.

4. **Phone-resident improvement is valid**  
   An improvement-family job with `execution_host=phone` and `execution_plane=improvement_workspace` is valid when other gates pass.

5. **The live runtime loads committed active packs only**  
   The runtime cannot claim a new active pack without a durable promotion or hot-update record.

6. **One active pack at a time is enforced**  
   The phone has exactly one active runtime pack for live work at a time.

7. **Baseline is recorded before evaluated mutation**  
   An improvement run cannot mark candidate mutation eligible until required baseline record exists.

8. **Evaluator immutability is enforced**  
   An improvement run that touches evaluator, rubric, train corpus, or holdout corpus is rejected.

9. **Holdout is required when policy requires it**  
   A candidate without required holdout evidence cannot be promoted or hot-updated.

10. **Smoke check is required for hot update**  
    A Class 1+ hot update cannot commit without smoke-check evidence.

11. **Canary is enforced when policy requires it**  
    Promotion or hot update is rejected when policy requires canary and no canary evidence exists.

12. **Hot update requires rollback target**  
    No candidate is hot-updated without previous active or last-known-good target.

13. **Rollback is deterministic**  
    A rollback restores the previous active or last-known-good pack and records the event explicitly.

14. **Replay does not duplicate hot update**  
    Replaying the same hot-update record does not create a second activation event.

15. **Workspace crash does not corrupt live runtime**  
    Failure in the improvement workspace does not modify the current active pack pointer.

16. **Hot update respects deploy lock**  
    Hot update is blocked when policy or runtime state forbids changing packs during active unsafe live work.

17. **Append-only ledger is enforced**  
    Historical improvement and hot-update outcomes are not silently overwritten.

18. **Repeated crash rolls back**  
    Repeated crashes after a hot update trigger rollback or quarantine according to policy.

### Behavior-contract criteria

19. **Train-only wins are insufficient**  
    A candidate that wins only on train is not promoted unless policy explicitly permits the surface class and target.

20. **Decision-complete outcome is recorded**  
    Every candidate attempt ends in explicit `keep`, `discard`, `blocked`, `crash`, `hot_updated`, `promoted`, or `rolled_back` status.

21. **Mutable-surface discipline is enforced**  
    A candidate touching undeclared or forbidden surfaces is non-compliant.

22. **Topology changes require enablement**  
    Add/remove/split/merge skill operations are rejected unless topology mode is explicitly enabled.

23. **Source patch proposals remain artifacts**  
    A `propose_source_patch` run may generate a patch artifact, but may not autonomously apply or deploy it.

24. **Policy surfaces stay frozen**  
    Improvement and hot-update runs do not autonomously rewrite authority, approval, autonomy, treasury, or campaign rules.

25. **Canary scope is truthful**  
    A canary may not silently exceed its declared job/surface scope.

26. **Rollback remains possible after hot update**  
    Every hot-updated pack has a durable predecessor.

27. **Live external guardrails survive improvement**  
    A hot-updated pack may not bypass identity boundaries, eligibility checks, campaign requirements, treasury state, or capability-onboarding rules.

28. **Self-improvement can run on-phone**  
    “Improve yourself” may launch a bounded phone-resident improvement-workspace run.

29. **Ad hoc active mutation is rejected**  
    Direct file-copy or direct active-pack editing is rejected as active runtime truth.

30. **Pi-like reload works through the gate**  
    A prompt/skill reload refreshes committed reloadable surfaces without full manual redeploy and records reload generation.

31. **Extension hot reload is permission-aware**  
    A new extension declaring external side-effect tools requires approval or is rejected.

32. **Package content does not grant authority**  
    A package that includes a tool, skill, or prompt cannot create provider, spending, owner-control, or campaign authority by declaration.

### Continuous-autonomy criteria

33. **Standing directive can schedule work**  
    Frank wakes from a standing directive and creates or resumes one eligible bounded mission.

34. **No eligible work results in heartbeat**  
    If no eligible work exists, Frank records `E_NO_ELIGIBLE_AUTONOMOUS_ACTION` or idle heartbeat rather than inventing authority.

35. **Autonomy budget is enforced**  
    Budget exhaustion pauses further autonomous work in the exhausted category.

36. **Repeated failure pauses**  
    Repeated failure threshold creates a durable pause/blocker instead of infinite retry.

37. **Owner pause stops autonomous hot updates**  
    Owner pause prevents new autonomous hot-update commits while preserving inspection and rollback.

---

## Review and Acceptance Scenarios

- A prompt-pack improvement run records baseline, mutates one pack, wins on train but fails holdout, and is discarded.
- A single-skill improvement run improves holdout, passes smoke check, hot-updates on the phone, and records the previous pack as rollback target.
- A candidate attempts to edit the evaluator during the run and is rejected with `E_EVAL_IMMUTABLE`.
- A candidate attempts to change the autonomy predicate or treasury rules and is rejected with `E_POLICY_MUTATION_FORBIDDEN`.
- A topology-improvement request arrives while topology mode is disabled and is rejected with `E_TOPOLOGY_CHANGE_DISABLED`.
- A hot update is attempted during an active unsafe live job and is blocked with `E_ACTIVE_JOB_DEPLOY_LOCK`.
- A Class 3 extension declares a new network posting tool and is blocked pending approval with `E_HOT_UPDATE_APPROVAL_REQUIRED`.
- A canary regresses after hot-update criteria and triggers rollback to the previous active pack.
- A workspace crash occurs during mutation, and the live runtime continues using the previous active pack.
- A request to “rewrite your own runtime source and deploy it now” yields a patch artifact or rejection, but not autonomous core-source deployment.
- A request to “improve how you pursue bootstrap_revenue” launches a bounded improvement mission that may optimize packs/skills for that family without changing treasury rules.
- A `/reload skills` command refreshes committed skill metadata and increments reload generation without treating uncommitted candidate files as active.
- A package import from a Pi-style package becomes candidate content and is not active until the hot-update gate commits it.
- A repeated crash after a hot update rolls back to last-known-good and quarantines the candidate.

---

## Non-Blocking Implementation Choices

The following are implementation choices rather than frozen semantic requirements:

- exact phone filesystem layout,
- exact storage backend for pack registry and ledgers,
- exact scoring formula,
- exact model/provider used for mutation,
- exact model/provider used for evaluation,
- exact file naming for pack bundles,
- exact package format for rollback bundles,
- exact hot-reload implementation detail,
- whether extension reload is process-local or sidecar-mediated,
- whether desktop is used during development for heavy tests.

A reference phone-resident implementation shape consistent with this spec is:

```text
frank/
  live/
    active_pack_pointer.json
    last_known_good_pointer.json
    reload_generation
    runtime_state/
    logs/
  packs/
    active/
    last_known_good/
    candidates/
    canary/
    quarantined/
  workspace/
    promptpacks/
    skillpacks/
    extensionpacks/
    manifests/
    evals/
    runs/
    scratch/
  hot_updates/
    envelopes/
    staged/
    committed/
    rollback/
    ledger.jsonl
  autonomy/
    standing_directives/
    budgets/
    wake_cycles.jsonl
```

This layout is guidance only. The frozen requirement is the behavior contract: phone-resident improvement workspace, immutable evaluator per run, bounded mutable target, transactional hot update, append-only results, promotion/rollback gates, and no policy-surface self-mutation.

---

## Future Scope

The following remain outside Frank V4:

- autonomous deployment of core runtime-source control-plane changes,
- evaluator evolution inside a single run,
- silent high-risk auto-promotion,
- broader self-modification of policy surfaces,
- multi-phone staged deployment as a requirement,
- multi-agent improvement orchestration as a requirement,
- richer embodied interfaces,
- broad sensor exposure,
- regulated legal/custodial wrappers,
- autonomous provider onboarding beyond explicitly accepted provider lanes.

These are valid future directions, but they are not part of the frozen Frank V4 contract.

---

## Lowering Diff / Revision Notes

### What changed from the earlier V4 spec

This revision removes the permanent desktop-only lab invariant.

Earlier framing:

```text
adaptive improvement lab = desktop-only
phone = deployed body only
self-improvement stays off-phone
```

Revised framing:

```text
improvement workspace = isolated plane
phone = final host for live runtime + improvement workspace + hot-update gate
desktop = optional development/recovery host
self-improvement stays out of ad hoc active-pack mutation, not off-phone
```

### What stayed

This revision keeps:

- V3 mission-control invariants,
- baseline-first evaluation,
- immutable eval surfaces per run,
- train/holdout separation,
- explicit keep/discard/block/crash decisions,
- promotion and rollback records,
- append-only improvement ledger,
- forbidden policy-surface mutation,
- forbidden uncontrolled runtime-source self-deploy.

### What was added

This revision adds:

- phone-only final deployment target,
- `ExecutionHost`,
- `improvement_workspace`,
- `hot_update_gate`,
- `HotUpdateEnvelope`,
- `HotUpdatePolicy`,
- `HotUpdateRecord`,
- `HotUpdateLedgerEntry`,
- reload modes,
- surface classes,
- runtime extension packs,
- autonomous hot-promotion policy,
- continuous autonomy records,
- Pi-like reload command semantics.

### Why this is more like Pi

Pi’s useful lesson is not that every file edit is safe. The useful lesson is that an agent becomes much more adaptive when prompts, skills, tools, extensions, and packages are normal reloadable surfaces.

Frank V4 adopts that lesson by making those surfaces first-class pack content and adding a hot-update path.

Frank V4 differs from Pi by preserving mission-control authority, durable promotion records, rollback records, budgets, eligibility checks, and replay semantics.

---

## Assumptions and Defaults

- This file is a frozen engineering spec, not a PRD or prompt-pack bundle.
- The file lives at `docs/FRANK_V4_SPEC.md` when installed into the repo.
- Frank V4 preserves the full Frank V3 live runtime contract unless this file tightens or extends it.
- Final Frank V4 deployment is phone-only.
- The phone may host both live runtime and improvement workspace.
- The desktop may still be used for implementation, heavy tests, backups, and recovery.
- Pi-agent architecture is admitted as donor inspiration for hot-reloadable skills, prompts, extensions, packages, and sessions.
- A Pi-style package imported into Frank is candidate pack content until committed through the hot-update gate.
- The current deployed runtime may lag this frozen target.
