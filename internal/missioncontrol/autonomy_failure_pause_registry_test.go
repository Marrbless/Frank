package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecordAutonomyFailureFromWakeCycleTriggersRepeatedFailurePause(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	directive := storeAutonomyFailurePauseDirective(t, root, now, 3)

	var pause *AutonomyPauseRecord
	var pauseChanged bool
	for i := 0; i < 3; i++ {
		wakeAt := now.Add(time.Duration(i) * time.Minute)
		wake := storeAutonomyFailurePauseWakeCycle(t, root, directive, wakeAt)
		failure, gotPause, changed, err := RecordAutonomyFailureFromWakeCycle(root, wake.WakeCycleID, AutonomyFailureKindWakeCycle, "local eval failed", "autonomy-loop", wakeAt.Add(30*time.Second))
		if err != nil {
			t.Fatalf("RecordAutonomyFailureFromWakeCycle(%d) error = %v", i, err)
		}
		if failure.FailureID != AutonomyFailureIDFromWakeCycle(wake.WakeCycleID) {
			t.Fatalf("FailureID = %q, want deterministic id for %q", failure.FailureID, wake.WakeCycleID)
		}
		if i < 2 && gotPause != nil {
			t.Fatalf("pause after failure %d = %#v, want nil before threshold", i+1, gotPause)
		}
		if i == 2 {
			pause = gotPause
			pauseChanged = changed
		}
	}

	if pause == nil {
		t.Fatal("pause = nil, want repeated-failure pause at threshold")
	}
	if !pauseChanged {
		t.Fatal("pauseChanged = false, want true on first pause creation")
	}
	if pause.PauseID != AutonomyRepeatedFailurePauseID(directive.BudgetRef) {
		t.Fatalf("PauseID = %q, want deterministic repeated-failure pause id", pause.PauseID)
	}
	if pause.State != AutonomyPauseStateActive || pause.PauseKind != AutonomyPauseKindRepeatedFailure {
		t.Fatalf("pause state/kind = %q/%q, want active/repeated_failure", pause.State, pause.PauseKind)
	}
	if len(pause.FailureIDs) != 3 {
		t.Fatalf("FailureIDs len = %d, want 3", len(pause.FailureIDs))
	}
	if !strings.Contains(pause.Reason, string(RejectionCodeV4RepeatedFailurePause)) {
		t.Fatalf("Reason = %q, want repeated failure code", pause.Reason)
	}

	loaded, err := LoadAutonomyPauseRecord(root, pause.PauseID)
	if err != nil {
		t.Fatalf("LoadAutonomyPauseRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, *pause) {
		t.Fatalf("LoadAutonomyPauseRecord() = %#v, want %#v", loaded, *pause)
	}

	blockedAt := now.Add(10 * time.Minute)
	blocked, changed, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, "mission-proposal-blocked", MissionFamilyAutonomousMissionProposal, ExecutionPlaneLiveRuntime, ExecutionHostPhone, "autonomy-loop", blockedAt)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(blocked) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateWakeCycleProposalFromStandingDirective(blocked) changed = false, want durable blocked cycle")
	}
	if blocked.Decision != WakeCycleDecisionBlocked || !containsAutonomyReason(blocked.BlockedReasons, string(RejectionCodeV4RepeatedFailurePause)) {
		t.Fatalf("blocked wake cycle = %#v, want repeated-failure pause blocker", blocked)
	}
	if len(blocked.BudgetDebits) != 0 {
		t.Fatalf("blocked BudgetDebits = %#v, want no debit while paused", blocked.BudgetDebits)
	}
}

