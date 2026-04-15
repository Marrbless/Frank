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
}
