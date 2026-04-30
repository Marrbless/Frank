# Total Repo 10x Assessment

Date: 2026-04-30

This is an assessment and execution plan. It does not authorize broad rewrites, destructive cleanup, runtime migrations, schema changes, or dependency changes.

## Facts

- Repo: `/mnt/d/pbot/picobot`
- Branch: `main`
- HEAD: `f6d3cb1`
- Worktree at assessment time: clean against `origin/main`
- Module: `github.com/local/picobot`
- Go version declared by `go.mod`: `1.26`
- Package count from `go list ./...`: `15`
- Tracked files: `794`
- Tracked Go files: `351`
- Tracked test files: `184`
- Top-level tracked-file distribution: `docs=421`, `internal=336`, `cmd=15`
- Full Go LOC from `wc -l` over `*.go`: `170852`
- Test LOC from `wc -l` over `*_test.go`: `104686`

## Current Shape

The repo is a Go agent runtime with two overlapping identities:

- **Picobot**: the public lightweight agent frame in `README.md`.
- **Frank**: the governed private operator/runtime frame in the Frank specs, mission-control code, phone deployment docs, hot-update runbook, and maintenance artifacts.

The code is broadly healthy: it compiles, has a large executable-spec test suite, and uses append-only/atomic store patterns in mission-control. The biggest issue is not lack of tests. The biggest issue is that key concepts have grown into very large modules whose interfaces force maintainers to know too much implementation detail.

## Architectural Hotspots

| Area | Evidence | Assessment |
| --- | --- | --- |
| Mission-control domain module | `internal/missioncontrol` has `236` Go files by local count and dominates repo code volume. Largest files include `status.go`, `hot_update_gate_registry.go`, `treasury_registry.go`, and `runtime.go`. | Strong domain locality, but too many lifecycle families share one package and repeated status/registry/read-model patterns. |
| Task state | `internal/agent/tools/taskstate.go` is `4726` lines with runtime activation, treasury, Zoho, approvals, hot-update, rollback, canary, persistence hooks, and audit emission. | This module is too shallow at the interface: callers and maintainers pay nearly the full implementation complexity. |
| Agent loop | `internal/agent/loop.go` is `2365` lines and mixes inbound routing, direct processing, sessions, memory, tool execution, periodic notifications, provider errors, and operator command parsing. | Operator commands and runtime notification accounting need deeper modules behind smaller interfaces. |
| CLI/bootstrap | `cmd/picobot/main.go` is `1940` lines and startup/bootstrap/watch logic is spread across the CLI root. | CLI root is still doing too much orchestration. Startup safety is hard to review locally. |
| Status/read-model surface | `internal/missioncontrol/status.go` has repeated `LoadOperator...IdentityStatus` and `With...Identity` families. | High-leverage candidate for table-driven helpers or generated code, but only after behavior is locked by fixtures. |
| Test files | Largest test files include `internal/agent/loop_processdirect_test.go` (`9867` lines), `internal/agent/tools/taskstate_test.go` (`6013`), `cmd/picobot/main_runtime_bootstrap_test.go` (`4407`). | Tests are valuable executable specs, but their size slows review and makes focused changes harder. |
| Docs/process | `docs/maintenance` is hundreds of retained artifacts. `docs/CANONICAL_RUNTIME_TRUTH.md` still names older branch truth while live branch is `main`. | Provenance is strong, current routing is weak. Operators and agents need one current truth entry point. |

## Validation Surface

Observed deterministic gates:

- `go test -count=1 ./...` passed outside the sandbox.
- Subagent validation reported `go vet ./...` passed.
- Subagent validation reported `go test -count=1 -tags lite ./...` passed.
- CI runs `golangci-lint`, `go vet ./...`, and `go test ./...`.

Validation gaps:

- The `Makefile` has build targets only; no `test`, `vet`, `lint`, or `verify` target.
- CI does not test `-tags lite`, even though lite is a supported shipped build surface.
- Local sandboxed test runs can false-fail because `httptest.NewServer` cannot bind loopback ports there.
- `golangci-lint` is configured narrowly and was not installed locally during this assessment.
- There is no coverage profile or explicit "fast gate vs full gate" split.

## Highest-Leverage Risks

