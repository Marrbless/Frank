package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type AutonomyBudgetResetWindow string

const (
	AutonomyBudgetResetWindowDailyUTC AutonomyBudgetResetWindow = "daily_utc"
)

const (
	AutonomyBudgetDebitKindExternalAction    = "external_action"
	AutonomyBudgetDebitKindHotUpdate         = "hot_update"
	AutonomyBudgetDebitKindCandidateMutation = "candidate_mutation"
	AutonomyBudgetDebitKindAPISpend          = "api_spend"
	AutonomyBudgetDebitKindRuntimeMinute     = "runtime_minute"
	AutonomyBudgetDebitUnitCount             = "count"
	AutonomyBudgetDebitUnitCents             = "cents"
	AutonomyBudgetDebitUnitMinutes           = "minutes"
)

type AutonomyBudgetRef struct {
	BudgetID string `json:"budget_id"`
}

type AutonomyBudgetRecord struct {
	RecordVersion                int                       `json:"record_version"`
	BudgetID                     string                    `json:"budget_id"`
	MaxExternalActionsPerDay     int64                     `json:"max_external_actions_per_day"`
	MaxHotUpdatesPerDay          int64                     `json:"max_hot_updates_per_day"`
	MaxCandidateMutationsPerDay  int64                     `json:"max_candidate_mutations_per_day"`
	MaxAPISpendPerDay            int64                     `json:"max_api_spend_per_day"`
	MaxRuntimeMinutesPerCycle    int64                     `json:"max_runtime_minutes_per_cycle"`
	MaxFailedAttemptsBeforePause int                       `json:"max_failed_attempts_before_pause"`
	QuietHours                   []string                  `json:"quiet_hours"`
	ResetWindow                  AutonomyBudgetResetWindow `json:"reset_window"`
	LedgerRefs                   []string                  `json:"ledger_refs"`
	CreatedAt                    time.Time                 `json:"created_at"`
	UpdatedAt                    time.Time                 `json:"updated_at"`
	CreatedBy                    string                    `json:"created_by"`
}

type AutonomyBudgetAssessment struct {
	BudgetID        string                      `json:"budget_id"`
	Allowed         bool                        `json:"allowed"`
	Reason          string                      `json:"reason,omitempty"`
	WindowStartedAt time.Time                   `json:"window_started_at"`
	WindowEndsAt    time.Time                   `json:"window_ends_at"`
	RequestedDebits []AutonomyBudgetDebitRecord `json:"requested_debits,omitempty"`
	UsedDebits      []AutonomyBudgetDebitRecord `json:"used_debits,omitempty"`
}

var ErrAutonomyBudgetRecordNotFound = errors.New("mission store autonomy budget record not found")

func StoreAutonomyBudgetsDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "budgets")
}

func StoreAutonomyBudgetPath(root, budgetID string) string {
	return filepath.Join(StoreAutonomyBudgetsDir(root), strings.TrimSpace(budgetID)+".json")
}

func NormalizeAutonomyBudgetRef(ref AutonomyBudgetRef) AutonomyBudgetRef {
	ref.BudgetID = strings.TrimSpace(ref.BudgetID)
	return ref
}

func NormalizeAutonomyBudgetRecord(record AutonomyBudgetRecord) AutonomyBudgetRecord {
	record.BudgetID = strings.TrimSpace(record.BudgetID)
	record.QuietHours = normalizeAutonomyStrings(record.QuietHours)
	record.ResetWindow = AutonomyBudgetResetWindow(strings.TrimSpace(string(record.ResetWindow)))
	record.LedgerRefs = normalizeAutonomyStrings(record.LedgerRefs)
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateAutonomyBudgetRef(ref AutonomyBudgetRef) error {
	return validateAutonomyIdentifierField("autonomy budget ref", "budget_id", ref.BudgetID)
}

func ValidateAutonomyBudgetRecord(record AutonomyBudgetRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store autonomy budget record_version must be positive")
	}
	if record.RecordVersion != StoreRecordVersion {
		return fmt.Errorf("mission store autonomy budget record_version %d is unsupported; want %d", record.RecordVersion, StoreRecordVersion)
	}
	if err := ValidateAutonomyBudgetRef(AutonomyBudgetRef{BudgetID: record.BudgetID}); err != nil {
		return err
	}
	if record.MaxExternalActionsPerDay < 0 {
		return fmt.Errorf("mission store autonomy budget max_external_actions_per_day must be non-negative")
	}
	if record.MaxHotUpdatesPerDay < 0 {
		return fmt.Errorf("mission store autonomy budget max_hot_updates_per_day must be non-negative")
	}
	if record.MaxCandidateMutationsPerDay < 0 {
		return fmt.Errorf("mission store autonomy budget max_candidate_mutations_per_day must be non-negative")
	}
	if record.MaxAPISpendPerDay < 0 {
		return fmt.Errorf("mission store autonomy budget max_api_spend_per_day must be non-negative")
	}
	if record.MaxRuntimeMinutesPerCycle <= 0 {
		return fmt.Errorf("mission store autonomy budget max_runtime_minutes_per_cycle must be positive")
	}
	if record.MaxFailedAttemptsBeforePause <= 0 {
		return fmt.Errorf("mission store autonomy budget max_failed_attempts_before_pause must be positive")
	}
	if len(record.QuietHours) == 0 {
		return fmt.Errorf("mission store autonomy budget quiet_hours are required")
	}
	if record.ResetWindow != AutonomyBudgetResetWindowDailyUTC {
		return fmt.Errorf("mission store autonomy budget reset_window %q is invalid", record.ResetWindow)
	}
	if len(record.LedgerRefs) == 0 {
		return fmt.Errorf("mission store autonomy budget ledger_refs are required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store autonomy budget created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store autonomy budget updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store autonomy budget updated_at must not be before created_at")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store autonomy budget created_by is required")
	}
	return nil
}

