package tools

import (
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
	mu                                 sync.Mutex
	currentTaskID                      string
	projectInitialized                 bool
	missionStoreRoot                   string
	executionContext                   missioncontrol.ExecutionContext
	hasExecutionContext                bool
	missionJob                         missioncontrol.Job
	hasMissionJob                      bool
	runtimeControl                     missioncontrol.RuntimeControlContext
	hasRuntimeControl                  bool
	runtimeState                       missioncontrol.JobRuntimeState
	hasRuntimeState                    bool
	operatorChannel                    string
	operatorChatID                     string
	auditEvents                        []missioncontrol.AuditEvent
	runtimePersistHook                 func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeProjectionHook              func(*missioncontrol.Job, missioncontrol.JobRuntimeState, *missioncontrol.RuntimeControlContext) error
	runtimeChangeHook                  func()
	campaignReadinessGuardHook         func(missioncontrol.ExecutionContext) error
	zohoMailboxBootstrapHook           func(string, missioncontrol.ResolvedExecutionContextFrankZohoMailboxBootstrapPair, time.Time) error
	telegramOwnerControlOnboardingHook func(string, missioncontrol.ResolvedExecutionContextFrankTelegramOwnerControlOnboardingBundle, time.Time) error
	treasuryFirstAcquisitionHook       func(string, missioncontrol.WriterLockLease, missioncontrol.FirstTreasuryAcquisitionInput, time.Time) error
	treasuryBootstrapProducerHook      func(string, missioncontrol.WriterLockLease, missioncontrol.FirstValueTreasuryBootstrapInput, time.Time) error
	treasuryPostActiveSuspendHook      func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySuspendInput, time.Time) error
	treasuryPostSuspendResumeHook      func(string, missioncontrol.WriterLockLease, missioncontrol.PostSuspendTreasuryResumeInput, time.Time) error
	treasuryPostActiveAllocateHook     func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryAllocateInput, time.Time) error
	treasuryPostActiveReinvestHook     func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryReinvestInput, time.Time) error
	treasuryPostActiveSpendHook        func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySpendInput, time.Time) error
	treasuryPostActiveTransferHook     func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasuryTransferInput, time.Time) error
	treasuryPostActiveSaveHook         func(string, missioncontrol.WriterLockLease, missioncontrol.PostActiveTreasurySaveInput, time.Time) error
	treasuryPostAcquisitionHook        func(string, missioncontrol.WriterLockLease, missioncontrol.PostBootstrapTreasuryAcquisitionInput, time.Time) error
	treasuryActivationProducerHook     func(string, missioncontrol.WriterLockLease, missioncontrol.DefaultTreasuryActivationPolicyInput, time.Time) error
	notificationsCapabilityHook        func(string, missioncontrol.ExecutionContext, time.Time) error
	sharedStorageCapabilityHook        func(string, missioncontrol.ExecutionContext, time.Time) error
	contactsCapabilityHook             func(string, missioncontrol.ExecutionContext, time.Time) error
	locationCapabilityHook             func(string, missioncontrol.ExecutionContext, time.Time) error
	cameraCapabilityHook               func(string, missioncontrol.ExecutionContext, time.Time) error
	microphoneCapabilityHook           func(string, missioncontrol.ExecutionContext, time.Time) error
	smsPhoneCapabilityHook             func(string, missioncontrol.ExecutionContext, time.Time) error
	bluetoothNFCCapabilityHook         func(string, missioncontrol.ExecutionContext, time.Time) error
	broadAppControlCapabilityHook      func(string, missioncontrol.ExecutionContext, time.Time) error
}

const taskStateTreasuryExecutionLeaseHolderID = "taskstate-activate-step-treasury"

var taskStateNowUTC = func() time.Time {
	return time.Now().UTC()
}

