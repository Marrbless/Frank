# V4-163 Autonomy Failure Pause After

Branch: `frank-v4-163-autonomy-failure-pause`

## Completed

- Added `AutonomyFailureRecord` storage under `autonomy/failures`.
- Added `AutonomyPauseRecord` storage under `autonomy/pauses`.
- Added deterministic repeated-failure pause IDs per budget.
- Added consecutive wake-cycle failure evaluation that resets after a successful wake cycle.
- Added proposal blocking with `E_REPEATED_FAILURE_PAUSE` when an active repeated-failure pause exists.
- Added failure and pause status fields to `autonomy_identity`.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`

## Remaining

- `AC-037`: owner pause record that blocks autonomy-originated hot-update proposals.
