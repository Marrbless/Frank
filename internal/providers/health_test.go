package providers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/local/picobot/internal/config"
)

func TestCheckModelHealthHealthyModelsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"model"}]}`))
	}))
	defer server.Close()

	result := checkModelHealthForTest(t, server.URL+"/v1", 0)
	if result.Status != ModelHealthHealthy {
		t.Fatalf("Status = %q, want healthy result %#v", result.Status, result)
	}
	if result.LastErrorClass != "" {
		t.Fatalf("LastErrorClass = %q, want empty", result.LastErrorClass)
	}
	if !result.FallbackAvailable {
		t.Fatal("FallbackAvailable = false, want true")
	}
}

func TestCheckModelHealthClassifiesAuthHTTPAndSchemaErrors(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantClass ModelHealthErrorClass
	}{
		{name: "auth", status: http.StatusUnauthorized, body: `{"error":"bad auth","token":"sk-secret"}`, wantClass: ModelHealthErrorAuth},
		{name: "http", status: http.StatusInternalServerError, body: `{"error":"failed","token":"sk-secret"}`, wantClass: ModelHealthErrorHTTP},
		{name: "schema", status: http.StatusOK, body: `{"not_data":[]}`, wantClass: ModelHealthErrorSchema},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			result := checkModelHealthForTest(t, server.URL+"/v1", 0)
			if result.Status != ModelHealthUnhealthy {
				t.Fatalf("Status = %q, want unhealthy result %#v", result.Status, result)
			}
			if result.LastErrorClass != tc.wantClass {
				t.Fatalf("LastErrorClass = %q, want %q", result.LastErrorClass, tc.wantClass)
			}
		})
	}
}

func TestCheckModelHealthClassifiesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	result := checkModelHealthForTest(t, server.URL+"/v1", 5*time.Millisecond)
	if result.Status != ModelHealthUnhealthy {
		t.Fatalf("Status = %q, want unhealthy", result.Status)
	}
	if result.LastErrorClass != ModelHealthErrorTimeout {
		t.Fatalf("LastErrorClass = %q, want timeout", result.LastErrorClass)
	}
}

func TestCheckModelHealthClassifiesConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	result := checkModelHealthForTest(t, "http://"+addr+"/v1", 100*time.Millisecond)
	if result.Status != ModelHealthUnhealthy {
		t.Fatalf("Status = %q, want unhealthy", result.Status)
	}
	if result.LastErrorClass != ModelHealthErrorConnectionRefused {
		t.Fatalf("LastErrorClass = %q, want connection_refused", result.LastErrorClass)
	}
}

func TestCheckModelHealthDisabledWhenNoEndpoint(t *testing.T) {
	cfg := healthTestConfig("")
	cfg.Providers.Named["openrouter"] = config.ProviderConfig{Type: config.ProviderTypeOpenAICompatible}
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	result, err := CheckModelHealth(context.Background(), reg, "best", ModelHealthCheckOptions{Now: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatalf("CheckModelHealth() error = %v", err)
	}
	if result.Status != ModelHealthDisabled {
		t.Fatalf("Status = %q, want disabled", result.Status)
	}
}

func checkModelHealthForTest(t *testing.T, apiBase string, timeout time.Duration) ModelHealthResult {
	t.Helper()
	cfg := healthTestConfig(apiBase)
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	result, err := CheckModelHealth(context.Background(), reg, "best", ModelHealthCheckOptions{
		Now:     time.Unix(0, 0).UTC(),
		Timeout: timeout,
	})
	if err != nil {
		t.Fatalf("CheckModelHealth() error = %v", err)
	}
	if result.ModelRef != "cloud_reasoning" || result.ProviderRef != "openrouter" {
		t.Fatalf("result identity = %#v, want cloud_reasoning/openrouter", result)
	}
	if result.LastCheckedAt != "1970-01-01T00:00:00Z" {
		t.Fatalf("LastCheckedAt = %q, want RFC3339 epoch", result.LastCheckedAt)
	}
	return result
}

func healthTestConfig(apiBase string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "cloud_reasoning"
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"openrouter": {
			Type:    config.ProviderTypeOpenAICompatible,
			APIKey:  "health-secret",
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
		"fallback_model": {
			Provider:      "openrouter",
			ProviderModel: "fallback-test",
			Capabilities: config.ModelCapabilities{
				SupportsTools: true,
				AuthorityTier: config.ModelAuthorityHigh,
				CostTier:      config.ModelCostStandard,
				LatencyTier:   config.ModelLatencyNormal,
			},
		},
	}
	cfg.ModelAliases = map[string]string{"best": "cloud_reasoning"}
	cfg.ModelRouting = config.ModelRoutingConfig{
		DefaultModel: "cloud_reasoning",
		Fallbacks:    map[string][]string{"cloud_reasoning": {"fallback_model"}},
	}
	return cfg
}
