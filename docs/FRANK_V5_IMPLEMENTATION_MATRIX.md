# Frank V5 Implementation Matrix

Frank V5 is the Model Control Plane upgrade. This matrix is the execution
contract for finishing V5 after the V5-001 config and resolver slice. It is
based on repo inspection of the current V5-001 work, the V5 spec, and the
existing config, provider, agent, CLI, gateway, and mission-control code.

## 1. Current Verified Baseline

### Confirmed implemented

- V5 spec exists in `docs/FRANK_V5_MODEL_CONTROL_SPEC.md`.
- Config schema contains V5 model-control fields in `internal/config/schema.go`:
  `models`, `modelAliases`, `modelRouting`, and `localRuntimes`.
- Provider config supports the legacy `providers.openai` entry plus named
  provider entries through `internal/config/providers_json.go`.
- Model registry and resolver exist in `internal/config/model_registry.go`:
  normalized provider refs, normalized model refs, aliases, model profiles,
  local defaults, route requirements, fallback policy checks, request override
  resolution, and legacy implicit profile behavior.
- Resolver tests exist in `internal/config/model_registry_test.go` for legacy
  behavior, normalization, invalid refs, aliases, alias chains, unknown
  providers, local defaults, fallback denial, authority fallback denial, request
  overrides, and raw legacy `-M` model handling.
- Read-only model CLI commands exist in `cmd/picobot/main_models_commands.go`:
  `picobot models list`, `picobot models inspect`, and
  `picobot models route`.
- CLI tests exist in `cmd/picobot/main_models_commands_test.go` for model list,
  inspect, route, legacy raw model routing, API key redaction, and invalid
  registry validation.
- Startup config validation calls `config.BuildModelRegistry` from
  `cmd/picobot/main_gateway_startup_safety.go`, so invalid V5 registry config
  fails validation.
- Existing OpenAI-compatible provider supports Chat Completions and Responses
  API paths in `internal/providers/openai.go`.
- Existing mission-control runtime already has governed jobs and steps,
  allowed tool fields, approval records, durable store behavior, status
  snapshots, and fail-closed validation across `internal/missioncontrol`.
- Android/Termux deployment docs exist in `docs/ANDROID_PHONE_DEPLOYMENT.md`,
  but they still describe the legacy `providers.openai` endpoint override path.

### Partially implemented

- Resolver output can say tool definitions are suppressed, but live agent calls
  do not yet suppress tool schemas based on the selected model.
- Resolver fallback rules are unit-tested with deterministic unavailable model
  inputs, but live execution does not yet use provider health or record fallback
  routes.
- Request override resolution exists in the resolver, but runtime provider calls
  still use the provider created from `providers.openai` and the single selected
  model string.
- Local runtime profiles are represented in config, but no operator docs or
  runtime health integration use them yet.
- Model CLI output is secret-safe for inspected config, but live provider
  errors, health output, and runtime status do not yet expose model-control
  state.
- Mission-control status surfaces exist, but they do not yet include selected
  model route data.

### Not implemented

- Named V5 providers are not constructed for live agent or gateway calls.
  `internal/providers/factory.go` still builds only from `cfg.Providers.OpenAI`
  or falls back to the stub provider.
- Live agent and gateway startup do not route through the V5 resolver.
  `cmd/picobot/main.go` still chooses `-M`, then `agents.defaults.model`, then
  `provider.GetDefaultModel()`.
- `providerModel` is not yet the runtime model sent to the backend for named
  V5 model profiles.
- Per-model request overrides are not applied to live provider calls.
- `supportsTools=false` does not yet cause zero tool schemas at provider-call
  time. `internal/agent/loop.go` still passes active tool definitions directly
  to `provider.Chat`.
- No `picobot models health` command exists.
- No provider/model health checker or health error classifier exists.
- No mission job or step `model_policy` schema exists in
  `internal/missioncontrol/types.go`.
- No model policy enforcement exists for mission job or step execution.
- No live local/cloud fallback enforcement exists outside resolver tests.
- No runtime route object is persisted or exposed in operator status snapshots.
- No V5 model-control metrics or counters are implemented.
- No Android/Termux V5 local llama.cpp or Ollama operational guide exists.
- No end-to-end fake-provider tests prove named-provider runtime behavior,
  tool-schema suppression, fallback denial, policy denial, or secret redaction.

