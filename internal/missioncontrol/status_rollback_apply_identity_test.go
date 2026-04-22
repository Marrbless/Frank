package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorRollbackApplyIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-apply-root"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if err := StoreRollbackApplyRecord(root, validRollbackApplyRecord(now.Add(14*time.Minute), func(record *RollbackApplyRecord) {
		record.ApplyID = "apply-1"
		record.RollbackID = "rollback-apply-root"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackApplyRecord() error = %v", err)
	}

	got := LoadOperatorRollbackApplyIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Applies) != 1 {
		t.Fatalf("Applies len = %d, want 1", len(got.Applies))
	}
	apply := got.Applies[0]
	if apply.State != "configured" {
		t.Fatalf("Applies[0].State = %q, want configured", apply.State)
	}
	if apply.RollbackApplyID != "apply-1" {
		t.Fatalf("Applies[0].RollbackApplyID = %q, want apply-1", apply.RollbackApplyID)
	}
	if apply.RollbackID != "rollback-apply-root" {
		t.Fatalf("Applies[0].RollbackID = %q, want rollback-apply-root", apply.RollbackID)
	}
	if apply.Phase != string(RollbackApplyPhaseRecorded) {
		t.Fatalf("Applies[0].Phase = %q, want recorded", apply.Phase)
	}
	if apply.ActivationState != string(RollbackApplyActivationStateUnchanged) {
		t.Fatalf("Applies[0].ActivationState = %q, want unchanged", apply.ActivationState)
	}
	if apply.CreatedAt == nil || *apply.CreatedAt != "2026-04-30T19:15:00Z" {
		t.Fatalf("Applies[0].CreatedAt = %#v, want 2026-04-30T19:15:00Z", apply.CreatedAt)
	}
	if apply.CreatedBy != "operator" {
		t.Fatalf("Applies[0].CreatedBy = %q, want operator", apply.CreatedBy)
	}
	if apply.Error != "" {
		t.Fatalf("Applies[0].Error = %q, want empty", apply.Error)
	}
}

func TestLoadOperatorRollbackApplyIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorRollbackApplyIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Applies) != 0 {
		t.Fatalf("Applies len = %d, want 0", len(got.Applies))
	}
}

func TestLoadOperatorRollbackApplyIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 20, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, "apply-orphan"), validRollbackApplyRecord(now.Add(14*time.Minute), func(record *RollbackApplyRecord) {
		record.ApplyID = "apply-orphan"
		record.RollbackID = "rollback-orphan"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(apply-orphan) error = %v", err)
	}

	got := LoadOperatorRollbackApplyIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Applies) != 1 {
		t.Fatalf("Applies len = %d, want 1", len(got.Applies))
	}
	apply := got.Applies[0]
	if apply.State != "invalid" {
		t.Fatalf("Applies[0].State = %q, want invalid", apply.State)
	}
	if apply.RollbackApplyID != "apply-orphan" {
		t.Fatalf("Applies[0].RollbackApplyID = %q, want apply-orphan", apply.RollbackApplyID)
	}
	if apply.RollbackID != "rollback-orphan" {
		t.Fatalf("Applies[0].RollbackID = %q, want rollback-orphan", apply.RollbackID)
	}
	if !strings.Contains(apply.Error, `rollback_id "rollback-orphan"`) {
		t.Fatalf("Applies[0].Error = %q, want missing rollback_id context", apply.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesRollbackApplyIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := testLeaseSafeNow()
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

	storeRollbackFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(-30*time.Second), func(record *RollbackRecord) {
		record.RollbackID = "rollback-apply-root"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}
	if err := StoreRollbackApplyRecord(root, validRollbackApplyRecord(now.Add(-20*time.Second), func(record *RollbackApplyRecord) {
		record.ApplyID = "apply-1"
		record.RollbackID = "rollback-apply-root"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackApplyRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.RollbackApplyIdentity == nil {
		t.Fatalf("RuntimeSummary.RollbackApplyIdentity = %#v, want populated rollback-apply identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.RollbackApplyIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.RollbackApplyIdentity.State = %q, want configured", snapshot.RuntimeSummary.RollbackApplyIdentity.State)
	}
	if len(snapshot.RuntimeSummary.RollbackApplyIdentity.Applies) != 1 {
		t.Fatalf("RuntimeSummary.RollbackApplyIdentity.Applies len = %d, want 1", len(snapshot.RuntimeSummary.RollbackApplyIdentity.Applies))
	}
	if snapshot.RuntimeSummary.RollbackApplyIdentity.Applies[0].RollbackApplyID != "apply-1" {
		t.Fatalf("RuntimeSummary.RollbackApplyIdentity.Applies[0].RollbackApplyID = %q, want apply-1", snapshot.RuntimeSummary.RollbackApplyIdentity.Applies[0].RollbackApplyID)
	}
}

func validRollbackApplyRecord(now time.Time, mutate func(*RollbackApplyRecord)) RollbackApplyRecord {
	record := RollbackApplyRecord{
		RecordVersion:   StoreRecordVersion,
		ApplyID:         "apply-root",
		RollbackID:      "rollback-root",
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     now,
		CreatedAt:       now.Add(time.Minute),
		CreatedBy:       "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
