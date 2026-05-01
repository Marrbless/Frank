package config

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	ProviderTypeOpenAICompatible = "openai_compatible"

	LegacyProviderRef = "openai"
	LegacyModelRef    = "legacy_default"

	RouteReasonCLIOverride         = "cli_override"
	RouteReasonConfigDefault       = "config_default"
	RouteReasonRoutingDefault      = "routing_default"
	RouteReasonLocalPreference     = "local_preference"
	RouteReasonLegacyRawModel      = "legacy_raw_model"
	RouteReasonModelPolicyDefault  = "model_policy_default"
	RouteReasonModelPolicyAllowed  = "model_policy_allowed"
	RouteReasonFallback            = "fallback"
	DefaultModelRoutingPolicyID    = "default"
	DefaultLegacyProviderModel     = "gpt-4o-mini"
	DefaultLegacyStubProviderModel = "stub-model"
)

type ModelAuthorityTier string

const (
	ModelAuthorityLow    ModelAuthorityTier = "low"
	ModelAuthorityMedium ModelAuthorityTier = "medium"
	ModelAuthorityHigh   ModelAuthorityTier = "high"
)

type ModelCostTier string

const (
	ModelCostFree      ModelCostTier = "free"
	ModelCostCheap     ModelCostTier = "cheap"
	ModelCostStandard  ModelCostTier = "standard"
	ModelCostExpensive ModelCostTier = "expensive"
)

type ModelLatencyTier string

const (
	ModelLatencyFast     ModelLatencyTier = "fast"
	ModelLatencyNormal   ModelLatencyTier = "normal"
	ModelLatencySlow     ModelLatencyTier = "slow"
	ModelLatencyVerySlow ModelLatencyTier = "very_slow"
)

type ModelRegistry struct {
	Providers map[string]ProviderConfig
	Models    map[string]ModelProfile
	Aliases   map[string]string
	Routing   ResolvedModelRouting
	Defaults  AgentDefaults
	Warnings  []string
}

type ModelProfile struct {
	Ref           string             `json:"model_ref"`
	ProviderRef   string             `json:"provider_ref"`
	ProviderModel string             `json:"provider_model"`
	DisplayName   string             `json:"display_name,omitempty"`
	Capabilities  ModelCapabilities  `json:"capabilities"`
	Request       ModelRequestConfig `json:"request,omitempty"`
}

type ResolvedModelRouting struct {
	DefaultModel                string
	LocalPreferredModel         string
	Fallbacks                   map[string][]string
	AllowCloudFallbackFromLocal bool
	AllowLowerAuthorityFallback bool
}

type ModelRequiredCapabilities struct {
	SupportsTools        bool
	Local                *bool
	Offline              *bool
	SupportsResponsesAPI *bool
	AuthorityTierAtLeast ModelAuthorityTier
}

type ModelRouteOptions struct {
	ExplicitModel          string
	DefaultModel           string
	AllowedModels          []string
	PreferLocal            bool
	AllowFallback          bool
	RequiredCapability     ModelRequiredCapabilities
	UnavailableModels      map[string]bool
	AllowRawProviderModel  bool
	RawProviderModelSource string
	PolicyID               string
}

type ModelRoute struct {
	SelectedModelRef          string               `json:"selected_model_ref"`
	ProviderRef               string               `json:"provider_ref"`
	ProviderModel             string               `json:"provider_model"`
	SelectionReason           string               `json:"selection_reason"`
	FallbackDepth             int                  `json:"fallback_depth"`
	PolicyID                  string               `json:"policy_id"`
	Capabilities              ModelCapabilities    `json:"capabilities"`
	Request                   ResolvedModelRequest `json:"request"`
	ToolDefinitionsAllowed    bool                 `json:"tool_definitions_allowed"`
	ToolDefinitionsSuppressed bool                 `json:"tool_definitions_suppressed"`
}

type ResolvedModelRequest struct {
	MaxTokens       int      `json:"max_tokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	TimeoutS        int      `json:"timeout_s,omitempty"`
	UseResponses    bool     `json:"use_responses"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
}

