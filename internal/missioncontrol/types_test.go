package missioncontrol

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEnumValues(t *testing.T) {
	t.Parallel()

	if JobStateRunning != "running" {
		t.Fatalf("JobStateRunning = %q, want %q", JobStateRunning, "running")
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

	if StepTypeFinalResponse != "final_response" {
		t.Fatalf("StepTypeFinalResponse = %q, want %q", StepTypeFinalResponse, "final_response")
	}

	if AuthorityTierHigh != "high" {
		t.Fatalf("AuthorityTierHigh = %q, want %q", AuthorityTierHigh, "high")
	}

	if RejectionCodeApprovalRequired != "approval_required" {
		t.Fatalf("RejectionCodeApprovalRequired = %q, want %q", RejectionCodeApprovalRequired, "approval_required")
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
					Type:              StepTypeDiscussion,
					DependsOn:         []string{},
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"shell"},
					RequiresApproval:  false,
					SuccessCriteria:   []string{"share a concise plan"},
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
