package modelsetup

import (
	"fmt"
	"sort"
	"strings"

	"github.com/local/picobot/internal/config"
)

func MinimalUnknownEnvSnapshot(configPath string) EnvSnapshot {
	return EnvSnapshot{
		Platform:   "unknown",
		OS:         "unknown",
		Arch:       "unknown",
		ConfigPath: strings.TrimSpace(configPath),
		Termux:     StateUnknown,
		TermuxBoot: StateUnknown,
		Tmux:       StateUnknown,
		Ollama:     StateUnknown,
		LlamaCPP:   StateUnknown,
	}
}

func BuildPlan(env EnvSnapshot, choices OperatorChoices) (Plan, error) {
	if err := ValidateCatalog(); err != nil {
		return Plan{}, err
	}
	presetName := strings.TrimSpace(choices.PresetName)
	if presetName == "" {
		return Plan{}, fmt.Errorf("preset is required")
	}
	preset, ok := PresetByName(presetName)
	if !ok {
		return Plan{}, fmt.Errorf("unknown preset %q", presetName)
	}
	if strings.TrimSpace(choices.ConfigPath) != "" {
		env.ConfigPath = strings.TrimSpace(choices.ConfigPath)
	}

	providerRef := firstNonEmpty(choices.ProviderRef, preset.ProviderRef)
	modelRef := firstNonEmpty(choices.ModelRef, preset.ModelRef)
	normalizedProvider, err := config.NormalizeProviderRef(providerRef)
	if err != nil {
		return Plan{}, err
	}
	normalizedModel, err := config.NormalizeModelRef(modelRef)
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{
		PresetName:          preset.Name,
		Status:              PlanStatusPlanned,
		Environment:         normalizedEnvSnapshot(env),
		ProviderRef:         normalizedProvider,
		ModelRef:            normalizedModel,
		ProviderModel:       preset.ProviderModel,
		RuntimeKind:         preset.RuntimeKind,
		BindAddress:         firstNonEmpty(choices.BindAddress, preset.BindAddress),
		Port:                firstPositive(choices.Port, preset.Port),
		CloudFallback:       choices.AllowCloudFallback,
		ToolSupport:         preset.Capabilities.SupportsTools,
		AuthorityTier:       preset.Capabilities.AuthorityTier,
		RedactionPolicy:     "redact secrets, prompts, message content, tool arguments, raw request bodies, raw response bodies, and raw provider errors",
		TruncationPolicy:    "drop low-priority successful details first; preserve failed, blocked, rolled_back, and manual_required diagnostics",
		GeneratedReportHint: "safe_to_paste",
	}
	plan.Assumptions = append(plan.Assumptions, "V6-001 planner uses supplied EnvSnapshot; real detector-backed dry-run begins in V6-004")
	if env.ConfigPath == "" {
		plan.Assumptions = append(plan.Assumptions, "config path is unknown")
	}

	addNormalizedCollisionBlocks(&plan, "provider_ref", env.ExistingProviders, config.NormalizeProviderRef)
	addNormalizedCollisionBlocks(&plan, "model_ref", env.ExistingModels, config.NormalizeModelRef)
	addNormalizedCollisionBlocks(&plan, "alias", env.ExistingAliases, config.NormalizeModelRef)
	addNormalizedCollisionBlocks(&plan, "local_runtime_ref", env.ExistingLocalRuntimes, config.NormalizeProviderRef)
	for _, unsafe := range sortedStrings(env.UnsafeStates) {
		plan.BlockedReasons = append(plan.BlockedReasons, fmt.Sprintf("unsafe existing state: %s", unsafe))
	}

	if preset.ExplicitlyGated && !choices.ApproveLANBind {
		plan.BlockedReasons = append(plan.BlockedReasons, fmt.Sprintf("preset %q requires explicit approval for gated LAN exposure", preset.Name))
	}
	if plan.BindAddress == "" && preset.RuntimeKind != RuntimeKindCloud {
		plan.BindAddress = "127.0.0.1"
	}
	if preset.RuntimeKind != RuntimeKindCloud && plan.BindAddress != "127.0.0.1" && !choices.ApproveLANBind {
		plan.BlockedReasons = append(plan.BlockedReasons, fmt.Sprintf("bind address %q requires explicit LAN approval", plan.BindAddress))
	}
	if choices.AllowCloudFallback && !preset.CloudFallbackDefault {
		plan.BlockedReasons = append(plan.BlockedReasons, "cloud fallback requires a later explicit approval path")
	}
	if preset.RuntimeKind != RuntimeKindCloud {
		if preset.Capabilities.SupportsTools {
			plan.BlockedReasons = append(plan.BlockedReasons, "local model cannot support tools by default")
		}
		if preset.Capabilities.AuthorityTier != config.ModelAuthorityLow {
			plan.BlockedReasons = append(plan.BlockedReasons, "local model authority must default to low")
		}
	}

	plan.ConfigPatch = buildConfigPatch(preset, plan)
	plan.Steps = buildPlanSteps(preset, plan, choices)
	if len(plan.BlockedReasons) > 0 {
		plan.Status = PlanStatusBlocked
		markPendingStepsBlocked(&plan)
	} else if hasStepStatus(plan, PlanStatusManualRequired) {
		plan.Status = PlanStatusManualRequired
	}
	return plan, nil
}

