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
)

type AutonomyFailureKind string

const (
	AutonomyFailureKindWakeCycle AutonomyFailureKind = "wake_cycle"
	AutonomyFailureKindEval      AutonomyFailureKind = "eval"
	AutonomyFailureKindRuntime   AutonomyFailureKind = "runtime"
)

type AutonomyPauseKind string

const (
	AutonomyPauseKindRepeatedFailure AutonomyPauseKind = "repeated_failure"
)

type AutonomyPauseState string

const (
	AutonomyPauseStateActive AutonomyPauseState = "active"
)

type AutonomyFailureRecord struct {
	RecordVersion       int                 `json:"record_version"`
	FailureID           string              `json:"failure_id"`
	WakeCycleID         string              `json:"wake_cycle_id"`
	StandingDirectiveID string              `json:"standing_directive_id,omitempty"`
	BudgetID            string              `json:"budget_id"`
	FailureKind         AutonomyFailureKind `json:"failure_kind"`
	Reason              string              `json:"reason"`
	OccurredAt          time.Time           `json:"occurred_at"`
	CreatedAt           time.Time           `json:"created_at"`
	CreatedBy           string              `json:"created_by"`
}

type AutonomyPauseRecord struct {
	RecordVersion       int                `json:"record_version"`
	PauseID             string             `json:"pause_id"`
	BudgetID            string             `json:"budget_id"`
	StandingDirectiveID string             `json:"standing_directive_id,omitempty"`
	PauseKind           AutonomyPauseKind  `json:"pause_kind"`
	State               AutonomyPauseState `json:"state"`
	Reason              string             `json:"reason"`
	FailureIDs          []string           `json:"failure_ids"`
	PausedAt            time.Time          `json:"paused_at"`
	CreatedAt           time.Time          `json:"created_at"`
	CreatedBy           string             `json:"created_by"`
}

var (
	ErrAutonomyFailureRecordNotFound = errors.New("mission store autonomy failure record not found")
	ErrAutonomyPauseRecordNotFound   = errors.New("mission store autonomy pause record not found")
)

func StoreAutonomyFailuresDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "failures")
}

func StoreAutonomyFailurePath(root, failureID string) string {
	return filepath.Join(StoreAutonomyFailuresDir(root), strings.TrimSpace(failureID)+".json")
}

func StoreAutonomyPausesDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "pauses")
}

func StoreAutonomyPausePath(root, pauseID string) string {
	return filepath.Join(StoreAutonomyPausesDir(root), strings.TrimSpace(pauseID)+".json")
}

func AutonomyFailureIDFromWakeCycle(wakeCycleID string) string {
	return "autonomy-failure-" + strings.TrimSpace(wakeCycleID)
}

func AutonomyRepeatedFailurePauseID(budgetID string) string {
	return "autonomy-pause-repeated-failure-" + strings.TrimSpace(budgetID)
}

