git_diff_stat: |
  internal/agent/loop.go                             |  15 +
  internal/agent/loop_processdirect_test.go          | 106 +++++++
  internal/agent/tools/taskstate.go                  |  89 ++++++
  .../missioncontrol/hot_update_gate_registry.go     | 222 ++++++++++++++-
  .../hot_update_gate_registry_test.go               | 312 +++++++++++++++++++++
  5 files changed, 731 insertions(+), 13 deletions(-)

git_diff_numstat: |
  15	0	internal/agent/loop.go
  106	0	internal/agent/loop_processdirect_test.go
  89	0	internal/agent/tools/taskstate.go
  209	13	internal/missioncontrol/hot_update_gate_registry.go
  312	0	internal/missioncontrol/hot_update_gate_registry_test.go

files_changed:
  - internal/missioncontrol/hot_update_gate_registry.go
  - internal/missioncontrol/hot_update_gate_registry_test.go
  - internal/agent/tools/taskstate.go
  - internal/agent/loop.go
  - internal/agent/loop_processdirect_test.go
  - docs/maintenance/V4_041_HOT_UPDATE_RELOAD_APPLY_BEFORE.md
  - docs/maintenance/V4_041_HOT_UPDATE_RELOAD_APPLY_AFTER.md

exact_helpers_state_transitions_added:
  - added durable gate states:
      - reload_apply_in_progress
      - reload_apply_succeeded
      - reload_apply_failed
  - added public helper: ExecuteHotUpdateGateReloadApply(...)
  - added internal helper: executeHotUpdateGateReloadApplyWithConvergence(...)
  - added linkage guard: validateHotUpdateGateReloadApplyLinkage(...)
  - added bounded convergence function: hotUpdateGateRestartStyleConvergence(...)
  - added operator wrapper: TaskState.ExecuteHotUpdateGateReloadApply(...)
  - added direct command: HOT_UPDATE_GATE_RELOAD <job_id> <hot_update_id>
  - retained internal/agent/loop.go, internal/agent/tools/taskstate.go, and internal/agent/loop_processdirect_test.go because the existing direct-command path is the chosen tiny control hook for this bounded slice
  - state transitions implemented:
      - reloading -> reload_apply_in_progress -> reload_apply_succeeded
      - reloading -> reload_apply_in_progress -> reload_apply_failed
      - reload_apply_succeeded replay returns idempotently
      - reload_apply_failed remains closed

exact_tests_added:
  - internal/missioncontrol/hot_update_gate_registry_test.go
      - TestExecuteHotUpdateGateReloadApplyHappyPathPreservesPointerAndLastKnownGood
      - TestExecuteHotUpdateGateReloadApplyRecordsFailureWithoutMutatingPointer
      - TestExecuteHotUpdateGateReloadApplyRejectsInvalidStateAndBadAttribution
  - internal/agent/loop_processdirect_test.go
      - TestProcessDirectHotUpdateGateReloadCommandRecordsConvergenceResultWithoutFurtherPointerMutation

validation_commands_and_results:
  - gofmt -w internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go internal/agent/tools/taskstate.go internal/agent/loop.go internal/agent/loop_processdirect_test.go
    result: pass
  - git diff --check
    result: pass
  - go test -count=1 ./internal/missioncontrol
    result: pass
  - go test -count=1 ./internal/agent
    result: pass
  - go test -count=1 ./internal/agent/tools
    result: pass
  - go test -count=1 ./...
    result: pass
  - git status --short --branch --untracked-files=all
    result: pass

explicit_statements:
  - no second pointer switch or reload_generation increment was implemented
  - no last_known_good mutation was implemented
  - no HotUpdateOutcomeRecord or PromotionRecord was created

deferred_next_v4_candidates:
  - hot-update recovery normalization for persisted reload_apply_in_progress
  - explicit retry or terminal-resolution policy after reload_apply_failed
  - later terminal-result slice for HotUpdateOutcomeRecord creation and any promotion handoff
