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
	ContactsCapabilityName               = "contacts"
	ContactsLocalFileCapabilityID        = "contacts-local-file"
	ContactsLocalFileCapabilityValidator = "shared_storage exposed and committed contacts source file exists and is readable"
	ContactsLocalFileCapabilityNotes     = "contacts capability exposed through a committed local contacts source file under shared storage"

	ContactsLocalFileSourceID    = "contacts-local-file-source"
	ContactsLocalFileSourceKind  = "local_file"
	ContactsLocalFileDefaultPath = "contacts/contacts.json"
)

type ContactsSourceRecord struct {
	RecordVersion  int    `json:"record_version"`
	SourceID       string `json:"source_id"`
	CapabilityName string `json:"capability_name"`
	Kind           string `json:"kind"`
	Path           string `json:"path"`
}

var ErrContactsSourceRecordNotFound = errors.New("mission store contacts source record not found")

func StoreContactsSourcesDir(root string) string {
	return filepath.Join(root, "contacts_sources")
}

func StoreContactsSourcePath(root, sourceID string) string {
	return filepath.Join(StoreContactsSourcesDir(root), sourceID+".json")
}

func StepRequiresContactsCapability(step Step) bool {
	for _, capability := range NormalizeStepRequiredCapabilities(step.RequiredCapabilities) {
		if capability == ContactsCapabilityName {
			return true
		}
	}
	return false
}

func ResolveContactsCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveCapabilityRecordByName(root, ContactsCapabilityName)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireExposedContactsCapabilityRecord(root string) (*CapabilityRecord, error) {
	record, err := ResolveContactsCapabilityRecord(root)
	if err != nil {
		if errors.Is(err, ErrCapabilityRecordNotFound) {
			return nil, fmt.Errorf(`contacts capability requires one committed capability record named %q`, ContactsCapabilityName)
		}
		return nil, err
	}
	if !record.Exposed {
		return nil, fmt.Errorf("contacts capability record %q is not exposed", record.CapabilityID)
	}
	return record, nil
}

func RequireApprovedContactsCapabilityOnboardingProposal(ec ExecutionContext) (*CapabilityOnboardingProposalRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if !StepRequiresContactsCapability(*ec.Step) {
		return nil, nil
	}

	proposal, err := ResolveExecutionContextCapabilityOnboardingProposal(ec)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("execution context contacts capability requires capability onboarding proposal ref")
	}
	if strings.TrimSpace(proposal.CapabilityName) != ContactsCapabilityName {
		return nil, fmt.Errorf(
			"execution context contacts capability requires capability onboarding proposal for %q, got %q",
			ContactsCapabilityName,
			proposal.CapabilityName,
		)
	}
	if NormalizeCapabilityOnboardingProposalState(proposal.State) != CapabilityOnboardingProposalStateApproved {
		return nil, fmt.Errorf(
			"execution context contacts capability requires approved capability onboarding proposal %q, got state %q",
			proposal.ProposalID,
			proposal.State,
		)
	}

	proposalCopy := *proposal
	return &proposalCopy, nil
}

func NormalizeContactsSourceRecord(record ContactsSourceRecord) ContactsSourceRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Path = normalizeContactsSourcePath(record.Path)
	return record
}

func ValidateContactsSourceRecord(record ContactsSourceRecord) error {
	record = NormalizeContactsSourceRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store contacts source record_version must be positive")
	}
	if err := validateContactsSourceID(record.SourceID); err != nil {
		return err
	}
	if record.CapabilityName != ContactsCapabilityName {
		return fmt.Errorf("mission store contacts source capability_name must be %q", ContactsCapabilityName)
	}
	if record.Kind != ContactsLocalFileSourceKind {
		return fmt.Errorf("mission store contacts source kind %q is invalid", record.Kind)
	}
	if err := validateContactsSourcePath(record.Path); err != nil {
		return err
	}
	return nil
}

func StoreContactsSourceRecord(root string, record ContactsSourceRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeContactsSourceRecord(record)
	if err := ValidateContactsSourceRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreContactsSourcePath(root, record.SourceID), record)
}

func LoadContactsSourceRecord(root, sourceID string) (ContactsSourceRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ContactsSourceRecord{}, err
	}
	if err := validateContactsSourceID(strings.TrimSpace(sourceID)); err != nil {
		return ContactsSourceRecord{}, err
	}
	var record ContactsSourceRecord
	if err := LoadStoreJSON(StoreContactsSourcePath(root, strings.TrimSpace(sourceID)), &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ContactsSourceRecord{}, ErrContactsSourceRecordNotFound
		}
		return ContactsSourceRecord{}, err
	}
	record = NormalizeContactsSourceRecord(record)
	if err := ValidateContactsSourceRecord(record); err != nil {
		return ContactsSourceRecord{}, err
	}
	return record, nil
}

func ResolveContactsSourceRecord(root string) (*ContactsSourceRecord, error) {
	record, err := LoadContactsSourceRecord(root, ContactsLocalFileSourceID)
	if err != nil {
		return nil, err
	}
	recordCopy := record
	return &recordCopy, nil
}

func RequireReadableContactsSourceRecord(root string, workspacePath string) (*ContactsSourceRecord, error) {
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return nil, fmt.Errorf("contacts capability requires shared_storage exposure: %w", err)
	}

	record, err := ResolveContactsSourceRecord(root)
	if err != nil {
		if errors.Is(err, ErrContactsSourceRecordNotFound) {
			return nil, fmt.Errorf(`contacts capability requires one committed contacts source record %q`, ContactsLocalFileSourceID)
		}
		return nil, err
	}
	if err := ensureContactsSourceFileReadable(workspacePath, record.Path, false); err != nil {
		return nil, err
	}
	return record, nil
}

