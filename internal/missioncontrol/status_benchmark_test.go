package missioncontrol

import (
	"fmt"
	"testing"
	"time"
)

var benchmarkOperatorStatusSummarySink OperatorStatusSummary
var benchmarkMissionStatusSnapshotSink MissionStatusSnapshot
var benchmarkCommittedStoreStateSink hydratedCommittedStoreState

func BenchmarkBuildOperatorStatusSummaryLargeRuntime(b *testing.B) {
	now := time.Date(2026, 4, 30, 14, 0, 0, 0, time.UTC)
	runtime := benchmarkLargeOperatorStatusRuntime(now)
	allowedTools := []string{"read", "reply", "write_file"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkOperatorStatusSummarySink = BuildOperatorStatusSummaryWithAllowedTools(runtime, allowedTools)
	}
}

func BenchmarkBuildCommittedMissionStatusSnapshotProjectedRuntime(b *testing.B) {
	storeFixture := writeMissionStoreRuntimeFixture(b)
	opts := MissionStatusSnapshotOptions{
		MissionRequired: true,
		MissionFile:     "mission.json",
		UpdatedAt:       storeFixture.now,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot, err := BuildCommittedMissionStatusSnapshot(storeFixture.root, storeFixture.job.ID, opts)
		if err != nil {
			b.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
		}
		benchmarkMissionStatusSnapshotSink = snapshot
	}
}

func BenchmarkHydrateCommittedStoreStateLargeFixture(b *testing.B) {
	root, jobID, now := writeBenchmarkLargeCommittedStoreFixture(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state, err := hydrateCommittedStoreState(root, jobID, now)
		if err != nil {
			b.Fatalf("hydrateCommittedStoreState() error = %v", err)
		}
		benchmarkCommittedStoreStateSink = state
	}
}

func benchmarkLargeOperatorStatusRuntime(now time.Time) JobRuntimeState {
	runtime := JobRuntimeState{
		JobID:          "job-status-benchmark",
		State:          JobStateRunning,
		ActiveStepID:   "step-active",
		CreatedAt:      now.Add(-2 * time.Hour),
		UpdatedAt:      now,
		StartedAt:      now.Add(-2 * time.Hour),
		ActiveStepAt:   now.Add(-time.Minute),
		ExecutionPlane: ExecutionPlaneLiveRuntime,
		ExecutionHost:  ExecutionHostPhone,
		MissionFamily:  MissionFamilyBootstrapRevenue,
	}

	for i := 0; i < 128; i++ {
		stepID := fmt.Sprintf("artifact-%03d", i)
		at := now.Add(time.Duration(i-128) * time.Second)
		runtime.CompletedSteps = append(runtime.CompletedSteps, RuntimeStepRecord{
			StepID: stepID,
			At:     at,
			ResultingState: &RuntimeResultingStateRecord{
				Kind:   string(StepTypeStaticArtifact),
				Target: fmt.Sprintf("dist/report-%03d.json", i),
				State:  "already_present",
			},
		})
		runtime.AuditHistory = append(runtime.AuditHistory, AuditEvent{
			EventID:     fmt.Sprintf("audit-%03d", i),
			JobID:       runtime.JobID,
			StepID:      stepID,
			ToolName:    "read",
			ActionClass: AuditActionClassToolCall,
			Result:      AuditResultApplied,
			Allowed:     true,
			Timestamp:   at,
		})
		runtime.ApprovalRequests = append(runtime.ApprovalRequests, ApprovalRequest{
			JobID:           runtime.JobID,
			StepID:          stepID,
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			State:           ApprovalStateGranted,
			SessionChannel:  "telegram",
			SessionChatID:   fmt.Sprintf("chat-%03d", i),
			RequestedAt:     at.Add(-time.Minute),
			ResolvedAt:      at,
		})
		runtime.ApprovalGrants = append(runtime.ApprovalGrants, ApprovalGrant{
			JobID:           runtime.JobID,
			StepID:          stepID,
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			GrantedVia:      ApprovalGrantedViaOperatorCommand,
			State:           ApprovalStateGranted,
			SessionChannel:  "telegram",
			SessionChatID:   fmt.Sprintf("chat-%03d", i),
			GrantedAt:       at,
		})
	}

	return runtime
}

func writeBenchmarkLargeCommittedStoreFixture(t missionStoreRuntimeFixtureTB) (string, string, time.Time) {
	t.Helper()

	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	runtime := benchmarkLargeOperatorStatusRuntime(now)
	control := RuntimeControlContext{
		JobID:        runtime.JobID,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "reply", "write_file"},
		Step: Step{
			ID:                runtime.ActiveStepID,
			Type:              StepTypeOneShotCode,
			RequiredAuthority: AuthorityTierLow,
			AllowedTools:      []string{"read"},
		},
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "benchmark"}, nil, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}
	return root, runtime.JobID, now
}
