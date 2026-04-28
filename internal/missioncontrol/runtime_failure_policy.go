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

const DefaultRepeatedActivePackFailureThreshold = 3

type RuntimeFailureKind string

const (
	RuntimeFailureKindSmoke   RuntimeFailureKind = "smoke"
	RuntimeFailureKindRuntime RuntimeFailureKind = "runtime"
)

type RuntimeFailureEventRecord struct {
	RecordVersion int                `json:"record_version"`
	EventID       string             `json:"event_id"`
	PackID        string             `json:"pack_id"`
	HotUpdateID   string             `json:"hot_update_id,omitempty"`
	FailureKind   RuntimeFailureKind `json:"failure_kind"`
	FailureReason string             `json:"failure_reason"`
	ObservedAt    time.Time          `json:"observed_at"`
	CreatedAt     time.Time          `json:"created_at"`
	CreatedBy     string             `json:"created_by"`
}

type RuntimePackQuarantineState string

const (
	RuntimePackQuarantineStateQuarantined RuntimePackQuarantineState = "quarantined"
)

type RuntimePackQuarantineRecord struct {
	RecordVersion   int                        `json:"record_version"`
	QuarantineID    string                     `json:"quarantine_id"`
	PackID          string                     `json:"pack_id"`
	FailureEventIDs []string                   `json:"failure_event_ids"`
	RollbackID      string                     `json:"rollback_id,omitempty"`
	RollbackApplyID string                     `json:"rollback_apply_id,omitempty"`
	State           RuntimePackQuarantineState `json:"state"`
	Reason          string                     `json:"reason"`
	CreatedAt       time.Time                  `json:"created_at"`
	CreatedBy       string                     `json:"created_by"`
}

type RepeatedFailureTerminalBlockerRecord struct {
	RecordVersion   int       `json:"record_version"`
	BlockerID       string    `json:"blocker_id"`
	PackID          string    `json:"pack_id"`
	FailureEventIDs []string  `json:"failure_event_ids"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
}

type RepeatedActivePackFailureAction string

const (
	RepeatedActivePackFailureActionNone              RepeatedActivePackFailureAction = "none"
	RepeatedActivePackFailureActionRollbackTriggered RepeatedActivePackFailureAction = "rollback_triggered"
	RepeatedActivePackFailureActionTerminalBlocked   RepeatedActivePackFailureAction = "terminal_blocked"
)

type RepeatedActivePackFailurePolicySpec struct {
	AssessmentID string    `json:"assessment_id"`
	Threshold    int       `json:"threshold,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
}

type RepeatedActivePackFailurePolicyResult struct {
	Action                  RepeatedActivePackFailureAction `json:"action"`
	ActivePackID            string                          `json:"active_pack_id"`
	LastKnownGoodPackID     string                          `json:"last_known_good_pack_id,omitempty"`
	ConsecutiveFailureCount int                             `json:"consecutive_failure_count"`
	FailureEventIDs         []string                        `json:"failure_event_ids,omitempty"`
	RollbackID              string                          `json:"rollback_id,omitempty"`
	RollbackApplyID         string                          `json:"rollback_apply_id,omitempty"`
	QuarantineID            string                          `json:"quarantine_id,omitempty"`
	TerminalBlockerID       string                          `json:"terminal_blocker_id,omitempty"`
}

var (
	ErrRuntimeFailureEventRecordNotFound            = errors.New("mission store runtime failure event record not found")
	ErrRuntimePackQuarantineRecordNotFound          = errors.New("mission store runtime pack quarantine record not found")
	ErrRepeatedFailureTerminalBlockerRecordNotFound = errors.New("mission store repeated failure terminal blocker record not found")
)

func StoreRuntimeFailureEventsDir(root string) string {
	return filepath.Join(root, "runtime_packs", "runtime_failure_events")
}

func StoreRuntimeFailureEventPath(root, eventID string) string {
	return filepath.Join(StoreRuntimeFailureEventsDir(root), strings.TrimSpace(eventID)+".json")
}

func StoreRuntimePackQuarantinesDir(root string) string {
	return filepath.Join(root, "runtime_packs", "quarantined")
}

func StoreRuntimePackQuarantinePath(root, quarantineID string) string {
	return filepath.Join(StoreRuntimePackQuarantinesDir(root), strings.TrimSpace(quarantineID)+".json")
}

