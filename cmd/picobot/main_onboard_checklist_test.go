package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestOnboardChecklistCommandReportsPendingWithoutConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"onboard", "checklist"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("onboard checklist error = %v", err)
	}

	var report onboardingChecklistReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if report.Complete {
		t.Fatalf("report.Complete = true, want false: %#v", report.Steps)
	}
	steps := onboardingStepsByID(report)
	if steps["config"].State != "pending" {
		t.Fatalf("config state = %q, want pending: %#v", steps["config"].State, steps["config"])
	}
}

func TestOnboardChecklistCommandReportsCompleteSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspace := filepath.Join(home, "workspace")
	if err := mkdirAllForDiagnosticsTest(workspace); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Providers.OpenAI.APIKey = "configured-key"
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "telegram-token"
	cfg.Channels.Telegram.AllowFrom = []string{"12345"}
	if err := config.SaveConfig(cfg, filepath.Join(home, ".picobot", "config.json")); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"onboard", "checklist"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("onboard checklist error = %v", err)
	}

	var report onboardingChecklistReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n%s", err, stdout.String())
	}
	if !report.Complete {
		t.Fatalf("report.Complete = false, want true: %#v", report.Steps)
	}
	for _, step := range report.Steps {
		if step.State != "done" {
			t.Fatalf("step %s state = %q, want done: %#v", step.ID, step.State, step)
		}
	}
}

func onboardingStepsByID(report onboardingChecklistReport) map[string]onboardingChecklistStep {
	steps := make(map[string]onboardingChecklistStep, len(report.Steps))
	for _, step := range report.Steps {
		steps[step.ID] = step
	}
	return steps
}
