package missioncontrol

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestAssessHotUpdateExecutionReadinessClassifiesTransitions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name              string
		transition        HotUpdateExecutionTransition
		wantClass         HotUpdateExecutionTransitionClass
		wantSensitive     bool
		wantReplay        HotUpdateExecutionReplayClass
		wantReady         bool
		wantRejectionCode RejectionCode
	}{
		{
			name:          "prepared gate create metadata",
			transition:    HotUpdateExecutionTransitionPreparedGateCreate,
			wantClass:     HotUpdateExecutionTransitionClassMetadata,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "phase validated metadata",
			transition:    HotUpdateExecutionTransitionPhaseValidated,
			wantClass:     HotUpdateExecutionTransitionClassMetadata,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "phase staged metadata",
			transition:    HotUpdateExecutionTransitionPhaseStaged,
			wantClass:     HotUpdateExecutionTransitionClassMetadata,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "pointer switch execution",
			transition:    HotUpdateExecutionTransitionPointerSwitch,
			wantClass:     HotUpdateExecutionTransitionClassExecution,
			wantSensitive: true,
			wantReplay:    HotUpdateExecutionReplayClassNone,
			wantReady:     true,
		},
		{
			name:          "reload apply execution",
			transition:    HotUpdateExecutionTransitionReloadApply,
			wantClass:     HotUpdateExecutionTransitionClassExecution,
			wantSensitive: true,
			wantReplay:    HotUpdateExecutionReplayClassNone,
			wantReady:     true,
		},
		{
			name:          "terminal failure recovery metadata",
			transition:    HotUpdateExecutionTransitionTerminalFailure,
			wantClass:     HotUpdateExecutionTransitionClassMetadataRecovery,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "outcome create ledger",
			transition:    HotUpdateExecutionTransitionOutcomeCreate,
			wantClass:     HotUpdateExecutionTransitionClassLedger,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "promotion create ledger",
			transition:    HotUpdateExecutionTransitionPromotionCreate,
			wantClass:     HotUpdateExecutionTransitionClassLedger,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
		{
			name:          "lkg recertify outside guard",
			transition:    HotUpdateExecutionTransitionLKGRecertify,
			wantClass:     HotUpdateExecutionTransitionClassOutside,
			wantSensitive: false,
			wantReplay:    HotUpdateExecutionReplayClassNotApplicable,
			wantReady:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
				Transition:  tt.transition,
				HotUpdateID: "hot-update-readiness",
			})
			if err != nil {
				t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
			}
			if got.TransitionClass != tt.wantClass {
				t.Fatalf("TransitionClass = %q, want %q", got.TransitionClass, tt.wantClass)
			}
			if got.ExecutionSensitive != tt.wantSensitive {
				t.Fatalf("ExecutionSensitive = %t, want %t", got.ExecutionSensitive, tt.wantSensitive)
			}
			if got.ReplayClass != tt.wantReplay {
				t.Fatalf("ReplayClass = %q, want %q", got.ReplayClass, tt.wantReplay)
			}
			if got.Ready != tt.wantReady {
				t.Fatalf("Ready = %t, want %t", got.Ready, tt.wantReady)
			}
			if got.RejectionCode != tt.wantRejectionCode {
				t.Fatalf("RejectionCode = %q, want %q", got.RejectionCode, tt.wantRejectionCode)
			}
		})
	}
}