func StoreRepeatedFailureTerminalBlockersDir(root string) string {
	return filepath.Join(root, "runtime_packs", "repeated_failure_blockers")
}

func StoreRepeatedFailureTerminalBlockerPath(root, blockerID string) string {
	return filepath.Join(StoreRepeatedFailureTerminalBlockersDir(root), strings.TrimSpace(blockerID)+".json")
}

func NormalizeRuntimeFailureEventRecord(record RuntimeFailureEventRecord) RuntimeFailureEventRecord {
	record.EventID = strings.TrimSpace(record.EventID)
	record.PackID = strings.TrimSpace(record.PackID)
	record.HotUpdateID = strings.TrimSpace(record.HotUpdateID)
	record.FailureKind = RuntimeFailureKind(strings.TrimSpace(string(record.FailureKind)))
	record.FailureReason = strings.TrimSpace(record.FailureReason)
	record.ObservedAt = record.ObservedAt.UTC()
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func NormalizeRuntimePackQuarantineRecord(record RuntimePackQuarantineRecord) RuntimePackQuarantineRecord {
	record.QuarantineID = strings.TrimSpace(record.QuarantineID)
	record.PackID = strings.TrimSpace(record.PackID)
	record.FailureEventIDs = normalizeRuntimePackStrings(record.FailureEventIDs)
	record.RollbackID = strings.TrimSpace(record.RollbackID)
	record.RollbackApplyID = strings.TrimSpace(record.RollbackApplyID)
	record.State = RuntimePackQuarantineState(strings.TrimSpace(string(record.State)))
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func NormalizeRepeatedFailureTerminalBlockerRecord(record RepeatedFailureTerminalBlockerRecord) RepeatedFailureTerminalBlockerRecord {
	record.BlockerID = strings.TrimSpace(record.BlockerID)
	record.PackID = strings.TrimSpace(record.PackID)
	record.FailureEventIDs = normalizeRuntimePackStrings(record.FailureEventIDs)
	record.Reason = strings.TrimSpace(record.Reason)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func NormalizeRepeatedActivePackFailurePolicySpec(spec RepeatedActivePackFailurePolicySpec) RepeatedActivePackFailurePolicySpec {
	spec.AssessmentID = strings.TrimSpace(spec.AssessmentID)
	if spec.Threshold <= 0 {
		spec.Threshold = DefaultRepeatedActivePackFailureThreshold
	}
	spec.CreatedAt = spec.CreatedAt.UTC()
	spec.CreatedBy = strings.TrimSpace(spec.CreatedBy)
	return spec
}

func ValidateRuntimeFailureEventRecord(record RuntimeFailureEventRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime failure event record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store runtime failure event", "event_id", record.EventID); err != nil {
		return err
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.PackID}); err != nil {
		return fmt.Errorf("mission store runtime failure event pack_id %q: %w", record.PackID, err)
	}
	if record.HotUpdateID != "" {
		if err := ValidateHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: record.HotUpdateID}); err != nil {
			return fmt.Errorf("mission store runtime failure event hot_update_id %q: %w", record.HotUpdateID, err)
		}
	}
	if !isValidRuntimeFailureKind(record.FailureKind) {
		return fmt.Errorf("mission store runtime failure event failure_kind %q is invalid", record.FailureKind)
	}
	if record.FailureReason == "" {
		return fmt.Errorf("mission store runtime failure event failure_reason is required")
	}
	if record.ObservedAt.IsZero() {
		return fmt.Errorf("mission store runtime failure event observed_at is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store runtime failure event created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store runtime failure event created_by is required")
	}
	return nil
}

