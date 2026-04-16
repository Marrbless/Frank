package missioncontrol

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProduceFrankZohoMailboxBootstrapSuccessfulCommittedConfirmation(t *testing.T) {
	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	now := time.Date(2026, 4, 16, 16, 0, 0, 0, time.UTC)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}

	originalClient := newFrankZohoMailboxBootstrapHTTPClient
	originalBase := frankZohoMailboxBootstrapAPIBase
	defer func() {
		newFrankZohoMailboxBootstrapHTTPClient = originalClient
		frankZohoMailboxBootstrapAPIBase = originalBase
	}()

	var requests atomic.Int32
	newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
		return &http.Client{
			Transport: frankZohoMailboxBootstrapRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				return nil, fmt.Errorf("unexpected provider request %s %s", r.Method, r.URL.String())
			}),
		}
	}

	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, now); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("provider requests = %d, want 0 for committed-confirmed replay-safe no-op", got)
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
	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}

	originalClient := newFrankZohoMailboxBootstrapHTTPClient
	defer func() { newFrankZohoMailboxBootstrapHTTPClient = originalClient }()

	var requests atomic.Int32
	newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
		return &http.Client{
			Transport: frankZohoMailboxBootstrapRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				return nil, fmt.Errorf("unexpected provider request %s %s", r.Method, r.URL.String())
			}),
		}
	}

	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 10, 0, 0, time.UTC)); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap(first) error = %v", err)
	}
	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 11, 0, 0, time.UTC)); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap(replay) error = %v, want deterministic no-op success", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("provider requests = %d, want 0 across replay", got)
	}
}

