package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

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
		record.State = TreasuryStateActive
		record.PostBootstrapAcquisition = &TreasuryPostBootstrapAcquisition{
			AssetCode:       "USD",
			Amount:          "2.25",
			SourceRef:       "payout:listing-b",
			EvidenceLocator: "https://evidence.example/payout-b",
			ConfirmedAt:     now.Add(time.Minute),
			ConsumedEntryID: "entry-post-value",
		}
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
	record.PostBootstrapAcquisition = &TreasuryPostBootstrapAcquisition{
		AssetCode:       "USD",
		Amount:          "2.25",
		SourceRef:       "payout:listing-b",
		EvidenceLocator: "https://evidence.example/payout-b",
		ConfirmedAt:     now.Add(time.Minute).UTC(),
		ConsumedEntryID: "entry-post-value",
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
