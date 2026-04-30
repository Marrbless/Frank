package tools

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func (s *TaskState) EnsureHotUpdateGateRecord(jobID string, hotUpdateID string, candidatePackID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
			return false, err
		}
	}

	_, created, err := missioncontrol.EnsureHotUpdateGateRecordFromCandidate(root, hotUpdateID, candidatePackID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_record", nil)
	return created, nil
}

func (s *TaskState) CreateHotUpdateGateFromCandidatePromotionDecision(jobID string, promotionDecisionID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
			return false, err
		}
	}

	createdAt := now
	hotUpdateID := taskStateHotUpdateGateIDFromPromotionDecision(promotionDecisionID)
	if existing, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID); err == nil {
		createdAt = existing.PreparedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateGateRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
		return false, err
	}

	_, changed, err := missioncontrol.CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, "operator", createdAt)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_from_decision", nil)
	return changed, nil
}

func taskStateHotUpdateGateIDFromPromotionDecision(promotionDecisionID string) string {
	return "hot-update-" + strings.TrimSpace(promotionDecisionID)
}

func (s *TaskState) CreateHotUpdateGateFromCanarySatisfactionAuthority(jobID string, canarySatisfactionAuthorityID string, ownerApprovalDecisionID string) (missioncontrol.HotUpdateGateRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateGateRecord{}, false, nil
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

	const action = "hot_update_canary_gate_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateGateRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
	}

	authorityRef := missioncontrol.NormalizeHotUpdateCanarySatisfactionAuthorityRef(missioncontrol.HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: canarySatisfactionAuthorityID})
	if err := missioncontrol.ValidateHotUpdateCanarySatisfactionAuthorityRef(authorityRef); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateGateRecord{}, false, err
	}

	ownerApprovalDecisionID = strings.TrimSpace(ownerApprovalDecisionID)
	if ownerApprovalDecisionID != "" {
		decisionRef := missioncontrol.NormalizeHotUpdateOwnerApprovalDecisionRef(missioncontrol.HotUpdateOwnerApprovalDecisionRef{OwnerApprovalDecisionID: ownerApprovalDecisionID})
		if err := missioncontrol.ValidateHotUpdateOwnerApprovalDecisionRef(decisionRef); err != nil {
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateGateRecord{}, false, err
		}
		ownerApprovalDecisionID = decisionRef.OwnerApprovalDecisionID
	}

	hotUpdateID := missioncontrol.HotUpdateGateIDFromCanarySatisfactionAuthority(authorityRef.CanarySatisfactionAuthorityID)
	createdAt := now
	if existing, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID); err == nil {
		createdAt = existing.PreparedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateGateRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateGateRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateGateFromCanarySatisfactionAuthority(root, authorityRef.CanarySatisfactionAuthorityID, ownerApprovalDecisionID, "operator", createdAt)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateGateRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func (s *TaskState) CreateHotUpdateCanaryRequirementFromCandidateResult(jobID string, resultID string) (missioncontrol.HotUpdateCanaryRequirementRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, nil
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

	const action = "hot_update_canary_requirement_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
		}
	}

	createdAt := now
	canaryRequirementID := missioncontrol.HotUpdateCanaryRequirementIDFromResult(resultID)
	if existing, err := missioncontrol.LoadHotUpdateCanaryRequirementRecord(root, canaryRequirementID); err == nil {
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateCanaryRequirementRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, "operator", createdAt)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryRequirementRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func (s *TaskState) CreateHotUpdateCanaryEvidenceFromRequirement(jobID string, canaryRequirementID string, state missioncontrol.HotUpdateCanaryEvidenceState, observedAt time.Time, reason string) (missioncontrol.HotUpdateCanaryEvidenceRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, nil
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

	const action = "hot_update_canary_evidence_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
		}
	}

	observedAt = observedAt.UTC()
	if observedAt.IsZero() {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "hot-update canary evidence observed_at is required",
		}
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
	}
	state = missioncontrol.HotUpdateCanaryEvidenceState(strings.TrimSpace(string(state)))
	if !isTaskStateHotUpdateCanaryEvidenceState(state) {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: fmt.Sprintf("hot-update canary evidence state %q is invalid", state),
		}
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = taskStateDefaultHotUpdateCanaryEvidenceReason(state)
	}

	createdAt := now
	canaryEvidenceID := missioncontrol.HotUpdateCanaryEvidenceIDFromRequirementObservedAt(canaryRequirementID, observedAt)
	if existing, err := missioncontrol.LoadHotUpdateCanaryEvidenceRecord(root, canaryEvidenceID); err == nil {
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateCanaryEvidenceRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID, state, observedAt, "operator", createdAt, reason)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanaryEvidenceRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func isTaskStateHotUpdateCanaryEvidenceState(state missioncontrol.HotUpdateCanaryEvidenceState) bool {
	switch state {
	case missioncontrol.HotUpdateCanaryEvidenceStatePassed,
		missioncontrol.HotUpdateCanaryEvidenceStateFailed,
		missioncontrol.HotUpdateCanaryEvidenceStateBlocked,
		missioncontrol.HotUpdateCanaryEvidenceStateExpired:
		return true
	default:
		return false
	}
}

