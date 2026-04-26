package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
	}

	got := LoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Authorities) != 1 {
		t.Fatalf("Authorities len = %d, want 1", len(got.Authorities))
	}
	status := got.Authorities[0]
	if status.State != "configured" || status.AuthorityState != string(HotUpdateCanarySatisfactionAuthorityStateAuthorized) {
		t.Fatalf("authority status = %#v, want configured/authorized", status)
	}
	if status.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		status.CanaryRequirementID != requirement.CanaryRequirementID ||
		status.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("authority refs = %#v, want authority/requirement/evidence refs", status)
	}
	if status.ResultID != requirement.ResultID || status.RunID != requirement.RunID || status.CandidateID != requirement.CandidateID ||
		status.EvalSuiteID != requirement.EvalSuiteID || status.PromotionPolicyID != requirement.PromotionPolicyID ||
		status.BaselinePackID != requirement.BaselinePackID || status.CandidatePackID != requirement.CandidatePackID {
		t.Fatalf("authority source refs = %#v, want copied source refs", status)
	}
	if status.SatisfactionState != string(HotUpdateCanarySatisfactionStateSatisfied) || status.OwnerApprovalRequired {
		t.Fatalf("satisfaction/owner = %q/%v, want satisfied/false", status.SatisfactionState, status.OwnerApprovalRequired)
	}
	if status.CreatedAt == nil || *status.CreatedAt != "2026-04-27T20:22:00Z" {
		t.Fatalf("CreatedAt = %#v, want 2026-04-27T20:22:00Z", status.CreatedAt)
	}
	if status.CreatedBy != "operator" || status.Error != "" {
		t.Fatalf("CreatedBy/Error = %q/%q, want operator/empty", status.CreatedBy, status.Error)
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Authorities) != 0 {
		t.Fatalf("Authorities len = %d, want 0", len(got.Authorities))
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatusInvalidDoesNotHideValid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 21, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	valid, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
	}
	bad := valid
	bad.CanarySatisfactionAuthorityID = bad.CanarySatisfactionAuthorityID + "-bad"
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanarySatisfactionAuthorityPath(root, bad.CanarySatisfactionAuthorityID), bad); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid authority) error = %v", err)
	}

	got := LoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Authorities) != 2 {
		t.Fatalf("Authorities len = %d, want 2", len(got.Authorities))
	}
	var foundValid, foundInvalid bool
	for _, status := range got.Authorities {
		switch status.State {
		case "configured":
			foundValid = status.CanarySatisfactionAuthorityID == valid.CanarySatisfactionAuthorityID
		case "invalid":
			foundInvalid = strings.Contains(status.Error, "does not match deterministic")
		}
	}
	if !foundValid || !foundInvalid {
		t.Fatalf("Authorities = %#v, want valid and invalid authority records", got.Authorities)
	}
}

func TestLoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 22, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
	if err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(22*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
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
		StoreHotUpdateCanarySatisfactionAuthorityPath(root, authority.CanarySatisfactionAuthorityID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatus(root)
	second := LoadOperatorHotUpdateCanarySatisfactionAuthorityIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("states = %q/%q, want configured/configured", first.State, second.State)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after authority status read", path)
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

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateCanarySatisfactionAuthorityIdentity(t *testing.T) {
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
	if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(-30*time.Second), "operator", now.Add(-20*time.Second), "canary passed"); err != nil {
		t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
	}
	authority, _, err := CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, requirement.CanaryRequirementID, "operator", now.Add(-10*time.Second))
	if err != nil {
		t.Fatalf("CreateHotUpdateCanarySatisfactionAuthorityFromRequirement() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateCanarySatisfactionAuthorityIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateCanarySatisfactionAuthorityIdentity = %#v, want populated identity", snapshot.RuntimeSummary)
	}
	identity := snapshot.RuntimeSummary.HotUpdateCanarySatisfactionAuthorityIdentity
	if identity.State != "configured" {
		t.Fatalf("identity.State = %q, want configured", identity.State)
	}
	if len(identity.Authorities) != 1 {
		t.Fatalf("Authorities len = %d, want 1", len(identity.Authorities))
	}
	if identity.Authorities[0].CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID {
		t.Fatalf("Authorities[0].CanarySatisfactionAuthorityID = %q, want %q", identity.Authorities[0].CanarySatisfactionAuthorityID, authority.CanarySatisfactionAuthorityID)
	}
}
