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
	Registry                   config.ModelRegistry
	Route                      config.ModelRoute
	Provider                   providers.LLMProvider
	ProviderHealthFailureCount int64
	HealthResults              []providers.ModelHealthResult
}

func resolveRuntimeModelSelection(cfg config.Config, explicitModel string) (runtimeModelSelection, error) {
	return resolveRuntimeModelSelectionWithContext(context.Background(), cfg, explicitModel)
}

func resolveRuntimeModelSelectionWithContext(ctx context.Context, cfg config.Config, explicitModel string) (runtimeModelSelection, error) {
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		return runtimeModelSelection{}, fmt.Errorf("failed to build model registry: %w", err)
	}
	unavailable, healthFailures, healthResults := runtimeUnavailableModelsWithHealthFailures(ctx, reg, cfg.LocalRuntimes)
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
		Registry:                   reg,
		Route:                      route,
		Provider:                   provider,
		ProviderHealthFailureCount: healthFailures,
		HealthResults:              healthResults,
	}, nil
}

func newRuntimeMissionModelRouter(ctx context.Context, cfg config.Config, explicitModel string) func(missioncontrol.Job, string) (config.ModelRoute, providers.LLMProvider, missioncontrol.OperatorModelControlMetricsStatus, []missioncontrol.OperatorModelHealthStatus, bool, error) {
	return func(job missioncontrol.Job, stepID string) (config.ModelRoute, providers.LLMProvider, missioncontrol.OperatorModelControlMetricsStatus, []missioncontrol.OperatorModelHealthStatus, bool, error) {
		selection, ok, err := resolveRuntimeMissionModelSelectionWithContext(ctx, cfg, explicitModel, job, stepID)
		if err != nil {
			return config.ModelRoute{}, nil, missioncontrol.OperatorModelControlMetricsStatus{}, nil, false, err
		}
		if !ok {
			return config.ModelRoute{}, nil, missioncontrol.OperatorModelControlMetricsStatus{}, nil, false, nil
		}
		metrics := missioncontrol.OperatorModelControlMetricsStatus{
			ProviderHealthFailureCount: selection.ProviderHealthFailureCount,
		}
		return selection.Route, selection.Provider, metrics, modelHealthStatusesFromResults(selection.HealthResults), true, nil
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
	unavailable, healthFailures, healthResults := runtimeUnavailableModelsWithHealthFailures(ctx, reg, cfg.LocalRuntimes)
	route, err := reg.Route(config.ModelRouteOptions{
		ExplicitModel:          explicitModel,
		DefaultModel:           policy.DefaultModel,
		AllowedModels:          policy.AllowedModels,
		AllowFallback:          modelPolicyAllowFallback(policy, reg),
		RequiredCapability:     required,
		UnavailableModels:      unavailable,
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
		Registry:                   reg,
		Route:                      route,
		Provider:                   provider,
		ProviderHealthFailureCount: healthFailures,
		HealthResults:              healthResults,
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
	unavailable, _, _ := runtimeUnavailableModelsWithHealthFailures(ctx, reg, runtimes)
	return unavailable
}

func runtimeUnavailableModelsWithHealthFailures(ctx context.Context, reg config.ModelRegistry, runtimes map[string]config.LocalRuntimeConfig) (map[string]bool, int64, []providers.ModelHealthResult) {
	if len(runtimes) == 0 || len(reg.Models) == 0 {
		return nil, 0, providers.ModelHealthUnknownSnapshot(reg, time.Now().UTC())
	}

	unavailable := make(map[string]bool)
	var healthFailures int64
	healthResultsByModel := make(map[string]providers.ModelHealthResult, len(reg.Models))
	for modelRef, model := range reg.Models {
		if !runtimeProviderHasHealthURL(model.ProviderRef, runtimes) {
			continue
		}
		result, err := providers.CheckModelHealth(ctx, reg, modelRef, providers.ModelHealthCheckOptions{
			Timeout:                    2 * time.Second,
			LocalRuntimes:              runtimes,
			SkipProviderModelsEndpoint: true,
		})
		if err == nil {
			healthResultsByModel[result.ModelRef] = result
		}
		if err != nil {
			unavailable[modelRef] = true
			healthFailures++
			continue
		}
		if result.Status == providers.ModelHealthUnhealthy {
			unavailable[modelRef] = true
			healthFailures++
		}
	}
	healthResults := providers.ModelHealthUnknownSnapshot(reg, time.Now().UTC())
	for i, result := range healthResults {
		if checked, ok := healthResultsByModel[result.ModelRef]; ok {
			healthResults[i] = checked
		}
	}
	if len(unavailable) == 0 {
		return nil, healthFailures, healthResults
	}
	return unavailable, healthFailures, healthResults
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

func modelHealthStatusesFromResults(results []providers.ModelHealthResult) []missioncontrol.OperatorModelHealthStatus {
	if len(results) == 0 {
		return nil
	}
	statuses := make([]missioncontrol.OperatorModelHealthStatus, 0, len(results))
	for _, result := range results {
		statuses = append(statuses, missioncontrol.OperatorModelHealthStatus{
			ModelRef:          result.ModelRef,
			ProviderRef:       result.ProviderRef,
			Status:            string(result.Status),
			LastCheckedAt:     result.LastCheckedAt,
			LastErrorClass:    string(result.LastErrorClass),
			FallbackAvailable: result.FallbackAvailable,
		})
	}
	return statuses
}
