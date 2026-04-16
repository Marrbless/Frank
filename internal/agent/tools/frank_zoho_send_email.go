package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

const (
	frankZohoSendEmailToolName     = "frank_zoho_send_email"
	frankZohoMailAPIBase           = "https://mail.zoho.com/api"
	frankZohoMailAccountID         = "3323462000000008002"
	frankZohoMailDefaultBodyFormat = "plaintext"
	frankZohoMailDefaultEncoding   = "UTF-8"
)

const FrankZohoSendEmailToolName = frankZohoSendEmailToolName

var verifyFrankZohoCampaignSendProof = func(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]FrankZohoSendProofVerification, error) {
	return NewFrankZohoSendProofVerifier().Verify(ctx, proof)
}

var readFrankZohoCampaignInboundReplies = func(ctx context.Context) ([]missioncontrol.FrankZohoInboundReply, error) {
	return NewFrankZohoInboundReplyReader().Read(ctx)
}

type FrankZohoSendEmailTool struct {
	client      *http.Client
	apiBase     string
	accessToken func(context.Context) (string, error)
	send        func(context.Context, frankZohoSendRequest) (frankZohoSendResponseData, error)
}

type FrankZohoSendProofVerifier struct {
	client      *http.Client
	accessToken func(context.Context) (string, error)
}

type FrankZohoInboundReplyReader struct {
	client      *http.Client
	apiBase     string
	accessToken func(context.Context) (string, error)
}

type FrankZohoSendReceipt struct {
	Provider           string `json:"provider"`
	ProviderAccountID  string `json:"provider_account_id"`
	FromAddress        string `json:"from_address"`
	FromDisplayName    string `json:"from_display_name"`
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	OriginalMessageURL string `json:"original_message_url"`
}

type FrankZohoSendProofVerification struct {
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	ProviderAccountID  string `json:"provider_account_id"`
	OriginalMessageURL string `json:"original_message_url"`
	OriginalMessage    string `json:"original_message"`
}

type frankZohoSendRequest struct {
	AccountID                string
	From                     string
	To                       []string
	CC                       []string
	BCC                      []string
	Subject                  string
	Body                     string
	BodyFormat               string
	Encoding                 string
	ReplyToProviderMessageID string
}

