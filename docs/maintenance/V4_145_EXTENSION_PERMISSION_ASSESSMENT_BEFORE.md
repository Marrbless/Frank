# V4-145 Extension Permission Assessment Before

Branch: `frank-v4-145-extension-permission-assessment`

## Matrix Rows

- `AC-031` is `MISSING`: extension rejection codes exist, but no extension pack permission model/admission checks exist.
- `SF-005` is `MISSING`: permission widening and compatibility blockers are not represented as deterministic read/assessment records.
- `AC-027` remains `PARTIAL`: external guardrails exist at the V3/control-plane level, but extension packs can not yet be compared for permission widening.

## Slice

Add a local deterministic runtime extension pack manifest registry and permission/compatibility assessment:

- declared tools,
- declared events,
- declared permissions,
- external side effects,
- compatibility contract,
- hot-reloadability,
- widening blockers for new permissions and new external side-effect tools.

This slice does not create approval authority and does not wire promotion/status yet.