func StoreAutonomyBudgetRecord(root string, record AutonomyBudgetRecord) (AutonomyBudgetRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyBudgetRecord{}, false, err
	}
	record = NormalizeAutonomyBudgetRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateAutonomyBudgetRecord(record); err != nil {
		return AutonomyBudgetRecord{}, false, err
	}

	path := StoreAutonomyBudgetPath(root, record.BudgetID)
	if existing, err := loadAutonomyBudgetRecordFile(path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return AutonomyBudgetRecord{}, false, fmt.Errorf("mission store autonomy budget %q already exists", record.BudgetID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return AutonomyBudgetRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return AutonomyBudgetRecord{}, false, err
	}
	return record, true, nil
}

func LoadAutonomyBudgetRecord(root, budgetID string) (AutonomyBudgetRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyBudgetRecord{}, err
	}
	ref := NormalizeAutonomyBudgetRef(AutonomyBudgetRef{BudgetID: budgetID})
	if err := ValidateAutonomyBudgetRef(ref); err != nil {
		return AutonomyBudgetRecord{}, err
	}
	record, err := loadAutonomyBudgetRecordFile(StoreAutonomyBudgetPath(root, ref.BudgetID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AutonomyBudgetRecord{}, ErrAutonomyBudgetRecordNotFound
		}
		return AutonomyBudgetRecord{}, err
	}
	return record, nil
}

func ListAutonomyBudgetRecords(root string) ([]AutonomyBudgetRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreAutonomyBudgetsDir(root), loadAutonomyBudgetRecordFile)
}

func AssessAutonomyBudgetForDebits(root, budgetID string, requestedDebits []AutonomyBudgetDebitRecord, at time.Time) (AutonomyBudgetAssessment, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyBudgetAssessment{}, err
	}
	budget, err := LoadAutonomyBudgetRecord(root, budgetID)
	if err != nil {
		return AutonomyBudgetAssessment{}, err
	}
	at = at.UTC()
	if at.IsZero() {
		return AutonomyBudgetAssessment{}, fmt.Errorf("autonomy budget assessment time is required")
	}
	windowStart, windowEnd := autonomyBudgetWindow(budget, at)
	requested := normalizeAutonomyBudgetDebits(requestedDebits)
	for i, debit := range requested {
		if err := ValidateAutonomyBudgetDebitRecord(debit); err != nil {
			return AutonomyBudgetAssessment{}, fmt.Errorf("autonomy budget requested_debits[%d]: %w", i, err)
		}
		if debit.BudgetID != budget.BudgetID {
			return AutonomyBudgetAssessment{}, fmt.Errorf("autonomy budget requested_debits[%d] budget_id %q does not match budget %q", i, debit.BudgetID, budget.BudgetID)
		}
	}

	used, err := autonomyBudgetUsedDebits(root, budget.BudgetID, windowStart, windowEnd)
	if err != nil {
		return AutonomyBudgetAssessment{}, err
	}
	assessment := AutonomyBudgetAssessment{
		BudgetID:        budget.BudgetID,
		Allowed:         true,
		WindowStartedAt: windowStart,
		WindowEndsAt:    windowEnd,
		RequestedDebits: requested,
		UsedDebits:      used,
	}
	if reason := autonomyBudgetExceededReason(budget, used, requested); reason != "" {
		assessment.Allowed = false
		assessment.Reason = string(RejectionCodeV4AutonomyBudgetExceeded) + ": " + reason
	}
	return assessment, nil
}

