package tools

import (
	"fmt"
	"sync"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

// TaskState tracks per-message execution state shared across tools.
// The main use right now is enforcing that a new deliverable task must
// initialize projects/current via frank_new_project before writing there.
type TaskState struct {
	mu                  sync.Mutex
	currentTaskID       string
	projectInitialized  bool
	executionContext    missioncontrol.ExecutionContext
	hasExecutionContext bool
	missionJob          missioncontrol.Job
	hasMissionJob       bool
	runtimeControl      missioncontrol.RuntimeControlContext
	hasRuntimeControl   bool
	runtimeState        missioncontrol.JobRuntimeState
	hasRuntimeState     bool
	operatorChannel     string
	operatorChatID      string
	auditEvents         []missioncontrol.AuditEvent
	runtimeChangeHook   func()
}

func NewTaskState() *TaskState {
	return &TaskState{}
}

func (s *TaskState) BeginTask(taskID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentTaskID != taskID {
		s.currentTaskID = taskID
		s.projectInitialized = false
	}
}

func (s *TaskState) MarkProjectInitialized() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projectInitialized = true
}

func (s *TaskState) ProjectInitialized() bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.projectInitialized
}

func (s *TaskState) SetExecutionContext(ec missioncontrol.ExecutionContext) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := missioncontrol.CloneExecutionContext(ec)
	s.executionContext = cloned
	s.hasExecutionContext = true
	s.storeMissionJobLocked(cloned.Job)
	if cloned.Job != nil && cloned.Step != nil {
		control, err := missioncontrol.BuildRuntimeControlContext(*cloned.Job, cloned.Step.ID)
		if err == nil {
			s.runtimeControl = control
			s.hasRuntimeControl = true
		}
	} else {
		s.runtimeControl = missioncontrol.RuntimeControlContext{}
		s.hasRuntimeControl = false
	}
	if cloned.Runtime != nil {
		s.runtimeState = *missioncontrol.CloneJobRuntimeState(cloned.Runtime)
		s.hasRuntimeState = true
		s.auditEvents = missioncontrol.CloneAuditHistory(s.runtimeState.AuditHistory)
	} else {
		s.runtimeState = missioncontrol.JobRuntimeState{}
		s.hasRuntimeState = false
		s.auditEvents = nil
	}
}

