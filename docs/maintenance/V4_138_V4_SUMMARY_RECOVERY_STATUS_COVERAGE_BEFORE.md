# V4-138 V4 Summary Recovery Status Coverage - Before

## Live starting point

- Previous slice: V4-137 V4 Summary Recovery Refs
- Working branch: `frank-v4-138-v4-summary-recovery-status-coverage`
- Starting commit: `546a6a36812a6c38c693c21c3d667957af7bc9a9`
- Starting tag: `frank-v4-137-v4-summary-recovery-refs`
- Startup worktree: clean

## Gap found

V4-137 added compact recovery refs to `v4_summary` and documented them in the operator runbook. The existing direct-command rollback-apply tests already call `STATUS <job_id>` after recovery commands, but they only asserted detailed rollback / rollback-apply identity sections.

The operator surface did not yet have executable coverage proving that direct-command `STATUS` includes the compact recovery refs after rollback-apply record creation and phase advancement.

## Planned scope

- Update existing `internal/agent/loop_processdirect_test.go` rollback-apply tests.
- Assert `v4_summary.state = rollback_apply_recorded`.
- Assert `selected_rollback_id`.
- Assert `selected_rollback_apply_id`.
- Keep this slice test-only.

## Non-goals

- No production behavior changes.
- No status schema changes.
- No runbook text changes.
- No command syntax changes.
- No rollback or rollback-apply lifecycle changes.
- No canary-gate widening.
