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
	"sort"
	"strings"
	"sync"
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
	Now                        time.Time
	Timeout                    time.Duration
	Client                     *http.Client
	LocalRuntimes              map[string]config.LocalRuntimeConfig
	SkipProviderModelsEndpoint bool
}

type ModelHealthCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	entries    map[string]modelHealthCacheEntry
}

type modelHealthCacheEntry struct {
	result    ModelHealthResult
	expiresAt time.Time
}

func NewModelHealthCache(ttl time.Duration, maxEntries int) *ModelHealthCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if maxEntries <= 0 {
		maxEntries = 128
	}
	return &ModelHealthCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    make(map[string]modelHealthCacheEntry),
	}
}

func (c *ModelHealthCache) CheckModelHealth(ctx context.Context, reg config.ModelRegistry, modelRefOrAlias string, opts ModelHealthCheckOptions) (ModelHealthResult, error) {
	if c == nil {
		return CheckModelHealth(ctx, reg, modelRefOrAlias, opts)
	}
	modelRef, err := reg.ResolveModelRef(modelRefOrAlias)
	if err != nil {
		return ModelHealthResult{}, err
	}
	now := modelHealthNow(opts.Now)

	c.mu.Lock()
	if entry, ok := c.entries[modelRef]; ok && now.Before(entry.expiresAt) {
		result := entry.result
		c.mu.Unlock()
		return result, nil
	}
	c.mu.Unlock()

	opts.Now = now
	result, err := CheckModelHealth(ctx, reg, modelRef, opts)
	if err != nil {
		return ModelHealthResult{}, err
	}
	c.Store(result, now)
	return result, nil
}

func (c *ModelHealthCache) Store(result ModelHealthResult, now time.Time) {
	if c == nil || strings.TrimSpace(result.ModelRef) == "" {
		return
	}
	now = modelHealthNow(now)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[result.ModelRef] = modelHealthCacheEntry{
		result:    result,
		expiresAt: now.Add(c.ttl),
	}
	c.enforceLimitLocked()
}

func (c *ModelHealthCache) Snapshot(reg config.ModelRegistry, now time.Time) []ModelHealthResult {
	if c == nil {
		return ModelHealthUnknownSnapshot(reg, now)
	}
	now = modelHealthNow(now)
	refs := make([]string, 0, len(reg.Models))
	for ref := range reg.Models {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	c.mu.Lock()
	defer c.mu.Unlock()
	results := make([]ModelHealthResult, 0, len(refs))
	for _, ref := range refs {
		if entry, ok := c.entries[ref]; ok && now.Before(entry.expiresAt) {
			results = append(results, entry.result)
			continue
		}
		results = append(results, unknownModelHealthResult(reg, ref, now))
	}
	return results
}

func (c *ModelHealthCache) enforceLimitLocked() {
	if c.maxEntries <= 0 || len(c.entries) <= c.maxEntries {
		return
	}
	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := c.entries[keys[i]]
		right := c.entries[keys[j]]
		if !left.expiresAt.Equal(right.expiresAt) {
			return left.expiresAt.Before(right.expiresAt)
		}
		return keys[i] < keys[j]
	})
	for len(c.entries) > c.maxEntries && len(keys) > 0 {
		delete(c.entries, keys[0])
		keys = keys[1:]
	}
}

func ModelHealthUnknownSnapshot(reg config.ModelRegistry, now time.Time) []ModelHealthResult {
	now = modelHealthNow(now)
	refs := make([]string, 0, len(reg.Models))
	for ref := range reg.Models {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	results := make([]ModelHealthResult, 0, len(refs))
	for _, ref := range refs {
		results = append(results, unknownModelHealthResult(reg, ref, now))
	}
	return results
}

func unknownModelHealthResult(reg config.ModelRegistry, modelRef string, now time.Time) ModelHealthResult {
	model := reg.Models[modelRef]
	return ModelHealthResult{
		ModelRef:          model.Ref,
		ProviderRef:       model.ProviderRef,
		Status:            ModelHealthUnknown,
		LastCheckedAt:     modelHealthNow(now).Format(time.RFC3339),
		FallbackAvailable: len(reg.Routing.Fallbacks[model.Ref]) > 0,
	}
}

func CheckModelHealth(ctx context.Context, reg config.ModelRegistry, modelRefOrAlias string, opts ModelHealthCheckOptions) (ModelHealthResult, error) {
	modelRef, err := reg.ResolveModelRef(modelRefOrAlias)
	if err != nil {
		return ModelHealthResult{}, err
	}
	model := reg.Models[modelRef]
	now := modelHealthNow(opts.Now)
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

	healthURL, schemaRequired := modelHealthURL(model.ProviderRef, providerCfg, opts.LocalRuntimes, opts.SkipProviderModelsEndpoint)
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

func modelHealthNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func modelHealthURL(providerRef string, providerCfg config.ProviderConfig, runtimes map[string]config.LocalRuntimeConfig, skipProviderModelsEndpoint bool) (string, bool) {
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

	if skipProviderModelsEndpoint {
		return "", false
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