func (s *TaskState) ExecutionContext() (missioncontrol.ExecutionContext, bool) {
	if s == nil {
		return missioncontrol.ExecutionContext{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return missioncontrol.CloneExecutionContext(s.executionContext), s.hasExecutionContext
}

func (s *TaskState) MissionRuntimeState() (missioncontrol.JobRuntimeState, bool) {
	if s == nil {
		return missioncontrol.JobRuntimeState{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasRuntimeState {
		return missioncontrol.JobRuntimeState{}, false
	}
	return *missioncontrol.CloneJobRuntimeState(&s.runtimeState), true
}

func (s *TaskState) MissionRuntimeControl() (missioncontrol.RuntimeControlContext, bool) {
	if s == nil {
		return missioncontrol.RuntimeControlContext{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasRuntimeControl {
		return missioncontrol.RuntimeControlContext{}, false
	}
	return *missioncontrol.CloneRuntimeControlContext(&s.runtimeControl), true
}

func (s *TaskState) SetRuntimeChangeHook(hook func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeChangeHook = hook
}

func (s *TaskState) SetOperatorSession(channel string, chatID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operatorChannel = channel
	s.operatorChatID = chatID
}

func (s *TaskState) EmitAuditEvent(event missioncontrol.AuditEvent) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.auditEvents = missioncontrol.AppendAuditHistory(s.auditEvents, event)
	persisted := s.hasRuntimeState
	if s.hasRuntimeState {
		s.runtimeState.AuditHistory = missioncontrol.AppendAuditHistory(s.runtimeState.AuditHistory, event)
	}
	if s.hasExecutionContext && s.executionContext.Runtime != nil {
		s.executionContext.Runtime.AuditHistory = missioncontrol.AppendAuditHistory(s.executionContext.Runtime.AuditHistory, event)
	}
	s.mu.Unlock()
	if persisted {
		s.notifyRuntimeChanged()
	}
}

func (s *TaskState) AuditEvents() []missioncontrol.AuditEvent {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return missioncontrol.CloneAuditHistory(s.auditEvents)
}

func (s *TaskState) ActivateStep(job missioncontrol.Job, stepID string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	var current *missioncontrol.JobRuntimeState
	if s.hasRuntimeState {
		cloned := *missioncontrol.CloneJobRuntimeState(&s.runtimeState)
		current = &cloned
	}
	s.mu.Unlock()

	runtimeState, err := missioncontrol.SetJobRuntimeActiveStep(job, current, stepID, time.Now())
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(&job, runtimeState, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ResumeRuntime(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext, resumeApproved bool) error {
	now := time.Now()
	normalizedRuntime, _ := missioncontrol.NormalizeHydratedApprovalRequests(runtimeState, now)
	nextRuntime, err := missioncontrol.ResumeJobRuntimeAfterBoot(normalizedRuntime, now, resumeApproved)
	if err != nil {
		return err
	}

	if s == nil {
		return nil
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(&job, nextRuntime, control)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) HydrateRuntimeControl(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	if s == nil {
		return nil
	}
	if runtimeState.JobID != "" && runtimeState.JobID != job.ID {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("runtime job %q does not match mission job %q", runtimeState.JobID, job.ID),
		}
	}

	normalizedRuntime, changed := missioncontrol.NormalizeHydratedApprovalRequests(runtimeState, time.Now())

	s.mu.Lock()
	err := s.hydrateRuntimeControlLocked(job, normalizedRuntime, control)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if changed {
		s.notifyRuntimeChanged()
	}
	return nil
}

func (s *TaskState) ApplyStepOutput(finalContent string, successfulTools []missioncontrol.RuntimeToolCallEvidence) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	operatorChannel := s.operatorChannel
	operatorChatID := s.operatorChatID
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	if exhausted, err := s.EnforceUnattendedWallClockBudget(); err != nil {
		return err
	} else if exhausted {
		return nil
	}

	now := time.Now()
	nextRuntime, err := missioncontrol.CompleteRuntimeStep(ec, now, missioncontrol.StepValidationInput{
		FinalResponse:   finalContent,
		SessionChannel:  operatorChannel,
		SessionChatID:   operatorChatID,
		SuccessfulTools: successfulTools,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) EnforceUnattendedWallClockBudget() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.PauseJobRuntimeForUnattendedWallClock(*ec.Runtime, time.Now())
	if err != nil || !exhausted {
		return exhausted, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return true, err
}

func (s *TaskState) ApplyWaitingUserInput(input string) (missioncontrol.WaitingUserInputKind, error) {
	if s == nil {
		return missioncontrol.WaitingUserInputNone, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return missioncontrol.WaitingUserInputNone, nil
	}

	refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*ec.Runtime, time.Now())
	if changed {
		ec.Runtime = &refreshedRuntime
		s.mu.Lock()
		err := s.storeRuntimeStateLocked(ec.Job, refreshedRuntime, nil)
		s.mu.Unlock()
		if err != nil {
			return missioncontrol.WaitingUserInputNone, err
		}
		s.notifyRuntimeChanged()
	}

	inputKind := missioncontrol.ClassifyWaitingUserInput(input)
	if inputKind == missioncontrol.WaitingUserInputNone {
		return inputKind, nil
	}

	nextRuntime, err := missioncontrol.CompleteRuntimeStep(ec, time.Now(), missioncontrol.StepValidationInput{
		UserInput:     input,
		UserInputKind: inputKind,
	})
	if err != nil {
		return missioncontrol.WaitingUserInputNone, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err != nil {
		return missioncontrol.WaitingUserInputNone, err
	}
	s.notifyRuntimeChanged()
	return inputKind, nil
}

func (s *TaskState) ApplyNaturalApprovalDecision(input string) (bool, string, error) {
	if s == nil {
		return false, "", nil
	}

	decision, ok := missioncontrol.ParsePlainApprovalDecision(input)
	if !ok {
		return false, "", nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	s.mu.Unlock()

	if hasExecutionContext && ec.Runtime != nil {
		refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*ec.Runtime, time.Now())
		if changed {
			ec.Runtime = &refreshedRuntime
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(ec.Job, refreshedRuntime, nil)
			s.mu.Unlock()
			if err != nil {
				return true, "", err
			}
			s.notifyRuntimeChanged()
		}
		request, handled, err := resolveNaturalApprovalRequestFromExecutionContext(ec)
		if err != nil {
			return true, "", err
		}
		if !handled {
			return false, "", nil
		}
		if err := s.ApplyApprovalDecision(request.JobID, request.StepID, decision, missioncontrol.ApprovalGrantedViaOperatorReply); err != nil {
			return true, "", err
		}
		return true, naturalApprovalResponse(decision, request.JobID, request.StepID), nil
	}

	if !hasRuntimeState || runtimeState == nil {
		return false, "", nil
	}

	refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*runtimeState, time.Now())
	if changed {
		if hasRuntimeControl && control != nil {
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(nil, refreshedRuntime, control)
			s.mu.Unlock()
			if err != nil {
				return true, "", err
			}
		} else {
			s.mu.Lock()
			s.runtimeState = refreshedRuntime
			s.hasRuntimeState = true
			s.mu.Unlock()
		}
		s.notifyRuntimeChanged()
		runtimeState = &refreshedRuntime
	}

	request, handled, err := resolveNaturalApprovalRequestFromPersistedRuntime(runtimeState, control, hasRuntimeControl)
	if err != nil {
		return true, "", err
	}
	if !handled {
		return false, "", nil
	}
	if err := s.ApplyApprovalDecision(request.JobID, request.StepID, decision, missioncontrol.ApprovalGrantedViaOperatorReply); err != nil {
		return true, "", err
	}
	return true, naturalApprovalResponse(decision, request.JobID, request.StepID), nil
}

func (s *TaskState) ApplyApprovalDecision(jobID string, stepID string, decision missioncontrol.ApprovalDecision, via string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	operatorChannel := s.operatorChannel
	operatorChatID := s.operatorChatID
	s.mu.Unlock()

	storeJob := ec.Job
	storeControl := (*missioncontrol.RuntimeControlContext)(nil)
	rebootSafePath := false
	if hasExecutionContext && ec.Job != nil && ec.Step != nil && ec.Runtime != nil {
		refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*ec.Runtime, time.Now())
		if changed {
			ec.Runtime = &refreshedRuntime
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(ec.Job, refreshedRuntime, nil)
			s.mu.Unlock()
			if err != nil {
				return err
			}
			s.notifyRuntimeChanged()
		}
		if ec.Job.ID != jobID || ec.Step.ID != stepID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "approval command requires an active mission step",
			}
		}
		if runtimeState.JobID != jobID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
		if runtimeState.State != missioncontrol.JobStateWaitingUser {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  stepID,
				Message: fmt.Sprintf("approval decision requires waiting_user runtime state, got %q", runtimeState.State),
			}
		}
		if runtimeState.ActiveStepID == "" {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  stepID,
				Message: "approval decision requires an active step",
			}
		}
		if !hasRuntimeControl || control == nil {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  stepID,
				Message: "approval command requires persisted mission control context",
			}
		}
		if control.JobID != "" && control.JobID != jobID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
		if control.Step.ID != stepID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}

		refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*runtimeState, time.Now())
		if changed {
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(nil, refreshedRuntime, control)
			s.mu.Unlock()
			if err != nil {
				return err
			}
			s.notifyRuntimeChanged()
			runtimeState = &refreshedRuntime
		}

		resolved, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, *runtimeState)
		if err != nil {
			return err
		}
		ec = resolved
		storeControl = control
		rebootSafePath = true
	}

	nextRuntime, err := missioncontrol.ApplyApprovalDecisionWithSession(ec, time.Now(), decision, via, operatorChannel, operatorChatID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if rebootSafePath && nextRuntime.ActiveStepID != "" {
		err = s.hydrateRuntimeControlLocked(missioncontrol.Job{ID: jobID}, nextRuntime, storeControl)
	} else {
		err = s.storeRuntimeStateLocked(storeJob, nextRuntime, storeControl)
	}
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) RevokeApproval(jobID string, stepID string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	operatorChannel := s.operatorChannel
	operatorChatID := s.operatorChatID
	s.mu.Unlock()

	storeJob := ec.Job
	storeControl := (*missioncontrol.RuntimeControlContext)(nil)
	rebootSafePath := false
	if hasExecutionContext && ec.Job != nil && ec.Step != nil && ec.Runtime != nil {
		refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*ec.Runtime, time.Now())
		if changed {
			ec.Runtime = &refreshedRuntime
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(ec.Job, refreshedRuntime, nil)
			s.mu.Unlock()
			if err != nil {
				return err
			}
			s.notifyRuntimeChanged()
		}
		if ec.Job.ID != jobID || ec.Step.ID != stepID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "approval command requires an active mission step",
			}
		}
		if runtimeState.JobID != jobID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
		if runtimeState.ActiveStepID == "" {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  stepID,
				Message: "approval revocation requires an active step",
			}
		}
		if !hasRuntimeControl || control == nil {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				StepID:  stepID,
				Message: "approval command requires persisted mission control context",
			}
		}
		if control.JobID != "" && control.JobID != jobID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}
		if control.Step.ID != stepID {
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				StepID:  stepID,
				Message: "approval command does not match the active job and step",
			}
		}

		refreshedRuntime, changed := missioncontrol.RefreshApprovalRequests(*runtimeState, time.Now())
		if changed {
			s.mu.Lock()
			err := s.storeRuntimeStateLocked(nil, refreshedRuntime, control)
			s.mu.Unlock()
			if err != nil {
				return err
			}
			s.notifyRuntimeChanged()
			runtimeState = &refreshedRuntime
		}

		resolved, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, *runtimeState)
		if err != nil {
			return err
		}
		ec = resolved
		storeControl = control
		rebootSafePath = true
	}

	nextRuntime, err := missioncontrol.RevokeLatestApprovalGrantWithSession(ec, time.Now(), operatorChannel, operatorChatID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if rebootSafePath && nextRuntime.ActiveStepID != "" {
		err = s.hydrateRuntimeControlLocked(missioncontrol.Job{ID: jobID}, nextRuntime, storeControl)
	} else {
		err = s.storeRuntimeStateLocked(storeJob, nextRuntime, storeControl)
	}
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) PauseRuntime(jobID string) error {
	return s.applyRuntimeControl(jobID, "pause", missioncontrol.PauseJobRuntime, false)
}

