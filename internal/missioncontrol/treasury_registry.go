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

type TreasuryCustodyModel string

const (
	TreasuryCustodyModelFrankContainerRegistry TreasuryCustodyModel = "frank_container_registry"
)

type TreasuryTransactionClass string

const (
	TreasuryTransactionClassAllocate TreasuryTransactionClass = "allocate"
	TreasuryTransactionClassSave     TreasuryTransactionClass = "save"
	TreasuryTransactionClassSpend    TreasuryTransactionClass = "spend"
	TreasuryTransactionClassReinvest TreasuryTransactionClass = "reinvest"
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

// TreasuryObjectView exposes the currently-grounded subset of the frozen V3
// treasury contract without forcing a durable storage migration.
type TreasuryObjectView struct {
	TreasuryID                  string                     `json:"treasury_id"`
	State                       TreasuryState              `json:"state"`
	ZeroSeedPolicy              TreasuryZeroSeedPolicy     `json:"zero_seed_policy"`
	ActiveContainerID           string                     `json:"active_container_id,omitempty"`
	CustodyModel                TreasuryCustodyModel       `json:"custody_model,omitempty"`
	PermittedTransactionClasses []TreasuryTransactionClass `json:"permitted_transaction_classes,omitempty"`
	ForbiddenTransactionClasses []TreasuryTransactionClass `json:"forbidden_transaction_classes,omitempty"`
	LedgerRef                   string                     `json:"ledger_ref"`
	UpdatedAt                   time.Time                  `json:"updated_at"`
}

type TreasuryLedgerDirection string

const (
	TreasuryLedgerDirectionInflow   TreasuryLedgerDirection = "inflow"
	TreasuryLedgerDirectionInternal TreasuryLedgerDirection = "internal"
	TreasuryLedgerDirectionOutflow  TreasuryLedgerDirection = "outflow"
)

type TreasuryLedgerStatus string

const (
	TreasuryLedgerStatusRecorded TreasuryLedgerStatus = "recorded"
)

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

// TreasuryLedgerEntryObjectView exposes the currently-grounded subset of the
// frozen V3 ledger contract without forcing a durable storage migration.
type TreasuryLedgerEntryObjectView struct {
	EntryID     string                  `json:"entry_id"`
	TreasuryID  string                  `json:"treasury_id"`
	ContainerID string                  `json:"container_id,omitempty"`
	EntryClass  TreasuryLedgerEntryKind `json:"entry_class"`
	Asset       string                  `json:"asset"`
	Amount      string                  `json:"amount"`
	Direction   TreasuryLedgerDirection `json:"direction"`
	Source      string                  `json:"source,omitempty"`
	RecordedAt  time.Time               `json:"recorded_at"`
	Status      TreasuryLedgerStatus    `json:"status"`
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
	if err := validateTreasuryActiveContainerContract(record); err != nil {
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
	if err := ValidateTreasuryContainerLinks(root, record.ContainerRefs); err != nil {
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
	record, err := loadTreasuryRecordFile(root, StoreTreasuryPath(root, strings.TrimSpace(treasuryID)))
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
	return listStoreJSONRecords(StoreTreasuriesDir(root), func(path string) (TreasuryRecord, error) {
		return loadTreasuryRecordFile(root, path)
	})
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
	if err := ValidateTreasuryLedgerEntryLink(root, entry); err != nil {
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
	record, err := loadTreasuryLedgerEntryFile(root, StoreTreasuryLedgerEntryPath(root, normalizedTreasuryID, strings.TrimSpace(entryID)), normalizedTreasuryID)
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
		return loadTreasuryLedgerEntryFile(root, path, normalizedTreasuryID)
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

func (record TreasuryRecord) AsObjectView() TreasuryObjectView {
	activeContainerID, _ := TreasuryActiveContainerID(record)
	permitted, forbidden := DefaultTreasuryTransactionPolicy(record.State)
	return TreasuryObjectView{
		TreasuryID:                  record.TreasuryID,
		State:                       record.State,
		ZeroSeedPolicy:              record.ZeroSeedPolicy,
		ActiveContainerID:           activeContainerID,
		CustodyModel:                ResolveTreasuryCustodyModel(record),
		PermittedTransactionClasses: permitted,
		ForbiddenTransactionClasses: forbidden,
		LedgerRef:                   record.TreasuryID,
		UpdatedAt:                   record.UpdatedAt,
	}
}

func ResolveTreasuryLedgerEntryObjectView(root string, entry TreasuryLedgerEntry) (TreasuryLedgerEntryObjectView, error) {
	containerID, err := ResolveTreasuryLedgerEntryContainerID(root, entry)
	if err != nil {
		return TreasuryLedgerEntryObjectView{}, err
	}
	return TreasuryLedgerEntryObjectView{
		EntryID:     entry.EntryID,
		TreasuryID:  entry.TreasuryID,
		ContainerID: containerID,
		EntryClass:  entry.EntryKind,
		Asset:       entry.AssetCode,
		Amount:      entry.Amount,
		Direction:   ResolveTreasuryLedgerEntryDirection(entry),
		Source:      entry.SourceRef,
		RecordedAt:  entry.CreatedAt,
		Status:      ResolveTreasuryLedgerEntryStatus(entry),
	}, nil
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

func loadTreasuryRecordFile(root, path string) (TreasuryRecord, error) {
	var record TreasuryRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return TreasuryRecord{}, err
	}
	record = normalizeTreasuryRecord(record)
	if err := ValidateTreasuryRecord(record); err != nil {
		return TreasuryRecord{}, err
	}
	if err := ValidateTreasuryContainerLinks(root, record.ContainerRefs); err != nil {
		return TreasuryRecord{}, err
	}
	return record, nil
}

func loadTreasuryLedgerEntryFile(root, path string, expectedTreasuryID string) (TreasuryLedgerEntry, error) {
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
	if err := ValidateTreasuryLedgerEntryLink(root, entry); err != nil {
		return TreasuryLedgerEntry{}, err
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

func ValidateTreasuryContainerLinks(root string, refs []FrankRegistryObjectRef) error {
	for _, ref := range refs {
		resolved, err := ResolveFrankRegistryObjectRef(root, ref)
		if err != nil {
			return fmt.Errorf(
				"mission store treasury container_refs ref kind %q object_id %q: %w",
				strings.TrimSpace(string(ref.Kind)),
				strings.TrimSpace(ref.ObjectID),
				err,
			)
		}
		if resolved.Container == nil {
			return fmt.Errorf(
				"mission store treasury container_refs ref kind %q object_id %q: expected Frank container record",
				strings.TrimSpace(string(ref.Kind)),
				strings.TrimSpace(ref.ObjectID),
			)
		}
	}
	return nil
}

func TreasuryActiveContainerID(record TreasuryRecord) (string, bool) {
	if len(record.ContainerRefs) != 1 {
		return "", false
	}
	ref := NormalizeFrankRegistryObjectRef(record.ContainerRefs[0])
	if ref.Kind != FrankRegistryObjectKindContainer || strings.TrimSpace(ref.ObjectID) == "" {
		return "", false
	}
	return ref.ObjectID, true
}

func ResolveTreasuryCustodyModel(record TreasuryRecord) TreasuryCustodyModel {
	if len(record.ContainerRefs) == 0 {
		return ""
	}
	return TreasuryCustodyModelFrankContainerRegistry
}

func DefaultTreasuryTransactionPolicy(state TreasuryState) ([]TreasuryTransactionClass, []TreasuryTransactionClass) {
	all := []TreasuryTransactionClass{
		TreasuryTransactionClassAllocate,
		TreasuryTransactionClassSave,
		TreasuryTransactionClassSpend,
		TreasuryTransactionClassReinvest,
	}
	if NormalizeTreasuryState(state) == TreasuryStateActive {
		return append([]TreasuryTransactionClass(nil), all...), nil
	}
	return nil, append([]TreasuryTransactionClass(nil), all...)
}

func ResolveTreasuryLedgerEntryContainerID(root string, entry TreasuryLedgerEntry) (string, error) {
	treasury, err := LoadTreasuryRecord(root, entry.TreasuryID)
	if err != nil {
		return "", fmt.Errorf("mission store treasury ledger entry treasury_id %q: %w", strings.TrimSpace(entry.TreasuryID), err)
	}
	containerID, ok := TreasuryActiveContainerID(treasury)
	if !ok {
		switch len(treasury.ContainerRefs) {
		case 0:
			return "", fmt.Errorf("mission store treasury ledger entry treasury_id %q has no active treasury container", treasury.TreasuryID)
		default:
			return "", fmt.Errorf(
				"mission store treasury ledger entry treasury_id %q has ambiguous active treasury container across %d container_refs",
				treasury.TreasuryID,
				len(treasury.ContainerRefs),
			)
		}
	}
	return containerID, nil
}

func ValidateTreasuryLedgerEntryLink(root string, entry TreasuryLedgerEntry) error {
	if _, err := ResolveTreasuryLedgerEntryContainerID(root, entry); err != nil {
		return err
	}
	return nil
}

func ResolveTreasuryLedgerEntryDirection(entry TreasuryLedgerEntry) TreasuryLedgerDirection {
	switch NormalizeTreasuryLedgerEntryKind(entry.EntryKind) {
	case TreasuryLedgerEntryKindAcquisition:
		return TreasuryLedgerDirectionInflow
	case TreasuryLedgerEntryKindMovement:
		return TreasuryLedgerDirectionInternal
	case TreasuryLedgerEntryKindDisposition:
		return TreasuryLedgerDirectionOutflow
	default:
		return ""
	}
}

func ResolveTreasuryLedgerEntryStatus(entry TreasuryLedgerEntry) TreasuryLedgerStatus {
	if NormalizeTreasuryLedgerEntryKind(entry.EntryKind) == "" {
		return ""
	}
	return TreasuryLedgerStatusRecorded
}

func validateTreasuryActiveContainerContract(record TreasuryRecord) error {
	switch NormalizeTreasuryState(record.State) {
	case TreasuryStateFunded, TreasuryStateActive:
		if _, ok := TreasuryActiveContainerID(record); !ok {
			return fmt.Errorf(
				"mission store treasury state %q requires exactly one active_container_id derivable from container_refs",
				record.State,
			)
		}
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
