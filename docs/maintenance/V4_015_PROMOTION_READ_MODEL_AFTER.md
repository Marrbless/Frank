## V4-015 Promotion Read-Model After

### git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |   1 +
 internal/agent/tools/taskstate_status_test.go | 234 +++++++++++++++++++++++++-
 internal/missioncontrol/status.go             | 139 +++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 372 insertions(+), 3 deletions(-)
```

Note: `git diff --stat` does not include the new untracked files created in this slice.

### git diff --numstat

```text
1	0	internal/agent/tools/taskstate_readout.go
231	3	internal/agent/tools/taskstate_status_test.go
139	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

Note: `git diff --numstat` does not include the new untracked files created in this slice.

### Files Changed

- `docs/maintenance/V4_015_PROMOTION_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_015_PROMOTION_READ_MODEL_AFTER.md`
- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/missioncontrol/status_promotion_identity_test.go`
- `internal/agent/tools/taskstate_status_test.go`

### Exact Read-Model Helpers Added

- `OperatorStatusSummary.PromotionIdentity`
- `OperatorPromotionIdentityStatus`
- `OperatorPromotionStatus`
- `WithPromotionIdentity`
- `LoadOperatorPromotionIdentityStatus`
- `loadOperatorPromotionStatuses`
- `loadOperatorPromotionStatus`
- `operatorPromotionStatusFromRecord`

### Exact Tests Added

- `TestLoadOperatorPromotionIdentityStatusConfigured`
- `TestLoadOperatorPromotionIdentityStatusNotConfigured`
- `TestLoadOperatorPromotionIdentityStatusInvalidMissingLinkedRefs`
- `TestBuildCommittedMissionStatusSnapshotIncludesPromotionIdentity`
- `TestTaskStateOperatorStatusSurfacesPromotionIdentity`
- top-level taskstate schema-lock key expectations updated to include `promotion_identity`

### Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_promotion_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed

### Deferred Next V4 Candidates

- rollback record skeleton
- rollback read-model / inspect exposure
- promotion or rollback control-surface work that consumes the read-model without changing this slice’s read-only contract
