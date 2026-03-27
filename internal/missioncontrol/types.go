package missioncontrol

import "fmt"

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
	ID                        string        `json:"id"`
	Type                      StepType      `json:"type"`
	Subtype                   StepSubtype   `json:"subtype,omitempty"`
	ApprovalScope             string        `json:"approval_scope,omitempty"`
	DependsOn                 []string      `json:"depends_on"`
	RequiredAuthority         AuthorityTier `json:"required_authority"`
	AllowedTools              []string      `json:"allowed_tools"`
	RequiresApproval          bool          `json:"requires_approval"`
	SuccessCriteria           []string      `json:"success_criteria"`
	StaticArtifactPath        string        `json:"static_artifact_path,omitempty"`
	StaticArtifactFormat      string        `json:"static_artifact_format,omitempty"`
	LongRunningStartupCommand []string      `json:"long_running_startup_command,omitempty"`
	LongRunningArtifactPath   string        `json:"long_running_artifact_path,omitempty"`
	SystemAction              *SystemAction `json:"system_action,omitempty"`
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
