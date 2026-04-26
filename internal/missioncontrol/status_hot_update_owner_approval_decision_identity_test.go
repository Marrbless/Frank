package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 20, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	decision, _, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
	}

	got := LoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Decisions) != 1 {
		t.Fatalf("Decisions len = %d, want 1", len(got.Decisions))
	}
	status := got.Decisions[0]
	if status.State != "configured" || status.Decision != string(HotUpdateOwnerApprovalDecisionGranted) {
		t.Fatalf("decision status = %#v, want configured/granted", status)
	}
	if status.OwnerApprovalDecisionID != decision.OwnerApprovalDecisionID ||
		status.OwnerApprovalRequestID != request.OwnerApprovalRequestID ||
		status.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		status.CanaryRequirementID != requirement.CanaryRequirementID ||
		status.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		t.Fatalf("decision refs = %#v, want decision/request/authority/requirement/evidence refs", status)
	}
	if status.ResultID != requirement.ResultID || status.RunID != requirement.RunID || status.CandidateID != requirement.CandidateID ||
		status.EvalSuiteID != requirement.EvalSuiteID || status.PromotionPolicyID != requirement.PromotionPolicyID ||
		status.BaselinePackID != requirement.BaselinePackID || status.CandidatePackID != requirement.CandidatePackID {
		t.Fatalf("decision source refs = %#v, want copied source refs", status)
	}
	if status.RequestState != string(HotUpdateOwnerApprovalRequestStateRequested) ||
		status.AuthorityState != string(HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval) ||
		status.SatisfactionState != string(HotUpdateCanarySatisfactionStateWaitingOwnerApproval) ||
		!status.OwnerApprovalRequired {
		t.Fatalf("state fields = %#v, want requested/waiting_owner_approval/true", status)
	}
	if status.DecidedAt == nil || *status.DecidedAt != "2026-04-29T20:24:00Z" {
		t.Fatalf("DecidedAt = %#v, want 2026-04-29T20:24:00Z", status.DecidedAt)
	}
	if status.DecidedBy != "operator" || status.Reason != "approved" || status.Error != "" {
		t.Fatalf("provenance/error = %#v, want operator/approved/empty", status)
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Decisions) != 0 {
		t.Fatalf("Decisions len = %d, want 0", len(got.Decisions))
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatusInvalidDoesNotHideValid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 21, 0, 0, 0, time.UTC)
	_, _, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	valid, _, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
	}
	bad := valid
	bad.OwnerApprovalDecisionID = "hot-update-owner-approval-decision-bad"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalDecisionPath(root, bad.OwnerApprovalDecisionID), bad); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid decision) error = %v", err)
	}

	got := LoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Decisions) != 2 {
		t.Fatalf("Decisions len = %d, want 2", len(got.Decisions))
	}
	var foundValid, foundInvalid bool
	for _, status := range got.Decisions {
		switch status.State {
		case "configured":
			foundValid = status.OwnerApprovalDecisionID == valid.OwnerApprovalDecisionID
		case "invalid":
			foundInvalid = strings.Contains(status.Error, "does not match deterministic")
		}
	}
	if !foundValid || !foundInvalid {
		t.Fatalf("Decisions = %#v, want valid and invalid decision records", got.Decisions)
	}
}

func TestLoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 22, 0, 0, 0, time.UTC)
	requirement, evidence, authority := storeWaitingOwnerApprovalAuthority(t, root, now)
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(23*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	decision, _, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(24*time.Minute), "approved")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
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
		StoreHotUpdateOwnerApprovalDecisionPath(root, decision.OwnerApprovalDecisionID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatus(root)
	second := LoadOperatorHotUpdateOwnerApprovalDecisionIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("states = %q/%q, want configured/configured", first.State, second.State)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after decision status read", path)
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

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateOwnerApprovalDecisionIdentity(t *testing.T) {
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
	request, _, err := CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, authority.CanarySatisfactionAuthorityID, "operator", now.Add(-20*time.Second))
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority() error = %v", err)
	}
	decision, _, err := CreateHotUpdateOwnerApprovalDecisionFromRequest(root, request.OwnerApprovalRequestID, HotUpdateOwnerApprovalDecisionGranted, "operator", now.Add(-10*time.Second), "approved")
	if err != nil {
		t.Fatalf("CreateHotUpdateOwnerApprovalDecisionFromRequest() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateOwnerApprovalDecisionIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateOwnerApprovalDecisionIdentity = %#v, want populated identity", snapshot.RuntimeSummary)
	}
	identity := snapshot.RuntimeSummary.HotUpdateOwnerApprovalDecisionIdentity
	if identity.State != "configured" {
		t.Fatalf("identity.State = %q, want configured", identity.State)
	}
	if len(identity.Decisions) != 1 {
		t.Fatalf("Decisions len = %d, want 1", len(identity.Decisions))
	}
	if identity.Decisions[0].OwnerApprovalDecisionID != decision.OwnerApprovalDecisionID {
		t.Fatalf("Decisions[0].OwnerApprovalDecisionID = %q, want %q", identity.Decisions[0].OwnerApprovalDecisionID, decision.OwnerApprovalDecisionID)
	}
}
