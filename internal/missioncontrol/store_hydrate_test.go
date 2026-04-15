package missioncontrol

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func hydrateTestJobRuntimeRecord(now time.Time, writerEpoch, appliedSeq uint64, state JobState, activeStepID string) JobRuntimeRecord {
	record := testJobRuntimeRecord(now, writerEpoch, appliedSeq)
	record.State = state
	record.ActiveStepID = activeStepID
	return record
}

func hydrateTestStepRuntimeRecord(seq uint64, stepID string, stepType StepType, status StepRuntimeStatus, at time.Time) StepRuntimeRecord {
	record := StepRuntimeRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       seq,
		JobID:         "job-1",
		StepID:        stepID,
		StepType:      stepType,
		Status:        status,
	}
	switch status {
	case StepRuntimeStatusActive, StepRuntimeStatusPending:
		record.ActivatedAt = at.UTC()
	case StepRuntimeStatusCompleted:
		record.CompletedAt = at.UTC()
	case StepRuntimeStatusFailed:
		record.FailedAt = at.UTC()
	}
	return record
}

func hydrateTestRuntimeControlRecord(seq, writerEpoch uint64, step Step) RuntimeControlRecord {
	return RuntimeControlRecord{
		RecordVersion: StoreRecordVersion,
		WriterEpoch:   writerEpoch,
		LastSeq:       seq,
		JobID:         "job-1",
		StepID:        step.ID,
		MaxAuthority:  AuthorityTierHigh,
		AllowedTools:  append([]string(nil), step.AllowedTools...),
		Step:          copyStep(step),
	}
}

func hydrateTestActiveJobRecord(t *testing.T, writerEpoch uint64, state JobState, activeStepID string, updatedAt time.Time, activationSeq uint64) ActiveJobRecord {
	t.Helper()

	record, err := NewActiveJobRecord(
		writerEpoch,
		"job-1",
		state,
		activeStepID,
		"holder-1",
		updatedAt.Add(time.Minute),
		updatedAt,
		activationSeq,
	)
	if err != nil {
		t.Fatalf("NewActiveJobRecord() error = %v", err)
	}
	return record
}

