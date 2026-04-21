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

type RuntimePackRef struct {
	PackID string `json:"pack_id"`
}

type RuntimePackRecord struct {
	RecordVersion            int       `json:"record_version"`
	PackID                   string    `json:"pack_id"`
	ParentPackID             string    `json:"parent_pack_id,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	Channel                  string    `json:"channel"`
	PromptPackRef            string    `json:"prompt_pack_ref"`
	SkillPackRef             string    `json:"skill_pack_ref"`
	ManifestRef              string    `json:"manifest_ref"`
	ExtensionPackRef         string    `json:"extension_pack_ref"`
	PolicyRef                string    `json:"policy_ref"`
	SourceSummary            string    `json:"source_summary"`
	MutableSurfaces          []string  `json:"mutable_surfaces"`
	ImmutableSurfaces        []string  `json:"immutable_surfaces"`
	SurfaceClasses           []string  `json:"surface_classes"`
	CompatibilityContractRef string    `json:"compatibility_contract_ref"`
	RollbackTargetPackID     string    `json:"rollback_target_pack_id,omitempty"`
}

type ActiveRuntimePackPointer struct {
	RecordVersion        int       `json:"record_version"`
	ActivePackID         string    `json:"active_pack_id"`
	PreviousActivePackID string    `json:"previous_active_pack_id,omitempty"`
	LastKnownGoodPackID  string    `json:"last_known_good_pack_id,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
	UpdatedBy            string    `json:"updated_by"`
	UpdateRecordRef      string    `json:"update_record_ref"`
	ReloadGeneration     uint64    `json:"reload_generation"`
}

type LastKnownGoodRuntimePackPointer struct {
	RecordVersion     int       `json:"record_version"`
	PackID            string    `json:"pack_id"`
	Basis             string    `json:"basis"`
	VerifiedAt        time.Time `json:"verified_at"`
	VerifiedBy        string    `json:"verified_by"`
	RollbackRecordRef string    `json:"rollback_record_ref"`
}

var (
	ErrActiveRuntimePackPointerNotFound        = errors.New("mission store active runtime pack pointer not found")
	ErrLastKnownGoodRuntimePackPointerNotFound = errors.New("mission store last-known-good runtime pack pointer not found")
	ErrRuntimePackRecordNotFound               = errors.New("mission store runtime pack record not found")
)

func StoreRuntimePacksDir(root string) string {
	return filepath.Join(root, "runtime_packs", "packs")
}

func StoreRuntimePackPath(root, packID string) string {
	return filepath.Join(StoreRuntimePacksDir(root), strings.TrimSpace(packID)+".json")
}

func StoreActiveRuntimePackPointerPath(root string) string {
	return filepath.Join(root, "runtime_packs", "active_pointer.json")
}

func StoreLastKnownGoodRuntimePackPointerPath(root string) string {
	return filepath.Join(root, "runtime_packs", "last_known_good_pointer.json")
}

func NormalizeRuntimePackRef(ref RuntimePackRef) RuntimePackRef {
	ref.PackID = strings.TrimSpace(ref.PackID)
	return ref
}

func NormalizeRuntimePackRecord(record RuntimePackRecord) RuntimePackRecord {
	record.PackID = strings.TrimSpace(record.PackID)
	record.ParentPackID = strings.TrimSpace(record.ParentPackID)
	record.CreatedAt = record.CreatedAt.UTC()
	record.Channel = strings.TrimSpace(record.Channel)
	record.PromptPackRef = strings.TrimSpace(record.PromptPackRef)
	record.SkillPackRef = strings.TrimSpace(record.SkillPackRef)
	record.ManifestRef = strings.TrimSpace(record.ManifestRef)
	record.ExtensionPackRef = strings.TrimSpace(record.ExtensionPackRef)
	record.PolicyRef = strings.TrimSpace(record.PolicyRef)
	record.SourceSummary = strings.TrimSpace(record.SourceSummary)
	record.MutableSurfaces = normalizeRuntimePackStrings(record.MutableSurfaces)
	record.ImmutableSurfaces = normalizeRuntimePackStrings(record.ImmutableSurfaces)
	record.SurfaceClasses = normalizeRuntimePackStrings(record.SurfaceClasses)
	record.CompatibilityContractRef = strings.TrimSpace(record.CompatibilityContractRef)
	record.RollbackTargetPackID = strings.TrimSpace(record.RollbackTargetPackID)
	return record
}

