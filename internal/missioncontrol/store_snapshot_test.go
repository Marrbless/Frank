package missioncontrol

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWriteMissionStatusSnapshotAtomicAndProjectedShareAtomicWriter(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateRunning,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		ActiveStepAt:    now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	originalWrite := missionStatusSnapshotWriteFileAtomic
	t.Cleanup(func() { missionStatusSnapshotWriteFileAtomic = originalWrite })

	calls := make([]string, 0, 2)
	missionStatusSnapshotWriteFileAtomic = func(path string, data []byte) error {
		calls = append(calls, path)
		if len(data) == 0 {
			t.Fatal("atomic writer data = empty, want encoded snapshot bytes")
		}
		return nil
	}

	livePath := filepath.Join(t.TempDir(), "live.json")
	if err := WriteMissionStatusSnapshotAtomic(livePath, MissionStatusSnapshot{
		MissionFile: "mission.json",
		JobID:       job.ID,
		StepID:      "build",
		UpdatedAt:   now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("WriteMissionStatusSnapshotAtomic() error = %v", err)
	}

	projectedPath := filepath.Join(t.TempDir(), "projected.json")
	if err := WriteProjectedMissionStatusSnapshot(projectedPath, root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteProjectedMissionStatusSnapshot() error = %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("atomic writer calls = %d, want 2", len(calls))
	}
	if calls[0] != livePath {
		t.Fatalf("atomic writer live path = %q, want %q", calls[0], livePath)
	}
	if calls[1] != projectedPath {
		t.Fatalf("atomic writer projected path = %q, want %q", calls[1], projectedPath)
	}
}

func TestLoadMissionStatusObservationAndLegacyFallbackShareDecodeHelper(t *testing.T) {
	originalRead := missionStatusSnapshotReadFile
	originalDecode := missionStatusSnapshotDecode
	t.Cleanup(func() {
		missionStatusSnapshotReadFile = originalRead
		missionStatusSnapshotDecode = originalDecode
	})

	missionStatusSnapshotReadFile = func(path string) ([]byte, error) {
		return []byte(`{"job_id":"job-1"}`), nil
	}

	calls := 0
	missionStatusSnapshotDecode = func(path string, data []byte) (MissionStatusSnapshot, error) {
		calls++
		if path != "status.json" {
			t.Fatalf("decode helper path = %q, want %q", path, "status.json")
		}
		return MissionStatusSnapshot{
			JobID:  "job-1",
			StepID: "build",
			Runtime: &JobRuntimeState{
				JobID:        "job-1",
				State:        JobStatePaused,
				ActiveStepID: "build",
			},
		}, nil
	}

	observation, err := LoadMissionStatusObservation("status.json")
	if err != nil {
		t.Fatalf("LoadMissionStatusObservation() error = %v", err)
	}
	if observation.JobID != "job-1" {
		t.Fatalf("LoadMissionStatusObservation().JobID = %q, want %q", observation.JobID, "job-1")
	}

	snapshot, ok, err := LoadValidatedLegacyMissionStatusSnapshot("status.json", "job-1")
	if err != nil {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = false, want true")
	}
	if snapshot.JobID != "job-1" {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot().JobID = %q, want %q", snapshot.JobID, "job-1")
	}
	if calls != 2 {
		t.Fatalf("shared decode helper calls = %d, want 2", calls)
	}
}

func TestLoadValidatedLegacyMissionStatusSnapshotMissingFileReturnsNotOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	snapshot, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err != nil {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %v, want nil", err)
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !reflect.DeepEqual(snapshot, MissionStatusSnapshot{}) {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() snapshot = %#v, want zero value", snapshot)
	}
}

func TestLoadValidatedLegacyMissionStatusSnapshotInvalidJSONReturnsDecodeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	originalRead := missionStatusSnapshotReadFile
	t.Cleanup(func() { missionStatusSnapshotReadFile = originalRead })

	missionStatusSnapshotReadFile = func(path string) ([]byte, error) {
		return []byte("{not-json"), nil
	}

	_, ok, err := LoadValidatedLegacyMissionStatusSnapshot(path, "job-1")
	if err == nil {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() error = nil, want non-nil")
	}
	if ok {
		t.Fatal("LoadValidatedLegacyMissionStatusSnapshot() ok = true, want false")
	}
	if !strings.Contains(err.Error(), "failed to decode mission status file") {
		t.Fatalf("LoadValidatedLegacyMissionStatusSnapshot() error = %q, want decode failure", err)
	}
}
