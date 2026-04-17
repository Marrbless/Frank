package tools

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func writeMalformedTreasuryRecordForTaskStateStatusTest(t *testing.T, root string, treasury missioncontrol.TreasuryRecord) {
	t.Helper()

	if err := missioncontrol.WriteStoreJSONAtomic(missioncontrol.StoreTreasuryPath(root, treasury.TreasuryID), map[string]interface{}{
		"record_version":   treasury.RecordVersion,
		"treasury_id":      treasury.TreasuryID,
		"display_name":     treasury.DisplayName,
		"state":            string(treasury.State),
		"zero_seed_policy": string(treasury.ZeroSeedPolicy),
		"container_refs": []map[string]interface{}{
			{
				"kind":      string(treasury.ContainerRefs[0].Kind),
				"object_id": treasury.ContainerRefs[0].ObjectID,
			},
		},
		"created_at": treasury.CreatedAt,
		"updated_at": treasury.UpdatedAt,
	}); err != nil {
		t.Fatalf("WriteStoreJSONAtomic() error = %v", err)
	}
}

func writeDeferredSchedulerTriggerForTaskStateStatusTest(t *testing.T, root string, filename string, payload map[string]any) {
	t.Helper()

	if err := missioncontrol.WriteStoreJSONAtomic(filepath.Join(root, "scheduler", "deferred_triggers", filename), payload); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(deferred trigger) error = %v", err)
	}
}

func TestTaskStateOperatorStatusActiveExecutionContextZeroTreasuryRefPathUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %#v, want %q", got.JobID, "job-1")
	}
	if got.State != missioncontrol.JobStateRunning {
		t.Fatalf("State = %#v, want %q", got.State, missioncontrol.JobStateRunning)
	}
	if got.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %#v, want %q", got.ActiveStepID, "build")
	}
	if !reflect.DeepEqual(got.AllowedTools, []string{"read"}) {
		t.Fatalf("AllowedTools = %#v, want [%q]", got.AllowedTools, "read")
	}
	if got.CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil for zero-ref path", got.CampaignPreflight)
	}
	if got.TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for zero-ref path", got.TreasuryPreflight)
	}

	runtime, ok := state.MissionRuntimeState()
	if !ok {
		t.Fatal("MissionRuntimeState() ok = false, want true")
	}
	if runtime.State != missioncontrol.JobStateRunning {
		t.Fatalf("MissionRuntimeState().State = %q, want unchanged %q", runtime.State, missioncontrol.JobStateRunning)
	}
}

func TestTaskStateOperatorStatusActiveExecutionContextSurfacesResolvedTreasuryPreflight(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on treasury-only path", got.CampaignPreflight)
	}
	if got.TreasuryPreflight == nil {
		t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
	}
	if got.TreasuryPreflight.Treasury == nil {
		t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
	}
	if got.TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
		t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", got.TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
	}
	if got.TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
		t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", got.TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
	}
	if !reflect.DeepEqual(got.TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
		t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", got.TreasuryPreflight.Containers, container)
	}
}

func TestTaskStateOperatorStatusActiveExecutionContextSurfacesResolvedCampaignPreflight(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.CampaignPreflight == nil || got.CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", got.CampaignPreflight)
	}
	if got.CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", got.CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
	}
	if len(got.CampaignPreflight.Identities) != 1 || len(got.CampaignPreflight.Accounts) != 1 || len(got.CampaignPreflight.Containers) != 1 {
		t.Fatalf("CampaignPreflight = %#v, want one identity/account/container", got.CampaignPreflight)
	}
	if got.TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on campaign-only path", got.TreasuryPreflight)
	}
}

func TestTaskStateOperatorStatusActiveExecutionContextSurfacesFrankZohoMailboxBootstrapPreflight(t *testing.T) {
	t.Parallel()

	root, identity, account := writeTaskStateZohoMailboxBootstrapFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].GovernedExternalTargets = []missioncontrol.AutonomyEligibilityTargetRef{
		{Kind: missioncontrol.EligibilityTargetKindProvider, RegistryID: "provider-mail"},
		{Kind: missioncontrol.EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
	}
	job.Plan.Steps[0].FrankObjectRefs = []missioncontrol.FrankRegistryObjectRef{
		{Kind: missioncontrol.FrankRegistryObjectKindIdentity, ObjectID: identity.IdentityID},
		{Kind: missioncontrol.FrankRegistryObjectKindAccount, ObjectID: account.AccountID},
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	envelope := mustTaskStateJSONObject(t, summary)
	assertTaskStateJSONObjectKeys(t, envelope, "active_step_id", "allowed_tools", "frank_zoho_mailbox_bootstrap_preflight", "job_id", "state")
	assertTaskStateResolvedFrankZohoMailboxBootstrapPreflightJSONEnvelope(t, envelope["frank_zoho_mailbox_bootstrap_preflight"])
	if got.CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on bootstrap-only path", got.CampaignPreflight)
	}
	if got.TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on bootstrap-only path", got.TreasuryPreflight)
	}
	if got.FrankZohoMailboxBootstrapPreflight == nil {
		t.Fatal("FrankZohoMailboxBootstrapPreflight = nil, want resolved bootstrap pair")
	}
	if got.FrankZohoMailboxBootstrapPreflight.Identity == nil || !reflect.DeepEqual(*got.FrankZohoMailboxBootstrapPreflight.Identity, identity) {
		t.Fatalf("FrankZohoMailboxBootstrapPreflight.Identity = %#v, want %#v", got.FrankZohoMailboxBootstrapPreflight.Identity, identity)
	}
	if got.FrankZohoMailboxBootstrapPreflight.Account == nil || !reflect.DeepEqual(*got.FrankZohoMailboxBootstrapPreflight.Account, account) {
		t.Fatalf("FrankZohoMailboxBootstrapPreflight.Account = %#v, want %#v", got.FrankZohoMailboxBootstrapPreflight.Account, account)
	}
}

