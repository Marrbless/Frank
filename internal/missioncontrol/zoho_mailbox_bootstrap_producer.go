package missioncontrol

import (
	"bytes"
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

var frankZohoMailboxBootstrapAPIBase = "https://mail.zoho.com/api"

type frankZohoMailboxBootstrapHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
	return &http.Client{Timeout: 30 * time.Second}
}

type frankZohoMailboxBootstrapStatus struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

type frankZohoMailboxBootstrapReadResponse struct {
	Status frankZohoMailboxBootstrapStatus     `json:"status"`
	Data   frankZohoMailboxBootstrapAccountDTO `json:"data"`
}

type frankZohoMailboxBootstrapCreateResponse struct {
	Status frankZohoMailboxBootstrapStatus `json:"status"`
}

type frankZohoMailboxBootstrapCreateRequest struct {
	PrimaryEmailAddress string `json:"primaryEmailAddress"`
	DisplayName         string `json:"displayName,omitempty"`
	Password            string `json:"password"`
}

type frankZohoMailboxBootstrapAccountDTO struct {
	AccountID           frankZohoMailboxBootstrapFlexibleString `json:"accountId"`
	PrimaryEmailAddress string                                  `json:"primaryEmailAddress"`
	MailboxAddress      string                                  `json:"mailboxAddress"`
}

type frankZohoMailboxBootstrapFlexibleString string

func (s *frankZohoMailboxBootstrapFlexibleString) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	switch {
	case trimmed == "", trimmed == "null":
		*s = ""
		return nil
	case strings.HasPrefix(trimmed, `"`):
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = frankZohoMailboxBootstrapFlexibleString(strings.TrimSpace(v))
		return nil
	default:
		*s = frankZohoMailboxBootstrapFlexibleString(trimmed)
		return nil
	}
}

// ProduceFrankZohoMailboxBootstrap is the single missioncontrol-owned
// execution producer for provider-specific Zoho mailbox bootstrap. It resolves
// only from the committed Frank identity/account pair, reads Zoho Mail admin
// state before create, creates only when the mailbox is absent, and marks the
// Frank account committed only after a provider read-back confirms the mailbox.
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
	if pair.Account.ZohoMailbox.ConfirmedCreated && strings.TrimSpace(pair.Account.ZohoMailbox.ProviderAccountID) != "" {
		return nil
	}

	providerAccountID, err := reconcileFrankZohoMailboxBootstrap(pair)
	if err != nil {
		return err
	}

	updated := pair.Account
	zohoMailbox := *updated.ZohoMailbox
	zohoMailbox.ProviderAccountID = providerAccountID
	zohoMailbox.ConfirmedCreated = true
	updated.ZohoMailbox = &zohoMailbox
	updated.UpdatedAt = now
	if updated.UpdatedAt.Before(updated.CreatedAt) {
		updated.UpdatedAt = updated.CreatedAt
	}
	return StoreFrankAccountRecord(root, updated)
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

func reconcileFrankZohoMailboxBootstrap(pair ResolvedExecutionContextFrankZohoMailboxBootstrapPair) (string, error) {
	mailboxAddress := strings.TrimSpace(pair.Identity.ZohoMailbox.FromAddress)
	if mailboxAddress == "" {
		return "", fmt.Errorf("mission store Frank zoho mailbox bootstrap identity %q requires committed zoho_mailbox.from_address", pair.Identity.IdentityID)
	}
	adminToken, err := loadFrankZohoMailboxBootstrapEnvRef(pair.Account.AccountID, pair.Account.ZohoMailbox.AdminOAuthTokenEnvVarRef, "zoho_mailbox.admin_oauth_token_env_var_ref")
	if err != nil {
		return "", err
	}

	client := newFrankZohoMailboxBootstrapHTTPClient()
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	accountData, found, err := readFrankZohoMailboxBootstrapAccount(context.Background(), client, pair.Account.AccountID, pair.Account.ZohoMailbox.OrganizationID, adminToken, mailboxAddress)
	if err != nil {
		return "", err
	}
	if found {
		return providerAccountIDFromFrankZohoMailboxBootstrapRead(pair.Account.AccountID, mailboxAddress, accountData)
	}

	password, err := loadFrankZohoMailboxBootstrapEnvRef(pair.Account.AccountID, pair.Account.ZohoMailbox.BootstrapPasswordEnvVarRef, "zoho_mailbox.bootstrap_password_env_var_ref")
	if err != nil {
		return "", err
	}
	if err := createFrankZohoMailboxBootstrapAccount(context.Background(), client, pair, adminToken, password); err != nil {
		return "", err
	}

	accountData, found, err = readFrankZohoMailboxBootstrapAccount(context.Background(), client, pair.Account.AccountID, pair.Account.ZohoMailbox.OrganizationID, adminToken, mailboxAddress)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q create may have succeeded but provider read-back could not confirm mailbox %q",
			pair.Account.AccountID,
			mailboxAddress,
		)
	}
	return providerAccountIDFromFrankZohoMailboxBootstrapRead(pair.Account.AccountID, mailboxAddress, accountData)
}

