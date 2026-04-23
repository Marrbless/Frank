package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type RollbackApplyRef struct {
	ApplyID string `json:"apply_id"`
}

type RollbackApplyPhase string

const (
	RollbackApplyPhaseRecorded                     RollbackApplyPhase = "recorded"
	RollbackApplyPhaseValidated                    RollbackApplyPhase = "validated"
	RollbackApplyPhaseReadyToApply                 RollbackApplyPhase = "ready_to_apply"
	RollbackApplyPhasePointerSwitchedReloadPending RollbackApplyPhase = "pointer_switched_reload_pending"
	RollbackApplyPhaseReloadApplyInProgress        RollbackApplyPhase = "reload_apply_in_progress"
	RollbackApplyPhaseReloadApplyRecoveryNeeded    RollbackApplyPhase = "reload_apply_recovery_needed"
	RollbackApplyPhaseReloadApplySucceeded         RollbackApplyPhase = "reload_apply_succeeded"
	RollbackApplyPhaseReloadApplyFailed            RollbackApplyPhase = "reload_apply_failed"
)

type RollbackApplyActivationState string

const (
	RollbackApplyActivationStateUnchanged RollbackApplyActivationState = "unchanged"
)

type RollbackApplyRecord struct {
	RecordVersion   int                          `json:"record_version"`
	ApplyID         string                       `json:"apply_id"`
	RollbackID      string                       `json:"rollback_id"`
	Phase           RollbackApplyPhase           `json:"phase"`
	ActivationState RollbackApplyActivationState `json:"activation_state"`
	ExecutionError  string                       `json:"execution_error,omitempty"`
	RequestedAt     time.Time                    `json:"requested_at"`
	CreatedAt       time.Time                    `json:"created_at"`
	CreatedBy       string                       `json:"created_by"`
	PhaseUpdatedAt  time.Time                    `json:"phase_updated_at,omitempty"`
	PhaseUpdatedBy  string                       `json:"phase_updated_by,omitempty"`
}

var ErrRollbackApplyRecordNotFound = errors.New("mission store rollback apply record not found")

func StoreRollbackAppliesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "rollback_applies")
}

func StoreRollbackApplyPath(root, applyID string) string {
	return filepath.Join(StoreRollbackAppliesDir(root), strings.TrimSpace(applyID)+".json")
}

func NormalizeRollbackApplyRef(ref RollbackApplyRef) RollbackApplyRef {
	ref.ApplyID = strings.TrimSpace(ref.ApplyID)
	return ref
}

func NormalizeRollbackApplyRecord(record RollbackApplyRecord) RollbackApplyRecord {
	record.ApplyID = strings.TrimSpace(record.ApplyID)
	record.RollbackID = strings.TrimSpace(record.RollbackID)
	record.Phase = RollbackApplyPhase(strings.TrimSpace(string(record.Phase)))
	record.ActivationState = RollbackApplyActivationState(strings.TrimSpace(string(record.ActivationState)))
	record.ExecutionError = strings.TrimSpace(record.ExecutionError)
	record.RequestedAt = record.RequestedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	record.PhaseUpdatedAt = record.PhaseUpdatedAt.UTC()
	record.PhaseUpdatedBy = strings.TrimSpace(record.PhaseUpdatedBy)
	if record.PhaseUpdatedAt.IsZero() {
		record.PhaseUpdatedAt = record.CreatedAt
	}
	if record.PhaseUpdatedBy == "" {
		record.PhaseUpdatedBy = record.CreatedBy
	}
	return record
}

func RollbackApplyRollbackRef(record RollbackApplyRecord) RollbackRef {
	return RollbackRef{RollbackID: strings.TrimSpace(record.RollbackID)}
}

func ValidateRollbackApplyRef(ref RollbackApplyRef) error {
	return validateRollbackApplyIdentifierField("rollback apply ref", "apply_id", ref.ApplyID)
}

