package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateCanaryRequirementIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 19, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-1", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}

	got := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Requirements) != 1 {
		t.Fatalf("Requirements len = %d, want 1", len(got.Requirements))
	}
	requirement := got.Requirements[0]
	if requirement.State != "configured" {
		t.Fatalf("Requirements[0].State = %q, want configured", requirement.State)
	}
	if requirement.CanaryRequirementID != "hot-update-canary-requirement-result-1" {
		t.Fatalf("Requirements[0].CanaryRequirementID = %q, want hot-update-canary-requirement-result-1", requirement.CanaryRequirementID)
	}
	if requirement.ResultID != "result-1" {
		t.Fatalf("Requirements[0].ResultID = %q, want result-1", requirement.ResultID)
	}
	if requirement.RunID != "run-result" {
		t.Fatalf("Requirements[0].RunID = %q, want run-result", requirement.RunID)
	}
	if requirement.CandidateID != "candidate-1" {
		t.Fatalf("Requirements[0].CandidateID = %q, want candidate-1", requirement.CandidateID)
	}
	if requirement.EvalSuiteID != "eval-suite-1" {
		t.Fatalf("Requirements[0].EvalSuiteID = %q, want eval-suite-1", requirement.EvalSuiteID)
	}
	if requirement.PromotionPolicyID != "promotion-policy-result" {
		t.Fatalf("Requirements[0].PromotionPolicyID = %q, want promotion-policy-result", requirement.PromotionPolicyID)
	}
	if requirement.BaselinePackID != "pack-base" {
		t.Fatalf("Requirements[0].BaselinePackID = %q, want pack-base", requirement.BaselinePackID)
	}
	if requirement.CandidatePackID != "pack-candidate" {
		t.Fatalf("Requirements[0].CandidatePackID = %q, want pack-candidate", requirement.CandidatePackID)
	}
	if requirement.EligibilityState != CandidatePromotionEligibilityStateCanaryRequired {
		t.Fatalf("Requirements[0].EligibilityState = %q, want canary_required", requirement.EligibilityState)
	}
	if !requirement.RequiredByPolicy {
		t.Fatal("Requirements[0].RequiredByPolicy = false, want true")
	}
	if requirement.OwnerApprovalRequired {
		t.Fatal("Requirements[0].OwnerApprovalRequired = true, want false")
	}
	if requirement.RequirementState != string(HotUpdateCanaryRequirementStateRequired) {
		t.Fatalf("Requirements[0].RequirementState = %q, want required", requirement.RequirementState)
	}
	if requirement.Reason != "candidate result requires canary before promotion" {
		t.Fatalf("Requirements[0].Reason = %q, want candidate result requires canary before promotion", requirement.Reason)
	}
	if requirement.CreatedAt == nil || *requirement.CreatedAt != "2026-04-25T19:10:00Z" {
		t.Fatalf("Requirements[0].CreatedAt = %#v, want 2026-04-25T19:10:00Z", requirement.CreatedAt)
	}
	if requirement.CreatedBy != "operator" {
		t.Fatalf("Requirements[0].CreatedBy = %q, want operator", requirement.CreatedBy)
	}
	if requirement.Error != "" {
		t.Fatalf("Requirements[0].Error = %q, want empty", requirement.Error)
	}
}

func TestLoadOperatorHotUpdateCanaryRequirementIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Requirements) != 0 {
		t.Fatalf("Requirements len = %d, want 0", len(got.Requirements))
	}
}

func TestLoadOperatorHotUpdateCanaryRequirementIdentityStatusInvalidDoesNotHideValidRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 20, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-1", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryRequirementPath(root, "hot-update-canary-requirement-bad"), validHotUpdateCanaryRequirementRecord(now.Add(11*time.Minute), func(record *HotUpdateCanaryRequirementRecord) {
		record.RecordVersion = StoreRecordVersion
		record.CanaryRequirementID = "hot-update-canary-requirement-bad"
		record.ResultID = "result-1"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid requirement) error = %v", err)
	}

	got := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Requirements) != 2 {
		t.Fatalf("Requirements len = %d, want 2", len(got.Requirements))
	}
	if got.Requirements[0].State != "invalid" {
		t.Fatalf("Requirements[0].State = %q, want invalid", got.Requirements[0].State)
	}
	if got.Requirements[0].CanaryRequirementID != "hot-update-canary-requirement-bad" {
		t.Fatalf("Requirements[0].CanaryRequirementID = %q, want hot-update-canary-requirement-bad", got.Requirements[0].CanaryRequirementID)
	}
	if !strings.Contains(got.Requirements[0].Error, "does not match deterministic canary_requirement_id") {
		t.Fatalf("Requirements[0].Error = %q, want deterministic ID mismatch", got.Requirements[0].Error)
	}
	if got.Requirements[1].State != "configured" {
		t.Fatalf("Requirements[1].State = %q, want configured", got.Requirements[1].State)
	}
	if got.Requirements[1].CanaryRequirementID != "hot-update-canary-requirement-result-1" {
		t.Fatalf("Requirements[1].CanaryRequirementID = %q, want valid result requirement", got.Requirements[1].CanaryRequirementID)
	}
}

func TestLoadOperatorHotUpdateCanaryRequirementIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 21, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-read-only"
	})
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-read-only", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, "pack-base"),
		StoreRuntimePackPath(root, "pack-candidate"),
		StoreImprovementCandidatePath(root, "candidate-1"),
		StoreEvalSuitePath(root, "eval-suite-1"),
		StoreImprovementRunPath(root, "run-result"),
		StoreCandidateResultPath(root, "result-read-only"),
		StorePromotionPolicyPath(root, "promotion-policy-result"),
		StoreHotUpdateGatePath(root, "hot-update-1"),
		StoreHotUpdateCanaryRequirementPath(root, "hot-update-canary-requirement-result-read-only"),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root)
	second := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("read-model states = %q/%q, want configured/configured", first.State, second.State)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after hot-update canary requirement status read", path)
		}
	}

	absentPaths := []string{
		StoreCandidatePromotionDecisionsDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after status read: %v", path, err)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateCanaryRequirementIdentity(t *testing.T) {
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

	storeCandidatePromotionEligibilityFixtures(t, root, now.Add(-10*time.Minute), func(record *PromotionPolicyRecord) {
		record.RequiresCanary = true
	}, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateHotUpdateCanaryRequirementFromCandidateResult(root, "result-1", "operator", now.Add(-30*time.Second)); err != nil {
		t.Fatalf("CreateHotUpdateCanaryRequirementFromCandidateResult() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryRequirementIdentity = %#v, want populated hot-update canary requirement identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryRequirementIdentity.State = %q, want configured", snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.State)
	}
	if len(snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements) != 1 {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements len = %d, want 1", len(snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements))
	}
	if snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements[0].CanaryRequirementID != "hot-update-canary-requirement-result-1" {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements[0].CanaryRequirementID = %q, want hot-update-canary-requirement-result-1", snapshot.RuntimeSummary.HotUpdateCanaryRequirementIdentity.Requirements[0].CanaryRequirementID)
	}
}