func TestTaskStateOperatorStatusSurfacesCampaignZohoEmailAddressing(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreFrankZohoAddressedCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	envelope := mustTaskStateJSONObject(t, summary)
	preflight := envelope["campaign_preflight"].(map[string]any)
	campaignJSON := preflight["campaign"].(map[string]any)
	addressingJSON, ok := campaignJSON["zoho_email_addressing"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing = %#v, want object", campaignJSON["zoho_email_addressing"])
	}
	assertTaskStateJSONObjectKeys(t, campaignJSON, "campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at", "zoho_email_addressing")
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["to"], "campaign_preflight.campaign.zoho_email_addressing.to"), []any{"person@example.com", "team@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.to = %#v, want [person@example.com team@example.com]", addressingJSON["to"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["cc"], "campaign_preflight.campaign.zoho_email_addressing.cc"), []any{"copy@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.cc = %#v, want [copy@example.com]", addressingJSON["cc"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["bcc"], "campaign_preflight.campaign.zoho_email_addressing.bcc"), []any{"blind@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.bcc = %#v, want [blind@example.com]", addressingJSON["bcc"])
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.CampaignPreflight == nil || got.CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", got.CampaignPreflight)
	}
	addressing := got.CampaignPreflight.Campaign.ZohoEmailAddressing
	if addressing == nil {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing = nil, want campaign-owned Zoho addressing")
	}
	if !reflect.DeepEqual(addressing.To, []string{"person@example.com", "team@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.To = %#v, want [person@example.com team@example.com]", addressing.To)
	}
	if !reflect.DeepEqual(addressing.CC, []string{"copy@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.CC = %#v, want [copy@example.com]", addressing.CC)
	}
	if !reflect.DeepEqual(addressing.BCC, []string{"blind@example.com"}) {
		t.Fatalf("CampaignPreflight.Campaign.ZohoEmailAddressing.BCC = %#v, want [blind@example.com]", addressing.BCC)
	}
}

func TestTaskStateOperatorStatusSurfacesCampaignZohoEmailSendGateOnActivePath(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	state.SetExecutionContext(missioncontrol.ExecutionContext{
		Job:     &job,
		Step:    &job.Plan.Steps[0],
		Runtime: missioncontrol.CloneJobRuntimeState(&runtime),
	})

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	envelope := mustTaskStateJSONObject(t, summary)
	gateJSON, ok := envelope["campaign_zoho_email_send_gate"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want object", envelope["campaign_zoho_email_send_gate"])
	}
	assertTaskStateJSONObjectKeys(t, gateJSON, "allowed", "ambiguous_outcome_count", "attributed_reply_count", "campaign_id", "failure_count", "failure_threshold_limit", "failure_threshold_metric", "halted", "verified_success_count")
	if gateJSON["campaign_id"] != campaign.CampaignID {
		t.Fatalf("campaign_zoho_email_send_gate.campaign_id = %#v, want %q", gateJSON["campaign_id"], campaign.CampaignID)
	}
	if gateJSON["allowed"] != true || gateJSON["halted"] != false {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want allowed non-halted gate", gateJSON)
	}
	if gateJSON["failure_threshold_metric"] != "rejections" {
		t.Fatalf("campaign_zoho_email_send_gate.failure_threshold_metric = %#v, want rejections", gateJSON["failure_threshold_metric"])
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.CampaignZohoEmailSendGate == nil {
		t.Fatal("CampaignZohoEmailSendGate = nil, want active-path derived send gate")
	}
	if got.CampaignZohoEmailSendGate.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignZohoEmailSendGate.CampaignID = %q, want %q", got.CampaignZohoEmailSendGate.CampaignID, campaign.CampaignID)
	}
	if !got.CampaignZohoEmailSendGate.Allowed || got.CampaignZohoEmailSendGate.Halted {
		t.Fatalf("CampaignZohoEmailSendGate = %#v, want allowed non-halted gate", got.CampaignZohoEmailSendGate)
	}
}

