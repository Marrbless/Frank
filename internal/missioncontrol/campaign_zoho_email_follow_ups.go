package missioncontrol

import (
	"fmt"
	"sort"
	"strings"
)

type CampaignZohoEmailFollowUpTarget struct {
	InboundReply   FrankZohoInboundReplyRecord
	OutboundAction CampaignZohoEmailOutboundActionRecord
}

func LoadCommittedCampaignZohoEmailFollowUpTarget(root, campaignID, inboundReplyID string) (CampaignZohoEmailFollowUpTarget, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CampaignZohoEmailFollowUpTarget{}, err
	}
	if err := validateCampaignID(campaignID, "campaign zoho email follow-up"); err != nil {
		return CampaignZohoEmailFollowUpTarget{}, err
	}
	inboundReplyID = strings.TrimSpace(inboundReplyID)
	if inboundReplyID == "" {
		return CampaignZohoEmailFollowUpTarget{}, fmt.Errorf("campaign zoho email follow-up inbound_reply_id is required")
	}

	inboundRecords, err := ListCommittedAllFrankZohoInboundReplyRecords(root)
	if err != nil {
		return CampaignZohoEmailFollowUpTarget{}, err
	}
	var reply FrankZohoInboundReplyRecord
	found := false
	for _, record := range inboundRecords {
		if strings.TrimSpace(record.ReplyID) != inboundReplyID {
			continue
		}
		reply = record
		found = true
		break
	}
	if !found {
		return CampaignZohoEmailFollowUpTarget{}, fmt.Errorf("campaign zoho email follow-up inbound_reply_id %q was not found in committed Zoho inbound reply records", inboundReplyID)
	}

	outboundRecords, err := ListCommittedAllCampaignZohoEmailOutboundActionRecords(root)
	if err != nil {
		return CampaignZohoEmailFollowUpTarget{}, err
	}
	action, ok := attributedCampaignZohoEmailOutboundActionForReply(reply, outboundRecords)
	if !ok {
		return CampaignZohoEmailFollowUpTarget{}, fmt.Errorf("campaign zoho email follow-up inbound_reply_id %q is not uniquely attributable from committed outbound MIME linkage", inboundReplyID)
	}
	if strings.TrimSpace(action.CampaignID) != strings.TrimSpace(campaignID) {
		return CampaignZohoEmailFollowUpTarget{}, fmt.Errorf("campaign zoho email follow-up inbound_reply_id %q is attributable to campaign %q, not %q", inboundReplyID, strings.TrimSpace(action.CampaignID), strings.TrimSpace(campaignID))
	}

	return CampaignZohoEmailFollowUpTarget{
		InboundReply:   reply,
		OutboundAction: action,
	}, nil
}

func ListCommittedCampaignZohoEmailFollowUpActionsByInboundReply(root, inboundReplyID string) ([]CampaignZohoEmailOutboundActionRecord, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	inboundReplyID = strings.TrimSpace(inboundReplyID)
	if inboundReplyID == "" {
		return nil, fmt.Errorf("campaign zoho email follow-up inbound_reply_id is required")
	}

	allActions, err := ListCommittedAllCampaignZohoEmailOutboundActionRecords(root)
	if err != nil {
		return nil, err
	}
	matched := make([]CampaignZohoEmailOutboundActionRecord, 0, len(allActions))
	for _, record := range allActions {
		if strings.TrimSpace(record.ReplyToInboundReplyID) != inboundReplyID {
			continue
		}
		matched = append(matched, record)
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].PreparedAt.Equal(matched[j].PreparedAt) {
			return matched[i].PreparedAt.Before(matched[j].PreparedAt)
		}
		return matched[i].ActionID < matched[j].ActionID
	})
	return matched, nil
}

func attributedCampaignZohoEmailOutboundActionForReply(reply FrankZohoInboundReplyRecord, outboundRecords []CampaignZohoEmailOutboundActionRecord) (CampaignZohoEmailOutboundActionRecord, bool) {
	matches := make([]CampaignZohoEmailOutboundActionRecord, 0, 1)
	for _, record := range outboundRecords {
		action := NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:                record.ActionID,
			StepID:                  record.StepID,
			CampaignID:              record.CampaignID,
			State:                   CampaignZohoEmailOutboundActionState(record.State),
			Provider:                record.Provider,
			ProviderAccountID:       record.ProviderAccountID,
			FromAddress:             record.FromAddress,
			FromDisplayName:         record.FromDisplayName,
			Addressing:              record.Addressing,
			Subject:                 record.Subject,
			BodyFormat:              record.BodyFormat,
			BodySHA256:              record.BodySHA256,
			PreparedAt:              record.PreparedAt,
			SentAt:                  record.SentAt,
			VerifiedAt:              record.VerifiedAt,
			FailedAt:                record.FailedAt,
			ReplyToInboundReplyID:   record.ReplyToInboundReplyID,
			ReplyToOutboundActionID: record.ReplyToOutboundActionID,
			ProviderMessageID:       record.ProviderMessageID,
			ProviderMailID:          record.ProviderMailID,
			MIMEMessageID:           record.MIMEMessageID,
			OriginalMessageURL:      record.OriginalMessageURL,
			Failure:                 record.Failure,
		})
		if attributedCampaignZohoEmailReplyMatchesAction(reply, action) {
			matches = append(matches, record)
		}
	}
	if len(matches) != 1 {
		return CampaignZohoEmailOutboundActionRecord{}, false
	}
	return matches[0], true
}

