package missioncontrol

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExecutionContextRoundTrip(t *testing.T) {
	t.Parallel()

	job := &Job{ID: "job-1"}
	step := &Step{ID: "step-1"}

	ctx := WithExecutionContext(context.Background(), ExecutionContext{
		Job:  job,
		Step: step,
	})

	got, ok := ExecutionContextFromContext(ctx)
	if !ok {
		t.Fatal("ExecutionContextFromContext() ok = false, want true")
	}

	if got.Job != job {
		t.Fatalf("ExecutionContextFromContext().Job = %p, want %p", got.Job, job)
	}

	if got.Step != step {
		t.Fatalf("ExecutionContextFromContext().Step = %p, want %p", got.Step, step)
	}
}

func TestExecutionContextFromContextMissing(t *testing.T) {
	t.Parallel()

	if _, ok := ExecutionContextFromContext(context.Background()); ok {
		t.Fatal("ExecutionContextFromContext() ok = true, want false")
	}
}

func TestDefaultToolGuardAllow(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), testExecutionContext(), "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}

	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}

	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardNoExternalTargetsPreservesBehavior(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.MissionStoreRoot = ""
	ec.GovernedExternalTargets = nil

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardNoExternalTargetsPreservesBehaviorWhenIdentityModeOmitted(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	if ec.Step.IdentityMode != IdentityModeAgentAlias {
		t.Fatalf("ResolveExecutionContext().Step.IdentityMode = %q, want %q", ec.Step.IdentityMode, IdentityModeAgentAlias)
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardCampaignZeroRefPathUnchanged(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	calls := 0

	decision := defaultToolGuard{
		campaignReadinessGuard: func(ExecutionContext) error {
			calls++
			return nil
		},
	}.EvaluateTool(context.Background(), ec, "read", nil)

	if calls != 0 {
		t.Fatalf("campaignReadinessGuard calls = %d, want 0 for zero-campaign-ref path", calls)
	}
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardCampaignDeclaredStepCallsReadinessOnce(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.CampaignRef = &CampaignRef{CampaignID: "campaign-1"}

	calls := 0
	var gotEC ExecutionContext
	decision := defaultToolGuard{
		campaignReadinessGuard: func(ec ExecutionContext) error {
			calls++
			gotEC = CloneExecutionContext(ec)
			return nil
		},
	}.EvaluateTool(context.Background(), ec, "read", nil)

	if calls != 1 {
		t.Fatalf("campaignReadinessGuard calls = %d, want 1", calls)
	}
	if gotEC.Step == nil || gotEC.Step.CampaignRef == nil {
		t.Fatalf("campaignReadinessGuard execution context = %#v, want campaign-aware step", gotEC)
	}
	if gotEC.Step.CampaignRef.CampaignID != "campaign-1" {
		t.Fatalf("campaignReadinessGuard campaign_id = %q, want %q", gotEC.Step.CampaignRef.CampaignID, "campaign-1")
	}
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestRequireExecutionContextCampaignReadinessZeroRefPathPreservesBehavior(t *testing.T) {
	t.Parallel()

	ec, err := ResolveExecutionContext(testExecutionJob(), "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	if err := RequireExecutionContextCampaignReadiness(ec); err != nil {
		t.Fatalf("RequireExecutionContextCampaignReadiness() error = %v", err)
	}
}

func TestRequireExecutionContextCampaignReadinessActiveCampaignPasses(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	root := fixtures.root
	now := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	record := validCampaignRecord(now, func(record *CampaignRecord) {
		record.CampaignID = "campaign-ready"
		record.State = CampaignStateActive
		record.FrankObjectRefs = []FrankRegistryObjectRef{
			{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			{Kind: FrankRegistryObjectKindAccount, ObjectID: fixtures.account.AccountID},
			{Kind: FrankRegistryObjectKindContainer, ObjectID: fixtures.container.ContainerID},
		}
	})
	if err := StoreCampaignRecord(root, record); err != nil {
		t.Fatalf("StoreCampaignRecord() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: record.CampaignID}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	if err := RequireExecutionContextCampaignReadiness(ec); err != nil {
		t.Fatalf("RequireExecutionContextCampaignReadiness() error = %v", err)
	}
}

func TestRequireExecutionContextCampaignReadinessFailsClosedForDisallowedOrBrokenCampaignState(t *testing.T) {
	t.Parallel()

	t.Run("draft campaign disallowed", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		now := time.Date(2026, 4, 11, 18, 15, 0, 0, time.UTC)
		record := validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-draft"
			record.State = CampaignStateDraft
			record.FrankObjectRefs = []FrankRegistryObjectRef{
				{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			}
		})
		if err := StoreCampaignRecord(root, record); err != nil {
			t.Fatalf("StoreCampaignRecord() error = %v", err)
		}

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: record.CampaignID}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = root

		err = RequireExecutionContextCampaignReadiness(ec)
		if err == nil {
			t.Fatal("RequireExecutionContextCampaignReadiness() error = nil, want draft-state rejection")
		}
		if !strings.Contains(err.Error(), `campaign readiness requires state "active"; got "draft"`) {
			t.Fatalf("RequireExecutionContextCampaignReadiness() error = %q, want draft-state rejection", err.Error())
		}
	})

	t.Run("ineligible governed target", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		now := time.Date(2026, 4, 11, 18, 20, 0, 0, time.UTC)
		target := AutonomyEligibilityTargetRef{
			Kind:       EligibilityTargetKindProvider,
			RegistryID: "provider-human-id",
		}
		writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "provider-human-id", "check-provider-human-id", now)
		record := validCampaignRecord(now.Add(time.Minute), func(record *CampaignRecord) {
			record.CampaignID = "campaign-ineligible"
			record.State = CampaignStateActive
			record.GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}
			record.FrankObjectRefs = []FrankRegistryObjectRef{
				{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			}
		})
		if err := StoreCampaignRecord(root, record); err != nil {
			t.Fatalf("StoreCampaignRecord() error = %v", err)
		}

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: record.CampaignID}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = root

		_, wantErr := RequireAutonomyEligibleTarget(root, target)
		if wantErr == nil {
			t.Fatal("RequireAutonomyEligibleTarget() error = nil, want ineligible target rejection")
		}

		err = RequireExecutionContextCampaignReadiness(ec)
		if err == nil {
			t.Fatal("RequireExecutionContextCampaignReadiness() error = nil, want ineligible target rejection")
		}
		if err.Error() != wantErr.Error() {
			t.Fatalf("RequireExecutionContextCampaignReadiness() error = %q, want %q", err.Error(), wantErr.Error())
		}
	})
}

func TestDefaultToolGuardCampaignReadinessFailsClosedForDisallowedOrMalformedCampaignState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)

	t.Run("missing campaign", func(t *testing.T) {
		t.Parallel()

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: "campaign-missing"}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = t.TempDir()

		decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

		assertDenied(t, decision, RejectionCodeInvalidRuntimeState, ErrCampaignRecordNotFound.Error())
	})

	t.Run("malformed campaign record", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		writeMalformedCampaignRecordForPreflightTest(t, root, validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-malformed"
			record.FrankObjectRefs = []FrankRegistryObjectRef{{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID}}
			record.ComplianceChecks = nil
		}))

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: "campaign-malformed"}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = root

		decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

		if decision.Allowed {
			t.Fatalf("EvaluateTool().Allowed = true, want false: %#v", decision)
		}
		if decision.Code != RejectionCodeInvalidRuntimeState {
			t.Fatalf("EvaluateTool().Code = %q, want %q", decision.Code, RejectionCodeInvalidRuntimeState)
		}
		if !strings.Contains(decision.Reason, "mission store campaign compliance_checks are required") {
			t.Fatalf("EvaluateTool().Reason = %q, want malformed campaign rejection", decision.Reason)
		}
	})

	t.Run("broken linked object", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		writeMalformedCampaignRecordForPreflightTest(t, root, validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-broken-link"
			record.FrankObjectRefs = []FrankRegistryObjectRef{{Kind: FrankRegistryObjectKindIdentity, ObjectID: "identity-missing"}}
		}))

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: "campaign-broken-link"}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = root

		decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

		if decision.Allowed {
			t.Fatalf("EvaluateTool().Allowed = true, want false: %#v", decision)
		}
		if decision.Code != RejectionCodeInvalidRuntimeState {
			t.Fatalf("EvaluateTool().Code = %q, want %q", decision.Code, RejectionCodeInvalidRuntimeState)
		}
		if !strings.Contains(decision.Reason, ErrFrankIdentityRecordNotFound.Error()) {
			t.Fatalf("EvaluateTool().Reason = %q, want missing identity rejection", decision.Reason)
		}
	})

	t.Run("draft campaign disallowed", func(t *testing.T) {
		t.Parallel()

		fixtures := writeExecutionContextFrankRegistryFixtures(t)
		root := fixtures.root
		record := validCampaignRecord(now, func(record *CampaignRecord) {
			record.CampaignID = "campaign-draft"
			record.State = CampaignStateDraft
			record.FrankObjectRefs = []FrankRegistryObjectRef{
				{Kind: FrankRegistryObjectKindIdentity, ObjectID: fixtures.identity.IdentityID},
			}
		})
		if err := StoreCampaignRecord(root, record); err != nil {
			t.Fatalf("StoreCampaignRecord() error = %v", err)
		}

		job := testExecutionJob()
		job.Plan.Steps[0].CampaignRef = &CampaignRef{CampaignID: record.CampaignID}
		ec, err := ResolveExecutionContext(job, "build")
		if err != nil {
			t.Fatalf("ResolveExecutionContext() error = %v", err)
		}
		ec.MissionStoreRoot = root

		decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

		assertDenied(t, decision, RejectionCodeInvalidRuntimeState, `campaign readiness requires state "active"; got "draft"`)
	})
}

