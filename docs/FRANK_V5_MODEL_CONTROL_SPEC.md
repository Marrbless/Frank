# Frank V5 Model Control Plane

## Objective

Frank V5 makes model selection a governed runtime concern.

Frank must support multiple named model profiles across local and remote providers, route work to the correct model based on task, step, and policy, expose health and fallback state to the operator, and prevent weak local models from accidentally receiving high-authority tool surfaces.

Target deployment:

- Phone-local model for always-on lightweight control.
- Local LAN models for private or offline work when available.
- Cloud, OpenRouter, and OpenAI models for high-reasoning or high-stakes work.
- Mission-control policy decides which models can be used for which steps and tools.

## Non-Goals

- Do not implement model downloading in the first slice.
- Do not embed llama.cpp or Ollama into the Go binary.
- Do not add Python, Node, CGO, CUDA, or a model runtime dependency.
- Do not remove the existing OpenAI-compatible provider.
- Do not break existing config files.
- Do not make local tiny models eligible for dangerous tools by default.
- Do not silently fallback from a governed local-only step to cloud unless policy explicitly permits it.
- Do not silently fallback from a high-capability model to a lower-capability model for tool execution unless policy explicitly permits it.

## Current Baseline

Current behavior before V5 is model-as-string:

- CLI model override via `-M` / `--model`.
- Default model via `agents.defaults.model`.
- Provider fallback model if no model is specified.
- One OpenAI-compatible provider config under `providers.openai`.
- Local endpoints can be manually used by changing `providers.openai.apiBase`.

This lacks named providers, named models, capabilities, aliases, model health, fallback policy, routing policy, local runtime profiles, per-step mission constraints, deterministic selection records, and status visibility.

## New Concepts

### provider_ref

A stable normalized identifier for a model-serving endpoint.

Examples:

- `openrouter`
- `openai`
- `ollama_phone`
- `llamacpp_phone`
- `ollama_lan`
- `llamacpp_lan`

Rules:

- Lowercase.
- Trim whitespace.
- Allowed chars: `a-z`, `0-9`, `_`, `-`.
- Reject empty.
- Reject path separators.
- Duplicate normalized refs must be handled deterministically.

### model_ref

A stable normalized identifier for a logical model profile.

Examples:

- `local_fast`
- `local_reasoning`
- `cloud_reasoning`
- `cloud_coding`
- `router_default`
- `qwen3_1_7b_phone`
- `gemini_flash_cloud`

Rules:

- Lowercase.
- Trim whitespace.
- Allowed chars: `a-z`, `0-9`, `_`, `-`.
- Reject empty.
- Reject path separators.
- Must point to exactly one `provider_ref`.
- Must include `providerModel`, the actual model name sent to the backend.

### model alias

A human-friendly name that resolves to a `model_ref`.

Examples:

- `default -> cloud_reasoning`
- `phone -> qwen3_1_7b_phone`
- `cheap -> local_fast`
- `best -> cloud_reasoning`
- `code -> cloud_coding`

Rules:

- Alias resolution must be deterministic.
- Alias chains are forbidden in the first slice.
- If an alias and model_ref conflict, model_ref wins and validation fails deterministically in V5-001.
- Unknown alias or model_ref fails closed.

### model capabilities

Each model profile declares capabilities:

```json
{
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
```

Allowed authority tiers:

- `low`: may answer, summarize, classify, route, and request approval; no filesystem, exec, network, or account tools unless mission explicitly grants and model policy allows.
- `medium`: may use bounded read-only tools and low-risk local tools.
- `high`: may use the full mission-approved tool surface, still bounded by mission `allowed_tools` and approvals.

Allowed cost tiers:

- `free`
- `cheap`
- `standard`
- `expensive`

Allowed latency tiers:

- `fast`
- `normal`
- `slow`
- `very_slow`

## Routing

A model route is a deterministic policy result:

```json
{
  "selected_model_ref": "local_fast",
  "provider_ref": "llamacpp_phone",
  "provider_model": "qwen3-1.7b-q8_0",
  "selection_reason": "mission_step_default",
  "fallback_depth": 0,
  "policy_id": "default"
}
```

Routing inputs:

- explicit CLI `-M` / `--model`
- explicit channel command, in a later slice
- mission step model policy
- task kind
- required capabilities
- required authority tier
- local/offline preference
- provider health
- fallback policy

The selected model route must be recordable in runtime status and testable without a network call.

## Config Schema

V5 adds backward-compatible config fields:

- `providers.<provider_ref>` for named OpenAI-compatible endpoints.
- `models.<model_ref>` for logical profiles.
- `modelAliases` for deterministic alias resolution.
- `modelRouting` for defaults and fallback policy.
- `localRuntimes` for external runtime profiles modeled for later process supervision.

Existing config files remain valid. If `models` is absent, V5-001 builds an implicit legacy model profile from:

- `agents.defaults.model`
- `providers.openai.apiBase`
- `providers.openai.apiKey`
- existing max token, temperature, timeout, `useResponses`, and `reasoningEffort` settings.

Existing `-M some-model-name` remains valid. If it does not resolve to a V5 `model_ref` or alias, it is treated as a raw provider model on the legacy default provider when the legacy profile is active.

## Mission-Control Integration

