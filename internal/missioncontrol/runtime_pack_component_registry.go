package missioncontrol

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type RuntimePackComponentKind string

const (
	RuntimePackComponentKindPromptPack    RuntimePackComponentKind = "prompt_pack"
	RuntimePackComponentKindSkillPack     RuntimePackComponentKind = "skill_pack"
	RuntimePackComponentKindManifestPack  RuntimePackComponentKind = "manifest_pack"
	RuntimePackComponentKindExtensionPack RuntimePackComponentKind = "extension_pack"
)

type RuntimePackComponentRef struct {
	Kind        RuntimePackComponentKind `json:"kind"`
	ComponentID string                   `json:"component_id"`
}

type RuntimePackComponentRecord struct {
	RecordVersion     int                      `json:"record_version"`
	Kind              RuntimePackComponentKind `json:"kind"`
	ComponentID       string                   `json:"component_id"`
	ParentComponentID string                   `json:"parent_component_id,omitempty"`
	ContentRef        string                   `json:"content_ref"`
	ContentSHA256     string                   `json:"content_sha256"`
	SurfaceClass      string                   `json:"surface_class,omitempty"`
	DeclaredSurfaces  []string                 `json:"declared_surfaces,omitempty"`
	HotReloadable     bool                     `json:"hot_reloadable,omitempty"`
	SourceSummary     string                   `json:"source_summary"`
	ProvenanceRef     string                   `json:"provenance_ref"`
	CreatedAt         time.Time                `json:"created_at"`
	CreatedBy         string                   `json:"created_by"`
}

type ResolvedRuntimePackComponents struct {
	PromptPack    RuntimePackComponentRecord `json:"prompt_pack"`
	SkillPack     RuntimePackComponentRecord `json:"skill_pack"`
	ManifestPack  RuntimePackComponentRecord `json:"manifest_pack"`
	ExtensionPack RuntimePackComponentRecord `json:"extension_pack"`
}

var ErrRuntimePackComponentRecordNotFound = errors.New("mission store runtime pack component record not found")

func StoreRuntimePackComponentsDir(root string, kind RuntimePackComponentKind) string {
	return filepath.Join(root, "runtime_packs", "components", string(kind))
}

func StoreRuntimePackComponentPath(root string, kind RuntimePackComponentKind, componentID string) string {
	return filepath.Join(StoreRuntimePackComponentsDir(root, kind), strings.TrimSpace(componentID)+".json")
}

func NormalizeRuntimePackComponentRef(ref RuntimePackComponentRef) RuntimePackComponentRef {
	ref.Kind = RuntimePackComponentKind(strings.TrimSpace(string(ref.Kind)))
	ref.ComponentID = strings.TrimSpace(ref.ComponentID)
	return ref
}

func NormalizeRuntimePackComponentRecord(record RuntimePackComponentRecord) RuntimePackComponentRecord {
	record.Kind = RuntimePackComponentKind(strings.TrimSpace(string(record.Kind)))
	record.ComponentID = strings.TrimSpace(record.ComponentID)
	record.ParentComponentID = strings.TrimSpace(record.ParentComponentID)
	record.ContentRef = strings.TrimSpace(record.ContentRef)
	record.ContentSHA256 = strings.ToLower(strings.TrimSpace(record.ContentSHA256))
	record.SurfaceClass = strings.TrimSpace(record.SurfaceClass)
	record.DeclaredSurfaces = normalizeRuntimePackStrings(record.DeclaredSurfaces)
	record.SourceSummary = strings.TrimSpace(record.SourceSummary)
	record.ProvenanceRef = strings.TrimSpace(record.ProvenanceRef)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func RuntimePackComponentRefs(record RuntimePackRecord) []RuntimePackComponentRef {
	record = NormalizeRuntimePackRecord(record)
	return []RuntimePackComponentRef{
		{Kind: RuntimePackComponentKindPromptPack, ComponentID: record.PromptPackRef},
		{Kind: RuntimePackComponentKindSkillPack, ComponentID: record.SkillPackRef},
		{Kind: RuntimePackComponentKindManifestPack, ComponentID: record.ManifestRef},
		{Kind: RuntimePackComponentKindExtensionPack, ComponentID: record.ExtensionPackRef},
	}
}

func ValidateRuntimePackComponentKind(kind RuntimePackComponentKind) error {
	switch kind {
	case RuntimePackComponentKindPromptPack,
		RuntimePackComponentKindSkillPack,
		RuntimePackComponentKindManifestPack,
		RuntimePackComponentKindExtensionPack:
		return nil
	default:
		if strings.TrimSpace(string(kind)) == "" {
			return fmt.Errorf("mission store runtime pack component kind is required")
		}
		return fmt.Errorf("mission store runtime pack component kind %q is invalid", strings.TrimSpace(string(kind)))
	}
}

func ValidateRuntimePackComponentRef(ref RuntimePackComponentRef) error {
	ref = NormalizeRuntimePackComponentRef(ref)
	if err := ValidateRuntimePackComponentKind(ref.Kind); err != nil {
		return err
	}
	return validateRuntimePackIDField("mission store runtime pack component", "component_id", ref.ComponentID)
}

func ValidateRuntimePackComponentRecord(record RuntimePackComponentRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime pack component record_version must be positive")
	}
	if err := ValidateRuntimePackComponentRef(RuntimePackComponentRef{Kind: record.Kind, ComponentID: record.ComponentID}); err != nil {
		return err
	}
	if record.ParentComponentID != "" {
		if err := validateRuntimePackIDField("mission store runtime pack component", "parent_component_id", record.ParentComponentID); err != nil {
			return err
		}
	}
	if record.ContentRef == "" {
		return fmt.Errorf("mission store runtime pack component content_ref is required")
	}
	if err := validateRuntimePackComponentSHA256(record.ContentSHA256); err != nil {
		return err
	}
	if record.SourceSummary == "" {
		return fmt.Errorf("mission store runtime pack component source_summary is required")
	}
	if record.ProvenanceRef == "" {
		return fmt.Errorf("mission store runtime pack component provenance_ref is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store runtime pack component created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store runtime pack component created_by is required")
	}
	return nil
}