func TestTaskStateOperatorStatusSurfacesCampaignZohoEmailSendGateOnPersistedPath(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	assertTaskStateReadoutAdapterBoundary(t, summary, false, false)

	envelope := mustTaskStateJSONObject(t, summary)
	assertTaskStateJSONObjectKeys(t, envelope, "active_step_id", "allowed_tools", "campaign_zoho_email_send_gate", "job_id", "state")
	gateJSON, ok := envelope["campaign_zoho_email_send_gate"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want object", envelope["campaign_zoho_email_send_gate"])
	}
	assertTaskStateJSONObjectKeys(t, gateJSON, "allowed", "ambiguous_outcome_count", "attributed_reply_count", "campaign_id", "failure_count", "failure_threshold_limit", "failure_threshold_metric", "halted", "verified_success_count")
	if gateJSON["campaign_id"] != campaign.CampaignID {
		t.Fatalf("campaign_zoho_email_send_gate.campaign_id = %#v, want %q", gateJSON["campaign_id"], campaign.CampaignID)
	}
	if gateJSON["allowed"] != true || gateJSON["halted"] != false {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want allowed non-halted gate", gateJSON)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if got.CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on persisted path", got.CampaignPreflight)
	}
	if got.CampaignZohoEmailSendGate == nil {
		t.Fatal("CampaignZohoEmailSendGate = nil, want persisted-path committed send gate")
	}
	if got.CampaignZohoEmailSendGate.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignZohoEmailSendGate.CampaignID = %q, want %q", got.CampaignZohoEmailSendGate.CampaignID, campaign.CampaignID)
	}
	if !got.CampaignZohoEmailSendGate.Allowed || got.CampaignZohoEmailSendGate.Halted {
		t.Fatalf("CampaignZohoEmailSendGate = %#v, want allowed non-halted gate", got.CampaignZohoEmailSendGate)
	}
}

func TestTaskStateOperatorStatusSurfacesUnsupportedCampaignZohoEmailStopConditionAsClosedGate(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	campaign.StopConditions = []string{"stop after 3 opens"}
	campaign.UpdatedAt = campaign.UpdatedAt.Add(time.Minute)
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}
	wantReason := `campaign zoho email stop_condition "stop after 3 opens" is not evaluable from committed outbound and inbound reply records`

	t.Run("active", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		state.SetExecutionContext(missioncontrol.ExecutionContext{
			Job:     &job,
			Step:    &job.Plan.Steps[0],
			Runtime: missioncontrol.CloneJobRuntimeState(&runtime),
		})

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		if got.CampaignZohoEmailSendGate == nil {
			t.Fatal("CampaignZohoEmailSendGate = nil, want closed gate")
		}
		if got.CampaignZohoEmailSendGate.Allowed {
			t.Fatalf("CampaignZohoEmailSendGate.Allowed = true, want closed gate: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Halted {
			t.Fatalf("CampaignZohoEmailSendGate.Halted = true, want fail-closed unsupported gate without triggered halt: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Reason != wantReason {
			t.Fatalf("CampaignZohoEmailSendGate.Reason = %q, want %q", got.CampaignZohoEmailSendGate.Reason, wantReason)
		}
	})

	t.Run("persisted", func(t *testing.T) {
		t.Parallel()

		control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
		if err != nil {
			t.Fatalf("BuildRuntimeControlContext() error = %v", err)
		}

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
			t.Fatalf("HydrateRuntimeControl() error = %v", err)
		}
		state.ClearExecutionContext()

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		if got.CampaignPreflight != nil {
			t.Fatalf("CampaignPreflight = %#v, want nil on persisted path", got.CampaignPreflight)
		}
		if got.CampaignZohoEmailSendGate == nil {
			t.Fatal("CampaignZohoEmailSendGate = nil, want closed gate")
		}
		if got.CampaignZohoEmailSendGate.Allowed {
			t.Fatalf("CampaignZohoEmailSendGate.Allowed = true, want closed gate: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Halted {
			t.Fatalf("CampaignZohoEmailSendGate.Halted = true, want fail-closed unsupported gate without triggered halt: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Reason != wantReason {
			t.Fatalf("CampaignZohoEmailSendGate.Reason = %q, want %q", got.CampaignZohoEmailSendGate.Reason, wantReason)
		}
	})
}

func TestTaskStateOperatorStatusSurfacesUnsupportedCampaignZohoEmailFailureThresholdMetricAsClosedGate(t *testing.T) {
	t.Parallel()

	root, _, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	campaign.StopConditions = []string{"stop after 3 verified sends"}
	campaign.FailureThreshold = missioncontrol.CampaignFailureThreshold{Metric: "bounced_messages", Limit: 3}
	campaign.UpdatedAt = campaign.UpdatedAt.Add(time.Minute)
	if err := missioncontrol.StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
	}
	wantReason := `campaign zoho email failure_threshold.metric "bounced_messages" is not evaluable from committed outbound action records`

	t.Run("active", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		state.SetExecutionContext(missioncontrol.ExecutionContext{
			Job:     &job,
			Step:    &job.Plan.Steps[0],
			Runtime: missioncontrol.CloneJobRuntimeState(&runtime),
		})

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		if got.CampaignZohoEmailSendGate == nil {
			t.Fatal("CampaignZohoEmailSendGate = nil, want closed gate")
		}
		if got.CampaignZohoEmailSendGate.Allowed {
			t.Fatalf("CampaignZohoEmailSendGate.Allowed = true, want closed gate: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Halted {
			t.Fatalf("CampaignZohoEmailSendGate.Halted = true, want fail-closed unsupported gate without triggered halt: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Reason != wantReason {
			t.Fatalf("CampaignZohoEmailSendGate.Reason = %q, want %q", got.CampaignZohoEmailSendGate.Reason, wantReason)
		}
	})

	t.Run("persisted", func(t *testing.T) {
		t.Parallel()

		control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
		if err != nil {
			t.Fatalf("BuildRuntimeControlContext() error = %v", err)
		}

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
			t.Fatalf("HydrateRuntimeControl() error = %v", err)
		}
		state.ClearExecutionContext()

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		if got.CampaignPreflight != nil {
			t.Fatalf("CampaignPreflight = %#v, want nil on persisted path", got.CampaignPreflight)
		}
		if got.CampaignZohoEmailSendGate == nil {
			t.Fatal("CampaignZohoEmailSendGate = nil, want closed gate")
		}
		if got.CampaignZohoEmailSendGate.Allowed {
			t.Fatalf("CampaignZohoEmailSendGate.Allowed = true, want closed gate: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Halted {
			t.Fatalf("CampaignZohoEmailSendGate.Halted = true, want fail-closed unsupported gate without triggered halt: %#v", got.CampaignZohoEmailSendGate)
		}
		if got.CampaignZohoEmailSendGate.Reason != wantReason {
			t.Fatalf("CampaignZohoEmailSendGate.Reason = %q, want %q", got.CampaignZohoEmailSendGate.Reason, wantReason)
		}
	})
}

func TestTaskStateOperatorStatusShowsDeferredSchedulerTriggersOnChosenReadoutPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	writeDeferredSchedulerTriggerForTaskStateStatusTest(t, root, "later.json", map[string]any{
		"record_version":   1,
		"trigger_id":       "scheduled-trigger-job-2-20260413T150000.000000000Z",
		"scheduler_job_id": "job-2",
		"name":             "stretch",
		"message":          "stand up and stretch",
		"fire_at":          "2026-04-13T15:00:00Z",
		"deferred_at":      "2026-04-13T15:01:00Z",
	})
	writeDeferredSchedulerTriggerForTaskStateStatusTest(t, root, "earlier.json", map[string]any{
		"record_version":   1,
		"trigger_id":       "scheduled-trigger-job-1-20260413T140000.000000000Z",
		"scheduler_job_id": "job-1",
		"name":             "water",
		"message":          "drink water",
		"fire_at":          "2026-04-13T14:00:00Z",
		"deferred_at":      "2026-04-13T14:02:00Z",
	})

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
	if len(got.DeferredSchedulerTriggers) != 2 {
		t.Fatalf("DeferredSchedulerTriggers = %#v, want 2 queued deferred scheduler triggers", got.DeferredSchedulerTriggers)
	}
	if got.DeferredSchedulerTriggers[0].TriggerID != "scheduled-trigger-job-1-20260413T140000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[0] = %#v, want earliest fire first", got.DeferredSchedulerTriggers[0])
	}
	if got.DeferredSchedulerTriggers[0].Message != "drink water" {
		t.Fatalf("DeferredSchedulerTriggers[0].Message = %q, want %q", got.DeferredSchedulerTriggers[0].Message, "drink water")
	}
	if got.DeferredSchedulerTriggers[1].TriggerID != "scheduled-trigger-job-2-20260413T150000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[1] = %#v, want later fire second", got.DeferredSchedulerTriggers[1])
	}
}

