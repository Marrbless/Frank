package missioncontrol

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadValidatedLegacyMissionStatusSnapshotRejectsInconsistentStepEnvelope(t *testing.T) {
	path := writeLegacyMissionStatusSnapshotFile(t, MissionStatusSnapshot{
		MissionFile: "mission.json",
		JobID:       "job-1",
		StepID:      "final",
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStatePaused,
			ActiveStepID: "build",
			PausedReason: RuntimePauseReasonOperatorCommand,
		},
	})

	_, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err == nil {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() error = nil, want step envelope mismatch")
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !strings.Contains(err.Error(), `snapshot step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %q, want step envelope mismatch", err)
	}
}

func TestLoadValidatedLegacyMissionStatusSnapshotRejectsRuntimeControlMismatch(t *testing.T) {
	path := writeLegacyMissionStatusSnapshotFile(t, MissionStatusSnapshot{
		MissionFile: "mission.json",
		JobID:       "job-1",
		StepID:      "build",
		RuntimeControl: &RuntimeControlContext{
			JobID:        "job-1",
			MaxAuthority: AuthorityTierHigh,
			AllowedTools: []string{"read"},
			Step: Step{
				ID:   "final",
				Type: StepTypeFinalResponse,
			},
		},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStatePaused,
			ActiveStepID: "build",
			PausedReason: RuntimePauseReasonOperatorCommand,
		},
	})

	_, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err == nil {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() error = nil, want runtime-control mismatch")
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !strings.Contains(err.Error(), `runtime control step_id "final" does not match runtime active_step_id "build"`) {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %q, want runtime-control mismatch", err)
	}
}

func TestLoadValidatedLegacyMissionStatusSnapshotRejectsActiveCompletedStepRecord(t *testing.T) {
	path := writeLegacyMissionStatusSnapshotFile(t, MissionStatusSnapshot{
		MissionFile: "mission.json",
		JobID:       "job-1",
		StepID:      "build",
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStatePaused,
			ActiveStepID: "build",
			PausedReason: RuntimePauseReasonOperatorCommand,
			CompletedSteps: []RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
		},
	})

	_, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err == nil {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() error = nil, want completed-step replay marker failure")
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in completed_steps`) {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %q, want completed-step replay marker mismatch", err)
	}
}

func TestLoadValidatedLegacyMissionStatusSnapshotRejectsActiveFailedStepRecord(t *testing.T) {
	path := writeLegacyMissionStatusSnapshotFile(t, MissionStatusSnapshot{
		MissionFile: "mission.json",
		JobID:       "job-1",
		StepID:      "build",
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStatePaused,
			ActiveStepID: "build",
			PausedReason: RuntimePauseReasonOperatorCommand,
			FailedSteps: []RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)},
			},
		},
	})

	_, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err == nil {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() error = nil, want failed-step replay marker failure")
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !strings.Contains(err.Error(), `active_step_id "build" is already recorded in failed_steps`) {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %q, want failed-step replay marker mismatch", err)
	}
}

func writeLegacyMissionStatusSnapshotFile(t *testing.T, snapshot MissionStatusSnapshot) string {
	t.Helper()

	path := t.TempDir() + "/status.json"
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
