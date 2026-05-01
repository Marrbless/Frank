package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestModelsListCommandPrintsConfiguredModelsWithoutSecrets(t *testing.T) {
	writeModelsCommandConfig(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "list"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models list error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"local_fast", "cloud_reasoning", "llamacpp_phone", "openrouter"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models list output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "router-secret") {
		t.Fatalf("models list output leaked API key: %q", out)
	}
}

func TestModelsInspectCommandResolvesAliasWithoutSecrets(t *testing.T) {
	writeModelsCommandConfig(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "inspect", "phone"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models inspect error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"model_ref: local_fast", "provider_ref: llamacpp_phone", "supports_tools: false", "authority_tier: low"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models inspect output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "router-secret") {
		t.Fatalf("models inspect output leaked API key: %q", out)
	}
}

func TestModelsRouteCommandResolvesToolCapableModel(t *testing.T) {
	writeModelsCommandConfig(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "route", "--model", "best", "--requires-tools"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models route error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"selected_model_ref: cloud_reasoning", "provider_ref: openrouter", "tool_definitions_allowed: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models route output = %q, want %q", out, want)
		}
	}
}

func TestModelsRouteCommandKeepsLegacyRawModelOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "config-default"
	cfg.Models = nil
	cfg.ModelAliases = nil
	cfg.ModelRouting = config.ModelRoutingConfig{}
	cfg.Providers.OpenAI = &config.ProviderConfig{APIKey: "test-key", APIBase: "https://api.openai.com/v1"}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "route", "--model", "unregistered-provider-model"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models route error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"selected_model_ref: legacy_default", "provider_ref: openai", "provider_model: unregistered-provider-model"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models route output = %q, want %q", out, want)
		}
	}
}

func TestConfigValidateCommandRejectsInvalidModelRegistry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Models = map[string]config.ModelProfileConfig{
		"bad_model": {Provider: "missing", ProviderModel: "model"},
	}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "validate"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("config validate error = nil, want model registry failure")
	}
	if !strings.Contains(stderr.String(), `unknown provider_ref "missing"`) {
		t.Fatalf("stderr = %q, want model registry error", stderr.String())
	}
}

