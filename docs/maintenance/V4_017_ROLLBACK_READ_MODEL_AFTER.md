## V4-017 Rollback Read-Model After

### git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go | 257 +++++++++++++++++++++++++-
 internal/missioncontrol/status.go             | 133 +++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 389 insertions(+), 3 deletions(-)
```

Note: `git diff --stat` does not include the new untracked files created in this slice.

### git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
254	3	internal/agent/tools/taskstate_status_test.go
133	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

Note: `git diff --numstat` does not include the new untracked files created in this slice.

### Files Changed

- `docs/maintenance/V4_017_ROLLBACK_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_017_ROLLBACK_READ_MODEL_AFTER.md`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/missioncontrol/status_rollback_identity_test.go`
- `internal/agent/tools/taskstate_status_test.go`

### Exact Read-Model Helpers Added

- `OperatorStatusSummary.RollbackIdentity`
- `OperatorRollbackIdentityStatus`
- `OperatorRollbackStatus`
- `WithRollbackIdentity`
- `LoadOperatorRollbackIdentityStatus`
- `loadOperatorRollbackStatuses`
- `loadOperatorRollbackStatus`
- `operatorRollbackStatusFromRecord`

### Exact Tests Added

- `TestLoadOperatorRollbackIdentityStatusConfigured`
- `TestLoadOperatorRollbackIdentityStatusNotConfigured`
- `TestLoadOperatorRollbackIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesRollbackIdentity`
- `TestTaskStateOperatorStatusSurfacesRollbackIdentity`
- top-level taskstate schema-lock key expectations updated to include `rollback_identity`

### Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_rollback_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed

### Deferred Next V4 Candidates

- rollback control-surface work that consumes rollback records without implementing apply behavior in this slice
- broader inspect exposure only if a later slice needs more than rollback identity and linkage
- follow-on orchestration that reuses the immutable rollback storage contract and this read-only surface
