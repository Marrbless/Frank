package missioncontrol

import (
	"fmt"
	"strings"
)

type JobState string

const (
	JobSpecVersionV2 = "frank_v2"
	JobSpecVersionV4 = "frank_v4"
)

const (
	ExecutionPlaneLiveRuntime          = "live_runtime"
	ExecutionPlaneImprovementWorkspace = "improvement_workspace"
	ExecutionPlaneHotUpdateGate        = "hot_update_gate"
)

const (
	ExecutionHostPhone          = "phone"
	ExecutionHostDesktop        = "desktop"
	ExecutionHostWorkspace      = "workspace"
	ExecutionHostDesktopDev     = "desktop_dev"
	ExecutionHostRemoteProvider = "remote_provider"
)

const (
	MissionFamilyBuild                        = "build"
	MissionFamilyResearch                     = "research"
	MissionFamilyMonitor                      = "monitor"
	MissionFamilyOperate                      = "operate"
	MissionFamilyMaintenance                  = "maintenance"
	MissionFamilyOutreach                     = "outreach"
	MissionFamilyCommunityDiscovery           = "community_discovery"
	MissionFamilyOpportunityScan              = "opportunity_scan"
	MissionFamilyBootstrapRevenue             = "bootstrap_revenue"
	MissionFamilyBootstrapIdentityAndAccounts = "bootstrap_identity_and_accounts"
	MissionFamilyContinuousAutonomyTick       = "continuous_autonomy_tick"
	MissionFamilyStandingDirectiveReview      = "standing_directive_review"
	MissionFamilyAutonomousMissionProposal    = "autonomous_mission_proposal"
	MissionFamilyAutonomyBudgetReport         = "autonomy_budget_report"
	MissionFamilyAutonomyPause                = "autonomy_pause"
	MissionFamilyAutonomyResume               = "autonomy_resume"
	MissionFamilyImprovePromptpack            = "improve_promptpack"
	MissionFamilyImproveSkills                = "improve_skills"
	MissionFamilyImproveRoutingManifest       = "improve_routing_manifest"
	MissionFamilyImproveRuntimeExtension      = "improve_runtime_extension"
	MissionFamilyEvaluateCandidate            = "evaluate_candidate"
	MissionFamilyPromoteCandidate             = "promote_candidate"
	MissionFamilyRollbackCandidate            = "rollback_candidate"
	MissionFamilyImproveTopology              = "improve_topology"
	MissionFamilyProposeSourcePatch           = "propose_source_patch"
	MissionFamilyPrepareHotUpdate             = "prepare_hot_update"
	MissionFamilyValidateHotUpdate            = "validate_hot_update"
	MissionFamilyStageHotUpdate               = "stage_hot_update"
	MissionFamilyApplyHotUpdate               = "apply_hot_update"
	MissionFamilySmokeTestHotUpdate           = "smoke_test_hot_update"
	MissionFamilyCanaryHotUpdate              = "canary_hot_update"
	MissionFamilyCommitHotUpdate              = "commit_hot_update"
	MissionFamilyRollbackHotUpdate            = "rollback_hot_update"
)

const (
	JobSurfaceClassPromptPack          = "prompt_pack"
	JobSurfaceClassSkill               = "skill"
	JobSurfaceClassManifestEntry       = "manifest_entry"
	JobSurfaceClassSkillTopology       = "skill_topology"
	JobSurfaceClassSourcePatchArtifact = "source_patch_artifact"
)

const (
	JobStatePending   JobState = "pending"
	JobStateRunning   JobState = "running"
	JobStateCompleted JobState = "completed"
	JobStateRejected  JobState = "rejected"
)

type StepType string

const (
	StepTypeDiscussion      StepType = "discussion"
	StepTypeStaticArtifact  StepType = "static_artifact"
	StepTypeOneShotCode     StepType = "one_shot_code"
	StepTypeLongRunningCode StepType = "long_running_code"
	StepTypeSystemAction    StepType = "system_action"
	StepTypeWaitUser        StepType = "wait_user"
	StepTypeFinalResponse   StepType = "final_response"
)

