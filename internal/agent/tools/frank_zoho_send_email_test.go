package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestFrankZohoSendEmailToolApprovalGateBlocksSend(t *testing.T) {
	t.Parallel()

	var sendCalls atomic.Int32
	tool := NewFrankZohoSendEmailTool()
	tool.send = func(ctx context.Context, req frankZohoSendRequest) (frankZohoSendResponseData, error) {
		sendCalls.Add(1)
		return frankZohoSendResponseData{}, nil
	}

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)

	reg := NewRegistry()
	reg.Register(tool)
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())

	ec := testFrankZohoSendExecutionContext(root, campaign.CampaignID, tool.Name())
	ec.Step.RequiresApproval = true

	_, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
			"to":      []interface{}{"person@example.com"},
			"subject": "Hello",
			"body":    "World",
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want approval rejection")
	}
	if !strings.Contains(err.Error(), "E_APPROVAL_REQUIRED") {
		t.Fatalf("Execute() error = %q, want E_APPROVAL_REQUIRED", err)
	}
	if !strings.Contains(err.Error(), "step requires approval") {
		t.Fatalf("Execute() error = %q, want approval reason", err)
	}
	if got := sendCalls.Load(); got != 0 {
		t.Fatalf("send calls = %d, want 0", got)
	}
}

func TestFrankZohoSendEmailToolCampaignGateBlocksSend(t *testing.T) {
	t.Parallel()

	var sendCalls atomic.Int32
	tool := NewFrankZohoSendEmailTool()
	tool.send = func(ctx context.Context, req frankZohoSendRequest) (frankZohoSendResponseData, error) {
		sendCalls.Add(1)
		return frankZohoSendResponseData{}, nil
	}

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	campaign.State = missioncontrol.CampaignStateDraft
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	reg := NewRegistry()
	reg.Register(tool)
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())

	ec := testFrankZohoSendExecutionContext(root, campaign.CampaignID, tool.Name())

	_, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
			"to":      []interface{}{"person@example.com"},
			"subject": "Hello",
			"body":    "World",
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want campaign readiness rejection")
	}
	if !strings.Contains(err.Error(), "E_STEP_OUT_OF_ORDER") {
		t.Fatalf("Execute() error = %q, want E_STEP_OUT_OF_ORDER", err)
	}
	if !strings.Contains(err.Error(), `campaign readiness requires state "active"; got "draft"`) {
		t.Fatalf("Execute() error = %q, want campaign readiness reason", err)
	}
	if got := sendCalls.Load(); got != 0 {
		t.Fatalf("send calls = %d, want 0", got)
	}
}

func TestFrankZohoSendEmailToolUsesFixedFrankZohoAccountAndMapsReceipt(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotPath string
	var gotMethod string
	var gotBody map[string]interface{}

	tool := NewFrankZohoSendEmailTool()
	tool.apiBase = "https://mail.zoho.test/api"
	tool.accessToken = func(context.Context) (string, error) {
		return "test-zoho-token", nil
	}
	tool.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("Decode(request body) error = %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{
					"status": {"code": 200, "description": "success"},
					"data": {
						"messageId": 1711540357880100000,
						"mailId": "<18e7fc14ae8.11615619c4.6313519471843772533@140.com>"
					}
				}`)),
			}, nil
		}),
	}

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)

	reg := NewRegistry()
	reg.Register(tool)
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())

	ec := testFrankZohoSendExecutionContext(root, campaign.CampaignID, tool.Name())

	out, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
			"to":          []interface{}{"person@example.com", "team@example.com"},
			"cc":          []interface{}{"copy@example.com"},
			"subject":     "Frank intro",
			"body":        "Hello from Frank",
			"body_format": "plaintext",
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("request method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/accounts/3323462000000008002/messages" {
		t.Fatalf("request path = %q, want fixed Zoho account path", gotPath)
	}
	if gotAuth != "Zoho-oauthtoken test-zoho-token" {
		t.Fatalf("Authorization = %q, want Zoho OAuth header", gotAuth)
	}
	if gotBody["fromAddress"] != "frank@omou.online" {
		t.Fatalf("fromAddress = %#v, want fixed Frank mailbox", gotBody["fromAddress"])
	}
	if gotBody["toAddress"] != "person@example.com,team@example.com" {
		t.Fatalf("toAddress = %#v, want joined recipients", gotBody["toAddress"])
	}
	if gotBody["ccAddress"] != "copy@example.com" {
		t.Fatalf("ccAddress = %#v, want joined cc recipient", gotBody["ccAddress"])
	}
	if gotBody["subject"] != "Frank intro" {
		t.Fatalf("subject = %#v, want %#v", gotBody["subject"], "Frank intro")
	}
	if gotBody["content"] != "Hello from Frank" {
		t.Fatalf("content = %#v, want %#v", gotBody["content"], "Hello from Frank")
	}
	if gotBody["mailFormat"] != "plaintext" {
		t.Fatalf("mailFormat = %#v, want %#v", gotBody["mailFormat"], "plaintext")
	}
	if gotBody["encoding"] != "UTF-8" {
		t.Fatalf("encoding = %#v, want %#v", gotBody["encoding"], "UTF-8")
	}

	var receipt FrankZohoSendReceipt
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatalf("Unmarshal(receipt) error = %v", err)
	}

	if receipt.Provider != "zoho_mail" {
		t.Fatalf("Provider = %q, want %q", receipt.Provider, "zoho_mail")
	}
	if receipt.ProviderAccountID != "3323462000000008002" {
		t.Fatalf("ProviderAccountID = %q, want fixed Zoho account ID", receipt.ProviderAccountID)
	}
	if receipt.FromAddress != "frank@omou.online" {
		t.Fatalf("FromAddress = %q, want fixed Frank mailbox", receipt.FromAddress)
	}
	if receipt.FromDisplayName != "Frank" {
		t.Fatalf("FromDisplayName = %q, want %q", receipt.FromDisplayName, "Frank")
	}
	if receipt.ProviderMessageID != "1711540357880100000" {
		t.Fatalf("ProviderMessageID = %q, want canonical Zoho messageId", receipt.ProviderMessageID)
	}
	if receipt.ProviderMailID != "<18e7fc14ae8.11615619c4.6313519471843772533@140.com>" {
		t.Fatalf("ProviderMailID = %q, want secondary Zoho mailId", receipt.ProviderMailID)
	}
	if receipt.MIMEMessageID != "" {
		t.Fatalf("MIMEMessageID = %q, want empty when send response omits it", receipt.MIMEMessageID)
	}
	wantOriginalMessageURL := "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage"
	if receipt.OriginalMessageURL != wantOriginalMessageURL {
		t.Fatalf("OriginalMessageURL = %q, want %q", receipt.OriginalMessageURL, wantOriginalMessageURL)
	}
}

func testFrankZohoSendExecutionContext(root string, campaignID string, toolName string) missioncontrol.ExecutionContext {
	job := &missioncontrol.Job{
		ID:           "job-frank-zoho-send",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{toolName},
	}
	step := &missioncontrol.Step{
		ID:                "send-outbound-email",
		Type:              missioncontrol.StepTypeSystemAction,
		RequiredAuthority: missioncontrol.AuthorityTierLow,
		AllowedTools:      []string{toolName},
		CampaignRef:       &missioncontrol.CampaignRef{CampaignID: campaignID},
	}
	return missioncontrol.ExecutionContext{
		Job:              job,
		Step:             step,
		MissionStoreRoot: root,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
