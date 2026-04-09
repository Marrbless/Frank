package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTreasuryRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 9, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	if err := StoreTreasuryRecord(root, TreasuryRecord{
		TreasuryID:     "treasury-b",
		DisplayName:    "Frank Treasury B",
		State:          TreasuryStateBootstrap,
		ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreTreasuryRecord(treasury-b) error = %v", err)
	}

	want := TreasuryRecord{
		TreasuryID:     " treasury-a ",
		DisplayName:    " Frank Treasury A ",
		State:          TreasuryState(" funded "),
		ZeroSeedPolicy: TreasuryZeroSeedPolicy(" owner_seed_forbidden "),
		ContainerRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKind(" container "),
				ObjectID: " " + fixtures.container.ContainerID + " ",
			},
		},
		CreatedAt: now.Add(2 * time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}
	if err := StoreTreasuryRecord(root, want); err != nil {
		t.Fatalf("StoreTreasuryRecord(treasury-a) error = %v", err)
	}

	got, err := LoadTreasuryRecord(root, "treasury-a")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.TreasuryID = "treasury-a"
	want.DisplayName = "Frank Treasury A"
	want.State = TreasuryStateFunded
	want.ZeroSeedPolicy = TreasuryZeroSeedPolicyOwnerSeedForbidden
	want.ContainerRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		},
	}
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadTreasuryRecord() = %#v, want %#v", got, want)
	}

	records, err := ListTreasuryRecords(root)
	if err != nil {
		t.Fatalf("ListTreasuryRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListTreasuryRecords() len = %d, want 2", len(records))
	}
	if records[0].TreasuryID != "treasury-a" || records[1].TreasuryID != "treasury-b" {
		t.Fatalf("ListTreasuryRecords() ids = [%q %q], want [treasury-a treasury-b]", records[0].TreasuryID, records[1].TreasuryID)
	}
}

func TestTreasuryLedgerEntryRoundTripAndList(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 10, 0, 0, 0, time.FixedZone("offset", 2*60*60))

	if err := StoreTreasuryRecord(root, validTreasuryRecord(now.Add(-time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-a"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})); err != nil {
		t.Fatalf("StoreTreasuryRecord(treasury-a) error = %v", err)
	}

	if err := StoreTreasuryLedgerEntry(root, TreasuryLedgerEntry{
		EntryID:    "entry-b",
		TreasuryID: "treasury-a",
		EntryKind:  TreasuryLedgerEntryKindMovement,
		AssetCode:  "USDC",
		Amount:     "50.25",
		CreatedAt:  now,
		SourceRef:  "campaign:community-a",
	}); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry(entry-b) error = %v", err)
	}

	want := TreasuryLedgerEntry{
		EntryID:    " entry-a ",
		TreasuryID: " treasury-a ",
		EntryKind:  TreasuryLedgerEntryKind(" acquisition "),
		AssetCode:  " USD ",
		Amount:     " 100.00 ",
		CreatedAt:  now.Add(time.Minute),
		SourceRef:  " payout:listing-1 ",
	}
	if err := StoreTreasuryLedgerEntry(root, want); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry(entry-a) error = %v", err)
	}

	got, err := LoadTreasuryLedgerEntry(root, "treasury-a", "entry-a")
	if err != nil {
		t.Fatalf("LoadTreasuryLedgerEntry() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.EntryID = "entry-a"
	want.TreasuryID = "treasury-a"
	want.EntryKind = TreasuryLedgerEntryKindAcquisition
	want.AssetCode = "USD"
	want.Amount = "100.00"
	want.CreatedAt = want.CreatedAt.UTC()
	want.SourceRef = "payout:listing-1"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadTreasuryLedgerEntry() = %#v, want %#v", got, want)
	}

	entries, err := ListTreasuryLedgerEntries(root, "treasury-a")
	if err != nil {
		t.Fatalf("ListTreasuryLedgerEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ListTreasuryLedgerEntries() len = %d, want 2", len(entries))
	}
	if entries[0].EntryID != "entry-a" || entries[1].EntryID != "entry-b" {
		t.Fatalf("ListTreasuryLedgerEntries() ids = [%q %q], want [entry-a entry-b]", entries[0].EntryID, entries[1].EntryID)
	}
}