func TestTaskStateOperatorStatusPreservesDeterministicDeferredSchedulerOrdering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	writeDeferredSchedulerTriggerForTaskStateStatusTest(t, root, "b.json", map[string]any{
		"record_version":   1,
		"trigger_id":       "scheduled-trigger-b-20260413T140000.000000000Z",
		"scheduler_job_id": "job-b",
		"name":             "later-name",
		"message":          "later message",
		"fire_at":          "2026-04-13T14:00:00Z",
		"deferred_at":      "2026-04-13T14:05:00Z",
	})
	writeDeferredSchedulerTriggerForTaskStateStatusTest(t, root, "a.json", map[string]any{
		"record_version":   1,
		"trigger_id":       "scheduled-trigger-a-20260413T140000.000000000Z",
		"scheduler_job_id": "job-a",
		"name":             "earlier-name",
		"message":          "earlier message",
		"fire_at":          "2026-04-13T14:00:00Z",
		"deferred_at":      "2026-04-13T14:01:00Z",
	})

	first, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus(first) error = %v", err)
	}
	second, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus(second) error = %v", err)
	}
	if first != second {
		t.Fatalf("OperatorStatus() differs across identical reads\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, first)
	if got.DeferredSchedulerTriggers[0].TriggerID != "scheduled-trigger-a-20260413T140000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[0] = %#v, want lexicographically first trigger ID tie-breaker", got.DeferredSchedulerTriggers[0])
	}
}

