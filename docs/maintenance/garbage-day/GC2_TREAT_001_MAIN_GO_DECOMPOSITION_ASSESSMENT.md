# GC2-TREAT-001 Main.go Decomposition Assessment

## 1. Current checkpoint

- Branch: `frank-v3-foundation`
- HEAD: `c5f9c87dafa73aa1c3207130294baef7501a4927`
- Tags at HEAD: `frank-garbage-campaign-006c-clean`
- Ahead/behind `upstream/main`: `376 ahead / 0 behind`
- Repo green status: `go test -count=1 ./...` passed at assessment start

This assessment is structural only. It does not propose runtime behavior changes, V4 work, or truth-model rewrites.

## 2. `main.go` profile

- Current line count: `3219`
- Primary file role: one giant mixed-responsibility CLI executable that currently owns root command construction, runtime bootstrap, mission-control integration, channel onboarding/setup, scheduler governance, memory command wiring, status/assertion helpers, and several operator-facing file workflows.

### Top responsibility clusters inside `cmd/picobot/main.go`

1. Scheduled-trigger governance and deferral
   - Region: roughly lines `50-428`
   - Key functions:
     - `newGovernedScheduledTriggerDeferrer`
     - `routeScheduledTriggerThroughGovernedJob`
     - `(*governedScheduledTriggerDeferrer).routeOrDefer`
     - `(*governedScheduledTriggerDeferrer).drainReady`
   - Role: mission-aware cron gating and deferred replay.

2. Root command and subcommand construction
   - Region: `430-1519`
   - Key surface:
     - `version`
     - `onboard`
     - `channels login`
     - `agent`
     - `gateway`
     - `mission *`
     - `memory *`
   - Role: Cobra wiring plus a large amount of embedded business logic in `Run` / `RunE` closures.

3. Mission bootstrap, logging, and runtime hook wiring
   - Region: `1537-1945`
   - Key functions:
     - `addMissionBootstrapFlags`
     - `configureGatewayMissionStoreLogging`
     - `watchGatewayLogDayRollover`
     - `installMissionRuntimeChangeHookWithExtension`
     - `installMissionOperatorSetStepHook`
     - `configureMissionBootstrapJob`
   - Role: runtime boot coupling between CLI, mission store, and agent loop lifecycle.

4. Mission status/proof loading and validation helpers
   - Region: `1969-2190`
   - Key functions:
     - `loadMissionStatusFrankZohoSendProofFile`
     - `loadMissionStatusFrankZohoVerifiedSendProofFile`
     - `validateMissionJob`
     - `validateMissionStepSelection`
     - `loadMissionJobFile`
     - committed/persisted runtime loaders
   - Role: operator truth and mission file validation.

5. Mission inspect read models
   - Region: `2205-2498`
   - Key functions:
     - `newMissionInspectSummary`
     - capability-specific `newMissionInspect...Capability`
   - Role: read-only projection layer for mission inspection commands.

6. Mission step control activation/watch helpers
   - Region: `2500-2591`
   - Key functions:
     - `activateMissionStepFromControlData`
     - `restoreMissionStepControlFileOnStartup`
     - `watchMissionStepControlFile`
   - Role: gateway-side operational control path for switching steps.

7. Mission status snapshot/assertion/projection helpers
   - Region: `2625-2980`
   - Key functions:
     - `writeMissionStatusSnapshotFromCommand`
     - `waitForMissionStatusStepConfirmation`
     - `assertMissionGatewayStatusSnapshot`
     - `writeMissionStatusSnapshot`
     - `writeProjectedMissionStatusSnapshot`
     - `intersectAllowedTools`
   - Role: operator-facing truth surface for mission state and assertions.

8. Process entrypoint and interactive onboarding helpers
   - Region: `2990-3219`
   - Key functions:
     - `main`
     - `promptLine`
     - `promptSecret`
     - `parseAllowFrom`
     - `setupTelegramInteractive`
     - `setupDiscordInteractive`
     - `setupSlackInteractive`
     - `setupWhatsAppInteractive`
   - Role: interactive config mutation and channel login UX.

### Public entrypoints / commands / setup flows