func ValidateRollbackApplyRecord(record RollbackApplyRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store rollback apply record_version must be positive")
	}
	if err := ValidateRollbackApplyRef(RollbackApplyRef{ApplyID: record.ApplyID}); err != nil {
		return err
	}
	if err := ValidateRollbackRef(RollbackApplyRollbackRef(record)); err != nil {
		return err
	}
	switch record.Phase {
	case RollbackApplyPhaseRecorded,
		RollbackApplyPhaseValidated,
		RollbackApplyPhaseReadyToApply,
		RollbackApplyPhasePointerSwitchedReloadPending,
		RollbackApplyPhaseReloadApplyInProgress,
		RollbackApplyPhaseReloadApplyRecoveryNeeded,
		RollbackApplyPhaseReloadApplySucceeded,
		RollbackApplyPhaseReloadApplyFailed:
	default:
		return fmt.Errorf("mission store rollback apply phase %q is invalid", record.Phase)
	}
	if record.ActivationState != RollbackApplyActivationStateUnchanged {
		return fmt.Errorf("mission store rollback apply activation_state must remain %q", RollbackApplyActivationStateUnchanged)
	}
	if record.RequestedAt.IsZero() {
		return fmt.Errorf("mission store rollback apply requested_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store rollback apply created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store rollback apply created_by is required")
	}
	if record.PhaseUpdatedAt.IsZero() {
		return fmt.Errorf("mission store rollback apply phase_updated_at is required")
	}
	if record.PhaseUpdatedBy == "" {
		return fmt.Errorf("mission store rollback apply phase_updated_by is required")
	}
	if record.PhaseUpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store rollback apply phase_updated_at must not precede created_at")
	}
	if record.Phase == RollbackApplyPhaseReloadApplyFailed {
		if record.ExecutionError == "" {
			return fmt.Errorf("mission store rollback apply execution_error is required when phase is %q", RollbackApplyPhaseReloadApplyFailed)
		}
	} else if record.ExecutionError != "" {
		return fmt.Errorf("mission store rollback apply execution_error is only valid when phase is %q", RollbackApplyPhaseReloadApplyFailed)
	}
	return nil
}

func StoreRollbackApplyRecord(root string, record RollbackApplyRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeRollbackApplyRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return err
	}

	path := StoreRollbackApplyPath(root, record.ApplyID)
	if existing, err := loadRollbackApplyRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store rollback apply %q already exists", record.ApplyID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func CreateRollbackApplyRecordFromRollback(root string, applyID string, rollbackID string, createdBy string, requestedAt time.Time) (RollbackApplyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, err
	}
	rollback, err := LoadRollbackRecord(root, rollbackID)
	if err != nil {
		return RollbackApplyRecord{}, err
	}
	record := RollbackApplyRecord{
		ApplyID:         ref.ApplyID,
		RollbackID:      rollback.RollbackID,
		Phase:           RollbackApplyPhaseRecorded,
		ActivationState: RollbackApplyActivationStateUnchanged,
		RequestedAt:     requestedAt,
		CreatedAt:       requestedAt,
		CreatedBy:       createdBy,
		PhaseUpdatedAt:  requestedAt,
		PhaseUpdatedBy:  createdBy,
	}
	if err := StoreRollbackApplyRecord(root, record); err != nil {
		return RollbackApplyRecord{}, err
	}
	return LoadRollbackApplyRecord(root, record.ApplyID)
}

func EnsureRollbackApplyRecordFromRollback(root string, applyID string, rollbackID string, createdBy string, requestedAt time.Time) (RollbackApplyRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	applyRef := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(applyRef); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	rollbackRef := NormalizeRollbackRef(RollbackRef{RollbackID: rollbackID})
	if err := ValidateRollbackRef(rollbackRef); err != nil {
		return RollbackApplyRecord{}, false, err
	}

	existing, err := LoadRollbackApplyRecord(root, applyRef.ApplyID)
	if err == nil {
		if existing.RollbackID != rollbackRef.RollbackID {
			return RollbackApplyRecord{}, false, fmt.Errorf(
				"mission store rollback apply %q rollback_id %q does not match requested rollback_id %q",
				applyRef.ApplyID,
				existing.RollbackID,
				rollbackRef.RollbackID,
			)
		}
		return existing, false, nil
	}
	if !errors.Is(err, ErrRollbackApplyRecordNotFound) {
		return RollbackApplyRecord{}, false, err
	}

	record, err := CreateRollbackApplyRecordFromRollback(root, applyRef.ApplyID, rollbackRef.RollbackID, createdBy, requestedAt)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	return record, true, nil
}

