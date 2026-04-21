# Garbage Campaign Structural Decision After 002C

Date: 2026-04-20

## Current checkpoint

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `af9719a1ae5d77375b0577c571900b5c29b8fa02`
- Tags at HEAD:
  - `frank-garbage-campaign-002c-main-test-status-clean`
- Ahead/behind `upstream/main`: `386 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this checkpoint

## Completed structural slices

### `main.go` slices

- `GC2-TREAT-001A` extracted channel login and prompt/setup helpers
- `GC2-TREAT-001B` extracted memory command builders
- `GC2-TREAT-001C` extracted mission inspect read-model helpers
- `GC2-TREAT-001D` extracted mission status/assertion helpers

### `main_test.go` slices

- `GC2-TREAT-002` split memory CLI tests
- `GC2-TREAT-002A` split scheduled-trigger governance tests
- `GC2-TREAT-002B` split mission inspect tests
- `GC2-TREAT-002C` split mission status/assertion tests

## Structural risk removed so far

### What `001A-001D` removed

- `cmd/picobot/main.go` is down to `1939` lines from the `3219` line structural assessment baseline.
- Net reduction from the assessment baseline: `1280` lines, about `40%`.
- The file is no longer dominated by mixed CLI-local helper clusters.
- Operator onboarding, memory CLI, mission inspect read-model, and mission status/assertion helpers are no longer packed directly into the root command file.
- Future edits to those command families no longer land adjacent to every gateway/bootstrap/runtime hook by default.

### What `002-002C` removed

- `cmd/picobot/main_test.go` is down to `6943` lines from the `10997` line assessment baseline.
- Net reduction from the assessment baseline: `4054` lines, about `37%`.
- The safest command-family seams are now separated:
  - memory CLI
  - scheduled-trigger governance
  - mission inspect
  - mission status/assertion
- Review and merge risk is materially lower because those families no longer share one giant omnibus test file.
- The remaining `main_test.go` mass is now much more obviously the heavyweight runtime/control family instead of a general junk drawer.

## What remains in `main.go`

The file is smaller, but what remains is now disproportionately runtime-sensitive:

1. scheduled-trigger governance and deferral
   - `newGovernedScheduledTriggerDeferrer`
   - `routeScheduledTriggerThroughGovernedJob`
2. gateway boot path
   - gateway startup wiring
   - scheduler/heartbeat/channels startup
3. mission bootstrap and runtime hooks
   - `configureGatewayMissionStoreLogging`
   - `installMissionRuntimeChangeHookWithExtension`
   - `installMissionOperatorSetStepHook`
   - `configureMissionBootstrapJob`
4. persisted runtime hydration
   - `loadPersistedMissionRuntime`
   - `loadPersistedMissionRuntimeSnapshot`
5. mission step control activation/watch path
   - `activateMissionStepFromControlData`
   - `restoreMissionStepControlFileOnStartup`
   - `watchMissionStepControlFile`

This is no longer “easy helper clutter.” It is the protected runtime seam.

## What remains in `main_test.go`

The file still contains the heavyweight family that mirrors the remaining protected runtime surface:

1. mission package/prune/gateway mission-store logging
2. mission set-step confirmation path
3. mission bootstrap configuration
4. bootstrap rehydration and runtime persistence
5. approval lifecycle persistence under runtime hooks
6. step-control apply/watch behavior
7. operator set-step control path
8. durable runtime fallback/preference/fail-closed behavior

This is the runtime/bootstrap/watcher/operator-control family that the earlier split assessment deliberately left for later because it is denser and more coupled than the already-completed families.

## Compare the next three options

### Option 1: finish the remaining `main_test.go` heavyweight family split

- Expected value:
  - high
  - likely removes the last giant omnibus test hotspot in `cmd/picobot`
  - makes future runtime/bootstrap edits more reviewable by isolating the protected runtime family on purpose
  - keeps structural momentum in the same area where the campaign already has clean seam evidence
- Risk:
  - medium
  - this is still test-only work, but the remaining family is denser and more fixture-heavy than the earlier splits
  - careless movement could scatter shared helpers or weaken the runtime-truth signal
- Confidence:
  - high
  - the family is already visible in the file map and now relatively isolated after `002-002C`
  - earlier splits proved the family-by-family approach works
- Why now:
  - this is the cleanest continuation of current momentum
  - it completes the `main_test.go` de-omnibus pass before context cools
  - it lowers the review cost of any later `main.go` or runtime work
- Why not now:
  - it does not reduce production runtime concentration directly
  - the remaining family is the most complex `main_test.go` seam, so this is no longer the cheap part of the split campaign

### Option 2: pivot to TaskState structural cleanup

- Expected value:
  - very high long term
  - `internal/agent/tools/taskstate.go` remains `3343` lines and `taskstate_test.go` remains `7346`
  - this is still one of the biggest protected structural hotspots in the repo
  - successful decomposition there would pay down deeper policy/runtime slop than one more CLI/test slice
- Risk:
  - very high
  - TaskState is a protected surface spanning approvals, treasury, capabilities, campaign state, runtime persistence, and operator control
  - even a “small” extraction can drift into behavior or policy work fast
- Confidence:
  - medium
  - the assessment identified plausible seams, but they are narrower and more dangerous than the already-finished `main.go` and `main_test.go` slices
  - this likely needs a fresh bounded sub-assessment before code movement
- Why now:
  - if the goal is to attack the deepest remaining anti-slop hotspot, this is the right target
  - delaying it does not make it simpler
- Why not now:
  - it is the highest-risk option on the table
  - this checkpoint still has an unfinished, lower-risk structural lane in `main_test.go`

### Option 3: stop here and define AI-slop-free-enough for V4 entry

- Expected value:
  - medium to high
  - forces an explicit gate instead of continuing structural cleanup indefinitely
  - could prevent the campaign from turning into open-ended repo gardening
  - may be enough if the remaining slop is judged tolerable relative to V4 planning needs
- Risk:
  - medium
  - stopping here leaves one giant heavyweight `main_test.go` family and the TaskState hotspot untouched
  - V4 work would begin with unresolved structural concentration in the most runtime-sensitive areas
- Confidence:
  - medium
  - the repo is materially healthier now, but the remaining hotspots are still real
  - whether this is “enough” depends on how much protected-runtime change V4 will demand
- Why now:
  - the campaign has already removed the highest-volume low-risk clutter from both `main.go` and `main_test.go`
  - this is a credible checkpoint to define an exit gate instead of chasing every possible cleanup
- Why not now:
  - the current checkpoint still has one adjacent, lower-risk, high-signal cleanup lane left before a stop decision
  - stopping now would leave `cmd/picobot/main_test.go` still obviously oversized and structurally mixed

## Recommendation

Recommended next lane: finish the remaining `main_test.go` heavyweight family split.

### Rationale

1. It is the best remaining moderate-risk structural win.
   - The remaining `main_test.go` file is still `6943` lines.
   - That is smaller than before, but still large enough to keep review friction and helper sprawl high.

2. It is safer than jumping straight into TaskState.
   - TaskState cleanup is important, but it is a much sharper protected-runtime surface.
   - The remaining `main_test.go` split is still test-only and already structurally mapped.

3. It creates a cleaner stop point if the next decision is “enough for V4.”
   - Finishing the last heavyweight `main_test.go` family would make the CLI structural cleanup feel deliberately completed rather than abandoned mid-family.
   - That gives a stronger basis for defining “AI-slop-free-enough” after one more bounded slice.

4. It improves the cost of any later runtime work.
   - Whether the next major move is TaskState or V4 planning, isolated runtime/bootstrap tests are more valuable than another partial omnibus file.

## Bottom line

- `001A-001D` removed the low-risk mixed helper clutter from `main.go` and exposed the real protected runtime seam.
- `002-002C` removed the safe and medium-risk command families from `main_test.go` and exposed the remaining heavyweight runtime/bootstrap family.
- The best next move from this checkpoint is not TaskState yet and not an immediate stop.
- The best next lane is:
  - finish the remaining `main_test.go` heavyweight family split

After that, the repo will be in a much stronger position to make a clean yes/no decision on “AI-slop-free-enough” before V4 entry.
