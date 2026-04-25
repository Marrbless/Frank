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

type HotUpdateCanaryEvidenceState string

const (
	HotUpdateCanaryEvidenceStatePassed  HotUpdateCanaryEvidenceState = "passed"
	HotUpdateCanaryEvidenceStateFailed  HotUpdateCanaryEvidenceState = "failed"
	HotUpdateCanaryEvidenceStateBlocked HotUpdateCanaryEvidenceState = "blocked"
	HotUpdateCanaryEvidenceStateExpired HotUpdateCanaryEvidenceState = "expired"
)

type HotUpdateCanaryEvidenceRef struct {
	CanaryEvidenceID string `json:"canary_evidence_id"`
}

type HotUpdateCanaryEvidenceRecord struct {
	RecordVersion       int                          `json:"record_version"`
	CanaryEvidenceID    string                       `json:"canary_evidence_id"`
	CanaryRequirementID string                       `json:"canary_requirement_id"`
	ResultID            string                       `json:"result_id"`
	RunID               string                       `json:"run_id"`
	CandidateID         string                       `json:"candidate_id"`
	EvalSuiteID         string                       `json:"eval_suite_id"`
	PromotionPolicyID   string                       `json:"promotion_policy_id"`
	BaselinePackID      string                       `json:"baseline_pack_id"`
	CandidatePackID     string                       `json:"candidate_pack_id"`
	EvidenceState       HotUpdateCanaryEvidenceState `json:"evidence_state"`
	Passed              bool                         `json:"passed"`
	Reason              string                       `json:"reason"`
	ObservedAt          time.Time                    `json:"observed_at"`
	CreatedAt           time.Time                    `json:"created_at"`
	CreatedBy           string                       `json:"created_by"`
}

var ErrHotUpdateCanaryEvidenceRecordNotFound = errors.New("mission store hot-update canary evidence record not found")

func StoreHotUpdateCanaryEvidenceDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_canary_evidence")
}

func StoreHotUpdateCanaryEvidencePath(root, canaryEvidenceID string) string {
	return filepath.Join(StoreHotUpdateCanaryEvidenceDir(root), strings.TrimSpace(canaryEvidenceID)+".json")
}

func HotUpdateCanaryEvidenceIDFromRequirementObservedAt(canaryRequirementID string, observedAt time.Time) string {
	observedAt = observedAt.UTC()
	observedAtCompact := strings.ReplaceAll(observedAt.Format("20060102T150405.000000000Z"), ".", "")
	return "hot-update-canary-evidence-" + strings.TrimSpace(canaryRequirementID) + "-" + observedAtCompact
}

func NormalizeHotUpdateCanaryEvidenceRef(ref HotUpdateCanaryEvidenceRef) HotUpdateCanaryEvidenceRef {
	ref.CanaryEvidenceID = strings.TrimSpace(ref.CanaryEvidenceID)
	return ref
}

