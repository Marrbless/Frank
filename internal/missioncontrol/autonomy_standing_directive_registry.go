package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type StandingDirectiveState string

const (
	StandingDirectiveStateActive  StandingDirectiveState = "active"
	StandingDirectiveStateRetired StandingDirectiveState = "retired"
)

type StandingDirectiveOwnerPauseState string

const (
	StandingDirectiveOwnerPauseStateNotPaused StandingDirectiveOwnerPauseState = "not_paused"
	StandingDirectiveOwnerPauseStatePaused    StandingDirectiveOwnerPauseState = "paused"
)

type StandingDirectiveScheduleKind string

const (
	StandingDirectiveScheduleKindInterval StandingDirectiveScheduleKind = "interval"
)

type WakeCycleTrigger string

const (
	WakeCycleTriggerStandingDirective WakeCycleTrigger = "standing_directive"
	WakeCycleTriggerIdleHeartbeat     WakeCycleTrigger = "idle_heartbeat"
)

type WakeCycleDecision string

const (
	WakeCycleDecisionMissionProposed WakeCycleDecision = "mission_proposed"
	WakeCycleDecisionNoEligible      WakeCycleDecision = "no_eligible_autonomous_action"
	WakeCycleDecisionBlocked         WakeCycleDecision = "blocked"
)

type StandingDirectiveRef struct {
	StandingDirectiveID string `json:"standing_directive_id"`
}

type StandingDirectiveSchedule struct {
	Kind            StandingDirectiveScheduleKind `json:"kind"`
	DueAt           time.Time                     `json:"due_at"`
	IntervalSeconds int64                         `json:"interval_seconds"`
}

type StandingDirectiveRecord struct {
	RecordVersion          int                              `json:"record_version"`
	StandingDirectiveID    string                           `json:"standing_directive_id"`
	Objective              string                           `json:"objective"`
	AllowedMissionFamilies []string                         `json:"allowed_mission_families"`
	AllowedExecutionPlanes []string                         `json:"allowed_execution_planes"`
	AllowedExecutionHosts  []string                         `json:"allowed_execution_hosts"`
	AutonomyEnvelopeRef    string                           `json:"autonomy_envelope_ref"`
	BudgetRef              string                           `json:"budget_ref"`
	Schedule               StandingDirectiveSchedule        `json:"schedule"`
	SuccessCriteria        []string                         `json:"success_criteria"`
	StopConditions         []string                         `json:"stop_conditions"`
	OwnerPauseState        StandingDirectiveOwnerPauseState `json:"owner_pause_state"`
	State                  StandingDirectiveState           `json:"state"`
	CreatedAt              time.Time                        `json:"created_at"`
	UpdatedAt              time.Time                        `json:"updated_at"`
	CreatedBy              string                           `json:"created_by"`
}

type WakeCycleRef struct {
	WakeCycleID string `json:"wake_cycle_id"`
}

type AutonomyBudgetDebitRecord struct {
	BudgetID  string `json:"budget_id"`
	DebitKind string `json:"debit_kind"`
	Amount    int64  `json:"amount"`
	Unit      string `json:"unit"`
}

type WakeCycleRecord struct {
	RecordVersion          int                         `json:"record_version"`
	WakeCycleID            string                      `json:"wake_cycle_id"`
	StartedAt              time.Time                   `json:"started_at"`
	CompletedAt            time.Time                   `json:"completed_at"`
	Trigger                WakeCycleTrigger            `json:"trigger"`
	SelectedDirectiveID    string                      `json:"selected_directive_id,omitempty"`
	SelectedJobID          string                      `json:"selected_job_id,omitempty"`
	SelectedMissionFamily  string                      `json:"selected_mission_family,omitempty"`
	SelectedExecutionPlane string                      `json:"selected_execution_plane,omitempty"`
	SelectedExecutionHost  string                      `json:"selected_execution_host,omitempty"`
	Decision               WakeCycleDecision           `json:"decision"`
	BudgetDebits           []AutonomyBudgetDebitRecord `json:"budget_debits"`
	BlockedReasons         []string                    `json:"blocked_reasons"`
	NextWakeAt             time.Time                   `json:"next_wake_at"`
	AutonomyEnvelopeRef    string                      `json:"autonomy_envelope_ref,omitempty"`
	BudgetRef              string                      `json:"budget_ref,omitempty"`
	CreatedAt              time.Time                   `json:"created_at"`
	CreatedBy              string                      `json:"created_by"`
}

