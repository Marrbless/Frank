# GC5-TREAT-007 Tool Schema Typing Boundary Decision

Date: 2026-04-28

## Scope

Decide whether repo-wide `map[string]interface{}` tool schema and argument usage should be mechanically typed as part of this garbage campaign.

## Evidence

The generic JSON boundary is shared by several public and integration-facing surfaces:

- `internal/agent/tools/registry.go`
  - `Tool.Parameters() map[string]interface{}`
  - `Tool.Execute(ctx context.Context, args map[string]interface{})`
  - registry execution, audit, and guard evaluation pass the same argument map.
- `internal/providers/provider.go`
  - `ToolDefinition.Parameters map[string]interface{}`
  - `ToolCall.Arguments map[string]interface{}`
- `internal/providers/openai.go`
  - OpenAI Chat Completions and Responses serialization/deserialization use JSON-like maps for provider payloads and tool call arguments.
- `internal/mcp/client.go`
  - `Tool.InputSchema map[string]interface{}`
  - `CallTool(..., arguments map[string]interface{})`
- `internal/agent/tools/mcp.go`
  - MCP tools expose server-provided input schemas directly through `Parameters()`.

Individual tools already parse and validate their argument values at the execution boundary.

## Decision

Defer broad tool-schema typing.

`map[string]interface{}` is currently the compatibility boundary between local tools, model provider payloads, and MCP-discovered schemas. Replacing it repo-wide would be an API decision, not a garbage cleanup. A mechanical change would touch high-churn surfaces and tests without proving safer behavior.

## Approved Boundary For Future Work

Future cleanup may add small local aliases or helpers if they reduce noise without changing the public contract, for example:

- an internal `JSONSchema` alias for `map[string]interface{}` in provider/tool definitions,
- per-tool typed argument parsers that convert from `map[string]interface{}` into tool-local structs,
- narrow validation helpers for repeated primitive argument parsing.

Any future change should keep provider and MCP interoperability explicit, preserve existing serialized tool schemas, and include registry/provider/MCP tests.

## Non-Goals

- No repo-wide replacement of `interface{}` with `any`.
- No provider payload reshaping.
- No MCP schema normalization.
- No generated JSON Schema system.
- No behavior changes to tool execution, guard evaluation, audit summaries, or tool-call replay.

## Validation

- `rg -n "type Tool|Parameters\\(\\)|Execute\\(ctx context\\.Context|map\\[string\\]interface\\{\\}|interface\\{\\}|jsonschema|Schema" internal/agent/tools internal/providers internal/mcp cmd -g '*.go'`
  - Result: confirmed the generic JSON boundary spans tools, providers, MCP, and tests.
- `sed -n '1,220p' internal/agent/tools/registry.go`
  - Result: confirmed the registry contract.
- `sed -n '1,220p' internal/providers/provider.go`
  - Result: confirmed provider-facing tool definitions and calls.
- `sed -n '1,120p' internal/mcp/client.go`
  - Result: confirmed MCP-discovered input schemas use the same generic map form.

No code changes were made for this row.
