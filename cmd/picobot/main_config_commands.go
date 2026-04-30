package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate local configuration",
	}

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate local configuration without starting services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if warning, err := config.ConfigFilePermissionWarning(config.DefaultConfigPath(), cfg); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not inspect config file permissions: %v\n", err)
			} else if warning != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
			}
			validationErrs := validateConfigForStartup(cfg)
			if len(validationErrs) > 0 {
				for _, validationErr := range validationErrs {
					fmt.Fprintln(cmd.ErrOrStderr(), validationErr)
				}
				return fmt.Errorf("config validation failed with %d error(s)", len(validationErrs))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "config valid")
			return nil
		},
	}

	configCmd.AddCommand(validateCmd)
	return configCmd
}