var (
	ErrStandingDirectiveRecordNotFound = errors.New("mission store standing directive record not found")
	ErrWakeCycleRecordNotFound         = errors.New("mission store wake cycle record not found")
)

func StoreAutonomyDir(root string) string {
	return filepath.Join(root, "autonomy")
}

func StoreStandingDirectivesDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "standing_directives")
}

func StoreStandingDirectivePath(root, standingDirectiveID string) string {
	return filepath.Join(StoreStandingDirectivesDir(root), strings.TrimSpace(standingDirectiveID)+".json")
}

func StoreWakeCyclesDir(root string) string {
	return filepath.Join(StoreAutonomyDir(root), "wake_cycles")
}

func StoreWakeCyclePath(root, wakeCycleID string) string {
	return filepath.Join(StoreWakeCyclesDir(root), strings.TrimSpace(wakeCycleID)+".json")
}

func WakeCycleIDFromDirectiveStartedAt(standingDirectiveID string, startedAt time.Time) string {
	startedAt = startedAt.UTC()
	startedAtCompact := strings.ReplaceAll(startedAt.Format("20060102T150405.000000000Z"), ".", "")
	return "wake-cycle-" + strings.TrimSpace(standingDirectiveID) + "-" + startedAtCompact
}

func WakeCycleIDFromNoEligibleStartedAt(startedAt time.Time) string {
	startedAt = startedAt.UTC()
	startedAtCompact := strings.ReplaceAll(startedAt.Format("20060102T150405.000000000Z"), ".", "")
	return "wake-cycle-no-eligible-" + startedAtCompact
}

func NormalizeStandingDirectiveRef(ref StandingDirectiveRef) StandingDirectiveRef {
	ref.StandingDirectiveID = strings.TrimSpace(ref.StandingDirectiveID)
	return ref
}

func NormalizeWakeCycleRef(ref WakeCycleRef) WakeCycleRef {
	ref.WakeCycleID = strings.TrimSpace(ref.WakeCycleID)
	return ref
}

func NormalizeStandingDirectiveSchedule(schedule StandingDirectiveSchedule) StandingDirectiveSchedule {
	schedule.Kind = StandingDirectiveScheduleKind(strings.TrimSpace(string(schedule.Kind)))
	schedule.DueAt = schedule.DueAt.UTC()
	return schedule
}

