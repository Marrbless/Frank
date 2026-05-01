package modelsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
)

type ReadinessStatus string

const (
	ReadinessHealthy        ReadinessStatus = "healthy"
	ReadinessUnhealthy      ReadinessStatus = "unhealthy"
	ReadinessManualRequired ReadinessStatus = "manual_required"
	ReadinessUnknown        ReadinessStatus = "unknown"
)

type ReadinessResult struct {
	ModelRef      string
	ProviderRef   string
	Status        ReadinessStatus
	ErrorClass    string
	URL           string
	RouteProvider string
	RouteModel    string
	CheckedAt     string
}

type ReadinessOptions struct {
	Client        *http.Client
	Timeout       time.Duration
	Now           time.Time
	LocalRuntimes map[string]config.LocalRuntimeConfig
}

func CheckNoPromptReadiness(ctx context.Context, reg config.ModelRegistry, modelRefOrAlias string, opts ReadinessOptions) (ReadinessResult, error) {
	modelRef, err := reg.ResolveModelRef(modelRefOrAlias)
	if err != nil {
		return ReadinessResult{}, err
	}
	model := reg.Models[modelRef]
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := ReadinessResult{
		ModelRef:    model.Ref,
		ProviderRef: model.ProviderRef,
		Status:      ReadinessUnknown,
		CheckedAt:   now.UTC().Format(time.RFC3339),
	}
	route, err := reg.Route(config.ModelRouteOptions{ExplicitModel: model.Ref})
	if err != nil {
		result.Status = ReadinessUnhealthy
		result.ErrorClass = "route_error"
		return result, nil
	}
	result.RouteProvider = route.ProviderRef
	result.RouteModel = route.ProviderModel

	providerCfg, ok := reg.Providers[model.ProviderRef]
	if !ok {
		result.Status = ReadinessUnhealthy
		result.ErrorClass = "missing_provider"
		return result, nil
	}
	if !model.Capabilities.Local {
		result.Status = ReadinessManualRequired
		result.ErrorClass = "cloud_probe_skipped"
		return result, nil
	}
	readinessURL, schemaRequired := noPromptReadinessURL(model.ProviderRef, providerCfg, opts.LocalRuntimes)
	if readinessURL == "" {
		result.Status = ReadinessManualRequired
		result.ErrorClass = "metadata_endpoint_missing"
		return result, nil
	}
	result.URL = readinessURL
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
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, readinessURL, nil)
	if err != nil {
		result.Status = ReadinessUnhealthy
		result.ErrorClass = "request_error"
		return result, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		result.Status = ReadinessUnhealthy
		result.ErrorClass = "connection_error"
		return result, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Status = ReadinessUnhealthy
		result.ErrorClass = "http_error"
		return result, nil
	}
	if schemaRequired {
		var payload struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&payload); err != nil || payload.Data == nil {
			result.Status = ReadinessUnhealthy
			result.ErrorClass = "schema_error"
			return result, nil
		}
	}
	result.Status = ReadinessHealthy
	return result, nil
}

func noPromptReadinessURL(providerRef string, providerCfg config.ProviderConfig, runtimes map[string]config.LocalRuntimeConfig) (string, bool) {
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
	if !isLocalHTTPBase(base) {
		return "", false
	}
	return base + "/models", true
}

func isLocalHTTPBase(base string) bool {
	return strings.HasPrefix(base, "http://127.0.0.1:") || strings.HasPrefix(base, "http://localhost:")
}

func FormatReadinessResult(result ReadinessResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "model_ref: %s\n", result.ModelRef)
	fmt.Fprintf(&b, "provider_ref: %s\n", result.ProviderRef)
	fmt.Fprintf(&b, "status: %s\n", result.Status)
	fmt.Fprintf(&b, "checked_at: %s\n", result.CheckedAt)
	if result.ErrorClass != "" {
		fmt.Fprintf(&b, "error_class: %s\n", result.ErrorClass)
	}
	if result.RouteProvider != "" {
		fmt.Fprintf(&b, "route_provider_ref: %s\n", result.RouteProvider)
	}
	if result.RouteModel != "" {
		fmt.Fprintf(&b, "route_provider_model: %s\n", result.RouteModel)
	}
	return b.String()
}
