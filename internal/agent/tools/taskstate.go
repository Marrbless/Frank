package tools

import (
	"context"
	"encoding/json"
	"errors"
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
	mu                             sync.Mutex
	currentTaskID                  string
	projectInitialized             bool
	missionStoreRoot               string
	executionContext               missioncontrol.ExecutionContext
	hasExecutionContext            bool
	missionJob                     missioncontrol.Job
	hasMissionJob                  bool
	runtimeControl                 missioncontrol.RuntimeControlContext
	hasRuntimeControl              bool
	runtimeState                   missioncontrol.JobRuntimeState
	hasRuntimeState                bool
	operatorChannel                string
	operatorChatID                 string
	auditEvents                    []missioncontrol.AuditEvent
	runtimePersistHook             func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeProjectionHook          func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeChangeHook              func()
	campaignReadinessGuardHook     func(missioncontrol.ExecutionContext) error
	zohoMailboxBootstrapHook       func(string, missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, time.Time) error
	treasuryFirstAcquisitionHook   func(string, missioncontrol.WriterLockLease, missioncontrol.FirstTreasuryAcquisitionInput, time.Time) error
	treasuryBootstrapProducerHook  func(string, missioncontrol.WriterLockLease, missioncontrol.FirstValueTreasuryBootstrapInput, time.Time) error
	treasuryPostActiveSuspendHook  func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySuspendInput, time.Time) error
	treasuryPostSuspendResumeHook  func(string, missioncontrol.WriterLockLease, missioncontrol.PostSuspendTreasuryResumeInput, time.Time) error
	treasuryPostActiveAllocateHook func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryAllocateInput, time.Time) error
	treasuryPostActiveReinvestHook func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryReinvestInput, time.Time) error
	treasuryPostActiveSpendHook    func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySpendInput, time.Time) error
	treasuryPostActiveTransferHook func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryTransferInput, time.Time) error
	treasuryPostActiveSaveHook     func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySaveInput, time.Time) error
	treasuryPostAcquisitionHook    func(string, missioncontrol.WriterLockLease, missioncontrol.PostBootstrapTreasuryAcquisitionInput, time.Time) error
	treasuryActivationProducerHook func(string, missioncontrol.WriterLockLease, missioncontrol.DefaultTreasuryActivationPolicyInput, time.Time) error
}

const taskStateTreasuryExecutionLeaseHolderID = "taskstate-activate-step-treasury"

