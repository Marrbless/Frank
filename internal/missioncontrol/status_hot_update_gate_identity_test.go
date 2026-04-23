package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateGateIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	storeHotUpdateGateIdentityFixtures(t, root, now)
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-2"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.TargetSurfaces = []string{"skills"}
		record.SurfaceClasses = []string{"class_1"}
		record.ReloadMode = HotUpdateReloadModeSkillReload
		record.CompatibilityContractRef = "compat-v1"
		record.PreparedAt = now.Add(3 * time.Minute)
		record.State = HotUpdateGateStatePrepared
		record.Decision = HotUpdateGateDecisionKeepStaged
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}

	got := LoadOperatorHotUpdateGateIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Gates) != 1 {
		t.Fatalf("Gates len = %d, want 1", len(got.Gates))
	}
	gate := got.Gates[0]
	if gate.HotUpdateID != "hot-update-2" {
		t.Fatalf("Gates[0].HotUpdateID = %q, want hot-update-2", gate.HotUpdateID)
	}
	if gate.CandidatePackID != "pack-candidate" {
		t.Fatalf("Gates[0].CandidatePackID = %q, want pack-candidate", gate.CandidatePackID)
	}
	if gate.PreviousActivePackID != "pack-base" {
		t.Fatalf("Gates[0].PreviousActivePackID = %q, want pack-base", gate.PreviousActivePackID)
	}
	if gate.RollbackTargetPackID != "pack-base" {
		t.Fatalf("Gates[0].RollbackTargetPackID = %q, want pack-base", gate.RollbackTargetPackID)
	}
	if got := strings.Join(gate.TargetSurfaces, ","); got != "skills" {
		t.Fatalf("Gates[0].TargetSurfaces = %#v, want [skills]", gate.TargetSurfaces)
	}
	if got := strings.Join(gate.SurfaceClasses, ","); got != "class_1" {
		t.Fatalf("Gates[0].SurfaceClasses = %#v, want [class_1]", gate.SurfaceClasses)
	}
	if gate.ReloadMode != string(HotUpdateReloadModeSkillReload) {
		t.Fatalf("Gates[0].ReloadMode = %q, want skill_reload", gate.ReloadMode)
	}
	if gate.CompatibilityContractRef != "compat-v1" {
		t.Fatalf("Gates[0].CompatibilityContractRef = %q, want compat-v1", gate.CompatibilityContractRef)
	}
	if gate.PreparedAt == nil || *gate.PreparedAt != "2026-04-30T19:03:00Z" {
		t.Fatalf("Gates[0].PreparedAt = %#v, want 2026-04-30T19:03:00Z", gate.PreparedAt)
	}
	if gate.State != string(HotUpdateGateStatePrepared) {
		t.Fatalf("Gates[0].State = %q, want prepared", gate.State)
	}
	if gate.Decision != string(HotUpdateGateDecisionKeepStaged) {
		t.Fatalf("Gates[0].Decision = %q, want keep_staged", gate.Decision)
	}
	if gate.Error != "" {
		t.Fatalf("Gates[0].Error = %q, want empty", gate.Error)
	}
}

func TestLoadOperatorHotUpdateGateIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateGateIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Gates) != 0 {
		t.Fatalf("Gates len = %d, want 0", len(got.Gates))
	}
}

func TestLoadOperatorHotUpdateGateIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 30, 20, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, "hot-update-bad"), HotUpdateGateRecord{
		RecordVersion:            StoreRecordVersion,
		HotUpdateID:              "hot-update-bad",
		Objective:                "broken linkage",
		CandidatePackID:          "pack-missing",
		PreviousActivePackID:     "pack-base",
		RollbackTargetPackID:     "pack-base",
		TargetSurfaces:           []string{"skills"},
		SurfaceClasses:           []string{"class_1"},
		ReloadMode:               HotUpdateReloadModeSkillReload,
		CompatibilityContractRef: "compat-v1",
		PreparedAt:               now,
		State:                    HotUpdateGateStatePrepared,
		Decision:                 HotUpdateGateDecisionKeepStaged,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(hot-update-bad) error = %v", err)
	}

	got := LoadOperatorHotUpdateGateIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Gates) != 1 {
		t.Fatalf("Gates len = %d, want 1", len(got.Gates))
	}
	gate := got.Gates[0]
	if gate.HotUpdateID != "hot-update-bad" {
		t.Fatalf("Gates[0].HotUpdateID = %q, want hot-update-bad", gate.HotUpdateID)
	}
	if gate.CandidatePackID != "pack-missing" {
		t.Fatalf("Gates[0].CandidatePackID = %q, want pack-missing", gate.CandidatePackID)
	}
	if gate.State != string(HotUpdateGateStatePrepared) {
		t.Fatalf("Gates[0].State = %q, want prepared", gate.State)
	}
	if !strings.Contains(gate.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Gates[0].Error = %q, want missing candidate_pack_id context", gate.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateGateIdentity(t *testing.T) {
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

	storeHotUpdateGateIdentityFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(-30*time.Second), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-2"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateGateIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateGateIdentity = %#v, want populated hot-update gate identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.HotUpdateGateIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.HotUpdateGateIdentity.State = %q, want configured", snapshot.RuntimeSummary.HotUpdateGateIdentity.State)
	}
	if len(snapshot.RuntimeSummary.HotUpdateGateIdentity.Gates) != 1 {
		t.Fatalf("RuntimeSummary.HotUpdateGateIdentity.Gates len = %d, want 1", len(snapshot.RuntimeSummary.HotUpdateGateIdentity.Gates))
	}
	if snapshot.RuntimeSummary.HotUpdateGateIdentity.Gates[0].HotUpdateID != "hot-update-2" {
		t.Fatalf("RuntimeSummary.HotUpdateGateIdentity.Gates[0].HotUpdateID = %q, want hot-update-2", snapshot.RuntimeSummary.HotUpdateGateIdentity.Gates[0].HotUpdateID)
	}
}

func storeHotUpdateGateIdentityFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.SourceSummary = "baseline pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "candidate pack"
	}))
}
