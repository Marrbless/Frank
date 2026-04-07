package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FrankIdentityRecord struct {
	RecordVersion        int                          `json:"record_version"`
	IdentityID           string                       `json:"identity_id"`
	IdentityKind         string                       `json:"identity_kind"`
	DisplayName          string                       `json:"display_name"`
	ProviderOrPlatformID string                       `json:"provider_or_platform_id"`
	IdentityMode         IdentityMode                 `json:"identity_mode"`
	State                string                       `json:"state"`
	EligibilityTargetRef AutonomyEligibilityTargetRef `json:"eligibility_target_ref"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

type FrankAccountRecord struct {
	RecordVersion        int                          `json:"record_version"`
	AccountID            string                       `json:"account_id"`
	AccountKind          string                       `json:"account_kind"`
	Label                string                       `json:"label"`
	ProviderOrPlatformID string                       `json:"provider_or_platform_id"`
	IdentityID           string                       `json:"identity_id"`
	ControlModel         string                       `json:"control_model"`
	RecoveryModel        string                       `json:"recovery_model"`
	State                string                       `json:"state"`
	EligibilityTargetRef AutonomyEligibilityTargetRef `json:"eligibility_target_ref"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

type FrankContainerRecord struct {
	RecordVersion        int                          `json:"record_version"`
	ContainerID          string                       `json:"container_id"`
	ContainerKind        string                       `json:"container_kind"`
	Label                string                       `json:"label"`
	ContainerClassID     string                       `json:"container_class_id"`
	State                string                       `json:"state"`
	EligibilityTargetRef AutonomyEligibilityTargetRef `json:"eligibility_target_ref"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

var (
	ErrFrankIdentityRecordNotFound  = errors.New("mission store Frank identity record not found")
	ErrFrankAccountRecordNotFound   = errors.New("mission store Frank account record not found")
	ErrFrankContainerRecordNotFound = errors.New("mission store Frank container record not found")
)

func StoreFrankRegistryDir(root string) string {
	return filepath.Join(root, "frank_registry")
}

func StoreFrankIdentitiesDir(root string) string {
	return filepath.Join(StoreFrankRegistryDir(root), "identities")
}

func StoreFrankIdentityPath(root, identityID string) string {
	return filepath.Join(StoreFrankIdentitiesDir(root), identityID+".json")
}

func StoreFrankAccountsDir(root string) string {
	return filepath.Join(StoreFrankRegistryDir(root), "accounts")
}

func StoreFrankAccountPath(root, accountID string) string {
	return filepath.Join(StoreFrankAccountsDir(root), accountID+".json")
}

func StoreFrankContainersDir(root string) string {
	return filepath.Join(StoreFrankRegistryDir(root), "containers")
}

func StoreFrankContainerPath(root, containerID string) string {
	return filepath.Join(StoreFrankContainersDir(root), containerID+".json")
}

func ValidateFrankRegistryEligibilityLink(root string, target AutonomyEligibilityTargetRef) (AutonomyEligibilityResult, error) {
	result, err := EvaluateAutonomyEligibility(root, target)
	if err != nil {
		return AutonomyEligibilityResult{}, err
	}
	if result.Decision == AutonomyEligibilityDecisionUnknown {
		return result, fmt.Errorf("mission store frank registry eligibility_target_ref %q has no linked eligibility registry record", target.RegistryID)
	}
	return result, nil
}

func ValidateFrankIdentityRecord(record FrankIdentityRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store Frank identity record_version must be positive")
	}
	if strings.TrimSpace(record.IdentityID) == "" {
		return fmt.Errorf("mission store Frank identity identity_id is required")
	}
	if strings.TrimSpace(record.IdentityKind) == "" {
		return fmt.Errorf("mission store Frank identity identity_kind is required")
	}
	if strings.TrimSpace(record.DisplayName) == "" {
		return fmt.Errorf("mission store Frank identity display_name is required")
	}
	if strings.TrimSpace(record.ProviderOrPlatformID) == "" {
		return fmt.Errorf("mission store Frank identity provider_or_platform_id is required")
	}
	if err := validateIdentityMode(record.IdentityMode); err != nil {
		return err
	}
	if strings.TrimSpace(record.State) == "" {
		return fmt.Errorf("mission store Frank identity state is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store Frank identity created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store Frank identity updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store Frank identity updated_at must be on or after created_at")
	}
	return nil
}

func ValidateFrankAccountRecord(record FrankAccountRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store Frank account record_version must be positive")
	}
	if strings.TrimSpace(record.AccountID) == "" {
		return fmt.Errorf("mission store Frank account account_id is required")
	}
	if strings.TrimSpace(record.AccountKind) == "" {
		return fmt.Errorf("mission store Frank account account_kind is required")
	}
	if strings.TrimSpace(record.Label) == "" {
		return fmt.Errorf("mission store Frank account label is required")
	}
	if strings.TrimSpace(record.ProviderOrPlatformID) == "" {
		return fmt.Errorf("mission store Frank account provider_or_platform_id is required")
	}
	if strings.TrimSpace(record.IdentityID) == "" {
		return fmt.Errorf("mission store Frank account identity_id is required")
	}
	if strings.TrimSpace(record.ControlModel) == "" {
		return fmt.Errorf("mission store Frank account control_model is required")
	}
	if strings.TrimSpace(record.RecoveryModel) == "" {
		return fmt.Errorf("mission store Frank account recovery_model is required")
	}
	if strings.TrimSpace(record.State) == "" {
		return fmt.Errorf("mission store Frank account state is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store Frank account created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store Frank account updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store Frank account updated_at must be on or after created_at")
	}
	return nil
}

func ValidateFrankContainerRecord(record FrankContainerRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store Frank container record_version must be positive")
	}
	if strings.TrimSpace(record.ContainerID) == "" {
		return fmt.Errorf("mission store Frank container container_id is required")
	}
	if strings.TrimSpace(record.ContainerKind) == "" {
		return fmt.Errorf("mission store Frank container container_kind is required")
	}
	if strings.TrimSpace(record.Label) == "" {
		return fmt.Errorf("mission store Frank container label is required")
	}
	if strings.TrimSpace(record.ContainerClassID) == "" {
		return fmt.Errorf("mission store Frank container container_class_id is required")
	}
	if strings.TrimSpace(record.State) == "" {
		return fmt.Errorf("mission store Frank container state is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store Frank container created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store Frank container updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store Frank container updated_at must be on or after created_at")
	}
	return nil
}

func StoreFrankIdentityRecord(root string, record FrankIdentityRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.IdentityMode = NormalizeIdentityMode(record.IdentityMode)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankIdentityRecord(record); err != nil {
		return err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreFrankIdentityPath(root, record.IdentityID), record)
}

func LoadFrankIdentityRecord(root, identityID string) (FrankIdentityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return FrankIdentityRecord{}, err
	}
	if strings.TrimSpace(identityID) == "" {
		return FrankIdentityRecord{}, fmt.Errorf("mission store Frank identity identity_id is required")
	}
	record, err := loadFrankIdentityRecordFile(root, StoreFrankIdentityPath(root, identityID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FrankIdentityRecord{}, ErrFrankIdentityRecordNotFound
		}
		return FrankIdentityRecord{}, err
	}
	return record, nil
}

func ListFrankIdentityRecords(root string) ([]FrankIdentityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreFrankIdentitiesDir(root), func(path string) (FrankIdentityRecord, error) {
		return loadFrankIdentityRecordFile(root, path)
	})
}

func StoreFrankAccountRecord(root string, record FrankAccountRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankAccountRecord(record); err != nil {
		return err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreFrankAccountPath(root, record.AccountID), record)
}

func LoadFrankAccountRecord(root, accountID string) (FrankAccountRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return FrankAccountRecord{}, err
	}
	if strings.TrimSpace(accountID) == "" {
		return FrankAccountRecord{}, fmt.Errorf("mission store Frank account account_id is required")
	}
	record, err := loadFrankAccountRecordFile(root, StoreFrankAccountPath(root, accountID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FrankAccountRecord{}, ErrFrankAccountRecordNotFound
		}
		return FrankAccountRecord{}, err
	}
	return record, nil
}

func ListFrankAccountRecords(root string) ([]FrankAccountRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreFrankAccountsDir(root), func(path string) (FrankAccountRecord, error) {
		return loadFrankAccountRecordFile(root, path)
	})
}

func StoreFrankContainerRecord(root string, record FrankContainerRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankContainerRecord(record); err != nil {
		return err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreFrankContainerPath(root, record.ContainerID), record)
}

func LoadFrankContainerRecord(root, containerID string) (FrankContainerRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return FrankContainerRecord{}, err
	}
	if strings.TrimSpace(containerID) == "" {
		return FrankContainerRecord{}, fmt.Errorf("mission store Frank container container_id is required")
	}
	record, err := loadFrankContainerRecordFile(root, StoreFrankContainerPath(root, containerID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FrankContainerRecord{}, ErrFrankContainerRecordNotFound
		}
		return FrankContainerRecord{}, err
	}
	return record, nil
}

func ListFrankContainerRecords(root string) ([]FrankContainerRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreFrankContainersDir(root), func(path string) (FrankContainerRecord, error) {
		return loadFrankContainerRecordFile(root, path)
	})
}

func loadFrankIdentityRecordFile(root, path string) (FrankIdentityRecord, error) {
	var record FrankIdentityRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankIdentityRecord{}, err
	}
	record.IdentityMode = NormalizeIdentityMode(record.IdentityMode)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankIdentityRecord(record); err != nil {
		return FrankIdentityRecord{}, err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return FrankIdentityRecord{}, err
	}
	return record, nil
}

func loadFrankAccountRecordFile(root, path string) (FrankAccountRecord, error) {
	var record FrankAccountRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankAccountRecord{}, err
	}
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankAccountRecord(record); err != nil {
		return FrankAccountRecord{}, err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return FrankAccountRecord{}, err
	}
	return record, nil
}

func loadFrankContainerRecordFile(root, path string) (FrankContainerRecord, error) {
	var record FrankContainerRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return FrankContainerRecord{}, err
	}
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	if err := ValidateFrankContainerRecord(record); err != nil {
		return FrankContainerRecord{}, err
	}
	if _, err := ValidateFrankRegistryEligibilityLink(root, record.EligibilityTargetRef); err != nil {
		return FrankContainerRecord{}, err
	}
	return record, nil
}