func TestDefaultToolGuardTreasuryPreflightZeroRefPathPreservesBehavior(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.MissionStoreRoot = ""

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardFrankObjectRefsDoNotCreateEligibilityOrIdentityModeSideChannel(t *testing.T) {
	t.Parallel()

	job := testExecutionJob()
	job.Plan.Steps[0].FrankObjectRefs = []FrankRegistryObjectRef{
		{
			Kind:     FrankRegistryObjectKindIdentity,
			ObjectID: "identity-1",
		},
	}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardApprovalRequired(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.RequiresApproval = true

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeApprovalRequired, "step requires approval")
}

func TestDefaultToolGuardAuthorityExceeded(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.MaxAuthority = AuthorityTierLow
	ec.Step.RequiredAuthority = AuthorityTierHigh

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeAuthorityExceeded, "step required authority exceeds job max authority")
}

func TestDefaultToolGuardDeniedByJobToolScope(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.AllowedTools = []string{"write"}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "tool is not allowed by job tool scope")
}

func TestDefaultToolGuardDeniedByStepToolScope(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Step.AllowedTools = []string{"write"}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "tool is not allowed by step tool scope")
}

func TestDefaultToolGuardLeastAuthorityProfileDeniesHighRiskToolsOutsideScope(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.MaxAuthority = AuthorityTierLow
	ec.Job.AllowedTools = []string{"message", "read_memory", "list_memory", "web_search"}
	ec.Step.RequiredAuthority = AuthorityTierLow
	ec.Step.AllowedTools = []string{"message", "read_memory", "list_memory", "web_search"}

	for _, toolName := range []string{
		"exec",
		"filesystem",
		"frank_zoho_send_email",
		"frank_zoho_manage_reply_work_item",
		"write_memory",
		"edit_memory",
		"delete_memory",
		"create_skill",
		"delete_skill",
		"mcp_mail_send",
	} {
		t.Run(toolName, func(t *testing.T) {
			t.Parallel()

			decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, toolName, nil)

			assertDenied(t, decision, RejectionCodeToolNotAllowed, "tool is not allowed by job tool scope")
		})
	}
}

