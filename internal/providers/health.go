package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
)

type ModelHealthStatus string

const (
	ModelHealthUnknown   ModelHealthStatus = "unknown"
	ModelHealthHealthy   ModelHealthStatus = "healthy"
	ModelHealthUnhealthy ModelHealthStatus = "unhealthy"
	ModelHealthDisabled  ModelHealthStatus = "disabled"
)

type ModelHealthErrorClass string

const (
	ModelHealthErrorNone              ModelHealthErrorClass = ""
	ModelHealthErrorConnectionRefused ModelHealthErrorClass = "connection_refused"
	ModelHealthErrorTimeout           ModelHealthErrorClass = "timeout"
	ModelHealthErrorHTTP              ModelHealthErrorClass = "http_error"
	ModelHealthErrorSchema            ModelHealthErrorClass = "schema_error"
	ModelHealthErrorAuth              ModelHealthErrorClass = "auth_error"
	ModelHealthErrorUnknown           ModelHealthErrorClass = "unknown"
)

type ModelHealthResult struct {
	ModelRef          string                `json:"model_ref"`
	ProviderRef       string                `json:"provider_ref"`
	Status            ModelHealthStatus     `json:"status"`
	LastCheckedAt     string                `json:"last_checked_at"`
	LastErrorClass    ModelHealthErrorClass `json:"last_error_class,omitempty"`
	FallbackAvailable bool                  `json:"fallback_available"`
}

type ModelHealthCheckOptions struct {
	Now           time.Time
	Timeout       time.Duration
	Client        *http.Client
	LocalRuntimes map[string]config.LocalRuntimeConfig
}

func CheckModelHealth(ctx context.Context, reg config.ModelRegistry, modelRefOrAlias string, opts ModelHealthCheckOptions) (ModelHealthResult, error) {
	modelRef, err := reg.ResolveModelRef(modelRefOrAlias)
	if err != nil {
		return ModelHealthResult{}, err
	}
	model := reg.Models[modelRef]
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	result := ModelHealthResult{
		ModelRef:          model.Ref,
		ProviderRef:       model.ProviderRef,
		Status:            ModelHealthUnknown,
		LastCheckedAt:     now.Format(time.RFC3339),
		FallbackAvailable: len(reg.Routing.Fallbacks[model.Ref]) > 0,
	}

	providerCfg, ok := reg.Providers[model.ProviderRef]
	if !ok {
		result.Status = ModelHealthUnhealthy
		result.LastErrorClass = ModelHealthErrorUnknown
		return result, nil
	}

	healthURL, schemaRequired := modelHealthURL(model.ProviderRef, providerCfg, opts.LocalRuntimes)
	if healthURL == "" {
		result.Status = ModelHealthDisabled
		return result, nil
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		result.Status = ModelHealthUnhealthy
		result.LastErrorClass = ModelHealthErrorUnknown
		return result, nil
	}
	if strings.TrimSpace(providerCfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+providerCfg.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = ModelHealthUnhealthy
		result.LastErrorClass = classifyModelHealthError(err)
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		result.Status = ModelHealthUnhealthy
		result.LastErrorClass = ModelHealthErrorAuth
		return result, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Status = ModelHealthUnhealthy
		result.LastErrorClass = ModelHealthErrorHTTP
		return result, nil
	}
	if schemaRequired {
		var payload struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || payload.Data == nil {
			result.Status = ModelHealthUnhealthy
			result.LastErrorClass = ModelHealthErrorSchema
			return result, nil
		}
	}

	result.Status = ModelHealthHealthy
	return result, nil
}

func modelHealthURL(providerRef string, providerCfg config.ProviderConfig, runtimes map[string]config.LocalRuntimeConfig) (string, bool) {
	for key, runtime := range runtimes {
		runtimeProvider, err := config.NormalizeProviderRef(runtime.Provider)
		if err != nil || runtimeProvider == "" {
			runtimeProvider, _ = config.NormalizeProviderRef(key)
		}
		if runtimeProvider != providerRef {
			continue
		}
		if strings.TrimSpace(runtime.HealthURL) != "" {
			return strings.TrimSpace(runtime.HealthURL), false
		}
	}

	base := strings.TrimRight(strings.TrimSpace(providerCfg.APIBase), "/")
	if base == "" {
		return "", false
	}
	return base + "/models", true
}

func classifyModelHealthError(err error) ModelHealthErrorClass {
	if err == nil {
		return ModelHealthErrorNone
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return ModelHealthErrorTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ModelHealthErrorTimeout
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		text := strings.ToLower(urlErr.Err.Error())
		if strings.Contains(text, "connection refused") || strings.Contains(text, "connectex") {
			return ModelHealthErrorConnectionRefused
		}
	}
	text := strings.ToLower(fmt.Sprintf("%v", err))
	if strings.Contains(text, "connection refused") || strings.Contains(text, "connectex") {
		return ModelHealthErrorConnectionRefused
	}
	return ModelHealthErrorUnknown
}
