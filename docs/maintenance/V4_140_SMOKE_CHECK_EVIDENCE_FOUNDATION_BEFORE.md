# V4-140 Smoke Check Evidence Foundation - Before

## Controller Rows

- `AC-010`: MISSING
- `SF-004`: MISSING

## Gap

`HotUpdateGateRecord` had `smoke_check_refs`, but there was no durable smoke evidence record, no readiness assessment, no status/read-model blocker, and no enforcement preventing Class 1+ hot updates from reaching reload/apply success, successful outcome creation, or promotion without passing smoke evidence.

## Intended Slice

Add deterministic local smoke evidence records and use them to block Class 1+ hot-update success/promotion when smoke evidence is missing or failed. Keep Class 0 exempt and avoid real external smoke execution.

## Validation Plan

- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./...`
