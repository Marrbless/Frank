package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestSetJobRuntimeActiveStepStartsRunningRuntime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	job := testExecutionJob()

	runtime, err := SetJobRuntimeActiveStep(job, nil, "build", now)
	if err != nil {
		t.Fatalf("SetJobRuntimeActiveStep() error = %v", err)
	}

	if runtime.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", runtime.JobID, job.ID)
	}
	if runtime.State != JobStateRunning {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateRunning)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.CreatedAt != now || runtime.UpdatedAt != now || runtime.StartedAt != now || runtime.ActiveStepAt != now {
		t.Fatalf("timestamps = %#v, want all primary timestamps set to %v", runtime, now)
	}
}

func TestBuildRuntimeControlContextCapturesMinimalStepBinding(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if control.JobID != job.ID {
		t.Fatalf("JobID = %q, want %q", control.JobID, job.ID)
	}
	if control.Step.ID != "build" {
		t.Fatalf("Step.ID = %q, want %q", control.Step.ID, "build")
	}
	if !reflect.DeepEqual(control.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %#v, want %#v", control.AllowedTools, job.AllowedTools)
	}
	if control.Step.GovernedExternalTargets != nil {
		t.Fatalf("Step.GovernedExternalTargets = %#v, want nil for zero-target step", control.Step.GovernedExternalTargets)
	}
	if control.Step.FrankObjectRefs != nil {
		t.Fatalf("Step.FrankObjectRefs = %#v, want nil for zero-ref step", control.Step.FrankObjectRefs)
	}
	if control.Step.CampaignRef != nil {
		t.Fatalf("Step.CampaignRef = %#v, want nil for zero-campaign step", control.Step.CampaignRef)
	}
	if control.Step.TreasuryRef != nil {
		t.Fatalf("Step.TreasuryRef = %#v, want nil for zero-treasury step", control.Step.TreasuryRef)
	}
	if control.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("Step.IdentityMode = %q, want %q", control.Step.IdentityMode, IdentityModeAgentAlias)
	}
}

func TestRuntimeContextsCarryPromotionPolicyID(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 30, 13, 0, 0, 0, time.UTC)
	job := testV4Job(ExecutionPlaneImprovementWorkspace, ExecutionHostWorkspace, MissionFamilyImprovePromptpack)

	runtime, err := SetJobRuntimeActiveStep(job, nil, "build", now)
	if err != nil {
		t.Fatalf("SetJobRuntimeActiveStep() error = %v", err)
	}
	if runtime.PromotionPolicyID != "promotion-policy-1" {
		t.Fatalf("runtime.PromotionPolicyID = %q, want promotion-policy-1", runtime.PromotionPolicyID)
	}
	if runtime.BaselineRef != "evidence/baseline" {
		t.Fatalf("runtime.BaselineRef = %q, want evidence/baseline", runtime.BaselineRef)
	}
	if runtime.TrainRef != "evidence/train" {
		t.Fatalf("runtime.TrainRef = %q, want evidence/train", runtime.TrainRef)
	}
	if runtime.HoldoutRef != "evidence/holdout" {
		t.Fatalf("runtime.HoldoutRef = %q, want evidence/holdout", runtime.HoldoutRef)
	}

	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if control.PromotionPolicyID != "promotion-policy-1" {
		t.Fatalf("control.PromotionPolicyID = %q, want promotion-policy-1", control.PromotionPolicyID)
	}
	if control.BaselineRef != "evidence/baseline" {
		t.Fatalf("control.BaselineRef = %q, want evidence/baseline", control.BaselineRef)
	}
	if control.TrainRef != "evidence/train" {
		t.Fatalf("control.TrainRef = %q, want evidence/train", control.TrainRef)
	}
	if control.HoldoutRef != "evidence/holdout" {
		t.Fatalf("control.HoldoutRef = %q, want evidence/holdout", control.HoldoutRef)
	}

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.PromotionPolicyID != "promotion-policy-1" {
		t.Fatalf("plan.PromotionPolicyID = %q, want promotion-policy-1", plan.PromotionPolicyID)
	}
	if plan.BaselineRef != "evidence/baseline" {
		t.Fatalf("plan.BaselineRef = %q, want evidence/baseline", plan.BaselineRef)
	}
	if plan.TrainRef != "evidence/train" {
		t.Fatalf("plan.TrainRef = %q, want evidence/train", plan.TrainRef)
	}
	if plan.HoldoutRef != "evidence/holdout" {
		t.Fatalf("plan.HoldoutRef = %q, want evidence/holdout", plan.HoldoutRef)
	}
}