func TestTaskStateOperatorStatusActiveAndPersistedPathsPreserveAdapterBoundaryContract(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	campaign := mustStoreTaskStateCampaignFixture(t, root, container)
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: campaign.CampaignID}
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	t.Run("active", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		state.SetMissionStoreRoot(root)
		if err := state.ActivateStep(job, "build"); err != nil {
			t.Fatalf("ActivateStep() error = %v", err)
		}

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, summary, true, true)

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		envelope := mustTaskStateJSONObject(t, summary)
		assertTaskStateJSONObjectKeys(t, envelope, "active_step_id", "allowed_tools", "campaign_preflight", "campaign_zoho_email_send_gate", "job_id", "state", "treasury_preflight")
		assertTaskStateResolvedCampaignPreflightJSONEnvelope(t, envelope["campaign_preflight"])
		if _, ok := envelope["campaign_zoho_email_send_gate"].(map[string]any); !ok {
			t.Fatalf("campaign_zoho_email_send_gate = %#v, want object", envelope["campaign_zoho_email_send_gate"])
		}
		assertTaskStateResolvedTreasuryPreflightJSONEnvelope(t, envelope["treasury_preflight"])
		if got.CampaignPreflight == nil || got.CampaignPreflight.Campaign == nil {
			t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight on active path", got.CampaignPreflight)
		}
		if got.CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
			t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", got.CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
		}
		if got.TreasuryPreflight == nil || got.TreasuryPreflight.Treasury == nil {
			t.Fatalf("TreasuryPreflight = %#v, want resolved treasury preflight on active path", got.TreasuryPreflight)
		}
		if got.TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
			t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", got.TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
		}
		if got.TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
			t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", got.TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
		}
		if !reflect.DeepEqual(got.TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
			t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", got.TreasuryPreflight.Containers, container)
		}
	})

	t.Run("persisted", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		runtime := missioncontrol.JobRuntimeState{
			JobID:        "job-1",
			State:        missioncontrol.JobStatePaused,
			ActiveStepID: "build",
			PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		}
		control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
		if err != nil {
			t.Fatalf("BuildRuntimeControlContext() error = %v", err)
		}
		if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
			t.Fatalf("HydrateRuntimeControl() error = %v", err)
		}
		state.ClearExecutionContext()

		summary, err := state.OperatorStatus("job-1")
		if err != nil {
			t.Fatalf("OperatorStatus() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, summary, false, false)

		got := mustTaskStateReadoutJSON[missioncontrol.OperatorStatusSummary](t, summary)
		envelope := mustTaskStateJSONObject(t, summary)
		assertTaskStateJSONObjectKeys(t, envelope, "active_step_id", "allowed_tools", "job_id", "paused_reason", "state")
		if got.CampaignPreflight != nil {
			t.Fatalf("CampaignPreflight = %#v, want nil for persisted runtime path", got.CampaignPreflight)
		}
		if got.TreasuryPreflight != nil {
			t.Fatalf("TreasuryPreflight = %#v, want nil for persisted runtime path", got.TreasuryPreflight)
		}
	})
}

func TestTaskStateActivateStepMissingCampaignFailsClosedForStatusPath(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0].CampaignRef = &missioncontrol.CampaignRef{CampaignID: "campaign-missing"}
	state.SetMissionStoreRoot(t.TempDir())
	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want missing campaign rejection")
	}
	if !strings.Contains(err.Error(), missioncontrol.ErrCampaignRecordNotFound.Error()) {
		t.Fatalf("ActivateStep() error = %q, want missing campaign rejection", err)
	}
}

func TestTaskStateActivateStepInvalidTreasuryStateFailsClosedForStatusPath(t *testing.T) {
	t.Parallel()

	root, treasury, _ := writeTaskStateTreasuryFixtures(t)
	treasury.State = missioncontrol.TreasuryStateBootstrap
	treasury.BootstrapAcquisition = nil
	if err := missioncontrol.StoreTreasuryRecord(root, treasury); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	err := state.ActivateStep(job, "build")
	if err == nil {
		t.Fatal("ActivateStep() error = nil, want fail-closed missing bootstrap acquisition rejection")
	}
	if !strings.Contains(err.Error(), `execution context treasury "treasury-wallet" requires committed treasury.bootstrap_acquisition for first-value acquisition`) {
		t.Fatalf("ActivateStep() error = %q, want missing bootstrap acquisition rejection", err)
	}
}

func TestTaskStateOperatorStatusIncludesRecentAuditForActiveRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}
	state.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "write_memory",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})
	state.EmitAuditEvent(missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   false,
		Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})
	expected := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   false,
		Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})[0]

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 2 {
		t.Fatalf("recent_audit = %#v, want two audit entries", got["recent_audit"])
	}
	first, ok := recentAudit[0].(map[string]any)
	if !ok {
		t.Fatalf("recent_audit[0] = %#v, want object", recentAudit[0])
	}
	if first["action"] != "pause" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", first["action"], "pause")
	}
	if first["event_id"] != expected.EventID {
		t.Fatalf("recent_audit[0].event_id = %#v, want %q", first["event_id"], expected.EventID)
	}
	if first["action_class"] != string(expected.ActionClass) {
		t.Fatalf("recent_audit[0].action_class = %#v, want %q", first["action_class"], expected.ActionClass)
	}
	if first["result"] != string(expected.Result) {
		t.Fatalf("recent_audit[0].result = %#v, want %q", first["result"], expected.Result)
	}
	if first["error_code"] != "E_STEP_OUT_OF_ORDER" {
		t.Fatalf("recent_audit[0].error_code = %#v, want %q", first["error_code"], "E_STEP_OUT_OF_ORDER")
	}
	if first["timestamp"] != "2026-03-24T12:01:00Z" {
		t.Fatalf("recent_audit[0].timestamp = %#v, want %q", first["timestamp"], "2026-03-24T12:01:00Z")
	}
}

