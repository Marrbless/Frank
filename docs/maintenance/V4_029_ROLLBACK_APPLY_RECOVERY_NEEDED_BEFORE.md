# V4-029 Rollback Apply Recovery-Needed Before

## Branch

- `frank-v4-029-rollback-apply-recovery-needed`

## HEAD

- `063e266c4eea512ad8aff5d8165a9fd4a2fc87ed`

## Tags at HEAD

- none

## Ahead/behind upstream

- `427 0`

## git status --short --branch

```text
## frank-v4-029-rollback-apply-recovery-needed
```

## Baseline go test -count=1 ./... result

```text
ok  	github.com/local/picobot/cmd/picobot	13.985s
?   	github.com/local/picobot/embeds	[no test files]
ok  	github.com/local/picobot/internal/agent	0.529s
ok  	github.com/local/picobot/internal/agent/memory	0.131s
ok  	github.com/local/picobot/internal/agent/skills	0.010s
ok  	github.com/local/picobot/internal/agent/tools	14.548s
ok  	github.com/local/picobot/internal/channels	0.595s
?   	github.com/local/picobot/internal/chat	[no test files]
ok  	github.com/local/picobot/internal/config	0.009s
ok  	github.com/local/picobot/internal/cron	2.306s
?   	github.com/local/picobot/internal/heartbeat	[no test files]
ok  	github.com/local/picobot/internal/mcp	0.033s
ok  	github.com/local/picobot/internal/missioncontrol	9.808s
ok  	github.com/local/picobot/internal/providers	0.018s
ok  	github.com/local/picobot/internal/session	0.092s
```

## Exact files planned

- `internal/missioncontrol/rollback_apply_registry.go`
- `internal/missioncontrol/rollback_apply_registry_test.go`
- `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_BEFORE.md`
- `docs/maintenance/V4_029_ROLLBACK_APPLY_RECOVERY_NEEDED_AFTER.md`

## Exact state transition(s) planned

- Extend rollback-apply durable phase with `reload_apply_recovery_needed`.
- Add a bounded reconciliation helper that normalizes:
  - `reload_apply_in_progress -> reload_apply_recovery_needed`
- Make exact replay idempotent by treating:
  - `reload_apply_recovery_needed -> reload_apply_recovery_needed` as a no-op return when linkage is still coherent.
- Fail closed with an error and no mutation when linked rollback or active-pointer linkage is missing or invalid.

## Exact non-goals

- no active runtime-pack pointer mutation
- no `reload_generation` increment
- no `last_known_good_pointer.json` mutation
- no reload/apply retry
- no automatic success
- no automatic terminal failure
- no control-surface expansion unless strictly required
- no promotion, evaluator, scoring, autonomy, provider, or channel changes
- no dependency changes
- no commit