func attributedCampaignZohoEmailOutboundActionForBounce(evidence FrankZohoBounceEvidence, outboundRecords []CampaignZohoEmailOutboundActionRecord) (CampaignZohoEmailOutboundActionRecord, bool) {
	matches := make([]CampaignZohoEmailOutboundActionRecord, 0, 1)
	for _, record := range outboundRecords {
		action := NormalizeCampaignZohoEmailOutboundAction(CampaignZohoEmailOutboundAction{
			ActionID:                record.ActionID,
			StepID:                  record.StepID,
			CampaignID:              record.CampaignID,
			State:                   CampaignZohoEmailOutboundActionState(record.State),
			Provider:                record.Provider,
			ProviderAccountID:       record.ProviderAccountID,
			FromAddress:             record.FromAddress,
			FromDisplayName:         record.FromDisplayName,
			Addressing:              record.Addressing,
			Subject:                 record.Subject,
			BodyFormat:              record.BodyFormat,
			BodySHA256:              record.BodySHA256,
			PreparedAt:              record.PreparedAt,
			SentAt:                  record.SentAt,
			VerifiedAt:              record.VerifiedAt,
			FailedAt:                record.FailedAt,
			ReplyToInboundReplyID:   record.ReplyToInboundReplyID,
			ReplyToOutboundActionID: record.ReplyToOutboundActionID,
			ProviderMessageID:       record.ProviderMessageID,
			ProviderMailID:          record.ProviderMailID,
			MIMEMessageID:           record.MIMEMessageID,
			OriginalMessageURL:      record.OriginalMessageURL,
			Failure:                 record.Failure,
		})
		if attributedCampaignZohoEmailBounceMatchesAction(evidence, action) {
			matches = append(matches, record)
		}
	}
	if len(matches) != 1 {
		return CampaignZohoEmailOutboundActionRecord{}, false
	}
	return matches[0], true
}

func AttributedCampaignZohoEmailOutboundActionForBounce(evidence FrankZohoBounceEvidence, outboundRecords []CampaignZohoEmailOutboundActionRecord) (CampaignZohoEmailOutboundActionRecord, bool) {
	return attributedCampaignZohoEmailOutboundActionForBounce(evidence, outboundRecords)
}

func attributedCampaignZohoEmailReplyMatchesAction(reply FrankZohoInboundReplyRecord, action CampaignZohoEmailOutboundAction) bool {
	if action.MIMEMessageID == "" {
		return false
	}
	matchIDs := make([]string, 0, 1+len(reply.References))
	if strings.TrimSpace(reply.InReplyTo) != "" {
		matchIDs = append(matchIDs, strings.TrimSpace(reply.InReplyTo))
	}
	matchIDs = append(matchIDs, reply.References...)

	matched := false
	for _, candidate := range matchIDs {
		if strings.TrimSpace(candidate) == action.MIMEMessageID {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	outboundAt := action.VerifiedAt
	if outboundAt.IsZero() {
		outboundAt = action.SentAt
	}
	if outboundAt.IsZero() {
		return false
	}
	return !reply.ReceivedAt.Before(outboundAt)
}

func attributedCampaignZohoEmailBounceMatchesAction(evidence FrankZohoBounceEvidence, action CampaignZohoEmailOutboundAction) bool {
	matched := (evidence.OriginalProviderMessageID != "" && evidence.OriginalProviderMessageID == action.ProviderMessageID) ||
		(evidence.OriginalProviderMailID != "" && evidence.OriginalProviderMailID == action.ProviderMailID) ||
		(evidence.OriginalMIMEMessageID != "" && evidence.OriginalMIMEMessageID == action.MIMEMessageID)
	if !matched {
		return false
	}

	outboundAt := action.SentAt
	if outboundAt.IsZero() {
		outboundAt = action.VerifiedAt
	}
	if outboundAt.IsZero() {
		outboundAt = action.FailedAt
	}
	if outboundAt.IsZero() {
		return false
	}
	return !evidence.ReceivedAt.Before(outboundAt)
}
