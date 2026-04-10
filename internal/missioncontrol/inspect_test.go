package missioncontrol

import (
	"encoding/json"
	"reflect"
	"strings"
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

func TestNewInspectSummaryFromInspectablePlanReturnsResolvableStep(t *testing.T) {
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
	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	summary, err := NewInspectSummaryFromInspectablePlan(job.ID, &plan, "final")
	if err != nil {
		t.Fatalf("NewInspectSummaryFromInspectablePlan() error = %v", err)
	}
	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if len(summary.Steps) != 1 || summary.Steps[0].StepID != "final" {
		t.Fatalf("Steps = %#v, want one final step", summary.Steps)
	}
	if !reflect.DeepEqual(summary.Steps[0].EffectiveAllowedTools, []string{"read"}) {
		t.Fatalf("EffectiveAllowedTools = %#v, want %#v", summary.Steps[0].EffectiveAllowedTools, []string{"read"})
	}
}

func TestInspectSummariesDoNotImplicitlySurfaceAdapterOnlyCampaignOrTreasuryFields(t *testing.T) {
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
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"produce code"},
					CampaignRef:       &CampaignRef{CampaignID: "campaign-mail"},
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	tests := []struct {
		name string
		run  func() (InspectSummary, error)
	}{
		{
			name: "job",
			run: func() (InspectSummary, error) {
				return NewInspectSummary(job, "build")
			},
		},
		{
			name: "inspectable_plan",
			run: func() (InspectSummary, error) {
				return NewInspectSummaryFromInspectablePlan(job.ID, &plan, "build")
			},
		},
	}

	forbidden := []string{
		"\"treasury_preflight\"",
		"\"audience_class_or_target\"",
		"\"message_family_or_participation_style\"",
		"\"cadence\"",
		"\"escalation_rules\"",
		"\"budget\":",
		"\"active_container_id\"",
		"\"custody_model\"",
		"\"permitted_transaction_classes\"",
		"\"forbidden_transaction_classes\"",
		"\"ledger_ref\"",
		"\"direction\":\"internal\"",
		"\"status\":\"recorded\"",
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			summary, err := tc.run()
			if err != nil {
				t.Fatalf("inspect summary error = %v", err)
			}
			if len(summary.Steps) != 1 || summary.Steps[0].StepID != "build" {
				t.Fatalf("Steps = %#v, want one build step", summary.Steps)
			}
			if summary.Steps[0].TreasuryPreflight != nil {
				t.Fatalf("TreasuryPreflight = %#v, want nil for implicit inspect surfaces", summary.Steps[0].TreasuryPreflight)
			}

			data, err := json.Marshal(summary)
			if err != nil {
				t.Fatalf("json.Marshal(summary) error = %v", err)
			}
			for _, key := range forbidden {
				if strings.Contains(string(data), key) {
					t.Fatalf("inspect JSON unexpectedly contains %s: %s", key, string(data))
				}
			}
		})
	}
}