Later V5 slices add optional model policy fields to jobs and/or steps:

```json
{
  "model_policy": {
    "allowed_models": ["local_fast", "cloud_reasoning"],
    "default_model": "local_fast",
    "required_capabilities": {
      "supportsTools": false,
      "authorityTierAtLeast": "low",
      "local": true
    },
    "allow_fallback": false,
    "allow_cloud": false
  }
}
```

Rules:

- Step policy narrows job policy.
- Job policy narrows config routing.
- Missing policy preserves existing behavior.
- `allowed_tools` and model authority are both required gates.
- A model with `supportsTools=false` must receive no tool definitions.
- A low-authority model must not receive high-risk tools unless policy explicitly grants that model/tool pairing.
- Fallback must be recorded.
- Fallback must not cross local/cloud boundary unless policy allows it.
- Fallback must not lower authority tier for tool-using steps unless policy allows it.

## Provider Health

V5 defines this CLI surface:

```sh
picobot models list
picobot models inspect local_fast
picobot models route --task chat
picobot models route --task tool --requires-tools
picobot models health
picobot models health local_fast
```

V5-001 implements read-only config commands:

```sh
picobot models list
picobot models inspect <model_ref_or_alias>
picobot models route [--model X] [--local] [--requires-tools]
```

Health checks are reserved for a later slice.

Health status shape:

```json
{
  "model_ref": "local_fast",
  "provider_ref": "llamacpp_phone",
  "status": "unknown",
  "last_checked_at": "2026-05-01T00:00:00Z",
  "last_error_class": "",
  "fallback_available": true
}
```

Health output must not leak API keys or raw secret-bearing request data.

## Runtime Status

Mission/operator status should eventually expose selected model route without leaking secrets:

```json
{
  "model": {
    "selected_model_ref": "local_fast",
    "provider_ref": "llamacpp_phone",
    "provider_model": "qwen3-1.7b-q8_0",
    "selection_reason": "mission_step_default",
    "fallback_depth": 0,
    "policy_id": "step:model_policy",
    "capabilities": {
      "local": true,
      "offline": true,
      "supportsTools": false,
      "authorityTier": "low"
    }
  }
}
```

Do not include API keys, full request bodies, message content, tool arguments, or raw provider error bodies.

## Local Runtime Profiles

V5 models local runtimes as external services in config. The first slice does not manage processes.

```json
{
  "localRuntimes": {
    "llamacpp_phone": {
      "kind": "external_http",
      "provider": "llamacpp_phone",
      "expectedBaseURL": "http://127.0.0.1:8080/v1",
      "startCommand": "",
      "healthURL": "http://127.0.0.1:8080/health",
      "notes": "Started separately in Termux/tmux."
    },
    "ollama_phone": {
      "kind": "external_http",
      "provider": "ollama_phone",
      "expectedBaseURL": "http://127.0.0.1:11434/v1",
      "startCommand": "",
      "healthURL": "http://127.0.0.1:11434/api/tags",
      "notes": "Started separately by Ollama."
    }
  }
}
```

## Determinism Requirements

- Normalize all refs.
- Reject invalid refs.
- Resolve aliases deterministically.
- Choose default and fallback routes deterministically.
- Report no-route cases deterministically.
- Never silently skip invalid model configs.
- Never silently downgrade tool capability.
- Never silently cross cloud/local boundary.
- Keep status output stable enough for golden tests.

## Security Requirements

- Secrets must never appear in logs, status JSON, CLI inspect output, health output, or tests.
- Local endpoint URLs are not secrets.
- API key presence may be reported as set, empty, or from_env; never show the value.
- Model route records may show `provider_ref` and `provider_model`.
- Raw provider error body must not be logged.
- Tool arguments must not be logged by model-routing code.
- A local model profile defaults to `authorityTier=low` and `supportsTools=false`.
- A model that does not support tools must receive zero tool definitions.
- A model that is not eligible for a tool must not see that tool schema.

## Metrics

Future slices should populate:

- `route_attempt_count`
- `route_success_count`
- `route_failure_count`
- `fallback_count`
- `provider_health_failure_count`
- `model_policy_denial_count`
- `tool_schema_suppressed_count`

Definitions:

- A route attempt is one model selection for one agent turn.
- A fallback is counted only when the initially selected model is replaced by another model.
- A model policy denial occurs when config or mission requests a model that violates capability or policy constraints.
- A tool schema suppression occurs when tools are available in the registry but withheld from the provider because the selected model cannot receive tools.

## V5-001: Config Model Registry and Resolver

The first implementation slice adds:

- Config structs for named providers, named models, aliases, routing defaults, fallbacks, capabilities, request overrides, and local runtime profiles.
- Normalization and validation helpers.
- Resolver that accepts explicit model string, default config, required capabilities, local preference, and fallback allowance.
- Backward-compatible legacy config support.
- Read-only CLI commands for list, inspect, and route.
- Unit tests for normalization, alias resolution, fallback guardrails, invalid configs, request overrides, and legacy behavior.

V5-001 intentionally does not add:

- model downloads,
- local runtime process supervision,
- live health checks,
- mission job/step `model_policy` schema,
- full gateway/provider routing across named providers,
- status snapshot persistence of selected model routes.
