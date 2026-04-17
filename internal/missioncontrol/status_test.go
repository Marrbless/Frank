package missioncontrol

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestBuildOperatorStatusSummaryIncludesLatestApprovalForActiveStep(t *testing.T) {
	t.Parallel()

	requestedAt := time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)
	supersededAt := time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateWaitingUser,
		ActiveStepID:  "build",
		WaitingReason: "discussion_authorization",
		WaitingAt:     time.Date(2026, 3, 24, 12, 2, 30, 0, time.UTC),
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "draft",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStateSuperseded,
				SupersededAt:    supersededAt,
			},
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				RequestedVia:    ApprovalRequestedViaRuntime,
				SessionChannel:  "telegram",
				SessionChatID:   "chat-42",
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt,
				ExpiresAt:       expiresAt,
				Content: &ApprovalRequestContent{
					ProposedAction:   "Continue build.",
					WhyNeeded:        "Operator approval is required.",
					AuthorityTier:    AuthorityTierMedium,
					FallbackIfDenied: "Stay waiting.",
				},
			},
		},
	})

	if summary.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", summary.JobID, "job-1")
	}
	if summary.State != JobStateWaitingUser {
		t.Fatalf("State = %q, want %q", summary.State, JobStateWaitingUser)
	}
	if summary.ActiveStepID != "build" {
		t.Fatalf("ActiveStepID = %q, want %q", summary.ActiveStepID, "build")
	}
	if summary.WaitingReason != "discussion_authorization" {
		t.Fatalf("WaitingReason = %q, want %q", summary.WaitingReason, "discussion_authorization")
	}
	if summary.WaitingAt == nil || *summary.WaitingAt != "2026-03-24T12:02:30Z" {
		t.Fatalf("WaitingAt = %#v, want RFC3339 time", summary.WaitingAt)
	}
	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.StepID != "build" {
		t.Fatalf("ApprovalRequest.StepID = %q, want %q", summary.ApprovalRequest.StepID, "build")
	}
	if summary.ApprovalRequest.State != ApprovalStatePending {
		t.Fatalf("ApprovalRequest.State = %q, want %q", summary.ApprovalRequest.State, ApprovalStatePending)
	}
	if summary.ApprovalRequest.RequestedVia != ApprovalRequestedViaRuntime {
		t.Fatalf("ApprovalRequest.RequestedVia = %q, want %q", summary.ApprovalRequest.RequestedVia, ApprovalRequestedViaRuntime)
	}
	if summary.ApprovalRequest.GrantedVia != "" {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want empty", summary.ApprovalRequest.GrantedVia)
	}
	if summary.ApprovalRequest.SessionChannel != "telegram" {
		t.Fatalf("ApprovalRequest.SessionChannel = %q, want %q", summary.ApprovalRequest.SessionChannel, "telegram")
	}
	if summary.ApprovalRequest.SessionChatID != "chat-42" {
		t.Fatalf("ApprovalRequest.SessionChatID = %q, want %q", summary.ApprovalRequest.SessionChatID, "chat-42")
	}
	if summary.ApprovalRequest.ProposedAction != "Continue build." {
		t.Fatalf("ApprovalRequest.ProposedAction = %q, want %q", summary.ApprovalRequest.ProposedAction, "Continue build.")
	}
	if summary.ApprovalRequest.WhyNeeded != "Operator approval is required." {
		t.Fatalf("ApprovalRequest.WhyNeeded = %q, want %q", summary.ApprovalRequest.WhyNeeded, "Operator approval is required.")
	}
	if summary.ApprovalRequest.AuthorityTier != AuthorityTierMedium {
		t.Fatalf("ApprovalRequest.AuthorityTier = %q, want %q", summary.ApprovalRequest.AuthorityTier, AuthorityTierMedium)
	}
	if summary.ApprovalRequest.FallbackIfDenied != "Stay waiting." {
		t.Fatalf("ApprovalRequest.FallbackIfDenied = %q, want %q", summary.ApprovalRequest.FallbackIfDenied, "Stay waiting.")
	}
	if summary.ApprovalRequest.ExpiresAt == nil || *summary.ApprovalRequest.ExpiresAt != "2026-03-24T12:05:00Z" {
		t.Fatalf("ApprovalRequest.ExpiresAt = %#v, want RFC3339 time", summary.ApprovalRequest.ExpiresAt)
	}
	if summary.ApprovalRequest.SupersededAt != nil {
		t.Fatalf("ApprovalRequest.SupersededAt = %#v, want nil for active request", summary.ApprovalRequest.SupersededAt)
	}
	if len(summary.ApprovalHistory) != 2 {
		t.Fatalf("ApprovalHistory count = %d, want %d", len(summary.ApprovalHistory), 2)
	}
	if summary.ApprovalHistory[0].StepID != "build" {
		t.Fatalf("ApprovalHistory[0].StepID = %q, want %q", summary.ApprovalHistory[0].StepID, "build")
	}
	if summary.ApprovalHistory[0].RequestedAt == nil || *summary.ApprovalHistory[0].RequestedAt != "2026-03-24T12:02:00Z" {
		t.Fatalf("ApprovalHistory[0].RequestedAt = %#v, want RFC3339 time", summary.ApprovalHistory[0].RequestedAt)
	}
	if summary.ApprovalHistory[1].StepID != "draft" {
		t.Fatalf("ApprovalHistory[1].StepID = %q, want %q", summary.ApprovalHistory[1].StepID, "draft")
	}
}

func TestBuildOperatorStatusSummaryIncludesBudgetExhaustedApprovalForPausedActiveStep(t *testing.T) {
	t.Parallel()

	requestedAt := time.Date(2026, 3, 28, 12, 2, 0, 0, time.UTC)
	triggeredAt := time.Date(2026, 3, 28, 12, 2, 30, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "wait",
		PausedReason: RuntimePauseReasonBudgetExhausted,
		PausedAt:     triggeredAt,
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "pending_approvals",
			Limit:       3,
			Observed:    4,
			Message:     "pending approval request budget exhausted",
			TriggeredAt: triggeredAt,
		},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "authorize-1",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt.Add(-2 * time.Minute),
			},
			{
				JobID:           "job-1",
				StepID:          "authorize-2",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt.Add(-time.Minute),
			},
			{
				JobID:           "job-1",
				StepID:          "authorize-3",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt.Add(-30 * time.Second),
			},
			{
				JobID:           "job-1",
				StepID:          "wait",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				RequestedVia:    ApprovalRequestedViaRuntime,
				State:           ApprovalStatePending,
				RequestedAt:     requestedAt,
				Content: &ApprovalRequestContent{
					ProposedAction:   "Continue wait step.",
					WhyNeeded:        "Operator approval is required before proceeding.",
					AuthorityTier:    AuthorityTierLow,
					FallbackIfDenied: "Keep the mission paused until an operator resolves the approval backlog.",
				},
			},
		},
	})

	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want pending request for paused active step")
	}
	if summary.ApprovalRequest.StepID != "wait" {
		t.Fatalf("ApprovalRequest.StepID = %q, want %q", summary.ApprovalRequest.StepID, "wait")
	}
	if summary.ApprovalRequest.State != ApprovalStatePending {
		t.Fatalf("ApprovalRequest.State = %q, want %q", summary.ApprovalRequest.State, ApprovalStatePending)
	}
	if summary.BudgetBlocker == nil || summary.BudgetBlocker.Observed != 4 {
		t.Fatalf("BudgetBlocker = %#v, want pending-approval blocker with observed=4", summary.BudgetBlocker)
	}
}