func ValidateRuntimePackQuarantineRecord(record RuntimePackQuarantineRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store runtime pack quarantine record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store runtime pack quarantine", "quarantine_id", record.QuarantineID); err != nil {
		return err
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.PackID}); err != nil {
		return fmt.Errorf("mission store runtime pack quarantine pack_id %q: %w", record.PackID, err)
	}
	if len(record.FailureEventIDs) == 0 {
		return fmt.Errorf("mission store runtime pack quarantine failure_event_ids are required")
	}
	for _, eventID := range record.FailureEventIDs {
		if err := validateRuntimePackIDField("mission store runtime pack quarantine", "failure_event_ids", eventID); err != nil {
			return err
		}
	}
	if record.RollbackID != "" {
		if err := ValidateRollbackRef(RollbackRef{RollbackID: record.RollbackID}); err != nil {
			return fmt.Errorf("mission store runtime pack quarantine rollback_id %q: %w", record.RollbackID, err)
		}
	}
	if record.RollbackApplyID != "" {
		if err := ValidateRollbackApplyRef(RollbackApplyRef{ApplyID: record.RollbackApplyID}); err != nil {
			return fmt.Errorf("mission store runtime pack quarantine rollback_apply_id %q: %w", record.RollbackApplyID, err)
		}
	}
	if record.State != RuntimePackQuarantineStateQuarantined {
		return fmt.Errorf("mission store runtime pack quarantine state must be %q", RuntimePackQuarantineStateQuarantined)
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store runtime pack quarantine reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store runtime pack quarantine created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store runtime pack quarantine created_by is required")
	}
	return nil
}

func ValidateRepeatedFailureTerminalBlockerRecord(record RepeatedFailureTerminalBlockerRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store repeated failure terminal blocker record_version must be positive")
	}
	if err := validateRuntimePackIDField("mission store repeated failure terminal blocker", "blocker_id", record.BlockerID); err != nil {
		return err
	}
	if err := ValidateRuntimePackRef(RuntimePackRef{PackID: record.PackID}); err != nil {
		return fmt.Errorf("mission store repeated failure terminal blocker pack_id %q: %w", record.PackID, err)
	}
	if len(record.FailureEventIDs) == 0 {
		return fmt.Errorf("mission store repeated failure terminal blocker failure_event_ids are required")
	}
	for _, eventID := range record.FailureEventIDs {
		if err := validateRuntimePackIDField("mission store repeated failure terminal blocker", "failure_event_ids", eventID); err != nil {
			return err
		}
	}
	if record.Reason == "" {
		return fmt.Errorf("mission store repeated failure terminal blocker reason is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store repeated failure terminal blocker created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store repeated failure terminal blocker created_by is required")
	}
	return nil
}

func ValidateRepeatedActivePackFailurePolicySpec(spec RepeatedActivePackFailurePolicySpec) error {
	if err := validateRuntimePackIDField("mission store repeated active-pack failure policy", "assessment_id", spec.AssessmentID); err != nil {
		return err
	}
	if spec.Threshold <= 0 {
		return fmt.Errorf("mission store repeated active-pack failure policy threshold must be positive")
	}
	if spec.CreatedAt.IsZero() {
		return fmt.Errorf("mission store repeated active-pack failure policy created_at is required")
	}
	if spec.CreatedBy == "" {
		return fmt.Errorf("mission store repeated active-pack failure policy created_by is required")
	}
	return nil
}

func StoreRuntimeFailureEventRecord(root string, record RuntimeFailureEventRecord) (RuntimeFailureEventRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeFailureEventRecord{}, false, err
	}
	record = NormalizeRuntimeFailureEventRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRuntimeFailureEventRecord(record); err != nil {
		return RuntimeFailureEventRecord{}, false, err
	}
	if err := validateRuntimeFailureEventLinkage(root, record); err != nil {
		return RuntimeFailureEventRecord{}, false, err
	}

	path := StoreRuntimeFailureEventPath(root, record.EventID)
	existing, err := loadRuntimeFailureEventRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return RuntimeFailureEventRecord{}, false, fmt.Errorf("mission store runtime failure event %q already exists", record.EventID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimeFailureEventRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return RuntimeFailureEventRecord{}, false, err
	}
	stored, err := LoadRuntimeFailureEventRecord(root, record.EventID)
	if err != nil {
		return RuntimeFailureEventRecord{}, false, err
	}
	return stored, true, nil
}

func LoadRuntimeFailureEventRecord(root, eventID string) (RuntimeFailureEventRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimeFailureEventRecord{}, err
	}
	normalizedEventID := strings.TrimSpace(eventID)
	if err := validateRuntimePackIDField("mission store runtime failure event", "event_id", normalizedEventID); err != nil {
		return RuntimeFailureEventRecord{}, err
	}
	record, err := loadRuntimeFailureEventRecordFile(root, StoreRuntimeFailureEventPath(root, normalizedEventID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeFailureEventRecord{}, ErrRuntimeFailureEventRecordNotFound
		}
		return RuntimeFailureEventRecord{}, err
	}
	return record, nil
}

