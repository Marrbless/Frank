package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
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

func newRuntimeMissionModelRouter(ctx context.Context, cfg config.Config, explicitModel string) func(missioncontrol.Job, string) (config.ModelRoute, providers.LLMProvider, bool, error) {
	return func(job missioncontrol.Job, stepID string) (config.ModelRoute, providers.LLMProvider, bool, error) {
		selection, ok, err := resolveRuntimeMissionModelSelectionWithContext(ctx, cfg, explicitModel, job, stepID)
		if err != nil {
			return config.ModelRoute{}, nil, false, err
		}
		if !ok {
			return config.ModelRoute{}, nil, false, nil
		}
		return selection.Route, selection.Provider, true, nil
	}
}

func resolveRuntimeMissionModelSelectionWithContext(ctx context.Context, cfg config.Config, explicitModel string, job missioncontrol.Job, stepID string) (runtimeModelSelection, bool, error) {
	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return runtimeModelSelection{}, false, err
	}
	policy, policyID := missioncontrol.EffectiveModelPolicy(ec.Job, ec.Step)
	if policy == nil {
		return runtimeModelSelection{}, false, nil
	}

	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		return runtimeModelSelection{}, false, fmt.Errorf("failed to build model registry: %w", err)
	}
	required, err := modelPolicyRequiredCapabilities(policy)
	if err != nil {
		return runtimeModelSelection{}, false, modelPolicyRouteError(stepID, err)
	}
	route, err := reg.Route(config.ModelRouteOptions{
		ExplicitModel:          explicitModel,
		DefaultModel:           policy.DefaultModel,
		AllowedModels:          policy.AllowedModels,
		AllowFallback:          modelPolicyAllowFallback(policy, reg),
		RequiredCapability:     required,
		UnavailableModels:      runtimeUnavailableModels(ctx, reg, cfg.LocalRuntimes),
		AllowRawProviderModel:  false,
		RawProviderModelSource: config.RouteReasonCLIOverride,
		PolicyID:               policyID,
	})
	if err != nil {
		return runtimeModelSelection{}, false, modelPolicyRouteError(stepID, err)
	}
	provider, err := providers.NewProviderFromModelRoute(reg, route)
	if err != nil {
		return runtimeModelSelection{}, false, err
	}
	return runtimeModelSelection{
		Registry: reg,
		Route:    route,
		Provider: provider,
	}, true, nil
}

func modelPolicyRequiredCapabilities(policy *missioncontrol.ModelPolicy) (config.ModelRequiredCapabilities, error) {
	var required config.ModelRequiredCapabilities
	if policy == nil {
		return required, nil
	}

	capabilities := policy.RequiredCapabilities
	if capabilities.SupportsTools != nil && *capabilities.SupportsTools {
		required.SupportsTools = true
	}
	required.Local = cloneBoolPtr(capabilities.Local)
	required.Offline = cloneBoolPtr(capabilities.Offline)
	required.SupportsResponsesAPI = cloneBoolPtr(capabilities.SupportsResponsesAPI)
	if capabilities.AuthorityTierAtLeast != "" {
		required.AuthorityTierAtLeast = config.ModelAuthorityTier(capabilities.AuthorityTierAtLeast)
	}
	if policy.AllowCloud != nil && !*policy.AllowCloud {
		if required.Local != nil && !*required.Local {
			return config.ModelRequiredCapabilities{}, fmt.Errorf("model_policy.allow_cloud=false conflicts with required_capabilities.local=false")
		}
		local := true
		required.Local = &local
	}
	return required, nil
}

func modelPolicyAllowFallback(policy *missioncontrol.ModelPolicy, reg config.ModelRegistry) bool {
	if policy != nil && policy.AllowFallback != nil {
		return *policy.AllowFallback
	}
	return runtimeModelFallbackConfigured(reg)
}

func modelPolicyRouteError(stepID string, err error) error {
	return missioncontrol.ValidationError{
		Code:    missioncontrol.RejectionCodeInvalidModelPolicy,
		StepID:  stepID,
		Message: "model_policy denied route: " + err.Error(),
	}
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
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
