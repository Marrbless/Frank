# GC4-TREAT-001 Status Autonomy Split Before

Branch: `frank-garbage-day-gc4-001-status-autonomy-split`

## Target

`GC4-001`: split the V4 autonomy/status read-model cluster out of `internal/missioncontrol/status.go`.

## Starting Evidence

- `internal/missioncontrol/status.go`: `4543` lines at post-V4 kickoff.
- The autonomy identity cluster was contiguous and included:
  - `OperatorAutonomyIdentityStatus`
  - autonomy budget, standing directive, wake-cycle, failure, pause, and owner-pause status structs
  - `WithAutonomyIdentity`
  - `LoadOperatorAutonomyIdentityStatus`
  - autonomy status loaders and last-error helpers
- Focused test coverage already existed in `internal/missioncontrol/status_autonomy_identity_test.go`.

## Slice Constraint

Move only the autonomy status/read-model cluster into a same-package file. Preserve names, JSON fields, loader behavior, read-only semantics, and all validation paths.