func NormalizeStandingDirectiveRecord(record StandingDirectiveRecord) StandingDirectiveRecord {
	record.StandingDirectiveID = strings.TrimSpace(record.StandingDirectiveID)
	record.Objective = strings.TrimSpace(record.Objective)
	record.AllowedMissionFamilies = normalizeAutonomyStrings(record.AllowedMissionFamilies)
	record.AllowedExecutionPlanes = normalizeAutonomyStrings(record.AllowedExecutionPlanes)
	record.AllowedExecutionHosts = normalizeAutonomyStrings(record.AllowedExecutionHosts)
	record.AutonomyEnvelopeRef = strings.TrimSpace(record.AutonomyEnvelopeRef)
	record.BudgetRef = strings.TrimSpace(record.BudgetRef)
	record.Schedule = NormalizeStandingDirectiveSchedule(record.Schedule)
	record.SuccessCriteria = normalizeAutonomyStrings(record.SuccessCriteria)
	record.StopConditions = normalizeAutonomyStrings(record.StopConditions)
	record.OwnerPauseState = StandingDirectiveOwnerPauseState(strings.TrimSpace(string(record.OwnerPauseState)))
	record.State = StandingDirectiveState(strings.TrimSpace(string(record.State)))
	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func NormalizeAutonomyBudgetDebitRecord(record AutonomyBudgetDebitRecord) AutonomyBudgetDebitRecord {
	record.BudgetID = strings.TrimSpace(record.BudgetID)
	record.DebitKind = strings.TrimSpace(record.DebitKind)
	record.Unit = strings.TrimSpace(record.Unit)
	return record
}

func NormalizeWakeCycleRecord(record WakeCycleRecord) WakeCycleRecord {
	record.WakeCycleID = strings.TrimSpace(record.WakeCycleID)
	record.StartedAt = record.StartedAt.UTC()
	record.CompletedAt = record.CompletedAt.UTC()
	record.Trigger = WakeCycleTrigger(strings.TrimSpace(string(record.Trigger)))
	record.SelectedDirectiveID = strings.TrimSpace(record.SelectedDirectiveID)
	record.SelectedJobID = strings.TrimSpace(record.SelectedJobID)
	record.SelectedMissionFamily = strings.TrimSpace(record.SelectedMissionFamily)
	record.SelectedExecutionPlane = strings.TrimSpace(record.SelectedExecutionPlane)
	record.SelectedExecutionHost = strings.TrimSpace(record.SelectedExecutionHost)
	record.Decision = WakeCycleDecision(strings.TrimSpace(string(record.Decision)))
	if len(record.BudgetDebits) > 0 {
		debits := make([]AutonomyBudgetDebitRecord, 0, len(record.BudgetDebits))
		for _, debit := range record.BudgetDebits {
			debit = NormalizeAutonomyBudgetDebitRecord(debit)
			if debit.BudgetID == "" && debit.DebitKind == "" && debit.Amount == 0 && debit.Unit == "" {
				continue
			}
			debits = append(debits, debit)
		}
		record.BudgetDebits = debits
		if len(record.BudgetDebits) == 0 {
			record.BudgetDebits = nil
		}
	}
	record.BlockedReasons = normalizeAutonomyStrings(record.BlockedReasons)
	record.NextWakeAt = record.NextWakeAt.UTC()
	record.AutonomyEnvelopeRef = strings.TrimSpace(record.AutonomyEnvelopeRef)
	record.BudgetRef = strings.TrimSpace(record.BudgetRef)
	record.CreatedAt = record.CreatedAt.UTC()
	record.CreatedBy = strings.TrimSpace(record.CreatedBy)
	return record
}

func ValidateStandingDirectiveRef(ref StandingDirectiveRef) error {
	return validateAutonomyIdentifierField("standing directive ref", "standing_directive_id", ref.StandingDirectiveID)
}

func ValidateWakeCycleRef(ref WakeCycleRef) error {
	return validateAutonomyIdentifierField("wake cycle ref", "wake_cycle_id", ref.WakeCycleID)
}

func ValidateStandingDirectiveRecord(record StandingDirectiveRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store standing directive record_version must be positive")
	}
	if err := ValidateStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: record.StandingDirectiveID}); err != nil {
		return err
	}
	if record.Objective == "" {
		return fmt.Errorf("mission store standing directive objective is required")
	}
	if err := validateStandingDirectiveMissionFamilies(record.AllowedMissionFamilies); err != nil {
		return err
	}
	if err := validateStandingDirectiveExecutionPlanes(record.AllowedExecutionPlanes); err != nil {
		return err
	}
	if err := validateStandingDirectiveExecutionHosts(record.AllowedExecutionHosts); err != nil {
		return err
	}
	if err := validateAutonomyIdentifierField("mission store standing directive", "autonomy_envelope_ref", record.AutonomyEnvelopeRef); err != nil {
		return err
	}
	if err := validateAutonomyIdentifierField("mission store standing directive", "budget_ref", record.BudgetRef); err != nil {
		return err
	}
	if err := ValidateStandingDirectiveSchedule(record.Schedule); err != nil {
		return err
	}
	if len(record.SuccessCriteria) == 0 {
		return fmt.Errorf("mission store standing directive success_criteria are required")
	}
	if len(record.StopConditions) == 0 {
		return fmt.Errorf("mission store standing directive stop_conditions are required")
	}
	if !isValidStandingDirectiveOwnerPauseState(record.OwnerPauseState) {
		return fmt.Errorf("mission store standing directive owner_pause_state %q is invalid", record.OwnerPauseState)
	}
	if !isValidStandingDirectiveState(record.State) {
		return fmt.Errorf("mission store standing directive state %q is invalid", record.State)
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store standing directive created_at is required")
	}
	if record.UpdatedAt.IsZero() {
		return fmt.Errorf("mission store standing directive updated_at is required")
	}
	if record.UpdatedAt.Before(record.CreatedAt) {
		return fmt.Errorf("mission store standing directive updated_at must not be before created_at")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store standing directive created_by is required")
	}
	return nil
}