func TestTaskStateOperatorStatusReturnsApprovalSummaryForPersistedWaitingRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := missioncontrol.Job{
		ID:           "job-1",
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{
					ID:      "build",
					Type:    missioncontrol.StepTypeDiscussion,
					Subtype: missioncontrol.StepSubtypeAuthorization,
				},
				{
					ID:        "final",
					Type:      missioncontrol.StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:         "job-1",
		State:         missioncontrol.JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_authorization",
		WaitingAt:     time.Date(2026, 3, 24, 12, 2, 30, 0, time.UTC),
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           missioncontrol.ApprovalStatePending,
				RequestedAt:     time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC),
				ExpiresAt:       time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC),
				Content: &missioncontrol.ApprovalRequestContent{
					ProposedAction:   "Continue build.",
					WhyNeeded:        "Operator approval is required.",
					AuthorityTier:    missioncontrol.AuthorityTierMedium,
					FallbackIfDenied: "Stay waiting.",
				},
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	for _, want := range []string{
		`"state": "waiting_user"`,
		`"waiting_reason": "discussion_authorization"`,
		`"waiting_at": "2026-03-24T12:02:30Z"`,
		`"requested_action": "step_complete"`,
		`"scope": "mission_step"`,
		`"requested_via": "runtime_waiting_user"`,
		`"session_channel": "telegram"`,
		`"session_chat_id": "chat-42"`,
		`"proposed_action": "Continue build."`,
		`"why_needed": "Operator approval is required."`,
		`"authority_tier": "medium"`,
		`"fallback_if_denied": "Stay waiting."`,
		`"approval_history": [`,
		`"requested_at": "2026-03-24T12:02:00Z"`,
		`"expires_at": "2026-03-24T12:05:00Z"`,
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("OperatorStatus() = %q, want substring %q", summary, want)
		}
	}
}

func TestTaskStateOperatorStatusPersistedRuntimePathUnchangedForTreasurySteps(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: "treasury-wallet"}
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got missioncontrol.OperatorStatusSummary
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for persisted runtime path", got.TreasuryPreflight)
	}
}

func TestTaskStateOperatorStatusMatchesActiveAndPersistedApprovalBindingMetadata(t *testing.T) {
	t.Parallel()

	job := testTaskStateJob()
	requestedAt := time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)
	resolvedAt := time.Date(2026, 3, 24, 12, 4, 0, 0, time.UTC)
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "draft",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeMissionStep,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				State:           missioncontrol.ApprovalStateDenied,
				RequestedAt:     requestedAt.Add(-2 * time.Minute),
				ResolvedAt:      requestedAt.Add(-1 * time.Minute),
			},
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneSession,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorReply,
				SessionChannel:  "slack",
				SessionChatID:   "C123::171234",
				State:           missioncontrol.ApprovalStateGranted,
				RequestedAt:     requestedAt,
				ResolvedAt:      resolvedAt,
			},
		},
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state := NewTaskState()
	state.SetExecutionContext(missioncontrol.ExecutionContext{
		Job:     &job,
		Step:    &job.Plan.Steps[0],
		Runtime: missioncontrol.CloneJobRuntimeState(&runtime),
	})

	activeSummary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus(active) error = %v", err)
	}

	if err := state.HydrateRuntimeControl(job, runtime, &control); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	persistedSummary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus(persisted) error = %v", err)
	}

	if activeSummary != persistedSummary {
		t.Fatalf("OperatorStatus active/persisted mismatch\nactive:\n%s\npersisted:\n%s", activeSummary, persistedSummary)
	}
	for _, want := range []string{
		`"requested_via": "runtime_waiting_user"`,
		`"granted_via": "operator_reply"`,
		`"session_channel": "slack"`,
		`"session_chat_id": "C123::171234"`,
		`"approval_history": [`,
		`"step_id": "build"`,
		`"requested_at": "2026-03-24T12:02:00Z"`,
		`"resolved_at": "2026-03-24T12:04:00Z"`,
	} {
		if !strings.Contains(persistedSummary, want) {
			t.Fatalf("OperatorStatus() = %q, want substring %q", persistedSummary, want)
		}
	}
}

func TestTaskStateOperatorStatusReturnsPausedReason(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		PausedAt:     time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC),
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	if !strings.Contains(summary, `"paused_reason": "operator_command"`) {
		t.Fatalf("OperatorStatus() = %q, want paused reason", summary)
	}
	if !strings.Contains(summary, `"paused_at": "2026-03-24T12:03:00Z"`) {
		t.Fatalf("OperatorStatus() = %q, want paused timestamp", summary)
	}
	if !strings.Contains(summary, `"allowed_tools": [`) || !strings.Contains(summary, `"read"`) {
		t.Fatalf("OperatorStatus() = %q, want effective allowed tools", summary)
	}
}

