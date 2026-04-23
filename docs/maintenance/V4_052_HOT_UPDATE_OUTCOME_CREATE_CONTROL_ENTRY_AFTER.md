# V4-052 Hot-Update Outcome Create Control Entry After

## Command Shape

V4-052 adds direct operator command support for:

- `HOT_UPDATE_OUTCOME_CREATE <job_id> <hot_update_id>`

The command has no manual outcome ID, outcome kind, reason, candidate/run/result refs, pack refs, or timestamp arguments. Those fields remain derived from the committed terminal gate and the TaskState wrapper timestamp.

## TaskState Wrapper Behavior

The direct command calls:

- `(*TaskState).CreateHotUpdateOutcomeFromTerminalGate(jobID, hotUpdateID string) (bool, error)`

The wrapper:

- validates the active or persisted runtime job context using the same pattern as the existing hot-update gate wrappers
- resolves the mission store root from TaskState
- derives `now` through `taskStateTransitionTimestamp(taskStateNowUTC())`
- calls `missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, "operator", now)`
- emits runtime control audit action `hot_update_outcome_create`
- returns the missioncontrol changed flag unchanged

For exact command replay, the wrapper reuses the existing deterministic outcome record's `created_at` when needed so the V4-050 helper can return its `changed=false` idempotent path without changing missioncontrol storage semantics.

## Direct Response Semantics

When the helper creates an outcome:

- `Created hot-update outcome job=<job_id> hot_update=<hot_update_id>.`

When the helper identifies exact replay:

- `Selected hot-update outcome job=<job_id> hot_update=<hot_update_id>.`

On failure, the direct command returns an empty response and the underlying fail-closed error, consistent with existing direct command behavior.

## Failure Behavior

The control entry preserves helper failures for:

- missing gate
- non-terminal gate
- `reload_apply_failed` with empty `failure_reason`
- divergent existing deterministic outcome
- existing outcome for the same `hot_update_id` with a different `outcome_id`
- wrong `job_id` through existing TaskState validation

The command does not manufacture success or infer terminal state from pointer state, missing records, or status output.

## Status / Read Model

After creation, existing status output surfaces the outcome through `hot_update_outcome_identity`.

`STATUS <job_id>` can show:

- deterministic `outcome_id`
- `hot_update_id`
- `candidate_pack_id`
- `outcome_kind`
- copied or fixed reason
- `outcome_at`
- `created_at`
- `created_by`

No separate status command was added.

## Invariants Preserved

V4-052 does not create a `PromotionRecord`, mutate the active runtime-pack pointer, increment `reload_generation`, mutate `last_known_good_pointer.json`, mutate hot-update gates, create a new hot-update gate, add manual outcome fields, alter missioncontrol outcome mapping, broaden policy/authorization, or start V4-053.