func TestBuildOperatorStatusSummaryIncludesGrantedApprovalBindingMetadata(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "build",
		PausedAt:     time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC),
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneSession,
				RequestedVia:    ApprovalRequestedViaRuntime,
				GrantedVia:      ApprovalGrantedViaOperatorReply,
				SessionChannel:  "slack",
				SessionChatID:   "C123::171234",
				State:           ApprovalStateGranted,
			},
		},
	})

	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.RequestedVia != ApprovalRequestedViaRuntime {
		t.Fatalf("ApprovalRequest.RequestedVia = %q, want %q", summary.ApprovalRequest.RequestedVia, ApprovalRequestedViaRuntime)
	}
	if summary.ApprovalRequest.GrantedVia != ApprovalGrantedViaOperatorReply {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want %q", summary.ApprovalRequest.GrantedVia, ApprovalGrantedViaOperatorReply)
	}
	if summary.ApprovalRequest.SessionChannel != "slack" {
		t.Fatalf("ApprovalRequest.SessionChannel = %q, want %q", summary.ApprovalRequest.SessionChannel, "slack")
	}
	if summary.ApprovalRequest.SessionChatID != "C123::171234" {
		t.Fatalf("ApprovalRequest.SessionChatID = %q, want %q", summary.ApprovalRequest.SessionChatID, "C123::171234")
	}
	if summary.PausedAt == nil || *summary.PausedAt != "2026-03-24T12:03:00Z" {
		t.Fatalf("PausedAt = %#v, want RFC3339 time", summary.PausedAt)
	}
}

func TestBuildOperatorStatusSummaryIncludesBudgetBlocker(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		ActiveStepID: "final",
		PausedReason: RuntimePauseReasonBudgetExhausted,
		PausedAt:     time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		BudgetBlocker: &RuntimeBudgetBlockerRecord{
			Ceiling:     "owner_messages",
			Limit:       20,
			Observed:    20,
			Message:     "owner-facing message budget exhausted",
			TriggeredAt: time.Date(2026, 3, 28, 11, 59, 0, 0, time.UTC),
		},
	})

	if summary.BudgetBlocker == nil {
		t.Fatal("BudgetBlocker = nil, want surfaced blocker")
	}
	if summary.BudgetBlocker.Ceiling != "owner_messages" {
		t.Fatalf("BudgetBlocker.Ceiling = %q, want %q", summary.BudgetBlocker.Ceiling, "owner_messages")
	}
	if summary.BudgetBlocker.Limit != 20 || summary.BudgetBlocker.Observed != 20 {
		t.Fatalf("BudgetBlocker limits = %#v, want limit=20 observed=20", summary.BudgetBlocker)
	}
	if summary.BudgetBlocker.Message != "owner-facing message budget exhausted" {
		t.Fatalf("BudgetBlocker.Message = %q, want exact blocker message", summary.BudgetBlocker.Message)
	}
	if summary.BudgetBlocker.TriggeredAt == nil || *summary.BudgetBlocker.TriggeredAt != "2026-03-28T11:59:00Z" {
		t.Fatalf("BudgetBlocker.TriggeredAt = %#v, want RFC3339 time", summary.BudgetBlocker.TriggeredAt)
	}
}

func TestBuildOperatorStatusSummaryIncludesRevokedApprovalState(t *testing.T) {
	t.Parallel()

	revokedAt := time.Date(2026, 3, 24, 12, 9, 0, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				RequestedVia:    ApprovalRequestedViaRuntime,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           ApprovalStateRevoked,
				RevokedAt:       revokedAt,
			},
		},
	})

	if summary.ApprovalRequest == nil {
		t.Fatal("ApprovalRequest = nil, want populated summary")
	}
	if summary.ApprovalRequest.State != ApprovalStateRevoked {
		t.Fatalf("ApprovalRequest.State = %q, want %q", summary.ApprovalRequest.State, ApprovalStateRevoked)
	}
	if summary.ApprovalRequest.GrantedVia != ApprovalGrantedViaOperatorCommand {
		t.Fatalf("ApprovalRequest.GrantedVia = %q, want %q", summary.ApprovalRequest.GrantedVia, ApprovalGrantedViaOperatorCommand)
	}
	if len(summary.ApprovalHistory) != 1 {
		t.Fatalf("ApprovalHistory count = %d, want 1", len(summary.ApprovalHistory))
	}
	if summary.ApprovalHistory[0].RevokedAt == nil || *summary.ApprovalHistory[0].RevokedAt != "2026-03-24T12:09:00Z" {
		t.Fatalf("ApprovalHistory[0].RevokedAt = %#v, want RFC3339 time", summary.ApprovalHistory[0].RevokedAt)
	}
}

func TestBuildOperatorStatusSummaryFallsBackToGrantRevokedAtForOlderSnapshots(t *testing.T) {
	t.Parallel()

	revokedAt := time.Date(2026, 3, 24, 12, 9, 0, 0, time.UTC)
	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				RequestedVia:    ApprovalRequestedViaRuntime,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           ApprovalStateRevoked,
			},
		},
		ApprovalGrants: []ApprovalGrant{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeOneJob,
				GrantedVia:      ApprovalGrantedViaOperatorCommand,
				SessionChannel:  "cli",
				SessionChatID:   "direct",
				State:           ApprovalStateRevoked,
				RevokedAt:       revokedAt,
			},
		},
	})

	if len(summary.ApprovalHistory) != 1 {
		t.Fatalf("ApprovalHistory count = %d, want 1", len(summary.ApprovalHistory))
	}
	if summary.ApprovalHistory[0].RevokedAt == nil || *summary.ApprovalHistory[0].RevokedAt != "2026-03-24T12:09:00Z" {
		t.Fatalf("ApprovalHistory[0].RevokedAt = %#v, want fallback RFC3339 time", summary.ApprovalHistory[0].RevokedAt)
	}
}