func ListRuntimeFailureEventRecords(root string) ([]RuntimeFailureEventRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	records, err := listStoreJSONRecords(StoreRuntimeFailureEventsDir(root), func(path string) (RuntimeFailureEventRecord, error) {
		return loadRuntimeFailureEventRecordFile(root, path)
	})
	if err != nil {
		return nil, err
	}
	sortRuntimeFailureEvents(records)
	return records, nil
}

func StoreRuntimePackQuarantineRecord(root string, record RuntimePackQuarantineRecord) (RuntimePackQuarantineRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimePackQuarantineRecord{}, false, err
	}
	record = NormalizeRuntimePackQuarantineRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRuntimePackQuarantineRecord(record); err != nil {
		return RuntimePackQuarantineRecord{}, false, err
	}
	if err := validateRuntimePackQuarantineLinkage(root, record); err != nil {
		return RuntimePackQuarantineRecord{}, false, err
	}

	path := StoreRuntimePackQuarantinePath(root, record.QuarantineID)
	existing, err := loadRuntimePackQuarantineRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return RuntimePackQuarantineRecord{}, false, fmt.Errorf("mission store runtime pack quarantine %q already exists", record.QuarantineID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RuntimePackQuarantineRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return RuntimePackQuarantineRecord{}, false, err
	}
	stored, err := LoadRuntimePackQuarantineRecord(root, record.QuarantineID)
	if err != nil {
		return RuntimePackQuarantineRecord{}, false, err
	}
	return stored, true, nil
}

func LoadRuntimePackQuarantineRecord(root, quarantineID string) (RuntimePackQuarantineRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RuntimePackQuarantineRecord{}, err
	}
	normalizedQuarantineID := strings.TrimSpace(quarantineID)
	if err := validateRuntimePackIDField("mission store runtime pack quarantine", "quarantine_id", normalizedQuarantineID); err != nil {
		return RuntimePackQuarantineRecord{}, err
	}
	record, err := loadRuntimePackQuarantineRecordFile(root, StoreRuntimePackQuarantinePath(root, normalizedQuarantineID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimePackQuarantineRecord{}, ErrRuntimePackQuarantineRecordNotFound
		}
		return RuntimePackQuarantineRecord{}, err
	}
	return record, nil
}

func ListRuntimePackQuarantineRecords(root string) ([]RuntimePackQuarantineRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRuntimePackQuarantinesDir(root), func(path string) (RuntimePackQuarantineRecord, error) {
		return loadRuntimePackQuarantineRecordFile(root, path)
	})
}

func StoreRepeatedFailureTerminalBlockerRecord(root string, record RepeatedFailureTerminalBlockerRecord) (RepeatedFailureTerminalBlockerRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}
	record = NormalizeRepeatedFailureTerminalBlockerRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateRepeatedFailureTerminalBlockerRecord(record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}
	if err := validateRepeatedFailureTerminalBlockerLinkage(root, record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}

	path := StoreRepeatedFailureTerminalBlockerPath(root, record.BlockerID)
	existing, err := loadRepeatedFailureTerminalBlockerRecordFile(root, path)
	if err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return RepeatedFailureTerminalBlockerRecord{}, false, fmt.Errorf("mission store repeated failure terminal blocker %q already exists", record.BlockerID)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}
	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}
	stored, err := LoadRepeatedFailureTerminalBlockerRecord(root, record.BlockerID)
	if err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, false, err
	}
	return stored, true, nil
}

func LoadRepeatedFailureTerminalBlockerRecord(root, blockerID string) (RepeatedFailureTerminalBlockerRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	normalizedBlockerID := strings.TrimSpace(blockerID)
	if err := validateRuntimePackIDField("mission store repeated failure terminal blocker", "blocker_id", normalizedBlockerID); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	record, err := loadRepeatedFailureTerminalBlockerRecordFile(root, StoreRepeatedFailureTerminalBlockerPath(root, normalizedBlockerID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RepeatedFailureTerminalBlockerRecord{}, ErrRepeatedFailureTerminalBlockerRecordNotFound
		}
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	return record, nil
}

func ListRepeatedFailureTerminalBlockerRecords(root string) ([]RepeatedFailureTerminalBlockerRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreRepeatedFailureTerminalBlockersDir(root), func(path string) (RepeatedFailureTerminalBlockerRecord, error) {
		return loadRepeatedFailureTerminalBlockerRecordFile(root, path)
	})
}