func TestTreasuryRegistryScaffoldingDoesNotAlterCurrentV2RuntimePath(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-v2",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID: "plan-v2",
			Steps: []Step{
				{
					ID:                   "artifact",
					Type:                 StepTypeStaticArtifact,
					AllowedTools:         []string{"read"},
					SuccessCriteria:      []string{"write report"},
					StaticArtifactPath:   "report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"artifact"},
				},
			},
		},
	}

	control, err := BuildRuntimeControlContext(job, "artifact")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	if control.Step.GovernedExternalTargets != nil {
		t.Fatalf("BuildRuntimeControlContext().Step.GovernedExternalTargets = %#v, want nil", control.Step.GovernedExternalTargets)
	}
	if control.Step.FrankObjectRefs != nil {
		t.Fatalf("BuildRuntimeControlContext().Step.FrankObjectRefs = %#v, want nil", control.Step.FrankObjectRefs)
	}
	if control.Step.CampaignRef != nil {
		t.Fatalf("BuildRuntimeControlContext().Step.CampaignRef = %#v, want nil", control.Step.CampaignRef)
	}
	if control.Step.TreasuryRef != nil {
		t.Fatalf("BuildRuntimeControlContext().Step.TreasuryRef = %#v, want nil", control.Step.TreasuryRef)
	}
	if control.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("BuildRuntimeControlContext().Step.IdentityMode = %q, want %q", control.Step.IdentityMode, IdentityModeAgentAlias)
	}

	ec, err := ResolveExecutionContextWithRuntimeControl(control, JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "artifact",
	})
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error = %v", err)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl().GovernedExternalTargets = %#v, want nil", ec.GovernedExternalTargets)
	}
	if ec.Step == nil {
		t.Fatal("ResolveExecutionContextWithRuntimeControl().Step = nil, want active step")
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl().Step.FrankObjectRefs = %#v, want nil", ec.Step.FrankObjectRefs)
	}
	if ec.Step.CampaignRef != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl().Step.CampaignRef = %#v, want nil", ec.Step.CampaignRef)
	}
	if ec.Step.TreasuryRef != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl().Step.TreasuryRef = %#v, want nil", ec.Step.TreasuryRef)
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
}

func TestBuildRuntimeControlContextCarriesDeclaredControlPlaneRefsAndNormalizedIdentityMode(t *testing.T) {
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
			Kind:     FrankRegistryObjectKind(" account "),
			ObjectID: " account-1 ",
		},
	}
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: " campaign-1 ",
	}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: " treasury-1 ",
	}
	job.Plan.Steps[0].IdentityMode = IdentityMode(" agent_alias ")

	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	want := []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	if !reflect.DeepEqual(control.Step.GovernedExternalTargets, want) {
		t.Fatalf("BuildRuntimeControlContext().Step.GovernedExternalTargets = %#v, want %#v", control.Step.GovernedExternalTargets, want)
	}
	wantRefs := []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindAccount,
			ObjectID: "account-1",
		},
	}
	if !reflect.DeepEqual(control.Step.FrankObjectRefs, wantRefs) {
		t.Fatalf("BuildRuntimeControlContext().Step.FrankObjectRefs = %#v, want %#v", control.Step.FrankObjectRefs, wantRefs)
	}
	wantCampaignRef := &CampaignRef{CampaignID: "campaign-1"}
	if !reflect.DeepEqual(control.Step.CampaignRef, wantCampaignRef) {
		t.Fatalf("BuildRuntimeControlContext().Step.CampaignRef = %#v, want %#v", control.Step.CampaignRef, wantCampaignRef)
	}
	wantTreasuryRef := &TreasuryRef{TreasuryID: "treasury-1"}
	if !reflect.DeepEqual(control.Step.TreasuryRef, wantTreasuryRef) {
		t.Fatalf("BuildRuntimeControlContext().Step.TreasuryRef = %#v, want %#v", control.Step.TreasuryRef, wantTreasuryRef)
	}
	if control.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("BuildRuntimeControlContext().Step.IdentityMode = %q, want %q", control.Step.IdentityMode, IdentityModeAgentAlias)
	}
}

func TestBuildInspectablePlanContextCapturesValidatedPlan(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.MaxAuthority != job.MaxAuthority {
		t.Fatalf("MaxAuthority = %q, want %q", plan.MaxAuthority, job.MaxAuthority)
	}
	if !reflect.DeepEqual(plan.AllowedTools, job.AllowedTools) {
		t.Fatalf("AllowedTools = %#v, want %#v", plan.AllowedTools, job.AllowedTools)
	}
	if len(plan.Steps) != len(job.Plan.Steps) {
		t.Fatalf("len(Steps) = %d, want %d", len(plan.Steps), len(job.Plan.Steps))
	}
	if plan.Steps[1].ID != "final" {
		t.Fatalf("Steps[1].ID = %q, want %q", plan.Steps[1].ID, "final")
	}
}

func TestBuildInspectablePlanContextPreservesStaticArtifactContractMetadata(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                   "artifact",
					Type:                 StepTypeStaticArtifact,
					SuccessCriteria:      []string{"write a report"},
					StaticArtifactPath:   "report.json",
					StaticArtifactFormat: "json",
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"artifact"},
				},
			},
		},
	}

	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}
	if plan.Steps[0].StaticArtifactPath != "report.json" {
		t.Fatalf("Steps[0].StaticArtifactPath = %q, want %q", plan.Steps[0].StaticArtifactPath, "report.json")
	}
	if plan.Steps[0].StaticArtifactFormat != "json" {
		t.Fatalf("Steps[0].StaticArtifactFormat = %q, want %q", plan.Steps[0].StaticArtifactFormat, "json")
	}
}

