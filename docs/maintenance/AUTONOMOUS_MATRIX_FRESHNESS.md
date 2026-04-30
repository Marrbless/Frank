# Autonomous Matrix Freshness

Last freshness review: 2026-04-30

Use this note to decide when `candidate` rows in
[AUTONOMOUS_IMPROVEMENT_MATRIX.md](./AUTONOMOUS_IMPROVEMENT_MATRIX.md) need a
fresh read before implementation.

## Review Triggers

Re-read and update affected candidate rows after any major change to:

- CLI command layout or operator-facing output.
- Mission-store durable schemas, record paths, or validation rules.
- Channel authorization, allowlist, or open-mode behavior.
- Build, release, Docker, Termux, or CI validation commands.
- Current maintenance routing docs or agent work rules.

## Current Open Classification

- `AIM-043`: blocked on human approval for secret-scan tool or dependency
  choice before implementation.
- `AIM-007`: blocked on real Termux phone access.
- `AIM-063`: blocked on Docker availability in the current environment.

No unblocked implementation candidate rows remain after the 2026-04-30
freshness review.

## Update Rule

When a row is reclassified, update the matrix row and this note in the same
change only after validation evidence exists. Do not churn the review date for
routine edits that do not affect candidate assumptions.
