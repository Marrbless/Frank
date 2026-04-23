## V4-046 Hot-Update Terminal Failure Before

- branch: `frank-v4-046-hot-update-terminal-failure-resolution`
- HEAD: `1744ecb82a98b1e6ba74d7129f16985660ed0426`
- ahead/behind `upstream/main`: `444/0`
- git status --short --branch:
  - `## frank-v4-046-hot-update-terminal-failure-resolution`
- baseline `go test -count=1 ./...` result:
  - pass when rerun outside the sandbox with Go build-cache and loopback socket access
  - initial sandboxed run failed because `/home/omar/.cache/go-build` was read-only and `httptest` could not bind loopback sockets

## Before-State Gap

- V4-043 introduced `reload_apply_recovery_needed` for hot-update reload/apply unknown-outcome recovery.
- V4-045 added explicit operator retry from `reload_apply_recovery_needed` through the existing `HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>` path.
- The hot-update lane still had no explicit operator-driven terminal-failure resolution from `reload_apply_recovery_needed`.
- Operators could retry the same committed hot-update gate, but could not deliberately close the same gate as terminally failed with deterministic operator reason text.

## Planned Behavior

- Add the hot-update sibling of the rollback V4-032 terminal-failure path.
- Permit only this explicit transition:
  - `reload_apply_recovery_needed -> reload_apply_failed`
- Require non-empty operator reason text.
- Store terminal failure detail exactly as:
  - `operator_terminal_failure: <reason>`
- Treat exact replay with the same terminal-failure decision and reason as idempotent.
- Fail closed if a different reason is submitted after terminal failure.
- Use the existing direct operator command path and the existing `TaskState` wrapper pattern.
- Keep the committed `HotUpdateGateRecord` as the sole workflow authority.

## Explicit Non-Goals

- no active runtime-pack pointer mutation
- no `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no `HotUpdateOutcomeRecord` creation
- no `PromotionRecord` creation
- no new hot-update gate or apply record
- no automatic retry
- no automatic success inference
- no automatic failure inference outside the explicit operator terminal-failure command
- no promotion behavior changes
- no evaluator, scoring, autonomy, provider, or channel behavior changes
- no dependency changes
