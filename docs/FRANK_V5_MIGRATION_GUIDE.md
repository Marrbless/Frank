# Frank V5 Migration Guide

Frank V5 adds the Model Control Plane while preserving existing legacy configs.
Use this guide to move from one `agents.defaults.model` string plus
`providers.openai` to named providers, named model profiles, aliases, health
checks, and mission model policy.

## Compatibility Contract

Legacy config remains valid:

```json
{
  "agents": {
    "defaults": {
      "model": "legacy-provider-model"
    }
  },
  "providers": {
    "openai": {
      "apiKey": "REPLACE_WITH_REAL_PROVIDER_API_KEY",
      "apiBase": "https://api.openai.com/v1"
    }
  }
}
```

If `models` is absent, Frank creates an implicit `legacy_default` model profile
from `agents.defaults.model` and `providers.openai`. A raw CLI override still
works:

```sh
picobot agent -M some-provider-model -m "hello"
picobot gateway -M some-provider-model
```

In a V5 config, `-M` first resolves a `model_ref`, then an alias. If it does
not resolve and the active config is legacy-compatible, it is treated as a raw
provider model on `providers.openai`.

## Migration Steps

1. Back up the current config:

```sh
cp ~/.picobot/config.json ~/.picobot/config.json.before-v5
```

2. Keep `providers.openai` if you still need legacy raw `-M` behavior.

3. Add named providers for every endpoint:

```json
{
  "providers": {
    "openai": {
      "type": "openai_compatible",
      "apiKey": "REPLACE_WITH_REAL_OPENAI_KEY",
      "apiBase": "https://api.openai.com/v1"
    },
    "openrouter": {
      "type": "openai_compatible",
      "apiKey": "REPLACE_WITH_REAL_OPENROUTER_KEY",
      "apiBase": "https://openrouter.ai/api/v1"
    },
    "llamacpp_phone": {
      "type": "openai_compatible",
      "apiKey": "not-needed",
      "apiBase": "http://127.0.0.1:8080/v1"
    },
    "ollama_phone": {
      "type": "openai_compatible",
      "apiKey": "ollama",
      "apiBase": "http://127.0.0.1:11434/v1"
    }
  }
}
```

4. Add model profiles. `providerModel` is the exact model string sent to the
backend:

```json
{
  "models": {
    "local_fast": {
      "provider": "llamacpp_phone",
      "providerModel": "qwen3-1.7b-q8_0",
      "capabilities": {
        "local": true,
        "offline": true,
        "supportsTools": false,
        "contextTokens": 4096,
        "maxOutputTokens": 1024,
        "authorityTier": "low",
        "costTier": "free",
        "latencyTier": "slow"
      },
      "request": {
        "maxTokens": 1024,
        "temperature": 0.3,
        "timeoutS": 300,
        "useResponses": false
      }
    },
    "cloud_reasoning": {
      "provider": "openrouter",
      "providerModel": "google/gemini-2.5-flash",
      "capabilities": {
        "local": false,
        "offline": false,
        "supportsTools": true,
        "contextTokens": 1000000,
        "maxOutputTokens": 8192,
        "authorityTier": "high",
        "costTier": "standard",
        "latencyTier": "normal"
      },
      "request": {
        "maxTokens": 8192,
        "temperature": 0.5,
        "timeoutS": 120,
        "useResponses": false
      }
    }
  }
}
```

5. Add aliases and routing:

```json
{
  "agents": {
    "defaults": {
      "model": "cloud_reasoning"
    }
  },
  "modelAliases": {
    "phone": "local_fast",
    "local": "local_fast",
    "best": "cloud_reasoning",
    "default": "cloud_reasoning"
  },
  "modelRouting": {
    "defaultModel": "cloud_reasoning",
    "localPreferredModel": "local_fast",
    "fallbacks": {
      "local_fast": ["cloud_reasoning"],
      "cloud_reasoning": []
    },
    "allowCloudFallbackFromLocal": false,
    "allowLowerAuthorityFallback": false
  }
}
```