type frankZohoSendAPIRequest struct {
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	CCAddress   string `json:"ccAddress,omitempty"`
	BCCAddress  string `json:"bccAddress,omitempty"`
	Subject     string `json:"subject"`
	Content     string `json:"content"`
	MailFormat  string `json:"mailFormat,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	Action      string `json:"action,omitempty"`
}

type frankZohoSendAPIResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data frankZohoSendResponseData `json:"data"`
}

type frankZohoMailboxMessagesResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []frankZohoMailboxMessageData `json:"data"`
}

type frankZohoMailboxMessageData struct {
	MessageID    frankZohoFlexibleString `json:"messageId"`
	MailID       frankZohoFlexibleString `json:"mailId"`
	ReceivedTime frankZohoFlexibleString `json:"receivedTime"`
}

type frankZohoCampaignSendIntent struct {
	Request                frankZohoSendRequest
	PreparedAction         missioncontrol.CampaignZohoEmailOutboundAction
	FollowUpInboundReplyID string
	ReplyWorkSelection     *missioncontrol.CampaignZohoEmailReplyWorkSelection
}

type frankZohoCampaignSender struct {
	ProviderAccountID string
	FromAddress       string
	FromDisplayName   string
}

type frankZohoTerminalSendFailureError struct {
	httpStatus int
	failure    missioncontrol.CampaignZohoEmailOutboundFailure
}

func (e *frankZohoTerminalSendFailureError) Error() string {
	if e == nil {
		return frankZohoSendEmailToolName + ": provider-declared terminal send rejection"
	}
	return fmt.Sprintf("%s: Zoho Mail rejected send with provider status %d: %s", frankZohoSendEmailToolName, e.failure.ProviderStatusCode, strings.TrimSpace(e.failure.ProviderStatusDescription))
}

func (e *frankZohoTerminalSendFailureError) Failure() missioncontrol.CampaignZohoEmailOutboundFailure {
	if e == nil {
		return missioncontrol.CampaignZohoEmailOutboundFailure{}
	}
	failure := e.failure
	if failure.HTTPStatus == 0 {
		failure.HTTPStatus = e.httpStatus
	}
	return failure
}

type frankZohoSendResponseData struct {
	MessageID       frankZohoFlexibleString `json:"messageId"`
	MailID          frankZohoFlexibleString `json:"mailId"`
	MIMEMessageID   frankZohoFlexibleString `json:"mimeMessageId"`
	MessageIDHeader frankZohoFlexibleString `json:"messageIdHeader"`
	InternetMessage frankZohoFlexibleString `json:"internetMessageId"`
}

type frankZohoFlexibleString string

func (s *frankZohoFlexibleString) UnmarshalJSON(data []byte) error {
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
		*s = frankZohoFlexibleString(strings.TrimSpace(v))
		return nil
	default:
		*s = frankZohoFlexibleString(trimmed)
		return nil
	}
}

func NewFrankZohoSendEmailTool() *FrankZohoSendEmailTool {
	return &FrankZohoSendEmailTool{
		client:      &http.Client{Timeout: 30 * time.Second},
		apiBase:     frankZohoMailAPIBase,
		accessToken: frankZohoMailAccessTokenFromEnv,
	}
}

func NewFrankZohoSendProofVerifier() *FrankZohoSendProofVerifier {
	return &FrankZohoSendProofVerifier{
		client:      &http.Client{Timeout: 30 * time.Second},
		accessToken: frankZohoMailAccessTokenFromEnv,
	}
}

func NewFrankZohoInboundReplyReader() *FrankZohoInboundReplyReader {
	return &FrankZohoInboundReplyReader{
		client:      &http.Client{Timeout: 30 * time.Second},
		apiBase:     frankZohoMailAPIBase,
		accessToken: frankZohoMailAccessTokenFromEnv,
	}
}

func (t *FrankZohoSendEmailTool) Name() string {
	return frankZohoSendEmailToolName
}

func (t *FrankZohoSendEmailTool) Description() string {
	return "Send one email from the campaign-linked Frank Zoho mailbox using committed Frank registry records"
}

func (t *FrankZohoSendEmailTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"subject": map[string]interface{}{
				"type":        "string",
				"description": "Email subject line",
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Email body content",
			},
			"body_format": map[string]interface{}{
				"type":        "string",
				"description": "Body format to send via Zoho Mail",
				"enum":        []string{"plaintext", "html"},
			},
			"inbound_reply_id": map[string]interface{}{
				"type":        "string",
				"description": "Committed Zoho inbound reply record to answer with a provider-threaded follow-up",
			},
		},
		"required": []string{"subject", "body"},
	}
}

func (t *FrankZohoSendEmailTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	ec, ok := missioncontrol.ExecutionContextFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("%s requires mission execution context", t.Name())
	}
	if ec.Step == nil || ec.Step.CampaignRef == nil {
		return "", fmt.Errorf("%s requires step.campaign_ref", t.Name())
	}
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return "", fmt.Errorf("%s requires mission_store_root to resolve campaign context", t.Name())
	}
	if err := missioncontrol.RequireExecutionContextCampaignReadiness(ec); err != nil {
		return "", err
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return "", err
	}
	followUpInboundReplyID, _, err := frankZohoOptionalStringArg(args, "inbound_reply_id")
	if err != nil {
		return "", err
	}
	if err := validateFrankZohoMailPreflight(preflight, followUpInboundReplyID == ""); err != nil {
		return "", err
	}
	if err := requireFrankZohoCampaignSendGate(ec, *preflight.Campaign); err != nil {
		return "", err
	}

	intent, err := buildFrankZohoCampaignSendIntent(ec, args, time.Now().UTC())
	if err != nil {
		return "", err
	}
	req := intent.Request
	req.AccountID = intent.PreparedAction.ProviderAccountID
	req.From = intent.PreparedAction.FromAddress

	send := t.send
	if send == nil {
		send = t.sendViaAPI
	}

	data, err := send(ctx, req)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data.MessageID)) == "" {
		return "", fmt.Errorf("%s: Zoho send response missing data.messageId", t.Name())
	}

	receipt := FrankZohoSendReceipt{
		Provider:           "zoho_mail",
		ProviderAccountID:  intent.PreparedAction.ProviderAccountID,
		FromAddress:        intent.PreparedAction.FromAddress,
		FromDisplayName:    intent.PreparedAction.FromDisplayName,
		ProviderMessageID:  string(data.MessageID),
		ProviderMailID:     string(data.MailID),
		MIMEMessageID:      firstNonEmpty(string(data.MIMEMessageID), string(data.MessageIDHeader), string(data.InternetMessage)),
		OriginalMessageURL: frankZohoOriginalMessageURL(t.apiBase, intent.PreparedAction.ProviderAccountID, string(data.MessageID)),
	}

	encoded, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (t *FrankZohoSendEmailTool) sendViaAPI(ctx context.Context, req frankZohoSendRequest) (frankZohoSendResponseData, error) {
	tokenProvider := t.accessToken
	if tokenProvider == nil {
		tokenProvider = frankZohoMailAccessTokenFromEnv
	}
	token, err := tokenProvider(ctx)
	if err != nil {
		return frankZohoSendResponseData{}, err
	}

	payload := frankZohoSendAPIRequest{
		FromAddress: req.From,
		ToAddress:   strings.Join(req.To, ","),
		CCAddress:   strings.Join(req.CC, ","),
		BCCAddress:  strings.Join(req.BCC, ","),
		Subject:     req.Subject,
		Content:     req.Body,
		MailFormat:  req.BodyFormat,
		Encoding:    req.Encoding,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return frankZohoSendResponseData{}, err
	}

	url := frankZohoMessagesURL(t.apiBase, req.AccountID)
	if strings.TrimSpace(req.ReplyToProviderMessageID) != "" {
		payload.Action = "reply"
		body, err = json.Marshal(payload)
		if err != nil {
			return frankZohoSendResponseData{}, err
		}
		url = frankZohoReplyMessageURL(t.apiBase, req.AccountID, req.ReplyToProviderMessageID)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return frankZohoSendResponseData{}, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Zoho-oauthtoken "+token)

	client := t.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return frankZohoSendResponseData{}, fmt.Errorf("%s request failed: %w", t.Name(), err)
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded frankZohoSendAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return frankZohoSendResponseData{}, fmt.Errorf("%s: failed to decode Zoho response: %w", t.Name(), err)
	}
	if terminalFailure, ok := frankZohoTerminalFailureFromResponse(resp.StatusCode, decoded); ok {
		return frankZohoSendResponseData{}, terminalFailure
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return frankZohoSendResponseData{}, fmt.Errorf("%s: Zoho Mail returned HTTP %d", t.Name(), resp.StatusCode)
	}
	if decoded.Status.Code != 0 && decoded.Status.Code != 200 {
		return frankZohoSendResponseData{}, fmt.Errorf("%s: Zoho Mail status %d: %s", t.Name(), decoded.Status.Code, strings.TrimSpace(decoded.Status.Description))
	}
	return decoded.Data, nil
}

func frankZohoTerminalFailureFromResponse(httpStatus int, decoded frankZohoSendAPIResponse) (*frankZohoTerminalSendFailureError, bool) {
	if decoded.Status.Code == 0 || decoded.Status.Code == 200 {
		return nil, false
	}
	if strings.TrimSpace(string(decoded.Data.MessageID)) != "" {
		return nil, false
	}
	return &frankZohoTerminalSendFailureError{
		httpStatus: httpStatus,
		failure: missioncontrol.CampaignZohoEmailOutboundFailure{
			HTTPStatus:                httpStatus,
			ProviderStatusCode:        decoded.Status.Code,
			ProviderStatusDescription: strings.TrimSpace(decoded.Status.Description),
		},
	}, true
}

func (v *FrankZohoSendProofVerifier) Verify(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]FrankZohoSendProofVerification, error) {
	if len(proof) == 0 {
		return []FrankZohoSendProofVerification{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tokenProvider := v.accessToken
	if tokenProvider == nil {
		tokenProvider = frankZohoMailAccessTokenFromEnv
	}
	token, err := tokenProvider(ctx)
	if err != nil {
		return nil, err
	}

	client := v.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	verified := make([]FrankZohoSendProofVerification, 0, len(proof))
	for _, candidate := range proof {
		originalMessageURL := strings.TrimSpace(candidate.OriginalMessageURL)
		if originalMessageURL == "" {
			return nil, fmt.Errorf("%s verification requires original_message_url", frankZohoSendEmailToolName)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, originalMessageURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept", "*/*")
		httpReq.Header.Set("Authorization", "Zoho-oauthtoken "+token)

		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("%s verification request failed for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(candidate.ProviderMessageID), err)
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%s verification failed to read original message for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(candidate.ProviderMessageID), readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("%s verification failed to close original message response for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(candidate.ProviderMessageID), closeErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%s verification: Zoho Mail returned HTTP %d for provider_message_id %q", frankZohoSendEmailToolName, resp.StatusCode, strings.TrimSpace(candidate.ProviderMessageID))
		}

		verified = append(verified, FrankZohoSendProofVerification{
			ProviderMessageID:  candidate.ProviderMessageID,
			ProviderMailID:     candidate.ProviderMailID,
			MIMEMessageID:      candidate.MIMEMessageID,
			ProviderAccountID:  candidate.ProviderAccountID,
			OriginalMessageURL: candidate.OriginalMessageURL,
			OriginalMessage:    string(body),
		})
	}

	return verified, nil
}

func (r *FrankZohoInboundReplyReader) Read(ctx context.Context) ([]missioncontrol.FrankZohoInboundReply, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tokenProvider := r.accessToken
	if tokenProvider == nil {
		tokenProvider = frankZohoMailAccessTokenFromEnv
	}
	token, err := tokenProvider(ctx)
	if err != nil {
		return nil, err
	}

	client := r.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	messagesURL := frankZohoMessagesURL(r.apiBase, frankZohoMailAccountID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, messagesURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Zoho-oauthtoken "+token)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s inbound reply read request failed: %w", frankZohoSendEmailToolName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded frankZohoMailboxMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%s inbound reply read failed to decode Zoho messages response: %w", frankZohoSendEmailToolName, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s inbound reply read: Zoho Mail returned HTTP %d", frankZohoSendEmailToolName, resp.StatusCode)
	}
	if decoded.Status.Code != 0 && decoded.Status.Code != 200 {
		return nil, fmt.Errorf("%s inbound reply read: Zoho Mail status %d: %s", frankZohoSendEmailToolName, decoded.Status.Code, strings.TrimSpace(decoded.Status.Description))
	}

	replies := make([]missioncontrol.FrankZohoInboundReply, 0, len(decoded.Data))
	for _, candidate := range decoded.Data {
		messageID := strings.TrimSpace(string(candidate.MessageID))
		if messageID == "" {
			continue
		}
		receivedAt, err := parseFrankZohoMailboxReceivedAt(candidate.ReceivedTime)
		if err != nil {
			return nil, fmt.Errorf("%s inbound reply read: parse received time for provider_message_id %q: %w", frankZohoSendEmailToolName, messageID, err)
		}
		originalMessageURL := frankZohoOriginalMessageURL(r.apiBase, frankZohoMailAccountID, messageID)
		originalMessage, err := frankZohoReadOriginalMessage(ctx, client, token, messageID, originalMessageURL)
		if err != nil {
			return nil, err
		}
		reply, ok, err := frankZohoInboundReplyFromOriginalMessage(
			messageID,
			strings.TrimSpace(string(candidate.MailID)),
			receivedAt,
			originalMessageURL,
			originalMessage,
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		replies = append(replies, reply)
	}
	return replies, nil
}

func validateFrankZohoMailPreflight(preflight missioncontrol.ResolvedExecutionContextCampaignPreflight, requireCampaignAddressing bool) error {
	_, err := resolveFrankZohoCampaignSender(preflight, requireCampaignAddressing)
	return err
}

func resolveFrankZohoCampaignSender(preflight missioncontrol.ResolvedExecutionContextCampaignPreflight, requireCampaignAddressing bool) (frankZohoCampaignSender, error) {
	if preflight.Campaign == nil {
		return frankZohoCampaignSender{}, fmt.Errorf("%s requires a resolved campaign preflight", frankZohoSendEmailToolName)
	}
	if requireCampaignAddressing && preflight.Campaign.ZohoEmailAddressing == nil {
		return frankZohoCampaignSender{}, fmt.Errorf("%s requires campaign zoho_email_addressing", frankZohoSendEmailToolName)
	}

	emailIdentities := make(map[string]missioncontrol.FrankIdentityRecord)
	for _, identity := range preflight.Identities {
		if strings.EqualFold(strings.TrimSpace(identity.IdentityKind), "email") {
			emailIdentities[strings.TrimSpace(identity.IdentityID)] = identity
		}
	}
	if len(emailIdentities) == 0 {
		return frankZohoCampaignSender{}, fmt.Errorf("%s requires a campaign-linked Frank email identity", frankZohoSendEmailToolName)
	}

	candidates := make([]frankZohoCampaignSender, 0, 1)
	for _, account := range preflight.Accounts {
		if !strings.EqualFold(strings.TrimSpace(account.AccountKind), "mailbox") {
			continue
		}
		identity, ok := emailIdentities[strings.TrimSpace(account.IdentityID)]
		if !ok {
			continue
		}
		if identity.ZohoMailbox == nil {
			return frankZohoCampaignSender{}, fmt.Errorf("%s requires campaign-linked Frank identity %q to declare zoho_mailbox sender fields", frankZohoSendEmailToolName, strings.TrimSpace(identity.IdentityID))
		}
		if account.ZohoMailbox == nil || !account.ZohoMailbox.ConfirmedCreated || strings.TrimSpace(account.ZohoMailbox.ProviderAccountID) == "" {
			return frankZohoCampaignSender{}, fmt.Errorf("%s requires campaign-linked Frank mailbox account %q to declare committed zoho_mailbox.provider_account_id plus confirmed_created", frankZohoSendEmailToolName, strings.TrimSpace(account.AccountID))
		}
		candidates = append(candidates, frankZohoCampaignSender{
			ProviderAccountID: strings.TrimSpace(account.ZohoMailbox.ProviderAccountID),
			FromAddress:       strings.TrimSpace(identity.ZohoMailbox.FromAddress),
			FromDisplayName:   strings.TrimSpace(identity.ZohoMailbox.FromDisplayName),
		})
	}

	if len(candidates) == 0 {
		return frankZohoCampaignSender{}, fmt.Errorf("%s requires exactly one campaign-linked Frank Zoho mailbox sender pair", frankZohoSendEmailToolName)
	}
	if len(candidates) != 1 {
		return frankZohoCampaignSender{}, fmt.Errorf("%s requires exactly one campaign-linked Frank Zoho mailbox sender pair; found %d", frankZohoSendEmailToolName, len(candidates))
	}

	return candidates[0], nil
}

func requireFrankZohoCampaignSendGate(ec missioncontrol.ExecutionContext, campaign missioncontrol.CampaignRecord) error {
	if strings.TrimSpace(ec.MissionStoreRoot) == "" {
		return fmt.Errorf("%s requires mission_store_root to derive campaign send gate", frankZohoSendEmailToolName)
	}
	decision, err := missioncontrol.LoadCommittedCampaignZohoEmailSendGateDecision(ec.MissionStoreRoot, campaign)
	if err != nil {
		return fmt.Errorf("%s: campaign send gate is closed: %w", frankZohoSendEmailToolName, err)
	}
	if !decision.Allowed {
		reason := strings.TrimSpace(decision.Reason)
		if reason == "" {
			reason = "campaign send gate denied further outbound sends"
		}
		return fmt.Errorf("%s: campaign send gate is closed: %s", frankZohoSendEmailToolName, reason)
	}
	return nil
}

func buildFrankZohoCampaignSendIntent(ec missioncontrol.ExecutionContext, args map[string]interface{}, now time.Time) (frankZohoCampaignSendIntent, error) {
	if err := missioncontrol.RequireExecutionContextCampaignReadiness(ec); err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	if err := frankZohoRejectAddressArgs(args); err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	followUpInboundReplyID, _, err := frankZohoOptionalStringArg(args, "inbound_reply_id")
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	var replyWorkSelection *missioncontrol.CampaignZohoEmailReplyWorkSelection
	if followUpInboundReplyID == "" {
		selection, ok, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkSelection(ec.MissionStoreRoot, preflight.Campaign.CampaignID, now)
		if err != nil {
			return frankZohoCampaignSendIntent{}, err
		}
		if ok {
			replyWorkSelection = &selection
			followUpInboundReplyID = selection.InboundReply.ReplyID
		}
	}
	if err := validateFrankZohoMailPreflight(preflight, followUpInboundReplyID == ""); err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	sender, err := resolveFrankZohoCampaignSender(preflight, followUpInboundReplyID == "")
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	if err := requireFrankZohoCampaignSendGate(ec, *preflight.Campaign); err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	subject, err := frankZohoRequiredStringArg(args, "subject")
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	body, err := frankZohoRequiredStringArg(args, "body")
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	bodyFormat, err := frankZohoBodyFormatArg(args, "body_format")
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	req := frankZohoSendRequest{
		Subject:    subject,
		Body:       body,
		BodyFormat: bodyFormat,
		Encoding:   frankZohoMailDefaultEncoding,
	}
	if followUpInboundReplyID == "" {
		addressing := preflight.Campaign.ZohoEmailAddressing
		req.To = append([]string(nil), addressing.To...)
		req.CC = append([]string(nil), addressing.CC...)
		req.BCC = append([]string(nil), addressing.BCC...)
		prepared, err := missioncontrol.BuildCampaignZohoEmailOutboundPreparedAction(
			ec.Step.ID,
			preflight.Campaign.CampaignID,
			sender.ProviderAccountID,
			sender.FromAddress,
			sender.FromDisplayName,
			missioncontrol.CampaignZohoEmailAddressing{
				To:  append([]string(nil), req.To...),
				CC:  append([]string(nil), req.CC...),
				BCC: append([]string(nil), req.BCC...),
			},
			req.Subject,
			req.BodyFormat,
			req.Body,
			now,
		)
		if err != nil {
			return frankZohoCampaignSendIntent{}, err
		}
		return frankZohoCampaignSendIntent{Request: req, PreparedAction: prepared}, nil
	}

	followUp, err := missioncontrol.LoadCommittedCampaignZohoEmailFollowUpTarget(ec.MissionStoreRoot, preflight.Campaign.CampaignID, followUpInboundReplyID)
	if err != nil {
		return frankZohoCampaignSendIntent{}, fmt.Errorf("%s: %w", frankZohoSendEmailToolName, err)
	}
	if followUp.InboundReply.FromAddressCount != 1 || strings.TrimSpace(followUp.InboundReply.FromAddress) == "" {
		return frankZohoCampaignSendIntent{}, fmt.Errorf("%s: inbound_reply_id %q does not resolve to exactly one durable sender identity", frankZohoSendEmailToolName, followUpInboundReplyID)
	}
	if strings.TrimSpace(followUp.InboundReply.ProviderMessageID) == "" {
		return frankZohoCampaignSendIntent{}, fmt.Errorf("%s: inbound_reply_id %q is missing provider message identity for provider-threaded reply send", frankZohoSendEmailToolName, followUpInboundReplyID)
	}
	req.To = []string{strings.TrimSpace(followUp.InboundReply.FromAddress)}
	req.ReplyToProviderMessageID = strings.TrimSpace(followUp.InboundReply.ProviderMessageID)

	prepared, err := missioncontrol.BuildCampaignZohoEmailOutboundPreparedReplyAction(
		ec.Step.ID,
		preflight.Campaign.CampaignID,
		sender.ProviderAccountID,
		sender.FromAddress,
		sender.FromDisplayName,
		missioncontrol.CampaignZohoEmailAddressing{To: append([]string(nil), req.To...)},
		req.Subject,
		req.BodyFormat,
		req.Body,
		now,
		followUp.InboundReply.ReplyID,
		followUp.OutboundAction.ActionID,
	)
	if err != nil {
		return frankZohoCampaignSendIntent{}, err
	}
	return frankZohoCampaignSendIntent{
		Request:                req,
		PreparedAction:         prepared,
		FollowUpInboundReplyID: followUp.InboundReply.ReplyID,
		ReplyWorkSelection:     replyWorkSelection,
	}, nil
}

func buildFrankZohoPreparedCampaignAction(ec missioncontrol.ExecutionContext, args map[string]interface{}, now time.Time) (missioncontrol.CampaignZohoEmailOutboundAction, error) {
	intent, err := buildFrankZohoCampaignSendIntent(ec, args, now)
	if err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	return intent.PreparedAction, nil
}

func frankZohoSendReceiptFromCampaignAction(action missioncontrol.CampaignZohoEmailOutboundAction) (string, error) {
	switch action.State {
	case missioncontrol.CampaignZohoEmailOutboundActionStateSent, missioncontrol.CampaignZohoEmailOutboundActionStateVerified:
	default:
		return "", fmt.Errorf("%s: campaign outbound action %q is not send-reconciled", frankZohoSendEmailToolName, action.ActionID)
	}
	receipt := FrankZohoSendReceipt{
		Provider:           action.Provider,
		ProviderAccountID:  action.ProviderAccountID,
		FromAddress:        action.FromAddress,
		FromDisplayName:    action.FromDisplayName,
		ProviderMessageID:  action.ProviderMessageID,
		ProviderMailID:     action.ProviderMailID,
		MIMEMessageID:      action.MIMEMessageID,
		OriginalMessageURL: action.OriginalMessageURL,
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func frankZohoCampaignProofFromAction(action missioncontrol.CampaignZohoEmailOutboundAction) []missioncontrol.OperatorFrankZohoSendProofStatus {
	return []missioncontrol.OperatorFrankZohoSendProofStatus{
		{
			StepID:             action.StepID,
			ProviderMessageID:  action.ProviderMessageID,
			ProviderMailID:     action.ProviderMailID,
			MIMEMessageID:      action.MIMEMessageID,
			ProviderAccountID:  action.ProviderAccountID,
			OriginalMessageURL: action.OriginalMessageURL,
		},
	}
}

func finalizeFrankZohoCampaignActionFromProof(action missioncontrol.CampaignZohoEmailOutboundAction, verification FrankZohoSendProofVerification, verifiedAt time.Time) (missioncontrol.CampaignZohoEmailOutboundAction, error) {
	normalized := missioncontrol.NormalizeCampaignZohoEmailOutboundAction(action)
	if normalized.State != missioncontrol.CampaignZohoEmailOutboundActionStateSent {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: campaign outbound action %q is not awaiting provider-mailbox verification", frankZohoSendEmailToolName, normalized.ActionID)
	}
	if strings.TrimSpace(verification.ProviderMessageID) != normalized.ProviderMessageID {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof provider_message_id %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(verification.ProviderMessageID), normalized.ProviderMessageID)
	}
	if strings.TrimSpace(verification.ProviderAccountID) != normalized.ProviderAccountID {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof provider_account_id %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(verification.ProviderAccountID), normalized.ProviderAccountID)
	}
	if strings.TrimSpace(verification.OriginalMessageURL) != normalized.OriginalMessageURL {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof original_message_url %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(verification.OriginalMessageURL), normalized.OriginalMessageURL)
	}
	if normalized.ProviderMailID != "" && strings.TrimSpace(verification.ProviderMailID) != normalized.ProviderMailID {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof provider_mail_id %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(verification.ProviderMailID), normalized.ProviderMailID)
	}
	if normalized.MIMEMessageID != "" && strings.TrimSpace(verification.MIMEMessageID) != normalized.MIMEMessageID {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof mime_message_id %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(verification.MIMEMessageID), normalized.MIMEMessageID)
	}

	parsed, err := mail.ReadMessage(strings.NewReader(verification.OriginalMessage))
	if err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: parse original_message: %w", frankZohoSendEmailToolName, err)
	}
	from, err := parsed.Header.AddressList("From")
	if err != nil || len(from) != 1 {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof original_message missing usable From header", frankZohoSendEmailToolName)
	}
	if !strings.EqualFold(strings.TrimSpace(from[0].Address), normalized.FromAddress) {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof From header %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(from[0].Address), normalized.FromAddress)
	}
	if strings.TrimSpace(parsed.Header.Get("Subject")) != normalized.Subject {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof Subject header %q does not match campaign outbound action %q", frankZohoSendEmailToolName, strings.TrimSpace(parsed.Header.Get("Subject")), normalized.Subject)
	}
	if err := frankZohoVerifyHeaderAddressList(parsed.Header, "To", normalized.Addressing.To); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	if err := frankZohoVerifyHeaderAddressList(parsed.Header, "Cc", normalized.Addressing.CC); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}

	body, err := io.ReadAll(parsed.Body)
	if err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: read original_message body: %w", frankZohoSendEmailToolName, err)
	}
	if got := missioncontrol.CampaignZohoEmailBodySHA256(string(body)); got != normalized.BodySHA256 {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, fmt.Errorf("%s: proof body sha256 %q does not match campaign outbound action %q", frankZohoSendEmailToolName, got, normalized.BodySHA256)
	}

	normalized.State = missioncontrol.CampaignZohoEmailOutboundActionStateVerified
	normalized.VerifiedAt = verifiedAt.UTC()
	if err := missioncontrol.ValidateCampaignZohoEmailOutboundAction(normalized); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	return normalized, nil
}

func frankZohoReadOriginalMessage(ctx context.Context, client *http.Client, token string, providerMessageID string, originalMessageURL string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, originalMessageURL, nil)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Accept", "*/*")
	httpReq.Header.Set("Authorization", "Zoho-oauthtoken "+token)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("%s inbound reply read failed for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(providerMessageID), err)
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("%s inbound reply read failed to read original message for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(providerMessageID), readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("%s inbound reply read failed to close original message response for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(providerMessageID), closeErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s inbound reply read: Zoho Mail returned HTTP %d for provider_message_id %q", frankZohoSendEmailToolName, resp.StatusCode, strings.TrimSpace(providerMessageID))
	}
	return string(body), nil
}

func frankZohoInboundReplyFromOriginalMessage(providerMessageID, providerMailID string, receivedAt time.Time, originalMessageURL string, originalMessage string) (missioncontrol.FrankZohoInboundReply, bool, error) {
	parsed, err := mail.ReadMessage(strings.NewReader(originalMessage))
	if err != nil {
		return missioncontrol.FrankZohoInboundReply{}, false, fmt.Errorf("%s inbound reply read: parse original message for provider_message_id %q: %w", frankZohoSendEmailToolName, strings.TrimSpace(providerMessageID), err)
	}
	inReplyTo := strings.TrimSpace(parsed.Header.Get("In-Reply-To"))
	references := strings.Fields(strings.TrimSpace(parsed.Header.Get("References")))
	if inReplyTo == "" && len(references) == 0 {
		return missioncontrol.FrankZohoInboundReply{}, false, nil
	}

	var fromAddress string
	var fromDisplayName string
	fromAddressCount := 0
	if from, err := parsed.Header.AddressList("From"); err == nil {
		fromAddressCount = len(from)
		if len(from) == 1 {
			fromAddress = strings.TrimSpace(from[0].Address)
			fromDisplayName = strings.TrimSpace(from[0].Name)
		}
	}

	reply := missioncontrol.NormalizeFrankZohoInboundReply(missioncontrol.FrankZohoInboundReply{
		Provider:           "zoho_mail",
		ProviderAccountID:  frankZohoMailAccountID,
		ProviderMessageID:  strings.TrimSpace(providerMessageID),
		ProviderMailID:     strings.TrimSpace(providerMailID),
		MIMEMessageID:      strings.TrimSpace(parsed.Header.Get("Message-ID")),
		InReplyTo:          inReplyTo,
		References:         references,
		FromAddress:        fromAddress,
		FromDisplayName:    fromDisplayName,
		FromAddressCount:   fromAddressCount,
		Subject:            strings.TrimSpace(parsed.Header.Get("Subject")),
		ReceivedAt:         receivedAt.UTC(),
		OriginalMessageURL: strings.TrimSpace(originalMessageURL),
	})
	return reply, true, nil
}

func frankZohoVerifyHeaderAddressList(header mail.Header, key string, want []string) error {
	got := make([]string, 0, len(want))
	if strings.TrimSpace(header.Get(key)) != "" {
		addresses, err := header.AddressList(key)
		if err != nil {
			return fmt.Errorf("%s: proof %s header is invalid: %w", frankZohoSendEmailToolName, key, err)
		}
		for _, address := range addresses {
			got = append(got, strings.ToLower(strings.TrimSpace(address.Address)))
		}
	}
	normalizedWant := make([]string, 0, len(want))
	for _, address := range want {
		normalizedWant = append(normalizedWant, strings.ToLower(strings.TrimSpace(address)))
	}
	if len(got) != len(normalizedWant) {
		return fmt.Errorf("%s: proof %s header recipient count %d does not match campaign outbound action %d", frankZohoSendEmailToolName, key, len(got), len(normalizedWant))
	}
	for i := range got {
		if got[i] != normalizedWant[i] {
			return fmt.Errorf("%s: proof %s header recipient %q does not match campaign outbound action %q", frankZohoSendEmailToolName, key, got[i], normalizedWant[i])
		}
	}
	return nil
}

func parseFrankZohoMailboxReceivedAt(value frankZohoFlexibleString) (time.Time, error) {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("receivedTime is required")
	}
	if millis, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if len(trimmed) >= 13 {
			return time.UnixMilli(millis).UTC(), nil
		}
		return time.Unix(millis, 0).UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("unsupported receivedTime %q", trimmed)
	}
	return parsed.UTC(), nil
}

func frankZohoRequiredStringArg(args map[string]interface{}, key string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s: %q is required", frankZohoSendEmailToolName, key)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s: %q must be a string", frankZohoSendEmailToolName, key)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s: %q is required", frankZohoSendEmailToolName, key)
	}
	return value, nil
}

func frankZohoOptionalStringArg(args map[string]interface{}, key string) (string, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", false, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", false, fmt.Errorf("%s: %q must be a string", frankZohoSendEmailToolName, key)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, fmt.Errorf("%s: %q must not be empty when provided", frankZohoSendEmailToolName, key)
	}
	return value, true, nil
}

func frankZohoBodyFormatArg(args map[string]interface{}, key string) (string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return frankZohoMailDefaultBodyFormat, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s: %q must be a string", frankZohoSendEmailToolName, key)
	}
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "":
		return frankZohoMailDefaultBodyFormat, nil
	case "html", "plaintext":
		return value, nil
	default:
		return "", fmt.Errorf("%s: %q must be %q or %q", frankZohoSendEmailToolName, key, "plaintext", "html")
	}
}

func frankZohoRejectAddressArgs(args map[string]interface{}) error {
	for _, key := range []string{"to", "cc", "bcc"} {
		raw, ok := args[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return fmt.Errorf("%s: %q is campaign-owned and must come from step.campaign_ref zoho_email_addressing", frankZohoSendEmailToolName, key)
			}
		case []string:
			for _, value := range typed {
				if strings.TrimSpace(value) != "" {
					return fmt.Errorf("%s: %q is campaign-owned and must come from step.campaign_ref zoho_email_addressing", frankZohoSendEmailToolName, key)
				}
			}
		case []interface{}:
			for _, value := range typed {
				text, ok := value.(string)
				if !ok || strings.TrimSpace(text) != "" {
					return fmt.Errorf("%s: %q is campaign-owned and must come from step.campaign_ref zoho_email_addressing", frankZohoSendEmailToolName, key)
				}
			}
		default:
			return fmt.Errorf("%s: %q is campaign-owned and must come from step.campaign_ref zoho_email_addressing", frankZohoSendEmailToolName, key)
		}
	}
	return nil
}

func frankZohoMailAccessTokenFromEnv(context.Context) (string, error) {
	for _, key := range []string{
		"PICOBOT_ZOHO_MAIL_OAUTH_ACCESS_TOKEN",
		"ZOHO_OAUTH_ACCESS_TOKEN",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("%s requires PICOBOT_ZOHO_MAIL_OAUTH_ACCESS_TOKEN or ZOHO_OAUTH_ACCESS_TOKEN", frankZohoSendEmailToolName)
}

func frankZohoMessagesURL(apiBase string, accountID string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	return base + "/accounts/" + accountID + "/messages"
}

func frankZohoOriginalMessageURL(apiBase string, accountID string, messageID string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	return base + "/accounts/" + accountID + "/messages/" + strings.TrimSpace(messageID) + "/originalmessage"
}

func frankZohoReplyMessageURL(apiBase string, accountID string, messageID string) string {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	return base + "/accounts/" + accountID + "/messages/" + strings.TrimSpace(messageID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