func NormalizeHotUpdateCanaryEvidenceRecord(record HotUpdateCanaryEvidenceRecord) HotUpdateCanaryEvidenceRecord {
	record.CanaryEvidenceID = strings.TrimSpace(record.CanaryEvidenceID)
	record.CanaryRequirementID = strings.TrimSpace(record.CanaryRequirementID)
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EvidenceState = HotUpdateCanaryEvidenceState(strings.TrimSpace(string(record.EvidenceState)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.ObservedAt = record.ObservedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateHotUpdateCanaryEvidenceRef(ref HotUpdateCanaryEvidenceRef) error {
	return validateHotUpdateCanaryRequirementIdentifierField("hot-update canary evidence ref", "canary_evidence_id", ref.CanaryEvidenceID)
}

func HotUpdateCanaryEvidenceRequirementRef(record HotUpdateCanaryEvidenceRecord) HotUpdateCanaryRequirementRef {
	return HotUpdateCanaryRequirementRef{CanaryRequirementID: strings.TrimSpace(record.CanaryRequirementID)}
}

func HotUpdateCanaryEvidenceCandidateResultRef(record HotUpdateCanaryEvidenceRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func HotUpdateCanaryEvidenceImprovementRunRef(record HotUpdateCanaryEvidenceRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func HotUpdateCanaryEvidenceImprovementCandidateRef(record HotUpdateCanaryEvidenceRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func HotUpdateCanaryEvidenceEvalSuiteRef(record HotUpdateCanaryEvidenceRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func HotUpdateCanaryEvidencePromotionPolicyRef(record HotUpdateCanaryEvidenceRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func HotUpdateCanaryEvidenceBaselinePackRef(record HotUpdateCanaryEvidenceRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func HotUpdateCanaryEvidenceCandidatePackRef(record HotUpdateCanaryEvidenceRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateHotUpdateCanaryEvidenceRecord(record HotUpdateCanaryEvidenceRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update canary evidence record_version must be positive")
	}
	if err := ValidateHotUpdateCanaryEvidenceRef(HotUpdateCanaryEvidenceRef{CanaryEvidenceID: record.CanaryEvidenceID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateCanaryRequirementRef(HotUpdateCanaryEvidenceRequirementRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	if err := ValidateCandidateResultRef(HotUpdateCanaryEvidenceCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(HotUpdateCanaryEvidenceImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(HotUpdateCanaryEvidenceImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(HotUpdateCanaryEvidenceEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(HotUpdateCanaryEvidencePromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanaryEvidenceBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanaryEvidenceCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.ObservedAt.IsZero() {
		return fmt.Errorf("mission store hot-update canary evidence observed_at is required")
	}
	if record.CanaryEvidenceID != HotUpdateCanaryEvidenceIDFromRequirementObservedAt(record.CanaryRequirementID, record.ObservedAt) {
		return fmt.Errorf("mission store hot-update canary evidence canary_evidence_id %q does not match deterministic canary_evidence_id %q", record.CanaryEvidenceID, HotUpdateCanaryEvidenceIDFromRequirementObservedAt(record.CanaryRequirementID, record.ObservedAt))
	}
	if !isValidHotUpdateCanaryEvidenceState(record.EvidenceState) {
		return fmt.Errorf("mission store hot-update canary evidence evidence_state %q is invalid", record.EvidenceState)
	}
	if record.Passed != (record.EvidenceState == HotUpdateCanaryEvidenceStatePassed) {
		return fmt.Errorf("mission store hot-update canary evidence passed must be true only when evidence_state is %q", HotUpdateCanaryEvidenceStatePassed)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update canary evidence reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update canary evidence created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update canary evidence created_by is required")
	}
	return nil
}

func StoreHotUpdateCanaryEvidenceRecord(root string, record HotUpdateCanaryEvidenceRecord) (HotUpdateCanaryEvidenceRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	record = NormalizeHotUpdateCanaryEvidenceRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateCanaryEvidenceRecord(record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	if err := validateHotUpdateCanaryEvidenceLinkage(root, record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}

	path := StoreHotUpdateCanaryEvidencePath(root, record.CanaryEvidenceID)
	existing, err := loadHotUpdateCanaryEvidenceRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence %q already exists", record.CanaryEvidenceID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	stored, err := LoadHotUpdateCanaryEvidenceRecord(root, record.CanaryEvidenceID)
	if err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateCanaryEvidenceRecord(root, canaryEvidenceID string) (HotUpdateCanaryEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	ref := NormalizeHotUpdateCanaryEvidenceRef(HotUpdateCanaryEvidenceRef{CanaryEvidenceID: canaryEvidenceID})
	if err := ValidateHotUpdateCanaryEvidenceRef(ref); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	record, err := loadHotUpdateCanaryEvidenceRecordFile(root, StoreHotUpdateCanaryEvidencePath(root, ref.CanaryEvidenceID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateCanaryEvidenceRecord{}, ErrHotUpdateCanaryEvidenceRecordNotFound
		}
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	return record, nil
}

func ListHotUpdateCanaryEvidenceRecords(root string) ([]HotUpdateCanaryEvidenceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreHotUpdateCanaryEvidenceDir(root), func(path string) (HotUpdateCanaryEvidenceRecord, error) {
		return loadHotUpdateCanaryEvidenceRecordFile(root, path)
	})
}

func CreateHotUpdateCanaryEvidenceFromRequirement(root, canaryRequirementID string, state HotUpdateCanaryEvidenceState, observedAt time.Time, createdBy string, createdAt time.Time, reason string) (HotUpdateCanaryEvidenceRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	requirementRef := NormalizeHotUpdateCanaryRequirementRef(HotUpdateCanaryRequirementRef{CanaryRequirementID: canaryRequirementID})
	if err := ValidateHotUpdateCanaryRequirementRef(requirementRef); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	state = HotUpdateCanaryEvidenceState(strings.TrimSpace(string(state)))
	if !isValidHotUpdateCanaryEvidenceState(state) {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence evidence_state %q is invalid", state)
	}
	observedAt = observedAt.UTC()
	if observedAt.IsZero() {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence observed_at is required")
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence created_at is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence reason is required")
	}

	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, requirementRef.CanaryRequirementID)
	if err != nil {
		return HotUpdateCanaryEvidenceRecord{}, false, err
	}
	if requirement.State != HotUpdateCanaryRequirementStateRequired {
		return HotUpdateCanaryEvidenceRecord{}, false, fmt.Errorf("mission store hot-update canary evidence requirement %q state must be %q", requirement.CanaryRequirementID, HotUpdateCanaryRequirementStateRequired)
	}

	record := NormalizeHotUpdateCanaryEvidenceRecord(HotUpdateCanaryEvidenceRecord{
		RecordVersion:       StoreRecordVersion,
		CanaryEvidenceID:    HotUpdateCanaryEvidenceIDFromRequirementObservedAt(requirement.CanaryRequirementID, observedAt),
		CanaryRequirementID: requirement.CanaryRequirementID,
		ResultID:            requirement.ResultID,
		RunID:               requirement.RunID,
		CandidateID:         requirement.CandidateID,
		EvalSuiteID:         requirement.EvalSuiteID,
		PromotionPolicyID:   requirement.PromotionPolicyID,
		BaselinePackID:      requirement.BaselinePackID,
		CandidatePackID:     requirement.CandidatePackID,
		EvidenceState:       state,
		Passed:              state == HotUpdateCanaryEvidenceStatePassed,
		Reason:              reason,
		ObservedAt:          observedAt,
		CreatedAt:           createdAt,
		CreatedBy:           createdBy,
	})
	return StoreHotUpdateCanaryEvidenceRecord(root, record)
}

func loadHotUpdateCanaryEvidenceRecordFile(root, path string) (HotUpdateCanaryEvidenceRecord, error) {
	var record HotUpdateCanaryEvidenceRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	record = NormalizeHotUpdateCanaryEvidenceRecord(record)
	if err := ValidateHotUpdateCanaryEvidenceRecord(record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	if err := validateHotUpdateCanaryEvidenceLinkage(root, record); err != nil {
		return HotUpdateCanaryEvidenceRecord{}, err
	}
	return record, nil
}

func validateHotUpdateCanaryEvidenceLinkage(root string, record HotUpdateCanaryEvidenceRecord) error {
	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, record.CanaryRequirementID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary evidence canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	if requirement.State != HotUpdateCanaryRequirementStateRequired {
		return fmt.Errorf("mission store hot-update canary evidence requirement %q state must be %q", requirement.CanaryRequirementID, HotUpdateCanaryRequirementStateRequired)
	}
	if record.ResultID != requirement.ResultID ||
		record.RunID != requirement.RunID ||
		record.CandidateID != requirement.CandidateID ||
		record.EvalSuiteID != requirement.EvalSuiteID ||
		record.PromotionPolicyID != requirement.PromotionPolicyID ||
		record.BaselinePackID != requirement.BaselinePackID ||
		record.CandidatePackID != requirement.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary evidence %q does not match hot-update canary requirement %q", record.CanaryEvidenceID, record.CanaryRequirementID)
	}

	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary evidence result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID ||
		result.CandidateID != record.CandidateID ||
		result.EvalSuiteID != record.EvalSuiteID ||
		result.PromotionPolicyID != record.PromotionPolicyID ||
		result.BaselinePackID != record.BaselinePackID ||
		result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary evidence %q does not match candidate result %q", record.CanaryEvidenceID, record.ResultID)
	}
	if _, err := LoadImprovementRunRecord(root, record.RunID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence run_id %q: %w", record.RunID, err)
	}
	if _, err := LoadImprovementCandidateRecord(root, record.CandidateID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence candidate_id %q: %w", record.CandidateID, err)
	}
	if _, err := LoadEvalSuiteRecord(root, record.EvalSuiteID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if _, err := LoadPromotionPolicyRecord(root, record.PromotionPolicyID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary evidence candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(status.State) {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update canary evidence", record.ResultID, status.State)
	}
	if status.RunID != record.RunID ||
		status.CandidateID != record.CandidateID ||
		status.EvalSuiteID != record.EvalSuiteID ||
		status.PromotionPolicyID != record.PromotionPolicyID ||
		status.BaselinePackID != record.BaselinePackID ||
		status.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary evidence %q does not match derived promotion eligibility status", record.CanaryEvidenceID)
	}
	return nil
}

func isValidHotUpdateCanaryEvidenceState(state HotUpdateCanaryEvidenceState) bool {
	switch state {
	case HotUpdateCanaryEvidenceStatePassed,
		HotUpdateCanaryEvidenceStateFailed,
		HotUpdateCanaryEvidenceStateBlocked,
		HotUpdateCanaryEvidenceStateExpired:
		return true
	default:
		return false
	}
}