func AssessRepeatedActivePackFailures(root string, spec RepeatedActivePackFailurePolicySpec) (RepeatedActivePackFailurePolicyResult, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	spec = NormalizeRepeatedActivePackFailurePolicySpec(spec)
	if err := ValidateRepeatedActivePackFailurePolicySpec(spec); err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	failureEvents, err := latestConsecutiveActivePackFailureEvents(root, activePointer.ActivePackID, spec.Threshold)
	if err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	result := RepeatedActivePackFailurePolicyResult{
		Action:                  RepeatedActivePackFailureActionNone,
		ActivePackID:            activePointer.ActivePackID,
		ConsecutiveFailureCount: len(failureEvents),
		FailureEventIDs:         runtimeFailureEventIDs(failureEvents),
	}
	if len(failureEvents) < spec.Threshold {
		return result, nil
	}

	lkg, err := LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil || lkg.PackID == activePointer.ActivePackID {
		if err != nil && !errors.Is(err, ErrLastKnownGoodRuntimePackPointerNotFound) {
			return RepeatedActivePackFailurePolicyResult{}, err
		}
		blocker, _, err := StoreRepeatedFailureTerminalBlockerRecord(root, RepeatedFailureTerminalBlockerRecord{
			BlockerID:       repeatedFailureTerminalBlockerID(spec.AssessmentID),
			PackID:          activePointer.ActivePackID,
			FailureEventIDs: result.FailureEventIDs,
			Reason:          repeatedFailureTerminalBlockerReason(activePointer.ActivePackID, spec.Threshold),
			CreatedAt:       spec.CreatedAt,
			CreatedBy:       spec.CreatedBy,
		})
		if err != nil {
			return RepeatedActivePackFailurePolicyResult{}, err
		}
		result.Action = RepeatedActivePackFailureActionTerminalBlocked
		result.TerminalBlockerID = blocker.BlockerID
		return result, nil
	}

	latestFailure := failureEvents[len(failureEvents)-1]
	result.LastKnownGoodPackID = lkg.PackID
	result.RollbackID = repeatedFailureRollbackID(spec.AssessmentID)
	result.RollbackApplyID = repeatedFailureRollbackApplyID(spec.AssessmentID)
	result.QuarantineID = repeatedFailureQuarantineID(spec.AssessmentID)

	rollback := RollbackRecord{
		RollbackID:          result.RollbackID,
		HotUpdateID:         latestFailure.HotUpdateID,
		FromPackID:          activePointer.ActivePackID,
		TargetPackID:        lkg.PackID,
		LastKnownGoodPackID: lkg.PackID,
		Reason:              repeatedFailureRollbackReason(activePointer.ActivePackID, spec.Threshold),
		Notes:               "automatic local repeated-failure policy",
		RollbackAt:          spec.CreatedAt,
		CreatedAt:           spec.CreatedAt,
		CreatedBy:           spec.CreatedBy,
	}
	if err := StoreRollbackRecord(root, rollback); err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	apply, _, err := EnsureRollbackApplyRecordFromRollback(root, result.RollbackApplyID, result.RollbackID, spec.CreatedBy, spec.CreatedAt)
	if err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	if apply.Phase == RollbackApplyPhaseRecorded {
		apply, _, err = AdvanceRollbackApplyPhase(root, apply.ApplyID, RollbackApplyPhaseValidated, spec.CreatedBy, spec.CreatedAt)
		if err != nil {
			return RepeatedActivePackFailurePolicyResult{}, err
		}
	}
	if apply.Phase == RollbackApplyPhaseValidated {
		apply, _, err = AdvanceRollbackApplyPhase(root, apply.ApplyID, RollbackApplyPhaseReadyToApply, spec.CreatedBy, spec.CreatedAt)
		if err != nil {
			return RepeatedActivePackFailurePolicyResult{}, err
		}
	}
	if _, _, err := StoreRuntimePackQuarantineRecord(root, RuntimePackQuarantineRecord{
		QuarantineID:    result.QuarantineID,
		PackID:          activePointer.ActivePackID,
		FailureEventIDs: result.FailureEventIDs,
		RollbackID:      result.RollbackID,
		RollbackApplyID: result.RollbackApplyID,
		State:           RuntimePackQuarantineStateQuarantined,
		Reason:          repeatedFailureQuarantineReason(activePointer.ActivePackID, spec.Threshold),
		CreatedAt:       spec.CreatedAt,
		CreatedBy:       spec.CreatedBy,
	}); err != nil {
		return RepeatedActivePackFailurePolicyResult{}, err
	}
	if apply.Phase == RollbackApplyPhaseReadyToApply {
		if _, _, err := ExecuteRollbackApplyPointerSwitch(root, apply.ApplyID, spec.CreatedBy, spec.CreatedAt); err != nil {
			return RepeatedActivePackFailurePolicyResult{}, err
		}
	}
	result.Action = RepeatedActivePackFailureActionRollbackTriggered
	return result, nil
}