func TestProduceFrankZohoMailboxBootstrapReconcilesExistingMailboxBeforeCreate(t *testing.T) {
	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}
	pair.Account.ZohoMailbox = &FrankZohoMailboxAccount{
		OrganizationID:             "zoid-123",
		AdminOAuthTokenEnvVarRef:   "PICOBOT_ZOHO_MAIL_ADMIN_TOKEN",
		BootstrapPasswordEnvVarRef: "PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD",
		ProviderAccountID:          "",
		ConfirmedCreated:           false,
	}
	if err := StoreFrankAccountRecord(fixtures.root, pair.Account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	t.Setenv("PICOBOT_ZOHO_MAIL_ADMIN_TOKEN", "test-admin-token")
	t.Setenv("PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD", "test-bootstrap-password")

	originalClient := newFrankZohoMailboxBootstrapHTTPClient
	originalBase := frankZohoMailboxBootstrapAPIBase
	defer func() {
		newFrankZohoMailboxBootstrapHTTPClient = originalClient
		frankZohoMailboxBootstrapAPIBase = originalBase
	}()
	frankZohoMailboxBootstrapAPIBase = "https://mail.zoho.test/api"

	var gotMethod string
	var gotPath string
	var gotAuth string
	var requests atomic.Int32
	newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
		return &http.Client{
			Transport: frankZohoMailboxBootstrapRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				return frankZohoMailboxBootstrapJSONResponse(http.StatusOK, `{
					"status": {"code": 200, "description": "success"},
					"data": {
						"accountId": 9988776655443322110,
						"primaryEmailAddress": "frank@example.com"
					}
				}`), nil
			}),
		}
	}

	now := time.Date(2026, 4, 16, 16, 15, 0, 0, time.UTC)
	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, now); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %v", err)
	}

	if got := requests.Load(); got != 1 {
		t.Fatalf("provider requests = %d, want 1 read-before-create reconciliation request", got)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("provider read method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/organization/zoid-123/accounts/frank@example.com" && gotPath != "/api/organization/zoid-123/accounts/frank%40example.com" {
		t.Fatalf("provider read path = %q, want committed mailbox read path", gotPath)
	}
	if gotAuth != "Zoho-oauthtoken test-admin-token" {
		t.Fatalf("provider read Authorization = %q, want Zoho OAuth header", gotAuth)
	}

	account, err := LoadFrankAccountRecord(fixtures.root, pair.Account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if account.ZohoMailbox == nil {
		t.Fatal("LoadFrankAccountRecord().ZohoMailbox = nil, want reconciled account")
	}
	if !account.ZohoMailbox.ConfirmedCreated {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox.ConfirmedCreated = false, want true after read-before-create reconciliation: %#v", account.ZohoMailbox)
	}
	if account.ZohoMailbox.ProviderAccountID != "9988776655443322110" {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox.ProviderAccountID = %q, want provider-confirmed accountId", account.ZohoMailbox.ProviderAccountID)
	}
	if account.UpdatedAt != now {
		t.Fatalf("LoadFrankAccountRecord().UpdatedAt = %s, want %s", account.UpdatedAt.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	}
}

func TestProduceFrankZohoMailboxBootstrapCreatesMailboxAndConfirmsReadBack(t *testing.T) {
	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}
	pair.Account.ZohoMailbox = &FrankZohoMailboxAccount{
		OrganizationID:             "zoid-123",
		AdminOAuthTokenEnvVarRef:   "PICOBOT_ZOHO_MAIL_ADMIN_TOKEN",
		BootstrapPasswordEnvVarRef: "PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD",
	}
	if err := StoreFrankAccountRecord(fixtures.root, pair.Account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	t.Setenv("PICOBOT_ZOHO_MAIL_ADMIN_TOKEN", "test-admin-token")
	t.Setenv("PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD", "test-bootstrap-password")

	originalClient := newFrankZohoMailboxBootstrapHTTPClient
	originalBase := frankZohoMailboxBootstrapAPIBase
	defer func() {
		newFrankZohoMailboxBootstrapHTTPClient = originalClient
		frankZohoMailboxBootstrapAPIBase = originalBase
	}()
	frankZohoMailboxBootstrapAPIBase = "https://mail.zoho.test/api"

	var requestCount int
	var createPayload frankZohoMailboxBootstrapCreateRequest
	newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
		return &http.Client{
			Transport: frankZohoMailboxBootstrapRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestCount++
				if r.Header.Get("Authorization") != "Zoho-oauthtoken test-admin-token" {
					t.Fatalf("Authorization = %q, want Zoho OAuth header", r.Header.Get("Authorization"))
				}
				switch requestCount {
				case 1:
					if r.Method != http.MethodGet {
						t.Fatalf("request 1 method = %q, want GET", r.Method)
					}
					return frankZohoMailboxBootstrapJSONResponse(http.StatusNotFound, `{
						"status": {"code": 404, "description": "not found"}
					}`), nil
				case 2:
					if r.Method != http.MethodPost {
						t.Fatalf("request 2 method = %q, want POST", r.Method)
					}
					if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
						t.Fatalf("json.Decode(create body) error = %v", err)
					}
					return frankZohoMailboxBootstrapJSONResponse(http.StatusOK, `{
						"status": {"code": 200, "description": "success"}
					}`), nil
				case 3:
					if r.Method != http.MethodGet {
						t.Fatalf("request 3 method = %q, want GET", r.Method)
					}
					return frankZohoMailboxBootstrapJSONResponse(http.StatusOK, `{
						"status": {"code": 200, "description": "success"},
						"data": {
							"accountId": "9988776655443322110",
							"primaryEmailAddress": "frank@example.com"
						}
					}`), nil
				default:
					t.Fatalf("unexpected provider request %d: %s %s", requestCount, r.Method, r.URL.String())
					return nil, fmt.Errorf("unexpected provider request")
				}
			}),
		}
	}

	now := time.Date(2026, 4, 16, 16, 20, 0, 0, time.UTC)
	if err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, now); err != nil {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %v", err)
	}
	if requestCount != 3 {
		t.Fatalf("provider requests = %d, want 3 (read, create, read-back)", requestCount)
	}
	if createPayload.PrimaryEmailAddress != "frank@example.com" {
		t.Fatalf("create payload primaryEmailAddress = %q, want committed mailbox address", createPayload.PrimaryEmailAddress)
	}
	if createPayload.DisplayName != "Frank" {
		t.Fatalf("create payload displayName = %q, want committed Frank mailbox display name", createPayload.DisplayName)
	}
	if createPayload.Password != "test-bootstrap-password" {
		t.Fatalf("create payload password = %q, want bootstrap password env value", createPayload.Password)
	}

	account, err := LoadFrankAccountRecord(fixtures.root, pair.Account.AccountID)
	if err != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", err)
	}
	if account.ZohoMailbox == nil || !account.ZohoMailbox.ConfirmedCreated || account.ZohoMailbox.ProviderAccountID != "9988776655443322110" {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox = %#v, want provider-confirmed committed account after create", account.ZohoMailbox)
	}
}