func TestBuildOperatorStatusSummaryDoesNotImplicitlySurfaceAdapterOnlyCampaignOrTreasuryFields(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		StartedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	}

	t.Run("without_treasury_preflight", func(t *testing.T) {
		t.Parallel()

		formatted, err := FormatOperatorStatusSummaryWithAllowedTools(runtime, []string{"read"})
		if err != nil {
			t.Fatalf("FormatOperatorStatusSummaryWithAllowedTools() error = %v", err)
		}

		got := mustOperatorReadoutJSONObject(t, formatted)
		assertJSONObjectKeys(t, got, "active_step_id", "allowed_tools", "job_id", "state")
		assertOperatorReadoutAdapterBoundary(t, formatted, "operator status JSON", false, false)
	})

	t.Run("with_resolved_campaign_and_treasury_preflight", func(t *testing.T) {
		t.Parallel()

		campaignPreflight := ResolvedExecutionContextCampaignPreflight{
			Campaign: &CampaignRecord{
				RecordVersion:           StoreRecordVersion,
				CampaignID:              "campaign-mail",
				CampaignKind:            CampaignKindOutreach,
				DisplayName:             "Frank Outreach",
				State:                   CampaignStateDraft,
				Objective:               "Reach aligned operators",
				GovernedExternalTargets: []AutonomyEligibilityTargetRef{{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"}},
				FrankObjectRefs: []FrankRegistryObjectRef{
					{Kind: FrankRegistryObjectKindIdentity, ObjectID: "identity-mail"},
					{Kind: FrankRegistryObjectKindAccount, ObjectID: "account-mail"},
					{Kind: FrankRegistryObjectKindContainer, ObjectID: "container-wallet"},
				},
				IdentityMode:     IdentityModeAgentAlias,
				StopConditions:   []string{"stop after 3 replies"},
				FailureThreshold: CampaignFailureThreshold{Metric: "rejections", Limit: 3},
				ComplianceChecks: []string{"can-spam-reviewed"},
				CreatedAt:        time.Date(2026, 4, 8, 20, 55, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2026, 4, 8, 20, 56, 0, 0, time.UTC),
			},
			Identities: []FrankIdentityRecord{{
				RecordVersion:        StoreRecordVersion,
				IdentityID:           "identity-mail",
				IdentityKind:         "email",
				DisplayName:          "Frank Mail",
				ProviderOrPlatformID: "provider-mail",
				IdentityMode:         IdentityModeAgentAlias,
				State:                "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
				CreatedAt:            time.Date(2026, 4, 8, 20, 50, 0, 0, time.UTC),
				UpdatedAt:            time.Date(2026, 4, 8, 20, 51, 0, 0, time.UTC),
			}},
			Accounts: []FrankAccountRecord{{
				RecordVersion:        StoreRecordVersion,
				AccountID:            "account-mail",
				AccountKind:          "mailbox",
				Label:                "Inbox",
				ProviderOrPlatformID: "provider-mail",
				IdentityID:           "identity-mail",
				ControlModel:         "agent_managed",
				RecoveryModel:        "agent_recoverable",
				State:                "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
				CreatedAt:            time.Date(2026, 4, 8, 20, 52, 0, 0, time.UTC),
				UpdatedAt:            time.Date(2026, 4, 8, 20, 53, 0, 0, time.UTC),
			}},
			Containers: []FrankContainerRecord{{
				RecordVersion:        StoreRecordVersion,
				ContainerID:          "container-wallet",
				ContainerKind:        "wallet",
				Label:                "Primary Wallet",
				ContainerClassID:     "container-class-wallet",
				State:                "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
				CreatedAt:            time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
				UpdatedAt:            time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
			}},
		}
		preflight := ResolvedExecutionContextTreasuryPreflight{
			Treasury: &TreasuryRecord{
				RecordVersion:  StoreRecordVersion,
				TreasuryID:     "treasury-wallet",
				DisplayName:    "Frank Treasury",
				State:          TreasuryStateBootstrap,
				ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
				BootstrapAcquisition: &TreasuryBootstrapAcquisition{
					EntryID:         "entry-first-value",
					AssetCode:       "USD",
					Amount:          "10.00",
					SourceRef:       "payout:listing-a",
					EvidenceLocator: "https://evidence.example/payout-a",
					ConfirmedAt:     time.Date(2026, 4, 8, 21, 2, 30, 0, time.UTC),
				},
				ContainerRefs: []FrankRegistryObjectRef{
					{
						Kind:     FrankRegistryObjectKindContainer,
						ObjectID: "container-wallet",
					},
				},
				CreatedAt: time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 8, 21, 3, 0, 0, time.UTC),
			},
			Containers: []FrankContainerRecord{
				{
					RecordVersion:    StoreRecordVersion,
					ContainerID:      "container-wallet",
					ContainerKind:    "wallet",
					Label:            "Primary Wallet",
					ContainerClassID: "container-class-wallet",
					State:            "active",
					EligibilityTargetRef: AutonomyEligibilityTargetRef{
						Kind:       EligibilityTargetKindTreasuryContainerClass,
						RegistryID: "container-class-wallet",
					},
					CreatedAt: time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
				},
			},
		}

		formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, []string{"read"}, &campaignPreflight, &preflight)
		if err != nil {
			t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight() error = %v", err)
		}

		got := mustOperatorReadoutJSONObject(t, formatted)
		assertJSONObjectKeys(t, got, "active_step_id", "allowed_tools", "campaign_preflight", "campaign_zoho_email_send_gate", "job_id", "state", "treasury_preflight")
		assertResolvedCampaignPreflightJSONEnvelope(t, got["campaign_preflight"])
		if _, ok := got["campaign_zoho_email_send_gate"].(map[string]any); !ok {
			t.Fatalf("campaign_zoho_email_send_gate = %#v, want object", got["campaign_zoho_email_send_gate"])
		}
		assertResolvedTreasuryPreflightJSONEnvelope(t, got["treasury_preflight"])
		treasuryPreflight := got["treasury_preflight"].(map[string]any)
		treasury := treasuryPreflight["treasury"].(map[string]any)
		bootstrap := treasury["bootstrap_acquisition"].(map[string]any)
		if bootstrap["entry_id"] != "entry-first-value" {
			t.Fatalf("treasury_preflight.treasury.bootstrap_acquisition.entry_id = %#v, want %q", bootstrap["entry_id"], "entry-first-value")
		}
		assertOperatorReadoutAdapterBoundary(t, formatted, "operator status JSON", true, true)
	})
}

