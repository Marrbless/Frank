package missioncontrol

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFrankIdentityRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-b",
	}, EligibilityLabelAutonomyCompatible, "provider-b.example", "check-provider-b", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-a",
	}, EligibilityLabelAutonomyCompatible, "provider-a.example", "check-provider-a", now.Add(time.Minute))

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-b",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail B",
		ProviderOrPlatformID: "provider-b",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-b"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord(identity-b) error = %v", err)
	}

	want := FrankIdentityRecord{
		IdentityID:           "identity-a",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail A",
		ProviderOrPlatformID: "provider-a",
		IdentityMode:         IdentityMode(" "),
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-a"},
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}
	if err := StoreFrankIdentityRecord(root, want); err != nil {
		t.Fatalf("StoreFrankIdentityRecord(identity-a) error = %v", err)
	}

	got, err := LoadFrankIdentityRecord(root, "identity-a")
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.IdentityMode = IdentityModeAgentAlias
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFrankIdentityRecord() = %#v, want %#v", got, want)
	}

	records, err := ListFrankIdentityRecords(root)
	if err != nil {
		t.Fatalf("ListFrankIdentityRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListFrankIdentityRecords() len = %d, want 2", len(records))
	}
	if records[0].IdentityID != "identity-a" || records[1].IdentityID != "identity-b" {
		t.Fatalf("ListFrankIdentityRecords() ids = [%q %q], want [identity-a identity-b]", records[0].IdentityID, records[1].IdentityID)
	}
}

func TestFrankAccountRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.FixedZone("offset", 2*60*60))

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-b",
	}, EligibilityLabelAutonomyCompatible, "account-class-b", "check-account-class-b", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-a",
	}, EligibilityLabelAutonomyCompatible, "account-class-a", "check-account-class-a", now.Add(time.Minute))

	if err := StoreFrankAccountRecord(root, FrankAccountRecord{
		AccountID:            "account-b",
		AccountKind:          "mailbox",
		Label:                "Inbox B",
		ProviderOrPlatformID: "provider-b",
		IdentityID:           "identity-b",
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-b"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankAccountRecord(account-b) error = %v", err)
	}

	want := FrankAccountRecord{
		AccountID:            "account-a",
		AccountKind:          "mailbox",
		Label:                "Inbox A",
		ProviderOrPlatformID: "provider-a",
		IdentityID:           "identity-a",
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-a"},
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}
	if err := StoreFrankAccountRecord(root, want); err != nil {
		t.Fatalf("StoreFrankAccountRecord(account-a) error = %v", err)
	}

	got, err := LoadFrankAccountRecord(root, "account-a")
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFrankAccountRecord() = %#v, want %#v", got, want)
	}

	records, err := ListFrankAccountRecords(root)
	if err != nil {
		t.Fatalf("ListFrankAccountRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListFrankAccountRecords() len = %d, want 2", len(records))
	}
	if records[0].AccountID != "account-a" || records[1].AccountID != "account-b" {
		t.Fatalf("ListFrankAccountRecords() ids = [%q %q], want [account-a account-b]", records[0].AccountID, records[1].AccountID)
	}
}

func TestFrankContainerRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 16, 0, 0, 0, time.FixedZone("offset", -7*60*60))

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-b",
	}, EligibilityLabelAutonomyCompatible, "container-class-b", "check-container-class-b", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-a",
	}, EligibilityLabelAutonomyCompatible, "container-class-a", "check-container-class-a", now.Add(time.Minute))

	if err := StoreFrankContainerRecord(root, FrankContainerRecord{
		ContainerID:          "container-b",
		ContainerKind:        "wallet",
		Label:                "Wallet B",
		ContainerClassID:     "container-class-b",
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-b"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankContainerRecord(container-b) error = %v", err)
	}

	want := FrankContainerRecord{
		ContainerID:          "container-a",
		ContainerKind:        "wallet",
		Label:                "Wallet A",
		ContainerClassID:     "container-class-a",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-a"},
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}
	if err := StoreFrankContainerRecord(root, want); err != nil {
		t.Fatalf("StoreFrankContainerRecord(container-a) error = %v", err)
	}

	got, err := LoadFrankContainerRecord(root, "container-a")
	if err != nil {
		t.Fatalf("LoadFrankContainerRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFrankContainerRecord() = %#v, want %#v", got, want)
	}

	records, err := ListFrankContainerRecords(root)
	if err != nil {
		t.Fatalf("ListFrankContainerRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListFrankContainerRecords() len = %d, want 2", len(records))
	}
	if records[0].ContainerID != "container-a" || records[1].ContainerID != "container-b" {
		t.Fatalf("ListFrankContainerRecords() ids = [%q %q], want [container-a container-b]", records[0].ContainerID, records[1].ContainerID)
	}
}

func TestFrankRegistryMalformedValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 18, 0, 0, 0, time.UTC)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "identity invalid mode",
			run: func() error {
				return StoreFrankIdentityRecord(root, FrankIdentityRecord{
					IdentityID:           "identity-1",
					IdentityKind:         "email",
					DisplayName:          "Frank Mail",
					ProviderOrPlatformID: "provider-mail",
					IdentityMode:         IdentityMode("owner-ish"),
					State:                "active",
					EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
					CreatedAt:            now,
					UpdatedAt:            now.Add(time.Minute),
				})
			},
			want: `identity_mode "owner-ish" is invalid`,
		},
		{
			name: "account missing recovery model",
			run: func() error {
				return StoreFrankAccountRecord(root, FrankAccountRecord{
					AccountID:            "account-1",
					AccountKind:          "mailbox",
					Label:                "Inbox",
					ProviderOrPlatformID: "provider-mail",
					IdentityID:           "identity-1",
					ControlModel:         "agent_managed",
					State:                "active",
					EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
					CreatedAt:            now,
					UpdatedAt:            now.Add(time.Minute),
				})
			},
			want: "mission store Frank account recovery_model is required",
		},
		{
			name: "container unknown linkage",
			run: func() error {
				return StoreFrankContainerRecord(root, FrankContainerRecord{
					ContainerID:          "container-1",
					ContainerKind:        "wallet",
					Label:                "Wallet",
					ContainerClassID:     "missing-container-class",
					State:                "active",
					EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "missing-container-class"},
					CreatedAt:            now,
					UpdatedAt:            now.Add(time.Minute),
				})
			},
			want: `mission store frank registry eligibility_target_ref "missing-container-class" has no linked eligibility registry record`,
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