func AdvanceRollbackApplyPhase(root string, applyID string, nextPhase RollbackApplyPhase, updatedBy string, updatedAt time.Time) (RollbackApplyRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_at is required")
	}
	nextPhase = RollbackApplyPhase(strings.TrimSpace(string(nextPhase)))

	record, err := LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}

	currentOrder := rollbackApplyPhaseOrder(record.Phase)
	nextOrder := rollbackApplyPhaseOrder(nextPhase)
	if nextOrder == 0 {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase %q is invalid", nextPhase)
	}
	if nextOrder == currentOrder {
		return record, false, nil
	}
	if nextOrder != currentOrder+1 {
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase transition %q -> %q is invalid",
			ref.ApplyID,
			record.Phase,
			nextPhase,
		)
	}

	record.Phase = nextPhase
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	record, err = LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	return record, true, nil
}

func ExecuteRollbackApplyPointerSwitch(root string, applyID string, updatedBy string, updatedAt time.Time) (RollbackApplyRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_at is required")
	}

	record, err := LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	rollback, err := LoadRollbackRecord(root, record.RollbackID)
	if err != nil {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply rollback_id %q: %w", record.RollbackID, err)
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}

	updateRecordRef := rollbackApplyPointerUpdateRecordRef(ref.ApplyID)
	switch record.Phase {
	case RollbackApplyPhasePointerSwitchedReloadPending:
		if activePointer.ActivePackID != rollback.TargetPackID {
			return RollbackApplyRecord{}, false, fmt.Errorf(
				"mission store rollback apply %q phase %q requires active runtime pack pointer active_pack_id %q, found %q",
				ref.ApplyID,
				record.Phase,
				rollback.TargetPackID,
				activePointer.ActivePackID,
			)
		}
		if activePointer.UpdateRecordRef != updateRecordRef {
			return RollbackApplyRecord{}, false, fmt.Errorf(
				"mission store rollback apply %q phase %q requires active runtime pack pointer update_record_ref %q, found %q",
				ref.ApplyID,
				record.Phase,
				updateRecordRef,
				activePointer.UpdateRecordRef,
			)
		}
		return record, false, nil
	case RollbackApplyPhaseReadyToApply:
	default:
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase %q does not permit pointer switch execution",
			ref.ApplyID,
			record.Phase,
		)
	}

	if activePointer.ActivePackID == rollback.TargetPackID && activePointer.UpdateRecordRef == updateRecordRef {
		record.Phase = RollbackApplyPhasePointerSwitchedReloadPending
		record.PhaseUpdatedAt = updatedAt
		record.PhaseUpdatedBy = updatedBy
		record = NormalizeRollbackApplyRecord(record)
		if err := ValidateRollbackApplyRecord(record); err != nil {
			return RollbackApplyRecord{}, false, err
		}
		if err := validateRollbackApplyLinkage(root, record); err != nil {
			return RollbackApplyRecord{}, false, err
		}
		if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
			return RollbackApplyRecord{}, false, err
		}
		record, err = LoadRollbackApplyRecord(root, ref.ApplyID)
		if err != nil {
			return RollbackApplyRecord{}, false, err
		}
		return record, true, nil
	}

	if activePointer.ActivePackID != rollback.FromPackID {
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q requires active runtime pack pointer active_pack_id %q before switch, found %q",
			ref.ApplyID,
			rollback.FromPackID,
			activePointer.ActivePackID,
		)
	}

	activePointer.ActivePackID = rollback.TargetPackID
	activePointer.PreviousActivePackID = rollback.FromPackID
	activePointer.UpdatedAt = updatedAt
	activePointer.UpdatedBy = updatedBy
	activePointer.UpdateRecordRef = updateRecordRef
	activePointer.ReloadGeneration++
	if err := StoreActiveRuntimePackPointer(root, activePointer); err != nil {
		return RollbackApplyRecord{}, false, err
	}

	record.Phase = RollbackApplyPhasePointerSwitchedReloadPending
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	record, err = LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	return record, true, nil
}

func ExecuteRollbackApplyReloadApply(root string, applyID string, updatedBy string, updatedAt time.Time) (RollbackApplyRecord, bool, error) {
	return executeRollbackApplyReloadApplyWithConvergence(root, applyID, updatedBy, updatedAt, rollbackApplyRestartStyleConvergence)
}