func TestFormatOperatorStatusSummaryWithTreasuryPreflightIncludesPostBootstrapAcquisitionWhenPresent(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		StartedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	}
	preflight := ResolvedExecutionContextTreasuryPreflight{
		Treasury: &TreasuryRecord{
			RecordVersion:  StoreRecordVersion,
			TreasuryID:     "treasury-wallet",
			DisplayName:    "Frank Treasury",
			State:          TreasuryStateActive,
			ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
			PostBootstrapAcquisition: &TreasuryPostBootstrapAcquisition{
				AssetCode:       "USD",
				Amount:          "2.25",
				SourceRef:       "payout:listing-b",
				EvidenceLocator: "https://evidence.example/payout-b",
				ConfirmedAt:     time.Date(2026, 4, 8, 21, 4, 0, 0, time.UTC),
				ConsumedEntryID: "entry-post-value",
			},
			ContainerRefs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: "container-wallet",
				},
			},
			CreatedAt: time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC),
		},
		Containers: []FrankContainerRecord{
			{
				RecordVersion:    StoreRecordVersion,
				ContainerID:      "container-wallet",
				ContainerKind:    "wallet",
				Label:            "Primary Wallet",
				ContainerClassID: "container-class-wallet",
				State:            "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{
					Kind:       EligibilityTargetKindTreasuryContainerClass,
					RegistryID: "container-class-wallet",
				},
				CreatedAt: time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
			},
		},
	}

	formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight(runtime, []string{"read"}, &preflight)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	assertResolvedTreasuryPreflightJSONEnvelope(t, got["treasury_preflight"])
	treasury := got["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postBootstrap := treasury["post_bootstrap_acquisition"].(map[string]any)
	if postBootstrap["source_ref"] != "payout:listing-b" {
		t.Fatalf("treasury_preflight.treasury.post_bootstrap_acquisition.source_ref = %#v, want %q", postBootstrap["source_ref"], "payout:listing-b")
	}
	if postBootstrap["consumed_entry_id"] != "entry-post-value" {
		t.Fatalf("treasury_preflight.treasury.post_bootstrap_acquisition.consumed_entry_id = %#v, want %q", postBootstrap["consumed_entry_id"], "entry-post-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "operator status JSON", false, true)
}

func TestFormatOperatorStatusSummaryWithTreasuryPreflightIncludesPostActiveSaveWhenPresent(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		StartedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	}
	preflight := ResolvedExecutionContextTreasuryPreflight{
		Treasury: &TreasuryRecord{
			RecordVersion:  StoreRecordVersion,
			TreasuryID:     "treasury-wallet",
			DisplayName:    "Frank Treasury",
			State:          TreasuryStateActive,
			ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
			PostActiveSave: &TreasuryPostActiveSave{
				AssetCode:         "USD",
				Amount:            "1.25",
				TargetContainerID: "container-savings",
				SourceRef:         "transfer:reserve-a",
				EvidenceLocator:   "https://evidence.example/save-a",
				ConsumedEntryID:   "entry-save-value",
			},
			ContainerRefs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: "container-wallet",
				},
			},
			CreatedAt: time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC),
		},
		Containers: []FrankContainerRecord{
			{
				RecordVersion:    StoreRecordVersion,
				ContainerID:      "container-wallet",
				ContainerKind:    "wallet",
				Label:            "Primary Wallet",
				ContainerClassID: "container-class-wallet",
				State:            "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{
					Kind:       EligibilityTargetKindTreasuryContainerClass,
					RegistryID: "container-class-wallet",
				},
				CreatedAt: time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
			},
		},
	}

	formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight(runtime, []string{"read"}, &preflight)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	assertResolvedTreasuryPreflightJSONEnvelope(t, got["treasury_preflight"])
	treasury := got["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveSave := treasury["post_active_save"].(map[string]any)
	if postActiveSave["target_container_id"] != "container-savings" {
		t.Fatalf("treasury_preflight.treasury.post_active_save.target_container_id = %#v, want %q", postActiveSave["target_container_id"], "container-savings")
	}
	if postActiveSave["consumed_entry_id"] != "entry-save-value" {
		t.Fatalf("treasury_preflight.treasury.post_active_save.consumed_entry_id = %#v, want %q", postActiveSave["consumed_entry_id"], "entry-save-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "operator status JSON", false, true)
}

func TestFormatOperatorStatusSummaryWithTreasuryPreflightIncludesPostActiveTransferWhenPresent(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		StartedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	}
	preflight := ResolvedExecutionContextTreasuryPreflight{
		Treasury: &TreasuryRecord{
			RecordVersion:  StoreRecordVersion,
			TreasuryID:     "treasury-wallet",
			DisplayName:    "Frank Treasury",
			State:          TreasuryStateActive,
			ZeroSeedPolicy: TreasuryZeroSeedPolicyOwnerSeedForbidden,
			PostActiveTransfer: &TreasuryPostActiveTransfer{
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
			},
			ContainerRefs: []FrankRegistryObjectRef{
				{
					Kind:     FrankRegistryObjectKindContainer,
					ObjectID: "container-wallet",
				},
			},
			CreatedAt: time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 8, 21, 5, 0, 0, time.UTC),
		},
		Containers: []FrankContainerRecord{
			{
				RecordVersion:    StoreRecordVersion,
				ContainerID:      "container-wallet",
				ContainerKind:    "wallet",
				Label:            "Primary Wallet",
				ContainerClassID: "container-class-wallet",
				State:            "active",
				EligibilityTargetRef: AutonomyEligibilityTargetRef{
					Kind:       EligibilityTargetKindTreasuryContainerClass,
					RegistryID: "container-class-wallet",
				},
				CreatedAt: time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
			},
		},
	}

	formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight(runtime, []string{"read"}, &preflight)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	assertResolvedTreasuryPreflightJSONEnvelope(t, got["treasury_preflight"])
	treasury := got["treasury_preflight"].(map[string]any)["treasury"].(map[string]any)
	postActiveTransfer := treasury["post_active_transfer"].(map[string]any)
	sourceRef := postActiveTransfer["source_container_ref"].(map[string]any)
	targetRef := postActiveTransfer["target_container_ref"].(map[string]any)
	if sourceRef["object_id"] != "container-wallet" {
		t.Fatalf("treasury_preflight.treasury.post_active_transfer.source_container_ref.object_id = %#v, want %q", sourceRef["object_id"], "container-wallet")
	}
	if targetRef["object_id"] != "container-vault" {
		t.Fatalf("treasury_preflight.treasury.post_active_transfer.target_container_ref.object_id = %#v, want %q", targetRef["object_id"], "container-vault")
	}
	if postActiveTransfer["consumed_entry_id"] != "entry-transfer-value" {
		t.Fatalf("treasury_preflight.treasury.post_active_transfer.consumed_entry_id = %#v, want %q", postActiveTransfer["consumed_entry_id"], "entry-transfer-value")
	}
	assertOperatorReadoutAdapterBoundary(t, formatted, "operator status JSON", false, true)
}

func TestBuildOperatorStatusSummaryIncludesBoundedDeterministicApprovalHistory(t *testing.T) {
	t.Parallel()

	requests := make([]ApprovalRequest, 0, OperatorStatusApprovalHistoryLimit+2)
	for i := 0; i < OperatorStatusApprovalHistoryLimit+2; i++ {
		requests = append(requests, ApprovalRequest{
			JobID:           "job-1",
			StepID:          "step-" + string(rune('a'+i)),
			RequestedAction: ApprovalRequestedActionStepComplete,
			Scope:           ApprovalScopeMissionStep,
			RequestedVia:    ApprovalRequestedViaRuntime,
			State:           ApprovalStatePending,
			RequestedAt:     time.Date(2026, 3, 24, 12, i, 0, 0, time.UTC),
		})
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:            "job-1",
		State:            JobStateWaitingUser,
		ApprovalRequests: requests,
	})

	if len(summary.ApprovalHistory) != OperatorStatusApprovalHistoryLimit {
		t.Fatalf("ApprovalHistory count = %d, want %d", len(summary.ApprovalHistory), OperatorStatusApprovalHistoryLimit)
	}
	for i, wantStep := range []string{"step-g", "step-f", "step-e", "step-d", "step-c"} {
		if summary.ApprovalHistory[i].StepID != wantStep {
			t.Fatalf("ApprovalHistory[%d].StepID = %q, want %q", i, summary.ApprovalHistory[i].StepID, wantStep)
		}
	}
	if summary.Truncation == nil {
		t.Fatal("Truncation = nil, want approval_history truncation metadata")
	}
	if summary.Truncation.ApprovalHistoryOmitted != 2 {
		t.Fatalf("Truncation.ApprovalHistoryOmitted = %d, want 2", summary.Truncation.ApprovalHistoryOmitted)
	}
	if summary.Truncation.RecentAuditOmitted != 0 {
		t.Fatalf("Truncation.RecentAuditOmitted = %d, want 0", summary.Truncation.RecentAuditOmitted)
	}
}

