# V4-033 Rollback State Machine Checkpoint

## Current branch / HEAD / tags

- Branch: `frank-v4-033-rollback-state-machine-checkpoint`
- HEAD: `082ce6b25ac3e28781488853069283bb79d78121`
- Tags at HEAD: none
- Ahead/behind `upstream/main`: `431 0`

## Repo green status

- `git status --short --branch` at checkpoint start was clean:
  - `## frank-v4-033-rollback-state-machine-checkpoint`
- `go test -count=1 ./...` passed at checkpoint start.

## Completed rollback-related slices

- `V4-016`: durable rollback record skeleton
- `V4-017`: rollback read-only inspect/status exposure
- `V4-018`: rollback control-surface entry
- `V4-019`: rollback-apply record skeleton
- `V4-020`: rollback-apply read-only inspect/status exposure
- `V4-021`: rollback-apply control-surface entry for record creation/selection
- `V4-022`: durable rollback-apply phase progression model
- `V4-023`: operator/control entry for rollback-apply phase advancement
- `V4-024`: execution assessment for the first rollback-apply pointer-switch slice
- `V4-025`: real pointer-switch slice to the rollback target with `reload_generation` increment and unchanged last-known-good
- `V4-026`: assessment for the first bounded reload/apply execution slice
- `V4-027`: bounded reload/apply execution with durable success/failure recording
- `V4-028`: assessment for persisted `reload_apply_in_progress` recovery
- `V4-029`: normalization from unknown `reload_apply_in_progress` crash state to explicit `reload_apply_recovery_needed`
- `V4-030`: assessment for operator-driven recovery resolution
- `V4-031`: retry from `reload_apply_recovery_needed` on the same `apply_id`
- `V4-032`: explicit operator-driven terminal failure from `reload_apply_recovery_needed`

## What state-machine capability now exists

- Durable rollback authority exists through committed rollback records.
- Durable rollback-apply authority exists through committed rollback-apply records keyed by `apply_id`.
- Read-only operator status already exposes rollback and rollback-apply identity, phase, activation-state, and linkage.
- Operator control already supports:
  - rollback record creation
  - rollback-apply record creation/selection
  - rollback-apply phase advancement
  - pointer switch execution
  - reload/apply execution
  - retry from `reload_apply_recovery_needed`
  - explicit terminal failure from `reload_apply_recovery_needed`
- Crash/recovery handling now exists for the known unknown-outcome window:
  - `reload_apply_in_progress -> reload_apply_recovery_needed`
- Deterministic terminal states now exist:
  - `reload_apply_succeeded`
  - `reload_apply_failed`
- Idempotence rules now exist for:
  - selecting existing immutable records
  - replay after pointer switch
  - replay after reload/apply success
  - replay of the same operator terminal-failure decision
- The active-pointer and last-known-good boundaries are explicit and covered:
  - no second pointer switch on retry/failure resolution
  - no second `reload_generation` increment after V4-025
  - no last-known-good mutation in rollback-apply execution, recovery, retry, or terminal failure resolution

## What still does NOT exist

- Read-only operator status does not expose `execution_error`.
- Read-only operator status does not expose transition metadata such as `phase_updated_at` or `phase_updated_by`.
- There is no retry policy from terminal `reload_apply_failed`.
- There is no broader automatic recovery/orchestration beyond the explicit committed control paths.
- There is no pack recertification or last-known-good update tied to rollback-apply completion.

## Option comparison

### 1. Stop widening rollback state machine here and pivot to another V4 domain

- Expected value:
  - high
  - the rollback lane already supports the full bounded workflow from record creation through execution, crash normalization, retry, and terminal resolution
  - frees attention for another V4 domain that is still missing core capability rather than polishing an already-functional lane
- Risk:
  - medium-low
  - operators would still lack read-only visibility into `execution_error` and transition metadata
  - debugging a terminal rollback-apply outcome would require reading mission-store files rather than relying only on status
- Confidence:
  - high
- Why now or not now:
  - now, because the state machine itself is no longer missing a blocking transition
  - the remaining gaps are observability or policy refinements, not missing core workflow control

### 2. Add read-only `execution_error` / transition-metadata visibility

- Expected value:
  - medium
  - improves operator inspection and makes terminal-failure and recovery states easier to understand without reading raw store JSON
  - small and low-risk compared with further behavior changes
- Risk:
  - low
  - mostly output-shape widening on existing read-only status surfaces
  - some risk of overexposing fields before their long-term presentation is settled
- Confidence:
  - high
- Why now or not now:
  - not now if the question is whether the state machine is complete enough to pivot
  - yes now only if operator visibility is treated as the last required polish before leaving this domain
  - this is an observability slice, not a missing state-machine capability slice

### 3. Add retry-from-terminal-failed policy

- Expected value:
  - low to medium
  - could help when an operator wants to retry after a concrete failure without going back through recovery-needed
- Risk:
  - high
  - broadens workflow semantics materially
  - weakens the current clean distinction between unknown-outcome recovery and terminal failure
  - would need new replay rules, reason-clearing rules, and probably a sharper operator model for why terminal failure is no longer terminal
- Confidence:
  - medium-high that it is not the right next slice
- Why now or not now:
  - not now
  - the current model is intentionally disciplined: retry exists from `reload_apply_recovery_needed`, while `reload_apply_failed` is terminal
  - widening terminal-failure retry would expand policy, not close a clearly missing adjacency

## Recommended next direction

Recommend option `1`: stop widening the rollback / rollback-apply state machine here and pivot to another V4 domain.

Rationale:

- The rollback lane now has a coherent and bounded lifecycle:
  - identity
  - control entry
  - phase progression
  - pointer switch
  - reload/apply
  - crash normalization
  - explicit retry
  - explicit terminal failure
- The only clearly adjacent remaining slice with low risk is option `2`, but that is read-only observability polish rather than unfinished state-machine behavior.
- Option `3` is not justified now because it would reopen semantics that were just intentionally made terminal.

Short answer to the checkpoint question:

- The rollback / rollback-apply state machine is complete enough to stop widening this area and pivot to another V4 domain.
- If one more bounded slice is ever taken before pivoting, it should be read-only `execution_error` / transition-metadata visibility, not more workflow semantics.