func ReconcileRollbackApplyRecoveryNeeded(root string, applyID string, updatedBy string, updatedAt time.Time) (RollbackApplyRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_at is required")
	}

	record, err := LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}

	switch record.Phase {
	case RollbackApplyPhaseReloadApplyRecoveryNeeded:
		if _, err := validateRollbackApplyExecutionLinkage(root, record); err != nil {
			return RollbackApplyRecord{}, false, err
		}
		return record, false, nil
	case RollbackApplyPhaseReloadApplyInProgress:
	default:
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase %q does not permit recovery-needed normalization",
			ref.ApplyID,
			record.Phase,
		)
	}

	if _, err := validateRollbackApplyExecutionLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}

	record.Phase = RollbackApplyPhaseReloadApplyRecoveryNeeded
	record.ExecutionError = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	record, err = LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	return record, true, nil
}

func LoadRollbackApplyRecord(root, applyID string) (RollbackApplyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, err
	}
	record, err := loadRollbackApplyRecordFile(root, StoreRollbackApplyPath(root, ref.ApplyID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RollbackApplyRecord{}, ErrRollbackApplyRecordNotFound
		}
		return RollbackApplyRecord{}, err
	}
	return record, nil
}

func ListRollbackApplyRecords(root string) ([]RollbackApplyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRollbackAppliesDir(root), func(path string) (RollbackApplyRecord, error) {
		return loadRollbackApplyRecordFile(root, path)
	})
}

func loadRollbackApplyRecordFile(root, path string) (RollbackApplyRecord, error) {
	var record RollbackApplyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RollbackApplyRecord{}, err
	}
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, err
	}
	return record, nil
}

func validateRollbackApplyLinkage(root string, record RollbackApplyRecord) error {
	rollbackRef := RollbackApplyRollbackRef(record)
	if _, err := LoadRollbackRecord(root, rollbackRef.RollbackID); err != nil {
		return fmt.Errorf("mission store rollback apply rollback_id %q: %w", rollbackRef.RollbackID, err)
	}
	return nil
}

func validateRollbackApplyIdentifierField(surface, fieldName, value string) error {
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

func executeRollbackApplyReloadApplyWithConvergence(root string, applyID string, updatedBy string, updatedAt time.Time, converge func(string, RollbackApplyRecord, RollbackRecord) error) (RollbackApplyRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	ref := NormalizeRollbackApplyRef(RollbackApplyRef{ApplyID: applyID})
	if err := ValidateRollbackApplyRef(ref); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_by is required")
	}
	updatedAt = updatedAt.UTC()
	if updatedAt.IsZero() {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply phase_updated_at is required")
	}
	if converge == nil {
		return RollbackApplyRecord{}, false, fmt.Errorf("mission store rollback apply convergence function is required")
	}

	record, err := LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}

	switch record.Phase {
	case RollbackApplyPhaseReloadApplySucceeded:
		return record, false, nil
	case RollbackApplyPhaseReloadApplyFailed:
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase %q does not permit reload/apply retry",
			ref.ApplyID,
			record.Phase,
		)
	case RollbackApplyPhaseReloadApplyInProgress:
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase %q requires recovery before retry",
			ref.ApplyID,
			record.Phase,
		)
	case RollbackApplyPhasePointerSwitchedReloadPending,
		RollbackApplyPhaseReloadApplyRecoveryNeeded:
	default:
		return RollbackApplyRecord{}, false, fmt.Errorf(
			"mission store rollback apply %q phase %q does not permit reload/apply execution",
			ref.ApplyID,
			record.Phase,
		)
	}

	rollback, err := validateRollbackApplyExecutionLinkage(root, record)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}

	record.Phase = RollbackApplyPhaseReloadApplyInProgress
	record.ExecutionError = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
		return RollbackApplyRecord{}, false, err
	}

	if err := converge(root, record, rollback); err != nil {
		record.Phase = RollbackApplyPhaseReloadApplyFailed
		record.ExecutionError = err.Error()
		record.PhaseUpdatedAt = updatedAt
		record.PhaseUpdatedBy = updatedBy
		record = NormalizeRollbackApplyRecord(record)
		if validationErr := ValidateRollbackApplyRecord(record); validationErr != nil {
			return RollbackApplyRecord{}, false, validationErr
		}
		if linkageErr := validateRollbackApplyLinkage(root, record); linkageErr != nil {
			return RollbackApplyRecord{}, false, linkageErr
		}
		if writeErr := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); writeErr != nil {
			return RollbackApplyRecord{}, false, writeErr
		}
		record, loadErr := LoadRollbackApplyRecord(root, ref.ApplyID)
		if loadErr != nil {
			return RollbackApplyRecord{}, false, loadErr
		}
		return record, true, err
	}

	record.Phase = RollbackApplyPhaseReloadApplySucceeded
	record.ExecutionError = ""
	record.PhaseUpdatedAt = updatedAt
	record.PhaseUpdatedBy = updatedBy
	record = NormalizeRollbackApplyRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(StoreRollbackApplyPath(root, record.ApplyID), record); err != nil {
		return RollbackApplyRecord{}, false, err
	}
	record, err = LoadRollbackApplyRecord(root, ref.ApplyID)
	if err != nil {
		return RollbackApplyRecord{}, false, err
	}
	return record, true, nil
}