func TestDefaultToolGuardLeastAuthorityProfileApprovalGateDeniesScopedHighRiskTool(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Job.MaxAuthority = AuthorityTierLow
	ec.Job.AllowedTools = []string{"exec"}
	ec.Step.RequiredAuthority = AuthorityTierLow
	ec.Step.AllowedTools = []string{"exec"}
	ec.Step.RequiresApproval = true

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "exec", map[string]interface{}{
		"cmd": []string{"go", "test"},
	})

	assertDenied(t, decision, RejectionCodeApprovalRequired, "step requires approval")
}

func TestDefaultToolGuardMissingJobOrStepContext(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ExecutionContext{}, "read", nil)

	assertDenied(t, decision, RejectionCodeToolNotAllowed, "missing job or step context")
}

func TestDefaultToolGuardWaitingUserDenied(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()
	ec.Runtime = &JobRuntimeState{
		JobID:        ec.Job.ID,
		State:        JobStateWaitingUser,
		ActiveStepID: ec.Step.ID,
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeWaitingUser, "job is waiting for user input")
}

func TestDefaultToolGuardEventFieldsPopulated(t *testing.T) {
	t.Parallel()

	ec := testExecutionContext()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	if decision.Event.JobID != ec.Job.ID {
		t.Fatalf("Event.JobID = %q, want %q", decision.Event.JobID, ec.Job.ID)
	}

	if decision.Event.StepID != ec.Step.ID {
		t.Fatalf("Event.StepID = %q, want %q", decision.Event.StepID, ec.Step.ID)
	}

	if decision.Event.ToolName != "read" {
		t.Fatalf("Event.ToolName = %q, want %q", decision.Event.ToolName, "read")
	}

	if !decision.Event.Allowed {
		t.Fatalf("Event.Allowed = false, want true")
	}

	if decision.Event.Code != decision.Code {
		t.Fatalf("Event.Code = %q, want %q", decision.Event.Code, decision.Code)
	}

	if decision.Event.Reason != decision.Reason {
		t.Fatalf("Event.Reason = %q, want %q", decision.Event.Reason, decision.Reason)
	}
}

