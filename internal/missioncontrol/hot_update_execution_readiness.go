package missioncontrol

import (
	"errors"
	"fmt"
	"strings"
)

type HotUpdateExecutionTransition string

const (
	HotUpdateExecutionTransitionPreparedGateCreate HotUpdateExecutionTransition = "prepared_gate_create"
	HotUpdateExecutionTransitionPhaseValidated     HotUpdateExecutionTransition = "phase_validated"
	HotUpdateExecutionTransitionPhaseStaged        HotUpdateExecutionTransition = "phase_staged"
	HotUpdateExecutionTransitionPointerSwitch      HotUpdateExecutionTransition = "pointer_switch"
	HotUpdateExecutionTransitionReloadApply        HotUpdateExecutionTransition = "reload_apply"
	HotUpdateExecutionTransitionTerminalFailure    HotUpdateExecutionTransition = "terminal_failure"
	HotUpdateExecutionTransitionOutcomeCreate      HotUpdateExecutionTransition = "outcome_create"
	HotUpdateExecutionTransitionPromotionCreate    HotUpdateExecutionTransition = "promotion_create"
	HotUpdateExecutionTransitionLKGRecertify       HotUpdateExecutionTransition = "lkg_recertify"
)

type HotUpdateExecutionTransitionClass string

const (
	HotUpdateExecutionTransitionClassMetadata         HotUpdateExecutionTransitionClass = "metadata"
	HotUpdateExecutionTransitionClassExecution        HotUpdateExecutionTransitionClass = "execution"
	HotUpdateExecutionTransitionClassMetadataRecovery HotUpdateExecutionTransitionClass = "metadata_recovery"
	HotUpdateExecutionTransitionClassLedger           HotUpdateExecutionTransitionClass = "ledger"
	HotUpdateExecutionTransitionClassOutside          HotUpdateExecutionTransitionClass = "outside_hot_update_execution_readiness"
)

type HotUpdateQuiesceState string

const (
	HotUpdateQuiesceStateNotConfigured HotUpdateQuiesceState = "not_configured"
	HotUpdateQuiesceStateUnknown       HotUpdateQuiesceState = "unknown"
	HotUpdateQuiesceStateReady         HotUpdateQuiesceState = "ready"
	HotUpdateQuiesceStateFailed        HotUpdateQuiesceState = "failed"
)

type HotUpdateExecutionReplayClass string

const (
	HotUpdateExecutionReplayClassNone                        HotUpdateExecutionReplayClass = "none"
	HotUpdateExecutionReplayClassNotApplicable               HotUpdateExecutionReplayClass = "not_applicable"
	HotUpdateExecutionReplayClassPointerSwitchAlreadyApplied HotUpdateExecutionReplayClass = "pointer_switch_already_applied"
	HotUpdateExecutionReplayClassReloadApplyAlreadySucceeded HotUpdateExecutionReplayClass = "reload_apply_already_succeeded"
)

type HotUpdateExecutionReadinessInput struct {
	Transition   HotUpdateExecutionTransition
	HotUpdateID  string
	CommandJobID string

	QuiesceState HotUpdateQuiesceState

	ActiveJob      *ActiveJobRecord
	JobRuntime     *JobRuntimeRecord
	RuntimeControl *RuntimeControlRecord
}

type HotUpdateExecutionReadinessAssessment struct {
	Transition           HotUpdateExecutionTransition      `json:"transition"`
	TransitionClass      HotUpdateExecutionTransitionClass `json:"transition_class"`
	HotUpdateID          string                            `json:"hot_update_id,omitempty"`
	GateState            HotUpdateGateState                `json:"gate_state,omitempty"`
	CommandJobID         string                            `json:"command_job_id,omitempty"`
	ExecutionSensitive   bool                              `json:"execution_sensitive"`
	Ready                bool                              `json:"ready"`
	RejectionCode        RejectionCode                     `json:"rejection_code,omitempty"`
	Reason               string                            `json:"reason,omitempty"`
	ActiveJobConsidered  bool                              `json:"active_job_considered"`
	ActiveJobID          string                            `json:"active_job_id,omitempty"`
	ActiveJobState       JobState                          `json:"active_job_state,omitempty"`
	ActiveExecutionPlane string                            `json:"active_execution_plane,omitempty"`
	ActiveMissionFamily  string                            `json:"active_mission_family,omitempty"`
	ActiveStepID         string                            `json:"active_step_id,omitempty"`
	QuiesceState         HotUpdateQuiesceState             `json:"quiesce_state"`
	ReplayClass          HotUpdateExecutionReplayClass     `json:"replay_class"`
}

