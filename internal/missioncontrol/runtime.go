package missioncontrol

import (
	"fmt"
	"time"
)

const (
	JobStateWaitingUser JobState = "waiting_user"
	JobStatePaused      JobState = "paused"
	JobStateFailed      JobState = "failed"
)

const (
	RejectionCodeInvalidRuntimeState    RejectionCode = "invalid_runtime_state"
	RejectionCodeResumeApprovalRequired RejectionCode = "resume_approval_required"
	RejectionCodeValidationRequired     RejectionCode = "validation_required"
	RejectionCodeWaitingUser            RejectionCode = "waiting_user"
)

type RuntimeStepRecord struct {
	StepID string    `json:"step_id"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

type JobRuntimeState struct {
	JobID            string              `json:"job_id"`
	State            JobState            `json:"state"`
	ActiveStepID     string              `json:"active_step_id,omitempty"`
	CompletedSteps   []RuntimeStepRecord `json:"completed_steps,omitempty"`
	FailedSteps      []RuntimeStepRecord `json:"failed_steps,omitempty"`
	ApprovalRequests []ApprovalRequest   `json:"approval_requests,omitempty"`
	ApprovalGrants   []ApprovalGrant     `json:"approval_grants,omitempty"`
	WaitingReason    string              `json:"waiting_reason,omitempty"`
	PausedReason     string              `json:"paused_reason,omitempty"`
	CreatedAt        time.Time           `json:"created_at,omitempty"`
	UpdatedAt        time.Time           `json:"updated_at,omitempty"`
	StartedAt        time.Time           `json:"started_at,omitempty"`
	ActiveStepAt     time.Time           `json:"active_step_at,omitempty"`
	WaitingAt        time.Time           `json:"waiting_at,omitempty"`
	PausedAt         time.Time           `json:"paused_at,omitempty"`
	CompletedAt      time.Time           `json:"completed_at,omitempty"`
	FailedAt         time.Time           `json:"failed_at,omitempty"`
}

type RuntimeTransitionOptions struct {
	StepID           string
	WaitingReason    string
	PausedReason     string
	FailureReason    string
	validationResult *stepValidationResult
}

func CloneJobRuntimeState(runtime *JobRuntimeState) *JobRuntimeState {
	if runtime == nil {
		return nil
	}

	cloned := *runtime
	cloned.CompletedSteps = append([]RuntimeStepRecord(nil), runtime.CompletedSteps...)
	cloned.FailedSteps = append([]RuntimeStepRecord(nil), runtime.FailedSteps...)
	cloned.ApprovalRequests = append([]ApprovalRequest(nil), runtime.ApprovalRequests...)
	cloned.ApprovalGrants = append([]ApprovalGrant(nil), runtime.ApprovalGrants...)
	return &cloned
}

func IsTerminalJobState(state JobState) bool {
	switch state {
	case JobStateCompleted, JobStateFailed, JobStateRejected:
		return true
	default:
		return false
	}
}

func ResolveExecutionContextWithRuntime(job Job, runtime JobRuntimeState) (ExecutionContext, error) {
	if runtime.JobID != "" && job.ID != "" && runtime.JobID != job.ID {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match mission job %q", runtime.JobID, job.ID),
		}
	}
	if runtime.ActiveStepID == "" {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "runtime execution requires an active step",
		}
	}

	ec, err := ResolveExecutionContext(job, runtime.ActiveStepID)
	if err != nil {
		return ExecutionContext{}, err
	}
	ec.Runtime = CloneJobRuntimeState(&runtime)
	return ec, nil
}

func SetJobRuntimeActiveStep(job Job, current *JobRuntimeState, stepID string, now time.Time) (JobRuntimeState, error) {
	if _, err := ResolveExecutionContext(job, stepID); err != nil {
		return JobRuntimeState{}, err
	}

	if current == nil || current.JobID == "" {
		return JobRuntimeState{
			JobID:        job.ID,
			State:        JobStateRunning,
			ActiveStepID: stepID,
			CreatedAt:    now,
			UpdatedAt:    now,
			StartedAt:    now,
			ActiveStepAt: now,
		}, nil
	}

	if current.JobID != job.ID {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match mission job %q", current.JobID, job.ID),
		}
	}

	next := *CloneJobRuntimeState(current)
	if next.CreatedAt.IsZero() {
		next.CreatedAt = now
	}

	switch next.State {
	case JobStatePending:
		return TransitionJobRuntime(next, JobStateRunning, now, RuntimeTransitionOptions{StepID: stepID})
	case JobStateRunning:
		next.ActiveStepID = stepID
		next.ActiveStepAt = now
		next.UpdatedAt = now
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		return next, nil
	case JobStateWaitingUser, JobStatePaused:
		next.ActiveStepID = stepID
		return TransitionJobRuntime(next, JobStateRunning, now, RuntimeTransitionOptions{StepID: stepID})
	default:
		if err := ValidateJobTransition(next.State, JobStateRunning); err != nil {
			return JobRuntimeState{}, err
		}
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("cannot activate a new step while job is %q", next.State),
		}
	}
}

func ResumeJobRuntimeAfterBoot(current JobRuntimeState, now time.Time, approved bool) (JobRuntimeState, error) {
	if !approved {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeResumeApprovalRequired,
			Message: "resuming a persisted runtime after reboot requires approval",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "resuming a persisted runtime requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	switch next.State {
	case JobStateRunning, JobStateWaitingUser, JobStatePaused:
		next.State = JobStateRunning
		next.UpdatedAt = now
		if next.CreatedAt.IsZero() {
			next.CreatedAt = now
		}
		if next.StartedAt.IsZero() {
			next.StartedAt = now
		}
		if next.ActiveStepAt.IsZero() {
			next.ActiveStepAt = now
		}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		return next, nil
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("cannot resume runtime while job is %q", next.State),
		}
	}
}

func TransitionJobRuntime(current JobRuntimeState, to JobState, now time.Time, opts RuntimeTransitionOptions) (JobRuntimeState, error) {
	if err := ValidateJobTransition(current.State, to); err != nil {
		return JobRuntimeState{}, err
	}
	if current.JobID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "runtime transition requires a job ID",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.State = to
	next.UpdatedAt = now
	if next.CreatedAt.IsZero() {
		next.CreatedAt = now
	}

	switch to {
	case JobStateRunning:
		stepID := opts.StepID
		if stepID == "" {
			stepID = next.ActiveStepID
		}
		if stepID == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "running state requires an active step",
			}
		}
		if next.StartedAt.IsZero() {
			next.StartedAt = now
		}
		if next.ActiveStepID != stepID || next.ActiveStepAt.IsZero() {
			next.ActiveStepAt = now
		}
		next.ActiveStepID = stepID
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
	case JobStateWaitingUser:
		stepID := opts.StepID
		if stepID == "" {
			stepID = next.ActiveStepID
		}
		if stepID == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "waiting_user state requires an active step",
			}
		}
		if opts.WaitingReason == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "waiting_user state requires a waiting reason",
			}
		}
		next.ActiveStepID = stepID
		next.WaitingReason = opts.WaitingReason
		next.WaitingAt = now
		next.PausedReason = ""
		next.PausedAt = time.Time{}
	case JobStatePaused:
		if opts.PausedReason == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "paused state requires a pause reason",
			}
		}
		next.PausedReason = opts.PausedReason
		next.PausedAt = now
		next.WaitingReason = ""
		next.WaitingAt = time.Time{}
		if opts.validationResult != nil && opts.validationResult.recordCompletion {
			if next.ActiveStepID == "" {
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeInvalidRuntimeState,
					Message: "paused state requires an active step to record completion",
				}
			}
			next.CompletedSteps = append(next.CompletedSteps, RuntimeStepRecord{
				StepID: next.ActiveStepID,
				At:     now,
			})
			next.ActiveStepID = ""
			next.ActiveStepAt = time.Time{}
		}
	case JobStateCompleted:
		if opts.validationResult == nil || !opts.validationResult.recordCompletion {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeValidationRequired,
				Message: "completing a job runtime requires validation",
			}
		}
		if next.ActiveStepID == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "completed state requires an active step to record",
			}
		}
		next.CompletedSteps = append(next.CompletedSteps, RuntimeStepRecord{
			StepID: next.ActiveStepID,
			At:     now,
		})
		next.CompletedAt = now
		next.ActiveStepID = ""
		next.ActiveStepAt = time.Time{}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
	case JobStateFailed:
		if next.ActiveStepID == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "failed state requires an active step to record",
			}
		}
		next.FailedSteps = append(next.FailedSteps, RuntimeStepRecord{
			StepID: next.ActiveStepID,
			Reason: opts.FailureReason,
			At:     now,
		})
		next.FailedAt = now
		next.ActiveStepID = ""
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
	case JobStateRejected:
		next.ActiveStepID = ""
		next.WaitingReason = ""
		next.PausedReason = ""
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
	}

	return next, nil
}

func ValidateRuntimeExecution(ec ExecutionContext) error {
	if ec.Runtime == nil {
		return nil
	}
	if ec.Job == nil || ec.Step == nil || ec.Job.ID == "" || ec.Step.ID == "" || ec.Runtime.ActiveStepID == "" {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "execution requires an active job and active step",
		}
	}
	if ec.Runtime.JobID != "" && ec.Runtime.JobID != ec.Job.ID {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match active job %q", ec.Runtime.JobID, ec.Job.ID),
		}
	}
	if ec.Runtime.ActiveStepID != ec.Step.ID {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime active step %q does not match active execution step %q", ec.Runtime.ActiveStepID, ec.Step.ID),
		}
	}

	switch ec.Runtime.State {
	case JobStateRunning:
		return nil
	case JobStateWaitingUser:
		return ValidationError{
			Code:    RejectionCodeWaitingUser,
			Message: "job is waiting for user input",
		}
	default:
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("job is not executable while in %q state", ec.Runtime.State),
		}
	}
}
