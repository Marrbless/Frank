package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type HotUpdateOwnerApprovalDecision string

const (
	HotUpdateOwnerApprovalDecisionGranted  HotUpdateOwnerApprovalDecision = "granted"
	HotUpdateOwnerApprovalDecisionRejected HotUpdateOwnerApprovalDecision = "rejected"
)

type HotUpdateOwnerApprovalDecisionRef struct {
	OwnerApprovalDecisionID string `json:"owner_approval_decision_id"`
}

type HotUpdateOwnerApprovalDecisionRecord struct {
	RecordVersion                 int                                       `json:"record_version"`
	OwnerApprovalDecisionID       string                                    `json:"owner_approval_decision_id"`
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
	RequestState                  HotUpdateOwnerApprovalRequestState        `json:"request_state"`
	AuthorityState                HotUpdateCanarySatisfactionAuthorityState `json:"authority_state"`
	SatisfactionState             HotUpdateCanarySatisfactionState          `json:"satisfaction_state"`
	OwnerApprovalRequired         bool                                      `json:"owner_approval_required"`
	Decision                      HotUpdateOwnerApprovalDecision            `json:"decision"`
	Reason                        string                                    `json:"reason"`
	DecidedAt                     time.Time                                 `json:"decided_at"`
	DecidedBy                     string                                    `json:"decided_by"`
}

var ErrHotUpdateOwnerApprovalDecisionRecordNotFound = errors.New("mission store hot-update owner approval decision record not found")

func StoreHotUpdateOwnerApprovalDecisionsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_owner_approval_decisions")
}

func StoreHotUpdateOwnerApprovalDecisionDir(root, ownerApprovalDecisionID string) string {
	return filepath.Join(StoreHotUpdateOwnerApprovalDecisionsDir(root), hotUpdateOwnerApprovalDecisionStorageKey(ownerApprovalDecisionID))
}

func StoreHotUpdateOwnerApprovalDecisionPath(root, ownerApprovalDecisionID string) string {
	return filepath.Join(StoreHotUpdateOwnerApprovalDecisionDir(root, ownerApprovalDecisionID), "record.json")
}

func HotUpdateOwnerApprovalDecisionIDFromRequest(ownerApprovalRequestID string) string {
	return "hot-update-owner-approval-decision-" + strings.TrimSpace(ownerApprovalRequestID)
}

func hotUpdateOwnerApprovalDecisionStorageKey(ownerApprovalDecisionID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ownerApprovalDecisionID)))
	return hex.EncodeToString(sum[:])
}

func NormalizeHotUpdateOwnerApprovalDecisionRef(ref HotUpdateOwnerApprovalDecisionRef) HotUpdateOwnerApprovalDecisionRef {
	ref.OwnerApprovalDecisionID = strings.TrimSpace(ref.OwnerApprovalDecisionID)
	return ref
}