func TestAssessHotUpdateExecutionReadinessActiveJobSemantics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 17, 30, 0, 0, time.UTC)

	t.Run("absent active occupied job allows execution readiness", func(t *testing.T) {
		t.Parallel()

		got, err := AssessHotUpdateExecutionReadiness(t.TempDir(), HotUpdateExecutionReadinessInput{
			Transition:  HotUpdateExecutionTransitionPointerSwitch,
			HotUpdateID: "hot-update-absent-active",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if !got.Ready {
			t.Fatalf("Ready = false, want true: %#v", got)
		}
		if got.ActiveJobConsidered {
			t.Fatalf("ActiveJobConsidered = true, want false")
		}
	})

	t.Run("same hot-update control job does not count as unsafe live work", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087ActiveJobEvidence(t, root, now, "job-hot-update", JobStateRunning, ExecutionPlaneHotUpdateGate, MissionFamilyApplyHotUpdate)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionPointerSwitch,
			HotUpdateID:  "hot-update-same-control-job",
			CommandJobID: "job-hot-update",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if !got.Ready {
			t.Fatalf("Ready = false, want true: %#v", got)
		}
		if !got.ActiveJobConsidered {
			t.Fatalf("ActiveJobConsidered = false, want true")
		}
		if got.ActiveExecutionPlane != ExecutionPlaneHotUpdateGate {
			t.Fatalf("ActiveExecutionPlane = %q, want %q", got.ActiveExecutionPlane, ExecutionPlaneHotUpdateGate)
		}
	})

	t.Run("active live runtime without quiesce proof blocks with deploy lock", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087ActiveJobEvidence(t, root, now, "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionPointerSwitch,
			HotUpdateID:  "hot-update-live-blocked",
			CommandJobID: "job-hot-update",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if got.Ready {
			t.Fatalf("Ready = true, want false: %#v", got)
		}
		if got.RejectionCode != RejectionCodeV4ActiveJobDeployLock {
			t.Fatalf("RejectionCode = %q, want %q", got.RejectionCode, RejectionCodeV4ActiveJobDeployLock)
		}
		if got.QuiesceState != HotUpdateQuiesceStateNotConfigured {
			t.Fatalf("QuiesceState = %q, want %q", got.QuiesceState, HotUpdateQuiesceStateNotConfigured)
		}
	})

	t.Run("active live runtime failed quiesce blocks with reload quiesce failed", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087ActiveJobEvidence(t, root, now, "job-live", JobStatePaused, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionReloadApply,
			HotUpdateID:  "hot-update-live-quiesce-failed",
			CommandJobID: "job-hot-update",
			QuiesceState: HotUpdateQuiesceStateFailed,
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if got.Ready {
			t.Fatalf("Ready = true, want false: %#v", got)
		}
		if got.RejectionCode != RejectionCodeV4ReloadQuiesceFailed {
			t.Fatalf("RejectionCode = %q, want %q", got.RejectionCode, RejectionCodeV4ReloadQuiesceFailed)
		}
		if got.QuiesceState != HotUpdateQuiesceStateFailed {
			t.Fatalf("QuiesceState = %q, want %q", got.QuiesceState, HotUpdateQuiesceStateFailed)
		}
	})
}

func TestAssessHotUpdateExecutionReadinessReplaySemantics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 18, 0, 0, 0, time.UTC)

	t.Run("pointer switch replay is allowed after pointer already switched", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087HotUpdateFixture(t, root, now, "hot-update-pointer-replay", HotUpdateGateStateReloading, true)
		storeV4087ActiveJobEvidence(t, root, now.Add(time.Hour), "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionPointerSwitch,
			HotUpdateID:  "hot-update-pointer-replay",
			CommandJobID: "job-hot-update",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if !got.Ready {
			t.Fatalf("Ready = false, want true: %#v", got)
		}
		if got.ReplayClass != HotUpdateExecutionReplayClassPointerSwitchAlreadyApplied {
			t.Fatalf("ReplayClass = %q, want %q", got.ReplayClass, HotUpdateExecutionReplayClassPointerSwitchAlreadyApplied)
		}
		if got.GateState != HotUpdateGateStateReloading {
			t.Fatalf("GateState = %q, want %q", got.GateState, HotUpdateGateStateReloading)
		}
	})

	t.Run("reload apply replay is allowed after success", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087HotUpdateFixture(t, root, now, "hot-update-reload-replay", HotUpdateGateStateReloadApplySucceeded, true)
		storeV4087ActiveJobEvidence(t, root, now.Add(time.Hour), "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionReloadApply,
			HotUpdateID:  "hot-update-reload-replay",
			CommandJobID: "job-hot-update",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if !got.Ready {
			t.Fatalf("Ready = false, want true: %#v", got)
		}
		if got.ReplayClass != HotUpdateExecutionReplayClassReloadApplyAlreadySucceeded {
			t.Fatalf("ReplayClass = %q, want %q", got.ReplayClass, HotUpdateExecutionReplayClassReloadApplyAlreadySucceeded)
		}
		if got.GateState != HotUpdateGateStateReloadApplySucceeded {
			t.Fatalf("GateState = %q, want %q", got.GateState, HotUpdateGateStateReloadApplySucceeded)
		}
	})

	t.Run("reload apply retry from recovery needed is blocked without readiness proof", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		storeV4087HotUpdateFixture(t, root, now, "hot-update-retry-blocked", HotUpdateGateStateReloadApplyRecoveryNeeded, true)
		storeV4087ActiveJobEvidence(t, root, now.Add(time.Hour), "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

		got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
			Transition:   HotUpdateExecutionTransitionReloadApply,
			HotUpdateID:  "hot-update-retry-blocked",
			CommandJobID: "job-hot-update",
		})
		if err != nil {
			t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
		}
		if got.Ready {
			t.Fatalf("Ready = true, want false: %#v", got)
		}
		if got.ReplayClass != HotUpdateExecutionReplayClassNone {
			t.Fatalf("ReplayClass = %q, want %q", got.ReplayClass, HotUpdateExecutionReplayClassNone)
		}
		if got.RejectionCode != RejectionCodeV4ActiveJobDeployLock {
			t.Fatalf("RejectionCode = %q, want %q", got.RejectionCode, RejectionCodeV4ActiveJobDeployLock)
		}
	})
}

