# V4 BLOCK 001 Store Writer Lease Before

## Branch

- `frank-v4-009-candidate-result-scorecard-skeleton`

## HEAD

- `a79f70496e14c987fb121f34aa562a49e49afe1d`

## Tags At HEAD

- `frank-v4-008-improvement-run-read-model`

## Ahead/Behind Upstream

- `406 0`

## git status --short --branch

```text
## frank-v4-009-candidate-result-scorecard-skeleton
```

## Baseline Reproduction

- `go test -count=1 -run TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity -v ./internal/missioncontrol || true`
  - failed
  - exact error:
    - `PersistProjectedRuntimeState() error = mission store writer lock lease has expired`
- `go test -count=1 ./internal/missioncontrol || true`
  - failed on the same test
- `go test -count=1 ./... || true`
  - failed
  - includes the same `internal/missioncontrol` writer-lock expiry
  - also includes sandbox/environment failures outside this slice (`go-build` cache read-only and `httptest` listen permission failures), so the authoritative repo-local blocker for this task is the deterministic `internal/missioncontrol` lease-expiry failure

## Exact Files Planned

- `internal/missioncontrol/status_runtime_pack_identity_test.go`
- `internal/missioncontrol/status_improvement_candidate_identity_test.go`
- `internal/missioncontrol/status_improvement_run_identity_test.go`
- `internal/missioncontrol/store_project_test.go`
- `docs/maintenance/V4_BLOCK_001_STORE_WRITER_LEASE_AFTER.md`

## Exact Fix Shape Planned

Planned fix is a bounded baseline-stabilization change in test scaffolding only:

- add one shared test helper in an allowed `*store*_test.go` file to generate a lease-safe current UTC timestamp
- switch the status snapshot tests that call `PersistProjectedRuntimeState` from fixed calendar dates to that lease-safe timestamp
- preserve the assertions by deriving expected RFC3339 strings from the chosen test time rather than weakening coverage

Current diagnosis:

- `PersistProjectedRuntimeState` acquires the writer lock using the caller-supplied `now`
- `CommitStoreBatch` validates the held lock using `time.Now().UTC()`
- tests that use historical fixed dates can therefore self-expire the lease during the same persist call once wall clock time moves past the fixed lease window

## Exact Non-Goals

- no V4-009 implementation
- no provider/channel/agent behavior changes
- no production writer-lock semantic change unless the test-only stabilization proves insufficient
- no assertion weakening or coverage deletion
- no dependency changes
- no cleanup outside this baseline fix
- no commit
