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

type GitHubAuthenticatedUser struct {
	GitHubUserID string
	Login        string
	NodeID       string
}

var readGitHubAuthenticatedUser = defaultReadGitHubAuthenticatedUser
var readGitHubPrimaryVerifiedEmail = defaultReadGitHubPrimaryVerifiedEmail

// ProduceFrankGitHubOnboarding is the single missioncontrol-owned execution
// producer for the GitHub onboarding lane. It confirms the configured
// authenticated GitHub identity through provider read-back only and persists
// only non-secret GitHub identifiers into the committed Frank identity/account
// records.
func ProduceFrankGitHubOnboarding(root string, bundle ResolvedExecutionContextFrankGitHubOnboardingBundle, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	bundle = normalizeResolvedExecutionContextFrankGitHubOnboardingBundle(bundle)
	if err := validateResolvedExecutionContextFrankGitHubOnboardingBundle(bundle); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Identity.EligibilityTargetRef); err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, bundle.Account.EligibilityTargetRef); err != nil {
		return err
	}

	token, err := loadFrankGitHubTokenEnvRef(bundle.Account.AccountID, bundle.Account.GitHub.TokenEnvVarRef, "github.token_env_var_ref")
	if err != nil {
		return err
	}

	readBack, err := readGitHubAuthenticatedUser(context.Background(), token)
	if err != nil {
		return fmt.Errorf("github onboarding provider read-back failed: %w", err)
	}
	readBack.GitHubUserID = strings.TrimSpace(readBack.GitHubUserID)
	readBack.Login = strings.TrimSpace(readBack.Login)
	readBack.NodeID = strings.TrimSpace(readBack.NodeID)
	if readBack.GitHubUserID == "" || readBack.Login == "" {
		return fmt.Errorf("github onboarding provider read-back did not return github_user_id and login")
	}

	primaryVerifiedEmail, emailAvailable, err := readGitHubPrimaryVerifiedEmail(context.Background(), token)
	if err != nil {
		return fmt.Errorf("github onboarding email read-back failed: %w", err)
	}
	primaryVerifiedEmail = strings.TrimSpace(primaryVerifiedEmail)

	committedUserID := ""
	committedLogin := ""
	committedNodeID := ""
	committedPrimaryVerifiedEmail := ""
	if bundle.Identity.GitHub != nil {
		committedUserID = strings.TrimSpace(bundle.Identity.GitHub.GitHubUserID)
		committedLogin = strings.TrimSpace(bundle.Identity.GitHub.Login)
		committedNodeID = strings.TrimSpace(bundle.Identity.GitHub.NodeID)
		committedPrimaryVerifiedEmail = strings.TrimSpace(bundle.Identity.GitHub.PrimaryVerifiedEmail)
	}
	if committedUserID != "" && committedUserID != readBack.GitHubUserID {
		return fmt.Errorf("github identity %q conflicts with provider github_user_id %q", bundle.Identity.IdentityID, readBack.GitHubUserID)
	}
	if committedLogin != "" && committedLogin != readBack.Login {
		return fmt.Errorf("github identity %q conflicts with provider login %q", bundle.Identity.IdentityID, readBack.Login)
	}
	if committedNodeID != "" && committedNodeID != readBack.NodeID {
		return fmt.Errorf("github identity %q conflicts with provider node_id %q", bundle.Identity.IdentityID, readBack.NodeID)
	}
	if emailAvailable && committedPrimaryVerifiedEmail != "" && committedPrimaryVerifiedEmail != primaryVerifiedEmail {
		return fmt.Errorf("github identity %q conflicts with provider primary_verified_email %q", bundle.Identity.IdentityID, primaryVerifiedEmail)
	}

	committedTokenEnvVarRef := ""
	committedConfirmedAuthenticated := false
	if bundle.Account.GitHub != nil {
		committedTokenEnvVarRef = strings.TrimSpace(bundle.Account.GitHub.TokenEnvVarRef)
		committedConfirmedAuthenticated = bundle.Account.GitHub.ConfirmedAuthenticated
	}

	if committedConfirmedAuthenticated &&
		committedUserID == readBack.GitHubUserID &&
		committedLogin == readBack.Login &&
		(committedNodeID == "" || committedNodeID == readBack.NodeID) &&
		(!emailAvailable || committedPrimaryVerifiedEmail == "" || committedPrimaryVerifiedEmail == primaryVerifiedEmail) {
		return nil
	}

	updatedIdentity := bundle.Identity
	if updatedIdentity.GitHub == nil {
		updatedIdentity.GitHub = &FrankGitHubIdentity{}
	}
	if updatedIdentity.GitHub.GitHubUserID == "" {
		updatedIdentity.GitHub.GitHubUserID = readBack.GitHubUserID
	}
	if updatedIdentity.GitHub.Login == "" {
		updatedIdentity.GitHub.Login = readBack.Login
	}
	if updatedIdentity.GitHub.NodeID == "" {
		updatedIdentity.GitHub.NodeID = readBack.NodeID
	}
	if !committedConfirmedAuthenticated && emailAvailable && updatedIdentity.GitHub.PrimaryVerifiedEmail == "" {
		updatedIdentity.GitHub.PrimaryVerifiedEmail = primaryVerifiedEmail
	}
	updatedIdentity.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedIdentity.CreatedAt, now)

	updatedAccount := bundle.Account
	if updatedAccount.GitHub == nil {
		updatedAccount.GitHub = &FrankGitHubAccount{}
	}
	if updatedAccount.GitHub.TokenEnvVarRef == "" {
		updatedAccount.GitHub.TokenEnvVarRef = committedTokenEnvVarRef
	}
	updatedAccount.GitHub.ConfirmedAuthenticated = true
	updatedAccount.UpdatedAt = updatedAtOnOrAfterCreatedAt(updatedAccount.CreatedAt, now)

	if err := StoreFrankIdentityRecord(root, updatedIdentity); err != nil {
		return err
	}
	return StoreFrankAccountRecord(root, updatedAccount)
}

func loadFrankGitHubTokenEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank github account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("mission store Frank github account %q env var %q referenced by %s is unset or empty", accountID, envVarRef, field)
	}
	return strings.TrimSpace(value), nil
}

func defaultReadGitHubAuthenticatedUser(ctx context.Context, token string) (GitHubAuthenticatedUser, error) {
	var payload struct {
		ID     json.Number `json:"id"`
		Login  string      `json:"login"`
		NodeID string      `json:"node_id"`
	}
	if err := doGitHubJSONRequest(ctx, token, "https://api.github.com/user", &payload); err != nil {
		return GitHubAuthenticatedUser{}, err
	}
	return GitHubAuthenticatedUser{
		GitHubUserID: strings.TrimSpace(payload.ID.String()),
		Login:        strings.TrimSpace(payload.Login),
		NodeID:       strings.TrimSpace(payload.NodeID),
	}, nil
}

func defaultReadGitHubPrimaryVerifiedEmail(ctx context.Context, token string) (string, bool, error) {
	req, err := newGitHubRequest(ctx, token, "https://api.github.com/user/emails")
	if err != nil {
		return "", false, err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", false, fmt.Errorf("github emails read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", true, fmt.Errorf("decode github emails read-back: %w", err)
	}
	for _, candidate := range payload {
		if candidate.Primary && candidate.Verified && strings.TrimSpace(candidate.Email) != "" {
			return strings.TrimSpace(candidate.Email), true, nil
		}
	}
	return "", true, nil
}

func doGitHubJSONRequest(ctx context.Context, token string, url string, target any) error {
	req, err := newGitHubRequest(ctx, token, url)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("github read-back returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode github read-back: %w", err)
	}
	return nil
}

func newGitHubRequest(ctx context.Context, token string, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("User-Agent", "picobot/frank-v3")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return req, nil
}
