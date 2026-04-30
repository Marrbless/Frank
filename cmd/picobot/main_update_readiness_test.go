package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestUpdateReadinessCommandReportsReady(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "env-key")

	binary := filepath.Join(home, "picobot")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	backupDir := filepath.Join(home, "backup")
	missionStoreRoot := filepath.Join(home, "mission-store")
	workspace := filepath.Join(home, "workspace")
	if err := mkdirAllForDiagnosticsTest(backupDir, missionStoreRoot, workspace); err != nil {
		t.Fatalf("mkdir readiness dirs: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Providers.OpenAI.APIKey = "configured-key"
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"update-readiness",
		"--binary", binary,
		"--backup-dir", backupDir,
		"--session", "frank",
		"--gateway-cmd", "./picobot gateway",
		"--mission-store-root", missionStoreRoot,
	})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update-readiness error = %v", err)
	}

	var report updateReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if !report.Ready {
		t.Fatalf("report.Ready = false, want true: %#v", report.Checks)
	}
	checks := diagnosticsChecksByName(diagnosticsReport{Checks: report.Checks})
	for _, name := range []string{"current_binary", "backup_dir", "updater_env", "config.path", "mission_store_root"} {
		if checks[name].State != "ok" {
			t.Fatalf("%s state = %q, want ok: %#v", name, checks[name].State, checks[name])
		}
	}
}

func TestUpdateReadinessCommandReportsMissingBinaryNotReady(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"update-readiness",
		"--binary", filepath.Join(home, "missing-picobot"),
		"--backup-dir", home,
		"--session", "frank",
		"--gateway-cmd", "./picobot gateway",
	})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update-readiness error = %v", err)
	}

	var report updateReadinessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if report.Ready {
		t.Fatalf("report.Ready = true, want false: %#v", report.Checks)
	}
	checks := diagnosticsChecksByName(diagnosticsReport{Checks: report.Checks})
	if checks["current_binary"].State != "error" {
		t.Fatalf("current_binary state = %q, want error: %#v", checks["current_binary"].State, checks["current_binary"])
	}
}
