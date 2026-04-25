package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorCandidatePromotionDecisionIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-1", "operator", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}

	got := LoadOperatorCandidatePromotionDecisionIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Decisions) != 1 {
		t.Fatalf("Decisions len = %d, want 1", len(got.Decisions))
	}
	decision := got.Decisions[0]
	if decision.State != "configured" {
		t.Fatalf("Decisions[0].State = %q, want configured", decision.State)
	}
	if decision.PromotionDecisionID != "candidate-promotion-decision-result-1" {
		t.Fatalf("Decisions[0].PromotionDecisionID = %q, want candidate-promotion-decision-result-1", decision.PromotionDecisionID)
	}
	if decision.ResultID != "result-1" {
		t.Fatalf("Decisions[0].ResultID = %q, want result-1", decision.ResultID)
	}
	if decision.RunID != "run-result" {
		t.Fatalf("Decisions[0].RunID = %q, want run-result", decision.RunID)
	}
	if decision.CandidateID != "candidate-1" {
		t.Fatalf("Decisions[0].CandidateID = %q, want candidate-1", decision.CandidateID)
	}
	if decision.EvalSuiteID != "eval-suite-1" {
		t.Fatalf("Decisions[0].EvalSuiteID = %q, want eval-suite-1", decision.EvalSuiteID)
	}
	if decision.PromotionPolicyID != "promotion-policy-result" {
		t.Fatalf("Decisions[0].PromotionPolicyID = %q, want promotion-policy-result", decision.PromotionPolicyID)
	}
	if decision.BaselinePackID != "pack-base" {
		t.Fatalf("Decisions[0].BaselinePackID = %q, want pack-base", decision.BaselinePackID)
	}
	if decision.CandidatePackID != "pack-candidate" {
		t.Fatalf("Decisions[0].CandidatePackID = %q, want pack-candidate", decision.CandidatePackID)
	}
	if decision.EligibilityState != CandidatePromotionEligibilityStateEligible {
		t.Fatalf("Decisions[0].EligibilityState = %q, want eligible", decision.EligibilityState)
	}
	if decision.Decision != string(CandidatePromotionDecisionSelectedForPromotion) {
		t.Fatalf("Decisions[0].Decision = %q, want selected_for_promotion", decision.Decision)
	}
	if decision.Reason != "candidate result eligible for promotion" {
		t.Fatalf("Decisions[0].Reason = %q, want candidate result eligible for promotion", decision.Reason)
	}
	if decision.CreatedAt == nil || *decision.CreatedAt != "2026-04-25T15:10:00Z" {
		t.Fatalf("Decisions[0].CreatedAt = %#v, want 2026-04-25T15:10:00Z", decision.CreatedAt)
	}
	if decision.CreatedBy != "operator" {
		t.Fatalf("Decisions[0].CreatedBy = %q, want operator", decision.CreatedBy)
	}
	if decision.Error != "" {
		t.Fatalf("Decisions[0].Error = %q, want empty", decision.Error)
	}
}

func TestLoadOperatorCandidatePromotionDecisionIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorCandidatePromotionDecisionIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Decisions) != 0 {
		t.Fatalf("Decisions len = %d, want 0", len(got.Decisions))
	}
}

func TestLoadOperatorCandidatePromotionDecisionIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 16, 0, 0, 0, time.UTC)
	storeCandidatePromotionEligibilityFixtures(t, root, now, nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if err := WriteStoreJSONAtomic(StoreCandidatePromotionDecisionPath(root, "candidate-promotion-decision-bad"), validCandidatePromotionDecisionRecord(now.Add(10*time.Minute), func(record *CandidatePromotionDecisionRecord) {
		record.RecordVersion = StoreRecordVersion
		record.PromotionDecisionID = "candidate-promotion-decision-bad"
		record.ResultID = "result-1"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(candidate-promotion-decision-bad) error = %v", err)
	}

	got := LoadOperatorCandidatePromotionDecisionIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Decisions) != 1 {
		t.Fatalf("Decisions len = %d, want 1", len(got.Decisions))
	}
	decision := got.Decisions[0]
	if decision.State != "invalid" {
		t.Fatalf("Decisions[0].State = %q, want invalid", decision.State)
	}
	if decision.PromotionDecisionID != "candidate-promotion-decision-bad" {
		t.Fatalf("Decisions[0].PromotionDecisionID = %q, want candidate-promotion-decision-bad", decision.PromotionDecisionID)
	}
	if decision.CandidatePackID != "pack-missing" {
		t.Fatalf("Decisions[0].CandidatePackID = %q, want pack-missing", decision.CandidatePackID)
	}
	if !strings.Contains(decision.Error, `candidate_pack_id "pack-missing" does not match candidate result candidate_pack_id "pack-candidate"`) {
		t.Fatalf("Decisions[0].Error = %q, want missing candidate_pack_id context", decision.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesCandidatePromotionDecisionIdentity(t *testing.T) {
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

	storeCandidatePromotionEligibilityFixtures(t, root, now.Add(-10*time.Minute), nil, func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
	})
	if _, _, err := CreateCandidatePromotionDecisionFromEligibleResult(root, "result-1", "operator", now.Add(-30*time.Second)); err != nil {
		t.Fatalf("CreateCandidatePromotionDecisionFromEligibleResult() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity == nil {
		t.Fatalf("RuntimeSummary.CandidatePromotionDecisionIdentity = %#v, want populated candidate promotion decision identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.CandidatePromotionDecisionIdentity.State = %q, want configured", snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.State)
	}
	if len(snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions) != 1 {
		t.Fatalf("RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions len = %d, want 1", len(snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions))
	}
	if snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions[0].PromotionDecisionID != "candidate-promotion-decision-result-1" {
		t.Fatalf("RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions[0].PromotionDecisionID = %q, want candidate-promotion-decision-result-1", snapshot.RuntimeSummary.CandidatePromotionDecisionIdentity.Decisions[0].PromotionDecisionID)
	}
}
