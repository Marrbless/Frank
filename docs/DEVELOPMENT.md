# Development Guide

This document covers how to set up a local development environment, build, test, and publish Picobot.

## What You'll Need

- [Go](https://go.dev/dl/) 1.26+ installed
- [Docker](https://www.docker.com/) installed (for container builds)
- A [Docker Hub](https://hub.docker.com/) account (for publishing)

## Project Structure

```
cmd/picobot/          CLI entry point (main.go)
embeds/               Embedded assets (sample skills bundled into binary)
  skills/             Sample skills extracted on onboard
internal/
  agent/              Agent loop, context, tools, skills
  chat/               Chat message hub (Inbound / Outbound channels)
  channels/           Telegram and Discord integration
  config/             Config schema, loader, onboarding
  cron/               Cron scheduler
  heartbeat/          Periodic task checker
  mcp/                MCP client (stdio + HTTP transports)
  memory/             Memory read/write/rank
  providers/          OpenAI-compatible provider (OpenAI, OpenRouter, Ollama, etc.)
  session/            Session manager
docker/               Dockerfile, compose, entrypoint
```

## Local Development

### Clone and install dependencies

```sh
git clone https://github.com/user/picobot.git
cd picobot
go mod download
```

### Build the binary

```sh
go build -o picobot ./cmd/picobot
```

The binary will be created in the current directory.

### Run locally

```sh
# First time? Run onboard to create ~/.picobot config and workspace
./picobot onboard

# Try a quick query
./picobot agent -m "Hello!"

# Login to channels (Telegram, Discord, Slack, WhatsApp)
./picobot channels login

# Start the full gateway (includes channels, heartbeat, etc.)
./picobot gateway
```

For Frank mission-control development and operator runbooks, the relevant runtime surface is CLI-driven rather than config-driven. See [`docs/HOW_TO_START.md`](./HOW_TO_START.md) for the current `gateway --mission-*` startup flags and `picobot mission ...` operator commands.

### Run tests

```sh
# Run the normal test surface
make test

# Run the lite build-tag test surface
make test-lite

# Prove full/lite build-tag contracts and build both binaries under /tmp
make test-build-tags

# Run the targeted race-sensitive package surface
make test-race

# Run all local checks used by the repo
make verify

# Generate a local coverage profile without enforcing a threshold
make coverage

# Build the Docker image and run a non-publishing --help smoke check
make docker-smoke

# Run tests for a specific package
go test -count=1 ./internal/cron/
go test -count=1 ./internal/agent/

# Run tests with verbose output
go test -count=1 -v ./...
```

### Run go vet

`go vet` catches common mistakes like unreachable code, misused format strings, and similar issues:

```sh
make vet
```

### Run golangci-lint

The project uses [golangci-lint](https://golangci-lint.run/) to enforce code quality. The expected local version is the Makefile value:

```sh
make lint-version
```

Install that pinned version if you haven't already:

```sh
make install-lint
golangci-lint --version
```

```sh
# Lint all packages
make lint

# Lint a specific package
golangci-lint run ./internal/agent/...

# Auto-fix some issues
golangci-lint run --fix
```

`make lint` sets `GOLANGCI_LINT_CACHE` to `/tmp/picobot-golangci-cache` by default so lint can run in read-only home environments. Override `GOLANGCI_LINT`, `GOLANGCI_LINT_VERSION`, or `GOLANGCI_LINT_CACHE` only when needed:

```sh
GOLANGCI_LINT=/custom/path/golangci-lint make lint
GOLANGCI_LINT_CACHE=/tmp/other-cache make lint
```

### Validation scope

Use focused package tests while editing, then broaden according to risk:

| Command | When to use |
|---------|-------------|
| `go test -count=1 ./path/to/package` | Fast focused check while editing one package |
| `make test` | Full normal test surface |
| `make test-lite` | Lite build-tag compatibility |
| `make test-build-tags` | Build-tag contract check: full WhatsApp implementation, lite WhatsApp stub, and both binary build modes under `/tmp/picobot-build-tags/` using `/tmp/picobot-go-cache/` by default |
| `make test-scripts` | Shell-script regression checks plus env example, current-doc link, and tagged shell snippet syntax checks |
| `make test-race` | Targeted race detector check for the small concurrency-heavy package set |
| `make coverage` | Optional local coverage profile; writes `coverage.out` and `coverage.func.txt` under `/tmp/picobot-coverage/` by default and does not enforce thresholds |
| `make docker-smoke` | Optional Docker packaging smoke; builds a local image and runs `picobot --help` without publishing |
| `make vet` | Static Go vet checks |
| `make lint` | Configured golangci-lint checks |
| `make verify` | Full local gate before handing off non-trivial work |

Some tests use `httptest` loopback listeners. If sandboxed tests fail because local bind is denied, rerun the same command with the required permission rather than changing code around the sandbox.

Docs can opt shell examples into syntax validation with a fenced code info string containing `picobot-check:shell-syntax`, for example `sh picobot-check:shell-syntax`. The checker runs `sh -n` only; it never executes the snippet.

### Test fixture naming

Use fixture names that make the domain and scenario clear at the call site. Prefer `write<Domain><Scenario>Fixture(s)` helpers, `<domain><Scenario>Fixture(s)` struct types, and local variables such as `telegramOwnerFixture` or `mailboxBootstrapFixtures` instead of a bare `fixture` or `fixtures` when the test touches mission store records, provider records, or more than one object type.

Durable golden files under `testdata/` should use `<surface>_<scenario>.golden.<ext>` so failures point back to the command or read model being checked.

## Versioning

For release gates, artifact builds, checksum capture, and rollback readiness,
use [RELEASE_CHECKLIST.md](./RELEASE_CHECKLIST.md).

The version string and build metadata defaults are defined in `cmd/picobot/main.go`:

```go
var (
	version     = "x.x.x"
	buildCommit = "unknown"
	buildDate   = "unknown"
)
```

Update `version` before building a new release. Release builds can stamp all
three values with linker flags:

```sh
PICOBOT_VERSION=1.0.1
PICOBOT_COMMIT="$(git rev-parse --short HEAD)"
PICOBOT_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
go build -ldflags="-s -w -X main.version=${PICOBOT_VERSION} -X main.buildCommit=${PICOBOT_COMMIT} -X main.buildDate=${PICOBOT_DATE}" -o picobot ./cmd/picobot
./picobot version
```

If the metadata linker flags are absent, `picobot version` reports
`commit: unknown` and `date: unknown`.

## Building for Different Platforms

### Quick builds with Make

The project ships a `Makefile` that cross-compiles all supported platforms in one command:

```sh
# Build all targets — full and lite variants for Linux amd64/arm64 and macOS arm64
make build

# Build individual targets
make linux_amd64        # full build, Linux x86-64
make linux_arm64        # full build, Linux ARM64
make mac_arm64          # full build, macOS Apple Silicon
make linux_amd64_lite   # lite build, Linux x86-64
make linux_arm64_lite   # lite build, Linux ARM64
make mac_arm64_lite     # lite build, macOS Apple Silicon

# Remove all built binaries
make clean
```

Output files are named `picobot_<os>_<arch>[_lite]` and dropped in the project root.

### Manual cross-compilation

If you prefer to invoke `go build` directly:

```sh
# Linux AMD64 (most VPS / servers)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_linux_amd64 ./cmd/picobot

# Linux ARM64 (Raspberry Pi, ARM servers)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_linux_arm64 ./cmd/picobot

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_mac_arm64 ./cmd/picobot

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot.exe ./cmd/picobot
```

**What the flags do:**
- `CGO_ENABLED=0` → pure static binary, no libc dependency
- `-ldflags="-s -w"` → strip debug symbols (~22 MB → ~9 MB for the lite build)

### Full vs Lite builds

Picobot ships in two variants controlled by the `lite` Go build tag:

| Variant | Tag | Binary size | Future heavy packages |
|---------|-----|-------------|----------------------|
| **Full** (default) | *(none)* | ~22 MB | All features |
| **Lite** | `-tags lite` | ~9 MB | ❌ WhatsApp not included |

**Why "Lite" exists:**

Some optional features — starting with WhatsApp via [whatsmeow](https://github.com/tulir/whatsmeow) + [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — pull in large dependencies that add ~13 MB to the binary. We know there are some users running Picobot on a standard server or desktop never need those features and shouldn't have to pay the size cost.

The lite build is aimed at resource-constrained environments: IoT devices, cheap VPS with limited storage, or any deployment where a ~9 MB static binary is strongly preferred over a ~22 MB one. It includes every core feature (agent loop, Telegram, Discord, Slack, memory, skills, cron, heartbeat) but omits packages gated behind the `!lite` build tag.

As new optional heavy integrations are added to Picobot in the future, they will follow the same pattern — included in the full build by default, excluded from the lite build.

The build-tag contract is executable:

```sh
make test-build-tags
```

This target verifies that the full build uses the real WhatsApp implementation,
the lite build uses the WhatsApp stub, and both command binaries compile into
`/tmp/picobot-build-tags/`. It sets `GOCACHE` to `/tmp/picobot-go-cache/` and
uses `-buildvcs=false` by default so fresh smoke binaries work in sandboxes where
the home-directory Go cache is read-only.

```sh
# Full build — all features including WhatsApp (default)
go build ./cmd/picobot

# Lite build — no WhatsApp or other heavy optional packages
go build -tags lite ./cmd/picobot
```

For cross-compilation, simply add `-tags lite` alongside the existing `GOOS`/`GOARCH` flags, or use `make linux_amd64_lite` etc.

```sh
# Lite, Linux ARM64 (e.g. Raspberry Pi)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lite -o picobot_linux_arm64_lite ./cmd/picobot
```

## Docker Workflow

### Build the image

We use a multi-stage Alpine-based build — keeps the final image around ~33MB:

```sh
docker build -f docker/Dockerfile -t louisho5/picobot:latest .
```

#### Multi-arch builds with BuildKit

Picobot's Dockerfile supports BuildKit/`buildx` so you can push both AMD64 and ARM64 images in a single run:

```sh
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --builder default \
  -t louisho5/picobot:latest .
```

Add `--push` to publish directly to a registry or `--load` to import one architecture into your local Docker engine.

> **Important:** Run this from the **project root**, not from inside `docker/`. The build context needs access to the whole codebase.

### Test it locally

Spin up a container to make sure it works:

```sh
docker run --rm -it \
  -e PICOBOT_MODEL="google/gemini-2.5-flash" \
  -v ./picobot-data:/home/picobot/.picobot \
  louisho5/picobot:latest
```

Put provider credentials and channel tokens in `./picobot-data/config.json`. The runtime loader currently honors only `PICOBOT_MODEL`, `PICOBOT_MAX_TOKENS`, and `PICOBOT_MAX_TOOL_ITERATIONS` as environment overrides.

Check logs:

```sh
docker logs -f picobot
```

### Push to Docker Hub

**Build and push** in one shot:

```sh
go build ./... && \
docker build -f docker/Dockerfile -t louisho5/picobot:latest . && \
docker push louisho5/picobot:latest
```

Docker hub: [hub.docker.com/r/louisho5/picobot](https://hub.docker.com/r/louisho5/picobot).

## Environment Variables

These environment variables configure the Docker container:

| Variable | Description | Required |
|---|---|---|
| `PICOBOT_MODEL` | LLM model to use (e.g. `google/gemini-2.5-flash`) | No |
| `PICOBOT_MAX_TOKENS` | Maximum tokens for LLM responses | No |
| `PICOBOT_MAX_TOOL_ITERATIONS` | Maximum tool iterations per request | No |

All provider credentials and channel tokens must come from the mounted Picobot config file.

## Extending Picobot

### Adding a new tool

Let's say you want to add a `database` tool that queries PostgreSQL:

1. **Create the file:**
   ```sh
   touch internal/agent/tools/database.go
   ```

2. **Implement the `Tool` interface:**
   ```go
   package tools
   
   import "context"
   
   type DatabaseTool struct{}
   
   func NewDatabaseTool() *DatabaseTool { return &DatabaseTool{} }
   
   func (t *DatabaseTool) Name() string { return "database" }
   func (t *DatabaseTool) Description() string { 
       return "Query PostgreSQL database"
   }
   func (t *DatabaseTool) Parameters() map[string]interface{} {
       // return JSON Schema for arguments
   }
   func (t *DatabaseTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
       // your implementation here
   }
   ```

3. **Register it in `internal/agent/loop.go`:**
   ```go
   reg.Register(tools.NewDatabaseTool())
   ```

4. **Test it:**
   ```sh
   go test ./internal/agent/tools/
   ```

That's it. The agent loop will automatically expose it to the LLM and route tool calls to your implementation.

### Connecting MCP servers (no code needed)

Picobot has a built-in MCP client that connects to any MCP-compliant server at startup. No code changes are needed — just add an entry to `mcpServers` in `~/.picobot/config.json`:

```json
"mcpServers": {
  "via-npx": {
    "command": "npx",
    "args": ["-y", "@some/mcp-server"]
  }
}
```

Picobot supports two transports:

- **Stdio** — spawns the server as a subprocess (`command` + `args`). Works with `npx`, `uvx`, plain binaries, and `docker run --rm -i <image>`.
- **HTTP** — POST to a remote endpoint (`url` + optional `headers`).

Each tool the server declares is registered as `mcp_{server}_{tool}` and is immediately visible to the agent. The MCP client lives in `internal/mcp/client.go`; the tool bridge is `internal/agent/tools/mcp.go`.

### Adding a new LLM provider

Want to add support for Anthropic, Cohere, or a custom provider?

1. **Create the provider file:**
   ```sh
   touch internal/providers/anthropic.go
   ```

2. **Implement the `LLMProvider` interface from `internal/providers/provider.go`:**
   ```go
   type LLMProvider interface {
       Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (ChatResponse, error)
       GetDefaultModel() string
   }
   ```

3. **Wire it up in the config schema:**
   - Add config fields in `internal/config/schema.go`
   - Update the factory logic in `internal/providers/factory.go`

4. **Test it:**
   ```sh
   go test ./internal/providers/
   ```

## Troubleshooting

### Build fails with weird errors

Try cleaning and re-downloading deps:

```sh
go clean -cache
go mod tidy
go build ./...
```