func (s *TaskState) ResumeRuntimeControl(jobID string) error {
	return s.applyRuntimeControl(jobID, "resume", missioncontrol.ResumePausedJobRuntime, true)
}

func (s *TaskState) AbortRuntime(jobID string) error {
	return s.applyRuntimeControl(jobID, "abort", missioncontrol.AbortJobRuntime, true)
}

func (s *TaskState) OperatorStatus(jobID string) (string, error) {
	if s == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	s.mu.Unlock()

	if hasExecutionContext && ec.Runtime != nil {
		if ec.Job != nil && ec.Job.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		if ec.Runtime.JobID != "" && ec.Runtime.JobID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		return missioncontrol.FormatOperatorStatusSummaryWithAllowedTools(*ec.Runtime, missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step))
	}

	if !hasRuntimeState || runtimeState == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if runtimeState.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}

	var allowedTools []string
	if hasRuntimeControl && control != nil {
		allowedTools = missioncontrol.EffectiveAllowedTools(&missioncontrol.Job{AllowedTools: append([]string(nil), control.AllowedTools...)}, &control.Step)
	}
	return missioncontrol.FormatOperatorStatusSummaryWithAllowedTools(*runtimeState, allowedTools)
}

func (s *TaskState) OperatorInspect(jobID string, stepID string) (string, error) {
	if s == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	s.mu.Unlock()

	if hasExecutionContext && ec.Runtime != nil {
		if ec.Job == nil {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
		}
		if ec.Job.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		if ec.Runtime.JobID != "" && ec.Runtime.JobID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}

		summary, err := missioncontrol.NewInspectSummary(*ec.Job, stepID)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}

	if !hasRuntimeState || runtimeState == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
	}
	if runtimeState.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}
	if hasMissionJob && missionJob != nil {
		if missionJob.ID != "" && missionJob.ID != jobID {
			return "", missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
		}
		summary, err := missioncontrol.NewInspectSummary(*missionJob, stepID)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}
	if runtimeState.InspectablePlan != nil {
		summary, err := missioncontrol.NewInspectSummaryFromInspectablePlan(runtimeState.JobID, runtimeState.InspectablePlan, stepID)
		if err != nil {
			return "", err
		}
		return missioncontrol.FormatInspectSummary(summary)
	}
	if !hasRuntimeControl || control == nil {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "inspect command requires validated mission plan",
		}
	}
	if control.JobID != "" && control.JobID != jobID {
		return "", missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
	}

	return "", missioncontrol.ValidationError{
		Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
		Message: "inspect command requires validated mission plan",
	}
}