func TestBuildOperatorStatusSummaryIncludesDeterministicRecentAuditSubset(t *testing.T) {
	t.Parallel()

	history := AppendAuditHistory(nil, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "write_memory", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "pause", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "resume", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "build", ToolName: "abort", Allowed: false, Code: RejectionCodeInvalidRuntimeState, Timestamp: time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "final", ToolName: "status", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 4, 0, 0, time.UTC)})
	history = AppendAuditHistory(history, AuditEvent{JobID: "job-1", StepID: "final", ToolName: "set_step", Allowed: true, Timestamp: time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)})

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStatePaused,
		AuditHistory: history,
	}

	summary := BuildOperatorStatusSummary(runtime)
	if len(summary.RecentAudit) != OperatorStatusRecentAuditLimit {
		t.Fatalf("RecentAudit count = %d, want %d", len(summary.RecentAudit), OperatorStatusRecentAuditLimit)
	}

	for i, want := range []struct {
		action string
		stepID string
		class  AuditActionClass
		result AuditResult
		code   RejectionCode
		at     string
	}{
		{action: "set_step", stepID: "final", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:05:00Z"},
		{action: "status", stepID: "final", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:04:00Z"},
		{action: "abort", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultRejected, code: RejectionCode("E_STEP_OUT_OF_ORDER"), at: "2026-03-24T12:03:00Z"},
		{action: "resume", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:02:00Z"},
		{action: "pause", stepID: "build", class: AuditActionClassOperatorCommand, result: AuditResultApplied, at: "2026-03-24T12:01:00Z"},
	} {
		got := summary.RecentAudit[i]
		if got.EventID == "" {
			t.Fatalf("RecentAudit[%d].EventID = empty, want deterministic id", i)
		}
		if got.JobID != "job-1" {
			t.Fatalf("RecentAudit[%d].JobID = %q, want %q", i, got.JobID, "job-1")
		}
		if got.StepID != want.stepID {
			t.Fatalf("RecentAudit[%d].StepID = %q, want %q", i, got.StepID, want.stepID)
		}
		if got.Action != want.action {
			t.Fatalf("RecentAudit[%d].Action = %q, want %q", i, got.Action, want.action)
		}
		if got.ActionClass != want.class {
			t.Fatalf("RecentAudit[%d].ActionClass = %q, want %q", i, got.ActionClass, want.class)
		}
		if got.Result != want.result {
			t.Fatalf("RecentAudit[%d].Result = %q, want %q", i, got.Result, want.result)
		}
		if got.Code != want.code {
			t.Fatalf("RecentAudit[%d].Code = %q, want %q", i, got.Code, want.code)
		}
		if got.Timestamp != want.at {
			t.Fatalf("RecentAudit[%d].Timestamp = %q, want %q", i, got.Timestamp, want.at)
		}
	}
	if summary.Truncation == nil {
		t.Fatal("Truncation = nil, want recent_audit truncation metadata")
	}
	if summary.Truncation.RecentAuditOmitted != 1 {
		t.Fatalf("Truncation.RecentAuditOmitted = %d, want 1", summary.Truncation.RecentAuditOmitted)
	}
	if summary.Truncation.ApprovalHistoryOmitted != 0 {
		t.Fatalf("Truncation.ApprovalHistoryOmitted = %d, want 0", summary.Truncation.ApprovalHistoryOmitted)
	}
}

func TestBuildOperatorStatusSummaryIncludesArtifactsInPlanOrder(t *testing.T) {
	t.Parallel()

	job := Job{
		ID:           "job-1",
		SpecVersion:  JobSpecVersionV2,
		MaxAuthority: AuthorityTierHigh,
		AllowedTools: []string{"read"},
		Plan: Plan{
			ID: "plan-1",
			Steps: []Step{
				{ID: "gamma", Type: StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
				{ID: "alpha", Type: StepTypeStaticArtifact, StaticArtifactPath: "alpha.json", StaticArtifactFormat: "json"},
				{ID: "beta", Type: StepTypeLongRunningCode, LongRunningArtifactPath: "service.bin", LongRunningStartupCommand: []string{"go", "build", "./cmd/service"}},
				{ID: "delta", Type: StepTypeStaticArtifact, StaticArtifactPath: "delta.md", StaticArtifactFormat: "markdown"},
				{ID: "epsilon", Type: StepTypeOneShotCode, OneShotArtifactPath: "epsilon.go"},
				{ID: "zeta", Type: StepTypeStaticArtifact, StaticArtifactPath: "zeta.yaml", StaticArtifactFormat: "yaml"},
				{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"zeta"}},
			},
		},
	}
	plan, err := BuildInspectablePlanContext(job)
	if err != nil {
		t.Fatalf("BuildInspectablePlanContext() error = %v", err)
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:           "job-1",
		State:           JobStatePaused,
		InspectablePlan: &plan,
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "zeta", At: time.Date(2026, 3, 24, 12, 5, 0, 0, time.UTC)},
			{StepID: "gamma", At: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)},
			{StepID: "beta", At: time.Date(2026, 3, 24, 12, 2, 0, 0, time.UTC), ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeLongRunningCode), Target: "service.bin", State: "already_present"}},
			{StepID: "alpha", At: time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC)},
			{StepID: "epsilon", At: time.Date(2026, 3, 24, 12, 4, 0, 0, time.UTC)},
			{StepID: "delta", At: time.Date(2026, 3, 24, 12, 3, 0, 0, time.UTC)},
		},
	})

	if len(summary.Artifacts) != OperatorStatusArtifactLimit {
		t.Fatalf("Artifacts count = %d, want %d", len(summary.Artifacts), OperatorStatusArtifactLimit)
	}
	for i, want := range []struct {
		stepID   string
		stepType StepType
		path     string
		state    string
	}{
		{stepID: "gamma", stepType: StepTypeOneShotCode, path: "zeta.txt"},
		{stepID: "alpha", stepType: StepTypeStaticArtifact, path: "alpha.json"},
		{stepID: "beta", stepType: StepTypeLongRunningCode, path: "service.bin", state: "already_present"},
		{stepID: "delta", stepType: StepTypeStaticArtifact, path: "delta.md"},
		{stepID: "epsilon", stepType: StepTypeOneShotCode, path: "epsilon.go"},
	} {
		got := summary.Artifacts[i]
		if got.StepID != want.stepID {
			t.Fatalf("Artifacts[%d].StepID = %q, want %q", i, got.StepID, want.stepID)
		}
		if got.StepType != want.stepType {
			t.Fatalf("Artifacts[%d].StepType = %q, want %q", i, got.StepType, want.stepType)
		}
		if got.Path != want.path {
			t.Fatalf("Artifacts[%d].Path = %q, want %q", i, got.Path, want.path)
		}
		if got.State != want.state {
			t.Fatalf("Artifacts[%d].State = %q, want %q", i, got.State, want.state)
		}
	}
	if summary.Truncation == nil {
		t.Fatal("Truncation = nil, want artifact truncation metadata")
	}
	if summary.Truncation.ArtifactsOmitted != 1 {
		t.Fatalf("Truncation.ArtifactsOmitted = %d, want 1", summary.Truncation.ArtifactsOmitted)
	}
}

func TestBuildOperatorStatusSummaryFallsBackToLexicographicArtifactsWithoutPlan(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStatePaused,
		CompletedSteps: []RuntimeStepRecord{
			{StepID: "step-b", ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeOneShotCode), Target: "b.go", State: "already_present"}},
			{StepID: "step-a", ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeStaticArtifact), Target: "a.json", State: "already_present"}},
		},
	})

	if len(summary.Artifacts) != 2 {
		t.Fatalf("Artifacts count = %d, want 2", len(summary.Artifacts))
	}
	if summary.Artifacts[0].Path != "a.json" || summary.Artifacts[1].Path != "b.go" {
		t.Fatalf("Artifacts paths = (%q, %q), want (%q, %q)", summary.Artifacts[0].Path, summary.Artifacts[1].Path, "a.json", "b.go")
	}
	if summary.Truncation != nil {
		t.Fatalf("Truncation = %#v, want nil without omissions", summary.Truncation)
	}
}

