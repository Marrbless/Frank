# V4-007 Improvement Run Ledger After

## git diff --stat

```text

```

`git diff --stat` is empty because this slice currently consists of new untracked files rather than modifications to tracked files.

## git diff --numstat

```text

```

`git diff --numstat` is empty for the same reason: all implementation files added in this slice are new and untracked.

## Files Changed

- `internal/missioncontrol/improvement_run_registry.go`
- `internal/missioncontrol/improvement_run_registry_test.go`
- `docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_BEFORE.md`
- `docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_AFTER.md`

## Exact Records / Helpers Added

Records and bounded enums:

- `ImprovementRunState`
- `ImprovementRunDecision`
- `ImprovementRunRef`
- `ImprovementRunRecord`

Not-found sentinel:

- `ErrImprovementRunRecordNotFound`

Storage, ref, normalization, and validation helpers:

- `StoreImprovementRunsDir`
- `StoreImprovementRunPath`
- `NormalizeImprovementRunRef`
- `NormalizeImprovementRunRecord`
- `ImprovementRunCandidateRef`
- `ImprovementRunEvalSuiteRef`
- `ImprovementRunBaselinePackRef`
- `ImprovementRunCandidatePackRef`
- `ImprovementRunHotUpdateGateRef`
- `ValidateImprovementRunRef`
- `ValidateImprovementRunRecord`
- `StoreImprovementRunRecord`
- `LoadImprovementRunRecord`
- `ListImprovementRunRecords`

Private/local helpers:

- `loadImprovementRunRecordFile`
- `validateImprovementRunLinkage`
- `validateImprovementRunIdentifierField`
- `isValidImprovementRunState`
- `isValidImprovementRunDecision`

## Exact Tests Added

- `TestImprovementRunRecordRoundTripAndList`
- `TestImprovementRunReplayIsIdempotentAndAppendOnly`
- `TestImprovementRunValidationFailsClosed`
- `TestImprovementRunRejectsMissingLinkedRefs`
- `TestLoadImprovementRunRecordNotFound`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/improvement_run_registry.go internal/missioncontrol/improvement_run_registry_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - final:

```text
## frank-v4-007-improvement-run-ledger-skeleton
?? docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_AFTER.md
?? docs/maintenance/V4_007_IMPROVEMENT_RUN_LEDGER_BEFORE.md
?? internal/missioncontrol/improvement_run_registry.go
?? internal/missioncontrol/improvement_run_registry_test.go
```

## Deferred Next V4 Candidates

- improvement-run read-model / inspect exposure after later control surfaces need operator visibility
- append-only candidate-result record skeleton without evaluator execution or scoring
- hot-update outcome record or ledger skeleton separate from gate envelope
- promotion and rollback durable records, still without apply or autonomy behavior