func BuildModelRegistry(cfg Config) (ModelRegistry, error) {
	reg := ModelRegistry{
		Providers: make(map[string]ProviderConfig),
		Models:    make(map[string]ModelProfile),
		Aliases:   make(map[string]string),
		Defaults:  cfg.Agents.Defaults,
		Routing: ResolvedModelRouting{
			Fallbacks: make(map[string][]string),
		},
	}

	if err := addProviderConfigs(&reg, cfg.Providers); err != nil {
		return ModelRegistry{}, err
	}

	if len(cfg.Models) == 0 {
		if err := addLegacyModelProfile(&reg, cfg); err != nil {
			return ModelRegistry{}, err
		}
	} else if err := addModelProfiles(&reg, cfg.Models); err != nil {
		return ModelRegistry{}, err
	}

	if err := addModelAliases(&reg, cfg.ModelAliases); err != nil {
		return ModelRegistry{}, err
	}
	if err := resolveModelRouting(&reg, cfg); err != nil {
		return ModelRegistry{}, err
	}
	return reg, nil
}

func NormalizeProviderRef(value string) (string, error) {
	return normalizeControlPlaneRef("provider_ref", value)
}

func NormalizeModelRef(value string) (string, error) {
	return normalizeControlPlaneRef("model_ref", value)
}

func normalizeControlPlaneRef(kind, value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("%s is required", kind)
	}
	for _, r := range normalized {
		if r == '/' || r == '\\' {
			return "", fmt.Errorf("%s %q must not contain path separators", kind, value)
		}
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return "", fmt.Errorf("%s %q contains invalid character %q", kind, value, r)
	}
	return normalized, nil
}

func addProviderConfigs(reg *ModelRegistry, providers ProvidersConfig) error {
	entries := make(map[string]ProviderConfig)
	if providers.OpenAI != nil {
		entries[LegacyProviderRef] = *providers.OpenAI
	}
	for key, provider := range providers.Named {
		entries[key] = provider
	}

	normalizedKeys := make(map[string][]string)
	keys := sortedKeys(entries)
	for _, key := range keys {
		normalized, err := NormalizeProviderRef(key)
		if err != nil {
			return err
		}
		normalizedKeys[normalized] = append(normalizedKeys[normalized], key)
	}
	if err := rejectDuplicateRefs("provider_ref", normalizedKeys); err != nil {
		return err
	}

	for _, key := range keys {
		normalized, _ := NormalizeProviderRef(key)
		provider := entries[key]
		provider.defaultType()
		if provider.Type != ProviderTypeOpenAICompatible {
			return fmt.Errorf("provider_ref %q has unsupported type %q", normalized, provider.Type)
		}
		reg.Providers[normalized] = provider
	}
	return nil
}

func addLegacyModelProfile(reg *ModelRegistry, cfg Config) error {
	provider, ok := reg.Providers[LegacyProviderRef]
	if !ok {
		provider = ProviderConfig{Type: ProviderTypeOpenAICompatible}
		reg.Providers[LegacyProviderRef] = provider
	}

	providerModel := strings.TrimSpace(cfg.Agents.Defaults.Model)
	if providerModel == "" {
		if cfg.Providers.OpenAI != nil && (strings.TrimSpace(cfg.Providers.OpenAI.APIKey) != "" || strings.TrimSpace(cfg.Providers.OpenAI.APIBase) != "") {
			providerModel = DefaultLegacyProviderModel
		} else {
			providerModel = DefaultLegacyStubProviderModel
		}
	}

	temp := cfg.Agents.Defaults.Temperature
	request := ModelRequestConfig{
		MaxTokens:       cfg.Agents.Defaults.MaxTokens,
		Temperature:     &temp,
		TimeoutS:        cfg.Agents.Defaults.RequestTimeoutS,
		UseResponses:    &provider.UseResponses,
		ReasoningEffort: provider.ReasoningEffort,
	}
	reg.Models[LegacyModelRef] = ModelProfile{
		Ref:           LegacyModelRef,
		ProviderRef:   LegacyProviderRef,
		ProviderModel: providerModel,
		DisplayName:   "Legacy default model",
		Capabilities: ModelCapabilities{
			SupportsTools:        true,
			SupportsResponsesAPI: provider.UseResponses,
			AuthorityTier:        ModelAuthorityHigh,
			CostTier:             ModelCostStandard,
			LatencyTier:          ModelLatencyNormal,
		},
		Request: request,
	}
	reg.Routing.DefaultModel = LegacyModelRef
	return nil
}

