package modelsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/config"
)

func TestApplyConfigPlanBacksUpWritesAndValidatesV5Config(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "telegram-secret"
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	plan := testConfigWritePlan(t, cfgPath)
	result, err := ApplyConfigPlan(plan, ConfigWriteOptions{Now: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("ApplyConfigPlan() error = %v", err)
	}
	if result.Status != PlanStatusChanged || !result.Changed {
		t.Fatalf("result = %#v, want changed", result)
	}
	if result.BackupPath == "" {
		t.Fatalf("BackupPath is empty")
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatalf("backup stat error = %v", err)
	}
	written, err := loadConfigFromPath(cfgPath)
	if err != nil {
		t.Fatalf("load written config error = %v", err)
	}
	if written.Channels.Telegram.Token != "telegram-secret" {
		t.Fatalf("telegram token = %q, want preserved secret", written.Channels.Telegram.Token)
	}
	if _, err := config.BuildModelRegistry(written); err != nil {
		t.Fatalf("written config registry error = %v", err)
	}
	if written.Models["local_fast"].Capabilities.SupportsTools {
		t.Fatal("written local model supports tools, want false")
	}
	if written.Models["local_fast"].Capabilities.AuthorityTier != config.ModelAuthorityLow {
		t.Fatalf("authority tier = %q, want low", written.Models["local_fast"].Capabilities.AuthorityTier)
	}
	if written.ModelRouting.AllowCloudFallbackFromLocal {
		t.Fatal("written routing allows cloud fallback, want false")
	}
}

func TestApplyConfigPlanConflictRequiresApprovalAfterNormalization(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Providers.Named = map[string]config.ProviderConfig{
		" Ollama_Phone ": {Type: config.ProviderTypeOpenAICompatible, APIKey: "different", APIBase: "http://127.0.0.1:11434/v1"},
	}
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	plan := testConfigWritePlan(t, cfgPath)
	_, err := ApplyConfigPlan(plan, ConfigWriteOptions{})
	if err == nil {
		t.Fatal("ApplyConfigPlan() error = nil, want provider conflict")
	}
	if !strings.Contains(err.Error(), "replacement requires approval") {
		t.Fatalf("error = %v, want replacement approval", err)
	}
	data, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if strings.Contains(string(data), "local_fast") {
		t.Fatalf("config changed after conflict: %s", string(data))
	}
}

func TestApplyConfigPlanInvalidGeneratedConfigIsNotWritten(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}
	plan := testConfigWritePlan(t, cfgPath)
	plan.ConfigPatch.ModelConfig.ProviderModel = ""
	_, err = ApplyConfigPlan(plan, ConfigWriteOptions{ApproveReplacements: true})
	if err == nil {
		t.Fatal("ApplyConfigPlan() error = nil, want validation failure")
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("config changed after invalid generated config")
	}
}

func TestApplyConfigPlanAlreadyPresentIsIdempotent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	plan := testConfigWritePlan(t, cfgPath)
	first, err := ApplyConfigPlan(plan, ConfigWriteOptions{})
	if err != nil {
		t.Fatalf("ApplyConfigPlan(first) error = %v", err)
	}
	if first.Status != PlanStatusChanged {
		t.Fatalf("first status = %q, want changed", first.Status)
	}
	second, err := ApplyConfigPlan(plan, ConfigWriteOptions{})
	if err != nil {
		t.Fatalf("ApplyConfigPlan(second) error = %v", err)
	}
	if second.Status != PlanStatusAlreadyPresent {
		t.Fatalf("second status = %q, want already_present", second.Status)
	}
	if second.BackupPath != "" || second.Changed {
		t.Fatalf("second result = %#v, want no backup or change", second)
	}
}

func testConfigWritePlan(t *testing.T, cfgPath string) Plan {
	t.Helper()
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{
		PresetName:      PresetPhoneOllamaTiny,
		ConfigPath:      cfgPath,
		DryRun:          true,
		InstallBehavior: "skip",
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.ConfigPatch == nil {
		t.Fatal("ConfigPatch = nil")
	}
	return plan
}