func StoreWorkspaceContactsCapabilityExposure(root string, workspacePath string) (CapabilityRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityRecord{}, err
	}
	if _, err := RequireExposedSharedStorageCapabilityRecord(root); err != nil {
		return CapabilityRecord{}, fmt.Errorf("contacts capability requires shared_storage exposure: %w", err)
	}

	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return CapabilityRecord{}, err
	}
	if err := config.InitializeWorkspace(resolvedWorkspacePath); err != nil {
		return CapabilityRecord{}, fmt.Errorf("contacts capability exposure requires initialized workspace root: %w", err)
	}

	source, err := ResolveContactsSourceRecord(root)
	switch {
	case err == nil:
		if err := ensureContactsSourceFileReadable(resolvedWorkspacePath, source.Path, false); err != nil {
			return CapabilityRecord{}, err
		}
	case errors.Is(err, ErrContactsSourceRecordNotFound):
		source = &ContactsSourceRecord{
			SourceID:       ContactsLocalFileSourceID,
			CapabilityName: ContactsCapabilityName,
			Kind:           ContactsLocalFileSourceKind,
			Path:           ContactsLocalFileDefaultPath,
		}
		if err := ensureContactsSourceFileReadable(resolvedWorkspacePath, source.Path, true); err != nil {
			return CapabilityRecord{}, err
		}
		if err := StoreContactsSourceRecord(root, *source); err != nil {
			return CapabilityRecord{}, err
		}
	case err != nil:
		return CapabilityRecord{}, err
	}

	existing, err := ResolveContactsCapabilityRecord(root)
	switch {
	case err == nil:
		if strings.TrimSpace(existing.CapabilityID) != ContactsLocalFileCapabilityID {
			return CapabilityRecord{}, fmt.Errorf(
				"contacts capability record %q is not local-file-specific",
				existing.CapabilityID,
			)
		}
	case errors.Is(err, ErrCapabilityRecordNotFound):
		existing = nil
	case err != nil:
		return CapabilityRecord{}, err
	}

	record := CapabilityRecord{
		CapabilityID:  ContactsLocalFileCapabilityID,
		Class:         ContactsCapabilityName,
		Name:          ContactsCapabilityName,
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     ContactsLocalFileCapabilityValidator,
		Notes:         ContactsLocalFileCapabilityNotes,
	}
	if existing != nil {
		record.RecordVersion = existing.RecordVersion
	}
	if err := StoreCapabilityRecord(root, record); err != nil {
		return CapabilityRecord{}, err
	}
	return LoadCapabilityRecord(root, ContactsLocalFileCapabilityID)
}

func validateContactsSourceID(sourceID string) error {
	if err := validateCapabilityIDValue(sourceID); err != nil {
		return fmt.Errorf("mission store contacts source %w", err)
	}
	return nil
}

func normalizeContactsSourcePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(trimmed))
}

func validateContactsSourcePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("mission store contacts source path is required")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("mission store contacts source path %q must be relative to shared storage", trimmed)
	}
	normalized := normalizeContactsSourcePath(trimmed)
	if normalized == "" || normalized == "." || normalized == ".." {
		return fmt.Errorf("mission store contacts source path %q is invalid", trimmed)
	}
	if strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("mission store contacts source path %q is invalid", trimmed)
	}
	return nil
}

func resolveContactsSourceAbsolutePath(workspacePath string, relativePath string) (string, error) {
	resolvedWorkspacePath, err := resolveSharedStorageWorkspacePath(workspacePath)
	if err != nil {
		return "", err
	}
	if err := validateContactsSourcePath(relativePath); err != nil {
		return "", err
	}

	joined := filepath.Join(resolvedWorkspacePath, filepath.FromSlash(normalizeContactsSourcePath(relativePath)))
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("contacts capability exposure requires resolvable contacts source path: %w", err)
	}
	relativeCheck, err := filepath.Rel(resolvedWorkspacePath, resolved)
	if err != nil {
		return "", fmt.Errorf("contacts capability exposure requires workspace-relative contacts source path: %w", err)
	}
	if relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("contacts capability exposure requires contacts source path under configured workspace root")
	}
	return resolved, nil
}

func ensureContactsSourceFileReadable(workspacePath string, relativePath string, createIfMissing bool) error {
	resolved, err := resolveContactsSourceAbsolutePath(workspacePath, relativePath)
	if err != nil {
		return err
	}

	if createIfMissing {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return fmt.Errorf("contacts capability exposure requires contacts source parent directory: %w", err)
		}
		if _, err := os.Stat(resolved); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(resolved, []byte("[]\n"), 0o644); err != nil {
				return fmt.Errorf("contacts capability exposure requires writable contacts source file: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("contacts capability exposure requires contacts source file stat: %w", err)
		}
	}

	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("contacts capability exposure requires readable contacts source file %q", normalizeContactsSourcePath(relativePath))
		}
		return fmt.Errorf("contacts capability exposure requires readable contacts source file %q: %w", normalizeContactsSourcePath(relativePath), err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("contacts capability exposure requires contacts source file stat %q: %w", normalizeContactsSourcePath(relativePath), err)
	}
	if info.IsDir() {
		return fmt.Errorf("contacts capability exposure requires contacts source path %q to be a file", normalizeContactsSourcePath(relativePath))
	}
	return nil
}