func loadRuntimeFailureEventRecordFile(root, path string) (RuntimeFailureEventRecord, error) {
	var record RuntimeFailureEventRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimeFailureEventRecord{}, err
	}
	record = NormalizeRuntimeFailureEventRecord(record)
	if err := ValidateRuntimeFailureEventRecord(record); err != nil {
		return RuntimeFailureEventRecord{}, err
	}
	if err := validateRuntimeFailureEventLinkage(root, record); err != nil {
		return RuntimeFailureEventRecord{}, err
	}
	return record, nil
}

func loadRuntimePackQuarantineRecordFile(root, path string) (RuntimePackQuarantineRecord, error) {
	var record RuntimePackQuarantineRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RuntimePackQuarantineRecord{}, err
	}
	record = NormalizeRuntimePackQuarantineRecord(record)
	if err := ValidateRuntimePackQuarantineRecord(record); err != nil {
		return RuntimePackQuarantineRecord{}, err
	}
	if err := validateRuntimePackQuarantineLinkage(root, record); err != nil {
		return RuntimePackQuarantineRecord{}, err
	}
	return record, nil
}

func loadRepeatedFailureTerminalBlockerRecordFile(root, path string) (RepeatedFailureTerminalBlockerRecord, error) {
	var record RepeatedFailureTerminalBlockerRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	record = NormalizeRepeatedFailureTerminalBlockerRecord(record)
	if err := ValidateRepeatedFailureTerminalBlockerRecord(record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	if err := validateRepeatedFailureTerminalBlockerLinkage(root, record); err != nil {
		return RepeatedFailureTerminalBlockerRecord{}, err
	}
	return record, nil
}

func validateRuntimeFailureEventLinkage(root string, record RuntimeFailureEventRecord) error {
	if _, err := LoadRuntimePackRecord(root, record.PackID); err != nil {
		return fmt.Errorf("mission store runtime failure event pack_id %q: %w", record.PackID, err)
	}
	if record.HotUpdateID != "" {
		gate, err := LoadHotUpdateGateRecord(root, record.HotUpdateID)
		if err != nil {
			return fmt.Errorf("mission store runtime failure event hot_update_id %q: %w", record.HotUpdateID, err)
		}
		if gate.CandidatePackID != record.PackID {
			return fmt.Errorf("mission store runtime failure event hot_update_id %q candidate_pack_id %q does not match pack_id %q", record.HotUpdateID, gate.CandidatePackID, record.PackID)
		}
	}
	return nil
}

func validateRuntimePackQuarantineLinkage(root string, record RuntimePackQuarantineRecord) error {
	if _, err := LoadRuntimePackRecord(root, record.PackID); err != nil {
		return fmt.Errorf("mission store runtime pack quarantine pack_id %q: %w", record.PackID, err)
	}
	for _, eventID := range record.FailureEventIDs {
		event, err := LoadRuntimeFailureEventRecord(root, eventID)
		if err != nil {
			return fmt.Errorf("mission store runtime pack quarantine failure_event_id %q: %w", eventID, err)
		}
		if event.PackID != record.PackID {
			return fmt.Errorf("mission store runtime pack quarantine failure_event_id %q pack_id %q does not match quarantine pack_id %q", eventID, event.PackID, record.PackID)
		}
	}
	if record.RollbackID != "" {
		rollback, err := LoadRollbackRecord(root, record.RollbackID)
		if err != nil {
			return fmt.Errorf("mission store runtime pack quarantine rollback_id %q: %w", record.RollbackID, err)
		}
		if rollback.FromPackID != record.PackID {
			return fmt.Errorf("mission store runtime pack quarantine rollback_id %q from_pack_id %q does not match pack_id %q", record.RollbackID, rollback.FromPackID, record.PackID)
		}
	}
	if record.RollbackApplyID != "" {
		if _, err := LoadRollbackApplyRecord(root, record.RollbackApplyID); err != nil {
			return fmt.Errorf("mission store runtime pack quarantine rollback_apply_id %q: %w", record.RollbackApplyID, err)
		}
	}
	return nil
}

func validateRepeatedFailureTerminalBlockerLinkage(root string, record RepeatedFailureTerminalBlockerRecord) error {
	if _, err := LoadRuntimePackRecord(root, record.PackID); err != nil {
		return fmt.Errorf("mission store repeated failure terminal blocker pack_id %q: %w", record.PackID, err)
	}
	for _, eventID := range record.FailureEventIDs {
		event, err := LoadRuntimeFailureEventRecord(root, eventID)
		if err != nil {
			return fmt.Errorf("mission store repeated failure terminal blocker failure_event_id %q: %w", eventID, err)
		}
		if event.PackID != record.PackID {
			return fmt.Errorf("mission store repeated failure terminal blocker failure_event_id %q pack_id %q does not match blocker pack_id %q", eventID, event.PackID, record.PackID)
		}
	}
	return nil
}

func latestConsecutiveActivePackFailureEvents(root string, activePackID string, threshold int) ([]RuntimeFailureEventRecord, error) {
	events, err := ListRuntimeFailureEventRecords(root)
	if err != nil {
		return nil, err
	}
	selected := make([]RuntimeFailureEventRecord, 0, threshold)
	seenActivePackFailure := false
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.PackID != activePackID {
			if seenActivePackFailure {
				break
			}
			continue
		}
		seenActivePackFailure = true
		selected = append(selected, event)
		if len(selected) == threshold {
			break
		}
	}
	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}
	return selected, nil
}

