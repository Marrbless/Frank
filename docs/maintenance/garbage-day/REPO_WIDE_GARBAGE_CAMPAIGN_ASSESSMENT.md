# Repo-Wide Garbage Campaign Assessment

Date: 2026-04-28

This is an assessment artifact only. It does not authorize deletion, behavior changes, schema changes, dependency changes, or broad rewrites.

## Facts

- Repo: `/mnt/d/pbot/picobot`
- Branch: `frank-garbage-day-gc4-005-maintenance-artifact-retention`
- HEAD: `a5f174d`
- Tag at HEAD: `frank-garbage-day-gc4-004-bootstrap-store-root-split`
- Worktree: dirty with the uncommitted docs-only `GC4-005` slice.
- Go: `go version go1.26.1 linux/amd64`
- Packages from `go list ./...`: `15`
- Tracked files: `765`
- Tracked Go files: `341`
- Tracked test files: `178`
- Current file count excluding `.git`: `811`
- Go LOC in `cmd` and `internal`: `169119`
- Test LOC in `cmd` and `internal`: `103976`
- Discoverable tests from `func Test`: `1861`
- Module count from `go list -m all`: `73`
- Maintenance markdown files: `385`

## Assumptions

- The current dirty docs are user-accepted local work from `GC4-005`; this assessment does not stage, commit, rewrite, or revert them.
- Runtime artifacts and ignored local data may be inventoried, but deletion requires explicit human approval.
- The first campaign step should prefer assessment, test-only splits, or same-package mechanical moves over behavior work.
- Protected runtime, approval, treasury, hot-update, and phone capability paths require focused tests before any treatment.

## Assessment Plan

1. Record repo, branch, toolchain, worktree, package, and file-count facts.
2. Measure largest code and test concentrations.
3. Identify ignored local artifacts and maintenance-document volume.
4. Search for cleanup-risk signals: generic tool schemas, direct writes, TODOs, and secret/log surfaces.
5. Run deterministic baseline validation.
6. Recommend bounded treatment candidates with smallest next slices.

## Execution Evidence

### Validation

- `/usr/local/go/bin/go test -count=1 ./...` passed.
  - `cmd/picobot`: `15.759s`
  - `internal/agent`: `11.158s`
  - `internal/agent/tools`: `18.708s`
  - `internal/missioncontrol`: `21.206s`
  - all other packages passed or had no tests
- `/usr/local/go/bin/go vet ./...` passed.

### Largest Go Files

| Lines | File |
| ---: | --- |
| 10873 | `internal/agent/loop_processdirect_test.go` |
| 7307 | `internal/agent/tools/taskstate_test.go` |
| 6510 | `cmd/picobot/main_runtime_bootstrap_test.go` |
| 4726 | `internal/agent/tools/taskstate.go` |
| 3879 | `internal/missioncontrol/status.go` |
| 3617 | `internal/missioncontrol/hot_update_gate_registry_test.go` |
| 3454 | `internal/missioncontrol/treasury_registry_test.go` |
| 3353 | `internal/agent/tools/taskstate_status_test.go` |
| 2994 | `internal/agent/tools/frank_zoho_send_email_test.go` |
| 2333 | `internal/agent/loop.go` |
| 2070 | `cmd/picobot/main_mission_inspect_test.go` |
| 2033 | `internal/missioncontrol/status_test.go` |
| 1956 | `internal/missioncontrol/runtime_test.go` |
| 1955 | `internal/agent/loop_checkin_test.go` |
| 1939 | `cmd/picobot/main.go` |

### Package Concentration

| Path | Go files | Approx LOC |
| --- | ---: | ---: |
| `internal/missioncontrol` | 234 | 103742 |
| `internal/agent/tools` | 37 | 26773 |
| `internal/agent` | 16 | 17433 |
| `cmd/picobot` | 12 | 14355 |
| `internal/channels` | 11 | 2612 |

`internal/missioncontrol` is now the dominant structural surface. That is not automatically bad: much of it is deliberately split into registries and tests. The risk is that read-model/status and lifecycle test files still collect unrelated families.

### Ignored Local Artifacts

`git status --ignored` shows local ignored runtime/build artifacts:

- `picobot`: `33M`, ignored by `/picobot`
- `internal/agent/memory/*.md`: `38` files, `336K`
- `internal/agent/sessions/`: `2` files
- `missions/`: `1` file
- `.codex`: ignored, currently `0`

These are not tracked-file problems, but they are repo-local operator state. They should not be deleted by a code cleanup slice without an explicit approval step.

### Cleanup-Risk Signals

- TODO/FIXME/HACK/XXX count is low: only `internal/mcp/client_test.go` matched, with `2` hits.
- Non-test direct writes remain in runtime-adjacent code:
  - `internal/config/onboard.go`: `4`
  - phone capability source initializers: `camera`, `contacts`, `location`, `microphone`, `sms_phone`, `bluetooth_nfc`, `broad_app_control`
  - `internal/agent/tools/exec.go`: `1`
