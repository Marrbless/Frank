package providers

import (
	"fmt"

	"github.com/local/picobot/internal/config"
)

// NewProviderFromConfig creates a provider based on the configuration.
//
// Simple rules (v0):
// - if OpenAI API key present or API base is set -> OpenAI
// - else fallback to stub
func NewProviderFromConfig(cfg config.Config) LLMProvider {
	if cfg.Providers.OpenAI != nil && (cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.APIBase != "") {
		return NewOpenAIProviderWithOptions(
			cfg.Providers.OpenAI.APIKey,
			cfg.Providers.OpenAI.APIBase,
			cfg.Agents.Defaults.RequestTimeoutS,
			cfg.Agents.Defaults.MaxTokens,
			cfg.Providers.OpenAI.UseResponses,
			cfg.Providers.OpenAI.ReasoningEffort,
		)
	}
	return NewStubProvider()
}

// NewProviderFromModelRoute creates the provider selected by the V5 model
// registry route. It never includes secret-bearing config values in errors.
func NewProviderFromModelRoute(reg config.ModelRegistry, route config.ModelRoute) (LLMProvider, error) {
	providerCfg, ok := reg.Providers[route.ProviderRef]
	if !ok {
		return nil, fmt.Errorf("provider_ref %q is not configured", route.ProviderRef)
	}
	if providerCfg.Type != "" && providerCfg.Type != config.ProviderTypeOpenAICompatible {
		return nil, fmt.Errorf("provider_ref %q has unsupported type %q", route.ProviderRef, providerCfg.Type)
	}
	if providerCfg.APIKey == "" && providerCfg.APIBase == "" {
		if route.ProviderRef == config.LegacyProviderRef {
			return NewStubProvider(), nil
		}
		return nil, fmt.Errorf("provider_ref %q requires apiKey or apiBase", route.ProviderRef)
	}

	return NewOpenAIProviderWithRequestOptions(
		providerCfg.APIKey,
		providerCfg.APIBase,
		route.Request.TimeoutS,
		route.Request.MaxTokens,
		route.Request.Temperature,
		route.Request.UseResponses,
		route.Request.ReasoningEffort,
	), nil
}