func resolveNaturalApprovalRequestFromExecutionContext(ec missioncontrol.ExecutionContext) (missioncontrol.ApprovalRequest, bool, error) {
	if ec.Runtime == nil {
		return missioncontrol.ApprovalRequest{}, false, nil
	}
	if missioncontrol.IsTerminalJobState(ec.Runtime.State) {
		return missioncontrol.ApprovalRequest{}, true, approvalDecisionStateError(ec.Runtime.ActiveStepID, ec.Runtime.State)
	}
	if ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return missioncontrol.ApprovalRequest{}, false, nil
	}
	if ec.Job == nil || ec.Step == nil {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "approval decision requires active job and step",
		}
	}

	request, handled, err := missioncontrol.ResolveSinglePendingApprovalRequest(*ec.Runtime)
	if err != nil || !handled {
		return request, handled, err
	}
	if !missioncontrol.ApprovalRequestMatchesStepBinding(request, ec.Job.ID, *ec.Step) {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  request.StepID,
			Message: "approval decision does not match the active job and step",
		}
	}
	return request, true, nil
}

func resolveNaturalApprovalRequestFromPersistedRuntime(runtimeState *missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext, hasRuntimeControl bool) (missioncontrol.ApprovalRequest, bool, error) {
	if runtimeState == nil {
		return missioncontrol.ApprovalRequest{}, false, nil
	}
	if missioncontrol.IsTerminalJobState(runtimeState.State) {
		return missioncontrol.ApprovalRequest{}, true, approvalDecisionStateError(runtimeState.ActiveStepID, runtimeState.State)
	}
	if runtimeState.State != missioncontrol.JobStateWaitingUser {
		return missioncontrol.ApprovalRequest{}, false, nil
	}

	request, handled, err := missioncontrol.ResolveSinglePendingApprovalRequest(*runtimeState)
	if err != nil || !handled {
		return request, handled, err
	}
	if runtimeState.ActiveStepID == "" {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			StepID:  request.StepID,
			Message: "approval decision requires an active step",
		}
	}
	if !hasRuntimeControl || control == nil {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			StepID:  request.StepID,
			Message: "approval decision requires persisted mission control context",
		}
	}
	if runtimeState.JobID != "" && request.JobID != runtimeState.JobID {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  request.StepID,
			Message: "approval decision does not match the active job and step",
		}
	}
	if control.JobID != "" && request.JobID != control.JobID {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  request.StepID,
			Message: "approval decision does not match the active job and step",
		}
	}
	if request.StepID != runtimeState.ActiveStepID || request.StepID != control.Step.ID {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  request.StepID,
			Message: "approval decision does not match the active job and step",
		}
	}
	if !missioncontrol.ApprovalRequestMatchesStepBinding(request, control.JobID, control.Step) {
		return missioncontrol.ApprovalRequest{}, true, missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			StepID:  request.StepID,
			Message: "approval decision does not match the active job and step",
		}
	}
	return request, true, nil
}

