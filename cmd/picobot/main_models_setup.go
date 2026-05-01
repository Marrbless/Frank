package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/modelsetup"
)

type modelsSetupOptions struct {
	DryRun         bool
	NonInteractive bool
	Preset         string
	Approve        bool
	ConfigPath     string
	Force          bool
}

func newModelsSetupCmd() *cobra.Command {
	var opts modelsSetupOptions
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Plan local model setup without unsafe defaults",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsSetupCommand(cmd, opts)
		},
	}
	setupCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Print the setup plan without side effects")
	setupCmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Fail unless every choice is explicit or safely defaulted")
	setupCmd.Flags().StringVar(&opts.Preset, "preset", "", "Setup preset to plan")
	setupCmd.Flags().BoolVar(&opts.Approve, "approve", false, "Execute a fully resolved approved plan in later V6 slices")
	setupCmd.Flags().StringVar(&opts.ConfigPath, "config", "", "Config file path to plan against")
	setupCmd.Flags().BoolVar(&opts.Force, "force", false, "Allow approved overwrites after backup in later V6 slices")
	return setupCmd
}

func newModelsPresetsCmd() *cobra.Command {
	presetsCmd := &cobra.Command{
		Use:   "presets",
		Short: "Inspect V6 model setup presets",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List V6 model setup presets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := modelsetup.ValidateCatalog(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PRESET\tDEFAULT_SAFE\tGATED\tRUNTIME\tPROVIDER_REF\tMODEL_REF")
			for _, preset := range modelsetup.Catalog() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%t\t%t\t%s\t%s\t%s\n",
					preset.Name,
					preset.DefaultSafe,
					preset.ExplicitlyGated,
					preset.RuntimeKind,
					preset.ProviderRef,
					preset.ModelRef,
				)
			}
			return nil
		},
	}
	inspectCmd := &cobra.Command{
		Use:   "inspect <preset>",
		Short: "Inspect one V6 model setup preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			preset, ok := modelsetup.PresetByName(args[0])
			if !ok {
				return fmt.Errorf("unknown preset %q", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "preset: %s\n", preset.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "display_name: %s\n", preset.DisplayName)
			fmt.Fprintf(cmd.OutOrStdout(), "default_safe: %t\n", preset.DefaultSafe)
			fmt.Fprintf(cmd.OutOrStdout(), "explicitly_gated: %t\n", preset.ExplicitlyGated)
			fmt.Fprintf(cmd.OutOrStdout(), "runtime_kind: %s\n", preset.RuntimeKind)
			fmt.Fprintf(cmd.OutOrStdout(), "provider_ref: %s\n", preset.ProviderRef)
			fmt.Fprintf(cmd.OutOrStdout(), "model_ref: %s\n", preset.ModelRef)
			fmt.Fprintf(cmd.OutOrStdout(), "provider_model: %s\n", preset.ProviderModel)
			fmt.Fprintf(cmd.OutOrStdout(), "bind_address: %s\n", preset.BindAddress)
			fmt.Fprintf(cmd.OutOrStdout(), "port: %d\n", preset.Port)
			fmt.Fprintf(cmd.OutOrStdout(), "supports_tools: %t\n", preset.Capabilities.SupportsTools)
			fmt.Fprintf(cmd.OutOrStdout(), "authority_tier: %s\n", preset.Capabilities.AuthorityTier)
			for _, note := range preset.SafetyNotes {
				fmt.Fprintf(cmd.OutOrStdout(), "safety_note: %s\n", note)
			}
			return nil
		},
	}
	presetsCmd.AddCommand(listCmd, inspectCmd)
	return presetsCmd
}

func newModelsLocalCmd() *cobra.Command {
	localCmd := &cobra.Command{
		Use:   "local",
		Short: "Inspect local model setup environment",
	}
	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect local model setup environment without side effects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = config.DefaultConfigPath()
			}
			env := modelsetup.DetectEnvironment(configPath, modelsetup.DefaultDetector())
			printSetupEnvSnapshot(cmd.OutOrStdout(), env)
			return nil
		},
	}
	detectCmd.Flags().String("config", "", "Config file path to inspect")
	localCmd.AddCommand(detectCmd)
	return localCmd
}

