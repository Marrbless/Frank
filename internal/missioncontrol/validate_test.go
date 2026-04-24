package missioncontrol

import (
	"reflect"
	"testing"
	"time"
)

func TestValidatePlanEmptyPlan(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob(nil))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingTerminalFinalStep,
			Message: "plan must include a terminal final_response step",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanDuplicateStepIDs(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "dup", Type: StepTypeDiscussion},
		{ID: "dup", Type: StepTypeStaticArtifact},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeDuplicateStepID,
			StepID:  "dup",
			Message: "duplicate step ID also used at index 0",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanMissingDependency(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, DependsOn: []string{"missing"}},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingDependencyTarget,
			StepID:  "draft",
			Message: "missing dependency target: missing",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanCycle(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "a", Type: StepTypeDiscussion, DependsOn: []string{"b"}},
		{ID: "b", Type: StepTypeOneShotCode, DependsOn: []string{"a"}},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeDependencyCycle,
			StepID:  "a",
			Message: "dependency cycle detected",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanMissingFinalResponse(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingTerminalFinalStep,
			Message: "plan must include a terminal final_response step",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanInvalidStepType(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepType(""), DependsOn: nil},
		{ID: "final", Type: StepTypeFinalResponse},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "draft",
			Message: "step type must be one of discussion, static_artifact, one_shot_code, long_running_code, system_action, wait_user, final_response",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsWaitUserStepWithoutV2SpecVersion(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "hold", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"hold"}},
	}))
	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "hold",
			Message: `step type "wait_user" requires job spec_version frank_v2`,
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsWaitUserStepWithSubtype(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{ID: "hold", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"hold"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanAcceptsV4LiveRuntimePhoneLiveFamily(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV4Job(ExecutionPlaneLiveRuntime, ExecutionHostPhone, MissionFamilyBootstrapRevenue))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsV4MissingExecutionMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  Job
		want ValidationError
	}{
		{
			name: "execution_plane",
			job:  testV4Job("", ExecutionHostPhone, MissionFamilyBootstrapRevenue),
			want: ValidationError{
				Code:    RejectionCodeV4ExecutionPlaneRequired,
				Message: "frank_v4 job requires execution_plane",
			},
		},
		{
			name: "execution_host",
			job:  testV4Job(ExecutionPlaneLiveRuntime, "", MissionFamilyBootstrapRevenue),
			want: ValidationError{
				Code:    RejectionCodeV4ExecutionHostRequired,
				Message: "frank_v4 job requires execution_host",
			},
		},
		{
			name: "mission_family",
			job:  testV4Job(ExecutionPlaneLiveRuntime, ExecutionHostPhone, ""),
			want: ValidationError{
				Code:    RejectionCodeV4MissionFamilyRequired,
				Message: "frank_v4 job requires mission_family",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(tt.job)
			if len(errors) == 0 {
				t.Fatalf("ValidatePlan() = nil, want %#v", tt.want)
			}
			if errors[0] != tt.want {
				t.Fatalf("ValidatePlan()[0] = %#v, want %#v; all errors = %#v", errors[0], tt.want, errors)
			}
		})
	}
}

func TestValidatePlanRejectsV4UnknownExecutionMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  Job
		want ValidationError
	}{
		{
			name: "execution_plane",
			job:  testV4Job("moon_base", ExecutionHostPhone, MissionFamilyBootstrapRevenue),
			want: ValidationError{
				Code:    RejectionCodeV4ExecutionPlaneUnknown,
				Message: "execution_plane must be live_runtime, improvement_workspace, or hot_update_gate",
			},
		},
		{
			name: "execution_host",
			job:  testV4Job(ExecutionPlaneLiveRuntime, "satellite", MissionFamilyBootstrapRevenue),
			want: ValidationError{
				Code:    RejectionCodeV4ExecutionHostUnknown,
				Message: "execution_host must be phone, desktop, workspace, desktop_dev, or remote_provider",
			},
		},
		{
			name: "mission_family",
			job:  testV4Job(ExecutionPlaneLiveRuntime, ExecutionHostPhone, "invented_family"),
			want: ValidationError{
				Code:    RejectionCodeV4MissionFamilyUnknown,
				Message: "mission_family is not recognized for frank_v4",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(tt.job)
			if len(errors) == 0 {
				t.Fatalf("ValidatePlan() = nil, want %#v", tt.want)
			}
			if errors[0] != tt.want {
				t.Fatalf("ValidatePlan()[0] = %#v, want %#v; all errors = %#v", errors[0], tt.want, errors)
			}
		})
	}
}

