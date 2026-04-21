# V4-004 Pack Identity Read-Model Exposure After

- Branch: `frank-v4-001-pack-registry-foundation`
- HEAD: `c9623c2155472337975a4392510175c9717d4df0`

## git diff --stat

```text
 internal/agent/tools/taskstate_readout.go     |  12 +-
 internal/agent/tools/taskstate_status_test.go |  99 +++++++++++++++-
 internal/missioncontrol/status.go             | 162 ++++++++++++++++++++++++++
 internal/missioncontrol/store_project.go      |   1 +
 4 files changed, 265 insertions(+), 9 deletions(-)
```

## git diff --numstat

```text
6	6	internal/agent/tools/taskstate_readout.go
96	3	internal/agent/tools/taskstate_status_test.go
162	0	internal/missioncontrol/status.go
1	0	internal/missioncontrol/store_project.go
```

`git diff` does not include new untracked files. This slice also added:

- `internal/missioncontrol/status_runtime_pack_identity_test.go`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_AFTER.md`

## Files Changed

- `internal/missioncontrol/status.go`
- `internal/missioncontrol/store_project.go`
- `internal/missioncontrol/status_runtime_pack_identity_test.go`
- `internal/agent/tools/taskstate_readout.go`
- `internal/agent/tools/taskstate_status_test.go`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_BEFORE.md`
- `docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_AFTER.md`

## Exact Read-Model Fields Added

Added `runtime_pack_identity` to `missioncontrol.OperatorStatusSummary`.

Added nested read-only status fields:

- `runtime_pack_identity.active.state`
- `runtime_pack_identity.active.active_pack_id`
- `runtime_pack_identity.active.previous_active_pack_id`
- `runtime_pack_identity.active.last_known_good_pack_id`
- `runtime_pack_identity.active.updated_at`
- `runtime_pack_identity.active.error`
- `runtime_pack_identity.last_known_good.state`
- `runtime_pack_identity.last_known_good.pack_id`
- `runtime_pack_identity.last_known_good.basis`
- `runtime_pack_identity.last_known_good.verified_at`
- `runtime_pack_identity.last_known_good.error`

State values used by this slice:

- `configured`
- `not_configured`
- `invalid`

## Exact Tests Added

Added in `internal/missioncontrol/status_runtime_pack_identity_test.go`:

- `TestLoadOperatorRuntimePackIdentityStatusConfigured`
- `TestLoadOperatorRuntimePackIdentityStatusNotConfigured`
- `TestLoadOperatorRuntimePackIdentityStatusInvalidMissingReferencedPack`
- `TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity`

Changed in `internal/agent/tools/taskstate_status_test.go`:

- `TestTaskStateOperatorStatusSurfacesRuntimePackIdentity`
- updated existing JSON-envelope assertions in:
  - `TestTaskStateOperatorStatusActiveExecutionContextSurfacesFrankZohoMailboxBootstrapPreflight`
  - `TestTaskStateOperatorStatusSurfacesCampaignZohoEmailSendGateOnPersistedPath`
  - `TestTaskStateOperatorStatusActiveAndPersistedPathsPreserveAdapterBoundaryContract`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/status.go internal/missioncontrol/store_project.go internal/missioncontrol/status_runtime_pack_identity_test.go internal/agent/tools/taskstate_readout.go internal/agent/tools/taskstate_status_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch --untracked-files=all`
  - result:

```text
## frank-v4-001-pack-registry-foundation
 M internal/agent/tools/taskstate_readout.go
 M internal/agent/tools/taskstate_status_test.go
 M internal/missioncontrol/status.go
 M internal/missioncontrol/store_project.go
?? docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_AFTER.md
?? docs/maintenance/V4_004_PACK_IDENTITY_READ_MODEL_BEFORE.md
?? internal/missioncontrol/status_runtime_pack_identity_test.go
```

## Deferred Next V4 Candidates

- V4 pack promotion record / promotion intent skeleton, still without apply behavior
- hot-update gate storage and operator-visible gate status
- explicit rollback record skeleton tied to last-known-good basis metadata
- inspect/status surfacing of pack compatibility-contract refs and channel metadata if later operator workflows need them
