package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent/skills"
	"github.com/local/picobot/internal/config"
)

func newSkillsCmd() *cobra.Command {
	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "Import governed skills into the workspace",
	}

	importCmd := &cobra.Command{
		Use:          "import [source...]",
		Short:        "Import Codex-style SKILL.md directories as governed Frank skills",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, _ := cmd.Flags().GetString("workspace")
			resolvedWorkspace, err := resolveSkillsWorkspace(workspace)
			if err != nil {
				return err
			}

			sources := append([]string(nil), args...)
			if len(sources) == 0 {
				cwd, _ := os.Getwd()
				home, _ := os.UserHomeDir()
				sources = skills.AutoDetectImportSources(resolvedWorkspace, cwd, home)
			}

			overwrite, _ := cmd.Flags().GetBool("overwrite")
			report, err := skills.ImportGovernedSkills(skills.ImportOptions{
				WorkspacePath: resolvedWorkspace,
				Sources:       sources,
				Overwrite:     overwrite,
			})
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode skills import report: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write skills import report: %w", err)
			}
			return nil
		},
	}
	importCmd.Flags().String("workspace", "", "Workspace root containing Frank's governed skills/ directory")
	importCmd.Flags().Bool("overwrite", false, "Overwrite an existing governed skill with the same id")

	skillsCmd.AddCommand(importCmd)
	return skillsCmd
}

func resolveSkillsWorkspace(value string) (string, error) {
	workspace := strings.TrimSpace(value)
	if workspace == "" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return "", fmt.Errorf("failed to load config for default workspace: %w", err)
		}
		workspace = strings.TrimSpace(cfg.Agents.Defaults.Workspace)
	}
	if workspace == "" {
		workspace = "~/.picobot/workspace"
	}
	if strings.HasPrefix(workspace, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		workspace = filepath.Join(home, workspace[2:])
	}
	if !filepath.IsAbs(workspace) {
		abs, err := filepath.Abs(workspace)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workspace %q: %w", workspace, err)
		}
		workspace = abs
	}
	return filepath.Clean(workspace), nil
}
