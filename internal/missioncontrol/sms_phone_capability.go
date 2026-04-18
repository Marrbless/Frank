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
	// Spec label "SMS/phone" is normalized to repo-safe identifier "sms_phone".
	SMSPhoneCapabilityName               = "sms_phone"
	SMSPhoneLocalFileCapabilityID        = "sms_phone-local-file"
	SMSPhoneLocalFileCapabilityValidator = "shared_storage exposed and committed sms_phone source file exists and is readable"
	SMSPhoneLocalFileCapabilityNotes     = "sms_phone capability exposed through a committed local sms_phone source file under shared storage"

	SMSPhoneLocalFileSourceID    = "sms_phone-local-file-source"
	SMSPhoneLocalFileSourceKind  = "local_file"
	SMSPhoneLocalFileDefaultPath = "sms_phone/current_source.json"
)

type SMSPhoneSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrSMSPhoneSourceRecordNotFound = errors.New("mission store sms_phone source record not found")

func StoreSMSPhoneSourcesDir(root string) string {
	return filepath.Join(root, "sms_phone_sources")
}

func StoreSMSPhoneSourcePath(root, sourceID string) string {
	return filepath.Join(StoreSMSPhoneSourcesDir(root), sourceID+".json")
}

func StepRequiresSMSPhoneCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == SMSPhoneCapabilityName {
			return true
		}
	}
	return false
}

func ResolveSMSPhoneCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, SMSPhoneCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedSMSPhoneCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveSMSPhoneCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`sms_phone capability requires one committed capability record named %q`, SMSPhoneCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("sms_phone capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedSMSPhoneCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresSMSPhoneCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context sms_phone capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != SMSPhoneCapabilityName {
		return nil, fmt.Errorf(
			"execution context sms_phone capability requires capability onboarding proposal for %q, got %q",
			SMSPhoneCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context sms_phone capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeSMSPhoneSourceRecord(record SMSPhoneSourceRecord) SMSPhoneSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeSMSPhoneSourcePath(record.Path)
	return record
}

func ValidateSMSPhoneSourceRecord(record SMSPhoneSourceRecord) error {
	record = NormalizeSMSPhoneSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store sms_phone source record_version must be positive")
	}
	if err := validateSMSPhoneSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != SMSPhoneCapabilityName {
		return fmt.Errorf("mission store sms_phone source capability_name must be %q", SMSPhoneCapabilityName)
	}
	if record.Kind != SMSPhoneLocalFileSourceKind {
		return fmt.Errorf("mission store sms_phone source kind %q is invalid", record.Kind)
	}
	if err := validateSMSPhoneSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreSMSPhoneSourceRecord(root string, record SMSPhoneSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeSMSPhoneSourceRecord(record)
	if err := ValidateSMSPhoneSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreSMSPhoneSourcePath(root, record.SourceID), record)
}

func LoadSMSPhoneSourceRecord(root, sourceID string) (SMSPhoneSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return SMSPhoneSourceRecord{}, err
	}
	if err := validateSMSPhoneSourceID(strings.TrimSpace(sourceID)); err != nil {
		return SMSPhoneSourceRecord{}, err
	}
	var record SMSPhoneSourceRecord
	if err := LoadStoreJSON(StoreSMSPhoneSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SMSPhoneSourceRecord{}, ErrSMSPhoneSourceRecordNotFound
		}
		return SMSPhoneSourceRecord{}, err
	}
	record = NormalizeSMSPhoneSourceRecord(record)
	if err := ValidateSMSPhoneSourceRecord(record); err != nil {
		return SMSPhoneSourceRecord{}, err
	}
	return record, nil
}

func ResolveSMSPhoneSourceRecord(root string) (*SMSPhoneSourceRecord, error) {
	record, err := LoadSMSPhoneSourceRecord(root, SMSPhoneLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableSMSPhoneSourceRecord(root string, workspacePath string) (*SMSPhoneSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("sms_phone capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveSMSPhoneSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrSMSPhoneSourceRecordNotFound) {
			return nil, fmt.Errorf(`sms_phone capability requires one committed sms_phone source record %q`, SMSPhoneLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureSMSPhoneSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceSMSPhoneCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("sms_phone capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("sms_phone capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveSMSPhoneSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureSMSPhoneSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrSMSPhoneSourceRecordNotFound):
		source = &SMSPhoneSourceRecord{
			SourceID:       SMSPhoneLocalFileSourceID,
			CapabilityName: SMSPhoneCapabilityName,
			Kind:           SMSPhoneLocalFileSourceKind,
			Path:           SMSPhoneLocalFileDefaultPath,
		}
		if err := ensureSMSPhoneSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreSMSPhoneSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveSMSPhoneCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != SMSPhoneLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"sms_phone capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  SMSPhoneLocalFileCapabilityID,
		Class:         SMSPhoneCapabilityName,
		Name:          SMSPhoneCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     SMSPhoneLocalFileCapabilityValidator,
		Notes:         SMSPhoneLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, SMSPhoneLocalFileCapabilityID)
}

func validateSMSPhoneSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store sms_phone source %w", err)
	}
	return nil
}

func normalizeSMSPhoneSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateSMSPhoneSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store sms_phone source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store sms_phone source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeSMSPhoneSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store sms_phone source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store sms_phone source path %q is invalid", trimmed)
	}
	return nil
}

func resolveSMSPhoneSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateSMSPhoneSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeSMSPhoneSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("sms_phone capability exposure requires resolvable sms_phone source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("sms_phone capability exposure requires workspace-relative sms_phone source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("sms_phone capability exposure requires sms_phone source path under configured workspace root")
	}
	return resolved, nil
}

func ensureSMSPhoneSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveSMSPhoneSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("sms_phone capability exposure requires sms_phone source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, []byte("{}\n"), 0o644); err != nil {
				return fmt.Errorf("sms_phone capability exposure requires writable sms_phone source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("sms_phone capability exposure requires sms_phone source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("sms_phone capability exposure requires readable sms_phone source file %q", normalizeSMSPhoneSourcePath(relativePath))
		}
		return fmt.Errorf("sms_phone capability exposure requires readable sms_phone source file %q: %w", normalizeSMSPhoneSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("sms_phone capability exposure requires sms_phone source file stat %q: %w", normalizeSMSPhoneSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("sms_phone capability exposure requires sms_phone source path %q to be a file", normalizeSMSPhoneSourcePath(relativePath))
	}
	return nil
}