func TestTreasuryLedgerEntriesAreAppendOnly(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	if err := StoreTreasuryRecord(root, validTreasuryRecord(now.Add(-time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-a"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})); err != nil {
		t.Fatalf("StoreTreasuryRecord(treasury-a) error = %v", err)
	}
	entry := TreasuryLedgerEntry{
		EntryID:    "entry-1",
		TreasuryID: "treasury-a",
		EntryKind:  TreasuryLedgerEntryKindAcquisition,
		AssetCode:  "USD",
		Amount:     "1.00",
		CreatedAt:  now,
	}

	if err := StoreTreasuryLedgerEntry(root, entry); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry(first) error = %v", err)
	}
	if err := StoreTreasuryLedgerEntry(root, entry); err == nil || err.Error() != `mission store treasury ledger entry "entry-1" already exists` {
		t.Fatalf("StoreTreasuryLedgerEntry(second) error = %v, want append-only rejection", err)
	}
}

func TestTreasuryRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	secondTarget := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet-2",
	}
	writeFrankRegistryEligibilityFixture(t, root, secondTarget, EligibilityLabelAutonomyCompatible, "container-class-wallet-2", "check-container-class-wallet-2", now)
	secondContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-wallet-2",
		ContainerKind:        "wallet",
		Label:                "Secondary Wallet",
		ContainerClassID:     "container-class-wallet-2",
		State:                "active",
		EligibilityTargetRef: secondTarget,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, secondContainer); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing treasury id",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.TreasuryID = "   "
				}))
			},
			want: "mission store treasury treasury_id is required",
		},
		{
			name: "malformed treasury id",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.TreasuryID = "treasury/a"
				}))
			},
			want: `mission store treasury treasury_id "treasury/a" is invalid`,
		},
		{
			name: "malformed zero seed policy",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.ZeroSeedPolicy = TreasuryZeroSeedPolicy("owner_seed_allowed")
				}))
			},
			want: `mission store treasury zero_seed_policy "owner_seed_allowed" is invalid`,
		},
		{
			name: "malformed container ref",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.ContainerRefs = []FrankRegistryObjectRef{
						{
							Kind:     FrankRegistryObjectKindContainer,
							ObjectID: "   ",
						},
					}
				}))
			},
			want: "mission store treasury container_refs contain invalid ref: Frank object ref object_id is required",
		},
		{
			name: "non-container ref",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.ContainerRefs = []FrankRegistryObjectRef{
						{
							Kind:     FrankRegistryObjectKindIdentity,
							ObjectID: "identity-a",
						},
					}
				}))
			},
			want: `mission store treasury container_refs require kind "container", got "identity"`,
		},
		{
			name: "funded treasury without derivable active container id",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.State = TreasuryStateFunded
					record.ContainerRefs = nil
				}))
			},
			want: `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`,
		},
		{
			name: "active treasury with ambiguous active container id",
			run: func() error {
				return StoreTreasuryRecord(root, validTreasuryRecord(now, func(record *TreasuryRecord) {
					record.State = TreasuryStateActive
					record.ContainerRefs = []FrankRegistryObjectRef{
						{
							Kind:     FrankRegistryObjectKindContainer,
							ObjectID: fixtures.container.ContainerID,
						},
						{
							Kind:     FrankRegistryObjectKindContainer,
							ObjectID: secondContainer.ContainerID,
						},
					}
				}))
			},
			want: `mission store treasury state "active" requires exactly one active_container_id derivable from container_refs`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestTreasuryLedgerEntryValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 13, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing entry id",
			run: func() error {
				return StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
					entry.EntryID = "   "
				}))
			},
			want: "mission store treasury ledger entry entry_id is required",
		},
		{
			name: "missing treasury id",
			run: func() error {
				return StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
					entry.TreasuryID = "   "
				}))
			},
			want: "mission store treasury ledger entry treasury_id is required",
		},
		{
			name: "malformed entry kind",
			run: func() error {
				return StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
					entry.EntryKind = TreasuryLedgerEntryKind("spend")
				}))
			},
			want: `mission store treasury ledger entry entry_kind "spend" is invalid`,
		},
		{
			name: "malformed amount",
			run: func() error {
				return StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
					entry.Amount = "01.0"
				}))
			},
			want: `mission store treasury ledger entry amount "01.0" is invalid`,
		},
		{
			name: "malformed asset code",
			run: func() error {
				return StoreTreasuryLedgerEntry(root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
					entry.AssetCode = "US D"
				}))
			},
			want: `mission store treasury ledger entry asset_code "US D" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestTreasuryRecordUsesFrankRegistryContainerRefs(t *testing.T) {
	t.Parallel()

	recordType := reflect.TypeOf(TreasuryRecord{})
	field, ok := recordType.FieldByName("ContainerRefs")
	if !ok {
		t.Fatal("TreasuryRecord.ContainerRefs field missing")
	}
	if field.Type != reflect.TypeOf([]FrankRegistryObjectRef(nil)) {
		t.Fatalf("TreasuryRecord.ContainerRefs type = %v, want %v", field.Type, reflect.TypeOf([]FrankRegistryObjectRef(nil)))
	}
}

func TestTreasuryRecordRequiresExistingContainerLinks(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 13, 30, 0, 0, time.UTC)

	err := StoreTreasuryRecord(fixtures.root, validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-missing-container"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-missing",
			},
		}
	}))
	if err == nil {
		t.Fatal("StoreTreasuryRecord() error = nil, want missing container-link rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury container_refs ref kind "container" object_id "container-missing": resolve Frank object ref kind "container" object_id "container-missing": mission store Frank container record not found`) {
		t.Fatalf("StoreTreasuryRecord() error = %q, want missing container-link rejection", err.Error())
	}
}

