package tools

import (
	"strings"

	"github.com/local/picobot/internal/missioncontrol"
)

func (s *TaskState) EnsureRollbackApplyRecord(jobID string, rollbackID string, applyID string) (bool, error) {
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
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
		return false, err
	}

	if hasExecutionContext {
		if ec.Job == nil || ec.Step == nil || ec.Runtime == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
		if ec.Job.ID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
	} else {
		if !hasRuntimeState || runtimeState == nil {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires an active mission step",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
		if runtimeState.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
		if !hasRuntimeControl || control == nil || strings.TrimSpace(control.JobID) == "" {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeInvalidRuntimeState,
				Message: "operator command requires persisted mission control context",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
		if control.JobID != jobID {
			err := missioncontrol.ValidationError{
				Code:    missioncontrol.RejectionCodeStepValidationFailed,
				Message: "operator command does not match the active job",
			}
			s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
			return false, err
		}
	}

	_, created, err := missioncontrol.EnsureRollbackApplyRecordFromRollback(root, applyID, rollbackID, "operator", now)
	if err != nil {
		s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", err)
		return false, err
	}

	s.emitRuntimeControlAuditEvent(auditEC, "rollback_apply_record", nil)
	return created, nil
}
