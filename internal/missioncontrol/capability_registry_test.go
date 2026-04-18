package missioncontrol

import (
	"errors"
	"reflect"
	"testing"
)

func TestCapabilityRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if err := StoreCapabilityRecord(root, CapabilityRecord{
		CapabilityID:  "notifications-terminal",
		Class:         "notifications",
		Name:          "notifications-terminal",
		Exposed:       false,
		AuthorityTier: AuthorityTierLow,
		Validator:     "termux notification bridge unavailable",
		Notes:         "Terminal notifications remain unexposed on this body",
	}); err != nil {
		t.Fatalf("StoreCapabilityRecord(notifications-terminal) error = %v", err)
	}

	want := CapabilityRecord{
		CapabilityID:  " notifications-telegram ",
		Class:         " notifications ",
		Name:          " notifications ",
		Exposed:       true,
		AuthorityTier: AuthorityTier(" medium "),
		Validator:     " telegram owner-control channel confirmed ",
		Notes:         " notifications capability exposed through the configured Telegram owner-control channel ",
	}
	if err := StoreCapabilityRecord(root, want); err != nil {
		t.Fatalf("StoreCapabilityRecord(notifications-telegram) error = %v", err)
	}

	got, err := LoadCapabilityRecord(root, "notifications-telegram")
	if err != nil {
		t.Fatalf("LoadCapabilityRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.CapabilityID = "notifications-telegram"
	want.Class = "notifications"
	want.Name = "notifications"
	want.AuthorityTier = AuthorityTierMedium
	want.Validator = "telegram owner-control channel confirmed"
	want.Notes = "notifications capability exposed through the configured Telegram owner-control channel"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadCapabilityRecord() = %#v, want %#v", got, want)
	}

	records, err := ListCapabilityRecords(root)
	if err != nil {
		t.Fatalf("ListCapabilityRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCapabilityRecords() len = %d, want 2", len(records))
	}
	if records[0].CapabilityID != "notifications-telegram" || records[1].CapabilityID != "notifications-terminal" {
		t.Fatalf("ListCapabilityRecords() ids = [%q %q], want [notifications-telegram notifications-terminal]", records[0].CapabilityID, records[1].CapabilityID)
	}
}

func TestCapabilityRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing capability id",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.CapabilityID = " "
				}))
			},
			want: "mission store capability capability_id is required",
		},
		{
			name: "malformed capability id",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.CapabilityID = "notifications/telegram"
				}))
			},
			want: `mission store capability capability_id "notifications/telegram" is invalid`,
		},
		{
			name: "missing class",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.Class = " "
				}))
			},
			want: "mission store capability class is required",
		},
		{
			name: "missing name",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.Name = " "
				}))
			},
			want: "mission store capability name is required",
		},
		{
			name: "invalid authority tier",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.AuthorityTier = AuthorityTier("owner")
				}))
			},
			want: `mission store capability authority_tier "owner" is invalid`,
		},
		{
			name: "missing validator",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.Validator = " "
				}))
			},
			want: "mission store capability validator is required",
		},
		{
			name: "missing notes",
			run: func() error {
				return StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
					record.Notes = " "
				}))
			},
			want: "mission store capability notes is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreCapabilityRecord() error = nil, want fail-closed validation")
			}
			if err.Error() != tc.want {
				t.Fatalf("StoreCapabilityRecord() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadCapabilityRecordFailsClosedOnMalformedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := WriteStoreJSONAtomic(StoreCapabilityPath(root, "notifications-bad"), map[string]any{
		"record_version": StoreRecordVersion,
		"capability_id":  "notifications-bad",
		"class":          "notifications",
		"name":           "notifications",
		"exposed":        true,
		"authority_tier": "owner",
		"validator":      "telegram owner-control channel confirmed",
		"notes":          "malformed record should fail closed",
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := LoadCapabilityRecord(root, "notifications-bad")
	if err == nil {
		t.Fatal("LoadCapabilityRecord() error = nil, want malformed-record rejection")
	}
	if got := err.Error(); got != `mission store capability authority_tier "owner" is invalid` {
		t.Fatalf("LoadCapabilityRecord() error = %q, want malformed-record rejection", got)
	}
}

func TestResolveCapabilityRecordByNameRequiresExactlyOneCommittedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := StoreCapabilityRecord(root, validCapabilityRecord(nil)); err != nil {
		t.Fatalf("StoreCapabilityRecord() error = %v", err)
	}

	record, err := ResolveCapabilityRecordByName(root, "notifications")
	if err != nil {
		t.Fatalf("ResolveCapabilityRecordByName() error = %v", err)
	}
	if record.CapabilityID != "notifications-telegram" {
		t.Fatalf("ResolveCapabilityRecordByName().CapabilityID = %q, want %q", record.CapabilityID, "notifications-telegram")
	}

	if err := StoreCapabilityRecord(root, validCapabilityRecord(func(record *CapabilityRecord) {
		record.CapabilityID = "notifications-ssh"
		record.Validator = "ssh owner-control terminal confirmed"
		record.Notes = "duplicate notifications record should make resolution fail closed"
	})); err != nil {
		t.Fatalf("StoreCapabilityRecord(duplicate name) error = %v", err)
	}

	_, err = ResolveCapabilityRecordByName(root, "notifications")
	if err == nil {
		t.Fatal("ResolveCapabilityRecordByName() error = nil, want ambiguity rejection")
	}
	if !errors.Is(err, ErrCapabilityRecordAmbiguous) {
		t.Fatalf("ResolveCapabilityRecordByName() error = %v, want ErrCapabilityRecordAmbiguous", err)
	}
}

func validCapabilityRecord(mutate func(*CapabilityRecord)) CapabilityRecord {
	record := CapabilityRecord{
		CapabilityID:  "notifications-telegram",
		Class:         "notifications",
		Name:          "notifications",
		Exposed:       true,
		AuthorityTier: AuthorityTierMedium,
		Validator:     "telegram owner-control channel confirmed",
		Notes:         "notifications capability exposed through the configured Telegram owner-control channel",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}