func ValidateStandingDirectiveSchedule(schedule StandingDirectiveSchedule) error {
	if !isValidStandingDirectiveScheduleKind(schedule.Kind) {
		return fmt.Errorf("mission store standing directive schedule kind %q is invalid", schedule.Kind)
	}
	if schedule.DueAt.IsZero() {
		return fmt.Errorf("mission store standing directive schedule due_at is required")
	}
	if schedule.IntervalSeconds <= 0 {
		return fmt.Errorf("mission store standing directive schedule interval_seconds must be positive")
	}
	return nil
}

func ValidateAutonomyBudgetDebitRecord(record AutonomyBudgetDebitRecord) error {
	if err := validateAutonomyIdentifierField("wake cycle budget debit", "budget_id", record.BudgetID); err != nil {
		return err
	}
	if record.DebitKind == "" {
		return fmt.Errorf("wake cycle budget debit debit_kind is required")
	}
	if !isValidAutonomyBudgetDebitKind(record.DebitKind) {
		return fmt.Errorf("wake cycle budget debit debit_kind %q is invalid", record.DebitKind)
	}
	if record.Amount <= 0 {
		return fmt.Errorf("wake cycle budget debit amount must be positive")
	}
	if record.Unit == "" {
		return fmt.Errorf("wake cycle budget debit unit is required")
	}
	if !isValidAutonomyBudgetDebitUnitForKind(record.DebitKind, record.Unit) {
		return fmt.Errorf("wake cycle budget debit unit %q is invalid for debit_kind %q", record.Unit, record.DebitKind)
	}
	return nil
}

func ValidateWakeCycleRecord(record WakeCycleRecord) error {
	if record.RecordVersion <= 0 {
		return fmt.Errorf("mission store wake cycle record_version must be positive")
	}
	if err := ValidateWakeCycleRef(WakeCycleRef{WakeCycleID: record.WakeCycleID}); err != nil {
		return err
	}
	if record.StartedAt.IsZero() {
		return fmt.Errorf("mission store wake cycle started_at is required")
	}
	if record.CompletedAt.IsZero() {
		return fmt.Errorf("mission store wake cycle completed_at is required")
	}
	if record.CompletedAt.Before(record.StartedAt) {
		return fmt.Errorf("mission store wake cycle completed_at must not be before started_at")
	}
	if !isValidWakeCycleTrigger(record.Trigger) {
		return fmt.Errorf("mission store wake cycle trigger %q is invalid", record.Trigger)
	}
	if !isValidWakeCycleDecision(record.Decision) {
		return fmt.Errorf("mission store wake cycle decision %q is invalid", record.Decision)
	}
	for i, debit := range record.BudgetDebits {
		if err := ValidateAutonomyBudgetDebitRecord(debit); err != nil {
			return fmt.Errorf("mission store wake cycle budget_debits[%d]: %w", i, err)
		}
	}
	if record.NextWakeAt.IsZero() {
		return fmt.Errorf("mission store wake cycle next_wake_at is required")
	}
	if record.NextWakeAt.Before(record.StartedAt) {
		return fmt.Errorf("mission store wake cycle next_wake_at must not be before started_at")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("mission store wake cycle created_at is required")
	}
	if record.CreatedBy == "" {
		return fmt.Errorf("mission store wake cycle created_by is required")
	}
	if record.AutonomyEnvelopeRef != "" {
		if err := validateAutonomyIdentifierField("mission store wake cycle", "autonomy_envelope_ref", record.AutonomyEnvelopeRef); err != nil {
			return err
		}
	}
	if record.BudgetRef != "" {
		if err := validateAutonomyIdentifierField("mission store wake cycle", "budget_ref", record.BudgetRef); err != nil {
			return err
		}
	}
	if record.Decision == WakeCycleDecisionMissionProposed {
		return validateWakeCycleMissionProposal(record)
	}
	if record.Decision == WakeCycleDecisionNoEligible {
		return validateWakeCycleNoEligible(record)
	}
	if record.Decision == WakeCycleDecisionBlocked {
		return validateWakeCycleBlocked(record)
	}
	return nil
}

func StoreStandingDirectiveRecord(root string, record StandingDirectiveRecord) (StandingDirectiveRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return StandingDirectiveRecord{}, false, err
	}
	record = NormalizeStandingDirectiveRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateStandingDirectiveRecord(record); err != nil {
		return StandingDirectiveRecord{}, false, err
	}

	path := StoreStandingDirectivePath(root, record.StandingDirectiveID)
	if existing, err := loadStandingDirectiveRecordFile(path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return StandingDirectiveRecord{}, false, fmt.Errorf("mission store standing directive %q already exists", record.StandingDirectiveID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return StandingDirectiveRecord{}, false, err
	}

	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return StandingDirectiveRecord{}, false, err
	}
	return record, true, nil
}