func TestAssessHotUpdateExecutionReadinessIsReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 18, 30, 0, 0, time.UTC)
	storeV4087HotUpdateFixture(t, root, now, "hot-update-read-only", HotUpdateGateStateReloadApplyRecoveryNeeded, true)
	storeV4087ActiveJobEvidence(t, root, now.Add(time.Hour), "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)

	beforeGateBytes := mustReadFileBytes(t, StoreHotUpdateGatePath(root, "hot-update-read-only"))
	beforePointerBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))
	beforeLKGBytes := mustReadFileBytes(t, StoreLastKnownGoodRuntimePackPointerPath(root))
	beforeRuntimeBytes := mustReadFileBytes(t, StoreJobRuntimePath(root, "job-live"))
	beforeActiveJobBytes := mustReadFileBytes(t, StoreActiveJobPath(root))
	beforeRuntime, err := LoadJobRuntimeRecord(root, "job-live")
	if err != nil {
		t.Fatalf("LoadJobRuntimeRecord(before) error = %v", err)
	}
	beforeControl, err := LoadRuntimeControlRecord(root, "job-live")
	if err != nil {
		t.Fatalf("LoadRuntimeControlRecord(before) error = %v", err)
	}
	beforePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(before) error = %v", err)
	}

	got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
		Transition:   HotUpdateExecutionTransitionReloadApply,
		HotUpdateID:  "hot-update-read-only",
		CommandJobID: "job-hot-update",
	})
	if err != nil {
		t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
	}
	if got.Ready {
		t.Fatalf("Ready = true, want false: %#v", got)
	}

	afterGateBytes := mustReadFileBytes(t, StoreHotUpdateGatePath(root, "hot-update-read-only"))
	afterPointerBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))
	afterLKGBytes := mustReadFileBytes(t, StoreLastKnownGoodRuntimePackPointerPath(root))
	afterRuntimeBytes := mustReadFileBytes(t, StoreJobRuntimePath(root, "job-live"))
	afterActiveJobBytes := mustReadFileBytes(t, StoreActiveJobPath(root))
	afterRuntime, err := LoadJobRuntimeRecord(root, "job-live")
	if err != nil {
		t.Fatalf("LoadJobRuntimeRecord(after) error = %v", err)
	}
	afterControl, err := LoadRuntimeControlRecord(root, "job-live")
	if err != nil {
		t.Fatalf("LoadRuntimeControlRecord(after) error = %v", err)
	}
	afterPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(after) error = %v", err)
	}

	assertBytesEqual(t, "hot-update gate", beforeGateBytes, afterGateBytes)
	assertBytesEqual(t, "active runtime-pack pointer", beforePointerBytes, afterPointerBytes)
	assertBytesEqual(t, "last-known-good pointer", beforeLKGBytes, afterLKGBytes)
	assertBytesEqual(t, "job runtime", beforeRuntimeBytes, afterRuntimeBytes)
	assertBytesEqual(t, "active job", beforeActiveJobBytes, afterActiveJobBytes)
	if !reflect.DeepEqual(beforeRuntime, afterRuntime) {
		t.Fatalf("JobRuntimeRecord changed\nbefore: %#v\nafter: %#v", beforeRuntime, afterRuntime)
	}
	if !reflect.DeepEqual(beforeControl, afterControl) {
		t.Fatalf("RuntimeControlRecord changed\nbefore: %#v\nafter: %#v", beforeControl, afterControl)
	}
	if afterPointer.ReloadGeneration != beforePointer.ReloadGeneration {
		t.Fatalf("ReloadGeneration = %d, want %d", afterPointer.ReloadGeneration, beforePointer.ReloadGeneration)
	}
}

