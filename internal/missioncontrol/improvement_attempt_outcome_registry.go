package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type ImprovementAttemptOutcomeRef struct {
	OutcomeID string `json:"outcome_id"`
}

type ImprovementAttemptOutcomeRecord struct {
	RecordVersion       int                    `json:"record_version"`
	OutcomeID           string                 `json:"outcome_id"`
	ResultID            string                 `json:"result_id"`
	RunID               string                 `json:"run_id"`
	CandidateID         string                 `json:"candidate_id"`
	EvalSuiteID         string                 `json:"eval_suite_id"`
	PromotionPolicyID   string                 `json:"promotion_policy_id,omitempty"`
	BaselinePackID      string                 `json:"baseline_pack_id"`
	CandidatePackID     string                 `json:"candidate_pack_id"`
	EligibilityState    string                 `json:"eligibility_state"`
	Decision            ImprovementRunDecision `json:"decision"`
	PromotionDecisionID string                 `json:"promotion_decision_id,omitempty"`
	BlockingReasons     []string               `json:"blocking_reasons,omitempty"`
	Reason              string                 `json:"reason"`
	CreatedAt           time.Time              `json:"created_at"`
	CreatedBy           string                 `json:"created_by"`
}

var ErrImprovementAttemptOutcomeRecordNotFound = errors.New("mission store improvement attempt outcome record not found")

func StoreImprovementAttemptOutcomesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "improvement_attempt_outcomes")
}

func StoreImprovementAttemptOutcomePath(root, outcomeID string) string {
	return filepath.Join(StoreImprovementAttemptOutcomesDir(root), strings.TrimSpace(outcomeID)+".json")
}

func NormalizeImprovementAttemptOutcomeRef(ref ImprovementAttemptOutcomeRef) ImprovementAttemptOutcomeRef {
	ref.OutcomeID = strings.TrimSpace(ref.OutcomeID)
	return ref
}