func TestLoadTreasuryRecordFailsClosedWhenLinkedContainerIsMissing(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 13, 45, 0, 0, time.UTC)
	if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, "treasury-missing-container"), map[string]interface{}{
		"record_version":   StoreRecordVersion,
		"treasury_id":      "treasury-missing-container",
		"display_name":     "Frank Treasury Missing Container",
		"state":            string(TreasuryStateBootstrap),
		"zero_seed_policy": string(TreasuryZeroSeedPolicyOwnerSeedForbidden),
		"container_refs": []map[string]interface{}{
			{
				"kind":      string(FrankRegistryObjectKindContainer),
				"object_id": "container-missing",
			},
		},
		"created_at": now,
		"updated_at": now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := LoadTreasuryRecord(fixtures.root, "treasury-missing-container")
	if err == nil {
		t.Fatal("LoadTreasuryRecord() error = nil, want missing container-link rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury container_refs ref kind "container" object_id "container-missing": resolve Frank object ref kind "container" object_id "container-missing": mission store Frank container record not found`) {
		t.Fatalf("LoadTreasuryRecord() error = %q, want missing container-link rejection", err.Error())
	}
}

func TestLoadTreasuryRecordFailsClosedWhenFundedStateCannotDeriveActiveContainerID(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 13, 50, 0, 0, time.UTC)
	if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, "treasury-funded-missing-active-container"), map[string]interface{}{
		"record_version":   StoreRecordVersion,
		"treasury_id":      "treasury-funded-missing-active-container",
		"display_name":     "Frank Treasury Funded Missing Container",
		"state":            string(TreasuryStateFunded),
		"zero_seed_policy": string(TreasuryZeroSeedPolicyOwnerSeedForbidden),
		"created_at":       now,
		"updated_at":       now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := LoadTreasuryRecord(fixtures.root, "treasury-funded-missing-active-container")
	if err == nil {
		t.Fatal("LoadTreasuryRecord() error = nil, want active-container derivation rejection")
	}
	if !strings.Contains(err.Error(), `mission store treasury state "funded" requires exactly one active_container_id derivable from container_refs`) {
		t.Fatalf("LoadTreasuryRecord() error = %q, want active-container derivation rejection", err.Error())
	}
}

func TestTreasuryZeroSeedPolicyOwnerSeedForbiddenIsStable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 14, 0, 0, 0, time.UTC)

	if err := StoreTreasuryRecord(root, TreasuryRecord{
		TreasuryID:     "treasury-policy",
		DisplayName:    "Policy Treasury",
		State:          TreasuryStateUnfunded,
		ZeroSeedPolicy: TreasuryZeroSeedPolicy(" owner_seed_forbidden "),
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	record, err := LoadTreasuryRecord(root, "treasury-policy")
	if err != nil {
		t.Fatalf("LoadTreasuryRecord() error = %v", err)
	}
	if record.ZeroSeedPolicy != TreasuryZeroSeedPolicyOwnerSeedForbidden {
		t.Fatalf("ZeroSeedPolicy = %q, want %q", record.ZeroSeedPolicy, TreasuryZeroSeedPolicyOwnerSeedForbidden)
	}
}

func TestTreasuryRefUsesSingleStepControlPlaneSurface(t *testing.T) {
	t.Parallel()

	stepType := reflect.TypeOf(Step{})
	treasuryRefField, ok := stepType.FieldByName("TreasuryRef")
	if !ok {
		t.Fatal("Step.TreasuryRef field missing")
	}
	if treasuryRefField.Type != reflect.TypeOf((*TreasuryRef)(nil)) {
		t.Fatalf("Step.TreasuryRef type = %v, want %v", treasuryRefField.Type, reflect.TypeOf((*TreasuryRef)(nil)))
	}

	executionContextType := reflect.TypeOf(ExecutionContext{})
	if _, ok := executionContextType.FieldByName("TreasuryRef"); ok {
		t.Fatal("ExecutionContext.TreasuryRef field present, want no duplicate top-level treasury-ref source")
	}
	if _, ok := executionContextType.FieldByName("TreasuryID"); ok {
		t.Fatal("ExecutionContext.TreasuryID field present, want no duplicate top-level treasury identity source")
	}
}

func TestResolveExecutionContextTreasuryRefZeroRefPathPreservesPriorBehavior(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.TreasuryRef != nil {
		t.Fatalf("ResolveExecutionContext().Step.TreasuryRef = %#v, want nil", ec.Step.TreasuryRef)
	}

	got, err := ResolveExecutionContextTreasuryRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryRef() error = %v", err)
	}
	if got != nil {
		t.Fatalf("ResolveExecutionContextTreasuryRef() = %#v, want nil for zero-treasury step", got)
	}
}

func TestResolveExecutionContextTreasuryRefResolvesActiveTreasuryRef(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 15, 0, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-active"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})
	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	record.RecordVersion = StoreRecordVersion

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: " treasury-active ",
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	got, err := ResolveExecutionContextTreasuryRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryRef() error = %v", err)
	}
	if got == nil {
		t.Fatal("ResolveExecutionContextTreasuryRef() = nil, want resolved treasury record")
	}
	if !reflect.DeepEqual(*got, record) {
		t.Fatalf("ResolveExecutionContextTreasuryRef() = %#v, want %#v", *got, record)
	}
}

