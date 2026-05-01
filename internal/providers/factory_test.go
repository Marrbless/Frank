package providers

import (
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestNewProviderFromConfig_PicksOpenAI(t *testing.T) {
	cfg := config.Config{}
	cfg.Providers.OpenAI = &config.ProviderConfig{APIKey: "test"}
	p := NewProviderFromConfig(cfg)
	_, ok := p.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected OpenAIProvider, got %T", p)
	}
}

func TestNewProviderFromConfig_FallbacksToStub(t *testing.T) {
	cfg := config.Config{}
	p := NewProviderFromConfig(cfg)
	_, ok := p.(*StubProvider)
	if !ok {
		t.Fatalf("expected StubProvider, got %T", p)
	}
}

func TestNewProviderFromModelRouteUsesNamedProvider(t *testing.T) {
	temp := 0.2
	useResponses := false
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.MaxTokens = 1234
	cfg.Agents.Defaults.RequestTimeoutS = 45
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter": {
			Type:            config.ProviderTypeOpenAICompatible,
			APIKey:          "named-secret",
			APIBase:         "https://openrouter.example/v1",
			UseResponses:    true,
			ReasoningEffort: "low",
		},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "google/gemini-test",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
				CostTier:      config.ModelCostStandard,
				LatencyTier:   config.ModelLatencyNormal,
			},
			Request: config.ModelRequestConfig{
				MaxTokens:       4321,
				Temperature:     &temp,
				TimeoutS:        7,
				UseResponses:    &useResponses,
				ReasoningEffort: "medium",
			},
		},
	}
	cfg.ModelRouting = config.ModelRoutingConfig{DefaultModel: "cloud_reasoning"}

	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	provider, err := NewProviderFromModelRoute(reg, route)
	if err != nil {
		t.Fatalf("NewProviderFromModelRoute() error = %v", err)
	}
	openAI, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *OpenAIProvider", provider)
	}
	if openAI.APIBase != "https://openrouter.example/v1" {
		t.Fatalf("APIBase = %q, want named provider base", openAI.APIBase)
	}
	if openAI.APIKey != "named-secret" {
		t.Fatalf("APIKey was not copied from named provider")
	}
	if openAI.MaxTokens != 4321 {
		t.Fatalf("MaxTokens = %d, want route request override", openAI.MaxTokens)
	}
	if openAI.Temperature == nil || *openAI.Temperature != 0.2 {
		t.Fatalf("Temperature = %v, want 0.2", openAI.Temperature)
	}
	if openAI.Client.Timeout.String() != "7s" {
		t.Fatalf("client timeout = %s, want 7s", openAI.Client.Timeout)
	}
	if openAI.UseResponses {
		t.Fatalf("UseResponses = true, want model request override false")
	}
	if openAI.ReasoningEffort != "medium" {
		t.Fatalf("ReasoningEffort = %q, want medium", openAI.ReasoningEffort)
	}
}

func TestNewProviderFromModelRouteLegacyEmptyProviderFallsBackToStub(t *testing.T) {
	cfg := config.Config{}
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	provider, err := NewProviderFromModelRoute(reg, route)
	if err != nil {
		t.Fatalf("NewProviderFromModelRoute() error = %v", err)
	}
	if _, ok := provider.(*StubProvider); !ok {
		t.Fatalf("provider type = %T, want *StubProvider", provider)
	}
}

func TestNewProviderFromModelRouteErrorDoesNotLeakAPIKey(t *testing.T) {
	const secret = "super-secret-api-key"
	reg := config.ModelRegistry{
		Providers: map[string]config.ProviderConfig{
			"bad": {
				Type:    "unsupported",
				APIKey:  secret,
				APIBase: "https://example.invalid/v1",
			},
		},
	}
	_, err := NewProviderFromModelRoute(reg, config.ModelRoute{ProviderRef: "bad"})
	if err == nil {
		t.Fatal("NewProviderFromModelRoute() error = nil, want unsupported provider error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked API key: %v", err)
	}
}