func TestBuildInspectablePlanContextPreservesOneShotCodeContractMetadata(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                  "build",
					Type:                StepTypeOneShotCode,
					SuccessCriteria:     []string{"write code"},
					OneShotArtifactPath: "main.go",
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
	if plan.Steps[0].OneShotArtifactPath != "main.go" {
		t.Fatalf("Steps[0].OneShotArtifactPath = %q, want %q", plan.Steps[0].OneShotArtifactPath, "main.go")
	}
}

func TestResolveExecutionContextWithRuntimeControlReconstructsExecutionContext(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	ec, err := ResolveExecutionContextWithRuntimeControl(control, runtime)
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error = %v", err)
	}
	if ec.Job == nil || ec.Job.ID != job.ID {
		t.Fatalf("ExecutionContext.Job = %#v, want job %q", ec.Job, job.ID)
	}
	if ec.Step == nil || ec.Step.ID != "build" {
		t.Fatalf("ExecutionContext.Step = %#v, want build", ec.Step)
	}
	if ec.Runtime == nil || ec.Runtime.State != JobStateRunning {
		t.Fatalf("ExecutionContext.Runtime = %#v, want running runtime", ec.Runtime)
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ExecutionContext.Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
	if ec.GovernedExternalTargets != nil {
		t.Fatalf("ExecutionContext.GovernedExternalTargets = %#v, want nil for zero-target step", ec.GovernedExternalTargets)
	}
	if ec.Step.FrankObjectRefs != nil {
		t.Fatalf("ExecutionContext.Step.FrankObjectRefs = %#v, want nil for zero-ref step", ec.Step.FrankObjectRefs)
	}
	if ec.Step.CampaignRef != nil {
		t.Fatalf("ExecutionContext.Step.CampaignRef = %#v, want nil for zero-campaign step", ec.Step.CampaignRef)
	}
	if ec.Step.TreasuryRef != nil {
		t.Fatalf("ExecutionContext.Step.TreasuryRef = %#v, want nil for zero-treasury step", ec.Step.TreasuryRef)
	}
}

func TestResolveExecutionContextWithRuntimeControlCarriesDeclaredControlPlaneRefsAndNormalizedIdentityMode(t *testing.T) {
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
			Kind:     FrankRegistryObjectKind(" container "),
			ObjectID: " container-1 ",
		},
	}
	job.Plan.Steps[0].CampaignRef = &CampaignRef{
		CampaignID: " campaign-1 ",
	}
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{
		TreasuryID: " treasury-1 ",
	}
	job.Plan.Steps[0].IdentityMode = IdentityMode("  ")
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	ec, err := ResolveExecutionContextWithRuntimeControl(control, runtime)
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error = %v", err)
	}

	want := []AutonomyEligibilityTargetRef{
		{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-mail",
		},
	}
	if !reflect.DeepEqual(ec.GovernedExternalTargets, want) {
		t.Fatalf("ExecutionContext.GovernedExternalTargets = %#v, want %#v", ec.GovernedExternalTargets, want)
	}
	wantRefs := []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindContainer,
			ObjectID: "container-1",
		},
	}
	if !reflect.DeepEqual(ec.Step.FrankObjectRefs, wantRefs) {
		t.Fatalf("ExecutionContext.Step.FrankObjectRefs = %#v, want %#v", ec.Step.FrankObjectRefs, wantRefs)
	}
	wantCampaignRef := &CampaignRef{CampaignID: "campaign-1"}
	if !reflect.DeepEqual(ec.Step.CampaignRef, wantCampaignRef) {
		t.Fatalf("ExecutionContext.Step.CampaignRef = %#v, want %#v", ec.Step.CampaignRef, wantCampaignRef)
	}
	wantTreasuryRef := &TreasuryRef{TreasuryID: "treasury-1"}
	if !reflect.DeepEqual(ec.Step.TreasuryRef, wantTreasuryRef) {
		t.Fatalf("ExecutionContext.Step.TreasuryRef = %#v, want %#v", ec.Step.TreasuryRef, wantTreasuryRef)
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ExecutionContext.Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}
}

func TestResolveExecutionContextWithRuntimeControlRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}

	_, err = ResolveExecutionContextWithRuntimeControl(control, JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	})
	if err == nil {
		t.Fatal("ResolveExecutionContextWithRuntimeControl() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContextWithRuntimeControl() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestTransitionJobRuntimeCompletedRequiresValidation(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC),
	}

	_, err := TransitionJobRuntime(current, JobStateCompleted, time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC), RuntimeTransitionOptions{})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want validation failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeValidationRequired {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeValidationRequired)
	}
}

