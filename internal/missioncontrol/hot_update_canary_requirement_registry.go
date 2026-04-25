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

type HotUpdateCanaryRequirementRef struct {
	CanaryRequirementID string `json:"canary_requirement_id"`
}

type HotUpdateCanaryRequirementState string

const (
	HotUpdateCanaryRequirementStateRequired HotUpdateCanaryRequirementState = "required"
)

type HotUpdateCanaryRequirementRecord struct {
	RecordVersion         int                             `json:"record_version"`
	CanaryRequirementID   string                          `json:"canary_requirement_id"`
	ResultID              string                          `json:"result_id"`
	RunID                 string                          `json:"run_id"`
	CandidateID           string                          `json:"candidate_id"`
	EvalSuiteID           string                          `json:"eval_suite_id"`
	PromotionPolicyID     string                          `json:"promotion_policy_id"`
	BaselinePackID        string                          `json:"baseline_pack_id"`
	CandidatePackID       string                          `json:"candidate_pack_id"`
	EligibilityState      string                          `json:"eligibility_state"`
	RequiredByPolicy      bool                            `json:"required_by_policy"`
	OwnerApprovalRequired bool                            `json:"owner_approval_required"`
	State                 HotUpdateCanaryRequirementState `json:"state"`
	Reason                string                          `json:"reason"`
	CreatedAt             time.Time                       `json:"created_at"`
	CreatedBy             string                          `json:"created_by"`
}

var ErrHotUpdateCanaryRequirementRecordNotFound = errors.New("mission store hot-update canary requirement record not found")

func StoreHotUpdateCanaryRequirementsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_canary_requirements")
}

func StoreHotUpdateCanaryRequirementPath(root, canaryRequirementID string) string {
	return filepath.Join(StoreHotUpdateCanaryRequirementsDir(root), strings.TrimSpace(canaryRequirementID)+".json")
}

func HotUpdateCanaryRequirementIDFromResult(resultID string) string {
	return "hot-update-canary-requirement-" + strings.TrimSpace(resultID)
}

func NormalizeHotUpdateCanaryRequirementRef(ref HotUpdateCanaryRequirementRef) HotUpdateCanaryRequirementRef {
	ref.CanaryRequirementID = strings.TrimSpace(ref.CanaryRequirementID)
	return ref
}