- Generic `map[string]interface{}` / `interface{}` is widespread by design in tool schemas, provider payloads, MCP, and tests. This is not a good broad cleanup target without a typed API boundary decision.
- Docs and examples use placeholder token strings; current evidence did not find live secret-looking docs outside test fixtures, but this was a grep-based screen, not a secret scanner.
- `picobot` is an ignored local binary, not tracked. It is a local cleanup candidate only.

## Candidate Treatment Matrix

| ID | Candidate | Evidence | Smallest next slice | Validation | Risk |
| --- | --- | --- | --- | --- | --- |
| GC5-000 | Checkpoint hygiene before new treatment. | Worktree is dirty with uncommitted GC4-005 docs. | Commit or explicitly leave GC4-005 uncommitted before starting code cleanup. | `git status --short --branch`; `git diff --check` | Mixing finished docs work with new code cleanup makes review and rollback noisy. |
| GC5-001 | Local ignored artifact inventory and prune decision. | `picobot` is `33M`; runtime memory/session/mission files live in the repo tree but are ignored. | Add a no-delete inventory note or request explicit approval for exact local deletions. | `git status --ignored`; optional `du -sh` | Accidental deletion can remove operator-local state; ignoring it forever keeps confusing local noise. |
| GC5-002 | Split another `internal/agent/loop_processdirect_test.go` command family. | File remains `10873` lines; hot-update and rollback command families dominate the middle of the file. | Move one contiguous family, such as rollback apply command tests, into a same-package test file. | Focused `go test ./internal/agent -run 'TestProcessDirectRollbackApply'`; full agent; full suite | Broad moves can break shared fixture assumptions or hide command-regex behavior changes. |
| GC5-003 | Split TaskState capability activation tests. | `taskstate_test.go` remains `7307` lines; capability activation tests are contiguous near the end. | Move capability activation tests into `taskstate_capability_exposure_test.go` or equivalent same-package file. | Focused capability tests; `go test ./internal/agent/tools`; full suite | Test-only move is low risk, but protected capability gates need exact assertions preserved. |
| GC5-004 | Split `cmd/picobot/main_runtime_bootstrap_test.go` by remaining subfamily. | File remains `6510` lines after store-root split; set-step, bootstrap, control-file, and durable runtime families coexist. | Move mission set-step/control-file tests into a same-package file. | Focused `go test ./cmd/picobot -run 'TestMissionSetStep|TestApplyMissionStepControl|TestWatchMissionStepControl'`; full cmd; full suite | CLI fixtures are shared and easy to disturb if helper placement changes. |
| GC5-005 | Split `internal/missioncontrol/status.go` read-model loaders further. | `status.go` remains `3879` lines after autonomy split; many V4 identity/status loaders are still clustered. | Move one cohesive identity/read-model family into a same-package file. | Focused status identity tests; `go test ./internal/missioncontrol`; full suite | Status JSON/read-model fields are operator-facing and must not drift. |
| GC5-006 | Assess repeated phone capability source initializers. | Similar source initializers exist across camera, contacts, location, microphone, SMS phone, Bluetooth/NFC, and broad app control. | Read-only design assessment first; no abstraction until duplication and behavior invariants are mapped. | Existing capability package tests | A premature helper could flatten meaningful per-capability policy differences. |
| GC5-007 | Tool schema typing boundary. | Tools still expose `map[string]interface{}` schemas and args. | Defer until a typed tool-schema boundary is selected; do not do broad string/map churn. | Tool registry tests; provider tests; full suite | High API-surface risk for small cleanup gain. |
| GC5-008 | Direct-write audit for runtime-adjacent non-test files. | Non-test `os.WriteFile` remains in onboarding, capability source creation, and exec project setup. | Audit whether each write is runtime state, config bootstrap, or safe local seed output before changing code. | Targeted tests per area | Replacing writes blindly can alter permissions, bootstrap behavior, or fixture expectations. |

## Recommended First Move

Do not start with behavior cleanup. The safest next move is:

1. Resolve `GC5-000` as a process checkpoint.
2. Then choose either `GC5-003` or `GC5-004` as the first code-adjacent slice because both are test-only, same-package, and bounded.

`GC5-003` is the best first cleanup candidate if the goal is reducing protected runtime review friction. `GC5-001` is the best first housekeeping candidate if the goal is removing local clutter, but it requires explicit deletion approval before any prune action.

## Risks

- The full suite is green, so cleanup should not be justified as fixing a broken repo.
- Large protected tests are valuable executable specs; splitting them is useful only if names, assertions, helper behavior, and package visibility are preserved.
- `internal/missioncontrol` is large but already organized into many narrow registries; broad refactors there are higher risk than test-file splits.
- Local ignored data may contain operator state. Inventory is safe; deletion is not safe without exact approval.
- Secret scanning here was grep-based and incomplete; do not treat it as a security audit.
