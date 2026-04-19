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

type MicrosoftUserInfoReadback struct {
	MicrosoftSub      string
	MicrosoftOID      string
	TenantID          string
	Email             string
	PreferredUsername string
	EmailVerified     *bool
	DisplayName       string
}

var readMicrosoftUserInfo = defaultReadMicrosoftUserInfo

func ProduceFrankMicrosoftOnboarding(root string, bundle ResolvedExecutionContextFrankMicrosoftOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankMicrosoftOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankMicrosoftOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	accessToken, err := loadFrankMicrosoftAccessTokenEnvRef(bundle.Account.AccountID, bundle.Account.Microsoft.OAuthAccessTokenEnvVarRef, "microsoft.oauth_access_token_env_var_ref")
	if err != nil {
		return err
	}

	readBack, err := readMicrosoftUserInfo(context.Background(), accessToken)
	if err != nil {
		return fmt.Errorf("microsoft onboarding provider read-back failed: %w", err)
	}
	readBack.MicrosoftSub = strings.TrimSpace(readBack.MicrosoftSub)
	readBack.MicrosoftOID = strings.TrimSpace(readBack.MicrosoftOID)
	readBack.TenantID = strings.TrimSpace(readBack.TenantID)
	readBack.Email = strings.TrimSpace(readBack.Email)
	readBack.PreferredUsername = strings.TrimSpace(readBack.PreferredUsername)
	readBack.DisplayName = strings.TrimSpace(readBack.DisplayName)
	if readBack.MicrosoftSub == "" {
		return fmt.Errorf("microsoft onboarding provider read-back did not return microsoft_sub")
	}

	committedMicrosoftSub := ""
	committedMicrosoftOID := ""
	committedTenantID := ""
	committedEmail := ""
	committedPreferredUsername := ""
	var committedEmailVerified *bool
	committedDisplayName := ""
	if bundle.Identity.Microsoft != nil {
		committedMicrosoftSub = strings.TrimSpace(bundle.Identity.Microsoft.MicrosoftSub)
		committedMicrosoftOID = strings.TrimSpace(bundle.Identity.Microsoft.MicrosoftOID)
		committedTenantID = strings.TrimSpace(bundle.Identity.Microsoft.TenantID)
		committedEmail = strings.TrimSpace(bundle.Identity.Microsoft.Email)
		committedPreferredUsername = strings.TrimSpace(bundle.Identity.Microsoft.PreferredUsername)
		committedEmailVerified = cloneBoolPtr(bundle.Identity.Microsoft.EmailVerified)
		committedDisplayName = strings.TrimSpace(bundle.Identity.Microsoft.DisplayName)
	}
	if committedMicrosoftSub != "" && committedMicrosoftSub != readBack.MicrosoftSub {
		return fmt.Errorf("microsoft identity %q conflicts with provider microsoft_sub %q", bundle.Identity.IdentityID, readBack.MicrosoftSub)
	}
	if committedMicrosoftOID != "" && readBack.MicrosoftOID != "" && committedMicrosoftOID != readBack.MicrosoftOID {
		return fmt.Errorf("microsoft identity %q conflicts with provider microsoft_oid %q", bundle.Identity.IdentityID, readBack.MicrosoftOID)
	}
	if committedTenantID != "" && readBack.TenantID != "" && committedTenantID != readBack.TenantID {
		return fmt.Errorf("microsoft identity %q conflicts with provider tenant_id %q", bundle.Identity.IdentityID, readBack.TenantID)
	}

	committedClientIDEnvVarRef := ""
	committedAccessTokenEnvVarRef := ""
	committedRefreshTokenEnvVarRef := ""
	committedConfirmedAuthenticated := false
	if bundle.Account.Microsoft != nil {
		committedClientIDEnvVarRef = strings.TrimSpace(bundle.Account.Microsoft.OAuthClientIDEnvVarRef)
		committedAccessTokenEnvVarRef = strings.TrimSpace(bundle.Account.Microsoft.OAuthAccessTokenEnvVarRef)
		committedRefreshTokenEnvVarRef = strings.TrimSpace(bundle.Account.Microsoft.OAuthRefreshTokenEnvVarRef)
		committedConfirmedAuthenticated = bundle.Account.Microsoft.ConfirmedAuthenticated
	}

	identityUnchanged := committedMicrosoftSub == readBack.MicrosoftSub &&
		optionalReadBackStringCompatible(committedMicrosoftOID, readBack.MicrosoftOID) &&
		optionalReadBackStringCompatible(committedTenantID, readBack.TenantID) &&
		optionalReadBackStringCompatible(committedEmail, readBack.Email) &&
		optionalReadBackStringCompatible(committedPreferredUsername, readBack.PreferredUsername) &&
		optionalReadBackBoolCompatible(committedEmailVerified, readBack.EmailVerified) &&
		optionalReadBackStringCompatible(committedDisplayName, readBack.DisplayName)
	if committedConfirmedAuthenticated && identityUnchanged {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.Microsoft == nil {
		updatedIdentity.Microsoft = &FrankMicrosoftIdentity{}
	}
	updatedIdentity.Microsoft.MicrosoftSub = readBack.MicrosoftSub
	if readBack.MicrosoftOID != "" {
		updatedIdentity.Microsoft.MicrosoftOID = readBack.MicrosoftOID
	}
	if readBack.TenantID != "" {
		updatedIdentity.Microsoft.TenantID = readBack.TenantID
	}
	if readBack.Email != "" {
		updatedIdentity.Microsoft.Email = readBack.Email
	}
	if readBack.PreferredUsername != "" {
		updatedIdentity.Microsoft.PreferredUsername = readBack.PreferredUsername
	}
	if readBack.EmailVerified != nil {
		updatedIdentity.Microsoft.EmailVerified = cloneBoolPtr(readBack.EmailVerified)
	}
	if readBack.DisplayName != "" {
		updatedIdentity.Microsoft.DisplayName = readBack.DisplayName
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.Microsoft == nil {
		updatedAccount.Microsoft = &FrankMicrosoftAccount{}
	}
	if updatedAccount.Microsoft.OAuthClientIDEnvVarRef == "" {
		updatedAccount.Microsoft.OAuthClientIDEnvVarRef = committedClientIDEnvVarRef
	}
	if updatedAccount.Microsoft.OAuthAccessTokenEnvVarRef == "" {
		updatedAccount.Microsoft.OAuthAccessTokenEnvVarRef = committedAccessTokenEnvVarRef
	}
	if updatedAccount.Microsoft.OAuthRefreshTokenEnvVarRef == "" {
		updatedAccount.Microsoft.OAuthRefreshTokenEnvVarRef = committedRefreshTokenEnvVarRef
	}
	updatedAccount.Microsoft.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankMicrosoftAccessTokenEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank microsoft account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank microsoft account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadMicrosoftUserInfo(ctx context.Context, accessToken string) (MicrosoftUserInfoReadback, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/oidc/userinfo", nil)
	if err != nil {
		return MicrosoftUserInfoReadback{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return MicrosoftUserInfoReadback{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return MicrosoftUserInfoReadback{}, fmt.Errorf("microsoft userinfo read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Sub               string `json:"sub"`
		OID               string `json:"oid"`
		TenantID          string `json:"tid"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		EmailVerified     *bool  `json:"email_verified"`
		Name              string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return MicrosoftUserInfoReadback{}, fmt.Errorf("decode microsoft userinfo read-back: %w", err)
	}
	return MicrosoftUserInfoReadback{
		MicrosoftSub:      strings.TrimSpace(payload.Sub),
		MicrosoftOID:      strings.TrimSpace(payload.OID),
		TenantID:          strings.TrimSpace(payload.TenantID),
		Email:             strings.TrimSpace(payload.Email),
		PreferredUsername: strings.TrimSpace(payload.PreferredUsername),
		EmailVerified:     cloneBoolPtr(payload.EmailVerified),
		DisplayName:       strings.TrimSpace(payload.Name),
	}, nil
}

func optionalReadBackStringCompatible(committed, readBack string) bool {
	committed = strings.TrimSpace(committed)
	readBack = strings.TrimSpace(readBack)
	return committed == "" || readBack == "" || committed == readBack
}

func optionalReadBackBoolCompatible(committed, readBack *bool) bool {
	if readBack == nil {
		return true
	}
	return boolPtrEqual(committed, readBack)
}