func NormalizeHotUpdateOwnerApprovalDecisionRecord(record HotUpdateOwnerApprovalDecisionRecord) HotUpdateOwnerApprovalDecisionRecord {
	record.OwnerApprovalDecisionID = strings.TrimSpace(record.OwnerApprovalDecisionID)
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
	record.RequestState = HotUpdateOwnerApprovalRequestState(strings.TrimSpace(string(record.RequestState)))
	record.AuthorityState = HotUpdateCanarySatisfactionAuthorityState(strings.TrimSpace(string(record.AuthorityState)))
	record.SatisfactionState = HotUpdateCanarySatisfactionState(strings.TrimSpace(string(record.SatisfactionState)))
	record.Decision = HotUpdateOwnerApprovalDecision(strings.TrimSpace(string(record.Decision)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.DecidedAt = record.DecidedAt.UTC()
	record.DecidedBy = strings.TrimSpace(record.DecidedBy)
	return record
}

func ValidateHotUpdateOwnerApprovalDecisionRef(ref HotUpdateOwnerApprovalDecisionRef) error {
	return validateHotUpdateIdentifierField("hot-update owner approval decision ref", "owner_approval_decision_id", ref.OwnerApprovalDecisionID)
}

func HotUpdateOwnerApprovalDecisionRequestRef(record HotUpdateOwnerApprovalDecisionRecord) HotUpdateOwnerApprovalRequestRef {
	return HotUpdateOwnerApprovalRequestRef{OwnerApprovalRequestID: strings.TrimSpace(record.OwnerApprovalRequestID)}
}

func HotUpdateOwnerApprovalDecisionCanarySatisfactionAuthorityRef(record HotUpdateOwnerApprovalDecisionRecord) HotUpdateCanarySatisfactionAuthorityRef {
	return HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: strings.TrimSpace(record.CanarySatisfactionAuthorityID)}
}

func HotUpdateOwnerApprovalDecisionRequirementRef(record HotUpdateOwnerApprovalDecisionRecord) HotUpdateCanaryRequirementRef {
	return HotUpdateCanaryRequirementRef{CanaryRequirementID: strings.TrimSpace(record.CanaryRequirementID)}
}

func HotUpdateOwnerApprovalDecisionEvidenceRef(record HotUpdateOwnerApprovalDecisionRecord) HotUpdateCanaryEvidenceRef {
	return HotUpdateCanaryEvidenceRef{CanaryEvidenceID: strings.TrimSpace(record.SelectedCanaryEvidenceID)}
}

func HotUpdateOwnerApprovalDecisionCandidateResultRef(record HotUpdateOwnerApprovalDecisionRecord) CandidateResultRef {
	return CandidateResultRef{ResultID: strings.TrimSpace(record.ResultID)}
}

func HotUpdateOwnerApprovalDecisionImprovementRunRef(record HotUpdateOwnerApprovalDecisionRecord) ImprovementRunRef {
	return ImprovementRunRef{RunID: strings.TrimSpace(record.RunID)}
}

func HotUpdateOwnerApprovalDecisionImprovementCandidateRef(record HotUpdateOwnerApprovalDecisionRecord) ImprovementCandidateRef {
	return ImprovementCandidateRef{CandidateID: strings.TrimSpace(record.CandidateID)}
}

func HotUpdateOwnerApprovalDecisionEvalSuiteRef(record HotUpdateOwnerApprovalDecisionRecord) EvalSuiteRef {
	return EvalSuiteRef{EvalSuiteID: strings.TrimSpace(record.EvalSuiteID)}
}

func HotUpdateOwnerApprovalDecisionPromotionPolicyRef(record HotUpdateOwnerApprovalDecisionRecord) PromotionPolicyRef {
	return PromotionPolicyRef{PromotionPolicyID: strings.TrimSpace(record.PromotionPolicyID)}
}

func HotUpdateOwnerApprovalDecisionBaselinePackRef(record HotUpdateOwnerApprovalDecisionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.BaselinePackID)}
}

func HotUpdateOwnerApprovalDecisionCandidatePackRef(record HotUpdateOwnerApprovalDecisionRecord) RuntimePackRef {
	return RuntimePackRef{PackID: strings.TrimSpace(record.CandidatePackID)}
}