func TestBuildOperatorStatusSummaryIncludesFrankZohoSendProof(t *testing.T) {
	t.Parallel()

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStatePaused,
		FrankZohoSendReceipts: []FrankZohoSendReceipt{
			{
				StepID:             "build",
				Provider:           "zoho_mail",
				ProviderAccountID:  "3323462000000008002",
				FromAddress:        "frank@omou.online",
				FromDisplayName:    "Frank",
				ProviderMessageID:  "1711540357880100000",
				ProviderMailID:     "<mail-1@zoho.test>",
				MIMEMessageID:      "<mime-1@example.test>",
				OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
			},
		},
	})

	if len(summary.FrankZohoSendProof) != 1 {
		t.Fatalf("FrankZohoSendProof len = %d, want 1", len(summary.FrankZohoSendProof))
	}
	proof := summary.FrankZohoSendProof[0]
	if proof.StepID != "build" {
		t.Fatalf("FrankZohoSendProof[0].StepID = %q, want %q", proof.StepID, "build")
	}
	if proof.ProviderMessageID != "1711540357880100000" {
		t.Fatalf("FrankZohoSendProof[0].ProviderMessageID = %q, want canonical provider message id", proof.ProviderMessageID)
	}
	if proof.ProviderMailID != "<mail-1@zoho.test>" {
		t.Fatalf("FrankZohoSendProof[0].ProviderMailID = %q, want secondary provider mail id", proof.ProviderMailID)
	}
	if proof.MIMEMessageID != "<mime-1@example.test>" {
		t.Fatalf("FrankZohoSendProof[0].MIMEMessageID = %q, want secondary MIME message id", proof.MIMEMessageID)
	}
	if proof.ProviderAccountID != "3323462000000008002" {
		t.Fatalf("FrankZohoSendProof[0].ProviderAccountID = %q, want proof locator account id", proof.ProviderAccountID)
	}
	if proof.OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("FrankZohoSendProof[0].OriginalMessageURL = %q, want proof-compatible originalmessage URL", proof.OriginalMessageURL)
	}
}

func TestBuildOperatorStatusSummaryIncludesCampaignZohoEmailOutbounds(t *testing.T) {
	t.Parallel()

	preparedAt := time.Date(2026, 4, 15, 16, 0, 0, 0, time.UTC)
	sentAt := preparedAt.Add(time.Minute)
	action, err := BuildCampaignZohoEmailOutboundPreparedAction(
		"build",
		"campaign-mail",
		"3323462000000008002",
		"frank@omou.online",
		"Frank",
		CampaignZohoEmailAddressing{
			To:  []string{"person@example.com"},
			CC:  []string{"copy@example.com"},
			BCC: []string{"blind@example.com"},
		},
		"Frank intro",
		"plaintext",
		"Hello from Frank",
		preparedAt,
	)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundPreparedAction() error = %v", err)
	}
	action, err = BuildCampaignZohoEmailOutboundSentAction(action, FrankZohoSendReceipt{
		StepID:             "build",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		FromAddress:        "frank@omou.online",
		FromDisplayName:    "Frank",
		ProviderMessageID:  "1711540357880100000",
		ProviderMailID:     "<mail-1@zoho.test>",
		MIMEMessageID:      "<mime-1@example.test>",
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage",
	}, sentAt)
	if err != nil {
		t.Fatalf("BuildCampaignZohoEmailOutboundSentAction() error = %v", err)
	}
	action.State = CampaignZohoEmailOutboundActionStateVerified
	action.VerifiedAt = sentAt.Add(time.Minute)
	if err := ValidateCampaignZohoEmailOutboundAction(action); err != nil {
		t.Fatalf("ValidateCampaignZohoEmailOutboundAction(verified) error = %v", err)
	}

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:                            "job-1",
		State:                            JobStatePaused,
		CampaignZohoEmailOutboundActions: []CampaignZohoEmailOutboundAction{action},
	})

	if len(summary.CampaignZohoEmailOutbounds) != 1 {
		t.Fatalf("CampaignZohoEmailOutbounds len = %d, want 1", len(summary.CampaignZohoEmailOutbounds))
	}
	outbound := summary.CampaignZohoEmailOutbounds[0]
	if outbound.ActionID != action.ActionID {
		t.Fatalf("CampaignZohoEmailOutbounds[0].ActionID = %q, want %q", outbound.ActionID, action.ActionID)
	}
	if outbound.State != "verified" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].State = %q, want verified", outbound.State)
	}
	if outbound.CampaignID != "campaign-mail" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].CampaignID = %q, want campaign-mail", outbound.CampaignID)
	}
	if outbound.To[0] != "person@example.com" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].To[0] = %q, want person@example.com", outbound.To[0])
	}
	if outbound.Subject != "Frank intro" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].Subject = %q, want Frank intro", outbound.Subject)
	}
	if outbound.BodySHA256 != CampaignZohoEmailBodySHA256("Hello from Frank") {
		t.Fatalf("CampaignZohoEmailOutbounds[0].BodySHA256 = %q, want outbound body digest", outbound.BodySHA256)
	}
	if outbound.ProviderMessageID != "1711540357880100000" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].ProviderMessageID = %q, want canonical provider message id", outbound.ProviderMessageID)
	}
	if outbound.OriginalMessageURL != "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880100000/originalmessage" {
		t.Fatalf("CampaignZohoEmailOutbounds[0].OriginalMessageURL = %q, want proof-compatible originalmessage URL", outbound.OriginalMessageURL)
	}
	if outbound.VerifiedAt == nil || *outbound.VerifiedAt != sentAt.Add(time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("CampaignZohoEmailOutbounds[0].VerifiedAt = %#v, want verified timestamp", outbound.VerifiedAt)
	}
}

func TestFormatOperatorStatusSummarySurfacesCampaignZohoEmailAddressingInCampaignPreflight(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
		StartedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 24, 12, 1, 0, 0, time.UTC),
	}
	campaignPreflight := ResolvedExecutionContextCampaignPreflight{
		Campaign: &CampaignRecord{
			RecordVersion:           StoreRecordVersion,
			CampaignID:              "campaign-mail",
			CampaignKind:            CampaignKindOutreach,
			DisplayName:             "Frank Outreach",
			State:                   CampaignStateDraft,
			Objective:               "Reach aligned operators",
			GovernedExternalTargets: []AutonomyEligibilityTargetRef{{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"}},
			FrankObjectRefs: []FrankRegistryObjectRef{
				{Kind: FrankRegistryObjectKindIdentity, ObjectID: "identity-mail"},
				{Kind: FrankRegistryObjectKindAccount, ObjectID: "account-mail"},
				{Kind: FrankRegistryObjectKindContainer, ObjectID: "container-wallet"},
			},
			IdentityMode:     IdentityModeAgentAlias,
			StopConditions:   []string{"stop after 3 replies"},
			FailureThreshold: CampaignFailureThreshold{Metric: "rejections", Limit: 3},
			ComplianceChecks: []string{"can-spam-reviewed"},
			ZohoEmailAddressing: &CampaignZohoEmailAddressing{
				To:  []string{"person@example.com", "team@example.com"},
				CC:  []string{"copy@example.com"},
				BCC: []string{"blind@example.com"},
			},
			CreatedAt: time.Date(2026, 4, 8, 20, 55, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 8, 20, 56, 0, 0, time.UTC),
		},
		Identities: []FrankIdentityRecord{{
			RecordVersion:        StoreRecordVersion,
			IdentityID:           "identity-mail",
			IdentityKind:         "email",
			DisplayName:          "Frank Mail",
			ProviderOrPlatformID: "provider-mail",
			IdentityMode:         IdentityModeAgentAlias,
			State:                "active",
			EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"},
			CreatedAt:            time.Date(2026, 4, 8, 20, 50, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 4, 8, 20, 51, 0, 0, time.UTC),
		}},
		Accounts: []FrankAccountRecord{{
			RecordVersion:        StoreRecordVersion,
			AccountID:            "account-mail",
			AccountKind:          "mailbox",
			Label:                "Inbox",
			ProviderOrPlatformID: "provider-mail",
			IdentityID:           "identity-mail",
			ControlModel:         "agent_managed",
			RecoveryModel:        "agent_recoverable",
			State:                "active",
			EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindAccountClass, RegistryID: "account-class-mailbox"},
			CreatedAt:            time.Date(2026, 4, 8, 20, 52, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 4, 8, 20, 53, 0, 0, time.UTC),
		}},
		Containers: []FrankContainerRecord{{
			RecordVersion:        StoreRecordVersion,
			ContainerID:          "container-wallet",
			ContainerKind:        "wallet",
			Label:                "Primary Wallet",
			ContainerClassID:     "container-class-wallet",
			State:                "active",
			EligibilityTargetRef: AutonomyEligibilityTargetRef{Kind: EligibilityTargetKindTreasuryContainerClass, RegistryID: "container-class-wallet"},
			CreatedAt:            time.Date(2026, 4, 8, 21, 1, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 4, 8, 21, 2, 0, 0, time.UTC),
		}},
	}

	formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, []string{"read"}, &campaignPreflight, nil)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	preflight := got["campaign_preflight"].(map[string]any)
	campaign := preflight["campaign"].(map[string]any)
	addressing, ok := campaign["zoho_email_addressing"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing = %#v, want object", campaign["zoho_email_addressing"])
	}
	assertJSONObjectKeys(t, campaign, "campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at", "zoho_email_addressing")
	if !reflect.DeepEqual(mustJSONArray(t, addressing["to"], "campaign_preflight.campaign.zoho_email_addressing.to"), []any{"person@example.com", "team@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.to = %#v, want [person@example.com team@example.com]", addressing["to"])
	}
	if !reflect.DeepEqual(mustJSONArray(t, addressing["cc"], "campaign_preflight.campaign.zoho_email_addressing.cc"), []any{"copy@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.cc = %#v, want [copy@example.com]", addressing["cc"])
	}
	if !reflect.DeepEqual(mustJSONArray(t, addressing["bcc"], "campaign_preflight.campaign.zoho_email_addressing.bcc"), []any{"blind@example.com"}) {
		t.Fatalf("campaign_preflight.campaign.zoho_email_addressing.bcc = %#v, want [blind@example.com]", addressing["bcc"])
	}
}

