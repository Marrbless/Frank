package missioncontrol

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type FrankZohoInboundReply struct {
	ReplyID            string    `json:"reply_id"`
	StepID             string    `json:"-"`
	Provider           string    `json:"provider"`
	ProviderAccountID  string    `json:"provider_account_id"`
	ProviderMessageID  string    `json:"provider_message_id"`
	ProviderMailID     string    `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string    `json:"mime_message_id,omitempty"`
	InReplyTo          string    `json:"in_reply_to,omitempty"`
	References         []string  `json:"references,omitempty"`
	FromAddress        string    `json:"from_address,omitempty"`
	FromDisplayName    string    `json:"from_display_name,omitempty"`
	Subject            string    `json:"subject,omitempty"`
	ReceivedAt         time.Time `json:"received_at"`
	OriginalMessageURL string    `json:"original_message_url"`
}

func NormalizeFrankZohoInboundReply(reply FrankZohoInboundReply) FrankZohoInboundReply {
	reply.ReplyID = strings.TrimSpace(reply.ReplyID)
	reply.StepID = strings.TrimSpace(reply.StepID)
	reply.Provider = strings.TrimSpace(reply.Provider)
	reply.ProviderAccountID = strings.TrimSpace(reply.ProviderAccountID)
	reply.ProviderMessageID = strings.TrimSpace(reply.ProviderMessageID)
	reply.ProviderMailID = strings.TrimSpace(reply.ProviderMailID)
	reply.MIMEMessageID = strings.TrimSpace(reply.MIMEMessageID)
	reply.InReplyTo = strings.TrimSpace(reply.InReplyTo)
	reply.References = normalizeFrankZohoInboundReplyReferences(reply.References)
	reply.FromAddress = strings.TrimSpace(reply.FromAddress)
	reply.FromDisplayName = strings.TrimSpace(reply.FromDisplayName)
	reply.Subject = strings.TrimSpace(reply.Subject)
	reply.ReceivedAt = reply.ReceivedAt.UTC()
	reply.OriginalMessageURL = strings.TrimSpace(reply.OriginalMessageURL)
	return reply
}

func ValidateFrankZohoInboundReply(reply FrankZohoInboundReply) error {
	normalized := NormalizeFrankZohoInboundReply(reply)
	if normalized.StepID == "" {
		return fmt.Errorf("mission runtime frank zoho inbound reply step_id is required")
	}
	if normalized.Provider != "zoho_mail" {
		return fmt.Errorf("mission runtime frank zoho inbound reply provider %q must be %q", normalized.Provider, "zoho_mail")
	}
	if normalized.ProviderAccountID == "" {
		return fmt.Errorf("mission runtime frank zoho inbound reply provider_account_id is required")
	}
	if normalized.ProviderMessageID == "" {
		return fmt.Errorf("mission runtime frank zoho inbound reply provider_message_id is required")
	}
	if normalized.ReceivedAt.IsZero() {
		return fmt.Errorf("mission runtime frank zoho inbound reply received_at is required")
	}
	if normalized.OriginalMessageURL == "" {
		return fmt.Errorf("mission runtime frank zoho inbound reply original_message_url is required")
	}
	if normalized.ReplyID != normalizedFrankZohoInboundReplyID(normalized) {
		return fmt.Errorf("mission runtime frank zoho inbound reply reply_id %q does not match normalized provider identity", normalized.ReplyID)
	}
	return nil
}

func AppendFrankZohoInboundReply(runtime JobRuntimeState, reply FrankZohoInboundReply) (JobRuntimeState, bool, error) {
	normalized := NormalizeFrankZohoInboundReply(reply)
	if normalized.ReplyID == "" {
		normalized.ReplyID = normalizedFrankZohoInboundReplyID(normalized)
	}
	if err := ValidateFrankZohoInboundReply(normalized); err != nil {
		return JobRuntimeState{}, false, err
	}

	next := *CloneJobRuntimeState(&runtime)
	for i, existing := range next.FrankZohoInboundReplies {
		if strings.TrimSpace(existing.ReplyID) != normalized.ReplyID {
			continue
		}
		if reflect.DeepEqual(NormalizeFrankZohoInboundReply(existing), normalized) {
			return next, false, nil
		}
		next.FrankZohoInboundReplies[i] = normalized
		return next, true, nil
	}
	next.FrankZohoInboundReplies = append(next.FrankZohoInboundReplies, normalized)
	return next, true, nil
}

func cloneFrankZohoInboundReplies(replies []FrankZohoInboundReply) []FrankZohoInboundReply {
	if len(replies) == 0 {
		return nil
	}
	cloned := make([]FrankZohoInboundReply, len(replies))
	for i, reply := range replies {
		cloned[i] = NormalizeFrankZohoInboundReply(reply)
	}
	return cloned
}

func normalizeFrankZohoInboundReplyReferences(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizedFrankZohoInboundReplyID(reply FrankZohoInboundReply) string {
	normalized := NormalizeFrankZohoInboundReply(reply)
	return "frank_zoho_inbound_reply_" + projectedStoreHash(
		normalized.Provider,
		normalized.ProviderAccountID,
		normalized.ProviderMessageID,
	)
}
