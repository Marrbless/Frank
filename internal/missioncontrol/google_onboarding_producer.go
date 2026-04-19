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

type GoogleUserInfoReadback struct {
	GoogleSub     string
	Email         string
	EmailVerified *bool
	Name          string
	PictureURL    string
}

var readGoogleUserInfo = defaultReadGoogleUserInfo

func ProduceFrankGoogleOnboarding(root string, bundle ResolvedExecutionContextFrankGoogleOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankGoogleOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankGoogleOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	accessToken, err := loadFrankGoogleAccessTokenEnvRef(bundle.Account.AccountID, bundle.Account.Google.OAuthAccessTokenEnvVarRef, "google.oauth_access_token_env_var_ref")
	if err != nil {
		return err
	}

	readBack, err := readGoogleUserInfo(context.Background(), accessToken)
	if err != nil {
		return fmt.Errorf("google onboarding provider read-back failed: %w", err)
	}
	readBack.GoogleSub = strings.TrimSpace(readBack.GoogleSub)
	readBack.Email = strings.TrimSpace(readBack.Email)
	readBack.Name = strings.TrimSpace(readBack.Name)
	readBack.PictureURL = strings.TrimSpace(readBack.PictureURL)
	if readBack.GoogleSub == "" {
		return fmt.Errorf("google onboarding provider read-back did not return google_sub")
	}

	committedGoogleSub := ""
	committedEmail := ""
	var committedEmailVerified *bool
	committedName := ""
	committedPictureURL := ""
	if bundle.Identity.Google != nil {
		committedGoogleSub = strings.TrimSpace(bundle.Identity.Google.GoogleSub)
		committedEmail = strings.TrimSpace(bundle.Identity.Google.Email)
		committedEmailVerified = cloneBoolPtr(bundle.Identity.Google.EmailVerified)
		committedName = strings.TrimSpace(bundle.Identity.Google.Name)
		committedPictureURL = strings.TrimSpace(bundle.Identity.Google.PictureURL)
	}
	if committedGoogleSub != "" && committedGoogleSub != readBack.GoogleSub {
		return fmt.Errorf("google identity %q conflicts with provider google_sub %q", bundle.Identity.IdentityID, readBack.GoogleSub)
	}

	committedClientIDEnvVarRef := ""
	committedAccessTokenEnvVarRef := ""
	committedConfirmedAuthenticated := false
	if bundle.Account.Google != nil {
		committedClientIDEnvVarRef = strings.TrimSpace(bundle.Account.Google.OAuthClientIDEnvVarRef)
		committedAccessTokenEnvVarRef = strings.TrimSpace(bundle.Account.Google.OAuthAccessTokenEnvVarRef)
		committedConfirmedAuthenticated = bundle.Account.Google.ConfirmedAuthenticated
	}

	identityUnchanged := committedGoogleSub == readBack.GoogleSub &&
		(committedEmail == "" || committedEmail == readBack.Email) &&
		boolPtrEqual(committedEmailVerified, readBack.EmailVerified) &&
		(committedName == "" || committedName == readBack.Name) &&
		(committedPictureURL == "" || committedPictureURL == readBack.PictureURL)
	if committedConfirmedAuthenticated && identityUnchanged {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.Google == nil {
		updatedIdentity.Google = &FrankGoogleIdentity{}
	}
	updatedIdentity.Google.GoogleSub = readBack.GoogleSub
	if readBack.Email != "" {
		updatedIdentity.Google.Email = readBack.Email
	}
	if readBack.EmailVerified != nil {
		updatedIdentity.Google.EmailVerified = cloneBoolPtr(readBack.EmailVerified)
	}
	if readBack.Name != "" {
		updatedIdentity.Google.Name = readBack.Name
	}
	if readBack.PictureURL != "" {
		updatedIdentity.Google.PictureURL = readBack.PictureURL
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.Google == nil {
		updatedAccount.Google = &FrankGoogleAccount{}
	}
	if updatedAccount.Google.OAuthClientIDEnvVarRef == "" {
		updatedAccount.Google.OAuthClientIDEnvVarRef = committedClientIDEnvVarRef
	}
	if updatedAccount.Google.OAuthAccessTokenEnvVarRef == "" {
		updatedAccount.Google.OAuthAccessTokenEnvVarRef = committedAccessTokenEnvVarRef
	}
	updatedAccount.Google.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankGoogleAccessTokenEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank google account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank google account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadGoogleUserInfo(ctx context.Context, accessToken string) (GoogleUserInfoReadback, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return GoogleUserInfoReadback{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return GoogleUserInfoReadback{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return GoogleUserInfoReadback{}, fmt.Errorf("google userinfo read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GoogleUserInfoReadback{}, fmt.Errorf("decode google userinfo read-back: %w", err)
	}
	return GoogleUserInfoReadback{
		GoogleSub:     strings.TrimSpace(payload.Sub),
		Email:         strings.TrimSpace(payload.Email),
		EmailVerified: cloneBoolPtr(payload.EmailVerified),
		Name:          strings.TrimSpace(payload.Name),
		PictureURL:    strings.TrimSpace(payload.Picture),
	}, nil
}
