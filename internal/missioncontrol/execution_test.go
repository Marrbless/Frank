package missioncontrol

import (
	"reflect"
	"testing"
)

func TestCloneExecutionContextDeepCopiesNestedData(t *testing.T) {
	t.Parallel()

	original := ExecutionContext{
		Job: &Job{
			ID:           "job-1",
			MaxAuthority: AuthorityTierHigh,
			AllowedTools: []string{"read", "write"},
			Plan: Plan{
				ID: "plan-1",
				Steps: []Step{
					{
						ID:                "build",
						Type:              StepTypeOneShotCode,
						DependsOn:         []string{"prep"},
						RequiredAuthority: AuthorityTierMedium,
						AllowedTools:      []string{"read"},
						RequiresApproval:  true,
						SuccessCriteria:   []string{"produce code"},
						CampaignRef: &CampaignRef{
							CampaignID: "campaign-1",
						},
						TreasuryRef: &TreasuryRef{
							TreasuryID: "treasury-1",
						},
					},
				},
			},
		},
		Step: &Step{
			ID:                "build",
			Type:              StepTypeOneShotCode,
			DependsOn:         []string{"prep"},
			RequiredAuthority: AuthorityTierMedium,
			AllowedTools:      []string{"read"},
			RequiresApproval:  true,
			SuccessCriteria:   []string{"produce code"},
			FrankObjectRefs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindIdentity,
					ObjectID: "identity-1",
				},
			},
			CampaignRef: &CampaignRef{
				CampaignID: "campaign-1",
			},
			TreasuryRef: &TreasuryRef{
				TreasuryID: "treasury-1",
			},
		},
		MissionStoreRoot: "/tmp/mission-store",
		GovernedExternalTargets: []AutonomyEligibilityTargetRef{
			{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "provider-mail",
			},
		},
	}

	cloned := CloneExecutionContext(original)
	if !reflect.DeepEqual(cloned, original) {
		t.Fatalf("CloneExecutionContext() = %#v, want %#v", cloned, original)
	}

	cloned.Job.AllowedTools[0] = "mutated-job-tool"
	cloned.Job.Plan.Steps[0].DependsOn[0] = "mutated-dependency"
	cloned.Job.Plan.Steps[0].AllowedTools[0] = "mutated-step-tool"
	cloned.Job.Plan.Steps[0].CampaignRef.CampaignID = "mutated-plan-campaign"
	cloned.Job.Plan.Steps[0].TreasuryRef.TreasuryID = "mutated-plan-treasury"
	cloned.Step.SuccessCriteria[0] = "mutated-success"
	cloned.Step.FrankObjectRefs[0].ObjectID = "mutated-object"
	cloned.Step.CampaignRef.CampaignID = "mutated-step-campaign"
	cloned.Step.TreasuryRef.TreasuryID = "mutated-step-treasury"
	cloned.GovernedExternalTargets[0].RegistryID = "mutated-target"

	if original.Job.AllowedTools[0] != "read" {
		t.Fatalf("original Job.AllowedTools[0] = %q, want %q", original.Job.AllowedTools[0], "read")
	}

	if original.Job.Plan.Steps[0].DependsOn[0] != "prep" {
		t.Fatalf("original Job.Plan.Steps[0].DependsOn[0] = %q, want %q", original.Job.Plan.Steps[0].DependsOn[0], "prep")
	}

	if original.Job.Plan.Steps[0].AllowedTools[0] != "read" {
		t.Fatalf("original Job.Plan.Steps[0].AllowedTools[0] = %q, want %q", original.Job.Plan.Steps[0].AllowedTools[0], "read")
	}

	if original.Step.SuccessCriteria[0] != "produce code" {
		t.Fatalf("original Step.SuccessCriteria[0] = %q, want %q", original.Step.SuccessCriteria[0], "produce code")
	}
	if original.Step.FrankObjectRefs[0].ObjectID != "identity-1" {
		t.Fatalf("original Step.FrankObjectRefs[0].ObjectID = %q, want %q", original.Step.FrankObjectRefs[0].ObjectID, "identity-1")
	}
	if original.Job.Plan.Steps[0].CampaignRef == nil || original.Job.Plan.Steps[0].CampaignRef.CampaignID != "campaign-1" {
		t.Fatalf("original Job.Plan.Steps[0].CampaignRef = %#v, want campaign-1", original.Job.Plan.Steps[0].CampaignRef)
	}
	if original.Step.CampaignRef == nil || original.Step.CampaignRef.CampaignID != "campaign-1" {
		t.Fatalf("original Step.CampaignRef = %#v, want campaign-1", original.Step.CampaignRef)
	}
	if original.Job.Plan.Steps[0].TreasuryRef == nil || original.Job.Plan.Steps[0].TreasuryRef.TreasuryID != "treasury-1" {
		t.Fatalf("original Job.Plan.Steps[0].TreasuryRef = %#v, want treasury-1", original.Job.Plan.Steps[0].TreasuryRef)
	}
	if original.Step.TreasuryRef == nil || original.Step.TreasuryRef.TreasuryID != "treasury-1" {
		t.Fatalf("original Step.TreasuryRef = %#v, want treasury-1", original.Step.TreasuryRef)
	}

	if original.GovernedExternalTargets[0].RegistryID != "provider-mail" {
		t.Fatalf("original GovernedExternalTargets[0].RegistryID = %q, want %q", original.GovernedExternalTargets[0].RegistryID, "provider-mail")
	}
}

