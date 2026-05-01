# Frank V6 Local Model Setup Wizard

**Status:** Draft V6 product target, not implemented
**Date:** 2026-05-01
**Source Basis:** Frank V5 Model Control Plane, Android phone deployment docs, and review feedback requiring deterministic planning, explicit approval, manifest-gated installs/downloads, no-prompt readiness checks, transactional config writes, and safe Termux:Boot wiring.

## Objective

Frank V6 turns V5's governed model-control layer into an operator-facing local model setup and bootstrap experience.

V5 lets Frank route to named local and cloud model profiles safely. V6 should let a user clone the repo, run one interactive command, review a concrete plan, approve it, and end with V5-compatible configuration plus an optional working local model runtime that Frank can use on startup.

Target command:

```sh
picobot models setup
```

The setup flow should detect the host, offer safe local and cloud setup paths, install or configure the selected runtime only when explicitly approved, pull or register selected models only when explicitly approved, write V5-compatible model profiles transactionally, create optional boot wiring, run no-prompt readiness checks, and report whether Frank is ready to boot with local models.

## Core Product Decision

Frank V6 owns **interactive model setup and runtime bootstrap**.

Frank V5 remains the model control plane:

- named providers,
- named model profiles,
- aliases,
- route policy,
- health checks,
- fallback policy,
- tool-schema suppression,
- mission model policy,
- route status.

Frank V6 adds a setup layer above V5:

- guided provider and model profile presets,
- typed environment snapshots,
- side-effect-free detection,
- pure deterministic planning,
- explicit plan approval,
- controlled plan execution,
- optional runtime installation,
- optional model download or local model registration,
- optional Termux:Boot startup script generation,
- post-setup no-prompt readiness validation,
- secret-safe setup reporting.

V6 must not weaken V5's safety model. V6 setup output must produce V5 config that is safe by default.

## Definition Of Frank V6

Frank V6 is complete when Frank can guide an operator through local model setup from a fresh checkout without requiring manual config editing for common paths.

Common paths:

- Android/Termux phone with Ollama,
- Android/Termux phone with llama.cpp server and a GGUF model,
- Linux desktop or localhost host with Ollama,
- Linux desktop or localhost host with llama.cpp server and a GGUF model,
- cloud-only OpenRouter or OpenAI profile stubs,
- mixed local plus cloud profiles with local/cloud fallback disabled by default.

The V6 setup command is allowed to install runtimes, download models, write config, and create boot scripts only after the user has reviewed and approved a fully resolved plan.

## Relationship To V5

V6 depends on V5 and emits ordinary V5 configuration.

V6 must use V5 concepts directly:

- `provider_ref`,
- `model_ref`,
- `modelAliases`,
- `modelRouting`,
- `localRuntimes`,
- model capabilities,
- request overrides,
- provider health/readiness,
- route inspection.

V6 must not create a parallel model registry, alternate provider format, or hidden runtime path.

V6 setup may add helper metadata for installed assets if needed, but Frank runtime behavior must continue to come from V5 config and mission policy.

## Non-Goals

- Do not make runtime installation or model download mandatory for all V6 users.
- Do not silently install runtimes.
- Do not silently download models.
- Do not silently create Termux:Boot scripts.
- Do not run opaque remote shell scripts.
- Do not guess installer URLs, model URLs, versions, or checksums.
- Do not require CGO, CUDA, Python, Node, or embedded model runtimes in Frank.
- Do not embed Ollama, llama.cpp, or model weights into the Go binary.
- Do not require real OpenAI, OpenRouter, Ollama, llama.cpp, Android, Termux, or API keys in automated tests.
- Do not collect, print, store, or transmit API keys during setup except through existing V5-safe config mechanisms.
- Do not make local tiny models tool-capable by default.
- Do not enable cloud fallback from local models by default.
- Do not open externally reachable model ports by default.
- Do not mutate existing config without conflict detection, backup, validation, and explicit approval for replacements.
- Do not change V5 mission model-policy semantics.

## Architecture Boundaries

V6 implementation must keep the following layers distinct. Tests must be able to exercise each layer independently.