func TestConsecutiveAutonomyFailuresResetAfterSuccessfulWakeCycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 17, 11, 0, 0, 0, time.UTC)
	directive := storeAutonomyFailurePauseDirective(t, root, now, 2)

	first := storeAutonomyFailurePauseWakeCycle(t, root, directive, now)
	if _, _, _, err := RecordAutonomyFailureFromWakeCycle(root, first.WakeCycleID, AutonomyFailureKindWakeCycle, "first failed", "autonomy-loop", now.Add(30*time.Second)); err != nil {
		t.Fatalf("RecordAutonomyFailureFromWakeCycle(first) error = %v", err)
	}
	_ = storeAutonomyFailurePauseWakeCycle(t, root, directive, now.Add(time.Minute))
	third := storeAutonomyFailurePauseWakeCycle(t, root, directive, now.Add(2*time.Minute))
	if _, pause, _, err := RecordAutonomyFailureFromWakeCycle(root, third.WakeCycleID, AutonomyFailureKindWakeCycle, "third failed", "autonomy-loop", now.Add(2*time.Minute+30*time.Second)); err != nil {
		t.Fatalf("RecordAutonomyFailureFromWakeCycle(third) error = %v", err)
	} else if pause != nil {
		t.Fatalf("pause = %#v, want nil because successful wake cycle reset the streak", pause)
	}
}

func TestAutonomyFailureAndPauseReplayRejectsDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	directive := storeAutonomyFailurePauseDirective(t, root, now, 1)
	wake := storeAutonomyFailurePauseWakeCycle(t, root, directive, now)
	failure, pause, _, err := RecordAutonomyFailureFromWakeCycle(root, wake.WakeCycleID, AutonomyFailureKindWakeCycle, "failed once", "autonomy-loop", now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("RecordAutonomyFailureFromWakeCycle() error = %v", err)
	}
	if pause == nil {
		t.Fatal("pause = nil, want threshold 1 pause")
	}

	replayedFailure, changed, err := StoreAutonomyFailureRecord(root, failure)
	if err != nil {
		t.Fatalf("StoreAutonomyFailureRecord(replay) error = %v", err)
	}
	if changed || !reflect.DeepEqual(replayedFailure, failure) {
		t.Fatalf("failure replay = %#v changed=%v, want idempotent %#v", replayedFailure, changed, failure)
	}
	divergentFailure := failure
	divergentFailure.Reason = "different"
	if _, _, err := StoreAutonomyFailureRecord(root, divergentFailure); err == nil {
		t.Fatal("StoreAutonomyFailureRecord(divergent) error = nil, want duplicate rejection")
	}

	replayedPause, changed, err := StoreAutonomyPauseRecord(root, *pause)
	if err != nil {
		t.Fatalf("StoreAutonomyPauseRecord(replay) error = %v", err)
	}
	if changed || !reflect.DeepEqual(replayedPause, *pause) {
		t.Fatalf("pause replay = %#v changed=%v, want idempotent %#v", replayedPause, changed, *pause)
	}
	divergentPause := *pause
	divergentPause.Reason = string(RejectionCodeV4RepeatedFailurePause) + ": different"
	if _, _, err := StoreAutonomyPauseRecord(root, divergentPause); err == nil {
		t.Fatal("StoreAutonomyPauseRecord(divergent) error = nil, want duplicate rejection")
	}
}

func storeAutonomyFailurePauseDirective(t *testing.T, root string, now time.Time, maxFailures int) StandingDirectiveRecord {
	t.Helper()

	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-failure-pause", func(record *StandingDirectiveRecord) {
		record.Schedule.DueAt = now.Add(-time.Minute)
		record.Schedule.IntervalSeconds = 60
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	storeAutonomyBudgetForDirective(t, root, now.Add(-time.Hour), directive, func(record *AutonomyBudgetRecord) {
		record.MaxFailedAttemptsBeforePause = maxFailures
		record.MaxCandidateMutationsPerDay = 20
	})
	return directive
}

func storeAutonomyFailurePauseWakeCycle(t *testing.T, root string, directive StandingDirectiveRecord, at time.Time) WakeCycleRecord {
	t.Helper()

	record, _, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, "mission-proposal-"+at.Format("150405"), MissionFamilyAutonomousMissionProposal, ExecutionPlaneLiveRuntime, ExecutionHostPhone, "autonomy-loop", at)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(%s) error = %v", at.Format(time.RFC3339), err)
	}
	if record.Decision != WakeCycleDecisionMissionProposed {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(%s).Decision = %q, want mission_proposed", at.Format(time.RFC3339), record.Decision)
	}
	return record
}
