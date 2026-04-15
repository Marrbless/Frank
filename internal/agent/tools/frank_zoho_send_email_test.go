package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)

	reg := NewRegistry()
	reg.Register(tool)
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())

	ec := testFrankZohoSendExecutionContext(root, campaign.CampaignID, tool.Name())

	out, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
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
		t.Fatalf("toAddress = %#v, want campaign-owned joined recipients", gotBody["toAddress"])
	}
	if gotBody["ccAddress"] != "copy@example.com" {
		t.Fatalf("ccAddress = %#v, want campaign-owned joined cc recipient", gotBody["ccAddress"])
	}
	if gotBody["bccAddress"] != "blind@example.com" {
		t.Fatalf("bccAddress = %#v, want campaign-owned joined bcc recipient", gotBody["bccAddress"])
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

func TestFrankZohoSendEmailToolRejectsCallerOwnedRecipientArgs(t *testing.T) {
	t.Parallel()

	tool := NewFrankZohoSendEmailTool()
	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)

	reg := NewRegistry()
	reg.Register(tool)
	reg.SetGuard(missioncontrol.NewDefaultToolGuard())

	ec := testFrankZohoSendExecutionContext(root, campaign.CampaignID, tool.Name())

	_, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
			"to":      []interface{}{"override@example.com"},
			"subject": "Frank intro",
			"body":    "Hello from Frank",
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want recipient override rejection")
	}
	if !strings.Contains(err.Error(), `"to" is campaign-owned and must come from step.campaign_ref zoho_email_addressing`) {
		t.Fatalf("Execute() error = %q, want campaign-owned recipient rejection", err)
	}
}

func TestFrankZohoCampaignSendPrepareFinalizeAndReplaySafety(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	tool := NewFrankZohoSendEmailTool()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-send-test"}, job, runtime, control, time.Now().UTC())
	})

	job := missioncontrol.Job{
		ID:           "job-frank-zoho-send-persist",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-persist",
			Steps: []missioncontrol.Step{
				{
					ID:                "send-outbound-email",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{tool.Name()},
					CampaignRef:       &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID},
				},
				{
					ID:                "final-response",
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	if err := state.ActivateStep(job, "send-outbound-email"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	args := map[string]interface{}{
		"subject": "Frank intro",
		"body":    "Hello from Frank",
	}

	if _, skip, err := state.PrepareFrankZohoCampaignSend(args); err != nil {
		t.Fatalf("PrepareFrankZohoCampaignSend(first) error = %v", err)
	} else if skip {
		t.Fatal("PrepareFrankZohoCampaignSend(first) skip = true, want new prepared action")
	}

	preparedRecords, err := missioncontrol.ListCommittedCampaignZohoEmailOutboundActionRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(prepared) error = %v", err)
	}
	if len(preparedRecords) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(prepared) len = %d, want 1", len(preparedRecords))
	}
	if preparedRecords[0].State != string(missioncontrol.CampaignZohoEmailOutboundActionStatePrepared) {
		t.Fatalf("prepared state = %q, want prepared", preparedRecords[0].State)
	}
	if preparedRecords[0].Addressing.To[0] != "person@example.com" {
		t.Fatalf("prepared Addressing.To[0] = %q, want campaign-owned recipient", preparedRecords[0].Addressing.To[0])
	}

	if _, skip, err := state.PrepareFrankZohoCampaignSend(args); err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) error = nil, want prepared replay rejection")
	} else if !skip {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) skip = false, want replay blocked")
	} else if !strings.Contains(err.Error(), "already prepared without provider receipt proof") {
		t.Fatalf("PrepareFrankZohoCampaignSend(replay) error = %q, want prepared replay rejection", err)
	}

	receiptJSON := `{
		"provider": "zoho_mail",
		"provider_account_id": "3323462000000008002",
		"from_address": "frank@omou.online",
		"from_display_name": "Frank",
		"provider_message_id": "1711540357880100000",
		"provider_mail_id": "<mail-1@zoho.test>",
		"mime_message_id": "<mime-1@example.test>",
		"original_message_url": "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage"
	}`
	if err := state.RecordFrankZohoCampaignSend(args, receiptJSON); err != nil {
		t.Fatalf("RecordFrankZohoCampaignSend() error = %v", err)
	}

	receiptRecords, err := missioncontrol.ListCommittedFrankZohoSendReceiptRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedFrankZohoSendReceiptRecords() error = %v", err)
	}
	if len(receiptRecords) != 1 {
		t.Fatalf("ListCommittedFrankZohoSendReceiptRecords() len = %d, want 1", len(receiptRecords))
	}

	sentRecords, err := missioncontrol.ListCommittedCampaignZohoEmailOutboundActionRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(sent) error = %v", err)
	}
	if len(sentRecords) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(sent) len = %d, want 1", len(sentRecords))
	}
	if sentRecords[0].State != string(missioncontrol.CampaignZohoEmailOutboundActionStateSent) {
		t.Fatalf("sent state = %q, want sent", sentRecords[0].State)
	}
	if sentRecords[0].ProviderMessageID != "1711540357880100000" {
		t.Fatalf("sent ProviderMessageID = %q, want canonical Zoho message id", sentRecords[0].ProviderMessageID)
	}

	gotReceipt, skip, err := state.PrepareFrankZohoCampaignSend(args)
	if err != nil {
		t.Fatalf("PrepareFrankZohoCampaignSend(sent replay) error = %v", err)
	}
	if !skip {
		t.Fatal("PrepareFrankZohoCampaignSend(sent replay) skip = false, want sent replay short-circuit")
	}
	var replayReceipt FrankZohoSendReceipt
	if err := json.Unmarshal([]byte(gotReceipt), &replayReceipt); err != nil {
		t.Fatalf("json.Unmarshal(replay receipt) error = %v", err)
	}
	if replayReceipt.ProviderMessageID != "1711540357880100000" {
		t.Fatalf("replay receipt ProviderMessageID = %q, want canonical Zoho message id", replayReceipt.ProviderMessageID)
	}
}

