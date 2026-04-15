package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

const (
	JobStateWaitingUser JobState = "waiting_user"
	JobStatePaused      JobState = "paused"
	JobStateFailed      JobState = "failed"
	JobStateAborted     JobState = "aborted"
)

const (
	RuntimePauseReasonOperatorCommand = "operator_command"
	RuntimePauseReasonBudgetExhausted = "budget_exhausted"
	RuntimeAbortReasonOperatorCommand = "operator_command"
)

const (
	unattendedWallClockBudgetCeiling      = "unattended_wall_clock"
	maxUnattendedWallClockPerJob          = 4 * time.Hour
	maxUnattendedWallClockPerJobInMinutes = int(maxUnattendedWallClockPerJob / time.Minute)
	ownerMessagesBudgetCeiling            = "owner_messages"
	maxOwnerFacingMessagesPerJob          = 20
	failedActionsBudgetCeiling            = "failed_actions"
	maxFailedActionsBeforePause           = 5
)

const (
	ownerFacingApprovalRequestAction = "approval_request"
	ownerFacingBudgetPauseAction     = "budget_pause_notification"
	ownerFacingCompletionAction      = "completion_notification"
	ownerFacingDailySummaryAction    = "daily_summary"
	ownerFacingDenyAckAction         = "deny_ack"
	ownerFacingMessageAction         = "message"
	ownerFacingCheckInAction         = "check_in"
	ownerFacingPauseAckAction        = "pause_ack"
	ownerFacingRevokeAckAction       = "revoke_approval_ack"
	ownerFacingStepOutputAction      = "step_output"
	ownerFacingResumeAckAction       = "resume_ack"
	ownerFacingSetStepAckAction      = "set_step_ack"
	ownerFacingWaitingUserAction     = "waiting_user_notification"
)

const (
	RejectionCodeInvalidRuntimeState    RejectionCode = "invalid_runtime_state"
	RejectionCodeResumeApprovalRequired RejectionCode = "resume_approval_required"
	RejectionCodeValidationRequired     RejectionCode = "validation_required"
	RejectionCodeWaitingUser            RejectionCode = "waiting_user"
)

type RuntimeStepRecord struct {
	StepID         string                       `json:"step_id"`
	Reason         string                       `json:"reason,omitempty"`
	At             time.Time                    `json:"at"`
	ResultingState *RuntimeResultingStateRecord `json:"resulting_state,omitempty"`
	Rollback       *RuntimeRollbackRecord       `json:"rollback,omitempty"`
}

type RuntimeBudgetBlockerRecord struct {
	Ceiling     string    `json:"ceiling"`
	Limit       int       `json:"limit,omitempty"`
	Observed    int       `json:"observed,omitempty"`
	Message     string    `json:"message,omitempty"`
	TriggeredAt time.Time `json:"triggered_at,omitempty"`
}

type InspectablePlanContext struct {
	MaxAuthority AuthorityTier `json:"max_authority"`
	AllowedTools []string      `json:"allowed_tools,omitempty"`
	Steps        []Step        `json:"steps,omitempty"`
}

type JobRuntimeState struct {
	JobID                            string                            `json:"job_id"`
	State                            JobState                          `json:"state"`
	ActiveStepID                     string                            `json:"active_step_id,omitempty"`
	InspectablePlan                  *InspectablePlanContext           `json:"inspectable_plan,omitempty"`
	CompletedSteps                   []RuntimeStepRecord               `json:"completed_steps,omitempty"`
	FailedSteps                      []RuntimeStepRecord               `json:"failed_steps,omitempty"`
	AuditHistory                     []AuditEvent                      `json:"audit_history,omitempty"`
	CampaignZohoEmailOutboundActions []CampaignZohoEmailOutboundAction `json:"-"`
	FrankZohoSendReceipts            []FrankZohoSendReceipt            `json:"-"`
	ApprovalRequests                 []ApprovalRequest                 `json:"approval_requests,omitempty"`
	ApprovalGrants                   []ApprovalGrant                   `json:"approval_grants,omitempty"`
	BudgetBlocker                    *RuntimeBudgetBlockerRecord       `json:"budget_blocker,omitempty"`
	WaitingReason                    string                            `json:"waiting_reason,omitempty"`
	PausedReason                     string                            `json:"paused_reason,omitempty"`
	AbortedReason                    string                            `json:"aborted_reason,omitempty"`
	CreatedAt                        time.Time                         `json:"created_at,omitempty"`
	UpdatedAt                        time.Time                         `json:"updated_at,omitempty"`
	StartedAt                        time.Time                         `json:"started_at,omitempty"`
	ActiveStepAt                     time.Time                         `json:"active_step_at,omitempty"`
	WaitingAt                        time.Time                         `json:"waiting_at,omitempty"`
	PausedAt                         time.Time                         `json:"paused_at,omitempty"`
	AbortedAt                        time.Time                         `json:"aborted_at,omitempty"`
	CompletedAt                      time.Time                         `json:"completed_at,omitempty"`
	FailedAt                         time.Time                         `json:"failed_at,omitempty"`
}