### Preset Catalog

The preset catalog is a static, checked-in set of setup recipes.

Rules:

- Versioned with the repo.
- Parsed and validated without network access.
- Contains no secrets.
- Contains no unverified installer or model URLs.
- May reference manifest IDs for installer/model downloads, but automatic downloads are blocked unless the referenced manifest is checked in and valid.
- Produces V5-compatible provider/model/routing/local runtime defaults.

### Environment Snapshot

An environment snapshot is a typed record of observed facts.

It may include:

- OS and architecture,
- Termux and Termux:Boot state,
- command availability,
- relevant paths,
- existing config path and config facts,
- existing providers, model profiles, aliases, and local runtimes,
- existing boot scripts,
- known runtime processes where detectable,
- relevant ports and bind addresses,
- known model files or runtime data directories.

Tests must be able to inject an environment snapshot directly. Pure planner tests must not run the detector.

Environment state values should be typed as:

- `present`,
- `missing`,
- `unknown`,
- `unsupported`,
- `ambiguous`.

### Detector

The detector is the side-effect-free probe layer that reads the host and produces an environment snapshot.

Allowed detector actions:

- inspect files and directories,
- inspect command availability,
- inspect OS and architecture facts,
- inspect configured paths,
- inspect local port state when supported,
- parse Frank config,
- inspect existing V5 providers, models, aliases, routing, and local runtimes.

Forbidden detector actions:

- install,
- download,
- start or stop runtimes,
- write files,
- mutate config,
- create directories,
- contact model providers,
- call cloud APIs,
- pull models,
- run provider prompt requests.

### Planner

The planner is a pure deterministic function:

```text
Plan = Planner(Preset, EnvSnapshot, OperatorChoices)
```

The planner must not:

- read files,
- write files,
- run commands,
- make network calls,
- download,
- install,
- start or stop runtimes,
- mutate config,
- call model providers,
- inspect live process state.

The planner must return either:

- a fully resolved plan,
- a blocked plan with explicit reasons,
- or a request for operator choices that cannot be safely defaulted.

### Plan

A plan is a stable list of proposed steps. It is safe to print and safe to snapshot in tests.

Each plan step must declare at least:

- stable step id,
- human-readable summary,
- side effect kind,
- command to run when applicable,
- files to read or write when applicable,
- network URL when applicable,
- expected download size or disk impact when applicable,
- runtime port and bind address when applicable,
- approval requirement,
- idempotency key,
- already-present detection rule,
- rollback or cleanup behavior,
- redaction policy,
- dependencies on earlier steps.

Allowed side effect kinds:

- `none`,
- `read_file`,
- `write_config`,
- `write_boot_script`,
- `run_command`,
- `download`,
- `install_runtime`,
- `pull_model`,
- `start_runtime`,
- `health_check`,
- `route_check`.

Each step status must use deterministic values:

- `planned`,
- `skipped`,
- `already_present`,
- `manual_required`,
- `changed`,
- `failed`,
- `rolled_back`,
- `blocked`.

`manual_required` means V6 can provide safe instructions but must stop before automatic execution. Use it when a path is supportable manually but no approved manifest, detector evidence, or safe executor path exists.

`already_present` counts as success only when the detected state matches the planned safe state. Existing unsafe state must be `blocked` or require explicit remediation; it must not be silently accepted.

### Executor

The executor is the only side-effecting layer.

Rules:

- Runs only explicit approved plan steps.
- Does not reinterpret the plan.
- Does not invent missing choices.
- Does not fetch unknown URLs.
- Does not run unknown installer scripts.
- Applies idempotency checks before side effects.
- Records deterministic step statuses.
- Redacts secrets in logs and reports.
- Stops on failed required steps unless the plan explicitly marks the step optional.

## Dry-Run Semantics

Early V6 dry-run may use test-provided or detector-provided environment facts.

Pure planner tests must inject environment snapshots. They must not depend on the real host, file system outside temp dirs, live ports, installed runtimes, Android, Termux, Ollama, llama.cpp, OpenAI, OpenRouter, or API keys.

