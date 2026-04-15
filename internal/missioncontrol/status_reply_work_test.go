package missioncontrol

import (
	"testing"
	"time"
)

func TestBuildOperatorStatusSummaryIncludesCampaignZohoEmailReplyWorkItems(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	item, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}
	item, err = BuildCampaignZohoEmailReplyWorkItemDeferred(item, now.Add(time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemDeferred() error = %v", err)
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:                           "job-1",
		State:                           JobStateRunning,
		CampaignZohoEmailReplyWorkItems: []CampaignZohoEmailReplyWorkItem{item},
	})
	if len(summary.CampaignZohoEmailReplyWork) != 1 {
		t.Fatalf("CampaignZohoEmailReplyWork len = %d, want 1", len(summary.CampaignZohoEmailReplyWork))
	}
	if summary.CampaignZohoEmailReplyWork[0].State != "deferred" {
		t.Fatalf("CampaignZohoEmailReplyWork[0].State = %q, want deferred", summary.CampaignZohoEmailReplyWork[0].State)
	}
	if summary.CampaignZohoEmailReplyWork[0].DerivedIterationState != "deferred" {
		t.Fatalf("CampaignZohoEmailReplyWork[0].DerivedIterationState = %q, want deferred", summary.CampaignZohoEmailReplyWork[0].DerivedIterationState)
	}
}

func TestBuildOperatorStatusSummaryClassifiesBlockedTerminalFailureReplyWork(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	item, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}
	item, err = BuildCampaignZohoEmailReplyWorkItemClaimed(item, "campaign_zoho_email_outbound_failed", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemClaimed() error = %v", err)
	}

	followUp, err := BuildCampaignZohoEmailOutboundPreparedReplyAction(
		"send-outbound-email",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{To: []string{"person@example.com"}},
		"Re: Frank intro",
		"plaintext",
		"Thanks for writing back.",
		now.Add(2*time.Minute),
		item.InboundReplyID,
		"campaign_zoho_email_outbound_parent",
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedReplyAction() error = %v", err)
	}
	followUp, err = BuildCampaignZohoEmailOutboundFailedAction(followUp, CampaignZohoEmailOutboundFailure{
		HTTPStatus:                400,
		ProviderStatusCode:        1510,
		ProviderStatusDescription: "recipient rejected",
	}, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundFailedAction() error = %v", err)
	}
	item.ClaimedFollowUpActionID = followUp.ActionID

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:                            "job-1",
		State:                            JobStateRunning,
		CampaignZohoEmailReplyWorkItems:  []CampaignZohoEmailReplyWorkItem{item},
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{followUp},
	})
	if len(summary.CampaignZohoEmailReplyWork) != 1 {
		t.Fatalf("CampaignZohoEmailReplyWork len = %d, want 1", len(summary.CampaignZohoEmailReplyWork))
	}
	got := summary.CampaignZohoEmailReplyWork[0]
	if got.DerivedIterationState != "blocked_terminal_failure" {
		t.Fatalf("DerivedIterationState = %q, want blocked_terminal_failure", got.DerivedIterationState)
	}
	if got.LatestFollowUpActionID != followUp.ActionID {
		t.Fatalf("LatestFollowUpActionID = %q, want %q", got.LatestFollowUpActionID, followUp.ActionID)
	}
	if got.LatestFollowUpActionState != string(CampaignZohoEmailOutboundActionStateFailed) {
		t.Fatalf("LatestFollowUpActionState = %q, want failed", got.LatestFollowUpActionState)
	}
}

func TestBuildOperatorStatusSummaryClassifiesReopenedTerminalFailureReplyWork(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	item, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	followUp, err := BuildCampaignZohoEmailOutboundPreparedReplyAction(
		"send-outbound-email",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{To: []string{"person@example.com"}},
		"Re: Frank intro",
		"plaintext",
		"Thanks for writing back.",
		now.Add(time.Minute),
		item.InboundReplyID,
		"campaign_zoho_email_outbound_parent",
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedReplyAction() error = %v", err)
	}
	followUp, err = BuildCampaignZohoEmailOutboundFailedAction(followUp, CampaignZohoEmailOutboundFailure{
		HTTPStatus:                400,
		ProviderStatusCode:        1510,
		ProviderStatusDescription: "recipient rejected",
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundFailedAction() error = %v", err)
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:                            "job-1",
		State:                            JobStateRunning,
		CampaignZohoEmailReplyWorkItems:  []CampaignZohoEmailReplyWorkItem{item},
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{followUp},
	})
	if len(summary.CampaignZohoEmailReplyWork) != 1 {
		t.Fatalf("CampaignZohoEmailReplyWork len = %d, want 1", len(summary.CampaignZohoEmailReplyWork))
	}
	got := summary.CampaignZohoEmailReplyWork[0]
	if got.DerivedIterationState != "reopened_after_terminal_failure" {
		t.Fatalf("DerivedIterationState = %q, want reopened_after_terminal_failure", got.DerivedIterationState)
	}
	if got.LatestFollowUpActionState != string(CampaignZohoEmailOutboundActionStateFailed) {
		t.Fatalf("LatestFollowUpActionState = %q, want failed", got.LatestFollowUpActionState)
	}
}
