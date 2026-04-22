# V4-023 Rollback-Apply Phase Control Entry After

## `git diff --stat`

```text
 internal/agent/loop.go                    |  16 +++
 internal/agent/loop_processdirect_test.go | 163 ++++++++++++++++++++++++++++++
 internal/agent/tools/taskstate.go         |  89 ++++++++++++++++
 3 files changed, 268 insertions(+)
```

Note: tracked `git diff --stat` output does not include new untracked files created in this slice.

## `git diff --numstat`

```text
16	0	internal/agent/loop.go
163	0	internal/agent/loop_processdirect_test.go
89	0	internal/agent/tools/taskstate.go
```

Note: tracked `git diff --numstat` output does not include new untracked files created in this slice.

## Files Changed

- `docs/maintenance/V4_023_ROLLBACK_APPLY_PHASE_CONTROL_ENTRY_BEFORE.md`
- `docs/maintenance/V4_023_ROLLBACK_APPLY_PHASE_CONTROL_ENTRY_AFTER.md`
- `internal/agent/loop.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop_processdirect_test.go`

## Exact Control-Entry Fields / Helpers Added

- direct operator command:
  - `ROLLBACK_APPLY_PHASE <job_id> <apply_id> <phase>`
- `rollbackApplyPhaseCommandRE` in `internal/agent/loop.go`
- `(*TaskState).AdvanceRollbackApplyPhase(jobID, applyID, phase string) (bool, error)`
- control-entry behavior:
  - advances phase on an existing committed rollback-apply record by `apply_id`
  - delegates to existing durable helper `missioncontrol.AdvanceRollbackApplyPhase`
  - returns deterministic acknowledgements:
    - `Advanced rollback-apply workflow ...`
    - `Selected rollback-apply workflow ...` on idempotent same-phase replay
- existing `STATUS <job_id>` surface reused unchanged for post-advance read-model inspection

## Exact Tests Added

- `TestProcessDirectRollbackApplyPhaseCommandAdvancesWorkflowAndPreservesActiveRuntimePackPointer`
- `TestProcessDirectRollbackApplyPhaseCommandRejectsInvalidTransition`

## Validation Commands And Results

- `gofmt -w internal/agent/loop.go internal/agent/tools/taskstate.go internal/agent/loop_processdirect_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - current status after validation and report writing:

```text
## frank-v4-023-rollback-apply-phase-control-entry
 M internal/agent/loop.go
 M internal/agent/loop_processdirect_test.go
 M internal/agent/tools/taskstate.go
?? docs/maintenance/V4_023_ROLLBACK_APPLY_PHASE_CONTROL_ENTRY_AFTER.md
?? docs/maintenance/V4_023_ROLLBACK_APPLY_PHASE_CONTROL_ENTRY_BEFORE.md
```

## Explicit No-Execution Statement

- No activation mutation was implemented.
- No rollback apply behavior or reload behavior was implemented.
- No active runtime-pack pointer mutation, promotion behavior changes, evaluator execution, scoring behavior, or autonomy changes were implemented.
- This slice only adds a non-executing control entry that advances committed rollback-apply workflow phase state.

## Deferred Next V4 Candidates

- explicit operator-facing idempotent same-phase acknowledgement testing if a later slice depends on the selected-path wording
- any read-model expansion for `phase_updated_at` / `phase_updated_by` only if a later checkpoint needs operator visibility
- eventual execution orchestration that consumes advanced rollback-apply records in a separate slice
