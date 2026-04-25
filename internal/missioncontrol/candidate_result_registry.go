package missioncontrol

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

type CandidateResultRef struct {
	ResultID string `json:"result_id"`
}

type CandidateResultRecord struct {
	RecordVersion      int                    `json:"record_version"`
	ResultID           string                 `json:"result_id"`
	RunID              string                 `json:"run_id"`
	CandidateID        string                 `json:"candidate_id"`
	EvalSuiteID        string                 `json:"eval_suite_id"`
	PromotionPolicyID  string                 `json:"promotion_policy_id,omitempty"`
	BaselinePackID     string                 `json:"baseline_pack_id"`
	CandidatePackID    string                 `json:"candidate_pack_id"`
	HotUpdateID        string                 `json:"hot_update_id,omitempty"`
	BaselineScore      float64                `json:"baseline_score"`
	TrainScore         float64                `json:"train_score"`
	HoldoutScore       float64                `json:"holdout_score"`
	ComplexityScore    float64                `json:"complexity_score"`
	CompatibilityScore float64                `json:"compatibility_score"`
	ResourceScore      float64                `json:"resource_score"`
	RegressionFlags    []string               `json:"regression_flags,omitempty"`
	Decision           ImprovementRunDecision `json:"decision"`
	Notes              string                 `json:"notes"`
	CreatedAt          time.Time              `json:"created_at"`
	CreatedBy          string                 `json:"created_by"`
}

var ErrCandidateResultRecordNotFound = errors.New("mission store candidate result record not found")

func StoreCandidateResultsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "candidate_results")
}

func StoreCandidateResultPath(root, resultID string) string {
	return filepath.Join(StoreCandidateResultsDir(root), strings.TrimSpace(resultID)+".json")
}

func NormalizeCandidateResultRef(ref CandidateResultRef) CandidateResultRef {
	ref.ResultID = strings.TrimSpace(ref.ResultID)
	return ref
}

