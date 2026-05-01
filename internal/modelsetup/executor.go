package modelsetup

import (
	"context"
	"fmt"
)

type CommandRunner interface {
	Run(ctx context.Context, command []string) error
}

type ExecutorOptions struct {
	Approved            bool
	ApproveReplacements bool
	Force               bool
	ConfigWriteOptions  ConfigWriteOptions
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
			step.Status = PlanStatusManualRequired
			result.Status = PlanStatusManualRequired
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
