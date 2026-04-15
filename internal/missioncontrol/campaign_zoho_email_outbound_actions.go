package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type CampaignZohoEmailOutboundActionState string

const (
	CampaignZohoEmailOutboundActionStatePrepared CampaignZohoEmailOutboundActionState = "prepared"
	CampaignZohoEmailOutboundActionStateSent     CampaignZohoEmailOutboundActionState = "sent"
	CampaignZohoEmailOutboundActionStateVerified CampaignZohoEmailOutboundActionState = "verified"
	CampaignZohoEmailOutboundActionStateFailed   CampaignZohoEmailOutboundActionState = "failed"
)

type CampaignZohoEmailOutboundFailure struct {
	HTTPStatus                int    `json:"http_status"`
	ProviderStatusCode        int    `json:"provider_status_code"`
	ProviderStatusDescription string `json:"provider_status_description,omitempty"`
}

type CampaignZohoEmailOutboundAction struct {
	ActionID           string                               `json:"action_id"`
	StepID             string                               `json:"-"`
	CampaignID         string                               `json:"campaign_id"`
	State              CampaignZohoEmailOutboundActionState `json:"state"`
	Provider           string                               `json:"provider"`
	ProviderAccountID  string                               `json:"provider_account_id"`
	FromAddress        string                               `json:"from_address"`
	FromDisplayName    string                               `json:"from_display_name,omitempty"`
	Addressing         CampaignZohoEmailAddressing          `json:"addressing"`
	Subject            string                               `json:"subject"`
	BodyFormat         string                               `json:"body_format"`
	BodySHA256         string                               `json:"body_sha256"`
	PreparedAt         time.Time                            `json:"prepared_at"`
	SentAt             time.Time                            `json:"sent_at,omitempty"`
	VerifiedAt         time.Time                            `json:"verified_at,omitempty"`
	FailedAt           time.Time                            `json:"failed_at,omitempty"`
	ProviderMessageID  string                               `json:"provider_message_id,omitempty"`
	ProviderMailID     string                               `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string                               `json:"mime_message_id,omitempty"`
	OriginalMessageURL string                               `json:"original_message_url,omitempty"`
	Failure            CampaignZohoEmailOutboundFailure     `json:"failure,omitempty"`
}

func NormalizeCampaignZohoEmailOutboundAction(action CampaignZohoEmailOutboundAction) CampaignZohoEmailOutboundAction {
	action.ActionID = strings.TrimSpace(action.ActionID)
	action.StepID = strings.TrimSpace(action.StepID)
	action.CampaignID = strings.TrimSpace(action.CampaignID)
	action.State = CampaignZohoEmailOutboundActionState(strings.TrimSpace(string(action.State)))
	action.Provider = strings.TrimSpace(action.Provider)
	action.ProviderAccountID = strings.TrimSpace(action.ProviderAccountID)
	action.FromAddress = strings.TrimSpace(action.FromAddress)
	action.FromDisplayName = strings.TrimSpace(action.FromDisplayName)
	action.Addressing = CampaignZohoEmailAddressing{
		To:  normalizeCampaignEmailAddressList(action.Addressing.To),
		CC:  normalizeCampaignEmailAddressList(action.Addressing.CC),
		BCC: normalizeCampaignEmailAddressList(action.Addressing.BCC),
	}
	action.Subject = strings.TrimSpace(action.Subject)
	action.BodyFormat = strings.TrimSpace(action.BodyFormat)
	action.BodySHA256 = strings.ToLower(strings.TrimSpace(action.BodySHA256))
	action.PreparedAt = action.PreparedAt.UTC()
	action.SentAt = action.SentAt.UTC()
	action.VerifiedAt = action.VerifiedAt.UTC()
	action.FailedAt = action.FailedAt.UTC()
	action.ProviderMessageID = strings.TrimSpace(action.ProviderMessageID)
	action.ProviderMailID = strings.TrimSpace(action.ProviderMailID)
	action.MIMEMessageID = strings.TrimSpace(action.MIMEMessageID)
	action.OriginalMessageURL = strings.TrimSpace(action.OriginalMessageURL)
	action.Failure.ProviderStatusDescription = strings.TrimSpace(action.Failure.ProviderStatusDescription)
	return action
}

func ValidateCampaignZohoEmailOutboundAction(action CampaignZohoEmailOutboundAction) error {
	normalized := NormalizeCampaignZohoEmailOutboundAction(action)
	if strings.TrimSpace(normalized.ActionID) == "" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action action_id is required")
	}
	if strings.TrimSpace(normalized.StepID) == "" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action step_id is required")
	}
	if err := validateCampaignID(normalized.CampaignID, "mission runtime campaign zoho email outbound action"); err != nil {
		return err
	}
	switch normalized.State {
	case CampaignZohoEmailOutboundActionStatePrepared, CampaignZohoEmailOutboundActionStateSent, CampaignZohoEmailOutboundActionStateVerified, CampaignZohoEmailOutboundActionStateFailed:
	default:
		return fmt.Errorf("mission runtime campaign zoho email outbound action state %q is invalid", strings.TrimSpace(string(normalized.State)))
	}
	if normalized.Provider != "zoho_mail" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action provider %q must be %q", normalized.Provider, "zoho_mail")
	}
	if strings.TrimSpace(normalized.ProviderAccountID) == "" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action provider_account_id is required")
	}
	if strings.TrimSpace(normalized.FromAddress) == "" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action from_address is required")
	}
	if err := validateCampaignZohoEmailAddressing(&normalized.Addressing); err != nil {
		return err
	}
	if strings.TrimSpace(normalized.Subject) == "" {
		return fmt.Errorf("mission runtime campaign zoho email outbound action subject is required")
	}
	switch normalized.BodyFormat {
	case "plaintext", "html":
	default:
		return fmt.Errorf("mission runtime campaign zoho email outbound action body_format %q is invalid", normalized.BodyFormat)
	}
	if len(normalized.BodySHA256) != 64 || !isLowerHexString(normalized.BodySHA256) {
		return fmt.Errorf("mission runtime campaign zoho email outbound action body_sha256 must be a lowercase sha256 hex digest")
	}
	if normalized.PreparedAt.IsZero() {
		return fmt.Errorf("mission runtime campaign zoho email outbound action prepared_at is required")
	}
	switch normalized.State {
	case CampaignZohoEmailOutboundActionStatePrepared:
		if normalized.ProviderMessageID != "" || normalized.ProviderMailID != "" || normalized.MIMEMessageID != "" || normalized.OriginalMessageURL != "" || !normalized.SentAt.IsZero() || !normalized.VerifiedAt.IsZero() || !normalized.FailedAt.IsZero() || normalized.Failure.HTTPStatus != 0 || normalized.Failure.ProviderStatusCode != 0 || normalized.Failure.ProviderStatusDescription != "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound prepared action must not include provider receipt proof")
		}
	case CampaignZohoEmailOutboundActionStateSent:
		if normalized.SentAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action sent_at is required")
		}
		if strings.TrimSpace(normalized.ProviderMessageID) == "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action provider_message_id is required")
		}
		if strings.TrimSpace(normalized.OriginalMessageURL) == "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action original_message_url is required")
		}
		if normalized.SentAt.Before(normalized.PreparedAt) {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action sent_at must be on or after prepared_at")
		}
		if !normalized.VerifiedAt.IsZero() || !normalized.FailedAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action must not include terminal finalize timestamps beyond sent_at")
		}
		if normalized.Failure.HTTPStatus != 0 || normalized.Failure.ProviderStatusCode != 0 || normalized.Failure.ProviderStatusDescription != "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound sent action must not include provider rejection evidence")
		}
	case CampaignZohoEmailOutboundActionStateVerified:
		if normalized.SentAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action sent_at is required")
		}
		if normalized.VerifiedAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action verified_at is required")
		}
		if strings.TrimSpace(normalized.ProviderMessageID) == "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action provider_message_id is required")
		}
		if strings.TrimSpace(normalized.OriginalMessageURL) == "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action original_message_url is required")
		}
		if normalized.SentAt.Before(normalized.PreparedAt) {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action sent_at must be on or after prepared_at")
		}
		if normalized.VerifiedAt.Before(normalized.SentAt) {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action verified_at must be on or after sent_at")
		}
		if !normalized.FailedAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action failed_at must be empty")
		}
		if normalized.Failure.HTTPStatus != 0 || normalized.Failure.ProviderStatusCode != 0 || normalized.Failure.ProviderStatusDescription != "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound verified action must not include provider rejection evidence")
		}
	case CampaignZohoEmailOutboundActionStateFailed:
		if !normalized.SentAt.IsZero() || !normalized.VerifiedAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action must not include sent or verified timestamps")
		}
		if normalized.FailedAt.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action failed_at is required")
		}
		if normalized.FailedAt.Before(normalized.PreparedAt) {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action failed_at must be on or after prepared_at")
		}
		if normalized.ProviderMessageID != "" || normalized.ProviderMailID != "" || normalized.MIMEMessageID != "" || normalized.OriginalMessageURL != "" {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action must not include accepted-send proof")
		}
		if normalized.Failure.HTTPStatus <= 0 {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action failure.http_status is required")
		}
		if normalized.Failure.ProviderStatusCode == 0 {
			return fmt.Errorf("mission runtime campaign zoho email outbound failed action failure.provider_status_code is required")
		}
	}
	if normalized.ActionID != normalizedCampaignZohoEmailOutboundActionID(normalized) {
		return fmt.Errorf("mission runtime campaign zoho email outbound action_id %q does not match normalized intent", normalized.ActionID)
	}
	return nil
}

func CampaignZohoEmailBodySHA256(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func BuildCampaignZohoEmailOutboundPreparedAction(stepID, campaignID, providerAccountID, fromAddress, fromDisplayName string, addressing CampaignZohoEmailAddressing, subject, bodyFormat, body string, preparedAt time.Time) (CampaignZohoEmailOutboundAction, error) {
	action := CampaignZohoEmailOutboundAction{
		StepID:            stepID,
		CampaignID:        campaignID,
		State:             CampaignZohoEmailOutboundActionStatePrepared,
		Provider:          "zoho_mail",
		ProviderAccountID: providerAccountID,
		FromAddress:       fromAddress,
		FromDisplayName:   fromDisplayName,
		Addressing:        addressing,
		Subject:           subject,
		BodyFormat:        bodyFormat,
		BodySHA256:        CampaignZohoEmailBodySHA256(body),
		PreparedAt:        preparedAt.UTC(),
	}
	action = NormalizeCampaignZohoEmailOutboundAction(action)
	action.ActionID = normalizedCampaignZohoEmailOutboundActionID(action)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		return CampaignZohoEmailOutboundAction{}, err
	}
	return action, nil
}

func BuildCampaignZohoEmailOutboundSentAction(prepared CampaignZohoEmailOutboundAction, receipt FrankZohoSendReceipt, sentAt time.Time) (CampaignZohoEmailOutboundAction, error) {
	action := NormalizeCampaignZohoEmailOutboundAction(prepared)
	if action.State != CampaignZohoEmailOutboundActionStatePrepared {
		return CampaignZohoEmailOutboundAction{}, fmt.Errorf("mission runtime campaign zoho email outbound sent action requires prepared state")
	}
	normalizedReceipt := NormalizeFrankZohoSendReceipt(receipt)
	action.State = CampaignZohoEmailOutboundActionStateSent
	action.SentAt = sentAt.UTC()
	action.Provider = normalizedReceipt.Provider
	action.ProviderAccountID = normalizedReceipt.ProviderAccountID
	action.FromAddress = normalizedReceipt.FromAddress
	action.FromDisplayName = normalizedReceipt.FromDisplayName
	action.ProviderMessageID = normalizedReceipt.ProviderMessageID
	action.ProviderMailID = normalizedReceipt.ProviderMailID
	action.MIMEMessageID = normalizedReceipt.MIMEMessageID
	action.OriginalMessageURL = normalizedReceipt.OriginalMessageURL
	action.ActionID = normalizedCampaignZohoEmailOutboundActionID(action)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		return CampaignZohoEmailOutboundAction{}, err
	}
	return action, nil
}

func BuildCampaignZohoEmailOutboundFailedAction(prepared CampaignZohoEmailOutboundAction, failure CampaignZohoEmailOutboundFailure, failedAt time.Time) (CampaignZohoEmailOutboundAction, error) {
	action := NormalizeCampaignZohoEmailOutboundAction(prepared)
	if action.State != CampaignZohoEmailOutboundActionStatePrepared {
		return CampaignZohoEmailOutboundAction{}, fmt.Errorf("mission runtime campaign zoho email outbound failed action requires prepared state")
	}
	action.State = CampaignZohoEmailOutboundActionStateFailed
	action.FailedAt = failedAt.UTC()
	action.Failure = CampaignZohoEmailOutboundFailure{
		HTTPStatus:                failure.HTTPStatus,
		ProviderStatusCode:        failure.ProviderStatusCode,
		ProviderStatusDescription: strings.TrimSpace(failure.ProviderStatusDescription),
	}
	action.ActionID = normalizedCampaignZohoEmailOutboundActionID(action)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		return CampaignZohoEmailOutboundAction{}, err
	}
	return action, nil
}

func UpsertCampaignZohoEmailOutboundAction(runtime JobRuntimeState, action CampaignZohoEmailOutboundAction) (JobRuntimeState, bool, error) {
	normalized := NormalizeCampaignZohoEmailOutboundAction(action)
	if err := ValidateCampaignZohoEmailOutboundAction(normalized); err != nil {
		return JobRuntimeState{}, false, err
	}

	next := *CloneJobRuntimeState(&runtime)
	for i, existing := range next.CampaignZohoEmailOutboundActions {
		if existing.ActionID != normalized.ActionID {
			continue
		}
		if reflect.DeepEqual(NormalizeCampaignZohoEmailOutboundAction(existing), normalized) {
			return next, false, nil
		}
		next.CampaignZohoEmailOutboundActions[i] = normalized
		return next, true, nil
	}
	next.CampaignZohoEmailOutboundActions = append(next.CampaignZohoEmailOutboundActions, normalized)
	return next, true, nil
}

func FindCampaignZohoEmailOutboundAction(runtime JobRuntimeState, actionID string) (CampaignZohoEmailOutboundAction, bool) {
	normalizedActionID := strings.TrimSpace(actionID)
	if normalizedActionID == "" {
		return CampaignZohoEmailOutboundAction{}, false
	}
	for _, action := range runtime.CampaignZohoEmailOutboundActions {
		if action.ActionID == normalizedActionID {
			return NormalizeCampaignZohoEmailOutboundAction(action), true
		}
	}
	return CampaignZohoEmailOutboundAction{}, false
}

func cloneCampaignZohoEmailOutboundActions(actions []CampaignZohoEmailOutboundAction) []CampaignZohoEmailOutboundAction {
	if len(actions) == 0 {
		return nil
	}
	cloned := make([]CampaignZohoEmailOutboundAction, len(actions))
	copy(cloned, actions)
	return cloned
}

func normalizedCampaignZohoEmailOutboundActionID(action CampaignZohoEmailOutboundAction) string {
	normalized := NormalizeCampaignZohoEmailOutboundAction(action)
	return "campaign_zoho_email_outbound_" + projectedStoreHash(
		normalized.StepID,
		normalized.CampaignID,
		normalized.Provider,
		normalized.ProviderAccountID,
		normalized.FromAddress,
		strings.Join(normalized.Addressing.To, "\x1e"),
		strings.Join(normalized.Addressing.CC, "\x1e"),
		strings.Join(normalized.Addressing.BCC, "\x1e"),
		normalized.Subject,
		normalized.BodyFormat,
		normalized.BodySHA256,
	)
}

func isLowerHexString(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
