package modelsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BootScriptWriteOptions struct {
	Force bool
	Now   time.Time
}

type BootScriptWriteResult struct {
	Status     PlanStatus
	Path       string
	BackupPath string
}

func GenerateTermuxModelRuntimeScript(repoDir, sessionName, runtimeCommand string) (string, error) {
	repoDir = strings.TrimSpace(repoDir)
	sessionName = strings.TrimSpace(sessionName)
	runtimeCommand = strings.TrimSpace(runtimeCommand)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	if sessionName == "" {
		sessionName = "frank-model"
	}
	if runtimeCommand == "" {
		return "", fmt.Errorf("runtime command is required")
	}
	if strings.Contains(runtimeCommand, " 0.0.0.0") || strings.Contains(runtimeCommand, "--host 0.0.0.0") {
		return "", fmt.Errorf("runtime command must not bind to 0.0.0.0 by default")
	}
	return fmt.Sprintf(`#!/data/data/com.termux/files/usr/bin/sh
set -eu
cd %s
if tmux has-session -t %s 2>/dev/null; then
  exit 0
fi
tmux new-session -d -s %s %s
`, shellQuote(repoDir), shellQuote(sessionName), shellQuote(sessionName), shellQuote(runtimeCommand)), nil
}

func GenerateTermuxGatewayScript(repoDir, sessionName, gatewayCommand string) (string, error) {
	repoDir = strings.TrimSpace(repoDir)
	sessionName = strings.TrimSpace(sessionName)
	gatewayCommand = strings.TrimSpace(gatewayCommand)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	if sessionName == "" {
		sessionName = "frank"
	}
	if gatewayCommand == "" {
		return "", fmt.Errorf("gateway command is required")
	}
	if strings.Contains(gatewayCommand, "--mission-resume-approved") {
		return "", fmt.Errorf("gateway boot script must not auto-approve mission resume")
	}
	return fmt.Sprintf(`#!/data/data/com.termux/files/usr/bin/sh
set -eu
cd %s
if tmux has-session -t %s 2>/dev/null; then
  exit 0
fi
tmux new-session -d -s %s %s
`, shellQuote(repoDir), shellQuote(sessionName), shellQuote(sessionName), shellQuote(gatewayCommand)), nil
}

func WriteBootScript(path, content string, opts BootScriptWriteOptions) (BootScriptWriteResult, error) {
	if strings.TrimSpace(path) == "" {
		return BootScriptWriteResult{Status: PlanStatusBlocked}, fmt.Errorf("boot script path is required")
	}
	existing, err := os.ReadFile(path)
	if err == nil {
		if string(existing) == content {
			return BootScriptWriteResult{Status: PlanStatusAlreadyPresent, Path: path}, nil
		}
		if !opts.Force {
			return BootScriptWriteResult{Status: PlanStatusBlocked, Path: path}, fmt.Errorf("boot script %q already exists; overwrite requires force", path)
		}
		backupPath, err := writeBootScriptBackup(path, existing, opts.Now)
		if err != nil {
			return BootScriptWriteResult{Status: PlanStatusFailed, Path: path}, err
		}
		if err := writeBootScriptFile(path, content); err != nil {
			return BootScriptWriteResult{Status: PlanStatusFailed, Path: path, BackupPath: backupPath}, err
		}
		return BootScriptWriteResult{Status: PlanStatusChanged, Path: path, BackupPath: backupPath}, nil
	}
	if !os.IsNotExist(err) {
		return BootScriptWriteResult{Status: PlanStatusFailed, Path: path}, err
	}
	if err := writeBootScriptFile(path, content); err != nil {
		return BootScriptWriteResult{Status: PlanStatusFailed, Path: path}, err
	}
	return BootScriptWriteResult{Status: PlanStatusChanged, Path: path}, nil
}

func writeBootScriptFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o700)
}

func writeBootScriptBackup(path string, data []byte, now time.Time) (string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	backupPath := fmt.Sprintf("%s.v6-backup-%s", path, now.UTC().Format("20060102T150405.000000000Z"))
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", err
	}
	return backupPath, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
