package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestCapabilityOnboardingProposalRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 17, 22, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	if err := StoreCapabilityOnboardingProposalRecord(root, CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-b",
		CapabilityName:   "notifications",
		WhyNeeded:        "Surface mandatory approvals to the operator",
		MissionFamilies:  []string{"operator-control"},
		Risks:            []string{"operator interruption"},
		Validators:       []string{"notification permission check"},
		KillSwitch:       "disable notification bridge",
		DataAccessed:     []string{"notification metadata"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            CapabilityOnboardingProposalStateApproved,
	}); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord(proposal-b) error = %v", err)
	}

	want := CapabilityOnboardingProposalRecord{
		ProposalID:       " proposal-a ",
		CapabilityName:   " camera ",
		WhyNeeded:        " Capture bounded visual state for troubleshooting ",
		MissionFamilies:  []string{" diagnostics ", " maintenance "},
		Risks:            []string{" oversharing sensitive imagery "},
		Validators:       []string{" camera permission smoke test ", " bounded screenshot validator "},
		KillSwitch:       " disable_camera_bridge ",
		DataAccessed:     []string{" photos/media ", " device camera frames "},
		ApprovalRequired: true,
		CreatedAt:        now.Add(time.Minute),
		State:            CapabilityOnboardingProposalState(" proposed "),
	}
	if err := StoreCapabilityOnboardingProposalRecord(root, want); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord(proposal-a) error = %v", err)
	}

	got, err := LoadCapabilityOnboardingProposalRecord(root, "proposal-a")
	if err != nil {
		t.Fatalf("LoadCapabilityOnboardingProposalRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.ProposalID = "proposal-a"
	want.CapabilityName = "camera"
	want.WhyNeeded = "Capture bounded visual state for troubleshooting"
	want.MissionFamilies = []string{"diagnostics", "maintenance"}
	want.Risks = []string{"oversharing sensitive imagery"}
	want.Validators = []string{"camera permission smoke test", "bounded screenshot validator"}
	want.KillSwitch = "disable_camera_bridge"
	want.DataAccessed = []string{"photos/media", "device camera frames"}
	want.CreatedAt = want.CreatedAt.UTC()
	want.State = CapabilityOnboardingProposalStateProposed
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCapabilityOnboardingProposalRecord() = %#v, want %#v", got, want)
	}

	records, err := ListCapabilityOnboardingProposalRecords(root)
	if err != nil {
		t.Fatalf("ListCapabilityOnboardingProposalRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCapabilityOnboardingProposalRecords() len = %d, want 2", len(records))
	}
	if records[0].ProposalID != "proposal-a" || records[1].ProposalID != "proposal-b" {
		t.Fatalf("ListCapabilityOnboardingProposalRecords() ids = [%q %q], want [proposal-a proposal-b]", records[0].ProposalID, records[1].ProposalID)
	}
}

func TestCapabilityOnboardingProposalRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 17, 22, 5, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing proposal id",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.ProposalID = "   "
				}))
			},
			want: "mission store capability onboarding proposal proposal_id is required",
		},
		{
			name: "malformed proposal id",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.ProposalID = "proposal/one"
				}))
			},
			want: `mission store capability onboarding proposal proposal_id "proposal/one" is invalid`,
		},
		{
			name: "missing capability name",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.CapabilityName = " "
				}))
			},
			want: "mission store capability onboarding proposal capability_name is required",
		},
		{
			name: "missing why needed",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.WhyNeeded = " "
				}))
			},
			want: "mission store capability onboarding proposal why_needed is required",
		},
		{
			name: "missing mission families",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.MissionFamilies = nil
				}))
			},
			want: "mission store capability onboarding proposal mission_families is required",
		},
		{
			name: "missing risks",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.Risks = nil
				}))
			},
			want: "mission store capability onboarding proposal risks is required",
		},
		{
			name: "missing validators",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.Validators = nil
				}))
			},
			want: "mission store capability onboarding proposal validators is required",
		},
		{
			name: "missing kill switch",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.KillSwitch = " "
				}))
			},
			want: "mission store capability onboarding proposal kill_switch is required",
		},
		{
			name: "missing data accessed",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.DataAccessed = nil
				}))
			},
			want: "mission store capability onboarding proposal data_accessed is required",
		},
		{
			name: "missing created at",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.CreatedAt = time.Time{}
				}))
			},
			want: "mission store capability onboarding proposal created_at is required",
		},
		{
			name: "invalid state",
			run: func() error {
				return StoreCapabilityOnboardingProposalRecord(root, validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
					record.State = CapabilityOnboardingProposalState("unknown")
				}))
			},
			want: `mission store capability onboarding proposal state "unknown" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreCapabilityOnboardingProposalRecord() error = nil, want fail-closed validation")
			}
			if err.Error() != tc.want {
				t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadCapabilityOnboardingProposalRecordFailsClosedOnMalformedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := WriteStoreJSONAtomic(StoreCapabilityOnboardingProposalPath(root, "proposal-bad"), map[string]any{
		"record_version":    StoreRecordVersion,
		"proposal_id":       "proposal-bad",
		"capability_name":   "camera",
		"why_needed":        "Capture bounded visual state",
		"mission_families":  []string{"diagnostics"},
		"risks":             []string{"sensitive imagery"},
		"validators":        []string{"camera smoke test"},
		"kill_switch":       "disable camera",
		"data_accessed":     []string{"photos/media"},
		"approval_required": true,
		"created_at":        time.Date(2026, 4, 17, 22, 10, 0, 0, time.UTC),
		"state":             "mystery",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := LoadCapabilityOnboardingProposalRecord(root, "proposal-bad")
	if err == nil {
		t.Fatal("LoadCapabilityOnboardingProposalRecord() error = nil, want malformed-record rejection")
	}
	if got := err.Error(); got != `mission store capability onboarding proposal state "mystery" is invalid` {
		t.Fatalf("LoadCapabilityOnboardingProposalRecord() error = %q, want malformed-record rejection", got)
	}
}

func validCapabilityOnboardingProposalRecord(now time.Time, mutate func(*CapabilityOnboardingProposalRecord)) CapabilityOnboardingProposalRecord {
	record := CapabilityOnboardingProposalRecord{
		ProposalID:       "proposal-camera",
		CapabilityName:   "camera",
		WhyNeeded:        "Capture bounded visual state for diagnostics",
		MissionFamilies:  []string{"diagnostics"},
		Risks:            []string{"sensitive imagery"},
		Validators:       []string{"camera permission smoke test"},
		KillSwitch:       "disable camera bridge",
		DataAccessed:     []string{"photos/media"},
		ApprovalRequired: true,
		CreatedAt:        now,
		State:            CapabilityOnboardingProposalStateProposed,
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