type StepSubtype string

const (
	StepSubtypeBlocker       StepSubtype = "blocker"
	StepSubtypeAuthorization StepSubtype = "authorization"
	StepSubtypeDefinition    StepSubtype = "definition"
)

type AuthorityTier string

const (
	AuthorityTierLow    AuthorityTier = "low"
	AuthorityTierMedium AuthorityTier = "medium"
	AuthorityTierHigh   AuthorityTier = "high"
)

type IdentityMode string

const (
	IdentityModeOwnerOnlyControl IdentityMode = "owner_only_control"
	IdentityModeAgentAlias       IdentityMode = "agent_alias"
)

type FrankRegistryObjectKind string

const (
	FrankRegistryObjectKindIdentity  FrankRegistryObjectKind = "identity"
	FrankRegistryObjectKindAccount   FrankRegistryObjectKind = "account"
	FrankRegistryObjectKindContainer FrankRegistryObjectKind = "container"
)

type FrankRegistryObjectRef struct {
	Kind     FrankRegistryObjectKind `json:"kind"`
	ObjectID string                  `json:"object_id"`
}

type CampaignRef struct {
	CampaignID string `json:"campaign_id"`
}

type TreasuryRef struct {
	TreasuryID string `json:"treasury_id"`
}

type CapabilityOnboardingProposalRef struct {
	ProposalID string `json:"proposal_id"`
}

type JobSurfaceRef struct {
	Class string `json:"class"`
	Ref   string `json:"ref"`
}

type Job struct {
	ID                  string          `json:"id"`
	SpecVersion         string          `json:"spec_version,omitempty"`
	ExecutionPlane      string          `json:"execution_plane,omitempty"`
	ExecutionHost       string          `json:"execution_host,omitempty"`
	MissionFamily       string          `json:"mission_family,omitempty"`
	TargetSurfaces      []JobSurfaceRef `json:"target_surfaces,omitempty"`
	MutableSurfaces     []JobSurfaceRef `json:"mutable_surfaces,omitempty"`
	ImmutableSurfaces   []JobSurfaceRef `json:"immutable_surfaces,omitempty"`
	TopologyModeEnabled bool            `json:"topology_mode_enabled,omitempty"`
	State               JobState        `json:"state"`
	MaxAuthority        AuthorityTier   `json:"max_authority"`
	AllowedTools        []string        `json:"allowed_tools"`
	Plan                Plan            `json:"plan"`
	MissionStoreRoot    string          `json:"-"`
}

type Plan struct {
	ID    string `json:"id"`
	Steps []Step `json:"steps"`
}

type Step struct {
	ID                              string                           `json:"id"`
	Type                            StepType                         `json:"type"`
	Subtype                         StepSubtype                      `json:"subtype,omitempty"`
	ApprovalScope                   string                           `json:"approval_scope,omitempty"`
	DependsOn                       []string                         `json:"depends_on"`
	RequiredAuthority               AuthorityTier                    `json:"required_authority"`
	AllowedTools                    []string                         `json:"allowed_tools"`
	RequiresApproval                bool                             `json:"requires_approval"`
	SuccessCriteria                 []string                         `json:"success_criteria"`
	StaticArtifactPath              string                           `json:"static_artifact_path,omitempty"`
	StaticArtifactFormat            string                           `json:"static_artifact_format,omitempty"`
	OneShotArtifactPath             string                           `json:"one_shot_artifact_path,omitempty"`
	LongRunningStartupCommand       []string                         `json:"long_running_startup_command,omitempty"`
	LongRunningArtifactPath         string                           `json:"long_running_artifact_path,omitempty"`
	IdentityMode                    IdentityMode                     `json:"identity_mode,omitempty"`
	RequiredCapabilities            []string                         `json:"required_capabilities,omitempty"`
	RequiredDataDomains             []string                         `json:"required_data_domains,omitempty"`
	GovernedExternalTargets         []AutonomyEligibilityTargetRef   `json:"governed_external_targets,omitempty"`
	FrankObjectRefs                 []FrankRegistryObjectRef         `json:"frank_object_refs,omitempty"`
	CampaignRef                     *CampaignRef                     `json:"campaign_ref,omitempty"`
	TreasuryRef                     *TreasuryRef                     `json:"treasury_ref,omitempty"`
	CapabilityOnboardingProposalRef *CapabilityOnboardingProposalRef `json:"capability_onboarding_proposal_ref,omitempty"`
	SystemAction                    *SystemAction                    `json:"system_action,omitempty"`
}