This command:

```sh
picobot models setup --dry-run --preset <preset>
```

must not:

- write files,
- execute commands,
- start runtimes,
- download,
- install,
- call provider APIs,
- send prompts,
- mutate config,
- print secrets.

Dry-run output may show detected facts only after detector support exists. Detection must remain side-effect-free.

Dry-run output must clearly distinguish:

- detected facts,
- assumed defaults,
- unresolved choices,
- proposed steps,
- required approvals,
- blocked steps.

## Non-Interactive And Approval Semantics

`--approve` never means "figure it out and do whatever is needed."

`--approve` may only execute a fully resolved non-interactive plan. If the plan has unresolved choices, blocked steps, unknown downloads, config conflicts, or unsafe defaults, execution must fail before side effects.

Non-interactive execution must fail unless all choices are explicit or deterministically defaulted by safe policy:

- preset,
- config path,
- runtime kind,
- model/profile choice,
- install/download/register-existing behavior,
- port and bind address,
- boot script behavior,
- config conflict policy,
- fallback policy,
- overwrite/force behavior.

`--approve` must not approve:

- unknown downloads,
- opaque installer scripts,
- config conflicts,
- LAN exposure,
- cloud fallback,
- local tool capability,
- medium or high authority for local models,
- replacement of existing refs or aliases,
- missing checksums,
- raw cloud secret collection outside V5 mechanisms.

Those actions require either a checked-in manifest plus explicit flags or an interactive approval prompt that names the exact risk.

## Operator UX

### Primary Command

```sh
picobot models setup
```

The command should be interactive by default and safe to abort at every stage.

Required flags:

- `--dry-run`: print detected facts and/or proposed plan without making changes.
- `--non-interactive`: require explicit flags for every unsafe or unresolved choice.
- `--preset <name>`: select a setup preset without using the menu.
- `--approve`: execute a fully resolved non-interactive plan.
- `--config <path>`: use a specific config file.
- `--force`: allow approved overwrites after backup.

Suggested additional commands:

```sh
picobot models presets list
picobot models presets inspect <preset>
picobot models local detect
picobot models local plan --preset phone-ollama-tiny
picobot models local install --preset phone-ollama-tiny --approve
```

These may be separate subcommands or implemented as phases of `picobot models setup`.

### Setup Flow

The wizard should:

1. Detect host environment.
2. Detect installed runtimes and safe/unsafe existing states.
3. Detect existing Frank config and V5 model profiles.
4. Ask which setup path the user wants.
5. Ask which model size/profile class the user wants.
6. Show a complete plan.
7. Require explicit approval.
8. Execute each approved step.
9. Run V5 validation, no-prompt readiness, and route checks.
10. Print a final redacted readiness summary.

Example:

```text
Detected:
- platform: android_termux_arm64
- Termux:Boot: present
- Ollama: missing
- llama.cpp server: missing
- Frank config: ~/.picobot/config.json

Selected preset:
- phone-ollama-tiny

Plan:
- install Ollama using checked-in manifest: ollama-termux-arm64
- pull model: qwen3:1.7b
- add provider: ollama_phone
- add model profile: local_fast
- add alias: phone -> local_fast
- set local_fast supportsTools=false authorityTier=low
- keep cloud fallback from local disabled
- create Termux:Boot script: ~/.termux/boot/frank-model-runtime
- run metadata-only readiness check
- run: picobot models route --model phone --local

No API keys will be printed.
Proceed? [y/N]
```

## Presets

V6 should ship a small checked-in preset catalog. Presets are setup recipes, not runtime policy overrides.

### Required Default-Safe Presets

- `phone-ollama-tiny`,
- `phone-llamacpp-tiny`,
- `desktop-ollama-local`,
- `desktop-llamacpp-local`,
- `cloud-openrouter`,
- `cloud-openai`,
- `mixed-local-cloud-safe`.

### Optional Explicitly Gated Presets

- `lan-llamacpp-local`.

`lan-llamacpp-local` is not a default-safe preset. It may remain in the catalog only if every LAN bind step is blocked until the operator explicitly approves the bind address and exposure risk.

