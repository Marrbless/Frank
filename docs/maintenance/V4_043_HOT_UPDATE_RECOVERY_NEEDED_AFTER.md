git_diff_stat: |
  .../missioncontrol/hot_update_gate_registry.go     |  96 +++++--
  .../hot_update_gate_registry_test.go               | 302 +++++++++++++++++++++
  2 files changed, 382 insertions(+), 16 deletions(-)

git_diff_numstat: |
  80	16	internal/missioncontrol/hot_update_gate_registry.go
  302	0	internal/missioncontrol/hot_update_gate_registry_test.go

files_changed:
  - internal/missioncontrol/hot_update_gate_registry.go
  - internal/missioncontrol/hot_update_gate_registry_test.go
  - docs/maintenance/V4_043_HOT_UPDATE_RECOVERY_NEEDED_BEFORE.md
  - docs/maintenance/V4_043_HOT_UPDATE_RECOVERY_NEEDED_AFTER.md

exact_phase_helper_changes_added:
  - added durable gate state:
      - reload_apply_recovery_needed
  - added bounded reconciliation helper:
      - ReconcileHotUpdateGateRecoveryNeeded(...)
  - normalization implemented:
      - reload_apply_in_progress -> reload_apply_recovery_needed
      - reload_apply_recovery_needed replay returns idempotently
  - invalid pointer attribution or broken gate linkage rejects without mutation

exact_tests_added:
  - internal/missioncontrol/hot_update_gate_registry_test.go
      - TestReconcileHotUpdateGateRecoveryNeededNormalizesInProgressWithoutMutatingPointerState
      - TestReconcileHotUpdateGateRecoveryNeededRejectsInvalidLinkageWithoutPointerMutation
      - TestReconcileHotUpdateGateRecoveryNeededReplayIsIdempotent
  - added helper fixture:
      - storeHotUpdateReloadInProgressFixture(...)

validation_commands_and_results:
  - gofmt -w internal/missioncontrol/hot_update_gate_registry.go internal/missioncontrol/hot_update_gate_registry_test.go
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
  - git status --short --branch
    result: pass

explicit_statement:
  - no pointer mutation, reload_generation increment, last_known_good mutation, outcome creation, or promotion creation was implemented

deferred_next_v4_candidates:
  - operator-facing recovery normalization entry if a later slice needs explicit control-plane invocation
  - explicit retry policy from reload_apply_recovery_needed
  - explicit terminal-failure resolution from reload_apply_recovery_needed
  - later terminal-result slice for HotUpdateOutcomeRecord creation and any promotion handoff