func TestHydrateCommittedStoreStateMatchesCommittedStepOutcomes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	plan := &InspectablePlanContext{
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"exec", "reply"},
		Steps: []Step{
			{ID: "step-complete", Type: StepTypeStaticArtifact, StaticArtifactPath: "dist/report.json"},
			{ID: "step-failed", Type: StepTypeDiscussion},
			{ID: "step-active", Type: StepTypeWaitUser, AllowedTools: []string{"reply"}},
		},
	}

	jobRuntime := hydrateTestJobRuntimeRecord(now.Add(3*time.Minute), 7, 3, JobStateRunning, "step-active")
	jobRuntime.InspectablePlan = CloneInspectablePlanContext(plan)
	if err := StoreJobRuntimeRecord(root, jobRuntime); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	completed := hydrateTestStepRuntimeRecord(1, "step-complete", StepTypeStaticArtifact, StepRuntimeStatusCompleted, now.Add(time.Minute))
	completed.ResultingState = &RuntimeResultingStateRecord{
		Kind:   string(StepTypeStaticArtifact),
		Target: "dist/report.json",
		State:  "already_present",
	}
	failed := hydrateTestStepRuntimeRecord(2, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now.Add(2*time.Minute))
	failed.Reason = "validation failed"
	active := hydrateTestStepRuntimeRecord(3, "step-active", StepTypeWaitUser, StepRuntimeStatusActive, now.Add(3*time.Minute))

	for _, record := range []StepRuntimeRecord{completed, failed, active} {
		if err := StoreStepRuntimeRecord(root, record); err != nil {
			t.Fatalf("StoreStepRuntimeRecord(%q) error = %v", record.StepID, err)
		}
	}

	controlStep := Step{ID: "step-active", Type: StepTypeWaitUser, AllowedTools: []string{"reply"}}
	if err := StoreRuntimeControlRecord(root, hydrateTestRuntimeControlRecord(3, 7, controlStep)); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}
	if err := StoreActiveJobRecord(root, hydrateTestActiveJobRecord(t, 7, JobStateRunning, "step-active", now.Add(3*time.Minute), 3)); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}

	hydrated, err := hydrateCommittedStoreState(root, "job-1", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("hydrateCommittedStoreState() error = %v", err)
	}
	if hydrated.Runtime.ActiveStepID != "step-active" {
		t.Fatalf("Runtime.ActiveStepID = %q, want %q", hydrated.Runtime.ActiveStepID, "step-active")
	}
	if len(hydrated.Runtime.CompletedSteps) != 1 {
		t.Fatalf("len(Runtime.CompletedSteps) = %d, want 1", len(hydrated.Runtime.CompletedSteps))
	}
	if hydrated.Runtime.CompletedSteps[0].StepID != "step-complete" {
		t.Fatalf("Runtime.CompletedSteps[0].StepID = %q, want %q", hydrated.Runtime.CompletedSteps[0].StepID, "step-complete")
	}
	if hydrated.Runtime.CompletedSteps[0].ResultingState == nil || hydrated.Runtime.CompletedSteps[0].ResultingState.Target != "dist/report.json" {
		t.Fatalf("Runtime.CompletedSteps[0].ResultingState = %#v, want target dist/report.json", hydrated.Runtime.CompletedSteps[0].ResultingState)
	}
	if len(hydrated.Runtime.FailedSteps) != 1 {
		t.Fatalf("len(Runtime.FailedSteps) = %d, want 1", len(hydrated.Runtime.FailedSteps))
	}
	if hydrated.Runtime.FailedSteps[0].Reason != "validation failed" {
		t.Fatalf("Runtime.FailedSteps[0].Reason = %q, want %q", hydrated.Runtime.FailedSteps[0].Reason, "validation failed")
	}
	if hydrated.Control == nil || hydrated.Control.Step.ID != "step-active" {
		t.Fatalf("Control = %#v, want active step control", hydrated.Control)
	}
	if hydrated.ActiveJob == nil || hydrated.ActiveJob.ActiveStepID != "step-active" {
		t.Fatalf("ActiveJob = %#v, want active step record", hydrated.ActiveJob)
	}
}

func TestHydrateCommittedJobRuntimeStateIgnoresUncommittedChildLeftovers(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	committed := hydrateTestStepRuntimeRecord(1, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now)
	committed.Reason = "committed"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  hydrateTestJobRuntimeRecord(now, lock.WriterEpoch, 1, JobStateRunning, "step-active"),
		StepRecords: []StepRuntimeRecord{committed},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(committed) error = %v", err)
	}

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") {
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	orphan := hydrateTestStepRuntimeRecord(2, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now.Add(time.Minute))
	orphan.Reason = "orphan"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  hydrateTestJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 2, JobStateRunning, "step-active"),
		StepRecords: []StepRuntimeRecord{orphan},
	}); err == nil {
		t.Fatal("CommitStoreBatch(orphan) error = nil, want forced failure")
	}

	runtime, err := HydrateCommittedJobRuntimeState(root, "job-1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(runtime.FailedSteps) != 1 {
		t.Fatalf("len(Runtime.FailedSteps) = %d, want 1", len(runtime.FailedSteps))
	}
	if runtime.FailedSteps[0].StepID != "step-failed" {
		t.Fatalf("Runtime.FailedSteps[0].StepID = %q, want %q", runtime.FailedSteps[0].StepID, "step-failed")
	}
	if runtime.FailedSteps[0].Reason != "committed" {
		t.Fatalf("Runtime.FailedSteps[0].Reason = %q, want %q", runtime.FailedSteps[0].Reason, "committed")
	}
}

