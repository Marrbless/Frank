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

func TestModelHealthCacheHitAvoidsSecondProbe(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"data":[{"id":"model"}]}`))
	}))
	defer server.Close()

	cfg := healthTestConfig(server.URL + "/v1")
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	cache := NewModelHealthCache(time.Minute, 8)
	opts := ModelHealthCheckOptions{Now: time.Unix(10, 0).UTC()}
	first, err := cache.CheckModelHealth(context.Background(), reg, "best", opts)
	if err != nil {
		t.Fatalf("CheckModelHealth(first) error = %v", err)
	}
	opts.Now = time.Unix(20, 0).UTC()
	second, err := cache.CheckModelHealth(context.Background(), reg, "cloud_reasoning", opts)
	if err != nil {
		t.Fatalf("CheckModelHealth(second) error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("health probe calls = %d, want 1 cache hit", calls)
	}
	if first.LastCheckedAt != second.LastCheckedAt {
		t.Fatalf("cached LastCheckedAt = %q, want first %q", second.LastCheckedAt, first.LastCheckedAt)
	}
}

func TestModelHealthCacheRefreshAfterTTL(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"model"}]}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"failed","api_key":"sk-health-secret"}`))
	}))
	defer server.Close()

	cfg := healthTestConfig(server.URL + "/v1")
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	cache := NewModelHealthCache(time.Second, 8)
	if _, err := cache.CheckModelHealth(context.Background(), reg, "best", ModelHealthCheckOptions{Now: time.Unix(10, 0).UTC()}); err != nil {
		t.Fatalf("CheckModelHealth(first) error = %v", err)
	}
	refreshed, err := cache.CheckModelHealth(context.Background(), reg, "best", ModelHealthCheckOptions{Now: time.Unix(12, 0).UTC()})
	if err != nil {
		t.Fatalf("CheckModelHealth(refreshed) error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("health probe calls = %d, want refresh", calls)
	}
	if refreshed.Status != ModelHealthUnhealthy || refreshed.LastErrorClass != ModelHealthErrorHTTP {
		t.Fatalf("refreshed result = %#v, want unhealthy http_error", refreshed)
	}
}

func TestModelHealthCacheSnapshotIncludesUnknownAndCachedStatuses(t *testing.T) {
	cfg := healthTestConfig("")
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	cache := NewModelHealthCache(time.Minute, 8)
	cache.Store(ModelHealthResult{
		ModelRef:          "cloud_reasoning",
		ProviderRef:       "openrouter",
		Status:            ModelHealthHealthy,
		LastCheckedAt:     "2026-05-01T12:00:00Z",
		FallbackAvailable: true,
	}, time.Unix(10, 0).UTC())

	snapshot := cache.Snapshot(reg, time.Unix(20, 0).UTC())
	if len(snapshot) != 2 {
		t.Fatalf("Snapshot len = %d, want 2: %#v", len(snapshot), snapshot)
	}
	if snapshot[0].ModelRef != "cloud_reasoning" || snapshot[0].Status != ModelHealthHealthy {
		t.Fatalf("Snapshot[0] = %#v, want cached healthy cloud_reasoning", snapshot[0])
	}
	if snapshot[1].ModelRef != "fallback_model" || snapshot[1].Status != ModelHealthUnknown {
		t.Fatalf("Snapshot[1] = %#v, want unknown fallback_model", snapshot[1])
	}
	if snapshot[1].LastCheckedAt != "1970-01-01T00:00:20Z" {
		t.Fatalf("unknown LastCheckedAt = %q, want snapshot time", snapshot[1].LastCheckedAt)
	}
}

func TestModelHealthCacheDisabledProvider(t *testing.T) {
	cfg := healthTestConfig("")
	cfg.Providers.Named["openrouter"] = config.ProviderConfig{Type: config.ProviderTypeOpenAICompatible}
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildModelRegistry() error = %v", err)
	}
	cache := NewModelHealthCache(time.Minute, 8)
	result, err := cache.CheckModelHealth(context.Background(), reg, "best", ModelHealthCheckOptions{Now: time.Unix(0, 0).UTC()})
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
