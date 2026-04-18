package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type CapabilityRecord struct {
	RecordVersion int           `json:"record_version"`
	CapabilityID  string        `json:"capability_id"`
	Class         string        `json:"class"`
	Name          string        `json:"name"`
	Exposed       bool          `json:"exposed"`
	AuthorityTier AuthorityTier `json:"authority_tier"`
	Validator     string        `json:"validator"`
	Notes         string        `json:"notes"`
}

var (
	ErrCapabilityRecordNotFound  = errors.New("mission store capability record not found")
	ErrCapabilityRecordAmbiguous = errors.New("mission store capability record is ambiguous")
)

func StoreCapabilitiesDir(root string) string {
	return filepath.Join(root, "capabilities")
}

func StoreCapabilityPath(root, capabilityID string) string {
	return filepath.Join(StoreCapabilitiesDir(root), capabilityID+".json")
}

func NormalizeCapabilityRecord(record CapabilityRecord) CapabilityRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.CapabilityID = strings.TrimSpace(record.CapabilityID)
	record.Class = strings.TrimSpace(record.Class)
	record.Name = strings.TrimSpace(record.Name)
	record.AuthorityTier = AuthorityTier(strings.TrimSpace(string(record.AuthorityTier)))
	record.Validator = strings.TrimSpace(record.Validator)
	record.Notes = strings.TrimSpace(record.Notes)
	return record
}

func ValidateCapabilityRecord(record CapabilityRecord) error {
	record = NormalizeCapabilityRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store capability record_version must be positive")
	}
	if err := validateCapabilityID(record.CapabilityID, "mission store capability"); err != nil {
		return err
	}
	if record.Class == "" {
		return fmt.Errorf("mission store capability class is required")
	}
	if record.Name == "" {
		return fmt.Errorf("mission store capability name is required")
	}
	if _, ok := authorityRank(record.AuthorityTier); !ok {
		return fmt.Errorf("mission store capability authority_tier %q is invalid", strings.TrimSpace(string(record.AuthorityTier)))
	}
	if record.Validator == "" {
		return fmt.Errorf("mission store capability validator is required")
	}
	if record.Notes == "" {
		return fmt.Errorf("mission store capability notes is required")
	}
	return nil
}

func StoreCapabilityRecord(root string, record CapabilityRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCapabilityRecord(record)
	if err := ValidateCapabilityRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCapabilityPath(root, record.CapabilityID), record)
}

func LoadCapabilityRecord(root, capabilityID string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if err := validateCapabilityID(capabilityID, "mission store capability"); err != nil {
		return CapabilityRecord{}, err
	}
	record, err := loadCapabilityRecordFile(StoreCapabilityPath(root, strings.TrimSpace(capabilityID)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CapabilityRecord{}, ErrCapabilityRecordNotFound
		}
		return CapabilityRecord{}, err
	}
	return record, nil
}

func ListCapabilityRecords(root string) ([]CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreCapabilitiesDir(root), loadCapabilityRecordFile)
}

func ResolveCapabilityRecordByName(root, name string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return CapabilityRecord{}, fmt.Errorf("mission store capability name is required")
	}

	records, err := ListCapabilityRecords(root)
	if err != nil {
		return CapabilityRecord{}, err
	}

	var matches []CapabilityRecord
	for _, record := range records {
		if record.Name == normalizedName {
			matches = append(matches, record)
		}
	}

	switch len(matches) {
	case 0:
		return CapabilityRecord{}, ErrCapabilityRecordNotFound
	case 1:
		return matches[0], nil
	default:
		return CapabilityRecord{}, fmt.Errorf("%w: capability name %q matched %d records", ErrCapabilityRecordAmbiguous, normalizedName, len(matches))
	}
}

func loadCapabilityRecordFile(path string) (CapabilityRecord, error) {
	var record CapabilityRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CapabilityRecord{}, err
	}
	record = NormalizeCapabilityRecord(record)
	if err := ValidateCapabilityRecord(record); err != nil {
		return CapabilityRecord{}, err
	}
	return record, nil
}

func validateCapabilityID(capabilityID, surface string) error {
	if err := validateCapabilityIDValue(capabilityID); err != nil {
		return fmt.Errorf("%s %w", surface, err)
	}
	return nil
}

func validateCapabilityIDValue(capabilityID string) error {
	normalized := strings.TrimSpace(capabilityID)
	if normalized == "" {
		return fmt.Errorf("capability_id is required")
	}
	if normalized == "." || normalized == ".." {
		return fmt.Errorf("capability_id %q is invalid", normalized)
	}
	if strings.ContainsAny(normalized, `/\`) {
		return fmt.Errorf("capability_id %q is invalid", normalized)
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("capability_id %q is invalid", normalized)
		}
	}
	return nil
}
