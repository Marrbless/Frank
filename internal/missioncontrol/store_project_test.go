package missioncontrol

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testLeaseSafeNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func TestBuildCommittedMissionStatusSnapshotDeterministic(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:             "job-1",
		SpecVersion:    JobSpecVersionV2,
		MaxAuthority:   AuthorityTierHigh,
		AllowedTools:   []string{"read"},
		SelectedSkills: []string{"job-skill"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{ID: "gamma", Type: StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"zeta"}, SelectedSkills: []string{"step-skill"}},
			},
		},
	}
	control, err := BuildRuntimeControlContext(job, "final")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	requests := make([]ApprovalRequest, 0, OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, ApprovalRequest{
			JobID:           job.ID,
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}
	history := make([]AuditEvent, 0, OperatorStatusRecentAuditLimit+1)
	for i := 0; i < OperatorStatusRecentAuditLimit+1; i++ {
		history = append(history, AuditEvent{
			JobID:     job.ID,
			StepID:    "build",
			ToolName:  "status",
			Allowed:   true,
			Timestamp: time.Date(2026, 3, 24, 13, i, 0, 0, time.UTC),
		})
	}
	now := testLeaseSafeNow()
	requestBase := now.Add(-8 * time.Hour)
	auditBase := now.Add(-7 * time.Hour)
	pausedAt := now.Add(-6 * time.Hour)
	for i := range requests {
		requests[i].RequestedAt = requestBase.Add(time.Duration(i) * time.Minute)
	}
	for i := range history {
		history[i].Timestamp = auditBase.Add(time.Duration(i) * time.Minute)
	}
	runtime := JobRuntimeState{
		JobID:            job.ID,
		State:            JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     RuntimePauseReasonOperatorCommand,
		PausedAt:         pausedAt,
		ApprovalRequests: requests,
		AuditHistory:     history,
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "zeta"},
			{StepID: "gamma"},
			{StepID: "beta", ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha"},
			{StepID: "epsilon"},
			{StepID: "delta"},
		},
	}

	root := t.TempDir()
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	first, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(first) error = %v", err)
	}
	second, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(second) error = %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("json.Marshal(first) error = %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("json.Marshal(second) error = %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("snapshot JSON differs across identical durable projections:\nfirst=%s\nsecond=%s", string(firstJSON), string(secondJSON))
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("snapshot differs across identical durable projections:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want deterministic summary")
	}
	if first.Skills == nil || !reflect.DeepEqual(first.Skills.Selected, []string{"job-skill", "step-skill"}) {
		t.Fatalf("Skills = %#v, want selected job and step skills", first.Skills)
	}
	if first.RuntimeSummary.Skills == nil || !reflect.DeepEqual(first.RuntimeSummary.Skills.Selected, []string{"job-skill", "step-skill"}) {
		t.Fatalf("RuntimeSummary.Skills = %#v, want selected job and step skills", first.RuntimeSummary.Skills)
	}
	if len(first.RuntimeSummary.RecentAudit) != OperatorStatusRecentAuditLimit {
		t.Fatalf("RecentAudit len = %d, want %d", len(first.RuntimeSummary.RecentAudit), OperatorStatusRecentAuditLimit)
	}
	wantRecentAudit0 := auditBase.Add(time.Duration(OperatorStatusRecentAuditLimit) * time.Minute).Format(time.RFC3339)
	if first.RuntimeSummary.RecentAudit[0].Timestamp != wantRecentAudit0 {
		t.Fatalf("RecentAudit[0].Timestamp = %q, want %q", first.RuntimeSummary.RecentAudit[0].Timestamp, wantRecentAudit0)
	}
	if len(first.RuntimeSummary.Artifacts) != OperatorStatusArtifactLimit {
		t.Fatalf("Artifacts len = %d, want %d", len(first.RuntimeSummary.Artifacts), OperatorStatusArtifactLimit)
	}
	if first.RuntimeSummary.Artifacts[0].StepID != "gamma" || first.RuntimeSummary.Artifacts[0].Path != "zeta.txt" {
		t.Fatalf("Artifacts[0] = %#v, want step_id=%q path=%q", first.RuntimeSummary.Artifacts[0], "gamma", "zeta.txt")
	}
	if first.RuntimeSummary.Artifacts[2].StepID != "beta" || first.RuntimeSummary.Artifacts[2].State != "already_present" {
		t.Fatalf("Artifacts[2] = %#v, want step_id=%q state=%q", first.RuntimeSummary.Artifacts[2], "beta", "already_present")
	}
}

