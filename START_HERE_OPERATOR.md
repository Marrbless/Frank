# Start Here Operator

This is the current entry point for operating, validating, or changing this repo.

## Current Truth

- Current branch truth is the live checkout on `main`.
- Live code and shell evidence win over older handoffs, branch-specific maintenance notes, and stale docs.
- Current runtime/control truth is the code under `cmd/picobot/`, `internal/agent/`, `internal/agent/tools/`, and `internal/missioncontrol/`.
- Current operator docs are:
  - [docs/CANONICAL_RUNTIME_TRUTH.md](docs/CANONICAL_RUNTIME_TRUTH.md)
  - [docs/HOW_TO_START.md](docs/HOW_TO_START.md)
  - [docs/CONFIG.md](docs/CONFIG.md)
  - [docs/FRANK_V5_MIGRATION_GUIDE.md](docs/FRANK_V5_MIGRATION_GUIDE.md)
  - [docs/FRANK_V6_LOCAL_MODEL_SETUP_SPEC.md](docs/FRANK_V6_LOCAL_MODEL_SETUP_SPEC.md)
  - [docs/HOT_UPDATE_OPERATOR_RUNBOOK.md](docs/HOT_UPDATE_OPERATOR_RUNBOOK.md)
  - [docs/ANDROID_PHONE_DEPLOYMENT.md](docs/ANDROID_PHONE_DEPLOYMENT.md)

## Fast Orientation

- Domain glossary: [CONTEXT.md](CONTEXT.md)
- Repo assessment and roadmap: [docs/maintenance/TOTAL_REPO_10X_ASSESSMENT.md](docs/maintenance/TOTAL_REPO_10X_ASSESSMENT.md)
- Current maintenance route: [docs/maintenance/CURRENT.md](docs/maintenance/CURRENT.md)
- Historical maintenance evidence retention: [docs/maintenance/garbage-day/MAINTENANCE_ARTIFACT_RETENTION.md](docs/maintenance/garbage-day/MAINTENANCE_ARTIFACT_RETENTION.md)

## Validation Gate

Use the smallest focused package test while editing, then run the full gate before merge:

```sh
go vet ./...
go test -count=1 ./...
go test -count=1 -tags lite ./...
golangci-lint run
```

Some tests use local `httptest` servers. Restricted sandboxes that block loopback binds can false-fail those tests; use an unsandboxed runner for the full gate.

## Historical Docs

Files under `docs/maintenance/` often record branch names, HEADs, and validation evidence from the moment a slice happened. Treat those facts as provenance unless [docs/CANONICAL_RUNTIME_TRUTH.md](docs/CANONICAL_RUNTIME_TRUTH.md) or live code says they are current.