- Root entrypoint: `main()` -> `NewRootCmd().Execute()`
- Public CLI commands built in this file:
  - `version`
  - `onboard`
  - `channels login`
  - `agent`
  - `gateway`
  - `mission status`
  - `mission inspect`
  - `mission assert`
  - `mission assert-step`
  - `mission set-step`
  - `mission package-logs`
  - `mission prune-store`
  - `memory read`
  - `memory append`
  - `memory write`
  - `memory recent`
  - `memory rank`

### Section classification

- Pure CLI wiring:
  - `NewRootCmd` structure itself
  - `wrapCommandRunEWithSurfacedValidationErrors`
  - command/flag declaration blocks
- Runtime wiring:
  - `agent` command `RunE`
  - `gateway` command `RunE`
  - scheduled-trigger deferral/routing
  - mission runtime change hooks
- Config/onboarding:
  - `onboard`
  - `channels login`
  - `promptLine`, `promptSecret`, `parseAllowFrom`
  - interactive setup helpers
- Channel/provider setup:
  - provider selection in `agent` and `gateway`
  - `channels.StartTelegram`, `StartDiscord`, `StartSlack`, `StartWhatsApp`
  - channel login helper functions
- Mission-control coupling:
  - bootstrap flags, log packaging/pruning wiring, runtime persistence hooks
  - mission status/assert/set-step/inspect helper families
  - scheduled-trigger governance

## 3. Candidate seams

### Seam A: Channel login and prompt helpers

- Seam name: `channel_login_cli`
- Exact functions/regions involved:
  - `NewRootCmd` `channels login` block, roughly `459-507`
  - `promptLine`, `promptSecret`, `parseAllowFrom`, `setupTelegramInteractive`, `setupDiscordInteractive`, `setupSlackInteractive`, `setupWhatsAppInteractive`, roughly `2999-3219`
- Why it is a coherent extraction boundary:
  - It is one operator-facing workflow family.
  - It already has a natural vocabulary boundary: interactive channel setup.
  - It mostly mutates config and prints instructions; it does not control gateway runtime truth.
- Type:
  - setup-only and side-effecting
- Risk level:
  - low to medium
- Likely destination file(s):
  - `cmd/picobot/channel_login.go`
  - optionally `cmd/picobot/prompt.go`
- Tests that would lock it:
  - `cmd/picobot/main_test.go:138` `TestPromptSecretFallsBackToReaderWhenNotTerminal`
  - `cmd/picobot/main_test.go:157` `TestPromptSecretUsesHiddenInputWhenTerminalAvailable`
  - any existing `channels login` CLI tests if later added should stay command-level, not helper-level

### Seam B: Memory command builder

- Seam name: `memory_command_family`
- Exact functions/regions involved:
  - `NewRootCmd` `memory` block, roughly `1312-1514`
- Why it is a coherent extraction boundary:
  - This is a self-contained command subtree with minimal mission-control coupling.
  - It is mostly CLI + config/workspace resolution + memory store usage.
  - The subtree can become `newMemoryCmd()` without changing root behavior.
- Type:
  - mixed read-only and side-effecting subcommands, but setup is CLI-local
- Risk level:
  - medium
- Likely destination file(s):
  - `cmd/picobot/memory_cmd.go`
- Tests that would lock it:
  - `cmd/picobot/main_test.go:475` `TestMemoryCLI_ReadAppendWriteRecent`
  - `cmd/picobot/main_test.go:961` `TestMemoryCLI_Rank`

### Seam C: Scheduled-trigger governance helpers

- Seam name: `scheduled_trigger_governance`
- Exact functions/regions involved:
  - top-level scheduled trigger cluster, roughly `50-428`
  - gateway usage inside `RunE`, roughly `594-625`
- Why it is a coherent extraction boundary:
  - The helper family is already internally cohesive and mostly self-contained.
  - It is conceptually separate from Cobra command construction.
  - It is a runtime subsystem, not a general main-file concern.
- Type:
  - side-effecting runtime wiring
- Risk level:
  - medium to high
- Likely destination file(s):
  - `cmd/picobot/scheduled_trigger.go`
