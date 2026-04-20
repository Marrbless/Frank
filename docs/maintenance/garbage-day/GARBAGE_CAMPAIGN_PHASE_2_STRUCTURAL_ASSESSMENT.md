# Garbage Campaign Phase 2 Structural Assessment

Date: 2026-04-20

## 1. Current checkpoint

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `f81567cbe7224de2b908449aa142bb54884eaebc`
- `git log --oneline --decorate -1`:
  - `f81567c (HEAD -> frank-v3-foundation, tag: frank-garbage-campaign-002c-clean) fix: harden onboarding secret handling and docs hygiene`
- Tags at HEAD:
  - `frank-garbage-campaign-002c-clean`
- Ahead/behind `upstream/main`: `371 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at this `HEAD`
- Worktree status at assessment start:
  - not clean
  - evidence: untracked `docs/maintenance/garbage-day/GARBAGE_CAMPAIGN_PHASE_1_COMPLETE.md`

## 2. Biggest structural anti-patterns

### Giant files

- The repo still has a concentrated set of production and test files that are dramatically larger than the rest of the codebase:
  - `cmd/picobot/main_test.go` `10997`
  - `internal/agent/tools/taskstate_test.go` `7346`
  - `internal/missioncontrol/treasury_registry_test.go` `3454`
  - `internal/agent/tools/taskstate.go` `3343`
  - `cmd/picobot/main.go` `3219`
  - `internal/agent/loop.go` `1847`
  - `internal/missioncontrol/treasury_registry.go` `1741`
- This is not cosmetic. These files are carrying too much policy surface, too many helper seams, and too much review burden per edit.

### Giant tests

- The test surface is especially AI-sloppy structurally:
  - `cmd/picobot/main_test.go` bundles command parsing, mission bootstrap, mission status, mission inspect, mission assert, mission set-step, scheduled trigger routing, and fixture builders in one place.
  - `internal/agent/tools/taskstate_test.go` bundles activation, treasury, capability exposure, approval lifecycle, operator input, runtime persistence, and inspection adapters in one file.
  - `internal/missioncontrol/treasury_registry_test.go` packs record roundtrip, ledger behavior, treasury preflight, execution-context resolution, and object-view assertions into one file.
- These giant test files are not just big; they hide natural behavioral families and make seam-preserving refactors harder than they need to be.

### Mixed-responsibility files

- `cmd/picobot/main.go`
  - CLI root, gateway startup, scheduled-trigger governance, mission store logging, mission bootstrap, mission status read/write, command assertion helpers, inspect helpers, onboarding prompts, and channel login all coexist.
- `internal/agent/tools/taskstate.go`
  - execution context, capability exposure, treasury step activation, Zoho send/reply work item state, approval application, waiting-user input, owner-facing counters, runtime persistence, and runtime-control auditing all coexist.
- `internal/missioncontrol`
  - still behaves like a large umbrella package spanning runtime transitions, persistence, approval, treasury, campaign, identity, capability exposure, operator readout, and validation.

### Package boundary drift

- `cmd/picobot` still imports almost every internal package directly, which is expected for a CLI root but currently means orchestration and policy-adjacent behavior are mixed together in one place.
- `internal/missioncontrol -> internal/channels` is still a suspicious dependency direction because policy/runtime code is reaching into transport/onboarding surfaces.
- `internal/agent/memory -> internal/providers` still couples ranking behavior to the provider abstraction instead of a smaller ranking interface.
- `internal/agent/tools` still combines generic tools, TaskState, registry gating, and Zoho-specific protected behavior in one package.

### Protected surfaces that are too concentrated

- The most protected V3 surfaces are still concentrated instead of isolated:
  - treasury resolution and mutation families
  - TaskState runtime/control logic
  - Zoho send/reply work-item behavior
  - mission bootstrap and operator status/readout logic
- This concentration raises the cost of safe edits because simple maintenance work lands adjacent to approval, treasury, campaign, runtime, and persistence semantics.

### Docs/spec drift still relevant to code layout

- The docs drift still matters structurally because it obscures what surfaces are actually canonical:
  - `README.md` still advertises `spawn` as if it were a built-in tool, but `spawn` is unregistered and not actually exposed.
  - `docs/FRANK_DEV_WORKFLOW.md` still says the desktop lab is the canonical development authority.
  - `docs/FRANK_V4_SPEC.md` frames the target as phone-resident and explicitly moves away from desktop as a permanent architectural requirement.
- This drift is not just wording debt. It directly affects what we think should be extracted, protected, or deferred.

## 3. Top giant-file targets

### `cmd/picobot/main.go`

- Line count: `3219`
- Why it is large:
  - Contains the CLI root command plus large helper clusters for scheduled-trigger governance, gateway log/store lifecycle, mission bootstrap, mission status/assert/set-step/inspect behavior, and interactive channel onboarding.
- Why that is or is not justified:
  - Partly justified:
    - a CLI root naturally centralizes command registration
    - this file sits at a major integration boundary
  - Not justified:
    - the file goes well beyond root wiring and contains multiple internally coherent command/helper families that can live behind package-local files without changing semantics
    - it is carrying policy-adjacent helpers that make every CLI touch high-risk
- Smallest safe extraction seams:
  - extract `channels login` interactive setup helpers and prompt helpers into a dedicated CLI-local file
  - extract scheduled-trigger deferrer and routing helpers into a dedicated CLI-local file
  - extract mission status/assert/set-step helper cluster into one or more CLI-local files while leaving Cobra wiring in `main.go`
  - leave `NewRootCmd()` in place initially and move command-family helpers out from under it
- Tests that lock each seam:
  - channel login / prompt seam:
    - `TestPromptSecretFallsBackToReaderWhenNotTerminal`
    - `TestPromptSecretUsesHiddenInputWhenTerminalAvailable`
  - scheduled trigger seam:
    - `TestRouteScheduledTriggerThroughGovernedJobCompletesMissionBoundReminder`
    - `TestGovernedScheduledTriggerDeferrerRecordsBlockedTriggerOnce`
    - `TestGovernedScheduledTriggerDeferrerDeduplicatesReplay`
    - `TestGovernedScheduledTriggerDeferrerDrainsDeferredTriggerThroughOrdinaryGovernedPath`
  - mission status/assert/set-step seam:
    - `TestMissionStatusCommandWithValidFilePrintsExpectedJSON`
    - `TestMissionStatusCommandPrintsCanonicalGatewayStatusJSON`
    - `TestMissionAssertCommandWithValidStatusFileAndNoConditionsSucceeds`
    - `TestMissionSetStepCommandWithMissionFileAndStatusFileSucceedsWhenFreshSnapshotMatchesStepAndJob`
    - `TestConfigureMissionBootstrapMissionFileActivatesStep`
- Danger level: `high`
  - reason: direct operator/runtime/control surface with prior upstream overlap

### `cmd/picobot/main_test.go`

- Line count: `10997`
- Why it is large:
  - Contains hundreds of command-family assertions plus a large amount of fixture-building and mission/test support logic.
- Why that is or is not justified:
  - Justified:
    - `cmd/picobot` is a wide CLI surface and should have strong end-to-end-ish coverage
  - Not justified:
    - one file should not be the test home for nearly every command family plus shared fixture infrastructure
    - the file size hides behavioral groupings and raises merge/review friction
- Smallest safe extraction seams:
  - split by command family, not by arbitrary line count:
    - `main_memory_test.go`
    - `main_scheduled_trigger_test.go`
    - `main_mission_status_test.go`
    - `main_mission_inspect_test.go`
    - `main_mission_assert_test.go`
    - `main_mission_set_step_test.go`
    - `main_bootstrap_test.go`
  - extract shared fixture builders into one helper file instead of repeating `t.TempDir`, status/control path builders, and mission fixture constructors across the family files
- Tests that lock each seam:
  - the existing tests themselves are the seam locks; the requirement is to preserve names, assertions, and helper semantics while relocating them
  - key family anchors:
    - `TestMemoryCLI_ReadAppendWriteRecent`
    - `TestMissionStatusCommandWithValidFilePrintsExpectedJSON`
    - `TestMissionInspectCommandWithValidFilePrintsExpectedSummary`
    - `TestMissionAssertCommandWithValidStatusFileAndNoConditionsSucceeds`
    - `TestMissionSetStepCommandWithMissionFileAndStatusFileSucceedsWhenFreshSnapshotMatchesStepAndJob`
    - `TestConfigureMissionBootstrapMissionFileActivatesStep`
- Danger level: `medium-high`
  - reason: test-only change, but coverage drift is easy when moving so many fixtures and command families at once

### `internal/agent/tools/taskstate.go`

- Line count: `3343`
- Why it is large:
  - Combines execution context state, capability exposure application, treasury execution activation, campaign/Zoho preconditions, approval logic, waiting-user logic, owner-facing message counters, runtime persistence, runtime projection, and runtime-control audit behavior.
- Why that is or is not justified:
  - Justified:
    - `TaskState` is a real protected coordination object and some cohesion is unavoidable
  - Not justified:
    - one file should not own nearly every protected subdomain beneath that coordination object
    - the current shape obscures boundaries between activation, approvals, counters, runtime persistence, and provider-specific work-item state
- Smallest safe extraction seams:
  - extract owner-facing counter family into a helper file while keeping method names and ordering unchanged
  - extract capability exposure appliers/hooks into a dedicated file with one explicit wrapper per capability
  - extract Zoho campaign send/reply work-item transitions into a dedicated file
  - extract runtime persistence/projection internals into a dedicated file while keeping `TaskState` public methods stable
- Tests that lock each seam:
  - activation / capability / treasury seams:
    - `TestTaskStateActivateStepStoresValidExecutionContext`
    - `TestTaskStateActivateStepTreasuryPathCallsActivationProducerOnce`
    - `TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation`
    - `TestTaskStateActivateStepSharedStorageCapabilityPathInvokesRealMutation`
  - approval / waiting-user seams:
    - `TestTaskStateApplyNaturalApprovalDecisionApprovesSinglePendingRequest`
    - `TestTaskStateApplyApprovalDecisionUsesPersistedRuntimeControlAfterExecutionContextTeardown`
    - `TestTaskStateApplyWaitingUserInputDoesNotBindPendingApproval`
  - owner-facing counter seam:
    - `TestTaskStateRecordOwnerFacingMessagePausesAtBudget`
    - `TestTaskStateRecordOwnerFacingSetStepAckPausesAtOwnerMessageBudget`
    - `TestTaskStateRecordOwnerFacingResumeAckPausesAtOwnerMessageBudget`
  - runtime persistence seam:
    - `TestTaskStateHydrateRuntimeControlResumesPausedRuntimeAfterRehydration`
    - `TestTaskStateEmitAuditEventPersistsIntoRuntimeHistoryAndTruncatesDeterministically`
- Danger level: `very high`
  - reason: protected approval/budget/runtime/treasury/campaign surface

### `internal/agent/tools/taskstate_test.go`

- Line count: `7346`
- Why it is large:
  - It mirrors the over-breadth of `taskstate.go`: activation, treasury, capabilities, approvals, runtime rehydration, waiting-user input, inspection adapters, and fixture generation all live together.
- Why that is or is not justified:
  - Justified:
    - `TaskState` needs heavy regression protection
  - Not justified:
    - the file is effectively a whole test package in one file
    - the size and mixed fixtures make local refactors harder than necessary
- Smallest safe extraction seams:
  - split by behavioral family:
    - activation and execution-context tests
    - treasury activation tests
    - capability exposure tests
    - approval/waiting-user tests
    - runtime rehydration/control tests
    - operator inspect adapter tests
  - move fixture writers into a shared helper file once, then let the behavioral files stay small and legible
- Tests that lock each seam:
  - activation anchors:
    - `TestTaskStateActivateStepStoresValidExecutionContext`
    - `TestTaskStateActivateStepUnknownStepDoesNotOverwriteExistingContext`
  - treasury anchors:
    - `TestTaskStateActivateStepTreasuryPathCallsActivationProducerOnce`
    - `TestTaskStateActivateStepTreasuryReplayStaysDeterministic`
  - capability anchors:
    - `TestTaskStateActivateStepNotificationsCapabilityPathInvokesRealMutation`
    - `TestTaskStateActivateStepContactsCapabilityFailsClosedWithoutSharedStorageExposure`
  - approval/runtime anchors:
    - `TestTaskStateApplyNaturalApprovalDecisionApprovesSinglePendingRequest`
    - `TestTaskStateHydrateRuntimeControlRejectsTerminalOperatorCommands`
    - `TestTaskStateAbortRuntimeTransitionsToAborted`
  - inspect anchors:
    - `TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedTreasuryPreflight`
    - `TestTaskStateOperatorInspectSurfacesCampaignZohoEmailAddressing`
- Danger level: `medium`
  - reason: test-only, but the fixture graph is dense and coverage must stay intact

### `internal/missioncontrol/treasury_registry_test.go`

- Line count: `3454`
- Why it is large:
  - Covers roundtrip and validation for treasury records and ledger entries, then a long family of execution-context treasury resolution and preflight read-model cases, then object-view adapters.
- Why that is or is not justified:
  - Justified:
    - treasury is a protected surface with many “fail closed” branches that deserve explicit tests
  - Not justified:
    - one file is currently carrying too many distinct treasury read-model families
    - the file’s size makes any future treasury registry edit look more dangerous than it needs to be
- Smallest safe extraction seams:
  - split record/ledger roundtrip and validation tests from execution-context resolver tests
  - split treasury preflight read-model tests from post-action resolver tests
  - split object-view adapter tests into a small dedicated file
  - keep helper constructors local to treasury test files instead of one giant omnibus file
- Tests that lock each seam:
  - record/ledger anchors:
    - `TestTreasuryRecordRoundTripAndList`
    - `TestTreasuryLedgerEntryRoundTripAndList`
    - `TestTreasuryRecordValidationFailsClosed`
    - `TestTreasuryLedgerEntryValidationFailsClosed`
  - execution-context resolver anchors:
    - `TestResolveExecutionContextTreasuryRefResolvesActiveTreasuryRef`
    - `TestResolveExecutionContextTreasuryBootstrapAcquisitionResolvesCommittedBootstrapBlock`
    - `TestResolveExecutionContextTreasuryPostActiveTransferResolvesCommittedActiveBlock`
  - preflight anchors:
    - `TestResolveExecutionContextTreasuryPreflightResolvesTreasuryAndContainers`
    - `TestResolveExecutionContextTreasuryPreflightFailsClosedOnMissingOrMalformedTreasuryRefs`
  - adapter/view anchors:
    - `TestTreasuryObjectViewsAdaptStorageFieldsWithoutMigration`
    - `TestActiveTreasuryObjectViewUsesDefaultActiveTransactionPolicyEnvelope`
- Danger level: `high`
  - reason: treasury surface is explicitly protected and fail-closed semantics are easy to perturb

## 4. Structural campaign backlog

### `GC2-TREAT-001` main.go decomposition assessment

- Exact files:
  - `cmd/picobot/main.go`
  - adjacent command-family tests in `cmd/picobot/main_test.go`
- Smallest safe slice:
  - extract one CLI-local command/helper family out of `main.go` without changing Cobra wiring or output contracts
  - best first seam: `channels login` plus prompt helpers, or mission status/assert/set-step helper family
- Risk: `high`
- Confidence: `high`
- Required tests:
  - targeted `cmd/picobot` tests for the extracted family
  - `go test -count=1 ./cmd/picobot`
  - `go test -count=1 ./...`
- Must happen before V4: `recommended but not mandatory`

### `GC2-TREAT-002` main_test.go split

- Exact files:
  - `cmd/picobot/main_test.go`
  - new shared helper file(s) under `cmd/picobot/`
- Smallest safe slice:
  - split one command-family cluster plus its fixtures into a new test file without changing assertions
  - best first seam: mission status/assert/set-step family or scheduled-trigger family
- Risk: `medium`
- Confidence: `high`
- Required tests:
  - `go test -count=1 ./cmd/picobot`
  - `go test -count=1 ./...`
- Must happen before V4: `no`

### `GC2-TREAT-003` TaskState helper extraction

- Exact files:
  - `internal/agent/tools/taskstate.go`
  - possibly one or more new same-package helper files
  - adjacent tests in `internal/agent/tools/taskstate_test.go` and `internal/agent/tools/taskstate_status_test.go`
- Smallest safe slice:
  - extract one protected helper family without changing method names, public behavior, or audit/runtime ordering
  - best first seam: owner-facing counter helper family
- Risk: `very high`
- Confidence: `medium-high`
- Required tests:
  - targeted `internal/agent/tools` tests covering the extracted family
  - `go test -count=1 ./internal/agent/tools`
  - `go test -count=1 ./...`
- Must happen before V4: `recommended`

### `GC2-TREAT-004` TaskState test decomposition

- Exact files:
  - `internal/agent/tools/taskstate_test.go`
  - shared helper fixture file(s) under `internal/agent/tools/`
- Smallest safe slice:
  - split one coherent behavioral family into a dedicated test file while preserving test names and helper semantics
  - best first seam: approval/runtime-control family or capability exposure family
- Risk: `medium`
- Confidence: `high`
- Required tests:
  - `go test -count=1 ./internal/agent/tools`
  - `go test -count=1 ./...`
- Must happen before V4: `no`

### `GC2-TREAT-005` treasury_registry_test split

- Exact files:
  - `internal/missioncontrol/treasury_registry_test.go`
  - possibly additional treasury-specific helper test files
- Smallest safe slice:
  - split one treasury read-model family into a dedicated test file, starting with record/ledger validation or preflight resolution
- Risk: `high`
- Confidence: `high`
- Required tests:
  - `go test -count=1 ./internal/missioncontrol`
  - `go test -count=1 ./...`
- Must happen before V4: `recommended`

### `GC2-TREAT-006` truth-surface drift cleanup

- Exact files:
  - `README.md`
  - `internal/config/onboard.go`
  - `docs/FRANK_DEV_WORKFLOW.md`
  - any minimal routing doc chosen as canonical
- Smallest safe slice:
  - remove or explicitly gate the dead `spawn` truth surface and reduce the desktop-vs-phone authority contradiction to one short canonical routing note
- Risk: `low-medium`
- Confidence: `high`
- Required tests:
  - doc review
  - if `spawn` exposure changes, `go test -count=1 ./...`
- Must happen before V4: `yes`

## 5. Exit gate proposal

“AI-slop-free enough” structurally should not mean “all large files are gone.” It should mean the repo is no longer structurally misleading in the places that would make V4 work unsafe or confused.

### Must be cleaned before V4

- At least one protected production surface must be decomposed by one surgical extraction lane:
  - either `cmd/picobot/main.go`
  - or `internal/agent/tools/taskstate.go`
- At least one giant test file must be split by behavioral family:
  - either `cmd/picobot/main_test.go`
  - or `internal/agent/tools/taskstate_test.go`
- Truth-surface drift that changes what people think the repo currently is must be cleaned:
  - `spawn` must stop being advertised as a live built-in tool unless it is intentionally wired
  - desktop-vs-phone authority drift must be reduced to one canonical routing note
- The structural campaign must leave a ranked protected-surface backlog, so V4 work does not start by casually editing giant files with no seam map

### Can wait until after V4

- Full treasury test decomposition
- Broader `internal/missioncontrol` package surgery
- Perfect package purity across `cmd`, `agent`, `tools`, and `missioncontrol`
- Elimination of every large file over `1000` lines
- Replacement of every `map[string]interface{}` with typed models

### Should never block V4

- Cosmetic file-count goals
- Chasing every low-signal duplication before protected seams are mapped
- Beautification-only splits that do not create a real safer change boundary
- Complete docs reconciliation across every historical Frank spec or maintenance artifact

### Proposed structural exit gate

The repo is “AI-slop-free enough” for V4 planning when all of the following are true:

1. The canonical branch is green and `0 behind upstream/main`.
2. One protected production seam has been extracted cleanly.
3. One giant test file has been split by real behavioral families.
4. `spawn` truth-surface drift is resolved.
5. One short canonical routing note explains the current runtime truth versus deferred V4 target truth.
6. The remaining structural backlog is explicit, ranked, and no longer masquerading as hidden incidental complexity.

If those six conditions are met, Phase 2 can stop without pretending the whole repo is elegant. That is the right standard for “AI-slop-free enough,” not perfection.
