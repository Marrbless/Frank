# V4-002 Improvement Candidate Skeleton After

## git diff --stat

```text

```

`git diff --stat` is empty because this slice currently consists of new untracked files rather than modifications to tracked files.

## git diff --numstat

```text

```

`git diff --numstat` is empty for the same reason: all implementation files added in this slice are new and untracked.

## Files Changed

- `internal/missioncontrol/improvement_candidate_registry.go`
- `internal/missioncontrol/improvement_candidate_registry_test.go`
- `docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_BEFORE.md`
- `docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_AFTER.md`

## Exact Records / Helpers Added

Records:

- `ImprovementCandidateRef`
- `ImprovementCandidateRecord`

Not-found sentinel:

- `ErrImprovementCandidateRecordNotFound`

Storage, ref, normalization, and validation helpers:

- `StoreImprovementCandidatesDir`
- `StoreImprovementCandidatePath`
- `NormalizeImprovementCandidateRef`
- `NormalizeImprovementCandidateRecord`
- `ImprovementCandidateBaselinePackRef`
- `ImprovementCandidateProposedPackRef`
- `ImprovementCandidateHotUpdateGateRef`
- `ValidateImprovementCandidateRef`
- `ValidateImprovementCandidateRecord`
- `StoreImprovementCandidateRecord`
- `LoadImprovementCandidateRecord`
- `ListImprovementCandidateRecords`

Private/local helpers:

- `loadImprovementCandidateRecordFile`
- `validateImprovementCandidateLinkage`
- `normalizeImprovementCandidateStrings`
- `validateImprovementCandidateIdentifierField`

## Exact Tests Added

- `TestImprovementCandidateRecordRoundTripAndList`
- `TestImprovementCandidateReplayIsIdempotent`
- `TestImprovementCandidateValidationFailsClosed`
- `TestImprovementCandidateRejectsMissingRefsAndInvalidLinkage`

## Validation Commands And Results

- `gofmt -w internal/missioncontrol/improvement_candidate_registry.go internal/missioncontrol/improvement_candidate_registry_test.go`
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
## frank-v4-001-pack-registry-foundation
?? docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_AFTER.md
?? docs/maintenance/V4_002_IMPROVEMENT_CANDIDATE_SKELETON_BEFORE.md
?? internal/missioncontrol/improvement_candidate_registry.go
?? internal/missioncontrol/improvement_candidate_registry_test.go
```

## Deferred Next V4 Candidates

- append-only improvement run / ledger skeleton linked to candidate records without evaluation execution
- eval-suite record skeleton and immutable validation-basis envelope
- improvement candidate read-model / inspect exposure after the storage contract settles
- promotion and rollback durable records, still without apply or autonomy behavior
