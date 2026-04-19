# GD2-TREAT-009B Upstream Integration Resolution

## Starting State

- Starting branch: `frank-v3-foundation`
- Starting HEAD: `e3f84cb00a5b292e2e0795c7616b491e831799fe`
- Integration branch used: `frank-upstream-sync-gd2-resolve`
- Upstream branch used: `upstream/main`
- Merge base: `8966cb41b5712cd7dcfa60206c29e79e62fb4123`

## Conflicts Encountered

- `cmd/picobot/main.go`
- `internal/agent/loop.go`
- `docs/HOW_TO_START.md`

## Resolution Summary By Conflict File

### `cmd/picobot/main.go`

- Preserved Frank mission bootstrap, mission status snapshot writes, operator set-step wiring, deferred scheduled-trigger governance, and gateway runtime-control startup.
- Accepted upstream version bump to `0.2.1`.
- Accepted upstream MCP wiring by passing `cfg.MCPServers` into `agent.NewAgentLoop(...)`.
- Accepted upstream tool-activity config by applying `SetToolActivityIndicator(false)` only when `EnableToolActivityIndicator` is explicitly false.
- Added `defer ag.Close()` in both one-shot agent mode and gateway mode so configured MCP clients are shut down cleanly.
- Did not remove or weaken Frank-specific owner-control, mission-control, treasury, capability, Telegram, approval, or runtime initialization.

### `internal/agent/loop.go`

- Preserved Frank runtime/control semantics:
  - mission runtime state
  - operator command processing
  - owner-facing budget accounting
  - approval/revocation behavior
  - Zoho send/reply preparation and persistence hooks
  - mission-required execution context and guard behavior
- Accepted upstream MCP integration as config-driven registration only:
  - optional MCP client connection list on `AgentLoop`
  - optional registration of `mcp_{server}_{tool}` tools when `mcpServers` is configured
  - explicit `Close()` lifecycle for MCP clients
- Accepted upstream tool-activity indicator as observability-only:
  - `enableToolActivity` defaults to `true`
  - interim `Running` / `done` / `failed` notices are gated by that flag
  - Frank mission-control decisions are unchanged
- Kept Frank authority controls in place:
  - registry-level mission-required enforcement
  - execution-context filtering of allowed tools
  - guard-based approval and protected-surface rejection logic
- Resolved constructor compatibility by making `NewAgentLoop(...)` accept optional MCP server config via a variadic final parameter. This keeps existing Frank call sites working while accepting upstream’s new call pattern.

### `docs/HOW_TO_START.md`

- Preserved Frank-specific startup and mission-control command surface.
- Rejected upstream’s hardcoded “16 built-in tools” wording because Frank’s exposed built-in surface differs.
- Kept the Frank wording that `cron` is conditional.
- Preserved and clarified the MCP section:
  - MCP tools are optional
  - MCP is inert unless configured
  - tool-activity messages are optional and config-gated
- Did not add any V4 claims.

## Upstream Features Accepted

- MCP server integration files and tests:
  - `internal/mcp/client.go`
  - `internal/mcp/client_test.go`
  - `internal/agent/tools/mcp.go`
  - `internal/agent/tools/mcp_test.go`
- MCP config surface:
  - `internal/config/schema.go`
  - `internal/config/onboard.go`
  - `docs/CONFIG.md`
  - `docs/DEVELOPMENT.md`
  - `README.md`
- Tool-activity indicator config:
  - `EnableToolActivityIndicator` in config
  - `PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR`
  - observability-only gating in the agent loop
- Version bump to `0.2.1`
- Upstream JSON encode/decode hardening in the MCP support surface and associated tests

## Upstream Features Rejected Or Gated

- MCP is not treated as an always-on authority expansion.
  - Status: `opt-in`, `disabled-by-default`, and inert unless `mcpServers` is configured.
- Tool-activity is not allowed to affect runtime or mission-control behavior.
  - Status: observability-only.
- Upstream did not reopen provider onboarding lanes in this resolution.
- Upstream did not alter treasury, approval, capability exposure, owner-control, or persistence semantics in this resolution.
- Upstream’s generic built-in tool count language in `docs/HOW_TO_START.md` was rejected in favor of Frank-accurate wording.

## MCP Status

- MCP status: `opt-in` and `disabled-by-default`
- Why:
  - default config initializes `mcpServers` as an empty map
  - MCP tools are registered only when a server is explicitly configured
  - Frank mission execution still filters tools through execution-context allowed-tool sets and the existing tool guard

## Tool-Activity Status

- Tool-activity status: `observability-only`
- Why:
  - the new config flag only controls interim chat notifications
  - no mission-control branching, approval decision, or runtime transition depends on it

## JSON Encode/Decode Handling

- Accepted:
  - upstream JSON encode/decode hardening in the new MCP support path and tests
- Not changed:
  - Frank-specific mission-control error semantics in the main loop
  - Frank approval, rejection-code, and protected-surface behavior

## Protected Frank Surfaces Checked

