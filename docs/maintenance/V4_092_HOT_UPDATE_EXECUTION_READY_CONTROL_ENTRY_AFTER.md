# V4-092 Hot-Update Execution Ready Control Entry After

## Implemented command

V4-092 adds:

`HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]`

The direct command parses job ID, hot-update ID, integer TTL, and optional free-text reason. It rejects missing arguments, non-integer TTL, TTL <= 0, and TTL > 300. On success it returns:

- `Recorded hot-update execution readiness job=<job_id> hot_update=<hot_update_id> expires_at=<rfc3339>.`
- `Selected hot-update execution readiness job=<job_id> hot_update=<hot_update_id> expires_at=<rfc3339>.`

Failures return an empty response plus the deterministic error from the TaskState/missioncontrol path.

## TaskState wrapper behavior

`TaskState.RecordHotUpdateExecutionReady(...)` resolves the mission store root, validates active or persisted job context, rejects a mismatched `job_id`, loads the current `ActiveJobRecord`, and requires the active job to hold global occupancy for the same job.

The wrapper loads committed `JobRuntimeRecord` and `RuntimeControlRecord`, requires `live_runtime`, cross-checks active step, attempt, writer epoch, and activation sequence evidence against the active job, loads the referenced `HotUpdateGateRecord`, and allows only `staged`, `reloading`, or `reload_apply_recovery_needed` gate states.

It writes only `deploy_unlocked` plus `ready` evidence, derives `created_by=operator`, derives `created_at` from TaskState time, derives `expires_at` from the bounded TTL, and emits audit action `hot_update_execution_ready` on created, selected, and rejected paths.

## Missioncontrol helper behavior

`EnsureHotUpdateExecutionReadyEvidence(...)` constructs the deterministic evidence record:

`hot-update-execution-safety-<hot_update_id>-<job_id>`

It preserves exact replay, rejects non-expired divergent duplicates, rejects stale active-job binding, and allows replacement only after expiry for the same hot-update/job identity and current active-job binding.

## Readiness and invariants

After successful evidence creation, `AssessHotUpdateExecutionReadiness(...)` can allow pointer-switch readiness and reload/apply readiness for the matching active `live_runtime` job until the evidence expires.

V4-092 does not add a broad arbitrary safety writer, does not create deploy-lock or quiesce producers from side effects, does not change pointer-switch or reload/apply semantics, does not execute pointer switch or reload/apply, does not mutate active runtime-pack pointer, last-known-good pointer, or `reload_generation`, and does not create hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, canary records, approval records, or LKG records.