func NormalizeAutonomyFailureRecord(record AutonomyFailureRecord) AutonomyFailureRecord {
	record.FailureID = strings.TrimSpace(record.FailureID)
	record.WakeCycleID = strings.TrimSpace(record.WakeCycleID)
	record.StandingDirectiveID = strings.TrimSpace(record.StandingDirectiveID)
	record.BudgetID = strings.TrimSpace(record.BudgetID)
	record.FailureKind = AutonomyFailureKind(strings.TrimSpace(string(record.FailureKind)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.OccurredAt = record.OccurredAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func NormalizeAutonomyPauseRecord(record AutonomyPauseRecord) AutonomyPauseRecord {
	record.PauseID = strings.TrimSpace(record.PauseID)
	record.BudgetID = strings.TrimSpace(record.BudgetID)
	record.StandingDirectiveID = strings.TrimSpace(record.StandingDirectiveID)
	record.PauseKind = AutonomyPauseKind(strings.TrimSpace(string(record.PauseKind)))
	record.State = AutonomyPauseState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.FailureIDs = normalizeAutonomyStrings(record.FailureIDs)
	record.PausedAt = record.PausedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateAutonomyFailureRecord(record AutonomyFailureRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store autonomy failure record_version must be positive")
	}
	if err := validateAutonomyIdentifierField("mission store autonomy failure", "failure_id", record.FailureID); err != nil {
		return err
	}
	if err := ValidateWakeCycleRef(WakeCycleRef{WakeCycleID: record.WakeCycleID}); err != nil {
		return fmt.Errorf("mission store autonomy failure wake_cycle_id %q: %w", record.WakeCycleID, err)
	}
	if record.StandingDirectiveID != "" {
		if err := ValidateStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: record.StandingDirectiveID}); err != nil {
			return fmt.Errorf("mission store autonomy failure standing_directive_id %q: %w", record.StandingDirectiveID, err)
		}
	}
	if err := ValidateAutonomyBudgetRef(AutonomyBudgetRef{BudgetID: record.BudgetID}); err != nil {
		return fmt.Errorf("mission store autonomy failure budget_id %q: %w", record.BudgetID, err)
	}
	if record.FailureID != AutonomyFailureIDFromWakeCycle(record.WakeCycleID) {
		return fmt.Errorf("mission store autonomy failure failure_id %q does not match deterministic failure_id %q", record.FailureID, AutonomyFailureIDFromWakeCycle(record.WakeCycleID))
	}
	if !isValidAutonomyFailureKind(record.FailureKind) {
		return fmt.Errorf("mission store autonomy failure failure_kind %q is invalid", record.FailureKind)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store autonomy failure reason is required")
	}
	if record.OccurredAt.IsZero() {
		return fmt.Errorf("mission store autonomy failure occurred_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store autonomy failure created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store autonomy failure created_by is required")
	}
	return nil
}

func ValidateAutonomyPauseRecord(record AutonomyPauseRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store autonomy pause record_version must be positive")
	}
	if err := validateAutonomyIdentifierField("mission store autonomy pause", "pause_id", record.PauseID); err != nil {
		return err
	}
	if err := ValidateAutonomyBudgetRef(AutonomyBudgetRef{BudgetID: record.BudgetID}); err != nil {
		return fmt.Errorf("mission store autonomy pause budget_id %q: %w", record.BudgetID, err)
	}
	if record.StandingDirectiveID != "" {
		if err := ValidateStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: record.StandingDirectiveID}); err != nil {
			return fmt.Errorf("mission store autonomy pause standing_directive_id %q: %w", record.StandingDirectiveID, err)
		}
	}
	if record.PauseID != AutonomyRepeatedFailurePauseID(record.BudgetID) {
		return fmt.Errorf("mission store autonomy pause pause_id %q does not match deterministic pause_id %q", record.PauseID, AutonomyRepeatedFailurePauseID(record.BudgetID))
	}
	if record.PauseKind != AutonomyPauseKindRepeatedFailure {
		return fmt.Errorf("mission store autonomy pause pause_kind %q is invalid", record.PauseKind)
	}
	if record.State != AutonomyPauseStateActive {
		return fmt.Errorf("mission store autonomy pause state %q is invalid", record.State)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store autonomy pause reason is required")
	}
	if !containsAutonomyReason([]string{record.Reason}, string(RejectionCodeV4RepeatedFailurePause)) {
		return fmt.Errorf("mission store autonomy pause reason must include %s", RejectionCodeV4RepeatedFailurePause)
	}
	if len(record.FailureIDs) == 0 {
		return fmt.Errorf("mission store autonomy pause failure_ids are required")
	}
	if record.PausedAt.IsZero() {
		return fmt.Errorf("mission store autonomy pause paused_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store autonomy pause created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store autonomy pause created_by is required")
	}
	return nil
}

func StoreAutonomyFailureRecord(root string, record AutonomyFailureRecord) (AutonomyFailureRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyFailureRecord{}, false, err
	}
	record = NormalizeAutonomyFailureRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateAutonomyFailureRecord(record); err != nil {
		return AutonomyFailureRecord{}, false, err
	}
	if err := validateAutonomyFailureLinkage(root, record); err != nil {
		return AutonomyFailureRecord{}, false, err
	}
	path := StoreAutonomyFailurePath(root, record.FailureID)
	if existing, err := loadAutonomyFailureRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return AutonomyFailureRecord{}, false, fmt.Errorf("mission store autonomy failure %q already exists", record.FailureID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return AutonomyFailureRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return AutonomyFailureRecord{}, false, err
	}
	return record, true, nil
}

func LoadAutonomyFailureRecord(root, failureID string) (AutonomyFailureRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyFailureRecord{}, err
	}
	failureID = strings.TrimSpace(failureID)
	if err := validateAutonomyIdentifierField("autonomy failure ref", "failure_id", failureID); err != nil {
		return AutonomyFailureRecord{}, err
	}
	record, err := loadAutonomyFailureRecordFile(root, StoreAutonomyFailurePath(root, failureID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AutonomyFailureRecord{}, ErrAutonomyFailureRecordNotFound
		}
		return AutonomyFailureRecord{}, err
	}
	return record, nil
}

func ListAutonomyFailureRecords(root string) ([]AutonomyFailureRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreAutonomyFailuresDir(root), func(path string) (AutonomyFailureRecord, error) {
		return loadAutonomyFailureRecordFile(root, path)
	})
}

