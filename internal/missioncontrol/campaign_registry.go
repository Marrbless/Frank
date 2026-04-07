package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CampaignKind string

const (
	CampaignKindOutreach  CampaignKind = "outreach"
	CampaignKindCommunity CampaignKind = "community"
)

type CampaignState string

const (
	CampaignStateDraft    CampaignState = "draft"
	CampaignStateActive   CampaignState = "active"
	CampaignStateStopped  CampaignState = "stopped"
	CampaignStateArchived CampaignState = "archived"
)

type CampaignFailureThreshold struct {
	Metric string `json:"metric"`
	Limit  int    `json:"limit"`
}

type CampaignRecord struct {
	RecordVersion           int                            `json:"record_version"`
	CampaignID              string                         `json:"campaign_id"`
	CampaignKind            CampaignKind                   `json:"campaign_kind"`
	DisplayName             string                         `json:"display_name"`
	State                   CampaignState                  `json:"state"`
	Objective               string                         `json:"objective"`
	GovernedExternalTargets []AutonomyEligibilityTargetRef `json:"governed_external_targets"`
	FrankObjectRefs         []FrankRegistryObjectRef       `json:"frank_object_refs"`
	IdentityMode            IdentityMode                   `json:"identity_mode"`
	StopConditions          []string                       `json:"stop_conditions"`
	FailureThreshold        CampaignFailureThreshold       `json:"failure_threshold"`
	ComplianceChecks        []string                       `json:"compliance_checks"`
	CreatedAt               time.Time                      `json:"created_at"`
	UpdatedAt               time.Time                      `json:"updated_at"`
}

var ErrCampaignRecordNotFound = errors.New("mission store campaign record not found")

func StoreCampaignsDir(root string) string {
	return filepath.Join(root, "campaigns")
}

func StoreCampaignPath(root, campaignID string) string {
	return filepath.Join(StoreCampaignsDir(root), campaignID+".json")
}

func NormalizeCampaignKind(kind CampaignKind) CampaignKind {
	return CampaignKind(strings.TrimSpace(string(kind)))
}

func NormalizeCampaignState(state CampaignState) CampaignState {
	return CampaignState(strings.TrimSpace(string(state)))
}

func ValidateCampaignRecord(record CampaignRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store campaign record_version must be positive")
	}
	if strings.TrimSpace(record.CampaignID) == "" {
		return fmt.Errorf("mission store campaign campaign_id is required")
	}
	if !isValidCampaignKind(record.CampaignKind) {
		return fmt.Errorf("mission store campaign campaign_kind %q is invalid", strings.TrimSpace(string(record.CampaignKind)))
	}
	if strings.TrimSpace(record.DisplayName) == "" {
		return fmt.Errorf("mission store campaign display_name is required")
	}
	if !isValidCampaignState(record.State) {
		return fmt.Errorf("mission store campaign state %q is invalid", strings.TrimSpace(string(record.State)))
	}
	if strings.TrimSpace(record.Objective) == "" {
		return fmt.Errorf("mission store campaign objective is required")
	}
	if err := validateCampaignGovernedExternalTargets(record.GovernedExternalTargets); err != nil {
		return err
	}
	if err := validateCampaignFrankObjectRefs(record.FrankObjectRefs); err != nil {
		return err
	}
	if err := validateIdentityMode(record.IdentityMode); err != nil {
		return err
	}
	if err := validateCampaignStopConditions(record.StopConditions); err != nil {
		return err
	}
	if err := ValidateCampaignFailureThreshold(record.FailureThreshold); err != nil {
		return err
	}
	if err := validateCampaignComplianceChecks(record.ComplianceChecks); err != nil {
		return err
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store campaign created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store campaign updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store campaign updated_at must be on or after created_at")
	}
	return nil
}

func ValidateCampaignFailureThreshold(threshold CampaignFailureThreshold) error {
	if strings.TrimSpace(threshold.Metric) == "" {
		return fmt.Errorf("mission store campaign failure_threshold.metric is required")
	}
	if threshold.Limit <= 0 {
		return fmt.Errorf("mission store campaign failure_threshold.limit must be positive")
	}
	return nil
}

func StoreCampaignRecord(root string, record CampaignRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = normalizeCampaignRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateCampaignRecord(record); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCampaignPath(root, record.CampaignID), record)
}

func LoadCampaignRecord(root, campaignID string) (CampaignRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CampaignRecord{}, err
	}
	if strings.TrimSpace(campaignID) == "" {
		return CampaignRecord{}, fmt.Errorf("mission store campaign campaign_id is required")
	}
	record, err := loadCampaignRecordFile(StoreCampaignPath(root, campaignID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CampaignRecord{}, ErrCampaignRecordNotFound
		}
		return CampaignRecord{}, err
	}
	return record, nil
}

