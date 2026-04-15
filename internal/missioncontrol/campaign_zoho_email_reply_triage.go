package missioncontrol

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type CampaignZohoEmailReplyWorkSelection struct {
	WorkItem     CampaignZohoEmailReplyWorkItem `json:"work_item"`
	InboundReply FrankZohoInboundReplyRecord    `json:"inbound_reply"`
	NeedsReopen  bool                           `json:"needs_reopen,omitempty"`
}

func DeriveMissingCampaignZohoEmailReplyWorkItems(campaignID string, outboundRecords []CampaignZohoEmailOutboundActionRecord, inboundReplyRecords []FrankZohoInboundReplyRecord, workItemRecords []CampaignZohoEmailReplyWorkItemRecord, now time.Time) ([]CampaignZohoEmailReplyWorkItem, error) {
	if err := validateCampaignID(campaignID, "campaign zoho email reply triage"); err != nil {
		return nil, err
	}
	normalizedCampaignID := strings.TrimSpace(campaignID)
	existing := make(map[string]struct{}, len(workItemRecords))
	for _, record := range workItemRecords {
		if err := ValidateCampaignZohoEmailReplyWorkItemRecord(record); err != nil {
			return nil, err
		}
		existing[strings.TrimSpace(record.InboundReplyID)] = struct{}{}
	}

	type candidate struct {
		reply FrankZohoInboundReplyRecord
		item  CampaignZohoEmailReplyWorkItem
	}
	candidates := make([]candidate, 0, len(inboundReplyRecords))
	for _, reply := range inboundReplyRecords {
		action, ok := attributedCampaignZohoEmailOutboundActionForReply(reply, outboundRecords)
		if !ok || strings.TrimSpace(action.CampaignID) != normalizedCampaignID {
			continue
		}
		if _, ok := existing[strings.TrimSpace(reply.ReplyID)]; ok {
			continue
		}
		item, err := BuildCampaignZohoEmailReplyWorkItemOpen(normalizedCampaignID, reply.ReplyID, now)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate{reply: reply, item: item})
		existing[strings.TrimSpace(reply.ReplyID)] = struct{}{}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if !candidates[i].reply.ReceivedAt.Equal(candidates[j].reply.ReceivedAt) {
			return candidates[i].reply.ReceivedAt.Before(candidates[j].reply.ReceivedAt)
		}
		return strings.TrimSpace(candidates[i].reply.ReplyID) < strings.TrimSpace(candidates[j].reply.ReplyID)
	})
	if len(candidates) == 0 {
		return nil, nil
	}
	items := make([]CampaignZohoEmailReplyWorkItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, candidate.item)
	}
	return items, nil
}

func LoadMissingCommittedCampaignZohoEmailReplyWorkItems(root, campaignID string, now time.Time) ([]CampaignZohoEmailReplyWorkItem, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	outboundRecords, err := ListCommittedAllCampaignZohoEmailOutboundActionRecords(root)
	if err != nil {
		return nil, err
	}
	inboundReplyRecords, err := ListCommittedAllFrankZohoInboundReplyRecords(root)
	if err != nil {
		return nil, err
	}
	workItemRecords, err := ListCommittedAllCampaignZohoEmailReplyWorkItemRecords(root)
	if err != nil {
		return nil, err
	}
	return DeriveMissingCampaignZohoEmailReplyWorkItems(campaignID, outboundRecords, inboundReplyRecords, workItemRecords, now)
}