func TestTransitionJobRuntimeRejectsCompletedReplayMarkerWhenRecordingCompletion(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	_, err := TransitionJobRuntime(current, JobStateCompleted, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), RuntimeTransitionOptions{
		validationResult: &stepValidationResult{recordCompletion: true},
	})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestTransitionJobRuntimeRejectsFailedReplayMarkerWhenRecordingFailure(t *testing.T) {
	t.Parallel()

	current := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}

	_, err := TransitionJobRuntime(current, JobStateFailed, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), RuntimeTransitionOptions{
		FailureReason: "validator failed",
	})
	if err == nil {
		t.Fatal("TransitionJobRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("TransitionJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumeJobRuntimeAfterBootRequiresApproval(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}, time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC), false)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want approval failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeResumeApprovalRequired {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeResumeApprovalRequired)
	}
}

func TestResumeJobRuntimeAfterBootRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), true)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumeJobRuntimeAfterBootRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumeJobRuntimeAfterBoot(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC), true)
	if err == nil {
		t.Fatal("ResumeJobRuntimeAfterBoot() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumeJobRuntimeAfterBoot() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestPauseJobRuntimeDoesNotCompleteActiveStep(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	runtime, err := PauseJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
		StartedAt:    time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
		ActiveStepAt: time.Date(2026, 3, 24, 11, 30, 0, 0, time.UTC),
	}, now)
	if err != nil {
		t.Fatalf("PauseJobRuntime() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.PausedReason != RuntimePauseReasonOperatorCommand {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonOperatorCommand)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestPauseJobRuntimeForBudgetExhaustionPersistsBudgetBlockerAndAudit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime, err := PauseJobRuntimeForBudgetExhaustion(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "final",
		WaitingReason: "discussion_authorization",
		WaitingAt:     time.Date(2026, 3, 28, 11, 58, 0, 0, time.UTC),
	}, now, RuntimeBudgetBlockerRecord{
		Ceiling:  "owner_messages",
		Limit:    20,
		Observed: 20,
		Message:  "owner-facing message budget exhausted",
	})
	if err != nil {
		t.Fatalf("PauseJobRuntimeForBudgetExhaustion() error = %v", err)
	}

	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.PausedReason != RuntimePauseReasonBudgetExhausted {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonBudgetExhausted)
	}
	if runtime.WaitingReason != "" {
		t.Fatalf("WaitingReason = %q, want empty after budget pause", runtime.WaitingReason)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("BudgetBlocker = nil, want persisted blocker")
	}
	if runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, "owner_messages")
	}
	if runtime.BudgetBlocker.Limit != 20 || runtime.BudgetBlocker.Observed != 20 {
		t.Fatalf("BudgetBlocker limits = %#v, want limit=20 observed=20", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Message != "owner-facing message budget exhausted" {
		t.Fatalf("BudgetBlocker.Message = %q, want exact blocker message", runtime.BudgetBlocker.Message)
	}
	if runtime.BudgetBlocker.TriggeredAt != now {
		t.Fatalf("BudgetBlocker.TriggeredAt = %v, want %v", runtime.BudgetBlocker.TriggeredAt, now)
	}
	if len(runtime.AuditHistory) != 1 {
		t.Fatalf("AuditHistory count = %d, want 1", len(runtime.AuditHistory))
	}
	if runtime.AuditHistory[0].ToolName != "budget_exhausted" {
		t.Fatalf("AuditHistory[0].ToolName = %q, want %q", runtime.AuditHistory[0].ToolName, "budget_exhausted")
	}
	if runtime.AuditHistory[0].ActionClass != AuditActionClassRuntime {
		t.Fatalf("AuditHistory[0].ActionClass = %q, want %q", runtime.AuditHistory[0].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[0].Result != AuditResultApplied {
		t.Fatalf("AuditHistory[0].Result = %q, want %q", runtime.AuditHistory[0].Result, AuditResultApplied)
	}
	if !runtime.AuditHistory[0].Allowed {
		t.Fatal("AuditHistory[0].Allowed = false, want true")
	}
	if runtime.AuditHistory[0].StepID != "final" {
		t.Fatalf("AuditHistory[0].StepID = %q, want %q", runtime.AuditHistory[0].StepID, "final")
	}
}

func TestPauseJobRuntimeForUnattendedWallClockPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime, exhausted, err := PauseJobRuntimeForUnattendedWallClock(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-5 * time.Hour),
		StartedAt:    now.Add(-5 * time.Hour),
		ActiveStepAt: now.Add(-30 * time.Minute),
	}, now)
	if err != nil {
		t.Fatalf("PauseJobRuntimeForUnattendedWallClock() error = %v", err)
	}
	if !exhausted {
		t.Fatal("PauseJobRuntimeForUnattendedWallClock() exhausted = false, want true")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.PausedReason != RuntimePauseReasonBudgetExhausted {
		t.Fatalf("PausedReason = %q, want %q", runtime.PausedReason, RuntimePauseReasonBudgetExhausted)
	}
	if runtime.BudgetBlocker == nil {
		t.Fatal("BudgetBlocker = nil, want persisted blocker")
	}
	if runtime.BudgetBlocker.Ceiling != unattendedWallClockBudgetCeiling {
		t.Fatalf("BudgetBlocker.Ceiling = %q, want %q", runtime.BudgetBlocker.Ceiling, unattendedWallClockBudgetCeiling)
	}
	if runtime.BudgetBlocker.Limit != maxUnattendedWallClockPerJobInMinutes {
		t.Fatalf("BudgetBlocker.Limit = %d, want %d", runtime.BudgetBlocker.Limit, maxUnattendedWallClockPerJobInMinutes)
	}
	if runtime.BudgetBlocker.Observed != 300 {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, 300)
	}
	if runtime.BudgetBlocker.Message != "unattended wall-clock budget exhausted" {
		t.Fatalf("BudgetBlocker.Message = %q, want exact budget message", runtime.BudgetBlocker.Message)
	}
	if len(runtime.CompletedSteps) != 0 {
		t.Fatalf("CompletedSteps = %#v, want empty", runtime.CompletedSteps)
	}
}

func TestRecordFailedToolActionPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxFailedActionsBeforePause; i++ {
		runtime, exhausted, err = RecordFailedToolAction(runtime, now.Add(time.Duration(i)*time.Minute), "message", "message tool: 'content' argument required")
		if err != nil {
			t.Fatalf("RecordFailedToolAction() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordFailedToolAction() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != failedActionsBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want failed_actions blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxFailedActionsBeforePause {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxFailedActionsBeforePause)
	}
	if len(runtime.AuditHistory) != maxFailedActionsBeforePause+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxFailedActionsBeforePause+1)
	}
	if runtime.AuditHistory[maxFailedActionsBeforePause].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxFailedActionsBeforePause].ToolName)
	}
}

func TestRecordOwnerFacingSetStepAckPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "final",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before set-step acknowledgement", i)
		}
		runtime = next
	}

	runtime, exhausted, err := RecordOwnerFacingSetStepAck(runtime, now.Add(19*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingSetStepAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingSetStepAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "final" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "final")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingSetStepAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingSetStepAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingResumeAckPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 12, 30, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before resume acknowledgement", i)
		}
		runtime = next
	}

	paused, err := PauseJobRuntime(runtime, now.Add(19*time.Second))
	if err != nil {
		t.Fatalf("PauseJobRuntime() error = %v", err)
	}
	resumed, err := ResumePausedJobRuntime(paused, now.Add(20*time.Second))
	if err != nil {
		t.Fatalf("ResumePausedJobRuntime() error = %v", err)
	}
	runtime, exhausted, err := RecordOwnerFacingResumeAck(resumed, now.Add(21*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingResumeAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingResumeAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingResumeAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingResumeAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingRevokeApprovalAckPausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 12, 45, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "authorize-2",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before revoke acknowledgement", i)
		}
		runtime = next
	}

	runtime, exhausted, err := RecordOwnerFacingRevokeApprovalAck(runtime, now.Add(19*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingRevokeApprovalAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingRevokeApprovalAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "authorize-2" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "authorize-2")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingRevokeAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingRevokeAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingDenyAckPausesWaitingRuntimeAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 13, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before deny acknowledgement", i)
		}
		runtime = next
	}

	waiting, err := TransitionJobRuntime(runtime, JobStateWaitingUser, now.Add(19*time.Second), RuntimeTransitionOptions{
		StepID:        "build",
		WaitingReason: "discussion_authorization",
	})
	if err != nil {
		t.Fatalf("TransitionJobRuntime(waiting_user) error = %v", err)
	}

	runtime, exhausted, err := RecordOwnerFacingDenyAck(waiting, now.Add(20*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingDenyAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingDenyAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingDenyAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingDenyAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingPauseAckPausesPausedRuntimeAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 13, 15, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	for i := 0; i < 19; i++ {
		next, exhausted, err := RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
		if exhausted {
			t.Fatalf("RecordOwnerFacingMessage() step %d exhausted = true, want false before pause acknowledgement", i)
		}
		runtime = next
	}

	paused, err := PauseJobRuntime(runtime, now.Add(19*time.Second))
	if err != nil {
		t.Fatalf("PauseJobRuntime() error = %v", err)
	}

	runtime, exhausted, err := RecordOwnerFacingPauseAck(paused, now.Add(20*time.Second))
	if err != nil {
		t.Fatalf("RecordOwnerFacingPauseAck() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingPauseAck() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", runtime.ActiveStepID, "build")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-2].ToolName; got != ownerFacingPauseAckAction {
		t.Fatalf("penultimate audit tool = %q, want %q", got, ownerFacingPauseAckAction)
	}
	if got := runtime.AuditHistory[len(runtime.AuditHistory)-1].ToolName; got != "budget_exhausted" {
		t.Fatalf("last audit tool = %q, want %q", got, "budget_exhausted")
	}
}

func TestRecordOwnerFacingMessagePausesAtCeiling(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 10, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingMessage() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingCheckInCountsTowardOwnerFacingMessageBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 10, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingCheckIn(runtime, now.Add(time.Duration(i)*30*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingCheckIn() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingCheckIn() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingCheckInAction {
		t.Fatalf("check-in audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingCheckInAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("check-in audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingDailySummaryCountsTowardOwnerFacingMessageBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingDailySummary(runtime, now.Add(time.Duration(i)*24*time.Hour))
		if err != nil {
			t.Fatalf("RecordOwnerFacingDailySummary() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingDailySummary() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingDailySummaryAction {
		t.Fatalf("daily summary audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingDailySummaryAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("daily summary audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingApprovalRequestCountsTowardOwnerFacingMessageBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 12, 10, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "authorize",
		WaitingReason: "discussion_authorization",
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingApprovalRequest(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingApprovalRequest() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingApprovalRequest() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingApprovalRequestAction {
		t.Fatalf("approval audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingApprovalRequestAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("approval audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingBudgetPauseCountsTowardOwnerFacingMessageBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 12, 40, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedReason: RuntimePauseReasonBudgetExhausted,
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     ownerMessagesBudgetCeiling,
			Limit:       maxOwnerFacingMessagesPerJob,
			Observed:    maxOwnerFacingMessagesPerJob,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: now.Add(-time.Minute),
		},
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingBudgetPause(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingBudgetPause() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingBudgetPause() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingBudgetPauseAction {
		t.Fatalf("budget pause audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingBudgetPauseAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("budget pause audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingWaitingUserCountsTowardOwnerFacingMessageBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 13, 10, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_blocker",
		WaitingAt:     now.Add(-time.Minute),
	}

	var exhausted bool
	var err error
	for i := 0; i < maxOwnerFacingMessagesPerJob; i++ {
		runtime, exhausted, err = RecordOwnerFacingWaitingUser(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingWaitingUser() step %d error = %v", i, err)
		}
	}

	if !exhausted {
		t.Fatal("RecordOwnerFacingWaitingUser() exhausted = false, want true on threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingWaitingUserAction {
		t.Fatalf("waiting-user audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingWaitingUserAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("waiting-user audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestRecordOwnerFacingCompletionAppendsAuditEventWithoutChangingTerminalState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 13, 30, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:       "job-1",
		State:       JobStateCompleted,
		CompletedAt: now.Add(-time.Minute),
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "final", At: now.Add(-time.Minute)},
		},
	}

	next, exhausted, err := RecordOwnerFacingCompletion(runtime, now)
	if err != nil {
		t.Fatalf("RecordOwnerFacingCompletion() error = %v", err)
	}
	if exhausted {
		t.Fatal("RecordOwnerFacingCompletion() exhausted = true, want false for completed runtime")
	}
	if next.State != JobStateCompleted {
		t.Fatalf("State = %q, want %q", next.State, JobStateCompleted)
	}
	if next.BudgetBlocker != nil {
		t.Fatalf("BudgetBlocker = %#v, want nil for completed runtime", next.BudgetBlocker)
	}
	if len(next.AuditHistory) != 1 {
		t.Fatalf("AuditHistory count = %d, want 1", len(next.AuditHistory))
	}
	if next.AuditHistory[0].ToolName != ownerFacingCompletionAction {
		t.Fatalf("completion audit tool = %q, want %q", next.AuditHistory[0].ToolName, ownerFacingCompletionAction)
	}
	if next.AuditHistory[0].ActionClass != AuditActionClassRuntime {
		t.Fatalf("completion audit class = %q, want %q", next.AuditHistory[0].ActionClass, AuditActionClassRuntime)
	}
}

func TestRecordOwnerFacingStepOutputPausesAtCeilingWithPriorMessages(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}

	for i := 0; i < maxOwnerFacingMessagesPerJob-1; i++ {
		var err error
		runtime, _, err = RecordOwnerFacingMessage(runtime, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("RecordOwnerFacingMessage() step %d error = %v", i, err)
		}
	}

	runtime, exhausted, err := RecordOwnerFacingStepOutput(runtime, now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("RecordOwnerFacingStepOutput() error = %v", err)
	}
	if !exhausted {
		t.Fatal("RecordOwnerFacingStepOutput() exhausted = false, want true at threshold")
	}
	if runtime.State != JobStatePaused {
		t.Fatalf("State = %q, want %q", runtime.State, JobStatePaused)
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != ownerMessagesBudgetCeiling {
		t.Fatalf("BudgetBlocker = %#v, want owner_messages blocker", runtime.BudgetBlocker)
	}
	if runtime.BudgetBlocker.Observed != maxOwnerFacingMessagesPerJob {
		t.Fatalf("BudgetBlocker.Observed = %d, want %d", runtime.BudgetBlocker.Observed, maxOwnerFacingMessagesPerJob)
	}
	if len(runtime.AuditHistory) != maxOwnerFacingMessagesPerJob+1 {
		t.Fatalf("AuditHistory count = %d, want %d including budget event", len(runtime.AuditHistory), maxOwnerFacingMessagesPerJob+1)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName != ownerFacingStepOutputAction {
		t.Fatalf("step-output audit tool = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ToolName, ownerFacingStepOutputAction)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass != AuditActionClassRuntime {
		t.Fatalf("step-output audit class = %q, want %q", runtime.AuditHistory[maxOwnerFacingMessagesPerJob-1].ActionClass, AuditActionClassRuntime)
	}
	if runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName != "budget_exhausted" {
		t.Fatalf("final audit event tool = %q, want budget_exhausted", runtime.AuditHistory[maxOwnerFacingMessagesPerJob].ToolName)
	}
}

func TestResumePausedJobRuntimeClearsBudgetBlocker(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 12, 5, 0, 0, time.UTC)
	runtime, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedReason: RuntimePauseReasonBudgetExhausted,
		PausedAt:     time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "owner_messages",
			Limit:       20,
			Observed:    20,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		},
	}, now)
	if err != nil {
		t.Fatalf("ResumePausedJobRuntime() error = %v", err)
	}

	if runtime.State != JobStateRunning {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateRunning)
	}
	if runtime.BudgetBlocker != nil {
		t.Fatalf("BudgetBlocker = %#v, want nil after resume", runtime.BudgetBlocker)
	}
}

func TestSetJobRuntimeActiveStepRejectsPreviouslyCompletedStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID:        job.ID,
		State:        JobStatePaused,
		ActiveStepID: "final",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, "build", time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestSetJobRuntimeActiveStepRejectsPreviouslyFailedStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID:        job.ID,
		State:        JobStatePaused,
		ActiveStepID: "final",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, "build", time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumePausedJobRuntimeRequiresPausedState(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateWaitingUser,
		ActiveStepID: "build",
	}, time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want paused-state failure")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
}

func TestResumePausedJobRuntimeRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResumePausedJobRuntimeRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	_, err := ResumePausedJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	}, time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("ResumePausedJobRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResumePausedJobRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestAbortJobRuntimeTransitionsToTerminalAbortedState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	runtime, err := AbortJobRuntime(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedReason: RuntimePauseReasonOperatorCommand,
		PausedAt:     time.Date(2026, 3, 24, 11, 45, 0, 0, time.UTC),
	}, now)
	if err != nil {
		t.Fatalf("AbortJobRuntime() error = %v", err)
	}

	if runtime.State != JobStateAborted {
		t.Fatalf("State = %q, want %q", runtime.State, JobStateAborted)
	}
	if runtime.AbortedReason != RuntimeAbortReasonOperatorCommand {
		t.Fatalf("AbortedReason = %q, want %q", runtime.AbortedReason, RuntimeAbortReasonOperatorCommand)
	}
	if runtime.AbortedAt != now {
		t.Fatalf("AbortedAt = %v, want %v", runtime.AbortedAt, now)
	}
	if runtime.ActiveStepID != "" {
		t.Fatalf("ActiveStepID = %q, want empty", runtime.ActiveStepID)
	}
	if !IsTerminalJobState(runtime.State) {
		t.Fatalf("IsTerminalJobState(%q) = false, want true", runtime.State)
	}
}

func TestRuntimeControlRejectsTerminalStatesDeterministically(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)
	testCases := []struct {
		name string
		run  func() error
		want RejectionCode
	}{
		{
			name: "resume completed",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateCompleted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "resume failed",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateFailed,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "resume aborted",
			run: func() error {
				_, err := ResumePausedJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateAborted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort completed",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateCompleted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort failed",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateFailed,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
		{
			name: "abort aborted",
			run: func() error {
				_, err := AbortJobRuntime(JobRuntimeState{
					JobID: "job-1",
					State: JobStateAborted,
				}, now)
				return err
			},
			want: RejectionCodeInvalidRuntimeState,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("runtime control error = nil, want deterministic rejection")
			}

			validationErr, ok := err.(ValidationError)
			if !ok {
				t.Fatalf("runtime control error type = %T, want ValidationError", err)
			}
			if validationErr.Code != tc.want {
				t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, tc.want)
			}
		})
	}
}

func TestSetJobRuntimeActiveStepRejectsAbortedRuntime(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	_, err := SetJobRuntimeActiveStep(job, &JobRuntimeState{
		JobID: job.ID,
		State: JobStateAborted,
	}, "build", time.Date(2026, 3, 24, 12, 45, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("SetJobRuntimeActiveStep() error = nil, want aborted-state rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("SetJobRuntimeActiveStep() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidJobTransition {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidJobTransition)
	}
}

func TestHasCompletedRuntimeStepMatchesRecordedCompletion(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "final",
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)},
		},
	}

	if !HasCompletedRuntimeStep(runtime, "build") {
		t.Fatal("HasCompletedRuntimeStep(build) = false, want true")
	}
	if HasCompletedRuntimeStep(runtime, "final") {
		t.Fatal("HasCompletedRuntimeStep(final) = true, want false")
	}
}

func TestHasFailedRuntimeStepMatchesRecordedFailure(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "final",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)},
		},
	}

	if !HasFailedRuntimeStep(runtime, "build") {
		t.Fatal("HasFailedRuntimeStep(build) = false, want true")
	}
	if HasFailedRuntimeStep(runtime, "final") {
		t.Fatal("HasFailedRuntimeStep(final) = true, want false")
	}
}

