package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorImprovementCandidateIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 19, 0, 0, 0, time.UTC)

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
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-1"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	if err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(3*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.SourceWorkspaceRef = "workspace/runs/run-1"
		record.SourceSummary = "candidate from workspace"
		record.ValidationBasisRefs = []string{"eval/baseline", "eval/holdout"}
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}

	got := LoadOperatorImprovementCandidateIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Candidates) != 1 {
		t.Fatalf("Candidates len = %d, want 1", len(got.Candidates))
	}
	candidate := got.Candidates[0]
	if candidate.State != "configured" {
		t.Fatalf("Candidates[0].State = %q, want configured", candidate.State)
	}
	if candidate.CandidateID != "candidate-1" {
		t.Fatalf("Candidates[0].CandidateID = %q, want candidate-1", candidate.CandidateID)
	}
	if candidate.BaselinePackID != "pack-base" {
		t.Fatalf("Candidates[0].BaselinePackID = %q, want pack-base", candidate.BaselinePackID)
	}
	if candidate.CandidatePackID != "pack-candidate" {
		t.Fatalf("Candidates[0].CandidatePackID = %q, want pack-candidate", candidate.CandidatePackID)
	}
	if candidate.SourceWorkspaceRef != "workspace/runs/run-1" {
		t.Fatalf("Candidates[0].SourceWorkspaceRef = %q, want workspace/runs/run-1", candidate.SourceWorkspaceRef)
	}
	if candidate.SourceSummary != "candidate from workspace" {
		t.Fatalf("Candidates[0].SourceSummary = %q, want candidate from workspace", candidate.SourceSummary)
	}
	if got := strings.Join(candidate.ValidationBasisRefs, ","); got != "eval/baseline,eval/holdout" {
		t.Fatalf("Candidates[0].ValidationBasisRefs = %#v, want baseline+holdout refs", candidate.ValidationBasisRefs)
	}
	if candidate.HotUpdateID != "hot-update-1" {
		t.Fatalf("Candidates[0].HotUpdateID = %q, want hot-update-1", candidate.HotUpdateID)
	}
	if candidate.CreatedAt == nil || *candidate.CreatedAt != "2026-04-22T19:03:00Z" {
		t.Fatalf("Candidates[0].CreatedAt = %#v, want 2026-04-22T19:03:00Z", candidate.CreatedAt)
	}
	if candidate.CreatedBy != "operator" {
		t.Fatalf("Candidates[0].CreatedBy = %q, want operator", candidate.CreatedBy)
	}
	if candidate.Error != "" {
		t.Fatalf("Candidates[0].Error = %q, want empty", candidate.Error)
	}
}

func TestLoadOperatorImprovementCandidateIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorImprovementCandidateIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Candidates) != 0 {
		t.Fatalf("Candidates len = %d, want 0", len(got.Candidates))
	}
}

func TestLoadOperatorImprovementCandidateIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 22, 20, 0, 0, 0, time.UTC)

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	if err := WriteStoreJSONAtomic(StoreImprovementCandidatePath(root, "candidate-bad"), ImprovementCandidateRecord{
		RecordVersion:       StoreRecordVersion,
		CandidateID:         "candidate-bad",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-missing",
		SourceWorkspaceRef:  "workspace/runs/bad",
		SourceSummary:       "broken candidate",
		ValidationBasisRefs: []string{"eval/baseline"},
		HotUpdateID:         "",
		CreatedAt:           now,
		CreatedBy:           "operator",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(candidate-bad) error = %v", err)
	}

	got := LoadOperatorImprovementCandidateIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Candidates) != 1 {
		t.Fatalf("Candidates len = %d, want 1", len(got.Candidates))
	}
	candidate := got.Candidates[0]
	if candidate.State != "invalid" {
		t.Fatalf("Candidates[0].State = %q, want invalid", candidate.State)
	}
	if candidate.CandidateID != "candidate-bad" {
		t.Fatalf("Candidates[0].CandidateID = %q, want candidate-bad", candidate.CandidateID)
	}
	if candidate.BaselinePackID != "pack-base" {
		t.Fatalf("Candidates[0].BaselinePackID = %q, want pack-base", candidate.BaselinePackID)
	}
	if candidate.CandidatePackID != "pack-missing" {
		t.Fatalf("Candidates[0].CandidatePackID = %q, want pack-missing", candidate.CandidatePackID)
	}
	if !strings.Contains(candidate.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Candidates[0].Error = %q, want missing candidate_pack_id context", candidate.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesImprovementCandidateIdentity(t *testing.T) {
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
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(-9*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
	if err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(-30*time.Second), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.SourceSummary = "candidate from committed snapshot"
		record.SourceWorkspaceRef = ""
	})); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.ImprovementCandidateIdentity == nil {
		t.Fatalf("RuntimeSummary.ImprovementCandidateIdentity = %#v, want populated improvement candidate identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.ImprovementCandidateIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.ImprovementCandidateIdentity.State = %q, want configured", snapshot.RuntimeSummary.ImprovementCandidateIdentity.State)
	}
	if len(snapshot.RuntimeSummary.ImprovementCandidateIdentity.Candidates) != 1 {
		t.Fatalf("RuntimeSummary.ImprovementCandidateIdentity.Candidates len = %d, want 1", len(snapshot.RuntimeSummary.ImprovementCandidateIdentity.Candidates))
	}
	if snapshot.RuntimeSummary.ImprovementCandidateIdentity.Candidates[0].CandidateID != "candidate-1" {
		t.Fatalf("RuntimeSummary.ImprovementCandidateIdentity.Candidates[0].CandidateID = %q, want candidate-1", snapshot.RuntimeSummary.ImprovementCandidateIdentity.Candidates[0].CandidateID)
	}
}
