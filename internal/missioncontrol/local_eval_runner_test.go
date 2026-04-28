package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestRunLocalDeterministicEvalCreatesCandidateResultAndEligibility(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC)
	storeLocalEvalRunnerFixtures(t, root, now)

	spec := LocalEvalRunnerSpec{
		RunID:              "run-root",
		CandidateID:        "candidate-1",
		EvalSuiteID:        "eval-suite-1",
		PromotionPolicyID:  "promotion-policy-local",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{" none "},
		Decision:           ImprovementRunDecisionKeep,
		Notes:              " fixture-backed local eval ",
		CreatedAt:          now.Add(8 * time.Minute),
		CreatedBy:          " local-runner ",
	}
	got, err := RunLocalDeterministicEval(root, spec)
	if err != nil {
		t.Fatalf("RunLocalDeterministicEval() error = %v", err)
	}
	if !got.Created {
		t.Fatal("Created = false, want true")
	}
	if got.CandidateResult.ResultID != "result-run-root" {
		t.Fatalf("ResultID = %q, want deterministic default", got.CandidateResult.ResultID)
	}
	if got.CandidateResult.BaselineScore != 0.52 || got.CandidateResult.TrainScore != 0.78 || got.CandidateResult.HoldoutScore != 0.74 {
		t.Fatalf("result scores = baseline %.2f train %.2f holdout %.2f, want fixture scores", got.CandidateResult.BaselineScore, got.CandidateResult.TrainScore, got.CandidateResult.HoldoutScore)
	}
	if got.CandidateResult.CreatedBy != "local-runner" || got.CandidateResult.Notes != "fixture-backed local eval" {
		t.Fatalf("result provenance = created_by %q notes %q, want normalized local runner provenance", got.CandidateResult.CreatedBy, got.CandidateResult.Notes)
	}
	if got.Eligibility.State != CandidatePromotionEligibilityStateEligible {
		t.Fatalf("Eligibility.State = %q, want eligible: %#v", got.Eligibility.State, got.Eligibility)
	}
	if !got.OutcomeCreated {
		t.Fatal("OutcomeCreated = false, want true")
	}
	if got.Outcome.Decision != ImprovementRunDecisionKeep {
		t.Fatalf("Outcome.Decision = %q, want keep", got.Outcome.Decision)
	}
	if got.Outcome.PromotionDecisionID != "candidate-promotion-decision-result-run-root" {
		t.Fatalf("Outcome.PromotionDecisionID = %q, want candidate promotion decision ref", got.Outcome.PromotionDecisionID)
	}
	if got.PromotionDecision == nil {
		t.Fatal("PromotionDecision = nil, want eligible promotion decision")
	}
	if !got.PromotionDecisionCreated {
		t.Fatal("PromotionDecisionCreated = false, want true")
	}

	replayed, err := RunLocalDeterministicEval(root, spec)
	if err != nil {
		t.Fatalf("RunLocalDeterministicEval(replay) error = %v", err)
	}
	if replayed.Created {
		t.Fatal("replay Created = true, want false")
	}
	if replayed.CandidateResult.ResultID != got.CandidateResult.ResultID {
		t.Fatalf("replay ResultID = %q, want %q", replayed.CandidateResult.ResultID, got.CandidateResult.ResultID)
	}
	if replayed.OutcomeCreated {
		t.Fatal("replay OutcomeCreated = true, want false")
	}
	if replayed.PromotionDecisionCreated {
		t.Fatal("replay PromotionDecisionCreated = true, want false")
	}

	spec.HoldoutScore = 0.55
	if _, err := RunLocalDeterministicEval(root, spec); err == nil {
		t.Fatal("RunLocalDeterministicEval(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), "already exists with divergent content") {
		t.Fatalf("divergent error = %q, want divergent duplicate context", err.Error())
	}
}

