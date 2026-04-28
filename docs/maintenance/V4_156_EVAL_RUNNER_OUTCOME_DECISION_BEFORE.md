# V4-156 Eval Runner Outcome Decision Before

Branch: `frank-v4-156-eval-runner-outcome-decision`

## Requirement Rows

- `AC-020` was `PARTIAL`.

## Observed Gap

- Candidate results included a runner decision field.
- Candidate promotion decisions existed for eligible results.
- Hot-update outcomes existed later in the lifecycle.
- The deterministic local eval runner did not write a terminal attempt outcome record for every run, so blocked/discarded attempts were less explicit than successful promotion-selected attempts.

## Intended Slice

- Add an append-only improvement attempt outcome record.
- Wire the local deterministic eval runner to create terminal `keep`, `discard`, or `blocked` outcomes.
- Preserve existing candidate promotion decisions for eligible `keep` results.
- Avoid activation, hot update, promotion, network calls, AI calls, and external services.