func DeriveCampaignZohoEmailReplyWorkSelection(campaignID string, workItemRecords []CampaignZohoEmailReplyWorkItemRecord, inboundReplyRecords []FrankZohoInboundReplyRecord, now time.Time) (CampaignZohoEmailReplyWorkSelection, bool, error) {
	if err := validateCampaignID(campaignID, "campaign zoho email reply triage"); err != nil {
		return CampaignZohoEmailReplyWorkSelection{}, false, err
	}
	normalizedCampaignID := strings.TrimSpace(campaignID)
	replyByID := make(map[string]FrankZohoInboundReplyRecord, len(inboundReplyRecords))
	for _, reply := range inboundReplyRecords {
		replyByID[strings.TrimSpace(reply.ReplyID)] = reply
	}

	selections := make([]CampaignZohoEmailReplyWorkSelection, 0, len(workItemRecords))
	for _, record := range workItemRecords {
		if err := ValidateCampaignZohoEmailReplyWorkItemRecord(record); err != nil {
			return CampaignZohoEmailReplyWorkSelection{}, false, err
		}
		if strings.TrimSpace(record.CampaignID) != normalizedCampaignID {
			continue
		}
		reply, ok := replyByID[strings.TrimSpace(record.InboundReplyID)]
		if !ok {
			return CampaignZohoEmailReplyWorkSelection{}, false, fmt.Errorf("campaign zoho email reply work item %q is missing committed inbound reply %q", strings.TrimSpace(record.ReplyWorkItemID), strings.TrimSpace(record.InboundReplyID))
		}

		normalizedItem := NormalizeCampaignZohoEmailReplyWorkItem(CampaignZohoEmailReplyWorkItem{
			ReplyWorkItemID:         record.ReplyWorkItemID,
			InboundReplyID:          record.InboundReplyID,
			CampaignID:              record.CampaignID,
			State:                   CampaignZohoEmailReplyWorkItemState(record.State),
			DeferredUntil:           record.DeferredUntil,
			ClaimedFollowUpActionID: record.ClaimedFollowUpActionID,
			CreatedAt:               record.CreatedAt,
			UpdatedAt:               record.UpdatedAt,
		})

		eligible := false
		needsReopen := false
		switch normalizedItem.State {
		case CampaignZohoEmailReplyWorkItemStateOpen:
			eligible = true
		case CampaignZohoEmailReplyWorkItemStateDeferred:
			if !normalizedItem.DeferredUntil.After(now.UTC()) {
				eligible = true
				needsReopen = true
			}
		case CampaignZohoEmailReplyWorkItemStateClaimed, CampaignZohoEmailReplyWorkItemStateResponded, CampaignZohoEmailReplyWorkItemStateIgnored:
		default:
			return CampaignZohoEmailReplyWorkSelection{}, false, fmt.Errorf("campaign zoho email reply work item %q state %q is not supported for selection", normalizedItem.ReplyWorkItemID, normalizedItem.State)
		}
		if !eligible {
			continue
		}
		selections = append(selections, CampaignZohoEmailReplyWorkSelection{
			WorkItem:     normalizedItem,
			InboundReply: reply,
			NeedsReopen:  needsReopen,
		})
	}
	if len(selections) == 0 {
		return CampaignZohoEmailReplyWorkSelection{}, false, nil
	}
	sort.SliceStable(selections, func(i, j int) bool {
		if !selections[i].InboundReply.ReceivedAt.Equal(selections[j].InboundReply.ReceivedAt) {
			return selections[i].InboundReply.ReceivedAt.Before(selections[j].InboundReply.ReceivedAt)
		}
		return strings.TrimSpace(selections[i].InboundReply.ReplyID) < strings.TrimSpace(selections[j].InboundReply.ReplyID)
	})
	return selections[0], true, nil
}

func LoadCommittedCampaignZohoEmailReplyWorkSelection(root, campaignID string, now time.Time) (CampaignZohoEmailReplyWorkSelection, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return CampaignZohoEmailReplyWorkSelection{}, false, err
	}
	workItemRecords, err := ListCommittedAllCampaignZohoEmailReplyWorkItemRecords(root)
	if err != nil {
		return CampaignZohoEmailReplyWorkSelection{}, false, err
	}
	inboundReplyRecords, err := ListCommittedAllFrankZohoInboundReplyRecords(root)
	if err != nil {
		return CampaignZohoEmailReplyWorkSelection{}, false, err
	}
	return DeriveCampaignZohoEmailReplyWorkSelection(campaignID, workItemRecords, inboundReplyRecords, now)
}