1. **Fail-open channel onboarding.** Empty allowlists are documented and implemented as open mode for multiple channels. WhatsApp setup enables the channel without collecting an allowlist. For an unattended agent with exec/filesystem/web/message tools, this is the sharpest safety footgun.
2. **Phone update cutover lacks rollback proof.** The Termux update script pulls/builds/replaces/restarts, but current docs and script do not prove post-start health, keep a previous binary as a rollback candidate, or run mission status/assert checks before declaring success.
3. **Current-truth docs are stale.** The repo retains excellent historical evidence, but the current branch/runtime truth is not cleanly routed from a single maintained file.
4. **Protected runtime logic is too concentrated.** Mission-control, TaskState, AgentLoop, and CLI bootstrap are all valid but oversized. This raises review cost and regression risk.
5. **Supported build surface exceeds CI gate.** Lite builds and cross-builds are documented/released but not part of the main CI validation job.

## Assumptions

- "10x" means improving engineering leverage, operator safety, review speed, and change confidence, not rewriting the product from scratch.
- The Frank V4/hot-update lifecycle is treated as current code on `main`, while older branch-specific maintenance artifacts are historical unless proven otherwise by live code.
- The best improvements are staged, test-first, and mostly additive before any consolidation or deletion.

## Plan

### Phase 0: Make Current Truth Cheap To Find

Goal: every human or agent can discover current runtime truth, validation gates, and historical-vs-current docs in under one minute.

1. Add a repo-root or docs-root `START_HERE_OPERATOR.md`.
2. Update `docs/CANONICAL_RUNTIME_TRUTH.md` so it names `main` as current branch truth and marks older branch facts as provenance.
3. Add a concise maintenance index that links only the current controller, current runbooks, and archived evidence policy.
4. Add a short `CONTEXT.md` domain glossary for stable terms: Picobot, Frank, live runtime plane, improvement workspace, hot-update gate, runtime pack, mission store, operator channel, owner approval, canary.

Expected 10x effect: drastically less context loading and fewer stale-doc mistakes.

Validation:

- `git diff --check`
- Link/path grep for renamed or newly routed docs

### Phase 1: Install A Real Verify Surface

Goal: one command expresses the local and CI quality gate.

1. Add `make test`, `make test-lite`, `make vet`, `make lint`, and `make verify`.
2. Update CI to run `go test -count=1 -tags lite ./...`.
3. Document sandbox caveat: tests using `httptest` require loopback bind permission.
4. Decide whether `golangci-lint` version should be pinned in docs, CI, or a helper script.

Expected 10x effect: every change has a deterministic proof path; release-only breakage in lite builds is caught earlier.

Validation:

- `make verify`
- CI pass on both normal and lite test surfaces

### Phase 2: Close Operator Safety Footguns

Goal: production deployments fail closed unless the owner explicitly chooses open mode.

1. Require allowlists during interactive channel setup for Telegram, Discord, Slack, and WhatsApp, or require an explicit open-mode acknowledgement.
2. Add WhatsApp allowlist prompting and document LID/JID expectations.
3. Add tests proving blank allowlists either fail setup or require an explicit open flag for production paths.
4. Audit log lines that print user IDs, chat IDs, request IDs, provider error metadata, or message summaries; preserve debuggability while avoiding unnecessary durable leakage.

Expected 10x effect: lower chance of accidental public control of a powerful unattended agent.

Validation:

- Focused channel setup tests
- Channel adapter tests
- Full `go test -count=1 ./...`

### Phase 3: Make Phone Updates Transactional

Goal: a phone update has build proof, smoke proof, health proof, and rollback.

1. Replace the Termux update script with a two-phase updater:
   - fetch/build into a side-by-side candidate path,
   - run smoke checks,
   - preserve previous binary and session metadata,
   - switch only after checks pass,
   - verify restarted process health,
   - provide one-command rollback.
2. Add a phone update runbook with exact commands, expected output, failure states, and rollback triggers.
3. Add script-level tests where practical, or a deterministic dry-run mode if shell testing is too expensive.
4. Tie post-update checks to `mission status` / `mission assert` where a mission store exists.

Expected 10x effect: fewer one-device outages and faster recovery from bad releases.

Validation:

- Script dry-run test
- Build check
- Documented manual phone smoke test until real device automation exists

### Phase 4: Deepen The Operator Command Interface

Goal: remove the long regex chain from `AgentLoop` without changing operator behavior.

1. Extract a same-package operator command parser/router from `internal/agent/loop.go`.
2. Preserve exact command strings, malformed-command behavior, and response text with existing tests.
3. Move one command family at a time: help, status/inspect/set-step, rollback, hot-update, canary/approval.
4. Add table-driven parser tests separate from full `ProcessDirect` integration tests.

