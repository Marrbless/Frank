# Post-V4 Garbage Day Matrix

Controller branch: `frank-garbage-day-post-v4-kickoff`

This matrix tracks post-V4 cleanup candidates. It is a maintenance controller, not product scope and not authorization for destructive actions.

## Status Counts

- DONE: 3
- PARTIAL: 0
- MISSING: 3
- BLOCKED: 0

## Matrix

| requirement_id | cleanup_target | current_evidence | status | gap_type | smallest_next_slice | suggested_tests | risk_if_skipped | can_implement_without_new_human_policy | last_slice_attempted | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GC4-000 | Post-V4 Garbage Day kickoff controller exists. | `POST_V4_GARBAGE_DAY_KICKOFF.md` and this matrix establish the post-V4 cleanup controller from the `frank-v4-full-spec-complete` tag. | DONE | docs | None. | `git diff --check`; full Go suite optional because docs-only. | Cleanup could restart as ad hoc broad refactors. | yes | kickoff | No destructive cleanup or behavior change. |
| GC4-001 | `internal/missioncontrol/status.go` V4 autonomy/status read-model cluster is split from the general status accumulator. | `status_autonomy.go` now owns autonomy identity types, loaders, record adapters, wrapper, and last-error helpers; `status.go` no longer carries that V4 cluster. | DONE | structure | None. | Focused autonomy identity/V4 summary tests; full missioncontrol; full suite. | V4 status growth remains concentrated and harder to review. | yes | GC4-TREAT-001 | Same-package mechanical split only; no JSON, state, validation, or read-model behavior changes. |
| GC4-002 | `internal/agent/loop_processdirect_test.go` process-direct command tests are split by command family. | `loop_processdirect_inspect_test.go` now owns the five `INSPECT` command tests; the process-direct omnibus dropped to 10873 lines while shared fixtures stayed in place. | DONE | tests | None. | Focused inspect command tests; full agent; full suite. | Largest test file remains high-friction for review and merge. | yes | GC4-TREAT-002 | Same-package test-only split; test names, command strings, assertions, and behavior preserved. |
| GC4-003 | `internal/agent/tools/taskstate.go` protected runtime/control clusters have smaller files. | File is 4780 lines and remains the central protected state surface. | MISSING | structure | Reassess current clusters after V4 before selecting a same-package extraction; do not start with persistence-core without fresh evidence. | Focused TaskState tests for selected seam; full agent/tools; full suite. | Protected state changes remain harder to audit. | yes | none | Higher risk than GC4-001 and GC4-002. |
| GC4-004 | `cmd/picobot/main_runtime_bootstrap_test.go` is split by runtime bootstrap subfamily. | File is 6555 lines, already dedicated but still very large. | MISSING | tests | Split one bootstrap subfamily into a same-package test file with no assertion changes. | Focused `go test -count=1 ./cmd/picobot -run '<moved family pattern>'`; full cmd package; full suite. | CLI runtime bootstrap reviews stay unnecessarily noisy. | yes | none | Test-only, but fixture coupling must be checked. |
| GC4-005 | Garbage Day docs distinguish retained evidence from pruneable scratch. | `docs/maintenance` now contains 287 V4 maintenance docs plus prior garbage-day artifacts. | MISSING | docs | Add a retention/prune policy doc for maintenance artifacts without deleting files. | `git diff --check`. | Future cleanup may delete useful audit history or keep unbounded scratch forever. | yes | none | Do not delete V4 evidence in this lane. |