func runModelsSetupCommand(cmd *cobra.Command, opts modelsSetupOptions) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	if opts.ConfigPath == "" {
		opts.ConfigPath = config.DefaultConfigPath()
	}
	if opts.Preset == "" {
		if opts.NonInteractive || opts.Approve || opts.DryRun {
			return fmt.Errorf("preset is required in dry-run, approved, or non-interactive setup")
		}
		preset, err := promptForSetupPreset(reader, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		opts.Preset = preset
	}
	env := modelsetup.DetectEnvironment(opts.ConfigPath, modelsetup.DefaultDetector())
	plan, err := modelsetup.BuildPlan(env, modelsetup.OperatorChoices{
		PresetName:     opts.Preset,
		ConfigPath:     opts.ConfigPath,
		NonInteractive: opts.NonInteractive,
		Approve:        opts.Approve,
		DryRun:         opts.DryRun,
		Force:          opts.Force,
	})
	if err != nil {
		return err
	}
	modelsetup.PrintPlan(cmd.OutOrStdout(), plan)
	if opts.DryRun {
		return nil
	}
	if opts.Approve {
		if plan.Status != modelsetup.PlanStatusPlanned {
			return fmt.Errorf("--approve requires a fully resolved planned setup; plan status is %s", plan.Status)
		}
		return fmt.Errorf("models setup executor is not implemented until later V6 slices")
	}
	approved, err := promptForSetupApproval(reader, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	if !approved {
		fmt.Fprintln(cmd.OutOrStdout(), "setup_aborted: true")
		return nil
	}
	if plan.Status != modelsetup.PlanStatusPlanned {
		fmt.Fprintf(cmd.OutOrStdout(), "setup_approved: false\n")
		return fmt.Errorf("approved setup cannot continue because plan status is %s", plan.Status)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "setup_approved: true")
	return fmt.Errorf("models setup executor is not implemented until later V6 slices")
}

func printSetupEnvSnapshot(w io.Writer, env modelsetup.EnvSnapshot) {
	fmt.Fprintf(w, "platform: %s\n", env.Platform)
	fmt.Fprintf(w, "os: %s\n", env.OS)
	fmt.Fprintf(w, "arch: %s\n", env.Arch)
	fmt.Fprintf(w, "config_path: %s\n", env.ConfigPath)
	fmt.Fprintf(w, "termux: %s\n", env.Termux)
	fmt.Fprintf(w, "termux_boot: %s\n", env.TermuxBoot)
	fmt.Fprintf(w, "tmux: %s\n", env.Tmux)
	fmt.Fprintf(w, "ollama: %s\n", env.Ollama)
	fmt.Fprintf(w, "llamacpp: %s\n", env.LlamaCPP)
	printSetupList(w, "existing_providers", env.ExistingProviders)
	printSetupList(w, "existing_models", env.ExistingModels)
	printSetupList(w, "existing_aliases", env.ExistingAliases)
	printSetupList(w, "existing_local_runtimes", env.ExistingLocalRuntimes)
	printSetupList(w, "existing_boot_scripts", env.ExistingBootScripts)
	printSetupList(w, "unsafe_states", env.UnsafeStates)
}

func printSetupList(w io.Writer, label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(w, "%s:\n", label)
	for _, value := range values {
		fmt.Fprintf(w, "- %s\n", value)
	}
}

func promptForSetupPreset(r io.Reader, w io.Writer) (string, error) {
	presets := modelsetup.Catalog()
	fmt.Fprintln(w, "Available model setup presets:")
	for i, preset := range presets {
		gated := ""
		if preset.ExplicitlyGated {
			gated = " gated"
		}
		fmt.Fprintf(w, "%d. %s%s\n", i+1, preset.Name, gated)
	}
	fmt.Fprint(w, "Select preset by number or name: ")
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	choice := strings.TrimSpace(line)
	if choice == "" {
		return "", fmt.Errorf("preset selection is required")
	}
	if n, err := strconv.Atoi(choice); err == nil {
		if n < 1 || n > len(presets) {
			return "", fmt.Errorf("preset selection %d is out of range", n)
		}
		return presets[n-1].Name, nil
	}
	if _, ok := modelsetup.PresetByName(choice); !ok {
		return "", fmt.Errorf("unknown preset %q", choice)
	}
	return choice, nil
}

func promptForSetupApproval(r io.Reader, w io.Writer) (bool, error) {
	fmt.Fprint(w, "Proceed? [y/N] ")
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
