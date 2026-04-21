# Garbage Campaign Decision After 002C Main Test

Date: 2026-04-20

## Current checkpoint

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `7cd28bfa572015955de14cb3b351b953622cd962`
- Tags at HEAD:
  - none
- Latest nearby structural checkpoint tag:
  - `frank-garbage-campaign-002c-main-test-status-clean` on `af9719a1ae5d77375b0577c571900b5c29b8fa02`
- Ahead/behind `upstream/main`: `387 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this checkpoint

## Completed structural slices

### `main.go` `001A-001D`

- `GC2-TREAT-001A` extracted channel login and prompt/setup helpers
- `GC2-TREAT-001B` extracted memory command builders
- `GC2-TREAT-001C` extracted mission inspect read-model helpers
- `GC2-TREAT-001D` extracted mission status/assertion helpers

### `main_test.go` `002-002C`

- `GC2-TREAT-002` split memory CLI tests
- `GC2-TREAT-002A` split scheduled-trigger governance tests
- `GC2-TREAT-002B` split mission inspect tests
- `GC2-TREAT-002C` split mission status/assertion tests

## Structural risk removed by these slices

- `cmd/picobot/main.go` is down to `1939` lines from the `3219` line assessment baseline.
- `cmd/picobot/main_test.go` is down to `6943` lines from the `10997` line assessment baseline.
- The low-risk and medium-risk CLI helper families are no longer packed directly into `main.go`.
- The easiest and clearest `main_test.go` behavioral families are no longer trapped in one giant omnibus file.
- Review risk is lower because memory, scheduled-trigger, mission inspect, and mission status/assertion work can now be changed and validated in more isolated files.
- The remaining structural hotspots are now more honest: they are the protected runtime/bootstrap/control seam, not generic helper clutter.

## What remains in `main.go`

The remaining `main.go` mass is mostly protected runtime wiring:

1. scheduled-trigger governance and deferral
2. gateway boot path
3. mission bootstrap and runtime hook installation
4. persisted runtime hydration
5. mission step control activation/watch path

This is the runtime-truth seam, not cheap extraction territory.

## What remains in `main_test.go`

The remaining `main_test.go` file is the heavyweight protected-runtime family:

1. mission package/prune/gateway mission-store logging
2. mission set-step confirmation path
3. mission bootstrap configuration
4. runtime rehydration and persistence behavior
5. runtime-hook approval lifecycle persistence
6. step-control apply/watch behavior
7. operator set-step control path
8. durable runtime fallback/preference/fail-closed behavior

That is the last large `cmd/picobot` test omnibus.

## Compare the next three options

### 1. Continue the heavyweight `main_test.go` family split

- Expected value:
  - high
  - completes the `cmd/picobot` test-family de-omnibus pass
  - makes later runtime/bootstrap edits easier to review and validate
- Risk:
  - medium
  - test-only work, but this remaining family is the densest and most fixture-coupled part of `main_test.go`
- Confidence:
  - high
  - the family is already mapped and the earlier `002-002C` splits proved the approach works
- Why now:
  - it is the clearest adjacent continuation
  - it finishes the `main_test.go` cleanup before context cools
- Why not now:
  - it does not reduce production-file concentration directly

### 2. Pivot to TaskState cleanup

- Expected value:
  - very high long term
  - `internal/agent/tools/taskstate.go` at `3343` lines and `taskstate_test.go` at `7346` lines remain one of the biggest protected hotspots in the repo
- Risk:
  - very high
  - TaskState spans approvals, treasury, capabilities, campaign/runtime state, persistence, and operator control
  - even small extractions can drift into semantic changes quickly
- Confidence:
  - medium
  - the structural assessment identified plausible seams, but they are sharper and more behavior-sensitive than the finished `cmd/picobot` slices
- Why now:
  - if the goal is to hit the deepest protected anti-slop hotspot next, this is the right target
- Why not now:
  - this is the highest-risk option on the table
  - one lower-risk, high-signal structural lane still remains in `main_test.go`

### 3. Stop the garbage campaign here and define AI-slop-free-enough

- Expected value:
  - medium to high
  - forces an explicit V4 entry gate instead of letting cleanup become open-ended
  - recognizes that the repo is materially healthier than it was at Phase 2 start
- Risk:
  - medium
  - leaves the final heavyweight `main_test.go` family unsplit
  - leaves the TaskState hotspot entirely deferred
- Confidence:
  - medium
  - this is a credible stopping point, but not the cleanest one
- Why now:
  - the biggest low-risk clutter has already been removed from both `main.go` and `main_test.go`
  - this is a plausible checkpoint for defining a stop condition
- Why not now:
  - stopping here leaves one obvious adjacent structural cleanup still incomplete
  - the repo would enter V4 planning with one large known CLI/runtime test omnibus still intact

## Recommended next direction

Recommended next direction: continue the remaining heavyweight `main_test.go` family split.

### Rationale

1. It is still the best moderate-risk structural win left in the current lane.
2. It is meaningfully safer than jumping straight into TaskState.
3. It creates a cleaner, more defensible stopping point for defining “AI-slop-free enough” after one more bounded slice.
4. It improves the validation surface for any later runtime or V4 planning work.

## Bottom line

- `001A-001D` removed the low-risk mixed helper clutter from `main.go`.
- `002-002C` removed the safe and medium-risk families from `main_test.go`.
- The remaining obvious adjacent structural lane is the heavyweight runtime/bootstrap/control family still living in `cmd/picobot/main_test.go`.
- The best next move from this checkpoint is:
  - finish that remaining heavyweight `main_test.go` family split

After that, deciding whether to stop and define “AI-slop-free enough” for V4 entry will be cleaner and more credible.