func AssessHotUpdateExecutionReadiness(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReadinessAssessment, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}
	input.Transition = HotUpdateExecutionTransition(strings.TrimSpace(string(input.Transition)))
	input.HotUpdateID = strings.TrimSpace(input.HotUpdateID)
	input.CommandJobID = strings.TrimSpace(input.CommandJobID)
	if input.QuiesceState == "" {
		input.QuiesceState = HotUpdateQuiesceStateNotConfigured
	}
	if err := validateHotUpdateExecutionReadinessInput(input); err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}

	assessment := HotUpdateExecutionReadinessAssessment{
		Transition:         input.Transition,
		TransitionClass:    hotUpdateExecutionTransitionClass(input.Transition),
		HotUpdateID:        input.HotUpdateID,
		CommandJobID:       input.CommandJobID,
		QuiesceState:       input.QuiesceState,
		ReplayClass:        HotUpdateExecutionReplayClassNone,
		Ready:              true,
		ExecutionSensitive: hotUpdateExecutionTransitionIsSensitive(input.Transition),
	}
	if !assessment.ExecutionSensitive {
		assessment.ReplayClass = HotUpdateExecutionReplayClassNotApplicable
		assessment.Reason = "transition does not mutate hot-update execution state"
		return assessment, nil
	}

	replay, gateState, err := hotUpdateExecutionReplayClass(root, input)
	if err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}
	assessment.ReplayClass = replay
	assessment.GateState = gateState
	if replay != HotUpdateExecutionReplayClassNone {
		assessment.Reason = "exact execution replay is already applied"
		return assessment, nil
	}

	activeJob, found, err := loadHotUpdateExecutionActiveJob(root, input)
	if err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}
	if !found || !HoldsGlobalActiveJobOccupancy(activeJob.State) {
		assessment.Reason = "no active occupied job blocks hot-update execution"
		return assessment, nil
	}
	assessment.ActiveJobConsidered = true
	assessment.ActiveJobID = strings.TrimSpace(activeJob.JobID)
	assessment.ActiveJobState = activeJob.State
	assessment.ActiveStepID = strings.TrimSpace(activeJob.ActiveStepID)

	runtime, runtimeFound, err := loadHotUpdateExecutionRuntime(root, input, activeJob.JobID)
	if err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}
	if runtimeFound {
		assessment.ActiveExecutionPlane = strings.TrimSpace(runtime.ExecutionPlane)
		assessment.ActiveMissionFamily = strings.TrimSpace(runtime.MissionFamily)
		if strings.TrimSpace(runtime.ActiveStepID) != "" {
			assessment.ActiveStepID = strings.TrimSpace(runtime.ActiveStepID)
		}
	}
	control, controlFound, err := loadHotUpdateExecutionRuntimeControl(root, input, activeJob.JobID)
	if err != nil {
		return HotUpdateExecutionReadinessAssessment{}, err
	}
	if controlFound {
		if assessment.ActiveExecutionPlane == "" {
			assessment.ActiveExecutionPlane = strings.TrimSpace(control.ExecutionPlane)
		}
		if assessment.ActiveMissionFamily == "" {
			assessment.ActiveMissionFamily = strings.TrimSpace(control.MissionFamily)
		}
		if assessment.ActiveStepID == "" {
			assessment.ActiveStepID = strings.TrimSpace(control.StepID)
		}
	}

	if assessment.ActiveJobID == input.CommandJobID && assessment.ActiveExecutionPlane == ExecutionPlaneHotUpdateGate {
		assessment.Reason = "active occupied job is the same hot-update control job"
		return assessment, nil
	}
	if assessment.ActiveExecutionPlane == ExecutionPlaneLiveRuntime {
		switch input.QuiesceState {
		case HotUpdateQuiesceStateReady:
			assessment.Reason = "active live-runtime job has explicit quiesce readiness"
			return assessment, nil
		case HotUpdateQuiesceStateFailed:
			assessment.Ready = false
			assessment.RejectionCode = RejectionCodeV4ReloadQuiesceFailed
			assessment.Reason = "active live-runtime job failed quiesce readiness"
			return assessment, nil
		default:
			assessment.Ready = false
			assessment.RejectionCode = RejectionCodeV4ActiveJobDeployLock
			assessment.Reason = "active live-runtime job has no explicit quiesce readiness proof"
			return assessment, nil
		}
	}
	if assessment.ActiveExecutionPlane == "" {
		assessment.Ready = false
		assessment.RejectionCode = RejectionCodeV4ActiveJobDeployLock
		assessment.Reason = "active occupied job has no runtime execution-plane evidence"
		return assessment, nil
	}

	assessment.Reason = "active occupied job is not live-runtime hot-update work"
	return assessment, nil
}

