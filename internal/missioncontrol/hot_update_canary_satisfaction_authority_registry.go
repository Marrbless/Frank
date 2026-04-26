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
)

type HotUpdateCanarySatisfactionAuthorityState string

const (
	HotUpdateCanarySatisfactionAuthorityStateAuthorized           HotUpdateCanarySatisfactionAuthorityState = "authorized"
	HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval HotUpdateCanarySatisfactionAuthorityState = "waiting_owner_approval"
)

type HotUpdateCanarySatisfactionAuthorityRef struct {
	CanarySatisfactionAuthorityID string `json:"canary_satisfaction_authority_id"`
}

type HotUpdateCanarySatisfactionAuthorityRecord struct {
	RecordVersion                 int                                       `json:"record_version"`
	CanarySatisfactionAuthorityID string                                    `json:"canary_satisfaction_authority_id"`
	CanaryRequirementID           string                                    `json:"canary_requirement_id"`
	SelectedCanaryEvidenceID      string                                    `json:"selected_canary_evidence_id"`
	ResultID                      string                                    `json:"result_id"`
	RunID                         string                                    `json:"run_id"`
	CandidateID                   string                                    `json:"candidate_id"`
	EvalSuiteID                   string                                    `json:"eval_suite_id"`
	PromotionPolicyID             string                                    `json:"promotion_policy_id"`
	BaselinePackID                string                                    `json:"baseline_pack_id"`
	CandidatePackID               string                                    `json:"candidate_pack_id"`
	EligibilityState              string                                    `json:"eligibility_state"`
	OwnerApprovalRequired         bool                                      `json:"owner_approval_required"`
	SatisfactionState             HotUpdateCanarySatisfactionState          `json:"satisfaction_state"`
	State                         HotUpdateCanarySatisfactionAuthorityState `json:"state"`
	Reason                        string                                    `json:"reason"`
	CreatedAt                     time.Time                                 `json:"created_at"`
	CreatedBy                     string                                    `json:"created_by"`
}

var ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound = errors.New("mission store hot-update canary satisfaction authority record not found")

func StoreHotUpdateCanarySatisfactionAuthoritiesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_canary_satisfaction_authorities")
}

func StoreHotUpdateCanarySatisfactionAuthorityPath(root, canarySatisfactionAuthorityID string) string {
	return filepath.Join(StoreHotUpdateCanarySatisfactionAuthoritiesDir(root), strings.TrimSpace(canarySatisfactionAuthorityID)+".json")
}

func HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(canaryRequirementID, selectedCanaryEvidenceID string) string {
	return "hot-update-canary-satisfaction-authority-" + strings.TrimSpace(canaryRequirementID) + "-" + strings.TrimSpace(selectedCanaryEvidenceID)
}

func NormalizeHotUpdateCanarySatisfactionAuthorityRef(ref HotUpdateCanarySatisfactionAuthorityRef) HotUpdateCanarySatisfactionAuthorityRef {
	ref.CanarySatisfactionAuthorityID = strings.TrimSpace(ref.CanarySatisfactionAuthorityID)
	return ref
}

