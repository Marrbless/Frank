package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

// ProduceFrankZohoMailboxBootstrap is the single missioncontrol-owned
// execution producer for provider-specific Zoho mailbox bootstrap. On this
// branch line it stays fail-closed and replay-safe: bootstrap completes only
// when the resolved Frank identity/account pair already carries a committed
// provider_account_id plus confirmed_created state on the same Frank account
// record. The producer does not read raw runtime receipts or invent provider
// mailbox-creation behavior.
func ProduceFrankZohoMailboxBootstrap(root string, pair ResolvedExecutionContextFrankZohoMailboxBootstrapPair, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	pair = normalizeResolvedExecutionContextFrankZohoMailboxBootstrapPair(pair)
	if err := validateResolvedExecutionContextFrankZohoMailboxBootstrapPair(pair); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, pair.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, pair.Account.EligibilityTargetRef); err != nil {
		return err
	}
	if !pair.Account.ZohoMailbox.ConfirmedCreated {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q requires committed zoho_mailbox.confirmed_created state",
			pair.Account.AccountID,
		)
	}
	if strings.TrimSpace(pair.Account.ZohoMailbox.ProviderAccountID) == "" {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q requires committed zoho_mailbox.provider_account_id",
			pair.Account.AccountID,
		)
	}

	return nil
}

func normalizeResolvedExecutionContextFrankZohoMailboxBootstrapPair(pair ResolvedExecutionContextFrankZohoMailboxBootstrapPair) ResolvedExecutionContextFrankZohoMailboxBootstrapPair {
	pair.Identity.ZohoMailbox = normalizeFrankZohoMailboxIdentity(pair.Identity.ZohoMailbox)
	pair.Identity.IdentityMode = NormalizeIdentityMode(pair.Identity.IdentityMode)
	pair.Identity.CreatedAt = pair.Identity.CreatedAt.UTC()
	pair.Identity.UpdatedAt = pair.Identity.UpdatedAt.UTC()

	pair.Account.ZohoMailbox = normalizeFrankZohoMailboxAccount(pair.Account.ZohoMailbox)
	pair.Account.CreatedAt = pair.Account.CreatedAt.UTC()
	pair.Account.UpdatedAt = pair.Account.UpdatedAt.UTC()

	return pair
}

func validateResolvedExecutionContextFrankZohoMailboxBootstrapPair(pair ResolvedExecutionContextFrankZohoMailboxBootstrapPair) error {
	if err := ValidateFrankIdentityRecord(pair.Identity); err != nil {
		return err
	}
	if err := ValidateFrankAccountRecord(pair.Account); err != nil {
		return err
	}
	if pair.Identity.ZohoMailbox == nil {
		return fmt.Errorf("mission store Frank zoho mailbox bootstrap requires zoho_mailbox identity record")
	}
	if pair.Account.ZohoMailbox == nil {
		return fmt.Errorf("mission store Frank zoho mailbox bootstrap requires zoho_mailbox account record")
	}
	if strings.TrimSpace(pair.Account.IdentityID) != strings.TrimSpace(pair.Identity.IdentityID) {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q must link identity_id %q, got %q",
			pair.Account.AccountID,
			pair.Identity.IdentityID,
			pair.Account.IdentityID,
		)
	}
	if strings.TrimSpace(pair.Account.ProviderOrPlatformID) != strings.TrimSpace(pair.Identity.ProviderOrPlatformID) {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider_or_platform_id %q does not match identity %q provider_or_platform_id %q",
			pair.Account.AccountID,
			pair.Account.ProviderOrPlatformID,
			pair.Identity.IdentityID,
			pair.Identity.ProviderOrPlatformID,
		)
	}
	return nil
}
