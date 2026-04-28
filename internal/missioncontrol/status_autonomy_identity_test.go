package missioncontrol

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorAutonomyIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorAutonomyIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.StandingDirectives) != 0 || len(got.WakeCycles) != 0 {
		t.Fatalf("directives/wake cycles = %d/%d, want empty", len(got.StandingDirectives), len(got.WakeCycles))
	}
}

func TestLoadOperatorAutonomyIdentityStatusSurfacesNoEligibleHeartbeat(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 18, 0, 0, 0, time.UTC)
	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-not-due", func(record *StandingDirectiveRecord) {
		record.Schedule.DueAt = now.Add(time.Hour)
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	heartbeat, _, err := CreateNoEligibleAutonomousActionHeartbeat(root, "autonomy-loop", now, now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("CreateNoEligibleAutonomousActionHeartbeat() error = %v", err)
	}

	got := LoadOperatorAutonomyIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if got.LastNoEligibleError != string(RejectionCodeV4NoEligibleAutonomousAction) {
		t.Fatalf("LastNoEligibleError = %q, want %s", got.LastNoEligibleError, RejectionCodeV4NoEligibleAutonomousAction)
	}
	if len(got.StandingDirectives) != 1 {
		t.Fatalf("StandingDirectives len = %d, want 1", len(got.StandingDirectives))
	}
	directiveStatus := got.StandingDirectives[0]
	if directiveStatus.State != "configured" || directiveStatus.StandingDirectiveID != directive.StandingDirectiveID {
		t.Fatalf("directive status = %#v, want configured %q", directiveStatus, directive.StandingDirectiveID)
	}
	if directiveStatus.OwnerPauseState != string(StandingDirectiveOwnerPauseStateNotPaused) || directiveStatus.DirectiveState != string(StandingDirectiveStateActive) {
		t.Fatalf("directive owner/state = %q/%q", directiveStatus.OwnerPauseState, directiveStatus.DirectiveState)
	}
	if directiveStatus.DueAt == nil || *directiveStatus.DueAt != "2026-05-15T19:00:00Z" {
		t.Fatalf("directive DueAt = %#v, want 2026-05-15T19:00:00Z", directiveStatus.DueAt)
	}
	if len(got.WakeCycles) != 1 {
		t.Fatalf("WakeCycles len = %d, want 1", len(got.WakeCycles))
	}
	wake := got.WakeCycles[0]
	if wake.State != "configured" || wake.WakeCycleID != heartbeat.WakeCycleID {
		t.Fatalf("wake status = %#v, want configured %q", wake, heartbeat.WakeCycleID)
	}
	if wake.Trigger != string(WakeCycleTriggerIdleHeartbeat) || wake.Decision != string(WakeCycleDecisionNoEligible) {
		t.Fatalf("wake trigger/decision = %q/%q, want idle heartbeat/no eligible", wake.Trigger, wake.Decision)
	}
	if !reflect.DeepEqual(wake.BlockedReasons, []string{string(RejectionCodeV4NoEligibleAutonomousAction)}) {
		t.Fatalf("wake BlockedReasons = %#v, want no eligible code", wake.BlockedReasons)
	}
	if wake.NextWakeAt == nil || *wake.NextWakeAt != "2026-05-15T18:15:00Z" {
		t.Fatalf("wake NextWakeAt = %#v, want 2026-05-15T18:15:00Z", wake.NextWakeAt)
	}
}

func TestLoadOperatorAutonomyIdentityStatusInvalidDoesNotHideValidWakeCycles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 19, 0, 0, 0, time.UTC)
	valid, _, err := CreateNoEligibleAutonomousActionHeartbeat(root, "autonomy-loop", now, now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("CreateNoEligibleAutonomousActionHeartbeat() error = %v", err)
	}

	invalid := validWakeCycleRecord(now.Add(-time.Minute), "wake-cycle-z-invalid", func(record *WakeCycleRecord) {
		record.RecordVersion = StoreRecordVersion
		record.Decision = WakeCycleDecisionNoEligible
		record.Trigger = WakeCycleTriggerIdleHeartbeat
		record.SelectedDirectiveID = ""
		record.SelectedJobID = ""
		record.SelectedMissionFamily = ""
		record.SelectedExecutionPlane = ""
		record.SelectedExecutionHost = ""
		record.BlockedReasons = nil
	})
	if err := WriteStoreJSONAtomic(StoreWakeCyclePath(root, invalid.WakeCycleID), invalid); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(invalid wake cycle) error = %v", err)
	}

	got := LoadOperatorAutonomyIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.WakeCycles) != 2 {
		t.Fatalf("WakeCycles len = %d, want 2", len(got.WakeCycles))
	}
	if got.WakeCycles[0].State != "configured" || got.WakeCycles[0].WakeCycleID != valid.WakeCycleID {
		t.Fatalf("WakeCycles[0] = %#v, want valid configured first", got.WakeCycles[0])
	}
	if got.WakeCycles[1].State != "invalid" || !strings.Contains(got.WakeCycles[1].Error, string(RejectionCodeV4NoEligibleAutonomousAction)) {
		t.Fatalf("WakeCycles[1] = %#v, want invalid no-eligible reason", got.WakeCycles[1])
	}
	if got.LastNoEligibleError != string(RejectionCodeV4NoEligibleAutonomousAction) {
		t.Fatalf("LastNoEligibleError = %q, want valid no eligible code", got.LastNoEligibleError)
	}
}