func loadFrankZohoMailboxBootstrapEnvRef(accountID, envVarRef, field string) (string, error) {
	envVarRef = strings.TrimSpace(envVarRef)
	if envVarRef == "" {
		return "", fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q requires committed %s", accountID, field)
	}
	value, ok := os.LookupEnv(envVarRef)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q env var %q referenced by %s is unset or empty",
			accountID,
			envVarRef,
			field,
		)
	}
	return strings.TrimSpace(value), nil
}

func readFrankZohoMailboxBootstrapAccount(ctx context.Context, client frankZohoMailboxBootstrapHTTPDoer, accountID, organizationID, adminToken, mailboxAddress string) (frankZohoMailboxBootstrapAccountDTO, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, frankZohoMailboxBootstrapAccountReadURL(organizationID, mailboxAddress), nil)
	if err != nil {
		return frankZohoMailboxBootstrapAccountDTO{}, false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Zoho-oauthtoken "+adminToken)

	resp, err := client.Do(req)
	if err != nil {
		return frankZohoMailboxBootstrapAccountDTO{}, false, fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider read failed: %w", accountID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return frankZohoMailboxBootstrapAccountDTO{}, false, fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider read failed to read response: %w", accountID, err)
	}

	var decoded frankZohoMailboxBootstrapReadResponse
	if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return frankZohoMailboxBootstrapAccountDTO{}, false, fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider read failed to decode response: %w", accountID, err)
		}
	}

	if resp.StatusCode == http.StatusNotFound || decoded.Status.Code == http.StatusNotFound {
		return frankZohoMailboxBootstrapAccountDTO{}, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return frankZohoMailboxBootstrapAccountDTO{}, false, fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider read returned HTTP %d",
			accountID,
			resp.StatusCode,
		)
	}
	if decoded.Status.Code != 0 && decoded.Status.Code != 200 {
		return frankZohoMailboxBootstrapAccountDTO{}, false, fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider read status %d: %s",
			accountID,
			decoded.Status.Code,
			strings.TrimSpace(decoded.Status.Description),
		)
	}
	return decoded.Data, true, nil
}

func createFrankZohoMailboxBootstrapAccount(ctx context.Context, client frankZohoMailboxBootstrapHTTPDoer, pair ResolvedExecutionContextFrankZohoMailboxBootstrapPair, adminToken, password string) error {
	payload := frankZohoMailboxBootstrapCreateRequest{
		PrimaryEmailAddress: strings.TrimSpace(pair.Identity.ZohoMailbox.FromAddress),
		DisplayName:         firstFrankZohoMailboxBootstrapNonEmpty(pair.Identity.ZohoMailbox.FromDisplayName, pair.Identity.DisplayName),
		Password:            password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, frankZohoMailboxBootstrapAccountsURL(pair.Account.ZohoMailbox.OrganizationID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Zoho-oauthtoken "+adminToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider create failed: %w", pair.Account.AccountID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider create failed to read response: %w", pair.Account.AccountID, err)
	}

	var decoded frankZohoMailboxBootstrapCreateResponse
	if trimmed := strings.TrimSpace(string(responseBody)); trimmed != "" {
		if err := json.Unmarshal(responseBody, &decoded); err != nil {
			return fmt.Errorf("mission store Frank zoho mailbox bootstrap account %q provider create failed to decode response: %w", pair.Account.AccountID, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider create returned HTTP %d",
			pair.Account.AccountID,
			resp.StatusCode,
		)
	}
	if decoded.Status.Code != 0 && decoded.Status.Code != 200 {
		return fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider create status %d: %s",
			pair.Account.AccountID,
			decoded.Status.Code,
			strings.TrimSpace(decoded.Status.Description),
		)
	}
	return nil
}

func providerAccountIDFromFrankZohoMailboxBootstrapRead(accountID, mailboxAddress string, data frankZohoMailboxBootstrapAccountDTO) (string, error) {
	confirmedMailboxAddress := strings.TrimSpace(firstFrankZohoMailboxBootstrapNonEmpty(data.PrimaryEmailAddress, data.MailboxAddress))
	if confirmedMailboxAddress == "" {
		return "", fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider read-back missing mailbox email for committed identity %q",
			accountID,
			mailboxAddress,
		)
	}
	if !strings.EqualFold(confirmedMailboxAddress, mailboxAddress) {
		return "", fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider read-back email %q does not match committed identity email %q",
			accountID,
			confirmedMailboxAddress,
			mailboxAddress,
		)
	}
	providerAccountID := strings.TrimSpace(string(data.AccountID))
	if providerAccountID == "" {
		return "", fmt.Errorf(
			"mission store Frank zoho mailbox bootstrap account %q provider read-back missing accountId for mailbox %q",
			accountID,
			mailboxAddress,
		)
	}
	return providerAccountID, nil
}

func frankZohoMailboxBootstrapAccountsURL(organizationID string) string {
	return strings.TrimRight(frankZohoMailboxBootstrapAPIBase, "/") + "/organization/" + url.PathEscape(strings.TrimSpace(organizationID)) + "/accounts"
}

func frankZohoMailboxBootstrapAccountReadURL(organizationID, mailboxAddress string) string {
	return frankZohoMailboxBootstrapAccountsURL(organizationID) + "/" + url.PathEscape(strings.TrimSpace(mailboxAddress))
}

func firstFrankZohoMailboxBootstrapNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