- Tests that would lock it:
  - `cmd/picobot/main_test.go:548` `TestRouteScheduledTriggerThroughGovernedJobCompletesMissionBoundReminder`
  - `cmd/picobot/main_test.go:647` `TestRouteScheduledTriggerThroughGovernedJobRejectsWhileAnotherMissionIsRunning`
  - `cmd/picobot/main_test.go:727` `TestGovernedScheduledTriggerDeferrerRecordsBlockedTriggerOnce`
  - `cmd/picobot/main_test.go:786` `TestGovernedScheduledTriggerDeferrerDeduplicatesReplay`
  - `cmd/picobot/main_test.go:827` `TestGovernedScheduledTriggerDeferrerDrainsDeferredTriggerThroughOrdinaryGovernedPath`

### Seam D: Mission inspect read-model helpers

- Seam name: `mission_inspect_read_models`
- Exact functions/regions involved:
  - `newMissionInspectSummary`
  - all capability-specific `newMissionInspect...Capability`
  - roughly `2205-2498`
- Why it is a coherent extraction boundary:
  - This family is primarily read-model projection logic for one command family.
  - It is much less entangled with live runtime mutation than bootstrap/set-step/status code.
  - It can move without changing operator semantics if imports and types remain stable.
- Type:
  - mostly read-only
- Risk level:
  - medium
- Likely destination file(s):
  - `cmd/picobot/mission_inspect.go`
- Tests that would lock it:
  - capability and summary tests in `cmd/picobot/main_test.go` around `1702-3084`

### Seam E: Mission status/assertion helpers

- Seam name: `mission_status_assertions`
- Exact functions/regions involved:
  - `loadMissionStatusFrankZoho...`
  - validation/runtime loader helpers
  - `writeMissionStatusSnapshotFromCommand`
  - assertion/wait/project/write helpers
  - roughly `1969-2190` and `2625-2980`
- Why it is a coherent extraction boundary:
  - The logic is thematically coherent around operator status truth.
  - It is still risky because the family mixes pure assertions, file I/O, runtime projection, and provider-specific proof loading.
- Type:
  - mixed read-only and side-effecting
- Risk level:
  - high
- Likely destination file(s):
  - `cmd/picobot/mission_status.go`
  - maybe later `cmd/picobot/mission_status_assert.go`
- Tests that would lock it:
  - status/assert tests in `cmd/picobot/main_test.go` around `1026-1282` and `3144-4121`

### Seam F: Mission runtime/bootstrap hooks

- Seam name: `mission_runtime_bootstrap`
- Exact functions/regions involved:
  - `addMissionBootstrapFlags`
  - `configureGatewayMissionStoreLogging`
  - `watchGatewayLogDayRollover`
  - `installMissionRuntimeChangeHookWithExtension`
  - `installMissionOperatorSetStepHook`
  - `configureMissionBootstrap`
  - `configureMissionBootstrapJob`
  - roughly `1537-1945`
- Why it is a coherent extraction boundary:
  - It is one family of runtime boot, durable store, and mission hook integration.
  - It is also the most behaviorally loaded part of the file.
- Type:
  - side-effecting runtime wiring
- Risk level:
  - high
- Likely destination file(s):
  - `cmd/picobot/mission_runtime.go`
  - `cmd/picobot/mission_bootstrap.go`
- Tests that would lock it:
  - mission bootstrap/runtime persistence tests in `cmd/picobot/main_test.go` around `4993-10447`

### Seam G: Mission step control activation/watch helpers

- Seam name: `mission_step_control_runtime`
- Exact functions/regions involved:
  - `activateMissionStepFromControlData`
  - `activateMissionStepFromControlFile`
  - `restoreMissionStepControlFileOnStartup`
  - `applyMissionStepControlFile`
  - `watchMissionStepControlFile`
  - roughly `2500-2591`
- Why it is a coherent extraction boundary:
  - It is one operator control path with a clear file-watcher lifecycle.
  - It still touches live runtime state and mission authority, so the seam is real but protected.
- Type:
  - side-effecting runtime wiring
- Risk level:
  - high
- Likely destination file(s):
  - `cmd/picobot/mission_step_control.go`
- Tests that would lock it:
  - mission set-step and runtime control tests in `cmd/picobot/main_test.go` around `4152-4961`