6. Add local runtime profiles if a local HTTP server should participate in
preflight readiness:

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
    }
  }
}
```

7. Validate without making provider requests:

```sh
picobot config validate
picobot models list
picobot models inspect phone
picobot models route --model phone --local
picobot models route --model best --requires-tools
```

8. Check readiness when the endpoints are running:

```sh
picobot models health
picobot models health phone
```

## Mission Model Policy

Mission jobs and steps can narrow global routing. Step policy narrows job policy.
Missing policy preserves existing behavior.

Example step policy for a local-only non-tool step:

```json
{
  "id": "classify",
  "type": "discussion",
  "allowed_tools": [],
  "model_policy": {
    "allowed_models": ["local_fast"],
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

Example high-authority tool-capable step:

```json
{
  "id": "execute",
  "type": "tool",
  "allowed_tools": ["filesystem"],
  "required_authority": "high",
  "model_policy": {
    "allowed_models": ["cloud_reasoning"],
    "default_model": "cloud_reasoning",
    "required_capabilities": {
      "supportsTools": true,
      "authorityTierAtLeast": "high"
    },
    "allow_fallback": false,
    "allow_cloud": true
  }
}
```

Policy denial is fail-closed and reported with a safe error class. It does not
print API keys, provider request bodies, prompts, messages, or tool arguments.

## Fallback Rules

- Fallback is selected only during preflight route selection.
- Frank does not automatically retry or replay an agent turn after a provider
  request starts.
- Local-to-cloud fallback is denied unless `allowCloudFallbackFromLocal` or
  mission policy explicitly allows it.
- Fallback to a lower-authority model is denied by default.
- Fallback cannot bypass `allowed_models` in mission policy.

Use the route command to inspect routing before starting a gateway:

```sh
picobot models route --model phone --allow-fallback
```

## Status Output

Mission status can include safe model-control fields:

```json
{
  "model": {
    "selected_model_ref": "local_fast",
    "provider_ref": "llamacpp_phone",
    "provider_model": "qwen3-1.7b-q8_0",
    "selection_reason": "routing_default",
    "fallback_depth": 0,
    "policy_id": "default",
    "capabilities": {
      "local": true,
      "offline": true,
      "supports_tools": false,
      "authority_tier": "low"
    }
  },
  "model_health": [
    {
      "model_ref": "local_fast",
      "provider_ref": "llamacpp_phone",
      "status": "unknown",
      "last_checked_at": "2026-05-01T00:00:00Z",
      "fallback_available": true
    }
  ],
  "model_metrics": {
    "route_attempt_count": 1,
    "route_success_count": 1,
    "tool_schema_suppressed_count": 1
  }
}
```

These fields are intentionally secret-safe. They must not include API keys,
authorization headers, prompts, message content, tool arguments, request bodies,
response bodies, or raw provider error bodies.

## Troubleshooting

`unknown provider_ref`

- Check that every model profile has a matching `providers.<provider_ref>`.
- Provider refs are normalized to lowercase and may contain only `a-z`, `0-9`,
  `_`, and `-`.

`unknown alias/model_ref`

- Check `modelAliases`.
- Alias chains are rejected. Point aliases directly at model refs.

`fallback is disabled`

- Set `modelRouting.fallbacks.<model_ref>` only when fallback is intended.
- For local-to-cloud fallback, also set `allowCloudFallbackFromLocal: true`.
- Mission `model_policy.allow_fallback: false` still blocks fallback.

`cloud fallback from local model_ref is not allowed`

- This is the default safety behavior.
- Keep it for private/offline steps.
- Enable cloud fallback only for steps where cloud execution is acceptable.

`selected model authority tier is below step required authority`

- Use a higher-authority model profile for that step.
- Do not raise a tiny local model above `low` without reviewed policy and tool
  risk evidence.

`supportsTools=false` but a step requires tools

- Select a model with `supportsTools: true`, or make the step no-tool.
- Tool schemas are withheld from models that cannot receive tools.

Health shows `disabled`

- The provider has no health URL and no usable `/models` endpoint configured.
- For local runtimes, set or clear `localRuntimes.<runtime>.healthURL`
  intentionally.

Health shows `unhealthy`

- Start the local runtime process.
- Check that `apiBase` and `healthURL` point at the same running service.
- Check Tailscale/VPN/firewall settings for LAN endpoints.

## Secret Hygiene

- Keep real keys only in local config or a local secret manager.
- Use placeholders in docs and support snippets.
- Do not paste `~/.picobot/config.json`, gateway logs, provider request bodies,
  provider response bodies, prompts, message content, or tool arguments into
  shared issue threads.