func TestTaskStateOperatorStatusReturnsRecentAuditForPersistedRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	persistedHistory := missioncontrol.AppendAuditHistory(nil, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "write_memory",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
	})
	persistedHistory = missioncontrol.AppendAuditHistory(persistedHistory, missioncontrol.AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "pause",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	})
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
		AuditHistory: persistedHistory,
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 2 {
		t.Fatalf("recent_audit = %#v, want two audit entries", got["recent_audit"])
	}
	first := recentAudit[0].(map[string]any)
	second := recentAudit[1].(map[string]any)
	if first["action"] != "pause" || second["action"] != "write_memory" {
		t.Fatalf("recent_audit actions = (%#v, %#v), want (%q, %q)", first["action"], second["action"], "pause", "write_memory")
	}
	if first["event_id"] != persistedHistory[1].EventID || second["event_id"] != persistedHistory[0].EventID {
		t.Fatalf("recent_audit event_ids = (%#v, %#v), want (%q, %q)", first["event_id"], second["event_id"], persistedHistory[1].EventID, persistedHistory[0].EventID)
	}
	if first["action_class"] != string(persistedHistory[1].ActionClass) || second["action_class"] != string(persistedHistory[0].ActionClass) {
		t.Fatalf("recent_audit action_class = (%#v, %#v), want (%q, %q)", first["action_class"], second["action_class"], persistedHistory[1].ActionClass, persistedHistory[0].ActionClass)
	}
	if first["result"] != string(persistedHistory[1].Result) || second["result"] != string(persistedHistory[0].Result) {
		t.Fatalf("recent_audit result = (%#v, %#v), want (%q, %q)", first["result"], second["result"], persistedHistory[1].Result, persistedHistory[0].Result)
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 1 || allowedTools[0] != "read" {
		t.Fatalf("allowed_tools = %#v, want [%q]", got["allowed_tools"], "read")
	}
}

func TestTaskStateOperatorStatusSurfacesTruncationMetadataForPersistedRuntime(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()

	requests := make([]missioncontrol.ApprovalRequest, 0, missioncontrol.OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < missioncontrol.OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, missioncontrol.ApprovalRequest{
			JobID:           "job-1",
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
			Scope:           missioncontrol.ApprovalScopeMissionStep,
			State:           missioncontrol.ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]missioncontrol.AuditEvent, 0, missioncontrol.OperatorStatusRecentAuditLimit+1)
	for i := 0; i < missioncontrol.OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, missioncontrol.AuditEvent{
			JobID:     "job-1",
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}

	runtime := missioncontrol.JobRuntimeState{
		JobID:            "job-1",
		State:            missioncontrol.JobStateWaitingUser,
		ActiveStepID:     "build",
		ApprovalRequests: requests,
		AuditHistory:     history,
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	for _, want := range []string{
		`"truncation": {`,
		`"approval_history_omitted": 2`,
		`"recent_audit_omitted": 1`,
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("OperatorStatus() = %q, want substring %q", summary, want)
		}
	}
}

func TestTaskStateOperatorStatusUsesPersistedRequestRevokedAtAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           missioncontrol.ApprovalStateRevoked,
				RequestedAt:     time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				ResolvedAt:      time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
				RevokedAt:       time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
			},
		},
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	if !strings.Contains(summary, `"revoked_at": "2026-03-24T12:01:00Z"`) {
		t.Fatalf("OperatorStatus() = %q, want persisted revoked_at", summary)
	}
}

func TestTaskStateOperatorStatusShowsNormalizedLegacyRevokedAtAfterRehydration(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	runtime := missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStateRunning,
		ActiveStepID: "build",
		ApprovalRequests: []missioncontrol.ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				RequestedVia:    missioncontrol.ApprovalRequestedViaRuntime,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           missioncontrol.ApprovalStateRevoked,
				RequestedAt:     time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				ResolvedAt:      time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
			},
		},
		ApprovalGrants: []missioncontrol.ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
				Scope:           missioncontrol.ApprovalScopeOneJob,
				GrantedVia:      missioncontrol.ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           missioncontrol.ApprovalStateRevoked,
				RevokedAt:       time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
			},
		},
	}
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}
	if !strings.Contains(summary, `"revoked_at": "2026-03-24T12:01:00Z"`) {
		t.Fatalf("OperatorStatus() = %q, want normalized revoked_at", summary)
	}
}

