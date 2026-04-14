package missioncontrol

import (
	"testing"
	"time"
)

func testProjectedRuntimeJob() Job {
	return Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "reply"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                   "artifact",
					Type:                 StepTypeStaticArtifact,
					StaticArtifactPath:   "dist/report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
}

func TestPersistProjectedRuntimeStateWritesCommittedRuntimeFamilies(t *testing.T) {
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
	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateRunning,
		ActiveStepID:    "build",
		InspectablePlan: &inspectablePlan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		ActiveStepAt:    now.Add(-time.Minute),
		CompletedSteps: []RuntimeStepRecord{
			{
				StepID: "artifact",
				At:     now.Add(-90 * time.Second),
				ResultingState: &RuntimeResultingStateRecord{
					Kind:   string(StepTypeStaticArtifact),
					Target: "dist/report.json",
					State:  "already_present",
				},
			},
		},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				RequestedVia:    ApprovalRequestedViaRuntime,
				State:           ApprovalStateGranted,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				RequestedAt:     now.Add(-70 * time.Second),
				ResolvedAt:      now.Add(-65 * time.Second),
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateGranted,
				GrantedAt:       now.Add(-65 * time.Second),
			},
		},
		AuditHistory: []AuditEvent{
			{
				JobID:       job.ID,
				StepID:      "build",
				ToolName:    "pause",
				ActionClass: AuditActionClassOperatorCommand,
				Result:      AuditResultApplied,
				Allowed:     true,
				Timestamp:   now.Add(-30 * time.Second),
			},
		},
	}

	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, job.ID)
	if err != nil {
		t.Fatalf("LoadCommittedJobRuntimeRecord() error = %v", err)
	}
	if jobRuntime.AppliedSeq != 1 {
		t.Fatalf("LoadCommittedJobRuntimeRecord().AppliedSeq = %d, want 1", jobRuntime.AppliedSeq)
	}

	steps, err := ListCommittedStepRuntimeRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedStepRuntimeRecords() error = %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("ListCommittedStepRuntimeRecords() len = %d, want 3", len(steps))
	}
	if steps[0].StepID != "artifact" || steps[0].Status != StepRuntimeStatusCompleted {
		t.Fatalf("Step[0] = %#v, want completed artifact step", steps[0])
	}
	if steps[1].StepID != "build" || steps[1].Status != StepRuntimeStatusActive {
		t.Fatalf("Step[1] = %#v, want active build step", steps[1])
	}
	if steps[2].StepID != "final" || steps[2].Status != StepRuntimeStatusPending {
		t.Fatalf("Step[2] = %#v, want pending final step", steps[2])
	}

	controlRecord, err := LoadCommittedRuntimeControlRecord(root, job.ID)
	if err != nil {
		t.Fatalf("LoadCommittedRuntimeControlRecord() error = %v", err)
	}
	if controlRecord.StepID != "build" {
		t.Fatalf("LoadCommittedRuntimeControlRecord().StepID = %q, want %q", controlRecord.StepID, "build")
	}

	requests, err := ListCommittedApprovalRequestRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedApprovalRequestRecords() error = %v", err)
	}
	if len(requests) != 1 || requests[0].State != ApprovalStateGranted {
		t.Fatalf("ListCommittedApprovalRequestRecords() = %#v, want one granted request", requests)
	}

	grants, err := ListCommittedApprovalGrantRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedApprovalGrantRecords() error = %v", err)
	}
	if len(grants) != 1 || grants[0].State != ApprovalStateGranted {
		t.Fatalf("ListCommittedApprovalGrantRecords() = %#v, want one granted grant", grants)
	}

	artifacts, err := ListCommittedArtifactRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedArtifactRecords() error = %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].Path != "dist/report.json" {
		t.Fatalf("ListCommittedArtifactRecords() = %#v, want dist/report.json artifact", artifacts)
	}

	activeJob, err := LoadCommittedActiveJobRecord(root, job.ID)
	if err != nil {
		t.Fatalf("LoadCommittedActiveJobRecord() error = %v", err)
	}
	if activeJob.ActiveStepID != "build" {
		t.Fatalf("LoadCommittedActiveJobRecord().ActiveStepID = %q, want %q", activeJob.ActiveStepID, "build")
	}

	audits, err := ListCommittedAuditEventRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(audits) != 1 || audits[0].Event.ToolName != "pause" {
		t.Fatalf("ListCommittedAuditEventRecords() = %#v, want one pause audit", audits)
	}
}

