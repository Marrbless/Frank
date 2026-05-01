package config

import (
	"strings"
	"testing"
)

func TestNormalizeModelRefTrimsAndLowercases(t *testing.T) {
	got, err := NormalizeModelRef("  Local_FAST  ")
	if err != nil {
		t.Fatalf("NormalizeModelRef() error = %v", err)
	}
	if got != "local_fast" {
		t.Fatalf("NormalizeModelRef() = %q, want local_fast", got)
	}
}

func TestNormalizeModelRefRejectsInvalidRef(t *testing.T) {
	for _, input := range []string{"", " ", "../model", "model/name", `model\name`, "model.name"} {
		t.Run(input, func(t *testing.T) {
			if got, err := NormalizeModelRef(input); err == nil {
				t.Fatalf("NormalizeModelRef(%q) = %q, nil error; want error", input, got)
			}
		})
	}
}

func TestBuildModelRegistryRejectsUnknownProviderRef(t *testing.T) {
	cfg := v5RegistryTestConfig()
	cfg.Models["local_fast"] = ModelProfileConfig{
		Provider:      "missing_provider",
		ProviderModel: "tiny",
	}

	_, err := BuildModelRegistry(cfg)
	if err == nil || !strings.Contains(err.Error(), `unknown provider_ref "missing_provider"`) {
		t.Fatalf("BuildModelRegistry() error = %v, want unknown provider_ref", err)
	}
}

func TestModelRegistryAliasResolvesToModelRef(t *testing.T) {
	reg := mustBuildModelRegistry(t, v5RegistryTestConfig())

	got, err := reg.ResolveModelRef("PHONE")
	if err != nil {
		t.Fatalf("ResolveModelRef() error = %v", err)
	}
	if got != "local_fast" {
		t.Fatalf("ResolveModelRef() = %q, want local_fast", got)
	}
}

func TestBuildModelRegistryRejectsUnknownAliasTarget(t *testing.T) {
	cfg := v5RegistryTestConfig()
	cfg.ModelAliases["cheap"] = "missing_model"

	_, err := BuildModelRegistry(cfg)
	if err == nil || !strings.Contains(err.Error(), `alias "cheap" targets unknown model_ref "missing_model"`) {
		t.Fatalf("BuildModelRegistry() error = %v, want unknown alias target", err)
	}
}

func TestBuildModelRegistryRejectsAliasChain(t *testing.T) {
	cfg := v5RegistryTestConfig()
	cfg.ModelAliases["cheap"] = "phone"

	_, err := BuildModelRegistry(cfg)
	if err == nil || !strings.Contains(err.Error(), `alias "cheap" targets alias "phone"`) {
		t.Fatalf("BuildModelRegistry() error = %v, want alias chain rejection", err)
	}
}

func TestBuildModelRegistryRejectsDuplicateNormalizedRefsDeterministically(t *testing.T) {
	cfg := v5RegistryTestConfig()
	cfg.Models[" Local_Fast "] = cfg.Models["local_fast"]

	_, err := BuildModelRegistry(cfg)
	if err == nil {
		t.Fatal("BuildModelRegistry() error = nil, want duplicate normalized ref error")
	}
	want := `duplicate model_ref "local_fast" from keys " Local_Fast ", "local_fast"`
	if err.Error() != want {
		t.Fatalf("BuildModelRegistry() error = %q, want %q", err.Error(), want)
	}
}

func TestLocalModelDefaultsToLowAuthorityAndNoTools(t *testing.T) {
	cfg := v5RegistryTestConfig()
	local := cfg.Models["local_fast"]
	local.Capabilities = ModelCapabilities{Local: true, Offline: true}
	cfg.Models["local_fast"] = local

	reg := mustBuildModelRegistry(t, cfg)
	model := reg.Models["local_fast"]
	if model.Capabilities.AuthorityTier != ModelAuthorityLow {
		t.Fatalf("AuthorityTier = %q, want %q", model.Capabilities.AuthorityTier, ModelAuthorityLow)
	}
	if model.Capabilities.SupportsTools {
		t.Fatal("SupportsTools = true, want default false for local model")
	}
}

