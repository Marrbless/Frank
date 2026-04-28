package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type HotUpdateReloadMode string

const (
	HotUpdateReloadModeSoftReload         HotUpdateReloadMode = "soft_reload"
	HotUpdateReloadModeSkillReload        HotUpdateReloadMode = "skill_reload"
	HotUpdateReloadModeExtensionReload    HotUpdateReloadMode = "extension_reload"
	HotUpdateReloadModePackReload         HotUpdateReloadMode = "pack_reload"
	HotUpdateReloadModeCanaryReload       HotUpdateReloadMode = "canary_reload"
	HotUpdateReloadModeProcessRestartSwap HotUpdateReloadMode = "process_restart_hot_swap"
	HotUpdateReloadModeColdRestart        HotUpdateReloadMode = "cold_restart_required"
)

type HotUpdateGateState string

const (
	HotUpdateGateStateDraft                     HotUpdateGateState = "draft"
	HotUpdateGateStatePrepared                  HotUpdateGateState = "prepared"
	HotUpdateGateStateValidated                 HotUpdateGateState = "validated"
	HotUpdateGateStateStaged                    HotUpdateGateState = "staged"
	HotUpdateGateStateQuiescing                 HotUpdateGateState = "quiescing"
	HotUpdateGateStateReloading                 HotUpdateGateState = "reloading"
	HotUpdateGateStateReloadApplyInProgress     HotUpdateGateState = "reload_apply_in_progress"
	HotUpdateGateStateReloadApplyRecoveryNeeded HotUpdateGateState = "reload_apply_recovery_needed"
	HotUpdateGateStateReloadApplySucceeded      HotUpdateGateState = "reload_apply_succeeded"
	HotUpdateGateStateReloadApplyFailed         HotUpdateGateState = "reload_apply_failed"
	HotUpdateGateStateSmokeTesting              HotUpdateGateState = "smoke_testing"
	HotUpdateGateStateCanarying                 HotUpdateGateState = "canarying"
	HotUpdateGateStateCommitted                 HotUpdateGateState = "committed"
	HotUpdateGateStateRolledBack                HotUpdateGateState = "rolled_back"
	HotUpdateGateStateRejected                  HotUpdateGateState = "rejected"
	HotUpdateGateStateFailed                    HotUpdateGateState = "failed"
	HotUpdateGateStateAborted                   HotUpdateGateState = "aborted"
)

type HotUpdateGateDecision string

const (
	HotUpdateGateDecisionKeepStaged         HotUpdateGateDecision = "keep_staged"
	HotUpdateGateDecisionDiscard            HotUpdateGateDecision = "discard"
	HotUpdateGateDecisionBlock              HotUpdateGateDecision = "block"
	HotUpdateGateDecisionApplyHotUpdate     HotUpdateGateDecision = "apply_hot_update"
	HotUpdateGateDecisionApplyCanary        HotUpdateGateDecision = "apply_canary"
	HotUpdateGateDecisionRequireApproval    HotUpdateGateDecision = "require_approval"
	HotUpdateGateDecisionRequireColdRestart HotUpdateGateDecision = "require_cold_restart"
	HotUpdateGateDecisionRollback           HotUpdateGateDecision = "rollback"
)

type HotUpdateGateRef struct {
	HotUpdateID string `json:"hot_update_id"`
}