func NormalizeActiveRuntimePackPointer(pointer ActiveRuntimePackPointer) ActiveRuntimePackPointer {
	pointer.ActivePackID = strings.TrimSpace(pointer.ActivePackID)
	pointer.PreviousActivePackID = strings.TrimSpace(pointer.PreviousActivePackID)
	pointer.LastKnownGoodPackID = strings.TrimSpace(pointer.LastKnownGoodPackID)
	pointer.UpdatedAt = pointer.UpdatedAt.UTC()
	pointer.UpdatedBy = strings.TrimSpace(pointer.UpdatedBy)
	pointer.UpdateRecordRef = strings.TrimSpace(pointer.UpdateRecordRef)
	return pointer
}

func NormalizeLastKnownGoodRuntimePackPointer(pointer LastKnownGoodRuntimePackPointer) LastKnownGoodRuntimePackPointer {
	pointer.PackID = strings.TrimSpace(pointer.PackID)
	pointer.Basis = strings.TrimSpace(pointer.Basis)
	pointer.VerifiedAt = pointer.VerifiedAt.UTC()
	pointer.VerifiedBy = strings.TrimSpace(pointer.VerifiedBy)
	pointer.RollbackRecordRef = strings.TrimSpace(pointer.RollbackRecordRef)
	return pointer
}

func ValidateRuntimePackRef(ref RuntimePackRef) error {
	return validateRuntimePackIDField("runtime pack ref", "pack_id", ref.PackID)
}

func ValidateRuntimePackRecord(record RuntimePackRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime pack record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store runtime pack", "pack_id", record.PackID); err != nil {
		return err
	}
	if record.ParentPackID != "" {
		if err := validateRuntimePackIDField("mission store runtime pack", "parent_pack_id", record.ParentPackID); err != nil {
			return err
		}
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store runtime pack created_at is required")
	}
	if record.Channel == "" {
		return fmt.Errorf("mission store runtime pack channel is required")
	}
	if record.PromptPackRef == "" {
		return fmt.Errorf("mission store runtime pack prompt_pack_ref is required")
	}
	if record.SkillPackRef == "" {
		return fmt.Errorf("mission store runtime pack skill_pack_ref is required")
	}
	if record.ManifestRef == "" {
		return fmt.Errorf("mission store runtime pack manifest_ref is required")
	}
	if record.ExtensionPackRef == "" {
		return fmt.Errorf("mission store runtime pack extension_pack_ref is required")
	}
	if record.PolicyRef == "" {
		return fmt.Errorf("mission store runtime pack policy_ref is required")
	}
	if record.SourceSummary == "" {
		return fmt.Errorf("mission store runtime pack source_summary is required")
	}
	if len(record.MutableSurfaces) == 0 {
		return fmt.Errorf("mission store runtime pack mutable_surfaces are required")
	}
	if len(record.ImmutableSurfaces) == 0 {
		return fmt.Errorf("mission store runtime pack immutable_surfaces are required")
	}
	if len(record.SurfaceClasses) == 0 {
		return fmt.Errorf("mission store runtime pack surface_classes are required")
	}
	if record.CompatibilityContractRef == "" {
		return fmt.Errorf("mission store runtime pack compatibility_contract_ref is required")
	}
	if record.RollbackTargetPackID != "" {
		if err := validateRuntimePackIDField("mission store runtime pack", "rollback_target_pack_id", record.RollbackTargetPackID); err != nil {
			return err
		}
	}
	return nil
}

