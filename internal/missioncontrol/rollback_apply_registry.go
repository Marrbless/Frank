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
	RollbackApplyPhaseRecorded     RollbackApplyPhase = "recorded"
	RollbackApplyPhaseValidated    RollbackApplyPhase = "validated"
	RollbackApplyPhaseReadyToApply RollbackApplyPhase = "ready_to_apply"
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
	RequestedAt     time.Time                    `json:"requested_at"`
	CreatedAt       time.Time                    `json:"created_at"`
	CreatedBy       string                       `json:"created_by"`
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
	record.RequestedAt = record.RequestedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
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
	case RollbackApplyPhaseRecorded, RollbackApplyPhaseValidated, RollbackApplyPhaseReadyToApply:
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
	}
	if err := StoreRollbackApplyRecord(root, record); err != nil {
		return RollbackApplyRecord{}, err
	}
	return LoadRollbackApplyRecord(root, record.ApplyID)
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