type CandidateRuntimePackPointer struct {
	RecordVersion   int       `json:"record_version"`
	HotUpdateID     string    `json:"hot_update_id"`
	CandidatePackID string    `json:"candidate_pack_id"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

type HotUpdateGateRecord struct {
	RecordVersion            int                   `json:"record_version"`
	HotUpdateID              string                `json:"hot_update_id"`
	Objective                string                `json:"objective"`
	CandidatePackID          string                `json:"candidate_pack_id"`
	PreviousActivePackID     string                `json:"previous_active_pack_id"`
	RollbackTargetPackID     string                `json:"rollback_target_pack_id"`
	TargetSurfaces           []string              `json:"target_surfaces"`
	SurfaceClasses           []string              `json:"surface_classes"`
	ReloadMode               HotUpdateReloadMode   `json:"reload_mode"`
	CompatibilityContractRef string                `json:"compatibility_contract_ref"`
	EvalEvidenceRefs         []string              `json:"eval_evidence_refs,omitempty"`
	SmokeCheckRefs           []string              `json:"smoke_check_refs,omitempty"`
	CanaryRef                string                `json:"canary_ref,omitempty"`
	ApprovalRef              string                `json:"approval_ref,omitempty"`
	BudgetRef                string                `json:"budget_ref,omitempty"`
	PreparedAt               time.Time             `json:"prepared_at"`
	PhaseUpdatedAt           time.Time             `json:"phase_updated_at,omitempty"`
	PhaseUpdatedBy           string                `json:"phase_updated_by,omitempty"`
	State                    HotUpdateGateState    `json:"state"`
	Decision                 HotUpdateGateDecision `json:"decision"`
	FailureReason            string                `json:"failure_reason,omitempty"`
}

type HotUpdateCanaryGateExecutionReadinessAssessment struct {
	State                         string                           `json:"state"`
	HotUpdateID                   string                           `json:"hot_update_id,omitempty"`
	CanaryRef                     string                           `json:"canary_ref,omitempty"`
	ApprovalRef                   string                           `json:"approval_ref,omitempty"`
	CanarySatisfactionAuthorityID string                           `json:"canary_satisfaction_authority_id,omitempty"`
	OwnerApprovalDecisionID       string                           `json:"owner_approval_decision_id,omitempty"`
	ResultID                      string                           `json:"result_id,omitempty"`
	RunID                         string                           `json:"run_id,omitempty"`
	CandidateID                   string                           `json:"candidate_id,omitempty"`
	EvalSuiteID                   string                           `json:"eval_suite_id,omitempty"`
	PromotionPolicyID             string                           `json:"promotion_policy_id,omitempty"`
	BaselinePackID                string                           `json:"baseline_pack_id,omitempty"`
	CandidatePackID               string                           `json:"candidate_pack_id,omitempty"`
	ExpectedEligibilityState      string                           `json:"expected_eligibility_state,omitempty"`
	SatisfactionState             HotUpdateCanarySatisfactionState `json:"satisfaction_state,omitempty"`
	OwnerApprovalRequired         bool                             `json:"owner_approval_required"`
	Ready                         bool                             `json:"ready"`
	Reason                        string                           `json:"reason,omitempty"`
	Error                         string                           `json:"error,omitempty"`
}

var (
	ErrHotUpdateGateRecordNotFound         = errors.New("mission store hot-update gate record not found")
	ErrCandidateRuntimePackPointerNotFound = errors.New("mission store candidate runtime pack pointer not found")
)

func StoreHotUpdateGatesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "hot_update_gates")
}

func StoreHotUpdateGatePath(root, hotUpdateID string) string {
	return filepath.Join(StoreHotUpdateGatesDir(root), strings.TrimSpace(hotUpdateID)+".json")
}

func StoreCandidateRuntimePackPointerPath(root string) string {
	return filepath.Join(root, "runtime_packs", "candidate_pointer.json")
}

func NormalizeHotUpdateGateRef(ref HotUpdateGateRef) HotUpdateGateRef {
	ref.HotUpdateID = strings.TrimSpace(ref.HotUpdateID)
	return ref
}

func NormalizeCandidateRuntimePackPointer(pointer CandidateRuntimePackPointer) CandidateRuntimePackPointer {
	pointer.HotUpdateID = strings.TrimSpace(pointer.HotUpdateID)
	pointer.CandidatePackID = strings.TrimSpace(pointer.CandidatePackID)
	pointer.UpdatedAt = pointer.UpdatedAt.UTC()
	pointer.UpdatedBy = strings.TrimSpace(pointer.UpdatedBy)
	return pointer
}

func NormalizeHotUpdateGateRecord(record HotUpdateGateRecord) HotUpdateGateRecord {
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.Objective = strings.TrimSpace(record.Objective)
	record.CandidatePackID = strings.TrimSpace(record.CandidatePackID)
	record.PreviousActivePackID = strings.TrimSpace(record.PreviousActivePackID)
	record.RollbackTargetPackID = strings.TrimSpace(record.RollbackTargetPackID)
	record.TargetSurfaces = normalizeHotUpdateStrings(record.TargetSurfaces)
	record.SurfaceClasses = normalizeHotUpdateStrings(record.SurfaceClasses)
	record.CompatibilityContractRef = strings.TrimSpace(record.CompatibilityContractRef)
	record.EvalEvidenceRefs = normalizeHotUpdateStrings(record.EvalEvidenceRefs)
	record.SmokeCheckRefs = normalizeHotUpdateStrings(record.SmokeCheckRefs)
	record.CanaryRef = strings.TrimSpace(record.CanaryRef)
	record.ApprovalRef = strings.TrimSpace(record.ApprovalRef)
	record.BudgetRef = strings.TrimSpace(record.BudgetRef)
	record.PreparedAt = record.PreparedAt.UTC()
	record.PhaseUpdatedAt = record.PhaseUpdatedAt.UTC()
	record.PhaseUpdatedBy = strings.TrimSpace(record.PhaseUpdatedBy)
	if record.PreparedAt.IsZero() {
		record.PhaseUpdatedAt = time.Time{}
		record.PhaseUpdatedBy = strings.TrimSpace(record.PhaseUpdatedBy)
	} else {
		if record.PhaseUpdatedAt.IsZero() {
			record.PhaseUpdatedAt = record.PreparedAt
		}
		if record.PhaseUpdatedBy == "" {
			record.PhaseUpdatedBy = "operator"
		}
	}
	record.FailureReason = strings.TrimSpace(record.FailureReason)
	return record
}

func ValidateHotUpdateGateRef(ref HotUpdateGateRef) error {
	return validateHotUpdateIdentifierField("hot-update gate ref", "hot_update_id", ref.HotUpdateID)
}

func ValidateCandidateRuntimePackPointer(pointer CandidateRuntimePackPointer) error {
	if pointer.RecordVersion <= 0 {
		return fmt.Errorf("mission store candidate runtime pack pointer record_version must be positive")
	}
	if err := validateHotUpdateIdentifierField("mission store candidate runtime pack pointer", "hot_update_id", pointer.HotUpdateID); err != nil {
		return err
	}
	if err := validateRuntimePackIDField("mission store candidate runtime pack pointer", "candidate_pack_id", pointer.CandidatePackID); err != nil {
		return err
	}
	if pointer.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store candidate runtime pack pointer updated_at is required")
	}
	if pointer.UpdatedBy == "" {
		return fmt.Errorf("mission store candidate runtime pack pointer updated_by is required")
	}
	return nil
}

func ValidateHotUpdateGateRecord(record HotUpdateGateRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store hot-update gate record_version must be positive")
	}
	if err := validateHotUpdateIdentifierField("mission store hot-update gate", "hot_update_id", record.HotUpdateID); err != nil {
		return err
	}
	if record.Objective == "" {
		return fmt.Errorf("mission store hot-update gate objective is required")
	}
	if err := validateRuntimePackIDField("mission store hot-update gate", "candidate_pack_id", record.CandidatePackID); err != nil {
		return err
	}
	if err := validateRuntimePackIDField("mission store hot-update gate", "previous_active_pack_id", record.PreviousActivePackID); err != nil {
		return err
	}
	if err := validateRuntimePackIDField("mission store hot-update gate", "rollback_target_pack_id", record.RollbackTargetPackID); err != nil {
		return err
	}
	if len(record.TargetSurfaces) == 0 {
		return fmt.Errorf("mission store hot-update gate target_surfaces are required")
	}
	if len(record.SurfaceClasses) == 0 {
		return fmt.Errorf("mission store hot-update gate surface_classes are required")
	}
	if !isValidHotUpdateReloadMode(record.ReloadMode) {
		return fmt.Errorf("mission store hot-update gate reload_mode %q is invalid", record.ReloadMode)
	}
	if record.CompatibilityContractRef == "" {
		return fmt.Errorf("mission store hot-update gate compatibility_contract_ref is required")
	}
	if record.PreparedAt.IsZero() {
		return fmt.Errorf("mission store hot-update gate prepared_at is required")
	}
	if record.PhaseUpdatedAt.IsZero() {
		return fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}
	if record.PhaseUpdatedBy == "" {
		return fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	if record.PhaseUpdatedAt.Before(record.PreparedAt) {
		return fmt.Errorf("mission store hot-update gate phase_updated_at must not precede prepared_at")
	}
	if !isValidHotUpdateGateState(record.State) {
		return fmt.Errorf("mission store hot-update gate state %q is invalid", record.State)
	}
	if !isValidHotUpdateGateDecision(record.Decision) {
		return fmt.Errorf("mission store hot-update gate decision %q is invalid", record.Decision)
	}
	return nil
}

func StoreHotUpdateGateRecord(root string, record HotUpdateGateRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeHotUpdateGateRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return err
	}
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.PreviousActivePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate previous_active_pack_id %q: %w", record.PreviousActivePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.RollbackTargetPackID); err != nil {
		return fmt.Errorf("mission store hot-update gate rollback_target_pack_id %q: %w", record.RollbackTargetPackID, err)
	}
	if err := requireHotUpdateGateExtensionPermissionAdmission(root, record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record)
}

func LoadHotUpdateGateRecord(root, hotUpdateID string) (HotUpdateGateRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, err
	}
	if err := validateHotUpdateIdentifierField("mission store hot-update gate", "hot_update_id", hotUpdateID); err != nil {
		return HotUpdateGateRecord{}, err
	}
	record, err := loadHotUpdateGateRecordFile(StoreHotUpdateGatePath(root, strings.TrimSpace(hotUpdateID)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HotUpdateGateRecord{}, ErrHotUpdateGateRecordNotFound
		}
		return HotUpdateGateRecord{}, err
	}
	return record, nil
}

func ListHotUpdateGateRecords(root string) ([]HotUpdateGateRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreHotUpdateGatesDir(root), loadHotUpdateGateRecordFile)
}

func EnsureHotUpdateGateRecordFromCandidate(root string, hotUpdateID string, candidatePackID string, createdBy string, requestedAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	candidateRef := NormalizeRuntimePackRef(RuntimePackRef{PackID: candidatePackID})
	if err := ValidateRuntimePackRef(candidateRef); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	existing, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err == nil {
		if existing.CandidatePackID != candidateRef.PackID {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q candidate_pack_id %q does not match requested candidate_pack_id %q",
				ref.HotUpdateID,
				existing.CandidatePackID,
				candidateRef.PackID,
			)
		}
		if err := validateHotUpdateGateDerivedLinkage(root, existing); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		return HotUpdateGateRecord{}, false, err
	}

	record, err := buildHotUpdateGateRecordFromCandidate(root, ref.HotUpdateID, candidateRef.PackID, createdBy, requestedAt)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	stored, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return stored, true, nil
}

func CreateHotUpdateGateFromCandidatePromotionDecision(root, promotionDecisionID, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	decisionRef := NormalizeCandidatePromotionDecisionRef(CandidatePromotionDecisionRef{PromotionDecisionID: promotionDecisionID})
	if err := ValidateCandidatePromotionDecisionRef(decisionRef); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate created_at is required")
	}

	decision, err := LoadCandidatePromotionDecisionRecord(root, decisionRef.PromotionDecisionID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if decision.Decision != CandidatePromotionDecisionSelectedForPromotion {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store candidate promotion decision %q decision %q does not permit hot-update gate creation", decision.PromotionDecisionID, decision.Decision)
	}
	if decision.EligibilityState != CandidatePromotionEligibilityStateEligible {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store candidate promotion decision %q eligibility_state %q does not permit hot-update gate creation", decision.PromotionDecisionID, decision.EligibilityState)
	}

	result, err := LoadCandidateResultRecord(root, decision.ResultID)
	if err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision result_id %q: %w", decision.ResultID, err)
	}
	if err := validateCandidatePromotionDecisionResultAuthority(decision, result); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	status, err := EvaluateCandidateResultPromotionEligibility(root, decision.ResultID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if status.State != CandidatePromotionEligibilityStateEligible {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update gate creation", decision.ResultID, status.State)
	}
	if err := validateCandidatePromotionDecisionEligibilityAuthority(decision, status); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	run, err := LoadImprovementRunRecord(root, decision.RunID)
	if err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision run_id %q: %w", decision.RunID, err)
	}
	if err := validateCandidatePromotionDecisionRunAuthority(decision, run); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	candidate, err := LoadImprovementCandidateRecord(root, decision.CandidateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision candidate_id %q: %w", decision.CandidateID, err)
	}
	if err := validateCandidatePromotionDecisionCandidateAuthority(decision, candidate); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	evalSuite, err := LoadEvalSuiteRecord(root, decision.EvalSuiteID)
	if err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision eval_suite_id %q: %w", decision.EvalSuiteID, err)
	}
	if err := validateCandidatePromotionDecisionEvalSuiteAuthority(decision, evalSuite); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if _, err := LoadPromotionPolicyRecord(root, decision.PromotionPolicyID); err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision promotion_policy_id %q: %w", decision.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, decision.BaselinePackID); err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision baseline_pack_id %q: %w", decision.BaselinePackID, err)
	}
	candidatePack, err := LoadRuntimePackRecord(root, decision.CandidatePackID)
	if err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate promotion decision candidate_pack_id %q: %w", decision.CandidatePackID, err)
	}
	if candidatePack.RollbackTargetPackID == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate_pack_id %q rollback_target_pack_id is required", candidatePack.PackID)
	}
	if _, err := LoadRuntimePackRecord(root, candidatePack.RollbackTargetPackID); err != nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate candidate_pack_id %q rollback_target_pack_id %q: %w", candidatePack.PackID, candidatePack.RollbackTargetPackID, err)
	}

	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if activePointer.ActivePackID != decision.BaselinePackID {
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate candidate promotion decision %q requires active runtime pack pointer active_pack_id %q, found %q",
			decision.PromotionDecisionID,
			decision.BaselinePackID,
			activePointer.ActivePackID,
		)
	}
	if _, err := LoadLastKnownGoodRuntimePackPointer(root); err != nil && !errors.Is(err, ErrLastKnownGoodRuntimePackPointerNotFound) {
		return HotUpdateGateRecord{}, false, err
	}

	hotUpdateID := hotUpdateIDFromCandidatePromotionDecision(decision.PromotionDecisionID)
	record, err := buildHotUpdateGateRecordFromCandidate(root, hotUpdateID, decision.CandidatePackID, createdBy, createdAt)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	record = NormalizeHotUpdateGateRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	existing, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		if existing.CandidatePackID != record.CandidatePackID {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q candidate_pack_id %q does not match candidate promotion decision candidate_pack_id %q",
				record.HotUpdateID,
				existing.CandidatePackID,
				record.CandidatePackID,
			)
		}
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate %q already exists with divergent candidate promotion decision authority", record.HotUpdateID)
	}
	if !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		return HotUpdateGateRecord{}, false, err
	}

	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	stored, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return stored, true, nil
}

func HotUpdateGateIDFromCanarySatisfactionAuthority(canarySatisfactionAuthorityID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(canarySatisfactionAuthorityID)))
	return "hot-update-canary-gate-" + hex.EncodeToString(sum[:])
}

func CreateHotUpdateGateFromCanarySatisfactionAuthority(root string, canarySatisfactionAuthorityID string, ownerApprovalDecisionID string, createdBy string, createdAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	authorityRef := NormalizeHotUpdateCanarySatisfactionAuthorityRef(HotUpdateCanarySatisfactionAuthorityRef{CanarySatisfactionAuthorityID: canarySatisfactionAuthorityID})
	if err := ValidateHotUpdateCanarySatisfactionAuthorityRef(authorityRef); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update canary gate created_by is required")
	}
	createdAt = createdAt.UTC()
	if createdAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update canary gate created_at is required")
	}

	authority, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, authorityRef.CanarySatisfactionAuthorityID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateCanaryAuthoritySource(root, authority); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	approvalRef, expectedEligibility, expectedSatisfaction, err := validateHotUpdateGateCanaryOwnerApprovalBranch(root, authority, ownerApprovalDecisionID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateCanaryAuthorityFreshness(root, authority, expectedSatisfaction, expectedEligibility); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateCanaryAuthorityRuntimeReadiness(root, authority); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	hotUpdateID := HotUpdateGateIDFromCanarySatisfactionAuthority(authority.CanarySatisfactionAuthorityID)
	record, err := buildHotUpdateGateRecordFromCandidate(root, hotUpdateID, authority.CandidatePackID, createdBy, createdAt)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	record.CanaryRef = authority.CanarySatisfactionAuthorityID
	record.ApprovalRef = approvalRef
	record = NormalizeHotUpdateGateRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	existing, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		if existing.CandidatePackID != record.CandidatePackID {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q candidate_pack_id %q does not match canary satisfaction authority candidate_pack_id %q",
				record.HotUpdateID,
				existing.CandidatePackID,
				record.CandidatePackID,
			)
		}
		if existing.CanaryRef != record.CanaryRef {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q canary_ref %q does not match canary satisfaction authority %q",
				record.HotUpdateID,
				existing.CanaryRef,
				record.CanaryRef,
			)
		}
		if existing.ApprovalRef != record.ApprovalRef {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q approval_ref %q does not match owner approval decision %q",
				record.HotUpdateID,
				existing.ApprovalRef,
				record.ApprovalRef,
			)
		}
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate %q already exists with divergent canary satisfaction authority", record.HotUpdateID)
	}
	if !errors.Is(err, ErrHotUpdateGateRecordNotFound) {
		return HotUpdateGateRecord{}, false, err
	}

	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	stored, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return stored, true, nil
}

func AssessHotUpdateCanaryGateExecutionReadiness(root string, hotUpdateID string) (HotUpdateCanaryGateExecutionReadinessAssessment, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateCanaryGateExecutionReadinessAssessment{}, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateCanaryGateExecutionReadinessAssessment{}, err
	}
	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateCanaryGateExecutionReadinessAssessment{}, err
	}
	assessment := hotUpdateCanaryGateExecutionReadinessAssessmentFromGate(record)
	if record.CanaryRef == "" {
		assessment.State = "not_applicable"
		assessment.Ready = true
		assessment.Reason = "hot-update gate is not canary-derived"
		return assessment, nil
	}

	authority, err := LoadHotUpdateCanarySatisfactionAuthorityRecord(root, record.CanaryRef)
	if err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}
	assessment = hotUpdateCanaryGateExecutionReadinessAssessmentFromGateAndAuthority(record, authority)
	approvalRef, expectedEligibility, expectedSatisfaction, err := validateHotUpdateGateCanaryOwnerApprovalBranch(root, authority, record.ApprovalRef)
	if err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}
	assessment.OwnerApprovalDecisionID = approvalRef
	assessment.ExpectedEligibilityState = expectedEligibility
	assessment.SatisfactionState = expectedSatisfaction

	if err := validateHotUpdateGateCanaryAuthorityMatchesGate(root, record, authority); err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}
	if err := validateHotUpdateGateCanaryAuthoritySource(root, authority); err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}
	if err := validateHotUpdateGateCanaryAuthorityFreshness(root, authority, expectedSatisfaction, expectedEligibility); err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}
	if err := validateHotUpdateGateCanaryExecutionRuntimeReadiness(root, record, authority); err != nil {
		return hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment, err), err
	}

	assessment.State = "ready"
	assessment.Ready = true
	assessment.Reason = "canary-derived hot-update gate authority is ready"
	return assessment, nil
}

func AdvanceHotUpdateGatePhase(root string, hotUpdateID string, nextState HotUpdateGateState, updatedBy string, updatedAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}
	if nextState != HotUpdateGateStatePrepared && nextState != HotUpdateGateStateValidated && nextState != HotUpdateGateStateStaged {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate target state %q is invalid", nextState)
	}

	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if !isValidHotUpdateGatePhaseStartState(record.State) {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate %q state %q cannot advance via phase control", ref.HotUpdateID, record.State)
	}
	if record.State == nextState {
		return record, false, nil
	}
	if !isValidHotUpdateGateAdjacentTransition(record.State, nextState) {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate %q transition %q -> %q is invalid", ref.HotUpdateID, record.State, nextState)
	}
	if record.CanaryRef != "" && (nextState == HotUpdateGateStateValidated || nextState == HotUpdateGateStateStaged) {
		if err := requireHotUpdateCanaryGateExecutionReadiness(root, record); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
	}

	record.State = nextState
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	stored, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return stored, true, nil
}

func ExecuteHotUpdateGatePointerSwitch(root string, hotUpdateID string, updatedBy string, updatedAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}

	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := requireHotUpdateCanaryGateExecutionReadiness(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	activePointer, err := validateHotUpdateGateExecutionLinkage(root, record)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	updateRecordRef := hotUpdateGatePointerUpdateRecordRef(ref.HotUpdateID)
	switch record.State {
	case HotUpdateGateStateReloading:
		if activePointer.ActivePackID != record.CandidatePackID {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q state %q requires active runtime pack pointer active_pack_id %q, found %q",
				ref.HotUpdateID,
				record.State,
				record.CandidatePackID,
				activePointer.ActivePackID,
			)
		}
		if activePointer.UpdateRecordRef != updateRecordRef {
			return HotUpdateGateRecord{}, false, fmt.Errorf(
				"mission store hot-update gate %q state %q requires active runtime pack pointer update_record_ref %q, found %q",
				ref.HotUpdateID,
				record.State,
				updateRecordRef,
				activePointer.UpdateRecordRef,
			)
		}
		return record, false, nil
	case HotUpdateGateStateStaged:
	default:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q does not permit pointer switch execution",
			ref.HotUpdateID,
			record.State,
		)
	}

	if activePointer.ActivePackID == record.CandidatePackID && activePointer.UpdateRecordRef == updateRecordRef {
		record.State = HotUpdateGateStateReloading
		record.PhaseUpdatedAt = updatedAt
		record.PhaseUpdatedBy = updatedBy
		if err := StoreHotUpdateGateRecord(root, record); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		stored, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
		if err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		return stored, true, nil
	}

	if activePointer.ActivePackID != record.PreviousActivePackID {
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q requires active runtime pack pointer active_pack_id %q before switch, found %q",
			ref.HotUpdateID,
			record.PreviousActivePackID,
			activePointer.ActivePackID,
		)
	}

	activePointer.ActivePackID = record.CandidatePackID
	activePointer.PreviousActivePackID = record.PreviousActivePackID
	activePointer.UpdatedAt = updatedAt
	activePointer.UpdatedBy = updatedBy
	activePointer.UpdateRecordRef = updateRecordRef
	activePointer.ReloadGeneration++
	if err := StoreActiveRuntimePackPointer(root, activePointer); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	record.State = HotUpdateGateStateReloading
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	if err := StoreHotUpdateGateRecord(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	stored, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return stored, true, nil
}

func ExecuteHotUpdateGateReloadApply(root string, hotUpdateID string, updatedBy string, updatedAt time.Time) (HotUpdateGateRecord, bool, error) {
	return executeHotUpdateGateReloadApplyWithConvergence(root, hotUpdateID, updatedBy, updatedAt, hotUpdateGateRestartStyleConvergence)
}

func ResolveHotUpdateGateTerminalFailure(root string, hotUpdateID string, reason string, updatedBy string, updatedAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate terminal failure reason is required")
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}

	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	expectedFailureReason := hotUpdateGateTerminalFailureReason(reason)
	switch record.State {
	case HotUpdateGateStateReloadApplyFailed:
		if record.FailureReason == expectedFailureReason {
			return record, false, nil
		}
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q already resolved with failure_reason %q",
			ref.HotUpdateID,
			record.State,
			record.FailureReason,
		)
	case HotUpdateGateStateReloadApplyRecoveryNeeded:
	default:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q does not permit terminal failure resolution",
			ref.HotUpdateID,
			record.State,
		)
	}

	if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	record.State = HotUpdateGateStateReloadApplyFailed
	record.FailureReason = expectedFailureReason
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeHotUpdateGateRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	record, err = LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return record, true, nil
}

func ReconcileHotUpdateGateRecoveryNeeded(root string, hotUpdateID string, updatedBy string, updatedAt time.Time) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}

	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	switch record.State {
	case HotUpdateGateStateReloadApplyRecoveryNeeded:
		if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		return record, false, nil
	case HotUpdateGateStateReloadApplyInProgress:
	default:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q does not permit recovery-needed normalization",
			ref.HotUpdateID,
			record.State,
		)
	}

	if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	record.State = HotUpdateGateStateReloadApplyRecoveryNeeded
	record.FailureReason = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeHotUpdateGateRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	record, err = LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return record, true, nil
}

func StoreCandidateRuntimePackPointer(root string, pointer CandidateRuntimePackPointer) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	pointer = NormalizeCandidateRuntimePackPointer(pointer)
	pointer.RecordVersion = normalizeRecordVersion(pointer.RecordVersion)
	if err := ValidateCandidateRuntimePackPointer(pointer); err != nil {
		return err
	}
	gate, err := LoadHotUpdateGateRecord(root, pointer.HotUpdateID)
	if err != nil {
		return fmt.Errorf("mission store candidate runtime pack pointer hot_update_id %q: %w", pointer.HotUpdateID, err)
	}
	if gate.CandidatePackID != pointer.CandidatePackID {
		return fmt.Errorf("mission store candidate runtime pack pointer candidate_pack_id %q does not match hot-update gate candidate_pack_id %q", pointer.CandidatePackID, gate.CandidatePackID)
	}
	if _, err := LoadRuntimePackRecord(root, pointer.CandidatePackID); err != nil {
		return fmt.Errorf("mission store candidate runtime pack pointer candidate_pack_id %q: %w", pointer.CandidatePackID, err)
	}
	return WriteStoreJSONAtomic(StoreCandidateRuntimePackPointerPath(root), pointer)
}

func LoadCandidateRuntimePackPointer(root string) (CandidateRuntimePackPointer, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CandidateRuntimePackPointer{}, err
	}
	var pointer CandidateRuntimePackPointer
	if err := LoadStoreJSON(StoreCandidateRuntimePackPointerPath(root), &pointer); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CandidateRuntimePackPointer{}, ErrCandidateRuntimePackPointerNotFound
		}
		return CandidateRuntimePackPointer{}, err
	}
	pointer = NormalizeCandidateRuntimePackPointer(pointer)
	if err := ValidateCandidateRuntimePackPointer(pointer); err != nil {
		return CandidateRuntimePackPointer{}, err
	}
	gate, err := LoadHotUpdateGateRecord(root, pointer.HotUpdateID)
	if err != nil {
		return CandidateRuntimePackPointer{}, fmt.Errorf("mission store candidate runtime pack pointer hot_update_id %q: %w", pointer.HotUpdateID, err)
	}
	if gate.CandidatePackID != pointer.CandidatePackID {
		return CandidateRuntimePackPointer{}, fmt.Errorf("mission store candidate runtime pack pointer candidate_pack_id %q does not match hot-update gate candidate_pack_id %q", pointer.CandidatePackID, gate.CandidatePackID)
	}
	if _, err := LoadRuntimePackRecord(root, pointer.CandidatePackID); err != nil {
		return CandidateRuntimePackPointer{}, fmt.Errorf("mission store candidate runtime pack pointer candidate_pack_id %q: %w", pointer.CandidatePackID, err)
	}
	return pointer, nil
}

func ResolveCandidateRuntimePackRecord(root string) (RuntimePackRecord, error) {
	pointer, err := LoadCandidateRuntimePackPointer(root)
	if err != nil {
		return RuntimePackRecord{}, err
	}
	return LoadRuntimePackRecord(root, pointer.CandidatePackID)
}

func buildHotUpdateGateRecordFromCandidate(root string, hotUpdateID string, candidatePackID string, createdBy string, requestedAt time.Time) (HotUpdateGateRecord, error) {
	candidate, err := LoadRuntimePackRecord(root, candidatePackID)
	if err != nil {
		return HotUpdateGateRecord{}, err
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return HotUpdateGateRecord{}, err
	}
	record := HotUpdateGateRecord{
		HotUpdateID:              hotUpdateID,
		Objective:                fmt.Sprintf("operator requested hot-update gate for candidate %s", candidate.PackID),
		CandidatePackID:          candidate.PackID,
		PreviousActivePackID:     activePointer.ActivePackID,
		RollbackTargetPackID:     candidate.RollbackTargetPackID,
		TargetSurfaces:           append([]string(nil), candidate.MutableSurfaces...),
		SurfaceClasses:           append([]string(nil), candidate.SurfaceClasses...),
		ReloadMode:               deriveHotUpdateReloadMode(candidate.MutableSurfaces),
		CompatibilityContractRef: candidate.CompatibilityContractRef,
		PreparedAt:               requestedAt.UTC(),
		PhaseUpdatedAt:           requestedAt.UTC(),
		PhaseUpdatedBy:           strings.TrimSpace(createdBy),
		State:                    HotUpdateGateStatePrepared,
		Decision:                 HotUpdateGateDecisionKeepStaged,
		FailureReason:            "",
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, err
	}
	return record, nil
}

func loadHotUpdateGateRecordFile(path string) (HotUpdateGateRecord, error) {
	var record HotUpdateGateRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return HotUpdateGateRecord{}, err
	}
	record = NormalizeHotUpdateGateRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, err
	}
	return record, nil
}

func normalizeHotUpdateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateHotUpdateIdentifierField(surface, fieldName, value string) error {
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

func isValidHotUpdateReloadMode(mode HotUpdateReloadMode) bool {
	switch mode {
	case HotUpdateReloadModeSoftReload,
		HotUpdateReloadModeSkillReload,
		HotUpdateReloadModeExtensionReload,
		HotUpdateReloadModePackReload,
		HotUpdateReloadModeCanaryReload,
		HotUpdateReloadModeProcessRestartSwap,
		HotUpdateReloadModeColdRestart:
		return true
	default:
		return false
	}
}

func isValidHotUpdateGateState(state HotUpdateGateState) bool {
	switch state {
	case HotUpdateGateStateDraft,
		HotUpdateGateStatePrepared,
		HotUpdateGateStateValidated,
		HotUpdateGateStateStaged,
		HotUpdateGateStateQuiescing,
		HotUpdateGateStateReloading,
		HotUpdateGateStateReloadApplyInProgress,
		HotUpdateGateStateReloadApplyRecoveryNeeded,
		HotUpdateGateStateReloadApplySucceeded,
		HotUpdateGateStateReloadApplyFailed,
		HotUpdateGateStateSmokeTesting,
		HotUpdateGateStateCanarying,
		HotUpdateGateStateCommitted,
		HotUpdateGateStateRolledBack,
		HotUpdateGateStateRejected,
		HotUpdateGateStateFailed,
		HotUpdateGateStateAborted:
		return true
	default:
		return false
	}
}

func isValidHotUpdateGateDecision(decision HotUpdateGateDecision) bool {
	switch decision {
	case HotUpdateGateDecisionKeepStaged,
		HotUpdateGateDecisionDiscard,
		HotUpdateGateDecisionBlock,
		HotUpdateGateDecisionApplyHotUpdate,
		HotUpdateGateDecisionApplyCanary,
		HotUpdateGateDecisionRequireApproval,
		HotUpdateGateDecisionRequireColdRestart,
		HotUpdateGateDecisionRollback:
		return true
	default:
		return false
	}
}

func hotUpdateGatePointerUpdateRecordRef(hotUpdateID string) string {
	return "hot_update:" + strings.TrimSpace(hotUpdateID)
}

func hotUpdateGateTerminalFailureReason(reason string) string {
	return "operator_terminal_failure: " + strings.TrimSpace(reason)
}

func isValidHotUpdateGatePhaseStartState(state HotUpdateGateState) bool {
	switch state {
	case HotUpdateGateStatePrepared, HotUpdateGateStateValidated, HotUpdateGateStateStaged:
		return true
	default:
		return false
	}
}

func isValidHotUpdateGateAdjacentTransition(current HotUpdateGateState, next HotUpdateGateState) bool {
	switch current {
	case HotUpdateGateStatePrepared:
		return next == HotUpdateGateStateValidated
	case HotUpdateGateStateValidated:
		return next == HotUpdateGateStateStaged
	default:
		return false
	}
}

func deriveHotUpdateReloadMode(targetSurfaces []string) HotUpdateReloadMode {
	if len(targetSurfaces) == 1 {
		switch strings.TrimSpace(targetSurfaces[0]) {
		case "skills":
			return HotUpdateReloadModeSkillReload
		case "extensions":
			return HotUpdateReloadModeExtensionReload
		}
	}
	return HotUpdateReloadModeSoftReload
}

func validateHotUpdateGateDerivedLinkage(root string, record HotUpdateGateRecord) error {
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.PreviousActivePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate previous_active_pack_id %q: %w", record.PreviousActivePackID, err)
	}
	if strings.TrimSpace(record.RollbackTargetPackID) == "" {
		return fmt.Errorf("mission store hot-update gate rollback_target_pack_id is required")
	}
	if _, err := LoadRuntimePackRecord(root, record.RollbackTargetPackID); err != nil {
		return fmt.Errorf("mission store hot-update gate rollback_target_pack_id %q: %w", record.RollbackTargetPackID, err)
	}
	if err := requireHotUpdateGateExtensionPermissionAdmission(root, record); err != nil {
		return err
	}
	return nil
}

func requireHotUpdateGateExtensionPermissionAdmission(root string, record HotUpdateGateRecord) error {
	if err := requireHotUpdateGateActivePointerBaseline(root, record); err != nil {
		return err
	}
	previousPack, err := LoadRuntimePackRecord(root, record.PreviousActivePackID)
	if err != nil {
		return fmt.Errorf("mission store hot-update gate previous_active_pack_id %q extension assessment: %w", record.PreviousActivePackID, err)
	}
	candidatePack, err := LoadRuntimePackRecord(root, record.CandidatePackID)
	if err != nil {
		return fmt.Errorf("mission store hot-update gate candidate_pack_id %q extension assessment: %w", record.CandidatePackID, err)
	}
	if previousPack.ExtensionPackRef == candidatePack.ExtensionPackRef {
		return nil
	}
	assessment, err := AssessRuntimeExtensionPermissionWidening(root, previousPack.ExtensionPackRef, candidatePack.ExtensionPackRef)
	if err != nil {
		return fmt.Errorf("mission store hot-update gate %q extension permission assessment: %w", record.HotUpdateID, err)
	}
	if assessment.State != RuntimeExtensionPermissionAssessmentStateAllowed {
		return fmt.Errorf("mission store hot-update gate %q extension permission assessment blocked: %s", record.HotUpdateID, hotUpdateGateExtensionBlockerSummary(assessment.Blockers))
	}
	return nil
}

func requireHotUpdateGateActivePointerBaseline(root string, record HotUpdateGateRecord) error {
	if record.State != HotUpdateGateStatePrepared {
		return nil
	}
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		if errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
			return nil
		}
		return fmt.Errorf("mission store hot-update gate %q active runtime pack pointer extension assessment: %w", record.HotUpdateID, err)
	}
	if pointer.ActivePackID == record.CandidatePackID {
		return nil
	}
	if pointer.ActivePackID != record.PreviousActivePackID {
		return fmt.Errorf("mission store hot-update gate %q previous_active_pack_id %q does not match active runtime pack pointer active_pack_id %q", record.HotUpdateID, record.PreviousActivePackID, pointer.ActivePackID)
	}
	return nil
}

func hotUpdateGateExtensionBlockerSummary(blockers []RuntimeExtensionPermissionBlocker) string {
	if len(blockers) == 0 {
		return "extension permission assessment is blocked"
	}
	parts := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		parts = append(parts, fmt.Sprintf("%s: %s", blocker.Code, blocker.Reason))
	}
	return strings.Join(parts, "; ")
}

func hotUpdateCanaryGateExecutionReadinessAssessmentFromGate(record HotUpdateGateRecord) HotUpdateCanaryGateExecutionReadinessAssessment {
	return HotUpdateCanaryGateExecutionReadinessAssessment{
		HotUpdateID:             record.HotUpdateID,
		CanaryRef:               record.CanaryRef,
		ApprovalRef:             record.ApprovalRef,
		OwnerApprovalDecisionID: record.ApprovalRef,
		BaselinePackID:          record.PreviousActivePackID,
		CandidatePackID:         record.CandidatePackID,
	}
}

func hotUpdateCanaryGateExecutionReadinessAssessmentFromGateAndAuthority(record HotUpdateGateRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) HotUpdateCanaryGateExecutionReadinessAssessment {
	assessment := hotUpdateCanaryGateExecutionReadinessAssessmentFromGate(record)
	assessment.CanarySatisfactionAuthorityID = authority.CanarySatisfactionAuthorityID
	assessment.ResultID = authority.ResultID
	assessment.RunID = authority.RunID
	assessment.CandidateID = authority.CandidateID
	assessment.EvalSuiteID = authority.EvalSuiteID
	assessment.PromotionPolicyID = authority.PromotionPolicyID
	assessment.BaselinePackID = authority.BaselinePackID
	assessment.CandidatePackID = authority.CandidatePackID
	assessment.ExpectedEligibilityState = authority.EligibilityState
	assessment.SatisfactionState = authority.SatisfactionState
	assessment.OwnerApprovalRequired = authority.OwnerApprovalRequired
	return assessment
}

func hotUpdateCanaryGateExecutionReadinessAssessmentError(assessment HotUpdateCanaryGateExecutionReadinessAssessment, err error) HotUpdateCanaryGateExecutionReadinessAssessment {
	assessment.State = "invalid"
	assessment.Ready = false
	if err != nil {
		assessment.Error = err.Error()
	}
	return assessment
}

func requireHotUpdateCanaryGateExecutionReadiness(root string, record HotUpdateGateRecord) error {
	if strings.TrimSpace(record.CanaryRef) == "" {
		return nil
	}
	assessment, err := AssessHotUpdateCanaryGateExecutionReadiness(root, record.HotUpdateID)
	if err != nil {
		return err
	}
	if !assessment.Ready {
		return fmt.Errorf("mission store hot-update gate %q canary execution readiness is not ready: %s", record.HotUpdateID, assessment.Error)
	}
	return nil
}

func validateHotUpdateGateCanaryAuthorityMatchesGate(root string, record HotUpdateGateRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	if record.CanaryRef != authority.CanarySatisfactionAuthorityID {
		return fmt.Errorf("mission store hot-update gate %q canary_ref %q does not match canary satisfaction authority %q", record.HotUpdateID, record.CanaryRef, authority.CanarySatisfactionAuthorityID)
	}
	if record.CandidatePackID != authority.CandidatePackID {
		return fmt.Errorf("mission store hot-update gate %q candidate_pack_id %q does not match canary satisfaction authority candidate_pack_id %q", record.HotUpdateID, record.CandidatePackID, authority.CandidatePackID)
	}
	if record.PreviousActivePackID != authority.BaselinePackID {
		return fmt.Errorf("mission store hot-update gate %q previous_active_pack_id %q does not match canary satisfaction authority baseline_pack_id %q", record.HotUpdateID, record.PreviousActivePackID, authority.BaselinePackID)
	}
	candidatePack, err := LoadRuntimePackRecord(root, authority.CandidatePackID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q: %w", authority.CandidatePackID, err)
	}
	if record.RollbackTargetPackID != candidatePack.RollbackTargetPackID {
		return fmt.Errorf("mission store hot-update gate %q rollback_target_pack_id %q does not match candidate rollback_target_pack_id %q", record.HotUpdateID, record.RollbackTargetPackID, candidatePack.RollbackTargetPackID)
	}
	return nil
}

func validateHotUpdateGateCanaryOwnerApprovalBranch(root string, authority HotUpdateCanarySatisfactionAuthorityRecord, ownerApprovalDecisionID string) (string, string, HotUpdateCanarySatisfactionState, error) {
	ownerApprovalDecisionID = strings.TrimSpace(ownerApprovalDecisionID)
	switch authority.State {
	case HotUpdateCanarySatisfactionAuthorityStateAuthorized:
		if authority.OwnerApprovalRequired {
			return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q owner_approval_required must be false for authorized gate creation", authority.CanarySatisfactionAuthorityID)
		}
		if authority.SatisfactionState != HotUpdateCanarySatisfactionStateSatisfied {
			return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q satisfaction_state must be %q for authorized gate creation", authority.CanarySatisfactionAuthorityID, HotUpdateCanarySatisfactionStateSatisfied)
		}
		if ownerApprovalDecisionID != "" {
			return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q does not accept owner approval decision for no-owner-approval gate creation", authority.CanarySatisfactionAuthorityID)
		}
		return "", CandidatePromotionEligibilityStateCanaryRequired, HotUpdateCanarySatisfactionStateSatisfied, nil
	case HotUpdateCanarySatisfactionAuthorityStateWaitingOwnerApproval:
		if !authority.OwnerApprovalRequired {
			return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q owner_approval_required must be true for owner-approved gate creation", authority.CanarySatisfactionAuthorityID)
		}
		if authority.SatisfactionState != HotUpdateCanarySatisfactionStateWaitingOwnerApproval {
			return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q satisfaction_state must be %q for owner-approved gate creation", authority.CanarySatisfactionAuthorityID, HotUpdateCanarySatisfactionStateWaitingOwnerApproval)
		}
		decisionRef := NormalizeHotUpdateOwnerApprovalDecisionRef(HotUpdateOwnerApprovalDecisionRef{OwnerApprovalDecisionID: ownerApprovalDecisionID})
		if err := ValidateHotUpdateOwnerApprovalDecisionRef(decisionRef); err != nil {
			return "", "", "", err
		}
		decision, err := LoadHotUpdateOwnerApprovalDecisionRecord(root, decisionRef.OwnerApprovalDecisionID)
		if err != nil {
			return "", "", "", err
		}
		if decision.Decision != HotUpdateOwnerApprovalDecisionGranted {
			return "", "", "", fmt.Errorf("mission store hot-update owner approval decision %q decision %q does not permit hot-update canary gate creation", decision.OwnerApprovalDecisionID, decision.Decision)
		}
		if err := validateHotUpdateGateOwnerApprovalDecisionMatchesAuthority(decision, authority); err != nil {
			return "", "", "", err
		}
		return decision.OwnerApprovalDecisionID, CandidatePromotionEligibilityStateCanaryAndOwnerApprovalRequired, HotUpdateCanarySatisfactionStateWaitingOwnerApproval, nil
	default:
		return "", "", "", fmt.Errorf("mission store hot-update canary satisfaction authority %q state %q does not permit hot-update canary gate creation", authority.CanarySatisfactionAuthorityID, authority.State)
	}
}

func validateHotUpdateGateCanaryAuthorityFreshness(root string, authority HotUpdateCanarySatisfactionAuthorityRecord, expectedSatisfaction HotUpdateCanarySatisfactionState, expectedEligibility string) error {
	requirement, err := LoadHotUpdateCanaryRequirementRecord(root, authority.CanaryRequirementID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate canary_requirement_id %q: %w", authority.CanaryRequirementID, err)
	}
	evidence, err := LoadHotUpdateCanaryEvidenceRecord(root, authority.SelectedCanaryEvidenceID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate selected_canary_evidence_id %q: %w", authority.SelectedCanaryEvidenceID, err)
	}
	assessment, err := AssessHotUpdateCanarySatisfaction(root, authority.CanaryRequirementID)
	if err != nil {
		return err
	}
	if assessment.State != "configured" {
		return fmt.Errorf("mission store hot-update canary gate requires configured canary satisfaction assessment, found state %q: %s", assessment.State, assessment.Error)
	}
	if assessment.SatisfactionState != expectedSatisfaction {
		return fmt.Errorf("mission store hot-update canary gate requires canary satisfaction_state %q, found %q", expectedSatisfaction, assessment.SatisfactionState)
	}
	if err := validateHotUpdateCanarySatisfactionAuthorityMatchesRequirementEvidenceAssessment(authority, requirement, evidence, assessment); err != nil {
		return err
	}

	status, err := EvaluateCandidateResultPromotionEligibility(root, authority.ResultID)
	if err != nil {
		return err
	}
	if status.State != expectedEligibility {
		return fmt.Errorf("mission store candidate result %q promotion eligibility state %q does not permit hot-update canary gate creation", authority.ResultID, status.State)
	}
	if status.RunID != authority.RunID ||
		status.CandidateID != authority.CandidateID ||
		status.EvalSuiteID != authority.EvalSuiteID ||
		status.PromotionPolicyID != authority.PromotionPolicyID ||
		status.BaselinePackID != authority.BaselinePackID ||
		status.CandidatePackID != authority.CandidatePackID ||
		status.OwnerApprovalRequired != authority.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match derived promotion eligibility status", authority.CanarySatisfactionAuthorityID)
	}
	return nil
}

func validateHotUpdateGateCanaryAuthorityRuntimeReadiness(root string, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	candidatePack, err := LoadRuntimePackRecord(root, authority.CandidatePackID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q: %w", authority.CandidatePackID, err)
	}
	if candidatePack.RollbackTargetPackID == "" {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q rollback_target_pack_id is required", candidatePack.PackID)
	}
	if _, err := LoadRuntimePackRecord(root, candidatePack.RollbackTargetPackID); err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q rollback_target_pack_id %q: %w", candidatePack.PackID, candidatePack.RollbackTargetPackID, err)
	}

	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return err
	}
	if activePointer.ActivePackID != authority.BaselinePackID {
		return fmt.Errorf(
			"mission store hot-update canary satisfaction authority %q requires active runtime pack pointer active_pack_id %q, found %q",
			authority.CanarySatisfactionAuthorityID,
			authority.BaselinePackID,
			activePointer.ActivePackID,
		)
	}
	if _, err := LoadLastKnownGoodRuntimePackPointer(root); err != nil && !errors.Is(err, ErrLastKnownGoodRuntimePackPointerNotFound) {
		return err
	}
	return nil
}

func validateHotUpdateGateCanaryExecutionRuntimeReadiness(root string, record HotUpdateGateRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	candidatePack, err := LoadRuntimePackRecord(root, authority.CandidatePackID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q: %w", authority.CandidatePackID, err)
	}
	if candidatePack.RollbackTargetPackID == "" {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q rollback_target_pack_id is required", candidatePack.PackID)
	}
	if record.RollbackTargetPackID != candidatePack.RollbackTargetPackID {
		return fmt.Errorf(
			"mission store hot-update gate %q rollback_target_pack_id %q does not match candidate rollback_target_pack_id %q",
			record.HotUpdateID,
			record.RollbackTargetPackID,
			candidatePack.RollbackTargetPackID,
		)
	}
	if _, err := LoadRuntimePackRecord(root, candidatePack.RollbackTargetPackID); err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q rollback_target_pack_id %q: %w", candidatePack.PackID, candidatePack.RollbackTargetPackID, err)
	}

	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return err
	}
	updateRecordRef := hotUpdateGatePointerUpdateRecordRef(record.HotUpdateID)
	switch record.State {
	case HotUpdateGateStatePrepared, HotUpdateGateStateValidated, HotUpdateGateStateStaged:
		if activePointer.ActivePackID != authority.BaselinePackID {
			return fmt.Errorf(
				"mission store hot-update canary satisfaction authority %q requires active runtime pack pointer active_pack_id %q before execution, found %q",
				authority.CanarySatisfactionAuthorityID,
				authority.BaselinePackID,
				activePointer.ActivePackID,
			)
		}
	case HotUpdateGateStateReloading, HotUpdateGateStateReloadApplyInProgress, HotUpdateGateStateReloadApplyRecoveryNeeded, HotUpdateGateStateReloadApplySucceeded:
		if activePointer.ActivePackID != record.CandidatePackID {
			return fmt.Errorf(
				"mission store hot-update gate %q requires active runtime pack pointer active_pack_id %q after pointer switch, found %q",
				record.HotUpdateID,
				record.CandidatePackID,
				activePointer.ActivePackID,
			)
		}
		if activePointer.UpdateRecordRef != updateRecordRef {
			return fmt.Errorf(
				"mission store hot-update gate %q requires active runtime pack pointer update_record_ref %q after pointer switch, found %q",
				record.HotUpdateID,
				updateRecordRef,
				activePointer.UpdateRecordRef,
			)
		}
		if activePointer.PreviousActivePackID != record.PreviousActivePackID {
			return fmt.Errorf(
				"mission store hot-update gate %q requires active runtime pack pointer previous_active_pack_id %q after pointer switch, found %q",
				record.HotUpdateID,
				record.PreviousActivePackID,
				activePointer.PreviousActivePackID,
			)
		}
	default:
		return fmt.Errorf("mission store hot-update gate %q state %q does not permit canary execution readiness", record.HotUpdateID, record.State)
	}
	if _, err := LoadLastKnownGoodRuntimePackPointer(root); err != nil && !errors.Is(err, ErrLastKnownGoodRuntimePackPointerNotFound) {
		return err
	}
	return nil
}

func validateHotUpdateGateCanaryAuthoritySource(root string, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	result, err := LoadCandidateResultRecord(root, authority.ResultID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate result_id %q: %w", authority.ResultID, err)
	}
	if result.RunID != authority.RunID ||
		result.CandidateID != authority.CandidateID ||
		result.EvalSuiteID != authority.EvalSuiteID ||
		result.PromotionPolicyID != authority.PromotionPolicyID ||
		result.BaselinePackID != authority.BaselinePackID ||
		result.CandidatePackID != authority.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority %q does not match candidate result %q", authority.CanarySatisfactionAuthorityID, authority.ResultID)
	}
	run, err := LoadImprovementRunRecord(root, authority.RunID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate run_id %q: %w", authority.RunID, err)
	}
	if err := validateHotUpdateGateCanaryAuthorityRun(authority, run); err != nil {
		return err
	}
	candidate, err := LoadImprovementCandidateRecord(root, authority.CandidateID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_id %q: %w", authority.CandidateID, err)
	}
	if err := validateHotUpdateGateCanaryAuthorityCandidate(authority, candidate); err != nil {
		return err
	}
	evalSuite, err := LoadEvalSuiteRecord(root, authority.EvalSuiteID)
	if err != nil {
		return fmt.Errorf("mission store hot-update canary gate eval_suite_id %q: %w", authority.EvalSuiteID, err)
	}
	if err := validateHotUpdateGateCanaryAuthorityEvalSuite(authority, evalSuite); err != nil {
		return err
	}
	if _, err := LoadPromotionPolicyRecord(root, authority.PromotionPolicyID); err != nil {
		return fmt.Errorf("mission store hot-update canary gate promotion_policy_id %q: %w", authority.PromotionPolicyID, err)
	}
	if _, err := LoadRuntimePackRecord(root, authority.BaselinePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary gate baseline_pack_id %q: %w", authority.BaselinePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, authority.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update canary gate candidate_pack_id %q: %w", authority.CandidatePackID, err)
	}
	return nil
}

func validateHotUpdateGateCanaryAuthorityRun(authority HotUpdateCanarySatisfactionAuthorityRecord, run ImprovementRunRecord) error {
	if run.RunID != authority.RunID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority run_id %q does not match improvement run run_id %q", authority.RunID, run.RunID)
	}
	if run.CandidateID != authority.CandidateID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_id %q does not match improvement run candidate_id %q", authority.CandidateID, run.CandidateID)
	}
	if run.EvalSuiteID != authority.EvalSuiteID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority eval_suite_id %q does not match improvement run eval_suite_id %q", authority.EvalSuiteID, run.EvalSuiteID)
	}
	if run.BaselinePackID != authority.BaselinePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority baseline_pack_id %q does not match improvement run baseline_pack_id %q", authority.BaselinePackID, run.BaselinePackID)
	}
	if run.CandidatePackID != authority.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_pack_id %q does not match improvement run candidate_pack_id %q", authority.CandidatePackID, run.CandidatePackID)
	}
	return nil
}

func validateHotUpdateGateCanaryAuthorityCandidate(authority HotUpdateCanarySatisfactionAuthorityRecord, candidate ImprovementCandidateRecord) error {
	if candidate.CandidateID != authority.CandidateID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_id %q does not match improvement candidate candidate_id %q", authority.CandidateID, candidate.CandidateID)
	}
	if candidate.BaselinePackID != authority.BaselinePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority baseline_pack_id %q does not match improvement candidate baseline_pack_id %q", authority.BaselinePackID, candidate.BaselinePackID)
	}
	if candidate.CandidatePackID != authority.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_pack_id %q does not match improvement candidate candidate_pack_id %q", authority.CandidatePackID, candidate.CandidatePackID)
	}
	return nil
}

func validateHotUpdateGateCanaryAuthorityEvalSuite(authority HotUpdateCanarySatisfactionAuthorityRecord, evalSuite EvalSuiteRecord) error {
	if evalSuite.EvalSuiteID != authority.EvalSuiteID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority eval_suite_id %q does not match eval-suite eval_suite_id %q", authority.EvalSuiteID, evalSuite.EvalSuiteID)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != authority.CandidateID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_id %q does not match eval-suite candidate_id %q", authority.CandidateID, evalSuite.CandidateID)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != authority.BaselinePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority baseline_pack_id %q does not match eval-suite baseline_pack_id %q", authority.BaselinePackID, evalSuite.BaselinePackID)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != authority.CandidatePackID {
		return fmt.Errorf("mission store hot-update canary satisfaction authority candidate_pack_id %q does not match eval-suite candidate_pack_id %q", authority.CandidatePackID, evalSuite.CandidatePackID)
	}
	return nil
}

func validateHotUpdateGateOwnerApprovalDecisionMatchesAuthority(decision HotUpdateOwnerApprovalDecisionRecord, authority HotUpdateCanarySatisfactionAuthorityRecord) error {
	if decision.CanarySatisfactionAuthorityID != authority.CanarySatisfactionAuthorityID ||
		decision.CanaryRequirementID != authority.CanaryRequirementID ||
		decision.SelectedCanaryEvidenceID != authority.SelectedCanaryEvidenceID ||
		decision.ResultID != authority.ResultID ||
		decision.RunID != authority.RunID ||
		decision.CandidateID != authority.CandidateID ||
		decision.EvalSuiteID != authority.EvalSuiteID ||
		decision.PromotionPolicyID != authority.PromotionPolicyID ||
		decision.BaselinePackID != authority.BaselinePackID ||
		decision.CandidatePackID != authority.CandidatePackID ||
		decision.AuthorityState != authority.State ||
		decision.SatisfactionState != authority.SatisfactionState ||
		decision.OwnerApprovalRequired != authority.OwnerApprovalRequired {
		return fmt.Errorf("mission store hot-update owner approval decision %q does not match canary satisfaction authority %q", decision.OwnerApprovalDecisionID, authority.CanarySatisfactionAuthorityID)
	}
	return nil
}

func validateHotUpdateGateExecutionLinkage(root string, record HotUpdateGateRecord) (ActiveRuntimePackPointer, error) {
	candidate, err := LoadRuntimePackRecord(root, record.CandidatePackID)
	if err != nil {
		return ActiveRuntimePackPointer{}, fmt.Errorf("mission store hot-update gate candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.PreviousActivePackID); err != nil {
		return ActiveRuntimePackPointer{}, fmt.Errorf("mission store hot-update gate previous_active_pack_id %q: %w", record.PreviousActivePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.RollbackTargetPackID); err != nil {
		return ActiveRuntimePackPointer{}, fmt.Errorf("mission store hot-update gate rollback_target_pack_id %q: %w", record.RollbackTargetPackID, err)
	}
	if candidate.RollbackTargetPackID != record.RollbackTargetPackID {
		return ActiveRuntimePackPointer{}, fmt.Errorf(
			"mission store hot-update gate %q rollback_target_pack_id %q does not match candidate rollback_target_pack_id %q",
			record.HotUpdateID,
			record.RollbackTargetPackID,
			candidate.RollbackTargetPackID,
		)
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return ActiveRuntimePackPointer{}, err
	}
	return activePointer, nil
}

func executeHotUpdateGateReloadApplyWithConvergence(root string, hotUpdateID string, updatedBy string, updatedAt time.Time, converge func(string, HotUpdateGateRecord) error) (HotUpdateGateRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate phase_updated_at is required")
	}
	if converge == nil {
		return HotUpdateGateRecord{}, false, fmt.Errorf("mission store hot-update gate convergence function is required")
	}

	record, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	switch record.State {
	case HotUpdateGateStateReloadApplySucceeded:
		if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		if err := requireHotUpdateSmokeReadiness(root, record); err != nil {
			return HotUpdateGateRecord{}, false, err
		}
		return record, false, nil
	case HotUpdateGateStateReloadApplyFailed:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q does not permit reload/apply retry",
			ref.HotUpdateID,
			record.State,
		)
	case HotUpdateGateStateReloadApplyInProgress:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q requires recovery before retry",
			ref.HotUpdateID,
			record.State,
		)
	case HotUpdateGateStateReloading,
		HotUpdateGateStateReloadApplyRecoveryNeeded:
	default:
		return HotUpdateGateRecord{}, false, fmt.Errorf(
			"mission store hot-update gate %q state %q does not permit reload/apply execution",
			ref.HotUpdateID,
			record.State,
		)
	}

	if err := requireHotUpdateCanaryGateExecutionReadiness(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := requireHotUpdateSmokeReadiness(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	record.State = HotUpdateGateStateReloadApplyInProgress
	record.FailureReason = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeHotUpdateGateRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}

	if err := converge(root, record); err != nil {
		record.State = HotUpdateGateStateReloadApplyFailed
		record.FailureReason = err.Error()
		record.PhaseUpdatedAt = updatedAt
		record.PhaseUpdatedBy = updatedBy
		record = NormalizeHotUpdateGateRecord(record)
		if validationErr := ValidateHotUpdateGateRecord(record); validationErr != nil {
			return HotUpdateGateRecord{}, false, validationErr
		}
		if linkageErr := validateHotUpdateGateDerivedLinkage(root, record); linkageErr != nil {
			return HotUpdateGateRecord{}, false, linkageErr
		}
		if writeErr := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); writeErr != nil {
			return HotUpdateGateRecord{}, false, writeErr
		}
		record, loadErr := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
		if loadErr != nil {
			return HotUpdateGateRecord{}, false, loadErr
		}
		return record, true, err
	}

	record.State = HotUpdateGateStateReloadApplySucceeded
	record.FailureReason = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeHotUpdateGateRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := validateHotUpdateGateDerivedLinkage(root, record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	record, err = LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		return HotUpdateGateRecord{}, false, err
	}
	return record, true, nil
}

func validateHotUpdateGateReloadApplyLinkage(root string, record HotUpdateGateRecord) error {
	activePointer, err := validateHotUpdateGateExecutionLinkage(root, record)
	if err != nil {
		return err
	}
	updateRecordRef := hotUpdateGatePointerUpdateRecordRef(record.HotUpdateID)
	if activePointer.ActivePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store hot-update gate %q requires active runtime pack pointer active_pack_id %q before reload/apply, found %q",
			record.HotUpdateID,
			record.CandidatePackID,
			activePointer.ActivePackID,
		)
	}
	if activePointer.UpdateRecordRef != updateRecordRef {
		return fmt.Errorf(
			"mission store hot-update gate %q requires active runtime pack pointer update_record_ref %q before reload/apply, found %q",
			record.HotUpdateID,
			updateRecordRef,
			activePointer.UpdateRecordRef,
		)
	}
	if activePointer.PreviousActivePackID != record.PreviousActivePackID {
		return fmt.Errorf(
			"mission store hot-update gate %q requires active runtime pack pointer previous_active_pack_id %q before reload/apply, found %q",
			record.HotUpdateID,
			record.PreviousActivePackID,
			activePointer.PreviousActivePackID,
		)
	}
	return nil
}

func hotUpdateIDFromCandidatePromotionDecision(promotionDecisionID string) string {
	return "hot-update-" + strings.TrimSpace(promotionDecisionID)
}

func validateCandidatePromotionDecisionResultAuthority(decision CandidatePromotionDecisionRecord, result CandidateResultRecord) error {
	if result.ResultID != decision.ResultID {
		return fmt.Errorf("mission store candidate promotion decision result_id %q does not match candidate result result_id %q", decision.ResultID, result.ResultID)
	}
	if result.RunID != decision.RunID {
		return fmt.Errorf("mission store candidate promotion decision run_id %q does not match candidate result run_id %q", decision.RunID, result.RunID)
	}
	if result.CandidateID != decision.CandidateID {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q does not match candidate result candidate_id %q", decision.CandidateID, result.CandidateID)
	}
	if result.EvalSuiteID != decision.EvalSuiteID {
		return fmt.Errorf("mission store candidate promotion decision eval_suite_id %q does not match candidate result eval_suite_id %q", decision.EvalSuiteID, result.EvalSuiteID)
	}
	if result.PromotionPolicyID != decision.PromotionPolicyID {
		return fmt.Errorf("mission store candidate promotion decision promotion_policy_id %q does not match candidate result promotion_policy_id %q", decision.PromotionPolicyID, result.PromotionPolicyID)
	}
	if result.BaselinePackID != decision.BaselinePackID {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q does not match candidate result baseline_pack_id %q", decision.BaselinePackID, result.BaselinePackID)
	}
	if result.CandidatePackID != decision.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q does not match candidate result candidate_pack_id %q", decision.CandidatePackID, result.CandidatePackID)
	}
	return nil
}

func validateCandidatePromotionDecisionEligibilityAuthority(decision CandidatePromotionDecisionRecord, status CandidatePromotionEligibilityStatus) error {
	if status.ResultID != decision.ResultID ||
		status.RunID != decision.RunID ||
		status.CandidateID != decision.CandidateID ||
		status.EvalSuiteID != decision.EvalSuiteID ||
		status.PromotionPolicyID != decision.PromotionPolicyID ||
		status.BaselinePackID != decision.BaselinePackID ||
		status.CandidatePackID != decision.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision %q does not match derived promotion eligibility status", decision.PromotionDecisionID)
	}
	return nil
}

func validateCandidatePromotionDecisionRunAuthority(decision CandidatePromotionDecisionRecord, run ImprovementRunRecord) error {
	if run.RunID != decision.RunID {
		return fmt.Errorf("mission store candidate promotion decision run_id %q does not match improvement run run_id %q", decision.RunID, run.RunID)
	}
	if run.CandidateID != decision.CandidateID {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q does not match improvement run candidate_id %q", decision.CandidateID, run.CandidateID)
	}
	if run.EvalSuiteID != decision.EvalSuiteID {
		return fmt.Errorf("mission store candidate promotion decision eval_suite_id %q does not match improvement run eval_suite_id %q", decision.EvalSuiteID, run.EvalSuiteID)
	}
	if run.BaselinePackID != decision.BaselinePackID {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q does not match improvement run baseline_pack_id %q", decision.BaselinePackID, run.BaselinePackID)
	}
	if run.CandidatePackID != decision.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q does not match improvement run candidate_pack_id %q", decision.CandidatePackID, run.CandidatePackID)
	}
	return nil
}

func validateCandidatePromotionDecisionCandidateAuthority(decision CandidatePromotionDecisionRecord, candidate ImprovementCandidateRecord) error {
	if candidate.CandidateID != decision.CandidateID {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q does not match improvement candidate candidate_id %q", decision.CandidateID, candidate.CandidateID)
	}
	if candidate.BaselinePackID != decision.BaselinePackID {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q does not match improvement candidate baseline_pack_id %q", decision.BaselinePackID, candidate.BaselinePackID)
	}
	if candidate.CandidatePackID != decision.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q does not match improvement candidate candidate_pack_id %q", decision.CandidatePackID, candidate.CandidatePackID)
	}
	return nil
}

func validateCandidatePromotionDecisionEvalSuiteAuthority(decision CandidatePromotionDecisionRecord, evalSuite EvalSuiteRecord) error {
	if evalSuite.EvalSuiteID != decision.EvalSuiteID {
		return fmt.Errorf("mission store candidate promotion decision eval_suite_id %q does not match eval-suite eval_suite_id %q", decision.EvalSuiteID, evalSuite.EvalSuiteID)
	}
	if evalSuite.CandidateID != "" && evalSuite.CandidateID != decision.CandidateID {
		return fmt.Errorf("mission store candidate promotion decision candidate_id %q does not match eval-suite candidate_id %q", decision.CandidateID, evalSuite.CandidateID)
	}
	if evalSuite.BaselinePackID != "" && evalSuite.BaselinePackID != decision.BaselinePackID {
		return fmt.Errorf("mission store candidate promotion decision baseline_pack_id %q does not match eval-suite baseline_pack_id %q", decision.BaselinePackID, evalSuite.BaselinePackID)
	}
	if evalSuite.CandidatePackID != "" && evalSuite.CandidatePackID != decision.CandidatePackID {
		return fmt.Errorf("mission store candidate promotion decision candidate_pack_id %q does not match eval-suite candidate_pack_id %q", decision.CandidatePackID, evalSuite.CandidatePackID)
	}
	return nil
}

// hotUpdateGateRestartStyleConvergence models the bounded local reload/apply
// convergence step available today: re-resolve the already-switched active
// runtime-pack pointer, verify that the candidate pack is still active, and
// load the active pack's local component metadata.
func hotUpdateGateRestartStyleConvergence(root string, record HotUpdateGateRecord) error {
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return err
	}
	if activePointer.ActivePackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store hot-update gate %q convergence requires active runtime pack pointer active_pack_id %q, found %q",
			record.HotUpdateID,
			record.CandidatePackID,
			activePointer.ActivePackID,
		)
	}
	if activePointer.UpdateRecordRef != hotUpdateGatePointerUpdateRecordRef(record.HotUpdateID) {
		return fmt.Errorf(
			"mission store hot-update gate %q convergence requires active runtime pack pointer update_record_ref %q, found %q",
			record.HotUpdateID,
			hotUpdateGatePointerUpdateRecordRef(record.HotUpdateID),
			activePointer.UpdateRecordRef,
		)
	}
	resolved, _, err := ResolveActiveRuntimePackComponents(root)
	if err != nil {
		return err
	}
	if resolved.PackID != record.CandidatePackID {
		return fmt.Errorf(
			"mission store hot-update gate %q convergence resolved active pack %q, want %q",
			record.HotUpdateID,
			resolved.PackID,
			record.CandidatePackID,
		)
	}
	return nil
}
