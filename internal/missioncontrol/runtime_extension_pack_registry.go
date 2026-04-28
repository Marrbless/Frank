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

type RuntimeExtensionToolDeclaration struct {
	ToolName             string   `json:"tool_name"`
	PermissionRefs       []string `json:"permission_refs,omitempty"`
	ExternalSideEffect   bool     `json:"external_side_effect,omitempty"`
	CompatibilitySummary string   `json:"compatibility_summary,omitempty"`
}

type RuntimeExtensionPackRecord struct {
	RecordVersion            int                               `json:"record_version"`
	ExtensionPackID          string                            `json:"extension_pack_id"`
	ParentExtensionPackID    string                            `json:"parent_extension_pack_id,omitempty"`
	Extensions               []string                          `json:"extensions"`
	DeclaredTools            []RuntimeExtensionToolDeclaration `json:"declared_tools"`
	DeclaredEvents           []string                          `json:"declared_events"`
	DeclaredPermissions      []string                          `json:"declared_permissions"`
	ExternalSideEffects      []string                          `json:"external_side_effects,omitempty"`
	CompatibilityContractRef string                            `json:"compatibility_contract_ref"`
	HotReloadable            bool                              `json:"hot_reloadable"`
	ApprovalRequired         bool                              `json:"approval_required,omitempty"`
	ChangeSummary            string                            `json:"change_summary"`
	CreatedAt                time.Time                         `json:"created_at"`
	CreatedBy                string                            `json:"created_by"`
}

var ErrRuntimeExtensionPackRecordNotFound = errors.New("mission store runtime extension pack record not found")

func StoreRuntimeExtensionPacksDir(root string) string {
	return filepath.Join(root, "runtime_packs", "extension_packs")
}

func StoreRuntimeExtensionPackPath(root, extensionPackID string) string {
	return filepath.Join(StoreRuntimeExtensionPacksDir(root), strings.TrimSpace(extensionPackID)+".json")
}

func NormalizeRuntimeExtensionToolDeclaration(tool RuntimeExtensionToolDeclaration) RuntimeExtensionToolDeclaration {
	tool.ToolName = strings.TrimSpace(tool.ToolName)
	tool.PermissionRefs = normalizeRuntimePackStrings(tool.PermissionRefs)
	tool.CompatibilitySummary = strings.TrimSpace(tool.CompatibilitySummary)
	return tool
}

func NormalizeRuntimeExtensionPackRecord(record RuntimeExtensionPackRecord) RuntimeExtensionPackRecord {
	record.ExtensionPackID = strings.TrimSpace(record.ExtensionPackID)
	record.ParentExtensionPackID = strings.TrimSpace(record.ParentExtensionPackID)
	record.Extensions = normalizeRuntimePackStrings(record.Extensions)
	record.DeclaredEvents = normalizeRuntimePackStrings(record.DeclaredEvents)
	record.DeclaredPermissions = normalizeRuntimePackStrings(record.DeclaredPermissions)
	record.ExternalSideEffects = normalizeRuntimePackStrings(record.ExternalSideEffects)
	record.CompatibilityContractRef = strings.TrimSpace(record.CompatibilityContractRef)
	record.ChangeSummary = strings.TrimSpace(record.ChangeSummary)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	if len(record.DeclaredTools) > 0 {
		tools := make([]RuntimeExtensionToolDeclaration, 0, len(record.DeclaredTools))
		for _, tool := range record.DeclaredTools {
			normalized := NormalizeRuntimeExtensionToolDeclaration(tool)
			if normalized.ToolName == "" && len(normalized.PermissionRefs) == 0 && !normalized.ExternalSideEffect && normalized.CompatibilitySummary == "" {
				continue
			}
			tools = append(tools, normalized)
		}
		record.DeclaredTools = tools
	}
	return record
}

func ValidateRuntimeExtensionPackRecord(record RuntimeExtensionPackRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime extension pack record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store runtime extension pack", "extension_pack_id", record.ExtensionPackID); err != nil {
		return err
	}
	if record.ParentExtensionPackID != "" {
		if err := validateRuntimePackIDField("mission store runtime extension pack", "parent_extension_pack_id", record.ParentExtensionPackID); err != nil {
			return err
		}
	}
	if len(record.Extensions) == 0 {
		return fmt.Errorf("mission store runtime extension pack extensions are required")
	}
	if len(record.DeclaredTools) == 0 {
		return fmt.Errorf("mission store runtime extension pack declared_tools are required")
	}
	for index, tool := range record.DeclaredTools {
		if tool.ToolName == "" {
			return fmt.Errorf("mission store runtime extension pack declared_tools[%d].tool_name is required", index)
		}
	}
	if len(record.DeclaredEvents) == 0 {
		return fmt.Errorf("mission store runtime extension pack declared_events are required")
	}
	if len(record.DeclaredPermissions) == 0 {
		return fmt.Errorf("mission store runtime extension pack declared_permissions are required")
	}
	if record.CompatibilityContractRef == "" {
		return fmt.Errorf("mission store runtime extension pack compatibility_contract_ref is required")
	}
	if !record.HotReloadable {
		return fmt.Errorf("mission store runtime extension pack hot_reloadable must be true")
	}
	if record.ChangeSummary == "" {
		return fmt.Errorf("mission store runtime extension pack change_summary is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store runtime extension pack created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store runtime extension pack created_by is required")
	}
	return nil
}

func StoreRuntimeExtensionPackRecord(root string, record RuntimeExtensionPackRecord) (RuntimeExtensionPackRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeExtensionPackRecord{}, false, err
	}
	record = NormalizeRuntimeExtensionPackRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRuntimeExtensionPackRecord(record); err != nil {
		return RuntimeExtensionPackRecord{}, false, err
	}
	path := StoreRuntimeExtensionPackPath(root, record.ExtensionPackID)
	existing, err := loadRuntimeExtensionPackRecordFile(path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return RuntimeExtensionPackRecord{}, false, fmt.Errorf("mission store runtime extension pack %q already exists", record.ExtensionPackID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimeExtensionPackRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return RuntimeExtensionPackRecord{}, false, err
	}
	stored, err := LoadRuntimeExtensionPackRecord(root, record.ExtensionPackID)
	if err != nil {
		return RuntimeExtensionPackRecord{}, false, err
	}
	return stored, true, nil
}

func LoadRuntimeExtensionPackRecord(root, extensionPackID string) (RuntimeExtensionPackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeExtensionPackRecord{}, err
	}
	extensionPackID = strings.TrimSpace(extensionPackID)
	if err := validateRuntimePackIDField("mission store runtime extension pack", "extension_pack_id", extensionPackID); err != nil {
		return RuntimeExtensionPackRecord{}, err
	}
	record, err := loadRuntimeExtensionPackRecordFile(StoreRuntimeExtensionPackPath(root, extensionPackID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeExtensionPackRecord{}, ErrRuntimeExtensionPackRecordNotFound
		}
		return RuntimeExtensionPackRecord{}, err
	}
	return record, nil
}

func ListRuntimeExtensionPackRecords(root string) ([]RuntimeExtensionPackRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRuntimeExtensionPacksDir(root), loadRuntimeExtensionPackRecordFile)
}

func loadRuntimeExtensionPackRecordFile(path string) (RuntimeExtensionPackRecord, error) {
	var record RuntimeExtensionPackRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimeExtensionPackRecord{}, err
	}
	record = NormalizeRuntimeExtensionPackRecord(record)
	if err := ValidateRuntimeExtensionPackRecord(record); err != nil {
		return RuntimeExtensionPackRecord{}, err
	}
	return record, nil
}
