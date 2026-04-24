package missioncontrol

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestEnumValues(t *testing.T) {
	t.Parallel()

	if JobSpecVersionV2 != "frank_v2" {
		t.Fatalf("JobSpecVersionV2 = %q, want %q", JobSpecVersionV2, "frank_v2")
	}
	if JobSpecVersionV4 != "frank_v4" {
		t.Fatalf("JobSpecVersionV4 = %q, want %q", JobSpecVersionV4, "frank_v4")
	}
	if ExecutionPlaneLiveRuntime != "live_runtime" {
		t.Fatalf("ExecutionPlaneLiveRuntime = %q, want %q", ExecutionPlaneLiveRuntime, "live_runtime")
	}
	if ExecutionPlaneImprovementWorkspace != "improvement_workspace" {
		t.Fatalf("ExecutionPlaneImprovementWorkspace = %q, want %q", ExecutionPlaneImprovementWorkspace, "improvement_workspace")
	}
	if ExecutionPlaneHotUpdateGate != "hot_update_gate" {
		t.Fatalf("ExecutionPlaneHotUpdateGate = %q, want %q", ExecutionPlaneHotUpdateGate, "hot_update_gate")
	}
	if ExecutionHostPhone != "phone" {
		t.Fatalf("ExecutionHostPhone = %q, want %q", ExecutionHostPhone, "phone")
	}
	if ExecutionHostDesktop != "desktop" {
		t.Fatalf("ExecutionHostDesktop = %q, want %q", ExecutionHostDesktop, "desktop")
	}
	if ExecutionHostWorkspace != "workspace" {
		t.Fatalf("ExecutionHostWorkspace = %q, want %q", ExecutionHostWorkspace, "workspace")
	}
	if JobStateRunning != "running" {
		t.Fatalf("JobStateRunning = %q, want %q", JobStateRunning, "running")
	}
	if JobStateWaitingUser != "waiting_user" {
		t.Fatalf("JobStateWaitingUser = %q, want %q", JobStateWaitingUser, "waiting_user")
	}
	if JobStatePaused != "paused" {
		t.Fatalf("JobStatePaused = %q, want %q", JobStatePaused, "paused")
	}
	if JobStateFailed != "failed" {
		t.Fatalf("JobStateFailed = %q, want %q", JobStateFailed, "failed")
	}
	if JobStateAborted != "aborted" {
		t.Fatalf("JobStateAborted = %q, want %q", JobStateAborted, "aborted")
	}

	if StepTypeDiscussion != "discussion" {
		t.Fatalf("StepTypeDiscussion = %q, want %q", StepTypeDiscussion, "discussion")
	}

	if StepTypeStaticArtifact != "static_artifact" {
		t.Fatalf("StepTypeStaticArtifact = %q, want %q", StepTypeStaticArtifact, "static_artifact")
	}

	if StepTypeOneShotCode != "one_shot_code" {
		t.Fatalf("StepTypeOneShotCode = %q, want %q", StepTypeOneShotCode, "one_shot_code")
	}
	if StepTypeLongRunningCode != "long_running_code" {
		t.Fatalf("StepTypeLongRunningCode = %q, want %q", StepTypeLongRunningCode, "long_running_code")
	}
	if StepTypeWaitUser != "wait_user" {
		t.Fatalf("StepTypeWaitUser = %q, want %q", StepTypeWaitUser, "wait_user")
	}

	if StepTypeFinalResponse != "final_response" {
		t.Fatalf("StepTypeFinalResponse = %q, want %q", StepTypeFinalResponse, "final_response")
	}

	if StepSubtypeBlocker != "blocker" {
		t.Fatalf("StepSubtypeBlocker = %q, want %q", StepSubtypeBlocker, "blocker")
	}
	if StepSubtypeAuthorization != "authorization" {
		t.Fatalf("StepSubtypeAuthorization = %q, want %q", StepSubtypeAuthorization, "authorization")
	}
	if StepSubtypeDefinition != "definition" {
		t.Fatalf("StepSubtypeDefinition = %q, want %q", StepSubtypeDefinition, "definition")
	}

	if AuthorityTierHigh != "high" {
		t.Fatalf("AuthorityTierHigh = %q, want %q", AuthorityTierHigh, "high")
	}

	if RejectionCodeApprovalRequired != "approval_required" {
		t.Fatalf("RejectionCodeApprovalRequired = %q, want %q", RejectionCodeApprovalRequired, "approval_required")
	}
	if RejectionCodeWaitingUser != "waiting_user" {
		t.Fatalf("RejectionCodeWaitingUser = %q, want %q", RejectionCodeWaitingUser, "waiting_user")
	}
	if RejectionCodeLongRunningStartForbidden != "longrun_start_forbidden" {
		t.Fatalf("RejectionCodeLongRunningStartForbidden = %q, want %q", RejectionCodeLongRunningStartForbidden, "longrun_start_forbidden")
	}
	if ApprovalStatePending != "pending" {
		t.Fatalf("ApprovalStatePending = %q, want %q", ApprovalStatePending, "pending")
	}
	if ApprovalStateGranted != "granted" {
		t.Fatalf("ApprovalStateGranted = %q, want %q", ApprovalStateGranted, "granted")
	}
	if ApprovalStateDenied != "denied" {
		t.Fatalf("ApprovalStateDenied = %q, want %q", ApprovalStateDenied, "denied")
	}
	if ApprovalStateExpired != "expired" {
		t.Fatalf("ApprovalStateExpired = %q, want %q", ApprovalStateExpired, "expired")
	}
	if ApprovalStateSuperseded != "superseded" {
		t.Fatalf("ApprovalStateSuperseded = %q, want %q", ApprovalStateSuperseded, "superseded")
	}
	if ApprovalStateRevoked != "revoked" {
		t.Fatalf("ApprovalStateRevoked = %q, want %q", ApprovalStateRevoked, "revoked")
	}
	if FrankRegistryObjectKindIdentity != "identity" {
		t.Fatalf("FrankRegistryObjectKindIdentity = %q, want %q", FrankRegistryObjectKindIdentity, "identity")
	}
	if FrankRegistryObjectKindAccount != "account" {
		t.Fatalf("FrankRegistryObjectKindAccount = %q, want %q", FrankRegistryObjectKindAccount, "account")
	}
	if FrankRegistryObjectKindContainer != "container" {
		t.Fatalf("FrankRegistryObjectKindContainer = %q, want %q", FrankRegistryObjectKindContainer, "container")
	}
}