func NewTaskState() *TaskState {
	return &TaskState{
		campaignReadinessGuardHook:         missioncontrol.RequireExecutionContextCampaignReadiness,
		zohoMailboxBootstrapHook:           missioncontrol.ProduceFrankZohoMailboxBootstrap,
		telegramOwnerControlOnboardingHook: missioncontrol.ProduceFrankTelegramOwnerControlOnboarding,
		treasuryFirstAcquisitionHook:       missioncontrol.RecordFirstTreasuryAcquisition,
		treasuryBootstrapProducerHook:      missioncontrol.ProduceFirstValueTreasuryBootstrap,
		treasuryPostActiveSuspendHook:      missioncontrol.ProducePostActiveTreasurySuspend,
		treasuryPostSuspendResumeHook:      missioncontrol.ProducePostSuspendTreasuryResume,
		treasuryPostActiveAllocateHook:     missioncontrol.ProducePostActiveTreasuryAllocate,
		treasuryPostActiveReinvestHook:     missioncontrol.ProducePostActiveTreasuryReinvest,
		treasuryPostActiveSpendHook:        missioncontrol.ProducePostActiveTreasurySpend,
		treasuryPostActiveTransferHook:     missioncontrol.ProducePostActiveTreasuryTransfer,
		treasuryPostActiveSaveHook:         missioncontrol.ProducePostActiveTreasurySave,
		treasuryPostAcquisitionHook:        missioncontrol.RecordPostBootstrapTreasuryAcquisition,
		treasuryActivationProducerHook:     missioncontrol.ProduceFundedTreasuryActivation,
		notificationsCapabilityHook:        defaultNotificationsCapabilityExposureHook,
		sharedStorageCapabilityHook:        defaultSharedStorageCapabilityExposureHook,
		contactsCapabilityHook:             defaultContactsCapabilityExposureHook,
		locationCapabilityHook:             defaultLocationCapabilityExposureHook,
		cameraCapabilityHook:               defaultCameraCapabilityExposureHook,
		microphoneCapabilityHook:           defaultMicrophoneCapabilityExposureHook,
		smsPhoneCapabilityHook:             defaultSMSPhoneCapabilityExposureHook,
		bluetoothNFCCapabilityHook:         defaultBluetoothNFCCapabilityExposureHook,
		broadAppControlCapabilityHook:      defaultBroadAppControlCapabilityExposureHook,
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
	job := missioncontrol.CloneJob(&s.missionJob)
	if job != nil {
		job.MissionStoreRoot = strings.TrimSpace(s.missionStoreRoot)
	}
	return *job, strings.TrimSpace(s.missionStoreRoot), true
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

	job.MissionStoreRoot = strings.TrimSpace(s.missionStoreRoot)
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
	if err := s.applyTelegramOwnerControlOnboardingForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyTreasuryExecutionForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyNotificationsCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applySharedStorageCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyContactsCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyLocationCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyCameraCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyMicrophoneCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applySMSPhoneCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyBluetoothNFCCapabilityForStep(job, stepID, now); err != nil {
		return err
	}
	if err := s.applyBroadAppControlCapabilityForStep(job, stepID, now); err != nil {
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
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || ec.Step.TreasuryRef == nil {
		return nil
	}
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

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.zohoMailboxBootstrapHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.DeclaresFrankZohoMailboxBootstrap(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

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

func (s *TaskState) applyTelegramOwnerControlOnboardingForStep(job missioncontrol.Job, stepID string, now time.Time) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.telegramOwnerControlOnboardingHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || !missioncontrol.DeclaresFrankTelegramOwnerControlOnboarding(*ec.Step) {
		return nil
	}
	ec.MissionStoreRoot = root

	bundle, ok, err := missioncontrol.ResolveExecutionContextFrankTelegramOwnerControlOnboardingBundle(ec)
	if err != nil {
		return err
	}
	if !ok || hook == nil {
		return nil
	}
	return hook(ec.MissionStoreRoot, bundle, now)
}

func (s *TaskState) applyCampaignReadinessGuardForStep(job missioncontrol.Job, stepID string) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	root := strings.TrimSpace(s.missionStoreRoot)
	hook := s.campaignReadinessGuardHook
	s.mu.Unlock()
	job.MissionStoreRoot = root

	ec, err := missioncontrol.ResolveExecutionContext(job, stepID)
	if err != nil {
		return err
	}
	if ec.Step == nil || ec.Step.CampaignRef == nil {
		return nil
	}
	ec.MissionStoreRoot = root
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

func taskStateTransitionTimestamp(now time.Time, lowerBounds ...time.Time) time.Time {
	stable := now.UTC()
	if stable.IsZero() {
		stable = taskStateNowUTC()
	}
	for _, lowerBound := range lowerBounds {
		lowerBound = lowerBound.UTC()
		if lowerBound.IsZero() {
			continue
		}
		if lowerBound.After(stable) {
			stable = lowerBound
		}
	}
	return stable
}

func formatTaskStateRFC3339(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
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

func (s *TaskState) RecordRollbackFromPromotion(jobID string, promotionID string, rollbackID string) error {
	if s == nil {
		return nil
	}

	now := taskStateTransitionTimestamp(taskStateNowUTC())

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	root := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
		return err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
			return err
		}
	}

	promotion, err := missioncontrol.LoadPromotionRecord(root, promotionID)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
		return err
	}

	record := missioncontrol.RollbackRecord{
		RollbackID:          rollbackID,
		PromotionID:         promotion.PromotionID,
		HotUpdateID:         promotion.HotUpdateID,
		OutcomeID:           promotion.OutcomeID,
		FromPackID:          promotion.PromotedPackID,
		TargetPackID:        promotion.PreviousActivePackID,
		LastKnownGoodPackID: promotion.LastKnownGoodPackID,
		Reason:              "operator requested rollback proposal",
		Notes:               fmt.Sprintf("derived from promotion %s", promotion.PromotionID),
		RollbackAt:          now,
		CreatedAt:           now,
		CreatedBy:           "operator",
	}
	if err := missioncontrol.StoreRollbackRecord(root, record); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", err)
		return err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_record", nil)
	return nil
}

func (s *TaskState) AdvanceRollbackApplyPhase(jobID string, applyID string, phase string) (bool, error) {
	if s == nil {
		return false, nil
	}

	now := taskStateTransitionTimestamp(taskStateNowUTC())

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	root := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.AdvanceRollbackApplyPhase(root, applyID, missioncontrol.RollbackApplyPhase(strings.TrimSpace(phase)), "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_phase", nil)
	return changed, nil
}

func (s *TaskState) ExecuteRollbackApplyPointerSwitch(jobID string, applyID string) (bool, error) {
	if s == nil {
		return false, nil
	}

	now := taskStateTransitionTimestamp(taskStateNowUTC())

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	root := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.ExecuteRollbackApplyPointerSwitch(root, applyID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_execute", nil)
	return changed, nil
}

func (s *TaskState) ExecuteRollbackApplyReloadApply(jobID string, applyID string) (bool, error) {
	if s == nil {
		return false, nil
	}

	now := taskStateTransitionTimestamp(taskStateNowUTC())

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	root := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.ExecuteRollbackApplyReloadApply(root, applyID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_reload", nil)
	return changed, nil
}

func (s *TaskState) ResolveRollbackApplyTerminalFailure(jobID string, applyID string, reason string) (bool, error) {
	if s == nil {
		return false, nil
	}

	now := taskStateTransitionTimestamp(taskStateNowUTC())

	s.mu.Lock()
	ec := missioncontrol.CloneExecutionContext(s.executionContext)
	hasExecutionContext := s.hasExecutionContext
	control := missioncontrol.CloneRuntimeControlContext(&s.runtimeControl)
	hasRuntimeControl := s.hasRuntimeControl
	runtimeState := missioncontrol.CloneJobRuntimeState(&s.runtimeState)
	hasRuntimeState := s.hasRuntimeState
	root := strings.TrimSpace(s.missionStoreRoot)
	s.mu.Unlock()

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.ResolveRollbackApplyTerminalFailure(root, applyID, reason, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_fail", nil)
	return changed, nil
}