func ValidateActiveRuntimePackPointer(pointer ActiveRuntimePackPointer) error {
	if pointer.RecordVersion <= 0 {
		return fmt.Errorf("mission store active runtime pack pointer record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store active runtime pack pointer", "active_pack_id", pointer.ActivePackID); err != nil {
		return err
	}
	if pointer.PreviousActivePackID != "" {
		if err := validateRuntimePackIDField("mission store active runtime pack pointer", "previous_active_pack_id", pointer.PreviousActivePackID); err != nil {
			return err
		}
	}
	if pointer.LastKnownGoodPackID != "" {
		if err := validateRuntimePackIDField("mission store active runtime pack pointer", "last_known_good_pack_id", pointer.LastKnownGoodPackID); err != nil {
			return err
		}
	}
	if pointer.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store active runtime pack pointer updated_at is required")
	}
	if pointer.UpdatedBy == "" {
		return fmt.Errorf("mission store active runtime pack pointer updated_by is required")
	}
	if pointer.UpdateRecordRef == "" {
		return fmt.Errorf("mission store active runtime pack pointer update_record_ref is required")
	}
	return nil
}

func ValidateLastKnownGoodRuntimePackPointer(pointer LastKnownGoodRuntimePackPointer) error {
	if pointer.RecordVersion <= 0 {
		return fmt.Errorf("mission store last-known-good runtime pack pointer record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store last-known-good runtime pack pointer", "pack_id", pointer.PackID); err != nil {
		return err
	}
	if pointer.Basis == "" {
		return fmt.Errorf("mission store last-known-good runtime pack pointer basis is required")
	}
	if pointer.VerifiedAt.IsZero() {
		return fmt.Errorf("mission store last-known-good runtime pack pointer verified_at is required")
	}
	if pointer.VerifiedBy == "" {
		return fmt.Errorf("mission store last-known-good runtime pack pointer verified_by is required")
	}
	if pointer.RollbackRecordRef == "" {
		return fmt.Errorf("mission store last-known-good runtime pack pointer rollback_record_ref is required")
	}
	return nil
}

func StoreRuntimePackRecord(root string, record RuntimePackRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeRuntimePackRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRuntimePackRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreRuntimePackPath(root, record.PackID), record)
}

func LoadRuntimePackRecord(root, packID string) (RuntimePackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimePackRecord{}, err
	}
	if err := validateRuntimePackIDField("mission store runtime pack", "pack_id", packID); err != nil {
		return RuntimePackRecord{}, err
	}
	record, err := loadRuntimePackRecordFile(StoreRuntimePackPath(root, strings.TrimSpace(packID)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimePackRecord{}, ErrRuntimePackRecordNotFound
		}
		return RuntimePackRecord{}, err
	}
	return record, nil
}

func ListRuntimePackRecords(root string) ([]RuntimePackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRuntimePacksDir(root), loadRuntimePackRecordFile)
}

func StoreActiveRuntimePackPointer(root string, pointer ActiveRuntimePackPointer) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	pointer = NormalizeActiveRuntimePackPointer(pointer)
	pointer.RecordVersion = normalizeRecordVersion(pointer.RecordVersion)
	if err := ValidateActiveRuntimePackPointer(pointer); err != nil {
		return err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.ActivePackID); err != nil {
		return fmt.Errorf("mission store active runtime pack pointer active_pack_id %q: %w", pointer.ActivePackID, err)
	}
	if pointer.PreviousActivePackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.PreviousActivePackID); err != nil {
			return fmt.Errorf("mission store active runtime pack pointer previous_active_pack_id %q: %w", pointer.PreviousActivePackID, err)
		}
	}
	if pointer.LastKnownGoodPackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.LastKnownGoodPackID); err != nil {
			return fmt.Errorf("mission store active runtime pack pointer last_known_good_pack_id %q: %w", pointer.LastKnownGoodPackID, err)
		}
	}
	return WriteStoreJSONAtomic(StoreActiveRuntimePackPointerPath(root), pointer)
}