func approvalDecisionStateError(stepID string, state missioncontrol.JobState) error {
	return missioncontrol.ValidationError{
		Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
		StepID:  stepID,
		Message: fmt.Sprintf("approval decision requires waiting_user runtime state, got %q", state),
	}
}

func naturalApprovalResponse(decision missioncontrol.ApprovalDecision, jobID string, stepID string) string {
	if decision == missioncontrol.ApprovalDecisionDeny {
		return fmt.Sprintf("Denied job=%s step=%s.", jobID, stepID)
	}
	return fmt.Sprintf("Approved job=%s step=%s.", jobID, stepID)
}

func (s *TaskState) ClearExecutionContext() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.executionContext = missioncontrol.ExecutionContext{}
	s.hasExecutionContext = false
	if s.hasRuntimeState {
		s.auditEvents = missioncontrol.CloneAuditHistory(s.runtimeState.AuditHistory)
	} else {
		s.auditEvents = nil
	}
	s.mu.Unlock()
	s.notifyRuntimeChanged()
}

func (s *TaskState) storeRuntimeStateLocked(job *missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	storedRuntime := *missioncontrol.CloneJobRuntimeState(&runtimeState)
	if job != nil && storedRuntime.InspectablePlan == nil {
		inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(*job)
		if err != nil {
			return err
		}
		storedRuntime.InspectablePlan = &inspectablePlan
	}

	s.runtimeState = storedRuntime
	s.hasRuntimeState = true
	s.auditEvents = missioncontrol.CloneAuditHistory(storedRuntime.AuditHistory)

	if storedRuntime.ActiveStepID != "" {
		if control != nil {
			if _, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, storedRuntime); err != nil {
				return err
			}
			s.runtimeControl = *missioncontrol.CloneRuntimeControlContext(control)
			s.hasRuntimeControl = true
		} else {
			if job == nil {
				return missioncontrol.ValidationError{
					Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
					Message: "runtime control requires a mission job or persisted control context",
				}
			}
			builtControl, err := missioncontrol.BuildRuntimeControlContext(*job, runtimeState.ActiveStepID)
			if err != nil {
				return err
			}
			s.runtimeControl = builtControl
			s.hasRuntimeControl = true
		}
	} else {
		s.runtimeControl = missioncontrol.RuntimeControlContext{}
		s.hasRuntimeControl = false
	}

	if storedRuntime.ActiveStepID == "" {
		s.storeMissionJobLocked(job)
		s.executionContext = missioncontrol.ExecutionContext{}
		s.hasExecutionContext = false
		return nil
	}

	if control != nil {
		ec, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, storedRuntime)
		if err != nil {
			return err
		}
		s.storeMissionJobLocked(job)
		s.executionContext = ec
		s.hasExecutionContext = true
		return nil
	}
	if job == nil {
		return missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "runtime execution requires a mission job or persisted control context",
		}
	}
	ec, err := missioncontrol.ResolveExecutionContextWithRuntime(*job, storedRuntime)
	if err != nil {
		return err
	}
	s.storeMissionJobLocked(job)
	s.executionContext = ec
	s.hasExecutionContext = true
	return nil
}

