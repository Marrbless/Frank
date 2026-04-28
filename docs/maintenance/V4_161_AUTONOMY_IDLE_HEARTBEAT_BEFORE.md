# V4-161 Autonomy Idle Heartbeat Before

Branch: `frank-v4-161-autonomy-idle-heartbeat`

## Matrix Row

- Requirement: `AC-034`
- Status before slice: `MISSING`
- Gap: V4-160 could record due standing-directive mission proposals, but no durable idle/no-eligible wake cycle existed.

## Intended Slice

Add a local deterministic heartbeat for no eligible autonomous work:

- store a wake-cycle record with `E_NO_ELIGIBLE_AUTONOMOUS_ACTION`,
- reject the heartbeat when an active unpaused directive is already due,
- surface heartbeat and directive state in operator status/read models.

No external side effects, budget debits, phone hardware, or hot-update activation are in scope.
