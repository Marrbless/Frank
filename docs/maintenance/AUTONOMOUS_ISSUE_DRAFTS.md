# Autonomous Issue Drafts

These are local issue drafts for the remaining open matrix work. They are not
published to an external issue tracker.

## Draft: AIM-043 Secret Scan Gate Decision

Labels: security, verification, human-decision

Summary: Choose whether to add a secret-scan gate and where it should run.

Acceptance criteria:

- Evaluate at least one lightweight scanner or a no-new-dependency local pattern set.
- Identify whether the gate runs in CI, locally, or both.
- Record the approved tool/dependency choice before changing CI.
- Add validation evidence without exposing real secrets.

Blocker: Requires human approval for dependency/tool choice. Current matrix
status is `blocked`.

## Draft: AIM-007 Real Termux Updater Proof

Labels: phone-deployment, release, blocked-device

Summary: Prove the Termux updater on the real phone runtime.

Acceptance criteria:

- Run `scripts/termux/update-and-restart-frank --dry-run` on the target phone.
- Run a successful transactional update on the target phone.
- Run manual rollback on the target phone.
- Capture exit codes and redacted transcript excerpts in a dated maintenance receipt.

Blocker: Requires physical/device access.

## Draft: AIM-063 Docker Smoke Validation

Labels: packaging, docker, blocked-environment

Summary: Produce real Docker smoke evidence for the existing non-publishing
Docker smoke script.

Acceptance criteria:

- Run `make docker-smoke` on a host with Docker available.
- Confirm the local image builds and `picobot --help` runs.
- Record command output and environment notes.

Blocker: Docker is unavailable in the current WSL distro.

## Draft: AIM-079 Review Brief Template

Labels: autonomous-workflow, review

Summary: Add a small review-brief template for row closeouts.

Acceptance criteria:

- Define required diff summary, validation evidence, risks, and reviewer focus.
- Keep it short enough to use after each matrix row.
- Link it from the maintenance route if it becomes current guidance.

Blocker: None.

## Draft: AIM-080 Matrix Freshness Check

Labels: autonomous-workflow, maintenance

Summary: Add a lightweight way to flag stale candidate rows after major
architecture changes.

Acceptance criteria:

- Record how to identify stale candidate rows.
- Prefer a docs-first or small script approach with deterministic output.
- Avoid noisy date churn in routine edits.

Blocker: None.