func (s *TaskState) hydrateRuntimeControlLocked(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	s.runtimeState = *missioncontrol.CloneJobRuntimeState(&runtimeState)
	s.hasRuntimeState = true
	s.auditEvents = missioncontrol.CloneAuditHistory(s.runtimeState.AuditHistory)
	s.executionContext = missioncontrol.ExecutionContext{}
	s.hasExecutionContext = false
	s.runtimeControl = missioncontrol.RuntimeControlContext{}
	s.hasRuntimeControl = false

	if runtimeState.ActiveStepID == "" {
		switch runtimeState.State {
		case missioncontrol.JobStateCompleted, missioncontrol.JobStateFailed, missioncontrol.JobStateRejected, missioncontrol.JobStateAborted:
			s.storeMissionJobLocked(&job)
			return nil
		default:
			return missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "persisted runtime requires an active step",
			}
		}
	}

	if control != nil {
		if _, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, runtimeState); err != nil {
			return err
		}
		s.storeMissionJobLocked(&job)
		s.runtimeControl = *missioncontrol.CloneRuntimeControlContext(control)
		s.hasRuntimeControl = true
		return nil
	}

	builtControl, err := missioncontrol.BuildRuntimeControlContext(job, runtimeState.ActiveStepID)
	if err != nil {
		return err
	}
	s.storeMissionJobLocked(&job)
	s.runtimeControl = builtControl
	s.hasRuntimeControl = true
	return nil
}

func (s *TaskState) notifyRuntimeChanged() {
	if s == nil {
		return
	}
	s.mu.Lock()
	hook := s.runtimeChangeHook
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
}