func LoadStandingDirectiveRecord(root, standingDirectiveID string) (StandingDirectiveRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return StandingDirectiveRecord{}, err
	}
	ref := NormalizeStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: standingDirectiveID})
	if err := ValidateStandingDirectiveRef(ref); err != nil {
		return StandingDirectiveRecord{}, err
	}
	record, err := loadStandingDirectiveRecordFile(StoreStandingDirectivePath(root, ref.StandingDirectiveID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StandingDirectiveRecord{}, ErrStandingDirectiveRecordNotFound
		}
		return StandingDirectiveRecord{}, err
	}
	return record, nil
}

func ListStandingDirectiveRecords(root string) ([]StandingDirectiveRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreStandingDirectivesDir(root), loadStandingDirectiveRecordFile)
}

func StoreWakeCycleRecord(root string, record WakeCycleRecord) (WakeCycleRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WakeCycleRecord{}, false, err
	}
	record = NormalizeWakeCycleRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := ValidateWakeCycleRecord(record); err != nil {
		return WakeCycleRecord{}, false, err
	}

	path := StoreWakeCyclePath(root, record.WakeCycleID)
	if existing, err := loadWakeCycleRecordFile(path); err == nil {
		if reflect.DeepEqual(existing, record) {
			return existing, false, nil
		}
		return WakeCycleRecord{}, false, fmt.Errorf("mission store wake cycle %q already exists", record.WakeCycleID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return WakeCycleRecord{}, false, err
	}

	if err := WriteStoreJSONAtomic(path, record); err != nil {
		return WakeCycleRecord{}, false, err
	}
	return record, true, nil
}

func LoadWakeCycleRecord(root, wakeCycleID string) (WakeCycleRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WakeCycleRecord{}, err
	}
	ref := NormalizeWakeCycleRef(WakeCycleRef{WakeCycleID: wakeCycleID})
	if err := ValidateWakeCycleRef(ref); err != nil {
		return WakeCycleRecord{}, err
	}
	record, err := loadWakeCycleRecordFile(StoreWakeCyclePath(root, ref.WakeCycleID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return WakeCycleRecord{}, ErrWakeCycleRecordNotFound
		}
		return WakeCycleRecord{}, err
	}
	return record, nil
}

func ListWakeCycleRecords(root string) ([]WakeCycleRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	return listStoreJSONRecords(StoreWakeCyclesDir(root), loadWakeCycleRecordFile)
}

