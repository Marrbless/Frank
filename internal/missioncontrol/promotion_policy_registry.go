package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

type PromotionPolicyRef struct {
	PromotionPolicyID string `json:"promotion_policy_id"`
}

type PromotionPolicyRecord struct {
	RecordVersion             int       `json:"record_version"`
	PromotionPolicyID         string    `json:"promotion_policy_id"`
	RequiresHoldoutPass       bool      `json:"requires_holdout_pass"`
	RequiresCanary            bool      `json:"requires_canary"`
	RequiresOwnerApproval     bool      `json:"requires_owner_approval"`
	AllowsAutonomousHotUpdate bool      `json:"allows_autonomous_hot_update"`
	AllowedSurfaceClasses     []string  `json:"allowed_surface_classes"`
	EpsilonRule               string    `json:"epsilon_rule"`
	RegressionRule            string    `json:"regression_rule"`
	CompatibilityRule         string    `json:"compatibility_rule"`
	ResourceRule              string    `json:"resource_rule"`
	MaxCanaryDuration         string    `json:"max_canary_duration"`
	ForbiddenSurfaceChanges   []string  `json:"forbidden_surface_changes"`
	CreatedAt                 time.Time `json:"created_at"`
	CreatedBy                 string    `json:"created_by"`
	Notes                     string    `json:"notes,omitempty"`
}

var ErrPromotionPolicyRecordNotFound = errors.New("mission store promotion policy record not found")

func StorePromotionPoliciesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "promotion_policies")
}

func StorePromotionPolicyPath(root, promotionPolicyID string) string {
	return filepath.Join(StorePromotionPoliciesDir(root), strings.TrimSpace(promotionPolicyID)+".json")
}

func NormalizePromotionPolicyRef(ref PromotionPolicyRef) PromotionPolicyRef {
	ref.PromotionPolicyID = strings.TrimSpace(ref.PromotionPolicyID)
	return ref
}

func NormalizePromotionPolicyRecord(record PromotionPolicyRecord) PromotionPolicyRecord {
	record.PromotionPolicyID = strings.TrimSpace(record.PromotionPolicyID)
	record.AllowedSurfaceClasses = normalizePromotionPolicyStrings(record.AllowedSurfaceClasses)
	record.EpsilonRule = strings.TrimSpace(record.EpsilonRule)
	record.RegressionRule = strings.TrimSpace(record.RegressionRule)
	record.CompatibilityRule = strings.TrimSpace(record.CompatibilityRule)
	record.ResourceRule = strings.TrimSpace(record.ResourceRule)
	record.MaxCanaryDuration = strings.TrimSpace(record.MaxCanaryDuration)
	record.ForbiddenSurfaceChanges = normalizePromotionPolicyStrings(record.ForbiddenSurfaceChanges)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	record.Notes = strings.TrimSpace(record.Notes)
	return record
}

func ValidatePromotionPolicyRef(ref PromotionPolicyRef) error {
	return validatePromotionPolicyIdentifierField("promotion policy ref", "promotion_policy_id", ref.PromotionPolicyID)
}

func ValidatePromotionPolicyRecord(record PromotionPolicyRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store promotion policy record_version must be positive")
	}
	if err := ValidatePromotionPolicyRef(PromotionPolicyRef{PromotionPolicyID: record.PromotionPolicyID}); err != nil {
		return err
	}
	if len(record.AllowedSurfaceClasses) == 0 {
		return fmt.Errorf("mission store promotion policy allowed_surface_classes are required")
	}
	for _, surfaceClass := range record.AllowedSurfaceClasses {
		if surfaceClass == "" {
			return fmt.Errorf("mission store promotion policy allowed_surface_classes must not contain empty values")
		}
	}
	if record.EpsilonRule == "" {
		return fmt.Errorf("mission store promotion policy epsilon_rule is required")
	}
	if record.RegressionRule == "" {
		return fmt.Errorf("mission store promotion policy regression_rule is required")
	}
	if record.CompatibilityRule == "" {
		return fmt.Errorf("mission store promotion policy compatibility_rule is required")
	}
	if record.ResourceRule == "" {
		return fmt.Errorf("mission store promotion policy resource_rule is required")
	}
	if err := validatePromotionPolicyMaxCanaryDuration(record.MaxCanaryDuration); err != nil {
		return err
	}
	if len(record.ForbiddenSurfaceChanges) == 0 {
		return fmt.Errorf("mission store promotion policy forbidden_surface_changes are required")
	}
	for _, surfaceChange := range record.ForbiddenSurfaceChanges {
		if surfaceChange == "" {
			return fmt.Errorf("mission store promotion policy forbidden_surface_changes must not contain empty values")
		}
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store promotion policy created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store promotion policy created_by is required")
	}
	return nil
}

func StorePromotionPolicyRecord(root string, record PromotionPolicyRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	record = NormalizePromotionPolicyRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidatePromotionPolicyRecord(record); err != nil {
		return err
	}

	path := StorePromotionPolicyPath(root, record.PromotionPolicyID)
	existing, err := loadPromotionPolicyRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return nil
		}
		return fmt.Errorf("mission store promotion policy %q already exists", record.PromotionPolicyID)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return WriteStoreJSONAtomic(path, record)
}

func LoadPromotionPolicyRecord(root, promotionPolicyID string) (PromotionPolicyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return PromotionPolicyRecord{}, err
	}
	ref := NormalizePromotionPolicyRef(PromotionPolicyRef{PromotionPolicyID: promotionPolicyID})
	if err := ValidatePromotionPolicyRef(ref); err != nil {
		return PromotionPolicyRecord{}, err
	}
	record, err := loadPromotionPolicyRecordFile(root, StorePromotionPolicyPath(root, ref.PromotionPolicyID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PromotionPolicyRecord{}, ErrPromotionPolicyRecordNotFound
		}
		return PromotionPolicyRecord{}, err
	}
	return record, nil
}

func ListPromotionPolicyRecords(root string) ([]PromotionPolicyRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(StorePromotionPoliciesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	records := make([]PromotionPolicyRecord, 0, len(names))
	for _, name := range names {
		record, err := loadPromotionPolicyRecordFile(root, filepath.Join(StorePromotionPoliciesDir(root), name))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func loadPromotionPolicyRecordFile(root, path string) (PromotionPolicyRecord, error) {
	var record PromotionPolicyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return PromotionPolicyRecord{}, err
	}
	record = NormalizePromotionPolicyRecord(record)
	if err := ValidatePromotionPolicyRecord(record); err != nil {
		return PromotionPolicyRecord{}, err
	}
	return record, nil
}

func validatePromotionPolicyIdentifierField(surface, fieldName, value string) error {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fmt.Errorf("%s %s is required", surface, fieldName)
	}
	if strings.HasPrefix(normalized, ".") || strings.HasSuffix(normalized, ".") {
		return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
	}
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
		}
	}
	return nil
}

func validatePromotionPolicyMaxCanaryDuration(value string) error {
	if value == "" {
		return fmt.Errorf("mission store promotion policy max_canary_duration is required")
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("mission store promotion policy max_canary_duration %q is invalid: %w", value, err)
	}
	if duration <= 0 {
		return fmt.Errorf("mission store promotion policy max_canary_duration must be positive")
	}
	return nil
}

func normalizePromotionPolicyStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}
