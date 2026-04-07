package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type EligibilityLabel string

const (
	EligibilityLabelAutonomyCompatible EligibilityLabel = "autonomy_compatible"
	EligibilityLabelHumanGated         EligibilityLabel = "human_gated"
	EligibilityLabelIneligible         EligibilityLabel = "ineligible"
)

type EligibilityTargetKind string

const (
	EligibilityTargetKindProvider               EligibilityTargetKind = "provider"
	EligibilityTargetKindPlatform               EligibilityTargetKind = "platform"
	EligibilityTargetKindAccountClass           EligibilityTargetKind = "account_class"
	EligibilityTargetKindTreasuryContainerClass EligibilityTargetKind = "treasury_container_class"
)

type EligibilityCheckRecord struct {
	RecordVersion               int                   `json:"record_version"`
	CheckID                     string                `json:"check_id"`
	TargetKind                  EligibilityTargetKind `json:"target_kind"`
	TargetName                  string                `json:"target_name"`
	CanCreateWithoutOwner       bool                  `json:"can_create_without_owner"`
	CanOnboardWithoutOwner      bool                  `json:"can_onboard_without_owner"`
	CanControlAsAgent           bool                  `json:"can_control_as_agent"`
	CanRecoverAsAgent           bool                  `json:"can_recover_as_agent"`
	RequiresHumanOnlyStep       bool                  `json:"requires_human_only_step"`
	RequiresOwnerOnlySecretOrID bool                  `json:"requires_owner_only_secret_or_identity"`
	RulesAsObservedOK           bool                  `json:"rules_as_observed_ok"`
	Label                       EligibilityLabel      `json:"label"`
	Reasons                     []string              `json:"reasons"`
	CheckedAt                   time.Time             `json:"checked_at"`
}

