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

type CapabilityOnboardingProposalState string

const (
	CapabilityOnboardingProposalStateProposed CapabilityOnboardingProposalState = "proposed"
	CapabilityOnboardingProposalStateApproved CapabilityOnboardingProposalState = "approved"
	CapabilityOnboardingProposalStateRejected CapabilityOnboardingProposalState = "rejected"
	CapabilityOnboardingProposalStateArchived CapabilityOnboardingProposalState = "archived"
)

type CapabilityOnboardingProposalRecord struct {
	RecordVersion    int                               `json:"record_version"`
	ProposalID       string                            `json:"proposal_id"`
	CapabilityName   string                            `json:"capability_name"`
	WhyNeeded        string                            `json:"why_needed"`
	MissionFamilies  []string                          `json:"mission_families"`
	Risks            []string                          `json:"risks"`
	Validators       []string                          `json:"validators"`
	KillSwitch       string                            `json:"kill_switch"`
	DataAccessed     []string                          `json:"data_accessed"`
	ApprovalRequired bool                              `json:"approval_required"`
	CreatedAt        time.Time                         `json:"created_at"`
	State            CapabilityOnboardingProposalState `json:"state"`
}

var ErrCapabilityOnboardingProposalRecordNotFound = errors.New("mission store capability onboarding proposal record not found")

func StoreCapabilityOnboardingProposalsDir(root string) string {
	return filepath.Join(root, "capability_onboarding_proposals")
}

func StoreCapabilityOnboardingProposalPath(root, proposalID string) string {
	return filepath.Join(StoreCapabilityOnboardingProposalsDir(root), proposalID+".json")
}

func NormalizeCapabilityOnboardingProposalState(state CapabilityOnboardingProposalState) CapabilityOnboardingProposalState {
	return CapabilityOnboardingProposalState(strings.TrimSpace(string(state)))
}

func NormalizeCapabilityOnboardingProposalRecord(record CapabilityOnboardingProposalRecord) CapabilityOnboardingProposalRecord {
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	record.ProposalID = strings.TrimSpace(record.ProposalID)
	record.CapabilityName = strings.TrimSpace(record.CapabilityName)
	record.WhyNeeded = strings.TrimSpace(record.WhyNeeded)
	record.MissionFamilies = normalizeCapabilityOnboardingProposalStrings(record.MissionFamilies)
	record.Risks = normalizeCapabilityOnboardingProposalStrings(record.Risks)
	record.Validators = normalizeCapabilityOnboardingProposalStrings(record.Validators)
	record.KillSwitch = strings.TrimSpace(record.KillSwitch)
	record.DataAccessed = normalizeCapabilityOnboardingProposalStrings(record.DataAccessed)
	record.CreatedAt = record.CreatedAt.UTC()
	record.State = NormalizeCapabilityOnboardingProposalState(record.State)
	return record
}

func ValidateCapabilityOnboardingProposalRecord(record CapabilityOnboardingProposalRecord) error {
	record = NormalizeCapabilityOnboardingProposalRecord(record)
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store capability onboarding proposal record_version must be positive")
	}
	if err := validateCapabilityOnboardingProposalID(record.ProposalID, "mission store capability onboarding proposal"); err != nil {
		return err
	}
	if record.CapabilityName == "" {
		return fmt.Errorf("mission store capability onboarding proposal capability_name is required")
	}
	if record.WhyNeeded == "" {
		return fmt.Errorf("mission store capability onboarding proposal why_needed is required")
	}
	if err := validateCapabilityOnboardingProposalStrings(record.MissionFamilies, "mission_families"); err != nil {
		return err
	}
	if err := validateCapabilityOnboardingProposalStrings(record.Risks, "risks"); err != nil {
		return err
	}
	if err := validateCapabilityOnboardingProposalStrings(record.Validators, "validators"); err != nil {
		return err
	}
	if record.KillSwitch == "" {
		return fmt.Errorf("mission store capability onboarding proposal kill_switch is required")
	}
	if err := validateCapabilityOnboardingProposalStrings(record.DataAccessed, "data_accessed"); err != nil {
		return err
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store capability onboarding proposal created_at is required")
	}
	if err := validateCapabilityOnboardingProposalState(record.State); err != nil {
		return err
	}
	return nil
}

func StoreCapabilityOnboardingProposalRecord(root string, record CapabilityOnboardingProposalRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizeCapabilityOnboardingProposalRecord(record)
	if err := ValidateCapabilityOnboardingProposalRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCapabilityOnboardingProposalPath(root, record.ProposalID), record)
}

func LoadCapabilityOnboardingProposalRecord(root, proposalID string) (CapabilityOnboardingProposalRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CapabilityOnboardingProposalRecord{}, err
	}
	if err := validateCapabilityOnboardingProposalID(proposalID, "mission store capability onboarding proposal"); err != nil {
		return CapabilityOnboardingProposalRecord{}, err
	}
	record, err := loadCapabilityOnboardingProposalRecordFile(StoreCapabilityOnboardingProposalPath(root, strings.TrimSpace(proposalID)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CapabilityOnboardingProposalRecord{}, ErrCapabilityOnboardingProposalRecordNotFound
		}
		return CapabilityOnboardingProposalRecord{}, err
	}
	return record, nil
}

func ListCapabilityOnboardingProposalRecords(root string) ([]CapabilityOnboardingProposalRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreCapabilityOnboardingProposalsDir(root), loadCapabilityOnboardingProposalRecordFile)
}

func loadCapabilityOnboardingProposalRecordFile(path string) (CapabilityOnboardingProposalRecord, error) {
	var record CapabilityOnboardingProposalRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CapabilityOnboardingProposalRecord{}, err
	}
	record = NormalizeCapabilityOnboardingProposalRecord(record)
	if err := ValidateCapabilityOnboardingProposalRecord(record); err != nil {
		return CapabilityOnboardingProposalRecord{}, err
	}
	return record, nil
}

func validateCapabilityOnboardingProposalID(proposalID string, surface string) error {
	if err := validateCapabilityOnboardingProposalIDValue(proposalID); err != nil {
		return fmt.Errorf("%s %w", surface, err)
	}
	return nil
}

func validateCapabilityOnboardingProposalIDValue(proposalID string) error {
	normalized := strings.TrimSpace(proposalID)
	if normalized == "" {
		return fmt.Errorf("proposal_id is required")
	}
	if normalized == "." || normalized == ".." {
		return fmt.Errorf("proposal_id %q is invalid", normalized)
	}
	if strings.ContainsAny(normalized, `/\`) {
		return fmt.Errorf("proposal_id %q is invalid", normalized)
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("proposal_id %q is invalid", normalized)
		}
	}
	return nil
}

func normalizeCapabilityOnboardingProposalStrings(values []string) []string {
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

func validateCapabilityOnboardingProposalStrings(values []string, field string) error {
	if len(values) == 0 {
		return fmt.Errorf("mission store capability onboarding proposal %s is required", field)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("mission store capability onboarding proposal %s entries must be non-empty", field)
		}
	}
	return nil
}

func validateCapabilityOnboardingProposalState(state CapabilityOnboardingProposalState) error {
	switch NormalizeCapabilityOnboardingProposalState(state) {
	case CapabilityOnboardingProposalStateProposed,
		CapabilityOnboardingProposalStateApproved,
		CapabilityOnboardingProposalStateRejected,
		CapabilityOnboardingProposalStateArchived:
		return nil
	default:
		return fmt.Errorf("mission store capability onboarding proposal state %q is invalid", strings.TrimSpace(string(state)))
	}
}