func addModelProfiles(reg *ModelRegistry, models map[string]ModelProfileConfig) error {
	normalizedKeys := make(map[string][]string)
	keys := sortedKeys(models)
	for _, key := range keys {
		normalized, err := NormalizeModelRef(key)
		if err != nil {
			return err
		}
		normalizedKeys[normalized] = append(normalizedKeys[normalized], key)
	}
	if err := rejectDuplicateRefs("model_ref", normalizedKeys); err != nil {
		return err
	}

	for _, key := range keys {
		normalized, _ := NormalizeModelRef(key)
		model := models[key]
		providerRef, err := NormalizeProviderRef(model.Provider)
		if err != nil {
			return err
		}
		if _, ok := reg.Providers[providerRef]; !ok {
			return fmt.Errorf("model_ref %q references unknown provider_ref %q", normalized, providerRef)
		}
		providerModel := strings.TrimSpace(model.ProviderModel)
		if providerModel == "" {
			return fmt.Errorf("model_ref %q providerModel is required", normalized)
		}
		capabilities, err := normalizeModelCapabilities(model.Capabilities)
		if err != nil {
			return fmt.Errorf("model_ref %q %w", normalized, err)
		}
		reg.Models[normalized] = ModelProfile{
			Ref:           normalized,
			ProviderRef:   providerRef,
			ProviderModel: providerModel,
			DisplayName:   strings.TrimSpace(model.DisplayName),
			Capabilities:  capabilities,
			Request:       model.Request,
		}
	}
	return nil
}

func normalizeModelCapabilities(capabilities ModelCapabilities) (ModelCapabilities, error) {
	if capabilities.Local {
		if capabilities.AuthorityTier == "" {
			capabilities.AuthorityTier = ModelAuthorityLow
		}
		if capabilities.CostTier == "" {
			capabilities.CostTier = ModelCostFree
		}
		if capabilities.LatencyTier == "" {
			capabilities.LatencyTier = ModelLatencySlow
		}
	} else {
		if capabilities.AuthorityTier == "" {
			capabilities.AuthorityTier = ModelAuthorityLow
		}
		if capabilities.CostTier == "" {
			capabilities.CostTier = ModelCostStandard
		}
		if capabilities.LatencyTier == "" {
			capabilities.LatencyTier = ModelLatencyNormal
		}
	}

	if _, ok := modelAuthorityRank(capabilities.AuthorityTier); !ok {
		return ModelCapabilities{}, fmt.Errorf("authorityTier %q is invalid", capabilities.AuthorityTier)
	}
	if !isValidModelCostTier(capabilities.CostTier) {
		return ModelCapabilities{}, fmt.Errorf("costTier %q is invalid", capabilities.CostTier)
	}
	if !isValidModelLatencyTier(capabilities.LatencyTier) {
		return ModelCapabilities{}, fmt.Errorf("latencyTier %q is invalid", capabilities.LatencyTier)
	}
	return capabilities, nil
}

func addModelAliases(reg *ModelRegistry, aliases map[string]string) error {
	if len(aliases) == 0 {
		return nil
	}

	normalizedKeys := make(map[string][]string)
	keys := sortedKeys(aliases)
	for _, key := range keys {
		normalized, err := NormalizeModelRef(key)
		if err != nil {
			return err
		}
		normalizedKeys[normalized] = append(normalizedKeys[normalized], key)
	}
	if err := rejectDuplicateRefs("model_alias", normalizedKeys); err != nil {
		return err
	}

	for _, key := range keys {
		alias, _ := NormalizeModelRef(key)
		target, err := NormalizeModelRef(aliases[key])
		if err != nil {
			return err
		}
		if _, conflicts := reg.Models[alias]; conflicts {
			return fmt.Errorf("alias %q conflicts with model_ref %q", alias, alias)
		}
		if _, isAlias := normalizedKeys[target]; isAlias {
			return fmt.Errorf("alias %q targets alias %q; alias chains are forbidden", alias, target)
		}
		if _, ok := reg.Models[target]; !ok {
			return fmt.Errorf("alias %q targets unknown model_ref %q", alias, target)
		}
		reg.Aliases[alias] = target
	}
	return nil
}