func TestHydrateCommittedJobRuntimeStateUsesCommittedAttemptWinnersOnly(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	lock := testHeldWriterLock(t, root, now)

	originalHook := storeBatchBeforeMutation
	t.Cleanup(func() { storeBatchBeforeMutation = originalHook })

	failCount := 0
	storeBatchBeforeMutation = func(path string) error {
		if path == StoreJobRuntimePath(root, "job-1") && failCount < 2 {
			failCount++
			return fmt.Errorf("forced job runtime write failure")
		}
		return nil
	}

	first := hydrateTestStepRuntimeRecord(1, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now)
	first.Reason = "first-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  hydrateTestJobRuntimeRecord(now, lock.WriterEpoch, 1, JobStateRunning, "step-active"),
		StepRecords: []StepRuntimeRecord{first},
	}); err == nil {
		t.Fatal("CommitStoreBatch(first) error = nil, want forced failure")
	}

	second := hydrateTestStepRuntimeRecord(1, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now.Add(time.Minute))
	second.Reason = "second-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  hydrateTestJobRuntimeRecord(now.Add(time.Minute), lock.WriterEpoch, 1, JobStateRunning, "step-active"),
		StepRecords: []StepRuntimeRecord{second},
	}); err == nil {
		t.Fatal("CommitStoreBatch(second) error = nil, want forced failure")
	}

	storeBatchBeforeMutation = func(string) error { return nil }
	winning := hydrateTestStepRuntimeRecord(1, "step-failed", StepTypeDiscussion, StepRuntimeStatusFailed, now.Add(2*time.Minute))
	winning.Reason = "winning-attempt"
	if err := CommitStoreBatch(root, lock, StoreBatch{
		JobRuntime:  hydrateTestJobRuntimeRecord(now.Add(2*time.Minute), lock.WriterEpoch, 1, JobStateRunning, "step-active"),
		StepRecords: []StepRuntimeRecord{winning},
	}); err != nil {
		t.Fatalf("CommitStoreBatch(winning) error = %v", err)
	}

	runtime, err := HydrateCommittedJobRuntimeState(root, "job-1", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(runtime.FailedSteps) != 1 {
		t.Fatalf("len(Runtime.FailedSteps) = %d, want 1", len(runtime.FailedSteps))
	}
	if runtime.FailedSteps[0].StepID != "step-failed" {
		t.Fatalf("Runtime.FailedSteps[0].StepID = %q, want %q", runtime.FailedSteps[0].StepID, "step-failed")
	}
	if runtime.FailedSteps[0].Reason != "winning-attempt" {
		t.Fatalf("Runtime.FailedSteps[0].Reason = %q, want %q", runtime.FailedSteps[0].Reason, "winning-attempt")
	}
}