func TestV4RejectionCodeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  RejectionCode
		want RejectionCode
	}{
		{name: "execution plane required", got: RejectionCodeV4ExecutionPlaneRequired, want: "E_EXECUTION_PLANE_REQUIRED"},
		{name: "execution host required", got: RejectionCodeV4ExecutionHostRequired, want: "E_EXECUTION_HOST_REQUIRED"},
		{name: "improvement workspace required", got: RejectionCodeV4ImprovementWorkspaceRequired, want: "E_IMPROVEMENT_WORKSPACE_REQUIRED"},
		{name: "hot update gate required", got: RejectionCodeV4HotUpdateGateRequired, want: "E_HOT_UPDATE_GATE_REQUIRED"},
		{name: "baseline required", got: RejectionCodeV4BaselineRequired, want: "E_BASELINE_REQUIRED"},
		{name: "holdout required", got: RejectionCodeV4HoldoutRequired, want: "E_HOLDOUT_REQUIRED"},
		{name: "smoke check required", got: RejectionCodeV4SmokeCheckRequired, want: "E_SMOKE_CHECK_REQUIRED"},
		{name: "eval immutable", got: RejectionCodeV4EvalImmutable, want: "E_EVAL_IMMUTABLE"},
		{name: "mutation scope violation", got: RejectionCodeV4MutationScopeViolation, want: "E_MUTATION_SCOPE_VIOLATION"},
		{name: "surface class required", got: RejectionCodeV4SurfaceClassRequired, want: "E_SURFACE_CLASS_REQUIRED"},
		{name: "forbidden surface change", got: RejectionCodeV4ForbiddenSurfaceChange, want: "E_FORBIDDEN_SURFACE_CHANGE"},
		{name: "topology change disabled", got: RejectionCodeV4TopologyChangeDisabled, want: "E_TOPOLOGY_CHANGE_DISABLED"},
		{name: "promotion policy required", got: RejectionCodeV4PromotionPolicyRequired, want: "E_PROMOTION_POLICY_REQUIRED"},
		{name: "hot update policy required", got: RejectionCodeV4HotUpdatePolicyRequired, want: "E_HOT_UPDATE_POLICY_REQUIRED"},
		{name: "canary required", got: RejectionCodeV4CanaryRequired, want: "E_CANARY_REQUIRED"},
		{name: "promotion approval required", got: RejectionCodeV4PromotionApprovalRequired, want: "E_PROMOTION_APPROVAL_REQUIRED"},
		{name: "hot update approval required", got: RejectionCodeV4HotUpdateApprovalRequired, want: "E_HOT_UPDATE_APPROVAL_REQUIRED"},
		{name: "active job deploy lock", got: RejectionCodeV4ActiveJobDeployLock, want: "E_ACTIVE_JOB_DEPLOY_LOCK"},
		{name: "pack not found", got: RejectionCodeV4PackNotFound, want: "E_PACK_NOT_FOUND"},
		{name: "last known good required", got: RejectionCodeV4LastKnownGoodRequired, want: "E_LAST_KNOWN_GOOD_REQUIRED"},
		{name: "canary failed", got: RejectionCodeV4CanaryFailed, want: "E_CANARY_FAILED"},
		{name: "smoke check failed", got: RejectionCodeV4SmokeCheckFailed, want: "E_SMOKE_CHECK_FAILED"},
		{name: "rollback required", got: RejectionCodeV4RollbackRequired, want: "E_ROLLBACK_REQUIRED"},
		{name: "promotion already applied", got: RejectionCodeV4PromotionAlreadyApplied, want: "E_PROMOTION_ALREADY_APPLIED"},
		{name: "hot update already applied", got: RejectionCodeV4HotUpdateAlreadyApplied, want: "E_HOT_UPDATE_ALREADY_APPLIED"},
		{name: "reload mode unsupported", got: RejectionCodeV4ReloadModeUnsupported, want: "E_RELOAD_MODE_UNSUPPORTED"},
		{name: "reload quiesce failed", got: RejectionCodeV4ReloadQuiesceFailed, want: "E_RELOAD_QUIESCE_FAILED"},
		{name: "extension compatibility required", got: RejectionCodeV4ExtensionCompatibilityRequired, want: "E_EXTENSION_COMPATIBILITY_REQUIRED"},
		{name: "extension permission widening", got: RejectionCodeV4ExtensionPermissionWidening, want: "E_EXTENSION_PERMISSION_WIDENING"},
		{name: "runtime source mutation forbidden", got: RejectionCodeV4RuntimeSourceMutationForbidden, want: "E_RUNTIME_SOURCE_MUTATION_FORBIDDEN"},
		{name: "policy mutation forbidden", got: RejectionCodeV4PolicyMutationForbidden, want: "E_POLICY_MUTATION_FORBIDDEN"},
		{name: "active pack adhoc mutation forbidden", got: RejectionCodeV4ActivePackAdhocMutationForbidden, want: "E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN"},
		{name: "autonomy envelope required", got: RejectionCodeV4AutonomyEnvelopeRequired, want: "E_AUTONOMY_ENVELOPE_REQUIRED"},
		{name: "standing directive required", got: RejectionCodeV4StandingDirectiveRequired, want: "E_STANDING_DIRECTIVE_REQUIRED"},
		{name: "autonomy budget exceeded", got: RejectionCodeV4AutonomyBudgetExceeded, want: "E_AUTONOMY_BUDGET_EXCEEDED"},
		{name: "no eligible autonomous action", got: RejectionCodeV4NoEligibleAutonomousAction, want: "E_NO_ELIGIBLE_AUTONOMOUS_ACTION"},
		{name: "autonomy paused", got: RejectionCodeV4AutonomyPaused, want: "E_AUTONOMY_PAUSED"},
		{name: "external action limit reached", got: RejectionCodeV4ExternalActionLimitReached, want: "E_EXTERNAL_ACTION_LIMIT_REACHED"},
		{name: "repeated failure pause", got: RejectionCodeV4RepeatedFailurePause, want: "E_REPEATED_FAILURE_PAUSE"},
		{name: "package authority grant forbidden", got: RejectionCodeV4PackageAuthorityGrantForbidden, want: "E_PACKAGE_AUTHORITY_GRANT_FORBIDDEN"},
		{name: "lab only family", got: RejectionCodeV4LabOnlyFamily, want: "E_LAB_ONLY_FAMILY"},
		{name: "mission family required", got: RejectionCodeV4MissionFamilyRequired, want: "E_MISSION_FAMILY_REQUIRED"},
		{name: "execution plane unknown", got: RejectionCodeV4ExecutionPlaneUnknown, want: "E_EXECUTION_PLANE_UNKNOWN"},
		{name: "execution host unknown", got: RejectionCodeV4ExecutionHostUnknown, want: "E_EXECUTION_HOST_UNKNOWN"},
		{name: "mission family unknown", got: RejectionCodeV4MissionFamilyUnknown, want: "E_MISSION_FAMILY_UNKNOWN"},
		{name: "execution plane incompatible", got: RejectionCodeV4ExecutionPlaneIncompatible, want: "E_EXECUTION_PLANE_INCOMPATIBLE"},
		{name: "live phone self edit forbidden", got: RejectionCodeV4LivePhoneSelfEditForbidden, want: "E_LIVE_PHONE_SELF_EDIT_FORBIDDEN"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("constant = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestJobJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := Job{
		ID:             "job-1",
		SpecVersion:    JobSpecVersionV4,
		ExecutionPlane: ExecutionPlaneLiveRuntime,
		ExecutionHost:  ExecutionHostPhone,
		MissionFamily:  MissionFamilyBootstrapRevenue,
		State:          JobStatePending,
		MaxAuthority:   AuthorityTierMedium,
		AllowedTools:   []string{"shell"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "step-1",
					Type:              StepTypeStaticArtifact,
					DependsOn:         []string{},
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"shell"},
					RequiresApproval:  false,
					SuccessCriteria:   []string{"write the report and verify the artifact"},
					FrankObjectRefs: []FrankRegistryObjectRef{
						{
							Kind:     FrankRegistryObjectKindIdentity,
							ObjectID: "identity-1",
						},
						{
							Kind:     FrankRegistryObjectKindAccount,
							ObjectID: "account-1",
						},
					},
					CampaignRef: &CampaignRef{
						CampaignID: "campaign-1",
					},
					TreasuryRef: &TreasuryRef{
						TreasuryID: "treasury-1",
					},
					StaticArtifactPath:   "dist/report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:                  "step-2",
					Type:                StepTypeOneShotCode,
					DependsOn:           []string{"step-1"},
					RequiredAuthority:   AuthorityTierLow,
					AllowedTools:        []string{"shell"},
					RequiresApproval:    false,
					SuccessCriteria:     []string{"write the code and run validation"},
					OneShotArtifactPath: "dist/main.go",
				},
			},
		},
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got Job
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch: got %#v want %#v", got, want)
	}
}

