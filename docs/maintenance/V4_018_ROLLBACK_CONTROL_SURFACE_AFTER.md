Git Diff --stat

```text
 internal/agent/loop.go                    |  12 ++
 internal/agent/loop_processdirect_test.go | 237 ++++++++++++++++++++++++++++++
 internal/agent/tools/taskstate.go         | 108 ++++++++++++++
 3 files changed, 357 insertions(+)




```

Git Diff --numstat

```text
12	0	internal/agent/loop.go
237	0	internal/agent/loop_processdirect_test.go
108	0	internal/agent/tools/taskstate.go



```

Files Changed

- `docs/maintenance/V4_018_ROLLBACK_CONTROL_SURFACE_BEFORE.md`
- `docs/maintenance/V4_018_ROLLBACK_CONTROL_SURFACE_AFTER.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/tools/taskstate.go`

Exact Control-Surface Fields/Helpers Added

- `rollbackRecordCommandRE` in `internal/agent/loop.go`
- Direct operator command: `rollback_record <job_id> <promotion_id> <rollback_id>`
- `(*TaskState).RecordRollbackFromPromotion(jobID, promotionID, rollbackID string) error`
- Deterministic rollback proposal derivation from an existing committed promotion record into the existing rollback ledger
- Existing `STATUS <job_id>` surface reused unchanged for rollback identity inspection after record creation

Exact Tests Added

- `TestProcessDirectRollbackRecordCommandCreatesProposalAndPreservesActiveRuntimePackPointer`
- `TestProcessDirectRollbackRecordCommandFailsClosedWhenPromotionIsMissing`
- `writeLoopRollbackPromotionFixtures` shared fixture helper

Validation Commands And Results

- `gofmt -w internal/agent/loop.go internal/agent/tools/taskstate.go internal/agent/loop_processdirect_test.go` -> pass
- `go test -count=1 -run 'TestProcessDirectRollbackRecordCommand' ./internal/agent` -> pass
- `go test -count=1 -run 'TestProcessDirectPauseResumeAbortCommandsControlActiveJob|TestProcessDirectStatusCommandReturnsDeterministicSummary' ./internal/agent` -> pass
- `git diff --check` -> pass
- `go test -count=1 ./internal/missioncontrol` -> pass
- `go test -count=1 ./...` -> pass

Explicit No-Apply Statement

- No rollback apply behavior was implemented.
- No runtime pack activation mutation was implemented.
- The control surface only derives and stores rollback records through the existing ledger contract and exposes them via the existing status read-model.

Deferred Next V4 Candidates

- Rollback apply workflow skeleton that consumes rollback records without widening this record-creation contract
- Rollback target-selection or approval-specific control inputs if a later slice needs more than promotion-derived default proposals
- Promotion/rollback execution orchestration that reuses the existing promotion, outcome, rollback, and status contracts