func ListCampaignRecords(root string) ([]CampaignRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreCampaignsDir(root), loadCampaignRecordFile)
}

func loadCampaignRecordFile(path string) (CampaignRecord, error) {
	var record CampaignRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CampaignRecord{}, err
	}
	record = normalizeCampaignRecord(record)
	if err := ValidateCampaignRecord(record); err != nil {
		return CampaignRecord{}, err
	}
	return record, nil
}

func normalizeCampaignRecord(record CampaignRecord) CampaignRecord {
	record.CampaignID = strings.TrimSpace(record.CampaignID)
	record.CampaignKind = NormalizeCampaignKind(record.CampaignKind)
	record.DisplayName = strings.TrimSpace(record.DisplayName)
	record.State = NormalizeCampaignState(record.State)
	record.Objective = strings.TrimSpace(record.Objective)
	record.GovernedExternalTargets = normalizeCampaignGovernedExternalTargets(record.GovernedExternalTargets)
	record.FrankObjectRefs = normalizeFrankRegistryObjectRefs(record.FrankObjectRefs)
	record.IdentityMode = NormalizeIdentityMode(record.IdentityMode)
	record.StopConditions = normalizeCampaignStringList(record.StopConditions)
	record.FailureThreshold.Metric = strings.TrimSpace(record.FailureThreshold.Metric)
	record.ComplianceChecks = normalizeCampaignStringList(record.ComplianceChecks)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	return record
}

func normalizeCampaignGovernedExternalTargets(targets []AutonomyEligibilityTargetRef) []AutonomyEligibilityTargetRef {
	if len(targets) == 0 {
		return nil
	}

	normalized := make([]AutonomyEligibilityTargetRef, len(targets))
	for i, target := range targets {
		normalized[i] = target
		normalized[i].RegistryID = strings.TrimSpace(target.RegistryID)
	}
	return normalized
}

func normalizeCampaignStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, len(values))
	for i, value := range values {
		normalized[i] = strings.TrimSpace(value)
	}
	return normalized
}

func validateCampaignGovernedExternalTargets(targets []AutonomyEligibilityTargetRef) error {
	if len(targets) == 0 {
		return fmt.Errorf("mission store campaign governed_external_targets are required")
	}

	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if err := validateAutonomyEligibilityTargetRef(target); err != nil {
			return fmt.Errorf("mission store campaign governed_external_targets contain invalid target: %w", err)
		}
		key := normalizedGovernedExternalTargetKey(target)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("mission store campaign governed_external_targets contain duplicate target kind %q registry_id %q", target.Kind, strings.TrimSpace(target.RegistryID))
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateCampaignFrankObjectRefs(refs []FrankRegistryObjectRef) error {
	if len(refs) == 0 {
		return fmt.Errorf("mission store campaign frank_object_refs are required")
	}

	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		normalized := NormalizeFrankRegistryObjectRef(ref)
		if err := validateFrankRegistryObjectRef(normalized); err != nil {
			return fmt.Errorf("mission store campaign frank_object_refs contain invalid ref: %w", err)
		}
		key := normalizedFrankRegistryObjectRefKey(normalized)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("mission store campaign frank_object_refs contain duplicate ref kind %q object_id %q", normalized.Kind, normalized.ObjectID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateCampaignStopConditions(conditions []string) error {
	if len(conditions) == 0 {
		return fmt.Errorf("mission store campaign stop_conditions are required")
	}
	for _, condition := range conditions {
		if strings.TrimSpace(condition) == "" {
			return fmt.Errorf("mission store campaign stop_conditions must not contain blanks")
		}
	}
	return nil
}

func validateCampaignComplianceChecks(checks []string) error {
	if len(checks) == 0 {
		return fmt.Errorf("mission store campaign compliance_checks are required")
	}
	for _, check := range checks {
		if strings.TrimSpace(check) == "" {
			return fmt.Errorf("mission store campaign compliance_checks must not contain blanks")
		}
	}
	return nil
}

func isValidCampaignKind(kind CampaignKind) bool {
	switch NormalizeCampaignKind(kind) {
	case CampaignKindOutreach, CampaignKindCommunity:
		return true
	default:
		return false
	}
}

func isValidCampaignState(state CampaignState) bool {
	switch NormalizeCampaignState(state) {
	case CampaignStateDraft, CampaignStateActive, CampaignStateStopped, CampaignStateArchived:
		return true
	default:
		return false
	}
}