type RuntimeControlContext struct {
	JobID        string        `json:"job_id"`
	MaxAuthority AuthorityTier `json:"max_authority"`
	AllowedTools []string      `json:"allowed_tools,omitempty"`
	Step         Step          `json:"step"`
}

type RuntimeTransitionOptions struct {
	StepID           string
	WaitingReason    string
	PausedReason     string
	AbortedReason    string
	FailureReason    string
	validationResult *stepValidationResult
}

func CloneJobRuntimeState(runtime *JobRuntimeState) *JobRuntimeState {
	if runtime == nil {
		return nil
	}

	cloned := *runtime
	cloned.InspectablePlan = CloneInspectablePlanContext(runtime.InspectablePlan)
	if len(runtime.CompletedSteps) > 0 {
		cloned.CompletedSteps = make([]RuntimeStepRecord, len(runtime.CompletedSteps))
		for i, record := range runtime.CompletedSteps {
			cloned.CompletedSteps[i] = cloneRuntimeStepRecord(record)
		}
	} else {
		cloned.CompletedSteps = nil
	}
	if len(runtime.FailedSteps) > 0 {
		cloned.FailedSteps = make([]RuntimeStepRecord, len(runtime.FailedSteps))
		for i, record := range runtime.FailedSteps {
			cloned.FailedSteps[i] = cloneRuntimeStepRecord(record)
		}
	} else {
		cloned.FailedSteps = nil
	}
	cloned.AuditHistory = CloneAuditHistory(runtime.AuditHistory)
	cloned.CampaignZohoEmailOutboundActions = cloneCampaignZohoEmailOutboundActions(runtime.CampaignZohoEmailOutboundActions)
	cloned.FrankZohoSendReceipts = cloneFrankZohoSendReceipts(runtime.FrankZohoSendReceipts)
	if len(runtime.ApprovalRequests) > 0 {
		cloned.ApprovalRequests = make([]ApprovalRequest, len(runtime.ApprovalRequests))
		for i, request := range runtime.ApprovalRequests {
			cloned.ApprovalRequests[i] = cloneApprovalRequest(request)
		}
	} else {
		cloned.ApprovalRequests = nil
	}
	cloned.ApprovalGrants = append([]ApprovalGrant(nil), runtime.ApprovalGrants...)
	cloned.BudgetBlocker = cloneRuntimeBudgetBlockerRecord(runtime.BudgetBlocker)
	return &cloned
}

func cloneRuntimeStepRecord(record RuntimeStepRecord) RuntimeStepRecord {
	cloned := record
	cloned.ResultingState = cloneRuntimeResultingStateRecord(record.ResultingState)
	cloned.Rollback = cloneRuntimeRollbackRecord(record.Rollback)
	return cloned
}

func cloneRuntimeBudgetBlockerRecord(record *RuntimeBudgetBlockerRecord) *RuntimeBudgetBlockerRecord {
	if record == nil {
		return nil
	}

	cloned := *record
	return &cloned
}

func CloneInspectablePlanContext(plan *InspectablePlanContext) *InspectablePlanContext {
	if plan == nil {
		return nil
	}

	cloned := *plan
	cloned.AllowedTools = append([]string(nil), plan.AllowedTools...)
	if len(plan.Steps) > 0 {
		cloned.Steps = make([]Step, len(plan.Steps))
		for i, step := range plan.Steps {
			cloned.Steps[i] = copyStep(step)
		}
	} else {
		cloned.Steps = nil
	}
	return &cloned
}

func cloneApprovalRequest(request ApprovalRequest) ApprovalRequest {
	cloned := request
	if request.Content != nil {
		content := *request.Content
		cloned.Content = &content
	}
	return cloned
}

func CloneRuntimeControlContext(control *RuntimeControlContext) *RuntimeControlContext {
	if control == nil {
		return nil
	}

	cloned := *control
	cloned.AllowedTools = append([]string(nil), control.AllowedTools...)
	cloned.Step = copyStep(control.Step)
	return &cloned
}

