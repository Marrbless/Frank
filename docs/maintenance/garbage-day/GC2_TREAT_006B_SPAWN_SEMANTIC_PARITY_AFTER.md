# GC2-TREAT-006B Spawn Semantic Parity After

Date: 2026-04-20

## Diff summary

### `git diff --stat`

```text
 internal/missioncontrol/step_validation.go      |  2 +-
 internal/missioncontrol/step_validation_test.go | 16 ++++++++++++++++
 2 files changed, 17 insertions(+), 1 deletion(-)
```

### `git diff --numstat`

```text
1	1	internal/missioncontrol/step_validation.go
16	0	internal/missioncontrol/step_validation_test.go
```

## Files changed

- `internal/missioncontrol/step_validation.go`
- `internal/missioncontrol/step_validation_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_006B_SPAWN_SEMANTIC_PARITY_BEFORE.md`

## Exact semantic spawn references corrected

- Corrected `internal/missioncontrol/step_validation.go`
  - `hasDiscussionSideEffects(...)` no longer treats `spawn` as an effectful/available tool
- Added focused tests in `internal/missioncontrol/step_validation_test.go`
  - `TestHasDiscussionSideEffectsIgnoresSpawn`
  - `TestHasDiscussionSideEffectsStillTreatsCronAsEffectful`

## Runtime semantics

- Runtime semantics were not changed.
- This slice did not modify `internal/agent/loop.go`.
- `spawn` is still not exposed at runtime, and this change only brings semantic validation into parity with that existing runtime truth.

## Validation commands and results

- `gofmt -w internal/missioncontrol/step_validation.go internal/missioncontrol/step_validation_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - before writing this after note:
    - `## frank-v3-foundation`
    - ` M internal/missioncontrol/step_validation.go`
    - ` M internal/missioncontrol/step_validation_test.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_006B_SPAWN_SEMANTIC_PARITY_BEFORE.md`
