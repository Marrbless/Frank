# V4-154 Append-Only Ledger Audit Before

Branch: `frank-v4-154-append-only-ledger-audit`

## Requirement Rows

- `AC-017` was `PARTIAL`.

## Observed Gap

- Many V4 record families already had idempotent replay and divergent duplicate tests.
- The matrix did not yet have a durable ledger audit tying implemented improvement, hot-update, pack, eval, package, rollback, smoke, and workspace-failure records back to AC-017.
- The new V4-153 improvement workspace run record had replay coverage but not an explicit divergent duplicate test.

## Intended Slice

- Add a durable append-only ledger audit.
- Add a focused divergent duplicate test for the improvement workspace run record.
- Move `AC-017` to `DONE` while leaving remaining capability rows tracked independently.
