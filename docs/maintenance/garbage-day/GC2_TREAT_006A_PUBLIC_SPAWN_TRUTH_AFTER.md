# GC2-TREAT-006A Public Spawn Truth Cleanup After

Date: 2026-04-20

## Diff summary

### `git diff --stat`

```text
 README.md                       | 13 ++++++-------
 internal/config/onboard.go      |  3 ---
 internal/config/onboard_test.go |  9 +++++++++
 3 files changed, 15 insertions(+), 10 deletions(-)
```

### `git diff --numstat`

```text
6	7	README.md
0	3	internal/config/onboard.go
9	0	internal/config/onboard_test.go
```

## Files changed

- `README.md`
- `internal/config/onboard.go`
- `internal/config/onboard_test.go`
- `docs/maintenance/garbage-day/GC2_TREAT_006A_PUBLIC_SPAWN_TRUTH_BEFORE.md`

## Exact public/operator-facing spawn references removed or corrected

- Removed the false public built-in tool row from `README.md`:
  - removed `` `spawn` | Return a stub acknowledgement for a background-subagent request; does not launch one ``
- Removed the false onboarding-generated `TOOLS.md` source entry from `internal/config/onboard.go`:
  - removed `### spawn`
  - removed `Spawn a background subagent process.`

## Exact generation/source path corrected

- Corrected the source of onboarding-generated operator-facing tool docs in `internal/config/onboard.go`.
- No generated artifact was hand-edited directly.

## Test coverage added

- `internal/config/onboard_test.go` now asserts the generated `TOOLS.md` content does not advertise `spawn`.

## Runtime semantics

- Runtime semantics were not changed.
- This slice did not modify:
  - `internal/agent/loop.go`
  - `internal/missioncontrol/step_validation.go`
- No tool registration or runtime behavior changed.

## Validation commands and results

- `gofmt -w internal/config/onboard.go internal/config/onboard_test.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - after validation and before writing this after note:
    - `## frank-v3-foundation`
    - ` M README.md`
    - ` M internal/config/onboard.go`
    - ` M internal/config/onboard_test.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_006A_PUBLIC_SPAWN_TRUTH_BEFORE.md`

## Deferred follow-on

- `GC2-TREAT-006B` spawn semantic parity cleanup
  - still deferred
  - this slice intentionally did not touch internal semantic parity or validation logic