func TestNormalizeFrankRegistryObjectRefTrimsFields(t *testing.T) {
	t.Parallel()

	got := NormalizeFrankRegistryObjectRef(FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKind(" account "),
		ObjectID: " account-1 ",
	})
	want := FrankRegistryObjectRef{
		Kind:     FrankRegistryObjectKindAccount,
		ObjectID: "account-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeFrankRegistryObjectRef() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCampaignRefTrimsFields(t *testing.T) {
	t.Parallel()

	got := NormalizeCampaignRef(CampaignRef{
		CampaignID: " campaign-1 ",
	})
	want := CampaignRef{
		CampaignID: "campaign-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCampaignRef() = %#v, want %#v", got, want)
	}
}

func TestNormalizeTreasuryRefTrimsFields(t *testing.T) {
	t.Parallel()

	got := NormalizeTreasuryRef(TreasuryRef{
		TreasuryID: " treasury-1 ",
	})
	want := TreasuryRef{
		TreasuryID: "treasury-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeTreasuryRef() = %#v, want %#v", got, want)
	}
}

func TestValidationErrorErrorIsNotEmpty(t *testing.T) {
	t.Parallel()

	err := ValidationError{
		Code:    RejectionCodeToolNotAllowed,
		StepID:  "step-1",
		Message: "tool is not allowed",
	}

	if err.Error() == "" {
		t.Fatal("ValidationError.Error() returned an empty string")
	}
}

