package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

func TestFrankV5ConfigFixturesBuildRegistries(t *testing.T) {
	cases := []struct {
		fixture           string
		wantModelRef      string
		wantProviderRef   string
		wantProviderModel string
	}{
		{fixture: "legacy_openai_only.json", wantModelRef: config.LegacyModelRef, wantProviderRef: "openai", wantProviderModel: "legacy-fixture-model"},
		{fixture: "v5_local_llamacpp.json", wantModelRef: "local_fast", wantProviderRef: "llamacpp_phone", wantProviderModel: "qwen3-1.7b-q8_0"},
		{fixture: "v5_ollama.json", wantModelRef: "ollama_chat", wantProviderRef: "ollama_phone", wantProviderModel: "qwen3:1.7b"},
		{fixture: "v5_openrouter.json", wantModelRef: "cloud_reasoning", wantProviderRef: "openrouter", wantProviderModel: "google/gemini-2.5-flash"},
		{fixture: "mixed_fallback_denied.json", wantModelRef: "local_fast", wantProviderRef: "llamacpp_phone", wantProviderModel: "qwen3-1.7b-q8_0"},
		{fixture: "mixed_fallback_allowed.json", wantModelRef: "local_fast", wantProviderRef: "llamacpp_phone", wantProviderModel: "qwen3-1.7b-q8_0"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			cfg := loadFrankV5Fixture(t, tc.fixture)
			reg, err := config.BuildModelRegistry(cfg)
			if err != nil {
				t.Fatalf("BuildModelRegistry(%s) error = %v", tc.fixture, err)
			}
			route, err := reg.Route(config.ModelRouteOptions{})
			if err != nil {
				t.Fatalf("Route(%s) error = %v", tc.fixture, err)
			}
			if route.SelectedModelRef != tc.wantModelRef || route.ProviderRef != tc.wantProviderRef || route.ProviderModel != tc.wantProviderModel {
				t.Fatalf("route = %#v, want model/provider/providerModel %s/%s/%s", route, tc.wantModelRef, tc.wantProviderRef, tc.wantProviderModel)
			}
		})
	}
}

func TestFrankV5OpenRouterFixtureRunsAgainstFakeProviderWithOverrides(t *testing.T) {
	const secret = "fixture-openrouter-secret"
	var captured struct {
		Model       string   `json:"model"`
		MaxTokens   int      `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("request path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
			t.Fatalf("Authorization header = %q, want bearer fixture secret", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fixture cloud ok"}}]}`))
	}))
	defer server.Close()

	cfg := loadFrankV5Fixture(t, "v5_openrouter.json")
	cfg.Agents.Defaults.Workspace = t.TempDir()
	setNamedProviderForFixture(t, &cfg, "openrouter", server.URL+"/v1", secret)
	writeFrankV5FixtureConfig(t, cfg)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"agent", "-m", "hello", "-M", "best"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent command error = %v", err)
	}
	if captured.Model != "google/gemini-2.5-flash" {
		t.Fatalf("captured model = %q, want providerModel", captured.Model)
	}
	if captured.MaxTokens != 777 {
		t.Fatalf("captured max_tokens = %d, want fixture override 777", captured.MaxTokens)
	}
	if captured.Temperature == nil || *captured.Temperature != 0.42 {
		t.Fatalf("captured temperature = %v, want fixture override 0.42", captured.Temperature)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(stdout.String(), "fixture cloud ok") {
		t.Fatalf("stdout = %q, want fake provider response", stdout.String())
	}
	if strings.Contains(combined, secret) {
		t.Fatalf("command output leaked API key: %q", combined)
	}
}

func TestFrankV5LocalFixtureSuppressesToolsAgainstFakeProvider(t *testing.T) {
	var captured struct {
		Model string `json:"model"`
		Tools any    `json:"tools"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`ok`))
			return
		case "/v1/chat/completions":
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fixture local ok"}}]}`))
			return
		default:
			t.Fatalf("unexpected request path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := loadFrankV5Fixture(t, "v5_local_llamacpp.json")
	cfg.Agents.Defaults.Workspace = t.TempDir()
	setNamedProviderForFixture(t, &cfg, "llamacpp_phone", server.URL+"/v1", "not-needed")
	setLocalRuntimeHealthURLForFixture(t, &cfg, "llamacpp_phone", server.URL+"/health")
	writeFrankV5FixtureConfig(t, cfg)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"agent", "-m", "hello", "-M", "phone"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent command error = %v", err)
	}
	if captured.Model != "qwen3-1.7b-q8_0" {
		t.Fatalf("captured model = %q, want local providerModel", captured.Model)
	}
	if captured.Tools != nil {
		t.Fatalf("captured tools = %#v, want no tools for local no-tool model", captured.Tools)
	}
	if !strings.Contains(stdout.String(), "fixture local ok") {
		t.Fatalf("stdout = %q, want fake provider response", stdout.String())
	}
}