func TestLoadOperatorAutonomyIdentityStatusSurfacesBudgetAndBudgetExceeded(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 19, 30, 0, 0, time.UTC)
	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-budget", func(record *StandingDirectiveRecord) {
		record.Schedule.DueAt = now.Add(-time.Minute)
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	budget := storeAutonomyBudgetForDirective(t, root, now.Add(-time.Hour), directive, func(record *AutonomyBudgetRecord) {
		record.MaxCandidateMutationsPerDay = 0
	})
	blocked, _, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, "mission-proposal-1", MissionFamilyAutonomousMissionProposal, ExecutionPlaneLiveRuntime, ExecutionHostPhone, "autonomy-loop", now)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective() error = %v", err)
	}

	got := LoadOperatorAutonomyIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if got.LastBudgetExceededError != string(RejectionCodeV4AutonomyBudgetExceeded) {
		t.Fatalf("LastBudgetExceededError = %q, want %s", got.LastBudgetExceededError, RejectionCodeV4AutonomyBudgetExceeded)
	}
	if len(got.Budgets) != 1 || got.Budgets[0].BudgetID != budget.BudgetID {
		t.Fatalf("Budgets = %#v, want budget %q", got.Budgets, budget.BudgetID)
	}
	if got.Budgets[0].MaxCandidateMutationsPerDay != 0 || got.Budgets[0].ResetWindow != string(AutonomyBudgetResetWindowDailyUTC) {
		t.Fatalf("budget status = %#v, want exhausted daily budget", got.Budgets[0])
	}
	if len(got.WakeCycles) != 1 || got.WakeCycles[0].WakeCycleID != blocked.WakeCycleID {
		t.Fatalf("WakeCycles = %#v, want blocked wake cycle %q", got.WakeCycles, blocked.WakeCycleID)
	}
	if got.WakeCycles[0].Decision != string(WakeCycleDecisionBlocked) || !containsAutonomyReason(got.WakeCycles[0].BlockedReasons, string(RejectionCodeV4AutonomyBudgetExceeded)) {
		t.Fatalf("wake cycle status = %#v, want budget exceeded blocker", got.WakeCycles[0])
	}
}

func TestLoadOperatorAutonomyIdentityStatusReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 20, 0, 0, 0, time.UTC)
	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-not-due", func(record *StandingDirectiveRecord) {
		record.Schedule.DueAt = now.Add(time.Hour)
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	heartbeat, _, err := CreateNoEligibleAutonomousActionHeartbeat(root, "autonomy-loop", now, now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("CreateNoEligibleAutonomousActionHeartbeat() error = %v", err)
	}

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreStandingDirectivePath(root, directive.StandingDirectiveID),
		StoreWakeCyclePath(root, heartbeat.WakeCycleID),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	first := LoadOperatorAutonomyIdentityStatus(root)
	second := LoadOperatorAutonomyIdentityStatus(root)
	if first.State != "configured" || second.State != "configured" {
		t.Fatalf("read-model states = %q/%q, want configured/configured", first.State, second.State)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after status error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("record %s changed after autonomy status read", path)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesAutonomyIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := testLeaseSafeNow()
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	heartbeat, _, err := CreateNoEligibleAutonomousActionHeartbeat(root, "autonomy-loop", now, now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("CreateNoEligibleAutonomousActionHeartbeat() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.AutonomyIdentity == nil {
		t.Fatalf("RuntimeSummary.AutonomyIdentity = %#v, want populated autonomy identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.AutonomyIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.AutonomyIdentity.State = %q, want configured", snapshot.RuntimeSummary.AutonomyIdentity.State)
	}
	if len(snapshot.RuntimeSummary.AutonomyIdentity.WakeCycles) != 1 {
		t.Fatalf("RuntimeSummary.AutonomyIdentity.WakeCycles len = %d, want 1", len(snapshot.RuntimeSummary.AutonomyIdentity.WakeCycles))
	}
	if snapshot.RuntimeSummary.AutonomyIdentity.WakeCycles[0].WakeCycleID != heartbeat.WakeCycleID {
		t.Fatalf("RuntimeSummary.AutonomyIdentity.WakeCycles[0].WakeCycleID = %q, want %q", snapshot.RuntimeSummary.AutonomyIdentity.WakeCycles[0].WakeCycleID, heartbeat.WakeCycleID)
	}
}
