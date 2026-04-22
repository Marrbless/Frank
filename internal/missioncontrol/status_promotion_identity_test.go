package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorPromotionIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 19, 0, 0, 0, time.UTC)
	storePromotionFixtures(t, root, now)
	if err := StorePromotionRecord(root, validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-2"
		record.Reason = "operator approved promotion"
		record.Notes = "read-only promotion status"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StorePromotionRecord() error = %v", err)
	}

	got := LoadOperatorPromotionIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Promotions) != 1 {
		t.Fatalf("Promotions len = %d, want 1", len(got.Promotions))
	}
	promotion := got.Promotions[0]
	if promotion.State != "configured" {
		t.Fatalf("Promotions[0].State = %q, want configured", promotion.State)
	}
	if promotion.PromotionID != "promotion-2" {
		t.Fatalf("Promotions[0].PromotionID = %q, want promotion-2", promotion.PromotionID)
	}
	if promotion.PromotedPackID != "pack-candidate" {
		t.Fatalf("Promotions[0].PromotedPackID = %q, want pack-candidate", promotion.PromotedPackID)
	}
	if promotion.PreviousActivePackID != "pack-base" {
		t.Fatalf("Promotions[0].PreviousActivePackID = %q, want pack-base", promotion.PreviousActivePackID)
	}
	if promotion.LastKnownGoodPackID != "pack-base" {
		t.Fatalf("Promotions[0].LastKnownGoodPackID = %q, want pack-base", promotion.LastKnownGoodPackID)
	}
	if promotion.LastKnownGoodBasis != "holdout_pass" {
		t.Fatalf("Promotions[0].LastKnownGoodBasis = %q, want holdout_pass", promotion.LastKnownGoodBasis)
	}
	if promotion.HotUpdateID != "hot-update-1" {
		t.Fatalf("Promotions[0].HotUpdateID = %q, want hot-update-1", promotion.HotUpdateID)
	}
	if promotion.OutcomeID != "outcome-1" {
		t.Fatalf("Promotions[0].OutcomeID = %q, want outcome-1", promotion.OutcomeID)
	}
	if promotion.CandidateID != "candidate-1" {
		t.Fatalf("Promotions[0].CandidateID = %q, want candidate-1", promotion.CandidateID)
	}
	if promotion.RunID != "run-1" {
		t.Fatalf("Promotions[0].RunID = %q, want run-1", promotion.RunID)
	}
	if promotion.CandidateResultID != "result-1" {
		t.Fatalf("Promotions[0].CandidateResultID = %q, want result-1", promotion.CandidateResultID)
	}
	if promotion.Reason != "operator approved promotion" {
		t.Fatalf("Promotions[0].Reason = %q, want operator approved promotion", promotion.Reason)
	}
	if promotion.Notes != "read-only promotion status" {
		t.Fatalf("Promotions[0].Notes = %q, want read-only promotion status", promotion.Notes)
	}
	if promotion.PromotedAt == nil || *promotion.PromotedAt != "2026-04-28T19:10:00Z" {
		t.Fatalf("Promotions[0].PromotedAt = %#v, want 2026-04-28T19:10:00Z", promotion.PromotedAt)
	}
	if promotion.CreatedAt == nil || *promotion.CreatedAt != "2026-04-28T19:11:00Z" {
		t.Fatalf("Promotions[0].CreatedAt = %#v, want 2026-04-28T19:11:00Z", promotion.CreatedAt)
	}
	if promotion.CreatedBy != "operator" {
		t.Fatalf("Promotions[0].CreatedBy = %q, want operator", promotion.CreatedBy)
	}
	if promotion.Error != "" {
		t.Fatalf("Promotions[0].Error = %q, want empty", promotion.Error)
	}
}

func TestLoadOperatorPromotionIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorPromotionIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Promotions) != 0 {
		t.Fatalf("Promotions len = %d, want 0", len(got.Promotions))
	}
}

func TestLoadOperatorPromotionIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 28, 20, 0, 0, 0, time.UTC)
	storePromotionFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StorePromotionPath(root, "promotion-bad"), validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.RecordVersion = StoreRecordVersion
		record.PromotionID = "promotion-bad"
		record.PromotedPackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(promotion-bad) error = %v", err)
	}

	got := LoadOperatorPromotionIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Promotions) != 1 {
		t.Fatalf("Promotions len = %d, want 1", len(got.Promotions))
	}
	promotion := got.Promotions[0]
	if promotion.State != "invalid" {
		t.Fatalf("Promotions[0].State = %q, want invalid", promotion.State)
	}
	if promotion.PromotionID != "promotion-bad" {
		t.Fatalf("Promotions[0].PromotionID = %q, want promotion-bad", promotion.PromotionID)
	}
	if promotion.PromotedPackID != "pack-missing" {
		t.Fatalf("Promotions[0].PromotedPackID = %q, want pack-missing", promotion.PromotedPackID)
	}
	if !strings.Contains(promotion.Error, `promoted_pack_id "pack-missing"`) {
		t.Fatalf("Promotions[0].Error = %q, want missing promoted_pack_id context", promotion.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesPromotionIdentity(t *testing.T) {
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

	storePromotionFixtures(t, root, now.Add(-10*time.Minute))
	if err := StorePromotionRecord(root, validPromotionRecord(now.Add(-30*time.Second), func(record *PromotionRecord) {
		record.PromotionID = "promotion-2"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StorePromotionRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.PromotionIdentity == nil {
		t.Fatalf("RuntimeSummary.PromotionIdentity = %#v, want populated promotion identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.PromotionIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.PromotionIdentity.State = %q, want configured", snapshot.RuntimeSummary.PromotionIdentity.State)
	}
	if len(snapshot.RuntimeSummary.PromotionIdentity.Promotions) != 1 {
		t.Fatalf("RuntimeSummary.PromotionIdentity.Promotions len = %d, want 1", len(snapshot.RuntimeSummary.PromotionIdentity.Promotions))
	}
	if snapshot.RuntimeSummary.PromotionIdentity.Promotions[0].PromotionID != "promotion-2" {
		t.Fatalf("RuntimeSummary.PromotionIdentity.Promotions[0].PromotionID = %q, want promotion-2", snapshot.RuntimeSummary.PromotionIdentity.Promotions[0].PromotionID)
	}
}
