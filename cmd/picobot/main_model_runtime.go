package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/providers"
)

type runtimeModelSelection struct {
	Registry config.ModelRegistry
	Route    config.ModelRoute
	Provider providers.LLMProvider
}

func resolveRuntimeModelSelection(cfg config.Config, explicitModel string) (runtimeModelSelection, error) {
	return resolveRuntimeModelSelectionWithContext(context.Background(), cfg, explicitModel)
}

func resolveRuntimeModelSelectionWithContext(ctx context.Context, cfg config.Config, explicitModel string) (runtimeModelSelection, error) {
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		return runtimeModelSelection{}, fmt.Errorf("failed to build model registry: %w", err)
	}
	unavailable := runtimeUnavailableModels(ctx, reg, cfg.LocalRuntimes)
	route, err := reg.Route(config.ModelRouteOptions{
		ExplicitModel:          explicitModel,
		AllowFallback:          runtimeModelFallbackConfigured(reg),
		UnavailableModels:      unavailable,
		AllowRawProviderModel:  true,
		RawProviderModelSource: config.RouteReasonCLIOverride,
	})
	if err != nil {
		return runtimeModelSelection{}, err
	}
	provider, err := providers.NewProviderFromModelRoute(reg, route)
	if err != nil {
		return runtimeModelSelection{}, err
	}
	return runtimeModelSelection{
		Registry: reg,
		Route:    route,
		Provider: provider,
	}, nil
}

func runtimeUnavailableModels(ctx context.Context, reg config.ModelRegistry, runtimes map[string]config.LocalRuntimeConfig) map[string]bool {
	if len(runtimes) == 0 || len(reg.Models) == 0 {
		return nil
	}

	unavailable := make(map[string]bool)
	for modelRef, model := range reg.Models {
		if !runtimeProviderHasHealthURL(model.ProviderRef, runtimes) {
			continue
		}
		result, err := providers.CheckModelHealth(ctx, reg, modelRef, providers.ModelHealthCheckOptions{
			Timeout:                    2 * time.Second,
			LocalRuntimes:              runtimes,
			SkipProviderModelsEndpoint: true,
		})
		if err != nil {
			unavailable[modelRef] = true
			continue
		}
		if result.Status == providers.ModelHealthUnhealthy {
			unavailable[modelRef] = true
		}
	}
	if len(unavailable) == 0 {
		return nil
	}
	return unavailable
}

func runtimeProviderHasHealthURL(providerRef string, runtimes map[string]config.LocalRuntimeConfig) bool {
	normalizedProvider, err := config.NormalizeProviderRef(providerRef)
	if err != nil {
		return false
	}
	for key, runtime := range runtimes {
		if strings.TrimSpace(runtime.HealthURL) == "" {
			continue
		}
		runtimeProvider, err := config.NormalizeProviderRef(runtime.Provider)
		if err != nil || runtimeProvider == "" {
			runtimeProvider, _ = config.NormalizeProviderRef(key)
		}
		if runtimeProvider == normalizedProvider {
			return true
		}
	}
	return false
}

func runtimeModelFallbackConfigured(reg config.ModelRegistry) bool {
	for _, fallbacks := range reg.Routing.Fallbacks {
		if len(fallbacks) > 0 {
			return true
		}
	}
	return false
}
