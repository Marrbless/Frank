package missioncontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type StripeReadbackAccount struct {
	StripeAccountID            string
	BusinessProfileName        string
	DashboardDisplayName       string
	Country                    string
	DefaultCurrency            string
	ChargesEnabled             *bool
	PayoutsEnabled             *bool
	DetailsSubmitted           *bool
	RequirementsDisabledReason string
	Livemode                   *bool
}

var readStripeAccount = defaultReadStripeAccount

// ProduceFrankStripeOnboarding is the single missioncontrol-owned execution
// producer for the Stripe onboarding lane. It confirms the configured
// authenticated Stripe account through provider read-back only and persists
// only non-secret Stripe identifiers and account status onto the committed
// Frank identity/account records.
func ProduceFrankStripeOnboarding(root string, bundle ResolvedExecutionContextFrankStripeOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankStripeOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankStripeOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	apiKey, err := loadFrankStripeSecretKeyEnvRef(bundle.Account.AccountID, bundle.Account.Stripe.SecretKeyEnvVarRef, "stripe.secret_key_env_var_ref")
	if err != nil {
		return err
	}

	readBack, err := readStripeAccount(context.Background(), apiKey)
	if err != nil {
		return fmt.Errorf("stripe onboarding provider read-back failed: %w", err)
	}
	readBack.StripeAccountID = strings.TrimSpace(readBack.StripeAccountID)
	readBack.BusinessProfileName = strings.TrimSpace(readBack.BusinessProfileName)
	readBack.DashboardDisplayName = strings.TrimSpace(readBack.DashboardDisplayName)
	readBack.Country = strings.ToUpper(strings.TrimSpace(readBack.Country))
	readBack.DefaultCurrency = strings.ToLower(strings.TrimSpace(readBack.DefaultCurrency))
	readBack.RequirementsDisabledReason = strings.TrimSpace(readBack.RequirementsDisabledReason)
	if readBack.StripeAccountID == "" {
		return fmt.Errorf("stripe onboarding provider read-back did not return stripe_account_id")
	}

	committedAccountID := ""
	committedBusinessProfileName := ""
	committedDashboardDisplayName := ""
	committedCountry := ""
	committedDefaultCurrency := ""
	if bundle.Identity.Stripe != nil {
		committedAccountID = strings.TrimSpace(bundle.Identity.Stripe.StripeAccountID)
		committedBusinessProfileName = strings.TrimSpace(bundle.Identity.Stripe.BusinessProfileName)
		committedDashboardDisplayName = strings.TrimSpace(bundle.Identity.Stripe.DashboardDisplayName)
		committedCountry = strings.ToUpper(strings.TrimSpace(bundle.Identity.Stripe.Country))
		committedDefaultCurrency = strings.ToLower(strings.TrimSpace(bundle.Identity.Stripe.DefaultCurrency))
	}
	if committedAccountID != "" && committedAccountID != readBack.StripeAccountID {
		return fmt.Errorf("stripe identity %q conflicts with provider stripe_account_id %q", bundle.Identity.IdentityID, readBack.StripeAccountID)
	}

	committedSecretKeyEnvVarRef := ""
	committedConfirmedAuthenticated := false
	var committedChargesEnabled *bool
	var committedPayoutsEnabled *bool
	var committedDetailsSubmitted *bool
	committedRequirementsDisabledReason := ""
	var committedLivemode *bool
	if bundle.Account.Stripe != nil {
		committedSecretKeyEnvVarRef = strings.TrimSpace(bundle.Account.Stripe.SecretKeyEnvVarRef)
		committedConfirmedAuthenticated = bundle.Account.Stripe.ConfirmedAuthenticated
		committedChargesEnabled = cloneBoolPtr(bundle.Account.Stripe.ChargesEnabled)
		committedPayoutsEnabled = cloneBoolPtr(bundle.Account.Stripe.PayoutsEnabled)
		committedDetailsSubmitted = cloneBoolPtr(bundle.Account.Stripe.DetailsSubmitted)
		committedRequirementsDisabledReason = strings.TrimSpace(bundle.Account.Stripe.RequirementsDisabledReason)
		committedLivemode = cloneBoolPtr(bundle.Account.Stripe.Livemode)
	}

	identityUnchanged := committedAccountID == readBack.StripeAccountID &&
		(committedBusinessProfileName == "" || committedBusinessProfileName == readBack.BusinessProfileName) &&
		(committedDashboardDisplayName == "" || committedDashboardDisplayName == readBack.DashboardDisplayName) &&
		(committedCountry == "" || committedCountry == readBack.Country) &&
		(committedDefaultCurrency == "" || committedDefaultCurrency == readBack.DefaultCurrency)
	accountStatusUnchanged := boolPtrEqual(committedChargesEnabled, readBack.ChargesEnabled) &&
		boolPtrEqual(committedPayoutsEnabled, readBack.PayoutsEnabled) &&
		boolPtrEqual(committedDetailsSubmitted, readBack.DetailsSubmitted) &&
		(committedRequirementsDisabledReason == "" || committedRequirementsDisabledReason == readBack.RequirementsDisabledReason) &&
		boolPtrEqual(committedLivemode, readBack.Livemode)
	if committedConfirmedAuthenticated && identityUnchanged && accountStatusUnchanged {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.Stripe == nil {
		updatedIdentity.Stripe = &FrankStripeIdentity{}
	}
	updatedIdentity.Stripe.StripeAccountID = readBack.StripeAccountID
	if readBack.BusinessProfileName != "" {
		updatedIdentity.Stripe.BusinessProfileName = readBack.BusinessProfileName
	}
	if readBack.DashboardDisplayName != "" {
		updatedIdentity.Stripe.DashboardDisplayName = readBack.DashboardDisplayName
	}
	if readBack.Country != "" {
		updatedIdentity.Stripe.Country = readBack.Country
	}
	if readBack.DefaultCurrency != "" {
		updatedIdentity.Stripe.DefaultCurrency = readBack.DefaultCurrency
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.Stripe == nil {
		updatedAccount.Stripe = &FrankStripeAccount{}
	}
	if updatedAccount.Stripe.SecretKeyEnvVarRef == "" {
		updatedAccount.Stripe.SecretKeyEnvVarRef = committedSecretKeyEnvVarRef
	}
	updatedAccount.Stripe.ConfirmedAuthenticated = true
	updatedAccount.Stripe.ChargesEnabled = cloneBoolPtr(readBack.ChargesEnabled)
	updatedAccount.Stripe.PayoutsEnabled = cloneBoolPtr(readBack.PayoutsEnabled)
	updatedAccount.Stripe.DetailsSubmitted = cloneBoolPtr(readBack.DetailsSubmitted)
	updatedAccount.Stripe.RequirementsDisabledReason = readBack.RequirementsDisabledReason
	updatedAccount.Stripe.Livemode = cloneBoolPtr(readBack.Livemode)
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankStripeSecretKeyEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank stripe account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank stripe account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadStripeAccount(ctx context.Context, apiKey string) (StripeReadbackAccount, error) {
	var payload struct {
		ID               string `json:"id"`
		Country          string `json:"country"`
		DefaultCurrency  string `json:"default_currency"`
		ChargesEnabled   *bool  `json:"charges_enabled"`
		PayoutsEnabled   *bool  `json:"payouts_enabled"`
		DetailsSubmitted *bool  `json:"details_submitted"`
		Livemode         *bool  `json:"livemode"`
		BusinessProfile  struct {
			Name string `json:"name"`
		} `json:"business_profile"`
		Settings struct {
			Dashboard struct {
				DisplayName string `json:"display_name"`
			} `json:"dashboard"`
		} `json:"settings"`
		Requirements struct {
			DisabledReason string `json:"disabled_reason"`
		} `json:"requirements"`
	}
	if err := doStripeJSONRequest(ctx, apiKey, "https://api.stripe.com/v1/account", &payload); err != nil {
		return StripeReadbackAccount{}, err
	}
	return StripeReadbackAccount{
		StripeAccountID:            payload.ID,
		BusinessProfileName:        payload.BusinessProfile.Name,
		DashboardDisplayName:       payload.Settings.Dashboard.DisplayName,
		Country:                    payload.Country,
		DefaultCurrency:            payload.DefaultCurrency,
		ChargesEnabled:             cloneBoolPtr(payload.ChargesEnabled),
		PayoutsEnabled:             cloneBoolPtr(payload.PayoutsEnabled),
		DetailsSubmitted:           cloneBoolPtr(payload.DetailsSubmitted),
		RequirementsDisabledReason: payload.Requirements.DisabledReason,
		Livemode:                   cloneBoolPtr(payload.Livemode),
	}, nil
}

func doStripeJSONRequest(ctx context.Context, apiKey string, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("stripe read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode stripe read-back: %w", err)
	}
	return nil
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