func TestCloneExecutionContextZeroValue(t *testing.T) {
	t.Parallel()

	if got := CloneExecutionContext(ExecutionContext{}); !reflect.DeepEqual(got, ExecutionContext{}) {
		t.Fatalf("CloneExecutionContext() = %#v, want zero value", got)
	}
}

func TestResolveExecutionContextValidPlan(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	if ec.Job == nil {
		t.Fatal("ResolveExecutionContext().Job = nil, want non-nil")
	}

	if ec.Step == nil {
		t.Fatal("ResolveExecutionContext().Step = nil, want non-nil")
	}

	if ec.Job.ID != job.ID {
		t.Fatalf("ResolveExecutionContext().Job.ID = %q, want %q", ec.Job.ID, job.ID)
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ResolveExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want nil for zero-target step", ec.GovernedExternalTargets)
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want nil for zero-ref step", ec.Step.FrankObjectRefs)
	}
	if ec.Step.CampaignRef != nil {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want nil for zero-campaign step", ec.Step.CampaignRef)
	}
	if ec.Step.TreasuryRef != nil {
		t.Fatalf("ResolveExecutionContext().Step.TreasuryRef = %#v, want nil for zero-treasury step", ec.Step.TreasuryRef)
	}
}

func TestResolveExecutionContextCarriesStepControlPlaneRefsAndNormalizedIdentityMode(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	job.Plan.Steps[0].GovernedExternalTargets = []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	job.Plan.Steps[0].FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKind(" identity "),
			ObjectID: " identity-1 ",
		},
	}
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: " campaign-1 ",
	}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: " treasury-1 ",
	}
	job.Plan.Steps[0].IdentityMode = IdentityMode("   ")

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	want := []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	if !reflect.DeepEqual(ec.GovernedExternalTargets, want) {
		t.Fatalf("ResolveExecutionContext().GovernedExternalTargets = %#v, want %#v", ec.GovernedExternalTargets, want)
	}
	if !reflect.DeepEqual(ec.Step.GovernedExternalTargets, want) {
		t.Fatalf("ResolveExecutionContext().Step.GovernedExternalTargets = %#v, want %#v", ec.Step.GovernedExternalTargets, want)
	}
	wantRefs := []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-1",
		},
	}
	if !reflect.DeepEqual(ec.Step.FrankObjectRefs, wantRefs) {
		t.Fatalf("ResolveExecutionContext().Step.FrankObjectRefs = %#v, want %#v", ec.Step.FrankObjectRefs, wantRefs)
	}
	wantCampaignRef := &CampaignRef{CampaignID: "campaign-1"}
	if !reflect.DeepEqual(ec.Step.CampaignRef, wantCampaignRef) {
		t.Fatalf("ResolveExecutionContext().Step.CampaignRef = %#v, want %#v", ec.Step.CampaignRef, wantCampaignRef)
	}
	wantTreasuryRef := &TreasuryRef{TreasuryID: "treasury-1"}
	if !reflect.DeepEqual(ec.Step.TreasuryRef, wantTreasuryRef) {
		t.Fatalf("ResolveExecutionContext().Step.TreasuryRef = %#v, want %#v", ec.Step.TreasuryRef, wantTreasuryRef)
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
}