func TestFrankZohoSendProofVerifierUsesZohoBearerTokenAndFetchesOriginalMessage(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotAuth string

	verifier := NewFrankZohoSendProofVerifier()
	verifier.accessToken = func(context.Context) (string, error) {
		return "test-zoho-token", nil
	}
	verifier.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"message/rfc822"}},
				Body:       io.NopCloser(strings.NewReader("From: Frank <frank@omou.online>\r\nSubject: Frank intro\r\n\r\nHello from Frank")),
			}, nil
		}),
	}

	proof := []missioncontrol.OperatorFrankZohoSendProofStatus{
		{
			ProviderMessageID:  "1711540357880100000",
			ProviderMailID:     "<mail-1@zoho.test>",
			MIMEMessageID:      "<mime-1@example.test>",
			ProviderAccountID:  "3323462000000008002",
			OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
		},
	}

	got, err := verifier.Verify(context.Background(), proof)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("request method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("request path = %q, want proof originalmessage path", gotPath)
	}
	if gotAuth != "Zoho-oauthtoken test-zoho-token" {
		t.Fatalf("Authorization = %q, want Zoho OAuth header", gotAuth)
	}
	if len(got) != 1 {
		t.Fatalf("len(verification records) = %d, want 1", len(got))
	}
	if got[0].ProviderMessageID != "1711540357880100000" {
		t.Fatalf("ProviderMessageID = %q, want committed proof locator", got[0].ProviderMessageID)
	}
	if got[0].ProviderMailID != "<mail-1@zoho.test>" {
		t.Fatalf("ProviderMailID = %q, want committed proof locator", got[0].ProviderMailID)
	}
	if got[0].MIMEMessageID != "<mime-1@example.test>" {
		t.Fatalf("MIMEMessageID = %q, want committed proof locator", got[0].MIMEMessageID)
	}
	if got[0].ProviderAccountID != "3323462000000008002" {
		t.Fatalf("ProviderAccountID = %q, want committed proof locator", got[0].ProviderAccountID)
	}
	if got[0].OriginalMessageURL != "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("OriginalMessageURL = %q, want committed proof locator", got[0].OriginalMessageURL)
	}
	if got[0].OriginalMessage != "From: Frank <frank@omou.online>\r\nSubject: Frank intro\r\n\r\nHello from Frank" {
		t.Fatalf("OriginalMessage = %q, want raw original message payload", got[0].OriginalMessage)
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

func mustStoreFrankZohoAddressedCampaignFixture(t *testing.T, root string, container missioncontrol.FrankContainerRecord) missioncontrol.CampaignRecord {
	t.Helper()

	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	campaign.ZohoEmailAddressing = &missioncontrol.CampaignZohoEmailAddressing{
		To:  []string{"person@example.com", "team@example.com"},
		CC:  []string{"copy@example.com"},
		BCC: []string{"blind@example.com"},
	}
	campaign.UpdatedAt = campaign.UpdatedAt.Add(time.Minute)
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord(addressed) error = %v", err)
	}
	return campaign
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