func loadAutonomyBudgetRecordFile(path string) (AutonomyBudgetRecord, error) {
	var record AutonomyBudgetRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return AutonomyBudgetRecord{}, err
	}
	record = NormalizeAutonomyBudgetRecord(record)
	if err := ValidateAutonomyBudgetRecord(record); err != nil {
		return AutonomyBudgetRecord{}, err
	}
	return record, nil
}

func normalizeAutonomyBudgetDebits(debits []AutonomyBudgetDebitRecord) []AutonomyBudgetDebitRecord {
	if len(debits) == 0 {
		return nil
	}
	normalized := make([]AutonomyBudgetDebitRecord, 0, len(debits))
	for _, debit := range debits {
		debit = NormalizeAutonomyBudgetDebitRecord(debit)
		if debit.BudgetID == "" && debit.DebitKind == "" && debit.Amount == 0 && debit.Unit == "" {
			continue
		}
		normalized = append(normalized, debit)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func autonomyBudgetWindow(record AutonomyBudgetRecord, at time.Time) (time.Time, time.Time) {
	at = at.UTC()
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC), time.Date(at.Year(), at.Month(), at.Day()+1, 0, 0, 0, 0, time.UTC)
}

func autonomyBudgetUsedDebits(root, budgetID string, windowStart, windowEnd time.Time) ([]AutonomyBudgetDebitRecord, error) {
	wakeCycles, err := ListWakeCycleRecords(root)
	if err != nil {
		return nil, err
	}
	var used []AutonomyBudgetDebitRecord
	for _, cycle := range wakeCycles {
		if cycle.StartedAt.Before(windowStart) || !cycle.StartedAt.Before(windowEnd) {
			continue
		}
		for _, debit := range cycle.BudgetDebits {
			if debit.BudgetID == budgetID {
				used = append(used, debit)
			}
		}
	}
	return used, nil
}

func autonomyBudgetExceededReason(budget AutonomyBudgetRecord, used, requested []AutonomyBudgetDebitRecord) string {
	usedByKind := sumAutonomyBudgetDebitsByKind(used)
	requestedByKind := sumAutonomyBudgetDebitsByKind(requested)
	check := func(kind string, max int64) string {
		if usedByKind[kind]+requestedByKind[kind] > max {
			return fmt.Sprintf("%s would exceed %d", kind, max)
		}
		return ""
	}
	if reason := check(AutonomyBudgetDebitKindExternalAction, budget.MaxExternalActionsPerDay); reason != "" {
		return reason
	}
	if reason := check(AutonomyBudgetDebitKindHotUpdate, budget.MaxHotUpdatesPerDay); reason != "" {
		return reason
	}
	if reason := check(AutonomyBudgetDebitKindCandidateMutation, budget.MaxCandidateMutationsPerDay); reason != "" {
		return reason
	}
	if reason := check(AutonomyBudgetDebitKindAPISpend, budget.MaxAPISpendPerDay); reason != "" {
		return reason
	}
	if requestedByKind[AutonomyBudgetDebitKindRuntimeMinute] > budget.MaxRuntimeMinutesPerCycle {
		return fmt.Sprintf("%s would exceed per-cycle limit %d", AutonomyBudgetDebitKindRuntimeMinute, budget.MaxRuntimeMinutesPerCycle)
	}
	return ""
}

func sumAutonomyBudgetDebitsByKind(debits []AutonomyBudgetDebitRecord) map[string]int64 {
	sums := map[string]int64{}
	for _, debit := range debits {
		sums[debit.DebitKind] += debit.Amount
	}
	return sums
}

func isValidAutonomyBudgetDebitKind(kind string) bool {
	switch kind {
	case AutonomyBudgetDebitKindExternalAction,
		AutonomyBudgetDebitKindHotUpdate,
		AutonomyBudgetDebitKindCandidateMutation,
		AutonomyBudgetDebitKindAPISpend,
		AutonomyBudgetDebitKindRuntimeMinute:
		return true
	default:
		return false
	}
}

func isValidAutonomyBudgetDebitUnitForKind(kind, unit string) bool {
	switch kind {
	case AutonomyBudgetDebitKindAPISpend:
		return unit == AutonomyBudgetDebitUnitCents
	case AutonomyBudgetDebitKindRuntimeMinute:
		return unit == AutonomyBudgetDebitUnitMinutes
	case AutonomyBudgetDebitKindExternalAction,
		AutonomyBudgetDebitKindHotUpdate,
		AutonomyBudgetDebitKindCandidateMutation:
		return unit == AutonomyBudgetDebitUnitCount
	default:
		return false
	}
}
