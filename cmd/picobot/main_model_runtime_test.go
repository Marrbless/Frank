package main

import (
	"bytes"
	"context"
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

func TestResolveRuntimeModelSelectionPreflightFallbackAllowed(t *testing.T) {
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("local path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer localServer.Close()

	cfg := v5FallbackRuntimeTestConfig("http://127.0.0.1:1/v1", localServer.URL+"/health", "https://cloud.example/v1")
	cfg.ModelRouting.AllowCloudFallbackFromLocal = true

	selection, err := resolveRuntimeModelSelectionWithContext(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("resolveRuntimeModelSelectionWithContext() error = %v", err)
	}
	if selection.Route.SelectedModelRef != "cloud_reasoning" {
		t.Fatalf("selected model = %q, want cloud_reasoning", selection.Route.SelectedModelRef)
	}
	if selection.Route.SelectionReason != config.RouteReasonFallback {
		t.Fatalf("selection reason = %q, want fallback", selection.Route.SelectionReason)
	}
	if selection.Route.FallbackDepth != 1 {
		t.Fatalf("fallback depth = %d, want 1", selection.Route.FallbackDepth)
	}
	if selection.Route.ProviderModel != "google/gemini-test" {
		t.Fatalf("provider model = %q, want fallback provider model", selection.Route.ProviderModel)
	}
}

func TestResolveRuntimeModelSelectionPreflightFallbackDisabled(t *testing.T) {
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer localServer.Close()

	cfg := v5FallbackRuntimeTestConfig("http://127.0.0.1:1/v1", localServer.URL, "https://cloud.example/v1")
	cfg.ModelRouting.Fallbacks = nil
	cfg.ModelRouting.AllowCloudFallbackFromLocal = true

	_, err := resolveRuntimeModelSelectionWithContext(context.Background(), cfg, "")
	if err == nil {
		t.Fatal("resolveRuntimeModelSelectionWithContext() error = nil, want fallback-disabled error")
	}
	if !strings.Contains(err.Error(), "fallback is disabled") {
		t.Fatalf("error = %q, want fallback-disabled error", err.Error())
	}
}

func TestResolveRuntimeModelSelectionPreflightCloudFallbackDeniedByDefault(t *testing.T) {
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer localServer.Close()

	cfg := v5FallbackRuntimeTestConfig("http://127.0.0.1:1/v1", localServer.URL, "https://cloud.example/v1")

	_, err := resolveRuntimeModelSelectionWithContext(context.Background(), cfg, "")
	if err == nil {
		t.Fatal("resolveRuntimeModelSelectionWithContext() error = nil, want cloud-fallback denial")
	}
	if !strings.Contains(err.Error(), "cloud fallback from local") {
		t.Fatalf("error = %q, want cloud-fallback denial", err.Error())
	}
}

func TestResolveRuntimeModelSelectionPreflightLowerAuthorityDeniedByDefault(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer healthServer.Close()

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "cloud_reasoning"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "high-secret",
			APIBase: "https://cloud.example/v1",
		},
		"cheap_cloud": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "low-secret",
			APIBase: "https://cheap.example/v1",
		},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"cloud_reasoning": {
			Provider:      "openrouter",
			ProviderModel: "high-model",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
				CostTier:      config.ModelCostStandard,
				LatencyTier:   config.ModelLatencyNormal,
			},
		},
		"cheap_cloud": {
			Provider:      "cheap_cloud",
			ProviderModel: "low-model",
			Capabilities: config.ModelCapabilities{
				AuthorityTier: config.ModelAuthorityLow,
				CostTier:      config.ModelCostCheap,
				LatencyTier:   config.ModelLatencyNormal,
			},
		},
	}
	cfg.ModelRouting = config.ModelRoutingConfig{
		DefaultModel: "cloud_reasoning",
		Fallbacks:    map[string][]string{"cloud_reasoning": {"cheap_cloud"}},
	}
	cfg.LocalRuntimes = map[string]config.LocalRuntimeConfig{
		"openrouter_readiness": {
			Provider:  "openrouter",
			HealthURL: healthServer.URL,
		},
	}

	_, err := resolveRuntimeModelSelectionWithContext(context.Background(), cfg, "")
	if err == nil {
		t.Fatal("resolveRuntimeModelSelectionWithContext() error = nil, want lower-authority denial")
	}
	if !strings.Contains(err.Error(), "lower-authority fallback") {
		t.Fatalf("error = %q, want lower-authority denial", err.Error())
	}
}

func TestAgentCommandDoesNotRetryFallbackAfterProviderRequestStarts(t *testing.T) {
	var localChatCount int
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`ok`))
		case "/v1/chat/completions":
			localChatCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"local failed","token":"local-secret"}`))
		default:
			t.Fatalf("local path = %q", r.URL.Path)
		}
	}))
	defer localServer.Close()

	var cloudChatCount int
	cloudServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloudChatCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"cloud fallback should not run"}}]}`))
	}))
	defer cloudServer.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := v5FallbackRuntimeTestConfig(localServer.URL+"/v1", localServer.URL+"/health", cloudServer.URL+"/v1")
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.ModelRouting.AllowCloudFallbackFromLocal = true
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"agent", "-m", "hello", "-M", "phone"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent command error = %v", err)
	}
	if localChatCount != 1 {
		t.Fatalf("local chat count = %d, want 1", localChatCount)
	}
	if cloudChatCount != 0 {
		t.Fatalf("cloud chat count = %d, want no post-request fallback retry", cloudChatCount)
	}
	if strings.Contains(stdout.String()+stderr.String(), "local-secret") {
		t.Fatalf("agent command output leaked provider error body: stdout=%q stderr=%q", stdout.String(), stderr.String())
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

func v5FallbackRuntimeTestConfig(localAPIBase, localHealthURL, cloudAPIBase string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "local_fast"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"llamacpp_phone": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "not-needed",
			APIBase: localAPIBase,
		},
		"openrouter": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "cloud-secret",
			APIBase: cloudAPIBase,
		},
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
	cfg.ModelAliases = map[string]string{
		"best":  "cloud_reasoning",
		"phone": "local_fast",
	}
	cfg.ModelRouting = config.ModelRoutingConfig{
		DefaultModel: "local_fast",
		Fallbacks:    map[string][]string{"local_fast": {"cloud_reasoning"}},
	}
	cfg.LocalRuntimes = map[string]config.LocalRuntimeConfig{
		"llamacpp_phone": {
			Provider:  "llamacpp_phone",
			HealthURL: localHealthURL,
		},
	}
	return cfg
}