func LoadActiveRuntimePackPointer(root string) (ActiveRuntimePackPointer, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ActiveRuntimePackPointer{}, err
	}
	var pointer ActiveRuntimePackPointer
	if err := LoadStoreJSON(StoreActiveRuntimePackPointerPath(root), &pointer); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ActiveRuntimePackPointer{}, ErrActiveRuntimePackPointerNotFound
		}
		return ActiveRuntimePackPointer{}, err
	}
	pointer = NormalizeActiveRuntimePackPointer(pointer)
	if err := ValidateActiveRuntimePackPointer(pointer); err != nil {
		return ActiveRuntimePackPointer{}, err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.ActivePackID); err != nil {
		return ActiveRuntimePackPointer{}, fmt.Errorf("mission store active runtime pack pointer active_pack_id %q: %w", pointer.ActivePackID, err)
	}
	if pointer.PreviousActivePackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.PreviousActivePackID); err != nil {
			return ActiveRuntimePackPointer{}, fmt.Errorf("mission store active runtime pack pointer previous_active_pack_id %q: %w", pointer.PreviousActivePackID, err)
		}
	}
	if pointer.LastKnownGoodPackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.LastKnownGoodPackID); err != nil {
			return ActiveRuntimePackPointer{}, fmt.Errorf("mission store active runtime pack pointer last_known_good_pack_id %q: %w", pointer.LastKnownGoodPackID, err)
		}
	}
	return pointer, nil
}

func StoreLastKnownGoodRuntimePackPointer(root string, pointer LastKnownGoodRuntimePackPointer) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	pointer = NormalizeLastKnownGoodRuntimePackPointer(pointer)
	pointer.RecordVersion = normalizeRecordVersion(pointer.RecordVersion)
	if err := ValidateLastKnownGoodRuntimePackPointer(pointer); err != nil {
		return err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.PackID); err != nil {
		return fmt.Errorf("mission store last-known-good runtime pack pointer pack_id %q: %w", pointer.PackID, err)
	}
	return WriteStoreJSONAtomic(StoreLastKnownGoodRuntimePackPointerPath(root), pointer)
}

func LoadLastKnownGoodRuntimePackPointer(root string) (LastKnownGoodRuntimePackPointer, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LastKnownGoodRuntimePackPointer{}, err
	}
	var pointer LastKnownGoodRuntimePackPointer
	if err := LoadStoreJSON(StoreLastKnownGoodRuntimePackPointerPath(root), &pointer); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LastKnownGoodRuntimePackPointer{}, ErrLastKnownGoodRuntimePackPointerNotFound
		}
		return LastKnownGoodRuntimePackPointer{}, err
	}
	pointer = NormalizeLastKnownGoodRuntimePackPointer(pointer)
	if err := ValidateLastKnownGoodRuntimePackPointer(pointer); err != nil {
		return LastKnownGoodRuntimePackPointer{}, err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.PackID); err != nil {
		return LastKnownGoodRuntimePackPointer{}, fmt.Errorf("mission store last-known-good runtime pack pointer pack_id %q: %w", pointer.PackID, err)
	}
	return pointer, nil
}

func ResolveActiveRuntimePackRecord(root string) (RuntimePackRecord, error) {
	pointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return RuntimePackRecord{}, err
	}
	return LoadRuntimePackRecord(root, pointer.ActivePackID)
}

func ResolveLastKnownGoodRuntimePackRecord(root string) (RuntimePackRecord, error) {
	pointer, err := LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil {
		return RuntimePackRecord{}, err
	}
	return LoadRuntimePackRecord(root, pointer.PackID)
}

func loadRuntimePackRecordFile(path string) (RuntimePackRecord, error) {
	var record RuntimePackRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimePackRecord{}, err
	}
	record = NormalizeRuntimePackRecord(record)
	if err := ValidateRuntimePackRecord(record); err != nil {
		return RuntimePackRecord{}, err
	}
	return record, nil
}

func normalizeRuntimePackStrings(values []string) []string {
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

func validateRuntimePackIDField(surface, fieldName, value string) error {
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