type PlatformRecord struct {
	RecordVersion    int                   `json:"record_version"`
	PlatformID       string                `json:"platform_id"`
	PlatformName     string                `json:"platform_name"`
	TargetClass      EligibilityTargetKind `json:"target_class"`
	EligibilityLabel EligibilityLabel      `json:"eligibility_label"`
	LastCheckID      string                `json:"last_check_id"`
	Notes            []string              `json:"notes"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

var (
	ErrEligibilityCheckRecordNotFound = errors.New("mission store eligibility check record not found")
	ErrPlatformRecordNotFound         = errors.New("mission store platform record not found")
)

func StoreEligibilityDir(root string) string {
	return filepath.Join(root, "eligibility")
}

func StoreEligibilityChecksDir(root string) string {
	return filepath.Join(StoreEligibilityDir(root), "checks")
}

func StoreEligibilityCheckPath(root, checkID string) string {
	return filepath.Join(StoreEligibilityChecksDir(root), checkID+".json")
}

func StorePlatformRecordsDir(root string) string {
	return filepath.Join(StoreEligibilityDir(root), "platform_records")
}

func StorePlatformRecordPath(root, platformID string) string {
	return filepath.Join(StorePlatformRecordsDir(root), platformID+".json")
}

func ValidateEligibilityCheckRecord(record EligibilityCheckRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store eligibility check record_version must be positive")
	}
	if strings.TrimSpace(record.CheckID) == "" {
		return fmt.Errorf("mission store eligibility check check_id is required")
	}
	if !isValidEligibilityTargetKind(record.TargetKind) {
		return fmt.Errorf("mission store eligibility check target_kind %q is invalid", record.TargetKind)
	}
	if strings.TrimSpace(record.TargetName) == "" {
		return fmt.Errorf("mission store eligibility check target_name is required")
	}
	if !isValidEligibilityLabel(record.Label) {
		return fmt.Errorf("mission store eligibility check label %q is invalid", record.Label)
	}
	if !hasNonBlankStrings(record.Reasons) {
		return fmt.Errorf("mission store eligibility check reasons are required")
	}
	if record.CheckedAt.IsZero() {
		return fmt.Errorf("mission store eligibility check checked_at is required")
	}
	return nil
}

func ValidatePlatformRecord(record PlatformRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store platform record record_version must be positive")
	}
	if strings.TrimSpace(record.PlatformID) == "" {
		return fmt.Errorf("mission store platform record platform_id is required")
	}
	if strings.TrimSpace(record.PlatformName) == "" {
		return fmt.Errorf("mission store platform record platform_name is required")
	}
	if !isValidEligibilityTargetKind(record.TargetClass) {
		return fmt.Errorf("mission store platform record target_class %q is invalid", record.TargetClass)
	}
	if !isValidEligibilityLabel(record.EligibilityLabel) {
		return fmt.Errorf("mission store platform record eligibility_label %q is invalid", record.EligibilityLabel)
	}
	if strings.TrimSpace(record.LastCheckID) == "" {
		return fmt.Errorf("mission store platform record last_check_id is required")
	}
	if !hasNonBlankStrings(record.Notes) {
		return fmt.Errorf("mission store platform record notes are required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store platform record updated_at is required")
	}
	return nil
}

func StoreEligibilityCheckRecord(root string, record EligibilityCheckRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.CheckedAt = record.CheckedAt.UTC()
	if err := ValidateEligibilityCheckRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreEligibilityCheckPath(root, record.CheckID), record)
}

func LoadEligibilityCheckRecord(root, checkID string) (EligibilityCheckRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return EligibilityCheckRecord{}, err
	}
	if strings.TrimSpace(checkID) == "" {
		return EligibilityCheckRecord{}, fmt.Errorf("mission store eligibility check check_id is required")
	}
	record, err := loadEligibilityCheckRecordFile(StoreEligibilityCheckPath(root, checkID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EligibilityCheckRecord{}, ErrEligibilityCheckRecordNotFound
		}
		return EligibilityCheckRecord{}, err
	}
	return record, nil
}

func ListEligibilityCheckRecords(root string) ([]EligibilityCheckRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	records, err := listStoreJSONRecords(StoreEligibilityChecksDir(root), loadEligibilityCheckRecordFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return records, nil
}

func StorePlatformRecord(root string, record PlatformRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidatePlatformRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StorePlatformRecordPath(root, record.PlatformID), record)
}

func LoadPlatformRecord(root, platformID string) (PlatformRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PlatformRecord{}, err
	}
	if strings.TrimSpace(platformID) == "" {
		return PlatformRecord{}, fmt.Errorf("mission store platform record platform_id is required")
	}
	record, err := loadPlatformRecordFile(StorePlatformRecordPath(root, platformID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PlatformRecord{}, ErrPlatformRecordNotFound
		}
		return PlatformRecord{}, err
	}
	return record, nil
}

func ListPlatformRecords(root string) ([]PlatformRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	records, err := listStoreJSONRecords(StorePlatformRecordsDir(root), loadPlatformRecordFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return records, nil
}

func loadEligibilityCheckRecordFile(path string) (EligibilityCheckRecord, error) {
	var record EligibilityCheckRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return EligibilityCheckRecord{}, err
	}
	if err := ValidateEligibilityCheckRecord(record); err != nil {
		return EligibilityCheckRecord{}, err
	}
	record.CheckedAt = record.CheckedAt.UTC()
	return record, nil
}

func loadPlatformRecordFile(path string) (PlatformRecord, error) {
	var record PlatformRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return PlatformRecord{}, err
	}
	if err := ValidatePlatformRecord(record); err != nil {
		return PlatformRecord{}, err
	}
	record.UpdatedAt = record.UpdatedAt.UTC()
	return record, nil
}

func normalizeRecordVersion(version int) int {
	if version > 0 {
		return version
	}
	return StoreRecordVersion
}

func isValidEligibilityLabel(label EligibilityLabel) bool {
	switch label {
	case EligibilityLabelAutonomyCompatible, EligibilityLabelHumanGated, EligibilityLabelIneligible:
		return true
	default:
		return false
	}
}

func isValidEligibilityTargetKind(kind EligibilityTargetKind) bool {
	switch kind {
	case EligibilityTargetKindProvider, EligibilityTargetKindPlatform, EligibilityTargetKindAccountClass, EligibilityTargetKindTreasuryContainerClass:
		return true
	default:
		return false
	}
}

func hasNonBlankStrings(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
