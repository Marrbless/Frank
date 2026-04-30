package missioncontrol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOperatorStatusSummaryGoldenV4KeySurface(t *testing.T) {
	t.Parallel()

	summary := OperatorStatusSummary{
		JobID:               "job-1",
		ExecutionPlane:      ExecutionPlaneHotUpdateGate,
		ExecutionHost:       ExecutionHostPhone,
		MissionFamily:       MissionFamilyApplyHotUpdate,
		TopologyModeEnabled: true,
		State:               JobStateRunning,
		ActiveStepID:        "apply_hot_update",
		AllowedTools:        []string{"read_file", "write_file"},
		V4Summary: &OperatorV4SummaryStatus{
			State:                         "promotion_recorded",
			ActivePackID:                  "pack-active",
			LastKnownGoodPackID:           "pack-lkg",
			SelectedHotUpdateID:           "hot-update-1",
			SelectedOutcomeID:             "outcome-1",
			SelectedPromotionID:           "promotion-1",
			HasCandidatePromotionDecision: true,
			HasCanaryAuthority:            true,
			HasOwnerApprovalDecision:      true,
			InvalidIdentityCount:          1,
			Warnings:                      []string{"hot_update_gate_identity invalid"},
		},
	}

	got, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile(filepath.Join("testdata", "operator_status_v4_key_surface.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile(golden) error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("operator status golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestOperatorV4SummaryEmptyStoreIsNotConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateRunning,
	})
	summary = withOperatorV4IdentitySurfaces(summary, root)
	summary = WithV4Summary(summary)

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want not_configured summary")
	}
	if summary.V4Summary.State != "not_configured" {
		t.Fatalf("V4Summary.State = %q, want not_configured", summary.V4Summary.State)
	}
	if summary.V4Summary.InvalidIdentityCount != 0 {
		t.Fatalf("V4Summary.InvalidIdentityCount = %d, want 0", summary.V4Summary.InvalidIdentityCount)
	}
	if len(summary.V4Summary.Warnings) != 0 {
		t.Fatalf("V4Summary.Warnings = %#v, want empty", summary.V4Summary.Warnings)
	}
}

func TestOperatorV4SummarySurfacesInvalidIdentityWithoutHidingDetails(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateRunning,
	})
	summary.HotUpdateGateIdentity = &OperatorHotUpdateGateIdentityStatus{
		State: "invalid",
		Gates: []OperatorHotUpdateGateStatus{{
			HotUpdateID: "hot-update-bad",
			State:       "invalid",
			Error:       "bad gate",
		}},
	}
	summary = WithV4Summary(summary)

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want invalid summary")
	}
	if summary.V4Summary.State != "invalid" {
		t.Fatalf("V4Summary.State = %q, want invalid", summary.V4Summary.State)
	}
	if summary.V4Summary.InvalidIdentityCount != 1 {
		t.Fatalf("V4Summary.InvalidIdentityCount = %d, want 1", summary.V4Summary.InvalidIdentityCount)
	}
	if len(summary.V4Summary.Warnings) != 1 || summary.V4Summary.Warnings[0] != "hot_update_gate_identity invalid" {
		t.Fatalf("V4Summary.Warnings = %#v, want hot-update gate warning", summary.V4Summary.Warnings)
	}
	if summary.V4Summary.SelectedHotUpdateID != "hot-update-bad" {
		t.Fatalf("V4Summary.SelectedHotUpdateID = %q, want invalid hot-update id", summary.V4Summary.SelectedHotUpdateID)
	}
	if summary.HotUpdateGateIdentity == nil || len(summary.HotUpdateGateIdentity.Gates) != 1 || summary.HotUpdateGateIdentity.Gates[0].Error != "bad gate" {
		t.Fatalf("HotUpdateGateIdentity details = %#v, want invalid detail preserved", summary.HotUpdateGateIdentity)
	}
}

func TestOperatorV4SummarySurfacesRecoveryRefs(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateRunning,
	})
	summary.RollbackIdentity = &OperatorRollbackIdentityStatus{
		State: "configured",
		Rollbacks: []OperatorRollbackStatus{{
			State:      "configured",
			RollbackID: "rollback-1",
		}},
	}
	summary.RollbackApplyIdentity = &OperatorRollbackApplyIdentityStatus{
		State: "configured",
		Applies: []OperatorRollbackApplyStatus{{
			State:           "configured",
			RollbackApplyID: "rollback-apply-1",
			RollbackID:      "rollback-1",
		}},
	}
	summary = WithV4Summary(summary)

	if summary.V4Summary == nil {
		t.Fatal("V4Summary = nil, want recovery summary")
	}
	if summary.V4Summary.State != "rollback_apply_recorded" {
		t.Fatalf("V4Summary.State = %q, want rollback_apply_recorded", summary.V4Summary.State)
	}
	if summary.V4Summary.SelectedRollbackID != "rollback-1" {
		t.Fatalf("V4Summary.SelectedRollbackID = %q, want rollback-1", summary.V4Summary.SelectedRollbackID)
	}
	if summary.V4Summary.SelectedRollbackApplyID != "rollback-apply-1" {
		t.Fatalf("V4Summary.SelectedRollbackApplyID = %q, want rollback-apply-1", summary.V4Summary.SelectedRollbackApplyID)
	}
	if !summary.V4Summary.HasRollback || !summary.V4Summary.HasRollbackApply {
		t.Fatalf("V4Summary recovery booleans = rollback %t apply %t, want both true", summary.V4Summary.HasRollback, summary.V4Summary.HasRollbackApply)
	}
}

func withOperatorV4IdentitySurfaces(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	summary = WithRuntimePackIdentity(summary, root)
	summary = WithImprovementCandidateIdentity(summary, root)
	summary = WithEvalSuiteIdentity(summary, root)
	summary = WithPromotionPolicyIdentity(summary, root)
	summary = WithImprovementRunIdentity(summary, root)
	summary = WithCandidateResultIdentity(summary, root)
	summary = WithHotUpdateCanaryRequirementIdentity(summary, root)
	summary = WithHotUpdateCanaryEvidenceIdentity(summary, root)
	summary = WithHotUpdateCanarySatisfactionIdentity(summary, root)
	summary = WithHotUpdateCanarySatisfactionAuthorityIdentity(summary, root)
	summary = WithHotUpdateOwnerApprovalRequestIdentity(summary, root)
	summary = WithHotUpdateOwnerApprovalDecisionIdentity(summary, root)
	summary = WithCandidatePromotionDecisionIdentity(summary, root)
	summary = WithHotUpdateGateIdentity(summary, root)
	summary = WithHotUpdateOutcomeIdentity(summary, root)
	summary = WithPromotionIdentity(summary, root)
	summary = WithRollbackIdentity(summary, root)
	summary = WithRollbackApplyIdentity(summary, root)
	return summary
}