func TestResolveExecutionContextTreasuryRefFailsClosedOnMissingOrMalformedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name string
		ref  *TreasuryRef
		want string
	}{
		{
			name: "missing record",
			ref: &TreasuryRef{
				TreasuryID: "treasury-missing",
			},
			want: ErrTreasuryRecordNotFound.Error(),
		},
		{
			name: "empty treasury id",
			ref: &TreasuryRef{
				TreasuryID: "   ",
			},
			want: "treasury_id is required",
		},
		{
			name: "malformed treasury id",
			ref: &TreasuryRef{
				TreasuryID: "treasury/malformed",
			},
			want: `treasury_id "treasury/malformed" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ec := testExecutionContext()
			ec.Step.TreasuryRef = tc.ref
			ec.MissionStoreRoot = root

			_, err := ResolveExecutionContextTreasuryRef(ec)
			if err == nil {
				t.Fatal("ResolveExecutionContextTreasuryRef() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ResolveExecutionContextTreasuryRef() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveExecutionContextTreasuryRefFailsClosedWithoutMissionStoreRoot(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.TreasuryRef = &TreasuryRef{
		TreasuryID: "treasury-1",
	}

	_, err := ResolveExecutionContextTreasuryRef(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextTreasuryRef() error = nil, want missing mission store root rejection")
	}
	if !strings.Contains(err.Error(), "mission store root is required to resolve treasury refs") {
		t.Fatalf("ResolveExecutionContextTreasuryRef() error = %q, want missing mission store root rejection", err.Error())
	}
}

func TestResolveExecutionContextTreasuryRefDoesNotIntroduceCampaignIdentityOrObjectSideChannel(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 16, 0, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-sidechannel"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})
	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].IdentityMode = IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].GovernedExternalTargets = []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	job.Plan.Steps[0].FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
	}
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: "campaign-1",
	}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: record.TreasuryID,
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	got, err := ResolveExecutionContextTreasuryRef(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryRef() error = %v", err)
	}
	if got == nil {
		t.Fatal("ResolveExecutionContextTreasuryRef() = nil, want resolved treasury record")
	}
	if ec.Step.IdentityMode != IdentityModeOwnerOnlyControl {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeOwnerOnlyControl)
	}
	if len(ec.GovernedExternalTargets) != 1 || ec.GovernedExternalTargets[0].RegistryID != "provider-mail" {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want step-owned target only", ec.GovernedExternalTargets)
	}
	if len(ec.Step.FrankObjectRefs) != 1 || ec.Step.FrankObjectRefs[0].ObjectID != "identity-mail" {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want step-owned ref only", ec.Step.FrankObjectRefs)
	}
	if ec.Step.CampaignRef == nil || ec.Step.CampaignRef.CampaignID != "campaign-1" {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want step-owned campaign only", ec.Step.CampaignRef)
	}
	if len(got.ContainerRefs) != 1 || got.ContainerRefs[0].ObjectID != fixtures.container.ContainerID {
		t.Fatalf("ResolveExecutionContextTreasuryRef().ContainerRefs = %#v, want treasury-owned container only", got.ContainerRefs)
	}
}

func TestResolveExecutionContextTreasuryPreflightUsesSingleReadOnlySurface(t *testing.T) {
	t.Parallel()

	preflightType := reflect.TypeOf(ResolvedExecutionContextTreasuryPreflight{})
	if preflightType.NumField() != 2 {
		t.Fatalf("ResolvedExecutionContextTreasuryPreflight field count = %d, want 2", preflightType.NumField())
	}

	treasuryField, ok := preflightType.FieldByName("Treasury")
	if !ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.Treasury field missing")
	}
	if treasuryField.Type != reflect.TypeOf((*TreasuryRecord)(nil)) {
		t.Fatalf("ResolvedExecutionContextTreasuryPreflight.Treasury type = %v, want %v", treasuryField.Type, reflect.TypeOf((*TreasuryRecord)(nil)))
	}

	containersField, ok := preflightType.FieldByName("Containers")
	if !ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.Containers field missing")
	}
	if containersField.Type != reflect.TypeOf([]FrankContainerRecord(nil)) {
		t.Fatalf("ResolvedExecutionContextTreasuryPreflight.Containers type = %v, want %v", containersField.Type, reflect.TypeOf([]FrankContainerRecord(nil)))
	}

	if _, ok := preflightType.FieldByName("IdentityMode"); ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.IdentityMode field present, want no duplicate identity-mode source")
	}
	if _, ok := preflightType.FieldByName("GovernedExternalTargets"); ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.GovernedExternalTargets field present, want no duplicate eligibility source")
	}
	if _, ok := preflightType.FieldByName("CampaignRef"); ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.CampaignRef field present, want no duplicate campaign source")
	}
	if _, ok := preflightType.FieldByName("TreasuryID"); ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.TreasuryID field present, want no duplicate treasury identity source")
	}
	if _, ok := preflightType.FieldByName("ContainerRefs"); ok {
		t.Fatal("ResolvedExecutionContextTreasuryPreflight.ContainerRefs field present, want no duplicate object-ref source")
	}
}

func TestResolveExecutionContextTreasuryPreflightZeroRefPathPreservesPriorBehavior(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	got, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %v", err)
	}
	if !reflect.DeepEqual(got, ResolvedExecutionContextTreasuryPreflight{}) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() = %#v, want zero value for zero-treasury step", got)
	}
}

func TestResolveExecutionContextTreasuryPreflightResolvesTreasuryAndContainers(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 17, 0, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-preflight"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKind(" container "),
				ObjectID: " " + fixtures.container.ContainerID + " ",
			},
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	record.RecordVersion = StoreRecordVersion
	record.ContainerRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: fixtures.container.ContainerID,
		},
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: " treasury-preflight "}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = fixtures.root

	got, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %v", err)
	}
	if got.Treasury == nil || !reflect.DeepEqual(*got.Treasury, record) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Treasury = %#v, want %#v", got.Treasury, record)
	}
	if len(got.Containers) != 1 || !reflect.DeepEqual(got.Containers[0], fixtures.container) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Containers = %#v, want [%#v]", got.Containers, fixtures.container)
	}
}

func TestResolveExecutionContextTreasuryPreflightFailsClosedWithoutMissionStoreRoot(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.TreasuryRef = &TreasuryRef{
		TreasuryID: "treasury-1",
	}

	_, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextTreasuryPreflight() error = nil, want missing mission store root rejection")
	}
	if !strings.Contains(err.Error(), "mission store root is required to resolve treasury refs") {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %q, want missing mission store root rejection", err.Error())
	}
}

func TestResolveExecutionContextTreasuryPreflightFailsClosedOnMissingOrMalformedTreasuryRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name string
		ref  *TreasuryRef
		want string
	}{
		{
			name: "missing record",
			ref: &TreasuryRef{
				TreasuryID: "treasury-missing",
			},
			want: ErrTreasuryRecordNotFound.Error(),
		},
		{
			name: "empty treasury id",
			ref: &TreasuryRef{
				TreasuryID: "   ",
			},
			want: "treasury_id is required",
		},
		{
			name: "malformed treasury id",
			ref: &TreasuryRef{
				TreasuryID: "treasury/malformed",
			},
			want: `treasury_id "treasury/malformed" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ec := testExecutionContext()
			ec.Step.TreasuryRef = tc.ref
			ec.MissionStoreRoot = root

			_, err := ResolveExecutionContextTreasuryPreflight(ec)
			if err == nil {
				t.Fatal("ResolveExecutionContextTreasuryPreflight() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveExecutionContextTreasuryPreflightFailsClosedOnMalformedTreasuryRecord(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 8, 18, 0, 0, 0, time.UTC)
	if err := WriteStoreJSONAtomic(StoreTreasuryPath(root, "treasury-bad"), map[string]interface{}{
		"record_version":   StoreRecordVersion,
		"treasury_id":      "treasury-bad",
		"display_name":     "",
		"state":            string(TreasuryStateBootstrap),
		"zero_seed_policy": string(TreasuryZeroSeedPolicyOwnerSeedForbidden),
		"container_refs": []map[string]interface{}{
			{
				"kind":      string(FrankRegistryObjectKindContainer),
				"object_id": fixtures.container.ContainerID,
			},
		},
		"created_at": now,
		"updated_at": now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	ec := testExecutionContext()
	ec.Step.TreasuryRef = &TreasuryRef{TreasuryID: "treasury-bad"}
	ec.MissionStoreRoot = root

	_, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextTreasuryPreflight() error = nil, want malformed treasury record rejection")
	}
	if !strings.Contains(err.Error(), "display_name is required") {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %q, want malformed treasury record rejection", err.Error())
	}
}

func TestResolveExecutionContextTreasuryPreflightFailsClosedOnMissingOrMalformedTreasuryContainerRefs(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 19, 0, 0, 0, time.UTC)
	badContainer := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-bad",
		ContainerKind:        "wallet",
		Label:                "",
		ContainerClassID:     "container-class-wallet",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := WriteStoreJSONAtomic(StoreFrankContainerPath(fixtures.root, badContainer.ContainerID), badContainer); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	assertPreflightError := func(t *testing.T, treasuryID, want string) {
		t.Helper()

		ec := testExecutionContext()
		ec.Step.TreasuryRef = &TreasuryRef{TreasuryID: treasuryID}
		ec.MissionStoreRoot = fixtures.root

		_, err := ResolveExecutionContextTreasuryPreflight(ec)
		if err == nil {
			t.Fatal("ResolveExecutionContextTreasuryPreflight() error = nil, want fail-closed rejection")
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %q, want substring %q", err.Error(), want)
		}
	}

	t.Run("missing container record", func(t *testing.T) {
		t.Parallel()

		record := validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-missing-container-record"
			record.ContainerRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: "missing-container",
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, record.TreasuryID), map[string]interface{}{
			"record_version":   StoreRecordVersion,
			"treasury_id":      record.TreasuryID,
			"display_name":     record.DisplayName,
			"state":            string(record.State),
			"zero_seed_policy": string(record.ZeroSeedPolicy),
			"container_refs": []map[string]interface{}{
				{
					"kind":      string(record.ContainerRefs[0].Kind),
					"object_id": record.ContainerRefs[0].ObjectID,
				},
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		assertPreflightError(t, record.TreasuryID, ErrFrankContainerRecordNotFound.Error())
	})

	t.Run("malformed stored container record", func(t *testing.T) {
		t.Parallel()

		record := validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-malformed-container-record"
			record.ContainerRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: badContainer.ContainerID,
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, record.TreasuryID), map[string]interface{}{
			"record_version":   StoreRecordVersion,
			"treasury_id":      record.TreasuryID,
			"display_name":     record.DisplayName,
			"state":            string(record.State),
			"zero_seed_policy": string(record.ZeroSeedPolicy),
			"container_refs": []map[string]interface{}{
				{
					"kind":      string(record.ContainerRefs[0].Kind),
					"object_id": record.ContainerRefs[0].ObjectID,
				},
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		assertPreflightError(t, record.TreasuryID, "label is required")
	})

	t.Run("empty container object id", func(t *testing.T) {
		t.Parallel()

		record := validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-empty-container-object-id"
			record.ContainerRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: "   ",
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, record.TreasuryID), map[string]interface{}{
			"record_version":   StoreRecordVersion,
			"treasury_id":      record.TreasuryID,
			"display_name":     record.DisplayName,
			"state":            string(record.State),
			"zero_seed_policy": string(record.ZeroSeedPolicy),
			"container_refs": []map[string]interface{}{
				{
					"kind":      string(record.ContainerRefs[0].Kind),
					"object_id": record.ContainerRefs[0].ObjectID,
				},
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		assertPreflightError(t, record.TreasuryID, "Frank object ref object_id is required")
	})

	t.Run("non-container ref kind", func(t *testing.T) {
		t.Parallel()

		record := validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-non-container-ref-kind"
			record.ContainerRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: fixtures.identity.IdentityID,
				},
			}
		})
		if err := WriteStoreJSONAtomic(StoreTreasuryPath(fixtures.root, record.TreasuryID), map[string]interface{}{
			"record_version":   StoreRecordVersion,
			"treasury_id":      record.TreasuryID,
			"display_name":     record.DisplayName,
			"state":            string(record.State),
			"zero_seed_policy": string(record.ZeroSeedPolicy),
			"container_refs": []map[string]interface{}{
				{
					"kind":      string(record.ContainerRefs[0].Kind),
					"object_id": record.ContainerRefs[0].ObjectID,
				},
			},
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		}); err != nil {
			t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
		}

		assertPreflightError(t, record.TreasuryID, `mission store treasury container_refs require kind "container", got "identity"`)
	})
}

func TestTreasuryLedgerEntryRequiresExistingTreasuryWithSingleLinkedContainer(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 19, 30, 0, 0, time.UTC)

	t.Run("missing treasury record", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		err := StoreTreasuryLedgerEntry(fixtures.root, validTreasuryLedgerEntry(now, func(entry *TreasuryLedgerEntry) {
			entry.EntryID = "entry-missing-treasury"
			entry.TreasuryID = "treasury-missing"
		}))
		if err == nil {
			t.Fatal("StoreTreasuryLedgerEntry() error = nil, want missing treasury rejection")
		}
		if !strings.Contains(err.Error(), `mission store treasury ledger entry treasury_id "treasury-missing": mission store treasury record not found`) {
			t.Fatalf("StoreTreasuryLedgerEntry() error = %q, want missing treasury rejection", err.Error())
		}
	})

	t.Run("treasury without active container", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		if err := StoreTreasuryRecord(fixtures.root, validTreasuryRecord(now, func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-no-container"
			record.ContainerRefs = nil
		})); err != nil {
			t.Fatalf("StoreTreasuryRecord() error = %v", err)
		}

		err := StoreTreasuryLedgerEntry(fixtures.root, validTreasuryLedgerEntry(now.Add(time.Minute), func(entry *TreasuryLedgerEntry) {
			entry.EntryID = "entry-no-container"
			entry.TreasuryID = "treasury-no-container"
		}))
		if err == nil {
			t.Fatal("StoreTreasuryLedgerEntry() error = nil, want missing active-container rejection")
		}
		if !strings.Contains(err.Error(), `mission store treasury ledger entry treasury_id "treasury-no-container" has no active treasury container`) {
			t.Fatalf("StoreTreasuryLedgerEntry() error = %q, want missing active-container rejection", err.Error())
		}
	})

	t.Run("treasury with ambiguous active container", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		target := AutonomyEligibilityTargetRef{
			Kind:       EligibilityTargetKindTreasuryContainerClass,
			RegistryID: "container-class-wallet-2",
		}
		writeFrankRegistryEligibilityFixture(t, fixtures.root, target, EligibilityLabelAutonomyCompatible, "container-class-wallet-2", "check-container-class-wallet-2", now)
		container2 := FrankContainerRecord{
			RecordVersion:        StoreRecordVersion,
			ContainerID:          "container-wallet-2",
			ContainerKind:        "wallet",
			Label:                "Secondary Wallet",
			ContainerClassID:     "container-class-wallet-2",
			State:                "active",
			EligibilityTargetRef: target,
			CreatedAt:            now.UTC(),
			UpdatedAt:            now.Add(time.Minute).UTC(),
		}
		if err := StoreFrankContainerRecord(fixtures.root, container2); err != nil {
			t.Fatalf("StoreFrankContainerRecord() error = %v", err)
		}
		if err := StoreTreasuryRecord(fixtures.root, validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
			record.TreasuryID = "treasury-ambiguous-container"
			record.ContainerRefs = []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: fixtures.container.ContainerID,
				},
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: container2.ContainerID,
				},
			}
		})); err != nil {
			t.Fatalf("StoreTreasuryRecord() error = %v", err)
		}

		err := StoreTreasuryLedgerEntry(fixtures.root, validTreasuryLedgerEntry(now.Add(3*time.Minute), func(entry *TreasuryLedgerEntry) {
			entry.EntryID = "entry-ambiguous-container"
			entry.TreasuryID = "treasury-ambiguous-container"
		}))
		if err == nil {
			t.Fatal("StoreTreasuryLedgerEntry() error = nil, want ambiguous active-container rejection")
		}
		if !strings.Contains(err.Error(), `mission store treasury ledger entry treasury_id "treasury-ambiguous-container" has ambiguous active treasury container across 2 container_refs`) {
			t.Fatalf("StoreTreasuryLedgerEntry() error = %q, want ambiguous active-container rejection", err.Error())
		}
	})
}