func TestFrankRegistryDuplicateIDsOverwriteThroughStoreHelper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 20, 0, 0, 0, time.UTC)

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}, EligibilityLabelAutonomyCompatible, "account-class-mailbox", "check-account-class-mailbox", now.Add(time.Minute))
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}, EligibilityLabelAutonomyCompatible, "container-class-wallet", "check-container-class-wallet", now.Add(2*time.Minute))

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-1",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail Old",
		ProviderOrPlatformID: "provider-mail",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord(first) error = %v", err)
	}
	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-1",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail New",
		ProviderOrPlatformID: "provider-mail",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord(second) error = %v", err)
	}

	if err := StoreFrankAccountRecord(root, FrankAccountRecord{
		AccountID:            "account-1",
		AccountKind:          "mailbox",
		Label:                "Inbox Old",
		ProviderOrPlatformID: "provider-mail",
		IdentityID:           "identity-1",
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankAccountRecord(first) error = %v", err)
	}
	if err := StoreFrankAccountRecord(root, FrankAccountRecord{
		AccountID:            "account-1",
		AccountKind:          "mailbox",
		Label:                "Inbox New",
		ProviderOrPlatformID: "provider-mail",
		IdentityID:           "identity-1",
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankAccountRecord(second) error = %v", err)
	}

	if err := StoreFrankContainerRecord(root, FrankContainerRecord{
		ContainerID:          "container-1",
		ContainerKind:        "wallet",
		Label:                "Wallet Old",
		ContainerClassID:     "container-class-wallet",
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankContainerRecord(first) error = %v", err)
	}
	if err := StoreFrankContainerRecord(root, FrankContainerRecord{
		ContainerID:          "container-1",
		ContainerKind:        "wallet",
		Label:                "Wallet New",
		ContainerClassID:     "container-class-wallet",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankContainerRecord(second) error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(root, "identity-1")
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	if identity.DisplayName != "Frank Mail New" || identity.State != "active" {
		t.Fatalf("LoadFrankIdentityRecord() = %#v, want overwritten identity", identity)
	}

	account, err := LoadFrankAccountRecord(root, "account-1")
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if account.Label != "Inbox New" || account.State != "active" {
		t.Fatalf("LoadFrankAccountRecord() = %#v, want overwritten account", account)
	}

	container, err := LoadFrankContainerRecord(root, "container-1")
	if err != nil {
		t.Fatalf("LoadFrankContainerRecord() error = %v", err)
	}
	if container.Label != "Wallet New" || container.State != "active" {
		t.Fatalf("LoadFrankContainerRecord() = %#v, want overwritten container", container)
	}
}

func TestFrankRegistryEligibilityLinkUsesLandedRegistryAsSingleSourceOfTruth(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 22, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "provider-human-id.example", "check-provider-human-id", now)

	result, err := ValidateFrankRegistryEligibilityLink(root, target)
	if err != nil {
		t.Fatalf("ValidateFrankRegistryEligibilityLink() error = %v", err)
	}
	if result.Decision != AutonomyEligibilityDecisionIneligible {
		t.Fatalf("ValidateFrankRegistryEligibilityLink().Decision = %q, want %q", result.Decision, AutonomyEligibilityDecisionIneligible)
	}

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-human-id",
		IdentityKind:         "email",
		DisplayName:          "Human-ID Candidate",
		ProviderOrPlatformID: target.RegistryID,
		IdentityMode:         IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	if _, err := RequireAutonomyEligibleTarget(root, target); !errors.Is(err, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", err, ErrAutonomyEligibleTargetRequired)
	}
}

func TestFrankRegistryScaffoldingPreservesZeroTargetExecutionBehavior(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 23, 0, 0, 0, time.UTC)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-mail",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail",
		ProviderOrPlatformID: "provider-mail",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "candidate",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want nil", ec.GovernedExternalTargets)
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" || decision.Reason != "" {
		t.Fatalf("EvaluateTool() = %#v, want allowed zero-target behavior", decision)
	}
}

func TestResolveFrankRegistryObjectRefSucceedsForEachSupportedKind(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}, EligibilityLabelAutonomyCompatible, "account-class-mailbox", "check-account-class-mailbox", now.Add(time.Minute))
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}, EligibilityLabelAutonomyCompatible, "container-class-wallet", "check-container-class-wallet", now.Add(2*time.Minute))

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-mail",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail",
		ProviderOrPlatformID: "provider-mail",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}
	if err := StoreFrankAccountRecord(root, FrankAccountRecord{
		AccountID:            "account-mail",
		AccountKind:          "mailbox",
		Label:                "Inbox",
		ProviderOrPlatformID: "provider-mail",
		IdentityID:           "identity-mail",
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
		CreatedAt:            now.Add(2 * time.Minute),
		UpdatedAt:            now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}
	if err := StoreFrankContainerRecord(root, FrankContainerRecord{
		ContainerID:          "container-wallet",
		ContainerKind:        "wallet",
		Label:                "Primary Wallet",
		ContainerClassID:     "container-class-wallet",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
		CreatedAt:            now.Add(4 * time.Minute),
		UpdatedAt:            now.Add(5 * time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	tests := []struct {
		name    string
		ref     FrankRegistryObjectRef
		wantRef FrankRegistryObjectRef
		check   func(t *testing.T, got ResolvedFrankRegistryObjectRef)
	}{
		{
			name: "identity",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKind(" identity "),
				ObjectID: " identity-mail ",
			},
			wantRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "identity-mail",
			},
			check: func(t *testing.T, got ResolvedFrankRegistryObjectRef) {
				t.Helper()
				if got.Identity == nil || got.Identity.IdentityID != "identity-mail" {
					t.Fatalf("resolved identity = %#v, want identity-mail", got.Identity)
				}
				if got.Account != nil || got.Container != nil {
					t.Fatalf("resolved identity payload = %#v, want only identity set", got)
				}
			},
		},
		{
			name: "account",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKind(" account "),
				ObjectID: " account-mail ",
			},
			wantRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindAccount,
				ObjectID: "account-mail",
			},
			check: func(t *testing.T, got ResolvedFrankRegistryObjectRef) {
				t.Helper()
				if got.Account == nil || got.Account.AccountID != "account-mail" {
					t.Fatalf("resolved account = %#v, want account-mail", got.Account)
				}
				if got.Identity != nil || got.Container != nil {
					t.Fatalf("resolved account payload = %#v, want only account set", got)
				}
			},
		},
		{
			name: "container",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKind(" container "),
				ObjectID: " container-wallet ",
			},
			wantRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
			check: func(t *testing.T, got ResolvedFrankRegistryObjectRef) {
				t.Helper()
				if got.Container == nil || got.Container.ContainerID != "container-wallet" {
					t.Fatalf("resolved container = %#v, want container-wallet", got.Container)
				}
				if got.Identity != nil || got.Account != nil {
					t.Fatalf("resolved container payload = %#v, want only container set", got)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveFrankRegistryObjectRef(root, tc.ref)
			if err != nil {
				t.Fatalf("ResolveFrankRegistryObjectRef() error = %v", err)
			}
			if !reflect.DeepEqual(got.Ref, tc.wantRef) {
				t.Fatalf("ResolveFrankRegistryObjectRef().Ref = %#v, want %#v", got.Ref, tc.wantRef)
			}
			tc.check(t, got)
		})
	}
}

func TestResolveFrankRegistryObjectRefFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 1, 0, 0, 0, time.UTC)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)

	tests := []struct {
		name string
		ref  FrankRegistryObjectRef
		want string
	}{
		{
			name: "invalid kind",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKind(""),
				ObjectID: "identity-mail",
			},
			want: `Frank object ref kind "" is invalid`,
		},
		{
			name: "empty object id",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "   ",
			},
			want: "Frank object ref object_id is required",
		},
		{
			name: "missing record",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "missing-identity",
			},
			want: ErrFrankIdentityRecordNotFound.Error(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ResolveFrankRegistryObjectRef(root, tc.ref)
			if err == nil {
				t.Fatal("ResolveFrankRegistryObjectRef() error = nil, want fail-closed error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ResolveFrankRegistryObjectRef() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveFrankRegistryObjectRefFailsClosedOnMalformedRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 2, 0, 0, 0, time.UTC)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)

	if err := WriteStoreJSONAtomic(StoreFrankIdentityPath(root, "identity-bad"), map[string]interface{}{
		"record_version":          StoreRecordVersion,
		"identity_id":             "identity-bad",
		"identity_kind":           "email",
		"display_name":            "",
		"provider_or_platform_id": "provider-mail",
		"identity_mode":           string(IdentityModeAgentAlias),
		"state":                   "active",
		"eligibility_target_ref": map[string]interface{}{
			"kind":        string(EligibilityTargetKindProvider),
			"registry_id": "provider-mail",
		},
		"created_at": now,
		"updated_at": now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	_, err := ResolveFrankRegistryObjectRef(root, FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindIdentity,
		ObjectID: "identity-bad",
	})
	if err == nil {
		t.Fatal("ResolveFrankRegistryObjectRef() error = nil, want malformed-record rejection")
	}
	if !strings.Contains(err.Error(), "display_name is required") {
		t.Fatalf("ResolveFrankRegistryObjectRef() error = %q, want malformed record validation failure", err.Error())
	}
}

func TestResolveFrankRegistryObjectRefsRejectsDuplicatesAfterNormalization(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	_, err := ResolveFrankRegistryObjectRefs(root, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
		{
			Kind:     FrankRegistryObjectKind(" identity "),
			ObjectID: " identity-mail ",
		},
	})
	if err == nil {
		t.Fatal("ResolveFrankRegistryObjectRefs() error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `duplicate Frank object ref kind "identity" object_id "identity-mail"`) {
		t.Fatalf("ResolveFrankRegistryObjectRefs() error = %q, want duplicate rejection", err.Error())
	}
}

func TestResolveFrankRegistryObjectRefDoesNotIntroduceEligibilityOrIdentityModeSideChannel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 3, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "provider-human-id.example", "check-provider-human-id", now)

	if err := StoreFrankIdentityRecord(root, FrankIdentityRecord{
		IdentityID:           "identity-human-id",
		IdentityKind:         "email",
		DisplayName:          "Human-ID Candidate",
		ProviderOrPlatformID: target.RegistryID,
		IdentityMode:         IdentityModeOwnerOnlyControl,
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now,
		UpdatedAt:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	if _, err := RequireAutonomyEligibleTarget(root, target); !errors.Is(err, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", err, ErrAutonomyEligibleTargetRequired)
	}

	got, err := ResolveFrankRegistryObjectRef(root, FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindIdentity,
		ObjectID: "identity-human-id",
	})
	if err != nil {
		t.Fatalf("ResolveFrankRegistryObjectRef() error = %v", err)
	}
	if got.Identity == nil {
		t.Fatal("ResolveFrankRegistryObjectRef().Identity = nil, want resolved identity record")
	}
	if got.Identity.IdentityMode != IdentityModeOwnerOnlyControl {
		t.Fatalf("ResolveFrankRegistryObjectRef().Identity.IdentityMode = %q, want %q", got.Identity.IdentityMode, IdentityModeOwnerOnlyControl)
	}
	if got.Identity.EligibilityTargetRef != target {
		t.Fatalf("ResolveFrankRegistryObjectRef().Identity.EligibilityTargetRef = %#v, want %#v", got.Identity.EligibilityTargetRef, target)
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsZeroRefPathPreservesPriorBehavior(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want nil", ec.Step.FrankObjectRefs)
	}

	got, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %v", err)
	}
	if !reflect.DeepEqual(got, ResolvedExecutionContextFrankRegistryObjects{}) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() = %#v, want zero value for zero-ref step", got)
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsResolvesActiveIdentityRef(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKind(" identity "),
			ObjectID: " identity-mail ",
		},
	})
	ec.MissionStoreRoot = fixtures.root

	got, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %v", err)
	}
	if len(got.ResolvedRefs) != 1 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs len = %d, want 1", len(got.ResolvedRefs))
	}
	wantRef := FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindIdentity,
		ObjectID: "identity-mail",
	}
	if !reflect.DeepEqual(got.ResolvedRefs[0].Ref, wantRef) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Ref = %#v, want %#v", got.ResolvedRefs[0].Ref, wantRef)
	}
	if got.ResolvedRefs[0].Identity == nil || !reflect.DeepEqual(*got.ResolvedRefs[0].Identity, fixtures.identity) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Identity = %#v, want %#v", got.ResolvedRefs[0].Identity, fixtures.identity)
	}
	if len(got.Identities) != 1 || !reflect.DeepEqual(got.Identities[0], fixtures.identity) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Identities = %#v, want [%#v]", got.Identities, fixtures.identity)
	}
	if len(got.Accounts) != 0 || len(got.Containers) != 0 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() = %#v, want only identity resolution", got)
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsResolvesActiveAccountRef(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKind(" account "),
			ObjectID: " account-mail ",
		},
	})
	ec.MissionStoreRoot = fixtures.root

	got, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %v", err)
	}
	if len(got.ResolvedRefs) != 1 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs len = %d, want 1", len(got.ResolvedRefs))
	}
	wantRef := FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindAccount,
		ObjectID: "account-mail",
	}
	if !reflect.DeepEqual(got.ResolvedRefs[0].Ref, wantRef) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Ref = %#v, want %#v", got.ResolvedRefs[0].Ref, wantRef)
	}
	if got.ResolvedRefs[0].Account == nil || !reflect.DeepEqual(*got.ResolvedRefs[0].Account, fixtures.account) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Account = %#v, want %#v", got.ResolvedRefs[0].Account, fixtures.account)
	}
	if len(got.Accounts) != 1 || !reflect.DeepEqual(got.Accounts[0], fixtures.account) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Accounts = %#v, want [%#v]", got.Accounts, fixtures.account)
	}
	if len(got.Identities) != 0 || len(got.Containers) != 0 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() = %#v, want only account resolution", got)
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsResolvesActiveContainerRef(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKind(" container "),
			ObjectID: " container-wallet ",
		},
	})
	ec.MissionStoreRoot = fixtures.root

	got, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %v", err)
	}
	if len(got.ResolvedRefs) != 1 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs len = %d, want 1", len(got.ResolvedRefs))
	}
	wantRef := FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindContainer,
		ObjectID: "container-wallet",
	}
	if !reflect.DeepEqual(got.ResolvedRefs[0].Ref, wantRef) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Ref = %#v, want %#v", got.ResolvedRefs[0].Ref, wantRef)
	}
	if got.ResolvedRefs[0].Container == nil || !reflect.DeepEqual(*got.ResolvedRefs[0].Container, fixtures.container) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().ResolvedRefs[0].Container = %#v, want %#v", got.ResolvedRefs[0].Container, fixtures.container)
	}
	if len(got.Containers) != 1 || !reflect.DeepEqual(got.Containers[0], fixtures.container) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Containers = %#v, want [%#v]", got.Containers, fixtures.container)
	}
	if len(got.Identities) != 0 || len(got.Accounts) != 0 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() = %#v, want only container resolution", got)
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsFailsClosedWithoutMissionStoreRoot(t *testing.T) {
	t.Parallel()

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
	})

	_, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankRegistryObjectRefs() error = nil, want missing mission store root rejection")
	}
	if !strings.Contains(err.Error(), "mission store root is required to resolve Frank object refs") {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %q, want missing mission store root rejection", err.Error())
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsFailsClosedOnMissingOrMalformedRefs(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	if err := WriteStoreJSONAtomic(StoreFrankIdentityPath(fixtures.root, "identity-bad"), map[string]interface{}{
		"record_version":          StoreRecordVersion,
		"identity_id":             "identity-bad",
		"identity_kind":           "email",
		"display_name":            "",
		"provider_or_platform_id": "provider-mail",
		"identity_mode":           string(IdentityModeAgentAlias),
		"state":                   "active",
		"eligibility_target_ref": map[string]interface{}{
			"kind":        string(EligibilityTargetKindProvider),
			"registry_id": "provider-mail",
		},
		"created_at": fixtures.identity.CreatedAt,
		"updated_at": fixtures.identity.UpdatedAt,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}

	tests := []struct {
		name string
		refs []FrankRegistryObjectRef
		want string
	}{
		{
			name: "missing record",
			refs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "missing-identity",
				},
			},
			want: ErrFrankIdentityRecordNotFound.Error(),
		},
		{
			name: "empty object id",
			refs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "   ",
				},
			},
			want: "Frank object ref object_id is required",
		},
		{
			name: "unknown kind",
			refs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKind("mystery"),
					ObjectID: "identity-mail",
				},
			},
			want: `Frank object ref kind "mystery" is invalid`,
		},
		{
			name: "malformed stored record",
			refs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "identity-bad",
				},
			},
			want: "display_name is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ec := testExecutionContext()
			ec.Step.FrankObjectRefs = tc.refs
			ec.MissionStoreRoot = fixtures.root

			_, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
			if err == nil {
				t.Fatal("ResolveExecutionContextFrankRegistryObjectRefs() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsRejectsDuplicatesAfterNormalization(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	ec := testExecutionContext()
	ec.Step.FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-mail",
		},
		{
			Kind:     FrankRegistryObjectKind(" identity "),
			ObjectID: " identity-mail ",
		},
	}
	ec.MissionStoreRoot = fixtures.root

	_, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err == nil {
		t.Fatal("ResolveExecutionContextFrankRegistryObjectRefs() error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `duplicate Frank object ref kind "identity" object_id "identity-mail"`) {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %q, want duplicate rejection", err.Error())
	}
}

