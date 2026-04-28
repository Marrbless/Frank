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

type AutonomyOwnerPauseState string

const (
	AutonomyOwnerPauseStateActive AutonomyOwnerPauseState = "active"
)

type AutonomyOwnerPauseRecord struct {
	RecordVersion       int                     `json:"record_version"`
	OwnerPauseID        string                  `json:"owner_pause_id"`
	BudgetID            string                  `json:"budget_id"`
	StandingDirectiveID string                  `json:"standing_directive_id,omitempty"`
	AppliesToHotUpdates bool                    `json:"applies_to_hot_updates"`
	State               AutonomyOwnerPauseState `json:"state"`
	Reason              string                  `json:"reason"`
	AuthorityRef        string                  `json:"authority_ref"`
	PausedAt            time.Time               `json:"paused_at"`
	CreatedAt           time.Time               `json:"created_at"`
	CreatedBy           string                  `json:"created_by"`
}

var ErrAutonomyOwnerPauseRecordNotFound = errors.New("mission store autonomy owner pause record not found")

func StoreAutonomyOwnerPausesDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "owner_pauses")
}

func StoreAutonomyOwnerPausePath(root, ownerPauseID string) string {
	return filepath.Join(StoreAutonomyOwnerPausesDir(root), strings.TrimSpace(ownerPauseID)+".json")
}

func AutonomyOwnerPauseIDForBudget(budgetID string) string {
	return "autonomy-owner-pause-" + strings.TrimSpace(budgetID)
}

func NormalizeAutonomyOwnerPauseRecord(record AutonomyOwnerPauseRecord) AutonomyOwnerPauseRecord {
	record.OwnerPauseID = strings.TrimSpace(record.OwnerPauseID)
	record.BudgetID = strings.TrimSpace(record.BudgetID)
	record.StandingDirectiveID = strings.TrimSpace(record.StandingDirectiveID)
	record.State = AutonomyOwnerPauseState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.AuthorityRef = strings.TrimSpace(record.AuthorityRef)
	record.PausedAt = record.PausedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateAutonomyOwnerPauseRecord(record AutonomyOwnerPauseRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store autonomy owner pause record_version must be positive")
	}
	if err := validateAutonomyIdentifierField("mission store autonomy owner pause", "owner_pause_id", record.OwnerPauseID); err != nil {
		return err
	}
	if err := ValidateAutonomyBudgetRef(AutonomyBudgetRef{BudgetID: record.BudgetID}); err != nil {
		return fmt.Errorf("mission store autonomy owner pause budget_id %q: %w", record.BudgetID, err)
	}
	if record.StandingDirectiveID != "" {
		if err := ValidateStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: record.StandingDirectiveID}); err != nil {
			return fmt.Errorf("mission store autonomy owner pause standing_directive_id %q: %w", record.StandingDirectiveID, err)
		}
	}
	if record.OwnerPauseID != AutonomyOwnerPauseIDForBudget(record.BudgetID) {
		return fmt.Errorf("mission store autonomy owner pause owner_pause_id %q does not match deterministic owner_pause_id %q", record.OwnerPauseID, AutonomyOwnerPauseIDForBudget(record.BudgetID))
	}
	if !record.AppliesToHotUpdates {
		return fmt.Errorf("mission store autonomy owner pause applies_to_hot_updates must be true")
	}
	if record.State != AutonomyOwnerPauseStateActive {
		return fmt.Errorf("mission store autonomy owner pause state %q is invalid", record.State)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store autonomy owner pause reason is required")
	}
	if record.AuthorityRef == "" {
		return fmt.Errorf("mission store autonomy owner pause authority_ref is required")
	}
	if isNaturalLanguageAuthorityRef(record.AuthorityRef) {
		return fmt.Errorf("mission store autonomy owner pause authority_ref must not be natural-language approval")
	}
	if record.PausedAt.IsZero() {
		return fmt.Errorf("mission store autonomy owner pause paused_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store autonomy owner pause created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store autonomy owner pause created_by is required")
	}
	return nil
}

