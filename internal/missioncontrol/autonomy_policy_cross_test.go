package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestWakeCyclePolicyBlockersAggregateWithoutBudgetDebit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-policy-cross-hot-update", func(record *StandingDirectiveRecord) {
		record.BudgetRef = "autonomy-budget-policy-cross"
		record.AllowedMissionFamilies = []string{MissionFamilyApplyHotUpdate}
		record.AllowedExecutionPlanes = []string{ExecutionPlaneHotUpdateGate}
		record.AllowedExecutionHosts = []string{ExecutionHostPhone}
		record.Schedule.DueAt = now.Add(-time.Minute)
		record.Schedule.IntervalSeconds = 300
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	storeAutonomyBudgetForDirective(t, root, now.Add(-time.Hour), directive, func(record *AutonomyBudgetRecord) {
		record.MaxHotUpdatesPerDay = 1
		record.MaxCandidateMutationsPerDay = 5
		record.MaxFailedAttemptsBeforePause = 1
	})

	first, changed, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, "hot-update-first", MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate, ExecutionHostPhone, "autonomy-loop", now)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(first) error = %v", err)
	}
	if !changed || first.Decision != WakeCycleDecisionMissionProposed {
		t.Fatalf("first wake cycle = %#v changed=%v, want mission proposal", first, changed)
	}
	if len(first.BudgetDebits) != 1 || first.BudgetDebits[0].DebitKind != AutonomyBudgetDebitKindHotUpdate {
		t.Fatalf("first BudgetDebits = %#v, want one hot-update debit", first.BudgetDebits)
	}

	_, failurePause, _, err := RecordAutonomyFailureFromWakeCycle(root, first.WakeCycleID, AutonomyFailureKindWakeCycle, "hot update failed", "autonomy-loop", now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("RecordAutonomyFailureFromWakeCycle() error = %v", err)
	}
	if failurePause == nil {
		t.Fatal("failurePause = nil, want repeated-failure pause after threshold")
	}
	ownerPause, _, err := StoreAutonomyOwnerPauseRecord(root, validAutonomyOwnerPauseRecord(now.Add(time.Minute), directive, nil))
	if err != nil {
		t.Fatalf("StoreAutonomyOwnerPauseRecord() error = %v", err)
	}

	blockedAt := now.Add(2 * time.Minute)
	blocked, changed, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, "hot-update-second", MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate, ExecutionHostPhone, "autonomy-loop", blockedAt)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(blocked) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateWakeCycleProposalFromStandingDirective(blocked) changed = false, want durable blocked cycle")
	}
	if blocked.Decision != WakeCycleDecisionBlocked {
		t.Fatalf("blocked Decision = %q, want blocked", blocked.Decision)
	}
	if len(blocked.BudgetDebits) != 0 {
		t.Fatalf("blocked BudgetDebits = %#v, want no debit when any policy blocks", blocked.BudgetDebits)
	}
	wantCodes := []string{
		string(RejectionCodeV4AutonomyPaused),
		string(RejectionCodeV4RepeatedFailurePause),
		string(RejectionCodeV4AutonomyBudgetExceeded),
	}
	if len(blocked.BlockedReasons) != len(wantCodes) {
		t.Fatalf("BlockedReasons = %#v, want %d policy blockers", blocked.BlockedReasons, len(wantCodes))
	}
	for i, wantCode := range wantCodes {
		if !containsAutonomyReason([]string{blocked.BlockedReasons[i]}, wantCode) {
			t.Fatalf("BlockedReasons[%d] = %q, want %s", i, blocked.BlockedReasons[i], wantCode)
		}
	}
	if !strings.Contains(blocked.BlockedReasons[0], ownerPause.OwnerPauseID) {
		t.Fatalf("BlockedReasons[0] = %q, want owner pause id %q", blocked.BlockedReasons[0], ownerPause.OwnerPauseID)
	}
	if !strings.Contains(blocked.BlockedReasons[1], failurePause.PauseID) {
		t.Fatalf("BlockedReasons[1] = %q, want failure pause id %q", blocked.BlockedReasons[1], failurePause.PauseID)
	}
	if blocked.BudgetRef != directive.BudgetRef {
		t.Fatalf("BudgetRef = %q, want %q", blocked.BudgetRef, directive.BudgetRef)
	}
	if !blocked.NextWakeAt.Equal(blockedAt.Add(5 * time.Minute)) {
		t.Fatalf("NextWakeAt = %s, want %s", blocked.NextWakeAt, blockedAt.Add(5*time.Minute))
	}

	status := LoadOperatorAutonomyIdentityStatus(root)
	if status.LastOwnerPauseError != string(RejectionCodeV4AutonomyPaused) {
		t.Fatalf("LastOwnerPauseError = %q, want %s", status.LastOwnerPauseError, RejectionCodeV4AutonomyPaused)
	}
	if status.LastRepeatedFailurePauseError != string(RejectionCodeV4RepeatedFailurePause) {
		t.Fatalf("LastRepeatedFailurePauseError = %q, want %s", status.LastRepeatedFailurePauseError, RejectionCodeV4RepeatedFailurePause)
	}
	if status.LastBudgetExceededError != string(RejectionCodeV4AutonomyBudgetExceeded) {
		t.Fatalf("LastBudgetExceededError = %q, want %s", status.LastBudgetExceededError, RejectionCodeV4AutonomyBudgetExceeded)
	}
}
