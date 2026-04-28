package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type OperatorAutonomyIdentityStatus struct {
	State                         string                             `json:"state"`
	Budgets                       []OperatorAutonomyBudgetStatus     `json:"budgets,omitempty"`
	StandingDirectives            []OperatorStandingDirectiveStatus  `json:"standing_directives,omitempty"`
	WakeCycles                    []OperatorWakeCycleStatus          `json:"wake_cycles,omitempty"`
	Failures                      []OperatorAutonomyFailureStatus    `json:"failures,omitempty"`
	Pauses                        []OperatorAutonomyPauseStatus      `json:"pauses,omitempty"`
	OwnerPauses                   []OperatorAutonomyOwnerPauseStatus `json:"owner_pauses,omitempty"`
	LastNoEligibleError           string                             `json:"last_no_eligible_error,omitempty"`
	LastBudgetExceededError       string                             `json:"last_budget_exceeded_error,omitempty"`
	LastRepeatedFailurePauseError string                             `json:"last_repeated_failure_pause_error,omitempty"`
	LastOwnerPauseError           string                             `json:"last_owner_pause_error,omitempty"`
}

type OperatorAutonomyBudgetStatus struct {
	State                        string   `json:"state"`
	BudgetID                     string   `json:"budget_id,omitempty"`
	MaxExternalActionsPerDay     int64    `json:"max_external_actions_per_day,omitempty"`
	MaxHotUpdatesPerDay          int64    `json:"max_hot_updates_per_day,omitempty"`
	MaxCandidateMutationsPerDay  int64    `json:"max_candidate_mutations_per_day,omitempty"`
	MaxAPISpendPerDay            int64    `json:"max_api_spend_per_day,omitempty"`
	MaxRuntimeMinutesPerCycle    int64    `json:"max_runtime_minutes_per_cycle,omitempty"`
	MaxFailedAttemptsBeforePause int      `json:"max_failed_attempts_before_pause,omitempty"`
	QuietHours                   []string `json:"quiet_hours,omitempty"`
	ResetWindow                  string   `json:"reset_window,omitempty"`
	LedgerRefs                   []string `json:"ledger_refs,omitempty"`
	CreatedAt                    *string  `json:"created_at,omitempty"`
	UpdatedAt                    *string  `json:"updated_at,omitempty"`
	CreatedBy                    string   `json:"created_by,omitempty"`
	Error                        string   `json:"error,omitempty"`
}

type OperatorStandingDirectiveStatus struct {
	State                  string   `json:"state"`
	StandingDirectiveID    string   `json:"standing_directive_id,omitempty"`
	Objective              string   `json:"objective,omitempty"`
	AllowedMissionFamilies []string `json:"allowed_mission_families,omitempty"`
	AllowedExecutionPlanes []string `json:"allowed_execution_planes,omitempty"`
	AllowedExecutionHosts  []string `json:"allowed_execution_hosts,omitempty"`
	AutonomyEnvelopeRef    string   `json:"autonomy_envelope_ref,omitempty"`
	BudgetRef              string   `json:"budget_ref,omitempty"`
	ScheduleKind           string   `json:"schedule_kind,omitempty"`
	DueAt                  *string  `json:"due_at,omitempty"`
	IntervalSeconds        int64    `json:"interval_seconds,omitempty"`
	SuccessCriteria        []string `json:"success_criteria,omitempty"`
	StopConditions         []string `json:"stop_conditions,omitempty"`
	OwnerPauseState        string   `json:"owner_pause_state,omitempty"`
	DirectiveState         string   `json:"directive_state,omitempty"`
	CreatedAt              *string  `json:"created_at,omitempty"`
	UpdatedAt              *string  `json:"updated_at,omitempty"`
	CreatedBy              string   `json:"created_by,omitempty"`
	Error                  string   `json:"error,omitempty"`
}

