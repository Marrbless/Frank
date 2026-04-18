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
	LocationCapabilityName               = "location"
	LocationLocalFileCapabilityID        = "location-local-file"
	LocationLocalFileCapabilityValidator = "shared_storage exposed and committed location source file exists and is readable"
	LocationLocalFileCapabilityNotes     = "location capability exposed through a committed local location source file under shared storage"

	LocationLocalFileSourceID    = "location-local-file-source"
	LocationLocalFileSourceKind  = "local_file"
	LocationLocalFileDefaultPath = "location/current_location.json"
)

type LocationSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrLocationSourceRecordNotFound = errors.New("mission store location source record not found")

func StoreLocationSourcesDir(root string) string {
	return filepath.Join(root, "location_sources")
}

func StoreLocationSourcePath(root, sourceID string) string {
	return filepath.Join(StoreLocationSourcesDir(root), sourceID+".json")
}

func StepRequiresLocationCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == LocationCapabilityName {
			return true
		}
	}
	return false
}

func ResolveLocationCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, LocationCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedLocationCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveLocationCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`location capability requires one committed capability record named %q`, LocationCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("location capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedLocationCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresLocationCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context location capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != LocationCapabilityName {
		return nil, fmt.Errorf(
			"execution context location capability requires capability onboarding proposal for %q, got %q",
			LocationCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context location capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeLocationSourceRecord(record LocationSourceRecord) LocationSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeLocationSourcePath(record.Path)
	return record
}

func ValidateLocationSourceRecord(record LocationSourceRecord) error {
	record = NormalizeLocationSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store location source record_version must be positive")
	}
	if err := validateLocationSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != LocationCapabilityName {
		return fmt.Errorf("mission store location source capability_name must be %q", LocationCapabilityName)
	}
	if record.Kind != LocationLocalFileSourceKind {
		return fmt.Errorf("mission store location source kind %q is invalid", record.Kind)
	}
	if err := validateLocationSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreLocationSourceRecord(root string, record LocationSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeLocationSourceRecord(record)
	if err := ValidateLocationSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreLocationSourcePath(root, record.SourceID), record)
}

func LoadLocationSourceRecord(root, sourceID string) (LocationSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LocationSourceRecord{}, err
	}
	if err := validateLocationSourceID(strings.TrimSpace(sourceID)); err != nil {
		return LocationSourceRecord{}, err
	}
	var record LocationSourceRecord
	if err := LoadStoreJSON(StoreLocationSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocationSourceRecord{}, ErrLocationSourceRecordNotFound
		}
		return LocationSourceRecord{}, err
	}
	record = NormalizeLocationSourceRecord(record)
	if err := ValidateLocationSourceRecord(record); err != nil {
		return LocationSourceRecord{}, err
	}
	return record, nil
}

func ResolveLocationSourceRecord(root string) (*LocationSourceRecord, error) {
	record, err := LoadLocationSourceRecord(root, LocationLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableLocationSourceRecord(root string, workspacePath string) (*LocationSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("location capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveLocationSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrLocationSourceRecordNotFound) {
			return nil, fmt.Errorf(`location capability requires one committed location source record %q`, LocationLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureLocationSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceLocationCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("location capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("location capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveLocationSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureLocationSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrLocationSourceRecordNotFound):
		source = &LocationSourceRecord{
			SourceID:       LocationLocalFileSourceID,
			CapabilityName: LocationCapabilityName,
			Kind:           LocationLocalFileSourceKind,
			Path:           LocationLocalFileDefaultPath,
		}
		if err := ensureLocationSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreLocationSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveLocationCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != LocationLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"location capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  LocationLocalFileCapabilityID,
		Class:         LocationCapabilityName,
		Name:          LocationCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     LocationLocalFileCapabilityValidator,
		Notes:         LocationLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, LocationLocalFileCapabilityID)
}

func validateLocationSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store location source %w", err)
	}
	return nil
}

func normalizeLocationSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateLocationSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store location source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store location source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeLocationSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store location source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store location source path %q is invalid", trimmed)
	}
	return nil
}

func resolveLocationSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateLocationSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeLocationSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("location capability exposure requires resolvable location source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("location capability exposure requires workspace-relative location source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("location capability exposure requires location source path under configured workspace root")
	}
	return resolved, nil
}

func ensureLocationSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveLocationSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("location capability exposure requires location source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, []byte("{}\n"), 0o644); err != nil {
				return fmt.Errorf("location capability exposure requires writable location source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("location capability exposure requires location source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("location capability exposure requires readable location source file %q", normalizeLocationSourcePath(relativePath))
		}
		return fmt.Errorf("location capability exposure requires readable location source file %q: %w", normalizeLocationSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("location capability exposure requires location source file stat %q: %w", normalizeLocationSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("location capability exposure requires location source path %q to be a file", normalizeLocationSourcePath(relativePath))
	}
	return nil
}
