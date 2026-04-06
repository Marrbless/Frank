package missioncontrol

import (
	"fmt"
	"strings"
)

func LoadMissionStatusSnapshotFile(path string) (MissionStatusSnapshot, error) {
	return LoadMissionStatusObservation(path)
}

func LoadValidatedLegacyMissionStatusSnapshot(path string, jobID string) (MissionStatusSnapshot, bool, error) {
	if path == "" {
		return MissionStatusSnapshot{}, false, nil
	}

	snapshot, err := LoadMissionStatusSnapshotFile(path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return MissionStatusSnapshot{}, false, nil
		}
		return MissionStatusSnapshot{}, false, err
	}
	if snapshot.Runtime == nil {
		return MissionStatusSnapshot{}, false, nil
	}
	if err := validateLegacyMissionStatusSnapshot(snapshot); err != nil {
		return MissionStatusSnapshot{}, false, err
	}
	if snapshot.Runtime.JobID != "" && jobID != "" && snapshot.Runtime.JobID != jobID {
		return MissionStatusSnapshot{}, false, nil
	}
	if snapshot.RuntimeControl != nil && snapshot.RuntimeControl.JobID != "" && jobID != "" && snapshot.RuntimeControl.JobID != jobID {
		return MissionStatusSnapshot{}, false, nil
	}
	return snapshot, true, nil
}

func validateLegacyMissionStatusSnapshot(snapshot MissionStatusSnapshot) error {
	if snapshot.Runtime == nil {
		return nil
	}

	runtime := snapshot.Runtime
	if snapshot.JobID != "" && runtime.JobID != "" && snapshot.JobID != runtime.JobID {
		return fmt.Errorf("persisted mission runtime snapshot job_id %q does not match runtime job_id %q", snapshot.JobID, runtime.JobID)
	}
	if snapshot.StepID != "" && runtime.ActiveStepID != "" && snapshot.StepID != runtime.ActiveStepID {
		return fmt.Errorf("persisted mission runtime snapshot step_id %q does not match runtime active_step_id %q", snapshot.StepID, runtime.ActiveStepID)
	}
	if snapshot.Active && IsTerminalJobState(runtime.State) {
		return fmt.Errorf("persisted mission runtime snapshot marks terminal runtime state %q as active", runtime.State)
	}
	if runtime.ActiveStepID != "" && HasCompletedRuntimeStep(*runtime, runtime.ActiveStepID) {
		return fmt.Errorf("persisted mission runtime active_step_id %q is already recorded in completed_steps", runtime.ActiveStepID)
	}
	if runtime.ActiveStepID != "" && HasFailedRuntimeStep(*runtime, runtime.ActiveStepID) {
		return fmt.Errorf("persisted mission runtime active_step_id %q is already recorded in failed_steps", runtime.ActiveStepID)
	}

	control := snapshot.RuntimeControl
	if control == nil {
		return nil
	}
	if control.JobID != "" && runtime.JobID != "" && control.JobID != runtime.JobID {
		return fmt.Errorf("persisted mission runtime control job_id %q does not match runtime job_id %q", control.JobID, runtime.JobID)
	}
	if runtime.ActiveStepID == "" {
		if control.Step.ID != "" {
			return fmt.Errorf("persisted mission runtime control step_id %q requires runtime active_step_id", control.Step.ID)
		}
		return nil
	}
	if control.Step.ID == "" {
		return fmt.Errorf("persisted mission runtime active_step_id %q requires runtime control step_id", runtime.ActiveStepID)
	}
	if control.Step.ID != runtime.ActiveStepID {
		return fmt.Errorf("persisted mission runtime control step_id %q does not match runtime active_step_id %q", control.Step.ID, runtime.ActiveStepID)
	}
	return nil
}
