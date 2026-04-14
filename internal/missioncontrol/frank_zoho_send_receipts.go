package missioncontrol

import (
	"encoding/json"
	"fmt"
	"strings"
)

const frankZohoSendEmailToolName = "frank_zoho_send_email"

type FrankZohoSendReceipt struct {
	StepID             string `json:"-"`
	Provider           string `json:"provider"`
	ProviderAccountID  string `json:"provider_account_id"`
	FromAddress        string `json:"from_address"`
	FromDisplayName    string `json:"from_display_name"`
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	OriginalMessageURL string `json:"original_message_url"`
}

func NormalizeFrankZohoSendReceipt(receipt FrankZohoSendReceipt) FrankZohoSendReceipt {
	receipt.StepID = strings.TrimSpace(receipt.StepID)
	receipt.Provider = strings.TrimSpace(receipt.Provider)
	receipt.ProviderAccountID = strings.TrimSpace(receipt.ProviderAccountID)
	receipt.FromAddress = strings.TrimSpace(receipt.FromAddress)
	receipt.FromDisplayName = strings.TrimSpace(receipt.FromDisplayName)
	receipt.ProviderMessageID = strings.TrimSpace(receipt.ProviderMessageID)
	receipt.ProviderMailID = strings.TrimSpace(receipt.ProviderMailID)
	receipt.MIMEMessageID = strings.TrimSpace(receipt.MIMEMessageID)
	receipt.OriginalMessageURL = strings.TrimSpace(receipt.OriginalMessageURL)
	return receipt
}

func ValidateFrankZohoSendReceipt(receipt FrankZohoSendReceipt) error {
	normalized := NormalizeFrankZohoSendReceipt(receipt)
	if normalized.StepID == "" {
		return fmt.Errorf("mission runtime frank zoho send receipt step_id is required")
	}
	if normalized.Provider == "" {
		return fmt.Errorf("mission runtime frank zoho send receipt provider is required")
	}
	if normalized.ProviderAccountID == "" {
		return fmt.Errorf("mission runtime frank zoho send receipt provider_account_id is required")
	}
	if normalized.ProviderMessageID == "" {
		return fmt.Errorf("mission runtime frank zoho send receipt provider_message_id is required")
	}
	if normalized.OriginalMessageURL == "" {
		return fmt.Errorf("mission runtime frank zoho send receipt original_message_url is required")
	}
	return nil
}

func ParseFrankZohoSendReceipt(result string) (FrankZohoSendReceipt, error) {
	var receipt FrankZohoSendReceipt
	if err := json.Unmarshal([]byte(strings.TrimSpace(result)), &receipt); err != nil {
		return FrankZohoSendReceipt{}, fmt.Errorf("parse frank zoho send receipt: %w", err)
	}
	return NormalizeFrankZohoSendReceipt(receipt), nil
}

func AppendFrankZohoSendReceipt(runtime JobRuntimeState, stepID string, result string) (JobRuntimeState, bool, error) {
	receipt, err := ParseFrankZohoSendReceipt(result)
	if err != nil {
		return JobRuntimeState{}, false, err
	}
	receipt.StepID = strings.TrimSpace(stepID)
	if err := ValidateFrankZohoSendReceipt(receipt); err != nil {
		return JobRuntimeState{}, false, err
	}

	next := *CloneJobRuntimeState(&runtime)
	key := normalizedFrankZohoSendReceiptKey(receipt)
	for _, existing := range next.FrankZohoSendReceipts {
		if normalizedFrankZohoSendReceiptKey(existing) == key {
			return next, false, nil
		}
	}
	next.FrankZohoSendReceipts = append(next.FrankZohoSendReceipts, receipt)
	return next, true, nil
}

func cloneFrankZohoSendReceipts(receipts []FrankZohoSendReceipt) []FrankZohoSendReceipt {
	if len(receipts) == 0 {
		return nil
	}
	cloned := make([]FrankZohoSendReceipt, len(receipts))
	copy(cloned, receipts)
	return cloned
}

func normalizedFrankZohoSendReceiptKey(receipt FrankZohoSendReceipt) string {
	normalized := NormalizeFrankZohoSendReceipt(receipt)
	return strings.Join([]string{
		normalized.StepID,
		normalized.Provider,
		normalized.ProviderAccountID,
		normalized.ProviderMessageID,
	}, "\x1f")
}
