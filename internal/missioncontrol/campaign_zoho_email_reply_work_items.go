package missioncontrol

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type CampaignZohoEmailReplyWorkItemState string

const (
	CampaignZohoEmailReplyWorkItemStateOpen      CampaignZohoEmailReplyWorkItemState = "open"
	CampaignZohoEmailReplyWorkItemStateDeferred  CampaignZohoEmailReplyWorkItemState = "deferred"
	CampaignZohoEmailReplyWorkItemStateClaimed   CampaignZohoEmailReplyWorkItemState = "claimed"
	CampaignZohoEmailReplyWorkItemStateResponded CampaignZohoEmailReplyWorkItemState = "responded"
	CampaignZohoEmailReplyWorkItemStateIgnored   CampaignZohoEmailReplyWorkItemState = "ignored"
)

type CampaignZohoEmailReplyWorkItem struct {
	ReplyWorkItemID         string                              `json:"reply_work_item_id"`
	InboundReplyID          string                              `json:"inbound_reply_id"`
	CampaignID              string                              `json:"campaign_id"`
	State                   CampaignZohoEmailReplyWorkItemState `json:"state"`
	DeferredUntil           time.Time                           `json:"deferred_until,omitempty"`
	ClaimedFollowUpActionID string                              `json:"claimed_followup_action_id,omitempty"`
	CreatedAt               time.Time                           `json:"created_at"`
	UpdatedAt               time.Time                           `json:"updated_at"`
}

func NormalizeCampaignZohoEmailReplyWorkItem(item CampaignZohoEmailReplyWorkItem) CampaignZohoEmailReplyWorkItem {
	item.ReplyWorkItemID = strings.TrimSpace(item.ReplyWorkItemID)
	item.InboundReplyID = strings.TrimSpace(item.InboundReplyID)
	item.CampaignID = strings.TrimSpace(item.CampaignID)
	item.State = CampaignZohoEmailReplyWorkItemState(strings.TrimSpace(string(item.State)))
	item.DeferredUntil = item.DeferredUntil.UTC()
	item.ClaimedFollowUpActionID = strings.TrimSpace(item.ClaimedFollowUpActionID)
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item
}

func ValidateCampaignZohoEmailReplyWorkItem(item CampaignZohoEmailReplyWorkItem) error {
	normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
	if normalized.ReplyWorkItemID == "" {
		return fmt.Errorf("mission runtime campaign zoho email reply work item reply_work_item_id is required")
	}
	if normalized.InboundReplyID == "" {
		return fmt.Errorf("mission runtime campaign zoho email reply work item inbound_reply_id is required")
	}
	if err := validateCampaignID(normalized.CampaignID, "mission runtime campaign zoho email reply work item"); err != nil {
		return err
	}
	switch normalized.State {
	case CampaignZohoEmailReplyWorkItemStateOpen, CampaignZohoEmailReplyWorkItemStateDeferred, CampaignZohoEmailReplyWorkItemStateClaimed, CampaignZohoEmailReplyWorkItemStateResponded, CampaignZohoEmailReplyWorkItemStateIgnored:
	default:
		return fmt.Errorf("mission runtime campaign zoho email reply work item state %q is invalid", normalized.State)
	}
	if normalized.CreatedAt.IsZero() {
		return fmt.Errorf("mission runtime campaign zoho email reply work item created_at is required")
	}
	if normalized.UpdatedAt.IsZero() {
		return fmt.Errorf("mission runtime campaign zoho email reply work item updated_at is required")
	}
	if normalized.UpdatedAt.Before(normalized.CreatedAt) {
		return fmt.Errorf("mission runtime campaign zoho email reply work item updated_at must be on or after created_at")
	}
	switch normalized.State {
	case CampaignZohoEmailReplyWorkItemStateOpen:
		if !normalized.DeferredUntil.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email reply work item open state must not include deferred_until")
		}
		if normalized.ClaimedFollowUpActionID != "" {
			return fmt.Errorf("mission runtime campaign zoho email reply work item open state must not include claimed_followup_action_id")
		}
	case CampaignZohoEmailReplyWorkItemStateDeferred:
		if normalized.DeferredUntil.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email reply work item deferred state requires deferred_until")
		}
		if normalized.ClaimedFollowUpActionID != "" {
			return fmt.Errorf("mission runtime campaign zoho email reply work item deferred state must not include claimed_followup_action_id")
		}
	case CampaignZohoEmailReplyWorkItemStateClaimed:
		if normalized.DeferredUntil.IsZero() == false {
			return fmt.Errorf("mission runtime campaign zoho email reply work item claimed state must not include deferred_until")
		}
		if normalized.ClaimedFollowUpActionID == "" {
			return fmt.Errorf("mission runtime campaign zoho email reply work item claimed state requires claimed_followup_action_id")
		}
	case CampaignZohoEmailReplyWorkItemStateResponded, CampaignZohoEmailReplyWorkItemStateIgnored:
		if !normalized.DeferredUntil.IsZero() {
			return fmt.Errorf("mission runtime campaign zoho email reply work item terminal state must not include deferred_until")
		}
		if normalized.ClaimedFollowUpActionID != "" {
			return fmt.Errorf("mission runtime campaign zoho email reply work item terminal state must not include claimed_followup_action_id")
		}
	}
	if normalized.ReplyWorkItemID != normalizedCampaignZohoEmailReplyWorkItemID(normalized) {
		return fmt.Errorf("mission runtime campaign zoho email reply work item reply_work_item_id %q does not match normalized inbound linkage", normalized.ReplyWorkItemID)
	}
	return nil
}