func IsTerminalJobState(state JobState) bool {
	switch state {
	case JobStateCompleted, JobStateFailed, JobStateRejected, JobStateAborted:
		return true
	default:
		return false
	}
}

func BuildRuntimeControlContext(job Job, stepID string) (RuntimeControlContext, error) {
	ec, err := ResolveExecutionContext(job, stepID)
	if err != nil {
		return RuntimeControlContext{}, err
	}
	if ec.Job == nil || ec.Step == nil {
		return RuntimeControlContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "runtime control requires an active job and active step",
		}
	}

	return RuntimeControlContext{
		JobID:        ec.Job.ID,
		MaxAuthority: ec.Job.MaxAuthority,
		AllowedTools: append([]string(nil), ec.Job.AllowedTools...),
		Step:         normalizedStep(*ec.Step),
	}, nil
}

func BuildInspectablePlanContext(job Job) (InspectablePlanContext, error) {
	if validationErrors := ValidatePlan(job); len(validationErrors) > 0 {
		return InspectablePlanContext{}, validationErrors[0]
	}

	context := InspectablePlanContext{
		MaxAuthority: job.MaxAuthority,
		AllowedTools: append([]string(nil), job.AllowedTools...),
	}
	if len(job.Plan.Steps) > 0 {
		context.Steps = make([]Step, len(job.Plan.Steps))
		for i, step := range job.Plan.Steps {
			context.Steps[i] = copyStep(step)
		}
	}
	return context, nil
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
	if err := validateRuntimeActiveStepReplayMarkers(runtime, "resolve"); err != nil {
		return ExecutionContext{}, err
	}

	ec, err := ResolveExecutionContext(job, runtime.ActiveStepID)
	if err != nil {
		return ExecutionContext{}, err
	}
	ec.Runtime = CloneJobRuntimeState(&runtime)
	return ec, nil
}

func ResolveExecutionContextWithRuntimeControl(control RuntimeControlContext, runtime JobRuntimeState) (ExecutionContext, error) {
	if runtime.JobID != "" && control.JobID != "" && runtime.JobID != control.JobID {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match mission job %q", runtime.JobID, control.JobID),
		}
	}
	if runtime.ActiveStepID == "" {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "runtime execution requires an active step",
		}
	}
	if control.Step.ID == "" {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "runtime control requires an active step",
		}
	}
	if runtime.ActiveStepID != control.Step.ID {
		return ExecutionContext{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime active step %q does not match control step %q", runtime.ActiveStepID, control.Step.ID),
		}
	}
	if err := validateRuntimeActiveStepReplayMarkers(runtime, "resolve"); err != nil {
		return ExecutionContext{}, err
	}

	job := Job{
		ID:           control.JobID,
		MaxAuthority: control.MaxAuthority,
		AllowedTools: append([]string(nil), control.AllowedTools...),
		Plan: Plan{
			Steps: []Step{normalizedStep(control.Step)},
		},
	}
	step := normalizedStep(control.Step)
	return ExecutionContext{
		Job:                     &job,
		Step:                    &step,
		Runtime:                 CloneJobRuntimeState(&runtime),
		GovernedExternalTargets: cloneAutonomyEligibilityTargetRefs(step.GovernedExternalTargets),
	}, nil
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
		if err := validateRuntimeStepReplayMarkers(next, stepID, "activate"); err != nil {
			return JobRuntimeState{}, err
		}
		next.ActiveStepID = stepID
		next.ActiveStepAt = now
		next.UpdatedAt = now
		next.WaitingReason = ""
		next.PausedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		return next, nil
	case JobStateWaitingUser, JobStatePaused:
		if err := validateRuntimeStepReplayMarkers(next, stepID, "activate"); err != nil {
			return JobRuntimeState{}, err
		}
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

func HasCompletedRuntimeStep(runtime JobRuntimeState, stepID string) bool {
	for _, completed := range runtime.CompletedSteps {
		if completed.StepID == stepID {
			return true
		}
	}
	return false
}

func HasFailedRuntimeStep(runtime JobRuntimeState, stepID string) bool {
	for _, failed := range runtime.FailedSteps {
		if failed.StepID == stepID {
			return true
		}
	}
	return false
}