func TestRunLocalDeterministicEvalSurfacesHoldoutRejectedEligibility(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	storeLocalEvalRunnerFixtures(t, root, now)

	got, err := RunLocalDeterministicEval(root, LocalEvalRunnerSpec{
		ResultID:           "result-holdout-failed",
		RunID:              "run-root",
		PromotionPolicyID:  "promotion-policy-local",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.51,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{"none"},
		Decision:           ImprovementRunDecisionKeep,
		CreatedAt:          now.Add(8 * time.Minute),
		CreatedBy:          "local-runner",
	})
	if err != nil {
		t.Fatalf("RunLocalDeterministicEval() error = %v", err)
	}
	if got.Eligibility.State != CandidatePromotionEligibilityStateRejected {
		t.Fatalf("Eligibility.State = %q, want rejected", got.Eligibility.State)
	}
	if got.Outcome.Decision != ImprovementRunDecisionBlocked {
		t.Fatalf("Outcome.Decision = %q, want blocked", got.Outcome.Decision)
	}
	if got.Outcome.PromotionDecisionID != "" {
		t.Fatalf("Outcome.PromotionDecisionID = %q, want empty for blocked outcome", got.Outcome.PromotionDecisionID)
	}
	if got.PromotionDecision != nil {
		t.Fatalf("PromotionDecision = %#v, want nil for blocked outcome", got.PromotionDecision)
	}
	if !containsLocalEvalBlockingReason(got.Eligibility.BlockingReasons, "holdout") {
		t.Fatalf("BlockingReasons = %#v, want holdout blocker", got.Eligibility.BlockingReasons)
	}
}

func TestRunLocalDeterministicEvalRecordsDiscardOutcome(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 2, 10, 30, 0, 0, time.UTC)
	storeLocalEvalRunnerFixtures(t, root, now)

	got, err := RunLocalDeterministicEval(root, LocalEvalRunnerSpec{
		ResultID:           "result-discarded",
		RunID:              "run-root",
		PromotionPolicyID:  "promotion-policy-local",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		RegressionFlags:    []string{"none"},
		Decision:           ImprovementRunDecisionDiscard,
		CreatedAt:          now.Add(8 * time.Minute),
		CreatedBy:          "local-runner",
	})
	if err != nil {
		t.Fatalf("RunLocalDeterministicEval() error = %v", err)
	}
	if got.Eligibility.State != CandidatePromotionEligibilityStateRejected {
		t.Fatalf("Eligibility.State = %q, want rejected", got.Eligibility.State)
	}
	if got.Outcome.Decision != ImprovementRunDecisionDiscard {
		t.Fatalf("Outcome.Decision = %q, want discard", got.Outcome.Decision)
	}
	if got.Outcome.PromotionDecisionID != "" {
		t.Fatalf("Outcome.PromotionDecisionID = %q, want empty for discard", got.Outcome.PromotionDecisionID)
	}
	if got.PromotionDecision != nil {
		t.Fatalf("PromotionDecision = %#v, want nil for discard", got.PromotionDecision)
	}
}

func TestRunLocalDeterministicEvalRejectsMismatchedOrPreRunSpec(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 2, 11, 0, 0, 0, time.UTC)
	storeLocalEvalRunnerFixtures(t, root, now)

	_, err := RunLocalDeterministicEval(root, LocalEvalRunnerSpec{
		RunID:              "run-root",
		CandidateID:        "candidate-other",
		PromotionPolicyID:  "promotion-policy-local",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		Decision:           ImprovementRunDecisionKeep,
		CreatedAt:          now.Add(8 * time.Minute),
		CreatedBy:          "local-runner",
	})
	if err == nil {
		t.Fatal("RunLocalDeterministicEval(mismatched candidate) error = nil, want rejection")
	}
	if !strings.Contains(err.Error(), `candidate_id "candidate-other" does not match run candidate_id "candidate-1"`) {
		t.Fatalf("mismatched candidate error = %q, want run linkage context", err.Error())
	}

	_, err = RunLocalDeterministicEval(root, LocalEvalRunnerSpec{
		ResultID:           "result-before-run",
		RunID:              "run-root",
		PromotionPolicyID:  "promotion-policy-local",
		BaselineScore:      0.52,
		TrainScore:         0.78,
		HoldoutScore:       0.74,
		ComplexityScore:    0.21,
		CompatibilityScore: 0.93,
		ResourceScore:      0.67,
		Decision:           ImprovementRunDecisionKeep,
		CreatedAt:          now.Add(-time.Minute),
		CreatedBy:          "local-runner",
	})
	if err == nil {
		t.Fatal("RunLocalDeterministicEval(pre-run) error = nil, want ordering rejection")
	}
	if !strings.Contains(err.Error(), "created_at must not precede improvement run created_at") {
		t.Fatalf("pre-run error = %q, want created_at ordering context", err.Error())
	}
}

func storeLocalEvalRunnerFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), nil)); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(7*time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-local"
		record.RequiresCanary = false
		record.RequiresOwnerApproval = false
		record.RequiresHoldoutPass = true
		record.EpsilonRule = "epsilon <= 0.01"
		record.RegressionRule = "no_regression_flags"
		record.CompatibilityRule = "compatibility_score >= 0.90"
		record.ResourceRule = "resource_score >= 0.60"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}
}

func containsLocalEvalBlockingReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, want) {
			return true
		}
	}
	return false
}