func TestHydrateCommittedApprovalAndGrantStateMatchesRuntimeExpectations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	jobRuntime := hydrateTestJobRuntimeRecord(now.Add(2*time.Minute), 9, 2, JobStateWaitingUser, "authorize-2")
	if err := StoreJobRuntimeRecord(root, jobRuntime); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	grantedRequest := ApprovalRequestRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         1,
		RequestID:       "req-1",
		JobID:           "job-1",
		StepID:          "authorize-1",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeOneJob,
		RequestedVia:    ApprovalRequestedViaRuntime,
		GrantedVia:      ApprovalGrantedViaOperatorCommand,
		State:           ApprovalStateGranted,
		RequestedAt:     now,
		ResolvedAt:      now.Add(time.Minute),
	}
	pendingRequest := ApprovalRequestRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         2,
		RequestID:       "req-2",
		JobID:           "job-1",
		StepID:          "authorize-2",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeMissionStep,
		RequestedVia:    ApprovalRequestedViaRuntime,
		State:           ApprovalStatePending,
		RequestedAt:     now.Add(2 * time.Minute),
		ExpiresAt:       now.Add(7 * time.Minute),
	}
	grant := ApprovalGrantRecord{
		RecordVersion:   StoreRecordVersion,
		LastSeq:         1,
		GrantID:         "grant-1",
		RequestID:       "req-1",
		JobID:           "job-1",
		StepID:          "authorize-1",
		RequestedAction: ApprovalRequestedActionStepComplete,
		Scope:           ApprovalScopeOneJob,
		GrantedVia:      ApprovalGrantedViaOperatorCommand,
		State:           ApprovalStateGranted,
		GrantedAt:       now.Add(time.Minute),
		ExpiresAt:       now.Add(time.Hour),
	}

	for _, record := range []ApprovalRequestRecord{grantedRequest, pendingRequest} {
		if err := StoreApprovalRequestRecord(root, record); err != nil {
			t.Fatalf("StoreApprovalRequestRecord(%q) error = %v", record.RequestID, err)
		}
	}
	if err := StoreApprovalGrantRecord(root, grant); err != nil {
		t.Fatalf("StoreApprovalGrantRecord() error = %v", err)
	}

	runtime, err := HydrateCommittedJobRuntimeState(root, "job-1", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}

	request, ok, err := ResolveSinglePendingApprovalRequest(runtime)
	if err != nil {
		t.Fatalf("ResolveSinglePendingApprovalRequest() error = %v", err)
	}
	if !ok || request.StepID != "authorize-2" {
		t.Fatalf("ResolveSinglePendingApprovalRequest() = (%#v, %t), want pending authorize-2", request, ok)
	}

	summary := BuildOperatorStatusSummary(runtime)
	if summary.ApprovalRequest == nil || summary.ApprovalRequest.StepID != "authorize-2" {
		t.Fatalf("BuildOperatorStatusSummary().ApprovalRequest = %#v, want active-step pending request", summary.ApprovalRequest)
	}

	reusableGrant, found := FindReusableApprovalGrant(runtime, now.Add(3*time.Minute), "job-1", Step{
		ID:            "authorize-3",
		Type:          StepTypeDiscussion,
		Subtype:       StepSubtypeAuthorization,
		ApprovalScope: ApprovalScopeOneJob,
	}, "", "")
	if !found {
		t.Fatal("FindReusableApprovalGrant() found = false, want true")
	}
	if reusableGrant.GrantedVia != ApprovalGrantedViaOperatorCommand {
		t.Fatalf("FindReusableApprovalGrant().GrantedVia = %q, want %q", reusableGrant.GrantedVia, ApprovalGrantedViaOperatorCommand)
	}
}

func TestHydrateCommittedAuditHistoryOrderingIsDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, hydrateTestJobRuntimeRecord(now.Add(2*time.Minute), 5, 2, JobStateRunning, "step-active")); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	records := []AuditEventRecord{
		{
			RecordVersion: StoreRecordVersion,
			Seq:           1,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-a",
				ToolName:  "later",
				Timestamp: now.Add(2 * time.Second),
				Allowed:   true,
			},
		},
		{
			RecordVersion: StoreRecordVersion,
			Seq:           1,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-a",
				ToolName:  "earlier",
				Timestamp: now.Add(time.Second),
				Allowed:   true,
			},
		},
		{
			RecordVersion: StoreRecordVersion,
			Seq:           2,
			Event: AuditEvent{
				JobID:     "job-1",
				StepID:    "step-b",
				ToolName:  "last",
				Timestamp: now.Add(3 * time.Second),
				Allowed:   true,
			},
		},
	}
	for _, record := range records {
		if err := StoreAuditEventRecord(root, record); err != nil {
			t.Fatalf("StoreAuditEventRecord(%q) error = %v", record.Event.ToolName, err)
		}
	}

	runtime, err := HydrateCommittedJobRuntimeState(root, "job-1", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(runtime.AuditHistory) != 3 {
		t.Fatalf("len(Runtime.AuditHistory) = %d, want 3", len(runtime.AuditHistory))
	}
	if runtime.AuditHistory[0].ToolName != "earlier" || runtime.AuditHistory[1].ToolName != "later" || runtime.AuditHistory[2].ToolName != "last" {
		t.Fatalf("Runtime.AuditHistory tool order = [%q %q %q], want [earlier later last]", runtime.AuditHistory[0].ToolName, runtime.AuditHistory[1].ToolName, runtime.AuditHistory[2].ToolName)
	}
}

func TestHydrateCommittedArtifactStateIsDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreJobRuntimeRecord(root, hydrateTestJobRuntimeRecord(now.Add(2*time.Minute), 5, 2, JobStateRunning, "step-active")); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	first := ArtifactRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       2,
		ArtifactID:    "artifact-b",
		JobID:         "job-1",
		StepID:        "step-b",
		StepType:      StepTypeStaticArtifact,
		Path:          "dist/b.json",
		State:         "verified",
	}
	second := ArtifactRecord{
		RecordVersion: StoreRecordVersion,
		LastSeq:       1,
		ArtifactID:    "artifact-a",
		JobID:         "job-1",
		StepID:        "step-a",
		StepType:      StepTypeStaticArtifact,
		Path:          "dist/a.json",
		State:         "verified",
	}
	for _, record := range []ArtifactRecord{first, second} {
		if err := StoreArtifactRecord(root, record); err != nil {
			t.Fatalf("StoreArtifactRecord(%q) error = %v", record.ArtifactID, err)
		}
	}

	hydrated, err := hydrateCommittedStoreState(root, "job-1", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("hydrateCommittedStoreState() error = %v", err)
	}
	if len(hydrated.Artifacts) != 2 {
		t.Fatalf("len(Artifacts) = %d, want 2", len(hydrated.Artifacts))
	}
	if hydrated.Artifacts[0].ArtifactID != "artifact-a" || hydrated.Artifacts[1].ArtifactID != "artifact-b" {
		t.Fatalf("Artifacts order = [%q %q], want [artifact-a artifact-b]", hydrated.Artifacts[0].ArtifactID, hydrated.Artifacts[1].ArtifactID)
	}
}