All generated local runtimes must bind to `127.0.0.1` by default. Binding to `0.0.0.0` or a LAN interface is a separately approved side effect and must never be part of the default path.

Each preset must declare:

- supported platforms,
- runtime kind,
- provider ref,
- model ref,
- provider model,
- model source or manifest id,
- expected disk size,
- expected RAM range,
- default capabilities,
- default request overrides,
- local runtime health/readiness URL,
- default bind address,
- default port,
- whether boot wiring is supported,
- whether downloads are required,
- whether cloud keys are required,
- safety notes.

The preset catalog must be deterministic, versioned, and testable without network access.

## Installer And Download Policy

Automatic installer/download support is allowed only from checked-in manifests.

A manifest must include:

- source URL,
- version or immutable release identifier,
- checksum,
- size,
- license notes,
- expected platform and architecture,
- expected unpack/install command,
- safety notes.

If no safe manifest exists, V6 must emit manual instructions and stop before unsafe execution.

V6 must not:

- run opaque remote shell scripts,
- invent installer URLs,
- invent model URLs,
- guess checksums,
- download unmanifested binaries,
- download unmanifested model weights,
- treat a mutable "latest" URL as immutable proof.

Manifest-gated download execution must be fakeable in tests and must support checksum mismatch failures without network access.

## Local Runtime Installation

### Ollama

V6 may install Ollama when the operator approves the plan and the selected platform has a safe package path or checked-in installer manifest.

Installation must be platform-specific and explicit:

- Prefer trusted package manager installation when available.
- If an installer URL is used, it must come from a checked-in manifest.
- Show the source URL, version, checksum, and size before execution.
- Verify checksum where downloaded artifacts are used.
- If the platform cannot be installed safely and deterministically, generate manual instructions and stop before unsafe execution.

Model pulls should use Ollama's own model mechanism only after approval:

```sh
ollama pull qwen3:1.7b
```

After installation and pull, V6 should verify readiness without prompts:

```sh
picobot models health phone
picobot models route --model phone --local
```

### llama.cpp

V6 supports llama.cpp in this safe sequence:

1. Register an existing `llama-server` binary and existing GGUF model.
2. Later, optionally add manifest-gated binary/model downloads.

Register-existing support is the first llama.cpp implementation path. It should validate that:

- the server binary path exists,
- the GGUF model path exists,
- generated command binds to `127.0.0.1` by default,
- generated `localRuntimes` health/readiness settings are V5-compatible.

Automatic llama.cpp binary or model downloads are optional and blocked until acceptable checked-in manifests exist.

The generated default runtime command should bind to localhost:

```sh
llama-server \
  -m "$HOME/models/qwen3-1.7b-q8_0.gguf" \
  --host 127.0.0.1 \
  --port 8080
```

## Cloud Profile Stubs

Cloud presets create V5 provider/model stubs and secret references only.

Rules:

- V6 setup must not require collecting raw API key values unless the existing V5 config mechanism already supports a safe secret flow.
- Reports may show key presence/status only, such as `set`, `empty`, `from_env`, or `missing`.
- Reports must never show key values.
- Cloud fallback from local remains disabled by default.
- Enabling cloud fallback requires explicit approval and must be visible in the plan.
- Cloud presets must not run prompt-bearing provider calls during setup.

## Termux And Boot Integration

On Android/Termux, V6 should detect:

- Termux,
- CPU architecture,
- Termux:Boot availability,
- tmux availability,
- battery optimization warning state where detectable,
- existing boot scripts,
- existing Frank gateway command.

V6 may generate boot scripts only after approval.

Recommended generated files:

- `~/.termux/boot/frank-model-runtime`,
- `~/.termux/boot/frank-gateway`,
- optional `~/.picobot/frank/model-runtime.env`.

The model runtime boot script should:

- use `/data/data/com.termux/files/usr/bin/sh`,
- avoid printing secrets,
- create needed directories,
- start the runtime in tmux,
- bind local model servers to `127.0.0.1` by default,
- be idempotent when the tmux session already exists,
- report unsafe existing sessions as blocked or requiring remediation,
- not start Frank until the local model endpoint is healthy unless the user explicitly chooses independent startup.

