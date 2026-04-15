package tools

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestManageFrankZohoCampaignReplyWorkItemDefersCommittedItem(t *testing.T) {
	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	now := time.Now().UTC().Add(-5 * time.Minute).Truncate(time.Second)

	reply := missioncontrol.FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880199000",
		FromAddress:        "person@example.com",
		FromAddressCount:   1,
		ReceivedAt:         now,
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880199000/originalmessage",
	}
	replyRecord := persistFrankZohoCampaignRuntime(t, root, "job-frank-zoho-manage-history", campaign.CampaignID, nil, []missioncontrol.FrankZohoInboundReply{reply}, now)[0]
	openItem, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemOpen(campaign.CampaignID, replyRecord.ReplyID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-follow-up-test"}, job, runtime, control, time.Now().UTC())
	})
	job := missioncontrol.Job{
		ID:           "job-frank-zoho-manage",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{FrankZohoManageReplyWorkItemToolName},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-manage",
			Steps: []missioncontrol.Step{
				{
					ID:                "send-outbound-email",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{FrankZohoManageReplyWorkItemToolName},
					CampaignRef:       &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID},
				},
				{
					ID:                "final-response",
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	if err := state.ActivateStep(job, "send-outbound-email"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want active runtime")
	}
	runtime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, openItem)
	if err != nil {
		t.Fatalf("UpsertCampaignZohoEmailReplyWorkItem() error = %v", err)
	}
	if !changed {
		t.Fatal("UpsertCampaignZohoEmailReplyWorkItem() changed = false, want seeded open item")
	}
	state.SetExecutionContext(missioncontrol.ExecutionContext{
		Job:              &job,
		Step:             &job.Plan.Steps[0],
		Runtime:          &runtime,
		MissionStoreRoot: root,
	})

	deferUntil := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	result, skip, err := state.ManageFrankZohoCampaignReplyWorkItem(map[string]interface{}{
		"inbound_reply_id": replyRecord.ReplyID,
		"action":           "defer",
		"defer_until":      deferUntil,
	})
	if err != nil {
		t.Fatalf("ManageFrankZohoCampaignReplyWorkItem() error = %v", err)
	}
	if !skip {
		t.Fatal("ManageFrankZohoCampaignReplyWorkItem() skip = false, want taskstate-owned mutation")
	}
	var payload struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("json.Unmarshal(result) error = %v", err)
	}
	if payload.State != "deferred" {
		t.Fatalf("payload.State = %q, want deferred", payload.State)
	}
}

func TestManageFrankZohoCampaignReplyWorkItemIgnoresCommittedItem(t *testing.T) {
	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	now := time.Now().UTC().Add(-5 * time.Minute).Truncate(time.Second)

	reply := missioncontrol.FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880199001",
		FromAddress:        "person@example.com",
		FromAddressCount:   1,
		ReceivedAt:         now,
		OriginalMessageURL: "https://mail.zoho.test/api/accounts/3323462000000008002/messages/1711540357880199001/originalmessage",
	}
	replyRecord := persistFrankZohoCampaignRuntime(t, root, "job-frank-zoho-manage-ignore-history", campaign.CampaignID, nil, []missioncontrol.FrankZohoInboundReply{reply}, now)[0]
	openItem, err := missioncontrol.BuildCampaignZohoEmailReplyWorkItemOpen(campaign.CampaignID, replyRecord.ReplyID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetRuntimePersistHook(func(job *missioncontrol.Job, runtime missioncontrol.JobRuntimeState, control *missioncontrol.RuntimeControlContext) error {
		return missioncontrol.PersistProjectedRuntimeState(root, missioncontrol.WriterLockLease{LeaseHolderID: "frank-zoho-follow-up-test"}, job, runtime, control, time.Now().UTC())
	})
	job := missioncontrol.Job{
		ID:           "job-frank-zoho-manage-ignore",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{FrankZohoManageReplyWorkItemToolName},
		Plan: missioncontrol.Plan{
			ID: "plan-frank-zoho-manage-ignore",
			Steps: []missioncontrol.Step{
				{
					ID:                "send-outbound-email",
					Type:              missioncontrol.StepTypeDiscussion,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
					AllowedTools:      []string{FrankZohoManageReplyWorkItemToolName},
					CampaignRef:       &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID},
				},
				{
					ID:                "final-response",
					Type:              missioncontrol.StepTypeFinalResponse,
					RequiredAuthority: missioncontrol.AuthorityTierLow,
				},
			},
		},
	}
	if err := state.ActivateStep(job, "send-outbound-email"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want active runtime")
	}
	runtime, changed, err := missioncontrol.UpsertCampaignZohoEmailReplyWorkItem(runtime, openItem)
	if err != nil {
		t.Fatalf("UpsertCampaignZohoEmailReplyWorkItem() error = %v", err)
	}
	if !changed {
		t.Fatal("UpsertCampaignZohoEmailReplyWorkItem() changed = false, want seeded open item")
	}
	state.SetExecutionContext(missioncontrol.ExecutionContext{
		Job:              &job,
		Step:             &job.Plan.Steps[0],
		Runtime:          &runtime,
		MissionStoreRoot: root,
	})

	result, skip, err := state.ManageFrankZohoCampaignReplyWorkItem(map[string]interface{}{
		"inbound_reply_id": replyRecord.ReplyID,
		"action":           "ignore",
	})
	if err != nil {
		t.Fatalf("ManageFrankZohoCampaignReplyWorkItem() error = %v", err)
	}
	if !skip {
		t.Fatal("ManageFrankZohoCampaignReplyWorkItem() skip = false, want taskstate-owned mutation")
	}
	var payload struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("json.Unmarshal(result) error = %v", err)
	}
	if payload.State != "ignored" {
		t.Fatalf("payload.State = %q, want ignored", payload.State)
	}

	workItems, err := missioncontrol.ListCommittedAllCampaignZohoEmailReplyWorkItemRecords(root)
	if err != nil {
		t.Fatalf("ListCommittedAllCampaignZohoEmailReplyWorkItemRecords() error = %v", err)
	}
	if len(workItems) != 1 {
		t.Fatalf("ListCommittedAllCampaignZohoEmailReplyWorkItemRecords() len = %d, want 1", len(workItems))
	}
	if workItems[0].State != string(missioncontrol.CampaignZohoEmailReplyWorkItemStateIgnored) {
		t.Fatalf("workItems[0].State = %q, want ignored", workItems[0].State)
	}

	selection, ok, err := missioncontrol.LoadCommittedCampaignZohoEmailReplyWorkSelection(root, campaign.CampaignID, time.Now().UTC())
	if err != nil {
		t.Fatalf("LoadCommittedCampaignZohoEmailReplyWorkSelection() error = %v", err)
	}
	if ok {
		t.Fatalf("LoadCommittedCampaignZohoEmailReplyWorkSelection() = %#v, want ignored item excluded from selection", selection)
	}
}
