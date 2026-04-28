package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAutonomyBudgetRecordRoundTripListReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	_, changed, err := StoreAutonomyBudgetRecord(root, validAutonomyBudgetRecord(now, "autonomy-budget-b", nil))
	if err != nil {
		t.Fatalf("StoreAutonomyBudgetRecord(autonomy-budget-b) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreAutonomyBudgetRecord(autonomy-budget-b) changed = false, want true")
	}

	want := validAutonomyBudgetRecord(now.Add(time.Minute), " autonomy-budget-a ", func(record *AutonomyBudgetRecord) {
		record.MaxCandidateMutationsPerDay = 3
		record.QuietHours = []string{" none ", " "}
		record.LedgerRefs = []string{" autonomy-ledger ", " "}
		record.CreatedBy = " operator "
	})
	got, changed, err := StoreAutonomyBudgetRecord(root, want)
	if err != nil {
		t.Fatalf("StoreAutonomyBudgetRecord(autonomy-budget-a) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreAutonomyBudgetRecord(autonomy-budget-a) changed = false, want true")
	}

	want.RecordVersion = StoreRecordVersion
	want = NormalizeAutonomyBudgetRecord(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StoreAutonomyBudgetRecord() = %#v, want %#v", got, want)
	}

	loaded, err := LoadAutonomyBudgetRecord(root, "autonomy-budget-a")
	if err != nil {
		t.Fatalf("LoadAutonomyBudgetRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadAutonomyBudgetRecord() = %#v, want %#v", loaded, want)
	}

	records, err := ListAutonomyBudgetRecords(root)
	if err != nil {
		t.Fatalf("ListAutonomyBudgetRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].BudgetID != "autonomy-budget-a" || records[1].BudgetID != "autonomy-budget-b" {
		t.Fatalf("ListAutonomyBudgetRecords() = %#v, want autonomy-budget-a then autonomy-budget-b", records)
	}

	replayed, changed, err := StoreAutonomyBudgetRecord(root, want)
	if err != nil {
		t.Fatalf("StoreAutonomyBudgetRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreAutonomyBudgetRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StoreAutonomyBudgetRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.MaxCandidateMutationsPerDay = 5
	if _, _, err := StoreAutonomyBudgetRecord(root, divergent); err == nil {
		t.Fatal("StoreAutonomyBudgetRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store autonomy budget "autonomy-budget-a" already exists`) {
		t.Fatalf("StoreAutonomyBudgetRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestAutonomyBudgetRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 16, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*AutonomyBudgetRecord)
		want string
	}{
		{name: "missing id", edit: func(record *AutonomyBudgetRecord) { record.BudgetID = " " }, want: "budget_id is required"},
		{name: "negative candidate mutations", edit: func(record *AutonomyBudgetRecord) { record.MaxCandidateMutationsPerDay = -1 }, want: "max_candidate_mutations_per_day must be non-negative"},
		{name: "missing runtime minutes", edit: func(record *AutonomyBudgetRecord) { record.MaxRuntimeMinutesPerCycle = 0 }, want: "max_runtime_minutes_per_cycle must be positive"},
		{name: "missing failed attempts", edit: func(record *AutonomyBudgetRecord) { record.MaxFailedAttemptsBeforePause = 0 }, want: "max_failed_attempts_before_pause must be positive"},
		{name: "missing quiet hours", edit: func(record *AutonomyBudgetRecord) { record.QuietHours = nil }, want: "quiet_hours are required"},
		{name: "invalid reset window", edit: func(record *AutonomyBudgetRecord) { record.ResetWindow = "weekly" }, want: "reset_window"},
		{name: "missing ledger refs", edit: func(record *AutonomyBudgetRecord) { record.LedgerRefs = nil }, want: "ledger_refs are required"},
		{name: "missing created at", edit: func(record *AutonomyBudgetRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "missing updated at", edit: func(record *AutonomyBudgetRecord) { record.UpdatedAt = time.Time{} }, want: "updated_at is required"},
		{name: "missing created by", edit: func(record *AutonomyBudgetRecord) { record.CreatedBy = " " }, want: "created_by is required"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := StoreAutonomyBudgetRecord(root, validAutonomyBudgetRecord(now, "autonomy-budget-"+strings.ReplaceAll(tc.name, " ", "-"), tc.edit))
			if err == nil {
				t.Fatal("StoreAutonomyBudgetRecord() error = nil, want fail-closed validation")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreAutonomyBudgetRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestAssessAutonomyBudgetForDebitsAllowsAndBlocksDeterministically(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	budget := validAutonomyBudgetRecord(now.Add(-time.Hour), "autonomy-budget-local", func(record *AutonomyBudgetRecord) {
		record.MaxCandidateMutationsPerDay = 1
	})
	if _, _, err := StoreAutonomyBudgetRecord(root, budget); err != nil {
		t.Fatalf("StoreAutonomyBudgetRecord() error = %v", err)
	}

	request := []AutonomyBudgetDebitRecord{{
		BudgetID:  budget.BudgetID,
		DebitKind: AutonomyBudgetDebitKindCandidateMutation,
		Amount:    1,
		Unit:      AutonomyBudgetDebitUnitCount,
	}}
	first, err := AssessAutonomyBudgetForDebits(root, budget.BudgetID, request, now)
	if err != nil {
		t.Fatalf("AssessAutonomyBudgetForDebits(first) error = %v", err)
	}
	if !first.Allowed || first.Reason != "" {
		t.Fatalf("first assessment = %#v, want allowed", first)
	}

	if _, _, err := StoreWakeCycleRecord(root, validWakeCycleRecord(now.Add(time.Minute), "wake-cycle-used", func(record *WakeCycleRecord) {
		record.BudgetRef = budget.BudgetID
		record.BudgetDebits = request
	})); err != nil {
		t.Fatalf("StoreWakeCycleRecord(used) error = %v", err)
	}

	blocked, err := AssessAutonomyBudgetForDebits(root, budget.BudgetID, request, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("AssessAutonomyBudgetForDebits(blocked) error = %v", err)
	}
	if blocked.Allowed || !strings.Contains(blocked.Reason, string(RejectionCodeV4AutonomyBudgetExceeded)) {
		t.Fatalf("blocked assessment = %#v, want budget exceeded", blocked)
	}

	nextDay, err := AssessAutonomyBudgetForDebits(root, budget.BudgetID, request, now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("AssessAutonomyBudgetForDebits(next day) error = %v", err)
	}
	if !nextDay.Allowed {
		t.Fatalf("next day assessment = %#v, want reset window allowed", nextDay)
	}
}

func validAutonomyBudgetRecord(now time.Time, budgetID string, edit func(*AutonomyBudgetRecord)) AutonomyBudgetRecord {
	record := AutonomyBudgetRecord{
		BudgetID:                     budgetID,
		MaxExternalActionsPerDay:     0,
		MaxHotUpdatesPerDay:          0,
		MaxCandidateMutationsPerDay:  10,
		MaxAPISpendPerDay:            0,
		MaxRuntimeMinutesPerCycle:    15,
		MaxFailedAttemptsBeforePause: 3,
		QuietHours:                   []string{"none"},
		ResetWindow:                  AutonomyBudgetResetWindowDailyUTC,
		LedgerRefs:                   []string{"autonomy-ledger-local"},
		CreatedAt:                    now,
		UpdatedAt:                    now,
		CreatedBy:                    "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}

func storeAutonomyBudgetForDirective(t *testing.T, root string, now time.Time, directive StandingDirectiveRecord, edit func(*AutonomyBudgetRecord)) AutonomyBudgetRecord {
	t.Helper()

	record := validAutonomyBudgetRecord(now, directive.BudgetRef, edit)
	stored, _, err := StoreAutonomyBudgetRecord(root, record)
	if err != nil {
		t.Fatalf("StoreAutonomyBudgetRecord(%s) error = %v", directive.BudgetRef, err)
	}
	return stored
}