- Owner-control behavior: preserved
- Telegram-only provider onboarding status: preserved
- Treasury semantics: preserved
- Capability exposure semantics: preserved
- Approval/revocation semantics: preserved
- Persistence/hydration behavior: preserved
- One-active-job/runtime-control expectations: preserved

## Files Changed By The Merge

- `README.md`
- `cmd/picobot/main.go`
- `docker/README.md`
- `docker/docker-compose.yml`
- `docker/entrypoint.sh`
- `docs/CONFIG.md`
- `docs/DEVELOPMENT.md`
- `docs/HOW_TO_START.md`
- `internal/agent/loop.go`
- `internal/agent/loop_processdirect_test.go`
- `internal/agent/loop_remember_test.go`
- `internal/agent/loop_test.go`
- `internal/agent/loop_tool_test.go`
- `internal/agent/loop_web_test.go`
- `internal/agent/loop_write_memory_test.go`
- `internal/agent/tools/mcp.go`
- `internal/agent/tools/mcp_test.go`
- `internal/channels/whatsapp_test.go`
- `internal/config/loader.go`
- `internal/config/onboard.go`
- `internal/config/schema.go`
- `internal/mcp/client.go`
- `internal/mcp/client_test.go`

## Git Diff Stat

```text
 README.md                                 |  28 ++-
 cmd/picobot/main.go                       |  14 +-
 docker/README.md                          |   2 +
 docker/docker-compose.yml                 |   2 +
 docker/entrypoint.sh                      |  11 +
 docs/CONFIG.md                            | 106 ++++++++-
 docs/DEVELOPMENT.md                       |  21 ++
 docs/HOW_TO_START.md                      |   6 +
 internal/agent/loop.go                    |  84 +++++--
 internal/agent/loop_processdirect_test.go |   2 +-
 internal/agent/loop_remember_test.go      |   2 +-
 internal/agent/loop_test.go               |   2 +-
 internal/agent/loop_tool_test.go          |   2 +-
 internal/agent/loop_web_test.go           |   2 +-
 internal/agent/loop_write_memory_test.go  |   2 +-
 internal/agent/tools/mcp.go               |  40 ++++
 internal/agent/tools/mcp_test.go          | 136 +++++++++++
 internal/channels/whatsapp_test.go        |  12 +-
 internal/config/loader.go                 |   4 +
 internal/config/onboard.go                |  19 +-
 internal/config/schema.go                 |  15 +-
 internal/mcp/client.go                    | 363 ++++++++++++++++++++++++++++++
 internal/mcp/client_test.go               | 175 ++++++++++++++
 23 files changed, 991 insertions(+), 59 deletions(-)
```

## Git Name Status

```text
M	README.md
M	cmd/picobot/main.go
M	docker/README.md
M	docker/docker-compose.yml
M	docker/entrypoint.sh
M	docs/CONFIG.md
M	docs/DEVELOPMENT.md
M	docs/HOW_TO_START.md
M	internal/agent/loop.go
M	internal/agent/loop_processdirect_test.go
M	internal/agent/loop_remember_test.go
M	internal/agent/loop_test.go
M	internal/agent/loop_tool_test.go
M	internal/agent/loop_web_test.go
M	internal/agent/loop_write_memory_test.go
A	internal/agent/tools/mcp.go
A	internal/agent/tools/mcp_test.go
M	internal/channels/whatsapp_test.go
M	internal/config/loader.go
M	internal/config/onboard.go
M	internal/config/schema.go
A	internal/mcp/client.go
A	internal/mcp/client_test.go
```

## Validation Commands And Results

- `gofmt -w cmd/picobot/main.go internal/agent/loop.go`
  - passed
- `git diff --check`
  - passed
- `git diff --check --cached`
  - failed due upstream-merged trailing whitespace in `README.md`
- `go test -count=1 ./internal/agent`
  - passed
- `go test -count=1 ./cmd/picobot`
  - passed
- `go test -count=1 ./internal/agent/tools`
  - passed
- `go test -count=1 ./internal/missioncontrol`
  - passed
- `go test -count=1 ./...`
  - passed

## Remaining Risks

- The merge is still uncommitted and should receive human diff review before any commit.
- The staged merge still contains upstream README trailing whitespace, so a commit in the current state would fail a cached diff-check gate unless a human approves or fixes that doc-only issue separately.
- MCP adds a new optional tool-registration surface. It is config-gated, but once configured it expands the callable tool set outside the current Frank-specific tool family.
- `cmd/picobot/main.go` and `internal/agent/loop.go` remain large protected files. The integration is validated, but future cleanup in those files should still happen only in narrow slices.
- Public docs now mention optional MCP and tool-activity config. Human review should confirm the final wording matches the intended operator surface.

## Recommendation

- Recommendation: `needs human diff review`

## Current Branch State

- Branch left for review: `frank-upstream-sync-gd2-resolve`
- Merge state: resolved, staged, uncommitted
- No commit was created