func ValidateHotUpdateOwnerApprovalDecisionRecord(record HotUpdateOwnerApprovalDecisionRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update owner approval decision record_version must be positive")
	}
	if err := ValidateHotUpdateOwnerApprovalDecisionRef(HotUpdateOwnerApprovalDecisionRef{OwnerApprovalDecisionID: record.OwnerApprovalDecisionID}); err != nil {
		return err
	}
	if err := ValidateHotUpdateOwnerApprovalRequestRef(HotUpdateOwnerApprovalDecisionRequestRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision owner_approval_request_id %q: %w", record.OwnerApprovalRequestID, err)
	}
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(HotUpdateOwnerApprovalDecisionCanarySatisfactionAuthorityRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision canary_satisfaction_authority_id %q: %w", record.CanarySatisfactionAuthorityID, err)
	}
	if err := ValidateHotUpdateCanaryRequirementRef(HotUpdateOwnerApprovalDecisionRequirementRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision canary_requirement_id %q: %w", record.CanaryRequirementID, err)
	}
	if err := ValidateHotUpdateCanaryEvidenceRef(HotUpdateOwnerApprovalDecisionEvidenceRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision selected_canary_evidence_id %q: %w", record.SelectedCanaryEvidenceID, err)
	}
	if err := ValidateCandidateResultRef(HotUpdateOwnerApprovalDecisionCandidateResultRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision result_id %q: %w", record.ResultID, err)
	}
	if err := ValidateImprovementRunRef(HotUpdateOwnerApprovalDecisionImprovementRunRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision run_id %q: %w", record.RunID, err)
	}
	if err := ValidateImprovementCandidateRef(HotUpdateOwnerApprovalDecisionImprovementCandidateRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision candidate_id %q: %w", record.CandidateID, err)
	}
	if err := ValidateEvalSuiteRef(HotUpdateOwnerApprovalDecisionEvalSuiteRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision eval_suite_id %q: %w", record.EvalSuiteID, err)
	}
	if err := ValidatePromotionPolicyRef(HotUpdateOwnerApprovalDecisionPromotionPolicyRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision promotion_policy_id %q: %w", record.PromotionPolicyID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateOwnerApprovalDecisionBaselinePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision baseline_pack_id %q: %w", record.BaselinePackID, err)
	}
	if err := ValidateRuntimePackRef(HotUpdateOwnerApprovalDecisionCandidatePackRef(record)); err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if record.OwnerApprovalDecisionID != HotUpdateOwnerApprovalDecisionIDFromRequest(record.OwnerApprovalRequestID) {
		return fmt.Errorf("mission store hot-update owner approval decision owner_approval_decision_id %q does not match deterministic owner_approval_decision_id %q", record.OwnerApprovalDecisionID, HotUpdateOwnerApprovalDecisionIDFromRequest(record.OwnerApprovalRequestID))
	}
	if !isValidHotUpdateOwnerApprovalRequestState(record.RequestState) {
		return fmt.Errorf("mission store hot-update owner approval decision request_state %q is invalid", record.RequestState)
	}
	if record.RequestState != HotUpdateOwnerApprovalRequestStateRequested {
		return fmt.Errorf("mission store hot-update owner approval decision request_state must be %q", HotUpdateOwnerApprovalRequestStateRequested)
	}
	if !isValidHotUpdateCanarySatisfactionAuthorityState(record.AuthorityState) {
		return fmt.Errorf("mission store hot-update owner approval decision authority_state %q is invalid", record.AuthorityState)
	}
	if record.AuthorityState != HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval decision authority_state must be %q", HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval)
	}
	if !isValidHotUpdateCanarySatisfactionAuthoritySatisfactionState(record.SatisfactionState) {
		return fmt.Errorf("mission store hot-update owner approval decision satisfaction_state %q is invalid", record.SatisfactionState)
	}
	if record.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
		return fmt.Errorf("mission store hot-update owner approval decision satisfaction_state must be %q", HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
	}
	if !record.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval decision owner_approval_required must be true")
	}
	if !isValidHotUpdateOwnerApprovalDecision(record.Decision) {
		return fmt.Errorf("mission store hot-update owner approval decision decision %q is invalid", record.Decision)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store hot-update owner approval decision reason is required")
	}
	if record.DecidedAt.IsZero() {
		return fmt.Errorf("mission store hot-update owner approval decision decided_at is required")
	}
	if record.DecidedBy == "" {
		return fmt.Errorf("mission store hot-update owner approval decision decided_by is required")
	}
	return nil
}

