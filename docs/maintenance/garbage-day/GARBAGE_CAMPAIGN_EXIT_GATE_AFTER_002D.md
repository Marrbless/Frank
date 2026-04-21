## Garbage Campaign Exit Gate After 002D

### Current Checkpoint

- Branch: `frank-v3-foundation`
- HEAD: `1816055bc5935e335488a369e54f50d723a9a603`
- Tags at HEAD: `frank-garbage-campaign-002d-main-test-runtime-clean`
- Ahead/behind upstream: `389 ahead / 0 behind`
- Repo green: yes
  - `go test -count=1 ./...` passed

### Completed Structural Slices

#### `main.go` 001A-001D

- `001A` extracted interactive channel login/setup helpers.
  - Removed operator/onboarding prompt noise from the main CLI omnibus without touching runtime truth.
- `001B` extracted memory command builders.
  - Removed a self-contained CLI subtree from `main.go` and clarified that memory commands are not part of mission bootstrap or gateway startup.
- `001C` extracted mission inspect read-model helpers.
  - Removed inspect projection/read-model mass from the main file without touching runtime/bootstrap hooks.
- `001D` extracted mission status/assertion helpers.
  - Removed status projection/assertion helper concentration from `main.go` while preserving command wiring.

#### `main_test.go` 002-002D

- `002` split the memory CLI family.
  - Removed the safest self-contained CLI test seam from the giant omnibus test file.
- `002A` split scheduled-trigger governance tests.
  - Isolated the cron/governance family and its local fixture surface.
- `002B` split mission inspect tests.
  - Moved inspect-specific fixture-heavy tests into their own family file.
- `002C` split mission status/assertion tests.
  - Removed the status/assertion family while keeping shared runtime helpers stable.
- `002D` split the remaining bootstrap/runtime/watcher/operator-control heavyweight family.
  - Completed the `main_test.go` de-omnibus campaign and left `main_test.go` as a small residual surface.

### Structural Risk Removed

- `cmd/picobot/main.go` is no longer carrying every operator-facing helper family in one file.
- `cmd/picobot/main_test.go` is no longer the giant mixed-family omnibus that hid behavioral boundaries and made future movement risky.
- The highest-noise CLI/operator surfaces are now decomposed by real behavioral family rather than arbitrary line ranges.
- Runtime-sensitive `cmd/picobot` test mass is now at least isolated to dedicated files, which makes later runtime-family work more deliberate.

### What Large Files Still Remain

Required remaining large files:

- `internal/agent/tools/taskstate_test.go` — `7346` lines
- `internal/agent/tools/taskstate.go` — `3343` lines
- `internal/missioncontrol/treasury_registry_test.go` — `3454` lines

Other top remaining files:

- `cmd/picobot/main_runtime_bootstrap_test.go` — `6555` lines
- `internal/agent/loop_processdirect_test.go` — `2743` lines
- `internal/agent/tools/frank_zoho_send_email_test.go` — `2728` lines
- `cmd/picobot/main_mission_inspect_test.go` — `2070` lines
- `internal/agent/loop_checkin_test.go` — `1955` lines
- `cmd/picobot/main.go` — `1939` lines
- `internal/missioncontrol/identity_registry_test.go` — `1917` lines
- `internal/missioncontrol/runtime_test.go` — `1898` lines
- `internal/missioncontrol/status_test.go` — `1894` lines
- `internal/agent/loop.go` — `1847` lines

### Option Comparison

#### 1. Stop Here And Declare AI-Slop-Free-Enough For V4 Entry

- Expected value:
  - Creates a clean human decision point.
  - Prevents the garbage campaign from turning into open-ended cleanup.
  - Starts V4 planning from a much healthier repo than the pre-001/pre-002 state.
- Risk:
  - Leaves the single biggest remaining structural concentration in `TaskState`.
  - Leaves a very large dedicated runtime/bootstrap test file in `cmd/picobot`.
  - Risks carrying one more round of agent-hostile concentration into V4 planning.
- Confidence: medium-high
- Why now or not now:
  - Now is plausible because the CLI and top-level operator surfaces are materially cleaner.
  - Not ideal because `TaskState` is still the clearest remaining “AI-sloppy” structural hotspot and it is central, not peripheral.

#### 2. Start A New Structural Campaign Centered On TaskState

- Expected value:
  - Attacks the highest-value remaining concentration in the repo.
  - Reduces risk before V4 planning by clarifying one of the most central agent/runtime state surfaces.
  - Produces a better “stop point” than stopping immediately after CLI/test decomposition.
- Risk:
  - Higher than the finished `main.go` and `main_test.go` slices.
  - `TaskState` is logic-dense and likely has deeper helper, persistence, and status coupling.
  - Easy to widen into broad refactor if not tightly staged.
- Confidence: medium
- Why now or not now:
  - Now is justified because the adjacent safer CLI/test cleanup is largely done.
  - The repo has already harvested the easier structural wins; `TaskState` is the next honest bottleneck.

### Recommendation

- Recommended next direction: start a new structural campaign centered on `TaskState`, but do it assessment-first and slice it narrowly.

Rationale:

- The repo is now clean enough to permit a deliberate V4 planning decision, but not yet clean enough that stopping here is the strongest choice.
- The `main.go` and `main_test.go` campaigns removed a large amount of obvious operator-surface slop. What remains is no longer mainly top-level CLI clutter; it is concentrated state/runtime complexity.
- `internal/agent/tools/taskstate.go` plus `internal/agent/tools/taskstate_test.go` are now the clearest remaining high-signal structural hotspot.
- If the campaign stops here, V4 planning will still inherit the largest central state-management concentration unchanged.

### Exit Gate Answer

- No: the repo is not yet at the strongest “AI-slop-free enough” stop point for V4 entry.
- It is good enough to permit a deliberate human V4 entry decision if needed.
- But the recommended path is one more structural campaign, centered on `TaskState`, before declaring the garbage campaign complete.