func storeV4087HotUpdateFixture(t *testing.T, root string, now time.Time, hotUpdateID string, state HotUpdateGateState, pointerAlreadySwitched bool) {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-prev"
		record.RollbackTargetPackID = "pack-prev"
		record.MutableSurfaces = []string{"skills"}
		record.SurfaceClasses = []string{"class_1"}
		record.CompatibilityContractRef = "compat-v1"
	}))

	activePackID := "pack-prev"
	updateRecordRef := "bootstrap"
	previousPackID := ""
	if pointerAlreadySwitched {
		activePackID = "pack-candidate"
		updateRecordRef = hotUpdateGatePointerUpdateRecordRef(hotUpdateID)
		previousPackID = "pack-prev"
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         activePackID,
		PreviousActivePackID: previousPackID,
		LastKnownGoodPackID:  "pack-prev",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      updateRecordRef,
		ReloadGeneration:     3,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-prev",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(2 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(3*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = hotUpdateID
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-prev"
		record.RollbackTargetPackID = "pack-prev"
		record.State = state
		record.PhaseUpdatedAt = now.Add(3 * time.Minute)
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
}

func storeV4087ActiveJobEvidence(t *testing.T, root string, now time.Time, jobID string, state JobState, executionPlane string, missionFamily string) {
	t.Helper()

	if err := StoreActiveJobRecord(root, ActiveJobRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    1,
		JobID:          jobID,
		State:          state,
		ActiveStepID:   "step-active",
		AttemptID:      "attempt-1",
		LeaseHolderID:  "lease-1",
		LeaseExpiresAt: now.Add(10 * time.Minute),
		UpdatedAt:      now,
		ActivationSeq:  1,
	}); err != nil {
		t.Fatalf("StoreActiveJobRecord() error = %v", err)
	}
	if err := StoreJobRuntimeRecord(root, JobRuntimeRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    1,
		AppliedSeq:     1,
		JobID:          jobID,
		ExecutionPlane: executionPlane,
		ExecutionHost:  ExecutionHostPhone,
		MissionFamily:  missionFamily,
		State:          state,
		ActiveStepID:   "step-active",
		CreatedAt:      now.Add(-time.Minute),
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("StoreJobRuntimeRecord() error = %v", err)
	}
	if err := StoreRuntimeControlRecord(root, RuntimeControlRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    1,
		LastSeq:        1,
		JobID:          jobID,
		StepID:         "step-active",
		AttemptID:      "attempt-1",
		ExecutionPlane: executionPlane,
		ExecutionHost:  ExecutionHostPhone,
		MissionFamily:  missionFamily,
		MaxAuthority:   AuthorityTierMedium,
		AllowedTools:   []string{"exec"},
		Step: Step{
			ID:           "step-active",
			Type:         StepTypeOneShotCode,
			AllowedTools: []string{"exec"},
		},
	}); err != nil {
		t.Fatalf("StoreRuntimeControlRecord() error = %v", err)
	}
}

func mustReadFileBytes(t *testing.T, path string) []byte {
	t.Helper()

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return bytes
}

func assertBytesEqual(t *testing.T, name string, before []byte, after []byte) {
	t.Helper()

	if string(before) != string(after) {
		t.Fatalf("%s changed\nbefore:\n%s\nafter:\n%s", name, string(before), string(after))
	}
}
