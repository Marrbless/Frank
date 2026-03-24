# Frank v1 How To Use

## What Frank v1 is

Frank v1 is the narrow, governed mission-control slice that exists in the current Picobot runtime.

In practice, v1 means:

- one bootstrapped mission job at a time
- one active step at a time
- tool use gated by the current mission step
- validation before a step is treated as complete
- fail-closed behavior when `--mission-required` is enabled

This is not a general autonomous operator. It is a bounded mission runner with a small control surface.

## Desktop vs phone

Use the model from [`docs/FRANK_V2_SPEC.md`](./FRANK_V2_SPEC.md):

- desktop = lab
  - author missions
  - inspect mission JSON
  - test step transitions
  - evolve runtime code
- phone = deployed body
  - run `picobot gateway`
  - expose the current mission status snapshot
  - accept explicit step switches through the control file path
  - stay smaller and more constrained than desktop

For v1, treat the phone as the thing that should execute only what the active mission step allows, and nothing else.

## Safe default phone posture

The safe default is fail-closed:

- start the phone gateway with `--mission-required`
- keep the mission status snapshot enabled with `--mission-status-file`
- use a step whose exposed `allowed_tools` matches the minimum you want
- when you want the phone unable to take side-effecting actions, verify `allowed_tools` is `[]`

`allowed_tools=[]` is the simplest zero-tools posture check in the current runtime.

## Core concepts

### Job

A job is one bounded mission. It has:

- a job ID
- a max authority ceiling
- a job-level allowed tool set
- a plan with ordered steps

### Step

A step is the current governed slice of that job. The active step controls:

- what kind of work is being done
- what tools are exposed
- whether approval is required
- what success criteria apply

### Mission-required

`--mission-required` means the runtime will not allow governed tool execution without an active mission step.

If the phone is in mission-required mode and no active step is present, tool execution fails closed.

### Mission status

The mission status snapshot is the operator-facing JSON view written by `--mission-status-file`.

It tells you at least:

- whether mission-required is on
- whether a step is active
- the current `job_id`
- the current `step_id`
- the current `step_type`
- the current `required_authority`
- whether the step `requires_approval`
- the effective `allowed_tools`
- the embedded runtime state, when present

### Step control

Step control is file-based in v1.

You switch steps by writing a JSON control file with `picobot mission set-step`, and the gateway watches that file when started with `--mission-step-control-file`.

### Reboot resume approval

If the status snapshot contains a persisted non-terminal runtime, the gateway will not resume it after reboot unless you start with `--mission-resume-approved`.

Without that flag, startup fails with `resume_approval_required`.

## Practical commands

Examples below use these placeholders:

```sh
MISSION_FILE=/absolute/path/to/mission.json
STATUS_FILE=/absolute/path/to/mission-status.json
CONTROL_FILE=/absolute/path/to/mission-step-control.json
START_STEP_ID=/exact/start-step-id-from-mission
TARGET_STEP_ID=/exact/target-step-id-from-mission
```

### Check status

Print the current status snapshot:

```sh
./picobot mission status --status-file "$STATUS_FILE"
```

### Assert safe posture

Strict zero-tools check:

```sh
./picobot mission assert --status-file "$STATUS_FILE" --active --no-tools
```

Manual check:

```sh
./picobot mission status --status-file "$STATUS_FILE"
```

Look for:

```json
"allowed_tools": []
```

### Restart the phone gateway

Normal mission-required startup:

```sh
./picobot gateway \
  --mission-required \
  --mission-file "$MISSION_FILE" \
  --mission-step "$START_STEP_ID" \
  --mission-status-file "$STATUS_FILE" \
  --mission-step-control-file "$CONTROL_FILE"
```

If you are using a wrapper, service, or Termux boot script on the phone, restart that wrapper with the same arguments rather than inventing a second launch path.

### Approved resume after reboot

Use this only when you intentionally want the persisted runtime to resume:

```sh
./picobot gateway \
  --mission-required \
  --mission-file "$MISSION_FILE" \
  --mission-step "$START_STEP_ID" \
  --mission-status-file "$STATUS_FILE" \
  --mission-step-control-file "$CONTROL_FILE" \
  --mission-resume-approved
```

If the persisted runtime step does not match `--mission-step`, startup fails rather than silently choosing between them.

### Inspect a mission

Inspect the whole mission:

```sh
./picobot mission inspect --mission-file "$MISSION_FILE"
```

Inspect one step:

```sh
./picobot mission inspect --mission-file "$MISSION_FILE" --step-id "$TARGET_STEP_ID"
```

This is the best way to confirm valid step IDs, step type, approval requirement, and effective tool scope before switching anything on the phone.

### Switch steps

Write the step-control file and wait for fresh status confirmation:

```sh
./picobot mission set-step \
  --control-file "$CONTROL_FILE" \
  --mission-file "$MISSION_FILE" \
  --status-file "$STATUS_FILE" \
  --step-id "$TARGET_STEP_ID"
```

If you want an explicit wait budget:

```sh
./picobot mission set-step \
  --control-file "$CONTROL_FILE" \
  --mission-file "$MISSION_FILE" \
  --status-file "$STATUS_FILE" \
  --step-id "$TARGET_STEP_ID" \
  --wait-timeout 5s
```

## What v1 supports

- one bootstrapped mission job from a mission JSON file
- one active step at a time
- step inspection and file-based step switching
- mission status snapshot output
- fail-closed mission-required gating
- approval-required flagging in mission status
- reboot resume protection through `--mission-resume-approved`
- current runtime step types:
  - `discussion`
  - `static_artifact`
  - `one_shot_code`
  - `final_response`

## What v1 does not support

- multiple active governed jobs on the phone
- a full durable multi-job control plane
- text-channel `APPROVE`, `DENY`, `PAUSE`, `RESUME`, `ABORT`, `STATUS`, `SET_STEP` commands as a frozen runtime contract
- `long_running_code` as a first-class step type
- `system_action` as a first-class step type
- `wait_user` as a first-class mission step type
- parallel governed execution
- broader product expansion beyond the narrow mission-control slice

## Troubleshooting

### Gateway not running

If `picobot mission status --status-file "$STATUS_FILE"` fails because the file is missing, the gateway may not be running or may have started without `--mission-status-file`.

Practical checks:

```sh
pgrep -af "picobot gateway"
```

If nothing is running, restart the gateway with the exact mission flags you expect.

### Stale status snapshot vs live process

The status file is just a snapshot on disk. It is not proof that the gateway is still live.

Signs of stale status:

- `updated_at` is old
- the step shown in the file does not change after `mission set-step`
- the process is gone but the file is still present

Use both checks together:

```sh
pgrep -af "picobot gateway"
./picobot mission status --status-file "$STATUS_FILE"
```

### `resume_approval_required`

This means the gateway found a persisted non-terminal runtime in the status snapshot and refused to resume it automatically.

If resume is intended, restart with:

```sh
./picobot gateway \
  --mission-required \
  --mission-file "$MISSION_FILE" \
  --mission-step "$START_STEP_ID" \
  --mission-status-file "$STATUS_FILE" \
  --mission-step-control-file "$CONTROL_FILE" \
  --mission-resume-approved
```

If resume is not intended, inspect the snapshot first and decide whether to clear or replace the runtime state through your normal ops process before restarting.

### Zero-tools safe posture check

Fast helper:

```sh
./picobot mission assert --status-file "$STATUS_FILE" --active --no-tools
```

Raw check:

```sh
./picobot mission status --status-file "$STATUS_FILE"
```

If `allowed_tools` is not `[]`, the phone is not in zero-tools posture.