## 4. Extraction ordering

Smallest safe future sequence:

1. Extract channel login and prompt helpers.
   - Lowest policy risk.
   - Minimal mission-control coupling.
   - Already covered by direct prompt tests.

2. Extract memory command builder.
   - Still CLI-local.
   - Lets `NewRootCmd` lose a large embedded subtree without touching gateway truth.

3. Extract scheduled-trigger governance helper family.
   - Clean subsystem boundary.
   - Worth doing before touching deeper mission bootstrap logic because the subsystem already exists outside Cobra concerns.

4. Extract mission inspect read-model helpers.
   - Mostly read-only.
   - Reduces command-body sprawl without moving runtime-control logic yet.

5. Extract mission status/assertion helpers.
   - Only after lighter seams are proven because this is operator truth surface code.

6. Extract mission runtime/bootstrap hooks and step-control runtime code.
   - Last, because these are the most semantically dangerous and most entangled with runtime truth.

## 5. Protected no-touch zones

These regions should not be casually moved before more groundwork:

1. `gateway` runtime boot path, roughly `569-731`
   - Reason: current implementation truth, provider/channel gating, cron startup, heartbeat, signal handling, and mission bootstrap all converge here.

2. Mission runtime change hooks and mission bootstrap, roughly `1537-1945`
   - Reason: durable runtime truth, persisted mission state, log/store wiring, operator step hooks.

3. Mission step control activation/watch path, roughly `2500-2591`
   - Reason: direct control over live mission step switching and startup restoration.

4. Mission status snapshot/projection/assertion path, roughly `2625-2980`
   - Reason: operator truth surface and validation semantics. Moving this too early risks changing what the repo treats as ground truth.

5. Mission inspect capability loaders, roughly `2205-2498`
   - Reason: these encode current desktop/runtime capability truth and should not be moved in ways that accidentally smuggle in V4 assumptions.

6. Provider/channel startup branches inside `gateway`, roughly `669-717`
   - Reason: current deployment/runtime truth for active channels. This is not just CLI plumbing.

## 6. Recommended first implementation slice

Recommended first slice: extract channel login and prompt helpers.

- Smallest safe boundary:
  - move the `channels login` helper family out of `main.go`
  - keep `NewRootCmd` behavior and command shape identical
  - do not alter gateway, mission, or runtime semantics
- Files likely touched:
  - `cmd/picobot/main.go`
  - new `cmd/picobot/channel_login.go`
  - maybe `cmd/picobot/main_test.go` only if a compile-time or symbol-level adjustment is needed
- Validation gate:
  - `go test -count=1 ./cmd/picobot`
  - `go test -count=1 ./...`
  - direct prompt tests must still pass:
    - `TestPromptSecretFallsBackToReaderWhenNotTerminal`
    - `TestPromptSecretUsesHiddenInputWhenTerminalAvailable`
- Why this should come before `main_test` split or `TaskState` extraction:
  - it reduces one giant-file pressure point now without crossing package boundaries
  - it establishes a safe extraction pattern inside the hottest executable file
  - it is less semantically risky than mission runtime code and less cross-cutting than `TaskState`
  - it makes later `main_test.go` decomposition easier because the helper family becomes a clearer test grouping

## 7. Explicit non-goals

Do not attempt these yet:

- Do not rewrite `gateway` runtime boot orchestration.
- Do not fold mission bootstrap/runtime/status logic into a new architecture in the same slice.
- Do not introduce V4 terminology or phone-resident assumptions into extracted files.
- Do not split `main_test.go` first and then guess extraction seams afterward.
- Do not combine `main.go` decomposition with `TaskState` extraction or other cross-package cleanup.
- Do not change command names, flags, output shape, config resolution order, or onboarding/runtime truth.

## Bottom line

`cmd/picobot/main.go` is large because it mixes three very different layers in one file:

1. Cobra CLI construction
2. current runtime/bootstrap truth
3. operator mission-control read/write surfaces

The first safe decomposition move is not the deepest logic. It is the smallest CLI-local helper family that can move without destabilizing current runtime truth: `channels login` plus prompt/setup helpers.
