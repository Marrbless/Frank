package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestRunDeterministicImprovementWorkspacePhoneProfileUsesFakeLocalCapabilities(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)
	beforePointer := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))

	record, assessment, changed, err := RunDeterministicImprovementWorkspace(root, DeterministicWorkspaceRunRequest{
		RunID:       "run-root",
		ProfileName: WorkspaceRunnerProfilePhone,
		Capabilities: WorkspaceRunnerHostCapabilities{
			HostProfile:               "dev-host",
			LocalWorkspaceAvailable:   true,
			FakePhoneProfileAvailable: true,
			NetworkDisabled:           true,
			ExternalServicesDisabled:  true,
		},
		StartedAt: now.Add(10 * time.Minute),
		CreatedBy: "autonomy-loop",
	})
	if err != nil {
		t.Fatalf("RunDeterministicImprovementWorkspace(phone) error = %v", err)
	}
	if !changed {
		t.Fatal("RunDeterministicImprovementWorkspace(phone) changed = false, want true")
	}
	if !assessment.Ready || assessment.ExecutionHost != ExecutionHostPhone {
		t.Fatalf("assessment = %#v, want ready phone host", assessment)
	}
	if record.ExecutionHost != ExecutionHostPhone || record.Outcome != ImprovementWorkspaceRunOutcomeSucceeded {
		t.Fatalf("record host/outcome = %q/%q, want phone/succeeded", record.ExecutionHost, record.Outcome)
	}
	if record.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty on success", record.FailureReason)
	}
	assertBytesEqual(t, "active runtime-pack pointer after phone runner", beforePointer, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))

	replayed, replayAssessment, replayChanged, err := RunDeterministicImprovementWorkspace(root, DeterministicWorkspaceRunRequest{
		RunID:       "run-root",
		ProfileName: WorkspaceRunnerProfilePhone,
		Capabilities: WorkspaceRunnerHostCapabilities{
			LocalWorkspaceAvailable:   true,
			FakePhoneProfileAvailable: true,
			NetworkDisabled:           true,
			ExternalServicesDisabled:  true,
		},
		StartedAt: now.Add(10 * time.Minute),
		CreatedBy: "autonomy-loop",
	})
	if err != nil {
		t.Fatalf("RunDeterministicImprovementWorkspace(phone replay) error = %v", err)
	}
	if !replayAssessment.Ready || replayChanged || replayed.WorkspaceRunID != record.WorkspaceRunID {
		t.Fatalf("replay = %#v assessment=%#v changed=%v, want idempotent phone run %#v", replayed, replayAssessment, replayChanged, record)
	}
}

func TestAssessWorkspaceRunnerProfileRejectsUnsupportedOrUnsafeCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		profileName  string
		capabilities WorkspaceRunnerHostCapabilities
		want         string
	}{
		{
			name:        "unsupported",
			profileName: "remote",
			capabilities: WorkspaceRunnerHostCapabilities{
				LocalWorkspaceAvailable:   true,
				FakePhoneProfileAvailable: true,
				NetworkDisabled:           true,
				ExternalServicesDisabled:  true,
			},
			want: "unsupported",
		},
		{
			name:        "phone without fake profile",
			profileName: WorkspaceRunnerProfilePhone,
			capabilities: WorkspaceRunnerHostCapabilities{
				LocalWorkspaceAvailable:  true,
				NetworkDisabled:          true,
				ExternalServicesDisabled: true,
			},
			want: "fake_phone_profile_available",
		},
		{
			name:        "network enabled",
			profileName: WorkspaceRunnerProfilePhone,
			capabilities: WorkspaceRunnerHostCapabilities{
				LocalWorkspaceAvailable:   true,
				FakePhoneProfileAvailable: true,
				ExternalServicesDisabled:  true,
			},
			want: "network_disabled",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := AssessWorkspaceRunnerProfile(tc.profileName, tc.capabilities)
			if got.Ready {
				t.Fatalf("Ready = true, want false for %s", tc.name)
			}
			if !strings.Contains(strings.Join(got.Blockers, "; "), tc.want) {
				t.Fatalf("Blockers = %#v, want %q", got.Blockers, tc.want)
			}
		})
	}
}

func TestRunDeterministicImprovementWorkspaceDesktopDevProfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 19, 11, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(7*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-desktop-dev"
		record.ExecutionHost = ExecutionHostDesktopDev
		record.State = ImprovementRunStateMutating
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord(desktop_dev) error = %v", err)
	}

	record, assessment, _, err := RunDeterministicImprovementWorkspace(root, DeterministicWorkspaceRunRequest{
		RunID:       "run-desktop-dev",
		ProfileName: WorkspaceRunnerProfileDesktopDev,
		Capabilities: WorkspaceRunnerHostCapabilities{
			LocalWorkspaceAvailable:  true,
			NetworkDisabled:          true,
			ExternalServicesDisabled: true,
		},
		StartedAt: now.Add(12 * time.Minute),
		CreatedBy: "autonomy-loop",
	})
	if err != nil {
		t.Fatalf("RunDeterministicImprovementWorkspace(desktop_dev) error = %v", err)
	}
	if !assessment.Ready || assessment.ExecutionHost != ExecutionHostDesktopDev {
		t.Fatalf("assessment = %#v, want ready desktop_dev host", assessment)
	}
	if record.ExecutionHost != ExecutionHostDesktopDev || record.Outcome != ImprovementWorkspaceRunOutcomeSucceeded {
		t.Fatalf("record host/outcome = %q/%q, want desktop_dev/succeeded", record.ExecutionHost, record.Outcome)
	}
}

func TestRunDeterministicImprovementWorkspaceRejectsHostMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	storeImprovementWorkspaceRunFixtures(t, root, now)

	_, assessment, _, err := RunDeterministicImprovementWorkspace(root, DeterministicWorkspaceRunRequest{
		RunID:       "run-root",
		ProfileName: WorkspaceRunnerProfileDesktopDev,
		Capabilities: WorkspaceRunnerHostCapabilities{
			LocalWorkspaceAvailable:  true,
			NetworkDisabled:          true,
			ExternalServicesDisabled: true,
		},
		StartedAt: now.Add(10 * time.Minute),
		CreatedBy: "autonomy-loop",
	})
	if err == nil {
		t.Fatal("RunDeterministicImprovementWorkspace() error = nil, want host mismatch rejection")
	}
	if !assessment.Ready || !strings.Contains(err.Error(), "does not match workspace runner profile host") {
		t.Fatalf("assessment=%#v error=%q, want ready profile plus host mismatch", assessment, err.Error())
	}
}
