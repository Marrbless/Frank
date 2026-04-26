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

type HotUpdateOwnerApprovalRequestState string

const (
	HotUpdateOwnerApprovalRequestStateRequested HotUpdateOwnerApprovalRequestState = "requested"
)

type HotUpdateOwnerApprovalRequestRef struct {
	OwnerApprovalRequestID string `json:"owner_approval_request_id"`
}

type HotUpdateOwnerApprovalRequestRecord struct {
	RecordVersion                 int                                       `json:"record_version"`
	OwnerApprovalRequestID        string                                    `json:"owner_approval_request_id"`
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
	AuthorityState                HotUpdateCanarySatisfactionAuthorityState `json:"authority_state"`
	SatisfactionState             HotUpdateCanarySatisfactionState          `json:"satisfaction_state"`
	OwnerApprovalRequired         bool                                      `json:"owner_approval_required"`
	State                         HotUpdateOwnerApprovalRequestState        `json:"state"`
	Reason                        string                                    `json:"reason"`
	CreatedAt                     time.Time                                 `json:"created_at"`
	CreatedBy                     string                                    `json:"created_by"`
}

var ErrHotUpdateOwnerApprovalRequestRecordNotFound = errors.New("mission store hot-update owner approval request record not found")

func StoreHotUpdateOwnerApprovalRequestsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_owner_approval_requests")
}

func StoreHotUpdateOwnerApprovalRequestPath(root, ownerApprovalRequestID string) string {
	return filepath.Join(StoreHotUpdateOwnerApprovalRequestsDir(root), strings.TrimSpace(ownerApprovalRequestID)+".json")
}

func HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string {
	return "hot-update-owner-approval-request-" + strings.TrimSpace(canarySatisfactionAuthorityID)
}

func NormalizeHotUpdateOwnerApprovalRequestRef(ref HotUpdateOwnerApprovalRequestRef) HotUpdateOwnerApprovalRequestRef {
	ref.OwnerApprovalRequestID = strings.TrimSpace(ref.OwnerApprovalRequestID)
	return ref
}

func NormalizeHotUpdateOwnerApprovalRequestRecord(record HotUpdateOwnerApprovalRequestRecord) HotUpdateOwnerApprovalRequestRecord {
	record.OwnerApprovalRequestID = strings.TrimSpace(record.OwnerApprovalRequestID)
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
	record.AuthorityState = HotUpdateCanarySatisfactionAuthorityState(strings.TrimSpace(string(record.AuthorityState)))
	record.SatisfactionState = HotUpdateCanarySatisfactionState(strings.TrimSpace(string(record.SatisfactionState)))
	record.State = HotUpdateOwnerApprovalRequestState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateHotUpdateOwnerApprovalRequestRef(ref HotUpdateOwnerApprovalRequestRef) error {
	return validateHotUpdateIdentifierField("hot-update owner approval request ref", "owner_approval_request_id", ref.OwnerApprovalRequestID)
}

func HotUpdateOwnerApprovalRequestCanarySatisfactionAuthorityRef(record HotUpdateOwnerApprovalRequestRecord) HotUpdateCanarySatisfactionAuthorityRef {
	return HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: strings.TrimSpace(record.CanarySatisfactionAuthorityID)}
}

func HotUpdateOwnerApprovalRequestRequirementRef(record HotUpdateOwnerApprovalRequestRecord) HotUpdateCanaryRequirementRef {
	return HotUpdateCanaryRequirementRef{CanaryRequirementID: strings.TrimSpace(record.CanaryRequirementID)}
}

func HotUpdateOwnerApprovalRequestEvidenceRef(record HotUpdateOwnerApprovalRequestRecord) HotUpdateCanaryEvidenceRef {
	return HotUpdateCanaryEvidenceRef{CanaryEvidenceID: strings.TrimSpace(record.SelectedCanaryEvidenceID)}
}