func TestHydrateCommittedJobRuntimeStateLoadsFrankZohoSendReceiptsAppendOnly(t *testing.T) {
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

	hydrated, err := HydrateCommittedJobRuntimeState(root, job.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(hydrated.FrankZohoSendReceipts) != 2 {
		t.Fatalf("len(Runtime.FrankZohoSendReceipts) = %d, want 2", len(hydrated.FrankZohoSendReceipts))
	}

	first := hydrated.FrankZohoSendReceipts[0]
	if first.ProviderMessageID != receiptOne.ProviderMessageID {
		t.Fatalf("first.ProviderMessageID = %q, want canonical provider message id %q", first.ProviderMessageID, receiptOne.ProviderMessageID)
	}
	if first.ProviderMailID != receiptOne.ProviderMailID {
		t.Fatalf("first.ProviderMailID = %q, want secondary provider mail id %q", first.ProviderMailID, receiptOne.ProviderMailID)
	}
	if first.MIMEMessageID != receiptOne.MIMEMessageID {
		t.Fatalf("first.MIMEMessageID = %q, want secondary MIME message id %q", first.MIMEMessageID, receiptOne.MIMEMessageID)
	}
	if first.ProviderAccountID != receiptOne.ProviderAccountID {
		t.Fatalf("first.ProviderAccountID = %q, want proof locator account id %q", first.ProviderAccountID, receiptOne.ProviderAccountID)
	}
	if first.OriginalMessageURL != receiptOne.OriginalMessageURL {
		t.Fatalf("first.OriginalMessageURL = %q, want proof-compatible originalmessage URL %q", first.OriginalMessageURL, receiptOne.OriginalMessageURL)
	}

	second := hydrated.FrankZohoSendReceipts[1]
	if second.ProviderMessageID != receiptTwo.ProviderMessageID {
		t.Fatalf("second.ProviderMessageID = %q, want canonical provider message id %q", second.ProviderMessageID, receiptTwo.ProviderMessageID)
	}
	if second.ProviderMailID != receiptTwo.ProviderMailID {
		t.Fatalf("second.ProviderMailID = %q, want secondary provider mail id %q", second.ProviderMailID, receiptTwo.ProviderMailID)
	}
	if second.MIMEMessageID != receiptTwo.MIMEMessageID {
		t.Fatalf("second.MIMEMessageID = %q, want secondary MIME message id %q", second.MIMEMessageID, receiptTwo.MIMEMessageID)
	}
	if second.ProviderAccountID != receiptTwo.ProviderAccountID {
		t.Fatalf("second.ProviderAccountID = %q, want proof locator account id %q", second.ProviderAccountID, receiptTwo.ProviderAccountID)
	}
	if second.OriginalMessageURL != receiptTwo.OriginalMessageURL {
		t.Fatalf("second.OriginalMessageURL = %q, want proof-compatible originalmessage URL %q", second.OriginalMessageURL, receiptTwo.OriginalMessageURL)
	}
}

func TestHydrateCommittedJobRuntimeStateLoadsFrankZohoInboundRepliesAppendOnly(t *testing.T) {
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

	replyOne := NormalizeFrankZohoInboundReply(FrankZohoInboundReply{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880101000",
		ProviderMailID:     "<reply-1@zoho.test>",
		MIMEMessageID:      "<inbound-1@example.test>",
		InReplyTo:          "<mime-1@example.test>",
		References:         []string{"<seed@example.test>", "<mime-1@example.test>"},
		FromAddress:        "person@example.com",
		FromDisplayName:    "Person One",
		Subject:            "Re: Frank intro",
		ReceivedAt:         now.Add(-40 * time.Second),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880101000/originalmessage",
	})
	replyOne.ReplyID = normalizedFrankZohoInboundReplyID(replyOne)
	replyTwo := NormalizeFrankZohoInboundReply(FrankZohoInboundReply{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880101001",
		ProviderMailID:     "<reply-2@zoho.test>",
		MIMEMessageID:      "<inbound-2@example.test>",
		InReplyTo:          "<mime-2@example.test>",
		References:         []string{"<mime-2@example.test>"},
		FromAddress:        "second@example.com",
		FromDisplayName:    "Person Two",
		Subject:            "Re: Another thread",
		ReceivedAt:         now.Add(-10 * time.Second),
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880101001/originalmessage",
	})
	replyTwo.ReplyID = normalizedFrankZohoInboundReplyID(replyTwo)

	runtime := JobRuntimeState{
		JobID:                   job.ID,
		State:                   JobStateRunning,
		ActiveStepID:            "build",
		InspectablePlan:         &inspectablePlan,
		FrankZohoInboundReplies: []FrankZohoInboundReply{replyOne},
		CreatedAt:               now.Add(-2 * time.Minute),
		UpdatedAt:               now,
		StartedAt:               now.Add(-2 * time.Minute),
		ActiveStepAt:            now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(first) error = %v", err)
	}

	runtime.FrankZohoInboundReplies = append(runtime.FrankZohoInboundReplies, replyTwo)
	runtime.UpdatedAt = now.Add(time.Minute)
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now.Add(time.Minute)); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(second) error = %v", err)
	}

	hydrated, err := HydrateCommittedJobRuntimeState(root, job.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(hydrated.FrankZohoInboundReplies) != 2 {
		t.Fatalf("len(Runtime.FrankZohoInboundReplies) = %d, want 2", len(hydrated.FrankZohoInboundReplies))
	}

	first := hydrated.FrankZohoInboundReplies[0]
	if first.ProviderMessageID != replyOne.ProviderMessageID {
		t.Fatalf("first.ProviderMessageID = %q, want %q", first.ProviderMessageID, replyOne.ProviderMessageID)
	}
	if first.InReplyTo != replyOne.InReplyTo {
		t.Fatalf("first.InReplyTo = %q, want %q", first.InReplyTo, replyOne.InReplyTo)
	}
	if len(first.References) != 2 || first.References[0] != "<seed@example.test>" {
		t.Fatalf("first.References = %#v, want preserved reference chain", first.References)
	}

	second := hydrated.FrankZohoInboundReplies[1]
	if second.ProviderMessageID != replyTwo.ProviderMessageID {
		t.Fatalf("second.ProviderMessageID = %q, want %q", second.ProviderMessageID, replyTwo.ProviderMessageID)
	}
	if second.FromAddress != replyTwo.FromAddress {
		t.Fatalf("second.FromAddress = %q, want %q", second.FromAddress, replyTwo.FromAddress)
	}
	if second.OriginalMessageURL != replyTwo.OriginalMessageURL {
		t.Fatalf("second.OriginalMessageURL = %q, want %q", second.OriginalMessageURL, replyTwo.OriginalMessageURL)
	}
}

