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

func TestJobJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := Job{
		ID:           "job-1",
		State:        JobStatePending,
		MaxAuthority: AuthorityTierMedium,
		AllowedTools: []string{"shell"},
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
