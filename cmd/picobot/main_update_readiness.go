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

type updateReadinessReport struct {
	Ready  bool              `json:"ready"`
	Checks []diagnosticCheck `json:"checks"`
}

type updateReadinessOptions struct {
	BinaryPath        string
	BackupDir         string
	Session           string
	GatewayCommand    string
	MissionStoreRoot  string
	MissionStatusFile string
}

func newUpdateReadinessCmd() *cobra.Command {
	defaults := updateReadinessOptions{
		BinaryPath:        envOrDefault("PICOBOT_BUILD_OUTPUT", "picobot"),
		BackupDir:         envOrDefault("PICOBOT_BACKUP_DIR", ".termux-frank-backup"),
		Session:           envOrDefault("PICOBOT_SESSION", "frank"),
		GatewayCommand:    envOrDefault("PICOBOT_GATEWAY_CMD", "./picobot gateway"),
		MissionStatusFile: os.Getenv("PICOBOT_MISSION_STATUS_FILE"),
	}

	cmd := &cobra.Command{
		Use:          "update-readiness",
		Short:        "Print read-only transactional update readiness",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			options := updateReadinessOptions{}
			options.BinaryPath, _ = cmd.Flags().GetString("binary")
			options.BackupDir, _ = cmd.Flags().GetString("backup-dir")
			options.Session, _ = cmd.Flags().GetString("session")
			options.GatewayCommand, _ = cmd.Flags().GetString("gateway-cmd")
			options.MissionStoreRoot, _ = cmd.Flags().GetString("mission-store-root")
			options.MissionStatusFile, _ = cmd.Flags().GetString("mission-status-file")

			report := buildUpdateReadinessReport(options)
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode update readiness report: %w", err)
			}
			data = append(data, '\n')
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return fmt.Errorf("failed to write update readiness report: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().String("binary", defaults.BinaryPath, "Current Picobot binary path")
	cmd.Flags().String("backup-dir", defaults.BackupDir, "Directory where the updater preserves rollback files")
	cmd.Flags().String("session", defaults.Session, "tmux session name used by the updater")
	cmd.Flags().String("gateway-cmd", defaults.GatewayCommand, "Gateway command the updater restarts")
	cmd.Flags().String("mission-store-root", "", "Path to the durable mission store root")
	cmd.Flags().String("mission-status-file", defaults.MissionStatusFile, "Mission status file used to derive the store root")
	return cmd
}

func buildUpdateReadinessReport(options updateReadinessOptions) updateReadinessReport {
	checks := []diagnosticCheck{
		binaryDiagnostic(options.BinaryPath),
		backupDirectoryDiagnostic(options.BackupDir),
		updaterEnvDiagnostic(options.Session, options.GatewayCommand),
	}

	cfg, configExists, configLoadErr := loadDiagnosticsConfig(config.DefaultConfigPath())
	checks = append(checks, configPathDiagnostic(config.DefaultConfigPath(), configExists, configLoadErr))
	if configLoadErr == nil && configExists {
		checks = append(checks, providerDiagnostics(cfg)...)
		checks = append(checks, channelDiagnostics(cfg)...)
	}

	storeRoot := missioncontrol.ResolveStoreRoot(options.MissionStoreRoot, options.MissionStatusFile)
	checks = append(checks, updateMissionStoreDiagnostic(storeRoot))

	ready := true
	for _, check := range checks {
		if check.State == "error" {
			ready = false
			break
		}
	}
	return updateReadinessReport{Ready: ready, Checks: checks}
}

func binaryDiagnostic(path string) diagnosticCheck {
	check := diagnosticCheck{Name: "current_binary", Path: path}
	if strings.TrimSpace(path) == "" {
		check.State = "error"
		check.Detail = "binary path is empty"
		return check
	}

	info, err := os.Stat(path)
	if err != nil {
		check.State = "error"
		check.Detail = err.Error()
		return check
	}
	if info.IsDir() {
		check.State = "error"
		check.Detail = "binary path is a directory"
		return check
	}
	if info.Mode().Perm()&0o111 == 0 {
		check.State = "error"
		check.Detail = "binary is not executable by mode"
		return check
	}
	check.State = "ok"
	check.Detail = "binary exists and is executable"
	return check
}

func backupDirectoryDiagnostic(path string) diagnosticCheck {
	check := diagnosticCheck{Name: "backup_dir", Path: path}
	if strings.TrimSpace(path) == "" {
		check.State = "error"
		check.Detail = "backup directory path is empty"
		return check
	}

	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			check.State = "error"
			check.Detail = "backup path exists but is not a directory"
			return check
		}
		if info.Mode().Perm()&0o222 == 0 {
			check.State = "error"
			check.Detail = "backup directory is not writable by mode"
			return check
		}
		check.State = "ok"
		check.Detail = "backup directory exists and has writable mode bits"
		return check
	}
	if !errors.Is(err, os.ErrNotExist) {
		check.State = "error"
		check.Detail = err.Error()
		return check
	}

	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		parent = "."
	}
	parentInfo, parentErr := os.Stat(parent)
	if parentErr != nil {
		check.State = "error"
		check.Detail = fmt.Sprintf("backup directory is absent and parent cannot be inspected: %v", parentErr)
		return check
	}
	if !parentInfo.IsDir() || parentInfo.Mode().Perm()&0o222 == 0 {
		check.State = "error"
		check.Detail = "backup directory is absent and parent is not writable"
		return check
	}
	check.State = "ok"
	check.Detail = "backup directory is absent but parent can create it"
	return check
}

func updaterEnvDiagnostic(session string, gatewayCommand string) diagnosticCheck {
	check := diagnosticCheck{Name: "updater_env"}
	switch {
	case strings.TrimSpace(session) == "":
		check.State = "error"
		check.Detail = "PICOBOT_SESSION/session is empty"
	case strings.TrimSpace(gatewayCommand) == "":
		check.State = "error"
		check.Detail = "PICOBOT_GATEWAY_CMD/gateway command is empty"
	default:
		check.State = "ok"
		check.Detail = fmt.Sprintf("session=%s gateway_cmd_configured=true", strings.TrimSpace(session))
	}
	return check
}

func updateMissionStoreDiagnostic(storeRoot string) diagnosticCheck {
	if strings.TrimSpace(storeRoot) == "" {
		return diagnosticCheck{
			Name:   "mission_store_root",
			State:  "ok",
			Detail: "not configured; updater mission status/assert checks will be skipped",
		}
	}
	return directoryDiagnostic("mission_store_root", storeRoot, false)
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