func TestDefaultToolGuardEventTimestampNonZero(t *testing.T) {
	t.Parallel()

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), testExecutionContext(), "read", nil)

	if decision.Event.Timestamp.IsZero() {
		t.Fatal("Event.Timestamp is zero")
	}

	if decision.Event.Timestamp.After(time.Now().Add(time.Second)) {
		t.Fatalf("Event.Timestamp = %v, looks invalid", decision.Event.Timestamp)
	}
}

func TestDefaultToolGuardSystemActionAuditsCanonicalExecuteAction(t *testing.T) {
	t.Parallel()

	ec := ExecutionContext{
		Job: &Job{
			ID:           "job-1",
			MaxAuthority: AuthorityTierMedium,
			AllowedTools: []string{"exec"},
		},
		Step: &Step{
			ID:           "start-service",
			Type:         StepTypeSystemAction,
			AllowedTools: []string{"exec"},
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStart,
				Target:    "demo-service",
				Command:   []string{"democtl", "start", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "start-service",
		},
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "exec", map[string]interface{}{
		"cmd": []string{"democtl", "start", "demo-service"},
	})

	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Event.ToolName != "system_action:execute:start:service:demo-service" {
		t.Fatalf("Event.ToolName = %q, want canonical system_action execute audit string", decision.Event.ToolName)
	}
}

func TestDefaultToolGuardSystemActionApprovalRejectionAuditsCanonicalVerificationAction(t *testing.T) {
	t.Parallel()

	ec := ExecutionContext{
		Job: &Job{
			ID:           "job-1",
			MaxAuthority: AuthorityTierMedium,
			AllowedTools: []string{"exec"},
		},
		Step: &Step{
			ID:               "start-service",
			Type:             StepTypeSystemAction,
			AllowedTools:     []string{"exec"},
			RequiresApproval: true,
			SystemAction: &SystemAction{
				Kind:      SystemActionKindService,
				Operation: SystemActionOperationStart,
				Target:    "demo-service",
				Command:   []string{"democtl", "start", "demo-service"},
				PostState: &SystemActionPostState{
					Command:         []string{"democtl", "status", "demo-service"},
					SuccessContains: []string{"state=running"},
				},
			},
		},
		Runtime: &JobRuntimeState{
			JobID:        "job-1",
			State:        JobStateRunning,
			ActiveStepID: "start-service",
		},
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "exec", map[string]interface{}{
		"cmd": []string{"democtl", "status", "demo-service"},
	})

	assertDenied(t, decision, RejectionCodeApprovalRequired, "step requires approval")
	if decision.Event.ToolName != "system_action:verify_post_state:start:service:demo-service" {
		t.Fatalf("Event.ToolName = %q, want canonical system_action verification audit string", decision.Event.ToolName)
	}
}

func TestDefaultToolGuardEligibleExternalTargetPasses(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "mail.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-mail",
		TargetKind:                  target.Kind,
		TargetName:                  "mail.example",
		CanCreateWithoutOwner:       true,
		CanOnboardWithoutOwner:      true,
		CanControlAsAgent:           true,
		CanRecoverAsAgent:           true,
		RequiresHumanOnlyStep:       false,
		RequiresOwnerOnlySecretOrID: false,
		RulesAsObservedOK:           true,
		Label:                       EligibilityLabelAutonomyCompatible,
		Reasons:                     []string{"operator-reviewed"},
		CheckedAt:                   now,
	})

	ec := testExecutionContext()
	ec.MissionStoreRoot = root
	ec.GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardEligibleTreasuryContainerPasses(t *testing.T) {
	t.Parallel()

	fixtures := writeExecutionContextFrankRegistryFixtures(t)
	now := time.Date(2026, 4, 8, 21, 0, 0, 0, time.UTC)
	record := validTreasuryRecord(now, func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-guard-eligible"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: fixtures.container.ContainerID,
			},
		}
	})
	if err := StoreTreasuryRecord(fixtures.root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].IdentityMode = IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: record.TreasuryID}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = fixtures.root

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)
	if !decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = false, want true: %#v", decision)
	}
	if decision.Code != "" {
		t.Fatalf("EvaluateTool().Code = %q, want empty", decision.Code)
	}
	if decision.Reason != "" {
		t.Fatalf("EvaluateTool().Reason = %q, want empty", decision.Reason)
	}
}