func TestBuildCommittedMissionStatusSnapshotClearsTerminalControl(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	running := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, running, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(running) error = %v", err)
	}

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	completed := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateCompleted,
		InspectablePlan: &plan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		CompletedAt:     now,
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "artifact", At: now.Add(-75 * time.Second), ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeStaticArtifact), Target: "dist/report.json", State: "verified"}},
			{StepID: "build", At: now.Add(-30 * time.Second)},
		},
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, completed, nil, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(completed) error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.Active {
		t.Fatal("Active = true, want false for terminal runtime")
	}
	if snapshot.StepID != "" {
		t.Fatalf("StepID = %q, want empty", snapshot.StepID)
	}
	if snapshot.StepType != "" {
		t.Fatalf("StepType = %q, want empty", snapshot.StepType)
	}
	if snapshot.RuntimeControl != nil {
		t.Fatalf("RuntimeControl = %#v, want nil for terminal runtime", snapshot.RuntimeControl)
	}
	if snapshot.Runtime == nil || snapshot.Runtime.State != JobStateCompleted {
		t.Fatalf("Runtime = %#v, want completed runtime", snapshot.Runtime)
	}
}

func TestBuildCommittedMissionStatusSnapshotDoesNotProjectTreasuryPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: "campaign-mail"}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: "treasury-wallet"}
	if err := StoreCampaignRecord(root, CampaignRecord{
		RecordVersion:  StoreRecordVersion,
		CampaignID:     "campaign-mail",
		CampaignKind:   CampaignKindOutreach,
		DisplayName:    "Mail Outreach",
		State:          CampaignStateActive,
		Objective:      "Reach aligned operators",
		IdentityMode:   IdentityModeAgentAlias,
		CreatedAt:      now.Add(-4 * time.Minute).UTC(),
		UpdatedAt:      now.Add(-3 * time.Minute).UTC(),
		StopConditions: []string{"stop after 3 replies"},
		FailureThreshold: CampaignFailureThreshold{
			Metric: "bounced_messages",
			Limit:  3,
		},
		ComplianceChecks: []string{"can-spam-reviewed"},
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		},
		FrankObjectRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: fixtures.identity.IdentityID,
			},
		},
	}); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}
	if err := StoreTreasuryRecord(root, TreasuryRecord{
		RecordVersion:  StoreRecordVersion,
		TreasuryID:     "treasury-wallet",
		DisplayName:    "Frank Treasury",
		State:          TreasuryStateBootstrap,
		ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
		ContainerRefs: []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		},
		CreatedAt: now.Add(-3 * time.Minute).UTC(),
		UpdatedAt: now.UTC(),
	}); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}
	if err := StoreTreasuryLedgerEntry(root, TreasuryLedgerEntry{
		RecordVersion: StoreRecordVersion,
		EntryID:       "entry-wallet",
		TreasuryID:    "treasury-wallet",
		EntryKind:     TreasuryLedgerEntryKindMovement,
		AssetCode:     "USDC",
		Amount:        "42.00",
		CreatedAt:     now.Add(-2 * time.Minute).UTC(),
		SourceRef:     "campaign:community-a",
	}); err != nil {
		t.Fatalf("StoreTreasuryLedgerEntry() error = %v", err)
	}

	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal(snapshot) error = %v", err)
	}

	forbidden := []string{
		"\"treasury_preflight\"",
		"\"audience_class_or_target\"",
		"\"message_family_or_participation_style\"",
		"\"cadence\"",
		"\"escalation_rules\"",
		"\"budget\":",
		"\"active_container_id\"",
		"\"custody_model\"",
		"\"permitted_transaction_classes\"",
		"\"forbidden_transaction_classes\"",
		"\"ledger_ref\"",
		"\"direction\":\"internal\"",
		"\"status\":\"recorded\"",
		"\"container_id\":\"container-wallet\"",
		"\"display_name\":\"Frank Treasury\"",
	}
	for _, key := range forbidden {
		if strings.Contains(string(data), key) {
			t.Fatalf("snapshot JSON unexpectedly contains %s: %s", key, string(data))
		}
	}
}

