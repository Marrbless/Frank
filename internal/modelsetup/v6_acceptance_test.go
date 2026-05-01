package modelsetup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestV6FakeSetupAcceptanceScenarios(t *testing.T) {
	t.Run("phone_ollama_manual_required_without_manifest", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{PresetName: PresetPhoneOllamaTiny, ConfigPath: cfgPath})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if plan.Status != PlanStatusManualRequired {
			t.Fatalf("status = %q, want manual_required", plan.Status)
		}
		if fileExists(cfgPath) {
			t.Fatal("config exists after planning manual-required Ollama path")
		}
	})

	t.Run("phone_llamacpp_register_existing", func(t *testing.T) {
		dir := t.TempDir()
		server, model := writeFakeLlamaAssets(t, dir)
		cfgPath := filepath.Join(dir, "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{
			PresetName:               PresetPhoneLlamaCPPTiny,
			ConfigPath:               cfgPath,
			RegisterExistingBehavior: "provided",
			LlamaCPPServerPath:       server,
			GGUFModelPath:            model,
		})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		result, err := ExecutePlan(context.Background(), plan, nil, ExecutorOptions{Approved: true})
		if err != nil {
			t.Fatalf("ExecutePlan() error = %v", err)
		}
		if result.Config.Status != PlanStatusChanged {
			t.Fatalf("config status = %q, want changed", result.Config.Status)
		}
		cfg := loadAcceptanceConfig(t, cfgPath)
		if cfg.LocalRuntimes["llamacpp_phone"].StartCommand == "" {
			t.Fatalf("llama.cpp runtime start command is empty")
		}
	})

	t.Run("desktop_ollama_config_only", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{
			PresetName:      PresetDesktopOllamaLocal,
			ConfigPath:      cfgPath,
			InstallBehavior: "skip",
		})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if _, err := ExecutePlan(context.Background(), plan, nil, ExecutorOptions{Approved: true}); err != nil {
			t.Fatalf("ExecutePlan() error = %v", err)
		}
		cfg := loadAcceptanceConfig(t, cfgPath)
		if cfg.Models["local_fast"].Capabilities.SupportsTools {
			t.Fatal("desktop Ollama local model supports tools, want false")
		}
	})

	t.Run("cloud_stubs", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{PresetName: PresetCloudOpenRouter, ConfigPath: cfgPath})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if _, err := ExecutePlan(context.Background(), plan, nil, ExecutorOptions{Approved: true}); err != nil {
			t.Fatalf("ExecutePlan() error = %v", err)
		}
		cfg := loadAcceptanceConfig(t, cfgPath)
		if cfg.Providers.Named["openrouter"].APIKey != "REPLACE_WITH_REAL_PROVIDER_API_KEY" {
			t.Fatalf("openrouter key = %q, want placeholder stub", cfg.Providers.Named["openrouter"].APIKey)
		}
	})

	t.Run("mixed_local_cloud_safe", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{
			PresetName:      PresetMixedLocalCloudSafe,
			ConfigPath:      cfgPath,
			InstallBehavior: "skip",
		})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if _, err := ExecutePlan(context.Background(), plan, nil, ExecutorOptions{Approved: true}); err != nil {
			t.Fatalf("ExecutePlan() error = %v", err)
		}
		cfg := loadAcceptanceConfig(t, cfgPath)
		if _, ok := cfg.Models["local_fast"]; !ok {
			t.Fatal("mixed setup missing local_fast")
		}
		if _, ok := cfg.Models["cloud_reasoning"]; !ok {
			t.Fatal("mixed setup missing cloud_reasoning")
		}
		if cfg.ModelRouting.AllowCloudFallbackFromLocal {
			t.Fatal("mixed setup enables cloud fallback, want disabled")
		}
		if len(cfg.ModelRouting.Fallbacks["local_fast"]) != 0 {
			t.Fatalf("local fallback = %#v, want none", cfg.ModelRouting.Fallbacks["local_fast"])
		}
	})

	t.Run("rerun_already_present", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{
			PresetName:      PresetDesktopOllamaLocal,
			ConfigPath:      cfgPath,
			InstallBehavior: "skip",
		})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if _, err := ExecutePlan(context.Background(), plan, nil, ExecutorOptions{Approved: true}); err != nil {
			t.Fatalf("ExecutePlan(first) error = %v", err)
		}
		second, err := ApplyConfigPlan(plan, ConfigWriteOptions{})
		if err != nil {
			t.Fatalf("ApplyConfigPlan(second) error = %v", err)
		}
		if second.Status != PlanStatusAlreadyPresent {
			t.Fatalf("second status = %q, want already_present", second.Status)
		}
	})

	t.Run("failure_recovery_before_config_write", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.json")
		plan := fakeOllamaExecutionPlan(t, cfgPath)
		_, err := ExecutePlan(context.Background(), plan, &fakeCommandRunner{failAt: 1}, ExecutorOptions{Approved: true})
		if err == nil {
			t.Fatal("ExecutePlan() error = nil, want command failure")
		}
		if fileExists(cfgPath) {
			t.Fatal("config exists after failed install")
		}
	})

	t.Run("unsafe_existing_state_blocks", func(t *testing.T) {
		env := MinimalUnknownEnvSnapshot(filepath.Join(t.TempDir(), "config.json"))
		env.UnsafeStates = []string{"runtime_bound_to_0.0.0.0"}
		plan, err := BuildPlan(env, OperatorChoices{PresetName: PresetPhoneOllamaTiny, InstallBehavior: "skip"})
		if err != nil {
			t.Fatalf("BuildPlan() error = %v", err)
		}
		if plan.Status != PlanStatusBlocked {
			t.Fatalf("status = %q, want blocked", plan.Status)
		}
	})
}

func writeFakeLlamaAssets(t *testing.T, dir string) (string, string) {
	t.Helper()
	server := filepath.Join(dir, "llama-server")
	model := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(server, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(server) error = %v", err)
	}
	if err := os.WriteFile(model, []byte("model"), 0o600); err != nil {
		t.Fatalf("WriteFile(model) error = %v", err)
	}
	return server, model
}

func loadAcceptanceConfig(t *testing.T, path string) config.Config {
	t.Helper()
	cfg, err := loadConfigFromPath(path)
	if err != nil {
		t.Fatalf("loadConfigFromPath(%q) error = %v", path, err)
	}
	if _, err := config.BuildModelRegistry(cfg); err != nil {
		t.Fatalf("BuildModelRegistry(written) error = %v", err)
	}
	return cfg
}