func validateRuntimeStepReplayMarkers(current JobRuntimeState, stepID string, action string) error {
	if strings.TrimSpace(stepID) == "" {
		return nil
	}
	if HasCompletedRuntimeStep(current, stepID) {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  stepID,
			Message: fmt.Sprintf("cannot %s step %q because it is already recorded as completed", action, stepID),
		}
	}
	if HasFailedRuntimeStep(current, stepID) {
		return ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			StepID:  stepID,
			Message: fmt.Sprintf("cannot %s step %q because it is already recorded as failed", action, stepID),
		}
	}
	return nil
}

func validateRuntimeActiveStepReplayMarkers(current JobRuntimeState, action string) error {
	if err := validateRuntimeStepReplayMarkers(current, current.ActiveStepID, action); err != nil {
		validationErr := err.(ValidationError)
		validationErr.Message = strings.Replace(validationErr.Message, "cannot "+action+" step", "cannot "+action+" runtime for step", 1)
		return validationErr
	}
	return nil
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
	if err := validateRuntimeActiveStepReplayMarkers(current, "resume"); err != nil {
		return JobRuntimeState{}, err
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
		next.BudgetBlocker = nil
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

func PauseJobRuntime(current JobRuntimeState, now time.Time) (JobRuntimeState, error) {
	if current.State != JobStateRunning {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("pause requires running runtime state, got %q", current.State),
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "pause requires an active step",
		}
	}
	return TransitionJobRuntime(current, JobStatePaused, now, RuntimeTransitionOptions{
		StepID:       current.ActiveStepID,
		PausedReason: RuntimePauseReasonOperatorCommand,
	})
}

func PauseJobRuntimeForBudgetExhaustion(current JobRuntimeState, now time.Time, blocker RuntimeBudgetBlockerRecord) (JobRuntimeState, error) {
	if current.JobID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "budget exhaustion requires a runtime job ID",
		}
	}

	blocker.Ceiling = strings.TrimSpace(blocker.Ceiling)
	blocker.Message = strings.TrimSpace(blocker.Message)
	if blocker.Ceiling == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "budget exhaustion requires a budget ceiling name",
		}
	}
	if blocker.TriggeredAt.IsZero() {
		blocker.TriggeredAt = now
	}

	var next JobRuntimeState
	switch current.State {
	case JobStateRunning, JobStateWaitingUser:
		var err error
		next, err = TransitionJobRuntime(current, JobStatePaused, now, RuntimeTransitionOptions{
			StepID:       current.ActiveStepID,
			PausedReason: RuntimePauseReasonBudgetExhausted,
		})
		if err != nil {
			return JobRuntimeState{}, err
		}
	case JobStatePaused:
		next = *CloneJobRuntimeState(&current)
		next.State = JobStatePaused
		next.UpdatedAt = now
		if next.CreatedAt.IsZero() {
			next.CreatedAt = now
		}
		next.PausedReason = RuntimePauseReasonBudgetExhausted
		next.PausedAt = now
		next.WaitingReason = ""
		next.AbortedReason = ""
		next.WaitingAt = time.Time{}
		next.AbortedAt = time.Time{}
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("budget exhaustion requires running, waiting_user, or paused runtime state, got %q", current.State),
		}
	}

	next.BudgetBlocker = cloneRuntimeBudgetBlockerRecord(&blocker)
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    "budget_exhausted",
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Reason:      blocker.Message,
		Timestamp:   blocker.TriggeredAt,
	})
	return next, nil
}

func PauseJobRuntimeForUnattendedWallClock(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	anchor := current.StartedAt
	if anchor.IsZero() {
		anchor = current.CreatedAt
	}
	if anchor.IsZero() {
		return current, false, nil
	}

	elapsed := now.Sub(anchor)
	if elapsed < maxUnattendedWallClockPerJob {
		return current, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(current, now, RuntimeBudgetBlockerRecord{
		Ceiling:  unattendedWallClockBudgetCeiling,
		Limit:    maxUnattendedWallClockPerJobInMinutes,
		Observed: int(elapsed / time.Minute),
		Message:  "unattended wall-clock budget exhausted",
	})
	return paused, true, err
}

func RecordFailedToolAction(current JobRuntimeState, now time.Time, toolName string, reason string) (JobRuntimeState, bool, error) {
	if current.State != JobStateRunning {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("failed action budget requires running runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "failed action budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "failed action budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    strings.TrimSpace(toolName),
		ActionClass: AuditActionClassToolCall,
		Result:      AuditResultRejected,
		Allowed:     true,
		Reason:      strings.TrimSpace(reason),
		Timestamp:   now,
	})

	observed := countFailedToolActions(next)
	if observed < maxFailedActionsBeforePause {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  failedActionsBudgetCeiling,
		Limit:    maxFailedActionsBeforePause,
		Observed: observed,
		Message:  "failed action budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingMessage(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingMessageAction, AuditActionClassToolCall)
}

func RecordOwnerFacingCheckIn(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingCheckInAction, AuditActionClassRuntime)
}

func RecordOwnerFacingDailySummary(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingDailySummaryAction, AuditActionClassRuntime)
}

