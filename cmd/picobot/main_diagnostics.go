package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

type diagnosticsReport struct {
	Status string            `json:"status"`
	Checks []diagnosticCheck `json:"checks"`
}

type diagnosticCheck struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
	Path   string `json:"path,omitempty"`
}

func newDiagnosticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "diagnostics",
		Short:        "Print read-only first-run diagnostics",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFile, _ := cmd.Flags().GetString("mission-status-file")
			storeRoot, _ := cmd.Flags().GetString("mission-store-root")

			report := buildDiagnosticsReport(config.DefaultConfigPath(), storeRoot, statusFile)
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode diagnostics report: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write diagnostics report: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	cmd.Flags().String("mission-status-file", "", "Path to the mission status snapshot used to derive the store root")
	return cmd
}

func buildDiagnosticsReport(configPath string, missionStoreRoot string, missionStatusFile string) diagnosticsReport {
	checks := make([]diagnosticCheck, 0, 8)

	cfg, configExists, configLoadErr := loadDiagnosticsConfig(configPath)
	checks = append(checks, configPathDiagnostic(configPath, configExists, configLoadErr))
	if configLoadErr == nil {
		checks = append(checks, providerDiagnostics(cfg)...)
		checks = append(checks, channelDiagnostics(cfg)...)
		checks = append(checks, directoryDiagnostic("workspace.dir", cfg.Agents.Defaults.Workspace, false))
	}

	storeRoot := missioncontrol.ResolveStoreRoot(missionStoreRoot, missionStatusFile)
	checks = append(checks, directoryDiagnostic("mission_store_root", storeRoot, true))

	status := "ready"
	for _, check := range checks {
		if check.State != "ok" {
			status = "attention"
			break
		}
	}
	return diagnosticsReport{Status: status, Checks: checks}
}

func loadDiagnosticsConfig(configPath string) (config.Config, bool, error) {
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.Config{}, false, nil
		}
		return config.Config{}, false, err
	}
	cfg, err := config.LoadConfig()
	return cfg, true, err
}

func configPathDiagnostic(path string, exists bool, loadErr error) diagnosticCheck {
	check := diagnosticCheck{Name: "config.path", Path: path}
	switch {
	case loadErr != nil:
		check.State = "error"
		check.Detail = loadErr.Error()
	case !exists:
		check.State = "warn"
		check.Detail = "config file is missing; run picobot onboard before starting services"
	default:
		check.State = "ok"
		check.Detail = "config file exists and decoded"
	}
	return check
}

func providerDiagnostics(cfg config.Config) []diagnosticCheck {
	checks := []diagnosticCheck{
		{Name: "provider.openai.config"},
		{Name: "provider.openai.env"},
	}

	provider := cfg.Providers.OpenAI
	switch {
	case provider == nil:
		checks[0].State = "warn"
		checks[0].Detail = "providers.openai is not configured"
	case strings.TrimSpace(provider.APIKey) == "" || strings.Contains(provider.APIKey, "REPLACE_WITH"):
		checks[0].State = "warn"
		checks[0].Detail = "providers.openai.apiKey is empty or still a placeholder"
	default:
		checks[0].State = "ok"
		checks[0].Detail = "providers.openai.apiKey is configured"
	}

	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
		checks[1].State = "warn"
		checks[1].Detail = "OPENAI_API_KEY is not set; config file value must be valid"
	} else {
		checks[1].State = "ok"
		checks[1].Detail = "OPENAI_API_KEY is set"
	}
	return checks
}

func channelDiagnostics(cfg config.Config) []diagnosticCheck {
	validationErrs := validateConfigForStartup(cfg)
	if len(validationErrs) == 0 {
		return []diagnosticCheck{{
			Name:   "channels.allowlists",
			State:  "ok",
			Detail: "enabled channels have allowlists or explicit open mode",
		}}
	}

	parts := make([]string, 0, len(validationErrs))
	for _, err := range validationErrs {
		parts = append(parts, err.Error())
	}
	return []diagnosticCheck{{
		Name:   "channels.allowlists",
		State:  "error",
		Detail: strings.Join(parts, "; "),
	}}
}

func directoryDiagnostic(name string, path string, optional bool) diagnosticCheck {
	check := diagnosticCheck{Name: name, Path: path}
	if strings.TrimSpace(path) == "" {
		check.State = "warn"
		if optional {
			check.Detail = "path is not configured"
		} else {
			check.Detail = "path is empty"
		}
		return check
	}

	resolved, err := expandDiagnosticsPath(path)
	if err != nil {
		check.State = "error"
		check.Detail = err.Error()
		return check
	}
	check.Path = resolved

	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			check.State = "warn"
			check.Detail = "directory does not exist"
			return check
		}
		check.State = "error"
		check.Detail = err.Error()
		return check
	}
	if !info.IsDir() {
		check.State = "error"
		check.Detail = "path exists but is not a directory"
		return check
	}
	if info.Mode().Perm()&0o222 == 0 {
		check.State = "error"
		check.Detail = "directory is not writable by mode"
		return check
	}
	check.State = "ok"
	check.Detail = "directory exists and has writable mode bits"
	return check
}

func expandDiagnosticsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