func TestPersistProjectedRuntimeStateDoesNotDuplicateCommittedAuditDelta(t *testing.T) {
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
	audit := normalizeAuditEvent(AuditEvent{
		JobID:       job.ID,
		StepID:      "build",
		ToolName:    "resume_ack",
		ActionClass: AuditActionClassRuntime,
		Result:      AuditResultApplied,
		Allowed:     true,
		Timestamp:   now.Add(-10 * time.Second),
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
		AuditHistory:    []AuditEvent{audit},
	}

	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(first) error = %v", err)
	}

	runtime.UpdatedAt = now.Add(time.Minute)
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now.Add(time.Minute)); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(second) error = %v", err)
	}

	jobRuntime, err := LoadCommittedJobRuntimeRecord(root, job.ID)
	if err != nil {
		t.Fatalf("LoadCommittedJobRuntimeRecord() error = %v", err)
	}
	if jobRuntime.AppliedSeq != 2 {
		t.Fatalf("LoadCommittedJobRuntimeRecord().AppliedSeq = %d, want 2", jobRuntime.AppliedSeq)
	}

	audits, err := ListCommittedAuditEventRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedAuditEventRecords() error = %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("ListCommittedAuditEventRecords() len = %d, want 1", len(audits))
	}
	if audits[0].Event.EventID != audit.EventID {
		t.Fatalf("ListCommittedAuditEventRecords()[0].Event.EventID = %q, want %q", audits[0].Event.EventID, audit.EventID)
	}
}

func TestPersistProjectedRuntimeStateNormalizesBlankArtifactState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	job := testProjectedRuntimeJob()
	inspectablePlan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:           job.ID,
		State:           JobStateCompleted,
		InspectablePlan: &inspectablePlan,
		CreatedAt:       now.Add(-2 * time.Minute),
		UpdatedAt:       now,
		StartedAt:       now.Add(-2 * time.Minute),
		CompletedAt:     now,
		CompletedSteps: []RuntimeStepRecord{
			{
				StepID: "artifact",
				At:     now.Add(-75 * time.Second),
				ResultingState: &RuntimeResultingStateRecord{
					Kind:   string(StepTypeStaticArtifact),
					Target: "dist/report.json",
					State:  "   ",
				},
			},
		},
	}

	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, nil, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	artifacts, err := ListCommittedArtifactRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedArtifactRecords() error = %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("ListCommittedArtifactRecords() len = %d, want 1", len(artifacts))
	}
	if artifacts[0].State != "verified" {
		t.Fatalf("ListCommittedArtifactRecords()[0].State = %q, want %q", artifacts[0].State, "verified")
	}
}

func TestPersistProjectedRuntimeStateProjectsFrankZohoSendReceiptsAppendOnly(t *testing.T) {
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

	receiptOne := FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}
	receiptTwo := FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100001",
		ProviderMailID:     "<mail-2@zoho.test>",
		MIMEMessageID:      "<mime-2@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100001/originalmessage",
	}

	runtime := JobRuntimeState{
		JobID:                 job.ID,
		State:                 JobStateRunning,
		ActiveStepID:          "build",
		InspectablePlan:       &inspectablePlan,
		FrankZohoSendReceipts: []FrankZohoSendReceipt{receiptOne},
		CreatedAt:             now.Add(-2 * time.Minute),
		UpdatedAt:             now,
		StartedAt:             now.Add(-2 * time.Minute),
		ActiveStepAt:          now.Add(-time.Minute),
	}

	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(first) error = %v", err)
	}

	runtime.FrankZohoSendReceipts = append(runtime.FrankZohoSendReceipts, receiptTwo)
	runtime.UpdatedAt = now.Add(time.Minute)
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now.Add(time.Minute)); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(second) error = %v", err)
	}

	records, err := ListCommittedFrankZohoSendReceiptRecords(root, job.ID)
	if err != nil {
		t.Fatalf("ListCommittedFrankZohoSendReceiptRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListCommittedFrankZohoSendReceiptRecords() len = %d, want 2", len(records))
	}
	if records[0].ProviderMessageID != receiptOne.ProviderMessageID {
		t.Fatalf("records[0].ProviderMessageID = %q, want %q", records[0].ProviderMessageID, receiptOne.ProviderMessageID)
	}
	if records[0].ProviderMailID != receiptOne.ProviderMailID {
		t.Fatalf("records[0].ProviderMailID = %q, want %q", records[0].ProviderMailID, receiptOne.ProviderMailID)
	}
	if records[0].MIMEMessageID != receiptOne.MIMEMessageID {
		t.Fatalf("records[0].MIMEMessageID = %q, want %q", records[0].MIMEMessageID, receiptOne.MIMEMessageID)
	}
	if records[0].OriginalMessageURL != receiptOne.OriginalMessageURL {
		t.Fatalf("records[0].OriginalMessageURL = %q, want %q", records[0].OriginalMessageURL, receiptOne.OriginalMessageURL)
	}
	if records[1].ProviderMessageID != receiptTwo.ProviderMessageID {
		t.Fatalf("records[1].ProviderMessageID = %q, want %q", records[1].ProviderMessageID, receiptTwo.ProviderMessageID)
	}
	if records[1].OriginalMessageURL != receiptTwo.OriginalMessageURL {
		t.Fatalf("records[1].OriginalMessageURL = %q, want %q", records[1].OriginalMessageURL, receiptTwo.OriginalMessageURL)
	}
}