func StoreAutonomyPauseRecord(root string, record AutonomyPauseRecord) (AutonomyPauseRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyPauseRecord{}, false, err
	}
	record = NormalizeAutonomyPauseRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateAutonomyPauseRecord(record); err != nil {
		return AutonomyPauseRecord{}, false, err
	}
	if err := validateAutonomyPauseLinkage(root, record); err != nil {
		return AutonomyPauseRecord{}, false, err
	}
	path := StoreAutonomyPausePath(root, record.PauseID)
	if existing, err := loadAutonomyPauseRecordFile(root, path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return AutonomyPauseRecord{}, false, fmt.Errorf("mission store autonomy pause %q already exists", record.PauseID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return AutonomyPauseRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return AutonomyPauseRecord{}, false, err
	}
	return record, true, nil
}

func LoadAutonomyPauseRecord(root, pauseID string) (AutonomyPauseRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return AutonomyPauseRecord{}, err
	}
	pauseID = strings.TrimSpace(pauseID)
	if err := validateAutonomyIdentifierField("autonomy pause ref", "pause_id", pauseID); err != nil {
		return AutonomyPauseRecord{}, err
	}
	record, err := loadAutonomyPauseRecordFile(root, StoreAutonomyPausePath(root, pauseID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AutonomyPauseRecord{}, ErrAutonomyPauseRecordNotFound
		}
		return AutonomyPauseRecord{}, err
	}
	return record, nil
}

func ListAutonomyPauseRecords(root string) ([]AutonomyPauseRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreAutonomyPausesDir(root), func(path string) (AutonomyPauseRecord, error) {
		return loadAutonomyPauseRecordFile(root, path)
	})
}

func RecordAutonomyFailureFromWakeCycle(root, wakeCycleID string, failureKind AutonomyFailureKind, reason, createdBy string, occurredAt time.Time) (AutonomyFailureRecord, *AutonomyPauseRecord, bool, error) {
	wake, err := LoadWakeCycleRecord(root, wakeCycleID)
	if err != nil {
		return AutonomyFailureRecord{}, nil, false, err
	}
	if wake.BudgetRef == "" {
		return AutonomyFailureRecord{}, nil, false, fmt.Errorf("mission store autonomy failure wake_cycle_id %q has no budget_ref", wake.WakeCycleID)
	}
	record := AutonomyFailureRecord{
		FailureID:           AutonomyFailureIDFromWakeCycle(wake.WakeCycleID),
		WakeCycleID:         wake.WakeCycleID,
		StandingDirectiveID: wake.SelectedDirectiveID,
		BudgetID:            wake.BudgetRef,
		FailureKind:         failureKind,
		Reason:              reason,
		OccurredAt:          occurredAt,
		CreatedAt:           occurredAt,
		CreatedBy:           createdBy,
	}
	stored, _, err := StoreAutonomyFailureRecord(root, record)
	if err != nil {
		return AutonomyFailureRecord{}, nil, false, err
	}
	pause, pauseChanged, err := CreateRepeatedFailurePauseIfNeeded(root, stored.BudgetID, stored.StandingDirectiveID, createdBy, occurredAt)
	if err != nil {
		return AutonomyFailureRecord{}, nil, false, err
	}
	if pause == nil {
		return stored, nil, false, nil
	}
	return stored, pause, pauseChanged, nil
}

func CreateRepeatedFailurePauseIfNeeded(root, budgetID, standingDirectiveID, createdBy string, pausedAt time.Time) (*AutonomyPauseRecord, bool, error) {
	budget, err := LoadAutonomyBudgetRecord(root, budgetID)
	if err != nil {
		return nil, false, err
	}
	failures, err := consecutiveAutonomyFailuresForBudget(root, budget.BudgetID)
	if err != nil {
		return nil, false, err
	}
	if len(failures) < budget.MaxFailedAttemptsBeforePause {
		return nil, false, nil
	}
	selected := failures[:budget.MaxFailedAttemptsBeforePause]
	failureIDs := make([]string, 0, len(selected))
	for _, failure := range selected {
		failureIDs = append(failureIDs, failure.FailureID)
	}
	record := AutonomyPauseRecord{
		PauseID:             AutonomyRepeatedFailurePauseID(budget.BudgetID),
		BudgetID:            budget.BudgetID,
		StandingDirectiveID: strings.TrimSpace(standingDirectiveID),
		PauseKind:           AutonomyPauseKindRepeatedFailure,
		State:               AutonomyPauseStateActive,
		Reason:              string(RejectionCodeV4RepeatedFailurePause) + ": consecutive autonomous wake-cycle failures reached budget threshold",
		FailureIDs:          failureIDs,
		PausedAt:            pausedAt,
		CreatedAt:           pausedAt,
		CreatedBy:           createdBy,
	}
	stored, changed, err := StoreAutonomyPauseRecord(root, record)
	if err != nil {
		return nil, false, err
	}
	return &stored, changed, nil
}

