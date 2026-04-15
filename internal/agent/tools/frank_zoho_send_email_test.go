package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestFrankZohoSendEmailToolFailsClosedOnUnsupportedCampaignStopCondition(t *testing.T) {
	t.Parallel()

	var sendCalls atomic.Int32
	tool := NewFrankZohoSendEmailTool()
	tool.send = func(ctx context.Context, req frankZohoSendRequest) (frankZohoSendResponseData, error) {
		sendCalls.Add(1)
		return frankZohoSendResponseData{}, nil
	}

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	campaign.StopConditions = []string{"stop after 3 opens"}
	campaign.UpdatedAt = campaign.UpdatedAt.Add(time.Minute)
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
			"subject": "Frank intro",
			"body":    "Hello from Frank",
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want unsupported stop-condition rejection")
	}
	if !strings.Contains(err.Error(), `campaign send gate is closed: campaign zoho email stop_condition "stop after 3 opens" is not evaluable from committed outbound and inbound reply records`) {
		t.Fatalf("Execute() error = %q, want unsupported stop-condition rejection", err)
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

func TestFrankZohoSendEmailToolClassifiesProviderDeclaredRejectionAsTerminalFailure(t *testing.T) {
	t.Parallel()

	tool := NewFrankZohoSendEmailTool()
	tool.apiBase = "https://mail.zoho.test/api"
	tool.accessToken = func(context.Context) (string, error) {
		return "test-zoho-token", nil
	}
	tool.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{
					"status": {"code": 1510, "description": "recipient rejected"},
					"data": {}
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
	_, err := reg.Execute(
		missioncontrol.WithExecutionContext(context.Background(), ec),
		tool.Name(),
		map[string]interface{}{
			"subject": "Frank intro",
			"body":    "Hello from Frank",
		},
	)
	if err == nil {
		t.Fatal("Execute() error = nil, want terminal provider rejection")
	}
	var terminal *frankZohoTerminalSendFailureError
	if !errors.As(err, &terminal) {
		t.Fatalf("Execute() error type = %T, want terminal provider rejection", err)
	}
	if terminal.Failure().ProviderStatusCode != 1510 {
		t.Fatalf("terminal failure provider status = %d, want 1510", terminal.Failure().ProviderStatusCode)
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
	originalVerify := verifyFrankZohoCampaignSendProof
	t.Cleanup(func() {
		verifyFrankZohoCampaignSendProof = originalVerify
	})

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

	verifyFrankZohoCampaignSendProof = func(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]FrankZohoSendProofVerification, error) {
		if len(proof) != 1 {
			t.Fatalf("len(proof) = %d, want 1", len(proof))
		}
		return []FrankZohoSendProofVerification{
			{
				ProviderMessageID:  proof[0].ProviderMessageID,
				ProviderMailID:     proof[0].ProviderMailID,
				MIMEMessageID:      proof[0].MIMEMessageID,
				ProviderAccountID:  proof[0].ProviderAccountID,
				OriginalMessageURL: proof[0].OriginalMessageURL,
				OriginalMessage: "From: Frank <frank@omou.online>\r\n" +
					"To: person@example.com, team@example.com\r\n" +
					"Cc: copy@example.com\r\n" +
					"Subject: Frank intro\r\n" +
					"\r\n" +
					"Hello from Frank",
			},
		}, nil
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

	verifiedRecords, err := missioncontrol.ListCommittedCampaignZohoEmailOutboundActionRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(verified) error = %v", err)
	}
	if len(verifiedRecords) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords(verified) len = %d, want 1", len(verifiedRecords))
	}
	if verifiedRecords[0].State != string(missioncontrol.CampaignZohoEmailOutboundActionStateVerified) {
		t.Fatalf("verified state = %q, want verified", verifiedRecords[0].State)
	}
	if verifiedRecords[0].VerifiedAt.IsZero() {
		t.Fatal("verified VerifiedAt = zero, want provider-mailbox finalize timestamp")
	}
}

func TestFrankZohoCampaignSendReplayStaysBlockedWhenProviderMailboxVerificationFails(t *testing.T) {
	originalVerify := verifyFrankZohoCampaignSendProof
	t.Cleanup(func() {
		verifyFrankZohoCampaignSendProof = originalVerify
	})

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	tool := NewFrankZohoSendEmailTool()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-send-test"}, job, runtime, control, time.Now().UTC())
	})

	job := missioncontrol.Job{
		ID:           "job-frank-zoho-send-blocked",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-blocked",
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

	verifyFrankZohoCampaignSendProof = func(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]FrankZohoSendProofVerification, error) {
		return nil, fmt.Errorf("zoho originalmessage unavailable")
	}

	_, skip, err := state.PrepareFrankZohoCampaignSend(args)
	if err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) error = nil, want verification block")
	}
	if !skip {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) skip = false, want resend blocked")
	}
	if !strings.Contains(err.Error(), "remains blocked until provider-mailbox verification/finalize succeeds") {
		t.Fatalf("PrepareFrankZohoCampaignSend(replay) error = %q, want provider verification block", err)
	}
}

