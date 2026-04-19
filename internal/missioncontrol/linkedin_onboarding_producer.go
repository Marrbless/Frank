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

type LinkedInUserInfoReadback struct {
	LinkedInSub   string
	Name          string
	PictureURL    string
	Email         string
	EmailVerified *bool
}

var readLinkedInUserInfo = defaultReadLinkedInUserInfo

func ProduceFrankLinkedInOnboarding(root string, bundle ResolvedExecutionContextFrankLinkedInOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankLinkedInOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankLinkedInOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	accessToken, err := loadFrankLinkedInAccessTokenEnvRef(bundle.Account.AccountID, bundle.Account.LinkedIn.OAuthAccessTokenEnvVarRef, "linkedin.oauth_access_token_env_var_ref")
	if err != nil {
		return err
	}

	readBack, err := readLinkedInUserInfo(context.Background(), accessToken)
	if err != nil {
		return fmt.Errorf("linkedin onboarding provider read-back failed: %w", err)
	}
	readBack.LinkedInSub = strings.TrimSpace(readBack.LinkedInSub)
	readBack.Name = strings.TrimSpace(readBack.Name)
	readBack.PictureURL = strings.TrimSpace(readBack.PictureURL)
	readBack.Email = strings.TrimSpace(readBack.Email)
	if readBack.LinkedInSub == "" {
		return fmt.Errorf("linkedin onboarding provider read-back did not return linkedin_sub")
	}

	committedLinkedInSub := ""
	committedName := ""
	committedPictureURL := ""
	committedEmail := ""
	var committedEmailVerified *bool
	if bundle.Identity.LinkedIn != nil {
		committedLinkedInSub = strings.TrimSpace(bundle.Identity.LinkedIn.LinkedInSub)
		committedName = strings.TrimSpace(bundle.Identity.LinkedIn.Name)
		committedPictureURL = strings.TrimSpace(bundle.Identity.LinkedIn.PictureURL)
		committedEmail = strings.TrimSpace(bundle.Identity.LinkedIn.Email)
		committedEmailVerified = cloneBoolPtr(bundle.Identity.LinkedIn.EmailVerified)
	}
	if committedLinkedInSub != "" && committedLinkedInSub != readBack.LinkedInSub {
		return fmt.Errorf("linkedin identity %q conflicts with provider linkedin_sub %q", bundle.Identity.IdentityID, readBack.LinkedInSub)
	}

	committedClientIDEnvVarRef := ""
	committedAccessTokenEnvVarRef := ""
	committedConfirmedAuthenticated := false
	if bundle.Account.LinkedIn != nil {
		committedClientIDEnvVarRef = strings.TrimSpace(bundle.Account.LinkedIn.OAuthClientIDEnvVarRef)
		committedAccessTokenEnvVarRef = strings.TrimSpace(bundle.Account.LinkedIn.OAuthAccessTokenEnvVarRef)
		committedConfirmedAuthenticated = bundle.Account.LinkedIn.ConfirmedAuthenticated
	}

	identityUnchanged := (committedLinkedInSub == "" || committedLinkedInSub == readBack.LinkedInSub) &&
		(committedName == "" || committedName == readBack.Name) &&
		(committedPictureURL == "" || committedPictureURL == readBack.PictureURL) &&
		(committedEmail == "" || committedEmail == readBack.Email) &&
		boolPtrEqual(committedEmailVerified, readBack.EmailVerified)
	if committedConfirmedAuthenticated && identityUnchanged {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.LinkedIn == nil {
		updatedIdentity.LinkedIn = &FrankLinkedInIdentity{}
	}
	updatedIdentity.LinkedIn.LinkedInSub = readBack.LinkedInSub
	if readBack.Name != "" {
		updatedIdentity.LinkedIn.Name = readBack.Name
	}
	if readBack.PictureURL != "" {
		updatedIdentity.LinkedIn.PictureURL = readBack.PictureURL
	}
	if readBack.Email != "" {
		updatedIdentity.LinkedIn.Email = readBack.Email
	}
	if readBack.EmailVerified != nil {
		updatedIdentity.LinkedIn.EmailVerified = cloneBoolPtr(readBack.EmailVerified)
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.LinkedIn == nil {
		updatedAccount.LinkedIn = &FrankLinkedInAccount{}
	}
	if updatedAccount.LinkedIn.OAuthClientIDEnvVarRef == "" {
		updatedAccount.LinkedIn.OAuthClientIDEnvVarRef = committedClientIDEnvVarRef
	}
	if updatedAccount.LinkedIn.OAuthAccessTokenEnvVarRef == "" {
		updatedAccount.LinkedIn.OAuthAccessTokenEnvVarRef = committedAccessTokenEnvVarRef
	}
	updatedAccount.LinkedIn.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankLinkedInAccessTokenEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank linkedin account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank linkedin account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadLinkedInUserInfo(ctx context.Context, accessToken string) (LinkedInUserInfoReadback, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return LinkedInUserInfoReadback{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("User-Agent", "picobot/frank-v3")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return LinkedInUserInfoReadback{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return LinkedInUserInfoReadback{}, fmt.Errorf("linkedin userinfo read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Sub           string `json:"sub"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return LinkedInUserInfoReadback{}, fmt.Errorf("decode linkedin userinfo read-back: %w", err)
	}
	return LinkedInUserInfoReadback{
		LinkedInSub:   strings.TrimSpace(payload.Sub),
		Name:          strings.TrimSpace(payload.Name),
		PictureURL:    strings.TrimSpace(payload.Picture),
		Email:         strings.TrimSpace(payload.Email),
		EmailVerified: cloneBoolPtr(payload.EmailVerified),
	}, nil
}
