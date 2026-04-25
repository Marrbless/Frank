package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateHotUpdateExecutionSafetyEvidenceRecordRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 19, 0, 0, 0, time.UTC)
	valid := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence-valid", "job-live")

	tests := []struct {
		name   string
		mutate func(*HotUpdateExecutionSafetyEvidenceRecord)
		want   string
	}{
		{
			name: "missing record_version",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.RecordVersion = 0
			},
			want: "record_version",
		},
		{
			name: "missing evidence_id",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.EvidenceID = ""
			},
			want: "evidence_id",
		},
		{
			name: "missing hot_update_id",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.HotUpdateID = ""
			},
			want: "hot_update_id",
		},
		{
			name: "missing job_id",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.JobID = ""
			},
			want: "job_id",
		},
		{
			name: "missing deploy_lock_state",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.DeployLockState = ""
			},
			want: "deploy_lock_state",
		},
		{
			name: "missing quiesce_state",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.QuiesceState = ""
			},
			want: "quiesce_state",
		},
		{
			name: "missing created_at",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.CreatedAt = time.Time{}
			},
			want: "created_at",
		},
		{
			name: "missing created_by",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.CreatedBy = ""
			},
			want: "created_by",
		},
		{
			name: "ready unlocked evidence requires expires_at",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.ExpiresAt = time.Time{}
			},
			want: "expires_at",
		},
		{
			name: "invalid deploy lock state",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.DeployLockState = "not-a-state"
			},
			want: "deploy_lock_state",
		},
		{
			name: "invalid quiesce state",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.QuiesceState = HotUpdateQuiesceStateNotConfigured
			},
			want: "quiesce_state",
		},
		{
			name: "deterministic id mismatch",
			mutate: func(record *HotUpdateExecutionSafetyEvidenceRecord) {
				record.EvidenceID = "hot-update-execution-safety-hot-update-other-job-live"
			},
			want: "deterministic evidence_id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := valid
			tt.mutate(&record)
			err := ValidateHotUpdateExecutionSafetyEvidenceRecord(record)
			if err == nil {
				t.Fatal("ValidateHotUpdateExecutionSafetyEvidenceRecord() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateHotUpdateExecutionSafetyEvidenceRecord() error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestHotUpdateExecutionSafetyEvidenceIDIsDeterministic(t *testing.T) {
	t.Parallel()

	got, err := HotUpdateExecutionSafetyEvidenceID(" hot-update-det ", " job-live ")
	if err != nil {
		t.Fatalf("HotUpdateExecutionSafetyEvidenceID() error = %v", err)
	}
	want := "hot-update-execution-safety-hot-update-det-job-live"
	if got != want {
		t.Fatalf("HotUpdateExecutionSafetyEvidenceID() = %q, want %q", got, want)
	}
}

func TestHotUpdateExecutionSafetyEvidenceStoresLoadsListsAndReplays(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 19, 15, 0, 0, time.UTC)
	first := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-b", "job-live")
	second := validHotUpdateExecutionSafetyEvidenceRecord(t, now.Add(time.Minute), "hot-update-a", "job-live")

	stored, changed, err := StoreHotUpdateExecutionSafetyEvidenceRecord(root, first)
	if err != nil {
		t.Fatalf("StoreHotUpdateExecutionSafetyEvidenceRecord(first) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreHotUpdateExecutionSafetyEvidenceRecord(first) changed = false, want true")
	}
	if !reflect.DeepEqual(stored, NormalizeHotUpdateExecutionSafetyEvidenceRecord(first)) {
		t.Fatalf("stored first = %#v, want %#v", stored, NormalizeHotUpdateExecutionSafetyEvidenceRecord(first))
	}
	beforeBytes := mustReadFileBytes(t, StoreHotUpdateExecutionSafetyEvidencePath(root, first.EvidenceID))

	replayed, changed, err := StoreHotUpdateExecutionSafetyEvidenceRecord(root, first)
	if err != nil {
		t.Fatalf("StoreHotUpdateExecutionSafetyEvidenceRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreHotUpdateExecutionSafetyEvidenceRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, stored) {
		t.Fatalf("replayed = %#v, want %#v", replayed, stored)
	}
	afterBytes := mustReadFileBytes(t, StoreHotUpdateExecutionSafetyEvidencePath(root, first.EvidenceID))
	assertBytesEqual(t, "execution safety evidence", beforeBytes, afterBytes)

	divergent := first
	divergent.Reason = "different"
	if _, _, err := StoreHotUpdateExecutionSafetyEvidenceRecord(root, divergent); err == nil {
		t.Fatal("StoreHotUpdateExecutionSafetyEvidenceRecord(divergent) error = nil, want duplicate rejection")
	}

	if _, _, err := StoreHotUpdateExecutionSafetyEvidenceRecord(root, second); err != nil {
		t.Fatalf("StoreHotUpdateExecutionSafetyEvidenceRecord(second) error = %v", err)
	}
	loaded, err := LoadHotUpdateExecutionSafetyEvidenceRecord(root, first.EvidenceID)
	if err != nil {
		t.Fatalf("LoadHotUpdateExecutionSafetyEvidenceRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, stored) {
		t.Fatalf("loaded = %#v, want %#v", loaded, stored)
	}
	current, err := LoadCurrentHotUpdateExecutionSafetyEvidenceRecord(root, first.HotUpdateID, first.JobID)
	if err != nil {
		t.Fatalf("LoadCurrentHotUpdateExecutionSafetyEvidenceRecord() error = %v", err)
	}
	if !reflect.DeepEqual(current, stored) {
		t.Fatalf("current = %#v, want %#v", current, stored)
	}

	listed, err := ListHotUpdateExecutionSafetyEvidenceRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateExecutionSafetyEvidenceRecords() error = %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len(listed) = %d, want 2", len(listed))
	}
	if listed[0].EvidenceID != second.EvidenceID || listed[1].EvidenceID != first.EvidenceID {
		t.Fatalf("listed order = [%q, %q], want [%q, %q]", listed[0].EvidenceID, listed[1].EvidenceID, second.EvidenceID, first.EvidenceID)
	}
}

func TestAssessHotUpdateExecutionReadinessConsumesSafetyEvidence(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 25, 19, 30, 0, 0, time.UTC)

	tests := []struct {
		name              string
		transition        HotUpdateExecutionTransition
		storeEvidence     func(t *testing.T, root string)
		wantReady         bool
		wantRejectionCode RejectionCode
		wantQuiesceState  HotUpdateQuiesceState
		wantExpired       bool
		wantStale         bool
	}{
		{
			name:              "missing evidence blocks with deploy lock",
			transition:        HotUpdateExecutionTransitionPointerSwitch,
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateNotConfigured,
		},
		{
			name:       "expired evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.ExpiresAt = now.Add(-time.Minute)
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
			wantExpired:       true,
		},
		{
			name:       "different hot update evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-other", "job-live")
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateNotConfigured,
		},
		{
			name:       "different job evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-other")
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateNotConfigured,
		},
		{
			name:       "deploy locked evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.DeployLockState = HotUpdateDeployLockStateDeployLocked
				record.QuiesceState = HotUpdateQuiesceStateReady
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
		},
		{
			name:       "unknown deploy lock evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.DeployLockState = HotUpdateDeployLockStateUnknown
				record.QuiesceState = HotUpdateQuiesceStateReady
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
		},
		{
			name:       "unknown quiesce evidence blocks with deploy lock",
			transition: HotUpdateExecutionTransitionReloadApply,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.DeployLockState = HotUpdateDeployLockStateDeployUnlocked
				record.QuiesceState = HotUpdateQuiesceStateUnknown
				record.ExpiresAt = time.Time{}
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateUnknown,
		},
		{
			name:       "quiesce failed evidence blocks with reload quiesce failed",
			transition: HotUpdateExecutionTransitionReloadApply,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.DeployLockState = HotUpdateDeployLockStateDeployUnlocked
				record.QuiesceState = HotUpdateQuiesceStateFailed
				record.ExpiresAt = time.Time{}
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ReloadQuiesceFailed,
			wantQuiesceState:  HotUpdateQuiesceStateFailed,
		},
		{
			name:       "ready evidence allows pointer switch readiness",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:        true,
			wantQuiesceState: HotUpdateQuiesceStateReady,
		},
		{
			name:       "ready evidence allows reload apply readiness",
			transition: HotUpdateExecutionTransitionReloadApply,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:        true,
			wantQuiesceState: HotUpdateQuiesceStateReady,
		},
		{
			name:       "active step mismatch blocks as stale",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.ActiveStepID = "step-other"
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
			wantStale:         true,
		},
		{
			name:       "writer epoch mismatch blocks as stale",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.WriterEpoch = 99
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
			wantStale:         true,
		},
		{
			name:       "activation sequence mismatch blocks as stale",
			transition: HotUpdateExecutionTransitionPointerSwitch,
			storeEvidence: func(t *testing.T, root string) {
				record := validHotUpdateExecutionSafetyEvidenceRecord(t, now, "hot-update-evidence", "job-live")
				record.ActivationSeq = 99
				mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)
			},
			wantReady:         false,
			wantRejectionCode: RejectionCodeV4ActiveJobDeployLock,
			wantQuiesceState:  HotUpdateQuiesceStateReady,
			wantStale:         true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			storeV4087ActiveJobEvidence(t, root, now, "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)
			if tt.storeEvidence != nil {
				tt.storeEvidence(t, root)
			}

			got, err := AssessHotUpdateExecutionReadiness(root, HotUpdateExecutionReadinessInput{
				Transition:   tt.transition,
				HotUpdateID:  "hot-update-evidence",
				CommandJobID: "job-hot-update",
				AssessedAt:   now,
			})
			if err != nil {
				t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
			}
			if got.Ready != tt.wantReady {
				t.Fatalf("Ready = %t, want %t: %#v", got.Ready, tt.wantReady, got)
			}
			if got.RejectionCode != tt.wantRejectionCode {
				t.Fatalf("RejectionCode = %q, want %q", got.RejectionCode, tt.wantRejectionCode)
			}
			if got.QuiesceState != tt.wantQuiesceState {
				t.Fatalf("QuiesceState = %q, want %q", got.QuiesceState, tt.wantQuiesceState)
			}
			if got.EvidenceExpired != tt.wantExpired {
				t.Fatalf("EvidenceExpired = %t, want %t", got.EvidenceExpired, tt.wantExpired)
			}
			if got.EvidenceStale != tt.wantStale {
				t.Fatalf("EvidenceStale = %t, want %t", got.EvidenceStale, tt.wantStale)
			}
		})
	}
}