func taskStateDefaultHotUpdateCanaryEvidenceReason(state missioncontrol.HotUpdateCanaryEvidenceState) string {
	return "operator recorded hot-update canary evidence " + string(state)
}

func (s *TaskState) CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(jobID string, canaryRequirementID string) (missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, nil
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

	const action = "hot_update_canary_satisfaction_authority_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
		}
	}

	assessment, err := missioncontrol.AssessHotUpdateCanarySatisfaction(root, canaryRequirementID)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if assessment.State != "configured" {
		err := fmt.Errorf("mission store hot-update canary satisfaction authority requires configured canary satisfaction assessment, found state %q: %s", assessment.State, assessment.Error)
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if !taskStateCanarySatisfactionAuthorityAcceptsAssessmentState(assessment.SatisfactionState) {
		err := fmt.Errorf("mission store hot-update canary satisfaction authority requires satisfaction_state %q or %q, found %q", missioncontrol.HotUpdateCanarySatisfactionStateSatisfied, missioncontrol.HotUpdateCanarySatisfactionStateWaitingOwnerApproval, assessment.SatisfactionState)
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if strings.TrimSpace(assessment.SelectedCanaryEvidenceID) == "" {
		err := fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id is required")
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	authorityID := missioncontrol.HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(assessment.CanaryRequirementID, assessment.SelectedCanaryEvidenceID)
	createdAt := now
	if existing, err := missioncontrol.LoadHotUpdateCanarySatisfactionAuthorityRecord(root, authorityID); err == nil {
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, "operator", createdAt)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func taskStateCanarySatisfactionAuthorityAcceptsAssessmentState(state missioncontrol.HotUpdateCanarySatisfactionState) bool {
	switch state {
	case missioncontrol.HotUpdateCanarySatisfactionStateSatisfied,
		missioncontrol.HotUpdateCanarySatisfactionStateWaitingOwnerApproval:
		return true
	default:
		return false
	}
}

func (s *TaskState) CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(jobID string, canarySatisfactionAuthorityID string) (missioncontrol.HotUpdateOwnerApprovalRequestRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, nil
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

	const action = "hot_update_owner_approval_request_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
		}
	}

	authorityRef := missioncontrol.NormalizeHotUpdateCanarySatisfactionAuthorityRef(missioncontrol.HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: canarySatisfactionAuthorityID})
	if err := missioncontrol.ValidateHotUpdateCanarySatisfactionAuthorityRef(authorityRef); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	requestID := missioncontrol.HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authorityRef.CanarySatisfactionAuthorityID)
	createdAt := now
	if existing, err := missioncontrol.LoadHotUpdateOwnerApprovalRequestRecord(root, requestID); err == nil {
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateOwnerApprovalRequestRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authorityRef.CanarySatisfactionAuthorityID, "operator", createdAt)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func (s *TaskState) CreateHotUpdateOwnerApprovalDecisionFromRequest(jobID string, ownerApprovalRequestID string, decision missioncontrol.HotUpdateOwnerApprovalDecision, reason string) (missioncontrol.HotUpdateOwnerApprovalDecisionRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, nil
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

	const action = "hot_update_owner_approval_decision_create"

	auditEC := ec
	if !hasExecutionContext {
		auditEC = s.runtimeAuditContext(control, runtimeState)
	}

	if err := missioncontrol.ValidateStoreRoot(root); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, action, err)
			return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
		}
	}

	requestRef := missioncontrol.NormalizeHotUpdateOwnerApprovalRequestRef(missioncontrol.HotUpdateOwnerApprovalRequestRef{OwnerApprovalRequestID: ownerApprovalRequestID})
	if err := missioncontrol.ValidateHotUpdateOwnerApprovalRequestRef(requestRef); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	decision = missioncontrol.HotUpdateOwnerApprovalDecision(strings.TrimSpace(string(decision)))
	switch decision {
	case missioncontrol.HotUpdateOwnerApprovalDecisionGranted, missioncontrol.HotUpdateOwnerApprovalDecisionRejected:
	default:
		err := fmt.Errorf("mission store hot-update owner approval decision decision %q is invalid", decision)
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "hot-update owner approval decision " + string(decision)
	}

	decisionID := missioncontrol.HotUpdateOwnerApprovalDecisionIDFromRequest(requestRef.OwnerApprovalRequestID)
	decidedAt := now
	if existing, err := missioncontrol.LoadHotUpdateOwnerApprovalDecisionRecord(root, decisionID); err == nil {
		decidedAt = existing.DecidedAt
	} else if !errors.Is(err, missioncontrol.ErrHotUpdateOwnerApprovalDecisionRecordNotFound) {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	record, changed, err := missioncontrol.CreateHotUpdateOwnerApprovalDecisionFromRequest(root, requestRef.OwnerApprovalRequestID, decision, "operator", decidedAt, reason)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, action, err)
		return missioncontrol.HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, action, nil)
	return record, changed, nil
}

