package modelsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	failAt int
	calls  [][]string
}

func (r *fakeCommandRunner) Run(ctx context.Context, command []string) error {
	r.calls = append(r.calls, append([]string(nil), command...))
	if r.failAt > 0 && len(r.calls) == r.failAt {
		return errors.New("fake command failure")
	}
	return nil
}

func TestExecutePlanRunsApprovedFakeOllamaCommands(t *testing.T) {
	plan := fakeOllamaExecutionPlan(t, filepath.Join(t.TempDir(), "config.json"))
	runner := &fakeCommandRunner{}
	result, err := ExecutePlan(context.Background(), plan, runner, ExecutorOptions{Approved: true})
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	want := [][]string{
		{"install-runtime", "ollama"},
		{"ollama", "pull", "qwen3:1.7b"},
		{"ollama", "serve"},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("runner calls = %#v, want %#v", runner.calls, want)
	}
	if result.Status != PlanStatusChanged {
		t.Fatalf("result status = %q, want changed", result.Status)
	}
}

func TestExecutePlanManualRequiredDoesNotRunCommands(t *testing.T) {
	plan, err := BuildPlan(MinimalUnknownEnvSnapshot(filepath.Join(t.TempDir(), "config.json")), OperatorChoices{
		PresetName: PresetPhoneOllamaTiny,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	runner := &fakeCommandRunner{}
	result, err := ExecutePlan(context.Background(), plan, runner, ExecutorOptions{Approved: true})
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %#v, want none for manual_required plan", runner.calls)
	}
	if result.Status != PlanStatusManualRequired {
		t.Fatalf("result status = %q, want manual_required", result.Status)
	}
}

func TestExecutePlanFailedInstallLeavesConfigUnchanged(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	plan := fakeOllamaExecutionPlan(t, cfgPath)
	runner := &fakeCommandRunner{failAt: 1}
	result, err := ExecutePlan(context.Background(), plan, runner, ExecutorOptions{Approved: true})
	if err == nil {
		t.Fatal("ExecutePlan() error = nil, want fake command failure")
	}
	if result.Status != PlanStatusFailed {
		t.Fatalf("result status = %q, want failed", result.Status)
	}
	if fileExists(cfgPath) {
		t.Fatalf("config file exists after failed install; install failure must happen before config write")
	}
}

func TestExecutePlanRequiresApproval(t *testing.T) {
	plan := fakeOllamaExecutionPlan(t, filepath.Join(t.TempDir(), "config.json"))
	runner := &fakeCommandRunner{}
	result, err := ExecutePlan(context.Background(), plan, runner, ExecutorOptions{})
	if err == nil {
		t.Fatal("ExecutePlan() error = nil, want approval error")
	}
	if result.Status != PlanStatusBlocked {
		t.Fatalf("result status = %q, want blocked", result.Status)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %#v, want none without approval", runner.calls)
	}
}

func TestExecutePlanPhoneLlamaCPPManifestPathWithFakeArtifacts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	plan, err := BuildPlan(supportedPhoneTermuxEnv(cfgPath), OperatorChoices{
		PresetName: PresetPhoneLlamaCPPTiny,
		ConfigPath: cfgPath,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Status != PlanStatusPlanned {
		t.Fatalf("plan status = %q, want planned", plan.Status)
	}
	data := []byte("fake approved artifact")
	registry := fakeApprovedPhoneLlamaCPPRegistry(data)
	runner := &fakeCommandRunner{}
	result, err := ExecutePlan(context.Background(), plan, runner, ExecutorOptions{
		Approved:         true,
		Downloader:       &fakeDownloader{data: data},
		ManifestRegistry: registry,
	})
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if result.Status != PlanStatusChanged {
		t.Fatalf("result status = %q, want changed", result.Status)
	}
	if len(runner.calls) != 4 {
		t.Fatalf("runner calls = %#v, want prepare/extract/locate/start", runner.calls)
	}
	cfg := loadAcceptanceConfig(t, cfgPath)
	model := cfg.Models["local_fast"]
	if model.ProviderModel != "qwen2.5-0.5b-instruct-q4_k_m" {
		t.Fatalf("provider model = %q", model.ProviderModel)
	}
	if model.Capabilities.SupportsTools {
		t.Fatal("supports tools = true, want false")
	}
	if model.Capabilities.AuthorityTier != "low" {
		t.Fatalf("authority tier = %q, want low", model.Capabilities.AuthorityTier)
	}
	if cfg.ModelRouting.AllowCloudFallbackFromLocal {
		t.Fatal("cloud fallback from local = true, want false")
	}
	start := cfg.LocalRuntimes["llamacpp_phone"].StartCommand
	for _, want := range []string{"find", "llama-server", "--host \"127.0.0.1\"", "--port 8080"} {
		if !strings.Contains(start, want) {
			t.Fatalf("start command = %q, want %q", start, want)
		}
	}
	if strings.Contains(start, "--mission-resume-approved") {
		t.Fatalf("start command contains mission resume flag: %q", start)
	}
}

func TestExecutePlanFailedManifestChecksumLeavesPresetNotReadyAndConfigUnchanged(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	plan, err := BuildPlan(supportedPhoneTermuxEnv(cfgPath), OperatorChoices{
		PresetName: PresetPhoneLlamaCPPTiny,
		ConfigPath: cfgPath,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	data := []byte("fake approved artifact")
	registry := fakeApprovedPhoneLlamaCPPRegistry(data)
	runtime := registry[ManifestLlamaCPPAndroidARM64B8994]
	runtime.ChecksumSHA256 = strings.Repeat("0", 64)
	registry[ManifestLlamaCPPAndroidARM64B8994] = runtime
	result, err := ExecutePlan(context.Background(), plan, &fakeCommandRunner{}, ExecutorOptions{
		Approved:         true,
		Downloader:       &fakeDownloader{data: data},
		ManifestRegistry: registry,
	})
	if err == nil {
		t.Fatal("ExecutePlan() error = nil, want checksum failure")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v, want checksum mismatch", err)
	}
	if result.Status != PlanStatusFailed {
		t.Fatalf("result status = %q, want failed", result.Status)
	}
	if plan.Ready {
		t.Fatal("plan Ready = true, want false")
	}
	if fileExists(cfgPath) {
		t.Fatal("config exists after failed checksum")
	}
}

func fakeOllamaExecutionPlan(t *testing.T, cfgPath string) Plan {
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
	plan.Status = PlanStatusPlanned
	plan.Steps = []PlanStep{
		{
			ID:               "install-ollama",
			SideEffect:       SideEffectInstallRuntime,
			Command:          []string{"install-runtime", "ollama"},
			Status:           PlanStatusPlanned,
			ApprovalRequired: true,
		},
		{
			ID:               "pull-ollama-model",
			SideEffect:       SideEffectPullModel,
			Command:          []string{"ollama", "pull", "qwen3:1.7b"},
			Status:           PlanStatusPlanned,
			ApprovalRequired: true,
		},
		{
			ID:               "start-ollama-runtime",
			SideEffect:       SideEffectStartRuntime,
			Command:          []string{"ollama", "serve"},
			Status:           PlanStatusPlanned,
			ApprovalRequired: true,
		},
		{
			ID:         "write-v5-config",
			SideEffect: SideEffectWriteConfig,
			Status:     PlanStatusPlanned,
		},
	}
	return plan
}

func fakeApprovedPhoneLlamaCPPRegistry(data []byte) ManifestRegistry {
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])
	registry := BuiltinManifests()
	runtime := registry[ManifestLlamaCPPAndroidARM64B8994]
	runtime.ChecksumSHA256 = checksum
	runtime.SizeBytes = int64(len(data))
	registry[ManifestLlamaCPPAndroidARM64B8994] = runtime
	model := registry[ManifestQwen25TinyGGUF]
	model.ChecksumSHA256 = checksum
	model.SizeBytes = int64(len(data))
	registry[ManifestQwen25TinyGGUF] = model
	return registry
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
