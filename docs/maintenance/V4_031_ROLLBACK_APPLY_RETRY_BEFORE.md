# V4-031 Rollback-Apply Retry Before

## Branch

- `frank-v4-031-rollback-apply-retry`

## HEAD

- `601a79f8fd225ba3081163a7f20786f0529dc482`

## Tags at HEAD

- none

## Ahead/behind upstream

- `429 0`

## git status --short --branch

```text
## frank-v4-031-rollback-apply-retry
```

## Baseline go test -count=1 ./... result

```text
ok  	github.com/local/picobot/cmd/picobot	13.661s
?   	github.com/local/picobot/embeds	[no test files]
ok  	github.com/local/picobot/internal/agent	0.557s
ok  	github.com/local/picobot/internal/agent/memory	0.119s
ok  	github.com/local/picobot/internal/agent/skills	0.009s
ok  	github.com/local/picobot/internal/agent/tools	14.508s
ok  	github.com/local/picobot/internal/channels	0.592s
?   	github.com/local/picobot/internal/chat	[no test files]
ok  	github.com/local/picobot/internal/config	0.014s
ok  	github.com/local/picobot/internal/cron	2.306s
?   	github.com/local/picobot/internal/heartbeat	[no test files]
ok  	github.com/local/picobot/internal/mcp	0.029s
ok  	github.com/local/picobot/internal/missioncontrol	10.063s
ok  	github.com/local/picobot/internal/providers	0.019s
ok  	github.com/local/picobot/internal/session	0.084s
```

## Exact files planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `internal/agent/loop_processdirect_test.go`
- `docs/maintenance/V4_031_ROLLBACK_APPLY_RETRY_BEFORE.md`
- `docs/maintenance/V4_031_ROLLBACK_APPLY_RETRY_AFTER.md`

## Exact state transitions planned

- Allow `ExecuteRollbackApplyReloadApply(...)` to start from:
  - `reload_apply_recovery_needed`
- On retry start:
  - clear stale `execution_error`
  - transition `reload_apply_recovery_needed -> reload_apply_in_progress`
- Then reuse the existing bounded reload/apply execution flow:
  - `reload_apply_in_progress -> reload_apply_succeeded`
  - `reload_apply_in_progress -> reload_apply_failed`
- Preserve existing idempotent replay for:
  - `reload_apply_succeeded`
- Preserve existing rejection for:
  - `reload_apply_failed`
  - all non-retry phases outside the existing execution path

## Exact non-goals

- no new operator command
- no new rollback-apply record creation
- no active runtime-pack pointer mutation beyond the already-switched state
- no second `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no explicit terminal-failure resolution action
- no promotion, evaluator, scoring, autonomy, provider, or channel changes
- no dependency changes
- no commit
