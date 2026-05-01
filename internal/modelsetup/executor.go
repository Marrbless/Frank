package modelsetup

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, command []string) error
}

type ExecutorOptions struct {
	Approved            bool
	ApproveReplacements bool
	Force               bool
	ConfigWriteOptions  ConfigWriteOptions
	Downloader          Downloader
	ManifestRegistry    ManifestRegistry
}

type ExecutionResult struct {
	Status PlanStatus
	Steps  []PlanStep
	Config ConfigWriteResult
}

func ExecutePlan(ctx context.Context, plan Plan, runner CommandRunner, opts ExecutorOptions) (ExecutionResult, error) {
	result := ExecutionResult{Status: PlanStatusPlanned, Steps: append([]PlanStep(nil), plan.Steps...)}
	if !opts.Approved {
		result.Status = PlanStatusBlocked
		return result, fmt.Errorf("plan execution requires explicit approval")
	}
	if plan.Status == PlanStatusBlocked {
		result.Status = PlanStatusBlocked
		return result, fmt.Errorf("cannot execute blocked plan")
	}
	for i := range result.Steps {
		step := &result.Steps[i]
		switch step.Status {
		case PlanStatusSkipped, PlanStatusAlreadyPresent:
			continue
		case PlanStatusManualRequired:
			result.Status = PlanStatusManualRequired
			continue
		case PlanStatusBlocked:
			result.Status = PlanStatusBlocked
			return result, fmt.Errorf("step %s is blocked", step.ID)
		}
		switch step.SideEffect {
		case SideEffectNone, SideEffectReadFile, SideEffectHealthCheck, SideEffectRouteCheck:
			step.Status = PlanStatusAlreadyPresent
		case SideEffectWriteConfig:
			writeOpts := opts.ConfigWriteOptions
			writeOpts.ApproveReplacements = opts.ApproveReplacements
			writeOpts.Force = opts.Force
			configResult, err := ApplyConfigPlan(plan, writeOpts)
			result.Config = configResult
			step.Status = configResult.Status
			if err != nil {
				result.Status = configResult.Status
				return result, err
			}
		case SideEffectInstallRuntime, SideEffectPullModel, SideEffectStartRuntime, SideEffectRunCommand:
			if runner == nil {
				step.Status = PlanStatusBlocked
				result.Status = PlanStatusBlocked
				return result, fmt.Errorf("step %s requires command runner", step.ID)
			}
			if len(step.Command) == 0 {
				step.Status = PlanStatusManualRequired
				result.Status = PlanStatusManualRequired
				continue
			}
			if err := runner.Run(ctx, step.Command); err != nil {
				step.Status = PlanStatusFailed
				result.Status = PlanStatusFailed
				return result, fmt.Errorf("step %s failed: %w", step.ID, err)
			}
			step.Status = PlanStatusChanged
		case SideEffectDownload:
			changed, err := executeDownloadStep(ctx, step, opts)
			if err != nil {
				step.Status = PlanStatusFailed
				result.Status = PlanStatusFailed
				return result, fmt.Errorf("step %s failed: %w", step.ID, err)
			}
			if changed {
				step.Status = PlanStatusChanged
			} else {
				step.Status = PlanStatusAlreadyPresent
			}
		case SideEffectWriteBootScript:
			step.Status = PlanStatusManualRequired
			result.Status = PlanStatusManualRequired
		default:
			step.Status = PlanStatusBlocked
			result.Status = PlanStatusBlocked
			return result, fmt.Errorf("step %s has unsupported side effect %q", step.ID, step.SideEffect)
		}
	}
	if result.Status == PlanStatusPlanned {
		result.Status = aggregateExecutionStatus(result.Steps)
	}
	return result, nil
}

type ShellCommandRunner struct{}

func (ShellCommandRunner) Run(ctx context.Context, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("command is required")
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func executeDownloadStep(ctx context.Context, step *PlanStep, opts ExecutorOptions) (bool, error) {
	if strings.TrimSpace(step.ManifestID) == "" {
		return false, fmt.Errorf("download step requires manifest id")
	}
	registry := opts.ManifestRegistry
	if registry == nil {
		registry = BuiltinManifests()
	}
	manifest, ok := registry.Lookup(step.ManifestID)
	if !ok {
		return false, fmt.Errorf("manifest %q is not available", step.ManifestID)
	}
	if len(step.FilesToWrite) == 0 {
		return false, fmt.Errorf("download step requires destination")
	}
	destination, err := ExpandHomePath(step.FilesToWrite[0])
	if err != nil {
		return false, err
	}
	if ready, err := ManifestDestinationReady(manifest, destination); err != nil {
		return false, err
	} else if ready {
		return false, nil
	}
	downloader := opts.Downloader
	if downloader == nil {
		downloader = HTTPDownloader{}
	}
	return true, DownloadAndVerifyManifest(ctx, manifest, downloader, destination)
}

func aggregateExecutionStatus(steps []PlanStep) PlanStatus {
	status := PlanStatusAlreadyPresent
	for _, step := range steps {
		switch step.Status {
		case PlanStatusFailed:
			return PlanStatusFailed
		case PlanStatusBlocked:
			return PlanStatusBlocked
		case PlanStatusRolledBack:
			return PlanStatusRolledBack
		case PlanStatusManualRequired:
			status = PlanStatusManualRequired
		case PlanStatusChanged:
			if status != PlanStatusManualRequired {
				status = PlanStatusChanged
			}
		case PlanStatusPlanned:
			if status != PlanStatusManualRequired && status != PlanStatusChanged {
				status = PlanStatusPlanned
			}
		}
	}
	return status
}
