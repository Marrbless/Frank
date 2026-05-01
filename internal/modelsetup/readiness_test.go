package modelsetup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/config"
)

func TestNoPromptReadinessUsesMetadataEndpointWithoutAuthorizationOrBody(t *testing.T) {
	var sawAuth bool
	var sawBody bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		if r.Body != nil && r.ContentLength > 0 {
			sawBody = true
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	cfg := readinessConfig(server.URL, true)
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	result, err := CheckNoPromptReadiness(context.Background(), reg, "local_fast", ReadinessOptions{
		Now:           time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		LocalRuntimes: cfg.LocalRuntimes,
	})
	if err != nil {
		t.Fatalf("CheckNoPromptReadiness() error = %v", err)
	}
	if result.Status != ReadinessHealthy {
		t.Fatalf("Status = %q, want healthy result %#v", result.Status, result)
	}
	if sawAuth {
		t.Fatal("readiness request sent Authorization header")
	}
	if sawBody {
		t.Fatal("readiness request sent request body")
	}
}

func TestNoPromptReadinessUsesLocalModelsMetadataSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatalf("readiness request sent Authorization header")
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen"}]}`))
	}))
	defer server.Close()
	cfg := readinessConfig(server.URL+"/v1", false)
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	result, err := CheckNoPromptReadiness(context.Background(), reg, "phone", ReadinessOptions{LocalRuntimes: cfg.LocalRuntimes})
	if err != nil {
		t.Fatalf("CheckNoPromptReadiness() error = %v", err)
	}
	if result.Status != ReadinessHealthy || result.RouteModel != "qwen3-test" {
		t.Fatalf("result = %#v, want healthy route model", result)
	}
}

func TestNoPromptReadinessSkipsCloudProbe(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter": {Type: config.ProviderTypeOpenAICompatible, APIKey: "sk-secret", APIBase: "https://openrouter.example/v1"},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "cloud-model",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
			},
		},
	}
	cfg.ModelAliases = map[string]string{"best": "cloud_reasoning"}
	cfg.ModelRouting.DefaultModel = "cloud_reasoning"
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	result, err := CheckNoPromptReadiness(context.Background(), reg, "best", ReadinessOptions{})
	if err != nil {
		t.Fatalf("CheckNoPromptReadiness() error = %v", err)
	}
	if result.Status != ReadinessManualRequired || result.ErrorClass != "cloud_probe_skipped" {
		t.Fatalf("result = %#v, want manual_required cloud skip", result)
	}
	out := FormatReadinessResult(result)
	if strings.Contains(out, "sk-secret") {
		t.Fatalf("readiness output leaked secret: %q", out)
	}
}

func readinessConfig(baseURL string, withRuntimeHealth bool) config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "local_fast"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"ollama_phone": {Type: config.ProviderTypeOpenAICompatible, APIKey: "ollama-secret", APIBase: baseURL},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"local_fast": {
			Provider:      "ollama_phone",
			ProviderModel: "qwen3-test",
			Capabilities: config.ModelCapabilities{
				Local:         true,
				Offline:       true,
				AuthorityTier: config.ModelAuthorityLow,
				CostTier:      config.ModelCostFree,
				LatencyTier:   config.ModelLatencySlow,
			},
		},
	}
	cfg.ModelAliases = map[string]string{"phone": "local_fast"}
	cfg.ModelRouting.DefaultModel = "local_fast"
	cfg.ModelRouting.LocalPreferredModel = "local_fast"
	cfg.ModelRouting.Fallbacks = map[string][]string{"local_fast": {}}
	if withRuntimeHealth {
		cfg.LocalRuntimes = map[string]config.LocalRuntimeConfig{
			"ollama_phone": {Provider: "ollama_phone", HealthURL: baseURL},
		}
	}
	return cfg
}
