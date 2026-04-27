package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorHotUpdateOutcomeIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 19, 0, 0, 0, time.UTC)
	storeHotUpdateOutcomeFixtures(t, root, now)
	if err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-2"
		record.OutcomeKind = HotUpdateOutcomeKindBlocked
		record.Reason = "operator blocked activation"
		record.Notes = "read-only block reason"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}

	got := LoadOperatorHotUpdateOutcomeIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Outcomes) != 1 {
		t.Fatalf("Outcomes len = %d, want 1", len(got.Outcomes))
	}
	outcome := got.Outcomes[0]
	if outcome.State != "configured" {
		t.Fatalf("Outcomes[1].State = %q, want configured", outcome.State)
	}
	if outcome.OutcomeID != "outcome-2" {
		t.Fatalf("Outcomes[1].OutcomeID = %q, want outcome-2", outcome.OutcomeID)
	}
	if outcome.HotUpdateID != "hot-update-1" {
		t.Fatalf("Outcomes[1].HotUpdateID = %q, want hot-update-1", outcome.HotUpdateID)
	}
	if outcome.CanaryRef != "" || outcome.ApprovalRef != "" {
		t.Fatalf("Outcomes[1] refs = %q/%q, want empty", outcome.CanaryRef, outcome.ApprovalRef)
	}
	if outcome.CandidateID != "candidate-1" {
		t.Fatalf("Outcomes[1].CandidateID = %q, want candidate-1", outcome.CandidateID)
	}
	if outcome.RunID != "run-1" {
		t.Fatalf("Outcomes[1].RunID = %q, want run-1", outcome.RunID)
	}
	if outcome.CandidateResultID != "result-1" {
		t.Fatalf("Outcomes[1].CandidateResultID = %q, want result-1", outcome.CandidateResultID)
	}
	if outcome.CandidatePackID != "pack-candidate" {
		t.Fatalf("Outcomes[1].CandidatePackID = %q, want pack-candidate", outcome.CandidatePackID)
	}
	if outcome.OutcomeKind != string(HotUpdateOutcomeKindBlocked) {
		t.Fatalf("Outcomes[1].OutcomeKind = %q, want blocked", outcome.OutcomeKind)
	}
	if outcome.Reason != "operator blocked activation" {
		t.Fatalf("Outcomes[1].Reason = %q, want operator blocked activation", outcome.Reason)
	}
	if outcome.Notes != "read-only block reason" {
		t.Fatalf("Outcomes[1].Notes = %q, want read-only block reason", outcome.Notes)
	}
	if outcome.OutcomeAt == nil || *outcome.OutcomeAt != "2026-04-27T19:08:00Z" {
		t.Fatalf("Outcomes[1].OutcomeAt = %#v, want 2026-04-27T19:08:00Z", outcome.OutcomeAt)
	}
	if outcome.CreatedAt == nil || *outcome.CreatedAt != "2026-04-27T19:09:00Z" {
		t.Fatalf("Outcomes[1].CreatedAt = %#v, want 2026-04-27T19:09:00Z", outcome.CreatedAt)
	}
	if outcome.CreatedBy != "operator" {
		t.Fatalf("Outcomes[1].CreatedBy = %q, want operator", outcome.CreatedBy)
	}
	if outcome.Error != "" {
		t.Fatalf("Outcomes[1].Error = %q, want empty", outcome.Error)
	}
}

func TestLoadOperatorHotUpdateOutcomeIdentityStatusIncludesCanaryRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 19, 30, 0, 0, time.UTC)
	fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", true)
	outcome, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}

	got := LoadOperatorHotUpdateOutcomeIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Outcomes) != 1 {
		t.Fatalf("Outcomes len = %d, want 1", len(got.Outcomes))
	}
	status := got.Outcomes[0]
	if status.OutcomeID != outcome.OutcomeID {
		t.Fatalf("OutcomeID = %q, want %q", status.OutcomeID, outcome.OutcomeID)
	}
	if status.CanaryRef != fixture.authority.CanarySatisfactionAuthorityID {
		t.Fatalf("CanaryRef = %q, want %q", status.CanaryRef, fixture.authority.CanarySatisfactionAuthorityID)
	}
	if status.ApprovalRef != fixture.decision.OwnerApprovalDecisionID {
		t.Fatalf("ApprovalRef = %q, want %q", status.ApprovalRef, fixture.decision.OwnerApprovalDecisionID)
	}
}

func TestLoadOperatorHotUpdateOutcomeIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorHotUpdateOutcomeIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Outcomes) != 0 {
		t.Fatalf("Outcomes len = %d, want 0", len(got.Outcomes))
	}
}

func TestLoadOperatorHotUpdateOutcomeIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)
	storeHotUpdateOutcomeFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StoreHotUpdateOutcomePath(root, "outcome-bad"), validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.RecordVersion = StoreRecordVersion
		record.OutcomeID = "outcome-bad"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(outcome-bad) error = %v", err)
	}

	got := LoadOperatorHotUpdateOutcomeIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Outcomes) != 1 {
		t.Fatalf("Outcomes len = %d, want 1", len(got.Outcomes))
	}
	outcome := got.Outcomes[0]
	if outcome.State != "invalid" {
		t.Fatalf("Outcomes[1].State = %q, want invalid", outcome.State)
	}
	if outcome.OutcomeID != "outcome-bad" {
		t.Fatalf("Outcomes[1].OutcomeID = %q, want outcome-bad", outcome.OutcomeID)
	}
	if outcome.CandidatePackID != "pack-missing" {
		t.Fatalf("Outcomes[1].CandidatePackID = %q, want pack-missing", outcome.CandidatePackID)
	}
	if !strings.Contains(outcome.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Outcomes[1].Error = %q, want missing candidate_pack_id context", outcome.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesHotUpdateOutcomeIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := testLeaseSafeNow()
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
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	storeHotUpdateOutcomeFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(-30*time.Second), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-2"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.HotUpdateOutcomeIdentity == nil {
		t.Fatalf("RuntimeSummary.HotUpdateOutcomeIdentity = %#v, want populated hot-update outcome identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.HotUpdateOutcomeIdentity.State = %q, want configured", snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.State)
	}
	if len(snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes) != 1 {
		t.Fatalf("RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes len = %d, want 1", len(snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes))
	}
	if snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes[0].OutcomeID != "outcome-2" {
		t.Fatalf("RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes[0].OutcomeID = %q, want outcome-2", snapshot.RuntimeSummary.HotUpdateOutcomeIdentity.Outcomes[0].OutcomeID)
	}
}
