# Garbage Campaign Main.go Checkpoint After 001D

Date: 2026-04-20

## Current checkpoint

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `70bd1e8aff636dff4591783737b751423c7d40ef`
- Tags at HEAD:
  - `frank-garbage-campaign-001d-maingo-clean`
- Ahead/behind `upstream/main`: `381 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this checkpoint

## Completed `main.go` slices

### `GC2-TREAT-001A` channel login helper extraction

- Extracted:
  - interactive channel login command builder
  - prompt helpers
  - channel setup helpers
- Structural risk removed:
  - removed operator-onboarding and prompt/config mutation clutter from the giant CLI root file
  - stopped simple channel-login edits from landing adjacent to gateway and mission runtime code
  - established the first safe same-package extraction pattern for `cmd/picobot`

### `GC2-TREAT-001B` memory command builder extraction

- Extracted:
  - `memory read`
  - `memory append`
  - `memory write`
  - `memory recent`
  - `memory rank`
- Structural risk removed:
  - removed one large CLI-local subtree that had no reason to stay embedded in `NewRootCmd`
  - reduced review noise from workspace/config resolution and memory ranking logic sitting next to runtime boot paths
  - made the root command construction easier to scan

### `GC2-TREAT-001C` mission inspect read-model helper extraction

- Extracted:
  - mission inspect read-model types
  - capability inspection constructors
  - summary projection helpers
- Structural risk removed:
  - separated read-only mission inspection logic from runtime-control and bootstrap code
  - reduced accidental coupling between operator inspection output and runtime mutation surfaces
  - made the inspect family easier to reason about as one read-model unit

### `GC2-TREAT-001D` mission status/assertion helper extraction

- Extracted:
  - mission status helper types
  - provider-specific Zoho proof helpers
  - status snapshot/projection writers
  - assertion/wait helpers
- Structural risk removed:
  - separated operator status/assertion logic from the remaining gateway/bootstrap/watcher code
  - reduced the amount of durable-store/status-output policy packed into the CLI root file
  - made the remaining `main.go` surface more obviously about runtime wiring than operator read-model helpers

## What changed structurally

- `cmd/picobot/main.go` is now `1939` lines, down from the `3219` line assessment baseline.
- Net reduction from the assessed baseline: `1280` lines, about `40%`.
- The main remaining `main.go` mass is no longer mostly CLI-local helper clutter. It is increasingly the protected runtime seam.

## Protected seams that remain in `main.go`

These are the main remaining high-sensitivity clusters:

1. Scheduled-trigger governance and deferral
   - `newGovernedScheduledTriggerDeferrer`
   - `routeScheduledTriggerThroughGovernedJob`
   - defer/drain helpers
   - current role: mission-aware cron routing and replay

2. Gateway boot path
   - `gateway` command startup wiring
   - current role: provider selection, agent loop boot, scheduler startup, heartbeat, channels, signal handling

3. Mission bootstrap and runtime hooks
   - `configureGatewayMissionStoreLogging`
   - `installMissionRuntimeChangeHookWithExtension`
   - `installMissionOperatorSetStepHook`
   - `configureMissionBootstrapJob`
   - current role: persisted runtime truth, store logging, runtime lifecycle wiring

4. Persisted runtime hydration
   - `loadPersistedMissionRuntime`
   - `loadCommittedMissionRuntime`
   - `loadPersistedMissionRuntimeSnapshot`
   - current role: reboot/resume truth and fail-closed runtime rehydration

5. Mission step control activation/watch path
   - `activateMissionStepFromControlData`
   - `restoreMissionStepControlFileOnStartup`
   - `watchMissionStepControlFile`
   - current role: operator control input surface for live step switching

These are not just leftover lines. They are the still-dangerous runtime seams.

## Compare the next three options

### Option 1: Continue `main.go` with scheduled-trigger governance extraction

- Expected value:
  - medium to high
  - removes one remaining coherent subsystem from `main.go`
  - continues the current momentum on the same production hotspot
  - would make the top of `main.go` less mixed and more obviously root wiring
- Risk:
  - medium to high
  - touches live gateway/runtime behavior
  - scheduled-trigger routing is mission-aware and adjacent to active agent/gateway state
  - mistakes here are not cosmetic; they can affect cron-trigger behavior and mission activation semantics
- Preconditions:
  - keep same-package only
  - preserve the existing scheduled-trigger tests as seam locks
  - do not widen into gateway boot or mission bootstrap in the same slice
- Why now:
  - context is still fresh from `001A-001D`
  - this is the next coherent production seam already identified in the assessment
- Why not now:
  - the easy CLI-local extractions are done
  - from here, `main.go` work is increasingly inside protected runtime logic rather than low-risk structural cleanup

### Option 2: Pivot to `cmd/picobot/main_test.go` split

- Expected value:
  - high
  - `cmd/picobot/main_test.go` is still `10997` lines and remains a major review/merge burden
  - splitting by command family would make future CLI refactors cheaper and more auditable
  - the production seams extracted in `001A-001D` now provide a clearer test-family map than before
- Risk:
  - medium
  - test-only changes are safer than runtime changes, but this file is dense with fixtures and command-family overlap
  - careless splitting could hide coverage drift or duplicate helper logic
- Preconditions:
  - split by real behavior families, not arbitrary size buckets
  - keep fixture semantics stable
  - preserve existing test names and assertions where possible
  - likely start with the clearest family such as mission status/assert or scheduled-trigger tests
- Why now:
  - the command-family boundaries are much clearer after `001A-001D`
  - this yields structural value without immediately entering more runtime-sensitive production code
  - it lowers the cost of future `main.go` work because command-family tests become easier to navigate
- Why not now:
  - does not reduce production-file concentration directly
  - leaves the remaining runtime-heavy `main.go` seams untouched

### Option 3: Pivot to TaskState structural cleanup

- Expected value:
  - very high long term
  - `internal/agent/tools/taskstate.go` is still `3343` lines and `taskstate_test.go` is `7346`
  - this is one of the densest protected policy/runtime surfaces in the repo
  - successful decomposition there would pay down deeper anti-slop debt than one more `main.go` slice
- Risk:
  - very high
  - TaskState combines approvals, runtime control, treasury, campaign, capability exposure, and persistence semantics
  - even a “small” cleanup can accidentally become policy or behavior work
- Preconditions:
  - pick one narrow helper seam first, not “cleanup TaskState” broadly
  - preserve the existing TaskState regression families as seam locks
  - likely needs a dedicated mini-assessment or exact sub-slice selection before code movement
- Why now:
  - it is the biggest remaining protected structural hotspot besides the giant tests
  - if deferred too long, it stays the repo’s most dangerous editing zone
- Why not now:
  - it is the highest-risk option on the table
  - the repo has just finished a successful sequence of bounded `main.go` same-package extractions; switching directly into TaskState raises the risk profile sharply

## Recommendation

Recommended next lane: `cmd/picobot/main_test.go` split.

### Rationale

1. `main.go` has already absorbed the highest-value low-risk extractions.
   - After `001A-001D`, what remains in `main.go` is disproportionately runtime-sensitive.

2. `main_test.go` is now the biggest cheap-ish structural win.
   - At `10997` lines, it is a larger structural liability than the remaining `main.go` production helper clutter.
   - The completed production-file extractions now make the test families easier to split cleanly.

3. This keeps campaign risk moderate instead of escalating immediately.
   - Scheduled-trigger extraction is plausible, but it is already inside protected runtime behavior.
   - TaskState cleanup is important, but it is the highest-risk pivot and should be entered intentionally, not as the next automatic move.

4. A test split improves the next production steps too.
   - Cleaner test-family boundaries will make later scheduled-trigger or runtime-hook extraction easier to validate and review.

## Bottom line

- `001A-001D` successfully converted `main.go` from a giant mixed helper bucket into a smaller, more obviously runtime-heavy file.
- The next best structural move is not the deepest protected runtime seam.
- The best next lane from this checkpoint is:
  - pivot to `cmd/picobot/main_test.go` split

That yields meaningful anti-slop value while holding risk below the remaining production-runtime and TaskState options.
