package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorImprovementRunIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 19, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	got := LoadOperatorImprovementRunIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("Runs len = %d, want 1", len(got.Runs))
	}
	run := got.Runs[0]
	if run.State != "configured" {
		t.Fatalf("Runs[0].State = %q, want configured", run.State)
	}
	if run.RunID != "run-1" {
		t.Fatalf("Runs[0].RunID = %q, want run-1", run.RunID)
	}
	if run.CandidateID != "candidate-1" {
		t.Fatalf("Runs[0].CandidateID = %q, want candidate-1", run.CandidateID)
	}
	if run.EvalSuiteID != "eval-suite-1" {
		t.Fatalf("Runs[0].EvalSuiteID = %q, want eval-suite-1", run.EvalSuiteID)
	}
	if run.BaselinePackID != "pack-base" {
		t.Fatalf("Runs[0].BaselinePackID = %q, want pack-base", run.BaselinePackID)
	}
	if run.CandidatePackID != "pack-candidate" {
		t.Fatalf("Runs[0].CandidatePackID = %q, want pack-candidate", run.CandidatePackID)
	}
	if run.HotUpdateID != "hot-update-1" {
		t.Fatalf("Runs[0].HotUpdateID = %q, want hot-update-1", run.HotUpdateID)
	}
	if run.CreatedAt == nil || *run.CreatedAt != "2026-04-23T19:05:00Z" {
		t.Fatalf("Runs[0].CreatedAt = %#v, want 2026-04-23T19:05:00Z", run.CreatedAt)
	}
	if run.CompletedAt == nil || *run.CompletedAt != "2026-04-23T19:06:00Z" {
		t.Fatalf("Runs[0].CompletedAt = %#v, want 2026-04-23T19:06:00Z", run.CompletedAt)
	}
	if run.CreatedBy != "operator" {
		t.Fatalf("Runs[0].CreatedBy = %q, want operator", run.CreatedBy)
	}
	if run.Error != "" {
		t.Fatalf("Runs[0].Error = %q, want empty", run.Error)
	}
}

func TestLoadOperatorImprovementRunIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorImprovementRunIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Runs) != 0 {
		t.Fatalf("Runs len = %d, want 0", len(got.Runs))
	}
}

func TestLoadOperatorImprovementRunIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 20, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StoreImprovementRunPath(root, "run-bad"), validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RecordVersion = StoreRecordVersion
		record.RunID = "run-bad"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(run-bad) error = %v", err)
	}

	got := LoadOperatorImprovementRunIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("Runs len = %d, want 1", len(got.Runs))
	}
	run := got.Runs[0]
	if run.State != "invalid" {
		t.Fatalf("Runs[0].State = %q, want invalid", run.State)
	}
	if run.RunID != "run-bad" {
		t.Fatalf("Runs[0].RunID = %q, want run-bad", run.RunID)
	}
	if run.CandidatePackID != "pack-missing" {
		t.Fatalf("Runs[0].CandidatePackID = %q, want pack-missing", run.CandidatePackID)
	}
	if !strings.Contains(run.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Runs[0].Error = %q, want missing candidate_pack_id context", run.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesImprovementRunIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 21, 0, 0, 0, time.UTC)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	storeImprovementRunFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(-30*time.Second), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.ImprovementRunIdentity == nil {
		t.Fatalf("RuntimeSummary.ImprovementRunIdentity = %#v, want populated improvement run identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.ImprovementRunIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.ImprovementRunIdentity.State = %q, want configured", snapshot.RuntimeSummary.ImprovementRunIdentity.State)
	}
	if len(snapshot.RuntimeSummary.ImprovementRunIdentity.Runs) != 1 {
		t.Fatalf("RuntimeSummary.ImprovementRunIdentity.Runs len = %d, want 1", len(snapshot.RuntimeSummary.ImprovementRunIdentity.Runs))
	}
	if snapshot.RuntimeSummary.ImprovementRunIdentity.Runs[0].RunID != "run-1" {
		t.Fatalf("RuntimeSummary.ImprovementRunIdentity.Runs[0].RunID = %q, want run-1", snapshot.RuntimeSummary.ImprovementRunIdentity.Runs[0].RunID)
	}
}