func resolveModelRouting(reg *ModelRegistry, cfg Config) error {
	routing := cfg.ModelRouting
	reg.Routing.AllowCloudFallbackFromLocal = routing.AllowCloudFallbackFromLocal
	reg.Routing.AllowLowerAuthorityFallback = routing.AllowLowerAuthorityFallback

	defaultRef := strings.TrimSpace(routing.DefaultModel)
	reason := RouteReasonRoutingDefault
	if defaultRef == "" && len(cfg.Models) > 0 {
		defaultRef = strings.TrimSpace(cfg.Agents.Defaults.Model)
		reason = RouteReasonConfigDefault
	}
	if defaultRef == "" && reg.Routing.DefaultModel != "" {
		defaultRef = reg.Routing.DefaultModel
	}
	if defaultRef != "" {
		resolved, err := reg.ResolveModelRef(defaultRef)
		if err != nil {
			return fmt.Errorf("%s model %q: %w", reason, defaultRef, err)
		}
		reg.Routing.DefaultModel = resolved
	}
	if reg.Routing.DefaultModel == "" {
		keys := sortedKeys(reg.Models)
		if len(keys) == 0 {
			return fmt.Errorf("model registry requires at least one model")
		}
		reg.Routing.DefaultModel = keys[0]
	}

	if strings.TrimSpace(routing.LocalPreferredModel) != "" {
		resolved, err := reg.ResolveModelRef(routing.LocalPreferredModel)
		if err != nil {
			return fmt.Errorf("localPreferredModel %q: %w", routing.LocalPreferredModel, err)
		}
		reg.Routing.LocalPreferredModel = resolved
	}

	fallbacks := routing.Fallbacks
	if fallbacks == nil {
		fallbacks = map[string][]string{}
	}
	keys := sortedKeys(fallbacks)
	for _, key := range keys {
		from, err := reg.ResolveModelRef(key)
		if err != nil {
			return fmt.Errorf("fallback source %q: %w", key, err)
		}
		targets := make([]string, 0, len(fallbacks[key]))
		for _, targetRaw := range fallbacks[key] {
			target, err := reg.ResolveModelRef(targetRaw)
			if err != nil {
				return fmt.Errorf("fallback target %q for %q: %w", targetRaw, key, err)
			}
			targets = append(targets, target)
		}
		reg.Routing.Fallbacks[from] = targets
	}
	return nil
}

func (r ModelRegistry) ResolveModelRef(value string) (string, error) {
	ref, err := NormalizeModelRef(value)
	if err != nil {
		return "", err
	}
	if _, ok := r.Models[ref]; ok {
		return ref, nil
	}
	if target, ok := r.Aliases[ref]; ok {
		return target, nil
	}
	return "", fmt.Errorf("unknown model_ref or alias %q", ref)
}

func (r ModelRegistry) Route(opts ModelRouteOptions) (ModelRoute, error) {
	allowedOrder, allowedSet, err := r.resolveAllowedRouteModels(opts.AllowedModels)
	if err != nil {
		return ModelRoute{}, err
	}
	selected, reason, rawProviderModel, err := r.selectInitialModel(opts)
	if err != nil {
		return ModelRoute{}, err
	}
	if len(allowedSet) > 0 {
		if _, ok := allowedSet[selected]; !ok {
			if strings.TrimSpace(opts.ExplicitModel) != "" || rawProviderModel != "" {
				return ModelRoute{}, fmt.Errorf("model_ref %q is not allowed by model policy", selected)
			}
			selected = allowedOrder[0]
			reason = RouteReasonModelPolicyAllowed
			rawProviderModel = ""
		}
	}

	model, ok := r.Models[selected]
	if !ok && selected == LegacyModelRef && rawProviderModel != "" {
		provider, providerOK := r.Providers[LegacyProviderRef]
		if !providerOK {
			return ModelRoute{}, fmt.Errorf("legacy raw provider model %q requires providers.openai", rawProviderModel)
		}
		model = r.legacyRawModelProfile(rawProviderModel, provider)
		ok = true
	}
	if !ok {
		return ModelRoute{}, fmt.Errorf("model_ref %q is not configured", selected)
	}
	if rawProviderModel != "" {
		model.ProviderModel = rawProviderModel
	}
	if err := r.validateRouteCapability(model, opts.RequiredCapability); err != nil {
		return ModelRoute{}, err
	}

	if !opts.UnavailableModels[selected] {
		return r.buildRoute(model, reason, 0, opts.PolicyID), nil
	}
	if !opts.AllowFallback {
		return ModelRoute{}, fmt.Errorf("model_ref %q is unavailable and fallback is disabled", selected)
	}

	for _, fallbackRef := range r.Routing.Fallbacks[selected] {
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[fallbackRef]; !ok {
				continue
			}
		}
		fallback := r.Models[fallbackRef]
		if opts.UnavailableModels[fallbackRef] {
			continue
		}
		if model.Capabilities.Local && !fallback.Capabilities.Local && !r.Routing.AllowCloudFallbackFromLocal {
			return ModelRoute{}, fmt.Errorf("cloud fallback from local model_ref %q is not allowed", selected)
		}
		fromRank, _ := modelAuthorityRank(model.Capabilities.AuthorityTier)
		toRank, _ := modelAuthorityRank(fallback.Capabilities.AuthorityTier)
		if toRank < fromRank && !r.Routing.AllowLowerAuthorityFallback {
			return ModelRoute{}, fmt.Errorf("lower-authority fallback from %s to %s is not allowed", model.Capabilities.AuthorityTier, fallback.Capabilities.AuthorityTier)
		}
		if err := r.validateRouteCapability(fallback, opts.RequiredCapability); err != nil {
			continue
		}
		return r.buildRoute(fallback, RouteReasonFallback, 1, opts.PolicyID), nil
	}
	return ModelRoute{}, fmt.Errorf("model_ref %q is unavailable and no valid fallback route exists", selected)
}