func buildConfigPatch(preset Preset, plan Plan) *ConfigPatch {
	provider := config.ProviderConfig{
		Type:    config.ProviderTypeOpenAICompatible,
		APIBase: preset.BaseURL,
	}
	if preset.RuntimeKind == RuntimeKindCloud {
		provider.APIKey = "REPLACE_WITH_REAL_PROVIDER_API_KEY"
	} else if preset.RuntimeKind == RuntimeKindOllama {
		provider.APIKey = "ollama"
	} else {
		provider.APIKey = "not-needed"
	}
	aliasRefs := map[string]string{}
	if preset.RuntimeKind == RuntimeKindCloud {
		aliasRefs["best"] = plan.ModelRef
	} else {
		aliasRefs["local"] = plan.ModelRef
		aliasRefs["phone"] = plan.ModelRef
	}
	routing := config.ModelRoutingConfig{
		DefaultModel:                plan.ModelRef,
		LocalPreferredModel:         plan.ModelRef,
		Fallbacks:                   map[string][]string{plan.ModelRef: {}},
		AllowCloudFallbackFromLocal: false,
		AllowLowerAuthorityFallback: false,
	}
	if preset.RuntimeKind == RuntimeKindCloud {
		routing.LocalPreferredModel = ""
	}
	runtime := config.LocalRuntimeConfig{}
	runtimeRef := ""
	if preset.RuntimeKind != RuntimeKindCloud {
		runtimeRef = plan.ProviderRef
		runtime = config.LocalRuntimeConfig{
			Kind:            "external_http",
			Provider:        plan.ProviderRef,
			ExpectedBaseURL: preset.BaseURL,
			StartCommand:    "",
			HealthURL:       preset.HealthURL,
			Notes:           "Configured by Frank V6 setup wizard.",
		}
	}
	return &ConfigPatch{
		ProviderRef:     plan.ProviderRef,
		ModelRef:        plan.ModelRef,
		AliasRefs:       aliasRefs,
		RuntimeRef:      runtimeRef,
		DefaultModelRef: plan.ModelRef,
		ProviderConfig:  provider,
		ModelConfig: config.ModelProfileConfig{
			Provider:      plan.ProviderRef,
			ProviderModel: preset.ProviderModel,
			DisplayName:   preset.DisplayName,
			Capabilities:  preset.Capabilities,
			Request:       preset.Request,
		},
		RoutingConfig: routing,
		RuntimeConfig: runtime,
	}
}

