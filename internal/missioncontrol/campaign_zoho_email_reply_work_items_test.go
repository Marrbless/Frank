package missioncontrol

import (
	"testing"
	"time"
)

func TestBuildCampaignZohoEmailReplyWorkItemOpenUsesDeterministicInboundKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	item, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	if item.ReplyWorkItemID == "" {
		t.Fatal("ReplyWorkItemID = empty, want deterministic committed key")
	}
	if item.State != CampaignZohoEmailReplyWorkItemStateOpen {
		t.Fatalf("State = %q, want open", item.State)
	}
	if item.InboundReplyID != "frank_zoho_inbound_reply_123" {
		t.Fatalf("InboundReplyID = %q, want stable inbound reply linkage", item.InboundReplyID)
	}
	if item.CreatedAt != now || item.UpdatedAt != now {
		t.Fatalf("timestamps = (%s, %s), want %s", item.CreatedAt, item.UpdatedAt, now)
	}
	if item.ReplyWorkItemID != normalizedCampaignZohoEmailReplyWorkItemID(item) {
		t.Fatalf("ReplyWorkItemID = %q, want normalized inbound-keyed id", item.ReplyWorkItemID)
	}
}

func TestBuildCampaignZohoEmailReplyWorkItemClaimedRequiresFollowUpAction(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	item, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	if _, err := BuildCampaignZohoEmailReplyWorkItemClaimed(item, "", now.Add(time.Minute)); err == nil {
		t.Fatal("BuildCampaignZohoEmailReplyWorkItemClaimed() error = nil, want claimed follow-up action validation")
	}
}

func TestUpsertCampaignZohoEmailReplyWorkItemReplacesByDeterministicKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	openItem, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}
	runtime, changed, err := UpsertCampaignZohoEmailReplyWorkItem(JobRuntimeState{}, openItem)
	if err != nil {
		t.Fatalf("UpsertCampaignZohoEmailReplyWorkItem(open) error = %v", err)
	}
	if !changed {
		t.Fatal("UpsertCampaignZohoEmailReplyWorkItem(open) changed = false, want insert")
	}

	claimedItem, err := BuildCampaignZohoEmailReplyWorkItemClaimed(openItem, "campaign_zoho_email_outbound_456", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemClaimed() error = %v", err)
	}
	runtime, changed, err = UpsertCampaignZohoEmailReplyWorkItem(runtime, claimedItem)
	if err != nil {
		t.Fatalf("UpsertCampaignZohoEmailReplyWorkItem(claimed) error = %v", err)
	}
	if !changed {
		t.Fatal("UpsertCampaignZohoEmailReplyWorkItem(claimed) changed = false, want state replacement")
	}
	if len(runtime.CampaignZohoEmailReplyWorkItems) != 1 {
		t.Fatalf("len(CampaignZohoEmailReplyWorkItems) = %d, want 1 stable work item", len(runtime.CampaignZohoEmailReplyWorkItems))
	}
	got := runtime.CampaignZohoEmailReplyWorkItems[0]
	if got.State != CampaignZohoEmailReplyWorkItemStateClaimed {
		t.Fatalf("State = %q, want claimed", got.State)
	}
	if got.ClaimedFollowUpActionID != "campaign_zoho_email_outbound_456" {
		t.Fatalf("ClaimedFollowUpActionID = %q, want claimed outbound linkage", got.ClaimedFollowUpActionID)
	}
	if got.ReplyWorkItemID != openItem.ReplyWorkItemID {
		t.Fatalf("ReplyWorkItemID = %q, want stable inbound-keyed id %q", got.ReplyWorkItemID, openItem.ReplyWorkItemID)
	}
}