func TestDefaultToolGuardIneligibleTreasuryContainerFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 8, 21, 15, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindTreasuryContainerClass,
		RegistryID: "container-class-human-wallet",
	}
	writeFrankRegistryEligibilityFixture(t, root, target, EligibilityLabelIneligible, "container-class-human-wallet", "check-container-class-human-wallet", now)

	container := FrankContainerRecord{
		RecordVersion:        StoreRecordVersion,
		ContainerID:          "container-human-wallet",
		ContainerKind:        "wallet",
		Label:                "Human Wallet",
		ContainerClassID:     "container-class-human-wallet",
		State:                "candidate",
		EligibilityTargetRef: target,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.Add(time.Minute).UTC(),
	}
	if err := StoreFrankContainerRecord(root, container); err != nil {
		t.Fatalf("StoreFrankContainerRecord() error = %v", err)
	}

	record := validTreasuryRecord(now.Add(2*time.Minute), func(record *TreasuryRecord) {
		record.TreasuryID = "treasury-guard-ineligible"
		record.ContainerRefs = []FrankRegistryObjectRef{
			{
				Kind:     FrankRegistryObjectKindContainer,
				ObjectID: container.ContainerID,
			},
		}
	})
	if err := StoreTreasuryRecord(root, record); err != nil {
		t.Fatalf("StoreTreasuryRecord() error = %v", err)
	}

	job := testExecutionJob()
	job.Plan.Steps[0].TreasuryRef = &TreasuryRef{TreasuryID: record.TreasuryID}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	_, wantErr := RequireAutonomyEligibleTarget(root, target)
	if !errors.Is(wantErr, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", wantErr, ErrAutonomyEligibleTargetRequired)
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, wantErr.Error())
}

