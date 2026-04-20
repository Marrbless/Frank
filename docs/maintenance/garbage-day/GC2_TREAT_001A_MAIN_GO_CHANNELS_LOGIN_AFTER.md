# GC2-TREAT-001A Main.go Channels Login After

## Git diff summaries

### `git diff --stat`

```text
 cmd/picobot/main.go | 280 +---------------------------------------------------
 1 file changed, 1 insertion(+), 279 deletions(-)
```

### `git diff --numstat`

```text
1	279	cmd/picobot/main.go
```

`git diff` does not include newly added untracked files. The extracted helper file and treatment notes are listed below under files changed.

## Files changed

- `cmd/picobot/main.go`
- `cmd/picobot/main_channels_login.go`
- `docs/maintenance/garbage-day/GC2_TREAT_001A_MAIN_GO_CHANNELS_LOGIN_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_001A_MAIN_GO_CHANNELS_LOGIN_AFTER.md`

## Exact functions moved

Moved from `cmd/picobot/main.go` into `cmd/picobot/main_channels_login.go`:

- `promptLine`
- `promptSecret`
- `parseAllowFrom`
- `setupTelegramInteractive`
- `setupDiscordInteractive`
- `setupSlackInteractive`
- `setupWhatsAppInteractive`
- prompt-secret support vars:
  - `promptSecretIsTerminal`
  - `promptSecretReadPassword`
- inline `channels login` subtree, reshaped into:
  - `newChannelsCmd`

## Exact functions intentionally left in `main.go` and why

- `NewRootCmd`
  - Left in place because it still owns top-level CLI construction for the executable. The only change is that it now adds `newChannelsCmd()` instead of embedding that subtree inline.
- `main`
  - Left in place because it is the executable entrypoint.
- All `agent`, `gateway`, mission bootstrap/runtime hook, scheduled-trigger, mission inspect, mission status/assertion, and watcher helpers
  - Left in place because they are protected runtime-truth or mission-control zones and explicitly out of scope for `001A`.

## Runtime behavior

Runtime behavior was not changed.

- The `channels login` command shape is unchanged.
- Prompts and prompt text are unchanged.
- Hidden secret input behavior is unchanged.
- Config write behavior is unchanged.
- Error messages are unchanged.
- No gateway, provider, MCP, mission bootstrap, or watcher behavior was touched.

## Validation commands and results

- `gofmt -w cmd/picobot/main.go cmd/picobot/main_channels_login.go`
  - passed
- `git diff --check`
  - passed
- `go test -count=1 ./cmd/picobot`
  - passed
- `go test -count=1 ./...`
  - passed
- `git status --short --branch`
  - result after slice:
    - `## frank-v3-foundation`
    - ` M cmd/picobot/main.go`
    - `?? cmd/picobot/main_channels_login.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001A_MAIN_GO_CHANNELS_LOGIN_BEFORE.md`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001A_MAIN_GO_CHANNELS_LOGIN_AFTER.md`

## Deferred next candidates from the main.go assessment

- Memory command builder extraction
- Scheduled-trigger governance helper extraction
- Mission inspect read-model helper extraction
- Mission status/assertion helper extraction
- Mission runtime/bootstrap hook extraction
