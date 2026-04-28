# V4-155 Repeated Crash Rollback After

Branch: `frank-v4-155-repeated-crash-rollback`

## Requirement Rows

- `AC-018` moved from `MISSING` to `DONE`.

## Implemented

- Added append-only `RuntimeFailureEventRecord` storage under `runtime_packs/runtime_failure_events`.
- Added `RuntimePackQuarantineRecord` storage under `runtime_packs/quarantined`.
- Added `RepeatedFailureTerminalBlockerRecord` storage under `runtime_packs/repeated_failure_blockers`.
- Added `AssessRepeatedActivePackFailures`.
- Defaulted the local policy threshold to 3 consecutive smoke/runtime failures.
- With a distinct LKG pointer, repeated active-candidate failures create rollback/rollback-apply records, advance rollback apply to ready, switch the active pointer to LKG through the existing rollback-apply path, and quarantine the failing pack.
- Without LKG, repeated failures create a terminal blocker and preserve the active pointer.

## Validation

- Focused package test: `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- Required full validation: `git diff --check` and `/usr/local/go/bin/go test -count=1 ./...`

No real process supervision, phone hardware, external service call, network call, or device side effect was added.
