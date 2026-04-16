package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestProduceFrankZohoMailboxBootstrapSuccessfulCommittedConfirmation(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	now := time.Date(2026, 4, 16, 16, 0, 0, 0, time.UTC)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}

	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, now); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %v", err)
	}

	identity, err := LoadFrankIdentityRecord(fixtures.root, fixtures.identity.IdentityID)
	if err != nil {
		t.Fatalf("LoadFrankIdentityRecord() error = %v", err)
	}
	account, err := LoadFrankAccountRecord(fixtures.root, fixtures.account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if identity.ZohoMailbox == nil || identity.ZohoMailbox.FromAddress != "frank@example.com" {
		t.Fatalf("LoadFrankIdentityRecord().ZohoMailbox = %#v, want committed from-address proof", identity.ZohoMailbox)
	}
	if account.ZohoMailbox == nil || account.ZohoMailbox.ProviderAccountID != "3323462000000008002" || !account.ZohoMailbox.ConfirmedCreated {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox = %#v, want committed confirmed-created provider account", account.ZohoMailbox)
	}
}

func TestProduceFrankZohoMailboxBootstrapReplayStaysDeterministic(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}

	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 10, 0, 0, time.UTC)); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap(first) error = %v", err)
	}
	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 11, 0, 0, time.UTC)); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap(replay) error = %v, want deterministic no-op success", err)
	}
}

func TestProduceFrankZohoMailboxBootstrapFailsClosedWithoutCommittedConfirmation(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}
	pair.Account.ZohoMailbox = &FrankZohoMailboxAccount{}

	err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 20, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ProduceFrankZohoMailboxBootstrap() error = nil, want committed confirmation rejection")
	}
	if !strings.Contains(err.Error(), `mission store Frank zoho mailbox bootstrap account "account-mail" requires committed zoho_mailbox.confirmed_created state`) {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %q, want committed confirmation rejection", err.Error())
	}
}

func TestProduceFrankZohoMailboxBootstrapFailsClosedWhenEligibilityIsNotAutonomyCompatible(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	now := time.Date(2026, 4, 16, 16, 30, 0, 0, time.UTC)
	writeFrankRegistryEligibilityFixture(t, fixtures.root, fixtures.account.EligibilityTargetRef, EligibilityLabelHumanGated, "account-class-mailbox", "check-account-class-mailbox-blocked", now)

	err := ProduceFrankZohoMailboxBootstrap(fixtures.root, ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}, now.Add(time.Minute))
	if err == nil {
		t.Fatal("ProduceFrankZohoMailboxBootstrap() error = nil, want autonomy-eligibility rejection")
	}
	if !strings.Contains(err.Error(), `autonomy eligibility target "account-class-mailbox" is not autonomy-compatible`) {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %q, want autonomy-eligibility rejection", err.Error())
	}
}
