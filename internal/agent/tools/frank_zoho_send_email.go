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
	"strings"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

const (
	frankZohoSendEmailToolName     = "frank_zoho_send_email"
	frankZohoMailAPIBase           = "https://mail.zoho.com/api"
	frankZohoMailAccountID         = "3323462000000008002"
	frankZohoMailFromAddress       = "frank@omou.online"
	frankZohoMailFromDisplayName   = "Frank"
	frankZohoMailDefaultBodyFormat = "plaintext"
	frankZohoMailDefaultEncoding   = "UTF-8"
)

const FrankZohoSendEmailToolName = frankZohoSendEmailToolName

var verifyFrankZohoCampaignSendProof = func(ctx context.Context, proof []missioncontrol.OperatorFrankZohoSendProofStatus) ([]FrankZohoSendProofVerification, error) {
	return NewFrankZohoSendProofVerifier().Verify(ctx, proof)
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
	AccountID  string
	From       string
	To         []string
	CC         []string
	BCC        []string
	Subject    string
	Body       string
	BodyFormat string
	Encoding   string
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
}

type frankZohoSendAPIResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data frankZohoSendResponseData `json:"data"`
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

func (t *FrankZohoSendEmailTool) Name() string {
	return frankZohoSendEmailToolName
}

func (t *FrankZohoSendEmailTool) Description() string {
	return "Send one email from Frank <frank@omou.online> using the fixed Zoho Mail account"
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
	if err := validateFrankZohoMailPreflight(preflight); err != nil {
		return "", err
	}
	if err := requireFrankZohoCampaignSendGate(ec, *preflight.Campaign); err != nil {
		return "", err
	}

	req, err := buildFrankZohoSendRequest(preflight, args)
	if err != nil {
		return "", err
	}

	req.AccountID = frankZohoMailAccountID
	req.From = frankZohoMailFromAddress

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
		ProviderAccountID:  frankZohoMailAccountID,
		FromAddress:        frankZohoMailFromAddress,
		FromDisplayName:    frankZohoMailFromDisplayName,
		ProviderMessageID:  string(data.MessageID),
		ProviderMailID:     string(data.MailID),
		MIMEMessageID:      firstNonEmpty(string(data.MIMEMessageID), string(data.MessageIDHeader), string(data.InternetMessage)),
		OriginalMessageURL: frankZohoOriginalMessageURL(t.apiBase, frankZohoMailAccountID, string(data.MessageID)),
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

func validateFrankZohoMailPreflight(preflight missioncontrol.ResolvedExecutionContextCampaignPreflight) error {
	if preflight.Campaign == nil {
		return fmt.Errorf("%s requires a resolved campaign preflight", frankZohoSendEmailToolName)
	}

	emailIdentityIDs := make(map[string]struct{})
	for _, identity := range preflight.Identities {
		if strings.EqualFold(strings.TrimSpace(identity.IdentityKind), "email") {
			emailIdentityIDs[strings.TrimSpace(identity.IdentityID)] = struct{}{}
		}
	}
	if len(emailIdentityIDs) == 0 {
		return fmt.Errorf("%s requires a campaign-linked Frank email identity", frankZohoSendEmailToolName)
	}
	if preflight.Campaign.ZohoEmailAddressing == nil {
		return fmt.Errorf("%s requires campaign zoho_email_addressing", frankZohoSendEmailToolName)
	}

	for _, account := range preflight.Accounts {
		if !strings.EqualFold(strings.TrimSpace(account.AccountKind), "mailbox") {
			continue
		}
		if _, ok := emailIdentityIDs[strings.TrimSpace(account.IdentityID)]; ok {
			return nil
		}
	}

	return fmt.Errorf("%s requires a campaign-linked Frank mailbox account", frankZohoSendEmailToolName)
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

func buildFrankZohoSendRequest(preflight missioncontrol.ResolvedExecutionContextCampaignPreflight, args map[string]interface{}) (frankZohoSendRequest, error) {
	if err := frankZohoRejectAddressArgs(args); err != nil {
		return frankZohoSendRequest{}, err
	}
	subject, err := frankZohoRequiredStringArg(args, "subject")
	if err != nil {
		return frankZohoSendRequest{}, err
	}
	body, err := frankZohoRequiredStringArg(args, "body")
	if err != nil {
		return frankZohoSendRequest{}, err
	}
	bodyFormat, err := frankZohoBodyFormatArg(args, "body_format")
	if err != nil {
		return frankZohoSendRequest{}, err
	}
	addressing := preflight.Campaign.ZohoEmailAddressing

	return frankZohoSendRequest{
		To:         append([]string(nil), addressing.To...),
		CC:         append([]string(nil), addressing.CC...),
		BCC:        append([]string(nil), addressing.BCC...),
		Subject:    subject,
		Body:       body,
		BodyFormat: bodyFormat,
		Encoding:   frankZohoMailDefaultEncoding,
	}, nil
}

func buildFrankZohoPreparedCampaignAction(ec missioncontrol.ExecutionContext, args map[string]interface{}, now time.Time) (missioncontrol.CampaignZohoEmailOutboundAction, error) {
	if err := missioncontrol.RequireExecutionContextCampaignReadiness(ec); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	preflight, err := missioncontrol.ResolveExecutionContextCampaignPreflight(ec)
	if err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	if err := validateFrankZohoMailPreflight(preflight); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	if err := requireFrankZohoCampaignSendGate(ec, *preflight.Campaign); err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	req, err := buildFrankZohoSendRequest(preflight, args)
	if err != nil {
		return missioncontrol.CampaignZohoEmailOutboundAction{}, err
	}
	return missioncontrol.BuildCampaignZohoEmailOutboundPreparedAction(
		ec.Step.ID,
		preflight.Campaign.CampaignID,
		frankZohoMailAccountID,
		frankZohoMailFromAddress,
		frankZohoMailFromDisplayName,
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
