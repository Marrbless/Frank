# Frank v1.0.1 Release Notes

Release tag: `v1.0.1`

## Purpose

Frank v1.0.1 fixes phone-runtime heartbeat noise from the default onboarding `HEARTBEAT.md` template.

## Changes

- Heartbeat now extracts explicit tasks from the `## Periodic Tasks` section instead of sending the entire file to the agent.
- The default onboarding heartbeat template is ignored because it contains only comments/example tasks.
- Plain task-only heartbeat files are still supported.
- System-channel final replies are swallowed inside the agent instead of being routed to the hub, preventing `no subscriber for channel "heartbeat"` log noise.

## Phone Update

On the phone:

```sh
cd ~/Frank
git pull --ff-only
PICOBOT_VERSION=1.0.1
PICOBOT_COMMIT="$(git rev-parse --short HEAD)"
PICOBOT_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
go build -tags lite -ldflags="-s -w -X main.version=${PICOBOT_VERSION} -X main.buildCommit=${PICOBOT_COMMIT} -X main.buildDate=${PICOBOT_DATE}" -o picobot ./cmd/picobot
./picobot version
```

Then restart the gateway.

## Validation

Use the current [release checklist](./RELEASE_CHECKLIST.md) for pre-release
gates, metadata stamping, artifact checksums, and rollback readiness.

```sh
go test -count=1 ./internal/heartbeat
go test -count=1 ./internal/agent
go test -count=1 ./...
make test-build-tags
go vet ./...
go build -tags lite -ldflags="-s -w" -o /tmp/picobot-v1.0.1-lite ./cmd/picobot
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -tags lite -ldflags="-s -w" -o /tmp/picobot-v1.0.1-android-arm64-lite ./cmd/picobot
```