func NewTaskState() *TaskState {
	return &TaskState{
		campaignReadinessGuardHook:     missioncontrol.RequireExecutionContextCampaignReadiness,
		zohoMailboxBootstrapHook:       missioncontrol.ProduceFrankZohoMailboxBootstrap,
		treasuryFirstAcquisitionHook:   missioncontrol.RecordFirstTreasuryAcquisition,
		treasuryBootstrapProducerHook:  missioncontrol.ProduceFirstValueTreasuryBootstrap,
		treasuryPostActiveSuspendHook:  missioncontrol.ProducePostActiveTreasurySuspend,
		treasuryPostSuspendResumeHook:  missioncontrol.ProducePostSuspendTreasuryResume,
		treasuryPostActiveAllocateHook: missioncontrol.ProducePostActiveTreasuryAllocate,
		treasuryPostActiveReinvestHook: missioncontrol.ProducePostActiveTreasuryReinvest,
		treasuryPostActiveSpendHook:    missioncontrol.ProducePostActiveTreasurySpend,
		treasuryPostActiveTransferHook: missioncontrol.ProducePostActiveTreasuryTransfer,
		treasuryPostActiveSaveHook:     missioncontrol.ProducePostActiveTreasurySave,
		treasuryPostAcquisitionHook:    missioncontrol.RecordPostBootstrapTreasuryAcquisition,
		treasuryActivationProducerHook: missioncontrol.ProduceFundedTreasuryActivation,
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
	if err := s.applyZohoMailboxBootstrapForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyTreasuryExecutionForStep(job, stepID, now); err != nil {
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

func (s *TaskState) applyTreasuryExecutionForStep(job missioncontrol.Job, stepID string, now time.Time) error {
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
	firstAcquisitionHook := s.treasuryFirstAcquisitionHook
	bootstrapHook := s.treasuryBootstrapProducerHook
	postActiveSuspendHook := s.treasuryPostActiveSuspendHook
	postSuspendResumeHook := s.treasuryPostSuspendResumeHook
	postActiveAllocateHook := s.treasuryPostActiveAllocateHook
	postActiveReinvestHook := s.treasuryPostActiveReinvestHook
	postActiveSpendHook := s.treasuryPostActiveSpendHook
	postActiveTransferHook := s.treasuryPostActiveTransferHook
	postActiveSaveHook := s.treasuryPostActiveSaveHook
	postAcquisitionHook := s.treasuryPostAcquisitionHook
	hook := s.treasuryActivationProducerHook
	s.mu.Unlock()
	ec.MissionStoreRoot = root

	treasury, err := missioncontrol.ResolveExecutionContextTreasuryRef(ec)
	if err != nil {
		return err
	}
	if treasury == nil {
		return nil
	}

	lease := missioncontrol.WriterLockLease{
		LeaseHolderID: taskStateTreasuryExecutionLeaseHolderID,
	}
	if treasury.State == missioncontrol.TreasuryStateBootstrap {
		if firstAcquisitionHook == nil || bootstrapHook == nil {
			return nil
		}
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryBootstrapAcquisition(ec)
		if err != nil {
			return err
		}
		if resolved == nil {
			return nil
		}

		if err := firstAcquisitionHook(root, lease, missioncontrol.FirstTreasuryAcquisitionInput{
			TreasuryID: resolved.Treasury.TreasuryID,
			EntryID:    resolved.BootstrapAcquisition.EntryID,
			AssetCode:  resolved.BootstrapAcquisition.AssetCode,
			Amount:     resolved.BootstrapAcquisition.Amount,
			SourceRef:  resolved.BootstrapAcquisition.SourceRef,
		}, now); err != nil {
			return err
		}
		return bootstrapHook(root, lease, missioncontrol.FirstValueTreasuryBootstrapInput{
			TreasuryRef: *ec.Step.TreasuryRef,
			EntryID:     resolved.BootstrapAcquisition.EntryID,
		}, now)
	}
	if treasury.State == missioncontrol.TreasuryStateActive {
		if postActiveSuspendHook != nil {
			resolvedSuspend, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveSuspend(ec)
			if err != nil {
				return err
			}
			if resolvedSuspend != nil {
				return postActiveSuspendHook(root, lease, missioncontrol.PostActiveTreasurySuspendInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postActiveAllocateHook != nil {
			resolvedAllocate, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveAllocate(ec)
			if err != nil {
				return err
			}
			if resolvedAllocate != nil {
				return postActiveAllocateHook(root, lease, missioncontrol.PostActiveTreasuryAllocateInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postActiveReinvestHook != nil {
			resolvedReinvest, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveReinvest(ec)
			if err != nil {
				return err
			}
			if resolvedReinvest != nil {
				return postActiveReinvestHook(root, lease, missioncontrol.PostActiveTreasuryReinvestInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postActiveSpendHook != nil {
			resolvedSpend, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveSpend(ec)
			if err != nil {
				return err
			}
			if resolvedSpend != nil {
				return postActiveSpendHook(root, lease, missioncontrol.PostActiveTreasurySpendInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postActiveTransferHook != nil {
			resolvedTransfer, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveTransfer(ec)
			if err != nil {
				return err
			}
			if resolvedTransfer != nil {
				return postActiveTransferHook(root, lease, missioncontrol.PostActiveTreasuryTransferInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postActiveSaveHook != nil {
			resolvedSave, err := missioncontrol.ResolveExecutionContextTreasuryPostActiveSave(ec)
			if err != nil {
				return err
			}
			if resolvedSave != nil {
				return postActiveSaveHook(root, lease, missioncontrol.PostActiveTreasurySaveInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		if postAcquisitionHook == nil {
			return nil
		}
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryPostBootstrapAcquisition(ec)
		if err != nil {
			return err
		}
		if resolved == nil {
			return nil
		}
		return postAcquisitionHook(root, lease, missioncontrol.PostBootstrapTreasuryAcquisitionInput{
			TreasuryID: resolved.Treasury.TreasuryID,
		}, now)
	}
	if treasury.State == missioncontrol.TreasuryStateSuspended {
		if postSuspendResumeHook != nil {
			resolvedResume, err := missioncontrol.ResolveExecutionContextTreasuryPostSuspendResume(ec)
			if err != nil {
				return err
			}
			if resolvedResume != nil {
				return postSuspendResumeHook(root, lease, missioncontrol.PostSuspendTreasuryResumeInput{
					TreasuryRef: *ec.Step.TreasuryRef,
				}, now)
			}
		}
		return nil
	}
	if hook == nil {
		return nil
	}

	return hook(root, lease, missioncontrol.DefaultTreasuryActivationPolicyInput{
		TreasuryRef: *ec.Step.TreasuryRef,
	}, now)
}

func (s *TaskState) applyZohoMailboxBootstrapForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.DeclaresFrankZohoMailboxBootstrap(*ec.Step) {
		return nil
	}

	s.mu.Lock()
	ec.MissionStoreRoot = strings.TrimSpace(s.missionStoreRoot)
	hook := s.zohoMailboxBootstrapHook
	s.mu.Unlock()

	pair, ok, err := missioncontrol.ResolveExecutionContextFrankZohoMailboxBootstrapPair(ec)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if hook == nil {
		return nil
	}

	return hook(ec.MissionStoreRoot, pair, now)
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

func (s *TaskState) RecordFrankZohoSendReceipt(result string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	nextRuntime, appended, err := missioncontrol.AppendFrankZohoSendReceipt(*ec.Runtime, ec.Step.ID, result)
	if err != nil || !appended {
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

func (s *TaskState) SyncFrankZohoCampaignInboundReplies() (int, error) {
	if s == nil {
		return 0, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Step.CampaignRef == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return 0, nil
	}

	if err := missioncontrol.RequireExecutionContextCampaignReadiness(ec); err != nil {
		return 0, err
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return 0, err
	}
	sender, err := resolveFrankZohoCampaignSender(preflight, false)
	if err != nil {
		return 0, err
	}

	replies, err := readFrankZohoCampaignInboundReplies(context.Background(), sender.ProviderAccountID)
	if err != nil {
		return 0, err
	}

	nextRuntime := *missioncontrol.CloneJobRuntimeState(ec.Runtime)
	appended := 0
	for _, reply := range replies {
		reply.StepID = ec.Step.ID
		updatedRuntime, changed, err := missioncontrol.AppendFrankZohoInboundReply(nextRuntime, reply)
		if err != nil {
			return 0, err
		}
		if !changed {
			continue
		}
		nextRuntime = updatedRuntime
		appended++
	}
	if appended > 0 {
		s.mu.Lock()
		err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		s.notifyRuntimeChanged()

		s.mu.Lock()
		ec = missioncontrol.CloneExecutionContext(s.executionContext)
		hasExecutionContext = s.hasExecutionContext
		s.mu.Unlock()
		if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
			return appended, nil
		}
		nextRuntime = *missioncontrol.CloneJobRuntimeState(ec.Runtime)
	}

	workItems, err := missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(ec.MissionStoreRoot, preflight.Campaign.CampaignID, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	workItemChanged := false
	for _, item := range workItems {
		if _, exists := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(nextRuntime, item.InboundReplyID); exists {
			continue
		}
		updatedRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, item)
		if err != nil {
			return 0, err
		}
		if !changed {
			continue
		}
		nextRuntime = updatedRuntime
		workItemChanged = true
	}
	if appended == 0 && !workItemChanged {
		return 0, nil
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return appended, err
}

func (s *TaskState) PrepareFrankZohoCampaignSend(args map[string]interface{}) (string, bool, error) {
	if s == nil {
		return "", false, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return "", false, nil
	}

	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return "", false, err
	}
	_, hasInboundReplyID, err := frankZohoOptionalStringArg(args, "inbound_reply_id")
	if err != nil {
		return "", false, err
	}
	if hasInboundReplyID || (preflight.Campaign != nil && missioncontrol.CampaignZohoEmailStopConditionsRequireInboundReplies(preflight.Campaign.StopConditions)) {
		if _, err := s.SyncFrankZohoCampaignInboundReplies(); err != nil {
			return "", false, err
		}
		s.mu.Lock()
		ec = missioncontrol.CloneExecutionContext(s.executionContext)
		hasExecutionContext = s.hasExecutionContext
		s.mu.Unlock()
		if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
			return "", false, nil
		}
	}

	now := time.Now().UTC()
	intent, err := buildFrankZohoCampaignSendIntent(ec, args, now)
	if err != nil {
		return "", false, err
	}
	action := intent.PreparedAction
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, action.ActionID); ok {
		switch existing.State {
		case missioncontrol.CampaignZohoEmailOutboundActionStateVerified:
			nextRuntime, runtimeChanged, err := transitionFrankZohoCampaignReplyWorkItemResponded(*ec.Runtime, existing, now)
			if err != nil {
				return "", true, err
			}
			if runtimeChanged {
				s.mu.Lock()
				err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
				s.mu.Unlock()
				if err != nil {
					return "", true, err
				}
				s.notifyRuntimeChanged()
				ec.Runtime = &nextRuntime
			}
			receipt, err := frankZohoSendReceiptFromCampaignAction(existing)
			if err != nil {
				return "", false, err
			}
			return receipt, true, nil
		case missioncontrol.CampaignZohoEmailOutboundActionStateSent:
			verifiedProof, err := verifyFrankZohoCampaignSendProof(context.Background(), frankZohoCampaignProofFromAction(existing))
			if err != nil {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize succeeds: %w", frankZohoSendEmailToolName, existing.ActionID, err)
			}
			if len(verifiedProof) != 1 {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize returns exactly one proof record", frankZohoSendEmailToolName, existing.ActionID)
			}
			finalized, err := finalizeFrankZohoCampaignActionFromProof(existing, verifiedProof[0], now)
			if err != nil {
				return "", true, fmt.Errorf("%s: campaign outbound action %q remains blocked until provider-mailbox verification/finalize reconciles it: %w", frankZohoSendEmailToolName, existing.ActionID, err)
			}
			nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, finalized)
			if err != nil {
				return "", true, err
			}
			nextRuntime, workItemChanged, err := transitionFrankZohoCampaignReplyWorkItemResponded(nextRuntime, finalized, now)
			if err != nil {
				return "", true, err
			}
			if changed || workItemChanged {
				s.mu.Lock()
				err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
				s.mu.Unlock()
				if err != nil {
					return "", true, err
				}
				s.notifyRuntimeChanged()
				ec.Runtime = &nextRuntime
			}
			receipt, err := frankZohoSendReceiptFromCampaignAction(finalized)
			if err != nil {
				return "", true, err
			}
			return receipt, true, nil
		case missioncontrol.CampaignZohoEmailOutboundActionStatePrepared:
			return "", true, fmt.Errorf("%s: campaign outbound action %q is already prepared without provider receipt proof; refusing to resend until reconciled", frankZohoSendEmailToolName, existing.ActionID)
		case missioncontrol.CampaignZohoEmailOutboundActionStateFailed:
			return "", true, fmt.Errorf("%s: campaign outbound action %q is terminally failed and will not be resent automatically", frankZohoSendEmailToolName, existing.ActionID)
		default:
			return "", true, fmt.Errorf("%s: campaign outbound action %q has unsupported state %q", frankZohoSendEmailToolName, existing.ActionID, existing.State)
		}
	}
	if action.ReplyToInboundReplyID != "" {
		nextRuntime, err := claimFrankZohoCampaignReplyWorkItem(ec, *ec.Runtime, action, now)
		if err != nil {
			return "", false, err
		}
		ec.Runtime = &nextRuntime
	}
	if hasInboundReplyID || action.ReplyToInboundReplyID != "" {
		inboundReplyID := action.ReplyToInboundReplyID
		followUpActions, err := missioncontrol.ListCommittedCampaignZohoEmailFollowUpActionsByInboundReply(ec.MissionStoreRoot, inboundReplyID)
		if err != nil {
			return "", false, err
		}
		for _, record := range followUpActions {
			if strings.TrimSpace(record.ActionID) == action.ActionID {
				continue
			}
			switch missioncontrol.CampaignZohoEmailOutboundActionState(strings.TrimSpace(record.State)) {
			case missioncontrol.CampaignZohoEmailOutboundActionStatePrepared, missioncontrol.CampaignZohoEmailOutboundActionStateSent:
				return "", false, fmt.Errorf("%s: inbound_reply_id %q already has unresolved follow-up action %q in state %q; refusing to prepare another follow-up until it is finalized", frankZohoSendEmailToolName, inboundReplyID, strings.TrimSpace(record.ActionID), strings.TrimSpace(record.State))
			}
		}
	}

	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, action)
	if err != nil || !changed {
		return "", false, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil {
		s.notifyRuntimeChanged()
	}
	return "", false, err
}

func (s *TaskState) RecordFrankZohoCampaignSend(args map[string]interface{}, result string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	prepared, err := buildFrankZohoPreparedCampaignAction(ec, args, time.Now().UTC())
	if err != nil {
		return err
	}
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, prepared.ActionID); ok {
		prepared = existing
	}
	receipt, err := missioncontrol.ParseFrankZohoSendReceipt(result)
	if err != nil {
		return err
	}
	receipt.StepID = ec.Step.ID
	if err := missioncontrol.ValidateFrankZohoSendReceipt(receipt); err != nil {
		return err
	}
	sent, err := missioncontrol.BuildCampaignZohoEmailOutboundSentAction(prepared, receipt, time.Now().UTC())
	if err != nil {
		return err
	}

	nextRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, sent)
	if err != nil {
		return err
	}
	nextRuntime, _, err = missioncontrol.AppendFrankZohoSendReceipt(nextRuntime, ec.Step.ID, result)
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

func (s *TaskState) RecordFrankZohoCampaignSendFailure(args map[string]interface{}, sendErr error) error {
	if s == nil || sendErr == nil {
		return nil
	}

	var terminalFailure interface {
		Failure() missioncontrol.CampaignZohoEmailOutboundFailure
	}
	if !errors.As(sendErr, &terminalFailure) {
		return nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return nil
	}

	prepared, err := buildFrankZohoPreparedCampaignAction(ec, args, time.Now().UTC())
	if err != nil {
		return err
	}
	if existing, ok := missioncontrol.FindCampaignZohoEmailOutboundAction(*ec.Runtime, prepared.ActionID); ok {
		prepared = existing
	}
	if prepared.State != missioncontrol.CampaignZohoEmailOutboundActionStatePrepared {
		return nil
	}
	failed, err := missioncontrol.BuildCampaignZohoEmailOutboundFailedAction(prepared, terminalFailure.Failure(), time.Now().UTC())
	if err != nil {
		return err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailOutboundAction(*ec.Runtime, failed)
	if err != nil || !changed {
		return err
	}
	nextRuntime, workItemChanged, err := transitionFrankZohoCampaignReplyWorkItemOnFailure(ec, nextRuntime, failed, time.Now().UTC())
	if err != nil {
		return err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err == nil && (changed || workItemChanged) {
		s.notifyRuntimeChanged()
	}
	return err
}

func (s *TaskState) ManageFrankZohoCampaignReplyWorkItem(args map[string]interface{}) (string, bool, error) {
	if s == nil {
		return "", true, nil
	}

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	s.mu.Unlock()
	if !hasExecutionContext || ec.Job == nil || ec.Step == nil || ec.Step.CampaignRef == nil || ec.Runtime == nil || ec.Runtime.State != missioncontrol.JobStateRunning {
		return "", true, nil
	}

	if _, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec); err != nil {
		return "", true, err
	}
	inboundReplyID, err := frankZohoRequiredStringArg(args, "inbound_reply_id")
	if err != nil {
		return "", true, err
	}
	action, err := frankZohoRequiredStringArg(args, "action")
	if err != nil {
		return "", true, err
	}
	now := time.Now().UTC()
	nextRuntime, item, err := ensureFrankZohoCampaignReplyWorkItem(ec, *ec.Runtime, inboundReplyID, now)
	if err != nil {
		return "", true, err
	}

	switch item.State {
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed:
		return "", true, fmt.Errorf("%s: inbound_reply_id %q is currently claimed by follow-up action %q", frankZohoManageReplyWorkItemToolName, inboundReplyID, item.ClaimedFollowUpActionID)
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded, missioncontrol.CampaignZohoEmailReplyWorkItemStateIgnored:
		return "", true, fmt.Errorf("%s: inbound_reply_id %q is already terminal in state %q", frankZohoManageReplyWorkItemToolName, inboundReplyID, item.State)
	}

	var mutated missioncontrol.CampaignZohoEmailReplyWorkItem
	switch action {
	case "ignore":
		mutated, err = missioncontrol.BuildCampaignZohoEmailReplyWorkItemIgnored(item, now)
	case "defer":
		deferUntilText, err := frankZohoRequiredStringArg(args, "defer_until")
		if err != nil {
			return "", true, err
		}
		deferUntil, parseErr := time.Parse(time.RFC3339, deferUntilText)
		if parseErr != nil {
			return "", true, fmt.Errorf("%s: defer_until must be RFC3339: %w", frankZohoManageReplyWorkItemToolName, parseErr)
		}
		if !deferUntil.UTC().After(now) {
			return "", true, fmt.Errorf("%s: defer_until must be in the future", frankZohoManageReplyWorkItemToolName)
		}
		mutated, err = missioncontrol.BuildCampaignZohoEmailReplyWorkItemDeferred(item, deferUntil.UTC(), now)
	default:
		return "", true, fmt.Errorf("%s: action %q is not supported", frankZohoManageReplyWorkItemToolName, action)
	}
	if err != nil {
		return "", true, err
	}

	nextRuntime, _, err = missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, mutated)
	if err != nil {
		return "", true, err
	}

	s.mu.Lock()
	err = s.storeRuntimeStateLocked(ec.Job, nextRuntime, nil)
	s.mu.Unlock()
	if err != nil {
		return "", true, err
	}
	s.notifyRuntimeChanged()

	payload, err := json.Marshal(struct {
		InboundReplyID string `json:"inbound_reply_id"`
		State          string `json:"state"`
		DeferredUntil  string `json:"deferred_until,omitempty"`
	}{
		InboundReplyID: mutated.InboundReplyID,
		State:          string(mutated.State),
		DeferredUntil:  formatTaskStateRFC3339(mutated.DeferredUntil),
	})
	if err != nil {
		return "", true, err
	}
	return string(payload), true, nil
}

func claimFrankZohoCampaignReplyWorkItem(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		loaded, found, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkItemByInboundReply(ec.MissionStoreRoot, action.CampaignID, action.ReplyToInboundReplyID)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, err
		}
		if !found {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is missing a committed reply work item", frankZohoSendEmailToolName, action.ReplyToInboundReplyID)
		}
		item = loaded
	}
	switch item.State {
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateOpen:
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateDeferred:
		if item.DeferredUntil.After(now.UTC()) {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is deferred until %s", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.DeferredUntil.Format(time.RFC3339))
		}
		reopened, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemReopened(item, now)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, err
		}
		item = reopened
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed:
		if strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
			return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q already has claimed follow-up action %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.ClaimedFollowUpActionID)
		}
		return runtime, nil
	case missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded, missioncontrol.CampaignZohoEmailReplyWorkItemStateIgnored:
		return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q is not eligible for follow-up in state %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.State)
	default:
		return missioncontrol.JobRuntimeState{}, fmt.Errorf("%s: inbound_reply_id %q has unsupported reply work item state %q", frankZohoSendEmailToolName, action.ReplyToInboundReplyID, item.State)
	}
	claimed, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemClaimed(item, action.ActionID, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, err
	}
	nextRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, claimed)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, err
	}
	return nextRuntime, nil
}

func transitionFrankZohoCampaignReplyWorkItemResponded(runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, bool, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, false, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		return runtime, false, nil
	}
	if item.State == missioncontrol.CampaignZohoEmailReplyWorkItemStateResponded {
		return runtime, false, nil
	}
	if item.State != missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed || strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
		return runtime, false, nil
	}
	responded, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemResponded(item, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, responded)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	return nextRuntime, changed, nil
}

func transitionFrankZohoCampaignReplyWorkItemOnFailure(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, action missioncontrol.CampaignZohoEmailOutboundAction, now time.Time) (missioncontrol.JobRuntimeState, bool, error) {
	if strings.TrimSpace(action.ReplyToInboundReplyID) == "" {
		return runtime, false, nil
	}
	item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, action.ReplyToInboundReplyID)
	if !ok {
		return runtime, false, nil
	}
	if item.State != missioncontrol.CampaignZohoEmailReplyWorkItemStateClaimed || strings.TrimSpace(item.ClaimedFollowUpActionID) != action.ActionID {
		return runtime, false, nil
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	decision, err := missioncontrol.DeriveCampaignZohoEmailSendGateDecisionFromRuntime(*preflight.Campaign, runtime)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	if !decision.Allowed {
		return runtime, false, nil
	}
	reopened, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemReopened(item, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	nextRuntime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, reopened)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, false, err
	}
	return nextRuntime, changed, nil
}

func ensureFrankZohoCampaignReplyWorkItem(ec missioncontrol.ExecutionContext, runtime missioncontrol.JobRuntimeState, inboundReplyID string, now time.Time) (missioncontrol.JobRuntimeState, missioncontrol.CampaignZohoEmailReplyWorkItem, error) {
	if item, ok := missioncontrol.FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime, inboundReplyID); ok {
		return runtime, item, nil
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	missingItems, err := missioncontrol.LoadMissingCommittedCampaignZohoEmailReplyWorkItems(ec.MissionStoreRoot, preflight.Campaign.CampaignID, now)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	nextRuntime := runtime
	for _, item := range missingItems {
		updatedRuntime, _, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, item)
		if err != nil {
			return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
		}
		nextRuntime = updatedRuntime
		if item.InboundReplyID == inboundReplyID {
			return nextRuntime, item, nil
		}
	}
	loaded, ok, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkItemByInboundReply(ec.MissionStoreRoot, preflight.Campaign.CampaignID, inboundReplyID)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	if !ok {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, fmt.Errorf("%s: inbound_reply_id %q does not resolve to a committed reply work item", frankZohoManageReplyWorkItemToolName, inboundReplyID)
	}
	nextRuntime, _, err = missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(nextRuntime, loaded)
	if err != nil {
		return missioncontrol.JobRuntimeState{}, missioncontrol.CampaignZohoEmailReplyWorkItem{}, err
	}
	return nextRuntime, loaded, nil
}

func formatTaskStateRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
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
		campaignPreflight, treasuryPreflight, bootstrapPreflight, err := resolveExecutionContextCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(ec)
		if err != nil {
			return "", err
		}
		summary, err := missioncontrol.FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(
			*ec.Runtime,
			missioncontrol.EffectiveAllowedTools(ec.Job, ec.Step),
			campaignPreflight,
			treasuryPreflight,
			bootstrapPreflight,
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
	summary := missioncontrol.BuildOperatorStatusSummaryWithAllowedTools(*runtimeState, allowedTools)
	if gate := persistedTaskStateCampaignZohoEmailSendGate(missionStoreRoot, runtimeState, control); gate != nil {
		summary.CampaignZohoEmailSendGate = gate
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	return formatOperatorStatusReadoutWithDeferredSchedulerTriggers(string(append(data, '\n')), missionStoreRoot)
}

func persistedTaskStateCampaignZohoEmailSendGate(missionStoreRoot string, runtime *missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) *missioncontrol.CampaignZohoEmailSendGateDecision {
	missionStoreRoot = strings.TrimSpace(missionStoreRoot)
	if missionStoreRoot == "" || runtime == nil || control == nil {
		return nil
	}
	if strings.TrimSpace(runtime.ActiveStepID) == "" || strings.TrimSpace(control.Step.ID) != strings.TrimSpace(runtime.ActiveStepID) {
		return nil
	}
	if control.Step.CampaignRef == nil {
		return nil
	}

	ec := missioncontrol.ExecutionContext{
		Step:             &control.Step,
		MissionStoreRoot: missionStoreRoot,
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return &missioncontrol.CampaignZohoEmailSendGateDecision{
			CampaignID: strings.TrimSpace(control.Step.CampaignRef.CampaignID),
			Allowed:    false,
			Halted:     false,
			Reason:     err.Error(),
		}
	}
	if preflight.Campaign == nil {
		return nil
	}

	decision, err := missioncontrol.LoadCommittedCampaignZohoEmailSendGateDecision(missionStoreRoot, *preflight.Campaign)
	if err != nil {
		return &missioncontrol.CampaignZohoEmailSendGateDecision{
			CampaignID: strings.TrimSpace(preflight.Campaign.CampaignID),
			Allowed:    false,
			Halted:     false,
			Reason:     err.Error(),
		}
	}
	return &decision
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

func resolveExecutionContextCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(ec missioncontrol.ExecutionContext) (*missioncontrol.ResolvedExecutionContextCampaignPreflight, *missioncontrol.ResolvedExecutionContextTreasuryPreflight, *missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight, error) {
	var campaignPreflight *missioncontrol.ResolvedExecutionContextCampaignPreflight
	if ec.Step != nil && ec.Step.CampaignRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		campaignPreflight = &resolved
	}

	var treasuryPreflight *missioncontrol.ResolvedExecutionContextTreasuryPreflight
	if ec.Step != nil && ec.Step.TreasuryRef != nil {
		resolved, err := missioncontrol.ResolveExecutionContextTreasuryPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		treasuryPreflight = &resolved
	}

	var bootstrapPreflight *missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight
	if ec.Step != nil && missioncontrol.DeclaresFrankZohoMailboxBootstrap(*ec.Step) {
		resolved, err := missioncontrol.ResolveExecutionContextFrankZohoMailboxBootstrapPreflight(ec)
		if err != nil {
			return nil, nil, nil, err
		}
		if resolved.Identity != nil && resolved.Account != nil {
			bootstrapPreflight = &resolved
		}
	}

	return campaignPreflight, treasuryPreflight, bootstrapPreflight, nil
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
