# Frank v1.0 Release Notes

Release tag: `v1.0.0`

## Purpose

Frank v1.0 is the first private release intended to be cloned from `Marrbless/Frank` onto the Android phone runtime and other trusted devices.

This release keeps the lightweight Picobot CLI/daemon foundation while carrying the Frank mission-control runtime surface:

- long-running gateway mode,
- owner-controlled channel integrations,
- memory and skills,
- guarded tool execution,
- durable mission store,
- operator status snapshots,
- approval/audit records,
- capability records,
- campaign/Zoho workflow records,
- V4-oriented hot-update, promotion, rollback, and runtime-pack ledgers/read models.

## Release Target

Recommended first deployment:

- Samsung Galaxy S21 Ultra,
- Android + Termux,
- lite build,
- no SIM,
- Tailscale always-on VPN,
- owner-allowlisted Telegram or Slack control channel,
- mission-control gateway after basic gateway is proven.

See [ANDROID_PHONE_DEPLOYMENT.md](./ANDROID_PHONE_DEPLOYMENT.md).

## What Is Stable Enough For v1.0

- Build and test surface is green on the release workstation.
- The repo-wide garbage campaign matrix is closed.
- Mission-control current implementation truth is documented in [CANONICAL_RUNTIME_TRUTH.md](./CANONICAL_RUNTIME_TRUTH.md).
- Runtime setup and config are documented in [HOW_TO_START.md](./HOW_TO_START.md) and [CONFIG.md](./CONFIG.md).
- Phone deployment is documented in [ANDROID_PHONE_DEPLOYMENT.md](./ANDROID_PHONE_DEPLOYMENT.md).

## Current Runtime Truth

Current implemented runtime truth is the Frank V3-style Picobot gateway plus mission-control surface.

The V4 specs and hot-update runbook describe implemented record/control surfaces plus future product direction. Do not treat every V4 ambition as fully automatic end-to-end behavior without checking current code and tests.

## Validation For Release

Use the current [release checklist](./RELEASE_CHECKLIST.md) for pre-release
gates, metadata stamping, artifact checksums, and rollback readiness.

Release validation should include:

```sh
git diff --check
go test -count=1 ./...
make test-build-tags
go vet ./...
go build -tags lite -ldflags="-s -w" -o /tmp/picobot-v1-lite ./cmd/picobot
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -tags lite -ldflags="-s -w" -o /tmp/picobot-v1-android-arm64-lite ./cmd/picobot
```

## Clone And Build

On a trusted machine:

```sh
gh repo clone Marrbless/Frank
cd Frank
PICOBOT_VERSION=1.0.1
PICOBOT_COMMIT="$(git rev-parse --short HEAD)"
PICOBOT_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
go build -tags lite -ldflags="-s -w -X main.version=${PICOBOT_VERSION} -X main.buildCommit=${PICOBOT_COMMIT} -X main.buildDate=${PICOBOT_DATE}" -o picobot ./cmd/picobot
./picobot version
```

On Termux, use the same commands after `pkg install git gh golang tmux`.

## Upgrade Notes

- Config and local runtime state live under `~/.picobot`.
- `~/.picobot/config.json` contains secrets in plaintext.
- Back up `~/.picobot` before replacing a long-running phone installation.
- Prefer `git pull --ff-only` for phone updates.