### Uncertain / needs inspection during implementation

- The safest exact status location for the selected route needs slice-level
  inspection. Candidate surfaces include mission-control operator summaries,
  gateway status snapshots, and committed mission status JSON. Existing gateway
  status tests lock current output shape.
- Runtime request override application may require a provider option layer or a
  per-call request options API. The current provider interface accepts only
  context, messages, tool definitions, and model string.
- Temperature override support needs provider-level inspection. The config and
  resolver carry temperature, but the current OpenAI request path does not
  appear to send temperature.
- Authority-tier enforcement needs to align with existing mission-control tool
  authority concepts. The first enforceable minimum can be conservative, but a
  richer tool-risk taxonomy may need a later reviewed design.

## 2. V5 Definition of Done

Frank V5 is complete only when all criteria below are met.

Functional criteria:

- Named model profiles can be selected and used at runtime.
- Named providers can be selected and used at runtime.
- Local OpenAI-compatible endpoints can be used through named providers without
  pretending to be `providers.openai`.
- Model aliases resolve correctly at runtime.
- `providerModel` is the model string sent to the backend.
- Per-model request overrides are applied at runtime for supported fields.
- Legacy config files and legacy CLI `-M` raw model overrides still work.
- Provider/model health and readiness can be inspected from the CLI without
  requiring live external services in tests.
- Mission jobs and steps can restrict allowed models, default models, required
  capabilities, fallback behavior, and cloud usage.
- Fallback behavior is deterministic, policy-bound, and recorded.

Safety criteria:

- `supportsTools=false` causes zero tool schemas to reach the provider.
- `supportsTools=true` preserves the existing tool behavior, still bounded by
  mission `allowed_tools` and approvals.
- Model authority and capability policy is enforced at least to the documented
  V5 minimum.
- Weak local models do not receive high-authority or unapproved tool surfaces.
- Local/cloud fallback never crosses the boundary unless policy allows it.
- Fallback from high authority to lower authority for tool-using work never
  happens unless policy allows it.
- Fallback is allowed during preflight route selection only. Automatic
  retry/fallback after a provider request has started is disabled unless the
  request is proven read-only and idempotent, and no tool call or side effect
  has occurred.
- Unknown aliases, unknown model refs, invalid refs, and invalid providers fail
  closed.
- Secret-bearing values never appear in logs, status JSON, CLI output, test
  failures, or health output.

Operator UX criteria:

- `picobot models list`, `inspect`, `route`, and `health` provide deterministic
  operator-readable output.
- Selected model route is visible in safe operator status.
- Health failures report provider ref, model ref, status, and error class
  without raw provider bodies or secrets.
- Android/Termux docs explain phone-local llama.cpp and Ollama operation using
  V5 named providers and model profiles.
- OpenAI and OpenRouter examples show cloud model configuration without leaking
  real credentials.

Test and documentation criteria:

- Unit tests cover legacy config, V5 config, aliases, named providers, local
  endpoints, request overrides, fallback allowed/denied paths, model policy
  denial paths, and tool-schema suppression.
- End-to-end fake-provider tests cover runtime model selection without real
  OpenAI, OpenRouter, Ollama, or llama.cpp servers.
- Docs cover migration, config examples, local runtime setup, cloud examples,
  health checks, status output, and final acceptance steps.
- Full repo tests pass with `/usr/local/go/bin/go test -count=1 ./...`.

## 3. Implementation Matrix

V5-001 is treated as already complete in this matrix. The table below starts
with the next implementation slice.