func CreateWakeCycleProposalFromStandingDirective(root, standingDirectiveID, selectedJobID, selectedMissionFamily, selectedExecutionPlane, selectedExecutionHost, createdBy string, startedAt time.Time) (WakeCycleRecord, bool, error) {
	directive, err := LoadStandingDirectiveRecord(root, standingDirectiveID)
	if err != nil {
		return WakeCycleRecord{}, false, err
	}
	directive = NormalizeStandingDirectiveRecord(directive)

	startedAt = startedAt.UTC()
	if startedAt.IsZero() {
		return WakeCycleRecord{}, false, fmt.Errorf("mission store wake cycle started_at is required")
	}
	if directive.State != StandingDirectiveStateActive {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q state %q is not active", directive.StandingDirectiveID, directive.State)
	}
	if directive.OwnerPauseState == StandingDirectiveOwnerPauseStatePaused {
		return WakeCycleRecord{}, false, fmt.Errorf("%s: standing directive %q owner_pause_state is paused", RejectionCodeV4AutonomyPaused, directive.StandingDirectiveID)
	}
	if startedAt.Before(directive.Schedule.DueAt) {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q is not due until %s", directive.StandingDirectiveID, directive.Schedule.DueAt.Format(time.RFC3339Nano))
	}

	selectedMissionFamily = strings.TrimSpace(selectedMissionFamily)
	selectedExecutionPlane = strings.TrimSpace(selectedExecutionPlane)
	selectedExecutionHost = strings.TrimSpace(selectedExecutionHost)
	if !containsString(directive.AllowedMissionFamilies, selectedMissionFamily) {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q does not allow mission_family %q", directive.StandingDirectiveID, selectedMissionFamily)
	}
	if !containsString(directive.AllowedExecutionPlanes, selectedExecutionPlane) {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q does not allow execution_plane %q", directive.StandingDirectiveID, selectedExecutionPlane)
	}
	if !containsString(directive.AllowedExecutionHosts, selectedExecutionHost) {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q does not allow execution_host %q", directive.StandingDirectiveID, selectedExecutionHost)
	}
	if requiredPlane, ok := requiredExecutionPlaneForMissionFamily(selectedMissionFamily); ok && requiredPlane != selectedExecutionPlane {
		return WakeCycleRecord{}, false, fmt.Errorf("standing directive %q selected mission_family %q requires execution_plane %q", directive.StandingDirectiveID, selectedMissionFamily, requiredPlane)
	}

	requestedDebits := autonomyBudgetDebitsForMissionFamily(directive.BudgetRef, selectedMissionFamily)
	blockedReasons := make([]string, 0, 3)
	if isHotUpdateMissionFamily(selectedMissionFamily) {
		if ownerPause, active, err := LoadActiveAutonomyOwnerPauseForBudget(root, directive.BudgetRef); err != nil {
			return WakeCycleRecord{}, false, err
		} else if active {
			blockedReasons = append(blockedReasons, string(RejectionCodeV4AutonomyPaused)+": "+ownerPause.OwnerPauseID)
		}
	}

	if pause, active, err := LoadActiveRepeatedFailurePauseForBudget(root, directive.BudgetRef); err != nil {
		return WakeCycleRecord{}, false, err
	} else if active {
		blockedReasons = append(blockedReasons, string(RejectionCodeV4RepeatedFailurePause)+": "+pause.PauseID)
	}

	budgetAssessment, err := AssessAutonomyBudgetForDebits(root, directive.BudgetRef, requestedDebits, startedAt)
	if err != nil {
		return WakeCycleRecord{}, false, err
	}
	if !budgetAssessment.Allowed {
		blockedReasons = append(blockedReasons, budgetAssessment.Reason)
	}

	record := WakeCycleRecord{
		WakeCycleID:            WakeCycleIDFromDirectiveStartedAt(directive.StandingDirectiveID, startedAt),
		StartedAt:              startedAt,
		CompletedAt:            startedAt,
		Trigger:                WakeCycleTriggerStandingDirective,
		SelectedDirectiveID:    directive.StandingDirectiveID,
		SelectedJobID:          strings.TrimSpace(selectedJobID),
		SelectedMissionFamily:  selectedMissionFamily,
		SelectedExecutionPlane: selectedExecutionPlane,
		SelectedExecutionHost:  selectedExecutionHost,
		Decision:               WakeCycleDecisionMissionProposed,
		BudgetDebits:           requestedDebits,
		BlockedReasons:         nil,
		NextWakeAt:             startedAt.Add(time.Duration(directive.Schedule.IntervalSeconds) * time.Second),
		AutonomyEnvelopeRef:    directive.AutonomyEnvelopeRef,
		BudgetRef:              directive.BudgetRef,
		CreatedAt:              startedAt,
		CreatedBy:              createdBy,
	}
	if len(blockedReasons) > 0 {
		record.Decision = WakeCycleDecisionBlocked
		record.BudgetDebits = nil
		record.BlockedReasons = blockedReasons
	}
	return StoreWakeCycleRecord(root, record)
}

func CreateNoEligibleAutonomousActionHeartbeat(root, createdBy string, startedAt, nextWakeAt time.Time) (WakeCycleRecord, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return WakeCycleRecord{}, false, err
	}
	startedAt = startedAt.UTC()
	nextWakeAt = nextWakeAt.UTC()
	if startedAt.IsZero() {
		return WakeCycleRecord{}, false, fmt.Errorf("mission store wake cycle started_at is required")
	}
	if nextWakeAt.IsZero() {
		return WakeCycleRecord{}, false, fmt.Errorf("mission store wake cycle next_wake_at is required")
	}
	if nextWakeAt.Before(startedAt) {
		return WakeCycleRecord{}, false, fmt.Errorf("mission store wake cycle next_wake_at must not be before started_at")
	}

	directives, err := ListStandingDirectiveRecords(root)
	if err != nil {
		return WakeCycleRecord{}, false, err
	}
	for _, directive := range directives {
		if standingDirectiveHasEligibleWake(directive, startedAt) {
			return WakeCycleRecord{}, false, fmt.Errorf("eligible autonomous action exists for standing directive %q", directive.StandingDirectiveID)
		}
	}

	record := WakeCycleRecord{
		WakeCycleID:         WakeCycleIDFromNoEligibleStartedAt(startedAt),
		StartedAt:           startedAt,
		CompletedAt:         startedAt,
		Trigger:             WakeCycleTriggerIdleHeartbeat,
		Decision:            WakeCycleDecisionNoEligible,
		BlockedReasons:      []string{string(RejectionCodeV4NoEligibleAutonomousAction)},
		NextWakeAt:          nextWakeAt,
		CreatedAt:           startedAt,
		CreatedBy:           createdBy,
		BudgetDebits:        nil,
		SelectedJobID:       "",
		AutonomyEnvelopeRef: "",
		BudgetRef:           "",
	}
	return StoreWakeCycleRecord(root, record)
}