func TestFrankV5MixedFallbackFixturesArePolicyBound(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer healthServer.Close()

	denied := loadFrankV5Fixture(t, "mixed_fallback_denied.json")
	setLocalRuntimeHealthURLForFixture(t, &denied, "llamacpp_phone", healthServer.URL)
	denied.Providers.Named["openrouter"] = config.ProviderConfig{Type: config.ProviderTypeOpenAICompatible, APIKey: "fallback-secret", APIBase: "https://cloud.invalid/v1"}
	_, err := resolveRuntimeModelSelectionWithContext(context.Background(), denied, "")
	if err == nil {
		t.Fatal("resolveRuntimeModelSelectionWithContext(denied) error = nil, want cloud-fallback denial")
	}
	if !strings.Contains(err.Error(), "cloud fallback from local") {
		t.Fatalf("denied error = %q, want cloud fallback denial", err.Error())
	}
	if strings.Contains(err.Error(), "fallback-secret") {
		t.Fatalf("fallback denial leaked API key: %q", err.Error())
	}

	allowed := loadFrankV5Fixture(t, "mixed_fallback_allowed.json")
	setLocalRuntimeHealthURLForFixture(t, &allowed, "llamacpp_phone", healthServer.URL)
	selection, err := resolveRuntimeModelSelectionWithContext(context.Background(), allowed, "")
	if err != nil {
		t.Fatalf("resolveRuntimeModelSelectionWithContext(allowed) error = %v", err)
	}
	if selection.Route.SelectedModelRef != "cloud_reasoning" || selection.Route.FallbackDepth != 1 {
		t.Fatalf("allowed route = %#v, want cloud fallback depth 1", selection.Route)
	}
}

func TestFrankV5FixtureMissionPolicyDenialIsSecretSafe(t *testing.T) {
	const secret = "policy-fixture-secret"
	cfg := loadFrankV5Fixture(t, "mixed_fallback_allowed.json")
	setLocalRuntimeHealthURLForFixture(t, &cfg, "llamacpp_phone", "")
	setNamedProviderForFixture(t, &cfg, "openrouter", "https://cloud.invalid/v1", secret)
	job := testMissionBootstrapJob()
	job.Plan.Steps[0].ModelPolicy = &missioncontrol.ModelPolicy{
		AllowedModels: []string{"local_fast"},
	}

	_, _, err := resolveRuntimeMissionModelSelectionWithContext(context.Background(), cfg, "best", job, "build")
	if err == nil {
		t.Fatal("resolveRuntimeMissionModelSelectionWithContext() error = nil, want policy denial")
	}
	var validationErr missioncontrol.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T %[1]v, want ValidationError", err)
	}
	if validationErr.Code != missioncontrol.RejectionCodeInvalidModelPolicy {
		t.Fatalf("ValidationError.Code = %q, want invalid_model_policy", validationErr.Code)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("policy denial leaked API key: %q", err.Error())
	}
}

func loadFrankV5Fixture(t *testing.T, name string) config.Config {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "frank_v5_configs", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", name, err)
	}
	return cfg
}

func writeFrankV5FixtureConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
}

func setNamedProviderForFixture(t *testing.T, cfg *config.Config, providerRef string, apiBase string, apiKey string) {
	t.Helper()
	if cfg.Providers.Named == nil {
		cfg.Providers.Named = make(map[string]config.ProviderConfig)
	}
	provider, ok := cfg.Providers.Named[providerRef]
	if !ok {
		t.Fatalf("fixture missing provider %q", providerRef)
	}
	provider.Type = config.ProviderTypeOpenAICompatible
	provider.APIBase = apiBase
	provider.APIKey = apiKey
	cfg.Providers.Named[providerRef] = provider
}

func setLocalRuntimeHealthURLForFixture(t *testing.T, cfg *config.Config, runtimeRef string, healthURL string) {
	t.Helper()
	runtime, ok := cfg.LocalRuntimes[runtimeRef]
	if !ok {
		t.Fatalf("fixture missing local runtime %q", runtimeRef)
	}
	runtime.HealthURL = healthURL
	cfg.LocalRuntimes[runtimeRef] = runtime
}
