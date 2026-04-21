# V4 Entry Decision

## 1. Current Checkpoint Facts

- Canonical repo: `/mnt/d/pbot/picobot`
- Branch: `frank-v3-foundation`
- HEAD: `9381ec800728aeddd7d95f8ab6b4e5bed49d02ea`
- Tags at HEAD:
  - `frank-garbage-campaign-stop-v4-entry`
- Ahead/behind `upstream/main`: `397 ahead / 0 behind`
- Worktree status at entry decision time: clean
- Repo green: yes
  - `go test -count=1 ./...` passed
- Garbage campaign stop point has already been accepted in:
  - `docs/maintenance/garbage-day/GARBAGE_CAMPAIGN_DECISION_AFTER_GC3_001C.md`

## 2. What The Garbage Campaign Removed

The garbage campaign removed the most obvious sources of structural slop that would have distorted V4 entry work.

- Truth-surface drift cleanup through `006C` separated current implementation truth from future V4 target truth.
- `main.go` cleanup through `001D` removed large mixed helper concentrations from the top-level operator CLI path.
- `main_test.go` split campaign through `002D` removed the omnibus mixed-family test concentration and left clearer family boundaries.
- `TaskState` cleanup through `GC3-TREAT-001C` removed two bounded production families and one bounded test family from the main `TaskState` concentration:
  - owner-facing counters
  - capability exposure appliers
  - `OperatorInspect` test family
- `GC3-FIX-001` removed a real Zoho timestamp flake family that would have poisoned V4 baseline confidence.

Net result:

- V4 entry no longer starts from an obviously slopped top-level CLI/test surface.
- V4 planning can now be done against a green repo whose remaining complexity is concentrated in real central runtime/state surfaces, not random omnibus leftovers.

## 3. Structural Hotspots Still Remaining But Explicitly Accepted For V4 Entry

These hotspots remain, but they are now accepted as carry-forward concentrations rather than blockers to V4 entry:

- `internal/agent/tools/taskstate.go` — central runtime/state coordination knot
- `internal/agent/tools/taskstate_test.go` — still the largest remaining TaskState test omnibus
- `internal/missioncontrol/treasury_registry_test.go` — large, dense missioncontrol test surface
- `internal/agent/tools/frank_zoho_send_email_test.go` — large integration-heavy campaign/Zoho test family
- `internal/agent/loop.go` and adjacent loop tests — still meaningful control-flow concentration

Acceptance rule:

- These files are not treated as “clean,” only as “clean enough not to block deliberate V4 entry.”
- Early V4 work must not opportunistically rewrite them just because they are still large.

## 4. Current Repo V4 Readiness Level

Current readiness level:

- **Decision-ready**
- **Planning-ready**
- **Not substrate-ready**
- **Not hot-update-ready**

Meaning:

- The repo is ready for a deliberate V4 starting decision.
- The repo is not yet ready for broad V4 behavior work.
- The missing pieces are the actual V4-specific substrate surfaces described by `docs/FRANK_V4_SPEC.md`, including:
  - active runtime pack boundary
  - candidate/improvement workspace boundary
  - hot-update gate
  - promotion/rollback records
  - append-only improvement/hot-update ledgers

So the correct interpretation is:

- V4 may start now.
- V4 must start with substrate definition and bounded scaffolding, not broad product behavior expansion.

## 5. First 3 Plausible V4 Starting Lanes

### Lane A — Active Runtime Pack Registry And Last-Known-Good Pointer

Goal:

- Introduce the smallest durable substrate for “one active pack at a time” plus an explicit rollback target model.

Why it is plausible:

- `FRANK_V4_SPEC.md` treats the active runtime pack pointer and rollback target as first-class truth.
- Improvement workspace and hot-update gate both depend on this substrate.
- This is the narrowest V4 lane that establishes new target truth without prematurely implementing hot update.

### Lane B — Improvement Workspace Envelope And Candidate Record Skeleton