func loadStandingDirectiveRecordFile(path string) (StandingDirectiveRecord, error) {
	var record StandingDirectiveRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return StandingDirectiveRecord{}, err
	}
	record = NormalizeStandingDirectiveRecord(record)
	if err := ValidateStandingDirectiveRecord(record); err != nil {
		return StandingDirectiveRecord{}, err
	}
	return record, nil
}

func loadWakeCycleRecordFile(path string) (WakeCycleRecord, error) {
	var record WakeCycleRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		return WakeCycleRecord{}, err
	}
	record = NormalizeWakeCycleRecord(record)
	if err := ValidateWakeCycleRecord(record); err != nil {
		return WakeCycleRecord{}, err
	}
	return record, nil
}

func validateWakeCycleMissionProposal(record WakeCycleRecord) error {
	if err := ValidateStandingDirectiveRef(StandingDirectiveRef{StandingDirectiveID: record.SelectedDirectiveID}); err != nil {
		return fmt.Errorf("mission store wake cycle selected_directive_id %q: %w", record.SelectedDirectiveID, err)
	}
	if err := validateAutonomyIdentifierField("mission store wake cycle", "selected_job_id", record.SelectedJobID); err != nil {
		return err
	}
	if !isKnownMissionFamily(record.SelectedMissionFamily) {
		return fmt.Errorf("mission store wake cycle selected_mission_family %q is invalid", record.SelectedMissionFamily)
	}
	if !isKnownExecutionPlane(record.SelectedExecutionPlane) {
		return fmt.Errorf("mission store wake cycle selected_execution_plane %q is invalid", record.SelectedExecutionPlane)
	}
	if !isKnownExecutionHost(record.SelectedExecutionHost) {
		return fmt.Errorf("mission store wake cycle selected_execution_host %q is invalid", record.SelectedExecutionHost)
	}
	if requiredPlane, ok := requiredExecutionPlaneForMissionFamily(record.SelectedMissionFamily); ok && requiredPlane != record.SelectedExecutionPlane {
		return fmt.Errorf("mission store wake cycle selected_mission_family %q requires selected_execution_plane %q", record.SelectedMissionFamily, requiredPlane)
	}
	if len(record.BlockedReasons) > 0 {
		return fmt.Errorf("mission store wake cycle blocked_reasons must be empty for decision %q", record.Decision)
	}
	return nil
}

func validateWakeCycleNoEligible(record WakeCycleRecord) error {
	if record.Trigger != WakeCycleTriggerIdleHeartbeat {
		return fmt.Errorf("mission store wake cycle trigger must be %q for decision %q", WakeCycleTriggerIdleHeartbeat, record.Decision)
	}
	if record.SelectedDirectiveID != "" || record.SelectedJobID != "" || record.SelectedMissionFamily != "" || record.SelectedExecutionPlane != "" || record.SelectedExecutionHost != "" {
		return fmt.Errorf("mission store wake cycle selected fields must be empty for decision %q", record.Decision)
	}
	if !containsString(record.BlockedReasons, string(RejectionCodeV4NoEligibleAutonomousAction)) {
		return fmt.Errorf("mission store wake cycle blocked_reasons must include %s for decision %q", RejectionCodeV4NoEligibleAutonomousAction, record.Decision)
	}
	return nil
}