func TestDefaultToolGuardUnknownExternalTargetFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "missing-provider",
	}

	ec := testExecutionContext()
	ec.MissionStoreRoot = root
	ec.GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, `autonomy eligibility target "missing-provider" has no autonomy-compatible registry record`)
}

func TestDefaultToolGuardIneligibleExternalTargetFailsClosedWithCanonicalReason(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 12, 30, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "human-id.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelIneligible,
		LastCheckID:      "check-provider-human-id",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-human-id",
		TargetKind:                  target.Kind,
		TargetName:                  "human-id.example",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       EligibilityLabelIneligible,
		Reasons:                     []string{string(AutonomyEligibilityReasonOwnerIdentityRequired)},
		CheckedAt:                   now,
	})

	wantResult, wantErr := RequireAutonomyEligibleTarget(root, target)
	if !errors.Is(wantErr, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", wantErr, ErrAutonomyEligibleTargetRequired)
	}
	if wantResult.Decision != AutonomyEligibilityDecisionIneligible {
		t.Fatalf("RequireAutonomyEligibleTarget().Decision = %q, want %q", wantResult.Decision, AutonomyEligibilityDecisionIneligible)
	}

	ec := testExecutionContext()
	ec.MissionStoreRoot = root
	ec.GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, wantErr.Error())
}

func TestDefaultToolGuardMultipleExternalTargetsRequireAllEligible(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 13, 0, 0, 0, time.UTC)
	eligible := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	writeAutonomyEligibilityFixture(t, root, eligible, PlatformRecord{
		PlatformID:       eligible.RegistryID,
		PlatformName:     "mail.example",
		TargetClass:      eligible.Kind,
		EligibilityLabel: EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-mail",
		TargetKind:                  eligible.Kind,
		TargetName:                  "mail.example",
		CanCreateWithoutOwner:       true,
		CanOnboardWithoutOwner:      true,
		CanControlAsAgent:           true,
		CanRecoverAsAgent:           true,
		RequiresHumanOnlyStep:       false,
		RequiresOwnerOnlySecretOrID: false,
		RulesAsObservedOK:           true,
		Label:                       EligibilityLabelAutonomyCompatible,
		Reasons:                     []string{"operator-reviewed"},
		CheckedAt:                   now,
	})

	unknown := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-unknown",
	}

	ec := testExecutionContext()
	ec.MissionStoreRoot = root
	ec.GovernedExternalTargets = []AutonomyEligibilityTargetRef{eligible, unknown}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, `autonomy eligibility target "provider-unknown" has no autonomy-compatible registry record`)
}

