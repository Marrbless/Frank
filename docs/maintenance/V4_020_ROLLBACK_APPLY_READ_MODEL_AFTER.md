# V4-020 Rollback Apply Read-Model After

## `git diff --stat`

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go |  42 +++++++++-
 internal/missioncontrol/status.go             | 114 ++++++++++++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 155 insertions(+), 3 deletions(-)
```

Note: tracked `git diff --stat` output does not include new untracked files created in this slice.

## `git diff --numstat`

```text
1	0	internal/agent/tools/taskstate_readout.go
39	3	internal/agent/tools/taskstate_status_test.go
114	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

Note: tracked `git diff --numstat` output does not include new untracked files created in this slice.

## Files Changed

- `docs/maintenance/V4_020_ROLLBACK_APPLY_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_020_ROLLBACK_APPLY_READ_MODEL_AFTER.md`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_rollback_apply_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`

## Exact Read-Model Fields Added

- `runtime_summary.rollback_apply_identity`
- `rollback_apply_identity.state`
- `rollback_apply_identity.applies[]`
- `rollback_apply_identity.applies[].rollback_apply_id`
- `rollback_apply_identity.applies[].rollback_id`
- `rollback_apply_identity.applies[].phase`
- `rollback_apply_identity.applies[].activation_state`
- `rollback_apply_identity.applies[].created_at`
- `rollback_apply_identity.applies[].created_by`
- `rollback_apply_identity.applies[].error`

## Exact Tests Added

- `TestLoadOperatorRollbackApplyIdentityStatusConfigured`
- `TestLoadOperatorRollbackApplyIdentityStatusNotConfigured`
- `TestLoadOperatorRollbackApplyIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesRollbackApplyIdentity`

## Existing Tests Expanded

- `TestTaskStateOperatorStatusSurfacesRollbackIdentity`
  - now asserts `rollback_apply_identity` exposure on the operator status surface
- taskstate status schema-lock key expectations updated to include `rollback_apply_identity`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_rollback_apply_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation and report writing:

```text
## frank-v4-020-rollback-apply-read-model
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_020_ROLLBACK_APPLY_READ_MODEL_AFTER.md
?? docs/maintenance/V4_020_ROLLBACK_APPLY_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_rollback_apply_identity_test.go
```

## Explicit No-Apply Statement

- No rollback apply behavior was implemented.
- No active runtime-pack pointer mutation was implemented.
- No promotion behavior changes, evaluator execution, scoring behavior, or autonomy changes were implemented.

## Deferred Next V4 Candidates

- rollback-apply control-surface entry that creates or selects durable rollback-apply records without executing them
- separate rollback-apply execution slice that mutates activation state or runtime-pack pointers
- any follow-on inspect exposure only if a later slice needs rollback-apply identity outside the existing status/runtime-summary surface
