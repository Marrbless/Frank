package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

// TaskState tracks per-message execution state shared across tools.
// The main use right now is enforcing that a new deliverable task must
// initialize projects/current via frank_new_project before writing there.
type TaskState struct {
	mu                           sync.Mutex
	currentTaskID                string
	projectInitialized           bool
	missionStoreRoot             string
	executionContext             missioncontrol.ExecutionContext
	hasExecutionContext          bool
	missionJob                   missioncontrol.Job
	hasMissionJob                bool
	runtimeControl               missioncontrol.RuntimeControlContext
	hasRuntimeControl            bool
	runtimeState                 missioncontrol.JobRuntimeState
	hasRuntimeState              bool
	operatorChannel              string
	operatorChatID               string
	auditEvents                  []missioncontrol.AuditEvent
	runtimePersistHook           func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeProjectionHook        func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeChangeHook            func()
	campaignReadinessGuardHook   func(missioncontrol.ExecutionContext) error
	treasuryActivationPolicyHook func(string, missioncontrol.WriterLockLease, missioncontrol.DefaultTreasuryActivationPolicyInput, time.Time) error
}

const taskStateTreasuryActivationLeaseHolderID = "taskstate-activate-step-treasury"

func NewTaskState() *TaskState {
	return &TaskState{
		campaignReadinessGuardHook:   missioncontrol.RequireExecutionContextCampaignReadiness,
		treasuryActivationPolicyHook: missioncontrol.ApplyDefaultTreasuryActivationPolicy,
	}
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

func (s *TaskState) SetMissionStoreRoot(root string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missionStoreRoot = strings.TrimSpace(root)
	if s.hasExecutionContext {
		s.executionContext.MissionStoreRoot = s.missionStoreRoot
	}
}

func (s *TaskState) SetExecutionContext(ec missioncontrol.ExecutionContext) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := s.withMissionStoreRootLocked(missioncontrol.CloneExecutionContext(ec))
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

func (s *TaskState) MissionJobWithStoreRoot() (missioncontrol.Job, string, bool) {
	if s == nil {
		return missioncontrol.Job{}, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasMissionJob {
		return missioncontrol.Job{}, strings.TrimSpace(s.missionStoreRoot), false
	}
	return *missioncontrol.CloneJob(&s.missionJob), strings.TrimSpace(s.missionStoreRoot), true
}

func (s *TaskState) OperatorSession() (string, string, bool) {
	if s == nil {
		return "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.operatorChannel == "" && s.operatorChatID == "" {
		return "", "", false
	}
	return s.operatorChannel, s.operatorChatID, true
}

func (s *TaskState) SetRuntimeChangeHook(hook func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeChangeHook = hook
}

func (s *TaskState) SetRuntimePersistHook(hook func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimePersistHook = hook
}

func (s *TaskState) SetRuntimeProjectionHook(hook func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeProjectionHook = hook
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
		if current.JobID != "" && current.JobID != job.ID && missioncontrol.IsTerminalJobState(current.State) {
			current = nil
		}
	}
	s.mu.Unlock()

	now := time.Now()
	runtimeState, err := missioncontrol.SetJobRuntimeActiveStep(job, current, stepID, now)
	if err != nil {
		return err
	}
	if err := s.applyCampaignReadinessGuardForStep(job, stepID); err != nil {
		return err
	}
	if err := s.applyTreasuryActivationPolicyForStep(job, stepID, now); err != nil {
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

func (s *TaskState) applyTreasuryActivationPolicyForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || ec.Step.TreasuryRef == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.treasuryActivationPolicyHook
	s.mu.Unlock()
	if hook == nil {
		return nil
	}

	return hook(root, missioncontrol.WriterLockLease{
		LeaseHolderID: taskStateTreasuryActivationLeaseHolderID,
	}, missioncontrol.DefaultTreasuryActivationPolicyInput{
		TreasuryRef: *ec.Step.TreasuryRef,
	}, now)
}

func (s *TaskState) applyCampaignReadinessGuardForStep(job missioncontrol.Job, stepID string) error {
	if s == nil {
		return nil
	}

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || ec.Step.CampaignRef == nil {
		return nil
	}

	s.mu.Lock()
	ec.MissionStoreRoot = strings.TrimSpace(s.missionStoreRoot)
	hook := s.campaignReadinessGuardHook
	s.mu.Unlock()
	if hook == nil {
		return nil
	}

	return hook(ec)
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
	runtimeWithOutput, exhausted, err := missioncontrol.RecordOwnerFacingStepOutput(*ec.Runtime, now)
	if err != nil {
		return err
	}
	if exhausted {
		s.mu.Lock()
		err = s.storeRuntimeStateLocked(ec.Job, runtimeWithOutput, nil)
		s.mu.Unlock()
		if err == nil {
			s.notifyRuntimeChanged()
		}
		return err
	}
	ec.Runtime = &runtimeWithOutput

	nextRuntime, err := missioncontrol.CompleteRuntimeStep(ec, now, missioncontrol.StepValidationInput{
		FinalResponse:   finalContent,
		SessionChannel:  operatorChannel,
		SessionChatID:   operatorChatID,
		SuccessfulTools: successfulTools,
	})
	if err != nil {
		s.mu.Lock()
		storeErr := s.storeRuntimeStateLocked(ec.Job, runtimeWithOutput, nil)
		s.mu.Unlock()
		if storeErr == nil {
			s.notifyRuntimeChanged()
		}
		if storeErr != nil {
			return storeErr
		}
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

func (s *TaskState) RecordFailedToolAction(toolName string, reason string) (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordFailedToolAction(*ec.Runtime, time.Now(), toolName, reason)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingMessage() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingMessage(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingCheckIn() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingCheckIn(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingDailySummary() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingDailySummary(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingApprovalRequest() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingApprovalRequest(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingCompletion() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
	s.mu.Unlock()
	if !hasRuntimeState || runtimeState == nil || runtimeState.State != missioncontrol.JobStateCompleted {
		return false, nil
	}
	if !hasMissionJob || missionJob == nil {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingCompletion(*runtimeState, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(missionJob, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingWaitingUser() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingWaitingUser(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingBudgetPause() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	missionJob := missioncontrol.CloneJob(&s.missionJob)
	hasMissionJob := s.hasMissionJob
	s.mu.Unlock()
	if !hasRuntimeState || runtimeState == nil || runtimeState.State != missioncontrol.JobStatePaused {
		return false, nil
	}
	if !hasMissionJob || missionJob == nil {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingBudgetPause(*runtimeState, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(missionJob, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingDenyAck() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateWaitingUser {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingDenyAck(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingPauseAck() (bool, error) {
	if s == nil {
		return false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStatePaused {
		return false, nil
	}

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingPauseAck(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingSetStepAck() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingSetStepAck(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingRevokeApprovalAck() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingRevokeApprovalAck(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
}

func (s *TaskState) RecordOwnerFacingResumeAck() (bool, error) {
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

	nextRuntime, exhausted, err := missioncontrol.RecordOwnerFacingResumeAck(*ec.Runtime, time.Now())
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return exhausted, err
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
	missionStoreRoot := strings.TrimSpace(s.missionStoreRoot)
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
		campaignPreflight, treasuryPreflight, err := resolveExecutionContextCampaignAndTreasuryPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(
			*ec.Runtime,
			missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step),
			campaignPreflight,
			treasuryPreflight,
		)
		if err != nil {
			return "", err
		}
		return formatOperatorStatusReadoutWithDeferredSchedulerTriggers(summary, ec.MissionStoreRoot)
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
	summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedTools(*runtimeState, allowedTools)
	if err != nil {
		return "", err
	}
	return formatOperatorStatusReadoutWithDeferredSchedulerTriggers(summary, missionStoreRoot)
}

func formatOperatorStatusReadoutWithDeferredSchedulerTriggers(summary string, missionStoreRoot string) (string, error) {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" {
		return summary, nil
	}

	deferred, err := missioncontrol.LoadDeferredSchedulerTriggerStatuses(missionStoreRoot)
	if err != nil || len(deferred) == 0 {
		return summary, nil
	}

	var statusSummary missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(summary), &statusSummary); err != nil {
		return "", err
	}
	statusSummary = missioncontrol.WithDeferredSchedulerTriggers(statusSummary, deferred)

	data, err := json.MarshalIndent(statusSummary, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func resolveExecutionContextCampaignAndTreasuryPreflight(ec missioncontrol.ExecutionContext) (*missioncontrol.ResolvedExecutionContextCampaignPreflight, *missioncontrol.ResolvedExecutionContextTreasuryPreflight, error) {
	var campaignPreflight *missioncontrol.ResolvedExecutionContextCampaignPreflight
	if ec.Step != nil && ec.Step.CampaignRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
		if err != nil {
			return nil, nil, err
		}
		campaignPreflight = &resolved
	}

	var treasuryPreflight *missioncontrol.ResolvedExecutionContextTreasuryPreflight
	if ec.Step != nil && ec.Step.TreasuryRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryPreflight(ec)
		if err != nil {
			return nil, nil, err
		}
		treasuryPreflight = &resolved
	}
	return campaignPreflight, treasuryPreflight, nil
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

		summary, err := missioncontrol.NewInspectSummaryWithCampaignAndTreasuryPreflight(*ec.Job, stepID, ec.MissionStoreRoot)
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
	storedRuntime, persistControl, err := s.persistPreparedRuntimeStateLocked(job, runtimeState, control)
	if err != nil {
		return err
	}
	if job != nil && storedRuntime.InspectablePlan == nil {
		inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(*job)
		if err != nil {
			return err
		}
		storedRuntime.InspectablePlan = &inspectablePlan
	}

	var storedControl *missioncontrol.RuntimeControlContext
	var storedExecutionContext missioncontrol.ExecutionContext
	hasExecutionContext := false
	if storedRuntime.ActiveStepID != "" {
		if control == nil {
			if job == nil {
				return missioncontrol.ValidationError{
					Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
					Message: "runtime execution requires a mission job or persisted control context",
				}
			}
			builtControl, err := missioncontrol.BuildRuntimeControlContext(*job, storedRuntime.ActiveStepID)
			if err != nil {
				return err
			}
			storedControl = &builtControl
			if persistControl == nil {
				persistControl = missioncontrol.CloneRuntimeControlContext(&builtControl)
			}
			resolved, err := missioncontrol.ResolveExecutionContextWithRuntime(*job, storedRuntime)
			if err != nil {
				return err
			}
			storedExecutionContext = s.withMissionStoreRootLocked(resolved)
		} else {
			if persistControl == nil {
				return missioncontrol.ValidationError{
					Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
					Message: "runtime execution requires a mission job or persisted control context",
				}
			}
			resolved, err := missioncontrol.ResolveExecutionContextWithRuntimeControl(*persistControl, storedRuntime)
			if err != nil {
				return err
			}
			storedExecutionContext = s.withMissionStoreRootLocked(resolved)
			storedControl = missioncontrol.CloneRuntimeControlContext(persistControl)
		}
		hasExecutionContext = true
	}

	s.runtimeState = storedRuntime
	s.hasRuntimeState = true
	s.auditEvents = missioncontrol.CloneAuditHistory(storedRuntime.AuditHistory)
	if storedControl != nil {
		s.runtimeControl = *missioncontrol.CloneRuntimeControlContext(storedControl)
		s.hasRuntimeControl = true
	} else {
		s.runtimeControl = missioncontrol.RuntimeControlContext{}
		s.hasRuntimeControl = false
	}
	s.storeMissionJobLocked(job)
	if hasExecutionContext {
		s.executionContext = storedExecutionContext
		s.hasExecutionContext = true
	} else {
		s.executionContext = missioncontrol.ExecutionContext{}
		s.hasExecutionContext = false
	}
	return s.projectRuntimeStateLocked(job, storedRuntime, storedControl)
}

func (s *TaskState) persistPreparedRuntimeStateLocked(job *missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) (missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext, error) {
	storedRuntime := *missioncontrol.CloneJobRuntimeState(&runtimeState)
	persistControl := missioncontrol.CloneRuntimeControlContext(control)
	if s.runtimePersistHook == nil {
		return storedRuntime, persistControl, nil
	}
	if job != nil && len(job.Plan.Steps) > 0 && storedRuntime.InspectablePlan == nil {
		inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(*job)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, nil, err
		}
		storedRuntime.InspectablePlan = &inspectablePlan
	}
	if storedRuntime.ActiveStepID != "" && persistControl == nil {
		if job == nil || len(job.Plan.Steps) == 0 {
			return missioncontrol.JobRuntimeState{}, nil, missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "runtime control requires a mission job or persisted control context",
			}
		}
		builtControl, err := missioncontrol.BuildRuntimeControlContext(*job, storedRuntime.ActiveStepID)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, nil, err
		}
		persistControl = &builtControl
	}
	if err := s.runtimePersistHook(
		missioncontrol.CloneJob(job),
		*missioncontrol.CloneJobRuntimeState(&storedRuntime),
		missioncontrol.CloneRuntimeControlContext(persistControl),
	); err != nil {
		return missioncontrol.JobRuntimeState{}, nil, err
	}
	return storedRuntime, persistControl, nil
}

func (s *TaskState) persistHydratedRuntimeStateLocked(job missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	storedRuntime, persistControl, err := s.persistPreparedRuntimeStateLocked(&job, runtimeState, control)
	if err != nil {
		return err
	}
	if err := s.hydrateRuntimeControlLocked(job, storedRuntime, persistControl); err != nil {
		return err
	}
	return s.projectRuntimeStateLocked(&job, storedRuntime, persistControl)
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

func (s *TaskState) projectRuntimeStateLocked(job *missioncontrol.Job, runtimeState missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
	if s == nil || s.runtimeProjectionHook == nil {
		return nil
	}
	return s.runtimeProjectionHook(
		missioncontrol.CloneJob(job),
		*missioncontrol.CloneJobRuntimeState(&runtimeState),
		missioncontrol.CloneRuntimeControlContext(control),
	)
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

func (s *TaskState) withMissionStoreRootLocked(ec missioncontrol.ExecutionContext) missioncontrol.ExecutionContext {
	if strings.TrimSpace(s.missionStoreRoot) == "" {
		return ec
	}
	ec.MissionStoreRoot = s.missionStoreRoot
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
