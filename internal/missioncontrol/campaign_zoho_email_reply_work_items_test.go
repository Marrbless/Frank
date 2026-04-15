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

func TestDeriveMissingCampaignZohoEmailReplyWorkItemsCreatesOnlyUniquelyAttributedReplies(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	outboundOne := testCampaignZohoEmailOutboundActionRecord("job-1", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-1", "campaign-mail", "Subject 1", now.Add(-2*time.Minute)))
	outboundTwo := testCampaignZohoEmailOutboundActionRecord("job-2", 1, mustBuildVerifiedCampaignZohoEmailOutboundAction(t, "step-2", "campaign-other", "Subject 2", now.Add(-time.Minute)))

	items, err := DeriveMissingCampaignZohoEmailReplyWorkItems("campaign-mail",
		[]CampaignZohoEmailOutboundActionRecord{outboundOne, outboundTwo},
		[]FrankZohoInboundReplyRecord{
			testFrankZohoInboundReplyRecord("job-r1", 1, FrankZohoInboundReply{
				StepID:             "sync-replies",
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				ProviderMessageID:  "reply-1",
				ProviderMailID:     "<reply-1@zoho.test>",
				MIMEMessageID:      "<reply-1@example.test>",
				InReplyTo:          outboundOne.MIMEMessageID,
				References:         []string{"<seed@example.test>", outboundOne.MIMEMessageID},
				FromAddress:        "person@example.com",
				FromDisplayName:    "Person One",
				FromAddressCount:   1,
				Subject:            "Re: Subject 1",
				ReceivedAt:         now.Add(-30 * time.Second),
				OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/reply-1/originalmessage",
			}),
			testFrankZohoInboundReplyRecord("job-r2", 1, FrankZohoInboundReply{
				StepID:             "sync-replies",
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				ProviderMessageID:  "reply-2",
				ProviderMailID:     "<reply-2@zoho.test>",
				MIMEMessageID:      "<reply-2@example.test>",
				InReplyTo:          "<unknown@example.test>",
				References:         []string{"<unknown@example.test>"},
				FromAddress:        "other@example.com",
				FromDisplayName:    "Other Person",
				FromAddressCount:   1,
				Subject:            "Re: Unknown",
				ReceivedAt:         now.Add(-20 * time.Second),
				OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/reply-2/originalmessage",
			}),
		},
		nil,
		now,
	)
	if err != nil {
		t.Fatalf("DeriveMissingCampaignZohoEmailReplyWorkItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("DeriveMissingCampaignZohoEmailReplyWorkItems() len = %d, want 1 uniquely attributable item", len(items))
	}
	if items[0].CampaignID != "campaign-mail" {
		t.Fatalf("CampaignID = %q, want campaign-mail", items[0].CampaignID)
	}
	if items[0].InboundReplyID == "" {
		t.Fatal("InboundReplyID = empty, want committed reply linkage")
	}
}

func TestDeriveCampaignZohoEmailReplyWorkSelectionChoosesOldestEligibleReply(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	oldestReply := testFrankZohoInboundReplyRecord("job-r1", 1, FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "reply-1",
		ReceivedAt:         now.Add(-2 * time.Hour),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/reply-1/originalmessage",
	})
	newerDeferredReply := testFrankZohoInboundReplyRecord("job-r2", 1, FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "reply-2",
		ReceivedAt:         now.Add(-time.Hour),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/reply-2/originalmessage",
	})

	selection, ok, err := DeriveCampaignZohoEmailReplyWorkSelection("campaign-mail",
		[]CampaignZohoEmailReplyWorkItemRecord{
			{
				RecordVersion:   StoreRecordVersion,
				LastSeq:         1,
				ReplyWorkItemID: normalizedCampaignZohoEmailReplyWorkItemID(CampaignZohoEmailReplyWorkItem{InboundReplyID: oldestReply.ReplyID}),
				JobID:           "job-1",
				StepID:          "send-outbound-email",
				InboundReplyID:  oldestReply.ReplyID,
				CampaignID:      "campaign-mail",
				State:           string(CampaignZohoEmailReplyWorkItemStateOpen),
				CreatedAt:       now.Add(-90 * time.Minute),
				UpdatedAt:       now.Add(-90 * time.Minute),
			},
			{
				RecordVersion:   StoreRecordVersion,
				LastSeq:         1,
				ReplyWorkItemID: normalizedCampaignZohoEmailReplyWorkItemID(CampaignZohoEmailReplyWorkItem{InboundReplyID: newerDeferredReply.ReplyID}),
				JobID:           "job-2",
				StepID:          "send-outbound-email",
				InboundReplyID:  newerDeferredReply.ReplyID,
				CampaignID:      "campaign-mail",
				State:           string(CampaignZohoEmailReplyWorkItemStateDeferred),
				DeferredUntil:   now.Add(-time.Minute),
				CreatedAt:       now.Add(-30 * time.Minute),
				UpdatedAt:       now.Add(-30 * time.Minute),
			},
		},
		[]FrankZohoInboundReplyRecord{oldestReply, newerDeferredReply},
		now,
	)
	if err != nil {
		t.Fatalf("DeriveCampaignZohoEmailReplyWorkSelection() error = %v", err)
	}
	if !ok {
		t.Fatal("DeriveCampaignZohoEmailReplyWorkSelection() ok = false, want oldest eligible reply")
	}
	if selection.InboundReply.ReplyID != oldestReply.ReplyID {
		t.Fatalf("InboundReply.ReplyID = %q, want oldest eligible reply %q", selection.InboundReply.ReplyID, oldestReply.ReplyID)
	}
	if selection.NeedsReopen {
		t.Fatal("NeedsReopen = true, want oldest already-open reply to win before expired deferred reply")
	}
}
