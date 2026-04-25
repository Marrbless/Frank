# V4-084 Candidate Decision Hot-Update Gate Control Entry Before State

## Gap From V4-083

V4-083 selected the smallest safe control entry for invoking the V4-082 helper, but no operator command or TaskState wrapper existed yet.

The available missioncontrol helper was:

```go
CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error)
```

It could create a prepared hot-update gate from a durable candidate promotion decision, but only direct Go callers could use it.

## Required Entry

V4-084 should expose exactly one direct command:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
```

The command should use the existing direct operator command path and a TaskState wrapper. It should not add new missioncontrol storage behavior.

## Required Wrapper Behavior

The TaskState wrapper should:

- validate active or persisted job context using existing TaskState patterns
- reject a supplied `job_id` that does not match the active or persisted job
- resolve and validate the mission store root through the existing TaskState path
- use `operator` as `created_by`
- derive `created_at` from the existing TaskState timestamp path
- call `missioncontrol.CreateHotUpdateGateFromCandidatePromotionDecision`
- emit audit action `hot_update_gate_from_decision`
- return the missioncontrol `changed` flag unchanged

## Timestamp Replay Requirement

The deterministic hot-update ID is:

```text
hot-update-<promotion_decision_id>
```

If that gate already exists, the wrapper must load it and reuse its `prepared_at` timestamp before rerunning the V4-082 helper. This keeps idempotent command replay byte-stable instead of turning a valid replay into a divergent duplicate.

## Response Semantics

On creation:

```text
Created hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

On idempotent selection:

```text
Selected hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

On failure, the direct command should return an empty response plus the error.

## Invariants To Preserve

V4-084 must not add deploy-lock enforcement, unsafe-live-job blocking, canary execution, owner approval requests, gate phase advancement, pointer switch execution, reload/apply behavior, active pointer mutation, last-known-good pointer mutation, `reload_generation` mutation, hot-update outcome creation, promotion creation, rollback creation, rollback-apply creation, LKG mutation, or V4-085 work.
