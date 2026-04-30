# Release Checklist

Use this checklist before tagging or replacing a long-running phone/runtime
binary. It is non-publishing by default.

## Pre-Release Gates

Run from the repo root on a trusted release workstation:

```sh
git status --short
git diff --check
make vet
make test
make test-lite
make test-build-tags
make test-scripts
make lint
```

If Docker is available and the change affects packaging, also run:

```sh
make docker-smoke
```

If a package uses `httptest` loopback listeners and the local sandbox blocks
binds, rerun the same package command with the needed local permission rather
than changing code around the sandbox.

## Build Metadata

Stamp release binaries with the release version, commit, and UTC build time:

```sh
PICOBOT_VERSION=1.0.1
PICOBOT_COMMIT="$(git rev-parse --short HEAD)"
PICOBOT_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
PICOBOT_LDFLAGS="-s -w -X main.version=${PICOBOT_VERSION} -X main.buildCommit=${PICOBOT_COMMIT} -X main.buildDate=${PICOBOT_DATE}"
```

`picobot version` reports `unknown` for commit/date only when those linker flags
are intentionally omitted.

## Build Artifacts

Build artifacts into `/tmp` first:

```sh
go build -tags lite -ldflags="${PICOBOT_LDFLAGS}" -o /tmp/picobot-${PICOBOT_VERSION}-lite ./cmd/picobot
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -tags lite -ldflags="${PICOBOT_LDFLAGS}" -o /tmp/picobot-${PICOBOT_VERSION}-android-arm64-lite ./cmd/picobot
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="${PICOBOT_LDFLAGS}" -o /tmp/picobot-${PICOBOT_VERSION}-linux-amd64 ./cmd/picobot
```

Smoke-check every built binary that can run on the release workstation:

```sh
/tmp/picobot-${PICOBOT_VERSION}-lite version
/tmp/picobot-${PICOBOT_VERSION}-linux-amd64 version
```

Generate checksums before moving binaries off the release workstation:

```sh
scripts/release-checksums /tmp/picobot-${PICOBOT_VERSION}-*
```

## Rollback Readiness

Before replacing a running phone binary:

- Back up `~/.picobot/config.json` and any mission store root.
- Confirm `scripts/termux/update-and-restart-frank --dry-run` succeeds on the phone when device access is available.
- Confirm the previous binary path is preserved by the updater transcript.
- Keep the manual rollback command available:

```sh
scripts/termux/update-and-restart-frank --rollback
```

Do not publish or replace a long-running runtime until validation output,
artifact paths, checksums, and rollback notes are recorded in the release
receipt or handoff.