func TestValidateRuntimeExecutionWaitingUserDenied(t *testing.T) {
	t.Parallel()

	err := ValidateRuntimeExecution(ExecutionContext{
		Job:  &Job{ID: "job-1"},
		Step: &Step{ID: "build"},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateWaitingUser,
			ActiveStepID: "build",
		},
	})
	if err == nil {
		t.Fatal("ValidateRuntimeExecution() error = nil, want waiting_user denial")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ValidateRuntimeExecution() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeWaitingUser {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeWaitingUser)
	}
}

func TestValidateRuntimeExecutionRejectsCompletedActiveStepReplay(t *testing.T) {
	t.Parallel()

	err := ValidateRuntimeExecution(ExecutionContext{
		Job:  &Job{ID: "job-1"},
		Step: &Step{ID: "build"},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "build",
			CompletedSteps: []RuntimeStepRecord{
				{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
			},
		},
	})
	if err == nil {
		t.Fatal("ValidateRuntimeExecution() error = nil, want completed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ValidateRuntimeExecution() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}

func TestResolveExecutionContextWithRuntimeIncludesIndependentRuntimeCopy(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "owner_messages",
			Limit:       20,
			Observed:    20,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: time.Date(2026, 3, 23, 11, 20, 0, 0, time.UTC),
		},
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "draft", At: time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)},
		},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           job.ID,
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				Content: &ApprovalRequestContent{
					ProposedAction: "Complete the authorization discussion step and continue to the next mission step.",
				},
				RequestedVia: ApprovalRequestedViaRuntime,
				State:        ApprovalStatePending,
				RequestedAt:  time.Date(2026, 3, 23, 11, 30, 0, 0, time.UTC),
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           job.ID,
				StepID:          "draft",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				State:           ApprovalStateGranted,
				GrantedAt:       time.Date(2026, 3, 23, 11, 45, 0, 0, time.UTC),
			},
		},
	}

	ec, err := ResolveExecutionContextWithRuntime(job, runtime)
	if err != nil {
		t.Fatalf("ResolveExecutionContextWithRuntime() error = %v", err)
	}

	if ec.Runtime == nil {
		t.Fatal("ResolveExecutionContextWithRuntime().Runtime = nil, want non-nil")
	}
	if !reflect.DeepEqual(*ec.Runtime, runtime) {
		t.Fatalf("ResolveExecutionContextWithRuntime().Runtime = %#v, want %#v", *ec.Runtime, runtime)
	}

	ec.Runtime.CompletedSteps[0].StepID = "mutated"
	ec.Runtime.BudgetBlocker.Ceiling = "mutated-budget"
	ec.Runtime.ApprovalRequests[0].StepID = "mutated-request"
	ec.Runtime.ApprovalRequests[0].Content.ProposedAction = "mutated-content"
	ec.Runtime.ApprovalGrants[0].StepID = "mutated-grant"
	if runtime.CompletedSteps[0].StepID != "draft" {
		t.Fatalf("original runtime step = %q, want %q", runtime.CompletedSteps[0].StepID, "draft")
	}
	if runtime.BudgetBlocker == nil || runtime.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("original runtime budget blocker = %#v, want preserved blocker", runtime.BudgetBlocker)
	}
	if runtime.ApprovalRequests[0].StepID != "build" {
		t.Fatalf("original approval request step = %q, want %q", runtime.ApprovalRequests[0].StepID, "build")
	}
	if runtime.ApprovalRequests[0].Content == nil || runtime.ApprovalRequests[0].Content.ProposedAction != "Complete the authorization discussion step and continue to the next mission step." {
		t.Fatalf("original approval request content = %#v, want preserved content", runtime.ApprovalRequests[0].Content)
	}
	if runtime.ApprovalGrants[0].StepID != "draft" {
		t.Fatalf("original approval grant step = %q, want %q", runtime.ApprovalGrants[0].StepID, "draft")
	}
}

func TestResolveExecutionContextWithRuntimeRejectsFailedActiveStepReplay(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()

	_, err := ResolveExecutionContextWithRuntime(job, JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", At: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		},
	})
	if err == nil {
		t.Fatal("ResolveExecutionContextWithRuntime() error = nil, want failed-step replay rejection")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("ResolveExecutionContextWithRuntime() error type = %T, want ValidationError", err)
	}
	if validationErr.Code != RejectionCodeInvalidRuntimeState {
		t.Fatalf("ValidationError.Code = %q, want %q", validationErr.Code, RejectionCodeInvalidRuntimeState)
	}
	if validationErr.StepID != "build" {
		t.Fatalf("ValidationError.StepID = %q, want %q", validationErr.StepID, "build")
	}
}
