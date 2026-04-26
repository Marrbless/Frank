package missioncontrol

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAssessHotUpdateCanarySatisfactionMissingRequirementIsInvalid(t *testing.T) {
	t.Parallel()

	got, err := AssessHotUpdateCanarySatisfaction(t.TempDir(), "hot-update-canary-requirement-missing")
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
	}
	if got.State != "invalid" || got.SatisfactionState != HotUpdateCanarySatisfactionStateInvalid {
		t.Fatalf("assessment = %#v, want invalid missing requirement", got)
	}
	if !strings.Contains(got.Error, ErrHotUpdateCanaryRequirementRecordNotFound.Error()) {
		t.Fatalf("Error = %q, want missing requirement", got.Error)
	}
}

func TestAssessHotUpdateCanarySatisfactionNoEvidenceIsNotSatisfied(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)

	got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
	}
	if got.State != "configured" || got.SatisfactionState != HotUpdateCanarySatisfactionStateNotSatisfied {
		t.Fatalf("assessment = %#v, want configured/not_satisfied", got)
	}
	if got.SelectedCanaryEvidenceID != "" || got.Passed {
		t.Fatalf("selected/pass = %q/%v, want none/false", got.SelectedCanaryEvidenceID, got.Passed)
	}
}

func TestAssessHotUpdateCanarySatisfactionPassedEvidenceStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		policyEdit       func(*PromotionPolicyRecord)
		wantSatisfaction HotUpdateCanarySatisfactionState
		wantOwner        bool
	}{
		{name: "satisfied", wantSatisfaction: HotUpdateCanarySatisfactionStateSatisfied},
		{name: "waiting owner approval", policyEdit: func(record *PromotionPolicyRecord) {
			record.RequiresOwnerApproval = true
		}, wantSatisfaction: HotUpdateCanarySatisfactionStateWaitingOwnerApproval, wantOwner: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, tt.policyEdit, nil)
			evidence, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed")
			if err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement() error = %v", err)
			}

			got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
			if err != nil {
				t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
			}
			if got.State != "configured" || got.SatisfactionState != tt.wantSatisfaction {
				t.Fatalf("assessment = %#v, want configured/%s", got, tt.wantSatisfaction)
			}
			if got.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID || !got.Passed || got.EvidenceState != HotUpdateCanaryEvidenceStatePassed {
				t.Fatalf("selected evidence = %#v, want passed evidence %q", got, evidence.CanaryEvidenceID)
			}
			if got.OwnerApprovalRequired != tt.wantOwner {
				t.Fatalf("OwnerApprovalRequired = %v, want %v", got.OwnerApprovalRequired, tt.wantOwner)
			}
		})
	}
}

func TestAssessHotUpdateCanarySatisfactionLatestTerminalEvidenceStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state HotUpdateCanaryEvidenceState
		want  HotUpdateCanarySatisfactionState
	}{
		{state: HotUpdateCanaryEvidenceStateFailed, want: HotUpdateCanarySatisfactionStateFailed},
		{state: HotUpdateCanaryEvidenceStateBlocked, want: HotUpdateCanarySatisfactionStateBlocked},
		{state: HotUpdateCanaryEvidenceStateExpired, want: HotUpdateCanarySatisfactionStateExpired},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.state), func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
			requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
			if _, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, HotUpdateCanaryEvidenceStatePassed, now.Add(20*time.Minute), "operator", now.Add(21*time.Minute), "canary passed"); err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(passed) error = %v", err)
			}
			latest, _, err := CreateHotUpdateCanaryEvidenceFromRequirement(root, requirement.CanaryRequirementID, tt.state, now.Add(30*time.Minute), "operator", now.Add(31*time.Minute), "latest canary evidence")
			if err != nil {
				t.Fatalf("CreateHotUpdateCanaryEvidenceFromRequirement(%s) error = %v", tt.state, err)
			}

			got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
			if err != nil {
				t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
			}
			if got.SatisfactionState != tt.want || got.SelectedCanaryEvidenceID != latest.CanaryEvidenceID || got.Passed {
				t.Fatalf("assessment = %#v, want latest %s evidence", got, tt.state)
			}
		})
	}
}

