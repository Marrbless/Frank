# Spec To Code Route Table

Date: 2026-04-30

This route table maps current Frank V4 terms to live code, operator surfaces,
and tests on this branch. It does not make `docs/FRANK_V4_SPEC.md` current by
itself. If a spec sentence conflicts with live code, current runbooks, or
validation evidence, live code and current evidence win.

## Current Truth Inputs

- Runtime truth router: [../CANONICAL_RUNTIME_TRUTH.md](../CANONICAL_RUNTIME_TRUTH.md)
- Hot-update operator flow: [../HOT_UPDATE_OPERATOR_RUNBOOK.md](../HOT_UPDATE_OPERATOR_RUNBOOK.md)
- Domain glossary: [../../CONTEXT.md](../../CONTEXT.md)
- V4 spec reference: [../FRANK_V4_SPEC.md](../FRANK_V4_SPEC.md)

## V4 Term Routes

| V4 term | Live code route | Operator/read route | Test route | Notes |
| --- | --- | --- | --- | --- |
| Execution plane | `internal/missioncontrol/types.go`, `internal/missioncontrol/validate.go` | Mission status/runtime summary fields | `internal/missioncontrol/validate_test.go`, `internal/missioncontrol/status_test.go` | Current values include `live_runtime`, `improvement_workspace`, and `hot_update_gate`. |
| Mission family | `internal/missioncontrol/types.go`, `internal/missioncontrol/validate.go` | Mission plans, status summaries, autonomy directives | `internal/missioncontrol/validate_test.go`, autonomy policy tests | Admission rules are implemented by validators and autonomy policy helpers, not by prose alone. |
| Runtime pack | `internal/missioncontrol/runtime_pack_registry.go`, `internal/missioncontrol/runtime_pack_component_registry.go` | `runtime_pack_identity` and `v4_summary` status sections | `internal/missioncontrol/status_runtime_pack_identity_test.go`, `internal/missioncontrol/status_v4_summary_test.go` | Active and last-known-good pointers are separate durable records. |
| Active runtime pack pointer | `internal/missioncontrol/runtime_pack_registry.go` | `runtime_pack_identity.active` | `internal/agent/loop_processdirect_test.go`, `internal/missioncontrol/status_runtime_pack_identity_test.go` | Pointer switch behavior is covered by hot-update and rollback command tests. |
| Last-known-good pack | `internal/missioncontrol/runtime_pack_registry.go` | `runtime_pack_identity.last_known_good` | `internal/missioncontrol/status_runtime_pack_identity_test.go`, LKG recertification tests in `internal/agent/loop_processdirect_test.go` | LKG recertification is generic post-promotion recovery logic. |
| Improvement workspace | `internal/missioncontrol/improvement_workspace_runner.go`, `internal/missioncontrol/improvement_run_registry.go` | `improvement_run_identity`, `improvement_candidate_identity` | `internal/missioncontrol/workspace_runner_profile_test.go`, improvement run/status tests | Current implementation records runs and candidates; it does not make every V4 autonomous improvement policy active. |
| Improvement candidate | `internal/missioncontrol/improvement_candidate_registry.go`, `internal/missioncontrol/candidate_mutation_registry.go` | `improvement_candidate_identity` | `internal/missioncontrol/status_improvement_candidate_identity_test.go`, candidate mutation tests | Candidate records point at baseline/candidate pack identity and validation basis refs. |
| Eval suite | `internal/missioncontrol/eval_suite_registry.go` | `eval_suite_identity` | `internal/missioncontrol/status_eval_suite_identity_test.go`, eval suite registry tests | Eval suite records are durable validation metadata, not live evaluator execution by themselves. |
| Candidate result | `internal/missioncontrol/candidate_result_registry.go` | `candidate_result_identity` | `internal/missioncontrol/status_candidate_result_identity_test.go`, candidate result registry tests | Candidate result records feed promotion eligibility and hot-update decision paths. |
| Promotion policy | `internal/missioncontrol/promotion_policy_registry.go` | `promotion_policy_identity` | `internal/missioncontrol/status_promotion_policy_identity_test.go`, promotion policy registry tests | Policy records carry canary, owner approval, holdout, and surface-class requirements. |
| Candidate promotion decision | `internal/missioncontrol/candidate_promotion_decision_registry.go` | `candidate_promotion_decision_identity`, `v4_summary.has_candidate_promotion_decision` | `internal/missioncontrol/status_candidate_promotion_decision_identity_test.go`, candidate promotion decision tests | This path remains distinct from canary-required hot-update promotion. |
| Hot-update gate | `internal/missioncontrol/hot_update_gate_registry.go`, `internal/missioncontrol/hot_update_execution_readiness.go` | `hot_update_gate_identity`, hot-update runbook | `internal/missioncontrol/status_hot_update_gate_identity_test.go`, `internal/missioncontrol/hot_update_execution_readiness_test.go`, hot-update direct-command tests | Gate execution readiness is read-only until an explicit command mutates state. |
| Hot-update canary requirement | `internal/missioncontrol/hot_update_canary_requirement_registry.go` | `hot_update_canary_requirement_identity`, runbook canary checklist | `internal/missioncontrol/status_hot_update_canary_requirement_identity_test.go`, canary runbook fixtures | Canary is policy-controlled and explicit. |
| Hot-update canary evidence | `internal/missioncontrol/hot_update_canary_evidence_registry.go` | `hot_update_canary_evidence_identity` | `internal/missioncontrol/status_hot_update_canary_evidence_identity_test.go`, canary fixture tests | Evidence must line up with the selected requirement and gate path. |
| Hot-update canary satisfaction | `internal/missioncontrol/hot_update_canary_satisfaction_registry.go` | `hot_update_canary_satisfaction_identity` | `internal/missioncontrol/status_hot_update_canary_satisfaction_identity_test.go` | Satisfaction is separate from authority to execute. |
| Canary satisfaction authority | `internal/missioncontrol/hot_update_canary_satisfaction_authority_registry.go` | `hot_update_canary_satisfaction_authority_identity`, `v4_summary.has_canary_authority` | `internal/missioncontrol/status_hot_update_canary_satisfaction_authority_identity_test.go`, canary runbook fixtures | Current canary-derived gate/outcome/promotion lineage preserves `canary_ref`. |
| Hot-update owner approval | `internal/missioncontrol/hot_update_owner_approval_request_registry.go`, `internal/missioncontrol/hot_update_owner_approval_decision_registry.go`, `internal/missioncontrol/approval.go` | owner approval identity status, `v4_summary.has_owner_approval_decision` | owner approval status tests and direct-command approval tests | Owner approval is durable, explicit, and separate from canary satisfaction. |
| Hot-update outcome | `internal/missioncontrol/hot_update_outcome_registry.go` | `hot_update_outcome_identity`, `v4_summary.selected_outcome_id` | `internal/missioncontrol/status_hot_update_outcome_identity_test.go`, direct-command outcome tests | Outcomes record terminal gate interpretation and are promotion inputs. |
| Promotion record | `internal/missioncontrol/promotion_registry.go` | `promotion_identity`, `v4_summary.selected_promotion_id` | `internal/missioncontrol/status_promotion_identity_test.go`, direct-command promotion tests | Promotion does not itself prove real-device deployment; it records runtime-store promotion lineage. |
| Rollback | `internal/missioncontrol/rollback_registry.go` | `rollback_identity`, `v4_summary.selected_rollback_id` | `internal/missioncontrol/status_rollback_identity_test.go`, rollback direct-command tests | Rollback remains a generic recovery path after hot update or promotion. |
| Rollback apply | `internal/missioncontrol/rollback_apply_registry.go` | `rollback_apply_identity`, `v4_summary.selected_rollback_apply_id` | `internal/missioncontrol/status_rollback_apply_identity_test.go`, rollback-apply direct-command tests | Rollback apply records execution and recovery phases separately from rollback intent. |
| V4 compact summary | `internal/missioncontrol/status.go` | `runtime_summary.v4_summary` in mission status and operator status | `internal/missioncontrol/status_v4_summary_test.go`, `internal/missioncontrol/testdata/operator_status_v4_key_surface.golden.json` | This is a compact routing/readout surface, not a replacement for detailed identity sections. |
| Phone transactional update | `scripts/termux/update-and-restart-frank`, `docs/ANDROID_PHONE_DEPLOYMENT.md` | updater stdout and `.termux-frank-backup/update.log` | `scripts/termux/test-update-and-restart-frank` | Real phone proof still requires device-run evidence; local tests cover script transaction behavior. |

## Update Rule

When adding or renaming a V4 runtime concept:

1. Update the durable record or command implementation.
2. Update the operator/read route and focused tests.
3. Update this route table if the term is part of the current implementation truth.
4. Keep historical spec files unchanged unless the task explicitly updates spec wording.