func buildPlanSteps(preset Preset, plan Plan, choices OperatorChoices) []PlanStep {
	var steps []PlanStep
	configPath := plan.Environment.ConfigPath
	steps = append(steps, PlanStep{
		ID:                       "validate-v5-config-patch",
		Summary:                  "Validate generated V5 model setup patch in memory",
		SideEffect:               SideEffectNone,
		FilesToRead:              nil,
		ApprovalRequired:         false,
		IdempotencyKey:           "validate:" + plan.PresetName + ":" + plan.ModelRef,
		AlreadyPresentRule:       "generated config must pass V5 registry validation",
		RollbackCleanup:          "none",
		RedactionPolicy:          plan.RedactionPolicy,
		Status:                   PlanStatusPlanned,
		DiagnosticsPriority:      10,
		PreserveWhenTruncating:   true,
		SafeToOmitWhenTruncating: false,
	})
	if preset.RuntimeKind != RuntimeKindCloud {
		addRuntimeSetupSteps(&steps, preset, plan, choices)
	}
	steps = append(steps, PlanStep{
		ID:                         "write-v5-config",
		Summary:                    "Write V5 provider, model, alias, routing, and local runtime config after backup",
		SideEffect:                 SideEffectWriteConfig,
		FilesToRead:                []string{configPath},
		FilesToWrite:               []string{configPath},
		ApprovalRequired:           true,
		ApprovalReason:             "config writes require explicit approval and backup",
		IdempotencyKey:             "config:" + configPath + ":" + plan.ProviderRef + ":" + plan.ModelRef,
		AlreadyPresentRule:         "provider, model, aliases, routing, and local runtime match generated safe state",
		RollbackCleanup:            "restore backup on post-write validation failure",
		RedactionPolicy:            plan.RedactionPolicy,
		Dependencies:               []string{"validate-v5-config-patch"},
		Status:                     PlanStatusPlanned,
		DiagnosticsPriority:        20,
		PreserveWhenTruncating:     true,
		SafeToOmitWhenTruncating:   false,
		RequiresExplicitLANApprove: plan.BindAddress != "127.0.0.1",
	})
	steps = append(steps, PlanStep{
		ID:                       "readiness-check",
		Summary:                  "Run metadata-only no-prompt readiness check",
		SideEffect:               SideEffectHealthCheck,
		RuntimePort:              plan.Port,
		RuntimeBindAddress:       plan.BindAddress,
		ApprovalRequired:         false,
		IdempotencyKey:           "readiness:" + plan.ProviderRef + ":" + plan.ModelRef,
		AlreadyPresentRule:       "metadata endpoint or local process state matches planned provider",
		RollbackCleanup:          "none",
		RedactionPolicy:          plan.RedactionPolicy,
		Dependencies:             []string{"write-v5-config"},
		Status:                   PlanStatusPlanned,
		DiagnosticsPriority:      30,
		PreserveWhenTruncating:   true,
		SafeToOmitWhenTruncating: false,
	})
	steps = append(steps, PlanStep{
		ID:                       "route-check",
		Summary:                  "Run V5 route check without provider prompt",
		SideEffect:               SideEffectRouteCheck,
		Command:                  []string{"picobot", "models", "route", "--model", plan.ModelRef},
		ApprovalRequired:         false,
		IdempotencyKey:           "route:" + plan.ModelRef,
		AlreadyPresentRule:       "V5 registry resolves selected model ref to planned provider model",
		RollbackCleanup:          "none",
		RedactionPolicy:          plan.RedactionPolicy,
		Dependencies:             []string{"write-v5-config"},
		Status:                   PlanStatusPlanned,
		DiagnosticsPriority:      40,
		PreserveWhenTruncating:   false,
		SafeToOmitWhenTruncating: true,
	})
	return steps
}

func addRuntimeSetupSteps(steps *[]PlanStep, preset Preset, plan Plan, choices OperatorChoices) {
	switch preset.RuntimeKind {
	case RuntimeKindOllama:
		status := PlanStatusManualRequired
		manual := []string{"Install or make Ollama available through a reviewed package path or checked-in manifest before automatic execution."}
		if choices.InstallBehavior == "skip" {
			status = PlanStatusSkipped
			manual = nil
		}
		*steps = append(*steps, PlanStep{
			ID:                     "install-ollama",
			Summary:                "Install or verify Ollama runtime",
			SideEffect:             SideEffectInstallRuntime,
			Command:                []string{"ollama", "--version"},
			RuntimePort:            plan.Port,
			RuntimeBindAddress:     plan.BindAddress,
			ApprovalRequired:       true,
			ApprovalReason:         "runtime installation/start requires explicit approval",
			IdempotencyKey:         "runtime:ollama:" + plan.BindAddress,
			AlreadyPresentRule:     "ollama is present and serves only the planned localhost endpoint",
			RollbackCleanup:        "stop any setup-started runtime session if later required step fails",
			RedactionPolicy:        plan.RedactionPolicy,
			Status:                 status,
			ManualInstructions:     manual,
			DiagnosticsPriority:    15,
			PreserveWhenTruncating: status == PlanStatusManualRequired,
			RequiresManifest:       true,
		})
		*steps = append(*steps, PlanStep{
			ID:                     "pull-ollama-model",
			Summary:                "Pull configured Ollama model after approval",
			SideEffect:             SideEffectPullModel,
			Command:                []string{"ollama", "pull", preset.ProviderModel},
			ExpectedDiskImpact:     preset.ExpectedDiskImpact,
			ApprovalRequired:       true,
			ApprovalReason:         "model pull downloads model data",
			IdempotencyKey:         "ollama-model:" + preset.ProviderModel,
			AlreadyPresentRule:     "ollama model list includes the planned provider model",
			RollbackCleanup:        "leave existing models in place; do not delete user-owned model data automatically",
			RedactionPolicy:        plan.RedactionPolicy,
			Dependencies:           []string{"install-ollama"},
			Status:                 status,
			ManualInstructions:     manualInstructionsIf(status == PlanStatusManualRequired, "Run the approved Ollama pull manually or provide a checked-in manifest/package path."),
			DiagnosticsPriority:    16,
			PreserveWhenTruncating: status == PlanStatusManualRequired,
		})
		*steps = append(*steps, PlanStep{
			ID:                     "start-ollama-runtime",
			Summary:                "Start Ollama runtime on the planned local endpoint",
			SideEffect:             SideEffectStartRuntime,
			Command:                []string{"ollama", "serve"},
			RuntimePort:            plan.Port,
			RuntimeBindAddress:     plan.BindAddress,
			ApprovalRequired:       true,
			ApprovalReason:         "starting runtimes changes local process state",
			IdempotencyKey:         "ollama-start:" + plan.BindAddress,
			AlreadyPresentRule:     "ollama is already serving the planned localhost endpoint",
			RollbackCleanup:        "stop setup-started runtime session if later required step fails",
			RedactionPolicy:        plan.RedactionPolicy,
			Dependencies:           []string{"install-ollama", "pull-ollama-model"},
			Status:                 status,
			ManualInstructions:     manualInstructionsIf(status == PlanStatusManualRequired, "Start Ollama manually after an approved install path is available."),
			DiagnosticsPriority:    17,
			PreserveWhenTruncating: status == PlanStatusManualRequired,
		})
	case RuntimeKindLlamaCPP:
		status := PlanStatusManualRequired
		if choices.RegisterExistingBehavior == "provided" {
			status = PlanStatusPlanned
		}
		*steps = append(*steps, PlanStep{
			ID:                     "register-llamacpp-existing",
			Summary:                "Register existing llama-server binary and GGUF model",
			SideEffect:             SideEffectNone,
			RuntimePort:            plan.Port,
			RuntimeBindAddress:     plan.BindAddress,
			ApprovalRequired:       false,
			IdempotencyKey:         "llamacpp-register:" + plan.ProviderRef + ":" + plan.ModelRef,
			AlreadyPresentRule:     "operator-provided binary and model paths exist and command binds to localhost",
			RollbackCleanup:        "none",
			RedactionPolicy:        plan.RedactionPolicy,
			Status:                 status,
			ManualInstructions:     manualInstructionsIf(status == PlanStatusManualRequired, "Provide existing llama-server and GGUF model paths; automatic downloads are blocked until manifests exist."),
			DiagnosticsPriority:    15,
			PreserveWhenTruncating: status == PlanStatusManualRequired,
		})
	}
	if preset.BootSupported && choices.BootScripts {
		*steps = append(*steps, PlanStep{
			ID:                     "write-termux-boot-script",
			Summary:                "Write idempotent Termux:Boot model runtime script",
			SideEffect:             SideEffectWriteBootScript,
			FilesToWrite:           []string{"~/.termux/boot/frank-model-runtime"},
			RuntimePort:            plan.Port,
			RuntimeBindAddress:     plan.BindAddress,
			ApprovalRequired:       true,
			ApprovalReason:         "boot script changes startup behavior",
			IdempotencyKey:         "termux-boot:" + plan.ProviderRef,
			AlreadyPresentRule:     "existing boot script exactly matches generated safe script",
			RollbackCleanup:        "restore previous boot script backup if validation fails",
			RedactionPolicy:        plan.RedactionPolicy,
			Status:                 PlanStatusPlanned,
			DiagnosticsPriority:    25,
			PreserveWhenTruncating: true,
		})
	}
}

