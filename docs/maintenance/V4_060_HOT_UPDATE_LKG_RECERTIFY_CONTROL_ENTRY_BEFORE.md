# V4-060 Hot-Update LKG Recertify Control Entry - Before

## Gap

V4-058 added the missioncontrol helper for recertifying a promoted hot-update runtime pack as last-known-good from a committed `PromotionRecord`, and V4-059 assessed the direct operator entry point. Before V4-060, operators still had no direct command for invoking that helper through the existing runtime-control path.

The hot-update lane could create gates, outcomes, and promotions, but last-known-good recertification still required direct missioncontrol helper use rather than the established direct command and `TaskState` wrapper pattern.

## Existing Pieces

The existing direct command path already supported hot-update gate, outcome, and promotion control commands. The established wrapper behavior included:

- Active or persisted job validation.
- Mission store root resolution from `TaskState`.
- Timestamp derivation through the existing TaskState timestamp helper.
- Missioncontrol helper invocation with actor `operator`.
- Runtime control audit emission.
- `changed` flag propagation for direct response selection.

The existing status read model already exposed `runtime_pack_identity.last_known_good` with state, pack id, basis, and verification timestamp.

## Required Boundary

The missing control entry needed to stay narrow. It must not add manual LKG fields, mutate the active pointer, mutate `reload_generation`, create hot-update outcomes, create or mutate promotions, mutate hot-update gates, add rollback behavior, recertify directly from gates or outcomes, broaden policy, or start V4-061.
