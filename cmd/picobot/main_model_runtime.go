package main

import (
	"fmt"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/providers"
)

type runtimeModelSelection struct {
	Registry config.ModelRegistry
	Route    config.ModelRoute
	Provider providers.LLMProvider
}

func resolveRuntimeModelSelection(cfg config.Config, explicitModel string) (runtimeModelSelection, error) {
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		return runtimeModelSelection{}, fmt.Errorf("failed to build model registry: %w", err)
	}
	route, err := reg.Route(config.ModelRouteOptions{
		ExplicitModel:          explicitModel,
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