func (r ModelRegistry) resolveAllowedRouteModels(values []string) ([]string, map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, nil, nil
	}

	order := make([]string, 0, len(values))
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		ref, err := r.ResolveModelRef(value)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := set[ref]; ok {
			continue
		}
		set[ref] = struct{}{}
		order = append(order, ref)
	}
	return order, set, nil
}

func (r ModelRegistry) selectInitialModel(opts ModelRouteOptions) (string, string, string, error) {
	if strings.TrimSpace(opts.ExplicitModel) != "" {
		selected, err := r.ResolveModelRef(opts.ExplicitModel)
		if err == nil {
			return selected, RouteReasonCLIOverride, "", nil
		}
		if opts.AllowRawProviderModel {
			source := opts.RawProviderModelSource
			if source == "" {
				source = RouteReasonLegacyRawModel
			}
			if _, ok := r.Models[LegacyModelRef]; ok {
				return LegacyModelRef, source, strings.TrimSpace(opts.ExplicitModel), nil
			}
			if _, ok := r.Providers[LegacyProviderRef]; ok {
				return LegacyModelRef, source, strings.TrimSpace(opts.ExplicitModel), nil
			}
		}
		return "", "", "", err
	}

	if opts.PreferLocal && r.Routing.LocalPreferredModel != "" {
		return r.Routing.LocalPreferredModel, RouteReasonLocalPreference, "", nil
	}
	if strings.TrimSpace(opts.DefaultModel) != "" {
		selected, err := r.ResolveModelRef(opts.DefaultModel)
		if err != nil {
			return "", "", "", err
		}
		return selected, RouteReasonModelPolicyDefault, "", nil
	}
	return r.Routing.DefaultModel, RouteReasonRoutingDefault, "", nil
}

func (r ModelRegistry) validateRouteCapability(model ModelProfile, required ModelRequiredCapabilities) error {
	if required.SupportsTools && !model.Capabilities.SupportsTools {
		return fmt.Errorf("model_ref %q does not support tools", model.Ref)
	}
	if required.Local != nil && model.Capabilities.Local != *required.Local {
		return fmt.Errorf("model_ref %q local capability is %t, want %t", model.Ref, model.Capabilities.Local, *required.Local)
	}
	if required.Offline != nil && model.Capabilities.Offline != *required.Offline {
		return fmt.Errorf("model_ref %q offline capability is %t, want %t", model.Ref, model.Capabilities.Offline, *required.Offline)
	}
	if required.SupportsResponsesAPI != nil && model.Capabilities.SupportsResponsesAPI != *required.SupportsResponsesAPI {
		return fmt.Errorf("model_ref %q supportsResponsesAPI capability is %t, want %t", model.Ref, model.Capabilities.SupportsResponsesAPI, *required.SupportsResponsesAPI)
	}
	if required.AuthorityTierAtLeast != "" {
		requiredRank, ok := modelAuthorityRank(required.AuthorityTierAtLeast)
		if !ok {
			return fmt.Errorf("required authority tier %q is invalid", required.AuthorityTierAtLeast)
		}
		modelRank, _ := modelAuthorityRank(model.Capabilities.AuthorityTier)
		if modelRank < requiredRank {
			return fmt.Errorf("model_ref %q authority tier %q is below required %q", model.Ref, model.Capabilities.AuthorityTier, required.AuthorityTierAtLeast)
		}
	}
	return nil
}

