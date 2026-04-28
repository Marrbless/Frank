package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateCanarySatisfactionIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 20, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-satisfaction"
	})
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}

	got := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Assessments) != 1 {
		t.Fatalf("Assessments len = %d, want 1", len(got.Assessments))
	}
	status := got.Assessments[0]
	if status.State != "configured" || status.SatisfactionState != string(HotUpdateCanarySatisfactionStateSatisfied) {
		t.Fatalf("assessment = %#v, want configured/satisfied", status)
	}
	if status.CanaryRequirementID != requirement.CanaryRequirementID || status.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("assessment refs = %#v, want requirement/evidence refs", status)
	}
	if status.ResultID != "result-satisfaction" || status.RunID != "run-result" || status.CandidateID != "candidate-1" ||
		status.EvalSuiteID != "eval-suite-1" || status.PromotionPolicyID != "promotion-policy-result" ||
		status.BaselinePackID != "pack-base" || status.CandidatePackID != "pack-candidate" {
		t.Fatalf("assessment source refs = %#v, want copied source refs", status)
	}
	if status.EvidenceState != string(HotUpdateCanaryEvidenceStatePassed) || !status.Passed {
		t.Fatalf("evidence state/pass = %q/%v, want passed/true", status.EvidenceState, status.Passed)
	}
	if !reflect.DeepEqual(status.CanaryScopeJobRefs, requirement.CanaryScopeJobRefs) || !reflect.DeepEqual(status.CanaryScopeSurfaces, requirement.CanaryScopeSurfaces) {
		t.Fatalf("canary scope = jobs %#v surfaces %#v, want requirement scope jobs %#v surfaces %#v", status.CanaryScopeJobRefs, status.CanaryScopeSurfaces, requirement.CanaryScopeJobRefs, requirement.CanaryScopeSurfaces)
	}
	if status.EvidenceSource != string(HotUpdateCanaryEvidenceSourceOperatorRecorded) || status.AutomaticTrafficExercised {
		t.Fatalf("evidence source/traffic = %q/%v, want operator_recorded/false", status.EvidenceSource, status.AutomaticTrafficExercised)
	}
	if !reflect.DeepEqual(status.ExercisedJobRefs, requirement.CanaryScopeJobRefs) || !reflect.DeepEqual(status.ExercisedSurfaces, requirement.CanaryScopeSurfaces) {
		t.Fatalf("exercised scope = jobs %#v surfaces %#v, want requirement scope jobs %#v surfaces %#v", status.ExercisedJobRefs, status.ExercisedSurfaces, requirement.CanaryScopeJobRefs, requirement.CanaryScopeSurfaces)
	}
	if status.ObservedAt == nil || *status.ObservedAt != "2026-04-26T20:20:00Z" {
		t.Fatalf("ObservedAt = %#v, want 2026-04-26T20:20:00Z", status.ObservedAt)
	}
	if status.Error != "" {
		t.Fatalf("Error = %q, want empty", status.Error)
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Assessments) != 0 {
		t.Fatalf("Assessments len = %d, want 0", len(got.Assessments))
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionIdentityStatusInvalidDoesNotHideValid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 21, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	valid, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	bad := validHotUpdateCanaryEvidenceRecord(now.Add(30*time.Minute), func(record *HotUpdateCanaryEvidenceRecord) {
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

	got := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Assessments) != 2 {
		t.Fatalf("Assessments len = %d, want configured assessment plus invalid evidence assessment", len(got.Assessments))
	}
	var foundValid, foundInvalid bool
	for _, status := range got.Assessments {
		switch status.State {
		case "configured":
			foundValid = status.SelectedCanaryEvidenceID == valid.CanaryEvidenceID && status.SatisfactionState == string(HotUpdateCanarySatisfactionStateSatisfied)
		case "invalid":
			foundInvalid = strings.Contains(status.Error, "does not match hot-update canary requirement")
		}
	}
	if !foundValid || !foundInvalid {
		t.Fatalf("Assessments = %#v, want valid selected evidence and surfaced invalid evidence", got.Assessments)
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionIdentityStatusAllInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 22, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	bad := validHotUpdateCanaryEvidenceRecord(now.Add(20*time.Minute), func(record *HotUpdateCanaryEvidenceRecord) {
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

	got := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Assessments) != 1 || got.Assessments[0].State != "invalid" || got.Assessments[0].SatisfactionState != string(HotUpdateCanarySatisfactionStateInvalid) {
		t.Fatalf("Assessments = %#v, want one invalid assessment", got.Assessments)
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 23, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
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

	first := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	second := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("states = %q/%q, want configured/configured", first.State, second.State)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after hot-update canary satisfaction status read", path)
		}
	}
	for _, path := range []string{
		StoreCandidatePromotionDecisionsDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after status read: %v", path, err)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateCanarySatisfactionIdentity(t *testing.T) {
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

	requirement := storeCanaryRequirementForEvidence(t, root, now.Add(-10*time.Minute), nil, nil)
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
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateCanarySatisfactionIdentity = %#v, want populated hot-update canary satisfaction identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.HotUpdateCanarySatisfactionIdentity.State = %q, want configured", snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity.State)
	}
	if len(snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity.Assessments) != 1 {
		t.Fatalf("Assessments len = %d, want 1", len(snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity.Assessments))
	}
	status := snapshot.RuntimeSummary.HotUpdateCanarySatisfactionIdentity.Assessments[0]
	if status.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID || status.SatisfactionState != string(HotUpdateCanarySatisfactionStateSatisfied) {
		t.Fatalf("assessment = %#v, want selected satisfied evidence %q", status, evidence.CanaryEvidenceID)
	}
}
