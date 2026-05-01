package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
)

func newModelsCmd() *cobra.Command {
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "Inspect model control-plane routing",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured model profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadModelRegistryForCommand()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "MODEL_REF\tPROVIDER_REF\tPROVIDER_MODEL\tAUTHORITY\tLOCAL\tTOOLS")
			for _, ref := range modelCommandSortedKeys(reg.Models) {
				model := reg.Models[ref]
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%t\t%t\n",
					model.Ref,
					model.ProviderRef,
					model.ProviderModel,
					model.Capabilities.AuthorityTier,
					model.Capabilities.Local,
					model.Capabilities.SupportsTools,
				)
			}
			return nil
		},
	}

	inspectCmd := &cobra.Command{
		Use:   "inspect <model_ref_or_alias>",
		Short: "Inspect one model profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadModelRegistryForCommand()
			if err != nil {
				return err
			}
			ref, err := reg.ResolveModelRef(args[0])
			if err != nil {
				return err
			}
			model := reg.Models[ref]
			route, err := reg.Route(config.ModelRouteOptions{ExplicitModel: model.Ref})
			if err != nil {
				return err
			}
			request := route.Request
			fmt.Fprintf(cmd.OutOrStdout(), "model_ref: %s\n", model.Ref)
			fmt.Fprintf(cmd.OutOrStdout(), "provider_ref: %s\n", model.ProviderRef)
			fmt.Fprintf(cmd.OutOrStdout(), "provider_model: %s\n", model.ProviderModel)
			if model.DisplayName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "display_name: %s\n", model.DisplayName)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "local: %t\n", model.Capabilities.Local)
			fmt.Fprintf(cmd.OutOrStdout(), "offline: %t\n", model.Capabilities.Offline)
			fmt.Fprintf(cmd.OutOrStdout(), "supports_tools: %t\n", model.Capabilities.SupportsTools)
			fmt.Fprintf(cmd.OutOrStdout(), "supports_responses_api: %t\n", model.Capabilities.SupportsResponsesAPI)
			fmt.Fprintf(cmd.OutOrStdout(), "authority_tier: %s\n", model.Capabilities.AuthorityTier)
			fmt.Fprintf(cmd.OutOrStdout(), "cost_tier: %s\n", model.Capabilities.CostTier)
			fmt.Fprintf(cmd.OutOrStdout(), "latency_tier: %s\n", model.Capabilities.LatencyTier)
			fmt.Fprintf(cmd.OutOrStdout(), "context_tokens: %d\n", model.Capabilities.ContextTokens)
			fmt.Fprintf(cmd.OutOrStdout(), "max_output_tokens: %d\n", model.Capabilities.MaxOutputTokens)
			fmt.Fprintf(cmd.OutOrStdout(), "request_max_tokens: %d\n", request.MaxTokens)
			if request.Temperature != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "request_temperature: %g\n", *request.Temperature)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "request_timeout_s: %d\n", request.TimeoutS)
			fmt.Fprintf(cmd.OutOrStdout(), "request_use_responses: %t\n", request.UseResponses)
			if request.ReasoningEffort != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "request_reasoning_effort: %s\n", request.ReasoningEffort)
			}
			return nil
		},
	}

	routeCmd := &cobra.Command{
		Use:   "route",
		Short: "Resolve the configured model route",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadModelRegistryForCommand()
			if err != nil {
				return err
			}
			modelFlag, _ := cmd.Flags().GetString("model")
			preferLocal, _ := cmd.Flags().GetBool("local")
			requiresTools, _ := cmd.Flags().GetBool("requires-tools")
			allowFallback, _ := cmd.Flags().GetBool("allow-fallback")
			route, err := reg.Route(config.ModelRouteOptions{
				ExplicitModel: modelFlag,
				PreferLocal:   preferLocal,
				AllowFallback: allowFallback,
				RequiredCapability: config.ModelRequiredCapabilities{
					SupportsTools: requiresTools,
				},
				AllowRawProviderModel:  true,
				RawProviderModelSource: config.RouteReasonCLIOverride,
			})
			if err != nil {
				return err
			}
			printModelRoute(cmd, route)
			return nil
		},
	}
	routeCmd.Flags().StringP("model", "M", "", "Model ref, alias, or legacy provider model to route")
	routeCmd.Flags().Bool("local", false, "Prefer the configured local model")
	routeCmd.Flags().Bool("requires-tools", false, "Require a model that supports tools")
	routeCmd.Flags().Bool("allow-fallback", false, "Allow configured fallback routes")

	modelsCmd.AddCommand(listCmd, inspectCmd, routeCmd)
	return modelsCmd
}

func loadModelRegistryForCommand() (config.ModelRegistry, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return config.ModelRegistry{}, fmt.Errorf("failed to load config: %w", err)
	}
	reg, err := config.BuildModelRegistry(cfg)
	if err != nil {
		return config.ModelRegistry{}, err
	}
	return reg, nil
}

func printModelRoute(cmd *cobra.Command, route config.ModelRoute) {
	fmt.Fprintf(cmd.OutOrStdout(), "selected_model_ref: %s\n", route.SelectedModelRef)
	fmt.Fprintf(cmd.OutOrStdout(), "provider_ref: %s\n", route.ProviderRef)
	fmt.Fprintf(cmd.OutOrStdout(), "provider_model: %s\n", route.ProviderModel)
	fmt.Fprintf(cmd.OutOrStdout(), "selection_reason: %s\n", route.SelectionReason)
	fmt.Fprintf(cmd.OutOrStdout(), "fallback_depth: %d\n", route.FallbackDepth)
	fmt.Fprintf(cmd.OutOrStdout(), "policy_id: %s\n", route.PolicyID)
	fmt.Fprintf(cmd.OutOrStdout(), "local: %t\n", route.Capabilities.Local)
	fmt.Fprintf(cmd.OutOrStdout(), "offline: %t\n", route.Capabilities.Offline)
	fmt.Fprintf(cmd.OutOrStdout(), "supports_tools: %t\n", route.Capabilities.SupportsTools)
	fmt.Fprintf(cmd.OutOrStdout(), "authority_tier: %s\n", route.Capabilities.AuthorityTier)
	fmt.Fprintf(cmd.OutOrStdout(), "tool_definitions_allowed: %t\n", route.ToolDefinitionsAllowed)
	fmt.Fprintf(cmd.OutOrStdout(), "tool_definitions_suppressed: %t\n", route.ToolDefinitionsSuppressed)
	fmt.Fprintf(cmd.OutOrStdout(), "request_max_tokens: %d\n", route.Request.MaxTokens)
	if route.Request.Temperature != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "request_temperature: %g\n", *route.Request.Temperature)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "request_timeout_s: %d\n", route.Request.TimeoutS)
	fmt.Fprintf(cmd.OutOrStdout(), "request_use_responses: %t\n", route.Request.UseResponses)
	if route.Request.ReasoningEffort != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "request_reasoning_effort: %s\n", route.Request.ReasoningEffort)
	}
}

func modelCommandSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