func StoreRuntimePackComponentRecord(root string, record RuntimePackComponentRecord) (RuntimePackComponentRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimePackComponentRecord{}, false, err
	}
	record = NormalizeRuntimePackComponentRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRuntimePackComponentRecord(record); err != nil {
		return RuntimePackComponentRecord{}, false, err
	}

	path := StoreRuntimePackComponentPath(root, record.Kind, record.ComponentID)
	existing, err := loadRuntimePackComponentRecordFile(path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return RuntimePackComponentRecord{}, false, fmt.Errorf("mission store runtime pack component %q kind %q already exists", record.ComponentID, record.Kind)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimePackComponentRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return RuntimePackComponentRecord{}, false, err
	}
	stored, err := LoadRuntimePackComponentRecord(root, record.Kind, record.ComponentID)
	if err != nil {
		return RuntimePackComponentRecord{}, false, err
	}
	return stored, true, nil
}

func LoadRuntimePackComponentRecord(root string, kind RuntimePackComponentKind, componentID string) (RuntimePackComponentRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimePackComponentRecord{}, err
	}
	ref := NormalizeRuntimePackComponentRef(RuntimePackComponentRef{Kind: kind, ComponentID: componentID})
	if err := ValidateRuntimePackComponentRef(ref); err != nil {
		return RuntimePackComponentRecord{}, err
	}
	record, err := loadRuntimePackComponentRecordFile(StoreRuntimePackComponentPath(root, ref.Kind, ref.ComponentID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimePackComponentRecord{}, ErrRuntimePackComponentRecordNotFound
		}
		return RuntimePackComponentRecord{}, err
	}
	return record, nil
}

func ListRuntimePackComponentRecords(root string, kind RuntimePackComponentKind) ([]RuntimePackComponentRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	if err := ValidateRuntimePackComponentKind(RuntimePackComponentKind(strings.TrimSpace(string(kind)))); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRuntimePackComponentsDir(root, RuntimePackComponentKind(strings.TrimSpace(string(kind)))), loadRuntimePackComponentRecordFile)
}

func ResolveRuntimePackComponents(root string, record RuntimePackRecord) (ResolvedRuntimePackComponents, error) {
	record = NormalizeRuntimePackRecord(record)
	if err := ValidateRuntimePackRecord(record); err != nil {
		return ResolvedRuntimePackComponents{}, err
	}

	prompt, err := loadRuntimePackComponentRef(root, RuntimePackComponentKindPromptPack, record.PromptPackRef, "prompt_pack_ref")
	if err != nil {
		return ResolvedRuntimePackComponents{}, err
	}
	skill, err := loadRuntimePackComponentRef(root, RuntimePackComponentKindSkillPack, record.SkillPackRef, "skill_pack_ref")
	if err != nil {
		return ResolvedRuntimePackComponents{}, err
	}
	manifest, err := loadRuntimePackComponentRef(root, RuntimePackComponentKindManifestPack, record.ManifestRef, "manifest_ref")
	if err != nil {
		return ResolvedRuntimePackComponents{}, err
	}
	extension, err := loadRuntimePackComponentRef(root, RuntimePackComponentKindExtensionPack, record.ExtensionPackRef, "extension_pack_ref")
	if err != nil {
		return ResolvedRuntimePackComponents{}, err
	}

	return ResolvedRuntimePackComponents{
		PromptPack:    prompt,
		SkillPack:     skill,
		ManifestPack:  manifest,
		ExtensionPack: extension,
	}, nil
}

func ResolveActiveRuntimePackComponents(root string) (RuntimePackRecord, ResolvedRuntimePackComponents, error) {
	record, err := ResolveActiveRuntimePackRecord(root)
	if err != nil {
		return RuntimePackRecord{}, ResolvedRuntimePackComponents{}, err
	}
	components, err := ResolveRuntimePackComponents(root, record)
	if err != nil {
		return RuntimePackRecord{}, ResolvedRuntimePackComponents{}, err
	}
	return record, components, nil
}

func loadRuntimePackComponentRecordFile(path string) (RuntimePackComponentRecord, error) {
	var record RuntimePackComponentRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimePackComponentRecord{}, err
	}
	record = NormalizeRuntimePackComponentRecord(record)
	if err := ValidateRuntimePackComponentRecord(record); err != nil {
		return RuntimePackComponentRecord{}, err
	}
	return record, nil
}

func loadRuntimePackComponentRef(root string, kind RuntimePackComponentKind, componentID, fieldName string) (RuntimePackComponentRecord, error) {
	component, err := LoadRuntimePackComponentRecord(root, kind, componentID)
	if err != nil {
		return RuntimePackComponentRecord{}, fmt.Errorf("mission store runtime pack %s %q: %w", fieldName, strings.TrimSpace(componentID), err)
	}
	return component, nil
}

func validateRuntimePackComponentSHA256(value string) error {
	if value == "" {
		return fmt.Errorf("mission store runtime pack component content_sha256 is required")
	}
	if len(value) != 64 {
		return fmt.Errorf("mission store runtime pack component content_sha256 %q is invalid", value)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("mission store runtime pack component content_sha256 %q is invalid", value)
	}
	return nil
}