func TestValidatePlanEnforcesV4MissionFamilyExecutionPlaneCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  Job
		want []ValidationError
	}{
		{
			name: "improvement family on live runtime rejects",
			job:  testV4Job(ExecutionPlaneLiveRuntime, ExecutionHostWorkspace, MissionFamilyImprovePromptpack),
			want: []ValidationError{
				{
					Code:    RejectionCodeV4LabOnlyFamily,
					Message: `mission_family "improve_promptpack" requires execution_plane "improvement_workspace"`,
				},
			},
		},
		{
			name: "improvement family on improvement workspace passes",
			job:  testV4Job(ExecutionPlaneImprovementWorkspace, ExecutionHostWorkspace, MissionFamilyImprovePromptpack),
			want: nil,
		},
		{
			name: "live family on improvement workspace rejects",
			job:  testV4Job(ExecutionPlaneImprovementWorkspace, ExecutionHostPhone, MissionFamilyBootstrapRevenue),
			want: []ValidationError{
				{
					Code:    RejectionCodeV4ExecutionPlaneIncompatible,
					Message: `mission_family "bootstrap_revenue" requires execution_plane "live_runtime"`,
				},
			},
		},
		{
			name: "hot update family on wrong plane rejects",
			job:  testV4Job(ExecutionPlaneLiveRuntime, ExecutionHostPhone, MissionFamilyApplyHotUpdate),
			want: []ValidationError{
				{
					Code:    RejectionCodeV4ExecutionPlaneIncompatible,
					Message: `mission_family "apply_hot_update" requires execution_plane "hot_update_gate"`,
				},
			},
		},
		{
			name: "hot update family on hot update gate passes",
			job:  testV4Job(ExecutionPlaneHotUpdateGate, ExecutionHostPhone, MissionFamilyApplyHotUpdate),
			want: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(tt.job)
			if len(tt.want) == 0 {
				if len(errors) != 0 {
					t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
				}
				return
			}
			if !reflect.DeepEqual(errors, tt.want) {
				t.Fatalf("ValidatePlan() = %#v, want %#v", errors, tt.want)
			}
		})
	}
}