func writeDeferredSchedulerTriggerForTest(t *testing.T, root string, record deferredScheduledTriggerStoreRecord, filename string) {
	t.Helper()

	if err := WriteStoreJSONAtomic(filepath.Join(deferredSchedulerTriggersDir(root), filename), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(deferred trigger) error = %v", err)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesDeferredSchedulerTriggersInRuntimeSummary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now,
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	records := []deferredScheduledTriggerStoreRecord{
		{
			RecordVersion:  1,
			TriggerID:      "scheduled-trigger-job-2-20260413T150000.000000000Z",
			SchedulerJobID: "job-2",
			Name:           "stretch",
			Message:        "stand up and stretch",
			FireAt:         time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC),
			DeferredAt:     time.Date(2026, 4, 13, 15, 1, 0, 0, time.UTC),
		},
		{
			RecordVersion:  1,
			TriggerID:      "scheduled-trigger-job-1-20260413T140000.000000000Z",
			SchedulerJobID: "job-1",
			Name:           "water",
			Message:        "drink water",
			FireAt:         time.Date(2026, 4, 13, 14, 0, 0, 0, time.UTC),
			DeferredAt:     time.Date(2026, 4, 13, 14, 2, 0, 0, time.UTC),
		},
	}
	for i, record := range records {
		writeDeferredSchedulerTriggerForTest(t, root, record, fmt.Sprintf("%02d.json", i))
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if !snapshot.Active {
		t.Fatal("Active = false, want unchanged active runtime state")
	}
	if snapshot.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", snapshot.JobID, job.ID)
	}
	if snapshot.StepID != "build" {
		t.Fatalf("StepID = %q, want %q", snapshot.StepID, "build")
	}
	if snapshot.Runtime == nil || snapshot.Runtime.State != JobStateRunning {
		t.Fatalf("Runtime = %#v, want running runtime state", snapshot.Runtime)
	}
	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want operator summary with deferred scheduler visibility")
	}
	if len(snapshot.RuntimeSummary.DeferredSchedulerTriggers) != 2 {
		t.Fatalf("DeferredSchedulerTriggers len = %d, want 2", len(snapshot.RuntimeSummary.DeferredSchedulerTriggers))
	}
	if snapshot.RuntimeSummary.DeferredSchedulerTriggers[0].TriggerID != "scheduled-trigger-job-1-20260413T140000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[0] = %#v, want earliest deferred trigger first", snapshot.RuntimeSummary.DeferredSchedulerTriggers[0])
	}
	if snapshot.RuntimeSummary.DeferredSchedulerTriggers[0].Message != "drink water" {
		t.Fatalf("DeferredSchedulerTriggers[0].Message = %q, want %q", snapshot.RuntimeSummary.DeferredSchedulerTriggers[0].Message, "drink water")
	}
	if snapshot.RuntimeSummary.DeferredSchedulerTriggers[1].TriggerID != "scheduled-trigger-job-2-20260413T150000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[1] = %#v, want later deferred trigger second", snapshot.RuntimeSummary.DeferredSchedulerTriggers[1])
	}
}

