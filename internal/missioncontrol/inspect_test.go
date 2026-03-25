package missioncontrol

import (
	"reflect"
	"testing"
)

func TestNewInspectSummaryReturnsFilteredResolvedStep(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read", "search"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read", "read"},
					SuccessCriteria:   []string{"produce code"},
				},
				{
					ID:           "final",
					Type:         StepTypeFinalResponse,
					DependsOn:    []string{"build"},
					AllowedTools: []string{"read"},
				},
			},
		},
	}

	summary, err := NewInspectSummary(job, "build")
	if err != nil {
		t.Fatalf("NewInspectSummary() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestNewInspectSummaryFromControlReturnsResolvableStep(t *testing.T) {
	t.Parallel()

	control := RuntimeControlContext{
		JobID:        "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read", "search"},
		Step: Step{
			ID:                "build",
			Type:              StepTypeOneShotCode,
			RequiredAuthority: AuthorityTierLow,
			AllowedTools:      []string{"read", "read"},
			SuccessCriteria:   []string{"produce code"},
		},
	}

	summary, err := NewInspectSummaryFromControl(control, "build")
	if err != nil {
		t.Fatalf("NewInspectSummaryFromControl() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
		t.Fatalf("Steps = %#v, want one build step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}
