package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAutonomyOwnerPauseRecordRoundTripReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	directive := storeAutonomyOwnerPauseDirective(t, root, now, MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate)
	record := validAutonomyOwnerPauseRecord(now, directive, nil)

	got, changed, err := StoreAutonomyOwnerPauseRecord(root, record)
	if err != nil {
		t.Fatalf("StoreAutonomyOwnerPauseRecord() error = %v", err)
	}
	if !changed {
		t.Fatal("StoreAutonomyOwnerPauseRecord() changed = false, want true")
	}
	record.RecordVersion = StoreRecordVersion
	record = NormalizeAutonomyOwnerPauseRecord(record)
	if !reflect.DeepEqual(got, record) {
		t.Fatalf("StoreAutonomyOwnerPauseRecord() = %#v, want %#v", got, record)
	}

	loaded, err := LoadAutonomyOwnerPauseRecord(root, record.OwnerPauseID)
	if err != nil {
		t.Fatalf("LoadAutonomyOwnerPauseRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, record) {
		t.Fatalf("LoadAutonomyOwnerPauseRecord() = %#v, want %#v", loaded, record)
	}

	replayed, changed, err := StoreAutonomyOwnerPauseRecord(root, record)
	if err != nil {
		t.Fatalf("StoreAutonomyOwnerPauseRecord(replay) error = %v", err)
	}
	if changed || !reflect.DeepEqual(replayed, record) {
		t.Fatalf("replay = %#v changed=%v, want idempotent %#v", replayed, changed, record)
	}

	divergent := record
	divergent.Reason = "different pause reason"
	if _, _, err := StoreAutonomyOwnerPauseRecord(root, divergent); err == nil {
		t.Fatal("StoreAutonomyOwnerPauseRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store autonomy owner pause`) {
		t.Fatalf("StoreAutonomyOwnerPauseRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestAutonomyOwnerPauseRecordRejectsNaturalLanguageAuthority(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)
	directive := storeAutonomyOwnerPauseDirective(t, root, now, MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate)

	if _, _, err := StoreAutonomyOwnerPauseRecord(root, validAutonomyOwnerPauseRecord(now, directive, func(record *AutonomyOwnerPauseRecord) {
		record.AuthorityRef = "natural_language:chat-ok"
	})); err == nil {
		t.Fatal("StoreAutonomyOwnerPauseRecord() error = nil, want natural-language authority rejection")
	} else if !strings.Contains(err.Error(), "must not be natural-language approval") {
		t.Fatalf("StoreAutonomyOwnerPauseRecord() error = %q, want natural-language rejection", err.Error())
	}
}

func TestOwnerPauseBlocksAutonomyOriginatedHotUpdateProposalOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	hotDirective := storeAutonomyOwnerPauseDirective(t, root, now, MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate)
	if _, _, err := StoreAutonomyOwnerPauseRecord(root, validAutonomyOwnerPauseRecord(now, hotDirective, nil)); err != nil {
		t.Fatalf("StoreAutonomyOwnerPauseRecord() error = %v", err)
	}

	blocked, changed, err := CreateWakeCycleProposalFromStandingDirective(root, hotDirective.StandingDirectiveID, "hot-update-proposal", MissionFamilyApplyHotUpdate, ExecutionPlaneHotUpdateGate, ExecutionHostPhone, "autonomy-loop", now)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(hot update) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateWakeCycleProposalFromStandingDirective(hot update) changed = false, want blocked record")
	}
	if blocked.Decision != WakeCycleDecisionBlocked || !containsAutonomyReason(blocked.BlockedReasons, string(RejectionCodeV4AutonomyPaused)) {
		t.Fatalf("blocked hot-update proposal = %#v, want autonomy paused blocker", blocked)
	}
	if len(blocked.BudgetDebits) != 0 {
		t.Fatalf("blocked BudgetDebits = %#v, want no debit while owner-paused", blocked.BudgetDebits)
	}

	nonHotDirective := storeAutonomyOwnerPauseDirective(t, root, now.Add(time.Minute), MissionFamilyAutonomousMissionProposal, ExecutionPlaneLiveRuntime)
	allowed, _, err := CreateWakeCycleProposalFromStandingDirective(root, nonHotDirective.StandingDirectiveID, "mission-proposal", MissionFamilyAutonomousMissionProposal, ExecutionPlaneLiveRuntime, ExecutionHostPhone, "autonomy-loop", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(non-hot) error = %v", err)
	}
	if allowed.Decision != WakeCycleDecisionMissionProposed {
		t.Fatalf("non-hot Decision = %q, want mission_proposed", allowed.Decision)
	}
}

func storeAutonomyOwnerPauseDirective(t *testing.T, root string, now time.Time, family, plane string) StandingDirectiveRecord {
	t.Helper()

	directive := validStandingDirectiveRecord(now.Add(-time.Hour), "standing-directive-"+strings.ReplaceAll(family, "_", "-"), func(record *StandingDirectiveRecord) {
		record.BudgetRef = "autonomy-budget-" + strings.ReplaceAll(family, "_", "-")
		record.AllowedMissionFamilies = []string{family}
		record.AllowedExecutionPlanes = []string{plane}
		record.AllowedExecutionHosts = []string{ExecutionHostPhone}
		record.Schedule.DueAt = now.Add(-time.Minute)
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}
	storeAutonomyBudgetForDirective(t, root, now.Add(-time.Hour), directive, func(record *AutonomyBudgetRecord) {
		record.MaxHotUpdatesPerDay = 5
		record.MaxCandidateMutationsPerDay = 5
	})
	return directive
}

func validAutonomyOwnerPauseRecord(now time.Time, directive StandingDirectiveRecord, edit func(*AutonomyOwnerPauseRecord)) AutonomyOwnerPauseRecord {
	record := AutonomyOwnerPauseRecord{
		OwnerPauseID:        AutonomyOwnerPauseIDForBudget(directive.BudgetRef),
		BudgetID:            directive.BudgetRef,
		StandingDirectiveID: directive.StandingDirectiveID,
		AppliesToHotUpdates: true,
		State:               AutonomyOwnerPauseStateActive,
		Reason:              "owner paused autonomous hot updates",
		AuthorityRef:        "owner-control:manual-pause-1",
		PausedAt:            now,
		CreatedAt:           now,
		CreatedBy:           "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}