func TestDefaultToolGuardMalformedOrConflictingRegistryFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, root string) AutonomyEligibilityTargetRef
	}{
		{
			name: "conflicting labels",
			setup: func(t *testing.T, root string) AutonomyEligibilityTargetRef {
				t.Helper()

				now := time.Date(2026, 4, 7, 14, 0, 0, 0, time.UTC)
				target := AutonomyEligibilityTargetRef{
					Kind:       EligibilityTargetKindProvider,
					RegistryID: "provider-conflict",
				}
				if err := StorePlatformRecord(root, PlatformRecord{
					PlatformID:       target.RegistryID,
					PlatformName:     "conflict.example",
					TargetClass:      target.Kind,
					EligibilityLabel: EligibilityLabelAutonomyCompatible,
					LastCheckID:      "check-conflict",
					Notes:            []string{"registry note"},
					UpdatedAt:        now,
				}); err != nil {
					t.Fatalf("StorePlatformRecord() error = %v", err)
				}
				if err := StoreEligibilityCheckRecord(root, EligibilityCheckRecord{
					CheckID:                     "check-conflict",
					TargetKind:                  target.Kind,
					TargetName:                  "conflict.example",
					CanCreateWithoutOwner:       false,
					CanOnboardWithoutOwner:      false,
					CanControlAsAgent:           false,
					CanRecoverAsAgent:           false,
					RequiresHumanOnlyStep:       true,
					RequiresOwnerOnlySecretOrID: false,
					RulesAsObservedOK:           false,
					Label:                       EligibilityLabelIneligible,
					Reasons:                     []string{string(AutonomyEligibilityReasonManualHumanCompletionRequired)},
					CheckedAt:                   now,
				}); err != nil {
					t.Fatalf("StoreEligibilityCheckRecord() error = %v", err)
				}
				return target
			},
		},
		{
			name: "malformed reason code",
			setup: func(t *testing.T, root string) AutonomyEligibilityTargetRef {
				t.Helper()

				now := time.Date(2026, 4, 7, 14, 5, 0, 0, time.UTC)
				target := AutonomyEligibilityTargetRef{
					Kind:       EligibilityTargetKindPlatform,
					RegistryID: "platform-bad-reason",
				}
				if err := StorePlatformRecord(root, PlatformRecord{
					PlatformID:       target.RegistryID,
					PlatformName:     "bad-reason.example",
					TargetClass:      target.Kind,
					EligibilityLabel: EligibilityLabelIneligible,
					LastCheckID:      "check-bad-reason",
					Notes:            []string{"registry note"},
					UpdatedAt:        now,
				}); err != nil {
					t.Fatalf("StorePlatformRecord() error = %v", err)
				}
				if err := StoreEligibilityCheckRecord(root, EligibilityCheckRecord{
					CheckID:                     "check-bad-reason",
					TargetKind:                  target.Kind,
					TargetName:                  "bad-reason.example",
					CanCreateWithoutOwner:       false,
					CanOnboardWithoutOwner:      false,
					CanControlAsAgent:           false,
					CanRecoverAsAgent:           false,
					RequiresHumanOnlyStep:       true,
					RequiresOwnerOnlySecretOrID: false,
					RulesAsObservedOK:           false,
					Label:                       EligibilityLabelIneligible,
					Reasons:                     []string{"unsupported_reason_code"},
					CheckedAt:                   now,
				}); err != nil {
					t.Fatalf("StoreEligibilityCheckRecord() error = %v", err)
				}
				return target
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			target := tc.setup(t, root)
			_, wantErr := RequireAutonomyEligibleTarget(root, target)
			if wantErr == nil {
				t.Fatal("RequireAutonomyEligibleTarget() error = nil, want fail-closed registry error")
			}

			ec := testExecutionContext()
			ec.MissionStoreRoot = root
			ec.GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}

			decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

			assertDenied(t, decision, RejectionCodeInvalidRuntimeState, wantErr.Error())
		})
	}
}

func TestDefaultToolGuardDeclaredStepTargetsDelegateThroughAutonomyHelper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 14, 10, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-human-id",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "human-id.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelIneligible,
		LastCheckID:      "check-provider-human-id",
		Notes:            []string{"registry note"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                     "check-provider-human-id",
		TargetKind:                  target.Kind,
		TargetName:                  "human-id.example",
		CanCreateWithoutOwner:       false,
		CanOnboardWithoutOwner:      false,
		CanControlAsAgent:           false,
		CanRecoverAsAgent:           false,
		RequiresHumanOnlyStep:       true,
		RequiresOwnerOnlySecretOrID: true,
		RulesAsObservedOK:           false,
		Label:                       EligibilityLabelIneligible,
		Reasons:                     []string{string(AutonomyEligibilityReasonOwnerIdentityRequired)},
		CheckedAt:                   now,
	})

	job := testExecutionJob()
	job.Plan.Steps[0].IdentityMode = IdentityModeAgentAlias
	job.Plan.Steps[0].GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	_, wantErr := RequireAutonomyEligibleTarget(root, target)
	if !errors.Is(wantErr, ErrAutonomyEligibleTargetRequired) {
		t.Fatalf("RequireAutonomyEligibleTarget() error = %v, want %v", wantErr, ErrAutonomyEligibleTargetRequired)
	}

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, wantErr.Error())
}

