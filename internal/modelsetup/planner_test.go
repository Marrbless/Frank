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
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot("/tmp/config.json"), OperatorChoices{PresetName: PresetPhoneLlamaCPPTiny, ConfigPath: "/tmp/config.json", DryRun: true})
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
	if plan.ConfigPatch.ModelConfig.Capabilities.ContextTokens != 2048 {
		t.Fatalf("context tokens = %d, want 2048", plan.ConfigPatch.ModelConfig.Capabilities.ContextTokens)
	}
	if plan.ConfigPatch.ModelConfig.Capabilities.MaxOutputTokens != 512 {
		t.Fatalf("max output tokens = %d, want 512", plan.ConfigPatch.ModelConfig.Capabilities.MaxOutputTokens)
	}
}

func TestPhoneLlamaCPPTinyManifestPlanIsDeterministicApprovedAndNotReady(t *testing.T) {
	env := supportedPhoneTermuxEnv("/tmp/config.json")
	choices := OperatorChoices{PresetName: PresetPhoneLlamaCPPTiny, ConfigPath: "/tmp/config.json", DryRun: true}
	first, err := BuildPlan(env, choices)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	second, err := BuildPlan(env, choices)
	if err != nil {
		t.Fatalf("BuildPlan() second error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("phone llama.cpp plan not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.Status != PlanStatusPlanned || !first.Approved {
		t.Fatalf("status/approved = %s/%t, want planned/true", first.Status, first.Approved)
	}
	if first.Ready {
		t.Fatal("Ready = true before install/readiness, want false")
	}
	var report bytes.Buffer
	PrintPlan(&report, first)
	out := report.String()
	for _, want := range []string{
		"approved: true",
		"ready: false",
		"network_url: https://github.com/ggml-org/llama.cpp/releases/download/b8994/llama-b8994-bin-android-arm64.tar.gz",
		"network_url: https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/df5bf01389a39c743ab467d734bf501681e041c5/qwen2.5-0.5b-instruct-q4_k_m.gguf",
		"checksum_sha256: bf0445968910d36ef85cc273501601189db3ed052a57c0393ba56bd346ab7d54",
		"checksum_sha256: 74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db",
		"bind_address: 127.0.0.1",
		"supports_tools: false",
		"authority_tier: low",
		"cloud_fallback: false",
		"locate-llamacpp-executable",
		"start-llamacpp-runtime",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan output = %q, want %q", out, want)
		}
	}
	for _, forbidden := range []string{"write-termux-boot-script", "frank-gateway", "--mission-resume-approved", "0.0.0.0"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("plan output contains forbidden %q:\n%s", forbidden, out)
		}
	}
}

func TestPhoneLlamaCPPTinyIncompleteSnapshotIsManualRequiredNotReady(t *testing.T) {
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot("/tmp/config.json"), OperatorChoices{PresetName: PresetPhoneLlamaCPPTiny, ConfigPath: "/tmp/config.json", DryRun: true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Status != PlanStatusManualRequired {
		t.Fatalf("status = %q, want manual_required for unconfirmed phone platform", plan.Status)
	}
	if plan.Approved || plan.Ready {
		t.Fatalf("approved/ready = %t/%t, want false/false", plan.Approved, plan.Ready)
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

func supportedPhoneTermuxEnv(configPath string) EnvSnapshot {
	env := MinimalUnknownEnvSnapshot(configPath)
	env.Platform = "android_termux_arm64"
	env.OS = "android"
	env.Arch = "arm64"
	env.Termux = StatePresent
	env.TermuxBoot = StateMissing
	return env
}