func validateRollbackApplyExecutionLinkage(root string, record RollbackApplyRecord) (RollbackRecord, error) {
	rollback, err := LoadRollbackRecord(root, record.RollbackID)
	if err != nil {
		return RollbackRecord{}, fmt.Errorf("mission store rollback apply rollback_id %q: %w", record.RollbackID, err)
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return RollbackRecord{}, err
	}
	updateRecordRef := rollbackApplyPointerUpdateRecordRef(record.ApplyID)
	if activePointer.ActivePackID != rollback.TargetPackID {
		return RollbackRecord{}, fmt.Errorf(
			"mission store rollback apply %q requires active runtime pack pointer active_pack_id %q before reload/apply, found %q",
			record.ApplyID,
			rollback.TargetPackID,
			activePointer.ActivePackID,
		)
	}
	if activePointer.UpdateRecordRef != updateRecordRef {
		return RollbackRecord{}, fmt.Errorf(
			"mission store rollback apply %q requires active runtime pack pointer update_record_ref %q before reload/apply, found %q",
			record.ApplyID,
			updateRecordRef,
			activePointer.UpdateRecordRef,
		)
	}
	if activePointer.PreviousActivePackID != rollback.FromPackID {
		return RollbackRecord{}, fmt.Errorf(
			"mission store rollback apply %q requires active runtime pack pointer previous_active_pack_id %q before reload/apply, found %q",
			record.ApplyID,
			rollback.FromPackID,
			activePointer.PreviousActivePackID,
		)
	}
	if _, err := LoadRuntimePackRecord(root, rollback.TargetPackID); err != nil {
		return RollbackRecord{}, fmt.Errorf("mission store rollback apply target_pack_id %q: %w", rollback.TargetPackID, err)
	}
	return rollback, nil
}

func rollbackApplyPhaseOrder(phase RollbackApplyPhase) int {
	switch phase {
	case RollbackApplyPhaseRecorded:
		return 1
	case RollbackApplyPhaseValidated:
		return 2
	case RollbackApplyPhaseReadyToApply:
		return 3
	default:
		return 0
	}
}

func rollbackApplyPointerUpdateRecordRef(applyID string) string {
	return "rollback_apply:" + strings.TrimSpace(applyID)
}

// rollbackApplyRestartStyleConvergence models the smallest bounded reload/apply
// convergence step available today: a fresh resolution of the already-switched
// active runtime-pack pointer and its target pack record.
func rollbackApplyRestartStyleConvergence(root string, record RollbackApplyRecord, rollback RollbackRecord) error {
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return err
	}
	if activePointer.ActivePackID != rollback.TargetPackID {
		return fmt.Errorf(
			"mission store rollback apply %q convergence requires active runtime pack pointer active_pack_id %q, found %q",
			record.ApplyID,
			rollback.TargetPackID,
			activePointer.ActivePackID,
		)
	}
	if activePointer.UpdateRecordRef != rollbackApplyPointerUpdateRecordRef(record.ApplyID) {
		return fmt.Errorf(
			"mission store rollback apply %q convergence requires active runtime pack pointer update_record_ref %q, found %q",
			record.ApplyID,
			rollbackApplyPointerUpdateRecordRef(record.ApplyID),
			activePointer.UpdateRecordRef,
		)
	}
	resolved, err := ResolveActiveRuntimePackRecord(root)
	if err != nil {
		return err
	}
	if resolved.PackID != rollback.TargetPackID {
		return fmt.Errorf(
			"mission store rollback apply %q convergence resolved active pack %q, want %q",
			record.ApplyID,
			resolved.PackID,
			rollback.TargetPackID,
		)
	}
	return nil
}
