package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/local/picobot/internal/config"
)

const (
	// Spec label "Bluetooth/NFC" is normalized to repo-safe identifier "bluetooth_nfc".
	BluetoothNFCCapabilityName               = "bluetooth_nfc"
	BluetoothNFCLocalFileCapabilityID        = "bluetooth_nfc-local-file"
	BluetoothNFCLocalFileCapabilityValidator = "shared_storage exposed and committed bluetooth_nfc source file exists and is readable"
	BluetoothNFCLocalFileCapabilityNotes     = "bluetooth_nfc capability exposed through a committed local bluetooth_nfc source file under shared storage"

	BluetoothNFCLocalFileSourceID    = "bluetooth_nfc-local-file-source"
	BluetoothNFCLocalFileSourceKind  = "local_file"
	BluetoothNFCLocalFileDefaultPath = "bluetooth_nfc/current_source.json"
)

type BluetoothNFCSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrBluetoothNFCSourceRecordNotFound = errors.New("mission store bluetooth_nfc source record not found")

func StoreBluetoothNFCSourcesDir(root string) string {
	return filepath.Join(root, "bluetooth_nfc_sources")
}

func StoreBluetoothNFCSourcePath(root, sourceID string) string {
	return filepath.Join(StoreBluetoothNFCSourcesDir(root), sourceID+".json")
}

func StepRequiresBluetoothNFCCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == BluetoothNFCCapabilityName {
			return true
		}
	}
	return false
}

func ResolveBluetoothNFCCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, BluetoothNFCCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedBluetoothNFCCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveBluetoothNFCCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`bluetooth_nfc capability requires one committed capability record named %q`, BluetoothNFCCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("bluetooth_nfc capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedBluetoothNFCCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresBluetoothNFCCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context bluetooth_nfc capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != BluetoothNFCCapabilityName {
		return nil, fmt.Errorf(
			"execution context bluetooth_nfc capability requires capability onboarding proposal for %q, got %q",
			BluetoothNFCCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context bluetooth_nfc capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeBluetoothNFCSourceRecord(record BluetoothNFCSourceRecord) BluetoothNFCSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeBluetoothNFCSourcePath(record.Path)
	return record
}

func ValidateBluetoothNFCSourceRecord(record BluetoothNFCSourceRecord) error {
	record = NormalizeBluetoothNFCSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store bluetooth_nfc source record_version must be positive")
	}
	if err := validateBluetoothNFCSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != BluetoothNFCCapabilityName {
		return fmt.Errorf("mission store bluetooth_nfc source capability_name must be %q", BluetoothNFCCapabilityName)
	}
	if record.Kind != BluetoothNFCLocalFileSourceKind {
		return fmt.Errorf("mission store bluetooth_nfc source kind %q is invalid", record.Kind)
	}
	if err := validateBluetoothNFCSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreBluetoothNFCSourceRecord(root string, record BluetoothNFCSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeBluetoothNFCSourceRecord(record)
	if err := ValidateBluetoothNFCSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreBluetoothNFCSourcePath(root, record.SourceID), record)
}

func LoadBluetoothNFCSourceRecord(root, sourceID string) (BluetoothNFCSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return BluetoothNFCSourceRecord{}, err
	}
	if err := validateBluetoothNFCSourceID(strings.TrimSpace(sourceID)); err != nil {
		return BluetoothNFCSourceRecord{}, err
	}
	var record BluetoothNFCSourceRecord
	if err := LoadStoreJSON(StoreBluetoothNFCSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return BluetoothNFCSourceRecord{}, ErrBluetoothNFCSourceRecordNotFound
		}
		return BluetoothNFCSourceRecord{}, err
	}
	record = NormalizeBluetoothNFCSourceRecord(record)
	if err := ValidateBluetoothNFCSourceRecord(record); err != nil {
		return BluetoothNFCSourceRecord{}, err
	}
	return record, nil
}

func ResolveBluetoothNFCSourceRecord(root string) (*BluetoothNFCSourceRecord, error) {
	record, err := LoadBluetoothNFCSourceRecord(root, BluetoothNFCLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableBluetoothNFCSourceRecord(root string, workspacePath string) (*BluetoothNFCSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("bluetooth_nfc capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveBluetoothNFCSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrBluetoothNFCSourceRecordNotFound) {
			return nil, fmt.Errorf(`bluetooth_nfc capability requires one committed bluetooth_nfc source record %q`, BluetoothNFCLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureBluetoothNFCSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceBluetoothNFCCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("bluetooth_nfc capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("bluetooth_nfc capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveBluetoothNFCSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureBluetoothNFCSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrBluetoothNFCSourceRecordNotFound):
		source = &BluetoothNFCSourceRecord{
			SourceID:       BluetoothNFCLocalFileSourceID,
			CapabilityName: BluetoothNFCCapabilityName,
			Kind:           BluetoothNFCLocalFileSourceKind,
			Path:           BluetoothNFCLocalFileDefaultPath,
		}
		if err := ensureBluetoothNFCSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreBluetoothNFCSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveBluetoothNFCCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != BluetoothNFCLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"bluetooth_nfc capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  BluetoothNFCLocalFileCapabilityID,
		Class:         BluetoothNFCCapabilityName,
		Name:          BluetoothNFCCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     BluetoothNFCLocalFileCapabilityValidator,
		Notes:         BluetoothNFCLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, BluetoothNFCLocalFileCapabilityID)
}

func validateBluetoothNFCSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store bluetooth_nfc source %w", err)
	}
	return nil
}

func normalizeBluetoothNFCSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateBluetoothNFCSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store bluetooth_nfc source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store bluetooth_nfc source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeBluetoothNFCSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store bluetooth_nfc source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store bluetooth_nfc source path %q is invalid", trimmed)
	}
	return nil
}

func resolveBluetoothNFCSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateBluetoothNFCSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeBluetoothNFCSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("bluetooth_nfc capability exposure requires resolvable bluetooth_nfc source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("bluetooth_nfc capability exposure requires workspace-relative bluetooth_nfc source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("bluetooth_nfc capability exposure requires bluetooth_nfc source path under configured workspace root")
	}
	return resolved, nil
}

func ensureBluetoothNFCSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveBluetoothNFCSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("bluetooth_nfc capability exposure requires bluetooth_nfc source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, []byte("{}\n"), 0o644); err != nil {
				return fmt.Errorf("bluetooth_nfc capability exposure requires writable bluetooth_nfc source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("bluetooth_nfc capability exposure requires bluetooth_nfc source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("bluetooth_nfc capability exposure requires readable bluetooth_nfc source file %q", normalizeBluetoothNFCSourcePath(relativePath))
		}
		return fmt.Errorf("bluetooth_nfc capability exposure requires readable bluetooth_nfc source file %q: %w", normalizeBluetoothNFCSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("bluetooth_nfc capability exposure requires bluetooth_nfc source file stat %q: %w", normalizeBluetoothNFCSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("bluetooth_nfc capability exposure requires bluetooth_nfc source path %q to be a file", normalizeBluetoothNFCSourcePath(relativePath))
	}
	return nil
}
