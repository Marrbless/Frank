package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateOwnerApprovalRequestIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 20, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}

	got := LoadOperatorHotUpdateOwnerApprovalRequestIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(got.Requests))
	}
	status := got.Requests[0]
	if status.State != "configured" || status.RequestState != string(HotUpdateOwnerApprovalRequestStateRequested) {
		t.Fatalf("request status = %#v, want configured/requested", status)
	}
	if status.OwnerApprovalRequestID != request.OwnerApprovalRequestID ||
		status.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		status.CanaryRequirementID != requirement.CanaryRequirementID ||
		status.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("request refs = %#v, want request/authority/requirement/evidence refs", status)
	}
	if status.ResultID != requirement.ResultID || status.RunID != requirement.RunID || status.CandidateID != requirement.CandidateID ||
		status.EvalSuiteID != requirement.EvalSuiteID || status.PromotionPolicyID != requirement.PromotionPolicyID ||
		status.BaselinePackID != requirement.BaselinePackID || status.CandidatePackID != requirement.CandidatePackID {
		t.Fatalf("request source refs = %#v, want copied source refs", status)
	}
	if status.AuthorityState != string(HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval) ||
		status.SatisfactionState != string(HotUpdateCanarySatisfactionStateWaitingOwnerApproval) ||
		!status.OwnerApprovalRequired {
		t.Fatalf("authority/satisfaction/owner = %#v, want waiting_owner_approval/true", status)
	}
	if status.CreatedAt == nil || *status.CreatedAt != "2026-04-28T20:23:00Z" {
		t.Fatalf("CreatedAt = %#v, want 2026-04-28T20:23:00Z", status.CreatedAt)
	}
	if status.CreatedBy != "operator" || status.Error != "" {
		t.Fatalf("CreatedBy/Error = %q/%q, want operator/empty", status.CreatedBy, status.Error)
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalRequestIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateOwnerApprovalRequestIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Requests) != 0 {
		t.Fatalf("Requests len = %d, want 0", len(got.Requests))
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalRequestIdentityStatusInvalidDoesNotHideValid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 21, 0, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	valid, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	bad := valid
	bad.OwnerApprovalRequestID = bad.OwnerApprovalRequestID + "-bad"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalRequestPath(root, bad.OwnerApprovalRequestID), bad); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid request) error = %v", err)
	}

	got := LoadOperatorHotUpdateOwnerApprovalRequestIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Requests) != 2 {
		t.Fatalf("Requests len = %d, want 2", len(got.Requests))
	}
	var foundValid, foundInvalid bool
	for _, status := range got.Requests {
		switch status.State {
		case "configured":
			foundValid = status.OwnerApprovalRequestID == valid.OwnerApprovalRequestID
		case "invalid":
			foundInvalid = strings.Contains(status.Error, "does not match deterministic")
		}
	}
	if !foundValid || !foundInvalid {
		t.Fatalf("Requests = %#v, want valid and invalid request records", got.Requests)
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalRequestIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 22, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
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
		StoreHotUpdateOwnerApprovalRequestPath(root, request.OwnerApprovalRequestID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorHotUpdateOwnerApprovalRequestIdentityStatus(root)
	second := LoadOperatorHotUpdateOwnerApprovalRequestIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("states = %q/%q, want configured/configured", first.State, second.State)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after request status read", path)
		}
	}
	assertNoHotUpdateOwnerApprovalRequestDownstreamRecords(t, root)
	for _, path := range []string{
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after status read: %v", path, err)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateOwnerApprovalRequestIdentity(t *testing.T) {
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

	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now.Add(-10*time.Minute))
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(-10*time.Second))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateOwnerApprovalRequestIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateOwnerApprovalRequestIdentity = %#v, want populated identity", snapshot.RuntimeSummary)
	}
	identity := snapshot.RuntimeSummary.HotUpdateOwnerApprovalRequestIdentity
	if identity.State != "configured" {
		t.Fatalf("identity.State = %q, want configured", identity.State)
	}
	if len(identity.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(identity.Requests))
	}
	if identity.Requests[0].OwnerApprovalRequestID != request.OwnerApprovalRequestID {
		t.Fatalf("Requests[0].OwnerApprovalRequestID = %q, want %q", identity.Requests[0].OwnerApprovalRequestID, request.OwnerApprovalRequestID)
	}
}
