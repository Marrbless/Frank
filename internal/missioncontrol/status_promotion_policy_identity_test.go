package missioncontrol

import (
	"testing"
	"time"
)

func TestLoadOperatorPromotionPolicyIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 19, 0, 0, 0, time.UTC)
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-b"
		record.AllowedSurfaceClasses = []string{"skill"}
		record.ForbiddenSurfaceChanges = []string{"runtime_source"}
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(promotion-policy-b) error = %v", err)
	}
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now, func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-a"
		record.AllowedSurfaceClasses = []string{" source_patch_artifact ", " prompt_pack "}
		record.ForbiddenSurfaceChanges = []string{" policy ", " runtime_source "}
		record.Notes = "read-only policy status"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(promotion-policy-a) error = %v", err)
	}

	got := LoadOperatorPromotionPolicyIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Policies) != 2 {
		t.Fatalf("Policies len = %d, want 2", len(got.Policies))
	}
	policy := got.Policies[0]
	if policy.State != "configured" {
		t.Fatalf("Policies[0].State = %q, want configured", policy.State)
	}
	if policy.PromotionPolicyID != "promotion-policy-a" {
		t.Fatalf("Policies[0].PromotionPolicyID = %q, want promotion-policy-a", policy.PromotionPolicyID)
	}
	if !policy.RequiresHoldoutPass {
		t.Fatal("Policies[0].RequiresHoldoutPass = false, want true")
	}
	if !policy.RequiresCanary {
		t.Fatal("Policies[0].RequiresCanary = false, want true")
	}
	if !policy.RequiresOwnerApproval {
		t.Fatal("Policies[0].RequiresOwnerApproval = false, want true")
	}
	if policy.AllowsAutonomousHotUpdate {
		t.Fatal("Policies[0].AllowsAutonomousHotUpdate = true, want false")
	}
	if len(policy.AllowedSurfaceClasses) != 2 || policy.AllowedSurfaceClasses[0] != "prompt_pack" || policy.AllowedSurfaceClasses[1] != "source_patch_artifact" {
		t.Fatalf("Policies[0].AllowedSurfaceClasses = %#v, want [prompt_pack source_patch_artifact]", policy.AllowedSurfaceClasses)
	}
	if policy.EpsilonRule != "epsilon <= 0.01" {
		t.Fatalf("Policies[0].EpsilonRule = %q, want epsilon <= 0.01", policy.EpsilonRule)
	}
	if policy.RegressionRule != "holdout_regression <= 0" {
		t.Fatalf("Policies[0].RegressionRule = %q, want holdout_regression <= 0", policy.RegressionRule)
	}
	if policy.CompatibilityRule != "compatibility_contract_passed" {
		t.Fatalf("Policies[0].CompatibilityRule = %q, want compatibility_contract_passed", policy.CompatibilityRule)
	}
	if policy.ResourceRule != "canary_budget_within_limit" {
		t.Fatalf("Policies[0].ResourceRule = %q, want canary_budget_within_limit", policy.ResourceRule)
	}
	if policy.MaxCanaryDuration != "15m" {
		t.Fatalf("Policies[0].MaxCanaryDuration = %q, want 15m", policy.MaxCanaryDuration)
	}
	if len(policy.ForbiddenSurfaceChanges) != 2 || policy.ForbiddenSurfaceChanges[0] != "policy" || policy.ForbiddenSurfaceChanges[1] != "runtime_source" {
		t.Fatalf("Policies[0].ForbiddenSurfaceChanges = %#v, want [policy runtime_source]", policy.ForbiddenSurfaceChanges)
	}
	if policy.CreatedAt == nil || *policy.CreatedAt != "2026-04-29T19:00:00Z" {
		t.Fatalf("Policies[0].CreatedAt = %#v, want 2026-04-29T19:00:00Z", policy.CreatedAt)
	}
	if policy.CreatedBy != "operator" {
		t.Fatalf("Policies[0].CreatedBy = %q, want operator", policy.CreatedBy)
	}
	if policy.Notes != "read-only policy status" {
		t.Fatalf("Policies[0].Notes = %q, want read-only policy status", policy.Notes)
	}
	if policy.Error != "" {
		t.Fatalf("Policies[0].Error = %q, want empty", policy.Error)
	}
}

func TestLoadOperatorPromotionPolicyIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorPromotionPolicyIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Policies) != 0 {
		t.Fatalf("Policies len = %d, want 0", len(got.Policies))
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesPromotionPolicyIdentity(t *testing.T) {
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
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now.Add(-30*time.Second), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-status"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.PromotionPolicyIdentity == nil {
		t.Fatalf("RuntimeSummary.PromotionPolicyIdentity = %#v, want populated promotion policy identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.PromotionPolicyIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.PromotionPolicyIdentity.State = %q, want configured", snapshot.RuntimeSummary.PromotionPolicyIdentity.State)
	}
	if len(snapshot.RuntimeSummary.PromotionPolicyIdentity.Policies) != 1 {
		t.Fatalf("RuntimeSummary.PromotionPolicyIdentity.Policies len = %d, want 1", len(snapshot.RuntimeSummary.PromotionPolicyIdentity.Policies))
	}
	if snapshot.RuntimeSummary.PromotionPolicyIdentity.Policies[0].PromotionPolicyID != "promotion-policy-status" {
		t.Fatalf("RuntimeSummary.PromotionPolicyIdentity.Policies[0].PromotionPolicyID = %q, want promotion-policy-status", snapshot.RuntimeSummary.PromotionPolicyIdentity.Policies[0].PromotionPolicyID)
	}
}
