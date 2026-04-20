# GC2-TREAT-002 Main Test Split Assessment

## 1. Current checkpoint facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `ce7fcba0e83f1816b9025ac3604d138b889fbf1d`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `382 ahead / 0 behind`
- Repo green status: yes
  - Evidence: `go test -count=1 ./...` passed at assessment start

## 2. Current line count for `cmd/picobot/main_test.go`

- `10997`

## 3. Major behavioral families in the file

1. channel/prompt tests
   - `TestPromptSecretFallsBackToReaderWhenNotTerminal`
   - `TestPromptSecretUsesHiddenInputWhenTerminalAvailable`

2. memory CLI tests
   - `TestMemoryCLI_ReadAppendWriteRecent`
   - `TestMemoryCLI_Rank`

3. scheduled-trigger governance tests
   - `TestRouteScheduledTriggerThroughGovernedJob...`
   - `TestGovernedScheduledTriggerDeferrer...`

4. agent CLI tests
   - `TestAgentCLI_ModelFlag`

5. mission status command tests
   - `TestMissionStatusCommand...`

6. mission inspect command tests
   - `TestMissionInspectCommand...`

7. mission assert and assert-step command tests
   - `TestMissionAssertCommand...`
   - `TestMissionAssertStepCommand...`

8. mission set-step command tests
   - `TestMissionSetStepCommand...`

9. mission package/prune/logging tests
   - `TestMissionPackageLogsCommand...`
   - `TestMissionPruneStoreCommand...`
   - `TestConfigureGatewayMissionStoreLogging...`

10. mission bootstrap / runtime persistence / watcher / operator-control tests
    - `TestConfigureMissionBootstrap...`
    - `TestWriteMissionStatusSnapshot...`
    - `TestMissionStatusBootstrap...`
    - `TestMissionStatusRuntimeChangeHook...`
    - `TestApplyMissionStepControlFile...`
    - `TestWatchMissionStepControlFile...`
    - `TestMissionOperatorSetStepCommand...`

## 4. Duplicated fixtures/helpers

- repeated HOME + `config.Onboard()` setup in small CLI tests
  - especially `memory` and `agent` tests
- repeated mission inspect capability fixture writers
  - `writeMissionInspect*CapabilityFixtures`
- repeated mission bootstrap/store fixture writers
  - `writeCommittedMissionBootstrap*`
  - `writeMissionBootstrapJobFile`
  - `runtimeControlForBootstrapStep`
- repeated mission status/control file readers/writers
  - `writeMissionStatusSnapshotFile`
  - `readMissionStatusSnapshotFile`
  - `writeMissionStepControlFile`
  - `readMissionStepControlFile`

The file has real duplication, but the fixture density is not uniform. Some families are nearly self-contained, others are deeply intertwined.

## 5. Safest first split seam

Safest first seam: `memory` CLI tests.

- Tests:
  - `TestMemoryCLI_ReadAppendWriteRecent`
  - `TestMemoryCLI_Rank`
- Why safest:
  - clear command-family boundary
  - no runtime-truth or mission bootstrap semantics
  - no dependency on the dense mission fixture graph
  - uses only ordinary CLI/config/workspace setup already local to the tests

## 6. Tests that must stay together

- mission status + mission assert + mission assert-step
  - these share assertion helpers, status-snapshot semantics, and gateway observation expectations
- mission bootstrap + runtime persistence + operator-control/watcher tests
  - these share runtime persistence fixtures and are tightly coupled to protected runtime-truth behavior
- scheduled-trigger governance tests
  - these should stay together until split as their own family because they share provider stubs and governed-job behavior

## 7. Protected runtime-truth areas that should not be casually split

- mission bootstrap / persisted runtime / resume-approval tests
- mission runtime change hook persistence tests
- mission step control watcher/apply tests
- operator set-step confirmation path tests
- scheduled-trigger governance tests

These are not unsplittable, but they should be handled as dedicated family moves, not mixed into an opportunistic first split.

## 8. Recommended first implementation slice

Recommended first slice: split the `memory` CLI tests into `cmd/picobot/main_memory_test.go`.

- Why:
  - smallest clearly bounded family
  - lowest semantic risk
  - no helper extraction required
  - preserves test names and behavior exactly
- Files:
  - update `cmd/picobot/main_test.go`
  - add `cmd/picobot/main_memory_test.go`
- Validation gate:
  - `gofmt -w` on touched test files
  - `git diff --check`
  - `go test -count=1 ./cmd/picobot`
  - `go test -count=1 ./...`
