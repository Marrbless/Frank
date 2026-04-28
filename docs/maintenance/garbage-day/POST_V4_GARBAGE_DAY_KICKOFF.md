# Post-V4 Garbage Day Kickoff

Date: 2026-04-28

## Current Checkpoint Facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-garbage-day-post-v4-kickoff`
- Starting HEAD: `a83bc769f5965a75955f378e90874126bc10e6af`
- Starting tag at HEAD: `frank-v4-full-spec-complete`
- Worktree at kickoff: clean
- V4 completion artifact: `docs/maintenance/V4_FULL_SPEC_COMPLETION_FINAL.md`
- V4 matrix: `docs/maintenance/V4_FULL_SPEC_COMPLETION_MATRIX.md`
  - DONE: 44
  - PARTIAL: 0
  - MISSING: 0
  - BLOCKED: 0

## Scope

Garbage Day resumes as repo-health maintenance after V4 completion.

This kickoff does not authorize destructive cleanup, behavior changes, policy-surface mutation, real network calls, external service calls, real phone deployment, or history rewriting.

## Current Structural Inventory

Largest current Go files from `internal` and `cmd`:

| file | lines | note |
| --- | ---: | --- |
| `internal/agent/loop_processdirect_test.go` | 11038 | largest test omnibus; V4 hot-update command coverage and older process-direct command coverage coexist with shared fixtures |
| `internal/agent/tools/taskstate.go` | 4780 | central protected runtime/control state surface; grew after V4-adjacent runtime work |
| `internal/missioncontrol/status.go` | 4543 | status/read-model accumulator; V4 identity surfaces added more coherent but concentrated families |
| `internal/agent/tools/taskstate_test.go` | 7307 | large protected TaskState test omnibus |
| `cmd/picobot/main_runtime_bootstrap_test.go` | 6555 | dedicated but very large runtime bootstrap command family |
| `internal/missioncontrol/hot_update_gate_registry_test.go` | 3617 | large but cohesive hot-update lifecycle tests |
| `internal/missioncontrol/treasury_registry_test.go` | 3454 | protected treasury registry/read-model test surface |
| `cmd/picobot/main.go` | 1939 | prior Garbage Day work reduced this from the earlier top hotspot |

## Post-V4 Treatment Rule

The prior Garbage Day campaign stopped because heavy TaskState cleanup was not justified before a concrete next-phase need. V4 is now complete, but the same treatment discipline still applies:

- prefer test-only or same-package mechanical splits before behavior work,
- preserve public method names, JSON fields, command names, and record formats,
- do not delete V4 evidence or maintenance history,
- do not collapse safety records into ad hoc state,
- do not mix cleanup with product behavior.

## Recommended First Treatment Lane

Recommended first code lane:

- `GC4-TREAT-001`: split the V4 autonomy/status read-model cluster out of `internal/missioncontrol/status.go` into a same-package file.

Why this lane first:

- It is a coherent V4-created concentration.
- It is a same-package move, so no API boundary changes are required.
- It is locked by focused autonomy identity and V4 summary tests.
- It reduces the most obvious post-V4 read-model accumulation without touching hot-update gate behavior, TaskState persistence, approvals, treasury, or loop control.

Validation target for that lane:

- `git diff --check`
- `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`
- `/usr/local/go/bin/go test -count=1 ./...`

## Deferred Higher-Risk Lanes

- Split `internal/agent/loop_processdirect_test.go` by command family.
- Split `internal/agent/tools/taskstate.go` runtime persistence/control internals.
- Split `internal/agent/tools/taskstate_test.go` remaining protected mutation families.
- Split `cmd/picobot/main_runtime_bootstrap_test.go` by bootstrap command subfamily.
- Split treasury test surfaces only after a narrow treasury-specific reason appears.

## Kickoff Decision

Begin post-V4 Garbage Day now with a durable controller and a first low-risk treatment target. Do not begin by deleting files, pruning history, or performing broad rewrites.
