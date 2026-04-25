package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

type CandidatePromotionDecisionRef struct {
	PromotionDecisionID string `json:"promotion_decision_id"`
}

type CandidatePromotionDecision string

const (
	CandidatePromotionDecisionSelectedForPromotion CandidatePromotionDecision = "selected_for_promotion"
)

type CandidatePromotionDecisionRecord struct {
	RecordVersion       int                        `json:"record_version"`
	PromotionDecisionID string                     `json:"promotion_decision_id"`
	ResultID            string                     `json:"result_id"`
	RunID               string                     `json:"run_id"`
	CandidateID         string                     `json:"candidate_id"`
	EvalSuiteID         string                     `json:"eval_suite_id"`
	PromotionPolicyID   string                     `json:"promotion_policy_id"`
	BaselinePackID      string                     `json:"baseline_pack_id"`
	CandidatePackID     string                     `json:"candidate_pack_id"`
	EligibilityState    string                     `json:"eligibility_state"`
	Decision            CandidatePromotionDecision `json:"decision"`
	Reason              string                     `json:"reason"`
	CreatedAt           time.Time                  `json:"created_at"`
	CreatedBy           string                     `json:"created_by"`
	Notes               string                     `json:"notes,omitempty"`
}

var ErrCandidatePromotionDecisionRecordNotFound = errors.New("mission store candidate promotion decision record not found")

func StoreCandidatePromotionDecisionsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "candidate_promotion_decisions")
}

func StoreCandidatePromotionDecisionPath(root, promotionDecisionID string) string {
	return filepath.Join(StoreCandidatePromotionDecisionsDir(root), strings.TrimSpace(promotionDecisionID)+".json")
}

func NormalizeCandidatePromotionDecisionRef(ref CandidatePromotionDecisionRef) CandidatePromotionDecisionRef {
	ref.PromotionDecisionID = strings.TrimSpace(ref.PromotionDecisionID)
	return ref
}