func TestResolveExecutionContextFrankRegistryObjectRefsDoesNotIntroduceEligibilityOrIdentityModeSideChannel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 4, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "provider-human-id.example", "check-provider-human-id", now)

	record := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-human-id",
		IdentityKind:         "email",
		DisplayName:          "Human-ID Candidate",
		ProviderOrPlatformID: target.RegistryID,
		IdentityMode:         IdentityModeOwnerOnlyControl,
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankIdentityRecord(root, record); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	if _, err := RequireAutonomyEligibleTarget(root, target); !errors.Is(err, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", err, ErrAutonomyEligibleTargetRequired)
	}

	ec := testExecutionContextWithFrankObjectRefs(t, []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: record.IdentityID,
		},
	})
	ec.MissionStoreRoot = root
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}

	got, err := ResolveExecutionContextFrankRegistryObjectRefs(ec)
	if err != nil {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs() error = %v", err)
	}
	if len(got.Identities) != 1 {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Identities len = %d, want 1", len(got.Identities))
	}
	if got.Identities[0].IdentityMode != IdentityModeOwnerOnlyControl {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Identities[0].IdentityMode = %q, want %q", got.Identities[0].IdentityMode, IdentityModeOwnerOnlyControl)
	}
	if got.Identities[0].EligibilityTargetRef != target {
		t.Fatalf("ResolveExecutionContextFrankRegistryObjectRefs().Identities[0].EligibilityTargetRef = %#v, want %#v", got.Identities[0].EligibilityTargetRef, target)
	}
}

