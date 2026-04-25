# V4-084 Candidate Decision Hot-Update Gate Control Entry After State

## Command

V4-084 adds the direct operator command:

```text
HOT_UPDATE_GATE_FROM_DECISION <job_id> <promotion_decision_id>
```

The command uses the existing direct command path and stays in the established `HOT_UPDATE_GATE_*` namespace.

## TaskState Wrapper

The command calls a TaskState wrapper that:

- validates the active or persisted job context
- rejects a supplied `job_id` that does not match the active or persisted job
- resolves and validates the mission store root through the existing TaskState path
- derives `created_at` from the TaskState timestamp path
- uses `operator` as `created_by`
- invokes `missioncontrol.CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, "operator", createdAt)`
- emits audit action `hot_update_gate_from_decision`
- returns the missioncontrol `changed` flag unchanged

## Deterministic Identity And Replay

The deterministic hot-update ID remains:

```text
hot-update-<promotion_decision_id>
```

When that deterministic gate already exists, the wrapper loads it and reuses the existing gate `prepared_at` timestamp before rerunning the missioncontrol helper. This preserves byte-stable idempotent command replay and avoids converting a valid replay into a divergent duplicate.

The wrapper does not retry with arbitrary timestamps and does not hide fail-closed missioncontrol errors.

## Response Semantics

On `changed=true`:

```text
Created hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

On `changed=false`:

```text
Selected hot-update gate from decision job=<job_id> promotion_decision=<promotion_decision_id> hot_update=<hot_update_id>.
```

On failure, the command returns an empty response plus the returned error, consistent with the existing direct command behavior.

## Failure Behavior

The command fails closed for malformed arguments, wrong job ID, missing promotion decision, non-selected or non-eligible decision records, stale derived eligibility, missing linked source records, mismatched decision/result/run/candidate/eval-suite/policy authority, missing or stale active runtime-pack pointer, missing candidate rollback target, missing rollback target runtime pack, divergent duplicate gates, and an existing deterministic gate with a different candidate pack.

## Audit Behavior

The wrapper emits `hot_update_gate_from_decision` on created and selected paths. Rejections also emit the same audit action with the validation or missioncontrol error.

## Invariants Preserved

V4-084 does not add new missioncontrol storage behavior, mutate the active runtime-pack pointer, mutate the last-known-good pointer, mutate `reload_generation`, create `HotUpdateOutcomeRecord`, create `PromotionRecord`, create rollback records, create rollback-apply records, create or mutate LKG records, advance hot-update phases, execute pointer switches, run reload/apply behavior, execute canaries, request owner approval, implement deploy-lock behavior, implement unsafe-live-job blocking, or start V4-085 work.
