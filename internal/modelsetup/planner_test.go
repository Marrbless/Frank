package modelsetup

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestCatalogContainsRequiredDefaultSafePresetsAndGatedLAN(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatalf("ValidateCatalog() error = %v", err)
	}
	required := map[string]bool{
		PresetPhoneOllamaTiny:      false,
		PresetPhoneLlamaCPPTiny:    false,
		PresetDesktopOllamaLocal:   false,
		PresetDesktopLlamaCPPLocal: false,
		PresetCloudOpenRouter:      false,
		PresetCloudOpenAI:          false,
		PresetMixedLocalCloudSafe:  false,
	}
	for _, preset := range Catalog() {
		if _, ok := required[preset.Name]; ok {
			if !preset.DefaultSafe {
				t.Fatalf("preset %q DefaultSafe = false, want true", preset.Name)
			}
			required[preset.Name] = true
		}
		if preset.RuntimeKind != RuntimeKindCloud && preset.BindAddress != "127.0.0.1" {
			t.Fatalf("preset %q bind address = %q, want localhost", preset.Name, preset.BindAddress)
		}
	}
	for name, found := range required {
		if !found {
			t.Fatalf("required preset %q missing", name)
		}
	}
	lan, ok := PresetByName(PresetLANLlamaCPPLocal)
	if !ok {
		t.Fatalf("preset %q missing", PresetLANLlamaCPPLocal)
	}
	if lan.DefaultSafe || !lan.ExplicitlyGated {
		t.Fatalf("LAN preset safety = defaultSafe:%t gated:%t, want false/true", lan.DefaultSafe, lan.ExplicitlyGated)
	}
}

func TestPlannerUsesInjectedSnapshotAndIsDeterministic(t *testing.T) {
	env := EnvSnapshot{
		Platform:          "test_platform",
		OS:                "linux",
		Arch:              "arm64",
		ConfigPath:        "/tmp/test-config.json",
		Termux:            StateMissing,
		Ollama:            StateUnknown,
		ExistingProviders: []string{"openrouter"},
	}
	choices := OperatorChoices{PresetName: PresetPhoneOllamaTiny, ConfigPath: env.ConfigPath, DryRun: true}
	first, err := BuildPlan(env, choices)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	second, err := BuildPlan(env, choices)
	if err != nil {
		t.Fatalf("BuildPlan() second error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("BuildPlan() not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.Environment.Platform != "test_platform" {
		t.Fatalf("Environment.Platform = %q, want injected snapshot", first.Environment.Platform)
	}
	if first.Status != PlanStatusManualRequired {
		t.Fatalf("Status = %q, want manual_required because no installer manifest exists", first.Status)
	}
}

func TestPlannerDoesNotTouchFilesystem(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot(cfgPath), OperatorChoices{PresetName: PresetPhoneOllamaTiny, ConfigPath: cfgPath, DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.PresetName == "" {
		t.Fatal("plan missing preset")
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exist; planner must not write files", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("planner wrote files: %v", entries)
	}
}

func TestPlannerLocalDefaultsDenyToolsCloudFallbackAndLAN(t *testing.T) {
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot("/tmp/config.json"), OperatorChoices{PresetName: PresetPhoneOllamaTiny, ConfigPath: "/tmp/config.json", DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.ToolSupport {
		t.Fatal("ToolSupport = true, want false for local preset")
	}
	if plan.AuthorityTier != config.ModelAuthorityLow {
		t.Fatalf("AuthorityTier = %q, want low", plan.AuthorityTier)
	}
	if plan.CloudFallback {
		t.Fatal("CloudFallback = true, want false")
	}
	if plan.BindAddress != "127.0.0.1" {
		t.Fatalf("BindAddress = %q, want localhost", plan.BindAddress)
	}
	if plan.ConfigPatch == nil {
		t.Fatal("ConfigPatch = nil")
	}
	if plan.ConfigPatch.ModelConfig.Capabilities.SupportsTools {
		t.Fatal("generated model supports tools, want false")
	}
	if plan.ConfigPatch.RoutingConfig.AllowCloudFallbackFromLocal {
		t.Fatal("generated routing allows cloud fallback, want false")
	}
}

func TestPlannerBlocksLANPresetWithoutExplicitApproval(t *testing.T) {
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot("/tmp/config.json"), OperatorChoices{PresetName: PresetLANLlamaCPPLocal, ConfigPath: "/tmp/config.json", DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Status != PlanStatusBlocked {
		t.Fatalf("Status = %q, want blocked", plan.Status)
	}
	if !containsString(plan.BlockedReasons, "requires explicit approval") {
		t.Fatalf("BlockedReasons = %#v, want LAN approval reason", plan.BlockedReasons)
	}
}

func TestPlannerDetectsNormalizedRefCollisions(t *testing.T) {
	env := MinimalUnknownEnvSnapshot("/tmp/config.json")
	env.ExistingModels = []string{" Local_Fast ", "local_fast"}
	env.ExistingAliases = []string{" Phone ", "phone"}
	plan, err := BuildPlan(env, OperatorChoices{PresetName: PresetPhoneOllamaTiny, ConfigPath: "/tmp/config.json", DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Status != PlanStatusBlocked {
		t.Fatalf("Status = %q, want blocked", plan.Status)
	}
	if !containsString(plan.BlockedReasons, `model_ref collision after normalization`) {
		t.Fatalf("BlockedReasons = %#v, want model collision", plan.BlockedReasons)
	}
	if !containsString(plan.BlockedReasons, `alias collision after normalization`) {
		t.Fatalf("BlockedReasons = %#v, want alias collision", plan.BlockedReasons)
	}
}

func TestPrintPlanRedactsSecretsAndIsStable(t *testing.T) {
	env := MinimalUnknownEnvSnapshot("/tmp/config.json")
	env.ExistingProviders = []string{"openrouter"}
	plan, err := BuildPlan(env, OperatorChoices{PresetName: PresetCloudOpenRouter, ConfigPath: "/tmp/config.json", DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	var first bytes.Buffer
	PrintPlan(&first, plan)
	var second bytes.Buffer
	PrintPlan(&second, plan)
	if first.String() != second.String() {
		t.Fatalf("PrintPlan() not deterministic\nfirst=%s\nsecond=%s", first.String(), second.String())
	}
	for _, forbidden := range []string{"sk-test-secret", "Authorization", "apiKey"} {
		if strings.Contains(first.String(), forbidden) {
			t.Fatalf("PrintPlan() leaked forbidden token %q in:\n%s", forbidden, first.String())
		}
	}
}

func TestTruncateReportStepsPreservesDiagnostics(t *testing.T) {
	steps := []PlanStep{
		{ID: "ok", Status: PlanStatusPlanned, DiagnosticsPriority: 1},
		{ID: "manual", Status: PlanStatusManualRequired, DiagnosticsPriority: 1},
		{ID: "blocked", Status: PlanStatusBlocked, DiagnosticsPriority: 1},
		{ID: "failed", Status: PlanStatusFailed, DiagnosticsPriority: 1},
	}
	got := TruncateReportSteps(steps, 2)
	if len(got) != 3 {
		t.Fatalf("len(TruncateReportSteps) = %d, want preserved diagnostics even above limit", len(got))
	}
	for _, status := range []PlanStatus{PlanStatusManualRequired, PlanStatusBlocked, PlanStatusFailed} {
		if !hasStatus(got, status) {
			t.Fatalf("truncated steps = %#v, missing %s", got, status)
		}
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func hasStatus(steps []PlanStep, status PlanStatus) bool {
	for _, step := range steps {
		if step.Status == status {
			return true
		}
	}
	return false
}