Expected 10x effect: new operator commands stop increasing AgentLoop complexity.

Validation:

- Existing `internal/agent` command tests
- New parser unit tests
- Full `go test -count=1 ./internal/agent`

### Phase 5: Split TaskState By Lifecycle Family

Goal: keep TaskState as the public stateful adapter but move implementation families behind narrower internal modules.

1. Preserve `TaskState` as the interface used by tools and AgentLoop.
2. Move implementation-only groups into same-package files first:
   - approvals/runtime control,
   - hot-update gate operations,
   - canary and owner approval path,
   - rollback apply path,
   - Frank Zoho campaign path.
3. After mechanical moves, identify true deep modules with smaller interfaces.
4. Do not change store schemas or record semantics in the same slices.

Expected 10x effect: protected runtime changes become reviewable without reading a 4700-line file.

Validation:

- Focused `internal/agent/tools` tests per moved family
- Full `go test -count=1 ./internal/agent/tools`
- Full suite after each family

### Phase 6: Consolidate Mission-Control Read Models

Goal: make status/read-model additions cheap and drift-resistant.

1. Freeze current status JSON/operator text fixtures for the most important identity surfaces.
2. Introduce shared helpers for repeated load/validate/error/status patterns.
3. Consider code generation only if helper consolidation still leaves too much repetition.
4. Keep generated or table-driven output readable; operator status is an audit surface, not just internal plumbing.

Expected 10x effect: new V4/V5 lifecycle records require less boilerplate and less copy/paste.

Validation:

- `internal/missioncontrol/status_*_identity_test.go`
- `status_v4_summary_test.go`
- Runbook consistency tests

### Phase 7: Define Store/Durability Boundaries Explicitly

Goal: every durable write uses the right write path by category.

1. Document categories: mission store record, config bootstrap, workspace seed file, session/memory, generated artifact, runtime log.
2. Audit remaining direct `os.WriteFile` calls in non-test code.
3. Convert runtime-state writes to atomic helpers where appropriate.
4. Keep bootstrap/config writes separate where permissions or plaintext secret handling differ.

Expected 10x effect: fewer silent durability and permission differences across runtime surfaces.

Validation:

- Existing store/session/memory tests
- Targeted tests for converted write paths

### Phase 8: Reduce Test Friction Without Weakening Specs

Goal: tests stay strong but become easier to navigate and run locally.

1. Continue same-package splits of giant tests by command/lifecycle family.
2. Extract common fixture helpers only when duplication causes drift.
3. Replace sleep-based channel tests with deterministic synchronization where practical.
4. Add a short testing guide naming slow packages and focused package commands.

Expected 10x effect: smaller review diffs, faster focused development, fewer timing flakes.

Validation:

- Focused package tests for every split
- Full suite
- Optional repeated run for timing-sensitive channel tests

## Execution Order

Recommended first 10 slices:

1. `START_HERE_OPERATOR.md` plus `CANONICAL_RUNTIME_TRUTH.md` refresh.
2. `Makefile` verify targets.
3. CI lite-test addition.
4. Channel setup fail-closed design note and tests.
5. WhatsApp allowlist setup implementation.
6. Termux updater dry-run and rollback design.
7. Termux updater two-phase implementation.
8. Agent operator parser extraction for help/status only.
9. TaskState rollback-apply or hot-update family mechanical split.
10. Mission-control status identity helper pilot for one lifecycle family.

This order intentionally starts with routing and gates, then safety, then architecture. Architecture work without gates will be slower and riskier.

## Validation Per Slice

Use the smallest useful gate first, then the full gate before merge:

```sh
go vet ./...
go test -count=1 ./...
go test -count=1 -tags lite ./...
golangci-lint run
```

When running in restricted local sandboxes, set writable cache/temp dirs and run loopback-dependent tests outside the sandbox:

```sh
GOCACHE=/tmp/picobot-gocache GOTMPDIR=/tmp/picobot-gotmp go test -count=1 ./...
```

## Risks

- Large protected tests are valuable. Splitting them mechanically is good; weakening assertions is not.
- Mission-control has strong locality today. Splitting into many packages too early could make the domain harder, not easier.
- Channel fail-closed changes may be behavior-breaking for existing users who rely on open mode. Require explicit migration notes.
- Phone updater rollback needs real-device proof eventually. A dry-run test is not enough to declare phone operations solved.
- Historical maintenance evidence should be archived or routed, not deleted casually.
