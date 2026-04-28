package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

type LocalEvalRunnerSpec struct {
	ResultID           string                 `json:"result_id,omitempty"`
	RunID              string                 `json:"run_id"`
	CandidateID        string                 `json:"candidate_id,omitempty"`
	EvalSuiteID        string                 `json:"eval_suite_id,omitempty"`
	PromotionPolicyID  string                 `json:"promotion_policy_id"`
	BaselineScore      float64                `json:"baseline_score"`
	TrainScore         float64                `json:"train_score"`
	HoldoutScore       float64                `json:"holdout_score"`
	ComplexityScore    float64                `json:"complexity_score"`
	CompatibilityScore float64                `json:"compatibility_score"`
	ResourceScore      float64                `json:"resource_score"`
	RegressionFlags    []string               `json:"regression_flags,omitempty"`
	Decision           ImprovementRunDecision `json:"decision"`
	Notes              string                 `json:"notes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	CreatedBy          string                 `json:"created_by"`
}

type LocalEvalRunnerResult struct {
	CandidateResult          CandidateResultRecord               `json:"candidate_result"`
	Eligibility              CandidatePromotionEligibilityStatus `json:"eligibility"`
	Outcome                  ImprovementAttemptOutcomeRecord     `json:"outcome"`
	PromotionDecision        *CandidatePromotionDecisionRecord   `json:"promotion_decision,omitempty"`
	Created                  bool                                `json:"created"`
	OutcomeCreated           bool                                `json:"outcome_created"`
	PromotionDecisionCreated bool                                `json:"promotion_decision_created,omitempty"`
}

func RunLocalDeterministicEval(root string, spec LocalEvalRunnerSpec) (LocalEvalRunnerResult, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LocalEvalRunnerResult{}, err
	}
	spec = NormalizeLocalEvalRunnerSpec(spec)
	if err := ValidateLocalEvalRunnerSpec(spec); err != nil {
		return LocalEvalRunnerResult{}, err
	}

	run, err := LoadImprovementRunRecord(root, spec.RunID)
	if err != nil {
		return LocalEvalRunnerResult{}, err
	}
	if err := validateLocalEvalRunnerSpecMatchesRun(spec, run); err != nil {
		return LocalEvalRunnerResult{}, err
	}
	candidate, err := LoadImprovementCandidateRecord(root, run.CandidateID)
	if err != nil {
		return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner candidate_id %q: %w", run.CandidateID, err)
	}
	if candidate.BaselinePackID != run.BaselinePackID || candidate.CandidatePackID != run.CandidatePackID {
		return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner run %q does not match improvement candidate %q pack refs", run.RunID, candidate.CandidateID)
	}
	evalSuite, err := LoadEvalSuiteRecord(root, run.EvalSuiteID)
	if err != nil {
		return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner eval_suite_id %q: %w", run.EvalSuiteID, err)
	}
	if err := validateLocalEvalRunnerFrozenEvalSuite(run, evalSuite); err != nil {
		return LocalEvalRunnerResult{}, err
	}
	if _, err := LoadPromotionPolicyRecord(root, spec.PromotionPolicyID); err != nil {
		return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner promotion_policy_id %q: %w", spec.PromotionPolicyID, err)
	}
	if spec.CreatedAt.Before(run.CreatedAt) {
		return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner created_at must not precede improvement run created_at")
	}

	record := NormalizeCandidateResultRecord(CandidateResultRecord{
		RecordVersion:      StoreRecordVersion,
		ResultID:           spec.ResultID,
		RunID:              run.RunID,
		CandidateID:        run.CandidateID,
		EvalSuiteID:        run.EvalSuiteID,
		PromotionPolicyID:  spec.PromotionPolicyID,
		BaselinePackID:     run.BaselinePackID,
		CandidatePackID:    run.CandidatePackID,
		HotUpdateID:        run.HotUpdateID,
		BaselineScore:      spec.BaselineScore,
		TrainScore:         spec.TrainScore,
		HoldoutScore:       spec.HoldoutScore,
		ComplexityScore:    spec.ComplexityScore,
		CompatibilityScore: spec.CompatibilityScore,
		ResourceScore:      spec.ResourceScore,
		RegressionFlags:    spec.RegressionFlags,
		Decision:           spec.Decision,
		Notes:              spec.Notes,
		CreatedAt:          spec.CreatedAt,
		CreatedBy:          spec.CreatedBy,
	})
	created := true
	existing, err := LoadCandidateResultRecord(root, record.ResultID)
	if err == nil {
		if !reflect.DeepEqual(existing, record) {
			return LocalEvalRunnerResult{}, fmt.Errorf("mission store local eval runner candidate result %q already exists with divergent content", record.ResultID)
		}
		created = false
	} else if !errors.Is(err, ErrCandidateResultRecordNotFound) && !errors.Is(err, os.ErrNotExist) {
		return LocalEvalRunnerResult{}, err
	}
	if created {
		if err := StoreCandidateResultRecord(root, record); err != nil {
			return LocalEvalRunnerResult{}, err
		}
	}
	stored, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return LocalEvalRunnerResult{}, err
	}
	eligibility, err := EvaluateCandidateResultPromotionEligibility(root, stored.ResultID)
	if err != nil {
		return LocalEvalRunnerResult{}, err
	}
	outcome, promotionDecision, outcomeCreated, promotionDecisionCreated, err := CreateImprovementAttemptOutcomeFromCandidateResult(root, stored.ResultID, spec.CreatedBy, spec.CreatedAt)
	if err != nil {
		return LocalEvalRunnerResult{}, err
	}
	return LocalEvalRunnerResult{
		CandidateResult:          stored,
		Eligibility:              eligibility,
		Outcome:                  outcome,
		PromotionDecision:        promotionDecision,
		Created:                  created,
		OutcomeCreated:           outcomeCreated,
		PromotionDecisionCreated: promotionDecisionCreated,
	}, nil
}

func NormalizeLocalEvalRunnerSpec(spec LocalEvalRunnerSpec) LocalEvalRunnerSpec {
	spec.RunID = strings.TrimSpace(spec.RunID)
	if strings.TrimSpace(spec.ResultID) == "" && spec.RunID != "" {
		spec.ResultID = "result-" + spec.RunID
	}
	spec.ResultID = strings.TrimSpace(spec.ResultID)
	spec.CandidateID = strings.TrimSpace(spec.CandidateID)
	spec.EvalSuiteID = strings.TrimSpace(spec.EvalSuiteID)
	spec.PromotionPolicyID = strings.TrimSpace(spec.PromotionPolicyID)
	spec.RegressionFlags = normalizeCandidateResultStrings(spec.RegressionFlags)
	spec.Decision = ImprovementRunDecision(strings.TrimSpace(string(spec.Decision)))
	spec.Notes = strings.TrimSpace(spec.Notes)
	spec.CreatedAt = spec.CreatedAt.UTC()
	spec.CreatedBy = strings.TrimSpace(spec.CreatedBy)
	return spec
}

func ValidateLocalEvalRunnerSpec(spec LocalEvalRunnerSpec) error {
	if err := ValidateCandidateResultRef(CandidateResultRef{ResultID: spec.ResultID}); err != nil {
		return err
	}
	if err := ValidateImprovementRunRef(ImprovementRunRef{RunID: spec.RunID}); err != nil {
		return fmt.Errorf("mission store local eval runner run_id %q: %w", spec.RunID, err)
	}
	if spec.CandidateID != "" {
		if err := ValidateImprovementCandidateRef(ImprovementCandidateRef{CandidateID: spec.CandidateID}); err != nil {
			return fmt.Errorf("mission store local eval runner candidate_id %q: %w", spec.CandidateID, err)
		}
	}
	if spec.EvalSuiteID != "" {
		if err := ValidateEvalSuiteRef(EvalSuiteRef{EvalSuiteID: spec.EvalSuiteID}); err != nil {
			return fmt.Errorf("mission store local eval runner eval_suite_id %q: %w", spec.EvalSuiteID, err)
		}
	}
	if err := ValidatePromotionPolicyRef(PromotionPolicyRef{PromotionPolicyID: spec.PromotionPolicyID}); err != nil {
		return fmt.Errorf("mission store local eval runner promotion_policy_id %q: %w", spec.PromotionPolicyID, err)
	}
	candidate := CandidateResultRecord{
		ResultID:           spec.ResultID,
		RunID:              spec.RunID,
		CandidateID:        "candidate-placeholder",
		EvalSuiteID:        "eval-suite-placeholder",
		BaselinePackID:     "pack-placeholder-a",
		CandidatePackID:    "pack-placeholder-b",
		BaselineScore:      spec.BaselineScore,
		TrainScore:         spec.TrainScore,
		HoldoutScore:       spec.HoldoutScore,
		ComplexityScore:    spec.ComplexityScore,
		CompatibilityScore: spec.CompatibilityScore,
		ResourceScore:      spec.ResourceScore,
		Decision:           spec.Decision,
		CreatedAt:          spec.CreatedAt,
		CreatedBy:          spec.CreatedBy,
	}
	if err := validateCandidateResultFiniteScores(candidate); err != nil {
		return err
	}
	if spec.Decision == "" {
		return fmt.Errorf("mission store local eval runner decision is required")
	}
	if !isValidImprovementRunDecision(spec.Decision) {
		return fmt.Errorf("mission store local eval runner decision %q is invalid", spec.Decision)
	}
	if spec.CreatedAt.IsZero() {
		return fmt.Errorf("mission store local eval runner created_at is required")
	}
	if spec.CreatedBy == "" {
		return fmt.Errorf("mission store local eval runner created_by is required")
	}
	return nil
}

func validateLocalEvalRunnerSpecMatchesRun(spec LocalEvalRunnerSpec, run ImprovementRunRecord) error {
	if spec.CandidateID != "" && spec.CandidateID != run.CandidateID {
		return fmt.Errorf("mission store local eval runner candidate_id %q does not match run candidate_id %q", spec.CandidateID, run.CandidateID)
	}
	if spec.EvalSuiteID != "" && spec.EvalSuiteID != run.EvalSuiteID {
		return fmt.Errorf("mission store local eval runner eval_suite_id %q does not match run eval_suite_id %q", spec.EvalSuiteID, run.EvalSuiteID)
	}
	if run.CandidateID == "" {
		return fmt.Errorf("mission store local eval runner run %q candidate_id is required", run.RunID)
	}
	if run.EvalSuiteID == "" {
		return fmt.Errorf("mission store local eval runner run %q eval_suite_id is required", run.RunID)
	}
	if run.BaselinePackID == "" {
		return fmt.Errorf("mission store local eval runner run %q baseline_pack_id is required", run.RunID)
	}
	if run.CandidatePackID == "" {
		return fmt.Errorf("mission store local eval runner run %q candidate_pack_id is required", run.RunID)
	}
	return nil
}

func validateLocalEvalRunnerFrozenEvalSuite(run ImprovementRunRecord, evalSuite EvalSuiteRecord) error {
	if !evalSuite.FrozenForRun {
		return fmt.Errorf("mission store local eval runner eval_suite_id %q must be frozen_for_run", evalSuite.EvalSuiteID)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != run.CandidateID {
		return fmt.Errorf("mission store local eval runner eval_suite_id %q candidate_id %q does not match run candidate_id %q", evalSuite.EvalSuiteID, evalSuite.CandidateID, run.CandidateID)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != run.BaselinePackID {
		return fmt.Errorf("mission store local eval runner eval_suite_id %q baseline_pack_id %q does not match run baseline_pack_id %q", evalSuite.EvalSuiteID, evalSuite.BaselinePackID, run.BaselinePackID)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != run.CandidatePackID {
		return fmt.Errorf("mission store local eval runner eval_suite_id %q candidate_pack_id %q does not match run candidate_pack_id %q", evalSuite.EvalSuiteID, evalSuite.CandidatePackID, run.CandidatePackID)
	}
	return nil
}
