package tools

import (
	"fmt"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

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
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
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
		pausedPendingApprovalBudget := runtimeState.State == missioncontrol.JobStatePaused &&
			runtimeState.PausedReason == missioncontrol.RuntimePauseReasonBudgetExhausted &&
			runtimeState.BudgetBlocker != nil &&
			runtimeState.BudgetBlocker.Ceiling == "pending_approvals"
		if runtimeState.State != missioncontrol.JobStateWaitingUser && !pausedPendingApprovalBudget {
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

	hydrateJob := missioncontrol.Job{ID: jobID}
	if hasMissionJob && missionJob != nil && missionJob.ID == jobID {
		hydrateJob = *missioncontrol.CloneJob(missionJob)
	}

	s.mu.Lock()
	if rebootSafePath && nextRuntime.ActiveStepID != "" {
		err = s.persistHydratedRuntimeStateLocked(hydrateJob, nextRuntime, storeControl)
	} else {
		liveJob := storeJob
		if liveJob == nil && hasMissionJob && missionJob != nil && missionJob.ID == jobID {
			liveJob = missioncontrol.CloneJob(missionJob)
		}
		err = s.storeRuntimeStateLocked(liveJob, nextRuntime, storeControl)
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
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
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

	hydrateJob := missioncontrol.Job{ID: jobID}
	if hasMissionJob && missionJob != nil && missionJob.ID == jobID {
		hydrateJob = *missioncontrol.CloneJob(missionJob)
	}

	s.mu.Lock()
	if rebootSafePath && nextRuntime.ActiveStepID != "" {
		err = s.persistHydratedRuntimeStateLocked(hydrateJob, nextRuntime, storeControl)
	} else {
		liveJob := storeJob
		if liveJob == nil && hasMissionJob && missionJob != nil && missionJob.ID == jobID {
			liveJob = missioncontrol.CloneJob(missionJob)
		}
		err = s.storeRuntimeStateLocked(liveJob, nextRuntime, storeControl)
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
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
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

		hydrateJob := missioncontrol.Job{ID: jobID}
		if hasMissionJob && missionJob != nil && missionJob.ID == jobID {
			hydrateJob = *missioncontrol.CloneJob(missionJob)
		}

		s.mu.Lock()
		if nextRuntime.State == missioncontrol.JobStateRunning {
			liveJob := (*missioncontrol.Job)(nil)
			if hasMissionJob && missionJob != nil && missionJob.ID == jobID {
				liveJob = missioncontrol.CloneJob(missionJob)
			}
			err = s.storeRuntimeStateLocked(liveJob, nextRuntime, control)
		} else {
			err = s.persistHydratedRuntimeStateLocked(hydrateJob, nextRuntime, control)
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

	ec := s.withMissionStoreRootLocked(missioncontrol.ExecutionContext{
		Runtime: missioncontrol.CloneJobRuntimeState(runtime),
	})
	if control == nil {
		return ec
	}
	if runtime.ActiveStepID != "" {
		if resolved, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*control, *runtime); err == nil {
			return s.withMissionStoreRootLocked(resolved)
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
		step.IdentityMode = missioncontrol.NormalizeIdentityMode(step.IdentityMode)
		ec.Step = &step
	}
	return s.withMissionStoreRootLocked(ec)
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