The gateway boot script should:

- start Frank only through an explicit saved command,
- preserve existing mission-control resume guardrails,
- not bypass `--mission-resume-approved` requirements,
- not auto-approve persisted mission runtime resume after reboot.

V6 may suggest Termux wake locks, battery setting changes, and Tailscale settings, but it must not silently change Android system settings.

## Transactional Config Writes

V6 setup writes must be conservative and transactional.

Required order:

1. Read existing config.
2. Normalize refs.
3. Compute an in-memory patch.
4. Validate generated config in memory through V5 registry validation.
5. Detect provider, model, alias, routing, local runtime, and secret-field conflicts.
6. Require explicit approval for replacements.
7. Backup current config before writing.
8. Write a temp file.
9. Atomically rename where supported.
10. Validate the written file.
11. Roll back or preserve the backup on post-write validation failure.
12. Report redacted write status.

Rules:

- Normalize provider refs, model refs, aliases, and local runtime refs before comparison.
- Raw refs that normalize to the same value are the same identity.
- Detect conflicts after normalization, not before.
- Preserve existing providers, models, aliases, routing, channels, mission settings, and secrets unless the plan explicitly replaces them.
- Never print API key values or secret-bearing config fields.
- Runtime install/download/start steps must not partially mutate config unless the plan explicitly separates and approves a partial config write.
- Failed install/download should leave existing config and boot scripts unchanged by default.
- Existing unsafe state must be reported as blocked or requiring explicit remediation.

Generated local model defaults:

```json
{
  "capabilities": {
    "local": true,
    "offline": true,
    "supportsTools": false,
    "supportsStreaming": false,
    "supportsResponsesAPI": false,
    "supportsVision": false,
    "supportsAudio": false,
    "contextTokens": 4096,
    "maxOutputTokens": 1024,
    "authorityTier": "low",
    "costTier": "free",
    "latencyTier": "slow"
  }
}
```

Generated routing defaults:

- Local/cloud fallback from local is disabled unless approved.
- Lower-authority fallback is disabled.
- Local tiny model aliases may include `phone` and `local`.
- `default` remains whatever the operator explicitly selects.

## Approval Model

The wizard must show a plan before side effects.

Side effects requiring approval:

- installing Ollama,
- downloading llama.cpp,
- downloading model weights,
- pulling Ollama models,
- writing Frank config,
- writing boot scripts,
- starting or stopping runtimes,
- opening listening ports,
- changing saved gateway commands,
- enabling cloud fallback,
- overwriting existing refs or aliases,
- binding to anything other than `127.0.0.1`,
- granting tools or medium/high authority to local models.

Approval must be specific enough for the user to understand:

- what command will run,
- what files will be written,
- what network downloads will happen,
- what source, version, checksum, and size apply,
- what approximate disk space is needed,
- what runtime ports and bind addresses will be used,
- what model authority and tool permissions will be configured,
- what rollback or cleanup behavior exists.

## No-Prompt Health And Readiness

Setup health/readiness checks must not send:

- user prompts,
- message content,
- tool arguments,
- raw provider request bodies,
- secret-bearing provider requests.

Allowed readiness inputs:

- V5 registry validation,
- route inspection,
- local metadata endpoints such as `/models` or configured local health URLs,
- process checks,
- port checks,
- fake HTTP servers in tests,
- existing V5 health checks only when they are metadata-only and no-prompt.

If existing V5 health checks send test prompts or provider requests, V6 must add or use a metadata-only/no-prompt readiness path instead.

This is a top integration risk because setup should prove readiness without exposing prompts or causing provider-side work.

## Idempotence And Reporting

The setup report and executor must use deterministic statuses:

- `planned`,
- `skipped`,
- `already_present`,
- `manual_required`,
- `changed`,
- `failed`,
- `rolled_back`,
- `blocked`.

Rules:

