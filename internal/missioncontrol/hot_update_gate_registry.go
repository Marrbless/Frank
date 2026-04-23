package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	if err := validateHotUpdateGateReloadApplyLinkage(root, record); err != nil {
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

// hotUpdateGateRestartStyleConvergence models the smallest bounded reload/apply
// convergence step available today: re-resolve the already-switched active
// runtime-pack pointer and verify that the candidate pack is still active.
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
	resolved, err := ResolveActiveRuntimePackRecord(root)
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