func TestValidatePlanDoesNotRequireExecutionMetadataForPreV4Jobs(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsWaitUserStepWithoutSubtype(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{ID: "hold", Type: StepTypeWaitUser},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"hold"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "hold",
			Message: "wait_user step requires blocker, authorization, or definition subtype",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAllowsStepsWithoutGovernedExternalTargets(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, SuccessCriteria: []string{"stay bounded"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanAllowsStepsWithoutFrankObjectRefs(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, SuccessCriteria: []string{"stay bounded"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanAllowsStepsWithoutCampaignRef(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, SuccessCriteria: []string{"stay bounded"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanAllowsStepsWithoutTreasuryRef(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, SuccessCriteria: []string{"stay bounded"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsMalformedIdentityMode(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:           "draft",
			Type:         StepTypeDiscussion,
			IdentityMode: IdentityMode("owner-ish"),
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidIdentityMode,
			StepID:  "draft",
			Message: `identity_mode "owner-ish" is invalid`,
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsMalformedCampaignRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  *CampaignRef
		want string
	}{
		{
			name: "empty campaign id",
			ref: &CampaignRef{
				CampaignID: "   ",
			},
			want: "campaign ref is invalid: campaign_id is required",
		},
		{
			name: "malformed campaign id",
			ref: &CampaignRef{
				CampaignID: "campaign/one",
			},
			want: `campaign ref is invalid: campaign_id "campaign/one" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(testJob([]Step{
				{
					ID:          "draft",
					Type:        StepTypeDiscussion,
					CampaignRef: tc.ref,
				},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
			}))

			want := []ValidationError{
				{
					Code:    RejectionCodeInvalidCampaignRef,
					StepID:  "draft",
					Message: tc.want,
				},
			}
			if !reflect.DeepEqual(errors, want) {
				t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
			}
		})
	}
}

func TestValidatePlanRejectsMalformedTreasuryRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  *TreasuryRef
		want string
	}{
		{
			name: "empty treasury id",
			ref: &TreasuryRef{
				TreasuryID: "   ",
			},
			want: "treasury ref is invalid: treasury_id is required",
		},
		{
			name: "malformed treasury id",
			ref: &TreasuryRef{
				TreasuryID: "treasury/one",
			},
			want: `treasury ref is invalid: treasury_id "treasury/one" is invalid`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(testJob([]Step{
				{
					ID:          "draft",
					Type:        StepTypeDiscussion,
					TreasuryRef: tc.ref,
				},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
			}))

			want := []ValidationError{
				{
					Code:    RejectionCodeInvalidTreasuryRef,
					StepID:  "draft",
					Message: tc.want,
				},
			}
			if !reflect.DeepEqual(errors, want) {
				t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
			}
		})
	}
}

func TestValidatePlanRejectsMalformedCapabilityRequirements(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:                   "draft",
			Type:                 StepTypeDiscussion,
			RequiredCapabilities: []string{"   "},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidCapabilityRequirement,
			StepID:  "draft",
			Message: "required_capabilities entries must be non-empty",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsRequiredCapabilityWithoutCommittedProposalRef(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:                  "draft",
			Type:                StepTypeDiscussion,
			RequiredDataDomains: []string{"contacts"},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingCapabilityProposal,
			StepID:  "draft",
			Message: "capability onboarding proposal ref is required when step declares required capabilities or data domains",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsCapabilityProposalRefWithoutMissionStoreRoot(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:                   "draft",
			Type:                 StepTypeDiscussion,
			RequiredCapabilities: []string{"camera"},
			CapabilityOnboardingProposalRef: &CapabilityOnboardingProposalRef{
				ProposalID: "proposal-1",
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeMissingCapabilityProposal,
			StepID:  "draft",
			Message: "mission store root is required to resolve capability onboarding proposal refs",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsCommittedCapabilityProposalForRequiredCapability(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-camera"
		record.CapabilityName = "camera"
		record.DataAccessed = []string{"photos/media"}
		record.State = CapabilityOnboardingProposalStateProposed
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := testJob([]Step{
		{
			ID:                   "draft",
			Type:                 StepTypeDiscussion,
			RequiredCapabilities: []string{"camera"},
			RequiredDataDomains:  []string{"photos/media"},
			CapabilityOnboardingProposalRef: &CapabilityOnboardingProposalRef{
				ProposalID: "proposal-camera",
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.MissionStoreRoot = root

	if errors := ValidatePlan(job); len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsCommittedCapabilityProposalInRejectedState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-camera"
		record.State = CapabilityOnboardingProposalStateRejected
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := testJob([]Step{
		{
			ID:                   "draft",
			Type:                 StepTypeDiscussion,
			RequiredCapabilities: []string{"camera"},
			CapabilityOnboardingProposalRef: &CapabilityOnboardingProposalRef{
				ProposalID: "proposal-camera",
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	})
	job.MissionStoreRoot = root

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidCapabilityProposalRef,
			StepID:  "draft",
			Message: `capability onboarding proposal "proposal-camera" state "rejected" is not valid for plan validation`,
		},
	}
	if got := ValidatePlan(job); !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", got, want)
	}
}

func TestValidatePlanRejectsMalformedGovernedExternalTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target AutonomyEligibilityTargetRef
		want   string
	}{
		{
			name: "invalid kind",
			target: AutonomyEligibilityTargetRef{
				Kind:       EligibilityTargetKind(""),
				RegistryID: "provider-mail",
			},
			want: `governed external target is invalid: autonomy eligibility target kind "" is invalid`,
		},
		{
			name: "empty registry id",
			target: AutonomyEligibilityTargetRef{
				Kind:       EligibilityTargetKindProvider,
				RegistryID: "   ",
			},
			want: "governed external target is invalid: autonomy eligibility target registry_id is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(testJob([]Step{
				{
					ID:                      "draft",
					Type:                    StepTypeDiscussion,
					GovernedExternalTargets: []AutonomyEligibilityTargetRef{tc.target},
				},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
			}))

			want := []ValidationError{
				{
					Code:    RejectionCodeInvalidGovernedExternalTarget,
					StepID:  "draft",
					Message: tc.want,
				},
			}
			if !reflect.DeepEqual(errors, want) {
				t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
			}
		})
	}
}

func TestValidatePlanRejectsMalformedFrankObjectRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  FrankRegistryObjectRef
		want string
	}{
		{
			name: "invalid kind",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKind(""),
				ObjectID: "identity-1",
			},
			want: `Frank object ref is invalid: Frank object ref kind "" is invalid`,
		},
		{
			name: "empty object id",
			ref: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindIdentity,
				ObjectID: "   ",
			},
			want: "Frank object ref is invalid: Frank object ref object_id is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errors := ValidatePlan(testJob([]Step{
				{
					ID:              "draft",
					Type:            StepTypeDiscussion,
					FrankObjectRefs: []FrankRegistryObjectRef{tc.ref},
				},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
			}))

			want := []ValidationError{
				{
					Code:    RejectionCodeInvalidFrankObjectRef,
					StepID:  "draft",
					Message: tc.want,
				},
			}
			if !reflect.DeepEqual(errors, want) {
				t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
			}
		})
	}
}

func TestValidatePlanRejectsDuplicateFrankObjectRefsAfterNormalization(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			FrankObjectRefs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindAccount,
					ObjectID: "account-mail",
				},
				{
					Kind:     FrankRegistryObjectKind(" account "),
					ObjectID: " account-mail ",
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidFrankObjectRef,
			StepID:  "draft",
			Message: `duplicate Frank object ref kind "account" object_id "account-mail"`,
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsDuplicateGovernedExternalTargetsAfterNormalization(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:   "draft",
			Type: StepTypeDiscussion,
			GovernedExternalTargets: []AutonomyEligibilityTargetRef{
				{
					Kind:       EligibilityTargetKindProvider,
					RegistryID: "provider-mail",
				},
				{
					Kind:       EligibilityTargetKindProvider,
					RegistryID: " provider-mail ",
				},
			},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"draft"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidGovernedExternalTarget,
			StepID:  "draft",
			Message: `duplicate governed external target kind "provider" registry_id "provider-mail"`,
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsV2StaticArtifactWithoutExplicitPathMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                   "artifact",
			Type:                 StepTypeStaticArtifact,
			SuccessCriteria:      []string{"Write `report.json` as valid JSON."},
			StaticArtifactFormat: "json",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"artifact"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "artifact",
			Message: "static_artifact step requires explicit static_artifact_path metadata for frank_v2",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsV2StaticArtifactWithoutExplicitFormatMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                 "artifact",
			Type:               StepTypeStaticArtifact,
			SuccessCriteria:    []string{"Write `report.json` as valid JSON."},
			StaticArtifactPath: "report.json",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"artifact"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "artifact",
			Message: "static_artifact step requires explicit static_artifact_format metadata for frank_v2",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsV2StaticArtifactWithExplicitContractMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                   "artifact",
			Type:                 StepTypeStaticArtifact,
			SuccessCriteria:      []string{"Write a report artifact."},
			StaticArtifactPath:   "report.json",
			StaticArtifactFormat: "json",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"artifact"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsV2OneShotCodeWithoutExplicitArtifactPathMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:              "build",
			Type:            StepTypeOneShotCode,
			SuccessCriteria: []string{"Write `main.go` and run `go test ./...`."},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))

	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "build",
			Message: "one_shot_code step requires explicit one_shot_artifact_path metadata for frank_v2",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsV2OneShotCodeWithExplicitArtifactPathMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                  "build",
			Type:                StepTypeOneShotCode,
			SuccessCriteria:     []string{"Write code and validate it."},
			OneShotArtifactPath: "main.go",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsLongRunningCodeWithoutV2SpecVersion(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{
			ID:                        "build",
			Type:                      StepTypeLongRunningCode,
			SuccessCriteria:           []string{"Record startup command `npm start` and verify the artifact builds."},
			LongRunningStartupCommand: []string{"npm", "start"},
			LongRunningArtifactPath:   "dist/service.js",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "build",
			Message: `step type "long_running_code" requires job spec_version frank_v2`,
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAcceptsLongRunningCodeStep(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                        "build",
			Type:                      StepTypeLongRunningCode,
			SuccessCriteria:           []string{"Record startup command `npm start` and verify the artifact builds."},
			LongRunningStartupCommand: []string{"npm", "start"},
			LongRunningArtifactPath:   "dist/service.js",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanRejectsLongRunningCodeWithoutStartupCommandMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                      "build",
			Type:                    StepTypeLongRunningCode,
			SuccessCriteria:         []string{"Record startup command `npm start` and verify the artifact builds."},
			LongRunningArtifactPath: "dist/service.js",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "build",
			Message: "long_running_code step requires explicit long_running_startup_command metadata",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsLongRunningCodeWithoutArtifactPathMetadata(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                        "build",
			Type:                      StepTypeLongRunningCode,
			SuccessCriteria:           []string{"Record startup command `npm start` and verify the artifact builds."},
			LongRunningStartupCommand: []string{"npm", "start"},
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	want := []ValidationError{
		{
			Code:    RejectionCodeInvalidStepType,
			StepID:  "build",
			Message: "long_running_code step requires explicit long_running_artifact_path metadata",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanRejectsLongRunningCodeStartIntent(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testV2Job([]Step{
		{
			ID:                        "build",
			Type:                      StepTypeLongRunningCode,
			SuccessCriteria:           []string{"Start the service and verify it stays running."},
			LongRunningStartupCommand: []string{"npm", "start"},
			LongRunningArtifactPath:   "dist/service.js",
		},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))
	want := []ValidationError{
		{
			Code:    RejectionCodeLongRunningStartForbidden,
			StepID:  "build",
			Message: "long_running_code must not start a process; move start/stop semantics to system_action",
		},
	}
	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanAuthorityExceeded(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, RequiredAuthority: AuthorityTierHigh},
		{ID: "final", Type: StepTypeFinalResponse},
	})
	job.MaxAuthority = AuthorityTierLow

	errors := ValidatePlan(job)

	want := []ValidationError{
		{
			Code:    RejectionCodeAuthorityExceeded,
			StepID:  "draft",
			Message: "step required authority exceeds job max authority",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanDisallowedStepTool(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "draft", Type: StepTypeDiscussion, AllowedTools: []string{"write"}},
		{ID: "final", Type: StepTypeFinalResponse},
	})
	job.AllowedTools = []string{"read"}

	errors := ValidatePlan(job)

	want := []ValidationError{
		{
			Code:    RejectionCodeToolNotAllowed,
			StepID:  "draft",
			Message: "step tool is not allowed by job tool scope: write",
		},
	}

	if !reflect.DeepEqual(errors, want) {
		t.Fatalf("ValidatePlan() = %#v, want %#v", errors, want)
	}
}

func TestValidatePlanValidPlan(t *testing.T) {
	t.Parallel()

	errors := ValidatePlan(testJob([]Step{
		{ID: "discuss", Type: StepTypeDiscussion, RequiredAuthority: AuthorityTierLow, AllowedTools: []string{"read"}},
		{ID: "build", Type: StepTypeOneShotCode, DependsOn: []string{"discuss"}, RequiredAuthority: AuthorityTierMedium, AllowedTools: []string{"read", "write"}},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	}))

	if len(errors) != 0 {
		t.Fatalf("ValidatePlan() = %#v, want no errors", errors)
	}
}

func TestValidatePlanErrorOrdering(t *testing.T) {
	t.Parallel()

	job := testJob([]Step{
		{ID: "dup", Type: StepTypeDiscussion, DependsOn: []string{"missing"}, RequiredAuthority: AuthorityTierHigh, AllowedTools: []string{"write"}},
		{ID: "dup", Type: StepTypeDiscussion},
		{ID: "cycle-a", Type: StepTypeDiscussion, DependsOn: []string{"cycle-b"}},
		{ID: "cycle-b", Type: StepTypeOneShotCode, DependsOn: []string{"cycle-a"}},
		{ID: "bad-type", Type: StepType("bogus")},
	})
	job.MaxAuthority = AuthorityTierLow
	job.AllowedTools = []string{"read"}

	errors := ValidatePlan(job)

	wantCodes := []RejectionCode{
		RejectionCodeDuplicateStepID,
		RejectionCodeMissingDependencyTarget,
		RejectionCodeDependencyCycle,
		RejectionCodeMissingTerminalFinalStep,
		RejectionCodeInvalidStepType,
		RejectionCodeAuthorityExceeded,
		RejectionCodeToolNotAllowed,
	}

	if len(errors) != len(wantCodes) {
		t.Fatalf("ValidatePlan() returned %d errors, want %d: %#v", len(errors), len(wantCodes), errors)
	}

	for i, wantCode := range wantCodes {
		if errors[i].Code != wantCode {
			t.Fatalf("error[%d].Code = %q, want %q; errors = %#v", i, errors[i].Code, wantCode, errors)
		}
	}
}

func testJob(steps []Step) Job {
	return Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierMedium,
		AllowedTools: []string{"read", "write"},
		Plan: Plan{
			ID:    "plan-1",
			Steps: steps,
		},
	}
}

func testV2Job(steps []Step) Job {
	job := testJob(steps)
	job.SpecVersion = JobSpecVersionV2
	return job
}

func testV4Job(executionPlane, executionHost, missionFamily string) Job {
	job := testJob([]Step{
		{ID: "build", Type: StepTypeDiscussion},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"build"}},
	})
	job.SpecVersion = JobSpecVersionV4
	job.ExecutionPlane = executionPlane
	job.ExecutionHost = executionHost
	job.MissionFamily = missionFamily
	return job
}
