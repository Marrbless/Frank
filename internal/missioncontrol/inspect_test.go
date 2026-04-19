package missioncontrol

import (
	"reflect"
	"testing"
	"time"
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

func TestNewInspectSummaryWithCampaignAndTreasuryPreflightIncludesCapabilityOnboardingProposalPreflight(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	record := validCapabilityOnboardingProposalRecord(now, func(record *CapabilityOnboardingProposalRecord) {
		record.ProposalID = "proposal-camera"
		record.CapabilityName = "camera"
		record.DataAccessed = []string{"photos/media"}
	})
	if err := StoreCapabilityOnboardingProposalRecord(root, record); err != nil {
		t.Fatalf("StoreCapabilityOnboardingProposalRecord() error = %v", err)
	}

	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"write", "read", "search"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                   "build",
					Type:                 StepTypeOneShotCode,
					RequiredAuthority:    AuthorityTierLow,
					AllowedTools:         []string{"read"},
					SuccessCriteria:      []string{"produce code"},
					RequiredCapabilities: []string{"camera"},
					RequiredDataDomains:  []string{"photos/media"},
					CapabilityOnboardingProposalRef: &CapabilityOnboardingProposalRef{
						ProposalID: "proposal-camera",
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	if len(summary.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1", len(summary.Steps))
	}
	preflight := summary.Steps[0].CapabilityOnboardingProposalPreflight
	if preflight == nil {
		t.Fatal("CapabilityOnboardingProposalPreflight = nil, want resolved proposal")
	}
	if preflight.Proposal == nil {
		t.Fatal("CapabilityOnboardingProposalPreflight.Proposal = nil, want proposal record")
	}
	if preflight.Proposal.ProposalID != "proposal-camera" {
		t.Fatalf("CapabilityOnboardingProposalPreflight.Proposal.ProposalID = %q, want %q", preflight.Proposal.ProposalID, "proposal-camera")
	}
	if !reflect.DeepEqual(preflight.RequiredCapabilities, []string{"camera"}) {
		t.Fatalf("CapabilityOnboardingProposalPreflight.RequiredCapabilities = %#v, want %#v", preflight.RequiredCapabilities, []string{"camera"})
	}
	if !reflect.DeepEqual(preflight.RequiredDataDomains, []string{"photos/media"}) {
		t.Fatalf("CapabilityOnboardingProposalPreflight.RequiredDataDomains = %#v, want %#v", preflight.RequiredDataDomains, []string{"photos/media"})
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
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	campaign := validCampaignRecord(time.Date(2026, 4, 8, 20, 55, 0, 0, time.UTC), func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
	})
	if err := StoreCampaignRecord(fixtures.root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.BootstrapAcquisition = &TreasuryBootstrapAcquisition{
			EntryID:         "entry-first-value",
			AssetCode:       "USD",
			Amount:          "10.00",
			SourceRef:       "payout:listing-a",
			EvidenceLocator: "https://evidence.example/payout-a",
			ConfirmedAt:     time.Date(2026, 4, 8, 21, 2, 30, 0, time.UTC),
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	tests := []struct {
		name string
		run  func() (string, error)
		keys []string
	}{
		{
			name: "job",
			run: func() (string, error) {
				summary, err := NewInspectSummary(job, "build")
				if err != nil {
					return "", err
				}
				return FormatInspectSummary(summary)
			},
			keys: []string{"allowed_tools", "job_id", "max_authority", "steps"},
		},
		{
			name: "inspectable_plan",
			run: func() (string, error) {
				summary, err := NewInspectSummaryFromInspectablePlan(job.ID, &plan, "build")
				if err != nil {
					return "", err
				}
				return FormatInspectSummary(summary)
			},
			keys: []string{"allowed_tools", "job_id", "max_authority", "steps"},
		},
		{
			name: "resolved_campaign_and_treasury_preflight",
			run: func() (string, error) {
				summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
				if err != nil {
					return "", err
				}
				return FormatInspectSummary(summary)
			},
			keys: []string{"allowed_tools", "job_id", "max_authority", "steps"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			formatted, err := tc.run()
			if err != nil {
				t.Fatalf("inspect summary error = %v", err)
			}

			got := mustOperatorReadoutJSONObject(t, formatted)
			assertJSONObjectKeys(t, got, tc.keys...)
			steps := mustJSONArray(t, got["steps"], "inspect.steps")
			if len(steps) != 1 {
				t.Fatalf("steps len = %d, want 1", len(steps))
			}
			step, ok := steps[0].(map[string]any)
			if !ok {
				t.Fatalf("steps[0] = %#v, want object", steps[0])
			}

			wantStepKeys := []string{"allowed_tools", "depends_on", "effective_allowed_tools", "required_authority", "requires_approval", "step_id", "step_type", "success_criteria"}
			if tc.name == "resolved_campaign_and_treasury_preflight" {
				wantStepKeys = append(wantStepKeys, "campaign_preflight", "treasury_preflight")
				assertResolvedCampaignPreflightJSONEnvelope(t, step["campaign_preflight"])
				assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
				treasuryPreflight := step["treasury_preflight"].(map[string]any)
				treasury := treasuryPreflight["treasury"].(map[string]any)
				bootstrap := treasury["bootstrap_acquisition"].(map[string]any)
				if bootstrap["entry_id"] != "entry-first-value" {
					t.Fatalf("steps[0].treasury_preflight.treasury.bootstrap_acquisition.entry_id = %#v, want %q", bootstrap["entry_id"], "entry-first-value")
				}
			} else {
				if _, ok := step["campaign_preflight"]; ok {
					t.Fatalf("campaign_preflight = %#v, want omitted on %s path", step["campaign_preflight"], tc.name)
				}
				if _, ok := step["treasury_preflight"]; ok {
					t.Fatalf("treasury_preflight = %#v, want omitted on %s path", step["treasury_preflight"], tc.name)
				}
			}
			assertJSONObjectKeys(t, step, wantStepKeys...)
			assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", tc.name == "resolved_campaign_and_treasury_preflight", tc.name == "resolved_campaign_and_treasury_preflight")
		})
	}
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostBootstrapAcquisitionWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostBootstrapAcquisition = &TreasuryPostBootstrapAcquisition{
			AssetCode:       "USD",
			Amount:          "2.25",
			SourceRef:       "payout:listing-b",
			EvidenceLocator: "https://evidence.example/payout-b",
			ConfirmedAt:     time.Date(2026, 4, 8, 21, 4, 0, 0, time.UTC),
			ConsumedEntryID: "entry-post-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postBootstrap := treasury["post_bootstrap_acquisition"].(map[string]any)
	if postBootstrap["source_ref"] != "payout:listing-b" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_bootstrap_acquisition.source_ref = %#v, want %q", postBootstrap["source_ref"], "payout:listing-b")
	}
	if postBootstrap["consumed_entry_id"] != "entry-post-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_bootstrap_acquisition.consumed_entry_id = %#v, want %q", postBootstrap["consumed_entry_id"], "entry-post-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveSaveWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostActiveSave = &TreasuryPostActiveSave{
			AssetCode:         "USD",
			Amount:            "1.25",
			TargetContainerID: "container-savings",
			SourceRef:         "transfer:reserve-a",
			EvidenceLocator:   "https://evidence.example/save-a",
			ConsumedEntryID:   "entry-save-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveSave := treasury["post_active_save"].(map[string]any)
	if postActiveSave["target_container_id"] != "container-savings" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_save.target_container_id = %#v, want %q", postActiveSave["target_container_id"], "container-savings")
	}
	if postActiveSave["consumed_entry_id"] != "entry-save-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_save.consumed_entry_id = %#v, want %q", postActiveSave["consumed_entry_id"], "entry-save-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveSuspendWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateSuspended
		record.PostActiveSuspend = &TreasuryPostActiveSuspend{
			Reason:               "risk:manual-review-required",
			SourceRef:            "suspend:risk-review-a",
			ConsumedTransitionID: "transition-suspend-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveSuspend := treasury["post_active_suspend"].(map[string]any)
	if postActiveSuspend["reason"] != "risk:manual-review-required" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_suspend.reason = %#v, want %q", postActiveSuspend["reason"], "risk:manual-review-required")
	}
	if postActiveSuspend["consumed_transition_id"] != "transition-suspend-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_suspend.consumed_transition_id = %#v, want %q", postActiveSuspend["consumed_transition_id"], "transition-suspend-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostSuspendResumeWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostSuspendResume = &TreasuryPostSuspendResume{
			Reason:               "ops:manual-clear",
			SourceRef:            "resume:manual-clear-a",
			ConsumedTransitionID: "transition-resume-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postSuspendResume := treasury["post_suspend_resume"].(map[string]any)
	if postSuspendResume["reason"] != "ops:manual-clear" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_suspend_resume.reason = %#v, want %q", postSuspendResume["reason"], "ops:manual-clear")
	}
	if postSuspendResume["consumed_transition_id"] != "transition-resume-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_suspend_resume.consumed_transition_id = %#v, want %q", postSuspendResume["consumed_transition_id"], "transition-resume-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveAllocateWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostActiveAllocate = &TreasuryPostActiveAllocate{
			AssetCode: "USD",
			Amount:    "1.10",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
			AllocationTargetRef: "allocation:ops-reserve",
			SourceRef:           "allocate:ops-reserve-a",
			ConsumedEntryID:     "entry-allocate-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveAllocate := treasury["post_active_allocate"].(map[string]any)
	sourceRef := postActiveAllocate["source_container_ref"].(map[string]any)
	if sourceRef["object_id"] != "container-wallet" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_allocate.source_container_ref.object_id = %#v, want %q", sourceRef["object_id"], "container-wallet")
	}
	if postActiveAllocate["allocation_target_ref"] != "allocation:ops-reserve" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_allocate.allocation_target_ref = %#v, want %q", postActiveAllocate["allocation_target_ref"], "allocation:ops-reserve")
	}
	if postActiveAllocate["consumed_entry_id"] != "entry-allocate-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_allocate.consumed_entry_id = %#v, want %q", postActiveAllocate["consumed_entry_id"], "entry-allocate-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveReinvestWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-investment",
	}
	writeFrankRegistryEligibilityFixture(t, fixtures.root, target, EligibilityLabelAutonomyCompatible, "container-class-investment", "check-container-class-investment", time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC))
	investment := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-investment",
		ContainerKind:        "wallet",
		Label:                "Investment Wallet",
		ContainerClassID:     "container-class-investment",
		State:                "active",
		EligibilityTargetRef: target,
		CreatedAt:            time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
	}
	if err := StoreFrankContainerRecord(fixtures.root, investment); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostActiveReinvest = &TreasuryPostActiveReinvest{
			SourceAssetCode: "USD",
			SourceAmount:    "0.75",
			TargetAssetCode: "BTC",
			TargetAmount:    "0.00001000",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-investment",
			},
			SourceRef:       "trade:reinvest-a",
			EvidenceLocator: "https://evidence.example/reinvest-a",
			ConfirmedAt:     time.Date(2026, 4, 8, 21, 4, 0, 0, time.UTC),
			ConsumedEntryID: "entry-reinvest-value-in",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveReinvest := treasury["post_active_reinvest"].(map[string]any)
	sourceRef := postActiveReinvest["source_container_ref"].(map[string]any)
	targetRef := postActiveReinvest["target_container_ref"].(map[string]any)
	if sourceRef["object_id"] != "container-wallet" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_reinvest.source_container_ref.object_id = %#v, want %q", sourceRef["object_id"], "container-wallet")
	}
	if targetRef["object_id"] != "container-investment" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_reinvest.target_container_ref.object_id = %#v, want %q", targetRef["object_id"], "container-investment")
	}
	if postActiveReinvest["consumed_entry_id"] != "entry-reinvest-value-in" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_reinvest.consumed_entry_id = %#v, want %q", postActiveReinvest["consumed_entry_id"], "entry-reinvest-value-in")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveSpendWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostActiveSpend = &TreasuryPostActiveSpend{
			AssetCode: "USD",
			Amount:    "0.75",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
			TargetRef:       "vendor:domain-renewal",
			SourceRef:       "spend:domain-renewal-a",
			EvidenceLocator: "https://evidence.example/spend-a",
			ConsumedEntryID: "entry-spend-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveSpend := treasury["post_active_spend"].(map[string]any)
	sourceRef := postActiveSpend["source_container_ref"].(map[string]any)
	if sourceRef["object_id"] != "container-wallet" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_spend.source_container_ref.object_id = %#v, want %q", sourceRef["object_id"], "container-wallet")
	}
	if postActiveSpend["target_ref"] != "vendor:domain-renewal" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_spend.target_ref = %#v, want %q", postActiveSpend["target_ref"], "vendor:domain-renewal")
	}
	if postActiveSpend["consumed_entry_id"] != "entry-spend-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_spend.consumed_entry_id = %#v, want %q", postActiveSpend["consumed_entry_id"], "entry-spend-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestInspectSummaryWithTreasuryPreflightIncludesPostActiveTransferWhenPresent(t *testing.T) {
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
					TreasuryRef:       &TreasuryRef{TreasuryID: "treasury-wallet"},
				},
				{
					ID:   "final",
					Type: StepTypeFinalResponse,
				},
			},
		},
	}

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	record := validTreasuryRecord(time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-wallet"
		record.State = TreasuryStateActive
		record.PostActiveTransfer = &TreasuryPostActiveTransfer{
			AssetCode: "USD",
			Amount:    "1.25",
			SourceContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-wallet",
			},
			TargetContainerRef: FrankRegistryObjectRef{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: "container-vault",
			},
			SourceRef:       "transfer:rebalance-a",
			EvidenceLocator: "https://evidence.example/transfer-a",
			ConsumedEntryID: "entry-transfer-value",
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	assertResolvedTreasuryPreflightJSONEnvelope(t, step["treasury_preflight"])
	treasury := step["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveTransfer := treasury["post_active_transfer"].(map[string]any)
	sourceRef := postActiveTransfer["source_container_ref"].(map[string]any)
	targetRef := postActiveTransfer["target_container_ref"].(map[string]any)
	if sourceRef["object_id"] != "container-wallet" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_transfer.source_container_ref.object_id = %#v, want %q", sourceRef["object_id"], "container-wallet")
	}
	if targetRef["object_id"] != "container-vault" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_transfer.target_container_ref.object_id = %#v, want %q", targetRef["object_id"], "container-vault")
	}
	if postActiveTransfer["consumed_entry_id"] != "entry-transfer-value" {
		t.Fatalf("steps[0].treasury_preflight.treasury.post_active_transfer.consumed_entry_id = %#v, want %q", postActiveTransfer["consumed_entry_id"], "entry-transfer-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "inspect JSON", false, true)
}

func TestFormatInspectSummarySurfacesCampaignZohoEmailAddressingInCampaignPreflight(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
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
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}
	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	campaign := validCampaignRecord(time.Date(2026, 4, 8, 20, 55, 0, 0, time.UTC), func(record *CampaignRecord) {
		record.CampaignID = "campaign-mail"
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
		record.ZohoEmailAddressing = &CampaignZohoEmailAddressing{
			To:  []string{"person@example.com", "team@example.com"},
			CC:  []string{"copy@example.com"},
			BCC: []string{"blind@example.com"},
		}
	})
	if err := StoreCampaignRecord(fixtures.root, campaign); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	preflight := step["campaign_preflight"].(map[string]any)
	campaignJSON := preflight["campaign"].(map[string]any)
	addressing, ok := campaignJSON["zoho_email_addressing"].(map[string]any)
	if !ok {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing = %#v, want object", campaignJSON["zoho_email_addressing"])
	}
	assertJSONObjectKeys(t, campaignJSON, "campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at", "zoho_email_addressing")
	if !reflect.DeepEqual(mustJSONArray(t, addressing["to"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.to"), []any{"person@example.com", "team@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.to = %#v, want [person@example.com team@example.com]", addressing["to"])
	}
	if !reflect.DeepEqual(mustJSONArray(t, addressing["cc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.cc"), []any{"copy@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.cc = %#v, want [copy@example.com]", addressing["cc"])
	}
	if !reflect.DeepEqual(mustJSONArray(t, addressing["bcc"], "steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc"), []any{"blind@example.com"}) {
		t.Fatalf("steps[0].campaign_preflight.campaign.zoho_email_addressing.bcc = %#v, want [blind@example.com]", addressing["bcc"])
	}
}

func TestFormatInspectSummarySurfacesZohoMailboxBootstrapPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankZohoMailboxFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"bootstrap mailbox"},
					GovernedExternalTargets: []AutonomyEligibilityTargetRef{
						{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
						{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
					},
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	preflight, ok := step["frank_zoho_mailbox_bootstrap_preflight"].(map[string]any)
	if !ok {
		t.Fatalf("steps[0].frank_zoho_mailbox_bootstrap_preflight = %#v, want object", step["frank_zoho_mailbox_bootstrap_preflight"])
	}
	assertJSONObjectKeys(t, preflight, "account", "identity")

	identity, ok := preflight["identity"].(map[string]any)
	if !ok {
		t.Fatalf("bootstrap preflight identity = %#v, want object", preflight["identity"])
	}
	assertJSONObjectKeys(t, identity, "created_at", "display_name", "eligibility_target_ref", "identity_id", "identity_kind", "identity_mode", "provider_or_platform_id", "record_version", "state", "updated_at", "zoho_mailbox")
	identityZoho, ok := identity["zoho_mailbox"].(map[string]any)
	if !ok {
		t.Fatalf("bootstrap preflight identity zoho_mailbox = %#v, want object", identity["zoho_mailbox"])
	}
	assertJSONObjectKeys(t, identityZoho, "from_address", "from_display_name")

	account, ok := preflight["account"].(map[string]any)
	if !ok {
		t.Fatalf("bootstrap preflight account = %#v, want object", preflight["account"])
	}
	assertJSONObjectKeys(t, account, "account_id", "account_kind", "control_model", "created_at", "eligibility_target_ref", "identity_id", "label", "provider_or_platform_id", "record_version", "recovery_model", "state", "updated_at", "zoho_mailbox")
	accountZoho, ok := account["zoho_mailbox"].(map[string]any)
	if !ok {
		t.Fatalf("bootstrap preflight account zoho_mailbox = %#v, want object", account["zoho_mailbox"])
	}
	assertJSONObjectKeys(t, accountZoho, "admin_oauth_token_env_var_ref", "bootstrap_password_env_var_ref", "confirmed_created", "organization_id", "provider_account_id")

	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on bootstrap-only inspect path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on bootstrap-only inspect path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankTelegramOwnerControlOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeTelegramOwnerControlOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm telegram owner-control onboarding"},
					IdentityMode:      IdentityModeOwnerOnlyControl,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankTelegramOwnerControlOnboardingPreflightJSONEnvelope(t, step["frank_telegram_owner_control_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on telegram owner-control onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on telegram owner-control onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankSlackOwnerControlOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeSlackOwnerControlOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm slack owner-control onboarding"},
					IdentityMode:      IdentityModeOwnerOnlyControl,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankSlackOwnerControlOnboardingPreflightJSONEnvelope(t, step["frank_slack_owner_control_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on slack owner-control onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on slack owner-control onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankDiscordOwnerControlOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeDiscordOwnerControlOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm discord owner-control onboarding"},
					IdentityMode:      IdentityModeOwnerOnlyControl,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankDiscordOwnerControlOnboardingPreflightJSONEnvelope(t, step["frank_discord_owner_control_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on discord owner-control onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on discord owner-control onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankWhatsAppOwnerControlOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeWhatsAppOwnerControlOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm whatsapp owner-control onboarding"},
					IdentityMode:      IdentityModeOwnerOnlyControl,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankWhatsAppOwnerControlOnboardingPreflightJSONEnvelope(t, step["frank_whatsapp_owner_control_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on whatsapp owner-control onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on whatsapp owner-control onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankGitHubOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeGitHubOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm github onboarding"},
					IdentityMode:      IdentityModeAgentAlias,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankGitHubOnboardingPreflightJSONEnvelope(t, step["frank_github_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on github onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on github onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankStripeOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeStripeOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm stripe onboarding"},
					IdentityMode:      IdentityModeAgentAlias,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankStripeOnboardingPreflightJSONEnvelope(t, step["frank_stripe_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on stripe onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on stripe onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankPayPalOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writePayPalOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm paypal onboarding"},
					IdentityMode:      IdentityModeAgentAlias,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankPayPalOnboardingPreflightJSONEnvelope(t, step["frank_paypal_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on paypal onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on paypal onboarding path", step["treasury_preflight"])
	}
}

func TestFormatInspectSummarySurfacesFrankGoogleOnboardingPreflight(t *testing.T) {
	t.Parallel()

	fixtures := writeGoogleOnboardingFixtures(t)
	job := Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{
					ID:                "build",
					Type:              StepTypeOneShotCode,
					RequiredAuthority: AuthorityTierLow,
					AllowedTools:      []string{"read"},
					SuccessCriteria:   []string{"confirm google onboarding"},
					IdentityMode:      IdentityModeAgentAlias,
					FrankObjectRefs: []FrankRegistryObjectRef{
						{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
						{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
					},
				},
				{
					ID:        "final",
					Type:      StepTypeFinalResponse,
					DependsOn: []string{"build"},
				},
			},
		},
	}

	summary, err := NewInspectSummaryWithCampaignAndTreasuryPreflight(job, "build", fixtures.root)
	if err != nil {
		t.Fatalf("NewInspectSummaryWithCampaignAndTreasuryPreflight() error = %v", err)
	}
	formatted, err := FormatInspectSummary(summary)
	if err != nil {
		t.Fatalf("FormatInspectSummary() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	steps := mustJSONArray(t, got["steps"], "inspect.steps")
	step := steps[0].(map[string]any)
	assertResolvedFrankGoogleOnboardingPreflightJSONEnvelope(t, step["frank_google_onboarding_preflight"])
	if _, ok := step["campaign_preflight"]; ok {
		t.Fatalf("campaign_preflight = %#v, want omitted on google onboarding path", step["campaign_preflight"])
	}
	if _, ok := step["treasury_preflight"]; ok {
		t.Fatalf("treasury_preflight = %#v, want omitted on google onboarding path", step["treasury_preflight"])
	}
}
