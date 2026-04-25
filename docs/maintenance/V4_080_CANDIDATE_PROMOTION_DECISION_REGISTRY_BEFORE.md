# V4-080 Candidate Promotion Decision Registry Before State

## Before-State Gap From V4-079

V4-078 added read-only candidate promotion eligibility derivation through `EvaluateCandidateResultPromotionEligibility(root, resultID)`.

V4-079 assessed the next safe boundary and concluded that an eligible candidate result needs a durable decision/proposal record before any actual pack promotion. The repo still had no `CandidatePromotionDecisionRecord` registry and no immutable record that captured "this eligible result was selected for future promotion handling" without creating a `PromotionRecord`.

## Existing Boundaries

- `CandidateResultRecord` stores scores, decision, and promotion policy linkage.
- `CandidatePromotionEligibilityStatus` derives `eligible`, gated, rejected, unsupported, or invalid states from committed records.
- `PromotionRecord` represents an actual hot-update promotion after successful hot-update outcome linkage.
- Hot-update gates, outcomes, rollbacks, active runtime-pack pointer, last-known-good pointer, and reload generation are separate durable surfaces.

## Required V4-080 Gap

V4-080 must add only the durable candidate promotion decision registry skeleton:

- immutable record storage under a deterministic path
- helper creation only from `promotion_eligibility.state == "eligible"`
- exact replay idempotence
- divergent duplicate rejection
- same-result duplicate rejection
- read-only status exposure if local and repo-consistent

## Non-Goals

V4-080 must not create `PromotionRecord`, create hot-update gates, create hot-update outcomes, create rollbacks, mutate active runtime-pack pointer, mutate last-known-good pointer, mutate `reload_generation`, execute canaries, request owner approval, run evals, score candidates, add commands, add TaskState wrappers, or start V4-081.
