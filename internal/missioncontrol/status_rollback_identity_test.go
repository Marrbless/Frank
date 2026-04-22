package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorRollbackIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 19, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := StoreRollbackRecord(root, validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RollbackID = "rollback-2"
		record.Reason = "operator approved rollback"
		record.Notes = "read-only rollback status"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	got := LoadOperatorRollbackIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Rollbacks) != 1 {
		t.Fatalf("Rollbacks len = %d, want 1", len(got.Rollbacks))
	}
	rollback := got.Rollbacks[0]
	if rollback.State != "configured" {
		t.Fatalf("Rollbacks[0].State = %q, want configured", rollback.State)
	}
	if rollback.RollbackID != "rollback-2" {
		t.Fatalf("Rollbacks[0].RollbackID = %q, want rollback-2", rollback.RollbackID)
	}
	if rollback.PromotionID != "promotion-1" {
		t.Fatalf("Rollbacks[0].PromotionID = %q, want promotion-1", rollback.PromotionID)
	}
	if rollback.HotUpdateID != "hot-update-1" {
		t.Fatalf("Rollbacks[0].HotUpdateID = %q, want hot-update-1", rollback.HotUpdateID)
	}
	if rollback.OutcomeID != "outcome-rollback" {
		t.Fatalf("Rollbacks[0].OutcomeID = %q, want outcome-rollback", rollback.OutcomeID)
	}
	if rollback.FromPackID != "pack-candidate" {
		t.Fatalf("Rollbacks[0].FromPackID = %q, want pack-candidate", rollback.FromPackID)
	}
	if rollback.TargetPackID != "pack-base" {
		t.Fatalf("Rollbacks[0].TargetPackID = %q, want pack-base", rollback.TargetPackID)
	}
	if rollback.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("Rollbacks[0].LastKnownGoodPackID = %q, want pack-base", rollback.LastKnownGoodPackID)
	}
	if rollback.Reason != "operator approved rollback" {
		t.Fatalf("Rollbacks[0].Reason = %q, want operator approved rollback", rollback.Reason)
	}
	if rollback.Notes != "read-only rollback status" {
		t.Fatalf("Rollbacks[0].Notes = %q, want read-only rollback status", rollback.Notes)
	}
	if rollback.RollbackAt == nil || *rollback.RollbackAt != "2026-04-29T19:12:00Z" {
		t.Fatalf("Rollbacks[0].RollbackAt = %#v, want 2026-04-29T19:12:00Z", rollback.RollbackAt)
	}
	if rollback.CreatedAt == nil || *rollback.CreatedAt != "2026-04-29T19:13:00Z" {
		t.Fatalf("Rollbacks[0].CreatedAt = %#v, want 2026-04-29T19:13:00Z", rollback.CreatedAt)
	}
	if rollback.CreatedBy != "operator" {
		t.Fatalf("Rollbacks[0].CreatedBy = %q, want operator", rollback.CreatedBy)
	}
	if rollback.Error != "" {
		t.Fatalf("Rollbacks[0].Error = %q, want empty", rollback.Error)
	}
}

func TestLoadOperatorRollbackIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorRollbackIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Rollbacks) != 0 {
		t.Fatalf("Rollbacks len = %d, want 0", len(got.Rollbacks))
	}
}

func TestLoadOperatorRollbackIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 20, 0, 0, 0, time.UTC)
	storeRollbackFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StoreRollbackPath(root, "rollback-bad"), validRollbackRecord(now.Add(12*time.Minute), func(record *RollbackRecord) {
		record.RecordVersion = StoreRecordVersion
		record.RollbackID = "rollback-bad"
		record.TargetPackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(rollback-bad) error = %v", err)
	}

	got := LoadOperatorRollbackIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Rollbacks) != 1 {
		t.Fatalf("Rollbacks len = %d, want 1", len(got.Rollbacks))
	}
	rollback := got.Rollbacks[0]
	if rollback.State != "invalid" {
		t.Fatalf("Rollbacks[0].State = %q, want invalid", rollback.State)
	}
	if rollback.RollbackID != "rollback-bad" {
		t.Fatalf("Rollbacks[0].RollbackID = %q, want rollback-bad", rollback.RollbackID)
	}
	if rollback.TargetPackID != "pack-missing" {
		t.Fatalf("Rollbacks[0].TargetPackID = %q, want pack-missing", rollback.TargetPackID)
	}
	if !strings.Contains(rollback.Error, `target_pack_id "pack-missing"`) {
		t.Fatalf("Rollbacks[0].Error = %q, want missing target_pack_id context", rollback.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesRollbackIdentity(t *testing.T) {
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
		record.RollbackID = "rollback-2"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreRollbackRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.RollbackIdentity == nil {
		t.Fatalf("RuntimeSummary.RollbackIdentity = %#v, want populated rollback identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.RollbackIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.RollbackIdentity.State = %q, want configured", snapshot.RuntimeSummary.RollbackIdentity.State)
	}
	if len(snapshot.RuntimeSummary.RollbackIdentity.Rollbacks) != 1 {
		t.Fatalf("RuntimeSummary.RollbackIdentity.Rollbacks len = %d, want 1", len(snapshot.RuntimeSummary.RollbackIdentity.Rollbacks))
	}
	if snapshot.RuntimeSummary.RollbackIdentity.Rollbacks[0].RollbackID != "rollback-2" {
		t.Fatalf("RuntimeSummary.RollbackIdentity.Rollbacks[0].RollbackID = %q, want rollback-2", snapshot.RuntimeSummary.RollbackIdentity.Rollbacks[0].RollbackID)
	}
}