func (r ModelRegistry) buildRoute(model ModelProfile, reason string, fallbackDepth int, policyID string) ModelRoute {
	if strings.TrimSpace(policyID) == "" {
		policyID = DefaultModelRoutingPolicyID
	}
	allowed := model.Capabilities.SupportsTools
	return ModelRoute{
		SelectedModelRef:          model.Ref,
		ProviderRef:               model.ProviderRef,
		ProviderModel:             model.ProviderModel,
		SelectionReason:           reason,
		FallbackDepth:             fallbackDepth,
		PolicyID:                  policyID,
		Capabilities:              model.Capabilities,
		Request:                   r.resolveRequest(model),
		ToolDefinitionsAllowed:    allowed,
		ToolDefinitionsSuppressed: !allowed,
	}
}

func (r ModelRegistry) resolveRequest(model ModelProfile) ResolvedModelRequest {
	provider := r.Providers[model.ProviderRef]
	request := model.Request
	out := ResolvedModelRequest{
		MaxTokens:       request.MaxTokens,
		Temperature:     request.Temperature,
		TimeoutS:        request.TimeoutS,
		ReasoningEffort: strings.TrimSpace(request.ReasoningEffort),
	}
	if out.MaxTokens <= 0 {
		out.MaxTokens = r.Defaults.MaxTokens
	}
	if out.Temperature == nil {
		temp := r.Defaults.Temperature
		out.Temperature = &temp
	}
	if out.TimeoutS <= 0 {
		out.TimeoutS = r.Defaults.RequestTimeoutS
	}
	if request.UseResponses != nil {
		out.UseResponses = *request.UseResponses
	} else {
		out.UseResponses = provider.UseResponses
	}
	if out.ReasoningEffort == "" {
		out.ReasoningEffort = strings.TrimSpace(provider.ReasoningEffort)
	}
	return out
}

func (r ModelRegistry) legacyRawModelProfile(providerModel string, provider ProviderConfig) ModelProfile {
	return ModelProfile{
		Ref:           LegacyModelRef,
		ProviderRef:   LegacyProviderRef,
		ProviderModel: strings.TrimSpace(providerModel),
		DisplayName:   "Legacy raw provider model",
		Capabilities: ModelCapabilities{
			SupportsTools:        true,
			SupportsResponsesAPI: provider.UseResponses,
			AuthorityTier:        ModelAuthorityHigh,
			CostTier:             ModelCostStandard,
			LatencyTier:          ModelLatencyNormal,
		},
		Request: ModelRequestConfig{
			MaxTokens:       r.Defaults.MaxTokens,
			Temperature:     &r.Defaults.Temperature,
			TimeoutS:        r.Defaults.RequestTimeoutS,
			UseResponses:    &provider.UseResponses,
			ReasoningEffort: provider.ReasoningEffort,
		},
	}
}

func rejectDuplicateRefs(kind string, normalizedKeys map[string][]string) error {
	refs := sortedKeys(normalizedKeys)
	for _, ref := range refs {
		keys := normalizedKeys[ref]
		if len(keys) <= 1 {
			continue
		}
		sort.Strings(keys)
		return fmt.Errorf("duplicate %s %q from keys %s", kind, ref, quoteList(keys))
	}
	return nil
}

func quoteList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func modelAuthorityRank(tier ModelAuthorityTier) (int, bool) {
	switch tier {
	case ModelAuthorityLow:
		return 1, true
	case ModelAuthorityMedium:
		return 2, true
	case ModelAuthorityHigh:
		return 3, true
	default:
		return 0, false
	}
}

func isValidModelCostTier(tier ModelCostTier) bool {
	switch tier {
	case ModelCostFree, ModelCostCheap, ModelCostStandard, ModelCostExpensive:
		return true
	default:
		return false
	}
}

func isValidModelLatencyTier(tier ModelLatencyTier) bool {
	switch tier {
	case ModelLatencyFast, ModelLatencyNormal, ModelLatencySlow, ModelLatencyVerySlow:
		return true
	default:
		return false
	}
}
