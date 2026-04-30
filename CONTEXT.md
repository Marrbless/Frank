# Context

This glossary names the stable domain concepts used by the repo and maintenance plans.

## Product Identity

- **Picobot**: the lightweight Go agent runtime and public project frame.
- **Frank**: the governed private operator identity built on Picobot runtime surfaces.
- **Operator**: the human owner/controller responsible for approving unsafe actions and runtime changes.

## Runtime Surfaces

- **Live runtime plane**: the currently acting Frank process, active runtime pack, channel adapters, tools, and mission-control state.
- **Improvement workspace**: the isolated workspace where candidate runtime changes are created and evaluated.
- **Hot-update gate**: the transactional path that stages, validates, applies, records, or rejects reloadable runtime updates.
- **Runtime pack**: versioned reloadable runtime content, such as prompts, skills, manifests, routing metadata, and permitted extension components.
- **Last-known-good pack**: the runtime pack certified as the rollback target after a successful promotion.

## Mission Control

- **Mission store**: durable local state under the configured mission store root.
- **Job**: a governed unit of mission work with steps, runtime state, control state, audit records, and artifacts.
- **Step**: the active execution unit inside a job.
- **Operator channel**: the Telegram/Discord/Slack/WhatsApp or CLI control surface where the owner inspects status and sends commands.
- **Owner approval**: an explicit operator authorization record required before selected governed actions.
- **Canary**: evidence-gathering and satisfaction path used before selected hot-update promotions.

## Safety Terms

- **Fail closed**: reject or pause when authority, identity, config, or evidence is missing.
- **Allowlist**: the configured set of owner-approved user, sender, or channel IDs that may reach a channel adapter.
- **Open mode**: an explicitly acknowledged channel mode where an empty allowlist accepts any sender; use only for deliberate public or testing deployments.
- **Transactional update**: an update that has candidate build proof, smoke proof, switch proof, health proof, and rollback proof.
- **Rollback candidate**: the preserved prior binary, runtime pack, or pointer target that can restore the last known good state after a failed update.
- **Mission assertion**: a deterministic `picobot mission assert` check that verifies the current mission status matches expected active, step, tool, or approval state before an operator treats a run as healthy.
- **Historical evidence**: retained maintenance docs that preserve what was true during a prior slice but do not override live repo truth.