func TestHydrateCommittedJobRuntimeStateLoadsCampaignZohoEmailOutboundActionsLatestState(t *testing.T) {
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

	prepared, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{
			To: []string{"alice@example.com"},
		},
		"Frank intro",
		"plaintext",
		"Hello from Frank",
		now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	sent, err := BuildCampaignZohoEmailOutboundSentAction(prepared, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:                            job.ID,
		State:                            JobStateRunning,
		ActiveStepID:                     "build",
		InspectablePlan:                  &inspectablePlan,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{prepared},
		CreatedAt:                        now.Add(-2 * time.Minute),
		UpdatedAt:                        now,
		StartedAt:                        now.Add(-2 * time.Minute),
		ActiveStepAt:                     now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(prepared) error = %v", err)
	}

	runtime.CampaignZohoEmailOutboundActions = []CampaignZohoEmailOutboundAction{sent}
	runtime.UpdatedAt = now.Add(time.Minute)
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now.Add(time.Minute)); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(sent) error = %v", err)
	}

	hydrated, err := HydrateCommittedJobRuntimeState(root, job.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(hydrated.CampaignZohoEmailOutboundActions) != 1 {
		t.Fatalf("len(Runtime.CampaignZohoEmailOutboundActions) = %d, want 1", len(hydrated.CampaignZohoEmailOutboundActions))
	}

	action := hydrated.CampaignZohoEmailOutboundActions[0]
	if action.ActionID != prepared.ActionID {
		t.Fatalf("ActionID = %q, want %q", action.ActionID, prepared.ActionID)
	}
	if action.State != CampaignZohoEmailOutboundActionStateSent {
		t.Fatalf("State = %q, want sent", action.State)
	}
	if action.CampaignID != "campaign-mail" {
		t.Fatalf("CampaignID = %q, want campaign-mail", action.CampaignID)
	}
	if action.Addressing.To[0] != "alice@example.com" {
		t.Fatalf("Addressing.To[0] = %q, want alice@example.com", action.Addressing.To[0])
	}
	if action.ProviderMessageID != "1711540357880100000" {
		t.Fatalf("ProviderMessageID = %q, want canonical Zoho message id", action.ProviderMessageID)
	}
	if action.OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("OriginalMessageURL = %q, want proof-compatible originalmessage URL", action.OriginalMessageURL)
	}
}

