# V4-160 Standing Directive Wake Cycle After

Branch: `frank-v4-160-standing-directive-wake-cycle`

## Completed

- Added `StandingDirectiveRecord` storage under `autonomy/standing_directives`.
- Added `WakeCycleRecord` storage under `autonomy/wake_cycles`.
- Added deterministic wake-cycle IDs derived from directive ID and tick start time.
- Added `CreateWakeCycleProposalFromStandingDirective` for due local mission proposals.
- Added fail-closed checks for not-due directives, retired directives, owner-paused directives, disallowed mission families, disallowed planes/hosts, and mission-family plane mismatches.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`

## Remaining

- `AC-034`: no eligible work heartbeat/status surface.
- `AC-035`: autonomy budget records and enforcement.
- `AC-036`: repeated autonomy failure pause.
- `AC-037`: owner pause record that blocks autonomy-originated hot-update proposals.