func (s *TaskState) RecordHotUpdateExecutionReady(jobID string, hotUpdateID string, ttlSeconds int, reason string) (missioncontrol.HotUpdateExecutionSafetyEvidenceRecord, bool, error) {
	if s == nil {
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, nil
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if ttlSeconds <= 0 {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "HOT_UPDATE_EXECUTION_READY ttl_seconds must be positive",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if ttlSeconds > 300 {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "HOT_UPDATE_EXECUTION_READY ttl_seconds must be <= 300",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
		}
	}

	activeJob, err := missioncontrol.LoadActiveJobRecord(root)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if activeJob.JobID != strings.TrimSpace(jobID) {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "hot-update execution readiness active job does not match requested job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if !missioncontrol.HoldsGlobalActiveJobOccupancy(activeJob.State) {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "hot-update execution readiness requires an active occupied job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	runtime, err := missioncontrol.LoadCommittedJobRuntimeRecord(root, jobID)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	controlRecord, err := missioncontrol.LoadCommittedRuntimeControlRecord(root, jobID)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if runtime.ExecutionPlane != missioncontrol.ExecutionPlaneLiveRuntime {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "hot-update execution readiness requires active live_runtime job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if strings.TrimSpace(controlRecord.ExecutionPlane) != "" && controlRecord.ExecutionPlane != missioncontrol.ExecutionPlaneLiveRuntime {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "hot-update execution readiness requires live_runtime control context",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if runtime.ActiveStepID != "" && runtime.ActiveStepID != activeJob.ActiveStepID {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "hot-update execution readiness runtime active step does not match active job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if controlRecord.StepID != "" && controlRecord.StepID != activeJob.ActiveStepID {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "hot-update execution readiness control step does not match active job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if controlRecord.AttemptID != "" && activeJob.AttemptID != "" && controlRecord.AttemptID != activeJob.AttemptID {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeStepValidationFailed,
			Message: "hot-update execution readiness control attempt does not match active job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if runtime.WriterEpoch != 0 && runtime.WriterEpoch != activeJob.WriterEpoch {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "hot-update execution readiness runtime writer epoch does not match active job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	if controlRecord.WriterEpoch != 0 && controlRecord.WriterEpoch != activeJob.WriterEpoch {
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: "hot-update execution readiness control writer epoch does not match active job",
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	gate, err := missioncontrol.LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}
	switch gate.State {
	case missioncontrol.HotUpdateGateStateStaged,
		missioncontrol.HotUpdateGateStateReloading,
		missioncontrol.HotUpdateGateStateReloadApplyRecoveryNeeded:
	default:
		err := missioncontrol.ValidationError{
			Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
			Message: fmt.Sprintf("hot-update execution readiness requires staged, reloading, or reload_apply_recovery_needed gate state, got %q", gate.State),
		}
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "operator asserted hot-update execution readiness"
	}
	createdAt := now
	expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)
	if evidenceID, idErr := missioncontrol.HotUpdateExecutionSafetyEvidenceID(hotUpdateID, activeJob.JobID); idErr == nil {
		if existing, loadErr := missioncontrol.LoadHotUpdateExecutionSafetyEvidenceRecord(root, evidenceID); loadErr == nil {
			existingTTL := int(existing.ExpiresAt.Sub(existing.CreatedAt).Seconds())
			if existingTTL == ttlSeconds &&
				existing.Reason == reason &&
				existing.DeployLockState == missioncontrol.HotUpdateDeployLockStateDeployUnlocked &&
				existing.QuiesceState == missioncontrol.HotUpdateQuiesceStateReady &&
				existing.ExpiresAt.After(now) {
				createdAt = existing.CreatedAt
				expiresAt = existing.ExpiresAt
			}
		} else if !errors.Is(loadErr, missioncontrol.ErrHotUpdateExecutionSafetyEvidenceRecordNotFound) {
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", loadErr)
			return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, loadErr
		}
	} else {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", idErr)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, idErr
	}
	record, changed, err := missioncontrol.EnsureHotUpdateExecutionReadyEvidence(root, hotUpdateID, activeJob, "operator", createdAt, expiresAt, reason)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", err)
		return missioncontrol.HotUpdateExecutionSafetyEvidenceRecord{}, false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_execution_ready", nil)
	return record, changed, nil
}

func (s *TaskState) AdvanceHotUpdateGatePhase(jobID string, hotUpdateID string, phase string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.AdvanceHotUpdateGatePhase(root, hotUpdateID, missioncontrol.HotUpdateGateState(strings.TrimSpace(phase)), "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_phase", nil)
	return changed, nil
}

func (s *TaskState) ExecuteHotUpdateGatePointerSwitch(jobID string, hotUpdateID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
			return false, err
		}
	}

	if err := taskStateAssessHotUpdateExecutionReadiness(root, missioncontrol.HotUpdateExecutionTransitionPointerSwitch, hotUpdateID, jobID); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
		return false, err
	}

	_, changed, err := missioncontrol.ExecuteHotUpdateGatePointerSwitch(root, hotUpdateID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_execute", nil)
	return changed, nil
}

func (s *TaskState) ExecuteHotUpdateGateReloadApply(jobID string, hotUpdateID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
			return false, err
		}
	}

	if err := taskStateAssessHotUpdateExecutionReadiness(root, missioncontrol.HotUpdateExecutionTransitionReloadApply, hotUpdateID, jobID); err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
		return false, err
	}

	_, changed, err := missioncontrol.ExecuteHotUpdateGateReloadApply(root, hotUpdateID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_reload", nil)
	return changed, nil
}

