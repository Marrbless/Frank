# V4-128 Canary Gate End-to-End Runbook Fixture - Before

## Scope

V4-128 starts from V4-127 (`ee8e3df`, tag `frank-v4-127-canary-gate-end-to-end-integration-fixture-assessment`) on branch `frank-v4-128-canary-gate-end-to-end-runbook-fixture`.

The slice is test-and-docs only. Production behavior, command syntax, TaskState wrappers, hot-update execution behavior, rollback, rollback-apply, LKG recertification, outcome kinds, and promotion behavior are out of scope.

## Before-State Gap From V4-127

V4-127 concluded that focused coverage across V4-095 through V4-126 covered the individual registries, guards, command handlers, and read-model surfaces, but did not yet prove the operator runbook sequence as one bounded direct-command path.

The missing fixture was an end-to-end operator-path test that:

- walks the no-owner canary branch from requirement creation through promotion audit lineage;
- walks the owner-approved branch from requirement creation through promotion audit lineage;
- records a small rejected-owner blocker assertion without duplicating the full fail-closed matrix;
- asserts final `STATUS <job_id>` read-model surfaces using `missioncontrol.OperatorStatusSummary`;
- verifies `canary_ref` and `approval_ref` propagation through gate, outcome, and promotion;
- verifies that canary-required paths do not create candidate promotion decisions;
- verifies rollback, rollback-apply, LKG recertification, natural-language runtime approval records, active pointer, and `reload_generation` side-effect boundaries.

## Existing Coverage Inspected

The direct-command tests in `internal/agent/loop_processdirect_test.go` already cover the individual V4 canary commands, owner approval decisions, canary gate creation, guarded lifecycle commands, generic outcome creation, generic promotion creation, and many fail-closed cases.

Missioncontrol registry and status tests already cover the lower-level deterministic helpers and read-model builders for canary requirement, evidence, satisfaction, satisfaction authority, owner approval request/decision, gate readiness, outcome/promotion lineage, rollback, rollback-apply, and LKG behavior.

## Intended Fixture Shape

The fixture should live in a dedicated same-package test file:

`internal/agent/loop_processdirect_canary_runbook_test.go`

It should reuse existing direct-command fixture helpers where possible, derive deterministic IDs with exported missioncontrol helpers, use fixed RFC3339Nano `observed_at` values, and avoid sleeps, network calls, provider calls, or realtime timestamps.

## Non-Goals

V4-128 does not change production Go code, command parsing, command names, record schemas, direct-command behavior, TaskState behavior, rollback behavior, rollback-apply behavior, LKG behavior, pointer-switch behavior, reload/apply behavior, outcome behavior, or promotion behavior.

V4-128 does not bind natural-language approval aliases, broaden `CandidatePromotionDecisionRecord`, change `CreateHotUpdateGateFromCandidatePromotionDecision(...)`, execute LKG recertification, create rollback records in happy paths, create rollback-apply records in happy paths, or implement V4-129.
