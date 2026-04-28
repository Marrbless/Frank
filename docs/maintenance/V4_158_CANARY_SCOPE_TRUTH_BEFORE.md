# V4-158 Canary Scope Truth Before

Branch: `frank-v4-158-canary-scope-truth`

## Requirement Rows

- `AC-025` was `PARTIAL`.

## Observed Gap

- Canary requirement, evidence, satisfaction, and authority records existed.
- Canary read models surfaced evidence refs and satisfaction state.
- The records did not carry declared canary job/surface scope.
- Status output did not distinguish operator-recorded canary evidence from automatic traffic exercise.

## Intended Slice

- Add deterministic declared canary scope to canary requirement records.
- Add evidence source, automatic-traffic flag, and exercised scope to canary evidence records.
- Reject evidence whose exercised jobs or surfaces exceed the declared canary scope.
- Surface the scope and evidence-source truth in canary requirement, evidence, and satisfaction read models.
