package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type TreasuryState string

const (
	TreasuryStateUnfunded  TreasuryState = "unfunded"
	TreasuryStateBootstrap TreasuryState = "bootstrap"
	TreasuryStateFunded    TreasuryState = "funded"
	TreasuryStateActive    TreasuryState = "active"
	TreasuryStateSuspended TreasuryState = "suspended"
)

type TreasuryZeroSeedPolicy string

const (
	TreasuryZeroSeedPolicyOwnerSeedForbidden TreasuryZeroSeedPolicy = "owner_seed_forbidden"
)

type TreasuryLedgerEntryKind string

const (
	TreasuryLedgerEntryKindAcquisition TreasuryLedgerEntryKind = "acquisition"
	TreasuryLedgerEntryKindMovement    TreasuryLedgerEntryKind = "movement"
	TreasuryLedgerEntryKindDisposition TreasuryLedgerEntryKind = "disposition"
)

type TreasuryRecord struct {
	RecordVersion  int                      `json:"record_version"`
	TreasuryID     string                   `json:"treasury_id"`
	DisplayName    string                   `json:"display_name"`
	State          TreasuryState            `json:"state"`
	ZeroSeedPolicy TreasuryZeroSeedPolicy   `json:"zero_seed_policy"`
	ContainerRefs  []FrankRegistryObjectRef `json:"container_refs,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type TreasuryLedgerEntry struct {
	RecordVersion int                     `json:"record_version"`
	EntryID       string                  `json:"entry_id"`
	TreasuryID    string                  `json:"treasury_id"`
	EntryKind     TreasuryLedgerEntryKind `json:"entry_kind"`
	AssetCode     string                  `json:"asset_code"`
	Amount        string                  `json:"amount"`
	CreatedAt     time.Time               `json:"created_at"`
	SourceRef     string                  `json:"source_ref,omitempty"`
}

type ResolvedExecutionContextTreasuryPreflight struct {
	Treasury   *TreasuryRecord        `json:"treasury,omitempty"`
	Containers []FrankContainerRecord `json:"containers,omitempty"`
}

var (
	ErrTreasuryLedgerEntryNotFound = errors.New("mission store treasury ledger entry not found")
	ErrTreasuryRecordNotFound      = errors.New("mission store treasury record not found")
)

var treasuryAmountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

func StoreTreasuriesDir(root string) string {
	return filepath.Join(root, "treasuries")
}

func StoreTreasuryPath(root, treasuryID string) string {
	return filepath.Join(StoreTreasuriesDir(root), treasuryID+".json")
}

func StoreTreasuryLedgersDir(root string) string {
	return filepath.Join(root, "treasury_ledgers")
}

func StoreTreasuryLedgerDir(root, treasuryID string) string {
	return filepath.Join(StoreTreasuryLedgersDir(root), treasuryID)
}

func StoreTreasuryLedgerEntryPath(root, treasuryID, entryID string) string {
	return filepath.Join(StoreTreasuryLedgerDir(root, treasuryID), entryID+".json")
}

func NormalizeTreasuryState(state TreasuryState) TreasuryState {
	return TreasuryState(strings.TrimSpace(string(state)))
}

func NormalizeTreasuryZeroSeedPolicy(policy TreasuryZeroSeedPolicy) TreasuryZeroSeedPolicy {
	return TreasuryZeroSeedPolicy(strings.TrimSpace(string(policy)))
}

func NormalizeTreasuryLedgerEntryKind(kind TreasuryLedgerEntryKind) TreasuryLedgerEntryKind {
	return TreasuryLedgerEntryKind(strings.TrimSpace(string(kind)))
}

func ValidateTreasuryRecord(record TreasuryRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store treasury record_version must be positive")
	}
	if err := validateTreasuryID(record.TreasuryID, "mission store treasury"); err != nil {
		return err
	}
	if strings.TrimSpace(record.DisplayName) == "" {
		return fmt.Errorf("mission store treasury display_name is required")
	}
	if !isValidTreasuryState(record.State) {
		return fmt.Errorf("mission store treasury state %q is invalid", strings.TrimSpace(string(record.State)))
	}
	if !isValidTreasuryZeroSeedPolicy(record.ZeroSeedPolicy) {
		return fmt.Errorf("mission store treasury zero_seed_policy %q is invalid", strings.TrimSpace(string(record.ZeroSeedPolicy)))
	}
	if err := validateTreasuryContainerRefs(record.ContainerRefs); err != nil {
		return err
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store treasury created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store treasury updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store treasury updated_at must be on or after created_at")
	}
	return nil
}

func ValidateTreasuryLedgerEntry(entry TreasuryLedgerEntry) error {
	if entry.RecordVersion <= 0 {
		return fmt.Errorf("mission store treasury ledger entry record_version must be positive")
	}
	if err := validateTreasuryEntryID(entry.EntryID, "mission store treasury ledger entry"); err != nil {
		return err
	}
	if err := validateTreasuryID(entry.TreasuryID, "mission store treasury ledger entry"); err != nil {
		return err
	}
	if !isValidTreasuryLedgerEntryKind(entry.EntryKind) {
		return fmt.Errorf("mission store treasury ledger entry entry_kind %q is invalid", strings.TrimSpace(string(entry.EntryKind)))
	}
	if err := validateTreasuryAssetCode(entry.AssetCode); err != nil {
		return err
	}
	if err := validateTreasuryAmount(entry.Amount); err != nil {
		return err
	}
	if entry.CreatedAt.IsZero() {
		return fmt.Errorf("mission store treasury ledger entry created_at is required")
	}
	return nil
}

func StoreTreasuryRecord(root string, record TreasuryRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = normalizeTreasuryRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateTreasuryRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreTreasuryPath(root, record.TreasuryID), record)
}

func LoadTreasuryRecord(root, treasuryID string) (TreasuryRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return TreasuryRecord{}, err
	}
	if err := validateTreasuryID(treasuryID, "mission store treasury"); err != nil {
		return TreasuryRecord{}, err
	}
	record, err := loadTreasuryRecordFile(StoreTreasuryPath(root, strings.TrimSpace(treasuryID)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TreasuryRecord{}, ErrTreasuryRecordNotFound
		}
		return TreasuryRecord{}, err
	}
	return record, nil
}

func ListTreasuryRecords(root string) ([]TreasuryRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreTreasuriesDir(root), loadTreasuryRecordFile)
}

func StoreTreasuryLedgerEntry(root string, entry TreasuryLedgerEntry) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	entry = normalizeTreasuryLedgerEntry(entry)
	entry.RecordVersion = normalizeRecordVersion(entry.RecordVersion)
	if err := ValidateTreasuryLedgerEntry(entry); err != nil {
		return err
	}

	path := StoreTreasuryLedgerEntryPath(root, entry.TreasuryID, entry.EntryID)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("mission store treasury ledger entry %q already exists", entry.EntryID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return WriteStoreJSONAtomic(path, entry)
}

func LoadTreasuryLedgerEntry(root, treasuryID, entryID string) (TreasuryLedgerEntry, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return TreasuryLedgerEntry{}, err
	}
	if err := validateTreasuryID(treasuryID, "mission store treasury ledger entry"); err != nil {
		return TreasuryLedgerEntry{}, err
	}
	if err := validateTreasuryEntryID(entryID, "mission store treasury ledger entry"); err != nil {
		return TreasuryLedgerEntry{}, err
	}
	normalizedTreasuryID := strings.TrimSpace(treasuryID)
	record, err := loadTreasuryLedgerEntryFile(StoreTreasuryLedgerEntryPath(root, normalizedTreasuryID, strings.TrimSpace(entryID)), normalizedTreasuryID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TreasuryLedgerEntry{}, ErrTreasuryLedgerEntryNotFound
		}
		return TreasuryLedgerEntry{}, err
	}
	return record, nil
}

func ListTreasuryLedgerEntries(root, treasuryID string) ([]TreasuryLedgerEntry, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	if err := validateTreasuryID(treasuryID, "mission store treasury ledger entry"); err != nil {
		return nil, err
	}
	normalizedTreasuryID := strings.TrimSpace(treasuryID)
	return listStoreJSONRecords(StoreTreasuryLedgerDir(root, normalizedTreasuryID), func(path string) (TreasuryLedgerEntry, error) {
		return loadTreasuryLedgerEntryFile(path, normalizedTreasuryID)
	})
}

func normalizeTreasuryRecord(record TreasuryRecord) TreasuryRecord {
	record.TreasuryID = strings.TrimSpace(record.TreasuryID)
	record.DisplayName = strings.TrimSpace(record.DisplayName)
	record.State = NormalizeTreasuryState(record.State)
	record.ZeroSeedPolicy = NormalizeTreasuryZeroSeedPolicy(record.ZeroSeedPolicy)
	record.ContainerRefs = normalizeFrankRegistryObjectRefs(record.ContainerRefs)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	return record
}

func normalizeTreasuryLedgerEntry(entry TreasuryLedgerEntry) TreasuryLedgerEntry {
	entry.EntryID = strings.TrimSpace(entry.EntryID)
	entry.TreasuryID = strings.TrimSpace(entry.TreasuryID)
	entry.EntryKind = NormalizeTreasuryLedgerEntryKind(entry.EntryKind)
	entry.AssetCode = strings.TrimSpace(entry.AssetCode)
	entry.Amount = strings.TrimSpace(entry.Amount)
	entry.SourceRef = strings.TrimSpace(entry.SourceRef)
	entry.CreatedAt = entry.CreatedAt.UTC()
	return entry
}

func ValidateTreasuryRef(ref TreasuryRef) error {
	return validateTreasuryIdentifierValue("treasury_id", NormalizeTreasuryRef(ref).TreasuryID)
}

func ResolveTreasuryRef(root string, ref TreasuryRef) (TreasuryRecord, error) {
	normalized := NormalizeTreasuryRef(ref)
	if err := ValidateTreasuryRef(normalized); err != nil {
		return TreasuryRecord{}, err
	}
	return LoadTreasuryRecord(root, normalized.TreasuryID)
}

func ResolveExecutionContextTreasuryRef(ec ExecutionContext) (*TreasuryRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if ec.Step.TreasuryRef == nil {
		return nil, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return nil, fmt.Errorf("mission store root is required to resolve treasury refs")
	}

	record, err := ResolveTreasuryRef(ec.MissionStoreRoot, *ec.Step.TreasuryRef)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func ResolveExecutionContextTreasuryPreflight(ec ExecutionContext) (ResolvedExecutionContextTreasuryPreflight, error) {
	treasury, err := ResolveExecutionContextTreasuryRef(ec)
	if err != nil {
		return ResolvedExecutionContextTreasuryPreflight{}, err
	}
	if treasury == nil {
		return ResolvedExecutionContextTreasuryPreflight{}, nil
	}

	resolvedRefs, err := ResolveFrankRegistryObjectRefs(ec.MissionStoreRoot, treasury.ContainerRefs)
	if err != nil {
		return ResolvedExecutionContextTreasuryPreflight{}, err
	}

	preflight := ResolvedExecutionContextTreasuryPreflight{
		Treasury: treasury,
	}
	if len(resolvedRefs) == 0 {
		return preflight, nil
	}

	preflight.Containers = make([]FrankContainerRecord, 0, len(resolvedRefs))
	for _, resolved := range resolvedRefs {
		if resolved.Container == nil {
			return ResolvedExecutionContextTreasuryPreflight{}, fmt.Errorf("resolve treasury container ref kind %q object_id %q: expected Frank container record", resolved.Ref.Kind, resolved.Ref.ObjectID)
		}
		preflight.Containers = append(preflight.Containers, *resolved.Container)
	}

	return preflight, nil
}

func loadTreasuryRecordFile(path string) (TreasuryRecord, error) {
	var record TreasuryRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return TreasuryRecord{}, err
	}
	record = normalizeTreasuryRecord(record)
	if err := ValidateTreasuryRecord(record); err != nil {
		return TreasuryRecord{}, err
	}
	return record, nil
}

func loadTreasuryLedgerEntryFile(path string, expectedTreasuryID string) (TreasuryLedgerEntry, error) {
	var entry TreasuryLedgerEntry
	if err := LoadStoreJSON(path, &entry); err != nil {
		return TreasuryLedgerEntry{}, err
	}
	entry = normalizeTreasuryLedgerEntry(entry)
	if err := ValidateTreasuryLedgerEntry(entry); err != nil {
		return TreasuryLedgerEntry{}, err
	}
	if entry.TreasuryID != strings.TrimSpace(expectedTreasuryID) {
		return TreasuryLedgerEntry{}, fmt.Errorf("mission store treasury ledger entry treasury_id %q does not match ledger %q", entry.TreasuryID, strings.TrimSpace(expectedTreasuryID))
	}
	return entry, nil
}

func isValidTreasuryState(state TreasuryState) bool {
	switch NormalizeTreasuryState(state) {
	case TreasuryStateUnfunded, TreasuryStateBootstrap, TreasuryStateFunded, TreasuryStateActive, TreasuryStateSuspended:
		return true
	default:
		return false
	}
}

func isValidTreasuryZeroSeedPolicy(policy TreasuryZeroSeedPolicy) bool {
	switch NormalizeTreasuryZeroSeedPolicy(policy) {
	case TreasuryZeroSeedPolicyOwnerSeedForbidden:
		return true
	default:
		return false
	}
}

func isValidTreasuryLedgerEntryKind(kind TreasuryLedgerEntryKind) bool {
	switch NormalizeTreasuryLedgerEntryKind(kind) {
	case TreasuryLedgerEntryKindAcquisition, TreasuryLedgerEntryKindMovement, TreasuryLedgerEntryKindDisposition:
		return true
	default:
		return false
	}
}

func validateTreasuryContainerRefs(refs []FrankRegistryObjectRef) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		normalized := NormalizeFrankRegistryObjectRef(ref)
		if err := validateFrankRegistryObjectRef(normalized); err != nil {
			return fmt.Errorf("mission store treasury container_refs contain invalid ref: %w", err)
		}
		if normalized.Kind != FrankRegistryObjectKindContainer {
			return fmt.Errorf("mission store treasury container_refs require kind %q, got %q", FrankRegistryObjectKindContainer, normalized.Kind)
		}
		key := normalizedFrankRegistryObjectRefKey(normalized)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("mission store treasury container_refs contain duplicate ref kind %q object_id %q", normalized.Kind, normalized.ObjectID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateTreasuryAmount(amount string) error {
	normalized := strings.TrimSpace(amount)
	if normalized == "" {
		return fmt.Errorf("mission store treasury ledger entry amount is required")
	}
	if !treasuryAmountPattern.MatchString(normalized) {
		return fmt.Errorf("mission store treasury ledger entry amount %q is invalid", normalized)
	}
	return nil
}

func validateTreasuryAssetCode(assetCode string) error {
	normalized := strings.TrimSpace(assetCode)
	if normalized == "" {
		return fmt.Errorf("mission store treasury ledger entry asset_code is required")
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("mission store treasury ledger entry asset_code %q is invalid", normalized)
		}
	}
	return nil
}

func validateTreasuryID(treasuryID string, surface string) error {
	if err := validateTreasuryIdentifierValue("treasury_id", treasuryID); err != nil {
		return fmt.Errorf("%s %w", surface, err)
	}
	return nil
}

func validateTreasuryEntryID(entryID string, surface string) error {
	if err := validateTreasuryIdentifierValue("entry_id", entryID); err != nil {
		return fmt.Errorf("%s %w", surface, err)
	}
	return nil
}

func validateTreasuryIdentifierValue(fieldName string, value string) error {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if normalized == "." || normalized == ".." {
		return fmt.Errorf("%s %q is invalid", fieldName, normalized)
	}
	if strings.ContainsAny(normalized, `/\`) {
		return fmt.Errorf("%s %q is invalid", fieldName, normalized)
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%s %q is invalid", fieldName, normalized)
		}
	}
	return nil
}