func TestTreasuryObjectViewsAdaptStorageFieldsWithoutMigration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 19, 45, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-view"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
		}
	})

	view := record.AsObjectView()
	if view.ActiveContainerID != "container-wallet" || view.LedgerRef != "treasury-view" || view.State != record.State {
		t.Fatalf("TreasuryRecord.AsObjectView() = %#v, want active_container_id/ledger_ref adapter", view)
	}
	if view.CustodyModel != TreasuryCustodyModelFrankContainerRegistry {
		t.Fatalf("TreasuryRecord.AsObjectView().CustodyModel = %q, want %q", view.CustodyModel, TreasuryCustodyModelFrankContainerRegistry)
	}
	if !reflect.DeepEqual(view.ForbiddenTransactionClasses, []TreasuryTransactionClass{
		TreasuryTransactionClassAllocate,
		TreasuryTransactionClassSave,
		TreasuryTransactionClassSpend,
		TreasuryTransactionClassReinvest,
	}) {
		t.Fatalf("TreasuryRecord.AsObjectView().ForbiddenTransactionClasses = %#v, want default non-active policy envelope", view.ForbiddenTransactionClasses)
	}
	if view.PermittedTransactionClasses != nil {
		t.Fatalf("TreasuryRecord.AsObjectView().PermittedTransactionClasses = %#v, want nil for non-active treasury", view.PermittedTransactionClasses)
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	if err := StoreTreasuryRecord(fixtures.root, validTreasuryRecord(now.Add(time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-view"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	entryView, err := ResolveTreasuryLedgerEntryObjectView(fixtures.root, validTreasuryLedgerEntry(now.Add(2*time.Minute), func(entry *TreasuryLedgerEntry) {
		entry.EntryID = "entry-view"
		entry.TreasuryID = "treasury-view"
		entry.EntryKind = TreasuryLedgerEntryKindMovement
		entry.AssetCode = "USDC"
		entry.Amount = "42.00"
		entry.SourceRef = "campaign:community-a"
	}))
	if err != nil {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() error = %v", err)
	}
	if entryView.ContainerID != fixtures.container.ContainerID || entryView.EntryClass != TreasuryLedgerEntryKindMovement || entryView.Asset != "USDC" || entryView.Source != "campaign:community-a" {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() = %#v, want canonical ledger contract fields", entryView)
	}
	if entryView.Direction != TreasuryLedgerDirectionInternal || entryView.Status != TreasuryLedgerStatusRecorded {
		t.Fatalf("ResolveTreasuryLedgerEntryObjectView() direction/status = (%q, %q), want (%q, %q)", entryView.Direction, entryView.Status, TreasuryLedgerDirectionInternal, TreasuryLedgerStatusRecorded)
	}
}