- Re-running the same approved plan must not duplicate providers, model profiles, aliases, boot scripts, downloads, or runtime start sessions.
- `already_present` counts as success only when the detected state matches the planned safe state.
- `manual_required` is a terminal non-error status for safe manual-instruction outcomes where automatic execution is unavailable or not approved.
- Existing unsafe state is `blocked` or requires explicit remediation.
- The executor must report the idempotency key used for each side-effecting step.
- Reports must be safe to paste into support logs.

Report fields:

- selected preset,
- detected platform,
- runtime kind,
- provider ref,
- model ref,
- provider model,
- files written,
- backup paths,
- install/download status,
- health/readiness status,
- route status,
- boot integration status,
- warnings,
- next command.

Report must not include:

- API keys,
- authorization headers,
- full prompts,
- message content,
- tool arguments,
- raw provider response bodies,
- raw provider error bodies.

If report output must be truncated:

- Drop low-priority fields first, such as successful unchanged step details and verbose environment facts.
- Never drop `failed`, `blocked`, `rolled_back`, or `manual_required` diagnostics.
- Never reveal secrets while truncating.
- Make truncation deterministic so repeated runs with the same inputs produce the same retained fields.

## Safety Requirements

- No local model installed by V6 is tool-capable by default.
- No local model installed by V6 receives `authorityTier: "medium"` or `authorityTier: "high"` by default.
- No setup path enables cloud fallback from a local model by default.
- No setup path starts a model endpoint bound to `0.0.0.0` or a LAN interface by default.
- No setup path logs API keys, Authorization headers, prompts, message content, tool arguments, raw request bodies, raw response bodies, or raw provider error bodies.
- Failed downloads or installs must leave existing config and boot scripts unchanged unless the user approved a partial write.
- Setup must be restartable and idempotent where practical.
- Setup must not use model output to decide whether to approve installation actions.
- Setup must not run provider prompt requests as part of health verification.

## Tests

Automated tests must not require live services, real downloads, Android, Termux, API keys, Ollama, or llama.cpp.

Required test coverage:

- preset catalog parses and validates,
- pure planner accepts injected environment snapshots,
- pure planner has no side effects,
- dry-run plan has no side effects,
- non-interactive mode fails without full safe choices,
- `--approve` fails on unresolved or unsafe plans,
- existing config is backed up before writes,
- generated config validates through V5 registry,
- local models default to low authority and no tools,
- fallback from local to cloud is denied by default,
- LAN bind is blocked unless explicitly approved,
- boot script generation is deterministic,
- existing boot scripts are not overwritten without force or approval,
- install commands are planned but not executed in dry-run,
- fake command runner captures approved install/pull/start commands,
- failed install leaves config unchanged,
- failed download leaves config and boot scripts unchanged,
- health checks use fake HTTP servers and no prompts,
- setup reports redact fake secrets,
- Termux paths are generated correctly,
- unsupported platform returns an actionable manual-instructions result,
- normalized ref collisions are detected after normalization,
- rerunning an already-applied safe plan produces `already_present` statuses,
- unsafe existing state is blocked.

## Implementation Slices

V6 should be implemented one slice at a time. The slice order below is the intended safe linear path.

