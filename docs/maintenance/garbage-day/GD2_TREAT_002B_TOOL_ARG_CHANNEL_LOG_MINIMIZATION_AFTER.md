## GD2-TREAT-002B After

Date: 2026-04-20

### Live state after treatment

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- Base HEAD before treatment: `a17eaf7997a65853d1b8a3c00411b5e6ef3c7572`
- Upstream containment: `HEAD` remained `0 behind upstream/main` throughout the slice
- Commit action: none

### Implemented changes

1. `internal/agent/tools/registry.go`
   - Replaced argument-key summaries with structural argument-type summaries.
   - Tool execution logs now keep count and coarse type mix without exposing raw key names such as `authorization`, `query`, `body`, or `content`.

2. `internal/agent/loop.go`
   - No direct code change required.
   - Tool-activity notifications now inherit the narrower registry summary and no longer surface raw argument key names back to the channel.

3. `internal/channels/slack.go`
   - Replaced inbound mention and DM content-fragment logs with message-size summaries.
   - Attachment URLs are no longer emitted in Slack inbound logs.

4. `internal/channels/discord.go`
   - Replaced inbound content-fragment logs with message-size summaries.
   - Inline attachment URLs are no longer emitted in Discord inbound logs.

5. `internal/channels/whatsapp.go`
   - Replaced inbound content-fragment logs with message-size summaries.

6. `internal/channels/logging.go`
   - Added one small shared helper so the three treated channel adapters emit the same bounded content summary shape.

### Focused test coverage added or updated

- `internal/agent/tools/registry_test.go`
  - asserts registry logs omit raw argument keys and values
  - asserts argument summaries stay structural
- `internal/agent/loop_tool_test.go`
  - asserts tool-activity notifications omit raw argument key names and values
- `internal/channels/logging_test.go`
  - asserts Slack and Discord inbound logs omit raw content and attachment URLs
- `internal/channels/whatsapp_test.go`
  - asserts WhatsApp inbound logs omit raw content

### Validation

- `gofmt -w internal/agent/tools/registry.go internal/agent/tools/registry_test.go internal/agent/loop_tool_test.go internal/channels/logging.go internal/channels/slack.go internal/channels/discord.go internal/channels/whatsapp.go internal/channels/logging_test.go internal/channels/whatsapp_test.go`
- `git diff --check`
  - passed
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/channels`
  - passed
- `go test -count=1 ./...`
  - passed

### Residual risk intentionally left for later slices

- Unauthorized-access logs still include identity details where they already existed; this slice only removed raw content exposure.
- Runtime validation evidence still stores full successful tool arguments and results where mission-control semantics require it; that audit-adjacent persistence was intentionally left outside this bounded treatment.