func TestActiveTreasuryObjectViewUsesDefaultActiveTransactionPolicyEnvelope(t *testing.T) {
	t.Parallel()

	record := TreasuryRecord{
		RecordVersion:  StoreRecordVersion,
		TreasuryID:     "treasury-active-view",
		DisplayName:    "Frank Treasury Active",
		State:          TreasuryStateActive,
		ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
		},
		CreatedAt: time.Date(2026, 4, 8, 20, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 8, 20, 31, 0, 0, time.UTC),
	}

	view := record.AsObjectView()
	if !reflect.DeepEqual(view.PermittedTransactionClasses, []TreasuryTransactionClass{
		TreasuryTransactionClassAllocate,
		TreasuryTransactionClassSave,
		TreasuryTransactionClassSpend,
		TreasuryTransactionClassReinvest,
	}) {
		t.Fatalf("TreasuryRecord.AsObjectView().PermittedTransactionClasses = %#v, want default active policy envelope", view.PermittedTransactionClasses)
	}
	if view.ForbiddenTransactionClasses != nil {
		t.Fatalf("TreasuryRecord.AsObjectView().ForbiddenTransactionClasses = %#v, want nil for active treasury", view.ForbiddenTransactionClasses)
	}
}

func TestResolveTreasuryLedgerEntryDirectionMapsEntryKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind TreasuryLedgerEntryKind
		want TreasuryLedgerDirection
	}{
		{kind: TreasuryLedgerEntryKindAcquisition, want: TreasuryLedgerDirectionInflow},
		{kind: TreasuryLedgerEntryKindMovement, want: TreasuryLedgerDirectionInternal},
		{kind: TreasuryLedgerEntryKindDisposition, want: TreasuryLedgerDirectionOutflow},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()
			got := ResolveTreasuryLedgerEntryDirection(TreasuryLedgerEntry{EntryKind: tc.kind})
			if got != tc.want {
				t.Fatalf("ResolveTreasuryLedgerEntryDirection(%q) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestResolveExecutionContextTreasuryPreflightDoesNotIntroduceEligibilityIdentityModeCampaignOrObjectSideChannel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 20, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-human-wallet",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-human-wallet", "check-container-class-human-wallet", now)

	container := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-human-wallet",
		ContainerKind:        "wallet",
		Label:                "Human Wallet",
		ContainerClassID:     "container-class-human-wallet",
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	record := validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-preflight-sidechannel"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: container.ContainerID,
			},
		}
	})
	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	record.RecordVersion = StoreRecordVersion

	job := testExecutionJob()
	job.Plan.Steps[0].IdentityMode = IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].GovernedExternalTargets = []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	job.Plan.Steps[0].FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
	}
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: "campaign-1",
	}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: record.TreasuryID,
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	got, err := ResolveExecutionContextTreasuryPreflight(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight() error = %v", err)
	}
	if got.Treasury == nil || !reflect.DeepEqual(*got.Treasury, record) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Treasury = %#v, want %#v", got.Treasury, record)
	}
	if len(got.Containers) != 1 || !reflect.DeepEqual(got.Containers[0], container) {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Containers = %#v, want [%#v]", got.Containers, container)
	}
	if got.Containers[0].EligibilityTargetRef != target {
		t.Fatalf("ResolveExecutionContextTreasuryPreflight().Containers[0].EligibilityTargetRef = %#v, want %#v", got.Containers[0].EligibilityTargetRef, target)
	}
	if ec.Step.IdentityMode != IdentityModeOwnerOnlyControl {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeOwnerOnlyControl)
	}
	if len(ec.GovernedExternalTargets) != 1 || ec.GovernedExternalTargets[0].RegistryID != "provider-mail" {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want step-owned target only", ec.GovernedExternalTargets)
	}
	if len(ec.Step.FrankObjectRefs) != 1 || ec.Step.FrankObjectRefs[0].ObjectID != "identity-mail" {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want step-owned ref only", ec.Step.FrankObjectRefs)
	}
	if ec.Step.CampaignRef == nil || ec.Step.CampaignRef.CampaignID != "campaign-1" {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want step-owned campaign only", ec.Step.CampaignRef)
	}
	if ec.Step.TreasuryRef == nil || ec.Step.TreasuryRef.TreasuryID != record.TreasuryID {
		t.Fatalf("ResolveExecutionContext().Step.TreasuryRef = %#v, want step-owned treasury ref only", ec.Step.TreasuryRef)
	}
}

func validTreasuryRecord(now time.Time, mutate func(*TreasuryRecord)) TreasuryRecord {
	record := TreasuryRecord{
		TreasuryID:     "treasury-a",
		DisplayName:    "Frank Treasury",
		State:          TreasuryStateBootstrap,
		ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
		},
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func validTreasuryLedgerEntry(now time.Time, mutate func(*TreasuryLedgerEntry)) TreasuryLedgerEntry {
	entry := TreasuryLedgerEntry{
		EntryID:    "entry-a",
		TreasuryID: "treasury-a",
		EntryKind:  TreasuryLedgerEntryKindAcquisition,
		AssetCode:  "USD",
		Amount:     "10.00",
		CreatedAt:  now,
		SourceRef:  "payout:listing-1",
	}
	if mutate != nil {
		mutate(&entry)
	}
	return entry
}
