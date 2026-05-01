package modelsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