func TestFrankZohoCampaignSendTerminalFailurePersistsAndBlocksReplay(t *testing.T) {
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
		ID:           "job-frank-zoho-send-terminal-failure",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-terminal-failure",
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
	if err := state.RecordFrankZohoCampaignSendFailure(args, &frankZohoTerminalSendFailureError{
		httpStatus: 400,
		failure: missioncontrol.CampaignZohoEmailOutboundFailure{
			HTTPStatus:                400,
			ProviderStatusCode:        1510,
			ProviderStatusDescription: "recipient rejected",
		},
	}); err != nil {
		t.Fatalf("RecordFrankZohoCampaignSendFailure() error = %v", err)
	}

	records, err := missioncontrol.ListCommittedCampaignZohoEmailOutboundActionRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords() len = %d, want 1", len(records))
	}
	if records[0].State != string(missioncontrol.CampaignZohoEmailOutboundActionStateFailed) {
		t.Fatalf("record state = %q, want failed", records[0].State)
	}
	if records[0].Failure.ProviderStatusCode != 1510 {
		t.Fatalf("record failure provider status = %d, want 1510", records[0].Failure.ProviderStatusCode)
	}

	_, skip, err := state.PrepareFrankZohoCampaignSend(args)
	if err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) error = nil, want terminal failure replay block")
	}
	if !skip {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) skip = false, want blocked replay")
	}
	if !strings.Contains(err.Error(), "terminally failed and will not be resent automatically") {
		t.Fatalf("PrepareFrankZohoCampaignSend(replay) error = %q, want terminal failure replay block", err)
	}
}

func TestFrankZohoCampaignSendAmbiguousFailureLeavesPreparedActionBlocked(t *testing.T) {
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
		ID:           "job-frank-zoho-send-ambiguous-failure",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-ambiguous-failure",
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
	if err := state.RecordFrankZohoCampaignSendFailure(args, fmt.Errorf("network timeout")); err != nil {
		t.Fatalf("RecordFrankZohoCampaignSendFailure() error = %v, want no-op on ambiguous failure", err)
	}

	records, err := missioncontrol.ListCommittedCampaignZohoEmailOutboundActionRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedCampaignZohoEmailOutboundActionRecords() len = %d, want 1", len(records))
	}
	if records[0].State != string(missioncontrol.CampaignZohoEmailOutboundActionStatePrepared) {
		t.Fatalf("record state = %q, want prepared after ambiguous failure", records[0].State)
	}

	_, skip, err := state.PrepareFrankZohoCampaignSend(args)
	if err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) error = nil, want ambiguous replay block")
	}
	if !skip {
		t.Fatal("PrepareFrankZohoCampaignSend(replay) skip = false, want blocked replay")
	}
	if !strings.Contains(err.Error(), "already prepared without provider receipt proof") {
		t.Fatalf("PrepareFrankZohoCampaignSend(replay) error = %q, want prepared replay block", err)
	}
}