func sortRuntimeFailureEvents(records []RuntimeFailureEventRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].ObservedAt.Equal(records[j].ObservedAt) {
			return records[i].EventID < records[j].EventID
		}
		return records[i].ObservedAt.Before(records[j].ObservedAt)
	})
}

func runtimeFailureEventIDs(events []RuntimeFailureEventRecord) []string {
	if len(events) == 0 {
		return nil
	}
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.EventID)
	}
	return ids
}

func isValidRuntimeFailureKind(kind RuntimeFailureKind) bool {
	switch kind {
	case RuntimeFailureKindSmoke, RuntimeFailureKindRuntime:
		return true
	default:
		return false
	}
}

func repeatedFailureRollbackID(assessmentID string) string {
	return "rollback-" + strings.TrimSpace(assessmentID)
}

func repeatedFailureRollbackApplyID(assessmentID string) string {
	return "apply-" + strings.TrimSpace(assessmentID)
}

func repeatedFailureQuarantineID(assessmentID string) string {
	return "quarantine-" + strings.TrimSpace(assessmentID)
}

func repeatedFailureTerminalBlockerID(assessmentID string) string {
	return "blocker-" + strings.TrimSpace(assessmentID)
}

func repeatedFailureRollbackReason(packID string, threshold int) string {
	return fmt.Sprintf("repeated_failure_threshold: %d consecutive smoke/runtime failures for active pack %s", threshold, strings.TrimSpace(packID))
}

func repeatedFailureQuarantineReason(packID string, threshold int) string {
	return fmt.Sprintf("quarantined_after_repeated_failure: %d consecutive smoke/runtime failures for pack %s", threshold, strings.TrimSpace(packID))
}

func repeatedFailureTerminalBlockerReason(packID string, threshold int) string {
	return fmt.Sprintf("terminal_repeated_failure_no_lkg: %d consecutive smoke/runtime failures for active pack %s and no distinct last-known-good recovery target", threshold, strings.TrimSpace(packID))
}
