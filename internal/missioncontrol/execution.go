package missioncontrol

import "fmt"

const RejectionCodeUnknownStep RejectionCode = "unknown_step"

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
	return ExecutionContext{
		Job:  &jobCopy,
		Step: &jobCopy.Plan.Steps[stepIndex],
	}, nil
}

func copyJob(job Job) Job {
	jobCopy := job
	jobCopy.AllowedTools = append([]string(nil), job.AllowedTools...)
	jobCopy.Plan = Plan{
		ID:    job.Plan.ID,
		Steps: make([]Step, len(job.Plan.Steps)),
	}
	for i, step := range job.Plan.Steps {
		jobCopy.Plan.Steps[i] = copyStep(step)
	}
	return jobCopy
}

func copyStep(step Step) Step {
	stepCopy := step
	stepCopy.DependsOn = append([]string(nil), step.DependsOn...)
	stepCopy.AllowedTools = append([]string(nil), step.AllowedTools...)
	stepCopy.SuccessCriteria = append([]string(nil), step.SuccessCriteria...)
	return stepCopy
}