| Slice ID | Title | Area | Objective | Dependencies | Files Likely Touched | Acceptance Criteria | Required Tests | Safety/Security Notes | Status | Evidence | Commit/Tag |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| V5-002 | Runtime Route Selection And Named Provider Construction | code, provider/runtime, CLI | Wire the V5 resolver into agent and gateway startup and construct the selected named provider for live calls. Preserve the legacy path and raw `-M` behavior. | V5-001 | `cmd/picobot/main.go`, `internal/providers/factory.go`, `internal/providers/*`, `cmd/picobot/*_test.go`, `internal/providers/*_test.go` | Agent and gateway startup resolve `-M`, alias, model ref, or default through the registry. Named provider config is used for V5 profiles. Legacy config still starts with `providers.openai`. Raw legacy `-M some-provider-model` still maps to the legacy provider. `providerModel` is passed as the runtime model. | Unit tests for named OpenAI-compatible provider construction, alias startup route, raw legacy `-M`, missing provider failure, and no API key in errors. | Do not log API keys. Do not change provider behavior for legacy config. Do not require network or real credentials. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `9fd152a`; tag none |
| V5-003 | Tool Schema Suppression At Provider Call Time | code, provider/runtime, tests | Enforce resolver capability output so models with `supportsTools=false` receive zero tool definitions in live `provider.Chat` calls. Preserve existing tool behavior for capable models. | V5-002 | `internal/agent/loop.go`, `internal/agent/tools/*`, `internal/providers/*_test.go`, `internal/agent/*_test.go` | A selected model with `supportsTools=false` receives no tool schemas. A selected model with `supportsTools=true` receives the same mission-filtered schemas as before. Tool suppression is test-visible and does not remove mission approval checks. | Agent loop tests with fake provider capturing tool definitions for local no-tool model and cloud tool-capable model. Regression test for mission `allowed_tools` still applying. | This is the first safety gate after live routing. Tool arguments and message content must not be logged by routing code. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`; milestone `/usr/local/go/bin/go test -count=1 ./...`. | Tag `frank-v5-003-tool-schema-suppression` |
| V5-004 | Runtime Request Overrides | code, provider/runtime, tests | Apply resolved model request overrides to live provider calls for max tokens, timeout, Responses API selection, reasoning effort, and temperature where supported. | V5-003 | `internal/providers/openai.go`, `internal/providers/provider.go`, `internal/agent/loop.go`, `cmd/picobot/main.go`, provider tests | Per-model override values affect the actual HTTP request or call timeout. Legacy defaults are unchanged. Unsupported override fields are either implemented or explicitly rejected in validation with tests. | Fake HTTP server tests capture request JSON for max tokens, temperature, Responses API path, and reasoning effort. Timeout tests use deterministic short contexts without external services. | Avoid mutating shared provider state across concurrent agent turns. Redact request bodies in errors. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `362bb1d`; tag none |
| V5-005 | Route Record Object In Agent Runtime | code, operator UX, tests | Carry the selected route, request overrides, and tool suppression result through the agent runtime as a safe route record for later status and metrics. | V5-003, V5-004 | `internal/agent/loop.go`, `internal/config/model_registry.go`, `internal/agent/*_test.go` | Agent runtime stores selected model ref, provider ref, provider model, selection reason, fallback depth, policy id, safe capabilities, request override summary, and tool-schema suppression boolean. The record contains no messages, tool args, request bodies, or secrets. | Unit tests inspect route record for legacy raw model, alias model, local model, and tool suppression. | Safe record shape should be stable enough for golden tests. Keep provider URLs only where already considered non-secret. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `6409b85`; tag none |
| V5-006 | Safe Operator Status Route Visibility | code, mission-control, operator UX, tests | Expose selected model route in safe mission/operator status without leaking secrets. | V5-005 | `internal/missioncontrol/status.go`, `internal/missioncontrol/gateway_status.go`, `cmd/picobot/main_mission_status.go`, status tests | Operator status includes a `model` object with selected model ref, provider ref, provider model, selection reason, fallback depth, policy id, and safe capabilities. No API keys, prompts, tool arguments, or raw provider errors appear. Existing status tests are updated intentionally. | Golden/status tests for new model object and redaction. Legacy status still works when no V5 route exists. | Adding fields to gateway/status JSON is an operator-visible schema change; keep additive and documented. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `94fb04e`; tag none |
| V5-007 | Read-Only Model Health CLI | CLI, provider/runtime, tests | Add `picobot models health [model_ref_or_alias]` with deterministic health checks and secret-safe output. | V5-002 | `cmd/picobot/main_models_commands.go`, `internal/config/model_registry.go`, `internal/providers/health.go`, CLI tests | Health command reports model ref, provider ref, status, last checked time, error class, and fallback availability. It checks `/models` or configured local health URLs when available. It does not probe chat completions unless an explicit flag is added in this slice. | Fake HTTP server tests for healthy `/models`, timeout, auth error, HTTP error, schema error, connection failure, alias input, legacy profile, and no API key leak. | Never print API keys, authorization headers, full request URLs with credentials, raw response bodies, prompts, or tool args. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `ecfe034`; tag none |
| V5-008 | Health-Aware Preflight Runtime Fallback | code, provider/runtime, tests | Use deterministic health or availability inputs during preflight route selection and enforce configured fallback policy before any provider request starts. | V5-007 | `internal/config/model_registry.go`, `cmd/picobot/main.go`, `internal/agent/loop.go`, provider health tests | If the preferred model is unavailable before request start and fallback is allowed, runtime selects the first valid configured fallback and records fallback depth. If fallback is disabled or violates local/cloud or authority policy, route selection fails closed. No automatic fallback or replay occurs after a provider request begins unless idempotence is proven and no tool call or side effect occurred. | Tests for fallback allowed, fallback disabled, cloud fallback denied from local, lower-authority fallback denied, fallback record values, and no post-request retry after a fake provider returns an error. | Do not silently cross local/cloud boundary. Do not silently downgrade authority for tool-using steps. Default fallback must be preflight-only to avoid duplicate side effects. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`; `/usr/local/go/bin/go test -count=1 ./...`. | Tag `frank-v5-008-preflight-fallback` |
| V5-009 | Mission Model Policy Schema And Validation | code, mission-control, tests | Add optional job and step `model_policy` schema with validation, without changing live routing behavior yet. | V5-001 | `internal/missioncontrol/types.go`, `internal/missioncontrol/validation*`, mission-control tests, docs/spec updates if needed | Job and step JSON can include `model_policy`. Step policy narrows job policy by schema semantics. Invalid model refs, unknown capability fields, invalid authority tiers, and contradictory policy fields fail validation. Missing policy preserves existing behavior. | Mission JSON decode/encode tests and validation tests for valid policy, invalid ref, invalid authority, step narrowing, and missing policy compatibility. | Do not migrate durable records destructively. Additive JSON fields only. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol` (rerun with escalation after sandbox Go cache read-only false failure); `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `4d2a149`; tag none |
| V5-010 | Mission Policy Enforcement In Routing | code, mission-control, provider/runtime, tests | Enforce job and step `model_policy` against global routing for mission execution. | V5-008, V5-009 | `internal/missioncontrol/*`, `internal/agent/loop.go`, `cmd/picobot/main.go`, resolver tests | Step policy narrows job policy, and job policy narrows global config. `allowed_models`, `default_model`, `required_capabilities`, `allow_fallback`, and `allow_cloud` affect preflight route selection. Policy denials fail closed and are recorded safely. | Tests for allowed model, denied model, step default, job default, required local, required tools, allow cloud false, allow fallback false, and policy denial status. | Unknown model or alias in policy must fail closed. No fallback should bypass mission policy. Mission fallback remains preflight-only unless a later reviewed slice proves idempotence. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/config`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `2a27433`; tag none |
| V5-011 | Tool Risk Taxonomy Discovery | code, mission-control, tests | Inspect and codify the existing tool metadata needed for authority-tier enforcement before broad enforcement is attempted. | V5-003, V5-010 | `internal/agent/tools/*`, `internal/missioncontrol/*`, discovery tests or docs notes in this matrix | Existing tool definitions are inventoried for risk-relevant metadata such as filesystem, exec, network, account, read-only, write, and side-effect behavior. If enough metadata exists, produce a narrow tested classifier. If metadata is insufficient, stop with an evidence-backed gap report and do not invent broad authority semantics. | Tests for any classifier added, plus a checked inventory assertion or documented evidence mapping existing tool metadata to risk classes. | Codex must not infer broad authority semantics from names alone when metadata is insufficient. Human review is required before adding new broad tool-risk concepts. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`. Added a conservative built-in tool risk classifier; dynamic MCP tools remain unclassified for fail-closed V5-012 handling. | Commit `b270eb6`; tag none |
| V5-012 | Authority-Tier Minimum Enforcement | code, mission-control, provider/runtime, tests | Enforce the V5 minimum model authority rules against tool use and mission step authority using only the taxonomy proven in V5-011. | V5-011 | `internal/config/model_registry.go`, `internal/agent/tools/*`, `internal/missioncontrol/*`, agent tests | Low authority models cannot receive tools classified as filesystem, exec, network, account, write, or high-risk unless an explicit reviewed model policy grant exists. Medium authority receives only bounded read-only or low-risk local tools where V5-011 proves classification is available. High authority remains bounded by mission `allowed_tools` and approvals. | Tests for low authority no dangerous tools, medium bounded tools where classifiable, high authority unchanged, explicit grant path if implemented, and stop/denial behavior when classification is missing. | If V5-011 shows metadata is insufficient, this slice must stop or narrow scope instead of inventing unverified semantics. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./internal/agent/tools`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`; `/usr/local/go/bin/go test -count=1 ./...`. | Tag `frank-v5-012-authority-tier-enforcement` |
| V5-013 | Route Metrics And Denial Counters | code, operator UX, tests | Add V5 route and safety counters where existing status/metrics patterns allow. | V5-005, V5-006, V5-010, V5-012 | `internal/missioncontrol/status.go`, `internal/agent/loop.go`, status tests | Expose or persist counters for route attempts, successes, failures, fallbacks, provider health failures, model policy denials, authority denials, and tool schema suppression. Counters are deterministic in tests. | Unit/status tests for each counter increment path. | Counters must not include prompts, request bodies, API keys, tool args, or raw provider errors. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `f3cb828`; tag none |
| V5-014 | Provider Health Cache And Readiness Status | code, provider/runtime, operator UX, tests | Add a bounded in-memory health cache/readiness surface for configured providers and models, usable by CLI and status. | V5-007, V5-013 | `internal/providers/health.go`, `internal/missioncontrol/status.go`, `cmd/picobot/main_models_commands.go` | Health checks have stable status values, checked-at timestamps, error classes, and fallback availability. Status can show `unknown`, `healthy`, `unhealthy`, or `disabled` without making every status read perform network I/O. | Tests for cache hit, cache refresh, disabled provider, unhealthy model, and status redaction. | Cache must not hide current policy denial. Health output must not show secrets or raw provider bodies. | Complete | Passed `git diff --check`; `/usr/local/go/bin/go test -count=1 ./internal/providers`; `/usr/local/go/bin/go test -count=1 ./internal/agent`; `/usr/local/go/bin/go test -count=1 ./internal/missioncontrol`; `/usr/local/go/bin/go test -count=1 ./cmd/picobot`. | Commit `160e22b`; tag none |
| V5-015 | Android And Local Runtime Docs | docs | Document V5 local model operation for Android/Termux, llama.cpp, Ollama, OpenRouter, and OpenAI. | V5-007, V5-008 | `docs/ANDROID_PHONE_DEPLOYMENT.md`, `docs/CONFIG.md`, `docs/FRANK_V5_MODEL_CONTROL_SPEC.md`, new local runtime docs if needed | Docs show named provider examples for `llamacpp_phone`, `ollama_phone`, OpenRouter, and OpenAI. Docs show phone-local model profiles, aliases, local runtime profiles, startup commands, health checks, fallback warnings, and no-real-secret placeholders. | Docs diff review. Optional command examples must be syntactically valid where practical. | Use placeholders only. Do not publish real keys. Keep local endpoint URLs non-secret. | Complete | Passed `git diff -- docs/ANDROID_PHONE_DEPLOYMENT.md docs/CONFIG.md docs/FRANK_V5_MODEL_CONTROL_SPEC.md`; `git diff --check`. Docs-only slice; Go tests not required. | Commit `61cb7ea`; tag none |
| V5-016 | End-To-End Fake Provider Acceptance Tests | tests, provider/runtime, mission-control | Add checked-in fixtures and end-to-end tests that prove V5 runtime behavior without external services. | V5-002 through V5-014 | `cmd/picobot/*_test.go`, `internal/agent/*_test.go`, `internal/providers/*_test.go`, `internal/missioncontrol/*_test.go`, `testdata/frank_v5_configs/*` or package-local `testdata/*` | Fake provider tests cover legacy runtime, named cloud provider, named local provider, alias runtime selection, `providerModel` mapping, request overrides, no-tool local route, policy denial, fallback allowed, fallback denied, and no secret leak. Checked-in config fixtures cover legacy `providers.openai` only, V5 local llama.cpp-compatible provider, V5 Ollama-compatible provider, V5 OpenRouter-compatible provider, mixed local/cloud with fallback denied, and mixed local/cloud with fallback allowed. | End-to-end tests using `httptest` only. Fixture tests parse every checked-in config and run the relevant route/provider assertions. No real OpenAI, OpenRouter, Ollama, or llama.cpp dependencies. | Tests must not require API keys or live local model servers. Redaction assertions should include fake secret strings. Fixtures must use placeholder secrets only. | Pending | Not started | Not recorded |
| V5-017 | Migration Docs And Backward Compatibility Guide | docs, operator UX | Add final migration and operator guide for moving from legacy `agents.defaults.model` plus `providers.openai` to V5 profiles. | V5-015, V5-016 | `docs/CONFIG.md`, `START_HERE_OPERATOR.md`, `docs/FRANK_V5_MODEL_CONTROL_SPEC.md`, `docs/FRANK_V5_IMPLEMENTATION_MATRIX.md` | Docs explain legacy compatibility, V5 config examples, CLI model commands, mission policy examples, fallback behavior, health output, status output, and troubleshooting. | Docs review plus link checks where existing tooling supports it. | Avoid implying live fallback to cloud is automatic. State policy guardrails clearly, including the preflight-only fallback default. | Pending | Not started | Not recorded |
| V5-018 | Final V5 Acceptance Gate | tests, docs, operator UX | Run final V5 acceptance validation and mark the matrix complete only with evidence. | V5-002 through V5-017 | `docs/FRANK_V5_IMPLEMENTATION_MATRIX.md`, any failing-test fixes in touched areas | All V5 Definition of Done items are satisfied. Matrix statuses are updated. Targeted package tests and full repo tests pass. Remaining risks are documented. | `git diff --check`, targeted package tests, and `/usr/local/go/bin/go test -count=1 ./...`. Add any missing acceptance tests discovered during the gate. | Do not mark complete with missing evidence. Any failing test that implies a behavior question is a stop condition. | Pending | Not started | Not recorded |

## 4. Dependency Graph

Safe linear execution order:

V5-002 -> V5-003 -> V5-004 -> V5-005 -> V5-006 -> V5-007 -> V5-008 -> V5-009 -> V5-010 -> V5-011 -> V5-012 -> V5-013 -> V5-014 -> V5-015 -> V5-016 -> V5-017 -> V5-018

Parallel candidates after V5-002:

- V5-003 must happen before request-override work so the first post-routing
  safety boundary is tool-schema suppression.
- V5-004 can be implemented independently from later status and health work
  after V5-003 is complete.
- V5-007 can start after V5-002 without waiting for V5-005 if it remains
  read-only and CLI-scoped.
- V5-015 can draft docs after V5-007 establishes health command shape, but it
  should not be marked complete until V5-008 fallback behavior is final.
- V5-009 can start after V5-001, but enforcement in V5-010 should wait for
  runtime routing and fallback behavior.
- V5-015 docs can draft in parallel with V5-016 fixture tests, but final docs
  should wait for fixture names and command behavior to settle.

For autonomous execution, prefer the safe linear order unless a human explicitly
asks for parallel work.

## 5. Stop Conditions

Codex must stop and ask for human input when any condition below is hit:

- The spec and existing code contradict each other in a way that changes public
  behavior.
- A destructive migration, durable store rewrite, or non-additive schema change
  appears necessary.
- There is a credible risk of exposing API keys, authorization headers, prompts,
  tool arguments, raw provider bodies, or other secret-bearing values.
- The runtime authority boundary is unclear, especially around low-authority
  local models and dangerous tools.
- Existing tests fail and cannot be fixed without changing intended behavior.
- A slice requires external services, a live local model server, real phone
  access, or real API keys.
- A human choice is needed between incompatible designs, such as status schema
  placement with different operator compatibility costs.
- The worktree is dirty for unrelated reasons in files the slice must edit.
- Existing mission-control durable records would be invalidated by the proposed
  change.
- New dependencies, CGO, Python, Node, CUDA, model downloads, or embedded model
  runtimes become necessary.
- Implementation would require automatic replay or retry of an agent turn after
  a provider request began and idempotence cannot be proven.

## 6. Autonomous Execution Rules

- Work one slice at a time in slice ID order unless a human explicitly changes
  the order.
- Do not ask for approval between slices when the current slice acceptance
  criteria pass and no stop condition is present.
- Update this matrix status after each completed slice with validation evidence.
- During autonomous execution, Codex may update only `Status`, `Evidence`,
  `Commit/Tag`, slice-local execution notes, and risk `Risk Status` after each
  slice. Codex must not change slice objectives, acceptance criteria,
  dependencies, safety notes, or the V5 Definition of Done unless a stop
  condition is hit and human review approves the matrix change.
- Keep V5 behavior backward-compatible unless a reviewed matrix update says
  otherwise.
- Prefer tests first for behavior changes.
- Run targeted tests after each slice.
- Run full repo tests at major milestones and for V5-018.
- Recommended stable milestones for commit/tag review are after V5-003,
  V5-008, V5-012, and V5-018.
- Stop only on matrix completion or a stop condition.
- Never hide uncertainty. Record unresolved ambiguity in the slice notes or risk
  register.
- Never mark a slice complete without passing tests or an explicit written reason
  tests are not possible.
- Never require live OpenAI, OpenRouter, Ollama, llama.cpp, Android, or Termux
  resources in automated tests.
- Never weaken mission-control fail-closed behavior to make model routing pass.

## 7. Validation Commands

Baseline commands before each slice:

```sh
git branch --show-current
git rev-parse HEAD
git status --short --branch --untracked-files=all
git tag --points-at HEAD
```

Formatting and diff validation:

```sh
git diff --check
/usr/local/go/bin/gofmt -w <changed-go-files>
```

Likely targeted package tests, adjusted to the files touched by each slice:

```sh
/usr/local/go/bin/go test -count=1 ./internal/config
/usr/local/go/bin/go test -count=1 ./internal/providers
/usr/local/go/bin/go test -count=1 ./internal/agent
/usr/local/go/bin/go test -count=1 ./internal/agent/tools
/usr/local/go/bin/go test -count=1 ./internal/missioncontrol
/usr/local/go/bin/go test -count=1 ./cmd/picobot
```

Full repository validation:

```sh
/usr/local/go/bin/go test -count=1 ./...
```

Repo-local broader gates when time and environment allow:

```sh
make test
make test-lite
make vet
make lint
make verify
```

Some tests may use `httptest` loopback listeners. If sandboxed tests fail due
to local bind restrictions, rerun the same command with the required permission
rather than changing production code around the sandbox.

## 8. Risk Register

| Risk ID | Risk | Affected Slices | Severity | Mitigation | Risk Status |
| --- | --- | --- | --- | --- | --- |
| R-001 | Provider factory assumptions keep runtime pinned to `providers.openai`. | V5-002, V5-004, V5-016 | High | Add named provider construction tests and keep legacy provider tests. Fail closed on unknown provider refs. | Open |
| R-002 | API keys or authorization headers leak through CLI, status, logs, health errors, or tests. | V5-002, V5-006, V5-007, V5-014, V5-016 | Critical | Use redacted config views, error classes, fake secrets in tests, and no raw request/response body logging. | Partially mitigated by V5-007 health error classes and V5-014 cached `model_health` status using status/error-class fields only; E2E redaction coverage remains open. |
| R-003 | Tool schemas leak to weak local models. | V5-003, V5-010, V5-012, V5-016 | Critical | Gate tool definitions immediately before provider calls. Add fake provider tests that assert zero schemas. | Mitigated for built-in low/medium authority paths by V5-012; E2E fixture coverage remains open. |
| R-004 | Local/cloud fallback violates policy. | V5-008, V5-010, V5-016 | Critical | Enforce resolver fallback rules in runtime and test local-to-cloud denial by default. | Partially mitigated by V5-008 preflight runtime tests; mission-policy and E2E coverage remain open. |
| R-005 | Legacy config or raw `-M` override breaks. | V5-002, V5-004, V5-016, V5-017 | High | Keep implicit legacy profile and raw model compatibility tests in every runtime slice touching startup. | Open |
| R-006 | CLI model override ambiguity between alias, model ref, and raw provider model changes behavior. | V5-002, V5-010, V5-016 | Medium | Preserve V5-001 rule: model ref wins, alias resolves deterministically, unknown value becomes raw legacy provider model only on legacy-compatible path. Document this clearly. | Partially mitigated by V5-010 policy denial tests for explicit CLI alias outside allowed policy; E2E docs/tests remain open. |
| R-007 | Mission step policy conflicts with global routing or job policy. | V5-009, V5-010, V5-012 | High | Implement narrowing semantics with tests. Fail closed on contradictions instead of widening policy. | Mitigated for V5 minimum by V5-009 schema validation, V5-010 routing, and V5-012 step authority checks; E2E coverage remains open. |
| R-008 | Tests accidentally require live llama.cpp, Ollama, OpenAI, OpenRouter, phone, or API keys. | V5-007, V5-014, V5-016, V5-018 | High | Use `httptest`, fake providers, and placeholder credentials only. Mark live probes as manual docs, not automated gates. | Open |
| R-009 | Android runtime docs drift from actual V5 config and CLI behavior. | V5-015, V5-017 | Medium | Write docs after health and fallback command shapes are implemented. Include config snippets that are covered by config parser tests where practical. | Partially mitigated by V5-015 docs for named phone providers, health commands, and preflight-only fallback; final migration guide remains open. |
| R-010 | Temperature and request override support require provider interface changes that introduce shared mutable state. | V5-004 | Medium | Prefer per-call options or immutable provider clones. Add concurrency-safe tests if shared provider instances remain. | Open |
| R-011 | Status schema additions break operator or gateway consumers. | V5-006, V5-013, V5-014 | Medium | Make additive changes, update golden tests, and document the new safe `model` object. Stop if non-additive schema changes are needed. | Mitigated for V5 status additions by additive `model`, `model_metrics`, and `model_health` schema tests. |
| R-012 | Authority-tier enforcement lacks enough tool risk metadata for precise medium/high distinctions. | V5-011, V5-012 | High | Run taxonomy discovery before enforcement. Stop if metadata is insufficient instead of inventing broad semantics. | Mitigated for known built-in tools by V5-011/V5-012; dynamic MCP tools remain unclassified and fail closed for low/medium authority. |
| R-013 | Automatic runtime fallback could duplicate side effects or hide partial execution. | V5-008, V5-010, V5-016 | Critical | Use preflight-only fallback by default. Do not retry or replay after provider request start unless the request is proven read-only, idempotent, and side-effect-free. | Partially mitigated by V5-008 no-post-request-retry test; mission-policy and E2E coverage remain open. |

## 9. Open Questions

No blocking questions are currently known. The matrix uses conservative defaults:
additive status changes, no live external test dependencies, fail-closed policy
handling, and no cloud fallback from local unless explicit policy allows it.

## 10. Recommended Next Action

After human and reviewer approval of this matrix, start V5-002 only:

- Wire V5 resolver output into agent and gateway startup.
- Construct the selected named OpenAI-compatible provider at runtime.
- Ensure `providerModel` is the string sent to the backend.
- Preserve legacy `providers.openai` behavior and raw legacy `-M` overrides.
- Prove the slice with focused provider, config, and CLI startup tests before
  moving to V5-003.