func StoreAutonomyOwnerPauseRecord(root string, record AutonomyOwnerPauseRecord) (AutonomyOwnerPauseRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyOwnerPauseRecord{}, false, err
	}
	record = NormalizeAutonomyOwnerPauseRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateAutonomyOwnerPauseRecord(record); err != nil {
		return AutonomyOwnerPauseRecord{}, false, err
	}
	if err := validateAutonomyOwnerPauseLinkage(root, record); err != nil {
		return AutonomyOwnerPauseRecord{}, false, err
	}
	path := StoreAutonomyOwnerPausePath(root, record.OwnerPauseID)
	if existing, err := loadAutonomyOwnerPauseRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return AutonomyOwnerPauseRecord{}, false, fmt.Errorf("mission store autonomy owner pause %q already exists", record.OwnerPauseID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return AutonomyOwnerPauseRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return AutonomyOwnerPauseRecord{}, false, err
	}
	return record, true, nil
}

func LoadAutonomyOwnerPauseRecord(root, ownerPauseID string) (AutonomyOwnerPauseRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyOwnerPauseRecord{}, err
	}
	ownerPauseID = strings.TrimSpace(ownerPauseID)
	if err := validateAutonomyIdentifierField("autonomy owner pause ref", "owner_pause_id", ownerPauseID); err != nil {
		return AutonomyOwnerPauseRecord{}, err
	}
	record, err := loadAutonomyOwnerPauseRecordFile(root, StoreAutonomyOwnerPausePath(root, ownerPauseID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AutonomyOwnerPauseRecord{}, ErrAutonomyOwnerPauseRecordNotFound
		}
		return AutonomyOwnerPauseRecord{}, err
	}
	return record, nil
}

func ListAutonomyOwnerPauseRecords(root string) ([]AutonomyOwnerPauseRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreAutonomyOwnerPausesDir(root), func(path string) (AutonomyOwnerPauseRecord, error) {
		return loadAutonomyOwnerPauseRecordFile(root, path)
	})
}

func LoadActiveAutonomyOwnerPauseForBudget(root, budgetID string) (AutonomyOwnerPauseRecord, bool, error) {
	pause, err := LoadAutonomyOwnerPauseRecord(root, AutonomyOwnerPauseIDForBudget(budgetID))
	if err != nil {
		if errors.Is(err, ErrAutonomyOwnerPauseRecordNotFound) {
			return AutonomyOwnerPauseRecord{}, false, nil
		}
		return AutonomyOwnerPauseRecord{}, false, err
	}
	return pause, pause.State == AutonomyOwnerPauseStateActive && pause.AppliesToHotUpdates, nil
}

func loadAutonomyOwnerPauseRecordFile(root, path string) (AutonomyOwnerPauseRecord, error) {
	var record AutonomyOwnerPauseRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return AutonomyOwnerPauseRecord{}, err
	}
	record = NormalizeAutonomyOwnerPauseRecord(record)
	if err := ValidateAutonomyOwnerPauseRecord(record); err != nil {
		return AutonomyOwnerPauseRecord{}, err
	}
	if err := validateAutonomyOwnerPauseLinkage(root, record); err != nil {
		return AutonomyOwnerPauseRecord{}, err
	}
	return record, nil
}

func validateAutonomyOwnerPauseLinkage(root string, record AutonomyOwnerPauseRecord) error {
	if _, err := LoadAutonomyBudgetRecord(root, record.BudgetID); err != nil {
		return err
	}
	return nil
}

func isNaturalLanguageAuthorityRef(ref string) bool {
	ref = strings.ToLower(strings.TrimSpace(ref))
	return ref == "natural_language" || strings.HasPrefix(ref, "natural_language:") || strings.HasPrefix(ref, "chat:")
}
