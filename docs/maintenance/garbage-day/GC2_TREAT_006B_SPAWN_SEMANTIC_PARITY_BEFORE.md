# GC2-TREAT-006B Spawn Semantic Parity Before

Date: 2026-04-20

## Live checkpoint

- Branch: `frank-v3-foundation`
- HEAD: `a92d0f01cee86f8dce2a73a0926ae37cf6b7df82`
- Tags at HEAD:
  - none
- Ahead/behind `upstream/main`: `374 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result:
  - passed

## Exact remaining spawn semantic drift in scope

- `internal/missioncontrol/step_validation.go:554`
  - `hasDiscussionSideEffects(...)` still treats `spawn` as an effectful tool alongside currently exposed tools such as `exec`, `message`, `write_memory`, `cron`, `create_skill`, and `delete_skill`
- `internal/agent/loop_tool_test.go:188-189`
  - existing parity test already asserts runtime definitions must not expose `spawn`
  - no change planned unless semantic-parity clarification becomes necessary

## Planned files

- `internal/missioncontrol/step_validation.go`
  - remove `spawn` from semantic side-effect classification
- `internal/missioncontrol/step_validation_test.go`
  - add focused tests proving `spawn` does not count as available/effectful for the changed semantic path
- `docs/maintenance/garbage-day/GC2_TREAT_006B_SPAWN_SEMANTIC_PARITY_AFTER.md`

## Non-goals

- Do not add runtime `spawn` support
- Do not modify `internal/agent/loop.go`
- Do not change public/operator-facing docs again
- Do not perform broad validation cleanup beyond the exact `spawn` semantic drift in scope
- Do not implement V4
- Do not add dependencies
- Do not commit
