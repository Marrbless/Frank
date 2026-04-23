branch: frank-v4-043-hot-update-recovery-needed
head: 9eca4703b96275124dc76a18ddea50b7100a355c
tags_at_head:
  - frank-v4-041-hot-update-reload-apply-skeleton
ahead_behind_upstream_main: 441 0
git_status_short_branch: |
  ## frank-v4-043-hot-update-recovery-needed
baseline_go_test_count_1_all: pass

exact_files_planned:
  - internal/missioncontrol/hot_update_gate_registry.go
  - internal/missioncontrol/hot_update_gate_registry_test.go
  - docs/maintenance/V4_043_HOT_UPDATE_RECOVERY_NEEDED_BEFORE.md
  - docs/maintenance/V4_043_HOT_UPDATE_RECOVERY_NEEDED_AFTER.md

exact_state_transitions_planned:
  - reload_apply_in_progress -> reload_apply_recovery_needed
  - reload_apply_recovery_needed replay remains idempotent
  - invalid pointer attribution or broken gate linkage rejects without mutation

exact_non_goals:
  - no active pointer mutation
  - no second reload_generation increment
  - no last_known_good_pointer.json mutation
  - no reload/apply retry
  - no automatic success
  - no automatic terminal failure
  - no HotUpdateOutcomeRecord creation
  - no PromotionRecord creation
  - no rollback behavior changes
  - no evaluator execution
  - no scoring behavior
  - no autonomy changes
  - no provider/channel behavior changes
