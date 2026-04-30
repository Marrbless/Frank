# AGENTS.md

This repo is a Go agent runtime with a governed private operator/runtime plane. Keep every change narrow, testable, and compatible with the existing mission-control store unless the human explicitly approves a migration.

## Start Here

- Current operator entry point: [START_HERE_OPERATOR.md](START_HERE_OPERATOR.md)
- Domain glossary: [CONTEXT.md](CONTEXT.md)
- Maintenance route: [docs/maintenance/CURRENT.md](docs/maintenance/CURRENT.md)
- Autonomous backlog: [docs/maintenance/AUTONOMOUS_IMPROVEMENT_MATRIX.md](docs/maintenance/AUTONOMOUS_IMPROVEMENT_MATRIX.md)
- Runtime truth router: [docs/CANONICAL_RUNTIME_TRUTH.md](docs/CANONICAL_RUNTIME_TRUTH.md)

## Work Rules

- Pick one bounded matrix row at a time unless rows are explicitly independent.
- Search code and tests before proposing architecture changes.
- Prefer tests first for behavior changes; preserve existing operator text and durable JSON schemas unless the task says otherwise.
- Do not rewrite broad modules just because they are large. Split mechanically first, then deepen interfaces with tests.
- Stop for human input before destructive actions, schema migrations, new dependencies, real credentials, live phone access, or intentional public behavior breaks.
- Update the relevant task or matrix status only after validation evidence exists.

## Validation

Use focused package tests while editing, then broaden according to risk:

```sh
make test
make test-lite
make vet
make lint
make verify
```

`make verify` is the full local gate. Some Go tests use `httptest` loopback listeners; sandboxed runs may false-fail if local bind is blocked, so rerun those tests with the required permission rather than changing code around the sandbox.

## Risky Areas

- `internal/missioncontrol`: durable records, validation, status/read models, and store invariants.
- `internal/agent/tools/taskstate.go`: runtime state adapter; split by lifecycle family, but keep behavior stable.
- `internal/agent/loop.go`: operator command routing and owner-facing messages; preserve response strings.
- `cmd/picobot`: CLI/bootstrap and production startup guards.
- `internal/channels`: owner-control surfaces; enabled channels should fail closed unless open mode is explicit.
- `scripts/termux`: phone updater; preserve rollback behavior and portable shell compatibility.

## Final Reports

For non-trivial work, report Facts, Assumptions, Plan, Execution, Validation, and Risks. Include exact validation commands and whether they passed, failed, or were blocked.