func BuildCampaignZohoEmailReplyWorkItemOpen(campaignID, inboundReplyID string, now time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := CampaignZohoEmailReplyWorkItem{
		InboundReplyID: inboundReplyID,
		CampaignID:     campaignID,
		State:          CampaignZohoEmailReplyWorkItemStateOpen,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}
	item = NormalizeCampaignZohoEmailReplyWorkItem(item)
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func BuildCampaignZohoEmailReplyWorkItemDeferred(existing CampaignZohoEmailReplyWorkItem, deferredUntil, updatedAt time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := NormalizeCampaignZohoEmailReplyWorkItem(existing)
	item.State = CampaignZohoEmailReplyWorkItemStateDeferred
	item.DeferredUntil = deferredUntil.UTC()
	item.ClaimedFollowUpActionID = ""
	item.UpdatedAt = updatedAt.UTC()
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func BuildCampaignZohoEmailReplyWorkItemClaimed(existing CampaignZohoEmailReplyWorkItem, claimedFollowUpActionID string, updatedAt time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := NormalizeCampaignZohoEmailReplyWorkItem(existing)
	item.State = CampaignZohoEmailReplyWorkItemStateClaimed
	item.DeferredUntil = time.Time{}
	item.ClaimedFollowUpActionID = strings.TrimSpace(claimedFollowUpActionID)
	item.UpdatedAt = updatedAt.UTC()
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func BuildCampaignZohoEmailReplyWorkItemResponded(existing CampaignZohoEmailReplyWorkItem, updatedAt time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := NormalizeCampaignZohoEmailReplyWorkItem(existing)
	item.State = CampaignZohoEmailReplyWorkItemStateResponded
	item.DeferredUntil = time.Time{}
	item.ClaimedFollowUpActionID = ""
	item.UpdatedAt = updatedAt.UTC()
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func BuildCampaignZohoEmailReplyWorkItemIgnored(existing CampaignZohoEmailReplyWorkItem, updatedAt time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := NormalizeCampaignZohoEmailReplyWorkItem(existing)
	item.State = CampaignZohoEmailReplyWorkItemStateIgnored
	item.DeferredUntil = time.Time{}
	item.ClaimedFollowUpActionID = ""
	item.UpdatedAt = updatedAt.UTC()
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func BuildCampaignZohoEmailReplyWorkItemReopened(existing CampaignZohoEmailReplyWorkItem, updatedAt time.Time) (CampaignZohoEmailReplyWorkItem, error) {
	item := NormalizeCampaignZohoEmailReplyWorkItem(existing)
	item.State = CampaignZohoEmailReplyWorkItemStateOpen
	item.DeferredUntil = time.Time{}
	item.ClaimedFollowUpActionID = ""
	item.UpdatedAt = updatedAt.UTC()
	item.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(item)
	if err := ValidateCampaignZohoEmailReplyWorkItem(item); err != nil {
		return CampaignZohoEmailReplyWorkItem{}, err
	}
	return item, nil
}

func UpsertCampaignZohoEmailReplyWorkItem(runtime JobRuntimeState, item CampaignZohoEmailReplyWorkItem) (JobRuntimeState, bool, error) {
	normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
	if normalized.ReplyWorkItemID == "" {
		normalized.ReplyWorkItemID = normalizedCampaignZohoEmailReplyWorkItemID(normalized)
	}
	if err := ValidateCampaignZohoEmailReplyWorkItem(normalized); err != nil {
		return JobRuntimeState{}, false, err
	}

	next := *CloneJobRuntimeState(&runtime)
	for i, existing := range next.CampaignZohoEmailReplyWorkItems {
		if existing.ReplyWorkItemID != normalized.ReplyWorkItemID {
			continue
		}
		if reflect.DeepEqual(NormalizeCampaignZohoEmailReplyWorkItem(existing), normalized) {
			return next, false, nil
		}
		next.CampaignZohoEmailReplyWorkItems[i] = normalized
		return next, true, nil
	}
	next.CampaignZohoEmailReplyWorkItems = append(next.CampaignZohoEmailReplyWorkItems, normalized)
	return next, true, nil
}

func FindCampaignZohoEmailReplyWorkItem(runtime JobRuntimeState, replyWorkItemID string) (CampaignZohoEmailReplyWorkItem, bool) {
	normalizedReplyWorkItemID := strings.TrimSpace(replyWorkItemID)
	if normalizedReplyWorkItemID == "" {
		return CampaignZohoEmailReplyWorkItem{}, false
	}
	for _, item := range runtime.CampaignZohoEmailReplyWorkItems {
		if item.ReplyWorkItemID == normalizedReplyWorkItemID {
			return NormalizeCampaignZohoEmailReplyWorkItem(item), true
		}
	}
	return CampaignZohoEmailReplyWorkItem{}, false
}

func FindCampaignZohoEmailReplyWorkItemByInboundReplyID(runtime JobRuntimeState, inboundReplyID string) (CampaignZohoEmailReplyWorkItem, bool) {
	normalizedInboundReplyID := strings.TrimSpace(inboundReplyID)
	if normalizedInboundReplyID == "" {
		return CampaignZohoEmailReplyWorkItem{}, false
	}
	for _, item := range runtime.CampaignZohoEmailReplyWorkItems {
		normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
		if normalized.InboundReplyID == normalizedInboundReplyID {
			return normalized, true
		}
	}
	return CampaignZohoEmailReplyWorkItem{}, false
}

func cloneCampaignZohoEmailReplyWorkItems(items []CampaignZohoEmailReplyWorkItem) []CampaignZohoEmailReplyWorkItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]CampaignZohoEmailReplyWorkItem, len(items))
	for i, item := range items {
		cloned[i] = NormalizeCampaignZohoEmailReplyWorkItem(item)
	}
	return cloned
}

func normalizedCampaignZohoEmailReplyWorkItemID(item CampaignZohoEmailReplyWorkItem) string {
	normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
	return "campaign_zoho_email_reply_work_" + projectedStoreHash(normalized.InboundReplyID)
}