func TestBuildCommittedMissionStatusSnapshotMayCarryProviderSpecificZohoRuntimeSummaryFieldsFromCommittedRuntime(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	action, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		},
		"Frank intro",
		"plaintext",
		"Hello from Frank",
		now.Add(-90*time.Second),
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	action, err = BuildCampaignZohoEmailOutboundSentAction(action, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}
	action.State = CampaignZohoEmailOutboundActionStateVerified
	action.VerifiedAt = now.Add(-15 * time.Second)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(verified) error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:                            job.ID,
		State:                            JobStateRunning,
		ActiveStepID:                     "build",
		InspectablePlan:                  &inspectablePlan,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{action},
		CreatedAt:                        now.Add(-2 * time.Minute),
		UpdatedAt:                        now,
		StartedAt:                        now.Add(-2 * time.Minute),
		ActiveStepAt:                     now.Add(-time.Minute),
		FrankZohoSendReceipts: []FrankZohoSendReceipt{
			{
				StepID:             "build",
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				FromAddress:        "frank@omou.online",
				FromDisplayName:    "Frank",
				ProviderMessageID:  "1711540357880100000",
				ProviderMailID:     "<mail-1@zoho.test>",
				MIMEMessageID:      "<mime-1@example.test>",
				OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
			},
		},
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	if len(snapshot.RuntimeSummary.CampaignZohoEmailOutbounds) > 0 {
		outbound := snapshot.RuntimeSummary.CampaignZohoEmailOutbounds[0]
		if outbound.ActionID != action.ActionID {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailOutbounds[0].ActionID = %q, want %q", outbound.ActionID, action.ActionID)
		}
		if outbound.CampaignID != "campaign-mail" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailOutbounds[0].CampaignID = %q, want campaign-mail", outbound.CampaignID)
		}
		if outbound.ProviderMessageID != "1711540357880100000" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailOutbounds[0].ProviderMessageID = %q, want canonical provider message id", outbound.ProviderMessageID)
		}
		if outbound.VerifiedAt == nil || *outbound.VerifiedAt != now.Add(-15*time.Second).Format(time.RFC3339Nano) {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailOutbounds[0].VerifiedAt = %#v, want verified timestamp", outbound.VerifiedAt)
		}
	}
	if len(snapshot.RuntimeSummary.FrankZohoSendProof) > 0 {
		proof := snapshot.RuntimeSummary.FrankZohoSendProof[0]
		if proof.ProviderMessageID != "1711540357880100000" {
			t.Fatalf("RuntimeSummary.FrankZohoSendProof[0].ProviderMessageID = %q, want canonical provider message id", proof.ProviderMessageID)
		}
		if proof.ProviderMailID != "<mail-1@zoho.test>" {
			t.Fatalf("RuntimeSummary.FrankZohoSendProof[0].ProviderMailID = %q, want secondary provider mail id", proof.ProviderMailID)
		}
		if proof.MIMEMessageID != "<mime-1@example.test>" {
			t.Fatalf("RuntimeSummary.FrankZohoSendProof[0].MIMEMessageID = %q, want secondary MIME message id", proof.MIMEMessageID)
		}
		if proof.ProviderAccountID != "3323462000000008002" {
			t.Fatalf("RuntimeSummary.FrankZohoSendProof[0].ProviderAccountID = %q, want proof locator account id", proof.ProviderAccountID)
		}
		if proof.OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
			t.Fatalf("RuntimeSummary.FrankZohoSendProof[0].OriginalMessageURL = %q, want proof-compatible originalmessage URL", proof.OriginalMessageURL)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotMayCarryFrankZohoInboundReplies(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	reply := NormalizeFrankZohoInboundReply(FrankZohoInboundReply{
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102000",
		ProviderMailID:     "<reply-1@zoho.test>",
		MIMEMessageID:      "<reply-1@example.test>",
		InReplyTo:          "<parent@example.test>",
		References:         []string{"<seed@example.test>", "<parent@example.test>"},
		FromAddress:        "person@example.com",
		FromDisplayName:    "Person One",
		FromAddressCount:   1,
		Subject:            "Re: Frank intro",
		ReceivedAt:         now.Add(-time.Minute),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage",
	})

	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateRunning,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		ActiveStepAt:    now.Add(-time.Minute),
	}
	runtime, changed, err := AppendFrankZohoInboundReply(runtime, reply)
	if err != nil {
		t.Fatalf("AppendFrankZohoInboundReply() error = %v", err)
	}
	if !changed || len(runtime.FrankZohoInboundReplies) != 1 {
		t.Fatalf("AppendFrankZohoInboundReply() changed = %v len = %d, want one normalized reply", changed, len(runtime.FrankZohoInboundReplies))
	}
	reply = runtime.FrankZohoInboundReplies[0]
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	if len(snapshot.RuntimeSummary.FrankZohoInboundReplies) > 0 {
		got := snapshot.RuntimeSummary.FrankZohoInboundReplies[0]
		if got.ReplyID != reply.ReplyID {
			t.Fatalf("RuntimeSummary.FrankZohoInboundReplies[0].ReplyID = %q, want %q", got.ReplyID, reply.ReplyID)
		}
		if got.FromAddressCount != 1 {
			t.Fatalf("RuntimeSummary.FrankZohoInboundReplies[0].FromAddressCount = %d, want 1", got.FromAddressCount)
		}
		if got.OriginalMessageURL != reply.OriginalMessageURL {
			t.Fatalf("RuntimeSummary.FrankZohoInboundReplies[0].OriginalMessageURL = %q, want %q", got.OriginalMessageURL, reply.OriginalMessageURL)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotMayCarryCampaignZohoEmailSendGate(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Now().UTC().Truncate(time.Second)

	job := testProjectedRuntimeJob()
	job.Plan.Steps[1].CampaignRef = &CampaignRef{CampaignID: "campaign-mail"}
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
		record.StopConditions = []string{"stop after 3 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
		record.ZohoEmailAddressing = &CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		}
	})
	if err := StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	action, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		},
		"Frank intro",
		"plaintext",
		"Hello from Frank",
		now.Add(-90*time.Second),
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	action, err = BuildCampaignZohoEmailOutboundSentAction(action, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}
	action.State = CampaignZohoEmailOutboundActionStateVerified
	action.VerifiedAt = now.Add(-15 * time.Second)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(verified) error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:                            job.ID,
		State:                            JobStateRunning,
		ActiveStepID:                     "build",
		InspectablePlan:                  &inspectablePlan,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{action},
		CreatedAt:                        now.Add(-2 * time.Minute),
		UpdatedAt:                        now,
		StartedAt:                        now.Add(-2 * time.Minute),
		ActiveStepAt:                     now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	if gate := snapshot.RuntimeSummary.CampaignZohoEmailSendGate; gate != nil {
		if gate.CampaignID != "campaign-mail" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.CampaignID = %q, want campaign-mail", gate.CampaignID)
		}
		if !gate.Allowed || gate.Halted {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate = %#v, want allowed non-halted gate", gate)
		}
		if gate.VerifiedSuccessCount != 1 {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.VerifiedSuccessCount = %d, want 1", gate.VerifiedSuccessCount)
		}
		if gate.FailureThresholdMetric != "rejections" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.FailureThresholdMetric = %q, want rejections", gate.FailureThresholdMetric)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotMayCarryUnsupportedCampaignZohoEmailStopConditionAsClosedGate(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Now().UTC().Truncate(time.Second)

	job := testProjectedRuntimeJob()
	job.Plan.Steps[1].CampaignRef = &CampaignRef{CampaignID: "campaign-mail"}
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
		record.StopConditions = []string{"stop after 3 opens"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "rejections", Limit: 3}
		record.ZohoEmailAddressing = &CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		}
	})
	if err := StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateRunning,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		ActiveStepAt:    now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	if gate := snapshot.RuntimeSummary.CampaignZohoEmailSendGate; gate != nil {
		if gate.CampaignID != "campaign-mail" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.CampaignID = %q, want campaign-mail", gate.CampaignID)
		}
		if gate.Allowed {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Allowed = true, want closed gate for unsupported stop condition: %#v", gate)
		}
		if gate.Halted {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Halted = true, want fail-closed unsupported gate without triggered halt: %#v", gate)
		}
		if gate.Reason != `campaign zoho email stop_condition "stop after 3 opens" is not evaluable from committed outbound and inbound reply records` {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Reason = %q, want unsupported stop-condition reason", gate.Reason)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotSupportsCampaignZohoEmailBouncedMessageFailureThreshold(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Now().UTC().Truncate(time.Second)

	job := testProjectedRuntimeJob()
	job.Plan.Steps[1].CampaignRef = &CampaignRef{CampaignID: "campaign-mail"}
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
		record.StopConditions = []string{"stop after 3 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "bounced_messages", Limit: 3}
		record.ZohoEmailAddressing = &CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		}
	})
	if err := StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	actionOne, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{To: []string{"person@example.com"}},
		"Frank intro one",
		"plaintext",
		"Hello from Frank one",
		now.Add(-3*time.Minute),
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction(actionOne) error = %v", err)
	}
	actionOne, err = BuildCampaignZohoEmailOutboundSentAction(actionOne, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100001",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100001/originalmessage",
	}, now.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction(actionOne) error = %v", err)
	}
	actionOne.State = CampaignZohoEmailOutboundActionStateVerified
	actionOne.VerifiedAt = now.Add(-110 * time.Second)
	if err := ValidateCampaignZohoEmailOutboundAction(actionOne); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(actionOne) error = %v", err)
	}

	actionTwo, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{To: []string{"person@example.com"}},
		"Frank intro two",
		"plaintext",
		"Hello from Frank two",
		now.Add(-150*time.Second),
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction(actionTwo) error = %v", err)
	}
	actionTwo, err = BuildCampaignZohoEmailOutboundSentAction(actionTwo, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100002",
		ProviderMailID:     "<mail-2@zoho.test>",
		MIMEMessageID:      "<mime-2@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100002/originalmessage",
	}, now.Add(-100*time.Second))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction(actionTwo) error = %v", err)
	}
	actionTwo.State = CampaignZohoEmailOutboundActionStateVerified
	actionTwo.VerifiedAt = now.Add(-90 * time.Second)
	if err := ValidateCampaignZohoEmailOutboundAction(actionTwo); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(actionTwo) error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:                            job.ID,
		State:                            JobStateRunning,
		ActiveStepID:                     "build",
		InspectablePlan:                  &inspectablePlan,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{actionOne, actionTwo},
		CreatedAt:                        now.Add(-2 * time.Minute),
		UpdatedAt:                        now,
		StartedAt:                        now.Add(-2 * time.Minute),
		ActiveStepAt:                     now.Add(-time.Minute),
	}
	var changed bool
	runtime, changed, err = AppendFrankZohoBounceEvidence(runtime, FrankZohoBounceEvidence{
		StepID:             "sync",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102001",
		ReceivedAt:         now.Add(-30 * time.Second),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880102001/originalmessage",
		CampaignID:         "campaign-mail",
		OutboundActionID:   actionOne.ActionID,
	})
	if err != nil || !changed {
		t.Fatalf("AppendFrankZohoBounceEvidence(first) changed=%v err=%v, want appended bounce evidence", changed, err)
	}
	runtime, changed, err = AppendFrankZohoBounceEvidence(runtime, FrankZohoBounceEvidence{
		StepID:             "sync",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102002",
		ReceivedAt:         now.Add(-20 * time.Second),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880102002/originalmessage",
		CampaignID:         "campaign-mail",
		OutboundActionID:   actionTwo.ActionID,
	})
	if err != nil || !changed {
		t.Fatalf("AppendFrankZohoBounceEvidence(second) changed=%v err=%v, want appended bounce evidence", changed, err)
	}
	runtime, changed, err = AppendFrankZohoBounceEvidence(runtime, FrankZohoBounceEvidence{
		StepID:             "sync",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102003",
		ReceivedAt:         now.Add(-10 * time.Second),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880102003/originalmessage",
	})
	if err != nil || !changed {
		t.Fatalf("AppendFrankZohoBounceEvidence(third) changed=%v err=%v, want appended bounce evidence", changed, err)
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	if gate := snapshot.RuntimeSummary.CampaignZohoEmailSendGate; gate != nil {
		if gate.CampaignID != "campaign-mail" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.CampaignID = %q, want campaign-mail", gate.CampaignID)
		}
		if !gate.Allowed || gate.Halted {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate = %#v, want allowed non-halted gate below bounced-message threshold", gate)
		}
		if gate.FailureThresholdMetric != "bounced_messages" {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.FailureThresholdMetric = %q, want bounced_messages", gate.FailureThresholdMetric)
		}
		if gate.AttributedBounceCount != 2 {
			t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.AttributedBounceCount = %d, want 2", gate.AttributedBounceCount)
		}
	}
}

