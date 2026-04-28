package missioncontrol

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStandingDirectiveRecordRoundTripListReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	_, changed, err := StoreStandingDirectiveRecord(root, validStandingDirectiveRecord(now, "standing-directive-b", nil))
	if err != nil {
		t.Fatalf("StoreStandingDirectiveRecord(standing-directive-b) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreStandingDirectiveRecord(standing-directive-b) changed = false, want true")
	}

	want := validStandingDirectiveRecord(now.Add(time.Minute), " standing-directive-a ", func(record *StandingDirectiveRecord) {
		record.Objective = " keep local improvement loop supplied with bounded work "
		record.AllowedMissionFamilies = []string{" " + MissionFamilyAutonomousMissionProposal + " ", MissionFamilyStandingDirectiveReview, " "}
		record.AllowedExecutionPlanes = []string{" " + ExecutionPlaneLiveRuntime + " ", " "}
		record.AllowedExecutionHosts = []string{" " + ExecutionHostPhone + " ", ExecutionHostDesktopDev, " "}
		record.AutonomyEnvelopeRef = " envelope-local "
		record.BudgetRef = " budget-local "
		record.SuccessCriteria = []string{" one eligible bounded mission proposal exists ", " "}
		record.StopConditions = []string{" owner pause ", " budget exhausted "}
		record.CreatedBy = " operator "
	})
	got, changed, err := StoreStandingDirectiveRecord(root, want)
	if err != nil {
		t.Fatalf("StoreStandingDirectiveRecord(standing-directive-a) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreStandingDirectiveRecord(standing-directive-a) changed = false, want true")
	}

	want.RecordVersion = StoreRecordVersion
	want = NormalizeStandingDirectiveRecord(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StoreStandingDirectiveRecord() = %#v, want %#v", got, want)
	}

	loaded, err := LoadStandingDirectiveRecord(root, "standing-directive-a")
	if err != nil {
		t.Fatalf("LoadStandingDirectiveRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadStandingDirectiveRecord() = %#v, want %#v", loaded, want)
	}

	records, err := ListStandingDirectiveRecords(root)
	if err != nil {
		t.Fatalf("ListStandingDirectiveRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].StandingDirectiveID != "standing-directive-a" || records[1].StandingDirectiveID != "standing-directive-b" {
		t.Fatalf("ListStandingDirectiveRecords() = %#v, want standing-directive-a then standing-directive-b", records)
	}

	replayed, changed, err := StoreStandingDirectiveRecord(root, want)
	if err != nil {
		t.Fatalf("StoreStandingDirectiveRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreStandingDirectiveRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StoreStandingDirectiveRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.Objective = "different objective"
	if _, _, err := StoreStandingDirectiveRecord(root, divergent); err == nil {
		t.Fatal("StoreStandingDirectiveRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store standing directive "standing-directive-a" already exists`) {
		t.Fatalf("StoreStandingDirectiveRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestStandingDirectiveRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*StandingDirectiveRecord)
		want string
	}{
		{name: "missing id", edit: func(record *StandingDirectiveRecord) { record.StandingDirectiveID = " " }, want: "standing_directive_id is required"},
		{name: "missing objective", edit: func(record *StandingDirectiveRecord) { record.Objective = " " }, want: "objective is required"},
		{name: "missing mission families", edit: func(record *StandingDirectiveRecord) { record.AllowedMissionFamilies = nil }, want: "allowed_mission_families are required"},
		{name: "unknown mission family", edit: func(record *StandingDirectiveRecord) { record.AllowedMissionFamilies = []string{"moonshot"} }, want: "allowed_mission_families[0]"},
		{name: "missing planes", edit: func(record *StandingDirectiveRecord) { record.AllowedExecutionPlanes = nil }, want: "allowed_execution_planes are required"},
		{name: "unknown plane", edit: func(record *StandingDirectiveRecord) { record.AllowedExecutionPlanes = []string{"satellite"} }, want: "allowed_execution_planes[0]"},
		{name: "missing hosts", edit: func(record *StandingDirectiveRecord) { record.AllowedExecutionHosts = nil }, want: "allowed_execution_hosts are required"},
		{name: "unknown host", edit: func(record *StandingDirectiveRecord) { record.AllowedExecutionHosts = []string{"pager"} }, want: "allowed_execution_hosts[0]"},
		{name: "missing envelope", edit: func(record *StandingDirectiveRecord) { record.AutonomyEnvelopeRef = " " }, want: "autonomy_envelope_ref is required"},
		{name: "missing budget", edit: func(record *StandingDirectiveRecord) { record.BudgetRef = " " }, want: "budget_ref is required"},
		{name: "invalid schedule", edit: func(record *StandingDirectiveRecord) { record.Schedule.IntervalSeconds = 0 }, want: "interval_seconds must be positive"},
		{name: "missing success criteria", edit: func(record *StandingDirectiveRecord) { record.SuccessCriteria = nil }, want: "success_criteria are required"},
		{name: "missing stop conditions", edit: func(record *StandingDirectiveRecord) { record.StopConditions = nil }, want: "stop_conditions are required"},
		{name: "invalid pause state", edit: func(record *StandingDirectiveRecord) { record.OwnerPauseState = "maybe" }, want: "owner_pause_state"},
		{name: "invalid state", edit: func(record *StandingDirectiveRecord) { record.State = "sleeping" }, want: "state"},
		{name: "missing created at", edit: func(record *StandingDirectiveRecord) { record.CreatedAt = time.Time{} }, want: "created_at is required"},
		{name: "missing updated at", edit: func(record *StandingDirectiveRecord) { record.UpdatedAt = time.Time{} }, want: "updated_at is required"},
		{name: "missing created by", edit: func(record *StandingDirectiveRecord) { record.CreatedBy = " " }, want: "created_by is required"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := StoreStandingDirectiveRecord(root, validStandingDirectiveRecord(now, "standing-directive-"+strings.ReplaceAll(tc.name, " ", "-"), tc.edit))
			if err == nil {
				t.Fatal("StoreStandingDirectiveRecord() error = nil, want fail-closed validation")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreStandingDirectiveRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestWakeCycleIDFromDirectiveStartedAt(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 5, 15, 9, 30, 45, 123456789, time.FixedZone("offset", -4*60*60))
	got := WakeCycleIDFromDirectiveStartedAt(" standing-directive-a ", startedAt)
	want := "wake-cycle-standing-directive-a-20260515T133045123456789Z"
	if got != want {
		t.Fatalf("WakeCycleIDFromDirectiveStartedAt() = %q, want %q", got, want)
	}
}

func TestCreateWakeCycleProposalFromStandingDirectiveCreatesDueMissionProposal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	directive := validStandingDirectiveRecord(now.Add(-2*time.Hour), "standing-directive-due", func(record *StandingDirectiveRecord) {
		record.Schedule.DueAt = now.Add(-time.Minute)
		record.Schedule.IntervalSeconds = 1800
		record.AllowedMissionFamilies = []string{MissionFamilyAutonomousMissionProposal, MissionFamilyStandingDirectiveReview}
		record.AllowedExecutionPlanes = []string{ExecutionPlaneLiveRuntime}
		record.AllowedExecutionHosts = []string{ExecutionHostPhone, ExecutionHostDesktopDev}
	})
	if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
		t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
	}

	record, changed, err := CreateWakeCycleProposalFromStandingDirective(
		root,
		" standing-directive-due ",
		" mission-proposal-1 ",
		" "+MissionFamilyAutonomousMissionProposal+" ",
		" "+ExecutionPlaneLiveRuntime+" ",
		" "+ExecutionHostPhone+" ",
		" autonomy-loop ",
		now,
	)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateWakeCycleProposalFromStandingDirective() changed = false, want true")
	}
	if record.WakeCycleID != WakeCycleIDFromDirectiveStartedAt("standing-directive-due", now) {
		t.Fatalf("WakeCycleID = %q, want deterministic id", record.WakeCycleID)
	}
	if record.SelectedDirectiveID != "standing-directive-due" || record.SelectedJobID != "mission-proposal-1" {
		t.Fatalf("selected directive/job = %q/%q, want standing-directive-due/mission-proposal-1", record.SelectedDirectiveID, record.SelectedJobID)
	}
	if record.Decision != WakeCycleDecisionMissionProposed || record.Trigger != WakeCycleTriggerStandingDirective {
		t.Fatalf("decision/trigger = %q/%q, want mission_proposed/standing_directive", record.Decision, record.Trigger)
	}
	if record.SelectedMissionFamily != MissionFamilyAutonomousMissionProposal || record.SelectedExecutionPlane != ExecutionPlaneLiveRuntime || record.SelectedExecutionHost != ExecutionHostPhone {
		t.Fatalf("selected mission/plane/host = %q/%q/%q", record.SelectedMissionFamily, record.SelectedExecutionPlane, record.SelectedExecutionHost)
	}
	if !record.NextWakeAt.Equal(now.Add(30 * time.Minute)) {
		t.Fatalf("NextWakeAt = %v, want %v", record.NextWakeAt, now.Add(30*time.Minute))
	}
	if record.AutonomyEnvelopeRef != directive.AutonomyEnvelopeRef || record.BudgetRef != directive.BudgetRef {
		t.Fatalf("envelope/budget refs = %q/%q, want %q/%q", record.AutonomyEnvelopeRef, record.BudgetRef, directive.AutonomyEnvelopeRef, directive.BudgetRef)
	}
	if len(record.BudgetDebits) != 0 || len(record.BlockedReasons) != 0 {
		t.Fatalf("budget debits/blocked reasons = %#v/%#v, want empty", record.BudgetDebits, record.BlockedReasons)
	}

	loaded, err := LoadWakeCycleRecord(root, record.WakeCycleID)
	if err != nil {
		t.Fatalf("LoadWakeCycleRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, record) {
		t.Fatalf("LoadWakeCycleRecord() = %#v, want %#v", loaded, record)
	}

	replayed, changed, err := CreateWakeCycleProposalFromStandingDirective(
		root,
		directive.StandingDirectiveID,
		record.SelectedJobID,
		record.SelectedMissionFamily,
		record.SelectedExecutionPlane,
		record.SelectedExecutionHost,
		record.CreatedBy,
		now,
	)
	if err != nil {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreateWakeCycleProposalFromStandingDirective(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, record) {
		t.Fatalf("CreateWakeCycleProposalFromStandingDirective(replay) = %#v, want %#v", replayed, record)
	}
}

func TestCreateWakeCycleProposalFromStandingDirectiveRejectsIneligibleSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		edit      func(*StandingDirectiveRecord, time.Time)
		jobID     string
		family    string
		plane     string
		host      string
		startedAt func(time.Time) time.Time
		want      string
	}{
		{
			name:      "not due",
			startedAt: func(now time.Time) time.Time { return now.Add(-time.Hour) },
			want:      "is not due until",
		},
		{
			name: "retired directive",
			edit: func(record *StandingDirectiveRecord, now time.Time) {
				record.State = StandingDirectiveStateRetired
			},
			want: "is not active",
		},
		{
			name: "paused directive",
			edit: func(record *StandingDirectiveRecord, now time.Time) {
				record.OwnerPauseState = StandingDirectiveOwnerPauseStatePaused
			},
			want: string(RejectionCodeV4AutonomyPaused),
		},
		{
			name:   "disallowed family",
			family: MissionFamilyStandingDirectiveReview,
			want:   "does not allow mission_family",
		},
		{
			name:  "disallowed plane",
			plane: ExecutionPlaneImprovementWorkspace,
			want:  "does not allow execution_plane",
		},
		{
			name: "disallowed host",
			host: ExecutionHostRemoteProvider,
			want: "does not allow execution_host",
		},
		{
			name: "mission family plane mismatch",
			edit: func(record *StandingDirectiveRecord, now time.Time) {
				record.AllowedExecutionPlanes = []string{ExecutionPlaneLiveRuntime, ExecutionPlaneImprovementWorkspace}
			},
			family: MissionFamilyAutonomousMissionProposal,
			plane:  ExecutionPlaneImprovementWorkspace,
			want:   "requires execution_plane",
		},
		{
			name:  "missing selected job id",
			jobID: " ",
			want:  "selected_job_id is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 15, 13, 0, 0, 0, time.UTC)
			directive := validStandingDirectiveRecord(now.Add(-2*time.Hour), "standing-directive-"+strings.ReplaceAll(tc.name, " ", "-"), func(record *StandingDirectiveRecord) {
				record.Schedule.DueAt = now.Add(-time.Minute)
				record.AllowedMissionFamilies = []string{MissionFamilyAutonomousMissionProposal}
				record.AllowedExecutionPlanes = []string{ExecutionPlaneLiveRuntime}
				record.AllowedExecutionHosts = []string{ExecutionHostPhone}
				if tc.edit != nil {
					tc.edit(record, now)
				}
			})
			if _, _, err := StoreStandingDirectiveRecord(root, directive); err != nil {
				t.Fatalf("StoreStandingDirectiveRecord() error = %v", err)
			}

			jobID := tc.jobID
			if jobID == "" {
				jobID = "mission-proposal"
			}
			family := tc.family
			if family == "" {
				family = MissionFamilyAutonomousMissionProposal
			}
			plane := tc.plane
			if plane == "" {
				plane = ExecutionPlaneLiveRuntime
			}
			host := tc.host
			if host == "" {
				host = ExecutionHostPhone
			}
			startedAt := now
			if tc.startedAt != nil {
				startedAt = tc.startedAt(now)
			}

			if _, _, err := CreateWakeCycleProposalFromStandingDirective(root, directive.StandingDirectiveID, jobID, family, plane, host, "autonomy-loop", startedAt); err == nil {
				t.Fatal("CreateWakeCycleProposalFromStandingDirective() error = nil, want rejection")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CreateWakeCycleProposalFromStandingDirective() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestWakeCycleRecordLoadMissingAndListOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadWakeCycleRecord(root, "wake-cycle-missing"); !errors.Is(err, ErrWakeCycleRecordNotFound) {
		t.Fatalf("LoadWakeCycleRecord() error = %v, want %v", err, ErrWakeCycleRecordNotFound)
	}
	now := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)
	for _, id := range []string{"wake-cycle-b", "wake-cycle-a"} {
		record := validWakeCycleRecord(now, id, nil)
		if _, _, err := StoreWakeCycleRecord(root, record); err != nil {
			t.Fatalf("StoreWakeCycleRecord(%s) error = %v", id, err)
		}
	}
	records, err := ListWakeCycleRecords(root)
	if err != nil {
		t.Fatalf("ListWakeCycleRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].WakeCycleID != "wake-cycle-a" || records[1].WakeCycleID != "wake-cycle-b" {
		t.Fatalf("ListWakeCycleRecords() = %#v, want wake-cycle-a then wake-cycle-b", records)
	}
}

func validStandingDirectiveRecord(now time.Time, directiveID string, edit func(*StandingDirectiveRecord)) StandingDirectiveRecord {
	record := StandingDirectiveRecord{
		StandingDirectiveID:    directiveID,
		Objective:              "propose bounded local autonomous work",
		AllowedMissionFamilies: []string{MissionFamilyAutonomousMissionProposal},
		AllowedExecutionPlanes: []string{ExecutionPlaneLiveRuntime},
		AllowedExecutionHosts:  []string{ExecutionHostPhone, ExecutionHostDesktopDev},
		AutonomyEnvelopeRef:    "autonomy-envelope-local",
		BudgetRef:              "autonomy-budget-local",
		Schedule: StandingDirectiveSchedule{
			Kind:            StandingDirectiveScheduleKindInterval,
			DueAt:           now.Add(-time.Minute),
			IntervalSeconds: 3600,
		},
		SuccessCriteria: []string{"eligible bounded mission is proposed"},
		StopConditions:  []string{"owner pause", "budget exhausted"},
		OwnerPauseState: StandingDirectiveOwnerPauseStateNotPaused,
		State:           StandingDirectiveStateActive,
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}

func validWakeCycleRecord(now time.Time, wakeCycleID string, edit func(*WakeCycleRecord)) WakeCycleRecord {
	record := WakeCycleRecord{
		WakeCycleID:            wakeCycleID,
		StartedAt:              now,
		CompletedAt:            now,
		Trigger:                WakeCycleTriggerStandingDirective,
		SelectedDirectiveID:    "standing-directive-a",
		SelectedJobID:          "mission-proposal-a",
		SelectedMissionFamily:  MissionFamilyAutonomousMissionProposal,
		SelectedExecutionPlane: ExecutionPlaneLiveRuntime,
		SelectedExecutionHost:  ExecutionHostPhone,
		Decision:               WakeCycleDecisionMissionProposed,
		BudgetDebits:           nil,
		BlockedReasons:         nil,
		NextWakeAt:             now.Add(time.Hour),
		AutonomyEnvelopeRef:    "autonomy-envelope-local",
		BudgetRef:              "autonomy-budget-local",
		CreatedAt:              now,
		CreatedBy:              "autonomy-loop",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}