func RecordOwnerFacingApprovalRequest(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStateWaitingUser {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("approval request notification budget requires waiting_user runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "approval request notification budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "approval request notification budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    ownerFacingApprovalRequestAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingCompletion(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStateCompleted {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("completion notification budget requires completed runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "completion notification budget requires a runtime job ID",
		}
	}
	if len(current.CompletedSteps) == 0 {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "completion notification budget requires at least one completed step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.CompletedSteps[len(next.CompletedSteps)-1].StepID,
		ToolName:    ownerFacingCompletionAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	_ = observed
	return next, false, nil
}

func RecordOwnerFacingWaitingUser(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStateWaitingUser {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("waiting_user notification budget requires waiting_user runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "waiting_user notification budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "waiting_user notification budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    ownerFacingWaitingUserAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingBudgetPause(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStatePaused {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("budget pause notification requires paused runtime state, got %q", current.State),
		}
	}
	if current.PausedReason != RuntimePauseReasonBudgetExhausted {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("budget pause notification requires budget-exhausted pause, got %q", current.PausedReason),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "budget pause notification requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "budget pause notification requires an active step",
		}
	}
	if current.BudgetBlocker == nil {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "budget pause notification requires a budget blocker",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    ownerFacingBudgetPauseAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingDenyAck(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStateWaitingUser {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("deny acknowledgement budget requires waiting_user runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "deny acknowledgement budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "deny acknowledgement budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    ownerFacingDenyAckAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingPauseAck(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	if current.State != JobStatePaused {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("pause acknowledgement budget requires paused runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "pause acknowledgement budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "pause acknowledgement budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    ownerFacingPauseAckAction,
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func RecordOwnerFacingRevokeApprovalAck(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingRevokeAckAction, AuditActionClassRuntime)
}

func RecordOwnerFacingStepOutput(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingStepOutputAction, AuditActionClassRuntime)
}

func RecordOwnerFacingResumeAck(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingResumeAckAction, AuditActionClassRuntime)
}

func RecordOwnerFacingSetStepAck(current JobRuntimeState, now time.Time) (JobRuntimeState, bool, error) {
	return recordOwnerFacingOutput(current, now, ownerFacingSetStepAckAction, AuditActionClassRuntime)
}

func recordOwnerFacingOutput(current JobRuntimeState, now time.Time, action string, actionClass AuditActionClass) (JobRuntimeState, bool, error) {
	if current.State != JobStateRunning {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("owner-facing message budget requires running runtime state, got %q", current.State),
		}
	}
	if current.JobID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "owner-facing message budget requires a runtime job ID",
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, false, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "owner-facing message budget requires an active step",
		}
	}

	next := *CloneJobRuntimeState(&current)
	next.UpdatedAt = now
	next.AuditHistory = AppendAuditHistory(next.AuditHistory, AuditEvent{
		JobID:       next.JobID,
		StepID:      next.ActiveStepID,
		ToolName:    action,
		ActionClass: actionClass,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now,
	})

	observed := countOwnerFacingMessages(next)
	if observed < maxOwnerFacingMessagesPerJob {
		return next, false, nil
	}

	paused, err := PauseJobRuntimeForBudgetExhaustion(next, now, RuntimeBudgetBlockerRecord{
		Ceiling:  ownerMessagesBudgetCeiling,
		Limit:    maxOwnerFacingMessagesPerJob,
		Observed: observed,
		Message:  "owner-facing message budget exhausted",
	})
	return paused, true, err
}

func countFailedToolActions(runtime JobRuntimeState) int {
	count := 0
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if event.ActionClass != AuditActionClassToolCall || !event.Allowed || event.Result != AuditResultRejected {
			continue
		}
		count++
	}
	return count
}

