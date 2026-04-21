package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorRuntimePackIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 19, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
		record.SourceSummary = "previous active"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.RollbackTargetPackID = "pack-prev"
	}))

	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-active",
		PreviousActivePackID: "pack-prev",
		LastKnownGoodPackID:  "pack-prev",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "bootstrap",
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-prev",
		Basis:             "smoke_check",
		VerifiedAt:        now.Add(3 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	got := LoadOperatorRuntimePackIdentityStatus(root)
	if got.Active.State != "configured" {
		t.Fatalf("Active.State = %q, want configured", got.Active.State)
	}
	if got.Active.ActivePackID != "pack-active" {
		t.Fatalf("Active.ActivePackID = %q, want pack-active", got.Active.ActivePackID)
	}
	if got.Active.PreviousActivePackID != "pack-prev" {
		t.Fatalf("Active.PreviousActivePackID = %q, want pack-prev", got.Active.PreviousActivePackID)
	}
	if got.Active.LastKnownGoodPackID != "pack-prev" {
		t.Fatalf("Active.LastKnownGoodPackID = %q, want pack-prev", got.Active.LastKnownGoodPackID)
	}
	if got.Active.UpdatedAt == nil || *got.Active.UpdatedAt != "2026-04-21T19:02:00Z" {
		t.Fatalf("Active.UpdatedAt = %#v, want 2026-04-21T19:02:00Z", got.Active.UpdatedAt)
	}
	if got.Active.Error != "" {
		t.Fatalf("Active.Error = %q, want empty", got.Active.Error)
	}
	if got.LastKnownGood.State != "configured" {
		t.Fatalf("LastKnownGood.State = %q, want configured", got.LastKnownGood.State)
	}
	if got.LastKnownGood.PackID != "pack-prev" {
		t.Fatalf("LastKnownGood.PackID = %q, want pack-prev", got.LastKnownGood.PackID)
	}
	if got.LastKnownGood.Basis != "smoke_check" {
		t.Fatalf("LastKnownGood.Basis = %q, want smoke_check", got.LastKnownGood.Basis)
	}
	if got.LastKnownGood.VerifiedAt == nil || *got.LastKnownGood.VerifiedAt != "2026-04-21T19:03:00Z" {
		t.Fatalf("LastKnownGood.VerifiedAt = %#v, want 2026-04-21T19:03:00Z", got.LastKnownGood.VerifiedAt)
	}
	if got.LastKnownGood.Error != "" {
		t.Fatalf("LastKnownGood.Error = %q, want empty", got.LastKnownGood.Error)
	}
}

func TestLoadOperatorRuntimePackIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorRuntimePackIdentityStatus(t.TempDir())
	if got.Active.State != "not_configured" {
		t.Fatalf("Active.State = %q, want not_configured", got.Active.State)
	}
	if got.LastKnownGood.State != "not_configured" {
		t.Fatalf("LastKnownGood.State = %q, want not_configured", got.LastKnownGood.State)
	}
}

func TestLoadOperatorRuntimePackIdentityStatusInvalidMissingReferencedPack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	if err := WriteStoreJSONAtomic(StoreActiveRuntimePackPointerPath(root), ActiveRuntimePackPointer{
		RecordVersion:        StoreRecordVersion,
		ActivePackID:         "pack-missing",
		PreviousActivePackID: "pack-prev",
		LastKnownGoodPackID:  "pack-lkg",
		UpdatedAt:            now,
		UpdatedBy:            "operator",
		UpdateRecordRef:      "bootstrap",
		ReloadGeneration:     1,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(active pointer) error = %v", err)
	}
	if err := WriteStoreJSONAtomic(StoreLastKnownGoodRuntimePackPointerPath(root), LastKnownGoodRuntimePackPointer{
		RecordVersion:     StoreRecordVersion,
		PackID:            "pack-lkg",
		Basis:             "smoke_check",
		VerifiedAt:        now.Add(time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(last-known-good pointer) error = %v", err)
	}

	got := LoadOperatorRuntimePackIdentityStatus(root)
	if got.Active.State != "invalid" {
		t.Fatalf("Active.State = %q, want invalid", got.Active.State)
	}
	if got.Active.ActivePackID != "pack-missing" {
		t.Fatalf("Active.ActivePackID = %q, want pack-missing", got.Active.ActivePackID)
	}
	if got.Active.UpdatedAt == nil || *got.Active.UpdatedAt != "2026-04-21T20:00:00Z" {
		t.Fatalf("Active.UpdatedAt = %#v, want 2026-04-21T20:00:00Z", got.Active.UpdatedAt)
	}
	if !strings.Contains(got.Active.Error, `active_pack_id "pack-missing"`) {
		t.Fatalf("Active.Error = %q, want missing active_pack_id context", got.Active.Error)
	}
	if got.LastKnownGood.State != "invalid" {
		t.Fatalf("LastKnownGood.State = %q, want invalid", got.LastKnownGood.State)
	}
	if got.LastKnownGood.PackID != "pack-lkg" {
		t.Fatalf("LastKnownGood.PackID = %q, want pack-lkg", got.LastKnownGood.PackID)
	}
	if !strings.Contains(got.LastKnownGood.Error, `pack_id "pack-lkg"`) {
		t.Fatalf("LastKnownGood.Error = %q, want missing pack_id context", got.LastKnownGood.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesRuntimePackIdentity(t *testing.T) {
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

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(-10*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(-9*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.RollbackTargetPackID = "pack-prev"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-active",
		PreviousActivePackID: "pack-prev",
		LastKnownGoodPackID:  "pack-prev",
		UpdatedAt:            now.Add(-30 * time.Second),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "bootstrap",
		ReloadGeneration:     2,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-prev",
		Basis:             "smoke_check",
		VerifiedAt:        now.Add(-time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.RuntimePackIdentity == nil {
		t.Fatalf("RuntimeSummary.RuntimePackIdentity = %#v, want populated runtime pack identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.RuntimePackIdentity.Active.ActivePackID != "pack-active" {
		t.Fatalf("RuntimeSummary.RuntimePackIdentity.Active.ActivePackID = %q, want pack-active", snapshot.RuntimeSummary.RuntimePackIdentity.Active.ActivePackID)
	}
	if snapshot.RuntimeSummary.RuntimePackIdentity.LastKnownGood.PackID != "pack-prev" {
		t.Fatalf("RuntimeSummary.RuntimePackIdentity.LastKnownGood.PackID = %q, want pack-prev", snapshot.RuntimeSummary.RuntimePackIdentity.LastKnownGood.PackID)
	}
	wantActiveUpdatedAt := now.Add(-30 * time.Second).Format(time.RFC3339Nano)
	if got := snapshot.RuntimeSummary.RuntimePackIdentity.Active.UpdatedAt; got == nil || *got != wantActiveUpdatedAt {
		t.Fatalf("RuntimeSummary.RuntimePackIdentity.Active.UpdatedAt = %#v, want %q", got, wantActiveUpdatedAt)
	}
	wantLastKnownGoodVerifiedAt := now.Add(-time.Minute).Format(time.RFC3339Nano)
	if got := snapshot.RuntimeSummary.RuntimePackIdentity.LastKnownGood.VerifiedAt; got == nil || *got != wantLastKnownGoodVerifiedAt {
		t.Fatalf("RuntimeSummary.RuntimePackIdentity.LastKnownGood.VerifiedAt = %#v, want %q", got, wantLastKnownGoodVerifiedAt)
	}
}