| Slice ID | Title | Objective | Acceptance Criteria |
| --- | --- | --- | --- |
| V6-001 | Preset Catalog, EnvSnapshot, And Pure Planner | Add checked-in setup presets, typed `EnvSnapshot`, typed operator choices, plan/step schema, and pure planner. No real detector or executor. Tests inject snapshots. | Planner tests use injected `EnvSnapshot` values and require no real host detection. CLI dry-run may use a minimal default/unknown `EnvSnapshot`; real detector-backed dry-run is deferred to V6-004. `picobot models setup --dry-run --preset phone-ollama-tiny` can render a deterministic plan from injected or minimal facts; tests prove planner performs no file writes, commands, network calls, runtime starts, installs, downloads, provider calls, or secret printing. |
| V6-002 | Interactive Wizard Shell And Approval UI | Add interactive selection, missing-choice prompts, plan display, abort path, and approval capture. No side effects except stdout/stderr prompts. | Scripted stdin/stdout tests cover choose preset, approve, abort, unresolved choices, and unsafe approvals rejected before executor exists. |
| V6-003 | Transactional Config Writer | Apply approved config plan steps with in-memory patching, V5 validation, conflict detection, backup, temp write, atomic rename where supported, post-write validation, and rollback/preserve-backup behavior. | Tests prove existing fields and secrets are preserved, conflicts require approval, invalid generated config is not written, backups exist, and fake secrets never print. |
| V6-004 | Runtime And Platform Detector | Add side-effect-free detector that returns typed states for platform, Termux, Termux:Boot, tmux, Ollama, llama.cpp, paths, ports, boot scripts, config, and existing V5 profiles. | Fake detector tests cover `present`, `missing`, `unknown`, `unsupported`, and `ambiguous`; detector does not install, download, write, start runtimes, or contact providers. |
| V6-005 | Ollama Executor With Fake Command Runner | Execute approved Ollama install/pull/start steps through a fakeable runner. Auto-install is allowed only through safe package paths or checked-in manifests; otherwise emit manual instructions and block execution. | Dry-run prints commands but executes none; approved fake execution records commands; missing manifest blocks auto-install; failures leave config and boot scripts unchanged by default. |
| V6-006 | llama.cpp Register-Existing Path | Support registering existing `llama-server` and GGUF model paths, generating V5 config/local runtime entries and localhost start command. | Tests validate path checks, config output, health/readiness URL, default `127.0.0.1` binding, and no automatic downloads. |
| V6-007 | Manifest-Gated llama.cpp And Model Downloads | Add optional manifest-based llama.cpp binary/model downloads with checksum validation. This slice is blocked until acceptable checked-in manifests exist. | Tests use local fake downloads and checksum mismatch paths; no real network is required; missing manifests produce blocked/manual-instructions plans. |
| V6-008 | Termux:Boot Script Generation | Generate deterministic, idempotent model-runtime and gateway boot scripts after approval, preserving existing scripts by default. | Tests compare generated scripts, refuse unsafe overwrite without approval, keep mission resume guards intact, default to localhost runtime binds, and verify shell syntax where existing tooling supports it. |
| V6-009 | No-Prompt Readiness And Route Finalization | Run metadata-only/no-prompt readiness, V5 validation, and route checks after setup; emit redacted final report. | Fake HTTP/process tests prove healthy/unhealthy paths, route status, `already_present` reporting, blocked unsafe state, and no fake secret or prompt leakage. |
| V6-010 | Android/Termux Operator Docs | Update Android docs with V6 one-command setup, dry-run, manual fallback paths, Ollama, llama.cpp register-existing, Termux:Boot, battery settings, local/cloud fallback, and recovery. | Docs explain exact safety boundaries and do not include real secrets, unverified installer URLs, or guessed checksums. |
| V6-011 | End-To-End Fake Setup Acceptance | Add E2E fake setup tests covering phone Ollama, phone llama.cpp register-existing, desktop Ollama, cloud stubs, mixed local/cloud, abort, rerun/already-present, failure recovery, and unsafe existing state. | Full fake setup scenarios pass without live services, real downloads, Android, Termux, Ollama, llama.cpp, cloud providers, or API keys. |
| V6-012 | Final V6 Acceptance Gate | Run final validation, update status docs if needed, and tag the complete V6 milestone. | Targeted tests and full repo tests pass; V6 definition of done is satisfied; remaining manual manifest/source choices are documented. |

## Stop Conditions

Stop and ask for human review if:

- installing a runtime requires running an opaque remote shell script,
- no safe checked-in manifest is available for an automatic installer/download path,
- an installer or model source URL/checksum would need to be guessed,
- Android/Termux installation behavior differs by device in a way tests cannot model,
- config writes would require a destructive migration,
- an existing config conflict cannot be resolved deterministically,
- setup would need real API keys,
- cloud secret flow would require printing or storing raw keys outside V5 mechanisms,
- setup would need live OpenAI, OpenRouter, Ollama, llama.cpp, Android, or Termux access for automated tests,
- health/readiness requires a prompt or provider request,
- LAN bind is requested without explicit approval,
- setup would expose a model endpoint beyond localhost by default,
- setup would grant local models tool capability or medium/high authority by default,
- setup would bypass mission-control reboot/resume approvals,
- a failing test cannot be fixed without changing V5 safety behavior,
- the user must choose between incompatible supported install methods.