func markPendingStepsBlocked(plan *Plan) {
	for i := range plan.Steps {
		if plan.Steps[i].Status == PlanStatusPlanned {
			plan.Steps[i].Status = PlanStatusBlocked
		}
		plan.Steps[i].PreserveWhenTruncating = true
		plan.Steps[i].SafeToOmitWhenTruncating = false
	}
}

func hasStepStatus(plan Plan, status PlanStatus) bool {
	for _, step := range plan.Steps {
		if step.Status == status {
			return true
		}
	}
	return false
}

func addNormalizedCollisionBlocks(plan *Plan, kind string, refs []string, normalize func(string) (string, error)) {
	normalized := make(map[string][]string)
	for _, ref := range refs {
		value, err := normalize(ref)
		if err != nil {
			plan.BlockedReasons = append(plan.BlockedReasons, fmt.Sprintf("%s %q is invalid after normalization: %v", kind, ref, err))
			continue
		}
		normalized[value] = append(normalized[value], ref)
	}
	keys := sortedStringKeys(normalized)
	for _, key := range keys {
		raws := uniqueSorted(normalized[key])
		if len(raws) > 1 {
			plan.BlockedReasons = append(plan.BlockedReasons, fmt.Sprintf("%s collision after normalization: %q from %s", kind, key, strings.Join(raws, ", ")))
		}
	}
}

func normalizedEnvSnapshot(env EnvSnapshot) EnvSnapshot {
	env.ExistingProviders = sortedStrings(env.ExistingProviders)
	env.ExistingModels = sortedStrings(env.ExistingModels)
	env.ExistingAliases = sortedStrings(env.ExistingAliases)
	env.ExistingLocalRuntimes = sortedStrings(env.ExistingLocalRuntimes)
	env.ExistingBootScripts = sortedStrings(env.ExistingBootScripts)
	env.UnsafeStates = sortedStrings(env.UnsafeStates)
	return env
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func manualInstructionsIf(include bool, text string) []string {
	if !include {
		return nil
	}
	return []string{text}
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