func NormalizeCandidateResultRecord(record CandidateResultRecord) CandidateResultRecord {
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.RegressionFlags = normalizeCandidateResultStrings(record.RegressionFlags)
	record.Notes = strings.TrimSpace(record.Notes)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func CandidateResultImprovementRunRef(record CandidateResultRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func CandidateResultImprovementCandidateRef(record CandidateResultRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func CandidateResultEvalSuiteRef(record CandidateResultRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func CandidateResultPromotionPolicyRef(record CandidateResultRecord) (PromotionPolicyRef, bool) {
	promotionPolicyID := strings.TrimSpace(record.PromotionPolicyID)
	if promotionPolicyID == "" {
		return PromotionPolicyRef{}, false
	}
	return PromotionPolicyRef{PromotionPolicyID: promotionPolicyID}, true
}

func CandidateResultBaselinePackRef(record CandidateResultRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func CandidateResultCandidatePackRef(record CandidateResultRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func CandidateResultHotUpdateGateRef(record CandidateResultRecord) (HotUpdateGateRef, bool) {
	hotUpdateID := strings.TrimSpace(record.HotUpdateID)
	if hotUpdateID == "" {
		return HotUpdateGateRef{}, false
	}
	return HotUpdateGateRef{HotUpdateID: hotUpdateID}, true
}

func ValidateCandidateResultRef(ref CandidateResultRef) error {
	return validateCandidateResultIdentifierField("candidate result ref", "result_id", ref.ResultID)
}

func ValidateCandidateResultRecord(record CandidateResultRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store candidate result record_version must be positive")
	}
	if err := ValidateCandidateResultRef(CandidateResultRef{ResultID: record.ResultID}); err != nil {
		return err
	}
	if err := ValidateImprovementRunRef(CandidateResultImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(CandidateResultImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(CandidateResultEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if promotionPolicyRef, ok := CandidateResultPromotionPolicyRef(record); ok {
		if err := ValidatePromotionPolicyRef(promotionPolicyRef); err != nil {
			return fmt.Errorf("mission store candidate result promotion_policy_id %q: %w", record.PromotionPolicyID, err)
		}
	}
	if err := ValidateRuntimePackRef(CandidateResultBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(CandidateResultCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store candidate result candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		if err := ValidateHotUpdateGateRef(gateRef); err != nil {
			return fmt.Errorf("mission store candidate result hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	for fieldName, score := range map[string]float64{
		"baseline_score":      record.BaselineScore,
		"train_score":         record.TrainScore,
		"holdout_score":       record.HoldoutScore,
		"complexity_score":    record.ComplexityScore,
		"compatibility_score": record.CompatibilityScore,
		"resource_score":      record.ResourceScore,
	} {
		if math.IsNaN(score) || math.IsInf(score, 0) {
			return fmt.Errorf("mission store candidate result %s must be finite", fieldName)
		}
	}
	if record.Decision == "" {
		return fmt.Errorf("mission store candidate result decision is required")
	}
	if !isValidImprovementRunDecision(record.Decision) {
		return fmt.Errorf("mission store candidate result decision %q is invalid", record.Decision)
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store candidate result created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store candidate result created_by is required")
	}
	return nil
}

func StoreCandidateResultRecord(root string, record CandidateResultRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCandidateResultRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateCandidateResultRecord(record); err != nil {
		return err
	}
	if err := validateCandidateResultLinkage(root, record); err != nil {
		return err
	}

	path := StoreCandidateResultPath(root, record.ResultID)
	if existing, err := loadCandidateResultRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store candidate result %q already exists", record.ResultID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadCandidateResultRecord(root, resultID string) (CandidateResultRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidateResultRecord{}, err
	}
	ref := NormalizeCandidateResultRef(CandidateResultRef{ResultID: resultID})
	if err := ValidateCandidateResultRef(ref); err != nil {
		return CandidateResultRecord{}, err
	}
	record, err := loadCandidateResultRecordFile(root, StoreCandidateResultPath(root, ref.ResultID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CandidateResultRecord{}, ErrCandidateResultRecordNotFound
		}
		return CandidateResultRecord{}, err
	}
	return record, nil
}

func ListCandidateResultRecords(root string) ([]CandidateResultRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreCandidateResultsDir(root))
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

	records := make([]CandidateResultRecord, 0, len(names))
	for _, name := range names {
		record, err := loadCandidateResultRecordFile(root, filepath.Join(StoreCandidateResultsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func loadCandidateResultRecordFile(root, path string) (CandidateResultRecord, error) {
	var record CandidateResultRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CandidateResultRecord{}, err
	}
	record = NormalizeCandidateResultRecord(record)
	if err := ValidateCandidateResultRecord(record); err != nil {
		return CandidateResultRecord{}, err
	}
	if err := validateCandidateResultLinkage(root, record); err != nil {
		return CandidateResultRecord{}, err
	}
	return record, nil
}

func validateCandidateResultLinkage(root string, record CandidateResultRecord) error {
	run, err := LoadImprovementRunRecord(root, record.RunID)
	if err != nil {
		return fmt.Errorf("mission store candidate result run_id %q: %w", record.RunID, err)
	}
	if run.CandidateID != record.CandidateID {
		return fmt.Errorf(
			"mission store candidate result candidate_id %q does not match run candidate_id %q",
			record.CandidateID,
			run.CandidateID,
		)
	}
	if run.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q does not match run eval_suite_id %q",
			record.EvalSuiteID,
			run.EvalSuiteID,
		)
	}
	if run.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result baseline_pack_id %q does not match run baseline_pack_id %q",
			record.BaselinePackID,
			run.BaselinePackID,
		)
	}
	if run.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result candidate_pack_id %q does not match run candidate_pack_id %q",
			record.CandidatePackID,
			run.CandidatePackID,
		)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		if run.HotUpdateID == "" {
			return fmt.Errorf("mission store candidate result hot_update_id %q requires run hot_update_id", gateRef.HotUpdateID)
		}
		if run.HotUpdateID != gateRef.HotUpdateID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q does not match run hot_update_id %q",
				gateRef.HotUpdateID,
				run.HotUpdateID,
			)
		}
	}

	candidate, err := LoadImprovementCandidateRecord(root, record.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store candidate result candidate_id %q: %w", record.CandidateID, err)
	}
	if candidate.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result baseline_pack_id %q does not match candidate baseline_pack_id %q",
			record.BaselinePackID,
			candidate.BaselinePackID,
		)
	}
	if candidate.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result candidate_pack_id %q does not match candidate candidate_pack_id %q",
			record.CandidatePackID,
			candidate.CandidatePackID,
		)
	}
	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok && candidate.HotUpdateID != "" && candidate.HotUpdateID != gateRef.HotUpdateID {
		return fmt.Errorf(
			"mission store candidate result hot_update_id %q does not match candidate hot_update_id %q",
			gateRef.HotUpdateID,
			candidate.HotUpdateID,
		)
	}

	evalSuite, err := LoadEvalSuiteRecord(root, record.EvalSuiteID)
	if err != nil {
		return fmt.Errorf("mission store candidate result eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != record.CandidateID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q candidate_id %q does not match result candidate_id %q",
			record.EvalSuiteID,
			evalSuite.CandidateID,
			record.CandidateID,
		)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q baseline_pack_id %q does not match result baseline_pack_id %q",
			record.EvalSuiteID,
			evalSuite.BaselinePackID,
			record.BaselinePackID,
		)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store candidate result eval_suite_id %q candidate_pack_id %q does not match result candidate_pack_id %q",
			record.EvalSuiteID,
			evalSuite.CandidatePackID,
			record.CandidatePackID,
		)
	}

	if promotionPolicyRef, ok := CandidateResultPromotionPolicyRef(record); ok {
		if _, err := LoadPromotionPolicyRecord(root, promotionPolicyRef.PromotionPolicyID); err != nil {
			return fmt.Errorf("mission store candidate result promotion_policy_id %q: %w", promotionPolicyRef.PromotionPolicyID, err)
		}
	}

	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store candidate result baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store candidate result candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	if gateRef, ok := CandidateResultHotUpdateGateRef(record); ok {
		gate, err := LoadHotUpdateGateRecord(root, gateRef.HotUpdateID)
		if err != nil {
			return fmt.Errorf("mission store candidate result hot_update_id %q: %w", gateRef.HotUpdateID, err)
		}
		if gate.CandidatePackID != record.CandidatePackID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q candidate_pack_id %q does not match result candidate_pack_id %q",
				gateRef.HotUpdateID,
				gate.CandidatePackID,
				record.CandidatePackID,
			)
		}
		if gate.PreviousActivePackID != record.BaselinePackID {
			return fmt.Errorf(
				"mission store candidate result hot_update_id %q previous_active_pack_id %q does not match result baseline_pack_id %q",
				gateRef.HotUpdateID,
				gate.PreviousActivePackID,
				record.BaselinePackID,
			)
		}
	}

	return nil
}

func validateCandidateResultIdentifierField(surface, fieldName, value string) error {
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

func normalizeCandidateResultStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
