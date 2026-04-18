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
	// Spec label "broad app control" is normalized to repo-safe identifier "broad_app_control".
	BroadAppControlCapabilityName               = "broad_app_control"
	BroadAppControlLocalFileCapabilityID        = "broad_app_control-local-file"
	BroadAppControlLocalFileCapabilityValidator = "shared_storage exposed and committed broad_app_control source file exists and is readable"
	BroadAppControlLocalFileCapabilityNotes     = "broad_app_control capability exposed through a committed local broad_app_control source file under shared storage"

	BroadAppControlLocalFileSourceID    = "broad_app_control-local-file-source"
	BroadAppControlLocalFileSourceKind  = "local_file"
	BroadAppControlLocalFileDefaultPath = "broad_app_control/current_source.json"
)

type BroadAppControlSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrBroadAppControlSourceRecordNotFound = errors.New("mission store broad_app_control source record not found")

func StoreBroadAppControlSourcesDir(root string) string {
	return filepath.Join(root, "broad_app_control_sources")
}

func StoreBroadAppControlSourcePath(root, sourceID string) string {
	return filepath.Join(StoreBroadAppControlSourcesDir(root), sourceID+".json")
}

func StepRequiresBroadAppControlCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == BroadAppControlCapabilityName {
			return true
		}
	}
	return false
}

func ResolveBroadAppControlCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, BroadAppControlCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedBroadAppControlCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveBroadAppControlCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`broad_app_control capability requires one committed capability record named %q`, BroadAppControlCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("broad_app_control capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedBroadAppControlCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresBroadAppControlCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context broad_app_control capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != BroadAppControlCapabilityName {
		return nil, fmt.Errorf(
			"execution context broad_app_control capability requires capability onboarding proposal for %q, got %q",
			BroadAppControlCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context broad_app_control capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeBroadAppControlSourceRecord(record BroadAppControlSourceRecord) BroadAppControlSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeBroadAppControlSourcePath(record.Path)
	return record
}

func ValidateBroadAppControlSourceRecord(record BroadAppControlSourceRecord) error {
	record = NormalizeBroadAppControlSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store broad_app_control source record_version must be positive")
	}
	if err := validateBroadAppControlSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != BroadAppControlCapabilityName {
		return fmt.Errorf("mission store broad_app_control source capability_name must be %q", BroadAppControlCapabilityName)
	}
	if record.Kind != BroadAppControlLocalFileSourceKind {
		return fmt.Errorf("mission store broad_app_control source kind %q is invalid", record.Kind)
	}
	if err := validateBroadAppControlSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreBroadAppControlSourceRecord(root string, record BroadAppControlSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeBroadAppControlSourceRecord(record)
	if err := ValidateBroadAppControlSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreBroadAppControlSourcePath(root, record.SourceID), record)
}

func LoadBroadAppControlSourceRecord(root, sourceID string) (BroadAppControlSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return BroadAppControlSourceRecord{}, err
	}
	if err := validateBroadAppControlSourceID(strings.TrimSpace(sourceID)); err != nil {
		return BroadAppControlSourceRecord{}, err
	}
	var record BroadAppControlSourceRecord
	if err := LoadStoreJSON(StoreBroadAppControlSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return BroadAppControlSourceRecord{}, ErrBroadAppControlSourceRecordNotFound
		}
		return BroadAppControlSourceRecord{}, err
	}
	record = NormalizeBroadAppControlSourceRecord(record)
	if err := ValidateBroadAppControlSourceRecord(record); err != nil {
		return BroadAppControlSourceRecord{}, err
	}
	return record, nil
}

func ResolveBroadAppControlSourceRecord(root string) (*BroadAppControlSourceRecord, error) {
	record, err := LoadBroadAppControlSourceRecord(root, BroadAppControlLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableBroadAppControlSourceRecord(root string, workspacePath string) (*BroadAppControlSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("broad_app_control capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveBroadAppControlSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrBroadAppControlSourceRecordNotFound) {
			return nil, fmt.Errorf(`broad_app_control capability requires one committed broad_app_control source record %q`, BroadAppControlLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureBroadAppControlSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceBroadAppControlCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("broad_app_control capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("broad_app_control capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveBroadAppControlSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureBroadAppControlSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrBroadAppControlSourceRecordNotFound):
		source = &BroadAppControlSourceRecord{
			SourceID:       BroadAppControlLocalFileSourceID,
			CapabilityName: BroadAppControlCapabilityName,
			Kind:           BroadAppControlLocalFileSourceKind,
			Path:           BroadAppControlLocalFileDefaultPath,
		}
		if err := ensureBroadAppControlSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreBroadAppControlSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveBroadAppControlCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != BroadAppControlLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"broad_app_control capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  BroadAppControlLocalFileCapabilityID,
		Class:         BroadAppControlCapabilityName,
		Name:          BroadAppControlCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     BroadAppControlLocalFileCapabilityValidator,
		Notes:         BroadAppControlLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, BroadAppControlLocalFileCapabilityID)
}

func validateBroadAppControlSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store broad_app_control source %w", err)
	}
	return nil
}

func normalizeBroadAppControlSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateBroadAppControlSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store broad_app_control source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store broad_app_control source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeBroadAppControlSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store broad_app_control source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store broad_app_control source path %q is invalid", trimmed)
	}
	return nil
}

func resolveBroadAppControlSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateBroadAppControlSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeBroadAppControlSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("broad_app_control capability exposure requires resolvable broad_app_control source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("broad_app_control capability exposure requires workspace-relative broad_app_control source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("broad_app_control capability exposure requires broad_app_control source path under configured workspace root")
	}
	return resolved, nil
}

func ensureBroadAppControlSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveBroadAppControlSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("broad_app_control capability exposure requires broad_app_control source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, []byte("{}\n"), 0o644); err != nil {
				return fmt.Errorf("broad_app_control capability exposure requires writable broad_app_control source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("broad_app_control capability exposure requires broad_app_control source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("broad_app_control capability exposure requires readable broad_app_control source file %q", normalizeBroadAppControlSourcePath(relativePath))
		}
		return fmt.Errorf("broad_app_control capability exposure requires readable broad_app_control source file %q: %w", normalizeBroadAppControlSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("broad_app_control capability exposure requires broad_app_control source file stat %q: %w", normalizeBroadAppControlSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("broad_app_control capability exposure requires broad_app_control source path %q to be a file", normalizeBroadAppControlSourcePath(relativePath))
	}
	return nil
}
