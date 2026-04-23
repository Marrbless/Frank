# V4-032 Rollback Terminal-Failure Before

## Branch

- `frank-v4-032-rollback-terminal-failure-resolution`

## HEAD

- `fd1a637134b2ed04bc7b60bf37bddcd4de332b2e`

## Tags at HEAD

- `frank-v4-031-rollback-apply-retry`

## Ahead/behind upstream

- `430 0`

## git status --short --branch

```text
## frank-v4-032-rollback-terminal-failure-resolution
```

## Baseline go test -count=1 ./... result

```text
ok  	github.com/local/picobot/cmd/picobot	14.204s
?   	github.com/local/picobot/embeds	[no test files]
ok  	github.com/local/picobot/internal/agent	0.621s
ok  	github.com/local/picobot/internal/agent/memory	0.177s
ok  	github.com/local/picobot/internal/agent/skills	0.073s
ok  	github.com/local/picobot/internal/agent/tools	14.815s
ok  	github.com/local/picobot/internal/channels	0.590s
?   	github.com/local/picobot/internal/chat	[no test files]
ok  	github.com/local/picobot/internal/config	0.008s
ok  	github.com/local/picobot/internal/cron	2.306s
?   	github.com/local/picobot/internal/heartbeat	[no test files]
ok  	github.com/local/picobot/internal/mcp	0.028s
ok  	github.com/local/picobot/internal/missioncontrol	10.398s
ok  	github.com/local/picobot/internal/providers	0.026s
ok  	github.com/local/picobot/internal/session	0.090s
```

## Exact files planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/tools/taskstate.go`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_032_ROLLBACK_TERMINAL_FAILURE_BEFORE.md`
- `docs/maintenance/V4_032_ROLLBACK_TERMINAL_FAILURE_AFTER.md`

## Exact transition planned

- Add one explicit operator-driven transition:
  - `reload_apply_recovery_needed -> reload_apply_failed`
- Require non-empty operator reason text.
- Persist deterministic failure detail into `execution_error`.
- Keep exact replay of the same failure decision idempotent.

## Exact non-goals

- no rollback/apply execution
- no automatic retry
- no automatic success/failure inference
- no active runtime-pack pointer mutation
- no `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no new rollback-apply record creation
- no promotion, evaluator, scoring, autonomy, provider, or channel changes
- no cleanup outside this slice
- no dependency changes
- no commit