func TestTaskStateOperatorStatusIncludesDeterministicArtifactsForPersistedRuntime(t *testing.T) {
	t.Parallel()

	job := missioncontrol.Job{
		ID:           "job-1",
		SpecVersion:  missioncontrol.JobSpecVersionV2,
		MaxAuthority: missioncontrol.AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: missioncontrol.Plan{
			ID: "plan-1",
			Steps: []missioncontrol.Step{
				{ID: "gamma", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: missioncontrol.StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: missioncontrol.StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: missioncontrol.StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: missioncontrol.StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	plan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	runtime := missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "final",
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
		InspectablePlan: &plan,
		CompletedSteps: []missioncontrol.RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &missioncontrol.RuntimeResultingStateRecord{Kind: string(missioncontrol.StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}

	state := NewTaskState()
	if err := state.HydrateRuntimeControl(job, runtime, nil); err != nil {
		t.Fatalf("HydrateRuntimeControl() error = %v", err)
	}
	state.ClearExecutionContext()

	summary, err := state.OperatorStatus("job-1")
	if err != nil {
		t.Fatalf("OperatorStatus() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(summary), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	artifacts, ok := got["artifacts"].([]any)
	if !ok || len(artifacts) != missioncontrol.OperatorStatusArtifactLimit {
		t.Fatalf("artifacts = %#v, want %d deterministic entries", got["artifacts"], missioncontrol.OperatorStatusArtifactLimit)
	}
	first, ok := artifacts[0].(map[string]any)
	if !ok {
		t.Fatalf("artifacts[0] = %#v, want object", artifacts[0])
	}
	if first["step_id"] != "gamma" || first["path"] != "zeta.txt" {
		t.Fatalf("artifacts[0] = %#v, want step_id=%q path=%q", first, "gamma", "zeta.txt")
	}
	third, ok := artifacts[2].(map[string]any)
	if !ok {
		t.Fatalf("artifacts[2] = %#v, want object", artifacts[2])
	}
	if third["step_id"] != "beta" || third["state"] != "already_present" {
		t.Fatalf("artifacts[2] = %#v, want step_id=%q state=%q", third, "beta", "already_present")
	}
	truncation, ok := got["truncation"].(map[string]any)
	if !ok {
		t.Fatalf("truncation = %#v, want object", got["truncation"])
	}
	if truncation["artifacts_omitted"] != float64(1) {
		t.Fatalf("truncation.artifacts_omitted = %#v, want 1", truncation["artifacts_omitted"])
	}
}

func TestTaskStateOperatorStatusReportsTerminalRuntimeDeterministically(t *testing.T) {
	t.Parallel()

	for _, runtime := range []missioncontrol.JobRuntimeState{
		{
			JobID:         "job-1",
			State:         missioncontrol.JobStateAborted,
			AbortedReason: missioncontrol.RuntimeAbortReasonOperatorCommand,
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           "job-1",
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStateSuperseded,
					RequestedAt:     time.Date(2026, 3, 24, 11, 59, 0, 0, time.UTC),
					ResolvedAt:      time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				},
			},
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "build",
					ToolName:  "abort",
					Allowed:   true,
					Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			JobID: "job-1",
			State: missioncontrol.JobStateCompleted,
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           "job-1",
					StepID:          "final",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStateGranted,
					RequestedAt:     time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
					ResolvedAt:      time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
				},
			},
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "final",
					ToolName:  "status",
					Allowed:   true,
					Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
				},
			},
		},
		{
			JobID:    "job-1",
			State:    missioncontrol.JobStateFailed,
			FailedAt: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC),
			FailedSteps: []missioncontrol.RuntimeStepRecord{
				{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)},
			},
			ApprovalRequests: []missioncontrol.ApprovalRequest{
				{
					JobID:           "job-1",
					StepID:          "build",
					RequestedAction: missioncontrol.ApprovalRequestedActionStepComplete,
					Scope:           missioncontrol.ApprovalScopeMissionStep,
					State:           missioncontrol.ApprovalStateDenied,
					RequestedAt:     time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
					ResolvedAt:      time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC),
				},
			},
			AuditHistory: []missioncontrol.AuditEvent{
				{
					JobID:     "job-1",
					StepID:    "build",
					ToolName:  "pause",
					Allowed:   false,
					Code:      missioncontrol.RejectionCodeInvalidRuntimeState,
					Timestamp: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC),
				},
			},
		},
	} {
		runtime := runtime
		t.Run(string(runtime.State), func(t *testing.T) {
			state := NewTaskState()
			if err := state.HydrateRuntimeControl(testTaskStateJob(), runtime, nil); err != nil {
				t.Fatalf("HydrateRuntimeControl() error = %v", err)
			}

			summary, err := state.OperatorStatus("job-1")
			if err != nil {
				t.Fatalf("OperatorStatus() error = %v", err)
			}
			if !strings.Contains(summary, `"state": "`+string(runtime.State)+`"`) {
				t.Fatalf("OperatorStatus() = %q, want state %q", summary, runtime.State)
			}
			if !strings.Contains(summary, `"recent_audit": [`) {
				t.Fatalf("OperatorStatus() = %q, want recent audit block", summary)
			}
			if !strings.Contains(summary, `"approval_history": [`) {
				t.Fatalf("OperatorStatus() = %q, want approval history block", summary)
			}
			if strings.Contains(summary, `"allowed_tools":`) {
				t.Fatalf("OperatorStatus() = %q, want allowed_tools omitted without control context", summary)
			}
			if runtime.State == missioncontrol.JobStateFailed {
				if !strings.Contains(summary, `"failed_step_id": "build"`) {
					t.Fatalf("OperatorStatus() = %q, want failed step id", summary)
				}
				if !strings.Contains(summary, `"failure_reason": "validator failed"`) {
					t.Fatalf("OperatorStatus() = %q, want failure reason", summary)
				}
				if !strings.Contains(summary, `"failed_at": "2026-03-24T12:02:00Z"`) {
					t.Fatalf("OperatorStatus() = %q, want failure timestamp", summary)
				}
			}
		})
	}
}

func TestTaskStateOperatorStatusWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	_, err := state.OperatorStatus("other-job")
	if err == nil {
		t.Fatal("OperatorStatus(other-job) error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("OperatorStatus(other-job) error = %q, want job mismatch", err)
	}
}
