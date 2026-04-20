## GD2-TREAT-002A Before

Date: 2026-04-19

Canonical repo: `/mnt/d/pbot/picobot`
Branch: `frank-v3-foundation`
HEAD: `b105d5de8ea4eeeace16d0761133523ae47169f2`

### Preconditions

- `pwd` = `/mnt/d/pbot/picobot`
- branch = `frank-v3-foundation`
- `git rev-list --left-right --count HEAD...upstream/main` = `367 0`
- worktree clean before changes
- `docs/maintenance/garbage-day/GD2_TREAT_002_WEB_PROVIDER_LOG_SURFACE_ASSESSMENT.md` exists
- baseline `go test -count=1 ./...` passed before changes

### In-scope raw exposure surfaces

1. `internal/providers/openai.go`
   - `doJSON` logs raw non-2xx provider bodies.
   - `doJSON` returns raw provider body text in surfaced errors.
2. `internal/agent/tools/registry.go`
   - `Execute` logs raw tool argument JSON for every tool call.
   - `Execute` logs raw downstream error text on failure.
3. `internal/agent/loop.go`
   - logs raw provider and MCP startup failures.
   - sends raw tool failure text into channel notifications.
   - feeds raw tool failure text back into provider/model tool messages.
   - returns raw provider/tool failures from `ProcessDirect`, which reach CLI stderr unchanged.

### Smallest safe treatment target

- Replace raw provider body logging with status-only or request-id-aware logging.
- Stop logging raw tool arguments; retain tool name plus low-risk structure only.
- Stop logging or surfacing raw provider/tool/MCP error payloads in loop-visible paths.
- Preserve operator debuggability with stable summaries such as tool name, status, elapsed time, request ID, and coarse failure class.

### Explicit non-goals

- No V4 work.
- No broad log cleanup outside the named files.
- No MCP client transport rewrite.
- No channel/provider onboarding changes.
- No dependency changes.
