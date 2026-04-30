package tools

import (
	"strings"

	"github.com/local/picobot/internal/missioncontrol"
)

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
	cloned := missioncontrol.CloneJob(job)
	if cloned != nil {
		cloned.MissionStoreRoot = strings.TrimSpace(s.missionStoreRoot)
		s.missionJob = *cloned
	}
	s.hasMissionJob = true
}

func (s *TaskState) withMissionStoreRootLocked(ec missioncontrol.ExecutionContext) missioncontrol.ExecutionContext {
	if strings.TrimSpace(s.missionStoreRoot) == "" {
		return ec
	}
	ec.MissionStoreRoot = s.missionStoreRoot
	return ec
}
