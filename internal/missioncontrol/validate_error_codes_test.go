package missioncontrol

import (
	"strings"
	"testing"
)

func TestValidatePlanOperatorFacingErrorCodeContracts(t *testing.T) {
	tests := []struct {
		name string
		job  Job
		want []expectedValidationError
	}{
		{
			name: "empty plan reports terminal final response code",
			job: Job{
				ID:           "job-empty",
				MaxAuthority: AuthorityTierLow,
			},
			want: []expectedValidationError{{
				code:            RejectionCodeMissingTerminalFinalStep,
				messageContains: "terminal final_response",
			}},
		},
		{
			name: "duplicate step reports duplicate step code and step id",
			job: Job{
				ID:           "job-duplicate",
				MaxAuthority: AuthorityTierLow,
				Plan: Plan{Steps: []Step{
					{ID: "draft", Type: StepTypeDiscussion, DependsOn: []string{"final"}},
					{ID: "draft", Type: StepTypeDiscussion, DependsOn: []string{"final"}},
					{ID: "final", Type: StepTypeFinalResponse},
				}},
			},
			want: []expectedValidationError{{
				code:            RejectionCodeDuplicateStepID,
				stepID:          "draft",
				messageContains: "duplicate step ID",
			}},
		},
		{
			name: "frank v4 metadata omissions use stable uppercase codes",
			job: Job{
				ID:           "job-v4-missing-metadata",
				SpecVersion:  JobSpecVersionV4,
				MaxAuthority: AuthorityTierLow,
				Plan: Plan{Steps: []Step{
					{ID: "final", Type: StepTypeFinalResponse},
				}},
			},
			want: []expectedValidationError{
				{code: RejectionCodeV4ExecutionPlaneRequired, messageContains: "requires execution_plane"},
				{code: RejectionCodeV4ExecutionHostRequired, messageContains: "requires execution_host"},
				{code: RejectionCodeV4MissionFamilyRequired, messageContains: "requires mission_family"},
			},
		},
		{
			name: "long running start intent uses dedicated forbidden code",
			job: Job{
				ID:           "job-longrun",
				MaxAuthority: AuthorityTierLow,
				Plan: Plan{Steps: []Step{
					{
						ID:                        "worker",
						Type:                      StepTypeLongRunningCode,
						DependsOn:                 []string{"final"},
						LongRunningStartupCommand: []string{"service", "start"},
						LongRunningArtifactPath:   "worker.log",
						SuccessCriteria:           []string{"start the service and keep it running"},
					},
					{ID: "final", Type: StepTypeFinalResponse},
				}},
			},
			want: []expectedValidationError{{
				code:            RejectionCodeLongRunningStartForbidden,
				stepID:          "worker",
				messageContains: "must not start a process",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePlan(tt.job)
			for _, want := range tt.want {
				assertValidationErrorContract(t, errs, want)
			}
		})
	}
}

type expectedValidationError struct {
	code            RejectionCode
	stepID          string
	messageContains string
}

func assertValidationErrorContract(t *testing.T, errs []ValidationError, want expectedValidationError) {
	t.Helper()
	for _, err := range errs {
		if err.Code != want.code {
			continue
		}
		if want.stepID != "" && err.StepID != want.stepID {
			continue
		}
		if want.messageContains != "" && !strings.Contains(err.Message, want.messageContains) {
			t.Fatalf("ValidationError for code %q message = %q, want substring %q", want.code, err.Message, want.messageContains)
		}
		return
	}
	t.Fatalf("ValidatePlan() errors = %#v, missing code=%q step_id=%q", errs, want.code, want.stepID)
}