func (s *TaskState) storeMissionJobLocked(job *missioncontrol.Job) {
	if job == nil {
		return
	}
	s.missionJob = *missioncontrol.CloneJob(job)
	s.hasMissionJob = true
}

func (s *TaskState) applyRuntimeControl(jobID string, action string, apply func(missioncontrol.JobRuntimeState, time.Time) (missioncontrol.JobRuntimeState, error), allowWithoutExecutionContext bool) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	s.mu.Unlock()

	if !hasExecutionContext {
		if !allowWithoutExecutionContext {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(ec, action, err)
			return err
		}
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(ec, action, err)
			return err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(control, runtimeState), action, err)
			return err
		}
		if runtimeState.ActiveStepID != "" && !hasRuntimeControl {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(control, runtimeState), action, err)
			return err
		}

		nextRuntime, err := apply(*runtimeState, time.Now())
		if err != nil {
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(control, runtimeState), action, err)
			return err
		}

		s.mu.Lock()
		if nextRuntime.State == missioncontrol.JobStateRunning {
			if control == nil {
				err = missioncontrol.ValidationError{
					Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
					Message: "operator command requires persisted mission control context",
				}
			} else {
				resolved, resolveErr := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, nextRuntime)
				if resolveErr != nil {
					err = resolveErr
				} else {
					s.runtimeState = nextRuntime
					s.hasRuntimeState = true
					s.runtimeControl = *missioncontrol.CloneRuntimeControlContext(control)
					s.hasRuntimeControl = true
					s.executionContext = resolved
					s.hasExecutionContext = true
					err = nil
				}
			}
		} else {
			err = s.hydrateRuntimeControlLocked(missioncontrol.Job{ID: jobID}, nextRuntime, nil)
		}
		s.mu.Unlock()
		if err != nil {
			s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(control, runtimeState), action, err)
			return err
		}

		s.emitRuntimeControlAuditEvent(s.runtimeAuditContext(control, &nextRuntime), action, nil)
		s.notifyRuntimeChanged()
		return nil
	}

	if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "operator command requires an active mission step",
		}
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}
	if ec.Job.ID != jobID {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "operator command does not match the active job",
		}
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	nextRuntime, err := apply(*missioncontrol.CloneJobRuntimeState(ec.Runtime), time.Now())
	if err != nil {
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err != nil {
		s.emitRuntimeControlAuditEvent(ec, action, err)
		return err
	}

	s.emitRuntimeControlAuditEvent(ec, action, nil)
	s.notifyRuntimeChanged()
	return nil
}

func (s *TaskState) runtimeAuditContext(control *missioncontrol.RuntimeControlContext, runtime *missioncontrol.JobRuntimeState) missioncontrol.ExecutionContext {
	if runtime == nil {
		return missioncontrol.ExecutionContext{}
	}

	ec := missioncontrol.ExecutionContext{
		Runtime: missioncontrol.CloneJobRuntimeState(runtime),
	}
	if control == nil {
		return ec
	}
	if runtime.ActiveStepID != "" {
		if resolved, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, *runtime); err == nil {
			return resolved
		}
	}
	job := missioncontrol.Job{
		ID:           control.JobID,
		MaxAuthority: control.MaxAuthority,
		AllowedTools: append([]string(nil), control.AllowedTools...),
	}
	ec.Job = &job
	if control.Step.ID != "" {
		step := control.Step
		ec.Step = &step
	}
	return ec
}

func (s *TaskState) emitRuntimeControlAuditEvent(ec missioncontrol.ExecutionContext, action string, err error) {
	if s == nil {
		return
	}

	event := missioncontrol.AuditEvent{
		ToolName:  action,
		Allowed:   err == nil,
		Timestamp: time.Now(),
	}
	if ec.Job != nil {
		event.JobID = ec.Job.ID
	}
	if ec.Step != nil {
		event.StepID = ec.Step.ID
	}
	if err != nil {
		event.Reason = err.Error()
		if validationErr, ok := err.(missioncontrol.ValidationError); ok {
			event.Code = validationErr.Code
		} else {
			event.Code = missioncontrol.RejectionCodeInvalidRuntimeState
		}
	}

	s.EmitAuditEvent(event)
}
