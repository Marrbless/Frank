# GC2-TREAT-001B Main.go Memory Commands After

## Git diff summaries

### `git diff --stat`

```text
 cmd/picobot/main.go | 206 +---------------------------------------------------
 1 file changed, 1 insertion(+), 205 deletions(-)
```

### `git diff --numstat`

```text
1	205	cmd/picobot/main.go
```

`git diff` does not include newly added untracked files. The extracted helper file and treatment notes are listed below under files changed.

## Files changed

- `cmd/picobot/main.go`
- `cmd/picobot/main_memory_commands.go`
- `docs/maintenance/garbage-day/GC2_TREAT_001B_MAIN_GO_MEMORY_COMMANDS_BEFORE.md`
- `docs/maintenance/garbage-day/GC2_TREAT_001B_MAIN_GO_MEMORY_COMMANDS_AFTER.md`

## Exact functions moved

Moved from `cmd/picobot/main.go` into `cmd/picobot/main_memory_commands.go`:

- inline `memory` command subtree, reshaped into:
  - `newMemoryCmd`

The moved subtree still contains the same command family and behavior for:

- `memory read`
- `memory append`
- `memory write`
- `memory recent`
- `memory rank`

## Exact functions intentionally left in `main.go` and why

- `NewRootCmd`
  - Left in place because it still owns top-level CLI construction. The only change is that it now adds `newMemoryCmd()` instead of embedding the `memory` subtree inline.
- `main`
  - Left in place because it is the executable entrypoint.
- `newChannelsCmd` and the extracted channel login/prompt helpers
  - Left untouched because that seam was already completed in `GC2-TREAT-001A`.
- All gateway, mission bootstrap/runtime hook, mission inspect, scheduled-trigger, and mission status/assertion helpers
  - Left in place because they are protected runtime-truth or mission-control zones and are explicitly out of scope for `001B`.

## Runtime behavior

Runtime behavior was not changed.

- Command names are unchanged.
- Flags and help text are unchanged.
- Workspace/config path behavior is unchanged.
- Output and error text are unchanged.
- No gateway, provider, MCP, scheduled-trigger, mission bootstrap, or mission status behavior was touched.

## Validation commands and results

- `gofmt -w cmd/picobot/main.go cmd/picobot/main_memory_commands.go`
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
    - `?? cmd/picobot/main_memory_commands.go`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001B_MAIN_GO_MEMORY_COMMANDS_BEFORE.md`
    - `?? docs/maintenance/garbage-day/GC2_TREAT_001B_MAIN_GO_MEMORY_COMMANDS_AFTER.md`

## Deferred next candidates from the main.go assessment

- Scheduled-trigger governance helper extraction
- Mission inspect read-model helper extraction
- Mission status/assertion helper extraction
- Mission runtime/bootstrap hook extraction
