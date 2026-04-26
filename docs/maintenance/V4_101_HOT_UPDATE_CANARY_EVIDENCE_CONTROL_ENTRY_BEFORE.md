# V4-101 Hot-Update Canary Evidence Control Entry Before State

## Gap From V4-100

V4-100 selected the smallest safe operator-control entry for durable canary evidence, but it was assessment-only. The live code had the V4-099 missioncontrol registry and read model, yet no direct command or TaskState wrapper to call:

```go
CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID, state, observedAt, createdBy, createdAt, reason)
```

Operators could create canary requirements through `HOT_UPDATE_CANARY_REQUIREMENT_CREATE`, but could not record passed, failed, blocked, or expired canary evidence through the existing direct-command path.

## Existing Authority

V4-099 made the committed `HotUpdateCanaryRequirementRecord` the source authority for evidence creation. The registry stores evidence under:

```text
runtime_packs/hot_update_canary_evidence/<canary_evidence_id>.json
```

The evidence ID is deterministic from the canary requirement ID and observed time:

```text
hot-update-canary-evidence-<canary_requirement_id>-<observed_at_utc_compact>
```

The registry validates the canary requirement, candidate result, improvement run, improvement candidate, frozen eval suite, promotion policy, baseline runtime pack, candidate runtime pack, and freshly derived promotion eligibility. Eligibility must remain `canary_required` or `canary_and_owner_approval_required`.

## Required Control Slice

The missing control surface is:

```text
HOT_UPDATE_CANARY_EVIDENCE_CREATE <job_id> <canary_requirement_id> <evidence_state> <observed_at> [reason...]
```

The command must parse `observed_at` as RFC3339 or RFC3339Nano, infer `passed` from `evidence_state`, default an omitted reason deterministically, and delegate to the V4-099 helper through TaskState.

## Invariants To Preserve

V4-101 must remain only a control wrapper around the registry. It must not execute canaries, create canary execution automation or proposals, request owner approval, create owner approval proposals, create candidate promotion decisions for canary-required states, create hot-update gates, create outcomes, promotions, rollbacks, rollback-apply records, mutate candidate results, mutate canary requirements, mutate promotion policies, mutate runtime packs, mutate the active runtime-pack pointer, mutate the last-known-good pointer, mutate `reload_generation`, change pointer-switch behavior, change reload/apply behavior, or implement V4-102.