func TestBuildCommittedMissionStatusSnapshotSupportsCampaignZohoEmailAmbiguousOutcomeFailureThreshold(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Now().UTC().Truncate(time.Second)

	job := testProjectedRuntimeJob()
	job.Plan.Steps[1].CampaignRef = &CampaignRef{CampaignID: "campaign-mail"}
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	campaign := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
		record.StopConditions = []string{"stop after 3 verified sends"}
		record.FailureThreshold = CampaignFailureThreshold{Metric: "ambiguous_outcomes", Limit: 2}
		record.ZohoEmailAddressing = &CampaignZohoEmailAddressing{
			To: []string{"person@example.com"},
		}
	})
	if err := StoreCampaignRecord(root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateRunning,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{
			mustBuildPreparedCampaignZohoEmailOutboundAction(t, "build", "campaign-mail", "subject-1", now.Add(-2*time.Minute)),
			mustBuildPreparedCampaignZohoEmailOutboundAction(t, "build", "campaign-mail", "subject-2", now.Add(-time.Minute)),
		},
		CreatedAt:    now.Add(-3 * time.Minute),
		UpdatedAt:    now,
		StartedAt:    now.Add(-3 * time.Minute),
		ActiveStepAt: now.Add(-2 * time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}

	if snapshot.RuntimeSummary == nil {
		t.Fatal("RuntimeSummary = nil, want committed runtime summary")
	}
	gate := snapshot.RuntimeSummary.CampaignZohoEmailSendGate
	if gate == nil {
		t.Fatal("RuntimeSummary.CampaignZohoEmailSendGate = nil, want derived gate")
	}
	if gate.CampaignID != "campaign-mail" {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.CampaignID = %q, want campaign-mail", gate.CampaignID)
	}
	if gate.Allowed {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Allowed = true, want false at ambiguous-outcome limit: %#v", gate)
	}
	if !gate.Halted {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Halted = false, want true at ambiguous-outcome limit: %#v", gate)
	}
	if gate.FailureThresholdMetric != "ambiguous_outcomes" {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.FailureThresholdMetric = %q, want ambiguous_outcomes", gate.FailureThresholdMetric)
	}
	if gate.AmbiguousOutcomeCount != 2 {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.AmbiguousOutcomeCount = %d, want 2", gate.AmbiguousOutcomeCount)
	}
	if gate.Reason != `campaign zoho email failure_threshold "ambiguous_outcomes" reached 2/2 counted ambiguous outcomes` {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailSendGate.Reason = %q, want ambiguous-outcome threshold reason", gate.Reason)
	}
}

