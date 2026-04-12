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

// CampaignObjectView is an adapter-only surface that exposes the
// currently-grounded subset of the frozen V3 campaign contract without
// forcing a durable storage migration or creating a second source of truth.
//
// platform_or_channel may be derived when current governed targets honestly
// support that projection. The remaining spec-facing campaign envelope fields
// below are intentionally non-durable for now:
// audience_class_or_target
// message_family_or_participation_style
// cadence
// escalation_rules
// budget
//
// Those fields must remain zero-valued until storage has a justified durable
// source owned by a real control-plane producer/consumer.
type CampaignObjectView struct {
	CampaignID        string       `json:"campaign_id"`
	CampaignKind      CampaignKind `json:"campaign_kind"`
	Objective         string       `json:"objective"`
	PlatformOrChannel string       `json:"platform_or_channel,omitempty"`
	// AudienceClassOrTarget is intentionally non-durable and must remain a
	// zero value until campaign storage grows a justified durable source.
	AudienceClassOrTarget string       `json:"audience_class_or_target,omitempty"`
	IdentityMode          IdentityMode `json:"identity_mode"`
	// MessageFamilyOrParticipationStyle is intentionally non-durable and must
	// remain a zero value until campaign storage grows a justified durable source.
	MessageFamilyOrParticipationStyle string `json:"message_family_or_participation_style,omitempty"`
	// Cadence is intentionally non-durable and must remain a zero value until
	// campaign storage grows a justified durable source.
	Cadence string `json:"cadence,omitempty"`
	// EscalationRules is intentionally non-durable and must remain a zero value
	// until campaign storage grows a justified durable source.
	EscalationRules         []string                       `json:"escalation_rules,omitempty"`
	GovernedExternalTargets []AutonomyEligibilityTargetRef `json:"governed_external_targets,omitempty"`
	FrankObjectRefs         []FrankRegistryObjectRef       `json:"frank_object_refs,omitempty"`
	StopConditions          []string                       `json:"stop_conditions"`
	FailureThreshold        CampaignFailureThreshold       `json:"failure_threshold"`
	ComplianceChecks        []string                       `json:"compliance_checks"`
	// Budget is intentionally non-durable and must remain a zero value until
	// campaign storage grows a justified durable source.
	Budget    string    `json:"budget,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ResolvedExecutionContextCampaignPreflight struct {
	Campaign   *CampaignRecord        `json:"campaign,omitempty"`
	Identities []FrankIdentityRecord  `json:"identities,omitempty"`
	Accounts   []FrankAccountRecord   `json:"accounts,omitempty"`
	Containers []FrankContainerRecord `json:"containers,omitempty"`
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
	if err := validateCampaignID(record.CampaignID, "mission store campaign"); err != nil {
		return err
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
	if err := ValidateCampaignGovernedTargetLinks(root, record.GovernedExternalTargets); err != nil {
		return err
	}
	if err := ValidateCampaignFrankObjectRefLinks(root, record.FrankObjectRefs); err != nil {
		return err
	}
	return WriteStoreJSONAtomic(StoreCampaignPath(root, record.CampaignID), record)
}

func LoadCampaignRecord(root, campaignID string) (CampaignRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CampaignRecord{}, err
	}
	if err := validateCampaignID(campaignID, "mission store campaign"); err != nil {
		return CampaignRecord{}, err
	}
	normalizedCampaignID := strings.TrimSpace(campaignID)
	record, err := loadCampaignRecordFile(root, StoreCampaignPath(root, normalizedCampaignID))
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
	return listStoreJSONRecords(StoreCampaignsDir(root), func(path string) (CampaignRecord, error) {
		return loadCampaignRecordFile(root, path)
	})
}

func loadCampaignRecordFile(root, path string) (CampaignRecord, error) {
	var record CampaignRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return CampaignRecord{}, err
	}
	record = normalizeCampaignRecord(record)
	if err := ValidateCampaignRecord(record); err != nil {
		return CampaignRecord{}, err
	}
	if err := ValidateCampaignGovernedTargetLinks(root, record.GovernedExternalTargets); err != nil {
		return CampaignRecord{}, err
	}
	if err := ValidateCampaignFrankObjectRefLinks(root, record.FrankObjectRefs); err != nil {
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

// AsObjectView adapts durable campaign storage to the current frozen-spec
// campaign view without inventing new persisted truth. Only
// platform_or_channel may be derived today, and only when governed targets
// support that projection honestly. The remaining unresolved campaign envelope
// fields, audience_class_or_target, message_family_or_participation_style,
// cadence, escalation_rules, and budget, are adapter-only, intentionally
// non-durable, and must remain zero-valued until a future justified storage
// migration exists.
func (record CampaignRecord) AsObjectView() CampaignObjectView {
	platformOrChannel, _ := ResolveCampaignPlatformOrChannel(record)
	return CampaignObjectView{
		CampaignID:              record.CampaignID,
		CampaignKind:            record.CampaignKind,
		Objective:               record.Objective,
		PlatformOrChannel:       platformOrChannel,
		IdentityMode:            record.IdentityMode,
		GovernedExternalTargets: record.GovernedExternalTargets,
		FrankObjectRefs:         record.FrankObjectRefs,
		StopConditions:          record.StopConditions,
		FailureThreshold:        record.FailureThreshold,
		ComplianceChecks:        record.ComplianceChecks,
		CreatedAt:               record.CreatedAt,
		UpdatedAt:               record.UpdatedAt,
	}
}

// ResolveCampaignPlatformOrChannel derives the spec-facing platform_or_channel
// only when current governed targets provide one honest, unambiguous provider
// or platform source. It does not infer any of the other unresolved envelope
// fields, which must remain non-durable zero values for now.
func ResolveCampaignPlatformOrChannel(record CampaignRecord) (string, bool) {
	if len(record.GovernedExternalTargets) != 1 {
		return "", false
	}
	target := record.GovernedExternalTargets[0]
	switch target.Kind {
	case EligibilityTargetKindPlatform, EligibilityTargetKindProvider:
		if strings.TrimSpace(target.RegistryID) == "" {
			return "", false
		}
		return strings.TrimSpace(target.RegistryID), true
	default:
		return "", false
	}
}

func ValidateCampaignRef(ref CampaignRef) error {
	return validateCampaignIDValue(NormalizeCampaignRef(ref).CampaignID)
}

func ResolveCampaignRef(root string, ref CampaignRef) (CampaignRecord, error) {
	normalized := NormalizeCampaignRef(ref)
	if err := ValidateCampaignRef(normalized); err != nil {
		return CampaignRecord{}, err
	}
	return LoadCampaignRecord(root, normalized.CampaignID)
}

func ResolveExecutionContextCampaignRef(ec ExecutionContext) (*CampaignRecord, error) {
	if ec.Step == nil {
		return nil, fmt.Errorf("execution context step is required")
	}
	if ec.Step.CampaignRef == nil {
		return nil, nil
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return nil, fmt.Errorf("mission store root is required to resolve campaign refs")
	}

	record, err := ResolveCampaignRef(ec.MissionStoreRoot, *ec.Step.CampaignRef)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func ResolveExecutionContextCampaignPreflight(ec ExecutionContext) (ResolvedExecutionContextCampaignPreflight, error) {
	campaign, err := ResolveExecutionContextCampaignRef(ec)
	if err != nil {
		return ResolvedExecutionContextCampaignPreflight{}, err
	}
	if campaign == nil {
		return ResolvedExecutionContextCampaignPreflight{}, nil
	}

	resolvedRefs, err := ResolveFrankRegistryObjectRefs(ec.MissionStoreRoot, campaign.FrankObjectRefs)
	if err != nil {
		return ResolvedExecutionContextCampaignPreflight{}, err
	}

	preflight := ResolvedExecutionContextCampaignPreflight{
		Campaign: campaign,
	}
	if len(resolvedRefs) == 0 {
		return preflight, nil
	}

	for _, resolved := range resolvedRefs {
		switch {
		case resolved.Identity != nil:
			preflight.Identities = append(preflight.Identities, *resolved.Identity)
		case resolved.Account != nil:
			preflight.Accounts = append(preflight.Accounts, *resolved.Account)
		case resolved.Container != nil:
			preflight.Containers = append(preflight.Containers, *resolved.Container)
		default:
			return ResolvedExecutionContextCampaignPreflight{}, fmt.Errorf("resolve campaign Frank object ref kind %q object_id %q: expected Frank registry object record", resolved.Ref.Kind, resolved.Ref.ObjectID)
		}
	}

	return preflight, nil
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

func ValidateCampaignGovernedTargetLinks(root string, targets []AutonomyEligibilityTargetRef) error {
	for _, target := range targets {
		if _, err := ValidateFrankRegistryEligibilityLink(root, target); err != nil {
			return fmt.Errorf(
				"mission store campaign governed_external_targets target kind %q registry_id %q: %w",
				strings.TrimSpace(string(target.Kind)),
				strings.TrimSpace(target.RegistryID),
				err,
			)
		}
	}
	return nil
}

func ValidateCampaignFrankObjectRefLinks(root string, refs []FrankRegistryObjectRef) error {
	for _, ref := range refs {
		if _, err := ResolveFrankRegistryObjectRef(root, ref); err != nil {
			return fmt.Errorf(
				"mission store campaign frank_object_refs ref kind %q object_id %q: %w",
				strings.TrimSpace(string(ref.Kind)),
				strings.TrimSpace(ref.ObjectID),
				err,
			)
		}
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

func validateCampaignID(campaignID string, surface string) error {
	if err := validateCampaignIDValue(campaignID); err != nil {
		return fmt.Errorf("%s %w", surface, err)
	}
	return nil
}

func validateCampaignIDValue(campaignID string) error {
	normalized := strings.TrimSpace(campaignID)
	if normalized == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if normalized == "." || normalized == ".." {
		return fmt.Errorf("campaign_id %q is invalid", normalized)
	}
	if strings.ContainsAny(normalized, `/\`) {
		return fmt.Errorf("campaign_id %q is invalid", normalized)
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("campaign_id %q is invalid", normalized)
		}
	}
	return nil
}