func TestTaskStateSyncFrankZohoCampaignInboundRepliesPersistsAppendOnly(t *testing.T) {
	originalRead := readFrankZohoCampaignInboundReplies
	t.Cleanup(func() {
		readFrankZohoCampaignInboundReplies = originalRead
	})

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-reply-sync-test"}, job, runtime, control, time.Now().UTC())
	})

	job := missioncontrol.Job{
		ID:           "job-frank-zoho-reply-sync",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{FrankZohoSendEmailToolName},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-reply-sync",
			Steps: []missioncontrol.Step{
				{
					ID:                "send-outbound-email",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{FrankZohoSendEmailToolName},
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

	readFrankZohoCampaignInboundReplies = func(context.Context) ([]missioncontrol.FrankZohoInboundReply, error) {
		return []missioncontrol.FrankZohoInboundReply{
			{
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				ProviderMessageID:  "1711540357880101000",
				ProviderMailID:     "<reply-1@zoho.test>",
				MIMEMessageID:      "<inbound-1@example.test>",
				InReplyTo:          "<mime-1@example.test>",
				References:         []string{"<seed@example.test>", "<mime-1@example.test>"},
				FromAddress:        "person@example.com",
				FromDisplayName:    "Person One",
				Subject:            "Re: Frank intro",
				ReceivedAt:         time.Date(2026, 4, 15, 16, 20, 0, 0, time.UTC),
				OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage",
			},
		}, nil
	}

	appended, err := state.SyncFrankZohoCampaignInboundReplies()
	if err != nil {
		t.Fatalf("SyncFrankZohoCampaignInboundReplies(first) error = %v", err)
	}
	if appended != 1 {
		t.Fatalf("SyncFrankZohoCampaignInboundReplies(first) appended = %d, want 1", appended)
	}

	appended, err = state.SyncFrankZohoCampaignInboundReplies()
	if err != nil {
		t.Fatalf("SyncFrankZohoCampaignInboundReplies(second) error = %v", err)
	}
	if appended != 0 {
		t.Fatalf("SyncFrankZohoCampaignInboundReplies(second) appended = %d, want 0 duplicate appends", appended)
	}

	records, err := missioncontrol.ListCommittedFrankZohoInboundReplyRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedFrankZohoInboundReplyRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListCommittedFrankZohoInboundReplyRecords() len = %d, want 1", len(records))
	}
	if records[0].StepID != "send-outbound-email" {
		t.Fatalf("records[0].StepID = %q, want send step provenance", records[0].StepID)
	}
	if records[0].InReplyTo != "<mime-1@example.test>" {
		t.Fatalf("records[0].InReplyTo = %q, want provider-fetched linkage header", records[0].InReplyTo)
	}
	if len(records[0].References) != 2 || records[0].References[0] != "<seed@example.test>" {
		t.Fatalf("records[0].References = %#v, want durable reference chain", records[0].References)
	}
}

func TestFrankZohoCampaignSendStopsAfterVerifiedSendLimit(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)

	now := time.Date(2026, 4, 15, 20, 0, 0, 0, time.UTC)
	for i, subject := range []string{"Frank intro 1", "Frank intro 2", "Frank intro 3"} {
		jobID := fmt.Sprintf("job-frank-zoho-send-history-%d", i+1)
		if err := missioncontrol.StoreJobRuntimeRecord(root, missioncontrol.JobRuntimeRecord{
			RecordVersion: missioncontrol.StoreRecordVersion,
			WriterEpoch:   1,
			AppliedSeq:    1,
			JobID:         jobID,
			State:         missioncontrol.JobStateRunning,
			ActiveStepID:  "send-outbound-email",
			CreatedAt:     now,
			UpdatedAt:     now,
		}); err != nil {
			t.Fatalf("StoreJobRuntimeRecord(%q) error = %v", jobID, err)
		}
		action := mustBuildVerifiedFrankZohoCampaignAction(t, "send-outbound-email", campaign.CampaignID, subject, now.Add(time.Duration(i)*time.Minute))
		if err := missioncontrol.StoreCampaignZohoEmailOutboundActionRecord(root, testFrankZohoCampaignActionRecord(jobID, 1, action)); err != nil {
			t.Fatalf("StoreCampaignZohoEmailOutboundActionRecord(%q) error = %v", jobID, err)
		}
	}

	tool := NewFrankZohoSendEmailTool()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-send-test"}, job, runtime, control, time.Now().UTC())
	})

	job := missioncontrol.Job{
		ID:           "job-frank-zoho-send-stop-limit",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-stop-limit",
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

	_, skip, err := state.PrepareFrankZohoCampaignSend(map[string]interface{}{
		"subject": "Frank intro 4",
		"body":    "Hello from Frank",
	})
	if err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend() error = nil, want stop-condition halt")
	}
	if skip {
		t.Fatal("PrepareFrankZohoCampaignSend() skip = true, want hard stop before send preparation")
	}
	if !strings.Contains(err.Error(), `campaign send gate is closed: campaign zoho email stop_condition "stop after 3 verified sends" triggered after 3 verified sends`) {
		t.Fatalf("PrepareFrankZohoCampaignSend() error = %q, want stop-condition halt", err)
	}
}