func TestFormatOperatorStatusSummarySurfacesCampaignZohoEmailSendGate(t *testing.T) {
	t.Parallel()

	runtime := JobRuntimeState{
		JobID:        "job-1",
		State:        JobStateRunning,
		ActiveStepID: "build",
	}
	campaignPreflight := ResolvedExecutionContextCampaignPreflight{
		Campaign: &CampaignRecord{
			RecordVersion:           StoreRecordVersion,
			CampaignID:              "campaign-mail",
			CampaignKind:            CampaignKindOutreach,
			DisplayName:             "Frank Outreach",
			State:                   CampaignStateActive,
			Objective:               "Reach aligned operators",
			GovernedExternalTargets: []AutonomyEligibilityTargetRef{{Kind: EligibilityTargetKindProvider, RegistryID: "provider-mail"}},
			FrankObjectRefs: []FrankRegistryObjectRef{
				{Kind: FrankRegistryObjectKindIdentity, ObjectID: "identity-mail"},
			},
			IdentityMode:     IdentityModeAgentAlias,
			StopConditions:   []string{"stop after 3 replies"},
			FailureThreshold: CampaignFailureThreshold{Metric: "rejections", Limit: 3},
			ComplianceChecks: []string{"can-spam-reviewed"},
			CreatedAt:        time.Date(2026, 4, 8, 20, 55, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 4, 8, 20, 56, 0, 0, time.UTC),
		},
	}

	formatted, err := FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, []string{"read"}, &campaignPreflight, nil)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight() error = %v", err)
	}

	got := mustOperatorReadoutJSONObject(t, formatted)
	gateJSON, ok := got["campaign_zoho_email_send_gate"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want object", got["campaign_zoho_email_send_gate"])
	}
	assertJSONObjectKeys(t, gateJSON, "allowed", "ambiguous_outcome_count", "attributed_reply_count", "campaign_id", "failure_count", "failure_threshold_limit", "failure_threshold_metric", "halted", "verified_success_count")
	if gateJSON["campaign_id"] != "campaign-mail" {
		t.Fatalf("campaign_zoho_email_send_gate.campaign_id = %#v, want campaign-mail", gateJSON["campaign_id"])
	}
	if gateJSON["allowed"] != true || gateJSON["halted"] != false {
		t.Fatalf("campaign_zoho_email_send_gate = %#v, want allowed non-halted gate", gateJSON)
	}
	if gateJSON["failure_threshold_metric"] != "rejections" {
		t.Fatalf("campaign_zoho_email_send_gate.failure_threshold_metric = %#v, want rejections", gateJSON["failure_threshold_metric"])
	}
	if gateJSON["failure_threshold_limit"] != float64(3) {
		t.Fatalf("campaign_zoho_email_send_gate.failure_threshold_limit = %#v, want 3", gateJSON["failure_threshold_limit"])
	}
}

func TestBuildOperatorStatusSummaryIncludesFrankZohoInboundReplies(t *testing.T) {
	t.Parallel()

	receivedAt := time.Date(2026, 4, 15, 20, 45, 0, 0, time.UTC)
	reply := NormalizeFrankZohoInboundReply(FrankZohoInboundReply{
		ReplyID:            "frank_zoho_inbound_reply_123",
		StepID:             "sync-replies",
		Provider:           "zoho_mail",
		ProviderAccountID:  "3323462000000008002",
		ProviderMessageID:  "1711540357880102000",
		ProviderMailID:     "<reply-1@zoho.test>",
		MIMEMessageID:      "<reply-1@example.test>",
		InReplyTo:          "<parent@example.test>",
		References:         []string{"<seed@example.test>", "<parent@example.test>"},
		FromAddress:        "person@example.com",
		FromDisplayName:    "Person One",
		FromAddressCount:   1,
		Subject:            "Re: Frank intro",
		ReceivedAt:         receivedAt,
		OriginalMessageURL: "https://mail.zoho.com/api/accounts/3323462000000008002/messages/1711540357880102000/originalmessage",
	})

	summary := BuildOperatorStatusSummary(JobRuntimeState{
		JobID:                   "job-1",
		State:                   JobStatePaused,
		FrankZohoInboundReplies: []FrankZohoInboundReply{reply},
	})

	if len(summary.FrankZohoInboundReplies) != 1 {
		t.Fatalf("FrankZohoInboundReplies len = %d, want 1", len(summary.FrankZohoInboundReplies))
	}
	got := summary.FrankZohoInboundReplies[0]
	if got.ReplyID != reply.ReplyID {
		t.Fatalf("FrankZohoInboundReplies[0].ReplyID = %q, want %q", got.ReplyID, reply.ReplyID)
	}
	if got.FromAddress != "person@example.com" {
		t.Fatalf("FrankZohoInboundReplies[0].FromAddress = %q, want person@example.com", got.FromAddress)
	}
	if got.FromAddressCount != 1 {
		t.Fatalf("FrankZohoInboundReplies[0].FromAddressCount = %d, want 1", got.FromAddressCount)
	}
	if got.InReplyTo != "<parent@example.test>" {
		t.Fatalf("FrankZohoInboundReplies[0].InReplyTo = %q, want parent linkage", got.InReplyTo)
	}
	if got.ReceivedAt == nil || *got.ReceivedAt != receivedAt.Format(time.RFC3339Nano) {
		t.Fatalf("FrankZohoInboundReplies[0].ReceivedAt = %#v, want received timestamp", got.ReceivedAt)
	}
}