func TestSelectHotUpdateCanarySatisfactionEvidenceIsDeterministic(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 4, 26, 15, 0, 0, 0, time.UTC)
	selected := selectHotUpdateCanarySatisfactionEvidence([]HotUpdateCanaryEvidenceRecord{
		{CanaryEvidenceID: "hot-update-canary-evidence-req-a-20260426T150000000000000Z", ObservedAt: observedAt},
		{CanaryEvidenceID: "hot-update-canary-evidence-req-a-20260426T150000000000001Z", ObservedAt: observedAt.Add(time.Nanosecond)},
		{CanaryEvidenceID: "hot-update-canary-evidence-req-a-z", ObservedAt: observedAt},
	})
	if selected.CanaryEvidenceID != "hot-update-canary-evidence-req-a-20260426T150000000000001Z" {
		t.Fatalf("selected = %q, want newest observed_at", selected.CanaryEvidenceID)
	}

	tie := selectHotUpdateCanarySatisfactionEvidence([]HotUpdateCanaryEvidenceRecord{
		{CanaryEvidenceID: "hot-update-canary-evidence-req-a-a", ObservedAt: observedAt},
		{CanaryEvidenceID: "hot-update-canary-evidence-req-a-b", ObservedAt: observedAt},
	})
	if tie.CanaryEvidenceID != "hot-update-canary-evidence-req-a-b" {
		t.Fatalf("tie selected = %q, want lexicographically largest canary_evidence_id", tie.CanaryEvidenceID)
	}
}

func TestAssessHotUpdateCanarySatisfactionInvalidEvidenceDoesNotHideValid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 16, 0, 0, 0, time.UTC)
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

	got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
	}
	if got.State != "configured" || got.SatisfactionState != HotUpdateCanarySatisfactionStateSatisfied || got.SelectedCanaryEvidenceID != valid.CanaryEvidenceID {
		t.Fatalf("assessment = %#v, want valid passed evidence not hidden by invalid evidence", got)
	}
}

func TestAssessHotUpdateCanarySatisfactionAllInvalidEvidenceIsInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 17, 0, 0, 0, time.UTC)
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

	got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
	}
	if got.State != "invalid" || got.SatisfactionState != HotUpdateCanarySatisfactionStateInvalid {
		t.Fatalf("assessment = %#v, want invalid", got)
	}
	if !strings.Contains(got.Error, "does not match hot-update canary requirement") {
		t.Fatalf("Error = %q, want requirement mismatch", got.Error)
	}
}

func TestAssessHotUpdateCanarySatisfactionStaleEligibilityIsInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC)
	requirement := storeCanaryRequirementForEvidence(t, root, now, nil, nil)
	policy, err := LoadPromotionPolicyRecord(root, requirement.PromotionPolicyID)
	if err != nil {
		t.Fatalf("LoadPromotionPolicyRecord() error = %v", err)
	}
	policy.RequiresCanary = false
	if err := WriteStoreJSONAtomic(StorePromotionPolicyPath(root, policy.PromotionPolicyID), policy); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(policy) error = %v", err)
	}

	got, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction() error = %v", err)
	}
	if got.State != "invalid" || !strings.Contains(got.Error, "does not permit hot-update canary requirement") {
		t.Fatalf("assessment = %#v, want invalid stale eligibility", got)
	}
}

func TestAssessHotUpdateCanarySatisfactionReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 19, 0, 0, 0, time.UTC)
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

	first, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction(first) error = %v", err)
	}
	second, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		t.Fatalf("AssessHotUpdateCanarySatisfaction(second) error = %v", err)
	}
	if first.SatisfactionState != HotUpdateCanarySatisfactionStateSatisfied || second.SatisfactionState != HotUpdateCanarySatisfactionStateSatisfied {
		t.Fatalf("states = %q/%q, want satisfied/satisfied", first.SatisfactionState, second.SatisfactionState)
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after assessment error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after hot-update canary satisfaction assessment", path)
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
			t.Fatalf("path %s exists or errored after assessment read: %v", path, err)
		}
	}
}