func taskStateAssessHotUpdateExecutionReadiness(root string, transition missioncontrol.HotUpdateExecutionTransition, hotUpdateID string, jobID string) error {
	assessment, err := missioncontrol.AssessHotUpdateExecutionReadiness(root, missioncontrol.HotUpdateExecutionReadinessInput{
		Transition:   transition,
		HotUpdateID:  hotUpdateID,
		CommandJobID: jobID,
	})
	if err != nil {
		return err
	}
	if assessment.Ready {
		return nil
	}
	code := assessment.RejectionCode
	if code == "" {
		code = missioncontrol.RejectionCodeInvalidRuntimeState
	}
	reason := strings.TrimSpace(assessment.Reason)
	if reason == "" {
		reason = "hot-update execution readiness blocked"
	}
	return missioncontrol.ValidationError{
		Code: code,
		Message: fmt.Sprintf(
			"hot-update execution readiness blocked hot_update_id=%s transition=%s reason=%s",
			strings.TrimSpace(hotUpdateID),
			transition,
			reason,
		),
	}
}

func (s *TaskState) ResolveHotUpdateGateTerminalFailure(jobID string, hotUpdateID string, reason string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.ResolveHotUpdateGateTerminalFailure(root, hotUpdateID, reason, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_gate_fail", nil)
	return changed, nil
}

func (s *TaskState) CreateHotUpdateOutcomeFromTerminalGate(jobID string, hotUpdateID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, "operator", now)
	if err != nil {
		outcome, loadErr := missioncontrol.LoadHotUpdateOutcomeRecord(root, taskStateHotUpdateOutcomeID(hotUpdateID))
		if loadErr == nil {
			_, changed, err = missioncontrol.CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, "operator", outcome.CreatedAt)
		}
		if err != nil {
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", err)
			return false, err
		}
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_outcome_create", nil)
	return changed, nil
}

