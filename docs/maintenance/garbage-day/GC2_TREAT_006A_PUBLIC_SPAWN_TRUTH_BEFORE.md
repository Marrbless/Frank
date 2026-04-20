# GC2-TREAT-006A Public Spawn Truth Cleanup Before

Date: 2026-04-20

## Live checkpoint

- Branch: `frank-v3-foundation`
- HEAD: `83593cd2752b0f25a22651cadc0e90765dfd3ec2`
- Tags at HEAD:
  - none
- Ahead/behind `upstream/main`: `373 ahead / 0 behind`
- `git status --short --branch`:
  - `## frank-v3-foundation`
- Baseline `go test -count=1 ./...` result:
  - passed

## Exact public/operator-facing spawn references found

### Public docs

- `README.md:114`
  - built-in tools table still advertises:
    - `` `spawn` | Return a stub acknowledgement for a background-subagent request; does not launch one ``

### Onboarding-generated operator-facing surface

- `internal/config/onboard.go:275`
  - generation source for onboarding-created `TOOLS.md` still includes:
    - `### spawn`
    - `Spawn a background subagent process.`

## References checked and not in scope for correction

- `docs/HOW_TO_START.md`
  - no public built-in `spawn` tool reference found
- `docs/CONFIG.md`
  - contains MCP transport wording such as “spawns the process” and “Executable to spawn”
  - this is transport/process wording, not false built-in tool advertising
  - no correction needed in this slice

## Exact files planned

- `README.md`
  - remove the false public built-in `spawn` tool row
- `internal/config/onboard.go`
  - remove the false `spawn` section from the generated/operator-facing `TOOLS.md` source
- `internal/config/onboard_test.go`
  - add or tighten a focused assertion so onboarding-generated `TOOLS.md` no longer advertises `spawn`
- `docs/maintenance/garbage-day/GC2_TREAT_006A_PUBLIC_SPAWN_TRUTH_AFTER.md`

## Exact non-goals

- Do not modify `internal/agent/loop.go`
- Do not modify `internal/missioncontrol/step_validation.go`
- Do not change runtime behavior or tool registration
- Do not implement semantic parity for `spawn`
- Do not implement V4
- Do not broaden into docs reconciliation beyond the false public/operator-facing `spawn` surface
- Do not add dependencies
- Do not commit