Goal:

- Introduce the isolated improvement-workspace record shape and candidate-pack envelope without making candidates active.

Why it is plausible:

- The spec requires a distinct candidate-building plane.
- This could define mutable targets, immutable evaluator inputs, and candidate identity without touching live runtime yet.

### Lane C — Hot-Update Gate Skeleton For Class-1 Reloadable Surfaces

Goal:

- Introduce a governed stage/validate/apply/rollback control path for the smallest admitted hot-update surface.

Why it is plausible:

- The spec is built around the hot-update gate as the only valid candidate-to-active transition path.
- This is the first lane that begins to exercise the core V4 operating model.

## 6. Which V4 Lane Should Go First And Why

Recommended first lane:

- **Lane A — Active Runtime Pack Registry And Last-Known-Good Pointer**

Why first:

- It is the foundational dependency for the other two lanes.
- It preserves the current truth boundary: no candidate content becomes active, no hot-update semantics are claimed, and no improvement workspace autonomy is implied yet.
- It can be implemented as a bounded, deterministic storage/control-plane slice rather than a behavior-heavy runtime rewrite.
- It aligns with the garbage-campaign conclusion that V4 should start deliberately, not by reopening broad high-risk refactors.

Why not start with Lane B:

- Candidate records without active-pack truth risk creating a second authority model before the primary runtime-pack truth exists.

Why not start with Lane C:

- A hot-update gate without established active-pack and rollback-target substrate is upside down.

## 7. What Must Still Be Avoided During Early V4 Work

- No broad rewrite of `TaskState`, missioncontrol, or loop control just because V4 has started.
- No silent mutation of active runtime files outside governed V4 pack truth.
- No policy-surface self-mutation:
  - authority
  - approval
  - autonomy predicate
  - treasury rules
  - campaign rules
  - capability-onboarding rules
- No speculative phone-only deployment rewrites before the substrate exists.
- No “Pi-like” package import behavior that bypasses explicit candidate/active/rollback records.
- No uncontrolled donor-surface copying into live runtime.
- No V4 lane that mixes substrate introduction with broad product-surface changes in the same slice.
- No early V4 work that reopens garbage-campaign cleanup as a disguised implementation task.

## 8. Exact Recommended Branch Strategy From Here

- Keep `frank-v3-foundation` as the accepted clean-green baseline and decision anchor.
- Do not continue feature work directly on `frank-v3-foundation`.
- Create one new dedicated V4 branch from the current `HEAD`:
  - `frank-v4-001-pack-registry-foundation`
- Treat that branch as the single bounded lane for the first V4 slice only.
- After that slice lands and validates, make the next branch from the updated V4 line, not from old garbage-campaign checkpoints.

Branch rule:

- one bounded V4 substrate lane per branch
- no mixed cleanup + substrate + behavior branch

## 9. Exact Recommended First Implementation Slice

Recommended first implementation slice:

- **V4-001: Active runtime pack registry foundation**

Scope intent:

- define a durable active-pack record
- define a durable last-known-good / rollback-target record shape
- define pack identity and pointer invariants
- expose inspection/read-model support for those records
- do **not** yet implement candidate mutation, hot-update apply, autonomous promotion, or broad runtime reload behavior

Acceptance target for the slice:

- the repo gains one canonical durable truth for:
  - current active runtime pack
  - previous last-known-good pack
  - rollback eligibility metadata
- replay/idempotence semantics are explicit for pack-pointer records
- no existing V3 runtime behavior changes
- no claim that improvement workspace or hot-update gate is already complete

Practical reason this should be first:

- It creates the minimum V4 substrate that later slices can safely build on without inventing parallel truth models.

## Decision

- Enter V4 now.
- Enter V4 through a bounded substrate-first branch.
- Start with `V4-001: Active runtime pack registry foundation`.
- Do not start V4 by reopening heavy structural cleanup or by attempting end-to-end hot-update behavior first.
