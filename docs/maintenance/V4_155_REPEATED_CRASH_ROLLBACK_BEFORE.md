# V4-155 Repeated Crash Rollback Before

Branch: `frank-v4-155-repeated-crash-rollback`

## Requirement Rows

- `AC-018` was `MISSING`.

## Observed Gap

- Rollback and rollback-apply records existed.
- Last-known-good pointers existed.
- Smoke check evidence existed.
- There was no deterministic repeated smoke/runtime failure ledger, quarantine record, or policy assessment that converted repeated active-candidate failures into rollback or terminal blocker records.

## Intended Slice

- Add local runtime failure event records.
- Add deterministic 3-consecutive-failure assessment using the human addendum policy.
- With LKG, create rollback/rollback-apply records, switch the active pointer through existing rollback-apply behavior, and quarantine the failing candidate.
- Without LKG, record a terminal blocker and leave the active pointer unchanged.
