package missioncontrol

import (
	"context"
	"errors"
	"reflect"
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