type OperatorWakeCycleStatus struct {
	State                  string   `json:"state"`
	WakeCycleID            string   `json:"wake_cycle_id,omitempty"`
	StartedAt              *string  `json:"started_at,omitempty"`
	CompletedAt            *string  `json:"completed_at,omitempty"`
	Trigger                string   `json:"trigger,omitempty"`
	SelectedDirectiveID    string   `json:"selected_directive_id,omitempty"`
	SelectedJobID          string   `json:"selected_job_id,omitempty"`
	SelectedMissionFamily  string   `json:"selected_mission_family,omitempty"`
	SelectedExecutionPlane string   `json:"selected_execution_plane,omitempty"`
	SelectedExecutionHost  string   `json:"selected_execution_host,omitempty"`
	Decision               string   `json:"decision,omitempty"`
	BlockedReasons         []string `json:"blocked_reasons,omitempty"`
	NextWakeAt             *string  `json:"next_wake_at,omitempty"`
	AutonomyEnvelopeRef    string   `json:"autonomy_envelope_ref,omitempty"`
	BudgetRef              string   `json:"budget_ref,omitempty"`
	CreatedAt              *string  `json:"created_at,omitempty"`
	CreatedBy              string   `json:"created_by,omitempty"`
	Error                  string   `json:"error,omitempty"`
}

