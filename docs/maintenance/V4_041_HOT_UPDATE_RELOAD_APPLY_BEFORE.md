branch: frank-v4-041-hot-update-reload-apply-skeleton
head: e1c3fc69528fd2a6f2e5562d2cf996a0398f0678
tags_at_head:
  - frank-v4-039-hot-update-pointer-switch-skeleton
ahead_behind_upstream_main: 439 0
git_status_short_branch: |
  ## frank-v4-041-hot-update-reload-apply-skeleton
baseline_go_test_count_1_all: pass

exact_files_planned:
  - internal/missioncontrol/hot_update_gate_registry.go
  - internal/missioncontrol/hot_update_gate_registry_test.go
  - internal/agent/tools/taskstate.go
  - internal/agent/loop.go
  - internal/agent/loop_processdirect_test.go
  - docs/maintenance/V4_041_HOT_UPDATE_RELOAD_APPLY_BEFORE.md
  - docs/maintenance/V4_041_HOT_UPDATE_RELOAD_APPLY_AFTER.md

exact_phase_transitions_planned:
  - reloading -> reload_apply_in_progress -> reload_apply_succeeded
  - reloading -> reload_apply_in_progress -> reload_apply_failed
  - reload_apply_succeeded replay remains idempotent
  - reload_apply_failed remains closed

exact_non_goals:
  - no second active pointer switch
  - no second reload_generation increment
  - no last_known_good_pointer.json mutation
  - no HotUpdateOutcomeRecord creation
  - no PromotionRecord creation
  - no rollback workflow changes
  - no evaluator execution
  - no scoring behavior
  - no autonomy changes
  - no provider/channel changes beyond the bounded restart-style convergence path
