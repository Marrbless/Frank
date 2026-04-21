package tools

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/local/picobot/internal/missioncontrol"
)

func TestTaskStateOperatorInspectWithoutValidatedPlanReturnsDeterministicError(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	control, err := missioncontrol.BuildRuntimeControlContext(testTaskStateJob(), "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:        "job-1",
		State:        missioncontrol.JobStatePaused,
		ActiveStepID: "build",
		PausedReason: missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	_, err = state.OperatorInspect("job-1", "build")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want missing-plan failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeInvalidRuntimeState)) {
		t.Fatalf("OperatorInspect() error = %q, want invalid_runtime_state code", err)
	}
	if !strings.Contains(err.Error(), "inspect command requires validated mission plan") {
		t.Fatalf("OperatorInspect() error = %q, want missing validated plan message", err)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextZeroTreasuryRefPathUnchanged(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	if err := state.ActivateStep(testTaskStateJob(), "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil for zero-ref path", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for zero-ref path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedTreasuryPreflight(t *testing.T) {
	t.Parallel()

	root, treasury, container := writeTaskStateTreasuryFixtures(t)
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: treasury.TreasuryID}

	state := NewTaskState()
	state.SetMissionStoreRoot(root)
	if err := state.ActivateStep(job, "build"); err != nil {
		t.Fatalf("ActivateStep() error = %v", err)
	}

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight != nil {
		t.Fatalf("CampaignPreflight = %#v, want nil on treasury-only path", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight == nil {
		t.Fatal("TreasuryPreflight = nil, want resolved treasury/container data")
	}
	if summary.Steps[0].TreasuryPreflight.Treasury == nil {
		t.Fatal("TreasuryPreflight.Treasury = nil, want resolved treasury record")
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
		t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
	}
	if summary.Steps[0].TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
		t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
	}
	if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
		t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, container)
	}
}

func TestTaskStateOperatorInspectActiveExecutionContextSurfacesResolvedCampaignPreflight(t *testing.T) {
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

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
		t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", summary.Steps[0].CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
	}
	if len(summary.Steps[0].CampaignPreflight.Identities) != 1 || len(summary.Steps[0].CampaignPreflight.Accounts) != 1 || len(summary.Steps[0].CampaignPreflight.Containers) != 1 {
		t.Fatalf("CampaignPreflight = %#v, want one identity/account/container", summary.Steps[0].CampaignPreflight)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil on campaign-only path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectSurfacesCampaignZohoEmailAddressing(t *testing.T) {
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

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	envelope := mustTaskStateJSONObject(t, got)
	steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	preflight := step["campaign_preflight"].(map[string]any)
	campaignJSON := preflight["campaign"].(map[string]any)
	addressingJSON, ok := campaignJSON["zoho_email_addressing"].(map[string]any)
	if !ok {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing = %#v, want object", campaignJSON["zoho_email_addressing"])
	}
	assertTaskStateJSONObjectKeys(t, campaignJSON, "campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at", "zoho_email_addressing")
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["to"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.to"), []any{"person@example.com", "team@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.to = %#v, want [person@example.com team@example.com]", addressingJSON["to"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["cc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.cc"), []any{"copy@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.cc = %#v, want [copy@example.com]", addressingJSON["cc"])
	}
	if !reflect.DeepEqual(mustTaskStateJSONArray(t, addressingJSON["bcc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc"), []any{"blind@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc = %#v, want [blind@example.com]", addressingJSON["bcc"])
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1", len(summary.Steps))
	}
	if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
		t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight", summary.Steps[0].CampaignPreflight)
	}
	addressing := summary.Steps[0].CampaignPreflight.Campaign.ZohoEmailAddressing
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

func TestTaskStateOperatorInspectActiveAndPersistedPathsPreserveAdapterBoundaryContract(t *testing.T) {
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

		got, err := state.OperatorInspect("job-1", "build")
		if err != nil {
			t.Fatalf("OperatorInspect() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, got, true, true)

		summary := mustTaskStateReadoutJSON[missioncontrol.InspectSummary](t, got)
		envelope := mustTaskStateJSONObject(t, got)
		assertTaskStateJSONObjectKeys(t, envelope, "allowed_tools", "job_id", "max_authority", "steps")
		steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
		if len(steps) != 1 {
			t.Fatalf("steps len = %d, want 1", len(steps))
		}
		step, ok := steps[0].(map[string]any)
		if !ok {
			t.Fatalf("steps[0] = %#v, want object", steps[0])
		}
		assertTaskStateJSONObjectKeys(t, step, "allowed_tools", "campaign_preflight", "depends_on", "effective_allowed_tools", "required_authority", "requires_approval", "step_id", "step_type", "success_criteria", "treasury_preflight")
		assertTaskStateResolvedCampaignPreflightJSONEnvelope(t, step["campaign_preflight"])
		assertTaskStateResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
		if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
			t.Fatalf("Steps = %#v, want one build step", summary.Steps)
		}
		if summary.Steps[0].CampaignPreflight == nil || summary.Steps[0].CampaignPreflight.Campaign == nil {
			t.Fatalf("CampaignPreflight = %#v, want resolved campaign preflight on active path", summary.Steps[0].CampaignPreflight)
		}
		if summary.Steps[0].CampaignPreflight.Campaign.CampaignID != campaign.CampaignID {
			t.Fatalf("CampaignPreflight.Campaign.CampaignID = %q, want %q", summary.Steps[0].CampaignPreflight.Campaign.CampaignID, campaign.CampaignID)
		}
		if summary.Steps[0].TreasuryPreflight == nil || summary.Steps[0].TreasuryPreflight.Treasury == nil {
			t.Fatalf("TreasuryPreflight = %#v, want resolved treasury preflight on active path", summary.Steps[0].TreasuryPreflight)
		}
		if summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID != treasury.TreasuryID {
			t.Fatalf("TreasuryPreflight.Treasury.TreasuryID = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.TreasuryID, treasury.TreasuryID)
		}
		if summary.Steps[0].TreasuryPreflight.Treasury.State != missioncontrol.TreasuryStateActive {
			t.Fatalf("TreasuryPreflight.Treasury.State = %q, want %q", summary.Steps[0].TreasuryPreflight.Treasury.State, missioncontrol.TreasuryStateActive)
		}
		if !reflect.DeepEqual(summary.Steps[0].TreasuryPreflight.Containers, []missioncontrol.FrankContainerRecord{container}) {
			t.Fatalf("TreasuryPreflight.Containers = %#v, want [%#v]", summary.Steps[0].TreasuryPreflight.Containers, container)
		}
	})

	t.Run("persisted", func(t *testing.T) {
		t.Parallel()

		state := NewTaskState()
		inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
		if err != nil {
			t.Fatalf("BuildInspectablePlanContext() error = %v", err)
		}
		control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
		if err != nil {
			t.Fatalf("BuildRuntimeControlContext() error = %v", err)
		}

		state.runtimeState = missioncontrol.JobRuntimeState{
			JobID:           "job-1",
			State:           missioncontrol.JobStatePaused,
			ActiveStepID:    "build",
			InspectablePlan: &inspectablePlan,
			PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
		}
		state.hasRuntimeState = true
		state.runtimeControl = control
		state.hasRuntimeControl = true

		got, err := state.OperatorInspect("job-1", "build")
		if err != nil {
			t.Fatalf("OperatorInspect() error = %v", err)
		}
		assertTaskStateReadoutAdapterBoundary(t, got, false, false)

		summary := mustTaskStateReadoutJSON[missioncontrol.InspectSummary](t, got)
		envelope := mustTaskStateJSONObject(t, got)
		assertTaskStateJSONObjectKeys(t, envelope, "allowed_tools", "job_id", "max_authority", "steps")
		steps := mustTaskStateJSONArray(t, envelope["steps"], "inspect.steps")
		if len(steps) != 1 {
			t.Fatalf("steps len = %d, want 1", len(steps))
		}
		step, ok := steps[0].(map[string]any)
		if !ok {
			t.Fatalf("steps[0] = %#v, want object", steps[0])
		}
		assertTaskStateJSONObjectKeys(t, step, "allowed_tools", "depends_on", "effective_allowed_tools", "required_authority", "requires_approval", "step_id", "step_type", "success_criteria")
		if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
			t.Fatalf("Steps = %#v, want one build step", summary.Steps)
		}
		if summary.Steps[0].CampaignPreflight != nil {
			t.Fatalf("CampaignPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].CampaignPreflight)
		}
		if summary.Steps[0].TreasuryPreflight != nil {
			t.Fatalf("TreasuryPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].TreasuryPreflight)
		}
	})
}

func TestTaskStateOperatorInspectUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanPathUnchangedForTreasurySteps(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	job := testTaskStateJob()
	job.Plan.Steps[0].TreasuryRef = &missioncontrol.TreasuryRef{TreasuryID: "treasury-wallet"}
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	control, err := missioncontrol.BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true
	state.runtimeControl = control
	state.hasRuntimeControl = true

	got, err := state.OperatorInspect("job-1", "build")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if summary.Steps[0].TreasuryPreflight != nil {
		t.Fatalf("TreasuryPreflight = %#v, want nil for persisted inspectable-plan path", summary.Steps[0].TreasuryPreflight)
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanWrongJobDoesNotBind(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("other-job", "final")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want mismatch failure")
	}
	if !strings.Contains(err.Error(), "does not match the active job") {
		t.Fatalf("OperatorInspect() error = %q, want job mismatch", err)
	}
}

func TestTaskStateOperatorInspectPersistedInspectablePlanRejectsInvalidStep(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStatePaused,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		PausedReason:    missioncontrol.RuntimePauseReasonOperatorCommand,
	}
	state.hasRuntimeState = true

	_, err = state.OperatorInspect("job-1", "missing")
	if err == nil {
		t.Fatal("OperatorInspect() error = nil, want unknown-step failure")
	}
	if !strings.Contains(err.Error(), string(missioncontrol.RejectionCodeUnknownStep)) {
		t.Fatalf("OperatorInspect() error = %q, want unknown_step code", err)
	}
	if !strings.Contains(err.Error(), `step "missing" not found in plan`) {
		t.Fatalf("OperatorInspect() error = %q, want missing-step message", err)
	}
}

func TestTaskStateOperatorInspectTerminalRuntimeUsesPersistedInspectablePlanWithoutMissionJob(t *testing.T) {
	t.Parallel()

	state := NewTaskState()
	inspectablePlan, err := missioncontrol.BuildInspectablePlanContext(testTaskStateJob())
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	state.runtimeState = missioncontrol.JobRuntimeState{
		JobID:           "job-1",
		State:           missioncontrol.JobStateCompleted,
		InspectablePlan: &inspectablePlan,
		CompletedAt:     time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}
	state.hasRuntimeState = true

	got, err := state.OperatorInspect("job-1", "final")
	if err != nil {
		t.Fatalf("OperatorInspect() error = %v", err)
	}

	var summary missioncontrol.InspectSummary
	if err := json.Unmarshal([]byte(got), &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
}