func countOwnerFacingMessages(runtime JobRuntimeState) int {
	count := 0
	for _, event := range runtime.AuditHistory {
		if runtime.JobID != "" && event.JobID != runtime.JobID {
			continue
		}
		if !event.Allowed || event.Result != AuditResultApplied {
			continue
		}
		switch {
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingApprovalRequestAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingWaitingUserAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingBudgetPauseAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingCompletionAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingDailySummaryAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingDenyAckAction:
		case event.ActionClass == AuditActionClassToolCall && event.ToolName == ownerFacingMessageAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingCheckInAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingPauseAckAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingRevokeAckAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingStepOutputAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingResumeAckAction:
		case event.ActionClass == AuditActionClassRuntime && event.ToolName == ownerFacingSetStepAckAction:
		default:
			continue
		}
		count++
	}
	return count
}

func ResumePausedJobRuntime(current JobRuntimeState, now time.Time) (JobRuntimeState, error) {
	if current.State != JobStatePaused {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("resume requires paused runtime state, got %q", current.State),
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "resume requires a paused active step",
		}
	}
	if err := validateRuntimeActiveStepReplayMarkers(current, "resume"); err != nil {
		return JobRuntimeState{}, err
	}
	return TransitionJobRuntime(current, JobStateRunning, now, RuntimeTransitionOptions{
		StepID: current.ActiveStepID,
	})
}

func AbortJobRuntime(current JobRuntimeState, now time.Time) (JobRuntimeState, error) {
	switch current.State {
	case JobStateRunning, JobStateWaitingUser, JobStatePaused:
	default:
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("abort requires running, waiting_user, or paused runtime state, got %q", current.State),
		}
	}
	if current.ActiveStepID == "" {
		return JobRuntimeState{}, ValidationError{
			Code:    RejectionCodeInvalidRuntimeState,
			Message: "abort requires an active step",
		}
	}
	return TransitionJobRuntime(current, JobStateAborted, now, RuntimeTransitionOptions{
		AbortedReason: RuntimeAbortReasonOperatorCommand,
	})
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
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		next.AbortedAt = time.Time{}
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
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.PausedAt = time.Time{}
		next.AbortedAt = time.Time{}
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
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.AbortedAt = time.Time{}
		if opts.validationResult != nil && opts.validationResult.recordCompletion {
			if err := validateRuntimeActiveStepReplayMarkers(next, "complete"); err != nil {
				return JobRuntimeState{}, err
			}
			if next.ActiveStepID == "" {
				return JobRuntimeState{}, ValidationError{
					Code:    RejectionCodeInvalidRuntimeState,
					Message: "paused state requires an active step to record completion",
				}
			}
			next.CompletedSteps = append(next.CompletedSteps, RuntimeStepRecord{
				StepID:         next.ActiveStepID,
				At:             now,
				ResultingState: cloneRuntimeResultingStateRecord(opts.validationResult.resultingState),
				Rollback:       cloneRuntimeRollbackRecord(opts.validationResult.rollback),
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
		if err := validateRuntimeActiveStepReplayMarkers(next, "complete"); err != nil {
			return JobRuntimeState{}, err
		}
		if next.ActiveStepID == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "completed state requires an active step to record",
			}
		}
		next.CompletedSteps = append(next.CompletedSteps, RuntimeStepRecord{
			StepID:         next.ActiveStepID,
			At:             now,
			ResultingState: cloneRuntimeResultingStateRecord(opts.validationResult.resultingState),
			Rollback:       cloneRuntimeRollbackRecord(opts.validationResult.rollback),
		})
		next.CompletedAt = now
		next.ActiveStepID = ""
		next.ActiveStepAt = time.Time{}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		next.AbortedAt = time.Time{}
	case JobStateFailed:
		if err := validateRuntimeActiveStepReplayMarkers(next, "fail"); err != nil {
			return JobRuntimeState{}, err
		}
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
		next.ActiveStepAt = time.Time{}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		next.AbortedAt = time.Time{}
	case JobStateAborted:
		if opts.AbortedReason == "" {
			return JobRuntimeState{}, ValidationError{
				Code:    RejectionCodeInvalidRuntimeState,
				Message: "aborted state requires an abort reason",
			}
		}
		next.AbortedReason = opts.AbortedReason
		next.AbortedAt = now
		next.ActiveStepID = ""
		next.ActiveStepAt = time.Time{}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
	case JobStateRejected:
		next.ActiveStepID = ""
		next.ActiveStepAt = time.Time{}
		next.WaitingReason = ""
		next.PausedReason = ""
		next.AbortedReason = ""
		next.BudgetBlocker = nil
		next.WaitingAt = time.Time{}
		next.PausedAt = time.Time{}
		next.AbortedAt = time.Time{}
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
	if err := validateRuntimeActiveStepReplayMarkers(*ec.Runtime, "execute"); err != nil {
		return err
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
