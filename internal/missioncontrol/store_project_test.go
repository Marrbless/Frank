package missioncontrol

import (
	"encoding/json"
	"reflect"
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
	runtime := JobRuntimeState{
		JobID:            job.ID,
		State:            JobStatePaused,
		ActiveStepID:     "final",
		InspectablePlan:  &plan,
		PausedReason:     RuntimePauseReasonOperatorCommand,
		PausedAt:         time.Date(2026, 3, 24, 13, 30, 0, 0, time.UTC),
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
	now := time.Date(2026, 4, 5, 20, 0, 0, 0, time.UTC)
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
	if first.RuntimeSummary.RecentAudit[0].Timestamp != "2026-03-24T13:05:00Z" {
		t.Fatalf("RecentAudit[0].Timestamp = %q, want %q", first.RuntimeSummary.RecentAudit[0].Timestamp, "2026-03-24T13:05:00Z")
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
	now := time.Date(2026, 4, 5, 20, 15, 0, 0, time.UTC)
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
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, running, &control, now.Add(-time.Minute)); err != nil {
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