func NormalizeHotUpdateCanaryRequirementRecord(record HotUpdateCanaryRequirementRecord) HotUpdateCanaryRequirementRecord {
	record.CanaryRequirementID = strings.TrimSpace(record.CanaryRequirementID)
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EligibilityState = strings.TrimSpace(record.EligibilityState)
	record.State = HotUpdateCanaryRequirementState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateHotUpdateCanaryRequirementRef(ref HotUpdateCanaryRequirementRef) error {
	return validateHotUpdateCanaryRequirementIdentifierField("hot-update canary requirement ref", "canary_requirement_id", ref.CanaryRequirementID)
}

func HotUpdateCanaryRequirementCandidateResultRef(record HotUpdateCanaryRequirementRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func HotUpdateCanaryRequirementImprovementRunRef(record HotUpdateCanaryRequirementRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func HotUpdateCanaryRequirementImprovementCandidateRef(record HotUpdateCanaryRequirementRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func HotUpdateCanaryRequirementEvalSuiteRef(record HotUpdateCanaryRequirementRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func HotUpdateCanaryRequirementPromotionPolicyRef(record HotUpdateCanaryRequirementRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func HotUpdateCanaryRequirementBaselinePackRef(record HotUpdateCanaryRequirementRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func HotUpdateCanaryRequirementCandidatePackRef(record HotUpdateCanaryRequirementRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateHotUpdateCanaryRequirementRecord(record HotUpdateCanaryRequirementRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update canary requirement record_version must be positive")
	}
	if err := ValidateHotUpdateCanaryRequirementRef(HotUpdateCanaryRequirementRef{CanaryRequirementID: record.CanaryRequirementID}); err != nil {
		return err
	}
	if err := ValidateCandidateResultRef(HotUpdateCanaryRequirementCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(HotUpdateCanaryRequirementImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(HotUpdateCanaryRequirementImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(HotUpdateCanaryRequirementEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(HotUpdateCanaryRequirementPromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanaryRequirementBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanaryRequirementCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.CanaryRequirementID != HotUpdateCanaryRequirementIDFromResult(record.ResultID) {
		return fmt.Errorf("mission store hot-update canary requirement canary_requirement_id %q does not match deterministic canary_requirement_id %q", record.CanaryRequirementID, HotUpdateCanaryRequirementIDFromResult(record.ResultID))
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(record.EligibilityState) {
		return fmt.Errorf("mission store hot-update canary requirement eligibility_state %q is invalid", record.EligibilityState)
	}
	if !record.RequiredByPolicy {
		return fmt.Errorf("mission store hot-update canary requirement required_by_policy must be true")
	}
	if record.State != HotUpdateCanaryRequirementStateRequired {
		return fmt.Errorf("mission store hot-update canary requirement state must be %q", HotUpdateCanaryRequirementStateRequired)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update canary requirement reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update canary requirement created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update canary requirement created_by is required")
	}
	return nil
}

func StoreHotUpdateCanaryRequirementRecord(root string, record HotUpdateCanaryRequirementRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeHotUpdateCanaryRequirementRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateCanaryRequirementRecord(record); err != nil {
		return err
	}
	if err := validateHotUpdateCanaryRequirementLinkage(root, record); err != nil {
		return err
	}

	path := StoreHotUpdateCanaryRequirementPath(root, record.CanaryRequirementID)
	if existing, err := loadHotUpdateCanaryRequirementRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store hot-update canary requirement %q already exists", record.CanaryRequirementID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	records, err := ListHotUpdateCanaryRequirementRecords(root)
	if err != nil {
		return err
	}
	for _, existing := range records {
		if existing.ResultID == record.ResultID && existing.CanaryRequirementID != record.CanaryRequirementID {
			return fmt.Errorf(
				"mission store hot-update canary requirement for result_id %q already exists as %q",
				record.ResultID,
				existing.CanaryRequirementID,
			)
		}
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadHotUpdateCanaryRequirementRecord(root, canaryRequirementID string) (HotUpdateCanaryRequirementRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryRequirementRecord{}, err
	}
	ref := NormalizeHotUpdateCanaryRequirementRef(HotUpdateCanaryRequirementRef{CanaryRequirementID: canaryRequirementID})
	if err := ValidateHotUpdateCanaryRequirementRef(ref); err != nil {
		return HotUpdateCanaryRequirementRecord{}, err
	}
	record, err := loadHotUpdateCanaryRequirementRecordFile(root, StoreHotUpdateCanaryRequirementPath(root, ref.CanaryRequirementID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateCanaryRequirementRecord{}, ErrHotUpdateCanaryRequirementRecordNotFound
		}
		return HotUpdateCanaryRequirementRecord{}, err
	}
	return record, nil
}

func ListHotUpdateCanaryRequirementRecords(root string) ([]HotUpdateCanaryRequirementRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreHotUpdateCanaryRequirementsDir(root))
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

	records := make([]HotUpdateCanaryRequirementRecord, 0, len(names))
	for _, name := range names {
		record, err := loadHotUpdateCanaryRequirementRecordFile(root, filepath.Join(StoreHotUpdateCanaryRequirementsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func CreateHotUpdateCanaryRequirementFromCandidateResult(root, resultID, createdBy string, createdAt time.Time) (HotUpdateCanaryRequirementRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}
	resultRef := NormalizeCandidateResultRef(CandidateResultRef{ResultID: resultID})
	if err := ValidateCandidateResultRef(resultRef); err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateCanaryRequirementRecord{}, false, fmt.Errorf("mission store hot-update canary requirement created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateCanaryRequirementRecord{}, false, fmt.Errorf("mission store hot-update canary requirement created_at is required")
	}
	if _, err := LoadCandidateResultRecord(root, resultRef.ResultID); err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, resultRef.ResultID)
	if err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(status.State) {
		return HotUpdateCanaryRequirementRecord{}, false, fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update canary requirement", resultRef.ResultID, status.State)
	}

	record := NormalizeHotUpdateCanaryRequirementRecord(HotUpdateCanaryRequirementRecord{
		RecordVersion:         StoreRecordVersion,
		CanaryRequirementID:   HotUpdateCanaryRequirementIDFromResult(resultRef.ResultID),
		ResultID:              status.ResultID,
		RunID:                 status.RunID,
		CandidateID:           status.CandidateID,
		EvalSuiteID:           status.EvalSuiteID,
		PromotionPolicyID:     status.PromotionPolicyID,
		BaselinePackID:        status.BaselinePackID,
		CandidatePackID:       status.CandidatePackID,
		EligibilityState:      status.State,
		RequiredByPolicy:      true,
		OwnerApprovalRequired: status.State == CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired,
		State:                 HotUpdateCanaryRequirementStateRequired,
		Reason:                "candidate result requires canary before promotion",
		CreatedAt:             createdAt,
		CreatedBy:             createdBy,
	})

	existing, err := LoadHotUpdateCanaryRequirementRecord(root, record.CanaryRequirementID)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateCanaryRequirementRecord{}, false, fmt.Errorf("mission store hot-update canary requirement %q already exists", record.CanaryRequirementID)
	}
	if !errors.Is(err, ErrHotUpdateCanaryRequirementRecordNotFound) {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}

	if err := StoreHotUpdateCanaryRequirementRecord(root, record); err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}
	stored, err := LoadHotUpdateCanaryRequirementRecord(root, record.CanaryRequirementID)
	if err != nil {
		return HotUpdateCanaryRequirementRecord{}, false, err
	}
	return stored, true, nil
}

func loadHotUpdateCanaryRequirementRecordFile(root, path string) (HotUpdateCanaryRequirementRecord, error) {
	var record HotUpdateCanaryRequirementRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateCanaryRequirementRecord{}, err
	}
	record = NormalizeHotUpdateCanaryRequirementRecord(record)
	if err := ValidateHotUpdateCanaryRequirementRecord(record); err != nil {
		return HotUpdateCanaryRequirementRecord{}, err
	}
	if err := validateHotUpdateCanaryRequirementLinkage(root, record); err != nil {
		return HotUpdateCanaryRequirementRecord{}, err
	}
	return record, nil
}

func validateHotUpdateCanaryRequirementLinkage(root string, record HotUpdateCanaryRequirementRecord) error {
	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary requirement result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID {
		return fmt.Errorf("mission store hot-update canary requirement run_id %q does not match candidate result run_id %q", record.RunID, result.RunID)
	}
	if result.CandidateID != record.CandidateID {
		return fmt.Errorf("mission store hot-update canary requirement candidate_id %q does not match candidate result candidate_id %q", record.CandidateID, result.CandidateID)
	}
	if result.EvalSuiteID != record.EvalSuiteID {
		return fmt.Errorf("mission store hot-update canary requirement eval_suite_id %q does not match candidate result eval_suite_id %q", record.EvalSuiteID, result.EvalSuiteID)
	}
	if result.PromotionPolicyID != record.PromotionPolicyID {
		return fmt.Errorf("mission store hot-update canary requirement promotion_policy_id %q does not match candidate result promotion_policy_id %q", record.PromotionPolicyID, result.PromotionPolicyID)
	}
	if result.BaselinePackID != record.BaselinePackID {
		return fmt.Errorf("mission store hot-update canary requirement baseline_pack_id %q does not match candidate result baseline_pack_id %q", record.BaselinePackID, result.BaselinePackID)
	}
	if result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary requirement candidate_pack_id %q does not match candidate result candidate_pack_id %q", record.CandidatePackID, result.CandidatePackID)
	}

	if _, err := LoadImprovementRunRecord(root, record.RunID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement run_id %q: %w", record.RunID, err)
	}
	if _, err := LoadImprovementCandidateRecord(root, record.CandidateID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement candidate_id %q: %w", record.CandidateID, err)
	}
	if _, err := LoadEvalSuiteRecord(root, record.EvalSuiteID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if _, err := LoadPromotionPolicyRecord(root, record.PromotionPolicyID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary requirement candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(status.State) {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update canary requirement", record.ResultID, status.State)
	}
	if status.RunID != record.RunID ||
		status.CandidateID != record.CandidateID ||
		status.EvalSuiteID != record.EvalSuiteID ||
		status.PromotionPolicyID != record.PromotionPolicyID ||
		status.BaselinePackID != record.BaselinePackID ||
		status.CandidatePackID != record.CandidatePackID ||
		status.State != record.EligibilityState ||
		status.CanaryRequired != record.RequiredByPolicy ||
		status.OwnerApprovalRequired != record.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update canary requirement %q does not match derived promotion eligibility status", record.CanaryRequirementID)
	}
	return nil
}

func isValidHotUpdateCanaryRequirementEligibilityState(state string) bool {
	switch state {
	case CandidatePromotionEligibilityStateCanaryRequired,
		CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired:
		return true
	default:
		return false
	}
}

func validateHotUpdateCanaryRequirementIdentifierField(surface, fieldName, value string) error {
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