func TestModelsHealthCommandResolvesAliasWithoutSecrets(t *testing.T) {
	const secret = "health-router-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen3-test"}]}`))
	}))
	defer server.Close()
	writeModelsHealthCommandConfig(t, server.URL+"/v1", secret)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "health", "phone"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models health error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"model_ref: local_fast", "provider_ref: llamacpp_phone", "status: healthy", "fallback_available: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models health output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, secret) {
		t.Fatalf("models health output leaked API key: %q", out)
	}
}

func TestModelsHealthCommandSupportsLegacyProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"legacy"}]}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "legacy-model"
	cfg.Models = nil
	cfg.ModelAliases = nil
	cfg.ModelRouting = config.ModelRoutingConfig{}
	cfg.Providers.OpenAI = &config.ProviderConfig{APIKey: "legacy-health-secret", APIBase: server.URL + "/v1"}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "health", "legacy_default"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models health error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"model_ref: legacy_default", "provider_ref: openai", "status: healthy"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models health output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "legacy-health-secret") {
		t.Fatalf("models health output leaked API key: %q", out)
	}
}

func TestModelsSetupDryRunUsesMinimalSnapshotWithoutSideEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "setup", "--dry-run", "--preset", "phone-ollama-tiny", "--config", cfgPath})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models setup dry-run error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"preset: phone-ollama-tiny",
		"status: manual_required",
		"platform: unknown",
		"provider_ref: ollama_phone",
		"model_ref: local_fast",
		"bind_address: 127.0.0.1",
		"side_effect: install_runtime",
		"manual_required",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("models setup dry-run output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "sk-") || strings.Contains(out, "Authorization") {
		t.Fatalf("models setup dry-run output leaked secret-shaped data: %q", out)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exist after dry-run", err)
	}
}

func TestModelsSetupWithoutDryRunCanAbortBeforeSideEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "setup", "--preset", "phone-ollama-tiny", "--config", cfgPath})
	cmd.SetIn(strings.NewReader("n\n"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models setup abort error = %v", err)
	}
	if !strings.Contains(stdout.String(), "setup_aborted: true") {
		t.Fatalf("models setup output = %q, want abort", stdout.String())
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exist after blocked setup", err)
	}
}

func TestModelsSetupInteractiveAbortHasNoSideEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "setup", "--config", cfgPath})
	cmd.SetIn(strings.NewReader("1\nn\n"))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models setup interactive abort error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"Available model setup presets:", "Proceed? [y/N]", "setup_aborted: true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("models setup output = %q, want %q", out, want)
		}
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exist after interactive abort", err)
	}
}

func TestModelsSetupNonInteractiveRequiresPreset(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "setup", "--non-interactive"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("models setup error = nil, want missing preset")
	}
	if !strings.Contains(err.Error(), "preset is required") {
		t.Fatalf("error = %v, want preset required", err)
	}
}

func TestModelsSetupApproveRejectsUnsafePlanBeforeExecutor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "setup", "--preset", "lan-llamacpp-local", "--approve", "--config", cfgPath})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("models setup approve error = nil, want blocked unsafe plan")
	}
	if !strings.Contains(err.Error(), "plan status is blocked") {
		t.Fatalf("error = %v, want blocked plan", err)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exist after rejected approve", err)
	}
}

func TestModelsPresetsCommandsExposeCatalog(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"models", "presets", "list"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("models presets list error = %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"phone-ollama-tiny", "lan-llamacpp-local", "DEFAULT_SAFE"} {
		if !strings.Contains(out, want) {
			t.Fatalf("presets list output = %q, want %q", out, want)
		}
	}

	cmd = NewRootCmd()
	cmd.SetArgs([]string{"models", "presets", "inspect", "lan-llamacpp-local"})
	stdout.Reset()
	cmd.SetOut(&stdout)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("models presets inspect error = %v", err)
	}
	out = stdout.String()
	for _, want := range []string{"explicitly_gated: true", "bind_address: 127.0.0.1", "authority_tier: low"} {
		if !strings.Contains(out, want) {
			t.Fatalf("presets inspect output = %q, want %q", out, want)
		}
	}
}

func writeModelsCommandConfig(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	tempCloud := 0.5
	tempLocal := 0.3
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "cloud_reasoning"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter":     {Type: config.ProviderTypeOpenAICompatible, APIKey: "router-secret", APIBase: "https://openrouter.ai/api/v1"},
		"llamacpp_phone": {Type: config.ProviderTypeOpenAICompatible, APIKey: "not-needed", APIBase: "http://127.0.0.1:8080/v1"},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"local_fast": {
			Provider:      "llamacpp_phone",
			ProviderModel: "qwen3-1.7b-q8_0",
			DisplayName:   "Qwen3 phone local",
			Capabilities: config.ModelCapabilities{
				Local:           true,
				Offline:         true,
				ContextTokens:   4096,
				MaxOutputTokens: 1024,
				AuthorityTier:   config.ModelAuthorityLow,
				CostTier:        config.ModelCostFree,
				LatencyTier:     config.ModelLatencySlow,
			},
			Request: config.ModelRequestConfig{MaxTokens: 1024, Temperature: &tempLocal, TimeoutS: 300},
		},
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "google/gemini-2.5-flash",
			Capabilities: config.ModelCapabilities{
				SupportsTools:   true,
				ContextTokens:   1000000,
				MaxOutputTokens: 8192,
				AuthorityTier:   config.ModelAuthorityHigh,
				CostTier:        config.ModelCostStandard,
				LatencyTier:     config.ModelLatencyNormal,
			},
			Request: config.ModelRequestConfig{MaxTokens: 8192, Temperature: &tempCloud, TimeoutS: 120},
		},
	}
	cfg.ModelAliases = map[string]string{"phone": "local_fast", "best": "cloud_reasoning"}
	cfg.ModelRouting = config.ModelRoutingConfig{
		DefaultModel:        "cloud_reasoning",
		LocalPreferredModel: "local_fast",
		Fallbacks:           map[string][]string{"local_fast": {"cloud_reasoning"}, "cloud_reasoning": {}},
	}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
}

func writeModelsHealthCommandConfig(t *testing.T, apiBase string, secret string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "local_fast"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"llamacpp_phone": {Type: config.ProviderTypeOpenAICompatible, APIKey: secret, APIBase: apiBase},
		"openrouter":     {Type: config.ProviderTypeOpenAICompatible, APIKey: "router-secret", APIBase: "https://openrouter.example/v1"},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"local_fast": {
			Provider:      "llamacpp_phone",
			ProviderModel: "qwen3-test-local",
			Capabilities: config.ModelCapabilities{
				Local:         true,
				Offline:       true,
				AuthorityTier: config.ModelAuthorityLow,
				CostTier:      config.ModelCostFree,
				LatencyTier:   config.ModelLatencySlow,
			},
		},
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "google/gemini-test",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
				CostTier:      config.ModelCostStandard,
				LatencyTier:   config.ModelLatencyNormal,
			},
		},
	}
	cfg.ModelAliases = map[string]string{"phone": "local_fast"}
	cfg.ModelRouting = config.ModelRoutingConfig{
		DefaultModel: "local_fast",
		Fallbacks:    map[string][]string{"local_fast": {"cloud_reasoning"}},
	}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
}