func taskStateHotUpdateOutcomeID(hotUpdateID string) string {
	return "hot-update-outcome-" + strings.TrimSpace(hotUpdateID)
}

func (s *TaskState) CreatePromotionFromSuccessfulHotUpdateOutcome(jobID string, outcomeID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", now)
	if err != nil {
		outcome, loadOutcomeErr := missioncontrol.LoadHotUpdateOutcomeRecord(root, outcomeID)
		if loadOutcomeErr == nil {
			promotion, loadPromotionErr := missioncontrol.LoadPromotionRecord(root, taskStateHotUpdatePromotionID(outcome.HotUpdateID))
			if loadPromotionErr == nil {
				_, changed, err = missioncontrol.CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", promotion.CreatedAt)
			}
		}
		if err != nil {
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", err)
			return false, err
		}
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_promotion_create", nil)
	return changed, nil
}

func taskStateHotUpdatePromotionID(hotUpdateID string) string {
	return "hot-update-promotion-" + strings.TrimSpace(hotUpdateID)
}

func (s *TaskState) RecertifyLastKnownGoodFromPromotion(jobID string, promotionID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
			return false, err
		}
	}

	_, changed, err := missioncontrol.RecertifyLastKnownGoodFromPromotion(root, promotionID, "operator", now)
	if err != nil && strings.Contains(err.Error(), "already points to promoted pack but differs from deterministic recertification") {
		promotion, loadPromotionErr := missioncontrol.LoadPromotionRecord(root, promotionID)
		pointer, loadPointerErr := missioncontrol.LoadLastKnownGoodRuntimePackPointer(root)
		recertificationRef := "hot_update_promotion:" + strings.TrimSpace(promotionID)
		if loadPromotionErr == nil &&
			loadPointerErr == nil &&
			pointer.PackID == promotion.PromotedPackID &&
			pointer.Basis == recertificationRef &&
			pointer.RollbackRecordRef == recertificationRef {
			_, changed, err = missioncontrol.RecertifyLastKnownGoodFromPromotion(root, promotionID, "operator", pointer.VerifiedAt)
		}
	}
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "hot_update_lkg_recertify", nil)
	return changed, nil
}
