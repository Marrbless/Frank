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
	MicrophoneCapabilityName               = "microphone"
	MicrophoneLocalFileCapabilityID        = "microphone-local-file"
	MicrophoneLocalFileCapabilityValidator = "shared_storage exposed and committed microphone source file exists and is readable"
	MicrophoneLocalFileCapabilityNotes     = "microphone capability exposed through a committed local microphone source file under shared storage"

	MicrophoneLocalFileSourceID    = "microphone-local-file-source"
	MicrophoneLocalFileSourceKind  = "local_file"
	MicrophoneLocalFileDefaultPath = "microphone/current_audio.wav"
)

type MicrophoneSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrMicrophoneSourceRecordNotFound = errors.New("mission store microphone source record not found")

func StoreMicrophoneSourcesDir(root string) string {
	return filepath.Join(root, "microphone_sources")
}

func StoreMicrophoneSourcePath(root, sourceID string) string {
	return filepath.Join(StoreMicrophoneSourcesDir(root), sourceID+".json")
}

func StepRequiresMicrophoneCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == MicrophoneCapabilityName {
			return true
		}
	}
	return false
}

func ResolveMicrophoneCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, MicrophoneCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedMicrophoneCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveMicrophoneCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`microphone capability requires one committed capability record named %q`, MicrophoneCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("microphone capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedMicrophoneCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresMicrophoneCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context microphone capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != MicrophoneCapabilityName {
		return nil, fmt.Errorf(
			"execution context microphone capability requires capability onboarding proposal for %q, got %q",
			MicrophoneCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context microphone capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeMicrophoneSourceRecord(record MicrophoneSourceRecord) MicrophoneSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeMicrophoneSourcePath(record.Path)
	return record
}

func ValidateMicrophoneSourceRecord(record MicrophoneSourceRecord) error {
	record = NormalizeMicrophoneSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store microphone source record_version must be positive")
	}
	if err := validateMicrophoneSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != MicrophoneCapabilityName {
		return fmt.Errorf("mission store microphone source capability_name must be %q", MicrophoneCapabilityName)
	}
	if record.Kind != MicrophoneLocalFileSourceKind {
		return fmt.Errorf("mission store microphone source kind %q is invalid", record.Kind)
	}
	if err := validateMicrophoneSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreMicrophoneSourceRecord(root string, record MicrophoneSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeMicrophoneSourceRecord(record)
	if err := ValidateMicrophoneSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreMicrophoneSourcePath(root, record.SourceID), record)
}

func LoadMicrophoneSourceRecord(root, sourceID string) (MicrophoneSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return MicrophoneSourceRecord{}, err
	}
	if err := validateMicrophoneSourceID(strings.TrimSpace(sourceID)); err != nil {
		return MicrophoneSourceRecord{}, err
	}
	var record MicrophoneSourceRecord
	if err := LoadStoreJSON(StoreMicrophoneSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MicrophoneSourceRecord{}, ErrMicrophoneSourceRecordNotFound
		}
		return MicrophoneSourceRecord{}, err
	}
	record = NormalizeMicrophoneSourceRecord(record)
	if err := ValidateMicrophoneSourceRecord(record); err != nil {
		return MicrophoneSourceRecord{}, err
	}
	return record, nil
}

func ResolveMicrophoneSourceRecord(root string) (*MicrophoneSourceRecord, error) {
	record, err := LoadMicrophoneSourceRecord(root, MicrophoneLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableMicrophoneSourceRecord(root string, workspacePath string) (*MicrophoneSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("microphone capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveMicrophoneSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrMicrophoneSourceRecordNotFound) {
			return nil, fmt.Errorf(`microphone capability requires one committed microphone source record %q`, MicrophoneLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureMicrophoneSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceMicrophoneCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("microphone capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("microphone capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveMicrophoneSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureMicrophoneSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrMicrophoneSourceRecordNotFound):
		source = &MicrophoneSourceRecord{
			SourceID:       MicrophoneLocalFileSourceID,
			CapabilityName: MicrophoneCapabilityName,
			Kind:           MicrophoneLocalFileSourceKind,
			Path:           MicrophoneLocalFileDefaultPath,
		}
		if err := ensureMicrophoneSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreMicrophoneSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveMicrophoneCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != MicrophoneLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"microphone capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  MicrophoneLocalFileCapabilityID,
		Class:         MicrophoneCapabilityName,
		Name:          MicrophoneCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     MicrophoneLocalFileCapabilityValidator,
		Notes:         MicrophoneLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, MicrophoneLocalFileCapabilityID)
}

func validateMicrophoneSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store microphone source %w", err)
	}
	return nil
}

func normalizeMicrophoneSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateMicrophoneSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store microphone source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store microphone source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeMicrophoneSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store microphone source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store microphone source path %q is invalid", trimmed)
	}
	return nil
}

func resolveMicrophoneSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateMicrophoneSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeMicrophoneSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("microphone capability exposure requires resolvable microphone source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("microphone capability exposure requires workspace-relative microphone source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("microphone capability exposure requires microphone source path under configured workspace root")
	}
	return resolved, nil
}

func ensureMicrophoneSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveMicrophoneSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("microphone capability exposure requires microphone source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, nil, 0o644); err != nil {
				return fmt.Errorf("microphone capability exposure requires writable microphone source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("microphone capability exposure requires microphone source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("microphone capability exposure requires readable microphone source file %q", normalizeMicrophoneSourcePath(relativePath))
		}
		return fmt.Errorf("microphone capability exposure requires readable microphone source file %q: %w", normalizeMicrophoneSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("microphone capability exposure requires microphone source file stat %q: %w", normalizeMicrophoneSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("microphone capability exposure requires microphone source path %q to be a file", normalizeMicrophoneSourcePath(relativePath))
	}
	return nil
}
