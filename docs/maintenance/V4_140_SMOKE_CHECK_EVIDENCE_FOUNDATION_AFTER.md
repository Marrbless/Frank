# V4-140 Smoke Check Evidence Foundation - After

## Result

Completed smoke evidence/enforcement foundation for `AC-010` and `SF-004`.

## Changes

- Added durable `HotUpdateSmokeCheckRecord` storage under `runtime_packs/hot_update_smoke_checks`.
- Added deterministic smoke check IDs derived from hot-update ID and observation time.
- Added create/load/list helpers with candidate/gate linkage and divergent duplicate rejection.
- Added `AssessHotUpdateSmokeReadiness` and reload/apply enforcement.
- Blocked successful hot-update outcome creation and promotion unless selected Class 1+ smoke evidence passed.
- Added smoke readiness fields to hot-update gate status/read models.

## Validation

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./...`
