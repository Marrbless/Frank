## GD2-TREAT-002A After

Date: 2026-04-19

Canonical repo: `/mnt/d/pbot/picobot`
Branch: `frank-v3-foundation`
HEAD at start of treatment: `b105d5de8ea4eeeace16d0761133523ae47169f2`

### Implemented slice

1. `internal/providers/openai.go`
   - Removed raw non-2xx provider body logging.
   - Replaced surfaced non-2xx errors with status-only text plus optional request ID.
2. `internal/agent/tools/registry.go`
   - Replaced raw tool argument JSON logging with key-only summaries.
   - Replaced raw remote/MCP failure logging with normalized summaries.
3. `internal/agent/loop.go`
   - Replaced raw provider-error logging with normalized provider summaries.
   - Replaced raw MCP startup failure logging with normalized connection summaries.
   - Replaced raw tool activity notifications with key-only argument summaries.
   - Replaced raw tool failure notifications, model-visible tool messages, and failed-action persistence text with normalized error summaries.
   - Replaced raw `ProcessDirect` provider errors with normalized surfaced errors.

### Focused tests added

- `internal/providers/openai_test.go`
  - asserts non-2xx logs and surfaced errors omit provider body content and secret-like strings.
- `internal/agent/tools/registry_test.go`
  - asserts tool logs omit raw argument values and raw remote error payloads.
- `internal/agent/loop_processdirect_test.go`
  - asserts direct provider errors omit raw provider payloads.
- `internal/agent/loop_tool_test.go`
  - asserts tool activity notifications and model-visible tool error messages omit secret-bearing payloads and raw arguments.

### Validation

- `gofmt -w internal/providers/openai.go internal/providers/openai_test.go internal/agent/tools/registry.go internal/agent/tools/registry_test.go internal/agent/loop.go internal/agent/loop_processdirect_test.go internal/agent/loop_tool_test.go`
- `git diff --check`
- `go test -count=1 ./internal/providers`
- `go test -count=1 ./internal/agent/tools`
- `go test -count=1 ./internal/agent`
- `go test -count=1 ./...`

All validation commands passed.

### Residual scope left untouched by design

- No V4 work.
- No MCP transport/client rewrite in `internal/mcp/client.go`.
- No channel logging cleanup outside this provider/tool/MCP error slice.
- No dependency or semantic changes outside the bounded treatment.