type executionContextFrankRegistryFixtures struct {
	root      string
	identity  FrankIdentityRecord
	account   FrankAccountRecord
	container FrankContainerRecord
}

func writeExecutionContextFrankRegistryFixtures(t *testing.T) executionContextFrankRegistryFixtures {
	t.Helper()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}, EligibilityLabelAutonomyCompatible, "provider-mail.example", "check-provider-mail", now)
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindAccountClass,
		RegistryID: "account-class-mailbox",
	}, EligibilityLabelAutonomyCompatible, "account-class-mailbox", "check-account-class-mailbox", now.Add(time.Minute))
	writeFrankRegistryEligibilityFixture(t, root, AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-wallet",
	}, EligibilityLabelAutonomyCompatible, "container-class-wallet", "check-container-class-wallet", now.Add(2*time.Minute))

	identity := FrankIdentityRecord{
		RecordVersion:        StoreRecordVersion,
		IdentityID:           "identity-mail",
		IdentityKind:         "email",
		DisplayName:          "Frank Mail",
		ProviderOrPlatformID: "provider-mail",
		IdentityMode:         IdentityModeAgentAlias,
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankIdentityRecord(root, identity); err != nil {
		t.Fatalf("StoreFrankIdentityRecord() error = %v", err)
	}

	account := FrankAccountRecord{
		RecordVersion:        StoreRecordVersion,
		AccountID:            "account-mail",
		AccountKind:          "mailbox",
		Label:                "Inbox",
		ProviderOrPlatformID: "provider-mail",
		IdentityID:           identity.IdentityID,
		ControlModel:         "agent_managed",
		RecoveryModel:        "agent_recoverable",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
		CreatedAt:            now.Add(2 * time.Minute).UTC(),
		UpdatedAt:            now.Add(3 * time.Minute).UTC(),
	}
	if err := StoreFrankAccountRecord(root, account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	container := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-wallet",
		ContainerKind:        "wallet",
		Label:                "Primary Wallet",
		ContainerClassID:     "container-class-wallet",
		State:                "active",
		EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
		CreatedAt:            now.Add(4 * time.Minute).UTC(),
		UpdatedAt:            now.Add(5 * time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	return executionContextFrankRegistryFixtures{
		root:      root,
		identity:  identity,
		account:   account,
		container: container,
	}
}

func testExecutionContextWithFrankObjectRefs(t *testing.T, refs []FrankRegistryObjectRef) ExecutionContext {
	t.Helper()

	job := testExecutionJob()
	job.Plan.Steps[0].FrankObjectRefs = refs

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	return ec
}

func writeFrankRegistryEligibilityFixture(t *testing.T, root string, target AutonomyEligibilityTargetRef, label EligibilityLabel, targetName string, checkID string, checkedAt time.Time) {
	t.Helper()

	check := EligibilityCheckRecord{
		CheckID:    checkID,
		TargetKind: target.Kind,
		TargetName: targetName,
		Label:      label,
		CheckedAt:  checkedAt,
	}

	switch label {
	case EligibilityLabelAutonomyCompatible:
		check.CanCreateWithoutOwner = true
		check.CanOnboardWithoutOwner = true
		check.CanControlAsAgent = true
		check.CanRecoverAsAgent = true
		check.RulesAsObservedOK = true
		check.Reasons = []string{"autonomy_compatible"}
	case EligibilityLabelHumanGated:
		check.CanCreateWithoutOwner = false
		check.CanOnboardWithoutOwner = false
		check.CanControlAsAgent = false
		check.CanRecoverAsAgent = false
		check.RequiresHumanOnlyStep = true
		check.RulesAsObservedOK = false
		check.Reasons = []string{string(AutonomyEligibilityReasonHumanGatedKYCOrCustodialOnboarding)}
	case EligibilityLabelIneligible:
		check.CanCreateWithoutOwner = false
		check.CanOnboardWithoutOwner = false
		check.CanControlAsAgent = false
		check.CanRecoverAsAgent = false
		check.RulesAsObservedOK = false
		check.Reasons = []string{string(AutonomyEligibilityReasonNotAutonomyCompatible)}
	default:
		t.Fatalf("unsupported eligibility label %q", label)
	}

	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     targetName,
		TargetClass:      target.Kind,
		EligibilityLabel: label,
		LastCheckID:      checkID,
		Notes:            []string{"registry fixture"},
		UpdatedAt:        checkedAt,
	}, check)
}
