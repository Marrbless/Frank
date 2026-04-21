# V4 BLOCK 001 Store Writer Lease After

## Root Cause

The failing path was deterministic, not flaky, on this branch:

- `TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity`
- `PersistProjectedRuntimeState(..., now)`

Exact failure mode:

- `PersistProjectedRuntimeState` acquires the writer lock using the caller-supplied `now`
- `CommitStoreBatch` then validates the held lock against `time.Now().UTC()`
- snapshot tests using a historical fixed calendar timestamp can therefore create a lock whose `lease_expires_at` is already in the past relative to real wall clock time
- that makes the same persist call fail with:
  - `mission store writer lock lease has expired`

This branch blocker was therefore caused by test scaffolding using stale lock time inputs on a production path that validates against live clock time.

## Exact Files Changed

- `internal/missioncontrol/status_runtime_pack_identity_test.go`
- `internal/missioncontrol/status_improvement_candidate_identity_test.go`
- `internal/missioncontrol/status_improvement_run_identity_test.go`
- `internal/missioncontrol/store_project_test.go`
- `docs/maintenance/V4_BLOCK_001_STORE_WRITER_LEASE_BEFORE.md`
- `docs/maintenance/V4_BLOCK_001_STORE_WRITER_LEASE_AFTER.md`

## Exact Fix

Added one shared helper in an allowed `*store*_test.go` file:

- `testLeaseSafeNow()`

Behavior:

- returns `time.Now().UTC().Truncate(time.Second)`
- gives projected-runtime persist tests a lease timestamp anchored to current wall clock

Applied that helper to the three status snapshot tests that persist projected runtime state:

- `TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity`
- `TestBuildCommittedMissionStatusSnapshotIncludesImprovementCandidateIdentity`
- `TestBuildCommittedMissionStatusSnapshotIncludesImprovementRunIdentity`

Coverage was preserved:

- no assertion was deleted
- runtime-pack-identity snapshot coverage was tightened by asserting derived `updated_at` and `verified_at` timestamps from the lease-safe `now`

## Why This Fix

This is the smallest safe stabilization fix inside the allowed edit surface.

It does not:

- change provider/channel/agent behavior
- change store mutation semantics
- change writer-lock production logic
- weaken assertions

It only removes the test fragility caused by mixing fixed historical timestamps with live-clock lease validation.

## git diff --stat

```text
 .../status_improvement_candidate_identity_test.go              |  2 +-
 .../missioncontrol/status_improvement_run_identity_test.go     |  2 +-
 internal/missioncontrol/status_runtime_pack_identity_test.go   | 10 +++++++++-
 internal/missioncontrol/store_project_test.go                  |  6 +++++-
 4 files changed, 16 insertions(+), 4 deletions(-)
```

## git diff --numstat

```text
1	1	internal/missioncontrol/status_improvement_candidate_identity_test.go
1	1	internal/missioncontrol/status_improvement_run_identity_test.go
9	1	internal/missioncontrol/status_runtime_pack_identity_test.go
5	1	internal/missioncontrol/store_project_test.go
```

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/status_runtime_pack_identity_test.go internal/missioncontrol/status_improvement_candidate_identity_test.go internal/missioncontrol/status_improvement_run_identity_test.go internal/missioncontrol/store_project_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 -run TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity -v ./internal/missioncontrol`
  - passed 5 consecutive times
- `go test -count=1 ./internal/missioncontrol`
  - passed 3 consecutive times
- `go test -count=1 ./...`
  - passed 3 consecutive times

## git status --short --branch

```text
## frank-v4-009-candidate-result-scorecard-skeleton
 M internal/missioncontrol/status_improvement_candidate_identity_test.go
 M internal/missioncontrol/status_improvement_run_identity_test.go
 M internal/missioncontrol/status_runtime_pack_identity_test.go
 M internal/missioncontrol/store_project_test.go
?? docs/maintenance/V4_BLOCK_001_STORE_WRITER_LEASE_BEFORE.md
?? docs/maintenance/V4_BLOCK_001_STORE_WRITER_LEASE_AFTER.md
```