func TestBuildCommittedMissionStatusSnapshotDeterministicallyOrdersDeferredSchedulerTriggers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now,
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	writeDeferredSchedulerTriggerForTest(t, root, deferredScheduledTriggerStoreRecord{
		RecordVersion:  1,
		TriggerID:      "scheduled-trigger-b-20260413T140000.000000000Z",
		SchedulerJobID: "job-b",
		Name:           "later-name",
		Message:        "later message",
		FireAt:         time.Date(2026, 4, 13, 14, 0, 0, 0, time.UTC),
		DeferredAt:     time.Date(2026, 4, 13, 14, 5, 0, 0, time.UTC),
	}, "b.json")
	writeDeferredSchedulerTriggerForTest(t, root, deferredScheduledTriggerStoreRecord{
		RecordVersion:  1,
		TriggerID:      "scheduled-trigger-a-20260413T140000.000000000Z",
		SchedulerJobID: "job-a",
		Name:           "earlier-name",
		Message:        "earlier message",
		FireAt:         time.Date(2026, 4, 13, 14, 0, 0, 0, time.UTC),
		DeferredAt:     time.Date(2026, 4, 13, 14, 1, 0, 0, time.UTC),
	}, "a.json")

	first, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(first) error = %v", err)
	}
	second, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot(second) error = %v", err)
	}

	if !reflect.DeepEqual(first.RuntimeSummary.DeferredSchedulerTriggers, second.RuntimeSummary.DeferredSchedulerTriggers) {
		t.Fatalf("DeferredSchedulerTriggers differ across identical projections:\nfirst=%#v\nsecond=%#v", first.RuntimeSummary.DeferredSchedulerTriggers, second.RuntimeSummary.DeferredSchedulerTriggers)
	}
	if got := first.RuntimeSummary.DeferredSchedulerTriggers[0].TriggerID; got != "scheduled-trigger-a-20260413T140000.000000000Z" {
		t.Fatalf("DeferredSchedulerTriggers[0].TriggerID = %q, want lexicographically first tie-breaker", got)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesModelRouteStatus(t *testing.T) {
	t.Parallel()

	storeFixture := writeMissionStoreRuntimeFixture(t)
	model := &OperatorModelRouteStatus{
		SelectedModelRef: "cloud_reasoning",
		ProviderRef:      "openrouter",
		ProviderModel:    "google/gemini-test",
		SelectionReason:  "routing_default",
		FallbackDepth:    0,
		PolicyID:         "default",
		Capabilities: OperatorModelCapabilitiesStatus{
			Local:         false,
			Offline:       false,
			SupportsTools: true,
			AuthorityTier: "high",
		},
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(storeFixture.root, storeFixture.job.ID, MissionStatusSnapshotOptions{
		MissionRequired: true,
		MissionFile:     "mission.json",
		UpdatedAt:       storeFixture.now,
		Model:           model,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.Model == nil || snapshot.Model.ProviderModel != "google/gemini-test" {
		t.Fatalf("snapshot.Model = %#v, want model route", snapshot.Model)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.Model == nil || snapshot.RuntimeSummary.Model.SelectedModelRef != "cloud_reasoning" {
		t.Fatalf("snapshot.RuntimeSummary.Model = %#v, want model route", snapshot.RuntimeSummary)
	}

	model.ProviderModel = "mutated"
	if snapshot.Model.ProviderModel != "google/gemini-test" {
		t.Fatalf("snapshot.Model.ProviderModel mutated to %q", snapshot.Model.ProviderModel)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesModelControlMetrics(t *testing.T) {
	t.Parallel()

	storeFixture := writeMissionStoreRuntimeFixture(t)
	metrics := &OperatorModelControlMetricsStatus{
		RouteAttemptCount:          3,
		RouteSuccessCount:          2,
		RouteFailureCount:          1,
		FallbackCount:              1,
		ProviderHealthFailureCount: 1,
		ModelPolicyDenialCount:     1,
		AuthorityDenialCount:       1,
		ToolSchemaSuppressedCount:  2,
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(storeFixture.root, storeFixture.job.ID, MissionStatusSnapshotOptions{
		MissionRequired: true,
		MissionFile:     "mission.json",
		UpdatedAt:       storeFixture.now,
		ModelMetrics:    metrics,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.ModelMetrics == nil || snapshot.ModelMetrics.RouteAttemptCount != 3 || snapshot.ModelMetrics.ToolSchemaSuppressedCount != 2 {
		t.Fatalf("snapshot.ModelMetrics = %#v, want metrics", snapshot.ModelMetrics)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.ModelMetrics == nil || snapshot.RuntimeSummary.ModelMetrics.FallbackCount != 1 {
		t.Fatalf("snapshot.RuntimeSummary.ModelMetrics = %#v, want metrics", snapshot.RuntimeSummary)
	}

	metrics.RouteAttemptCount = 99
	if snapshot.ModelMetrics.RouteAttemptCount != 3 {
		t.Fatalf("snapshot.ModelMetrics.RouteAttemptCount mutated to %d", snapshot.ModelMetrics.RouteAttemptCount)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesModelHealthStatus(t *testing.T) {
	t.Parallel()

	storeFixture := writeMissionStoreRuntimeFixture(t)
	health := []OperatorModelHealthStatus{
		{ModelRef: "cloud_reasoning", ProviderRef: "openrouter", Status: "healthy", LastCheckedAt: "2026-05-01T12:00:00Z", FallbackAvailable: true},
		{ModelRef: "local_fast", ProviderRef: "llamacpp_phone", Status: "unhealthy", LastCheckedAt: "2026-05-01T12:00:00Z", LastErrorClass: "connection_refused"},
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(storeFixture.root, storeFixture.job.ID, MissionStatusSnapshotOptions{
		MissionRequired: true,
		MissionFile:     "mission.json",
		UpdatedAt:       storeFixture.now,
		ModelHealth:     health,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if len(snapshot.ModelHealth) != 2 || snapshot.ModelHealth[1].LastErrorClass != "connection_refused" {
		t.Fatalf("snapshot.ModelHealth = %#v, want health status", snapshot.ModelHealth)
	}
	if snapshot.RuntimeSummary == nil || len(snapshot.RuntimeSummary.ModelHealth) != 2 || snapshot.RuntimeSummary.ModelHealth[0].Status != "healthy" {
		t.Fatalf("snapshot.RuntimeSummary.ModelHealth = %#v, want health status", snapshot.RuntimeSummary)
	}

	health[0].Status = "mutated"
	if snapshot.ModelHealth[0].Status != "healthy" {
		t.Fatalf("snapshot.ModelHealth[0].Status mutated to %q", snapshot.ModelHealth[0].Status)
	}
}

func TestMissionStatusSnapshotSchemaUnchanged(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(MissionStatusSnapshot{})
	expected := []struct {
		name string
		tag  string
	}{
		{name: "MissionRequired", tag: `json:"mission_required"`},
		{name: "Active", tag: `json:"active"`},
		{name: "MissionFile", tag: `json:"mission_file"`},
		{name: "JobID", tag: `json:"job_id"`},
		{name: "StepID", tag: `json:"step_id"`},
		{name: "StepType", tag: `json:"step_type"`},
		{name: "RequiredAuthority", tag: `json:"required_authority"`},
		{name: "RequiresApproval", tag: `json:"requires_approval"`},
		{name: "AllowedTools", tag: `json:"allowed_tools"`},
		{name: "Model", tag: `json:"model,omitempty"`},
		{name: "ModelMetrics", tag: `json:"model_metrics,omitempty"`},
		{name: "ModelHealth", tag: `json:"model_health,omitempty"`},
		{name: "Skills", tag: `json:"skills,omitempty"`},
		{name: "Runtime", tag: `json:"runtime,omitempty"`},
		{name: "RuntimeSummary", tag: `json:"runtime_summary,omitempty"`},
		{name: "RuntimeControl", tag: `json:"runtime_control,omitempty"`},
		{name: "UpdatedAt", tag: `json:"updated_at"`},
	}

	if typ.NumField() != len(expected) {
		t.Fatalf("MissionStatusSnapshot field count = %d, want %d", typ.NumField(), len(expected))
	}
	for i, want := range expected {
		field := typ.Field(i)
		if field.Name != want.name {
			t.Fatalf("MissionStatusSnapshot field[%d].Name = %q, want %q", i, field.Name, want.name)
		}
		if string(field.Tag) != want.tag {
			t.Fatalf("MissionStatusSnapshot field[%d].Tag = %q, want %q", i, string(field.Tag), want.tag)
		}
	}
}