type OperatorAutonomyFailureStatus struct {
	State               string  `json:"state"`
	FailureID           string  `json:"failure_id,omitempty"`
	WakeCycleID         string  `json:"wake_cycle_id,omitempty"`
	StandingDirectiveID string  `json:"standing_directive_id,omitempty"`
	BudgetID            string  `json:"budget_id,omitempty"`
	FailureKind         string  `json:"failure_kind,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	OccurredAt          *string `json:"occurred_at,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type OperatorAutonomyPauseStatus struct {
	State               string   `json:"state"`
	PauseID             string   `json:"pause_id,omitempty"`
	BudgetID            string   `json:"budget_id,omitempty"`
	StandingDirectiveID string   `json:"standing_directive_id,omitempty"`
	PauseKind           string   `json:"pause_kind,omitempty"`
	PauseState          string   `json:"pause_state,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	FailureIDs          []string `json:"failure_ids,omitempty"`
	PausedAt            *string  `json:"paused_at,omitempty"`
	CreatedAt           *string  `json:"created_at,omitempty"`
	CreatedBy           string   `json:"created_by,omitempty"`
	Error               string   `json:"error,omitempty"`
}

type OperatorAutonomyOwnerPauseStatus struct {
	State               string  `json:"state"`
	OwnerPauseID        string  `json:"owner_pause_id,omitempty"`
	BudgetID            string  `json:"budget_id,omitempty"`
	StandingDirectiveID string  `json:"standing_directive_id,omitempty"`
	AppliesToHotUpdates bool    `json:"applies_to_hot_updates"`
	PauseState          string  `json:"pause_state,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	AuthorityRef        string  `json:"authority_ref,omitempty"`
	PausedAt            *string `json:"paused_at,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

func WithAutonomyIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorAutonomyIdentityStatus(root)
	summary.AutonomyIdentity = &status
	return summary
}

func LoadOperatorAutonomyIdentityStatus(root string) OperatorAutonomyIdentityStatus {
	budgets, budgetsFound, budgetsInvalid, budgetsErr := loadOperatorAutonomyBudgetStatuses(root)
	directives, directivesFound, directivesInvalid, directivesErr := loadOperatorStandingDirectiveStatuses(root)
	wakeCycles, wakeCyclesFound, wakeCyclesInvalid, wakeCyclesErr := loadOperatorWakeCycleStatuses(root)
	failures, failuresFound, failuresInvalid, failuresErr := loadOperatorAutonomyFailureStatuses(root)
	pauses, pausesFound, pausesInvalid, pausesErr := loadOperatorAutonomyPauseStatuses(root)
	ownerPauses, ownerPausesFound, ownerPausesInvalid, ownerPausesErr := loadOperatorAutonomyOwnerPauseStatuses(root)
	if !budgetsFound && !directivesFound && !wakeCyclesFound && !failuresFound && !pausesFound && !ownerPausesFound {
		return OperatorAutonomyIdentityStatus{State: "not_configured"}
	}
	state := "configured"
	if budgetsErr != nil || directivesErr != nil || wakeCyclesErr != nil || failuresErr != nil || pausesErr != nil || ownerPausesErr != nil ||
		budgetsInvalid || directivesInvalid || wakeCyclesInvalid || failuresInvalid || pausesInvalid || ownerPausesInvalid {
		state = "invalid"
	}
	return OperatorAutonomyIdentityStatus{
		State:                         state,
		Budgets:                       budgets,
		StandingDirectives:            directives,
		WakeCycles:                    wakeCycles,
		Failures:                      failures,
		Pauses:                        pauses,
		OwnerPauses:                   ownerPauses,
		LastNoEligibleError:           operatorAutonomyLastNoEligibleError(wakeCycles),
		LastBudgetExceededError:       operatorAutonomyLastBudgetExceededError(wakeCycles),
		LastRepeatedFailurePauseError: operatorAutonomyLastRepeatedFailurePauseError(pauses),
		LastOwnerPauseError:           operatorAutonomyLastOwnerPauseError(ownerPauses),
	}
}

func loadOperatorAutonomyBudgetStatuses(root string) ([]OperatorAutonomyBudgetStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreAutonomyBudgetsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorAutonomyBudgetStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorAutonomyBudgetStatus(filepath.Join(StoreAutonomyBudgetsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorAutonomyBudgetStatus(path string) OperatorAutonomyBudgetStatus {
	status := OperatorAutonomyBudgetStatus{
		State:    "invalid",
		BudgetID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record AutonomyBudgetRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeAutonomyBudgetRecord(record)
	status = operatorAutonomyBudgetStatusFromRecord(record)
	if err := ValidateAutonomyBudgetRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorAutonomyBudgetStatusFromRecord(record AutonomyBudgetRecord) OperatorAutonomyBudgetStatus {
	return OperatorAutonomyBudgetStatus{
		BudgetID:                     record.BudgetID,
		MaxExternalActionsPerDay:     record.MaxExternalActionsPerDay,
		MaxHotUpdatesPerDay:          record.MaxHotUpdatesPerDay,
		MaxCandidateMutationsPerDay:  record.MaxCandidateMutationsPerDay,
		MaxAPISpendPerDay:            record.MaxAPISpendPerDay,
		MaxRuntimeMinutesPerCycle:    record.MaxRuntimeMinutesPerCycle,
		MaxFailedAttemptsBeforePause: record.MaxFailedAttemptsBeforePause,
		QuietHours:                   append([]string(nil), record.QuietHours...),
		ResetWindow:                  string(record.ResetWindow),
		LedgerRefs:                   append([]string(nil), record.LedgerRefs...),
		CreatedAt:                    formatOperatorStatusTime(record.CreatedAt),
		UpdatedAt:                    formatOperatorStatusTime(record.UpdatedAt),
		CreatedBy:                    record.CreatedBy,
	}
}

func loadOperatorAutonomyFailureStatuses(root string) ([]OperatorAutonomyFailureStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreAutonomyFailuresDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorAutonomyFailureStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorAutonomyFailureStatus(root, filepath.Join(StoreAutonomyFailuresDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorAutonomyFailureStatus(root, path string) OperatorAutonomyFailureStatus {
	status := OperatorAutonomyFailureStatus{
		State:     "invalid",
		FailureID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record AutonomyFailureRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeAutonomyFailureRecord(record)
	status = operatorAutonomyFailureStatusFromRecord(record)
	if err := ValidateAutonomyFailureRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateAutonomyFailureLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorAutonomyFailureStatusFromRecord(record AutonomyFailureRecord) OperatorAutonomyFailureStatus {
	return OperatorAutonomyFailureStatus{
		FailureID:           record.FailureID,
		WakeCycleID:         record.WakeCycleID,
		StandingDirectiveID: record.StandingDirectiveID,
		BudgetID:            record.BudgetID,
		FailureKind:         string(record.FailureKind),
		Reason:              record.Reason,
		OccurredAt:          formatOperatorStatusTime(record.OccurredAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorAutonomyPauseStatuses(root string) ([]OperatorAutonomyPauseStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreAutonomyPausesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorAutonomyPauseStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorAutonomyPauseStatus(root, filepath.Join(StoreAutonomyPausesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorAutonomyPauseStatus(root, path string) OperatorAutonomyPauseStatus {
	status := OperatorAutonomyPauseStatus{
		State:   "invalid",
		PauseID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record AutonomyPauseRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeAutonomyPauseRecord(record)
	status = operatorAutonomyPauseStatusFromRecord(record)
	if err := ValidateAutonomyPauseRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateAutonomyPauseLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorAutonomyPauseStatusFromRecord(record AutonomyPauseRecord) OperatorAutonomyPauseStatus {
	return OperatorAutonomyPauseStatus{
		PauseID:             record.PauseID,
		BudgetID:            record.BudgetID,
		StandingDirectiveID: record.StandingDirectiveID,
		PauseKind:           string(record.PauseKind),
		PauseState:          string(record.State),
		Reason:              record.Reason,
		FailureIDs:          append([]string(nil), record.FailureIDs...),
		PausedAt:            formatOperatorStatusTime(record.PausedAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorAutonomyOwnerPauseStatuses(root string) ([]OperatorAutonomyOwnerPauseStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreAutonomyOwnerPausesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorAutonomyOwnerPauseStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorAutonomyOwnerPauseStatus(root, filepath.Join(StoreAutonomyOwnerPausesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorAutonomyOwnerPauseStatus(root, path string) OperatorAutonomyOwnerPauseStatus {
	status := OperatorAutonomyOwnerPauseStatus{
		State:        "invalid",
		OwnerPauseID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record AutonomyOwnerPauseRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeAutonomyOwnerPauseRecord(record)
	status = operatorAutonomyOwnerPauseStatusFromRecord(record)
	if err := ValidateAutonomyOwnerPauseRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateAutonomyOwnerPauseLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorAutonomyOwnerPauseStatusFromRecord(record AutonomyOwnerPauseRecord) OperatorAutonomyOwnerPauseStatus {
	return OperatorAutonomyOwnerPauseStatus{
		OwnerPauseID:        record.OwnerPauseID,
		BudgetID:            record.BudgetID,
		StandingDirectiveID: record.StandingDirectiveID,
		AppliesToHotUpdates: record.AppliesToHotUpdates,
		PauseState:          string(record.State),
		Reason:              record.Reason,
		AuthorityRef:        record.AuthorityRef,
		PausedAt:            formatOperatorStatusTime(record.PausedAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorStandingDirectiveStatuses(root string) ([]OperatorStandingDirectiveStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreStandingDirectivesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorStandingDirectiveStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorStandingDirectiveStatus(filepath.Join(StoreStandingDirectivesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorStandingDirectiveStatus(path string) OperatorStandingDirectiveStatus {
	status := OperatorStandingDirectiveStatus{
		State:               "invalid",
		StandingDirectiveID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record StandingDirectiveRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeStandingDirectiveRecord(record)
	status = operatorStandingDirectiveStatusFromRecord(record)
	if err := ValidateStandingDirectiveRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorStandingDirectiveStatusFromRecord(record StandingDirectiveRecord) OperatorStandingDirectiveStatus {
	return OperatorStandingDirectiveStatus{
		StandingDirectiveID:    record.StandingDirectiveID,
		Objective:              record.Objective,
		AllowedMissionFamilies: append([]string(nil), record.AllowedMissionFamilies...),
		AllowedExecutionPlanes: append([]string(nil), record.AllowedExecutionPlanes...),
		AllowedExecutionHosts:  append([]string(nil), record.AllowedExecutionHosts...),
		AutonomyEnvelopeRef:    record.AutonomyEnvelopeRef,
		BudgetRef:              record.BudgetRef,
		ScheduleKind:           string(record.Schedule.Kind),
		DueAt:                  formatOperatorStatusTime(record.Schedule.DueAt),
		IntervalSeconds:        record.Schedule.IntervalSeconds,
		SuccessCriteria:        append([]string(nil), record.SuccessCriteria...),
		StopConditions:         append([]string(nil), record.StopConditions...),
		OwnerPauseState:        string(record.OwnerPauseState),
		DirectiveState:         string(record.State),
		CreatedAt:              formatOperatorStatusTime(record.CreatedAt),
		UpdatedAt:              formatOperatorStatusTime(record.UpdatedAt),
		CreatedBy:              record.CreatedBy,
	}
}

func loadOperatorWakeCycleStatuses(root string) ([]OperatorWakeCycleStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}
	entries, err := os.ReadDir(StoreWakeCyclesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorWakeCycleStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorWakeCycleStatus(filepath.Join(StoreWakeCyclesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, status)
	}
	return statuses, true, invalid, nil
}

func loadOperatorWakeCycleStatus(path string) OperatorWakeCycleStatus {
	status := OperatorWakeCycleStatus{
		State:       "invalid",
		WakeCycleID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record WakeCycleRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}
	record = NormalizeWakeCycleRecord(record)
	status = operatorWakeCycleStatusFromRecord(record)
	if err := ValidateWakeCycleRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorWakeCycleStatusFromRecord(record WakeCycleRecord) OperatorWakeCycleStatus {
	return OperatorWakeCycleStatus{
		WakeCycleID:            record.WakeCycleID,
		StartedAt:              formatOperatorStatusTime(record.StartedAt),
		CompletedAt:            formatOperatorStatusTime(record.CompletedAt),
		Trigger:                string(record.Trigger),
		SelectedDirectiveID:    record.SelectedDirectiveID,
		SelectedJobID:          record.SelectedJobID,
		SelectedMissionFamily:  record.SelectedMissionFamily,
		SelectedExecutionPlane: record.SelectedExecutionPlane,
		SelectedExecutionHost:  record.SelectedExecutionHost,
		Decision:               string(record.Decision),
		BlockedReasons:         append([]string(nil), record.BlockedReasons...),
		NextWakeAt:             formatOperatorStatusTime(record.NextWakeAt),
		AutonomyEnvelopeRef:    record.AutonomyEnvelopeRef,
		BudgetRef:              record.BudgetRef,
		CreatedAt:              formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:              record.CreatedBy,
	}
}

func operatorAutonomyLastNoEligibleError(wakeCycles []OperatorWakeCycleStatus) string {
	for i := len(wakeCycles) - 1; i >= 0; i-- {
		status := wakeCycles[i]
		if status.Decision != string(WakeCycleDecisionNoEligible) {
			continue
		}
		if containsString(status.BlockedReasons, string(RejectionCodeV4NoEligibleAutonomousAction)) {
			return string(RejectionCodeV4NoEligibleAutonomousAction)
		}
	}
	return ""
}

func operatorAutonomyLastBudgetExceededError(wakeCycles []OperatorWakeCycleStatus) string {
	for i := len(wakeCycles) - 1; i >= 0; i-- {
		status := wakeCycles[i]
		if containsAutonomyReason(status.BlockedReasons, string(RejectionCodeV4AutonomyBudgetExceeded)) {
			return string(RejectionCodeV4AutonomyBudgetExceeded)
		}
	}
	return ""
}

func operatorAutonomyLastRepeatedFailurePauseError(pauses []OperatorAutonomyPauseStatus) string {
	for i := len(pauses) - 1; i >= 0; i-- {
		status := pauses[i]
		if status.PauseKind == string(AutonomyPauseKindRepeatedFailure) && containsAutonomyReason([]string{status.Reason}, string(RejectionCodeV4RepeatedFailurePause)) {
			return string(RejectionCodeV4RepeatedFailurePause)
		}
	}
	return ""
}

func operatorAutonomyLastOwnerPauseError(pauses []OperatorAutonomyOwnerPauseStatus) string {
	for i := len(pauses) - 1; i >= 0; i-- {
		status := pauses[i]
		if status.PauseState == string(AutonomyOwnerPauseStateActive) && status.AppliesToHotUpdates {
			return string(RejectionCodeV4AutonomyPaused)
		}
	}
	return ""
}