## Risks

| Risk | Severity | Mitigation |
| --- | --- | --- |
| Existing V5 health checks may send prompts or provider requests. | High | V6 must use or add metadata-only/no-prompt readiness and test it with fake HTTP/process checks. |
| Installer manifests become stale. | High | Require checked-in manifests with immutable versions, checksums, sizes, platforms, and manual fallback instructions. |
| LAN endpoint exposure. | High | Default all local runtimes to `127.0.0.1`; block LAN bind unless explicitly approved and visible in the plan. |
| Config transaction failure. | High | Use in-memory validation, backup, temp write, atomic rename where supported, post-write validation, and rollback/preserve-backup behavior. |
| Idempotence drift between detector and executor. | High | Use stable idempotency keys and require `already_present` only when detected state matches planned safe state. |
| Cloud key handling ambiguity. | High | Cloud presets create stubs and key status only; stop if a raw-key flow would print/store secrets outside V5 mechanisms. |
| Runtime installers change upstream. | High | Prefer package managers or checked-in manifests; block unmanifested URLs and opaque scripts. |
| Model downloads are large and device-specific. | High | Show size/RAM requirements before approval; support dry-run and register-existing paths. |
| Termux:Boot scripts accidentally start unsafe work after reboot. | High | Keep mission resume approval intact; separate runtime boot from gateway boot; require explicit approval. |
| Local model gets too much authority. | High | Generated local models default to low authority and no tools; V5 still enforces tool suppression. |
| Setup leaks secrets in logs. | High | Use redacted setup reports and tests with fake secret sentinels. |
| Existing config is overwritten. | Medium | Backup before writes; refuse conflicts unless approved; validate before and after writes. |
| Tests depend on live services. | Medium | Use fake detectors, fake command runners, local temp dirs, and `httptest`. |
| Android docs drift from setup behavior. | Medium | Include docs in final acceptance and avoid hardcoding unverified installer URLs/checksums. |

## Open Questions

- Which Ollama installation sources are acceptable for Android/Termux auto-install manifests?
- Which first GGUF model sources are acceptable for allowlisted llama.cpp downloads?
- Should V6 initially support automatic llama.cpp binary download, or should V6-006 register existing binaries first and defer binary downloads until V6-007 manifests are reviewed?

## Acceptance Criteria

- The spec clearly separates preset catalog, environment snapshot, detector, pure planner, plan, and executor.
- The spec states that V6-001 is pure and side-effect-free.
- The spec hardens `--approve` and `--non-interactive` behavior.
- The spec gates `lan-llamacpp-local` behind explicit LAN bind approval.
- The spec blocks automatic installers/downloads unless checked-in manifests exist.
- The spec adds no-prompt health/readiness requirements.
- The spec adds transactional config write rules.
- The spec clarifies cloud provider stubs and secret handling.
- The spec adds deterministic idempotence/report statuses.
- The implementation slice table reflects the hardened boundaries.
- Markdown renders cleanly.

## Lowering Diff

The original V6 draft described an interactive flow that lists models, downloads and installs them, starts the runtime, configures boot behavior, and writes V5 config after approval.

This hardening pass constrains that idea by:

- separating preset catalog, environment snapshot, detector, planner, plan, and executor,
- making V6-001 pure and side-effect-free,
- tightening dry-run, `--non-interactive`, and `--approve` semantics,
- adding exact plan-step schema requirements,
- requiring localhost binding by default and gating LAN exposure,
- blocking installers/downloads unless checked-in manifests exist,
- keeping llama.cpp register-existing support ahead of automatic downloads,
- requiring no-prompt readiness checks,
- making config writes transactional,
- clarifying cloud stubs and key-status reporting,
- adding deterministic idempotence and report statuses,
- revising implementation slices to match the safer architecture.