func validateHotUpdateExecutionReadinessInput(input HotUpdateExecutionReadinessInput) error {
	switch input.Transition {
	case HotUpdateExecutionTransitionPreparedGateCreate,
		HotUpdateExecutionTransitionPhaseValidated,
		HotUpdateExecutionTransitionPhaseStaged,
		HotUpdateExecutionTransitionPointerSwitch,
		HotUpdateExecutionTransitionReloadApply,
		HotUpdateExecutionTransitionTerminalFailure,
		HotUpdateExecutionTransitionOutcomeCreate,
		HotUpdateExecutionTransitionPromotionCreate,
		HotUpdateExecutionTransitionLKGRecertify:
	default:
		return fmt.Errorf("mission store hot-update execution readiness transition %q is invalid", input.Transition)
	}
	switch input.QuiesceState {
	case HotUpdateQuiesceStateNotConfigured, HotUpdateQuiesceStateUnknown, HotUpdateQuiesceStateReady, HotUpdateQuiesceStateFailed:
	default:
		return fmt.Errorf("mission store hot-update execution readiness quiesce_state %q is invalid", input.QuiesceState)
	}
	if input.HotUpdateID != "" {
		if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: input.HotUpdateID}); err != nil {
			return err
		}
	}
	return nil
}

func hotUpdateExecutionTransitionClass(transition HotUpdateExecutionTransition) HotUpdateExecutionTransitionClass {
	switch transition {
	case HotUpdateExecutionTransitionPointerSwitch, HotUpdateExecutionTransitionReloadApply:
		return HotUpdateExecutionTransitionClassExecution
	case HotUpdateExecutionTransitionTerminalFailure:
		return HotUpdateExecutionTransitionClassMetadataRecovery
	case HotUpdateExecutionTransitionOutcomeCreate, HotUpdateExecutionTransitionPromotionCreate:
		return HotUpdateExecutionTransitionClassLedger
	case HotUpdateExecutionTransitionLKGRecertify:
		return HotUpdateExecutionTransitionClassOutside
	default:
		return HotUpdateExecutionTransitionClassMetadata
	}
}

func hotUpdateExecutionTransitionIsSensitive(transition HotUpdateExecutionTransition) bool {
	switch transition {
	case HotUpdateExecutionTransitionPointerSwitch, HotUpdateExecutionTransitionReloadApply:
		return true
	default:
		return false
	}
}

func hotUpdateExecutionReplayClass(root string, input HotUpdateExecutionReadinessInput) (HotUpdateExecutionReplayClass, HotUpdateGateState, error) {
	if input.HotUpdateID == "" {
		return HotUpdateExecutionReplayClassNone, "", nil
	}
	gate, err := LoadHotUpdateGateRecord(root, input.HotUpdateID)
	if err != nil {
		if errors.Is(err, ErrHotUpdateGateRecordNotFound) {
			return HotUpdateExecutionReplayClassNone, "", nil
		}
		return "", "", err
	}
	switch input.Transition {
	case HotUpdateExecutionTransitionPointerSwitch:
		if gate.State != HotUpdateGateStateReloading {
			return HotUpdateExecutionReplayClassNone, gate.State, nil
		}
		pointer, err := LoadActiveRuntimePackPointer(root)
		if err != nil {
			if errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
				return HotUpdateExecutionReplayClassNone, gate.State, nil
			}
			return "", "", err
		}
		if pointer.ActivePackID == gate.CandidatePackID && pointer.UpdateRecordRef == hotUpdateGatePointerUpdateRecordRef(gate.HotUpdateID) {
			return HotUpdateExecutionReplayClassPointerSwitchAlreadyApplied, gate.State, nil
		}
	case HotUpdateExecutionTransitionReloadApply:
		if gate.State == HotUpdateGateStateReloadApplySucceeded {
			return HotUpdateExecutionReplayClassReloadApplyAlreadySucceeded, gate.State, nil
		}
	}
	return HotUpdateExecutionReplayClassNone, gate.State, nil
}

func loadHotUpdateExecutionActiveJob(root string, input HotUpdateExecutionReadinessInput) (ActiveJobRecord, bool, error) {
	if input.ActiveJob != nil {
		return *input.ActiveJob, true, nil
	}
	activeJob, err := LoadActiveJobRecord(root)
	if err != nil {
		if errors.Is(err, ErrActiveJobRecordNotFound) {
			return ActiveJobRecord{}, false, nil
		}
		return ActiveJobRecord{}, false, err
	}
	return activeJob, true, nil
}

func loadHotUpdateExecutionRuntime(root string, input HotUpdateExecutionReadinessInput, jobID string) (JobRuntimeRecord, bool, error) {
	if input.JobRuntime != nil {
		return *input.JobRuntime, true, nil
	}
	runtime, err := LoadJobRuntimeRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrJobRuntimeRecordNotFound) {
			return JobRuntimeRecord{}, false, nil
		}
		return JobRuntimeRecord{}, false, err
	}
	return runtime, true, nil
}

func loadHotUpdateExecutionRuntimeControl(root string, input HotUpdateExecutionReadinessInput, jobID string) (RuntimeControlRecord, bool, error) {
	if input.RuntimeControl != nil {
		return *input.RuntimeControl, true, nil
	}
	control, err := LoadRuntimeControlRecord(root, jobID)
	if err != nil {
		if errors.Is(err, ErrRuntimeControlRecordNotFound) {
			return RuntimeControlRecord{}, false, nil
		}
		return RuntimeControlRecord{}, false, err
	}
	return control, true, nil
}
