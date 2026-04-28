package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateCanaryEvidenceIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 21, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	got := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Evidence) != 1 {
		t.Fatalf("Evidence len = %d, want 1", len(got.Evidence))
	}
	status := got.Evidence[0]
	if status.State != "configured" {
		t.Fatalf("Evidence[0].State = %q, want configured", status.State)
	}
	if status.CanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("Evidence[0].CanaryEvidenceID = %q, want %q", status.CanaryEvidenceID, evidence.CanaryEvidenceID)
	}
	if status.CanaryRequirementID != requirement.CanaryRequirementID {
		t.Fatalf("Evidence[0].CanaryRequirementID = %q, want %q", status.CanaryRequirementID, requirement.CanaryRequirementID)
	}
	if status.ResultID != "result-1" || status.RunID != "run-result" || status.CandidateID != "candidate-1" ||
		status.EvalSuiteID != "eval-suite-1" || status.PromotionPolicyID != "promotion-policy-result" ||
		status.BaselinePackID != "pack-base" || status.CandidatePackID != "pack-candidate" {
		t.Fatalf("Evidence[0] refs = %#v, want copied source refs", status)
	}
	if status.EvidenceState != string(HotUpdateCanaryEvidenceStatePassed) {
		t.Fatalf("Evidence[0].EvidenceState = %q, want passed", status.EvidenceState)
	}
	if !status.Passed {
		t.Fatal("Evidence[0].Passed = false, want true")
	}
	if status.EvidenceSource != string(HotUpdateCanaryEvidenceSourceOperatorRecorded) || status.AutomaticTrafficExercised {
		t.Fatalf("Evidence[0] source/traffic = %q/%v, want operator_recorded/false", status.EvidenceSource, status.AutomaticTrafficExercised)
	}
	if !reflect.DeepEqual(status.ExercisedJobRefs, requirement.CanaryScopeJobRefs) || !reflect.DeepEqual(status.ExercisedSurfaces, requirement.CanaryScopeSurfaces) {
		t.Fatalf("Evidence[0] exercised scope = jobs %#v surfaces %#v, want requirement scope jobs %#v surfaces %#v", status.ExercisedJobRefs, status.ExercisedSurfaces, requirement.CanaryScopeJobRefs, requirement.CanaryScopeSurfaces)
	}
	if status.Reason != "canary passed" {
		t.Fatalf("Evidence[0].Reason = %q, want canary passed", status.Reason)
	}
	if status.ObservedAt == nil || *status.ObservedAt != "2026-04-25T21:20:00Z" {
		t.Fatalf("Evidence[0].ObservedAt = %#v, want 2026-04-25T21:20:00Z", status.ObservedAt)
	}
	if status.CreatedAt == nil || *status.CreatedAt != "2026-04-25T21:21:00Z" {
		t.Fatalf("Evidence[0].CreatedAt = %#v, want 2026-04-25T21:21:00Z", status.CreatedAt)
	}
	if status.CreatedBy != "operator" {
		t.Fatalf("Evidence[0].CreatedBy = %q, want operator", status.CreatedBy)
	}
	if status.Error != "" {
		t.Fatalf("Evidence[0].Error = %q, want empty", status.Error)
	}
}

func TestLoadOperatorHotUpdateCanaryEvidenceIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Evidence) != 0 {
		t.Fatalf("Evidence len = %d, want 0", len(got.Evidence))
	}
}

func TestLoadOperatorHotUpdateCanaryEvidenceIdentityStatusInvalidDoesNotHideValidRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 22, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	bad := validHotUpdateCanaryEvidenceRecord(now.Add(19*time.Minute), func(record *HotUpdateCanaryEvidenceRecord) {
		record.CanaryRequirementID = requirement.CanaryRequirementID
		record.CanaryEvidenceID = HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirement.CanaryRequirementID, record.ObservedAt)
		record.ResultID = requirement.ResultID
		record.RunID = requirement.RunID
		record.CandidateID = requirement.CandidateID
		record.EvalSuiteID = requirement.EvalSuiteID
		record.PromotionPolicyID = requirement.PromotionPolicyID
		record.BaselinePackID = requirement.BaselinePackID
		record.CandidatePackID = "pack-missing"
	})
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryEvidencePath(root, bad.CanaryEvidenceID), bad); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid evidence) error = %v", err)
	}

	got := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Evidence) != 2 {
		t.Fatalf("Evidence len = %d, want 2", len(got.Evidence))
	}
	if got.Evidence[0].State != "invalid" {
		t.Fatalf("Evidence[0].State = %q, want invalid", got.Evidence[0].State)
	}
	if !strings.Contains(got.Evidence[0].Error, "does not match hot-update canary requirement") {
		t.Fatalf("Evidence[0].Error = %q, want requirement mismatch", got.Evidence[0].Error)
	}
	if got.Evidence[1].State != "configured" {
		t.Fatalf("Evidence[1].State = %q, want configured", got.Evidence[1].State)
	}
}

func TestLoadOperatorHotUpdateCanaryEvidenceIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 23, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-read-only"
	})
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, requirement.BaselinePackID),
		StoreRuntimePackPath(root, requirement.CandidatePackID),
		StoreImprovementCandidatePath(root, requirement.CandidateID),
		StoreEvalSuitePath(root, requirement.EvalSuiteID),
		StoreImprovementRunPath(root, requirement.RunID),
		StoreCandidateResultPath(root, requirement.ResultID),
		StorePromotionPolicyPath(root, requirement.PromotionPolicyID),
		StoreHotUpdateGatePath(root, "hot-update-1"),
		StoreHotUpdateCanaryRequirementPath(root, requirement.CanaryRequirementID),
		StoreHotUpdateCanaryEvidencePath(root, evidence.CanaryEvidenceID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root)
	second := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("read-model states = %q/%q, want configured/configured", first.State, second.State)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after hot-update canary evidence status read", path)
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

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateCanaryEvidenceIdentity(t *testing.T) {
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

	requirement := storeCanaryRequirementForEvidence(t, root, now.Add(-10*time.Minute), nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(-30*time.Second), "operator", now.Add(-20*time.Second), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryEvidenceIdentity = %#v, want populated hot-update canary evidence identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryEvidenceIdentity.State = %q, want configured", snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.State)
	}
	if len(snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence) != 1 {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence len = %d, want 1", len(snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence))
	}
	if snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence[0].CanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence[0].CanaryEvidenceID = %q, want %q", snapshot.RuntimeSummary.HotUpdateCanaryEvidenceIdentity.Evidence[0].CanaryEvidenceID, evidence.CanaryEvidenceID)
	}
}