func NormalizeImprovementAttemptOutcomeRecord(record ImprovementAttemptOutcomeRecord) ImprovementAttemptOutcomeRecord {
	record.OutcomeID = strings.TrimSpace(record.OutcomeID)
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EligibilityState = strings.TrimSpace(record.EligibilityState)
	record.Decision = ImprovementRunDecision(strings.TrimSpace(string(record.Decision)))
	record.PromotionDecisionID = strings.TrimSpace(record.PromotionDecisionID)
	record.BlockingReasons = normalizeCandidateResultStrings(record.BlockingReasons)
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateImprovementAttemptOutcomeRef(ref ImprovementAttemptOutcomeRef) error {
	return validateRuntimePackIDField("improvement attempt outcome ref", "outcome_id", ref.OutcomeID)
}

func ValidateImprovementAttemptOutcomeRecord(record ImprovementAttemptOutcomeRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store improvement attempt outcome record_version must be positive")
	}
	if err := ValidateImprovementAttemptOutcomeRef(ImprovementAttemptOutcomeRef{OutcomeID: record.OutcomeID}); err != nil {
		return err
	}
	if err := ValidateCandidateResultRef(CandidateResultRef{ResultID: record.ResultID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(ImprovementRunRef{RunID: record.RunID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(ImprovementCandidateRef{CandidateID: record.CandidateID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(EvalSuiteRef{EvalSuiteID: record.EvalSuiteID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if record.PromotionPolicyID != "" {
		if err := ValidatePromotionPolicyRef(PromotionPolicyRef{PromotionPolicyID: record.PromotionPolicyID}); err != nil {
			return fmt.Errorf("mission store improvement attempt outcome promotion_policy_id %q: %w", record.PromotionPolicyID, err)
		}
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.BaselinePackID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.CandidatePackID}); err != nil {
		return fmt.Errorf("mission store improvement attempt outcome candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.EligibilityState == "" {
		return fmt.Errorf("mission store improvement attempt outcome eligibility_state is required")
	}
	if record.Decision == "" {
		return fmt.Errorf("mission store improvement attempt outcome decision is required")
	}
	if !isValidImprovementRunDecision(record.Decision) {
		return fmt.Errorf("mission store improvement attempt outcome decision %q is invalid", record.Decision)
	}
	if record.PromotionDecisionID != "" {
		if err := ValidateCandidatePromotionDecisionRef(CandidatePromotionDecisionRef{PromotionDecisionID: record.PromotionDecisionID}); err != nil {
			return fmt.Errorf("mission store improvement attempt outcome promotion_decision_id %q: %w", record.PromotionDecisionID, err)
		}
	}
	if record.Decision == ImprovementRunDecisionBlocked && len(record.BlockingReasons) == 0 {
		return fmt.Errorf("mission store improvement attempt outcome blocking_reasons are required when decision is %q", ImprovementRunDecisionBlocked)
	}
	if record.Decision == ImprovementRunDecisionKeep && record.EligibilityState == CandidatePromotionEligibilityStateEligible && record.PromotionDecisionID == "" {
		return fmt.Errorf("mission store improvement attempt outcome promotion_decision_id is required for eligible keep decision")
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store improvement attempt outcome reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store improvement attempt outcome created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store improvement attempt outcome created_by is required")
	}
	return nil
}

func StoreImprovementAttemptOutcomeRecord(root string, record ImprovementAttemptOutcomeRecord) (ImprovementAttemptOutcomeRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}
	record = NormalizeImprovementAttemptOutcomeRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateImprovementAttemptOutcomeRecord(record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}
	if err := validateImprovementAttemptOutcomeLinkage(root, record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}

	path := StoreImprovementAttemptOutcomePath(root, record.OutcomeID)
	existing, err := loadImprovementAttemptOutcomeRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return ImprovementAttemptOutcomeRecord{}, false, fmt.Errorf("mission store improvement attempt outcome %q already exists", record.OutcomeID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}
	stored, err := LoadImprovementAttemptOutcomeRecord(root, record.OutcomeID)
	if err != nil {
		return ImprovementAttemptOutcomeRecord{}, false, err
	}
	return stored, true, nil
}

func LoadImprovementAttemptOutcomeRecord(root, outcomeID string) (ImprovementAttemptOutcomeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementAttemptOutcomeRecord{}, err
	}
	ref := NormalizeImprovementAttemptOutcomeRef(ImprovementAttemptOutcomeRef{OutcomeID: outcomeID})
	if err := ValidateImprovementAttemptOutcomeRef(ref); err != nil {
		return ImprovementAttemptOutcomeRecord{}, err
	}
	record, err := loadImprovementAttemptOutcomeRecordFile(root, StoreImprovementAttemptOutcomePath(root, ref.OutcomeID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ImprovementAttemptOutcomeRecord{}, ErrImprovementAttemptOutcomeRecordNotFound
		}
		return ImprovementAttemptOutcomeRecord{}, err
	}
	return record, nil
}

func ListImprovementAttemptOutcomeRecords(root string) ([]ImprovementAttemptOutcomeRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreImprovementAttemptOutcomesDir(root), func(path string) (ImprovementAttemptOutcomeRecord, error) {
		return loadImprovementAttemptOutcomeRecordFile(root, path)
	})
}

func CreateImprovementAttemptOutcomeFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (ImprovementAttemptOutcomeRecord, *CandidatePromotionDecisionRecord, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
	}
	resultRef := NormalizeCandidateResultRef(CandidateResultRef{ResultID: resultID})
	if err := ValidateCandidateResultRef(resultRef); err != nil {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, fmt.Errorf("mission store improvement attempt outcome created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, fmt.Errorf("mission store improvement attempt outcome created_at is required")
	}

	result, err := LoadCandidateResultRecord(root, resultRef.ResultID)
	if err != nil {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
	}
	eligibility, err := EvaluateCandidateResultPromotionEligibility(root, resultRef.ResultID)
	if err != nil {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
	}
	decision := terminalImprovementAttemptDecision(result.Decision, eligibility)

	var promotionDecision *CandidatePromotionDecisionRecord
	promotionDecisionCreated := false
	if decision == ImprovementRunDecisionKeep && eligibility.State == CandidatePromotionEligibilityStateEligible {
		record, created, err := CreateCandidatePromotionDecisionFromEligibleResult(root, resultRef.ResultID, createdBy, createdAt)
		if err != nil {
			return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
		}
		promotionDecision = &record
		promotionDecisionCreated = created
	}

	outcome := ImprovementAttemptOutcomeRecord{
		RecordVersion:     StoreRecordVersion,
		OutcomeID:         improvementAttemptOutcomeIDFromResult(result.ResultID),
		ResultID:          result.ResultID,
		RunID:             result.RunID,
		CandidateID:       result.CandidateID,
		EvalSuiteID:       result.EvalSuiteID,
		PromotionPolicyID: result.PromotionPolicyID,
		BaselinePackID:    result.BaselinePackID,
		CandidatePackID:   result.CandidatePackID,
		EligibilityState:  eligibility.State,
		Decision:          decision,
		BlockingReasons:   eligibility.BlockingReasons,
		Reason:            improvementAttemptOutcomeReason(decision, eligibility),
		CreatedAt:         createdAt,
		CreatedBy:         createdBy,
	}
	if promotionDecision != nil {
		outcome.PromotionDecisionID = promotionDecision.PromotionDecisionID
	}
	stored, outcomeCreated, err := StoreImprovementAttemptOutcomeRecord(root, outcome)
	if err != nil {
		return ImprovementAttemptOutcomeRecord{}, nil, false, false, err
	}
	return stored, promotionDecision, outcomeCreated, promotionDecisionCreated, nil
}

func loadImprovementAttemptOutcomeRecordFile(root, path string) (ImprovementAttemptOutcomeRecord, error) {
	var record ImprovementAttemptOutcomeRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, err
	}
	record = NormalizeImprovementAttemptOutcomeRecord(record)
	if err := ValidateImprovementAttemptOutcomeRecord(record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, err
	}
	if err := validateImprovementAttemptOutcomeLinkage(root, record); err != nil {
		return ImprovementAttemptOutcomeRecord{}, err
	}
	return record, nil
}

func validateImprovementAttemptOutcomeLinkage(root string, record ImprovementAttemptOutcomeRecord) error {
	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store improvement attempt outcome result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID {
		return fmt.Errorf("mission store improvement attempt outcome run_id %q does not match candidate result run_id %q", record.RunID, result.RunID)
	}
	if result.CandidateID != record.CandidateID {
		return fmt.Errorf("mission store improvement attempt outcome candidate_id %q does not match candidate result candidate_id %q", record.CandidateID, result.CandidateID)
	}
	if result.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf("mission store improvement attempt outcome eval_suite_id %q does not match candidate result eval_suite_id %q", record.EvalSuiteID, result.EvalSuiteID)
	}
	if result.PromotionPolicyID != record.PromotionPolicyID {
		return fmt.Errorf("mission store improvement attempt outcome promotion_policy_id %q does not match candidate result promotion_policy_id %q", record.PromotionPolicyID, result.PromotionPolicyID)
	}
	if result.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf("mission store improvement attempt outcome baseline_pack_id %q does not match candidate result baseline_pack_id %q", record.BaselinePackID, result.BaselinePackID)
	}
	if result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store improvement attempt outcome candidate_pack_id %q does not match candidate result candidate_pack_id %q", record.CandidatePackID, result.CandidatePackID)
	}
	eligibility, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if eligibility.State != record.EligibilityState {
		return fmt.Errorf("mission store improvement attempt outcome eligibility_state %q does not match current eligibility_state %q", record.EligibilityState, eligibility.State)
	}
	if expected := terminalImprovementAttemptDecision(result.Decision, eligibility); record.Decision != expected {
		return fmt.Errorf("mission store improvement attempt outcome decision %q does not match expected terminal decision %q", record.Decision, expected)
	}
	if record.PromotionDecisionID != "" {
		decision, err := LoadCandidatePromotionDecisionRecord(root, record.PromotionDecisionID)
		if err != nil {
			return fmt.Errorf("mission store improvement attempt outcome promotion_decision_id %q: %w", record.PromotionDecisionID, err)
		}
		if decision.ResultID != record.ResultID {
			return fmt.Errorf("mission store improvement attempt outcome promotion_decision_id %q result_id %q does not match outcome result_id %q", record.PromotionDecisionID, decision.ResultID, record.ResultID)
		}
	}
	return nil
}

func terminalImprovementAttemptDecision(resultDecision ImprovementRunDecision, eligibility CandidatePromotionEligibilityStatus) ImprovementRunDecision {
	switch resultDecision {
	case ImprovementRunDecisionDiscard,
		ImprovementRunDecisionCrash,
		ImprovementRunDecisionHotUpdated,
		ImprovementRunDecisionPromoted,
		ImprovementRunDecisionRolledBack:
		return resultDecision
	case ImprovementRunDecisionKeep:
		if eligibility.State == CandidatePromotionEligibilityStateEligible {
			return ImprovementRunDecisionKeep
		}
		return ImprovementRunDecisionBlocked
	default:
		return ImprovementRunDecisionBlocked
	}
}

func improvementAttemptOutcomeReason(decision ImprovementRunDecision, eligibility CandidatePromotionEligibilityStatus) string {
	switch decision {
	case ImprovementRunDecisionKeep:
		return "candidate result kept and eligible for promotion decision"
	case ImprovementRunDecisionDiscard:
		return "candidate result discarded by local eval runner"
	case ImprovementRunDecisionBlocked:
		if len(eligibility.BlockingReasons) > 0 {
			return "candidate result blocked: " + strings.Join(eligibility.BlockingReasons, "; ")
		}
		return "candidate result blocked by eligibility state " + eligibility.State
	case ImprovementRunDecisionCrash:
		return "candidate attempt ended with crash decision"
	case ImprovementRunDecisionHotUpdated:
		return "candidate attempt ended with hot-updated decision"
	case ImprovementRunDecisionPromoted:
		return "candidate attempt ended with promoted decision"
	case ImprovementRunDecisionRolledBack:
		return "candidate attempt ended with rolled-back decision"
	default:
		return "candidate attempt ended with blocked decision"
	}
}

func improvementAttemptOutcomeIDFromResult(resultID string) string {
	return "improvement-attempt-outcome-" + strings.TrimSpace(resultID)
}
