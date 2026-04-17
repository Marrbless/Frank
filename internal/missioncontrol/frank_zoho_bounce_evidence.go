package missioncontrol

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type FrankZohoBounceEvidence struct {
	BounceID                  string    `json:"bounce_id"`
	StepID                    string    `json:"-"`
	Provider                  string    `json:"provider"`
	ProviderAccountID         string    `json:"provider_account_id"`
	ProviderMessageID         string    `json:"provider_message_id"`
	ProviderMailID            string    `json:"provider_mail_id,omitempty"`
	MIMEMessageID             string    `json:"mime_message_id,omitempty"`
	InReplyTo                 string    `json:"in_reply_to,omitempty"`
	References                []string  `json:"references,omitempty"`
	OriginalProviderMessageID string    `json:"original_provider_message_id,omitempty"`
	OriginalProviderMailID    string    `json:"original_provider_mail_id,omitempty"`
	OriginalMIMEMessageID     string    `json:"original_mime_message_id,omitempty"`
	FinalRecipient            string    `json:"final_recipient,omitempty"`
	DiagnosticCode            string    `json:"diagnostic_code,omitempty"`
	ReceivedAt                time.Time `json:"received_at"`
	OriginalMessageURL        string    `json:"original_message_url"`
	CampaignID                string    `json:"campaign_id,omitempty"`
	OutboundActionID          string    `json:"outbound_action_id,omitempty"`
}

func NormalizeFrankZohoBounceEvidence(evidence FrankZohoBounceEvidence) FrankZohoBounceEvidence {
	evidence.BounceID = strings.TrimSpace(evidence.BounceID)
	evidence.StepID = strings.TrimSpace(evidence.StepID)
	evidence.Provider = strings.TrimSpace(evidence.Provider)
	evidence.ProviderAccountID = strings.TrimSpace(evidence.ProviderAccountID)
	evidence.ProviderMessageID = strings.TrimSpace(evidence.ProviderMessageID)
	evidence.ProviderMailID = strings.TrimSpace(evidence.ProviderMailID)
	evidence.MIMEMessageID = strings.TrimSpace(evidence.MIMEMessageID)
	evidence.InReplyTo = strings.TrimSpace(evidence.InReplyTo)
	evidence.References = normalizeFrankZohoInboundReplyReferences(evidence.References)
	evidence.OriginalProviderMessageID = strings.TrimSpace(evidence.OriginalProviderMessageID)
	evidence.OriginalProviderMailID = strings.TrimSpace(evidence.OriginalProviderMailID)
	evidence.OriginalMIMEMessageID = strings.TrimSpace(evidence.OriginalMIMEMessageID)
	evidence.FinalRecipient = strings.TrimSpace(evidence.FinalRecipient)
	evidence.DiagnosticCode = strings.TrimSpace(evidence.DiagnosticCode)
	evidence.ReceivedAt = evidence.ReceivedAt.UTC()
	evidence.OriginalMessageURL = strings.TrimSpace(evidence.OriginalMessageURL)
	evidence.CampaignID = strings.TrimSpace(evidence.CampaignID)
	evidence.OutboundActionID = strings.TrimSpace(evidence.OutboundActionID)
	return evidence
}

func ValidateFrankZohoBounceEvidence(evidence FrankZohoBounceEvidence) error {
	normalized := NormalizeFrankZohoBounceEvidence(evidence)
	if normalized.StepID == "" {
		return fmt.Errorf("mission runtime frank zoho bounce evidence step_id is required")
	}
	if normalized.Provider != "zoho_mail" {
		return fmt.Errorf("mission runtime frank zoho bounce evidence provider %q must be %q", normalized.Provider, "zoho_mail")
	}
	if normalized.ProviderAccountID == "" {
		return fmt.Errorf("mission runtime frank zoho bounce evidence provider_account_id is required")
	}
	if normalized.ProviderMessageID == "" {
		return fmt.Errorf("mission runtime frank zoho bounce evidence provider_message_id is required")
	}
	if normalized.ReceivedAt.IsZero() {
		return fmt.Errorf("mission runtime frank zoho bounce evidence received_at is required")
	}
	if normalized.OriginalMessageURL == "" {
		return fmt.Errorf("mission runtime frank zoho bounce evidence original_message_url is required")
	}
	if (normalized.CampaignID == "") != (normalized.OutboundActionID == "") {
		return fmt.Errorf("mission runtime frank zoho bounce evidence campaign_id and outbound_action_id must either both be set or both be empty")
	}
	if normalized.BounceID != normalizedFrankZohoBounceEvidenceID(normalized) {
		return fmt.Errorf("mission runtime frank zoho bounce evidence bounce_id %q does not match normalized provider identity", normalized.BounceID)
	}
	return nil
}

func AppendFrankZohoBounceEvidence(runtime JobRuntimeState, evidence FrankZohoBounceEvidence) (JobRuntimeState, bool, error) {
	normalized := NormalizeFrankZohoBounceEvidence(evidence)
	if normalized.BounceID == "" {
		normalized.BounceID = normalizedFrankZohoBounceEvidenceID(normalized)
	}
	if err := ValidateFrankZohoBounceEvidence(normalized); err != nil {
		return JobRuntimeState{}, false, err
	}

	next := *CloneJobRuntimeState(&runtime)
	for i, existing := range next.FrankZohoBounceEvidence {
		if strings.TrimSpace(existing.BounceID) != normalized.BounceID {
			continue
		}
		if reflect.DeepEqual(NormalizeFrankZohoBounceEvidence(existing), normalized) {
			return next, false, nil
		}
		next.FrankZohoBounceEvidence[i] = normalized
		return next, true, nil
	}
	next.FrankZohoBounceEvidence = append(next.FrankZohoBounceEvidence, normalized)
	return next, true, nil
}

func cloneFrankZohoBounceEvidence(values []FrankZohoBounceEvidence) []FrankZohoBounceEvidence {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]FrankZohoBounceEvidence, len(values))
	for i, value := range values {
		cloned[i] = NormalizeFrankZohoBounceEvidence(value)
	}
	return cloned
}

func normalizedFrankZohoBounceEvidenceID(evidence FrankZohoBounceEvidence) string {
	normalized := NormalizeFrankZohoBounceEvidence(evidence)
	return "frank_zoho_bounce_evidence_" + projectedStoreHash(
		normalized.Provider,
		normalized.ProviderAccountID,
		normalized.ProviderMessageID,
	)
}