func StoreHotUpdateOwnerApprovalDecisionRecord(root string, record HotUpdateOwnerApprovalDecisionRecord) (HotUpdateOwnerApprovalDecisionRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	record = NormalizeHotUpdateOwnerApprovalDecisionRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateOwnerApprovalDecisionRecord(record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	if err := validateHotUpdateOwnerApprovalDecisionLinkage(root, record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	records, err := ListHotUpdateOwnerApprovalDecisionRecords(root)
	if err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	for _, existing := range records {
		if existing.OwnerApprovalRequestID == record.OwnerApprovalRequestID &&
			existing.OwnerApprovalDecisionID != record.OwnerApprovalDecisionID {
			return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf(
				"mission store hot-update owner approval decision for owner_approval_request_id %q already exists as %q",
				record.OwnerApprovalRequestID,
				existing.OwnerApprovalDecisionID,
			)
		}
	}

	path := StoreHotUpdateOwnerApprovalDecisionPath(root, record.OwnerApprovalDecisionID)
	existing, err := loadHotUpdateOwnerApprovalDecisionRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision %q already exists", record.OwnerApprovalDecisionID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}

	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	stored, err := LoadHotUpdateOwnerApprovalDecisionRecord(root, record.OwnerApprovalDecisionID)
	if err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	return stored, true, nil
}

func LoadHotUpdateOwnerApprovalDecisionRecord(root, ownerApprovalDecisionID string) (HotUpdateOwnerApprovalDecisionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	ref := NormalizeHotUpdateOwnerApprovalDecisionRef(HotUpdateOwnerApprovalDecisionRef{OwnerApprovalDecisionID: ownerApprovalDecisionID})
	if err := ValidateHotUpdateOwnerApprovalDecisionRef(ref); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	record, err := loadHotUpdateOwnerApprovalDecisionRecordFile(root, StoreHotUpdateOwnerApprovalDecisionPath(root, ref.OwnerApprovalDecisionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateOwnerApprovalDecisionRecord{}, ErrHotUpdateOwnerApprovalDecisionRecordNotFound
		}
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	return record, nil
}

func ListHotUpdateOwnerApprovalDecisionRecords(root string) ([]HotUpdateOwnerApprovalDecisionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StoreHotUpdateOwnerApprovalDecisionsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	records := make([]HotUpdateOwnerApprovalDecisionRecord, 0, len(names))
	for _, name := range names {
		path := filepath.Join(StoreHotUpdateOwnerApprovalDecisionsDir(root), name)
		if filepath.Ext(name) == "" {
			path = filepath.Join(path, "record.json")
		}
		record, err := loadHotUpdateOwnerApprovalDecisionRecordFile(root, path)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func CreateHotUpdateOwnerApprovalDecisionFromRequest(root, ownerApprovalRequestID string, decision HotUpdateOwnerApprovalDecision, decidedBy string, decidedAt time.Time, reason string) (HotUpdateOwnerApprovalDecisionRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	requestRef := NormalizeHotUpdateOwnerApprovalRequestRef(HotUpdateOwnerApprovalRequestRef{OwnerApprovalRequestID: ownerApprovalRequestID})
	if err := ValidateHotUpdateOwnerApprovalRequestRef(requestRef); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	decision = HotUpdateOwnerApprovalDecision(strings.TrimSpace(string(decision)))
	if !isValidHotUpdateOwnerApprovalDecision(decision) {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision decision %q is invalid", decision)
	}
	decidedBy = strings.TrimSpace(decidedBy)
	if decidedBy == "" {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision decided_by is required")
	}
	decidedAt = decidedAt.UTC()
	if decidedAt.IsZero() {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision decided_at is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision reason is required")
	}

	request, err := LoadHotUpdateOwnerApprovalRequestRecord(root, requestRef.OwnerApprovalRequestID)
	if err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, err
	}
	if request.State != HotUpdateOwnerApprovalRequestStateRequested {
		return HotUpdateOwnerApprovalDecisionRecord{}, false, fmt.Errorf("mission store hot-update owner approval decision requires owner approval request state %q, found %q", HotUpdateOwnerApprovalRequestStateRequested, request.State)
	}

	record := NormalizeHotUpdateOwnerApprovalDecisionRecord(HotUpdateOwnerApprovalDecisionRecord{
		RecordVersion:                 StoreRecordVersion,
		OwnerApprovalDecisionID:       HotUpdateOwnerApprovalDecisionIDFromRequest(request.OwnerApprovalRequestID),
		OwnerApprovalRequestID:        request.OwnerApprovalRequestID,
		CanarySatisfactionAuthorityID: request.CanarySatisfactionAuthorityID,
		CanaryRequirementID:           request.CanaryRequirementID,
		SelectedCanaryEvidenceID:      request.SelectedCanaryEvidenceID,
		ResultID:                      request.ResultID,
		RunID:                         request.RunID,
		CandidateID:                   request.CandidateID,
		EvalSuiteID:                   request.EvalSuiteID,
		PromotionPolicyID:             request.PromotionPolicyID,
		BaselinePackID:                request.BaselinePackID,
		CandidatePackID:               request.CandidatePackID,
		RequestState:                  request.State,
		AuthorityState:                request.AuthorityState,
		SatisfactionState:             request.SatisfactionState,
		OwnerApprovalRequired:         request.OwnerApprovalRequired,
		Decision:                      decision,
		Reason:                        reason,
		DecidedAt:                     decidedAt,
		DecidedBy:                     decidedBy,
	})
	return StoreHotUpdateOwnerApprovalDecisionRecord(root, record)
}

func loadHotUpdateOwnerApprovalDecisionRecordFile(root, path string) (HotUpdateOwnerApprovalDecisionRecord, error) {
	var record HotUpdateOwnerApprovalDecisionRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	record = NormalizeHotUpdateOwnerApprovalDecisionRecord(record)
	if err := ValidateHotUpdateOwnerApprovalDecisionRecord(record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	if err := validateHotUpdateOwnerApprovalDecisionLinkage(root, record); err != nil {
		return HotUpdateOwnerApprovalDecisionRecord{}, err
	}
	return record, nil
}

func validateHotUpdateOwnerApprovalDecisionLinkage(root string, record HotUpdateOwnerApprovalDecisionRecord) error {
	request, err := LoadHotUpdateOwnerApprovalRequestRecord(root, record.OwnerApprovalRequestID)
	if err != nil {
		return fmt.Errorf("mission store hot-update owner approval decision owner_approval_request_id %q: %w", record.OwnerApprovalRequestID, err)
	}
	if request.State != HotUpdateOwnerApprovalRequestStateRequested {
		return fmt.Errorf("mission store hot-update owner approval decision requires owner approval request state %q, found %q", HotUpdateOwnerApprovalRequestStateRequested, request.State)
	}
	if err := validateHotUpdateOwnerApprovalDecisionMatchesRequest(record, request); err != nil {
		return err
	}
	return nil
}

func validateHotUpdateOwnerApprovalDecisionMatchesRequest(record HotUpdateOwnerApprovalDecisionRecord, request HotUpdateOwnerApprovalRequestRecord) error {
	if record.OwnerApprovalRequestID != request.OwnerApprovalRequestID ||
		record.CanarySatisfactionAuthorityID != request.CanarySatisfactionAuthorityID ||
		record.CanaryRequirementID != request.CanaryRequirementID ||
		record.SelectedCanaryEvidenceID != request.SelectedCanaryEvidenceID ||
		record.ResultID != request.ResultID ||
		record.RunID != request.RunID ||
		record.CandidateID != request.CandidateID ||
		record.EvalSuiteID != request.EvalSuiteID ||
		record.PromotionPolicyID != request.PromotionPolicyID ||
		record.BaselinePackID != request.BaselinePackID ||
		record.CandidatePackID != request.CandidatePackID ||
		record.RequestState != request.State ||
		record.AuthorityState != request.AuthorityState ||
		record.SatisfactionState != request.SatisfactionState ||
		record.OwnerApprovalRequired != request.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval decision %q does not match owner approval request %q", record.OwnerApprovalDecisionID, record.OwnerApprovalRequestID)
	}
	return nil
}

func isValidHotUpdateOwnerApprovalDecision(decision HotUpdateOwnerApprovalDecision) bool {
	switch decision {
	case HotUpdateOwnerApprovalDecisionGranted, HotUpdateOwnerApprovalDecisionRejected:
		return true
	default:
		return false
	}
}
