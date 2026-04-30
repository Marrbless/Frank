package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestDiagnosticsCommandReportsReadyLocalSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "env-key")

	workspace := filepath.Join(home, "workspace")
	missionStoreRoot := filepath.Join(home, "mission-store")
	if err := mkdirAllForDiagnosticsTest(workspace, missionStoreRoot); err != nil {
		t.Fatalf("mkdir diagnostics dirs: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Providers.OpenAI.APIKey = "configured-key"
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diagnostics", "--mission-store-root", missionStoreRoot})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("diagnostics error = %v", err)
	}

	var report diagnosticsReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if report.Status != "ready" {
		t.Fatalf("report.Status = %q, want ready: %#v", report.Status, report.Checks)
	}
	checks := diagnosticsChecksByName(report)
	for _, name := range []string{"config.path", "provider.openai.config", "provider.openai.env", "channels.allowlists", "workspace.dir", "mission_store_root"} {
		if checks[name].State != "ok" {
			t.Fatalf("%s state = %q, want ok: %#v", name, checks[name].State, checks[name])
		}
	}
}

func TestDiagnosticsCommandReportsAttentionWithoutConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diagnostics"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("diagnostics error = %v", err)
	}

	var report diagnosticsReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if report.Status != "attention" {
		t.Fatalf("report.Status = %q, want attention: %#v", report.Status, report.Checks)
	}
	checks := diagnosticsChecksByName(report)
	if checks["config.path"].State != "warn" {
		t.Fatalf("config.path state = %q, want warn: %#v", checks["config.path"].State, checks["config.path"])
	}
	if checks["provider.openai.config"].State != "warn" {
		t.Fatalf("provider.openai.config state = %q, want warn: %#v", checks["provider.openai.config"].State, checks["provider.openai.config"])
	}
}

func diagnosticsChecksByName(report diagnosticsReport) map[string]diagnosticCheck {
	checks := make(map[string]diagnosticCheck, len(report.Checks))
	for _, check := range report.Checks {
		checks[check.Name] = check
	}
	return checks
}

func mkdirAllForDiagnosticsTest(paths ...string) error {
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}