func LoadActiveRepeatedFailurePauseForBudget(root, budgetID string) (AutonomyPauseRecord, bool, error) {
	pause, err := LoadAutonomyPauseRecord(root, AutonomyRepeatedFailurePauseID(budgetID))
	if err != nil {
		if errors.Is(err, ErrAutonomyPauseRecordNotFound) {
			return AutonomyPauseRecord{}, false, nil
		}
		return AutonomyPauseRecord{}, false, err
	}
	return pause, pause.State == AutonomyPauseStateActive && pause.PauseKind == AutonomyPauseKindRepeatedFailure, nil
}

func loadAutonomyFailureRecordFile(root, path string) (AutonomyFailureRecord, error) {
	var record AutonomyFailureRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return AutonomyFailureRecord{}, err
	}
	record = NormalizeAutonomyFailureRecord(record)
	if err := ValidateAutonomyFailureRecord(record); err != nil {
		return AutonomyFailureRecord{}, err
	}
	if err := validateAutonomyFailureLinkage(root, record); err != nil {
		return AutonomyFailureRecord{}, err
	}
	return record, nil
}

func loadAutonomyPauseRecordFile(root, path string) (AutonomyPauseRecord, error) {
	var record AutonomyPauseRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return AutonomyPauseRecord{}, err
	}
	record = NormalizeAutonomyPauseRecord(record)
	if err := ValidateAutonomyPauseRecord(record); err != nil {
		return AutonomyPauseRecord{}, err
	}
	if err := validateAutonomyPauseLinkage(root, record); err != nil {
		return AutonomyPauseRecord{}, err
	}
	return record, nil
}

func validateAutonomyFailureLinkage(root string, record AutonomyFailureRecord) error {
	wake, err := LoadWakeCycleRecord(root, record.WakeCycleID)
	if err != nil {
		return err
	}
	if wake.BudgetRef != record.BudgetID {
		return fmt.Errorf("mission store autonomy failure budget_id %q does not match wake cycle budget_ref %q", record.BudgetID, wake.BudgetRef)
	}
	if wake.SelectedDirectiveID != record.StandingDirectiveID {
		return fmt.Errorf("mission store autonomy failure standing_directive_id %q does not match wake cycle selected_directive_id %q", record.StandingDirectiveID, wake.SelectedDirectiveID)
	}
	if _, err := LoadAutonomyBudgetRecord(root, record.BudgetID); err != nil {
		return err
	}
	return nil
}

func validateAutonomyPauseLinkage(root string, record AutonomyPauseRecord) error {
	if _, err := LoadAutonomyBudgetRecord(root, record.BudgetID); err != nil {
		return err
	}
	for _, failureID := range record.FailureIDs {
		failure, err := LoadAutonomyFailureRecord(root, failureID)
		if err != nil {
			return err
		}
		if failure.BudgetID != record.BudgetID {
			return fmt.Errorf("mission store autonomy pause failure_id %q budget_id %q does not match pause budget_id %q", failureID, failure.BudgetID, record.BudgetID)
		}
	}
	return nil
}

func consecutiveAutonomyFailuresForBudget(root, budgetID string) ([]AutonomyFailureRecord, error) {
	wakeCycles, err := ListWakeCycleRecords(root)
	if err != nil {
		return nil, err
	}
	failures, err := ListAutonomyFailureRecords(root)
	if err != nil {
		return nil, err
	}
	failuresByWake := make(map[string]AutonomyFailureRecord, len(failures))
	for _, failure := range failures {
		if failure.BudgetID == budgetID {
			failuresByWake[failure.WakeCycleID] = failure
		}
	}
	cycles := make([]WakeCycleRecord, 0, len(wakeCycles))
	for _, cycle := range wakeCycles {
		if cycle.BudgetRef == budgetID {
			cycles = append(cycles, cycle)
		}
	}
	sort.Slice(cycles, func(i, j int) bool {
		if cycles[i].StartedAt.Equal(cycles[j].StartedAt) {
			return cycles[i].WakeCycleID > cycles[j].WakeCycleID
		}
		return cycles[i].StartedAt.After(cycles[j].StartedAt)
	})
	consecutive := make([]AutonomyFailureRecord, 0)
	for _, cycle := range cycles {
		failure, ok := failuresByWake[cycle.WakeCycleID]
		if !ok {
			break
		}
		consecutive = append(consecutive, failure)
	}
	return consecutive, nil
}

func isValidAutonomyFailureKind(kind AutonomyFailureKind) bool {
	switch kind {
	case AutonomyFailureKindWakeCycle, AutonomyFailureKindEval, AutonomyFailureKindRuntime:
		return true
	default:
		return false
	}
}
