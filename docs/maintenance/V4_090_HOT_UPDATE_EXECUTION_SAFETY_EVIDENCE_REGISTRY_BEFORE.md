# V4-090 Hot-Update Execution Safety Evidence Registry Before

## Before-State Gap

V4-088 wired `AssessHotUpdateExecutionReadiness(...)` into the TaskState pointer-switch and reload/apply wrappers, so new execution-sensitive hot-update transitions fail closed when readiness is not proven.

V4-089 assessed the next missing surface: the repo could identify active live-runtime occupancy through `ActiveJobRecord`, `JobRuntimeRecord`, and `RuntimeControlRecord`, but it had no durable evidence record proving that an occupied live-runtime job was deploy-unlocked and quiesced for a specific hot update.

Before V4-090:

- there was no deploy-lock/quiesce evidence registry
- there was no deterministic evidence ID
- there was no current evidence lookup for `(hot_update_id, job_id)`
- the readiness guard could distinguish `not_configured` and explicit failed input, but TaskState had no durable evidence to pass
- active occupied `live_runtime` jobs remained fail-closed unless replay had already applied

## Required Shape

V4-090 is limited to missioncontrol storage/read-model behavior and readiness-guard consumption.

It must not add:

- direct commands
- TaskState write wrappers
- deploy-lock or quiesce producer logic
- pointer switch execution
- reload/apply execution
- active runtime-pack pointer mutation
- last-known-good pointer mutation
- `reload_generation` mutation
- hot-update gate creation
- hot-update outcomes
- promotions
- rollback records
- rollback-apply records
- canary records
- approval records
- LKG records
- V4-091 work