func HotUpdateOwnerApprovalRequestCandidateResultRef(record HotUpdateOwnerApprovalRequestRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func HotUpdateOwnerApprovalRequestImprovementRunRef(record HotUpdateOwnerApprovalRequestRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func HotUpdateOwnerApprovalRequestImprovementCandidateRef(record HotUpdateOwnerApprovalRequestRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func HotUpdateOwnerApprovalRequestEvalSuiteRef(record HotUpdateOwnerApprovalRequestRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func HotUpdateOwnerApprovalRequestPromotionPolicyRef(record HotUpdateOwnerApprovalRequestRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func HotUpdateOwnerApprovalRequestBaselinePackRef(record HotUpdateOwnerApprovalRequestRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func HotUpdateOwnerApprovalRequestCandidatePackRef(record HotUpdateOwnerApprovalRequestRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateHotUpdateOwnerApprovalRequestRecord(record HotUpdateOwnerApprovalRequestRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update owner approval request record_version must be positive")
	}
	if err := ValidateHotUpdateOwnerApprovalRequestRef(HotUpdateOwnerApprovalRequestRef{OwnerApprovalRequestID: record.OwnerApprovalRequestID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(HotUpdateOwnerApprovalRequestCanarySatisfactionAuthorityRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request canary_satisfaction_authority_id %q: %w", record.CanarySatisfactionAuthorityID, err)
	}
	if err := ValidateHotUpdateCanaryRequirementRef(HotUpdateOwnerApprovalRequestRequirementRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	if err := ValidateHotUpdateCanaryEvidenceRef(HotUpdateOwnerApprovalRequestEvidenceRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request selected_canary_evidence_id %q: %w", record.SelectedCanaryEvidenceID, err)
	}
	if err := ValidateCandidateResultRef(HotUpdateOwnerApprovalRequestCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(HotUpdateOwnerApprovalRequestImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(HotUpdateOwnerApprovalRequestImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(HotUpdateOwnerApprovalRequestEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(HotUpdateOwnerApprovalRequestPromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateOwnerApprovalRequestBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateOwnerApprovalRequestCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.OwnerApprovalRequestID != HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(record.CanarySatisfactionAuthorityID) {
		return fmt.Errorf("mission store hot-update owner approval request owner_approval_request_id %q does not match deterministic owner_approval_request_id %q", record.OwnerApprovalRequestID, HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(record.CanarySatisfactionAuthorityID))
	}
	if !isValidHotUpdateCanarySatisfactionAuthorityState(record.AuthorityState) {
		return fmt.Errorf("mission store hot-update owner approval request authority_state %q is invalid", record.AuthorityState)
	}
	if record.AuthorityState != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval request authority_state must be %q", HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval)
	}
	if !isValidHotUpdateCanarySatisfactionAuthoritySatisfactionState(record.SatisfactionState) {
		return fmt.Errorf("mission store hot-update owner approval request satisfaction_state %q is invalid", record.SatisfactionState)
	}
	if record.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval request satisfaction_state must be %q", HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
	}
	if !record.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request owner_approval_required must be true")
	}
	if !isValidHotUpdateOwnerApprovalRequestState(record.State) {
		return fmt.Errorf("mission store hot-update owner approval request state %q is invalid", record.State)
	}
	if record.State != HotUpdateOwnerApprovalRequestStateRequested {
		return fmt.Errorf("mission store hot-update owner approval request state must be %q", HotUpdateOwnerApprovalRequestStateRequested)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update owner approval request reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update owner approval request created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store hot-update owner approval request created_by is required")
	}
	return nil
}

func StoreHotUpdateOwnerApprovalRequestRecord(root string, record HotUpdateOwnerApprovalRequestRecord) (HotUpdateOwnerApprovalRequestRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	record = NormalizeHotUpdateOwnerApprovalRequestRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateOwnerApprovalRequestRecord(record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	if err := validateHotUpdateOwnerApprovalRequestLinkage(root, record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	records, err := ListHotUpdateOwnerApprovalRequestRecords(root)
	if err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	for _, existing := range records {
		if existing.CanarySatisfactionAuthorityID == record.CanarySatisfactionAuthorityID &&
			existing.OwnerApprovalRequestID != record.OwnerApprovalRequestID {
			return HotUpdateOwnerApprovalRequestRecord{}, false, fmt.Errorf(
				"mission store hot-update owner approval request for canary_satisfaction_authority_id %q already exists as %q",
				record.CanarySatisfactionAuthorityID,
				existing.OwnerApprovalRequestID,
			)
		}
	}

	path := StoreHotUpdateOwnerApprovalRequestPath(root, record.OwnerApprovalRequestID)
	existing, err := loadHotUpdateOwnerApprovalRequestRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateOwnerApprovalRequestRecord{}, false, fmt.Errorf("mission store hot-update owner approval request %q already exists", record.OwnerApprovalRequestID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}

	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	stored, err := LoadHotUpdateOwnerApprovalRequestRecord(root, record.OwnerApprovalRequestID)
	if err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateOwnerApprovalRequestRecord(root, ownerApprovalRequestID string) (HotUpdateOwnerApprovalRequestRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	ref := NormalizeHotUpdateOwnerApprovalRequestRef(HotUpdateOwnerApprovalRequestRef{OwnerApprovalRequestID: ownerApprovalRequestID})
	if err := ValidateHotUpdateOwnerApprovalRequestRef(ref); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	record, err := loadHotUpdateOwnerApprovalRequestRecordFile(root, StoreHotUpdateOwnerApprovalRequestPath(root, ref.OwnerApprovalRequestID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateOwnerApprovalRequestRecord{}, ErrHotUpdateOwnerApprovalRequestRecordNotFound
		}
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	return record, nil
}

func ListHotUpdateOwnerApprovalRequestRecords(root string) ([]HotUpdateOwnerApprovalRequestRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreHotUpdateOwnerApprovalRequestsDir(root))
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

	records := make([]HotUpdateOwnerApprovalRequestRecord, 0, len(names))
	for _, name := range names {
		record, err := loadHotUpdateOwnerApprovalRequestRecordFile(root, filepath.Join(StoreHotUpdateOwnerApprovalRequestsDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func CreateHotUpdateOwnerApprovalRequestFromCanarySatisfactionAuthority(root, canarySatisfactionAuthorityID, createdBy string, createdAt time.Time) (HotUpdateOwnerApprovalRequestRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	authorityRef := NormalizeHotUpdateCanarySatisfactionAuthorityRef(HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: canarySatisfactionAuthorityID})
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(authorityRef); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateOwnerApprovalRequestRecord{}, false, fmt.Errorf("mission store hot-update owner approval request created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateOwnerApprovalRequestRecord{}, false, fmt.Errorf("mission store hot-update owner approval request created_at is required")
	}

	authority, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, authorityRef.CanarySatisfactionAuthorityID)
	if err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	if err := requireHotUpdateOwnerApprovalRequestAuthority(authority); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, false, err
	}
	if authority.SelectedCanaryEvidenceID == "" {
		return HotUpdateOwnerApprovalRequestRecord{}, false, fmt.Errorf("mission store hot-update owner approval request selected_canary_evidence_id is required")
	}

	record := NormalizeHotUpdateOwnerApprovalRequestRecord(HotUpdateOwnerApprovalRequestRecord{
		RecordVersion:                 StoreRecordVersion,
		OwnerApprovalRequestID:        HotUpdateOwnerApprovalRequestIDFromCanarySatisfactionAuthority(authority.CanarySatisfactionAuthorityID),
		CanarySatisfactionAuthorityID: authority.CanarySatisfactionAuthorityID,
		CanaryRequirementID:           authority.CanaryRequirementID,
		SelectedCanaryEvidenceID:      authority.SelectedCanaryEvidenceID,
		ResultID:                      authority.ResultID,
		RunID:                         authority.RunID,
		CandidateID:                   authority.CandidateID,
		EvalSuiteID:                   authority.EvalSuiteID,
		PromotionPolicyID:             authority.PromotionPolicyID,
		BaselinePackID:                authority.BaselinePackID,
		CandidatePackID:               authority.CandidatePackID,
		AuthorityState:                authority.State,
		SatisfactionState:             authority.SatisfactionState,
		OwnerApprovalRequired:         authority.OwnerApprovalRequired,
		State:                         HotUpdateOwnerApprovalRequestStateRequested,
		Reason:                        "hot-update owner approval requested after canary satisfaction",
		CreatedAt:                     createdAt,
		CreatedBy:                     createdBy,
	})
	return StoreHotUpdateOwnerApprovalRequestRecord(root, record)
}

func loadHotUpdateOwnerApprovalRequestRecordFile(root, path string) (HotUpdateOwnerApprovalRequestRecord, error) {
	var record HotUpdateOwnerApprovalRequestRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	record = NormalizeHotUpdateOwnerApprovalRequestRecord(record)
	if err := ValidateHotUpdateOwnerApprovalRequestRecord(record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	if err := validateHotUpdateOwnerApprovalRequestLinkage(root, record); err != nil {
		return HotUpdateOwnerApprovalRequestRecord{}, err
	}
	return record, nil
}

func validateHotUpdateOwnerApprovalRequestLinkage(root string, record HotUpdateOwnerApprovalRequestRecord) error {
	authority, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, record.CanarySatisfactionAuthorityID)
	if err != nil {
		return fmt.Errorf("mission store hot-update owner approval request canary_satisfaction_authority_id %q: %w", record.CanarySatisfactionAuthorityID, err)
	}
	if err := requireHotUpdateOwnerApprovalRequestAuthority(authority); err != nil {
		return err
	}
	if err := validateHotUpdateOwnerApprovalRequestMatchesAuthority(record, authority); err != nil {
		return err
	}

	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, record.CanaryRequirementID)
	if err != nil {
		return fmt.Errorf("mission store hot-update owner approval request canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	evidence, err := LoadHotUpdateCanaryEvidenceRecord(root, record.SelectedCanaryEvidenceID)
	if err != nil {
		return fmt.Errorf("mission store hot-update owner approval request selected_canary_evidence_id %q: %w", record.SelectedCanaryEvidenceID, err)
	}
	if evidence.EvidenceState != HotUpdateCanaryEvidenceStatePassed || !evidence.Passed {
		return fmt.Errorf("mission store hot-update owner approval request selected_canary_evidence_id %q must be passed", evidence.CanaryEvidenceID)
	}
	if err := validateHotUpdateOwnerApprovalRequestRefs(record, authority, requirement, evidence); err != nil {
		return err
	}

	result, err := LoadCandidateResultRecord(root, record.ResultID)
	if err != nil {
		return fmt.Errorf("mission store hot-update owner approval request result_id %q: %w", record.ResultID, err)
	}
	if result.RunID != record.RunID ||
		result.CandidateID != record.CandidateID ||
		result.EvalSuiteID != record.EvalSuiteID ||
		result.PromotionPolicyID != record.PromotionPolicyID ||
		result.BaselinePackID != record.BaselinePackID ||
		result.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match candidate result %q", record.OwnerApprovalRequestID, record.ResultID)
	}
	if _, err := LoadImprovementRunRecord(root, record.RunID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request run_id %q: %w", record.RunID, err)
	}
	if _, err := LoadImprovementCandidateRecord(root, record.CandidateID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request candidate_id %q: %w", record.CandidateID, err)
	}
	if _, err := LoadEvalSuiteRecord(root, record.EvalSuiteID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if _, err := LoadPromotionPolicyRecord(root, record.PromotionPolicyID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.BaselinePackID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update owner approval request candidate_pack_id %q: %w", record.CandidatePackID, err)
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		return err
	}
	if status.State != CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update owner approval request", record.ResultID, status.State)
	}
	if status.RunID != record.RunID ||
		status.CandidateID != record.CandidateID ||
		status.EvalSuiteID != record.EvalSuiteID ||
		status.PromotionPolicyID != record.PromotionPolicyID ||
		status.BaselinePackID != record.BaselinePackID ||
		status.CandidatePackID != record.CandidatePackID ||
		!status.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match derived promotion eligibility status", record.OwnerApprovalRequestID)
	}

	assessment, err := AssessHotUpdateCanarySatisfaction(root, record.CanaryRequirementID)
	if err != nil {
		return err
	}
	if assessment.State != "configured" {
		return fmt.Errorf("mission store hot-update owner approval request requires configured canary satisfaction assessment, found state %q: %s", assessment.State, assessment.Error)
	}
	if assessment.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval request requires canary satisfaction_state %q, found %q", HotUpdateCanarySatisfactionStateWaitingOwnerApproval, assessment.SatisfactionState)
	}
	if assessment.CanaryRequirementID != record.CanaryRequirementID ||
		assessment.SelectedCanaryEvidenceID != record.SelectedCanaryEvidenceID ||
		assessment.ResultID != record.ResultID ||
		assessment.RunID != record.RunID ||
		assessment.CandidateID != record.CandidateID ||
		assessment.EvalSuiteID != record.EvalSuiteID ||
		assessment.PromotionPolicyID != record.PromotionPolicyID ||
		assessment.BaselinePackID != record.BaselinePackID ||
		assessment.CandidatePackID != record.CandidatePackID ||
		assessment.OwnerApprovalRequired != record.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match canary satisfaction assessment", record.OwnerApprovalRequestID)
	}
	if assessment.EvidenceState != HotUpdateCanaryEvidenceStatePassed || !assessment.Passed {
		return fmt.Errorf("mission store hot-update owner approval request assessment selected evidence must be passed")
	}
	return nil
}

func requireHotUpdateOwnerApprovalRequestAuthority(authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	if authority.State != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval request requires canary satisfaction authority state %q, found %q", HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval, authority.State)
	}
	if !authority.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request owner_approval_required must be true")
	}
	if authority.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval request requires satisfaction_state %q, found %q", HotUpdateCanarySatisfactionStateWaitingOwnerApproval, authority.SatisfactionState)
	}
	return nil
}

func validateHotUpdateOwnerApprovalRequestMatchesAuthority(record HotUpdateOwnerApprovalRequestRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	if record.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		record.CanaryRequirementID != authority.CanaryRequirementID ||
		record.SelectedCanaryEvidenceID != authority.SelectedCanaryEvidenceID ||
		record.ResultID != authority.ResultID ||
		record.RunID != authority.RunID ||
		record.CandidateID != authority.CandidateID ||
		record.EvalSuiteID != authority.EvalSuiteID ||
		record.PromotionPolicyID != authority.PromotionPolicyID ||
		record.BaselinePackID != authority.BaselinePackID ||
		record.CandidatePackID != authority.CandidatePackID ||
		record.AuthorityState != authority.State ||
		record.SatisfactionState != authority.SatisfactionState ||
		record.OwnerApprovalRequired != authority.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match canary satisfaction authority %q", record.OwnerApprovalRequestID, record.CanarySatisfactionAuthorityID)
	}
	return nil
}

func validateHotUpdateOwnerApprovalRequestRefs(record HotUpdateOwnerApprovalRequestRecord, authority HotUpdateCanarySatisfactionAuthorityRecord, requirement HotUpdateCanaryRequirementRecord, evidence HotUpdateCanaryEvidenceRecord) error {
	if authority.CanaryRequirementID != requirement.CanaryRequirementID ||
		authority.SelectedCanaryEvidenceID != evidence.CanaryEvidenceID {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match canary satisfaction authority source refs", record.OwnerApprovalRequestID)
	}
	if requirement.State != HotUpdateCanaryRequirementStateRequired {
		return fmt.Errorf("mission store hot-update owner approval request requirement %q state must be %q", requirement.CanaryRequirementID, HotUpdateCanaryRequirementStateRequired)
	}
	if record.CanaryRequirementID != requirement.CanaryRequirementID ||
		record.ResultID != requirement.ResultID ||
		record.RunID != requirement.RunID ||
		record.CandidateID != requirement.CandidateID ||
		record.EvalSuiteID != requirement.EvalSuiteID ||
		record.PromotionPolicyID != requirement.PromotionPolicyID ||
		record.BaselinePackID != requirement.BaselinePackID ||
		record.CandidatePackID != requirement.CandidatePackID ||
		!requirement.OwnerApprovalRequired ||
		requirement.EligibilityState != CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match hot-update canary requirement %q", record.OwnerApprovalRequestID, record.CanaryRequirementID)
	}
	if evidence.CanaryRequirementID != requirement.CanaryRequirementID ||
		evidence.ResultID != record.ResultID ||
		evidence.RunID != record.RunID ||
		evidence.CandidateID != record.CandidateID ||
		evidence.EvalSuiteID != record.EvalSuiteID ||
		evidence.PromotionPolicyID != record.PromotionPolicyID ||
		evidence.BaselinePackID != record.BaselinePackID ||
		evidence.CandidatePackID != record.CandidatePackID {
		return fmt.Errorf("mission store hot-update owner approval request %q does not match selected canary evidence %q", record.OwnerApprovalRequestID, record.SelectedCanaryEvidenceID)
	}
	return nil
}

func isValidHotUpdateOwnerApprovalRequestState(state HotUpdateOwnerApprovalRequestState) bool {
	switch state {
	case HotUpdateOwnerApprovalRequestStateRequested:
		return true
	default:
		return false
	}
}