func TestFrankZohoCampaignSendStopsAfterAttributedReplyLimit(t *testing.T) {
	originalRead := readFrankZohoCampaignInboundReplies
	t.Cleanup(func() {
		readFrankZohoCampaignInboundReplies = originalRead
	})

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	campaign.StopConditions = []string{"stop after first reply"}
	campaign.UpdatedAt = campaign.UpdatedAt.Add(2 * time.Minute)
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	now := time.Date(2026, 4, 15, 20, 15, 0, 0, time.UTC)
	if err := missioncontrol.StoreJobRuntimeRecord(root, missioncontrol.JobRuntimeRecord{
		RecordVersion: missioncontrol.StoreRecordVersion,
		WriterEpoch:   1,
		AppliedSeq:    1,
		JobID:         "job-frank-zoho-send-reply-history",
		State:         missioncontrol.JobStateRunning,
		ActiveStepID:  "send-outbound-email",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("StoreJobRuntimeRecord(history) error = %v", err)
	}
	action := mustBuildVerifiedFrankZohoCampaignAction(t, "send-outbound-email", campaign.CampaignID, "Frank intro", now)
	if err := missioncontrol.StoreCampaignZohoEmailOutboundActionRecord(root, testFrankZohoCampaignActionRecord("job-frank-zoho-send-reply-history", 1, action)); err != nil {
		t.Fatalf("StoreCampaignZohoEmailOutboundActionRecord() error = %v", err)
	}

	readFrankZohoCampaignInboundReplies = func(context.Context) ([]missioncontrol.FrankZohoInboundReply, error) {
		return []missioncontrol.FrankZohoInboundReply{
			{
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				ProviderMessageID:  "1711540357880101000",
				ProviderMailID:     "<reply-1@zoho.test>",
				MIMEMessageID:      "<inbound-1@example.test>",
				InReplyTo:          "<mime-1@example.test>",
				ReceivedAt:         now.Add(time.Minute),
				OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage",
			},
		}, nil
	}

	tool := NewFrankZohoSendEmailTool()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-send-test"}, job, runtime, control, time.Now().UTC())
	})

	job := missioncontrol.Job{
		ID:           "job-frank-zoho-send-reply-limit",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{tool.Name()},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-send-reply-limit",
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

	_, skip, err := state.PrepareFrankZohoCampaignSend(map[string]interface{}{
		"subject": "Frank intro 2",
		"body":    "Hello from Frank",
	})
	if err == nil {
		t.Fatal("PrepareFrankZohoCampaignSend() error = nil, want reply stop-condition halt")
	}
	if skip {
		t.Fatal("PrepareFrankZohoCampaignSend() skip = true, want hard stop before send preparation")
	}
	if !strings.Contains(err.Error(), `campaign send gate is closed: campaign zoho email stop_condition "stop after first reply" triggered after 1 attributed replies`) {
		t.Fatalf("PrepareFrankZohoCampaignSend() error = %q, want reply stop-condition halt", err)
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

func TestFrankZohoInboundReplyReaderUsesZohoBearerTokenAndMapsReplyHeaders(t *testing.T) {
	t.Parallel()

	var gotAuth []string
	var gotPaths []string

	reader := NewFrankZohoInboundReplyReader()
	reader.apiBase = "https://mail.zoho.test/api"
	reader.accessToken = func(context.Context) (string, error) {
		return "test-zoho-token", nil
	}
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotPaths = append(gotPaths, r.URL.Path)
			gotAuth = append(gotAuth, r.Header.Get("Authorization"))
			switch r.URL.Path {
			case "/api/accounts/3323462000000008002/messages":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"status": {"code": 200, "description": "success"},
						"data": [
							{"messageId": 1711540357880101000, "mailId": "<reply-1@zoho.test>", "receivedTime": 1711540357880},
							{"messageId": 1711540357880101999, "mailId": "<note@zoho.test>", "receivedTime": 1711540357999}
						]
					}`)),
				}, nil
			case "/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"message/rfc822"}},
					Body:       io.NopCloser(strings.NewReader("From: Person One <person@example.com>\r\nSubject: Re: Frank intro\r\nMessage-ID: <inbound-1@example.test>\r\nIn-Reply-To: <mime-1@example.test>\r\nReferences: <seed@example.test> <mime-1@example.test>\r\n\r\nReply body")),
				}, nil
			case "/api/accounts/3323462000000008002/messages/1711540357880101999/originalmessage":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"message/rfc822"}},
					Body:       io.NopCloser(strings.NewReader("From: Note <note@example.com>\r\nSubject: FYI\r\nMessage-ID: <note-1@example.test>\r\n\r\nNon-reply body")),
				}, nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
				return nil, nil
			}
		}),
	}

	got, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if len(gotPaths) != 3 {
		t.Fatalf("request count = %d, want 3", len(gotPaths))
	}
	for i, auth := range gotAuth {
		if auth != "Zoho-oauthtoken test-zoho-token" {
			t.Fatalf("Authorization[%d] = %q, want Zoho OAuth header", i, auth)
		}
	}
	if len(got) != 1 {
		t.Fatalf("len(replies) = %d, want 1 reply-shaped message", len(got))
	}
	if got[0].ProviderMessageID != "1711540357880101000" {
		t.Fatalf("ProviderMessageID = %q, want mailbox message id", got[0].ProviderMessageID)
	}
	if got[0].InReplyTo != "<mime-1@example.test>" {
		t.Fatalf("InReplyTo = %q, want reply-header linkage", got[0].InReplyTo)
	}
	if len(got[0].References) != 2 || got[0].References[1] != "<mime-1@example.test>" {
		t.Fatalf("References = %#v, want preserved References chain", got[0].References)
	}
	if got[0].FromAddress != "person@example.com" {
		t.Fatalf("FromAddress = %q, want provider-fetched sender", got[0].FromAddress)
	}
	if got[0].OriginalMessageURL != "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage" {
		t.Fatalf("OriginalMessageURL = %q, want originalmessage locator", got[0].OriginalMessageURL)
	}
	if got[0].ReceivedAt.IsZero() {
		t.Fatal("ReceivedAt = zero, want provider received timestamp")
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
	campaign.StopConditions = []string{"stop after 3 verified sends"}
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

func mustBuildVerifiedFrankZohoCampaignAction(t *testing.T, stepID, campaignID, subject string, preparedAt time.Time) missioncontrol.CampaignZohoEmailOutboundAction {
	t.Helper()

	prepared, err := missioncontrol.BuildCampaignZohoEmailOutboundPreparedAction(
		stepID,
		campaignID,
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		missioncontrol.CampaignZohoEmailAddressing{
			To:  []string{"person@example.com"},
			CC:  []string{"copy@example.com"},
			BCC: []string{"blind@example.com"},
		},
		subject,
		"plaintext",
		"Hello from Frank",
		preparedAt,
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	sent, err := missioncontrol.BuildCampaignZohoEmailOutboundSentAction(prepared, missioncontrol.FrankZohoSendReceipt{
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, preparedAt.Add(10*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}
	sent.State = missioncontrol.CampaignZohoEmailOutboundActionStateVerified
	sent.VerifiedAt = preparedAt.Add(20 * time.Second)
	if err := missioncontrol.ValidateCampaignZohoEmailOutboundAction(sent); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(verified) error = %v", err)
	}
	return sent
}

func testFrankZohoCampaignActionRecord(jobID string, lastSeq uint64, action missioncontrol.CampaignZohoEmailOutboundAction) missioncontrol.CampaignZohoEmailOutboundActionRecord {
	normalized := missioncontrol.NormalizeCampaignZohoEmailOutboundAction(action)
	return missioncontrol.CampaignZohoEmailOutboundActionRecord{
		RecordVersion:      missioncontrol.StoreRecordVersion,
		LastSeq:            lastSeq,
		ActionID:           normalized.ActionID,
		JobID:              jobID,
		StepID:             normalized.StepID,
		CampaignID:         normalized.CampaignID,
		State:              string(normalized.State),
		Provider:           normalized.Provider,
		ProviderAccountID:  normalized.ProviderAccountID,
		FromAddress:        normalized.FromAddress,
		FromDisplayName:    normalized.FromDisplayName,
		Addressing:         normalized.Addressing,
		Subject:            normalized.Subject,
		BodyFormat:         normalized.BodyFormat,
		BodySHA256:         normalized.BodySHA256,
		PreparedAt:         normalized.PreparedAt,
		SentAt:             normalized.SentAt,
		VerifiedAt:         normalized.VerifiedAt,
		ProviderMessageID:  normalized.ProviderMessageID,
		ProviderMailID:     normalized.ProviderMailID,
		MIMEMessageID:      normalized.MIMEMessageID,
		OriginalMessageURL: normalized.OriginalMessageURL,
	}
}