func TestResolveExecutionContextInvalidPlanReturnsFirstValidationError(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
		},
	}

	want := ValidatePlan(job)[0]

	_, err := ResolveExecutionContext(job, "missing")
	if err == nil {
		t.Fatal("ResolveExecutionContext() error = nil, want ValidationError")
	}

	got, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContext() error type = %T, want ValidationError", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExecutionContext() error = %#v, want %#v", got, want)
	}
}

func TestResolveExecutionContextMissingStep(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	_, err := ResolveExecutionContext(job, "missing")
	if err == nil {
		t.Fatal("ResolveExecutionContext() error = nil, want ValidationError")
	}

	got, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContext() error type = %T, want ValidationError", err)
	}

	want := ValidationError{
		Code:    RejectionCodeUnknownStep,
		StepID:  "missing",
		Message: `step "missing" not found in plan`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExecutionContext() error = %#v, want %#v", got, want)
	}
}

func TestResolveExecutionContextReturnsIndependentCopies(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	job.AllowedTools[0] = "mutated-job-tool"
	job.Plan.Steps[0].ID = "mutated-step-id"
	job.Plan.Steps[0].AllowedTools[0] = "mutated-step-tool"
	job.Plan.Steps[0].SuccessCriteria[0] = "mutated-success"

	if ec.Job.AllowedTools[0] != "read" {
		t.Fatalf("ResolveExecutionContext().Job.AllowedTools[0] = %q, want %q", ec.Job.AllowedTools[0], "read")
	}

	if ec.Step.ID != "build" {
		t.Fatalf("ResolveExecutionContext().Step.ID = %q, want %q", ec.Step.ID, "build")
	}

	if ec.Step.AllowedTools[0] != "read" {
		t.Fatalf("ResolveExecutionContext().Step.AllowedTools[0] = %q, want %q", ec.Step.AllowedTools[0], "read")
	}

	if ec.Step.SuccessCriteria[0] != "produce code" {
		t.Fatalf("ResolveExecutionContext().Step.SuccessCriteria[0] = %q, want %q", ec.Step.SuccessCriteria[0], "produce code")
	}
}

func TestCloneJobReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	cloned := CloneJob(&job)
	if cloned == nil {
		t.Fatal("CloneJob() = nil, want non-nil copy")
	}

	job.AllowedTools[0] = "mutated-job-tool"
	job.Plan.Steps[0].ID = "mutated-step-id"
	job.Plan.Steps[0].AllowedTools[0] = "mutated-step-tool"
	job.Plan.Steps[0].SuccessCriteria[0] = "mutated-success"

	if cloned.AllowedTools[0] != "read" {
		t.Fatalf("CloneJob().AllowedTools[0] = %q, want %q", cloned.AllowedTools[0], "read")
	}
	if cloned.Plan.Steps[0].ID != "build" {
		t.Fatalf("CloneJob().Plan.Steps[0].ID = %q, want %q", cloned.Plan.Steps[0].ID, "build")
	}
	if cloned.Plan.Steps[0].AllowedTools[0] != "read" {
		t.Fatalf("CloneJob().Plan.Steps[0].AllowedTools[0] = %q, want %q", cloned.Plan.Steps[0].AllowedTools[0], "read")
	}
	if cloned.Plan.Steps[0].SuccessCriteria[0] != "produce code" {
		t.Fatalf("CloneJob().Plan.Steps[0].SuccessCriteria[0] = %q, want %q", cloned.Plan.Steps[0].SuccessCriteria[0], "produce code")
	}
}

func testExecutionJob() Job {
	return Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierMedium,
					AllowedTools:      []string{"read"},
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
}