func TestDefaultToolGuardRejectsOwnerOnlyControlForGovernedExternalTargets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 7, 14, 15, 0, 0, time.UTC)
	target := AutonomyEligibilityTargetRef{
		Kind:       EligibilityTargetKindProvider,
		RegistryID: "provider-mail",
	}
	writeAutonomyEligibilityFixture(t, root, target, PlatformRecord{
		PlatformID:       target.RegistryID,
		PlatformName:     "mail.example",
		TargetClass:      target.Kind,
		EligibilityLabel: EligibilityLabelAutonomyCompatible,
		LastCheckID:      "check-provider-mail",
		Notes:            []string{"eligible fixture"},
		UpdatedAt:        now,
	}, EligibilityCheckRecord{
		CheckID:                "check-provider-mail",
		TargetKind:             target.Kind,
		TargetName:             "mail.example",
		CanCreateWithoutOwner:  true,
		CanOnboardWithoutOwner: true,
		CanControlAsAgent:      true,
		CanRecoverAsAgent:      true,
		RulesAsObservedOK:      true,
		Label:                  EligibilityLabelAutonomyCompatible,
		Reasons:                []string{"autonomy_compatible"},
		CheckedAt:              now,
	})

	job := testExecutionJob()
	job.Plan.Steps[0].IdentityMode = IdentityModeOwnerOnlyControl
	job.Plan.Steps[0].GovernedExternalTargets = []AutonomyEligibilityTargetRef{target}
	ec, err := ResolveExecutionContext(job, "build")
	if err != nil {
		t.Fatalf("ResolveExecutionContext() error = %v", err)
	}
	ec.MissionStoreRoot = root

	decision := NewDefaultToolGuard().EvaluateTool(context.Background(), ec, "read", nil)

	assertDenied(t, decision, RejectionCodeInvalidRuntimeState, `governed external target execution requires identity_mode "agent_alias"; got "owner_only_control"`)
}

func TestAuditEventJSONUsesRequiredFieldNames(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(AuditEvent{
		JobID:     "job-1",
		StepID:    "step-1",
		ToolName:  "read",
		Allowed:   false,
		Code:      RejectionCodeToolNotAllowed,
		Timestamp: time.Unix(789, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	got := string(payload)
	for _, want := range []string{`"job_id":"job-1"`, `"step_id":"step-1"`, `"proposed_action":"read"`, `"allowed":false`, `"error_code":"tool_not_allowed"`, `"timestamp":"1970-01-01T00:13:09Z"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("audit JSON %s missing %s", got, want)
		}
	}
}

func assertDenied(t *testing.T, decision GuardDecision, wantCode RejectionCode, wantReason string) {
	t.Helper()

	if decision.Allowed {
		t.Fatalf("EvaluateTool().Allowed = true, want false: %#v", decision)
	}

	if decision.Code != wantCode {
		t.Fatalf("EvaluateTool().Code = %q, want %q", decision.Code, wantCode)
	}

	if decision.Reason != wantReason {
		t.Fatalf("EvaluateTool().Reason = %q, want %q", decision.Reason, wantReason)
	}
}

func testExecutionContext() ExecutionContext {
	job := &Job{
		ID:           "job-1",
		MaxAuthority: AuthorityTierMedium,
		AllowedTools: []string{"read", "write"},
	}
	step := &Step{
		ID:                "step-1",
		RequiredAuthority: AuthorityTierLow,
		AllowedTools:      []string{"read"},
	}
	return ExecutionContext{
		Job:  job,
		Step: step,
		Runtime: &JobRuntimeState{
			JobID:        job.ID,
			State:        JobStateRunning,
			ActiveStepID: step.ID,
		},
	}
}
