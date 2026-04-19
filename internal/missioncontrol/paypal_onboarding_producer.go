package missioncontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PayPalUserInfoReadback struct {
	PayPalUserID    string
	Sub             string
	Email           string
	EmailVerified   *bool
	VerifiedAccount *bool
	AccountType     string
	ScopesSummary   string
}

var readPayPalAccessToken = defaultReadPayPalAccessToken
var readPayPalUserInfo = defaultReadPayPalUserInfo

func ProduceFrankPayPalOnboarding(root string, bundle ResolvedExecutionContextFrankPayPalOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankPayPalOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankPayPalOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	clientID, err := loadFrankPayPalClientIDEnvRef(bundle.Account.AccountID, bundle.Account.PayPal.ClientIDEnvVarRef, "paypal.client_id_env_var_ref")
	if err != nil {
		return err
	}
	clientSecret, err := loadFrankPayPalClientSecretEnvRef(bundle.Account.AccountID, bundle.Account.PayPal.ClientSecretEnvVarRef, "paypal.client_secret_env_var_ref")
	if err != nil {
		return err
	}

	environment := strings.ToLower(strings.TrimSpace(bundle.Account.PayPal.Environment))
	accessToken, scopesSummary, err := readPayPalAccessToken(context.Background(), environment, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("paypal onboarding provider access-token read-back failed: %w", err)
	}
	readBack, err := readPayPalUserInfo(context.Background(), environment, accessToken)
	if err != nil {
		return fmt.Errorf("paypal onboarding provider read-back failed: %w", err)
	}
	readBack.PayPalUserID = strings.TrimSpace(readBack.PayPalUserID)
	readBack.Sub = strings.TrimSpace(readBack.Sub)
	readBack.Email = strings.TrimSpace(readBack.Email)
	readBack.AccountType = strings.TrimSpace(readBack.AccountType)
	readBack.ScopesSummary = strings.TrimSpace(scopesSummary)
	if readBack.PayPalUserID == "" && readBack.Sub == "" {
		return fmt.Errorf("paypal onboarding provider read-back did not return paypal_user_id or sub")
	}

	committedPayPalUserID := ""
	committedSub := ""
	committedEmail := ""
	var committedEmailVerified *bool
	var committedVerifiedAccount *bool
	committedAccountType := ""
	if bundle.Identity.PayPal != nil {
		committedPayPalUserID = strings.TrimSpace(bundle.Identity.PayPal.PayPalUserID)
		committedSub = strings.TrimSpace(bundle.Identity.PayPal.Sub)
		committedEmail = strings.TrimSpace(bundle.Identity.PayPal.Email)
		committedEmailVerified = cloneBoolPtr(bundle.Identity.PayPal.EmailVerified)
		committedVerifiedAccount = cloneBoolPtr(bundle.Identity.PayPal.VerifiedAccount)
		committedAccountType = strings.TrimSpace(bundle.Identity.PayPal.AccountType)
	}
	if committedPayPalUserID != "" && committedPayPalUserID != readBack.PayPalUserID {
		return fmt.Errorf("paypal identity %q conflicts with provider paypal_user_id %q", bundle.Identity.IdentityID, readBack.PayPalUserID)
	}
	if committedSub != "" && committedSub != readBack.Sub {
		return fmt.Errorf("paypal identity %q conflicts with provider sub %q", bundle.Identity.IdentityID, readBack.Sub)
	}

	committedClientIDEnvVarRef := strings.TrimSpace(bundle.Account.PayPal.ClientIDEnvVarRef)
	committedClientSecretEnvVarRef := strings.TrimSpace(bundle.Account.PayPal.ClientSecretEnvVarRef)
	committedEnvironment := strings.ToLower(strings.TrimSpace(bundle.Account.PayPal.Environment))
	committedConfirmedAuthenticated := bundle.Account.PayPal.ConfirmedAuthenticated

	identityUnchanged := (committedPayPalUserID == "" || committedPayPalUserID == readBack.PayPalUserID) &&
		(committedSub == "" || committedSub == readBack.Sub) &&
		(committedEmail == "" || committedEmail == readBack.Email) &&
		(committedAccountType == "" || committedAccountType == readBack.AccountType) &&
		boolPtrEqual(committedEmailVerified, readBack.EmailVerified) &&
		boolPtrEqual(committedVerifiedAccount, readBack.VerifiedAccount)
	if committedConfirmedAuthenticated && identityUnchanged {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.PayPal == nil {
		updatedIdentity.PayPal = &FrankPayPalIdentity{}
	}
	if updatedIdentity.PayPal.PayPalUserID == "" {
		updatedIdentity.PayPal.PayPalUserID = readBack.PayPalUserID
	}
	if updatedIdentity.PayPal.Sub == "" {
		updatedIdentity.PayPal.Sub = readBack.Sub
	}
	if readBack.Email != "" {
		updatedIdentity.PayPal.Email = readBack.Email
	}
	if readBack.EmailVerified != nil {
		updatedIdentity.PayPal.EmailVerified = cloneBoolPtr(readBack.EmailVerified)
	}
	if readBack.VerifiedAccount != nil {
		updatedIdentity.PayPal.VerifiedAccount = cloneBoolPtr(readBack.VerifiedAccount)
	}
	if readBack.AccountType != "" {
		updatedIdentity.PayPal.AccountType = readBack.AccountType
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.PayPal == nil {
		updatedAccount.PayPal = &FrankPayPalAccount{}
	}
	if updatedAccount.PayPal.ClientIDEnvVarRef == "" {
		updatedAccount.PayPal.ClientIDEnvVarRef = committedClientIDEnvVarRef
	}
	if updatedAccount.PayPal.ClientSecretEnvVarRef == "" {
		updatedAccount.PayPal.ClientSecretEnvVarRef = committedClientSecretEnvVarRef
	}
	if updatedAccount.PayPal.Environment == "" {
		updatedAccount.PayPal.Environment = committedEnvironment
	}
	updatedAccount.PayPal.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankPayPalClientIDEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank paypal account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank paypal account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func loadFrankPayPalClientSecretEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank paypal account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank paypal account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadPayPalAccessToken(ctx context.Context, environment, clientID, clientSecret string) (string, string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, paypalBaseURL(environment)+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(strings.TrimSpace(clientID), strings.TrimSpace(clientSecret))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("paypal access-token read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", fmt.Errorf("decode paypal access-token read-back: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", "", fmt.Errorf("paypal access-token read-back returned empty access_token")
	}
	return strings.TrimSpace(payload.AccessToken), strings.TrimSpace(payload.Scope), nil
}

func defaultReadPayPalUserInfo(ctx context.Context, environment, accessToken string) (PayPalUserInfoReadback, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, paypalBaseURL(environment)+"/v1/identity/openidconnect/userinfo?schema=openid", nil)
	if err != nil {
		return PayPalUserInfoReadback{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return PayPalUserInfoReadback{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return PayPalUserInfoReadback{}, fmt.Errorf("paypal userinfo read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		UserID          string `json:"user_id"`
		Sub             string `json:"sub"`
		Email           string `json:"email"`
		EmailVerified   *bool  `json:"email_verified"`
		VerifiedAccount *bool  `json:"verified_account"`
		AccountType     string `json:"account_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return PayPalUserInfoReadback{}, fmt.Errorf("decode paypal userinfo read-back: %w", err)
	}
	return PayPalUserInfoReadback{
		PayPalUserID:    strings.TrimSpace(payload.UserID),
		Sub:             strings.TrimSpace(payload.Sub),
		Email:           strings.TrimSpace(payload.Email),
		EmailVerified:   cloneBoolPtr(payload.EmailVerified),
		VerifiedAccount: cloneBoolPtr(payload.VerifiedAccount),
		AccountType:     strings.TrimSpace(payload.AccountType),
	}, nil
}

func paypalBaseURL(environment string) string {
	if strings.EqualFold(strings.TrimSpace(environment), "live") {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}