func TestRouteSuppressesToolDefinitionsWhenModelDoesNotSupportTools(t *testing.T) {
	reg := mustBuildModelRegistry(t, v5RegistryTestConfig())

	route, err := reg.Route(ModelRouteOptions{ExplicitModel: "local_fast"})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if route.ToolDefinitionsAllowed {
		t.Fatal("ToolDefinitionsAllowed = true, want false")
	}
	if !route.ToolDefinitionsSuppressed {
		t.Fatal("ToolDefinitionsSuppressed = false, want true")
	}
}

func TestRouteFailsWhenFallbackDisabledAndSelectedModelUnavailable(t *testing.T) {
	reg := mustBuildModelRegistry(t, v5RegistryTestConfig())

	_, err := reg.Route(ModelRouteOptions{
		ExplicitModel:      "local_fast",
		UnavailableModels:  map[string]bool{"local_fast": true},
		AllowFallback:      false,
		RequiredCapability: ModelRequiredCapabilities{},
	})
	if err == nil || !strings.Contains(err.Error(), `model_ref "local_fast" is unavailable and fallback is disabled`) {
		t.Fatalf("Route() error = %v, want fallback disabled unavailable error", err)
	}
}

func TestRouteDeniesCloudFallbackFromLocalByDefault(t *testing.T) {
	reg := mustBuildModelRegistry(t, v5RegistryTestConfig())

	_, err := reg.Route(ModelRouteOptions{
		ExplicitModel:     "local_fast",
		UnavailableModels: map[string]bool{"local_fast": true},
		AllowFallback:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "cloud fallback from local model_ref \"local_fast\" is not allowed") {
		t.Fatalf("Route() error = %v, want cloud fallback denial", err)
	}
}

func TestRouteDeniesLowerAuthorityFallbackByDefault(t *testing.T) {
	cfg := v5RegistryTestConfig()
	cfg.Models["medium_local"] = ModelProfileConfig{
		Provider:      "llamacpp_phone",
		ProviderModel: "medium-local",
		Capabilities: ModelCapabilities{
			Local:         true,
			Offline:       true,
			SupportsTools: true,
			AuthorityTier: ModelAuthorityMedium,
		},
	}
	cfg.ModelRouting.Fallbacks["cloud_reasoning"] = []string{"medium_local"}
	cfg.ModelRouting.AllowCloudFallbackFromLocal = true

	reg := mustBuildModelRegistry(t, cfg)
	_, err := reg.Route(ModelRouteOptions{
		ExplicitModel:     "cloud_reasoning",
		UnavailableModels: map[string]bool{"cloud_reasoning": true},
		AllowFallback:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "lower-authority fallback from high to medium is not allowed") {
		t.Fatalf("Route() error = %v, want lower authority fallback denial", err)
	}
}

func TestRouteResolvesRequestOverrides(t *testing.T) {
	cfg := v5RegistryTestConfig()
	temp := 0.2
	useResponses := true
	model := cfg.Models["cloud_reasoning"]
	model.Request = ModelRequestConfig{
		MaxTokens:       1234,
		Temperature:     &temp,
		TimeoutS:        45,
		UseResponses:    &useResponses,
		ReasoningEffort: "medium",
	}
	cfg.Models["cloud_reasoning"] = model

	route, err := mustBuildModelRegistry(t, cfg).Route(ModelRouteOptions{ExplicitModel: "cloud_reasoning"})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if route.Request.MaxTokens != 1234 || route.Request.TimeoutS != 45 || route.Request.Temperature == nil || *route.Request.Temperature != 0.2 || !route.Request.UseResponses || route.Request.ReasoningEffort != "medium" {
		t.Fatalf("route.Request = %#v, want resolved overrides", route.Request)
	}
}

func TestLegacyConfigBuildsImplicitModelProfile(t *testing.T) {
	cfg := Config{
		Agents: AgentsConfig{Defaults: AgentDefaults{
			Model:           "google/gemini-2.5-flash",
			MaxTokens:       4096,
			Temperature:     0.4,
			RequestTimeoutS: 90,
		}},
		Providers: ProvidersConfig{
			OpenAI: &ProviderConfig{APIKey: "test-key", APIBase: "https://openrouter.ai/api/v1"},
		},
	}

	route, err := mustBuildModelRegistry(t, cfg).Route(ModelRouteOptions{})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if route.SelectedModelRef != LegacyModelRef || route.ProviderRef != LegacyProviderRef || route.ProviderModel != "google/gemini-2.5-flash" {
		t.Fatalf("route = %#v, want legacy provider/model route", route)
	}
	if route.Request.MaxTokens != 4096 || route.Request.TimeoutS != 90 || route.Request.Temperature == nil || *route.Request.Temperature != 0.4 {
		t.Fatalf("route.Request = %#v, want legacy defaults", route.Request)
	}
}

func TestLegacyRawExplicitModelMapsToLegacyProviderModel(t *testing.T) {
	cfg := Config{
		Agents: AgentsConfig{Defaults: AgentDefaults{Model: "config-default"}},
		Providers: ProvidersConfig{
			OpenAI: &ProviderConfig{APIKey: "test-key", APIBase: "https://api.openai.com/v1"},
		},
	}

	route, err := mustBuildModelRegistry(t, cfg).Route(ModelRouteOptions{
		ExplicitModel:          "unregistered-provider-model",
		AllowRawProviderModel:  true,
		RawProviderModelSource: RouteReasonCLIOverride,
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if route.SelectedModelRef != LegacyModelRef || route.ProviderRef != LegacyProviderRef || route.ProviderModel != "unregistered-provider-model" {
		t.Fatalf("route = %#v, want raw provider model on legacy provider", route)
	}
}

func mustBuildModelRegistry(t *testing.T, cfg Config) ModelRegistry {
	t.Helper()

	reg, err := BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	return reg
}

func v5RegistryTestConfig() Config {
	tempCloud := 0.5
	tempLocal := 0.3
	return Config{
		Agents: AgentsConfig{Defaults: AgentDefaults{
			Model:           "cloud_reasoning",
			MaxTokens:       8192,
			Temperature:     0.7,
			RequestTimeoutS: 60,
		}},
		Providers: ProvidersConfig{
			OpenAI: &ProviderConfig{Type: ProviderTypeOpenAICompatible, APIKey: "test-key", APIBase: "https://api.openai.com/v1"},
			Named: map[string]ProviderConfig{
				"openrouter":     {Type: ProviderTypeOpenAICompatible, APIKey: "router-key", APIBase: "https://openrouter.ai/api/v1"},
				"llamacpp_phone": {Type: ProviderTypeOpenAICompatible, APIKey: "not-needed", APIBase: "http://127.0.0.1:8080/v1"},
			},
		},
		Models: map[string]ModelProfileConfig{
			"local_fast": {
				Provider:      "llamacpp_phone",
				ProviderModel: "qwen3-1.7b-q8_0",
				DisplayName:   "Qwen3 phone local",
				Capabilities: ModelCapabilities{
					Local:           true,
					Offline:         true,
					ContextTokens:   4096,
					MaxOutputTokens: 1024,
					AuthorityTier:   ModelAuthorityLow,
					CostTier:        ModelCostFree,
					LatencyTier:     ModelLatencySlow,
				},
				Request: ModelRequestConfig{MaxTokens: 1024, Temperature: &tempLocal, TimeoutS: 300},
			},
			"cloud_reasoning": {
				Provider:      "openrouter",
				ProviderModel: "google/gemini-2.5-flash",
				DisplayName:   "Cloud reasoning",
				Capabilities: ModelCapabilities{
					SupportsTools:   true,
					ContextTokens:   1000000,
					MaxOutputTokens: 8192,
					AuthorityTier:   ModelAuthorityHigh,
					CostTier:        ModelCostStandard,
					LatencyTier:     ModelLatencyNormal,
				},
				Request: ModelRequestConfig{MaxTokens: 8192, Temperature: &tempCloud, TimeoutS: 120},
			},
		},
		ModelAliases: map[string]string{
			"default": "cloud_reasoning",
			"phone":   "local_fast",
		},
		ModelRouting: ModelRoutingConfig{
			DefaultModel:                "cloud_reasoning",
			LocalPreferredModel:         "local_fast",
			Fallbacks:                   map[string][]string{"local_fast": {"cloud_reasoning"}, "cloud_reasoning": {}},
			AllowCloudFallbackFromLocal: false,
			AllowLowerAuthorityFallback: false,
		},
	}
}
