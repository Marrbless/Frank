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
	CameraCapabilityName               = "camera"
	CameraLocalFileCapabilityID        = "camera-local-file"
	CameraLocalFileCapabilityValidator = "shared_storage exposed and committed camera source file exists and is readable"
	CameraLocalFileCapabilityNotes     = "camera capability exposed through a committed local camera source file under shared storage"

	CameraLocalFileSourceID    = "camera-local-file-source"
	CameraLocalFileSourceKind  = "local_file"
	CameraLocalFileDefaultPath = "camera/current_image.jpg"
)

type CameraSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrCameraSourceRecordNotFound = errors.New("mission store camera source record not found")

func StoreCameraSourcesDir(root string) string {
	return filepath.Join(root, "camera_sources")
}

func StoreCameraSourcePath(root, sourceID string) string {
	return filepath.Join(StoreCameraSourcesDir(root), sourceID+".json")
}

func StepRequiresCameraCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == CameraCapabilityName {
			return true
		}
	}
	return false
}

func ResolveCameraCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, CameraCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedCameraCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCameraCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`camera capability requires one committed capability record named %q`, CameraCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("camera capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedCameraCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresCameraCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context camera capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != CameraCapabilityName {
		return nil, fmt.Errorf(
			"execution context camera capability requires capability onboarding proposal for %q, got %q",
			CameraCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context camera capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeCameraSourceRecord(record CameraSourceRecord) CameraSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeCameraSourcePath(record.Path)
	return record
}

func ValidateCameraSourceRecord(record CameraSourceRecord) error {
	record = NormalizeCameraSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store camera source record_version must be positive")
	}
	if err := validateCameraSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != CameraCapabilityName {
		return fmt.Errorf("mission store camera source capability_name must be %q", CameraCapabilityName)
	}
	if record.Kind != CameraLocalFileSourceKind {
		return fmt.Errorf("mission store camera source kind %q is invalid", record.Kind)
	}
	if err := validateCameraSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreCameraSourceRecord(root string, record CameraSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCameraSourceRecord(record)
	if err := ValidateCameraSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCameraSourcePath(root, record.SourceID), record)
}

func LoadCameraSourceRecord(root, sourceID string) (CameraSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CameraSourceRecord{}, err
	}
	if err := validateCameraSourceID(strings.TrimSpace(sourceID)); err != nil {
		return CameraSourceRecord{}, err
	}
	var record CameraSourceRecord
	if err := LoadStoreJSON(StoreCameraSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CameraSourceRecord{}, ErrCameraSourceRecordNotFound
		}
		return CameraSourceRecord{}, err
	}
	record = NormalizeCameraSourceRecord(record)
	if err := ValidateCameraSourceRecord(record); err != nil {
		return CameraSourceRecord{}, err
	}
	return record, nil
}

func ResolveCameraSourceRecord(root string) (*CameraSourceRecord, error) {
	record, err := LoadCameraSourceRecord(root, CameraLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableCameraSourceRecord(root string, workspacePath string) (*CameraSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("camera capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveCameraSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrCameraSourceRecordNotFound) {
			return nil, fmt.Errorf(`camera capability requires one committed camera source record %q`, CameraLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureCameraSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceCameraCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("camera capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("camera capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveCameraSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureCameraSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrCameraSourceRecordNotFound):
		source = &CameraSourceRecord{
			SourceID:       CameraLocalFileSourceID,
			CapabilityName: CameraCapabilityName,
			Kind:           CameraLocalFileSourceKind,
			Path:           CameraLocalFileDefaultPath,
		}
		if err := ensureCameraSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreCameraSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveCameraCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != CameraLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"camera capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  CameraLocalFileCapabilityID,
		Class:         CameraCapabilityName,
		Name:          CameraCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     CameraLocalFileCapabilityValidator,
		Notes:         CameraLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, CameraLocalFileCapabilityID)
}

func validateCameraSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store camera source %w", err)
	}
	return nil
}

func normalizeCameraSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateCameraSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store camera source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store camera source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeCameraSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store camera source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store camera source path %q is invalid", trimmed)
	}
	return nil
}

func resolveCameraSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateCameraSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeCameraSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("camera capability exposure requires resolvable camera source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("camera capability exposure requires workspace-relative camera source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("camera capability exposure requires camera source path under configured workspace root")
	}
	return resolved, nil
}

func ensureCameraSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveCameraSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("camera capability exposure requires camera source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, nil, 0o644); err != nil {
				return fmt.Errorf("camera capability exposure requires writable camera source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("camera capability exposure requires camera source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("camera capability exposure requires readable camera source file %q", normalizeCameraSourcePath(relativePath))
		}
		return fmt.Errorf("camera capability exposure requires readable camera source file %q: %w", normalizeCameraSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("camera capability exposure requires camera source file stat %q: %w", normalizeCameraSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("camera capability exposure requires camera source path %q to be a file", normalizeCameraSourcePath(relativePath))
	}
	return nil
}