func NormalizeHotUpdateCanarySatisfactionAuthorityRecord(record HotUpdateCanarySatisfactionAuthorityRecord) HotUpdateCanarySatisfactionAuthorityRecord {
	record.CanarySatisfactionAuthorityID = strings.TrimSpace(record.CanarySatisfactionAuthorityID)
	record.CanaryRequirementID = strings.TrimSpace(record.CanaryRequirementID)
	record.SelectedCanaryEvidenceID = strings.TrimSpace(record.SelectedCanaryEvidenceID)
	record.ResultID = strings.TrimSpace(record.ResultID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.CandidateID = strings.TrimSpace(record.CandidateID)
	record.EvalSuiteID = strings.TrimSpace(record.EvalSuiteID)
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.BaselinePackID = strings.TrimSpace(record.BaselinePackID)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.EligibilityState = strings.TrimSpace(record.EligibilityState)
	record.SatisfactionState = HotUpdateCanarySatisfactionState(strings.TrimSpace(string(record.SatisfactionState)))
	record.State = HotUpdateCanarySatisfactionAuthorityState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateHotUpdateCanarySatisfactionAuthorityRef(ref HotUpdateCanarySatisfactionAuthorityRef) error {
	return validateHotUpdateCanaryRequirementIdentifierField("hot-update canary satisfaction authority ref", "canary_satisfaction_authority_id", ref.CanarySatisfactionAuthorityID)
}

func HotUpdateCanarySatisfactionAuthorityRequirementRef(record HotUpdateCanarySatisfactionAuthorityRecord) HotUpdateCanaryRequirementRef {
	return HotUpdateCanaryRequirementRef{CanaryRequirementID: strings.TrimSpace(record.CanaryRequirementID)}
}

func HotUpdateCanarySatisfactionAuthorityEvidenceRef(record HotUpdateCanarySatisfactionAuthorityRecord) HotUpdateCanaryEvidenceRef {
	return HotUpdateCanaryEvidenceRef{CanaryEvidenceID: strings.TrimSpace(record.SelectedCanaryEvidenceID)}
}

func HotUpdateCanarySatisfactionAuthorityCandidateResultRef(record HotUpdateCanarySatisfactionAuthorityRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func HotUpdateCanarySatisfactionAuthorityImprovementRunRef(record HotUpdateCanarySatisfactionAuthorityRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func HotUpdateCanarySatisfactionAuthorityImprovementCandidateRef(record HotUpdateCanarySatisfactionAuthorityRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func HotUpdateCanarySatisfactionAuthorityEvalSuiteRef(record HotUpdateCanarySatisfactionAuthorityRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func HotUpdateCanarySatisfactionAuthorityPromotionPolicyRef(record HotUpdateCanarySatisfactionAuthorityRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func HotUpdateCanarySatisfactionAuthorityBaselinePackRef(record HotUpdateCanarySatisfactionAuthorityRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func HotUpdateCanarySatisfactionAuthorityCandidatePackRef(record HotUpdateCanarySatisfactionAuthorityRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateHotUpdateCanarySatisfactionAuthorityRecord(record HotUpdateCanarySatisfactionAuthorityRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update canary satisfaction authority record_version must be positive")
	}
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: record.CanarySatisfactionAuthorityID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateCanaryRequirementRef(HotUpdateCanarySatisfactionAuthorityRequirementRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	if err := ValidateHotUpdateCanaryEvidenceRef(HotUpdateCanarySatisfactionAuthorityEvidenceRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id %q: %w", record.SelectedCanaryEvidenceID, err)
	}
	if err := ValidateCandidateResultRef(HotUpdateCanarySatisfactionAuthorityCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(HotUpdateCanarySatisfactionAuthorityImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(HotUpdateCanarySatisfactionAuthorityImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(HotUpdateCanarySatisfactionAuthorityEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(HotUpdateCanarySatisfactionAuthorityPromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanarySatisfactionAuthorityBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateCanarySatisfactionAuthorityCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.CanarySatisfactionAuthorityID != HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(record.CanaryRequirementID, record.SelectedCanaryEvidenceID) {
		return fmt.Errorf("mission store hot-update canary satisfaction authority canary_satisfaction_authority_id %q does not match deterministic canary_satisfaction_authority_id %q", record.CanarySatisfactionAuthorityID, HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(record.CanaryRequirementID, record.SelectedCanaryEvidenceID))
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(record.EligibilityState) {
		return fmt.Errorf("mission store hot-update canary satisfaction authority eligibility_state %q is invalid", record.EligibilityState)
	}
	if !isValidHotUpdateCanarySatisfactionAuthoritySatisfactionState(record.SatisfactionState) {
		return fmt.Errorf("mission store hot-update canary satisfaction authority satisfaction_state %q is invalid", record.SatisfactionState)
	}
	if !isValidHotUpdateCanarySatisfactionAuthorityState(record.State) {
		return fmt.Errorf("mission store hot-update canary satisfaction authority state %q is invalid", record.State)
	}
	switch record.SatisfactionState {
	case HotUpdateCanarySatisfactionStateSatisfied:
		if record.OwnerApprovalRequired {
			return fmt.Errorf("mission store hot-update canary satisfaction authority owner_approval_required must be false when satisfaction_state is %q", HotUpdateCanarySatisfactionStateSatisfied)
		}
		if record.State != HotUpdateCanarySatisfactionAuthorityStateAuthorized {
			return fmt.Errorf("mission store hot-update canary satisfaction authority state must be %q when satisfaction_state is %q", HotUpdateCanarySatisfactionAuthorityStateAuthorized, HotUpdateCanarySatisfactionStateSatisfied)
		}
	case HotUpdateCanarySatisfactionStateWaitingOwnerApproval:
		if !record.OwnerApprovalRequired {
			return fmt.Errorf("mission store hot-update canary satisfaction authority owner_approval_required must be true when satisfaction_state is %q", HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
		}
		if record.State != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval {
			return fmt.Errorf("mission store hot-update canary satisfaction authority state must be %q when satisfaction_state is %q", HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval, HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
		}
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update canary satisfaction authority reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update canary satisfaction authority created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update canary satisfaction authority created_by is required")
	}
	return nil
}

func StoreHotUpdateCanarySatisfactionAuthorityRecord(root string, record HotUpdateCanarySatisfactionAuthorityRecord) (HotUpdateCanarySatisfactionAuthorityRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	record = NormalizeHotUpdateCanarySatisfactionAuthorityRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRecord(record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if err := validateHotUpdateCanarySatisfactionAuthorityLinkage(root, record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	records, err := ListHotUpdateCanarySatisfactionAuthorityRecords(root)
	if err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	for _, existing := range records {
		if existing.CanaryRequirementID == record.CanaryRequirementID &&
			existing.SelectedCanaryEvidenceID == record.SelectedCanaryEvidenceID &&
			existing.CanarySatisfactionAuthorityID != record.CanarySatisfactionAuthorityID {
			return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf(
				"mission store hot-update canary satisfaction authority for canary_requirement_id %q selected_canary_evidence_id %q already exists as %q",
				record.CanaryRequirementID,
				record.SelectedCanaryEvidenceID,
				existing.CanarySatisfactionAuthorityID,
			)
		}
	}

	path := StoreHotUpdateCanarySatisfactionAuthorityPath(root, record.CanarySatisfactionAuthorityID)
	existing, err := loadHotUpdateCanarySatisfactionAuthorityRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority %q already exists", record.CanarySatisfactionAuthorityID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}

	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	stored, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, record.CanarySatisfactionAuthorityID)
	if err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateCanarySatisfactionAuthorityRecord(root, canarySatisfactionAuthorityID string) (HotUpdateCanarySatisfactionAuthorityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	ref := NormalizeHotUpdateCanarySatisfactionAuthorityRef(HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: canarySatisfactionAuthorityID})
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(ref); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	record, err := loadHotUpdateCanarySatisfactionAuthorityRecordFile(root, StoreHotUpdateCanarySatisfactionAuthorityPath(root, ref.CanarySatisfactionAuthorityID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateCanarySatisfactionAuthorityRecord{}, ErrHotUpdateCanarySatisfactionAuthorityRecordNotFound
		}
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	return record, nil
}

func ListHotUpdateCanarySatisfactionAuthorityRecords(root string) ([]HotUpdateCanarySatisfactionAuthorityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreHotUpdateCanarySatisfactionAuthoritiesDir(root))
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

	records := make([]HotUpdateCanarySatisfactionAuthorityRecord, 0, len(names))
	for _, name := range names {
		record, err := loadHotUpdateCanarySatisfactionAuthorityRecordFile(root, filepath.Join(StoreHotUpdateCanarySatisfactionAuthoritiesDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func CreateHotUpdateCanarySatisfactionAuthorityFromRequirement(root, canaryRequirementID, createdBy string, createdAt time.Time) (HotUpdateCanarySatisfactionAuthorityRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	requirementRef := NormalizeHotUpdateCanaryRequirementRef(HotUpdateCanaryRequirementRef{CanaryRequirementID: canaryRequirementID})
	if err := ValidateHotUpdateCanaryRequirementRef(requirementRef); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority created_at is required")
	}

	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, requirementRef.CanaryRequirementID)
	if err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	assessment, err := AssessHotUpdateCanarySatisfaction(root, requirement.CanaryRequirementID)
	if err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if assessment.State != "configured" {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority requires configured canary satisfaction assessment, found state %q: %s", assessment.State, assessment.Error)
	}
	if !isValidHotUpdateCanarySatisfactionAuthoritySatisfactionState(assessment.SatisfactionState) {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority requires satisfaction_state %q or %q, found %q", HotUpdateCanarySatisfactionStateSatisfied, HotUpdateCanarySatisfactionStateWaitingOwnerApproval, assessment.SatisfactionState)
	}
	if assessment.SelectedCanaryEvidenceID == "" {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id is required")
	}
	evidence, err := LoadHotUpdateCanaryEvidenceRecord(root, assessment.SelectedCanaryEvidenceID)
	if err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	if evidence.EvidenceState != HotUpdateCanaryEvidenceStatePassed || !evidence.Passed {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id %q must be passed", evidence.CanaryEvidenceID)
	}

	state := HotUpdateCanarySatisfactionAuthorityStateAuthorized
	if assessment.SatisfactionState == HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
		state = HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval
	}
	record := NormalizeHotUpdateCanarySatisfactionAuthorityRecord(HotUpdateCanarySatisfactionAuthorityRecord{
		RecordVersion:                 StoreRecordVersion,
		CanarySatisfactionAuthorityID: HotUpdateCanarySatisfactionAuthorityIDFromRequirementEvidence(assessment.CanaryRequirementID, assessment.SelectedCanaryEvidenceID),
		CanaryRequirementID:           assessment.CanaryRequirementID,
		SelectedCanaryEvidenceID:      assessment.SelectedCanaryEvidenceID,
		ResultID:                      assessment.ResultID,
		RunID:                         assessment.RunID,
		CandidateID:                   assessment.CandidateID,
		EvalSuiteID:                   assessment.EvalSuiteID,
		PromotionPolicyID:             assessment.PromotionPolicyID,
		BaselinePackID:                assessment.BaselinePackID,
		CandidatePackID:               assessment.CandidatePackID,
		EligibilityState:              assessment.EligibilityState,
		OwnerApprovalRequired:         assessment.OwnerApprovalRequired,
		SatisfactionState:             assessment.SatisfactionState,
		State:                         state,
		Reason:                        "hot-update canary satisfaction authority recorded from passed canary evidence",
		CreatedAt:                     createdAt,
		CreatedBy:                     createdBy,
	})
	if err := validateHotUpdateCanarySatisfactionAuthorityMatchesRequirementEvidenceAssessment(record, requirement, evidence, assessment); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, false, err
	}
	return StoreHotUpdateCanarySatisfactionAuthorityRecord(root, record)
}

func loadHotUpdateCanarySatisfactionAuthorityRecordFile(root, path string) (HotUpdateCanarySatisfactionAuthorityRecord, error) {
	var record HotUpdateCanarySatisfactionAuthorityRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	record = NormalizeHotUpdateCanarySatisfactionAuthorityRecord(record)
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRecord(record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	if err := validateHotUpdateCanarySatisfactionAuthorityLinkage(root, record); err != nil {
		return HotUpdateCanarySatisfactionAuthorityRecord{}, err
	}
	return record, nil
}

func validateHotUpdateCanarySatisfactionAuthorityLinkage(root string, record HotUpdateCanarySatisfactionAuthorityRecord) error {
	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, record.CanaryRequirementID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	evidence, err := LoadHotUpdateCanaryEvidenceRecord(root, record.SelectedCanaryEvidenceID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id %q: %w", record.SelectedCanaryEvidenceID, err)
	}
	if evidence.EvidenceState != HotUpdateCanaryEvidenceStatePassed || !evidence.Passed {
		return fmt.Errorf("mission store hot-update canary satisfaction authority selected_canary_evidence_id %q must be passed", evidence.CanaryEvidenceID)
	}
	if err := validateHotUpdateCanarySatisfactionAuthorityRefs(record, requirement, evidence); err != nil {
		return err
	}

	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID ||
		result.CandidateID != record.CandidateID ||
		result.EvalSuiteID != record.EvalSuiteID ||
		result.PromotionPolicyID != record.PromotionPolicyID ||
		result.BaselinePackID != record.BaselinePackID ||
		result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match candidate result %q", record.CanarySatisfactionAuthorityID, record.ResultID)
	}
	if _, err := LoadImprovementRunRecord(root, record.RunID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority run_id %q: %w", record.RunID, err)
	}
	if _, err := LoadImprovementCandidateRecord(root, record.CandidateID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_id %q: %w", record.CandidateID, err)
	}
	if _, err := LoadEvalSuiteRecord(root, record.EvalSuiteID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if _, err := LoadPromotionPolicyRecord(root, record.PromotionPolicyID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if !isValidHotUpdateCanaryRequirementEligibilityState(status.State) {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update canary satisfaction authority", record.ResultID, status.State)
	}
	if status.RunID != record.RunID ||
		status.CandidateID != record.CandidateID ||
		status.EvalSuiteID != record.EvalSuiteID ||
		status.PromotionPolicyID != record.PromotionPolicyID ||
		status.BaselinePackID != record.BaselinePackID ||
		status.CandidatePackID != record.CandidatePackID ||
		status.State != record.EligibilityState ||
		status.OwnerApprovalRequired != record.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match derived promotion eligibility status", record.CanarySatisfactionAuthorityID)
	}
	return nil
}

func validateHotUpdateCanarySatisfactionAuthorityMatchesRequirementEvidenceAssessment(record HotUpdateCanarySatisfactionAuthorityRecord, requirement HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord, assessment HotUpdateCanarySatisfactionAssessment) error {
	if err := validateHotUpdateCanarySatisfactionAuthorityRefs(record, requirement, evidence); err != nil {
		return err
	}
	if record.CanaryRequirementID != assessment.CanaryRequirementID ||
		record.SelectedCanaryEvidenceID != assessment.SelectedCanaryEvidenceID ||
		record.ResultID != assessment.ResultID ||
		record.RunID != assessment.RunID ||
		record.CandidateID != assessment.CandidateID ||
		record.EvalSuiteID != assessment.EvalSuiteID ||
		record.PromotionPolicyID != assessment.PromotionPolicyID ||
		record.BaselinePackID != assessment.BaselinePackID ||
		record.CandidatePackID != assessment.CandidatePackID ||
		record.EligibilityState != assessment.EligibilityState ||
		record.OwnerApprovalRequired != assessment.OwnerApprovalRequired ||
		record.SatisfactionState != assessment.SatisfactionState {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match canary satisfaction assessment", record.CanarySatisfactionAuthorityID)
	}
	if assessment.EvidenceState != HotUpdateCanaryEvidenceStatePassed || !assessment.Passed {
		return fmt.Errorf("mission store hot-update canary satisfaction authority assessment selected evidence must be passed")
	}
	return nil
}

func validateHotUpdateCanarySatisfactionAuthorityRefs(record HotUpdateCanarySatisfactionAuthorityRecord, requirement HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord) error {
	if requirement.State != HotUpdateCanaryRequirementStateRequired {
		return fmt.Errorf("mission store hot-update canary satisfaction authority requirement %q state must be %q", requirement.CanaryRequirementID, HotUpdateCanaryRequirementStateRequired)
	}
	if record.CanaryRequirementID != requirement.CanaryRequirementID ||
		record.ResultID != requirement.ResultID ||
		record.RunID != requirement.RunID ||
		record.CandidateID != requirement.CandidateID ||
		record.EvalSuiteID != requirement.EvalSuiteID ||
		record.PromotionPolicyID != requirement.PromotionPolicyID ||
		record.BaselinePackID != requirement.BaselinePackID ||
		record.CandidatePackID != requirement.CandidatePackID ||
		record.EligibilityState != requirement.EligibilityState ||
		record.OwnerApprovalRequired != requirement.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match hot-update canary requirement %q", record.CanarySatisfactionAuthorityID, record.CanaryRequirementID)
	}
	if evidence.CanaryRequirementID != requirement.CanaryRequirementID ||
		evidence.ResultID != record.ResultID ||
		evidence.RunID != record.RunID ||
		evidence.CandidateID != record.CandidateID ||
		evidence.EvalSuiteID != record.EvalSuiteID ||
		evidence.PromotionPolicyID != record.PromotionPolicyID ||
		evidence.BaselinePackID != record.BaselinePackID ||
		evidence.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match selected canary evidence %q", record.CanarySatisfactionAuthorityID, record.SelectedCanaryEvidenceID)
	}
	return nil
}

func isValidHotUpdateCanarySatisfactionAuthoritySatisfactionState(state HotUpdateCanarySatisfactionState) bool {
	switch state {
	case HotUpdateCanarySatisfactionStateSatisfied,
		HotUpdateCanarySatisfactionStateWaitingOwnerApproval:
		return true
	default:
		return false
	}
}

func isValidHotUpdateCanarySatisfactionAuthorityState(state HotUpdateCanarySatisfactionAuthorityState) bool {
	switch state {
	case HotUpdateCanarySatisfactionAuthorityStateAuthorized,
		HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval:
		return true
	default:
		return false
	}
}
