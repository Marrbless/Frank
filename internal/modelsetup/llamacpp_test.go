package modelsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLlamaCPPRegisterExistingValidatesPathsAndLocalhost(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "llama-server")
	model := filepath.Join(dir, "qwen.gguf")
	if err := os.WriteFile(server, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(server) error = %v", err)
	}
	if err := os.WriteFile(model, []byte("model"), 0o600); err != nil {
		t.Fatalf("WriteFile(model) error = %v", err)
	}
	reg, err := BuildLlamaCPPRegistration(server, model, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("BuildLlamaCPPRegistration() error = %v", err)
	}
	if err := ValidateLlamaCPPRegistration(reg); err != nil {
		t.Fatalf("ValidateLlamaCPPRegistration() error = %v", err)
	}
	if !strings.Contains(reg.Command, "--host 127.0.0.1") || !strings.Contains(reg.Command, "--port 8080") {
		t.Fatalf("command = %q, want localhost port", reg.Command)
	}
}

func TestLlamaCPPRegisterExistingRejectsLANBindAndMissingPaths(t *testing.T) {
	if _, err := BuildLlamaCPPRegistration("/tmp/server", "/tmp/model.gguf", "0.0.0.0", 8080); err == nil {
		t.Fatal("BuildLlamaCPPRegistration() error = nil, want LAN bind rejection")
	}
	reg, err := BuildLlamaCPPRegistration("/tmp/missing-server", "/tmp/missing-model.gguf", "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("BuildLlamaCPPRegistration() error = %v", err)
	}
	if err := ValidateLlamaCPPRegistration(reg); err == nil {
		t.Fatal("ValidateLlamaCPPRegistration() error = nil, want missing path")
	}
}

func TestPlannerLlamaCPPRegisterExistingProducesStartCommand(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "llama-server")
	model := filepath.Join(dir, "qwen.gguf")
	if err := os.WriteFile(server, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(server) error = %v", err)
	}
	if err := os.WriteFile(model, []byte("model"), 0o600); err != nil {
		t.Fatalf("WriteFile(model) error = %v", err)
	}
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot(filepath.Join(dir, "config.json")), OperatorChoices{
		PresetName:               PresetPhoneLlamaCPPTiny,
		ConfigPath:               filepath.Join(dir, "config.json"),
		DryRun:                   true,
		RegisterExistingBehavior: "provided",
		LlamaCPPServerPath:       server,
		GGUFModelPath:            model,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Status != PlanStatusPlanned {
		t.Fatalf("plan status = %q, want planned", plan.Status)
	}
	if plan.ConfigPatch == nil || !strings.Contains(plan.ConfigPatch.RuntimeConfig.StartCommand, "--host 127.0.0.1") {
		t.Fatalf("runtime start command = %q, want localhost command", plan.ConfigPatch.RuntimeConfig.StartCommand)
	}
	var registerStep PlanStep
	for _, step := range plan.Steps {
		if step.ID == "register-llamacpp-existing" {
			registerStep = step
		}
	}
	if registerStep.Status != PlanStatusPlanned {
		t.Fatalf("register step = %#v, want planned", registerStep)
	}
	if len(registerStep.FilesToRead) != 2 {
		t.Fatalf("FilesToRead = %#v, want server and model paths", registerStep.FilesToRead)
	}
}

func TestLlamaCPPManifestCommandsLocateAfterUnpackAndStayLocal(t *testing.T) {
	locate := BuildLlamaCPPLocateCommand(LlamaCPPRuntimeInstallDir)
	for _, want := range []string{"find", "llama-server", "llama-cli", LlamaCPPRuntimeInstallDir} {
		if !strings.Contains(locate, want) {
			t.Fatalf("locate command = %q, want %q", locate, want)
		}
	}
	start := BuildLlamaCPPManifestStartCommand(LlamaCPPRuntimeInstallDir, Qwen25TinyGGUFPath, "127.0.0.1", 8080)
	for _, want := range []string{"find", "llama-server", Qwen25TinyGGUFPath, "--host \"127.0.0.1\"", "--port 8080", "nohup"} {
		if !strings.Contains(start, want) {
			t.Fatalf("start command = %q, want %q", start, want)
		}
	}
	for _, forbidden := range []string{"--mission-resume-approved", "0.0.0.0"} {
		if strings.Contains(start, forbidden) || strings.Contains(locate, forbidden) {
			t.Fatalf("commands contain forbidden %q\nlocate=%s\nstart=%s", forbidden, locate, start)
		}
	}
}