func TestHydrateCommittedJobRuntimeStateLoadsCampaignZohoEmailReplyWorkItemsLatestState(t *testing.T) {
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

	openItem, err := BuildCampaignZohoEmailReplyWorkItemOpen("campaign-mail", "frank_zoho_inbound_reply_123", now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemOpen() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:                           job.ID,
		State:                           JobStateRunning,
		ActiveStepID:                    "build",
		InspectablePlan:                 &inspectablePlan,
		CampaignZohoEmailReplyWorkItems: []CampaignZohoEmailReplyWorkItem{openItem},
		CreatedAt:                       now.Add(-2 * time.Minute),
		UpdatedAt:                       now,
		StartedAt:                       now.Add(-2 * time.Minute),
		ActiveStepAt:                    now.Add(-time.Minute),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(open) error = %v", err)
	}

	deferredItem, err := BuildCampaignZohoEmailReplyWorkItemDeferred(openItem, now.Add(10*time.Minute), now)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailReplyWorkItemDeferred() error = %v", err)
	}
	runtime.CampaignZohoEmailReplyWorkItems = []CampaignZohoEmailReplyWorkItem{deferredItem}
	runtime.UpdatedAt = now.Add(time.Minute)
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now.Add(time.Minute)); err != nil {
		t.Fatalf("PersistProjectedRuntimeState(deferred) error = %v", err)
	}

	hydrated, err := HydrateCommittedJobRuntimeState(root, job.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedJobRuntimeState() error = %v", err)
	}
	if len(hydrated.CampaignZohoEmailReplyWorkItems) != 1 {
		t.Fatalf("len(Runtime.CampaignZohoEmailReplyWorkItems) = %d, want 1", len(hydrated.CampaignZohoEmailReplyWorkItems))
	}

	got := hydrated.CampaignZohoEmailReplyWorkItems[0]
	if got.ReplyWorkItemID != openItem.ReplyWorkItemID {
		t.Fatalf("ReplyWorkItemID = %q, want stable work item id %q", got.ReplyWorkItemID, openItem.ReplyWorkItemID)
	}
	if got.State != CampaignZohoEmailReplyWorkItemStateDeferred {
		t.Fatalf("State = %q, want deferred", got.State)
	}
	if got.DeferredUntil.IsZero() {
		t.Fatal("DeferredUntil = zero, want hydrated defer timestamp")
	}
}

func TestHydrateCommittedRuntimeControlMatchesActiveStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	jobRuntime := hydrateTestJobRuntimeRecord(now, 6, 1, JobStateRunning, "step-active")
	if err := StoreJobRuntimeRecord(root, jobRuntime); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	controlStep := Step{ID: "step-active", Type: StepTypeDiscussion, AllowedTools: []string{"exec"}}
	if err := StoreRuntimeControlRecord(root, hydrateTestRuntimeControlRecord(1, 6, controlStep)); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}

	control, err := HydrateCommittedRuntimeControlContext(root, "job-1", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("HydrateCommittedRuntimeControlContext() error = %v", err)
	}
	if control == nil || control.Step.ID != "step-active" {
		t.Fatalf("Control = %#v, want active-step control", control)
	}
}

func TestHydrateCommittedNonOccupancyJobDoesNotExposeCommittedActiveJob(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if err := StoreActiveJobRecord(root, hydrateTestActiveJobRecord(t, 8, JobStateRunning, "step-active", now, 1)); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}
	if err := StoreRuntimeControlRecord(root, hydrateTestRuntimeControlRecord(1, 8, Step{ID: "step-active", Type: StepTypeDiscussion, AllowedTools: []string{"exec"}})); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}

	jobRuntime := hydrateTestJobRuntimeRecord(now.Add(time.Minute), 8, 2, JobStateCompleted, "")
	if err := StoreJobRuntimeRecord(root, jobRuntime); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}

	hydrated, err := hydrateCommittedStoreState(root, "job-1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("hydrateCommittedStoreState() error = %v", err)
	}
	if hydrated.ActiveJob != nil {
		t.Fatalf("ActiveJob = %#v, want nil", hydrated.ActiveJob)
	}
	if hydrated.Control != nil {
		t.Fatalf("Control = %#v, want nil", hydrated.Control)
	}
	if hydrated.Runtime.ActiveStepID != "" {
		t.Fatalf("Runtime.ActiveStepID = %q, want empty", hydrated.Runtime.ActiveStepID)
	}

	_, err = LoadCommittedActiveJobRecord(root, "job-1")
	if !errors.Is(err, ErrActiveJobRecordNotFound) {
		t.Fatalf("LoadCommittedActiveJobRecord() error = %v, want %v", err, ErrActiveJobRecordNotFound)
	}
}
