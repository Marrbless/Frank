package missioncontrol

import (
	"fmt"
	"strings"
)

type JobState string

const JobSpecVersionV2 = "frank_v2"

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

type Job struct {
	ID           string        `json:"id"`
	SpecVersion  string        `json:"spec_version,omitempty"`
	State        JobState      `json:"state"`
	MaxAuthority AuthorityTier `json:"max_authority"`
	AllowedTools []string      `json:"allowed_tools"`
	Plan         Plan          `json:"plan"`
}

type Plan struct {
	ID    string `json:"id"`
	Steps []Step `json:"steps"`
}

type Step struct {
	ID                        string                         `json:"id"`
	Type                      StepType                       `json:"type"`
	Subtype                   StepSubtype                    `json:"subtype,omitempty"`
	ApprovalScope             string                         `json:"approval_scope,omitempty"`
	DependsOn                 []string                       `json:"depends_on"`
	RequiredAuthority         AuthorityTier                  `json:"required_authority"`
	AllowedTools              []string                       `json:"allowed_tools"`
	RequiresApproval          bool                           `json:"requires_approval"`
	SuccessCriteria           []string                       `json:"success_criteria"`
	StaticArtifactPath        string                         `json:"static_artifact_path,omitempty"`
	StaticArtifactFormat      string                         `json:"static_artifact_format,omitempty"`
	OneShotArtifactPath       string                         `json:"one_shot_artifact_path,omitempty"`
	LongRunningStartupCommand []string                       `json:"long_running_startup_command,omitempty"`
	LongRunningArtifactPath   string                         `json:"long_running_artifact_path,omitempty"`
	IdentityMode              IdentityMode                   `json:"identity_mode,omitempty"`
	GovernedExternalTargets   []AutonomyEligibilityTargetRef `json:"governed_external_targets,omitempty"`
	FrankObjectRefs           []FrankRegistryObjectRef       `json:"frank_object_refs,omitempty"`
	CampaignRef               *CampaignRef                   `json:"campaign_ref,omitempty"`
	TreasuryRef               *TreasuryRef                   `json:"treasury_ref,omitempty"`
	SystemAction              *SystemAction                  `json:"system_action,omitempty"`
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