func TestSurfaceValidationErrorCanonicalizesCodeForDirectValidationError(t *testing.T) {
	t.Parallel()

	err := SurfaceValidationError(ValidationError{
		Code:    RejectionCodeUnknownStep,
		StepID:  "missing",
		Message: `step "missing" not found in plan`,
	})

	if err == nil {
		t.Fatal("SurfaceValidationError() = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "E_INVALID_ACTION_FOR_STEP") {
		t.Fatalf("SurfaceValidationError() error = %q, want canonical unknown-step code", err)
	}
	if strings.Contains(err.Error(), string(RejectionCodeUnknownStep)) {
		t.Fatalf("SurfaceValidationError() error = %q, want internal code hidden", err)
	}
}

func TestSurfaceValidationErrorPreservesWrappedContext(t *testing.T) {
	t.Parallel()

	err := SurfaceValidationError(fmt.Errorf("failed to validate mission file %q: %w", "mission.json", ValidationError{
		Code:    RejectionCodeMissingTerminalFinalStep,
		Message: "plan must end with a final_response step",
	}))

	if err == nil {
		t.Fatal("SurfaceValidationError() = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to validate mission file \"mission.json\"") {
		t.Fatalf("SurfaceValidationError() error = %q, want wrapper context preserved", err)
	}
	if !strings.Contains(err.Error(), "E_PLAN_INVALID") {
		t.Fatalf("SurfaceValidationError() error = %q, want canonical plan-invalid code", err)
	}
}