func NormalizeIdentityMode(mode IdentityMode) IdentityMode {
	normalized := IdentityMode(strings.TrimSpace(string(mode)))
	if normalized == "" {
		return IdentityModeAgentAlias
	}
	return normalized
}

func validateIdentityMode(mode IdentityMode) error {
	switch normalized := NormalizeIdentityMode(mode); normalized {
	case IdentityModeOwnerOnlyControl, IdentityModeAgentAlias:
		return nil
	default:
		return fmt.Errorf("identity_mode %q is invalid", strings.TrimSpace(string(mode)))
	}
}

func NormalizeFrankRegistryObjectKind(kind FrankRegistryObjectKind) FrankRegistryObjectKind {
	return FrankRegistryObjectKind(strings.TrimSpace(string(kind)))
}

func NormalizeFrankRegistryObjectRef(ref FrankRegistryObjectRef) FrankRegistryObjectRef {
	ref.Kind = NormalizeFrankRegistryObjectKind(ref.Kind)
	ref.ObjectID = strings.TrimSpace(ref.ObjectID)
	return ref
}

func NormalizeCampaignRef(ref CampaignRef) CampaignRef {
	ref.CampaignID = strings.TrimSpace(ref.CampaignID)
	return ref
}

func NormalizeTreasuryRef(ref TreasuryRef) TreasuryRef {
	ref.TreasuryID = strings.TrimSpace(ref.TreasuryID)
	return ref
}

func NormalizeCapabilityOnboardingProposalRef(ref CapabilityOnboardingProposalRef) CapabilityOnboardingProposalRef {
	ref.ProposalID = strings.TrimSpace(ref.ProposalID)
	return ref
}

func NormalizeJobSurfaceRef(ref JobSurfaceRef) JobSurfaceRef {
	ref.Class = strings.TrimSpace(ref.Class)
	ref.Ref = strings.TrimSpace(ref.Ref)
	return ref
}

func NormalizeStepRequiredCapabilities(values []string) []string {
	return normalizeStepDeclarationStrings(values)
}

func NormalizeStepRequiredDataDomains(values []string) []string {
	return normalizeStepDeclarationStrings(values)
}

func normalizeStepDeclarationStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

type ValidationError struct {
	Code    RejectionCode `json:"code"`
	StepID  string        `json:"step_id"`
	Message string        `json:"message"`
}

func (e ValidationError) Error() string {
	switch {
	case e.Code != "" && e.StepID != "" && e.Message != "":
		return fmt.Sprintf("%s for step %s: %s", e.Code, e.StepID, e.Message)
	case e.StepID != "" && e.Message != "":
		return fmt.Sprintf("step %s: %s", e.StepID, e.Message)
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	case e.Message != "":
		return e.Message
	case e.Code != "" && e.StepID != "":
		return fmt.Sprintf("%s for step %s", e.Code, e.StepID)
	case e.Code != "":
		return string(e.Code)
	case e.StepID != "":
		return fmt.Sprintf("validation error for step %s", e.StepID)
	default:
		return "validation error"
	}
}
