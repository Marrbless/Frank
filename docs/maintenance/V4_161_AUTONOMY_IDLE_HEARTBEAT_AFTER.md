# V4-161 Autonomy Idle Heartbeat After

Branch: `frank-v4-161-autonomy-idle-heartbeat`

## Completed

- Added deterministic `wake-cycle-no-eligible-*` IDs.
- Added `CreateNoEligibleAutonomousActionHeartbeat` for local no-work ticks.
- Added fail-closed detection that prevents a no-eligible heartbeat when an active, unpaused standing directive is due.
- Added `autonomy_identity` read models for standing directives and wake cycles.
- Added committed mission snapshot wiring for the autonomy identity surface.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`

## Remaining

- `AC-035`: autonomy budget records and enforcement.
- `AC-036`: repeated autonomy failure pause.
- `AC-037`: owner pause record that blocks autonomy-originated hot-update proposals.