func TestAssessHotUpdateExecutionReadinessWithSafetyEvidenceIsReadOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 20, 0, 0, 0, time.UTC)
	storeV4087HotUpdateFixture(t, root, now, "hot-update-read-only-evidence", HotUpdateGateStateStaged, false)
	storeV4087ActiveJobEvidence(t, root, now.Add(time.Hour), "job-live", JobStateRunning, ExecutionPlaneLiveRuntime, MissionFamilyBootstrapRevenue)
	record := validHotUpdateExecutionSafetyEvidenceRecord(t, now.Add(time.Hour), "hot-update-read-only-evidence", "job-live")
	mustStoreHotUpdateExecutionSafetyEvidence(t, root, record)

	beforeGateBytes := mustReadFileBytes(t, StoreHotUpdateGatePath(root, "hot-update-read-only-evidence"))
	beforePointerBytes := mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root))
	beforeLKGBytes := mustReadFileBytes(t, StoreLastKnownGoodRuntimePackPointerPath(root))
	beforeRuntimeBytes := mustReadFileBytes(t, StoreJobRuntimePath(root, "job-live"))
	beforeActiveJobBytes := mustReadFileBytes(t, StoreActiveJobPath(root))
	beforeEvidenceBytes := mustReadFileBytes(t, StoreHotUpdateExecutionSafetyEvidencePath(root, record.EvidenceID))
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
		Transition:   HotUpdateExecutionTransitionPointerSwitch,
		HotUpdateID:  "hot-update-read-only-evidence",
		CommandJobID: "job-hot-update",
		AssessedAt:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("AssessHotUpdateExecutionReadiness() error = %v", err)
	}
	if !got.Ready {
		t.Fatalf("Ready = false, want true: %#v", got)
	}

	assertBytesEqual(t, "hot-update gate", beforeGateBytes, mustReadFileBytes(t, StoreHotUpdateGatePath(root, "hot-update-read-only-evidence")))
	assertBytesEqual(t, "active runtime-pack pointer", beforePointerBytes, mustReadFileBytes(t, StoreActiveRuntimePackPointerPath(root)))
	assertBytesEqual(t, "last-known-good pointer", beforeLKGBytes, mustReadFileBytes(t, StoreLastKnownGoodRuntimePackPointerPath(root)))
	assertBytesEqual(t, "job runtime", beforeRuntimeBytes, mustReadFileBytes(t, StoreJobRuntimePath(root, "job-live")))
	assertBytesEqual(t, "active job", beforeActiveJobBytes, mustReadFileBytes(t, StoreActiveJobPath(root)))
	assertBytesEqual(t, "execution safety evidence", beforeEvidenceBytes, mustReadFileBytes(t, StoreHotUpdateExecutionSafetyEvidencePath(root, record.EvidenceID)))
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
	if !reflect.DeepEqual(beforeRuntime, afterRuntime) {
		t.Fatalf("JobRuntimeRecord changed\nbefore: %#v\nafter: %#v", beforeRuntime, afterRuntime)
	}
	if !reflect.DeepEqual(beforeControl, afterControl) {
		t.Fatalf("RuntimeControlRecord changed\nbefore: %#v\nafter: %#v", beforeControl, afterControl)
	}
	if afterPointer.ReloadGeneration != beforePointer.ReloadGeneration {
		t.Fatalf("ReloadGeneration = %d, want %d", afterPointer.ReloadGeneration, beforePointer.ReloadGeneration)
	}
	assertNoHotUpdateExecutionSafetySideEffectRecords(t, root)
}