func TestFormatOperatorStatusSummaryWithAllowedToolsUsesSortedUniqueIntersection(t *testing.T) {
	t.Parallel()

	formatted, err := FormatOperatorStatusSummaryWithAllowedTools(
		JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "build",
		},
		EffectiveAllowedTools(
			&Job{AllowedTools: []string{"write", "read", "read", "shell"}},
			&Step{AllowedTools: []string{"shell", "read", "read", "missing"}},
		),
	)
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummaryWithAllowedTools() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(formatted), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	allowedTools, ok := got["allowed_tools"].([]any)
	if !ok || len(allowedTools) != 2 {
		t.Fatalf("allowed_tools = %#v, want two entries", got["allowed_tools"])
	}
	if allowedTools[0] != "read" || allowedTools[1] != "shell" {
		t.Fatalf("allowed_tools = %#v, want [read shell]", allowedTools)
	}
}

func TestFormatOperatorStatusSummaryProducesDeterministicJSONForTerminalRuntime(t *testing.T) {
	t.Parallel()

	terminalAudit := AppendAuditHistory(nil, AuditEvent{
		JobID:     "job-1",
		StepID:    "build",
		ToolName:  "abort",
		Allowed:   true,
		Timestamp: time.Date(2026, 3, 24, 12, 8, 0, 0, time.UTC),
	})[0]

	formatted, err := FormatOperatorStatusSummary(JobRuntimeState{
		JobID:         "job-1",
		State:         JobStateAborted,
		AbortedReason: RuntimeAbortReasonOperatorCommand,
		AuditHistory:  []AuditEvent{terminalAudit},
		ApprovalRequests: []ApprovalRequest{
			{
				JobID:           "job-1",
				StepID:          "build",
				RequestedAction: ApprovalRequestedActionStepComplete,
				Scope:           ApprovalScopeMissionStep,
				State:           ApprovalStateSuperseded,
				RequestedAt:     time.Date(2026, 3, 24, 12, 6, 0, 0, time.UTC),
				ResolvedAt:      time.Date(2026, 3, 24, 12, 7, 0, 0, time.UTC),
				ExpiresAt:       time.Date(2026, 3, 24, 12, 9, 0, 0, time.UTC),
				SupersededAt:    time.Date(2026, 3, 24, 12, 7, 0, 0, time.UTC),
			},
		},
	})
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummary() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(formatted), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got["job_id"] != "job-1" {
		t.Fatalf("job_id = %#v, want %q", got["job_id"], "job-1")
	}
	if got["state"] != string(JobStateAborted) {
		t.Fatalf("state = %#v, want %q", got["state"], JobStateAborted)
	}
	if got["aborted_reason"] != RuntimeAbortReasonOperatorCommand {
		t.Fatalf("aborted_reason = %#v, want %q", got["aborted_reason"], RuntimeAbortReasonOperatorCommand)
	}

	request, ok := got["approval_request"].(map[string]any)
	if !ok {
		t.Fatalf("approval_request = %#v, want object", got["approval_request"])
	}
	if request["state"] != string(ApprovalStateSuperseded) {
		t.Fatalf("approval_request.state = %#v, want %q", request["state"], ApprovalStateSuperseded)
	}
	if request["superseded_at"] != "2026-03-24T12:07:00Z" {
		t.Fatalf("approval_request.superseded_at = %#v, want RFC3339 time", request["superseded_at"])
	}
	approvalHistory, ok := got["approval_history"].([]any)
	if !ok || len(approvalHistory) != 1 {
		t.Fatalf("approval_history = %#v, want one approval entry", got["approval_history"])
	}
	historyEntry, ok := approvalHistory[0].(map[string]any)
	if !ok {
		t.Fatalf("approval_history[0] = %#v, want object", approvalHistory[0])
	}
	if historyEntry["state"] != string(ApprovalStateSuperseded) {
		t.Fatalf("approval_history[0].state = %#v, want %q", historyEntry["state"], ApprovalStateSuperseded)
	}
	if historyEntry["requested_at"] != "2026-03-24T12:06:00Z" {
		t.Fatalf("approval_history[0].requested_at = %#v, want %q", historyEntry["requested_at"], "2026-03-24T12:06:00Z")
	}
	if historyEntry["resolved_at"] != "2026-03-24T12:07:00Z" {
		t.Fatalf("approval_history[0].resolved_at = %#v, want %q", historyEntry["resolved_at"], "2026-03-24T12:07:00Z")
	}
	if historyEntry["expires_at"] != "2026-03-24T12:09:00Z" {
		t.Fatalf("approval_history[0].expires_at = %#v, want %q", historyEntry["expires_at"], "2026-03-24T12:09:00Z")
	}

	recentAudit, ok := got["recent_audit"].([]any)
	if !ok || len(recentAudit) != 1 {
		t.Fatalf("recent_audit = %#v, want one audit entry", got["recent_audit"])
	}
	entry, ok := recentAudit[0].(map[string]any)
	if !ok {
		t.Fatalf("recent_audit[0] = %#v, want object", recentAudit[0])
	}
	if entry["job_id"] != "job-1" {
		t.Fatalf("recent_audit[0].job_id = %#v, want %q", entry["job_id"], "job-1")
	}
	if entry["event_id"] != terminalAudit.EventID {
		t.Fatalf("recent_audit[0].event_id = %#v, want %q", entry["event_id"], terminalAudit.EventID)
	}
	if entry["action"] != "abort" {
		t.Fatalf("recent_audit[0].action = %#v, want %q", entry["action"], "abort")
	}
	if entry["action_class"] != string(AuditActionClassOperatorCommand) {
		t.Fatalf("recent_audit[0].action_class = %#v, want %q", entry["action_class"], AuditActionClassOperatorCommand)
	}
	if entry["result"] != string(AuditResultApplied) {
		t.Fatalf("recent_audit[0].result = %#v, want %q", entry["result"], AuditResultApplied)
	}
	if entry["timestamp"] != "2026-03-24T12:08:00Z" {
		t.Fatalf("recent_audit[0].timestamp = %#v, want %q", entry["timestamp"], "2026-03-24T12:08:00Z")
	}
	if _, ok := got["allowed_tools"]; ok {
		t.Fatalf("allowed_tools = %#v, want omitted without active or persisted control context", got["allowed_tools"])
	}
}

func TestFormatOperatorStatusSummaryIncludesFailedStepBlockerForFailedRuntime(t *testing.T) {
	t.Parallel()

	formatted, err := FormatOperatorStatusSummary(JobRuntimeState{
		JobID: "job-1",
		State: JobStateFailed,
		FailedSteps: []RuntimeStepRecord{
			{StepID: "build", Reason: "validator failed", At: time.Date(2026, 3, 24, 12, 8, 0, 0, time.UTC)},
		},
	})
	if err != nil {
		t.Fatalf("FormatOperatorStatusSummary() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(formatted), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["state"] != string(JobStateFailed) {
		t.Fatalf("state = %#v, want %q", got["state"], JobStateFailed)
	}
	if got["failed_step_id"] != "build" {
		t.Fatalf("failed_step_id = %#v, want %q", got["failed_step_id"], "build")
	}
	if got["failure_reason"] != "validator failed" {
		t.Fatalf("failure_reason = %#v, want %q", got["failure_reason"], "validator failed")
	}
	if got["failed_at"] != "2026-03-24T12:08:00Z" {
		t.Fatalf("failed_at = %#v, want %q", got["failed_at"], "2026-03-24T12:08:00Z")
	}
}
