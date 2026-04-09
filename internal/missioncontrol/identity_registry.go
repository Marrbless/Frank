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

// FrankIdentityObjectView is a read-model adapter that exposes canonical
// object names without forcing a durable storage migration.
type FrankIdentityObjectView struct {
	IdentityID           string                       `json:"identity_id"`
	IdentityKind         string                       `json:"identity_kind"`
	DisplayName          string                       `json:"display_name"`
	ProviderOrPlatform   string                       `json:"provider_or_platform"`
	IdentityMode         IdentityMode                 `json:"identity_mode"`
	Status               string                       `json:"status"`
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

// FrankAccountObjectView is a read-model adapter that exposes canonical
// object names without forcing a durable storage migration.
type FrankAccountObjectView struct {
	AccountID            string                       `json:"account_id"`
	AccountKind          string                       `json:"account_kind"`
	Label                string                       `json:"label"`
	ProviderOrPlatform   string                       `json:"provider_or_platform"`
	IdentityID           string                       `json:"identity_id"`
	ControlModel         string                       `json:"control_model"`
	RecoveryModel        string                       `json:"recovery_model"`
	Status               string                       `json:"status"`
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

// FrankContainerObjectView is a read-model adapter that exposes canonical
// object names without forcing a durable storage migration.
type FrankContainerObjectView struct {
	ContainerID          string                       `json:"container_id"`
	ContainerKind        string                       `json:"container_kind"`
	Label                string                       `json:"label"`
	ContainerClassID     string                       `json:"container_class_id"`
	Status               string                       `json:"status"`
	EligibilityTargetRef AutonomyEligibilityTargetRef `json:"eligibility_target_ref"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
}

var (
	ErrFrankIdentityRecordNotFound  = errors.New("mission store Frank identity record not found")
	ErrFrankAccountRecordNotFound   = errors.New("mission store Frank account record not found")
	ErrFrankContainerRecordNotFound = errors.New("mission store Frank container record not found")
)

type ResolvedFrankRegistryObjectRef struct {
	Ref       FrankRegistryObjectRef `json:"ref"`
	Identity  *FrankIdentityRecord   `json:"identity,omitempty"`
	Account   *FrankAccountRecord    `json:"account,omitempty"`
	Container *FrankContainerRecord  `json:"container,omitempty"`
}

type ResolvedExecutionContextFrankRegistryObjects struct {
	ResolvedRefs []ResolvedFrankRegistryObjectRef `json:"resolved_refs,omitempty"`
	Identities   []FrankIdentityRecord            `json:"identities,omitempty"`
	Accounts     []FrankAccountRecord             `json:"accounts,omitempty"`
	Containers   []FrankContainerRecord           `json:"containers,omitempty"`
}

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

func ValidateFrankIdentityLink(root, identityID string) error {
	if strings.TrimSpace(identityID) == "" {
		return fmt.Errorf("mission store Frank account identity_id is required")
	}
	if _, err := LoadFrankIdentityRecord(root, identityID); err != nil {
		return fmt.Errorf("mission store Frank account identity_id %q: %w", strings.TrimSpace(identityID), err)
	}
	return nil
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
	if record.EligibilityTargetRef.Kind != EligibilityTargetKindTreasuryContainerClass {
		return fmt.Errorf(
			"mission store Frank container eligibility_target_ref.kind %q must be %q",
			strings.TrimSpace(string(record.EligibilityTargetRef.Kind)),
			EligibilityTargetKindTreasuryContainerClass,
		)
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

func (record FrankIdentityRecord) AsObjectView() FrankIdentityObjectView {
	return FrankIdentityObjectView{
		IdentityID:           record.IdentityID,
		IdentityKind:         record.IdentityKind,
		DisplayName:          record.DisplayName,
		ProviderOrPlatform:   record.ProviderOrPlatformID,
		IdentityMode:         record.IdentityMode,
		Status:               record.State,
		EligibilityTargetRef: record.EligibilityTargetRef,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func (record FrankAccountRecord) AsObjectView() FrankAccountObjectView {
	return FrankAccountObjectView{
		AccountID:            record.AccountID,
		AccountKind:          record.AccountKind,
		Label:                record.Label,
		ProviderOrPlatform:   record.ProviderOrPlatformID,
		IdentityID:           record.IdentityID,
		ControlModel:         record.ControlModel,
		RecoveryModel:        record.RecoveryModel,
		Status:               record.State,
		EligibilityTargetRef: record.EligibilityTargetRef,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func (record FrankContainerRecord) AsObjectView() FrankContainerObjectView {
	return FrankContainerObjectView{
		ContainerID:          record.ContainerID,
		ContainerKind:        record.ContainerKind,
		Label:                record.Label,
		ContainerClassID:     record.ContainerClassID,
		Status:               record.State,
		EligibilityTargetRef: record.EligibilityTargetRef,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func ResolveFrankRegistryObjectRef(root string, ref FrankRegistryObjectRef) (ResolvedFrankRegistryObjectRef, error) {
	normalized := NormalizeFrankRegistryObjectRef(ref)
	if err := validateFrankRegistryObjectRef(normalized); err != nil {
		return ResolvedFrankRegistryObjectRef{}, err
	}

	switch normalized.Kind {
	case FrankRegistryObjectKindIdentity:
		record, err := LoadFrankIdentityRecord(root, normalized.ObjectID)
		if err != nil {
			return ResolvedFrankRegistryObjectRef{}, fmt.Errorf("resolve Frank object ref kind %q object_id %q: %w", normalized.Kind, normalized.ObjectID, err)
		}
		return ResolvedFrankRegistryObjectRef{
			Ref:      normalized,
			Identity: &record,
		}, nil
	case FrankRegistryObjectKindAccount:
		record, err := LoadFrankAccountRecord(root, normalized.ObjectID)
		if err != nil {
			return ResolvedFrankRegistryObjectRef{}, fmt.Errorf("resolve Frank object ref kind %q object_id %q: %w", normalized.Kind, normalized.ObjectID, err)
		}
		return ResolvedFrankRegistryObjectRef{
			Ref:     normalized,
			Account: &record,
		}, nil
	case FrankRegistryObjectKindContainer:
		record, err := LoadFrankContainerRecord(root, normalized.ObjectID)
		if err != nil {
			return ResolvedFrankRegistryObjectRef{}, fmt.Errorf("resolve Frank object ref kind %q object_id %q: %w", normalized.Kind, normalized.ObjectID, err)
		}
		return ResolvedFrankRegistryObjectRef{
			Ref:       normalized,
			Container: &record,
		}, nil
	default:
		return ResolvedFrankRegistryObjectRef{}, fmt.Errorf("Frank object ref kind %q is invalid", strings.TrimSpace(string(normalized.Kind)))
	}
}

func ResolveFrankRegistryObjectRefs(root string, refs []FrankRegistryObjectRef) ([]ResolvedFrankRegistryObjectRef, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(refs))
	normalizedRefs := make([]FrankRegistryObjectRef, len(refs))
	for i, ref := range refs {
		normalized := NormalizeFrankRegistryObjectRef(ref)
		if err := validateFrankRegistryObjectRef(normalized); err != nil {
			return nil, err
		}

		key := normalizedFrankRegistryObjectRefKey(normalized)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate Frank object ref kind %q object_id %q", normalized.Kind, normalized.ObjectID)
		}
		seen[key] = struct{}{}
		normalizedRefs[i] = normalized
	}

	resolved := make([]ResolvedFrankRegistryObjectRef, 0, len(normalizedRefs))
	for _, normalized := range normalizedRefs {
		record, err := ResolveFrankRegistryObjectRef(root, normalized)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, record)
	}

	return resolved, nil
}

func ResolveExecutionContextFrankRegistryObjectRefs(ec ExecutionContext) (ResolvedExecutionContextFrankRegistryObjects, error) {
	if ec.Step == nil {
		return ResolvedExecutionContextFrankRegistryObjects{}, fmt.Errorf("execution context step is required")
	}
	if len(ec.Step.FrankObjectRefs) == 0 {
		return ResolvedExecutionContextFrankRegistryObjects{}, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return ResolvedExecutionContextFrankRegistryObjects{}, fmt.Errorf("mission store root is required to resolve Frank object refs")
	}

	resolvedRefs, err := ResolveFrankRegistryObjectRefs(ec.MissionStoreRoot, ec.Step.FrankObjectRefs)
	if err != nil {
		return ResolvedExecutionContextFrankRegistryObjects{}, err
	}

	resolved := ResolvedExecutionContextFrankRegistryObjects{
		ResolvedRefs: resolvedRefs,
	}
	for _, record := range resolvedRefs {
		switch {
		case record.Identity != nil:
			resolved.Identities = append(resolved.Identities, *record.Identity)
		case record.Account != nil:
			resolved.Accounts = append(resolved.Accounts, *record.Account)
		case record.Container != nil:
			resolved.Containers = append(resolved.Containers, *record.Container)
		}
	}

	return resolved, nil
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
	if err := ValidateFrankIdentityLink(root, record.IdentityID); err != nil {
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
	if err := ValidateFrankIdentityLink(root, record.IdentityID); err != nil {
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
