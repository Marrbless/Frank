# GC3-TREAT-001 TaskState Structural Assessment

Date: 2026-04-20

## 1. Current checkpoint facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `f35102f308441179c6c089c6e58883a020cb26dc`
- `git log --oneline --decorate -20` starts with:
  - `f35102f (HEAD -> frank-v3-foundation) docs: record garbage campaign exit gate after 002D`
  - `1816055 (tag: frank-garbage-campaign-002d-main-test-runtime-clean) test: split main_test runtime bootstrap family`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `390 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this checkpoint
- Worktree status at assessment start: clean

## 2. Current line counts

- `internal/agent/tools/taskstate.go`: `3343`
- `internal/agent/tools/taskstate_test.go`: `7346`
- `internal/agent/tools/taskstate_status_test.go`: `1531`
- `internal/agent/tools/taskstate_readout.go`: `282`

## 3. Major responsibility clusters still present

### Core state container and hook registry

- `TaskState` struct plus constructor and basic getters/setters:
  - `NewTaskState`
  - `BeginTask`
  - `SetMissionStoreRoot`
  - `SetExecutionContext`
  - `MissionRuntimeState`
  - `MissionRuntimeControl`
  - hook setters
- This is acceptable as the central coordination object, but it currently also owns many families that should not live in the same file.

### Step activation and capability exposure

- `ActivateStep` remains the big front door.
- It fans out into:
  - notifications capability
  - shared storage capability
  - contacts capability
  - location capability
  - camera capability
  - microphone capability
  - SMS/phone capability
  - bluetooth/NFC capability
  - broad app control capability
- The pattern is repetitive and locally coherent, but still concentrated in one production file.

### Treasury / campaign / onboarding activation side effects

- `applyTreasuryExecutionForStep`
- `applyZohoMailboxBootstrapForStep`
- `applyTelegramOwnerControlOnboardingForStep`
- `applyCampaignReadinessGuardForStep`
- This is a high-risk coordination zone because it touches policy-heavy missioncontrol mutation producers and committed-store semantics.

### Runtime mutation and completion

- `ResumeRuntime`
- `HydrateRuntimeControl`
- `ApplyStepOutput`
- `EnforceUnattendedWallClockBudget`
- `RecordFailedToolAction`
- These methods are central to state progression and budget enforcement.

### Frank Zoho outbound / reply-work-item lifecycle

- `PrepareFrankZohoCampaignSend`
- `RecordFrankZohoCampaignSend`
- `RecordFrankZohoCampaignSendFailure`
- `ManageFrankZohoCampaignReplyWorkItem`
- plus helper transitions:
  - `claimFrankZohoCampaignReplyWorkItem`
  - `transitionFrankZohoCampaignReplyWorkItemResponded`
  - `transitionFrankZohoCampaignReplyWorkItemOnFailure`
  - `ensureFrankZohoCampaignReplyWorkItem`
- This is a distinct subdomain embedded inside TaskState.

### Owner-facing counters and acknowledgements

- `RecordOwnerFacingMessage`
- `RecordOwnerFacingCheckIn`
- `RecordOwnerFacingDailySummary`
- `RecordOwnerFacingApprovalRequest`
- `RecordOwnerFacingCompletion`
- `RecordOwnerFacingWaitingUser`
- `RecordOwnerFacingBudgetPause`
- `RecordOwnerFacingDenyAck`
- `RecordOwnerFacingPauseAck`
- `RecordOwnerFacingSetStepAck`
- `RecordOwnerFacingRevokeApprovalAck`
- `RecordOwnerFacingResumeAck`
- This is the cleanest repetitive wrapper family still inside `taskstate.go`.

### Waiting-user / approval / operator control

- `ApplyWaitingUserInput`
- `ApplyNaturalApprovalDecision`
- `ApplyApprovalDecision`
- `RevokeApproval`
- `PauseRuntime`
- `ResumeRuntimeControl`
- `AbortRuntime`
- `resolveNaturalApprovalRequestFromExecutionContext`
- `resolveNaturalApprovalRequestFromPersistedRuntime`
- `applyRuntimeControl`
- This is one of the most sensitive correctness zones because it handles active-path vs persisted-path parity and canonical rejection behavior.

### Persistence / hydration / projection internals

- `storeRuntimeStateLocked`
- `persistPreparedRuntimeStateLocked`
- `persistHydratedRuntimeStateLocked`
- `hydrateRuntimeControlLocked`
- `projectRuntimeStateLocked`
- `storeMissionJobLocked`
- `runtimeAuditContext`
- `emitRuntimeControlAuditEvent`
- This is the central mutation/persistence core and the highest-risk internal concentration in the file.

## 4. Which earlier cleanup seams are already done

- Readout-only operator surfaces are already extracted into [taskstate_readout.go](/mnt/d/pbot/picobot/internal/agent/tools/taskstate_readout.go):
  - `OperatorStatus`
  - `OperatorInspect`
  - readout-only campaign/treasury/bootstrap preflight helpers
- Status/readout-heavy test coverage is already split out into [taskstate_status_test.go](/mnt/d/pbot/picobot/internal/agent/tools/taskstate_status_test.go).
- This means TaskState cleanup is not starting from zero. The read-only adapter seam is already separated from the main mutation file.

## 5. Riskiest mutation / persistence zones

### Runtime persistence core

- Highest-risk cluster:
  - `storeRuntimeStateLocked` at `2962`
  - `persistPreparedRuntimeStateLocked` at `3037`
  - `persistHydratedRuntimeStateLocked` at `3073`
  - `hydrateRuntimeControlLocked` at `3084`
  - `projectRuntimeStateLocked` at `3126`
- Why risky:
  - controls active vs persisted runtime parity
  - controls execution-context teardown/rehydration behavior
  - drives both projection hooks and durable persistence hooks
  - a mistake here will silently corrupt multiple later behaviors

### Approval / reboot-safe operator control path

- High-risk cluster:
  - `ApplyNaturalApprovalDecision` at `2475`
  - `ApplyApprovalDecision` at `2555`
  - `RevokeApproval` at `2697`
  - `applyRuntimeControl` at `3161`
- Why risky:
  - mixes active execution-context path with persisted runtime-control path
  - carries rejection semantics the user has already treated as canonical
  - easy to break wrong-job/wrong-step/terminal-runtime handling

### Treasury activation branch set

- High-risk cluster:
  - `applyTreasuryExecutionForStep` at `1012`
- Why risky:
  - contains many distinct policy branches
  - depends on committed store state, writer leases, and multiple producer hooks
  - one function currently spans bootstrap, active, suspended, and default activation paths

### Zoho outbound / reply-work-item transitions

- High-risk cluster:
  - `PrepareFrankZohoCampaignSend` through `ensureFrankZohoCampaignReplyWorkItem`
- Why risky:
  - state-machine style behavior
  - dedupe/replay/finalization semantics
  - coupled to runtime persistence plus mailbox verification/follow-up gating

## 6. Safest next production seam

- Recommended safest next production seam:
  - extract the owner-facing counter / acknowledgement family from `taskstate.go` into a dedicated same-package file such as `taskstate_ownerfacing.go`

### Why this is the safest production seam

- The methods are repetitive wrappers around missioncontrol helpers with stable shape.
- They do not define canonical persistence logic; they use it.
- They form a contiguous and coherent family.
- They are already test-locked by focused budget/ack tests in `taskstate_test.go`.
- Extracting them reduces file size and clarifies that they are budget/accounting wrappers, not runtime-core logic.

### What this seam should include

- `RecordOwnerFacingMessage`
- `RecordOwnerFacingCheckIn`
- `RecordOwnerFacingDailySummary`
- `RecordOwnerFacingApprovalRequest`
- `RecordOwnerFacingCompletion`
- `RecordOwnerFacingWaitingUser`
- `RecordOwnerFacingBudgetPause`
- `RecordOwnerFacingDenyAck`
- `RecordOwnerFacingPauseAck`
- `RecordOwnerFacingSetStepAck`
- `RecordOwnerFacingRevokeApprovalAck`
- `RecordOwnerFacingResumeAck`

### Why not start with persistence-core extraction

- It is higher value long-term, but it is not the smallest safe first move.
- The persistence-core cluster is the place most likely to create semantic regressions while still “looking like” a mechanical extraction.

## 7. Safest next test-only seam

- Recommended safest next test-only seam:
  - split the `OperatorInspect` family out of `taskstate_test.go` into a dedicated file such as `taskstate_inspect_test.go`

### Why this is the safest test-only seam

- `taskstate_readout.go` already created the corresponding production seam.
- The inspect tests are already clustered together late in `taskstate_test.go`.
- They are conceptually readout-adjacent rather than mutation-core-adjacent.
- This continues the pattern already established by `taskstate_status_test.go`.

### What this seam should include

- `TestTaskStateOperatorInspectWithoutValidatedPlanReturnsDeterministicError`
- `TestTaskStateOperatorInspectActiveExecutionContextZeroTreasuryRefPathUnchanged`
- `TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedTreasuryPreflight`
- `TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedCampaignPreflight`
- `TestTaskStateOperatorInspectSurfacesCampaignZohoEmailAddressing`
- `TestTaskStateOperatorInspectActiveAndPersistedPathsPreserveAdapterBoundaryContract`
- `TestTaskStateOperatorInspectUsesPersistedInspectablePlanWithoutMissionJob`
- `TestTaskStateOperatorInspectPersistedInspectablePlanPathUnchangedForTreasurySteps`
- `TestTaskStateOperatorInspectPersistedInspectablePlanWrongJobDoesNotBind`
- `TestTaskStateOperatorInspectPersistedInspectablePlanRejectsInvalidStep`
- `TestTaskStateOperatorInspectTerminalRuntimeUsesPersistedInspectablePlanWithoutMissionJob`

## 8. Ranked TaskState backlog

### `GC3-TREAT-001A` Owner-facing counter wrapper extraction

- Files:
  - `internal/agent/tools/taskstate.go`
  - new same-package file such as `internal/agent/tools/taskstate_ownerfacing.go`
  - adjacent tests only if movement requires it
- Slice:
  - extract only owner-facing/budget acknowledgement wrappers
- Risk: low-medium
- Confidence: high
- Why it is small:
  - narrow, repetitive, already test-locked

### `GC3-TREAT-001B` Inspect-family test split

- Files:
  - `internal/agent/tools/taskstate_test.go`
  - new `internal/agent/tools/taskstate_inspect_test.go`
- Slice:
  - move only the `OperatorInspect` family
- Risk: low
- Confidence: high
- Why it is small:
  - matches existing `taskstate_readout.go` seam
  - test-only

### `GC3-TREAT-001C` Capability exposure applier extraction

- Files:
  - `internal/agent/tools/taskstate.go`
  - new same-package file such as `internal/agent/tools/taskstate_capabilities.go`
  - possibly capability-family tests in `taskstate_test.go`
- Slice:
  - extract the notifications/shared_storage/contacts/location/camera/microphone/SMS/bluetooth/broad-app-control applier family
- Risk: medium
- Confidence: medium-high
- Why it is attractive:
  - large contiguous cluster with strong naming symmetry
- Why not first:
  - more cross-hook and store-root surface than owner-facing wrappers

### `GC3-TREAT-001D` Runtime persistence-core extraction

- Files:
  - `internal/agent/tools/taskstate.go`
  - new same-package file such as `internal/agent/tools/taskstate_runtime_persistence.go`
  - many adjacent tests
- Slice:
  - move only persistence/hydration/projection internals
- Risk: high
- Confidence: medium-low
- Why it matters:
  - this is the central structural knot
- Why not first:
  - highest regression risk in the TaskState surface

### `GC3-TREAT-001E` Approval / reboot-safe control cleanup

- Files:
  - `internal/agent/tools/taskstate.go`
  - possibly new same-package file such as `internal/agent/tools/taskstate_approvals.go`
  - adjacent tests
- Slice:
  - isolate approval-decision, revoke, and persisted-runtime matching helpers
- Risk: high
- Confidence: medium-low
- Why it matters:
  - correctness-critical parity path
- Why not first:
  - rejection semantics are brittle and already user-sensitive

## 9. Recommended next lane with rationale

- Recommended next lane: `GC3-TREAT-001A` owner-facing counter wrapper extraction

### Rationale

- It is the smallest safe production seam still left in `taskstate.go`.
- It removes a visually noisy, repetitive cluster from the giant file without touching the runtime persistence core.
- It creates a cleaner boundary between:
  - TaskState’s mutation/persistence engine
  - owner-facing budget/accounting wrappers
- It is a good first move before any deeper TaskState work because it proves that post-`cmd/picobot` structural cleanup can continue safely in the TaskState area without starting with the riskiest core.

## 10. Should TaskState be cleaned before V4 entry or can it wait?

- TaskState should receive at least one bounded cleanup slice before a deliberate V4 entry decision is finalized.
- A full TaskState rewrite or full de-omnibus campaign does not need to block V4.
- But stopping immediately, with `taskstate.go` and `taskstate_test.go` still the clearest central structural hotspot, would be a weaker entry point than necessary.

### Explicit statement

- Recommended:
  - do one or more small TaskState structural slices before V4 entry
- Not recommended:
  - carry the entire current TaskState concentration unchanged into V4 planning if avoidable
