package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/agent"
	"github.com/local/picobot/internal/chat"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/providers"
)

func TestResolveRuntimeModelSelectionUsesAliasNamedProviderAndProviderModel(t *testing.T) {
	cfg := v5RuntimeTestConfig("https://openrouter.example/v1")
	selection, err := resolveRuntimeModelSelection(cfg, "best")
	if err != nil {
		t.Fatalf("resolveRuntimeModelSelection() error = %v", err)
	}
	if selection.Route.SelectedModelRef != "cloud_reasoning" {
		t.Fatalf("selected model = %q, want cloud_reasoning", selection.Route.SelectedModelRef)
	}
	if selection.Route.ProviderRef != "openrouter" {
		t.Fatalf("provider ref = %q, want openrouter", selection.Route.ProviderRef)
	}
	if selection.Route.ProviderModel != "google/gemini-test" {
		t.Fatalf("provider model = %q, want google/gemini-test", selection.Route.ProviderModel)
	}
	openAI, ok := selection.Provider.(*providers.OpenAIProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *OpenAIProvider", selection.Provider)
	}
	if openAI.APIBase != "https://openrouter.example/v1" {
		t.Fatalf("APIBase = %q, want named provider base", openAI.APIBase)
	}
}

func TestResolveRuntimeModelSelectionKeepsLegacyRawModelOverride(t *testing.T) {
	cfg := config.Config{}
	cfg.Providers.OpenAI = &config.ProviderConfig{APIKey: "legacy-secret", APIBase: "https://legacy.example/v1"}

	selection, err := resolveRuntimeModelSelection(cfg, "unregistered-provider-model")
	if err != nil {
		t.Fatalf("resolveRuntimeModelSelection() error = %v", err)
	}
	if selection.Route.SelectedModelRef != config.LegacyModelRef {
		t.Fatalf("selected model = %q, want %q", selection.Route.SelectedModelRef, config.LegacyModelRef)
	}
	if selection.Route.ProviderRef != config.LegacyProviderRef {
		t.Fatalf("provider ref = %q, want %q", selection.Route.ProviderRef, config.LegacyProviderRef)
	}
	if selection.Route.ProviderModel != "unregistered-provider-model" {
		t.Fatalf("provider model = %q, want raw override", selection.Route.ProviderModel)
	}
	openAI, ok := selection.Provider.(*providers.OpenAIProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *OpenAIProvider", selection.Provider)
	}
	if openAI.APIBase != "https://legacy.example/v1" {
		t.Fatalf("APIBase = %q, want legacy provider base", openAI.APIBase)
	}
}

func TestAgentCommandUsesV5ProviderModelAtRuntime(t *testing.T) {
	const secret = "runtime-secret-key"
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("request path = %q, want /v1/chat/completions", r.URL.Path)
		}
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		capturedModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"runtime ok"}}]}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := v5RuntimeTestConfig(server.URL + "/v1")
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Providers.Named["openrouter"] = config.ProviderConfig{
		Type:    config.ProviderTypeOpenAICompatible,
		APIKey:  secret,
		APIBase: server.URL + "/v1",
	}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"agent", "-m", "hello", "-M", "best"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent command error = %v", err)
	}
	if capturedModel != "google/gemini-test" {
		t.Fatalf("captured model = %q, want provider model", capturedModel)
	}
	combinedOutput := stdout.String() + stderr.String()
	if !strings.Contains(stdout.String(), "runtime ok") {
		t.Fatalf("stdout = %q, want provider response", stdout.String())
	}
	if strings.Contains(combinedOutput, secret) {
		t.Fatalf("agent command output leaked API key: %q", combinedOutput)
	}
}

func TestAgentCommandSuppressesToolsForNoToolModel(t *testing.T) {
	var toolsPresent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, toolsPresent = body["tools"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"local ok"}}]}`))
	}))
	defer server.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := v5RuntimeTestConfig(server.URL + "/v1")
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Providers.Named["llamacpp_phone"] = config.ProviderConfig{
		Type:    config.ProviderTypeOpenAICompatible,
		APIKey:  "not-needed",
		APIBase: server.URL + "/v1",
	}
	cfg.Models["local_fast"] = config.ModelProfileConfig{
		Provider:      "llamacpp_phone",
		ProviderModel: "qwen3-test-local",
		Capabilities: config.ModelCapabilities{
			Local:         true,
			Offline:       true,
			AuthorityTier: config.ModelAuthorityLow,
			CostTier:      config.ModelCostFree,
			LatencyTier:   config.ModelLatencySlow,
		},
	}
	cfg.ModelAliases["phone"] = "local_fast"
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"agent", "-m", "hello", "-M", "phone"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent command error = %v", err)
	}
	if toolsPresent {
		t.Fatal("provider request included tools for supportsTools=false model")
	}
	if !strings.Contains(stdout.String(), "local ok") {
		t.Fatalf("stdout = %q, want provider response", stdout.String())
	}
}

func TestMissionStatusSnapshotIncludesSafeModelRoute(t *testing.T) {
	const secret = "status-secret-key"
	cfg := v5RuntimeTestConfig("https://openrouter.example/v1")
	cfg.Providers.Named["openrouter"] = config.ProviderConfig{
		Type:    config.ProviderTypeOpenAICompatible,
		APIKey:  secret,
		APIBase: "https://openrouter.example/v1",
	}
	selection, err := resolveRuntimeModelSelection(cfg, "best")
	if err != nil {
		t.Fatalf("resolveRuntimeModelSelection() error = %v", err)
	}
	ag := agent.NewAgentLoop(chat.NewHub(10), selection.Provider, selection.Route.ProviderModel, 3, t.TempDir(), nil)
	defer ag.Close()
	ag.SetModelRoute(selection.Route)
	if err := ag.ActivateMissionStep(testMissionBootstrapJob(), "build"); err != nil {
		t.Fatalf("ActivateMissionStep() error = %v", err)
	}

	statusPath := filepath.Join(t.TempDir(), "mission-status.json")
	if err := writeMissionStatusSnapshot(statusPath, "mission.json", ag, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeMissionStatusSnapshot() error = %v", err)
	}
	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("mission status leaked API key: %s", data)
	}

	var snapshot missionStatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if snapshot.Model == nil {
		t.Fatal("snapshot.Model = nil, want route")
	}
	if snapshot.Model.SelectedModelRef != "cloud_reasoning" || snapshot.Model.ProviderRef != "openrouter" || snapshot.Model.ProviderModel != "google/gemini-test" {
		t.Fatalf("snapshot.Model = %#v, want safe route fields", snapshot.Model)
	}
	if snapshot.Model.Capabilities.AuthorityTier != "high" || !snapshot.Model.Capabilities.SupportsTools {
		t.Fatalf("snapshot.Model.Capabilities = %#v, want high tool-capable model", snapshot.Model.Capabilities)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.Model == nil {
		t.Fatalf("snapshot.RuntimeSummary.Model = %#v, want route", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.Model.ProviderModel != "google/gemini-test" {
		t.Fatalf("runtime_summary.model.provider_model = %q, want google/gemini-test", snapshot.RuntimeSummary.Model.ProviderModel)
	}
}

func v5RuntimeTestConfig(apiBase string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "cloud_reasoning"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "named-secret",
			APIBase: apiBase,
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
		},
	}
	cfg.ModelAliases = map[string]string{"best": "cloud_reasoning"}
	cfg.ModelRouting = config.ModelRoutingConfig{DefaultModel: "cloud_reasoning"}
	return cfg
}
