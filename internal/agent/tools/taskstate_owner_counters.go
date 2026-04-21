package tools

import (
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

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