func validateWakeCycleBlocked(record WakeCycleRecord) error {
	if len(record.BlockedReasons) == 0 {
		return fmt.Errorf("mission store wake cycle blocked_reasons are required for decision %q", record.Decision)
	}
	if containsAutonomyReason(record.BlockedReasons, string(RejectionCodeV4AutonomyBudgetExceeded)) && record.BudgetRef == "" {
		return fmt.Errorf("mission store wake cycle budget_ref is required for %s", RejectionCodeV4AutonomyBudgetExceeded)
	}
	return nil
}

func standingDirectiveHasEligibleWake(record StandingDirectiveRecord, startedAt time.Time) bool {
	record = NormalizeStandingDirectiveRecord(record)
	if record.State != StandingDirectiveStateActive {
		return false
	}
	if record.OwnerPauseState == StandingDirectiveOwnerPauseStatePaused {
		return false
	}
	if !isValidStandingDirectiveScheduleKind(record.Schedule.Kind) {
		return false
	}
	return !startedAt.Before(record.Schedule.DueAt)
}

func validateStandingDirectiveMissionFamilies(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("mission store standing directive allowed_mission_families are required")
	}
	for i, value := range values {
		if !isKnownMissionFamily(value) {
			return fmt.Errorf("mission store standing directive allowed_mission_families[%d] %q is invalid", i, value)
		}
	}
	return nil
}

func validateStandingDirectiveExecutionPlanes(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("mission store standing directive allowed_execution_planes are required")
	}
	for i, value := range values {
		if !isKnownExecutionPlane(value) {
			return fmt.Errorf("mission store standing directive allowed_execution_planes[%d] %q is invalid", i, value)
		}
	}
	return nil
}

func validateStandingDirectiveExecutionHosts(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("mission store standing directive allowed_execution_hosts are required")
	}
	for i, value := range values {
		if !isKnownExecutionHost(value) {
			return fmt.Errorf("mission store standing directive allowed_execution_hosts[%d] %q is invalid", i, value)
		}
	}
	return nil
}

func normalizeAutonomyStrings(values []string) []string {
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

func isValidStandingDirectiveState(state StandingDirectiveState) bool {
	switch state {
	case StandingDirectiveStateActive, StandingDirectiveStateRetired:
		return true
	default:
		return false
	}
}

func isValidStandingDirectiveOwnerPauseState(state StandingDirectiveOwnerPauseState) bool {
	switch state {
	case StandingDirectiveOwnerPauseStateNotPaused, StandingDirectiveOwnerPauseStatePaused:
		return true
	default:
		return false
	}
}

func isValidStandingDirectiveScheduleKind(kind StandingDirectiveScheduleKind) bool {
	switch kind {
	case StandingDirectiveScheduleKindInterval:
		return true
	default:
		return false
	}
}

func isValidWakeCycleTrigger(trigger WakeCycleTrigger) bool {
	switch trigger {
	case WakeCycleTriggerStandingDirective, WakeCycleTriggerIdleHeartbeat:
		return true
	default:
		return false
	}
}

func isValidWakeCycleDecision(decision WakeCycleDecision) bool {
	switch decision {
	case WakeCycleDecisionMissionProposed, WakeCycleDecisionNoEligible, WakeCycleDecisionBlocked:
		return true
	default:
		return false
	}
}

func containsString(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsAutonomyReason(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == want || strings.HasPrefix(value, want+":") {
			return true
		}
	}
	return false
}

func isHotUpdateMissionFamily(family string) bool {
	requiredPlane, ok := requiredExecutionPlaneForMissionFamily(strings.TrimSpace(family))
	return ok && requiredPlane == ExecutionPlaneHotUpdateGate
}

func autonomyBudgetDebitsForMissionFamily(budgetID, missionFamily string) []AutonomyBudgetDebitRecord {
	debitKind := AutonomyBudgetDebitKindCandidateMutation
	if isHotUpdateMissionFamily(missionFamily) {
		debitKind = AutonomyBudgetDebitKindHotUpdate
	}
	return []AutonomyBudgetDebitRecord{{
		BudgetID:  strings.TrimSpace(budgetID),
		DebitKind: debitKind,
		Amount:    1,
		Unit:      AutonomyBudgetDebitUnitCount,
	}}
}

func validateAutonomyIdentifierField(surface, fieldName, value string) error {
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
		case '-', '_', '.', ':':
			continue
		default:
			return fmt.Errorf("%s %s %q is invalid", surface, fieldName, normalized)
		}
	}
	return nil
}
