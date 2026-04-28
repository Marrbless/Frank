# V4 Append-Only Ledger Audit

Branch: `frank-v4-154-append-only-ledger-audit`

## Scope

This audit covers the V4 lifecycle records currently implemented in `internal/missioncontrol` for improvement, evaluation, runtime-pack, hot-update, promotion, rollback, smoke, canary, package import, and workspace failure flows.

The acceptance target is `docs/FRANK_V4_SPEC.md` Acceptance Criterion 17: historical improvement and hot-update outcomes are not silently overwritten.

## Coverage

| Lifecycle area | Durable record families | Append-only / replay evidence |
| --- | --- | --- |
| Improvement attempts | `ImprovementRunRecord`, `ImprovementCandidateRecord`, `CandidateMutationRecord`, `ImprovementWorkspaceRunRecord` | Store functions normalize, validate, replay identical records idempotently, and reject divergent duplicate IDs. Tests cover replay/divergent duplicate behavior for run, candidate, mutation, and workspace-run records. |
| Eval inputs/results | `EvalSuiteRecord`, `CandidateResultRecord`, local deterministic eval runner output | Eval suites and candidate results reject divergent duplicates. Local deterministic eval replays identical candidate results and rejects divergent result content. |
| Runtime pack identity/content | `RuntimePackRecord`, `RuntimePackComponentRecord`, `RuntimeExtensionPackRecord`, active pointer and LKG pointer records | Component and extension pack records reject divergent duplicates. Active/LKG pointers are current-state records and are only mutated by gate/rollback/promotion functions with durable update refs and replay checks. |
| Package/donor imports | `PackageImportRecord` | Imports are candidate-only records with provenance and content SHA-256 identity; replay is idempotent and divergent duplicate import IDs fail closed. |
| Hot-update lifecycle | `HotUpdateGateRecord`, owner approval request/decision, canary requirement/evidence/satisfaction authority, smoke check, execution safety evidence, hot-update outcome | Gate phase functions and evidence records preserve explicit refs, handle replay deterministically, and reject missing/divergent authority or evidence. Successful outcome/promotion paths are gated by durable eval/smoke/canary refs. |
| Promotion/LKG | `PromotionRecord`, candidate promotion decision, LKG recertification | Promotion and decision records reject divergent duplicates; LKG recertification is byte-stable on replay. |
| Rollback | `RollbackRecord`, `RollbackApplyRecord` | Rollback records and rollback-apply phase records reject divergent duplicates and replay completed pointer/reload phases idempotently. |

## Remaining Implementation Rows

The remaining V4 matrix rows must continue this pattern as they add new capabilities:

- `AC-033` through `AC-037`: autonomy directive, wake-cycle, budget, failure-pause, and owner-pause records.
- `SF-007`: phone deployment profile/host capability records.

These rows remain tracked separately in the matrix. AC-017 is complete for the ledger rule itself because the implemented lifecycle outcomes now have durable append-only/idempotent record coverage or are explicitly current-state pointers guarded by append-only lifecycle records.

## Non-Goals

- No destructive migration was performed.
- No history rewrite was performed.
- No external service, network, phone hardware, or real process supervision was added.
