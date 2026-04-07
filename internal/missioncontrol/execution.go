package missioncontrol

import "fmt"

const RejectionCodeUnknownStep RejectionCode = "unknown_step"

func CloneExecutionContext(ec ExecutionContext) ExecutionContext {
	var cloned ExecutionContext
	if ec.Job != nil {
		cloned.Job = CloneJob(ec.Job)
	}
	if ec.Step != nil {
		stepCopy := copyStep(*ec.Step)
		cloned.Step = &stepCopy
	}
	cloned.Runtime = CloneJobRuntimeState(ec.Runtime)
	cloned.MissionStoreRoot = ec.MissionStoreRoot
	cloned.GovernedExternalTargets = cloneAutonomyEligibilityTargetRefs(ec.GovernedExternalTargets)
	return cloned
}

func CloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	jobCopy := copyJob(*job)
	return &jobCopy
}

func ResolveExecutionContext(job Job, stepID string) (ExecutionContext, error) {
	if validationErrors := ValidatePlan(job); len(validationErrors) > 0 {
		return ExecutionContext{}, validationErrors[0]
	}

	stepIndex := -1
	for i, step := range job.Plan.Steps {
		if step.ID == stepID {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeUnknownStep,
			StepID:  stepID,
			Message: fmt.Sprintf(`step %q not found in plan`, stepID),
		}
	}

	jobCopy := copyJob(job)
	stepCopy := normalizedStep(jobCopy.Plan.Steps[stepIndex])
	jobCopy.Plan.Steps[stepIndex] = copyStep(stepCopy)
	return ExecutionContext{
		Job:                     &jobCopy,
		Step:                    &stepCopy,
		GovernedExternalTargets: cloneAutonomyEligibilityTargetRefs(stepCopy.GovernedExternalTargets),
	}, nil
}

func copyJob(job Job) Job {
	jobCopy := job
	jobCopy.AllowedTools = append([]string(nil), job.AllowedTools...)
	jobCopy.Plan = Plan{ID: job.Plan.ID}
	if job.Plan.Steps != nil {
		jobCopy.Plan.Steps = make([]Step, len(job.Plan.Steps))
		for i, step := range job.Plan.Steps {
			jobCopy.Plan.Steps[i] = copyStep(step)
		}
	}
	return jobCopy
}

func copyStep(step Step) Step {
	stepCopy := step
	stepCopy.DependsOn = append([]string(nil), step.DependsOn...)
	stepCopy.AllowedTools = append([]string(nil), step.AllowedTools...)
	stepCopy.SuccessCriteria = append([]string(nil), step.SuccessCriteria...)
	stepCopy.LongRunningStartupCommand = append([]string(nil), step.LongRunningStartupCommand...)
	stepCopy.GovernedExternalTargets = cloneAutonomyEligibilityTargetRefs(step.GovernedExternalTargets)
	stepCopy.SystemAction = cloneSystemActionSpec(step.SystemAction)
	return stepCopy
}

func normalizedStep(step Step) Step {
	stepCopy := copyStep(step)
	stepCopy.IdentityMode = NormalizeIdentityMode(stepCopy.IdentityMode)
	return stepCopy
}

func cloneAutonomyEligibilityTargetRefs(targets []AutonomyEligibilityTargetRef) []AutonomyEligibilityTargetRef {
	if len(targets) == 0 {
		return nil
	}

	cloned := make([]AutonomyEligibilityTargetRef, len(targets))
	copy(cloned, targets)
	return cloned
}
