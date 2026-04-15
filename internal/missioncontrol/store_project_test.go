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

func TestBuildCommittedMissionStatusSnapshotDeterministic(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{ID: "gamma", Type: StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"zeta"}},
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
	now := time.Now().UTC().Truncate(time.Second)
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

func TestBuildCommittedMissionStatusSnapshotIncludesFrankZohoSendProofFromCommittedRuntime(t *testing.T) {
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
	if len(snapshot.RuntimeSummary.CampaignZohoEmailOutbounds) != 1 {
		t.Fatalf("RuntimeSummary.CampaignZohoEmailOutbounds len = %d, want 1", len(snapshot.RuntimeSummary.CampaignZohoEmailOutbounds))
	}
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
	if len(snapshot.RuntimeSummary.FrankZohoSendProof) != 1 {
		t.Fatalf("RuntimeSummary.FrankZohoSendProof len = %d, want 1", len(snapshot.RuntimeSummary.FrankZohoSendProof))
	}
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
