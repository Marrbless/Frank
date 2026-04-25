package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorCandidateResultIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 19, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(6*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-result"
		record.RequiresCanary = false
		record.RequiresOwnerApproval = false
		record.EpsilonRule = "epsilon <= 0.01"
		record.RegressionRule = "no_regression_flags"
		record.CompatibilityRule = "compatibility_score >= 0.90"
		record.ResourceRule = "resource_score >= 0.60"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
		record.RunID = "run-1"
		record.PromotionPolicyID = "promotion-policy-result"
		record.CreatedBy = "operator"
		record.Notes = "candidate kept for later gate"
		record.RegressionFlags = []string{"none"}
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}

	got := LoadOperatorCandidateResultIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(got.Results))
	}
	result := got.Results[0]
	if result.State != "configured" {
		t.Fatalf("Results[0].State = %q, want configured", result.State)
	}
	if result.ResultID != "result-1" {
		t.Fatalf("Results[0].ResultID = %q, want result-1", result.ResultID)
	}
	if result.RunID != "run-1" {
		t.Fatalf("Results[0].RunID = %q, want run-1", result.RunID)
	}
	if result.CandidateID != "candidate-1" {
		t.Fatalf("Results[0].CandidateID = %q, want candidate-1", result.CandidateID)
	}
	if result.EvalSuiteID != "eval-suite-1" {
		t.Fatalf("Results[0].EvalSuiteID = %q, want eval-suite-1", result.EvalSuiteID)
	}
	if result.PromotionPolicyID != "promotion-policy-result" {
		t.Fatalf("Results[0].PromotionPolicyID = %q, want promotion-policy-result", result.PromotionPolicyID)
	}
	if result.BaselinePackID != "pack-base" {
		t.Fatalf("Results[0].BaselinePackID = %q, want pack-base", result.BaselinePackID)
	}
	if result.CandidatePackID != "pack-candidate" {
		t.Fatalf("Results[0].CandidatePackID = %q, want pack-candidate", result.CandidatePackID)
	}
	if result.HotUpdateID != "hot-update-1" {
		t.Fatalf("Results[0].HotUpdateID = %q, want hot-update-1", result.HotUpdateID)
	}
	if result.Decision != string(ImprovementRunDecisionKeep) {
		t.Fatalf("Results[0].Decision = %q, want keep", result.Decision)
	}
	if result.Notes != "candidate kept for later gate" {
		t.Fatalf("Results[0].Notes = %q, want notes", result.Notes)
	}
	if got := strings.Join(result.RegressionFlags, ","); got != "none" {
		t.Fatalf("Results[0].RegressionFlags = %#v, want none", result.RegressionFlags)
	}
	if result.PromotionEligibility == nil {
		t.Fatal("Results[0].PromotionEligibility = nil, want derived eligibility")
	}
	if result.PromotionEligibility.State != CandidatePromotionEligibilityStateEligible {
		t.Fatalf("Results[0].PromotionEligibility.State = %q, want eligible; eligibility = %#v", result.PromotionEligibility.State, result.PromotionEligibility)
	}
	if result.PromotionEligibility.PromotionPolicyID != "promotion-policy-result" {
		t.Fatalf("Results[0].PromotionEligibility.PromotionPolicyID = %q, want promotion-policy-result", result.PromotionEligibility.PromotionPolicyID)
	}
	if result.CreatedAt == nil || *result.CreatedAt != "2026-04-24T19:06:00Z" {
		t.Fatalf("Results[0].CreatedAt = %#v, want 2026-04-24T19:06:00Z", result.CreatedAt)
	}
	if result.CreatedBy != "operator" {
		t.Fatalf("Results[0].CreatedBy = %q, want operator", result.CreatedBy)
	}
	if result.Error != "" {
		t.Fatalf("Results[0].Error = %q, want empty", result.Error)
	}
}

func TestLoadOperatorCandidateResultIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorCandidateResultIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Results) != 0 {
		t.Fatalf("Results len = %d, want 0", len(got.Results))
	}
}

func TestLoadOperatorCandidateResultIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 24, 20, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := WriteStoreJSONAtomic(StoreCandidateResultPath(root, "result-bad"), validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.RecordVersion = StoreRecordVersion
		record.ResultID = "result-bad"
		record.RunID = "run-1"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(result-bad) error = %v", err)
	}

	got := LoadOperatorCandidateResultIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(got.Results))
	}
	result := got.Results[0]
	if result.State != "invalid" {
		t.Fatalf("Results[0].State = %q, want invalid", result.State)
	}
	if result.ResultID != "result-bad" {
		t.Fatalf("Results[0].ResultID = %q, want result-bad", result.ResultID)
	}
	if result.CandidatePackID != "pack-missing" {
		t.Fatalf("Results[0].CandidatePackID = %q, want pack-missing", result.CandidatePackID)
	}
	if !strings.Contains(result.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Results[0].Error = %q, want missing candidate_pack_id context", result.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesCandidateResultIdentity(t *testing.T) {
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

	storeImprovementRunFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(-30*time.Second), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(-15*time.Second), func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
		record.RunID = "run-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.CandidateResultIdentity == nil {
		t.Fatalf("RuntimeSummary.CandidateResultIdentity = %#v, want populated candidate result identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.CandidateResultIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.CandidateResultIdentity.State = %q, want configured", snapshot.RuntimeSummary.CandidateResultIdentity.State)
	}
	if len(snapshot.RuntimeSummary.CandidateResultIdentity.Results) != 1 {
		t.Fatalf("RuntimeSummary.CandidateResultIdentity.Results len = %d, want 1", len(snapshot.RuntimeSummary.CandidateResultIdentity.Results))
	}
	if snapshot.RuntimeSummary.CandidateResultIdentity.Results[0].ResultID != "result-1" {
		t.Fatalf("RuntimeSummary.CandidateResultIdentity.Results[0].ResultID = %q, want result-1", snapshot.RuntimeSummary.CandidateResultIdentity.Results[0].ResultID)
	}
}
