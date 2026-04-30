# Autonomous Stop Conditions

Autonomous work should stop and ask for human input when the next required action crosses one of these lines.

## Hard Stops

- Destructive filesystem or git operations.
- Durable schema migrations or backward-incompatible store changes.
- New runtime dependencies, network services, paid APIs, or credential requirements.
- Real provider credentials, personal accounts, or live channel setup.
- Live Termux phone access, production tmux sessions, or real rollback execution.
- Intentional public behavior breaks, including command text changes that operators depend on.
- Security policy choices such as secret-scanner selection, PII retention rules, or default network exposure.

## Soft Stops

- A matrix row is too broad to validate with one clear command.
- The validator fails for a reason unrelated to the row.
- Existing uncommitted user changes overlap the intended edit and cannot be worked around safely.
- The implementation would require guessing product policy instead of following current docs or code.

## Continue Conditions

Continue autonomously when the next row is bounded, non-destructive, covered by deterministic validators, and compatible with existing public behavior and durable data.
