# Frank Dev Workflow

> Historical workflow note: this document describes an older desktop-centric handoff model and is not the current canonical runtime-truth document for this repo. Start with [CANONICAL_RUNTIME_TRUTH.md](./CANONICAL_RUNTIME_TRUTH.md) for current implementation truth, authority routing, and the distinction between current runtime behavior and future V4 target truth.

## Purpose

Define the repo workflow for Frank-focused development when work moves between the desktop lab and a laptop.

This workflow is intended to keep:

- the desktop lab as the canonical source of truth
- the remote as transport and backup
- the laptop as a temporary working copy
- promotion back to desktop explicit and reviewable

This document fits the current Picobot repo state:

- Frank work is currently centered on the `mission-control-v1` branch
- the remote default branch is still `main`
- the desktop/phone operating model is described in [`docs/FRANK_V1_HOW_TO_USE.md`](./FRANK_V1_HOW_TO_USE.md) and [`docs/FRANK_V2_SPEC.md`](./FRANK_V2_SPEC.md)

## Core Principles

### Canonical authority

The desktop lab repository is the authoritative Frank development surface.

Authoritative state starts on desktop and is finalized on desktop.

### Remote is transport, not truth

The remote repository is used to:

- transfer state between machines
- store named checkpoints
- provide off-machine backup

The remote is not implicitly authoritative.

### Explicit state transitions

Cross-machine movement must be intentional, named, and reproducible.

### No hidden state

Avoid long-lived divergence between desktop and laptop. At any point, it should be obvious:

- what branch is canonical
- what branch is a handoff checkpoint
- what branch is temporary laptop work
- what has or has not been promoted back to desktop

## Repository Roles

| Surface | Role |
| --- | --- |
| Desktop repo | Canonical Frank state |
| Remote repo | Handoff, backup, promotion path |
| Laptop repo | Temporary working copy |

## Branching Model

### Canonical branch

Use `mission-control-v1` as the current canonical Frank branch on desktop.

This is a Frank workflow convention inside this repo. It does not replace the repository's remote default branch, which is currently `main`.

### Handoff branches

Use the `handoff/` prefix for frozen desktop checkpoints that exist only to transfer work between machines.

Example:

```sh
handoff/2026-03-26-system-action
```

### Laptop work branches

Use the `laptop/` prefix for temporary laptop-only follow-up work.

Example:

```sh
laptop/system-action-followup
```

Laptop branches are never authoritative and must be promoted explicitly back onto desktop.

## Workflow

### Phase 1: Desktop work

Start from the canonical Frank branch on desktop:

```sh
git checkout mission-control-v1
git status
```

When a meaningful slice is ready:

```sh
git add -A
git commit -m "frank v2: <slice description>"
```

Create and push a handoff branch:

```sh
git checkout -b handoff/<date>-<slice>
git push origin handoff/<date>-<slice>
```

Optional immutable checkpoint:

```sh
git tag handoff-<date>-<slice>
git push origin handoff-<date>-<slice>
```

After pushing the checkpoint, keep desktop as the place where final integration decisions happen.

### Phase 2: Laptop resume

On laptop:

```sh
git clone <remote> picobot
cd picobot
git checkout handoff/<date>-<slice>
git checkout -b laptop/<task>
```

Do the follow-up work on the `laptop/*` branch only.

### Phase 3: Laptop contribution

If the laptop work produces something worth promoting:

```sh
git add -A
git commit -m "laptop: <description>"
git push origin laptop/<task>
```

### Phase 4: Promotion back to desktop

On desktop:

```sh
git fetch origin
git checkout mission-control-v1
```

Promote the laptop work explicitly.

Fast-forward only, when history is already linear:

```sh
git merge --ff-only origin/laptop/<task>
```

Cherry-pick, when only selected commits should land:

```sh
git cherry-pick <commit>
```

Regular merge, when preserving the laptop branch history is useful:

```sh
git merge origin/laptop/<task>
```

After promotion, run final validation on desktop from the canonical branch. For this repo, that usually means the relevant Frank checks plus the normal Go validation from [`docs/DEVELOPMENT.md`](./DEVELOPMENT.md), such as:

```sh
go test ./...
```

## Hard Rules

### R1: Desktop stays authoritative

Laptop work never becomes truth by default.

### R2: No dual-active state

Do not allow both of these to exist at the same time for the same Frank slice:

- meaningful unpushed desktop changes
- independent laptop changes

Reconcile first.

### R3: Cross-machine work goes through the remote

Do not move code by manual copy, zip files, or ad hoc filesystem sync.

### R4: Promotion is explicit

Laptop work is not part of canonical Frank state until desktop reviews and integrates it.

### R5: Handoffs are named checkpoints

Use explicit branch or tag names. Avoid anonymous "latest" state.

## Failure Modes

### Drift

Cause:

- independent work happening on both machines

Result:

- unclear authority
- conflict-heavy integration
- broken mental model

Prevention:

- enforce `R2`
- promote explicitly

### Remote becomes implicit truth

Cause:

- treating a remote branch as the main working surface

Result:

- desktop loses canonical authority

Prevention:

- keep canonical decisions on desktop
- use remote branches as transport and checkpoints

### Environment mismatch

Cause:

- relying on laptop-only validation

Result:

- false green or false red

Prevention:

- do final validation on desktop before treating the slice as canonical

## Naming Standard

| Type | Format |
| --- | --- |
| Canonical | `mission-control-v1` |
| Handoff | `handoff/YYYY-MM-DD-description` |
| Laptop | `laptop/<task>` |
| Tag | `handoff-YYYY-MM-DD-description` |

## Summary

This model enforces:

- one canonical authority for Frank work: desktop on `mission-control-v1`
- explicit handoffs via `handoff/*`
- temporary laptop work via `laptop/*`
- deliberate promotion back to desktop

The remote is a transport and checkpoint layer, not a source of truth.

## Final Rule

If it is unclear which machine holds the real Frank state, stop and re-establish desktop on `mission-control-v1` as canonical before doing more work.
