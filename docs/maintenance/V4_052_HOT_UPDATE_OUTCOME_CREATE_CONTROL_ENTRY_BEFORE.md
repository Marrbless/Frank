# V4-052 Hot-Update Outcome Create Control Entry Before

## Before-State Gap

V4-050 added `missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(...)`, which can create a deterministic `HotUpdateOutcomeRecord` from an existing committed terminal `HotUpdateGateRecord`.

Before V4-052, operators did not have a direct command entry to invoke that helper. The hot-update gate lane already had direct commands for gate creation, phase progression, pointer switch, reload/apply, retry, and terminal failure resolution, but outcome creation still required direct Go-level helper access.

## Existing Control Surface

The existing operator control path is `ProcessDirect` in `internal/agent/loop.go`, backed by TaskState wrappers in `internal/agent/tools/taskstate.go`.

The adjacent hot-update commands already use this pattern:

- parse an uppercase direct command
- validate active or persisted runtime job context through TaskState
- resolve the mission store root from TaskState
- derive timestamps through the TaskState timestamp helper
- call the missioncontrol helper as `operator`
- emit a runtime control audit event
- return a deterministic changed or selected acknowledgement

## Required Boundary

The V4-052 entry must expose only the existing V4-050 helper. It must not add manual outcome fields, new mappings, promotions, pointer changes, reload generation changes, last-known-good changes, gate mutations, new gates, terminal-state inference, automatic success/failure inference, or V4-053 work.
