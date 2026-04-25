# V4-092 Hot-Update Execution Ready Control Entry Before

## Before-state gap

V4-090 added `HotUpdateExecutionSafetyEvidenceRecord` and taught `AssessHotUpdateExecutionReadiness(...)` to consume current matching evidence for active `live_runtime` work. V4-091 selected the smallest producer shape, but no operator control entry existed to create short-lived `deploy_unlocked` plus `quiesce ready` evidence.

The V4-088 guard wiring therefore still failed closed for new pointer-switch and reload/apply attempts whenever an active occupied `live_runtime` job existed, even when an operator had externally established that the job was safe for hot-update execution.

## Required control entry

V4-092 adds the narrow direct command:

`HOT_UPDATE_EXECUTION_READY <job_id> <hot_update_id> <ttl_seconds> [reason...]`

The command is intentionally not a broad arbitrary safety-state writer. It only records a short-lived positive readiness assertion:

- `deploy_lock_state=deploy_unlocked`
- `quiesce_state=ready`
- deterministic evidence ID `hot-update-execution-safety-<hot_update_id>-<job_id>`
- `created_by=operator`
- `created_at` from the existing TaskState timestamp path
- `expires_at=created_at+ttl_seconds`

TTL must be an integer from 1 through 300 seconds. Missing reason defaults to `operator asserted hot-update execution readiness`.

## Invariants to preserve

The wrapper must validate the active or persisted job context, require active global occupancy for the same job, require committed `live_runtime` job/runtime-control evidence, bind active step/attempt/writer epoch/activation sequence from the active job record, and require the hot-update gate to be `staged`, `reloading`, or `reload_apply_recovery_needed`.

Exact replay must select the existing evidence without byte changes. Expired evidence may be replaced only for the same deterministic hot-update/job identity and current active-job binding. Non-expired divergent evidence and stale binding evidence must fail closed.

V4-092 must not change pointer-switch or reload/apply behavior, infer quiesce automatically, mutate runtime-pack pointers, mutate last-known-good, mutate `reload_generation`, create hot-update gates, outcomes, promotions, rollbacks, rollback-apply records, canary records, approval records, or LKG records, or implement V4-093.