func validHotUpdateExecutionSafetyEvidenceRecord(t *testing.T, now time.Time, hotUpdateID string, jobID string) HotUpdateExecutionSafetyEvidenceRecord {
	t.Helper()

	evidenceID, err := HotUpdateExecutionSafetyEvidenceID(hotUpdateID, jobID)
	if err != nil {
		t.Fatalf("HotUpdateExecutionSafetyEvidenceID() error = %v", err)
	}
	return HotUpdateExecutionSafetyEvidenceRecord{
		RecordVersion:   StoreRecordVersion,
		EvidenceID:      evidenceID,
		HotUpdateID:     hotUpdateID,
		JobID:           jobID,
		ActiveStepID:    "step-active",
		AttemptID:       "attempt-1",
		WriterEpoch:     1,
		ActivationSeq:   1,
		DeployLockState: HotUpdateDeployLockStateDeployUnlocked,
		QuiesceState:    HotUpdateQuiesceStateReady,
		Reason:          "quiesced at active-step boundary",
		CreatedAt:       now,
		CreatedBy:       "operator",
		ExpiresAt:       now.Add(5 * time.Minute),
	}
}

func mustStoreHotUpdateExecutionSafetyEvidence(t *testing.T, root string, record HotUpdateExecutionSafetyEvidenceRecord) HotUpdateExecutionSafetyEvidenceRecord {
	t.Helper()

	stored, _, err := StoreHotUpdateExecutionSafetyEvidenceRecord(root, record)
	if err != nil {
		t.Fatalf("StoreHotUpdateExecutionSafetyEvidenceRecord() error = %v", err)
	}
	return stored
}

func assertNoHotUpdateExecutionSafetySideEffectRecords(t *testing.T, root string) {
	t.Helper()

	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("len(outcomes) = %d, want 0", len(outcomes))
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("len(promotions) = %d, want 0", len(promotions))
	}
	rollbacks, err := ListRollbackRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("len(rollbacks) = %d, want 0", len(rollbacks))
	}
	rollbackApplies, err := ListRollbackApplyRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(rollbackApplies) != 0 {
		t.Fatalf("len(rollbackApplies) = %d, want 0", len(rollbackApplies))
	}
}