func NormalizeCandidatePromotionDecisionRecord(record CandidatePromotionDecisionRecord) CandidatePromotionDecisionRecord {
	record.PromotionDecisionID = strings.TrimSpace(record.PromotionDecisionID)
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EligibilityState = strings.TrimSpace(record.EligibilityState)
	record.Decision = CandidatePromotionDecision(strings.TrimSpace(string(record.Decision)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	record.Notes = strings.TrimSpace(record.Notes)
	return record
}

func ValidateCandidatePromotionDecisionRef(ref CandidatePromotionDecisionRef) error {
	return validateCandidatePromotionDecisionIdentifierField("candidate promotion decision ref", "promotion_decision_id", ref.PromotionDecisionID)
}

func CandidatePromotionDecisionCandidateResultRef(record CandidatePromotionDecisionRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func CandidatePromotionDecisionImprovementRunRef(record CandidatePromotionDecisionRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func CandidatePromotionDecisionImprovementCandidateRef(record CandidatePromotionDecisionRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func CandidatePromotionDecisionEvalSuiteRef(record CandidatePromotionDecisionRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func CandidatePromotionDecisionPromotionPolicyRef(record CandidatePromotionDecisionRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func CandidatePromotionDecisionBaselinePackRef(record CandidatePromotionDecisionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func CandidatePromotionDecisionCandidatePackRef(record CandidatePromotionDecisionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateCandidatePromotionDecisionRecord(record CandidatePromotionDecisionRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store candidate promotion decision record_version must be positive")
	}
	if err := ValidateCandidatePromotionDecisionRef(CandidatePromotionDecisionRef{PromotionDecisionID: record.PromotionDecisionID}); err != nil {
		return err
	}
	if err := ValidateCandidateResultRef(CandidatePromotionDecisionCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(CandidatePromotionDecisionImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(CandidatePromotionDecisionImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(CandidatePromotionDecisionEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(CandidatePromotionDecisionPromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(CandidatePromotionDecisionBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(CandidatePromotionDecisionCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.EligibilityState != CandidatePromotionEligibilityStateEligible {
		return fmt.Errorf("mission store candidate promotion decision eligibility_state must be %q", CandidatePromotionEligibilityStateEligible)
	}
	if record.Decision != CandidatePromotionDecisionSelectedForPromotion {
		return fmt.Errorf("mission store candidate promotion decision decision must be %q", CandidatePromotionDecisionSelectedForPromotion)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store candidate promotion decision reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store candidate promotion decision created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store candidate promotion decision created_by is required")
	}
	return nil
}

func StoreCandidatePromotionDecisionRecord(root string, record CandidatePromotionDecisionRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCandidatePromotionDecisionRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateCandidatePromotionDecisionRecord(record); err != nil {
		return err
	}
	if err := validateCandidatePromotionDecisionLinkage(root, record); err != nil {
		return err
	}

	path := StoreCandidatePromotionDecisionPath(root, record.PromotionDecisionID)
	if existing, err := loadCandidatePromotionDecisionRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store candidate promotion decision %q already exists", record.PromotionDecisionID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	records, err := ListCandidatePromotionDecisionRecords(root)
	if err != nil {
		return err
	}
	for _, existing := range records {
		if existing.ResultID == record.ResultID && existing.PromotionDecisionID != record.PromotionDecisionID {
			return fmt.Errorf(
				"mission store candidate promotion decision for result_id %q already exists as %q",
				record.ResultID,
				existing.PromotionDecisionID,
			)
		}
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadCandidatePromotionDecisionRecord(root, promotionDecisionID string) (CandidatePromotionDecisionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidatePromotionDecisionRecord{}, err
	}
	ref := NormalizeCandidatePromotionDecisionRef(CandidatePromotionDecisionRef{PromotionDecisionID: promotionDecisionID})
	if err := ValidateCandidatePromotionDecisionRef(ref); err != nil {
		return CandidatePromotionDecisionRecord{}, err
	}
	record, err := loadCandidatePromotionDecisionRecordFile(root, StoreCandidatePromotionDecisionPath(root, ref.PromotionDecisionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CandidatePromotionDecisionRecord{}, ErrCandidatePromotionDecisionRecordNotFound
		}
		return CandidatePromotionDecisionRecord{}, err
	}
	return record, nil
}

func ListCandidatePromotionDecisionRecords(root string) ([]CandidatePromotionDecisionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreCandidatePromotionDecisionsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	records := make([]CandidatePromotionDecisionRecord, 0, len(names))
	for _, name := range names {
		record, err := loadCandidatePromotionDecisionRecordFile(root, filepath.Join(StoreCandidatePromotionDecisionsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func CreateCandidatePromotionDecisionFromEligibleResult(root, resultID, createdBy string, createdAt time.Time) (CandidatePromotionDecisionRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidatePromotionDecisionRecord{}, false, err
	}
	resultRef := NormalizeCandidateResultRef(CandidateResultRef{ResultID: resultID})
	if err := ValidateCandidateResultRef(resultRef); err != nil {
		return CandidatePromotionDecisionRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return CandidatePromotionDecisionRecord{}, false, fmt.Errorf("mission store candidate promotion decision created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return CandidatePromotionDecisionRecord{}, false, fmt.Errorf("mission store candidate promotion decision created_at is required")
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, resultRef.ResultID)
	if err != nil {
		return CandidatePromotionDecisionRecord{}, false, err
	}
	if status.State != CandidatePromotionEligibilityStateEligible {
		return CandidatePromotionDecisionRecord{}, false, fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit promotion decision", resultRef.ResultID, status.State)
	}

	record := NormalizeCandidatePromotionDecisionRecord(CandidatePromotionDecisionRecord{
		RecordVersion:       StoreRecordVersion,
		PromotionDecisionID: candidatePromotionDecisionIDFromResult(resultRef.ResultID),
		ResultID:            status.ResultID,
		RunID:               status.RunID,
		CandidateID:         status.CandidateID,
		EvalSuiteID:         status.EvalSuiteID,
		PromotionPolicyID:   status.PromotionPolicyID,
		BaselinePackID:      status.BaselinePackID,
		CandidatePackID:     status.CandidatePackID,
		EligibilityState:    status.State,
		Decision:            CandidatePromotionDecisionSelectedForPromotion,
		Reason:              "candidate result eligible for promotion",
		CreatedAt:           createdAt,
		CreatedBy:           createdBy,
	})

	existing, err := LoadCandidatePromotionDecisionRecord(root, record.PromotionDecisionID)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return CandidatePromotionDecisionRecord{}, false, fmt.Errorf("mission store candidate promotion decision %q already exists", record.PromotionDecisionID)
	}
	if !errors.Is(err, ErrCandidatePromotionDecisionRecordNotFound) {
		return CandidatePromotionDecisionRecord{}, false, err
	}

	if err := StoreCandidatePromotionDecisionRecord(root, record); err != nil {
		return CandidatePromotionDecisionRecord{}, false, err
	}
	stored, err := LoadCandidatePromotionDecisionRecord(root, record.PromotionDecisionID)
	if err != nil {
		return CandidatePromotionDecisionRecord{}, false, err
	}
	return stored, true, nil
}

func loadCandidatePromotionDecisionRecordFile(root, path string) (CandidatePromotionDecisionRecord, error) {
	var record CandidatePromotionDecisionRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CandidatePromotionDecisionRecord{}, err
	}
	record = NormalizeCandidatePromotionDecisionRecord(record)
	if err := ValidateCandidatePromotionDecisionRecord(record); err != nil {
		return CandidatePromotionDecisionRecord{}, err
	}
	if err := validateCandidatePromotionDecisionLinkage(root, record); err != nil {
		return CandidatePromotionDecisionRecord{}, err
	}
	return record, nil
}

func validateCandidatePromotionDecisionLinkage(root string, record CandidatePromotionDecisionRecord) error {
	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store candidate promotion decision result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID {
		return fmt.Errorf("mission store candidate promotion decision run_id %q does not match candidate result run_id %q", record.RunID, result.RunID)
	}
	if result.CandidateID != record.CandidateID {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q does not match candidate result candidate_id %q", record.CandidateID, result.CandidateID)
	}
	if result.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf("mission store candidate promotion decision eval_suite_id %q does not match candidate result eval_suite_id %q", record.EvalSuiteID, result.EvalSuiteID)
	}
	if result.PromotionPolicyID != record.PromotionPolicyID {
		return fmt.Errorf("mission store candidate promotion decision promotion_policy_id %q does not match candidate result promotion_policy_id %q", record.PromotionPolicyID, result.PromotionPolicyID)
	}
	if result.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q does not match candidate result baseline_pack_id %q", record.BaselinePackID, result.BaselinePackID)
	}
	if result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q does not match candidate result candidate_pack_id %q", record.CandidatePackID, result.CandidatePackID)
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if status.State != CandidatePromotionEligibilityStateEligible {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit promotion decision", record.ResultID, status.State)
	}
	if status.RunID != record.RunID ||
		status.CandidateID != record.CandidateID ||
		status.EvalSuiteID != record.EvalSuiteID ||
		status.PromotionPolicyID != record.PromotionPolicyID ||
		status.BaselinePackID != record.BaselinePackID ||
		status.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision %q does not match derived promotion eligibility status", record.PromotionDecisionID)
	}
	return nil
}

func candidatePromotionDecisionIDFromResult(resultID string) string {
	return "candidate-promotion-decision-" + strings.TrimSpace(resultID)
}

func validateCandidatePromotionDecisionIdentifierField(surface, fieldName, value string) error {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fmt.Errorf("%s %s is required", surface, fieldName)
	}
	if strings.HasPrefix(normalized, ".") || strings.HasSuffix(normalized, ".") {
		return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
	}
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
		}
	}
	return nil
}