func TestProduceFrankZohoMailboxBootstrapFailsClosedWhenCreateConfirmationUnavailable(t *testing.T) {
	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	pair := ResolvedExecutionContextFrankZohoMailboxBootstrapPair{
		Identity: fixtures.identity,
		Account:  fixtures.account,
	}
	pair.Account.ZohoMailbox = &FrankZohoMailboxAccount{
		OrganizationID:             "zoid-123",
		AdminOAuthTokenEnvVarRef:   "PICOBOT_ZOHO_MAIL_ADMIN_TOKEN",
		BootstrapPasswordEnvVarRef: "PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD",
	}
	if err := StoreFrankAccountRecord(fixtures.root, pair.Account); err != nil {
		t.Fatalf("StoreFrankAccountRecord() error = %v", err)
	}

	t.Setenv("PICOBOT_ZOHO_MAIL_ADMIN_TOKEN", "test-admin-token")
	t.Setenv("PICOBOT_ZOHO_MAIL_BOOTSTRAP_PASSWORD", "test-bootstrap-password")

	originalClient := newFrankZohoMailboxBootstrapHTTPClient
	originalBase := frankZohoMailboxBootstrapAPIBase
	defer func() {
		newFrankZohoMailboxBootstrapHTTPClient = originalClient
		frankZohoMailboxBootstrapAPIBase = originalBase
	}()
	frankZohoMailboxBootstrapAPIBase = "https://mail.zoho.test/api"

	var requestCount int
	newFrankZohoMailboxBootstrapHTTPClient = func() frankZohoMailboxBootstrapHTTPDoer {
		return &http.Client{
			Transport: frankZohoMailboxBootstrapRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestCount++
				switch requestCount {
				case 1:
					return frankZohoMailboxBootstrapJSONResponse(http.StatusNotFound, `{
						"status": {"code": 404, "description": "not found"}
					}`), nil
				case 2:
					return frankZohoMailboxBootstrapJSONResponse(http.StatusOK, `{
						"status": {"code": 200, "description": "success"}
					}`), nil
				case 3:
					return frankZohoMailboxBootstrapJSONResponse(http.StatusNotFound, `{
						"status": {"code": 404, "description": "not found"}
					}`), nil
				default:
					t.Fatalf("unexpected provider request %d: %s %s", requestCount, r.Method, r.URL.String())
					return nil, fmt.Errorf("unexpected provider request")
				}
			}),
		}
	}

	err := ProduceFrankZohoMailboxBootstrap(fixtures.root, pair, time.Date(2026, 4, 16, 16, 25, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ProduceFrankZohoMailboxBootstrap() error = nil, want read-back confirmation failure")
	}
	if !strings.Contains(err.Error(), `mission store Frank zoho mailbox bootstrap account "account-mail" create may have succeeded but provider read-back could not confirm mailbox "frank@example.com"`) {
		t.Fatalf("ProduceFrankZohoMailboxBootstrap() error = %q, want read-back confirmation failure", err.Error())
	}

	account, loadErr := LoadFrankAccountRecord(fixtures.root, pair.Account.AccountID)
	if loadErr != nil {
		t.Fatalf("LoadFrankAccountRecord() error = %v", loadErr)
	}
	if account.ZohoMailbox == nil {
		t.Fatal("LoadFrankAccountRecord().ZohoMailbox = nil, want unchanged unconfirmed account")
	}
	if account.ZohoMailbox.ConfirmedCreated {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox.ConfirmedCreated = true, want fail-closed unconfirmed state: %#v", account.ZohoMailbox)
	}
	if account.ZohoMailbox.ProviderAccountID != "" {
		t.Fatalf("LoadFrankAccountRecord().ZohoMailbox.ProviderAccountID = %q, want empty on failed confirmation", account.ZohoMailbox.ProviderAccountID)
	}
}

func TestProduceFrankZohoMailboxBootstrapFailsClosedWhenEligibilityIsNotAutonomyCompatible(t *testing.T) {
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

type frankZohoMailboxBootstrapRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn frankZohoMailboxBootstrapRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func frankZohoMailboxBootstrapJSONResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
