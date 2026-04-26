package missioncontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const OperatorStatusRecentAuditLimit = 5
const OperatorStatusApprovalHistoryLimit = 5
const OperatorStatusArtifactLimit = 5

type OperatorStatusSummary struct {
	JobID                               string                                                      `json:"job_id"`
	ExecutionPlane                      string                                                      `json:"execution_plane,omitempty"`
	ExecutionHost                       string                                                      `json:"execution_host,omitempty"`
	MissionFamily                       string                                                      `json:"mission_family,omitempty"`
	PromotionPolicyID                   string                                                      `json:"promotion_policy_id,omitempty"`
	BaselineRef                         string                                                      `json:"baseline_ref,omitempty"`
	TrainRef                            string                                                      `json:"train_ref,omitempty"`
	HoldoutRef                          string                                                      `json:"holdout_ref,omitempty"`
	TargetSurfaces                      []JobSurfaceRef                                             `json:"target_surfaces,omitempty"`
	MutableSurfaces                     []JobSurfaceRef                                             `json:"mutable_surfaces,omitempty"`
	ImmutableSurfaces                   []JobSurfaceRef                                             `json:"immutable_surfaces,omitempty"`
	TopologyModeEnabled                 bool                                                        `json:"topology_mode_enabled,omitempty"`
	State                               JobState                                                    `json:"state"`
	ActiveStepID                        string                                                      `json:"active_step_id,omitempty"`
	AllowedTools                        []string                                                    `json:"allowed_tools,omitempty"`
	RuntimePackIdentity                 *OperatorRuntimePackIdentityStatus                          `json:"runtime_pack_identity,omitempty"`
	ImprovementCandidateIdentity        *OperatorImprovementCandidateIdentityStatus                 `json:"improvement_candidate_identity,omitempty"`
	EvalSuiteIdentity                   *OperatorEvalSuiteIdentityStatus                            `json:"eval_suite_identity,omitempty"`
	PromotionPolicyIdentity             *OperatorPromotionPolicyIdentityStatus                      `json:"promotion_policy_identity,omitempty"`
	ImprovementRunIdentity              *OperatorImprovementRunIdentityStatus                       `json:"improvement_run_identity,omitempty"`
	CandidateResultIdentity             *OperatorCandidateResultIdentityStatus                      `json:"candidate_result_identity,omitempty"`
	HotUpdateCanaryRequirementIdentity  *OperatorHotUpdateCanaryRequirementIdentityStatus           `json:"hot_update_canary_requirement_identity,omitempty"`
	HotUpdateCanaryEvidenceIdentity     *OperatorHotUpdateCanaryEvidenceIdentityStatus              `json:"hot_update_canary_evidence_identity,omitempty"`
	HotUpdateCanarySatisfactionIdentity *OperatorHotUpdateCanarySatisfactionIdentityStatus          `json:"hot_update_canary_satisfaction_identity,omitempty"`
	CandidatePromotionDecisionIdentity  *OperatorCandidatePromotionDecisionIdentityStatus           `json:"candidate_promotion_decision_identity,omitempty"`
	HotUpdateGateIdentity               *OperatorHotUpdateGateIdentityStatus                        `json:"hot_update_gate_identity,omitempty"`
	HotUpdateOutcomeIdentity            *OperatorHotUpdateOutcomeIdentityStatus                     `json:"hot_update_outcome_identity,omitempty"`
	PromotionIdentity                   *OperatorPromotionIdentityStatus                            `json:"promotion_identity,omitempty"`
	RollbackIdentity                    *OperatorRollbackIdentityStatus                             `json:"rollback_identity,omitempty"`
	RollbackApplyIdentity               *OperatorRollbackApplyIdentityStatus                        `json:"rollback_apply_identity,omitempty"`
	DeferredSchedulerTriggers           []OperatorDeferredSchedulerTriggerStatus                    `json:"deferred_scheduler_triggers,omitempty"`
	CampaignPreflight                   *ResolvedExecutionContextCampaignPreflight                  `json:"campaign_preflight,omitempty"`
	TreasuryPreflight                   *ResolvedExecutionContextTreasuryPreflight                  `json:"treasury_preflight,omitempty"`
	FrankZohoMailboxBootstrapPreflight  *ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight `json:"frank_zoho_mailbox_bootstrap_preflight,omitempty"`
	BudgetBlocker                       *OperatorBudgetBlockerStatus                                `json:"budget_blocker,omitempty"`
	WaitingReason                       string                                                      `json:"waiting_reason,omitempty"`
	WaitingAt                           *string                                                     `json:"waiting_at,omitempty"`
	PausedReason                        string                                                      `json:"paused_reason,omitempty"`
	PausedAt                            *string                                                     `json:"paused_at,omitempty"`
	AbortedReason                       string                                                      `json:"aborted_reason,omitempty"`
	FailedStepID                        string                                                      `json:"failed_step_id,omitempty"`
	FailureReason                       string                                                      `json:"failure_reason,omitempty"`
	FailedAt                            *string                                                     `json:"failed_at,omitempty"`
	ApprovalRequest                     *OperatorApprovalRequestStatus                              `json:"approval_request,omitempty"`
	ApprovalHistory                     []OperatorApprovalHistoryEntry                              `json:"approval_history,omitempty"`
	RecentAudit                         []OperatorRecentAuditStatus                                 `json:"recent_audit,omitempty"`
	Artifacts                           []OperatorArtifactStatus                                    `json:"artifacts,omitempty"`
	CampaignZohoEmailOutbounds          []OperatorCampaignZohoEmailOutboundActionStatus             `json:"campaign_zoho_email_outbounds,omitempty"`
	CampaignZohoEmailReplyWork          []OperatorCampaignZohoEmailReplyWorkItemStatus              `json:"campaign_zoho_email_reply_work,omitempty"`
	FrankZohoInboundReplies             []OperatorFrankZohoInboundReplyStatus                       `json:"frank_zoho_inbound_replies,omitempty"`
	CampaignZohoEmailSendGate           *CampaignZohoEmailSendGateDecision                          `json:"campaign_zoho_email_send_gate,omitempty"`
	FrankZohoSendProof                  []OperatorFrankZohoSendProofStatus                          `json:"frank_zoho_send_proof,omitempty"`
	Truncation                          *OperatorStatusTruncation                                   `json:"truncation,omitempty"`
}

type OperatorStatusTruncation struct {
	ApprovalHistoryOmitted int `json:"approval_history_omitted,omitempty"`
	RecentAuditOmitted     int `json:"recent_audit_omitted,omitempty"`
	ArtifactsOmitted       int `json:"artifacts_omitted,omitempty"`
}

type OperatorDeferredSchedulerTriggerStatus struct {
	TriggerID      string `json:"trigger_id"`
	SchedulerJobID string `json:"scheduler_job_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Message        string `json:"message,omitempty"`
	FireAt         string `json:"fire_at"`
	DeferredAt     string `json:"deferred_at"`
}

type OperatorRuntimePackIdentityStatus struct {
	Active        OperatorActiveRuntimePackStatus        `json:"active"`
	LastKnownGood OperatorLastKnownGoodRuntimePackStatus `json:"last_known_good"`
}

type OperatorImprovementCandidateIdentityStatus struct {
	State      string                               `json:"state"`
	Candidates []OperatorImprovementCandidateStatus `json:"candidates,omitempty"`
}

type OperatorImprovementCandidateStatus struct {
	State               string   `json:"state"`
	CandidateID         string   `json:"candidate_id,omitempty"`
	BaselinePackID      string   `json:"baseline_pack_id,omitempty"`
	CandidatePackID     string   `json:"candidate_pack_id,omitempty"`
	SourceWorkspaceRef  string   `json:"source_workspace_ref,omitempty"`
	SourceSummary       string   `json:"source_summary,omitempty"`
	ValidationBasisRefs []string `json:"validation_basis_refs,omitempty"`
	HotUpdateID         string   `json:"hot_update_id,omitempty"`
	CreatedAt           *string  `json:"created_at,omitempty"`
	CreatedBy           string   `json:"created_by,omitempty"`
	Error               string   `json:"error,omitempty"`
}

type OperatorEvalSuiteIdentityStatus struct {
	State  string                    `json:"state"`
	Suites []OperatorEvalSuiteStatus `json:"suites,omitempty"`
}

type OperatorPromotionPolicyIdentityStatus struct {
	State    string                          `json:"state"`
	Policies []OperatorPromotionPolicyStatus `json:"policies,omitempty"`
}

type OperatorPromotionPolicyStatus struct {
	State                     string   `json:"state"`
	PromotionPolicyID         string   `json:"promotion_policy_id,omitempty"`
	RequiresHoldoutPass       bool     `json:"requires_holdout_pass,omitempty"`
	RequiresCanary            bool     `json:"requires_canary,omitempty"`
	RequiresOwnerApproval     bool     `json:"requires_owner_approval,omitempty"`
	AllowsAutonomousHotUpdate bool     `json:"allows_autonomous_hot_update,omitempty"`
	AllowedSurfaceClasses     []string `json:"allowed_surface_classes,omitempty"`
	EpsilonRule               string   `json:"epsilon_rule,omitempty"`
	RegressionRule            string   `json:"regression_rule,omitempty"`
	CompatibilityRule         string   `json:"compatibility_rule,omitempty"`
	ResourceRule              string   `json:"resource_rule,omitempty"`
	MaxCanaryDuration         string   `json:"max_canary_duration,omitempty"`
	ForbiddenSurfaceChanges   []string `json:"forbidden_surface_changes,omitempty"`
	CreatedAt                 *string  `json:"created_at,omitempty"`
	CreatedBy                 string   `json:"created_by,omitempty"`
	Notes                     string   `json:"notes,omitempty"`
	Error                     string   `json:"error,omitempty"`
}

type OperatorEvalSuiteStatus struct {
	State            string  `json:"state"`
	EvalSuiteID      string  `json:"eval_suite_id,omitempty"`
	CandidateID      string  `json:"candidate_id,omitempty"`
	BaselinePackID   string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID  string  `json:"candidate_pack_id,omitempty"`
	RubricRef        string  `json:"rubric_ref,omitempty"`
	TrainCorpusRef   string  `json:"train_corpus_ref,omitempty"`
	HoldoutCorpusRef string  `json:"holdout_corpus_ref,omitempty"`
	EvaluatorRef     string  `json:"evaluator_ref,omitempty"`
	FrozenForRun     bool    `json:"frozen_for_run,omitempty"`
	CreatedAt        *string `json:"created_at,omitempty"`
	CreatedBy        string  `json:"created_by,omitempty"`
	Error            string  `json:"error,omitempty"`
}

type OperatorImprovementRunIdentityStatus struct {
	State string                         `json:"state"`
	Runs  []OperatorImprovementRunStatus `json:"runs,omitempty"`
}

type OperatorImprovementRunStatus struct {
	State           string  `json:"state"`
	RunID           string  `json:"run_id,omitempty"`
	CandidateID     string  `json:"candidate_id,omitempty"`
	EvalSuiteID     string  `json:"eval_suite_id,omitempty"`
	BaselinePackID  string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID string  `json:"candidate_pack_id,omitempty"`
	HotUpdateID     string  `json:"hot_update_id,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	CreatedBy       string  `json:"created_by,omitempty"`
	Error           string  `json:"error,omitempty"`
}

type OperatorCandidateResultIdentityStatus struct {
	State   string                          `json:"state"`
	Results []OperatorCandidateResultStatus `json:"results,omitempty"`
}

type OperatorCandidatePromotionDecisionIdentityStatus struct {
	State     string                                     `json:"state"`
	Decisions []OperatorCandidatePromotionDecisionStatus `json:"decisions,omitempty"`
}

type OperatorHotUpdateCanaryRequirementIdentityStatus struct {
	State        string                                     `json:"state"`
	Requirements []OperatorHotUpdateCanaryRequirementStatus `json:"requirements,omitempty"`
}

type OperatorHotUpdateCanaryEvidenceIdentityStatus struct {
	State    string                                  `json:"state"`
	Evidence []OperatorHotUpdateCanaryEvidenceStatus `json:"evidence,omitempty"`
}

type OperatorHotUpdateCanarySatisfactionIdentityStatus struct {
	State       string                                      `json:"state"`
	Assessments []OperatorHotUpdateCanarySatisfactionStatus `json:"assessments,omitempty"`
}

type OperatorHotUpdateOutcomeIdentityStatus struct {
	State    string                           `json:"state"`
	Outcomes []OperatorHotUpdateOutcomeStatus `json:"outcomes,omitempty"`
}

type OperatorHotUpdateGateIdentityStatus struct {
	State string                        `json:"state"`
	Gates []OperatorHotUpdateGateStatus `json:"gates,omitempty"`
}

type OperatorPromotionIdentityStatus struct {
	State      string                    `json:"state"`
	Promotions []OperatorPromotionStatus `json:"promotions,omitempty"`
}

type OperatorRollbackIdentityStatus struct {
	State     string                   `json:"state"`
	Rollbacks []OperatorRollbackStatus `json:"rollbacks,omitempty"`
}

type OperatorRollbackApplyIdentityStatus struct {
	State   string                        `json:"state"`
	Applies []OperatorRollbackApplyStatus `json:"applies,omitempty"`
}

type OperatorHotUpdateOutcomeStatus struct {
	State             string  `json:"state"`
	OutcomeID         string  `json:"outcome_id,omitempty"`
	HotUpdateID       string  `json:"hot_update_id,omitempty"`
	CandidateID       string  `json:"candidate_id,omitempty"`
	RunID             string  `json:"run_id,omitempty"`
	CandidateResultID string  `json:"candidate_result_id,omitempty"`
	CandidatePackID   string  `json:"candidate_pack_id,omitempty"`
	OutcomeKind       string  `json:"outcome_kind,omitempty"`
	Reason            string  `json:"reason,omitempty"`
	Notes             string  `json:"notes,omitempty"`
	OutcomeAt         *string `json:"outcome_at,omitempty"`
	CreatedAt         *string `json:"created_at,omitempty"`
	CreatedBy         string  `json:"created_by,omitempty"`
	Error             string  `json:"error,omitempty"`
}

type OperatorHotUpdateGateStatus struct {
	HotUpdateID              string   `json:"hot_update_id,omitempty"`
	CandidatePackID          string   `json:"candidate_pack_id,omitempty"`
	PreviousActivePackID     string   `json:"previous_active_pack_id,omitempty"`
	RollbackTargetPackID     string   `json:"rollback_target_pack_id,omitempty"`
	TargetSurfaces           []string `json:"target_surfaces,omitempty"`
	SurfaceClasses           []string `json:"surface_classes,omitempty"`
	ReloadMode               string   `json:"reload_mode,omitempty"`
	CompatibilityContractRef string   `json:"compatibility_contract_ref,omitempty"`
	PreparedAt               *string  `json:"prepared_at,omitempty"`
	PhaseUpdatedAt           *string  `json:"phase_updated_at,omitempty"`
	PhaseUpdatedBy           string   `json:"phase_updated_by,omitempty"`
	State                    string   `json:"state,omitempty"`
	Decision                 string   `json:"decision,omitempty"`
	FailureReason            string   `json:"failure_reason,omitempty"`
	Error                    string   `json:"error,omitempty"`
}

type OperatorPromotionStatus struct {
	State                string  `json:"state"`
	PromotionID          string  `json:"promotion_id,omitempty"`
	PromotedPackID       string  `json:"promoted_pack_id,omitempty"`
	PreviousActivePackID string  `json:"previous_active_pack_id,omitempty"`
	LastKnownGoodPackID  string  `json:"last_known_good_pack_id,omitempty"`
	LastKnownGoodBasis   string  `json:"last_known_good_basis,omitempty"`
	HotUpdateID          string  `json:"hot_update_id,omitempty"`
	OutcomeID            string  `json:"outcome_id,omitempty"`
	CandidateID          string  `json:"candidate_id,omitempty"`
	RunID                string  `json:"run_id,omitempty"`
	CandidateResultID    string  `json:"candidate_result_id,omitempty"`
	Reason               string  `json:"reason,omitempty"`
	Notes                string  `json:"notes,omitempty"`
	PromotedAt           *string `json:"promoted_at,omitempty"`
	CreatedAt            *string `json:"created_at,omitempty"`
	CreatedBy            string  `json:"created_by,omitempty"`
	Error                string  `json:"error,omitempty"`
}

type OperatorRollbackStatus struct {
	State               string  `json:"state"`
	RollbackID          string  `json:"rollback_id,omitempty"`
	PromotionID         string  `json:"promotion_id,omitempty"`
	HotUpdateID         string  `json:"hot_update_id,omitempty"`
	OutcomeID           string  `json:"outcome_id,omitempty"`
	FromPackID          string  `json:"from_pack_id,omitempty"`
	TargetPackID        string  `json:"target_pack_id,omitempty"`
	LastKnownGoodPackID string  `json:"last_known_good_pack_id,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	Notes               string  `json:"notes,omitempty"`
	RollbackAt          *string `json:"rollback_at,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type OperatorRollbackApplyStatus struct {
	State           string  `json:"state"`
	RollbackApplyID string  `json:"rollback_apply_id,omitempty"`
	RollbackID      string  `json:"rollback_id,omitempty"`
	Phase           string  `json:"phase,omitempty"`
	ActivationState string  `json:"activation_state,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	CreatedBy       string  `json:"created_by,omitempty"`
	Error           string  `json:"error,omitempty"`
}

type OperatorCandidateResultStatus struct {
	State                string                               `json:"state"`
	ResultID             string                               `json:"result_id,omitempty"`
	RunID                string                               `json:"run_id,omitempty"`
	CandidateID          string                               `json:"candidate_id,omitempty"`
	EvalSuiteID          string                               `json:"eval_suite_id,omitempty"`
	PromotionPolicyID    string                               `json:"promotion_policy_id,omitempty"`
	BaselinePackID       string                               `json:"baseline_pack_id,omitempty"`
	CandidatePackID      string                               `json:"candidate_pack_id,omitempty"`
	HotUpdateID          string                               `json:"hot_update_id,omitempty"`
	BaselineScore        float64                              `json:"baseline_score"`
	TrainScore           float64                              `json:"train_score"`
	HoldoutScore         float64                              `json:"holdout_score"`
	ComplexityScore      float64                              `json:"complexity_score"`
	CompatibilityScore   float64                              `json:"compatibility_score"`
	ResourceScore        float64                              `json:"resource_score"`
	RegressionFlags      []string                             `json:"regression_flags,omitempty"`
	Decision             string                               `json:"decision,omitempty"`
	Notes                string                               `json:"notes,omitempty"`
	PromotionEligibility *CandidatePromotionEligibilityStatus `json:"promotion_eligibility,omitempty"`
	CreatedAt            *string                              `json:"created_at,omitempty"`
	CreatedBy            string                               `json:"created_by,omitempty"`
	Error                string                               `json:"error,omitempty"`
}

type OperatorCandidatePromotionDecisionStatus struct {
	State               string  `json:"state"`
	PromotionDecisionID string  `json:"promotion_decision_id,omitempty"`
	ResultID            string  `json:"result_id,omitempty"`
	RunID               string  `json:"run_id,omitempty"`
	CandidateID         string  `json:"candidate_id,omitempty"`
	EvalSuiteID         string  `json:"eval_suite_id,omitempty"`
	PromotionPolicyID   string  `json:"promotion_policy_id,omitempty"`
	BaselinePackID      string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID     string  `json:"candidate_pack_id,omitempty"`
	EligibilityState    string  `json:"eligibility_state,omitempty"`
	Decision            string  `json:"decision,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	Notes               string  `json:"notes,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type OperatorHotUpdateCanaryRequirementStatus struct {
	State                 string  `json:"state"`
	CanaryRequirementID   string  `json:"canary_requirement_id,omitempty"`
	ResultID              string  `json:"result_id,omitempty"`
	RunID                 string  `json:"run_id,omitempty"`
	CandidateID           string  `json:"candidate_id,omitempty"`
	EvalSuiteID           string  `json:"eval_suite_id,omitempty"`
	PromotionPolicyID     string  `json:"promotion_policy_id,omitempty"`
	BaselinePackID        string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID       string  `json:"candidate_pack_id,omitempty"`
	EligibilityState      string  `json:"eligibility_state,omitempty"`
	RequiredByPolicy      bool    `json:"required_by_policy"`
	OwnerApprovalRequired bool    `json:"owner_approval_required"`
	RequirementState      string  `json:"requirement_state,omitempty"`
	Reason                string  `json:"reason,omitempty"`
	CreatedAt             *string `json:"created_at,omitempty"`
	CreatedBy             string  `json:"created_by,omitempty"`
	Error                 string  `json:"error,omitempty"`
}

type OperatorHotUpdateCanaryEvidenceStatus struct {
	State               string  `json:"state"`
	CanaryEvidenceID    string  `json:"canary_evidence_id,omitempty"`
	CanaryRequirementID string  `json:"canary_requirement_id,omitempty"`
	ResultID            string  `json:"result_id,omitempty"`
	RunID               string  `json:"run_id,omitempty"`
	CandidateID         string  `json:"candidate_id,omitempty"`
	EvalSuiteID         string  `json:"eval_suite_id,omitempty"`
	PromotionPolicyID   string  `json:"promotion_policy_id,omitempty"`
	BaselinePackID      string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID     string  `json:"candidate_pack_id,omitempty"`
	EvidenceState       string  `json:"evidence_state,omitempty"`
	Passed              bool    `json:"passed"`
	Reason              string  `json:"reason,omitempty"`
	ObservedAt          *string `json:"observed_at,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	CreatedBy           string  `json:"created_by,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type OperatorHotUpdateCanarySatisfactionStatus struct {
	State                    string  `json:"state"`
	CanaryRequirementID      string  `json:"canary_requirement_id,omitempty"`
	SelectedCanaryEvidenceID string  `json:"selected_canary_evidence_id,omitempty"`
	ResultID                 string  `json:"result_id,omitempty"`
	RunID                    string  `json:"run_id,omitempty"`
	CandidateID              string  `json:"candidate_id,omitempty"`
	EvalSuiteID              string  `json:"eval_suite_id,omitempty"`
	PromotionPolicyID        string  `json:"promotion_policy_id,omitempty"`
	BaselinePackID           string  `json:"baseline_pack_id,omitempty"`
	CandidatePackID          string  `json:"candidate_pack_id,omitempty"`
	EligibilityState         string  `json:"eligibility_state,omitempty"`
	OwnerApprovalRequired    bool    `json:"owner_approval_required"`
	SatisfactionState        string  `json:"satisfaction_state,omitempty"`
	EvidenceState            string  `json:"evidence_state,omitempty"`
	Passed                   bool    `json:"passed"`
	ObservedAt               *string `json:"observed_at,omitempty"`
	Reason                   string  `json:"reason,omitempty"`
	Error                    string  `json:"error,omitempty"`
}

type OperatorActiveRuntimePackStatus struct {
	State                string  `json:"state"`
	ActivePackID         string  `json:"active_pack_id,omitempty"`
	PreviousActivePackID string  `json:"previous_active_pack_id,omitempty"`
	LastKnownGoodPackID  string  `json:"last_known_good_pack_id,omitempty"`
	UpdatedAt            *string `json:"updated_at,omitempty"`
	Error                string  `json:"error,omitempty"`
}

type OperatorLastKnownGoodRuntimePackStatus struct {
	State      string  `json:"state"`
	PackID     string  `json:"pack_id,omitempty"`
	Basis      string  `json:"basis,omitempty"`
	VerifiedAt *string `json:"verified_at,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type OperatorArtifactStatus struct {
	StepID   string   `json:"step_id"`
	StepType StepType `json:"step_type"`
	Path     string   `json:"path"`
	State    string   `json:"state,omitempty"`
}

type OperatorCampaignZohoEmailOutboundActionStatus struct {
	ActionID                         string   `json:"action_id"`
	StepID                           string   `json:"step_id,omitempty"`
	CampaignID                       string   `json:"campaign_id"`
	State                            string   `json:"state"`
	Provider                         string   `json:"provider"`
	ProviderAccountID                string   `json:"provider_account_id"`
	FromAddress                      string   `json:"from_address"`
	FromDisplayName                  string   `json:"from_display_name,omitempty"`
	To                               []string `json:"to"`
	CC                               []string `json:"cc,omitempty"`
	BCC                              []string `json:"bcc,omitempty"`
	Subject                          string   `json:"subject"`
	BodyFormat                       string   `json:"body_format"`
	BodySHA256                       string   `json:"body_sha256"`
	PreparedAt                       *string  `json:"prepared_at,omitempty"`
	SentAt                           *string  `json:"sent_at,omitempty"`
	VerifiedAt                       *string  `json:"verified_at,omitempty"`
	FailedAt                         *string  `json:"failed_at,omitempty"`
	ReplyToInboundReplyID            string   `json:"reply_to_inbound_reply_id,omitempty"`
	ReplyToOutboundActionID          string   `json:"reply_to_outbound_action_id,omitempty"`
	ProviderMessageID                string   `json:"provider_message_id,omitempty"`
	ProviderMailID                   string   `json:"provider_mail_id,omitempty"`
	MIMEMessageID                    string   `json:"mime_message_id,omitempty"`
	OriginalMessageURL               string   `json:"original_message_url,omitempty"`
	FailureHTTPStatus                int      `json:"failure_http_status,omitempty"`
	FailureProviderStatusCode        int      `json:"failure_provider_status_code,omitempty"`
	FailureProviderStatusDescription string   `json:"failure_provider_status_description,omitempty"`
}

type OperatorFrankZohoInboundReplyStatus struct {
	ReplyID            string   `json:"reply_id"`
	StepID             string   `json:"step_id,omitempty"`
	Provider           string   `json:"provider"`
	ProviderAccountID  string   `json:"provider_account_id"`
	ProviderMessageID  string   `json:"provider_message_id"`
	ProviderMailID     string   `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string   `json:"mime_message_id,omitempty"`
	InReplyTo          string   `json:"in_reply_to,omitempty"`
	References         []string `json:"references,omitempty"`
	FromAddress        string   `json:"from_address,omitempty"`
	FromDisplayName    string   `json:"from_display_name,omitempty"`
	FromAddressCount   int      `json:"from_address_count,omitempty"`
	Subject            string   `json:"subject,omitempty"`
	ReceivedAt         *string  `json:"received_at,omitempty"`
	OriginalMessageURL string   `json:"original_message_url"`
}

type OperatorCampaignZohoEmailReplyWorkItemStatus struct {
	ReplyWorkItemID           string  `json:"reply_work_item_id"`
	InboundReplyID            string  `json:"inbound_reply_id"`
	CampaignID                string  `json:"campaign_id"`
	State                     string  `json:"state"`
	DerivedIterationState     string  `json:"derived_iteration_state,omitempty"`
	LatestFollowUpActionID    string  `json:"latest_follow_up_action_id,omitempty"`
	LatestFollowUpActionState string  `json:"latest_follow_up_action_state,omitempty"`
	DeferredUntil             *string `json:"deferred_until,omitempty"`
	ClaimedFollowUpActionID   string  `json:"claimed_followup_action_id,omitempty"`
	CreatedAt                 *string `json:"created_at,omitempty"`
	UpdatedAt                 *string `json:"updated_at,omitempty"`
}

type OperatorFrankZohoSendProofStatus struct {
	StepID             string `json:"step_id,omitempty"`
	ProviderMessageID  string `json:"provider_message_id"`
	ProviderMailID     string `json:"provider_mail_id,omitempty"`
	MIMEMessageID      string `json:"mime_message_id,omitempty"`
	ProviderAccountID  string `json:"provider_account_id"`
	OriginalMessageURL string `json:"original_message_url"`
}

type OperatorBudgetBlockerStatus struct {
	Ceiling     string  `json:"ceiling"`
	Limit       int     `json:"limit,omitempty"`
	Observed    int     `json:"observed,omitempty"`
	Message     string  `json:"message,omitempty"`
	TriggeredAt *string `json:"triggered_at,omitempty"`
}

type OperatorApprovalRequestStatus struct {
	State            ApprovalState `json:"state"`
	StepID           string        `json:"step_id"`
	RequestedAction  string        `json:"requested_action"`
	Scope            string        `json:"scope"`
	RequestedVia     string        `json:"requested_via,omitempty"`
	GrantedVia       string        `json:"granted_via,omitempty"`
	SessionChannel   string        `json:"session_channel,omitempty"`
	SessionChatID    string        `json:"session_chat_id,omitempty"`
	ProposedAction   string        `json:"proposed_action,omitempty"`
	WhyNeeded        string        `json:"why_needed,omitempty"`
	AuthorityTier    AuthorityTier `json:"authority_tier,omitempty"`
	FallbackIfDenied string        `json:"fallback_if_denied,omitempty"`
	ExpiresAt        *string       `json:"expires_at,omitempty"`
	SupersededAt     *string       `json:"superseded_at,omitempty"`
}

type OperatorRecentAuditStatus struct {
	EventID     string           `json:"event_id,omitempty"`
	JobID       string           `json:"job_id"`
	StepID      string           `json:"step_id,omitempty"`
	Action      string           `json:"action"`
	ActionClass AuditActionClass `json:"action_class,omitempty"`
	Result      AuditResult      `json:"result,omitempty"`
	Allowed     bool             `json:"allowed"`
	Code        RejectionCode    `json:"error_code,omitempty"`
	Timestamp   string           `json:"timestamp"`
}

type OperatorApprovalHistoryEntry struct {
	StepID          string  `json:"step_id"`
	RequestedAction string  `json:"requested_action"`
	Scope           string  `json:"scope"`
	State           string  `json:"state"`
	RequestedVia    string  `json:"requested_via"`
	GrantedVia      string  `json:"granted_via"`
	SessionChannel  string  `json:"session_channel"`
	SessionChatID   string  `json:"session_chat_id"`
	RequestedAt     *string `json:"requested_at"`
	ResolvedAt      *string `json:"resolved_at"`
	ExpiresAt       *string `json:"expires_at"`
	RevokedAt       *string `json:"revoked_at,omitempty"`
}

func BuildOperatorStatusSummary(runtime JobRuntimeState) OperatorStatusSummary {
	return buildOperatorStatusSummary(runtime, nil)
}

func BuildOperatorStatusSummaryWithAllowedTools(runtime JobRuntimeState, allowedTools []string) OperatorStatusSummary {
	return buildOperatorStatusSummary(runtime, allowedTools)
}

func FormatOperatorStatusSummary(runtime JobRuntimeState) (string, error) {
	return formatOperatorStatusSummary(buildOperatorStatusSummary(runtime, nil))
}

func FormatOperatorStatusSummaryWithAllowedTools(runtime JobRuntimeState, allowedTools []string) (string, error) {
	return formatOperatorStatusSummary(buildOperatorStatusSummary(runtime, allowedTools))
}

func FormatOperatorStatusSummaryWithAllowedToolsAndTreasuryPreflight(runtime JobRuntimeState, allowedTools []string, preflight *ResolvedExecutionContextTreasuryPreflight) (string, error) {
	return FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime, allowedTools, nil, preflight)
}

func FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryPreflight(runtime JobRuntimeState, allowedTools []string, campaignPreflight *ResolvedExecutionContextCampaignPreflight, treasuryPreflight *ResolvedExecutionContextTreasuryPreflight) (string, error) {
	return FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(runtime, allowedTools, campaignPreflight, treasuryPreflight, nil)
}

func FormatOperatorStatusSummaryWithAllowedToolsAndCampaignAndTreasuryAndFrankZohoMailboxBootstrapPreflight(runtime JobRuntimeState, allowedTools []string, campaignPreflight *ResolvedExecutionContextCampaignPreflight, treasuryPreflight *ResolvedExecutionContextTreasuryPreflight, bootstrapPreflight *ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight) (string, error) {
	summary := buildOperatorStatusSummary(runtime, allowedTools)
	if campaignPreflight != nil && campaignPreflight.Campaign != nil {
		summary.CampaignPreflight = cloneResolvedExecutionContextCampaignPreflight(campaignPreflight)
		decision, err := DeriveCampaignZohoEmailSendGateDecisionFromRuntime(*campaignPreflight.Campaign, runtime)
		if err != nil {
			summary.CampaignZohoEmailSendGate = &CampaignZohoEmailSendGateDecision{
				CampaignID: strings.TrimSpace(campaignPreflight.Campaign.CampaignID),
				Allowed:    false,
				Halted:     false,
				Reason:     err.Error(),
			}
		} else {
			summary.CampaignZohoEmailSendGate = &decision
		}
	}
	if treasuryPreflight != nil && treasuryPreflight.Treasury != nil {
		summary.TreasuryPreflight = cloneResolvedExecutionContextTreasuryPreflight(treasuryPreflight)
	}
	if bootstrapPreflight != nil && bootstrapPreflight.Identity != nil && bootstrapPreflight.Account != nil {
		summary.FrankZohoMailboxBootstrapPreflight = cloneResolvedExecutionContextFrankZohoMailboxBootstrapPreflight(bootstrapPreflight)
	}
	return formatOperatorStatusSummary(summary)
}

func WithDeferredSchedulerTriggers(summary OperatorStatusSummary, deferred []OperatorDeferredSchedulerTriggerStatus) OperatorStatusSummary {
	if len(deferred) == 0 {
		return summary
	}
	summary.DeferredSchedulerTriggers = append([]OperatorDeferredSchedulerTriggerStatus(nil), deferred...)
	return summary
}

func WithRuntimePackIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorRuntimePackIdentityStatus(root)
	summary.RuntimePackIdentity = &status
	return summary
}

func WithImprovementCandidateIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorImprovementCandidateIdentityStatus(root)
	summary.ImprovementCandidateIdentity = &status
	return summary
}

func WithEvalSuiteIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorEvalSuiteIdentityStatus(root)
	summary.EvalSuiteIdentity = &status
	return summary
}

func WithPromotionPolicyIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorPromotionPolicyIdentityStatus(root)
	summary.PromotionPolicyIdentity = &status
	return summary
}

func WithImprovementRunIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorImprovementRunIdentityStatus(root)
	summary.ImprovementRunIdentity = &status
	return summary
}

func WithCandidateResultIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorCandidateResultIdentityStatus(root)
	summary.CandidateResultIdentity = &status
	return summary
}

func WithCandidatePromotionDecisionIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorCandidatePromotionDecisionIdentityStatus(root)
	summary.CandidatePromotionDecisionIdentity = &status
	return summary
}

func WithHotUpdateCanaryRequirementIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root)
	summary.HotUpdateCanaryRequirementIdentity = &status
	return summary
}

func WithHotUpdateCanaryEvidenceIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root)
	summary.HotUpdateCanaryEvidenceIdentity = &status
	return summary
}

func WithHotUpdateCanarySatisfactionIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root)
	summary.HotUpdateCanarySatisfactionIdentity = &status
	return summary
}

func WithHotUpdateGateIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorHotUpdateGateIdentityStatus(root)
	summary.HotUpdateGateIdentity = &status
	return summary
}

func WithHotUpdateOutcomeIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorHotUpdateOutcomeIdentityStatus(root)
	summary.HotUpdateOutcomeIdentity = &status
	return summary
}

func WithPromotionIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorPromotionIdentityStatus(root)
	summary.PromotionIdentity = &status
	return summary
}

func WithRollbackIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorRollbackIdentityStatus(root)
	summary.RollbackIdentity = &status
	return summary
}

func WithRollbackApplyIdentity(summary OperatorStatusSummary, root string) OperatorStatusSummary {
	root = strings.TrimSpace(root)
	if root == "" {
		return summary
	}
	status := LoadOperatorRollbackApplyIdentityStatus(root)
	summary.RollbackApplyIdentity = &status
	return summary
}

func LoadOperatorRuntimePackIdentityStatus(root string) OperatorRuntimePackIdentityStatus {
	return OperatorRuntimePackIdentityStatus{
		Active:        loadOperatorActiveRuntimePackStatus(root),
		LastKnownGood: loadOperatorLastKnownGoodRuntimePackStatus(root),
	}
}

func LoadOperatorImprovementCandidateIdentityStatus(root string) OperatorImprovementCandidateIdentityStatus {
	candidates, found, invalid, err := loadOperatorImprovementCandidateStatuses(root)
	if !found {
		return OperatorImprovementCandidateIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorImprovementCandidateIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorImprovementCandidateIdentityStatus{
		State:      state,
		Candidates: candidates,
	}
}

func LoadOperatorEvalSuiteIdentityStatus(root string) OperatorEvalSuiteIdentityStatus {
	suites, found, invalid, err := loadOperatorEvalSuiteStatuses(root)
	if !found {
		return OperatorEvalSuiteIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorEvalSuiteIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorEvalSuiteIdentityStatus{
		State:  state,
		Suites: suites,
	}
}

func LoadOperatorPromotionPolicyIdentityStatus(root string) OperatorPromotionPolicyIdentityStatus {
	policies, found, invalid, err := loadOperatorPromotionPolicyStatuses(root)
	if !found {
		return OperatorPromotionPolicyIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorPromotionPolicyIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorPromotionPolicyIdentityStatus{
		State:    state,
		Policies: policies,
	}
}

func LoadOperatorImprovementRunIdentityStatus(root string) OperatorImprovementRunIdentityStatus {
	runs, found, invalid, err := loadOperatorImprovementRunStatuses(root)
	if !found {
		return OperatorImprovementRunIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorImprovementRunIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorImprovementRunIdentityStatus{
		State: state,
		Runs:  runs,
	}
}

func LoadOperatorCandidateResultIdentityStatus(root string) OperatorCandidateResultIdentityStatus {
	results, found, invalid, err := loadOperatorCandidateResultStatuses(root)
	if !found {
		return OperatorCandidateResultIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorCandidateResultIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorCandidateResultIdentityStatus{
		State:   state,
		Results: results,
	}
}

func LoadOperatorCandidatePromotionDecisionIdentityStatus(root string) OperatorCandidatePromotionDecisionIdentityStatus {
	decisions, found, invalid, err := loadOperatorCandidatePromotionDecisionStatuses(root)
	if !found {
		return OperatorCandidatePromotionDecisionIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorCandidatePromotionDecisionIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorCandidatePromotionDecisionIdentityStatus{
		State:     state,
		Decisions: decisions,
	}
}

func LoadOperatorHotUpdateCanaryRequirementIdentityStatus(root string) OperatorHotUpdateCanaryRequirementIdentityStatus {
	requirements, found, invalid, err := loadOperatorHotUpdateCanaryRequirementStatuses(root)
	if !found {
		return OperatorHotUpdateCanaryRequirementIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorHotUpdateCanaryRequirementIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorHotUpdateCanaryRequirementIdentityStatus{
		State:        state,
		Requirements: requirements,
	}
}

func LoadOperatorHotUpdateCanaryEvidenceIdentityStatus(root string) OperatorHotUpdateCanaryEvidenceIdentityStatus {
	evidence, found, invalid, err := loadOperatorHotUpdateCanaryEvidenceStatuses(root)
	if !found {
		return OperatorHotUpdateCanaryEvidenceIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorHotUpdateCanaryEvidenceIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorHotUpdateCanaryEvidenceIdentityStatus{
		State:    state,
		Evidence: evidence,
	}
}

func LoadOperatorHotUpdateCanarySatisfactionIdentityStatus(root string) OperatorHotUpdateCanarySatisfactionIdentityStatus {
	assessments, found, invalid, err := loadOperatorHotUpdateCanarySatisfactionStatuses(root)
	if !found {
		return OperatorHotUpdateCanarySatisfactionIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorHotUpdateCanarySatisfactionIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorHotUpdateCanarySatisfactionIdentityStatus{
		State:       state,
		Assessments: assessments,
	}
}

func LoadOperatorHotUpdateGateIdentityStatus(root string) OperatorHotUpdateGateIdentityStatus {
	gates, found, invalid, err := loadOperatorHotUpdateGateStatuses(root)
	if !found {
		return OperatorHotUpdateGateIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorHotUpdateGateIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorHotUpdateGateIdentityStatus{
		State: state,
		Gates: gates,
	}
}

func LoadOperatorHotUpdateOutcomeIdentityStatus(root string) OperatorHotUpdateOutcomeIdentityStatus {
	outcomes, found, invalid, err := loadOperatorHotUpdateOutcomeStatuses(root)
	if !found {
		return OperatorHotUpdateOutcomeIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorHotUpdateOutcomeIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorHotUpdateOutcomeIdentityStatus{
		State:    state,
		Outcomes: outcomes,
	}
}

func LoadOperatorPromotionIdentityStatus(root string) OperatorPromotionIdentityStatus {
	promotions, found, invalid, err := loadOperatorPromotionStatuses(root)
	if !found {
		return OperatorPromotionIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorPromotionIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorPromotionIdentityStatus{
		State:      state,
		Promotions: promotions,
	}
}

func LoadOperatorRollbackIdentityStatus(root string) OperatorRollbackIdentityStatus {
	rollbacks, found, invalid, err := loadOperatorRollbackStatuses(root)
	if !found {
		return OperatorRollbackIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorRollbackIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorRollbackIdentityStatus{
		State:     state,
		Rollbacks: rollbacks,
	}
}

func LoadOperatorRollbackApplyIdentityStatus(root string) OperatorRollbackApplyIdentityStatus {
	applies, found, invalid, err := loadOperatorRollbackApplyStatuses(root)
	if !found {
		return OperatorRollbackApplyIdentityStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorRollbackApplyIdentityStatus{State: "invalid"}
	}
	state := "configured"
	if invalid {
		state = "invalid"
	}
	return OperatorRollbackApplyIdentityStatus{
		State:   state,
		Applies: applies,
	}
}

func EffectiveAllowedTools(job *Job, step *Step) []string {
	if job == nil {
		return nil
	}

	jobTools := make(map[string]struct{}, len(job.AllowedTools))
	for _, toolName := range job.AllowedTools {
		jobTools[toolName] = struct{}{}
	}

	allowed := make([]string, 0, len(jobTools))
	if step == nil || len(step.AllowedTools) == 0 {
		for toolName := range jobTools {
			allowed = append(allowed, toolName)
		}
		sort.Strings(allowed)
		return allowed
	}

	for _, toolName := range step.AllowedTools {
		if _, ok := jobTools[toolName]; ok {
			allowed = append(allowed, toolName)
			delete(jobTools, toolName)
		}
	}
	sort.Strings(allowed)
	return allowed
}

func buildOperatorStatusSummary(runtime JobRuntimeState, allowedTools []string) OperatorStatusSummary {
	executionPlane := strings.TrimSpace(runtime.ExecutionPlane)
	executionHost := strings.TrimSpace(runtime.ExecutionHost)
	missionFamily := strings.TrimSpace(runtime.MissionFamily)
	promotionPolicyID := strings.TrimSpace(runtime.PromotionPolicyID)
	baselineRef := strings.TrimSpace(runtime.BaselineRef)
	trainRef := strings.TrimSpace(runtime.TrainRef)
	holdoutRef := strings.TrimSpace(runtime.HoldoutRef)
	targetSurfaces := cloneJobSurfaceRefs(runtime.TargetSurfaces)
	mutableSurfaces := cloneJobSurfaceRefs(runtime.MutableSurfaces)
	immutableSurfaces := cloneJobSurfaceRefs(runtime.ImmutableSurfaces)
	topologyModeEnabled := runtime.TopologyModeEnabled
	if runtime.InspectablePlan != nil {
		if executionPlane == "" {
			executionPlane = strings.TrimSpace(runtime.InspectablePlan.ExecutionPlane)
		}
		if executionHost == "" {
			executionHost = strings.TrimSpace(runtime.InspectablePlan.ExecutionHost)
		}
		if missionFamily == "" {
			missionFamily = strings.TrimSpace(runtime.InspectablePlan.MissionFamily)
		}
		if promotionPolicyID == "" {
			promotionPolicyID = strings.TrimSpace(runtime.InspectablePlan.PromotionPolicyID)
		}
		if baselineRef == "" {
			baselineRef = strings.TrimSpace(runtime.InspectablePlan.BaselineRef)
		}
		if trainRef == "" {
			trainRef = strings.TrimSpace(runtime.InspectablePlan.TrainRef)
		}
		if holdoutRef == "" {
			holdoutRef = strings.TrimSpace(runtime.InspectablePlan.HoldoutRef)
		}
		if len(targetSurfaces) == 0 {
			targetSurfaces = cloneJobSurfaceRefs(runtime.InspectablePlan.TargetSurfaces)
		}
		if len(mutableSurfaces) == 0 {
			mutableSurfaces = cloneJobSurfaceRefs(runtime.InspectablePlan.MutableSurfaces)
		}
		if len(immutableSurfaces) == 0 {
			immutableSurfaces = cloneJobSurfaceRefs(runtime.InspectablePlan.ImmutableSurfaces)
		}
		if !topologyModeEnabled {
			topologyModeEnabled = runtime.InspectablePlan.TopologyModeEnabled
		}
	}

	summary := OperatorStatusSummary{
		JobID:               runtime.JobID,
		ExecutionPlane:      executionPlane,
		ExecutionHost:       executionHost,
		MissionFamily:       missionFamily,
		PromotionPolicyID:   promotionPolicyID,
		BaselineRef:         baselineRef,
		TrainRef:            trainRef,
		HoldoutRef:          holdoutRef,
		TargetSurfaces:      targetSurfaces,
		MutableSurfaces:     mutableSurfaces,
		ImmutableSurfaces:   immutableSurfaces,
		TopologyModeEnabled: topologyModeEnabled,
		State:               runtime.State,
		ActiveStepID:        runtime.ActiveStepID,
		WaitingReason:       runtime.WaitingReason,
		WaitingAt:           formatOperatorStatusTime(runtime.WaitingAt),
		PausedReason:        runtime.PausedReason,
		PausedAt:            formatOperatorStatusTime(runtime.PausedAt),
		AbortedReason:       runtime.AbortedReason,
	}
	if runtime.BudgetBlocker != nil {
		summary.BudgetBlocker = &OperatorBudgetBlockerStatus{
			Ceiling:     runtime.BudgetBlocker.Ceiling,
			Limit:       runtime.BudgetBlocker.Limit,
			Observed:    runtime.BudgetBlocker.Observed,
			Message:     runtime.BudgetBlocker.Message,
			TriggeredAt: formatOperatorStatusTime(runtime.BudgetBlocker.TriggeredAt),
		}
	}
	if runtime.State == JobStateFailed {
		if record, ok := selectOperatorStatusLatestFailedStep(runtime); ok {
			summary.FailedStepID = record.StepID
			summary.FailureReason = record.Reason
		}
		summary.FailedAt = selectOperatorStatusFailedAt(runtime)
	}
	if allowedTools != nil {
		summary.AllowedTools = append([]string(nil), allowedTools...)
	}

	if request, ok := selectOperatorStatusApprovalRequest(runtime); ok {
		status := OperatorApprovalRequestStatus{
			State:           request.State,
			StepID:          request.StepID,
			RequestedAction: request.RequestedAction,
			Scope:           request.Scope,
			RequestedVia:    request.RequestedVia,
			GrantedVia:      request.GrantedVia,
			SessionChannel:  request.SessionChannel,
			SessionChatID:   request.SessionChatID,
		}
		if request.Content != nil {
			status.ProposedAction = request.Content.ProposedAction
			status.WhyNeeded = request.Content.WhyNeeded
			status.AuthorityTier = request.Content.AuthorityTier
			status.FallbackIfDenied = request.Content.FallbackIfDenied
		}
		if !request.ExpiresAt.IsZero() {
			expiresAt := request.ExpiresAt.UTC().Format(time.RFC3339Nano)
			status.ExpiresAt = &expiresAt
		}
		if !request.SupersededAt.IsZero() {
			supersededAt := request.SupersededAt.UTC().Format(time.RFC3339Nano)
			status.SupersededAt = &supersededAt
		}
		summary.ApprovalRequest = &status
	}
	summary.ApprovalHistory = selectOperatorStatusApprovalHistory(runtime)
	summary.RecentAudit = selectOperatorStatusRecentAudit(runtime)
	summary.Artifacts = selectOperatorStatusArtifacts(runtime)
	summary.CampaignZohoEmailOutbounds = selectOperatorStatusCampaignZohoEmailOutbounds(runtime)
	summary.CampaignZohoEmailReplyWork = selectOperatorStatusCampaignZohoEmailReplyWorkItems(runtime)
	summary.FrankZohoInboundReplies = selectOperatorStatusFrankZohoInboundReplies(runtime)
	summary.FrankZohoSendProof = selectOperatorStatusFrankZohoSendProof(runtime)
	summary.Truncation = buildOperatorStatusTruncation(runtime, len(summary.ApprovalHistory), len(summary.RecentAudit), len(summary.Artifacts))

	return summary
}

func selectOperatorStatusLatestFailedStep(runtime JobRuntimeState) (RuntimeStepRecord, bool) {
	if len(runtime.FailedSteps) == 0 {
		return RuntimeStepRecord{}, false
	}
	return cloneRuntimeStepRecord(runtime.FailedSteps[len(runtime.FailedSteps)-1]), true
}

func selectOperatorStatusFailedAt(runtime JobRuntimeState) *string {
	if failedAt := formatOperatorStatusTime(runtime.FailedAt); failedAt != nil {
		return failedAt
	}
	if record, ok := selectOperatorStatusLatestFailedStep(runtime); ok {
		return formatOperatorStatusTime(record.At)
	}
	return nil
}

func cloneResolvedExecutionContextTreasuryPreflight(preflight *ResolvedExecutionContextTreasuryPreflight) *ResolvedExecutionContextTreasuryPreflight {
	if preflight == nil {
		return nil
	}

	cloned := ResolvedExecutionContextTreasuryPreflight{
		Containers: append([]FrankContainerRecord(nil), preflight.Containers...),
	}
	if preflight.Treasury != nil {
		treasury := *preflight.Treasury
		cloned.Treasury = &treasury
	}
	return &cloned
}

func cloneResolvedExecutionContextCampaignPreflight(preflight *ResolvedExecutionContextCampaignPreflight) *ResolvedExecutionContextCampaignPreflight {
	if preflight == nil {
		return nil
	}

	cloned := ResolvedExecutionContextCampaignPreflight{
		Identities: append([]FrankIdentityRecord(nil), preflight.Identities...),
		Accounts:   append([]FrankAccountRecord(nil), preflight.Accounts...),
		Containers: append([]FrankContainerRecord(nil), preflight.Containers...),
	}
	if preflight.Campaign != nil {
		campaign := *preflight.Campaign
		cloned.Campaign = &campaign
	}
	return &cloned
}

func cloneResolvedExecutionContextFrankZohoMailboxBootstrapPreflight(preflight *ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight) *ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight {
	if preflight == nil {
		return nil
	}

	cloned := ResolvedExecutionContextFrankZohoMailboxBootstrapPreflight{}
	if preflight.Identity != nil {
		identity := *preflight.Identity
		cloned.Identity = &identity
	}
	if preflight.Account != nil {
		account := *preflight.Account
		cloned.Account = &account
	}
	return &cloned
}

func formatOperatorStatusSummary(summary OperatorStatusSummary) (string, error) {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func loadOperatorActiveRuntimePackStatus(root string) OperatorActiveRuntimePackStatus {
	pointer, found, err := loadOperatorActiveRuntimePackPointerReadModel(root)
	if !found {
		return OperatorActiveRuntimePackStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorActiveRuntimePackStatus{
			State:                "invalid",
			ActivePackID:         pointer.ActivePackID,
			PreviousActivePackID: pointer.PreviousActivePackID,
			LastKnownGoodPackID:  pointer.LastKnownGoodPackID,
			UpdatedAt:            formatOperatorStatusTime(pointer.UpdatedAt),
			Error:                err.Error(),
		}
	}
	return OperatorActiveRuntimePackStatus{
		State:                "configured",
		ActivePackID:         pointer.ActivePackID,
		PreviousActivePackID: pointer.PreviousActivePackID,
		LastKnownGoodPackID:  pointer.LastKnownGoodPackID,
		UpdatedAt:            formatOperatorStatusTime(pointer.UpdatedAt),
	}
}

func loadOperatorLastKnownGoodRuntimePackStatus(root string) OperatorLastKnownGoodRuntimePackStatus {
	pointer, found, err := loadOperatorLastKnownGoodRuntimePackPointerReadModel(root)
	if !found {
		return OperatorLastKnownGoodRuntimePackStatus{State: "not_configured"}
	}
	if err != nil {
		return OperatorLastKnownGoodRuntimePackStatus{
			State:      "invalid",
			PackID:     pointer.PackID,
			Basis:      pointer.Basis,
			VerifiedAt: formatOperatorStatusTime(pointer.VerifiedAt),
			Error:      err.Error(),
		}
	}
	return OperatorLastKnownGoodRuntimePackStatus{
		State:      "configured",
		PackID:     pointer.PackID,
		Basis:      pointer.Basis,
		VerifiedAt: formatOperatorStatusTime(pointer.VerifiedAt),
	}
}

func loadOperatorActiveRuntimePackPointerReadModel(root string) (ActiveRuntimePackPointer, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return ActiveRuntimePackPointer{}, true, err
	}

	var pointer ActiveRuntimePackPointer
	if err := LoadStoreJSON(StoreActiveRuntimePackPointerPath(root), &pointer); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ActiveRuntimePackPointer{}, false, nil
		}
		return ActiveRuntimePackPointer{}, true, err
	}
	pointer = NormalizeActiveRuntimePackPointer(pointer)
	if err := ValidateActiveRuntimePackPointer(pointer); err != nil {
		return pointer, true, err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.ActivePackID); err != nil {
		return pointer, true, fmtOperatorRuntimePackIdentityPointerError("mission store active runtime pack pointer", "active_pack_id", pointer.ActivePackID, err)
	}
	if pointer.PreviousActivePackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.PreviousActivePackID); err != nil {
			return pointer, true, fmtOperatorRuntimePackIdentityPointerError("mission store active runtime pack pointer", "previous_active_pack_id", pointer.PreviousActivePackID, err)
		}
	}
	if pointer.LastKnownGoodPackID != "" {
		if _, err := LoadRuntimePackRecord(root, pointer.LastKnownGoodPackID); err != nil {
			return pointer, true, fmtOperatorRuntimePackIdentityPointerError("mission store active runtime pack pointer", "last_known_good_pack_id", pointer.LastKnownGoodPackID, err)
		}
	}
	return pointer, true, nil
}

func loadOperatorLastKnownGoodRuntimePackPointerReadModel(root string) (LastKnownGoodRuntimePackPointer, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return LastKnownGoodRuntimePackPointer{}, true, err
	}

	var pointer LastKnownGoodRuntimePackPointer
	if err := LoadStoreJSON(StoreLastKnownGoodRuntimePackPointerPath(root), &pointer); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LastKnownGoodRuntimePackPointer{}, false, nil
		}
		return LastKnownGoodRuntimePackPointer{}, true, err
	}
	pointer = NormalizeLastKnownGoodRuntimePackPointer(pointer)
	if err := ValidateLastKnownGoodRuntimePackPointer(pointer); err != nil {
		return pointer, true, err
	}
	if _, err := LoadRuntimePackRecord(root, pointer.PackID); err != nil {
		return pointer, true, fmtOperatorRuntimePackIdentityPointerError("mission store last-known-good runtime pack pointer", "pack_id", pointer.PackID, err)
	}
	return pointer, true, nil
}

func fmtOperatorRuntimePackIdentityPointerError(surface, fieldName, value string, err error) error {
	return &operatorRuntimePackIdentityPointerError{
		surface:   surface,
		fieldName: fieldName,
		value:     value,
		err:       err,
	}
}

type operatorRuntimePackIdentityPointerError struct {
	surface   string
	fieldName string
	value     string
	err       error
}

func (e *operatorRuntimePackIdentityPointerError) Error() string {
	return e.surface + " " + e.fieldName + ` "` + e.value + `": ` + e.err.Error()
}

func loadOperatorImprovementCandidateStatuses(root string) ([]OperatorImprovementCandidateStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreImprovementCandidatesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	candidates := make([]OperatorImprovementCandidateStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorImprovementCandidateStatus(root, filepath.Join(StoreImprovementCandidatesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		candidates = append(candidates, status)
	}
	return candidates, true, invalid, nil
}

func loadOperatorImprovementCandidateStatus(root, path string) OperatorImprovementCandidateStatus {
	status := OperatorImprovementCandidateStatus{
		State:       "invalid",
		CandidateID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record ImprovementCandidateRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeImprovementCandidateRecord(record)
	status = operatorImprovementCandidateStatusFromRecord(record)
	if err := ValidateImprovementCandidateRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateImprovementCandidateLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorImprovementCandidateStatusFromRecord(record ImprovementCandidateRecord) OperatorImprovementCandidateStatus {
	return OperatorImprovementCandidateStatus{
		CandidateID:         record.CandidateID,
		BaselinePackID:      record.BaselinePackID,
		CandidatePackID:     record.CandidatePackID,
		SourceWorkspaceRef:  record.SourceWorkspaceRef,
		SourceSummary:       record.SourceSummary,
		ValidationBasisRefs: append([]string(nil), record.ValidationBasisRefs...),
		HotUpdateID:         record.HotUpdateID,
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorEvalSuiteStatuses(root string) ([]OperatorEvalSuiteStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreEvalSuitesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	suites := make([]OperatorEvalSuiteStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorEvalSuiteStatus(root, filepath.Join(StoreEvalSuitesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		suites = append(suites, status)
	}
	return suites, true, invalid, nil
}

func loadOperatorEvalSuiteStatus(root, path string) OperatorEvalSuiteStatus {
	status := OperatorEvalSuiteStatus{
		State:       "invalid",
		EvalSuiteID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record EvalSuiteRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeEvalSuiteRecord(record)
	status = operatorEvalSuiteStatusFromRecord(record)
	if err := ValidateEvalSuiteRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateEvalSuiteLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorEvalSuiteStatusFromRecord(record EvalSuiteRecord) OperatorEvalSuiteStatus {
	return OperatorEvalSuiteStatus{
		EvalSuiteID:      record.EvalSuiteID,
		CandidateID:      record.CandidateID,
		BaselinePackID:   record.BaselinePackID,
		CandidatePackID:  record.CandidatePackID,
		RubricRef:        record.RubricRef,
		TrainCorpusRef:   record.TrainCorpusRef,
		HoldoutCorpusRef: record.HoldoutCorpusRef,
		EvaluatorRef:     record.EvaluatorRef,
		FrozenForRun:     record.FrozenForRun,
		CreatedAt:        formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:        record.CreatedBy,
	}
}

func loadOperatorPromotionPolicyStatuses(root string) ([]OperatorPromotionPolicyStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StorePromotionPoliciesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	policies := make([]OperatorPromotionPolicyStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorPromotionPolicyStatus(filepath.Join(StorePromotionPoliciesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		policies = append(policies, status)
	}
	return policies, true, invalid, nil
}

func loadOperatorPromotionPolicyStatus(path string) OperatorPromotionPolicyStatus {
	status := OperatorPromotionPolicyStatus{
		State:             "invalid",
		PromotionPolicyID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record PromotionPolicyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizePromotionPolicyRecord(record)
	status = operatorPromotionPolicyStatusFromRecord(record)
	if err := ValidatePromotionPolicyRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorPromotionPolicyStatusFromRecord(record PromotionPolicyRecord) OperatorPromotionPolicyStatus {
	return OperatorPromotionPolicyStatus{
		PromotionPolicyID:         record.PromotionPolicyID,
		RequiresHoldoutPass:       record.RequiresHoldoutPass,
		RequiresCanary:            record.RequiresCanary,
		RequiresOwnerApproval:     record.RequiresOwnerApproval,
		AllowsAutonomousHotUpdate: record.AllowsAutonomousHotUpdate,
		AllowedSurfaceClasses:     append([]string(nil), record.AllowedSurfaceClasses...),
		EpsilonRule:               record.EpsilonRule,
		RegressionRule:            record.RegressionRule,
		CompatibilityRule:         record.CompatibilityRule,
		ResourceRule:              record.ResourceRule,
		MaxCanaryDuration:         record.MaxCanaryDuration,
		ForbiddenSurfaceChanges:   append([]string(nil), record.ForbiddenSurfaceChanges...),
		CreatedAt:                 formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:                 record.CreatedBy,
		Notes:                     record.Notes,
	}
}

func loadOperatorImprovementRunStatuses(root string) ([]OperatorImprovementRunStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreImprovementRunsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	runs := make([]OperatorImprovementRunStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorImprovementRunStatus(root, filepath.Join(StoreImprovementRunsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		runs = append(runs, status)
	}
	return runs, true, invalid, nil
}

func loadOperatorImprovementRunStatus(root, path string) OperatorImprovementRunStatus {
	status := OperatorImprovementRunStatus{
		State: "invalid",
		RunID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record ImprovementRunRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeImprovementRunRecord(record)
	status = operatorImprovementRunStatusFromRecord(record)
	if err := ValidateImprovementRunRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateImprovementRunLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorImprovementRunStatusFromRecord(record ImprovementRunRecord) OperatorImprovementRunStatus {
	return OperatorImprovementRunStatus{
		RunID:           record.RunID,
		CandidateID:     record.CandidateID,
		EvalSuiteID:     record.EvalSuiteID,
		BaselinePackID:  record.BaselinePackID,
		CandidatePackID: record.CandidatePackID,
		HotUpdateID:     record.HotUpdateID,
		CreatedAt:       formatOperatorStatusTime(record.CreatedAt),
		CompletedAt:     formatOperatorStatusTime(record.CompletedAt),
		CreatedBy:       record.CreatedBy,
	}
}

func loadOperatorCandidateResultStatuses(root string) ([]OperatorCandidateResultStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreCandidateResultsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	results := make([]OperatorCandidateResultStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorCandidateResultStatus(root, filepath.Join(StoreCandidateResultsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		results = append(results, status)
	}
	return results, true, invalid, nil
}

func loadOperatorCandidateResultStatus(root, path string) OperatorCandidateResultStatus {
	status := OperatorCandidateResultStatus{
		State:    "invalid",
		ResultID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record CandidateResultRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeCandidateResultRecord(record)
	status = operatorCandidateResultStatusFromRecord(record)
	if err := ValidateCandidateResultRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateCandidateResultLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	eligibility, err := EvaluateCandidateResultPromotionEligibility(root, record.ResultID)
	if err != nil {
		eligibility = CandidatePromotionEligibilityStatus{
			State:    CandidatePromotionEligibilityStateInvalid,
			ResultID: record.ResultID,
			Error:    err.Error(),
		}
	}
	status.PromotionEligibility = &eligibility
	status.State = "configured"
	return status
}

func operatorCandidateResultStatusFromRecord(record CandidateResultRecord) OperatorCandidateResultStatus {
	return OperatorCandidateResultStatus{
		ResultID:           record.ResultID,
		RunID:              record.RunID,
		CandidateID:        record.CandidateID,
		EvalSuiteID:        record.EvalSuiteID,
		PromotionPolicyID:  record.PromotionPolicyID,
		BaselinePackID:     record.BaselinePackID,
		CandidatePackID:    record.CandidatePackID,
		HotUpdateID:        record.HotUpdateID,
		BaselineScore:      record.BaselineScore,
		TrainScore:         record.TrainScore,
		HoldoutScore:       record.HoldoutScore,
		ComplexityScore:    record.ComplexityScore,
		CompatibilityScore: record.CompatibilityScore,
		ResourceScore:      record.ResourceScore,
		RegressionFlags:    append([]string(nil), record.RegressionFlags...),
		Decision:           string(record.Decision),
		Notes:              record.Notes,
		CreatedAt:          formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:          record.CreatedBy,
	}
}

func loadOperatorCandidatePromotionDecisionStatuses(root string) ([]OperatorCandidatePromotionDecisionStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreCandidatePromotionDecisionsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	decisions := make([]OperatorCandidatePromotionDecisionStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorCandidatePromotionDecisionStatus(root, filepath.Join(StoreCandidatePromotionDecisionsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		decisions = append(decisions, status)
	}
	return decisions, true, invalid, nil
}

func loadOperatorCandidatePromotionDecisionStatus(root, path string) OperatorCandidatePromotionDecisionStatus {
	status := OperatorCandidatePromotionDecisionStatus{
		State:               "invalid",
		PromotionDecisionID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record CandidatePromotionDecisionRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeCandidatePromotionDecisionRecord(record)
	status = operatorCandidatePromotionDecisionStatusFromRecord(record)
	if err := ValidateCandidatePromotionDecisionRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateCandidatePromotionDecisionLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorCandidatePromotionDecisionStatusFromRecord(record CandidatePromotionDecisionRecord) OperatorCandidatePromotionDecisionStatus {
	return OperatorCandidatePromotionDecisionStatus{
		PromotionDecisionID: record.PromotionDecisionID,
		ResultID:            record.ResultID,
		RunID:               record.RunID,
		CandidateID:         record.CandidateID,
		EvalSuiteID:         record.EvalSuiteID,
		PromotionPolicyID:   record.PromotionPolicyID,
		BaselinePackID:      record.BaselinePackID,
		CandidatePackID:     record.CandidatePackID,
		EligibilityState:    record.EligibilityState,
		Decision:            string(record.Decision),
		Reason:              record.Reason,
		Notes:               record.Notes,
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorHotUpdateCanaryRequirementStatuses(root string) ([]OperatorHotUpdateCanaryRequirementStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreHotUpdateCanaryRequirementsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	requirements := make([]OperatorHotUpdateCanaryRequirementStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorHotUpdateCanaryRequirementStatus(root, filepath.Join(StoreHotUpdateCanaryRequirementsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		requirements = append(requirements, status)
	}
	return requirements, true, invalid, nil
}

func loadOperatorHotUpdateCanaryRequirementStatus(root, path string) OperatorHotUpdateCanaryRequirementStatus {
	status := OperatorHotUpdateCanaryRequirementStatus{
		State:               "invalid",
		CanaryRequirementID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record HotUpdateCanaryRequirementRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeHotUpdateCanaryRequirementRecord(record)
	status = operatorHotUpdateCanaryRequirementStatusFromRecord(record)
	if err := ValidateHotUpdateCanaryRequirementRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateHotUpdateCanaryRequirementLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorHotUpdateCanaryRequirementStatusFromRecord(record HotUpdateCanaryRequirementRecord) OperatorHotUpdateCanaryRequirementStatus {
	return OperatorHotUpdateCanaryRequirementStatus{
		CanaryRequirementID:   record.CanaryRequirementID,
		ResultID:              record.ResultID,
		RunID:                 record.RunID,
		CandidateID:           record.CandidateID,
		EvalSuiteID:           record.EvalSuiteID,
		PromotionPolicyID:     record.PromotionPolicyID,
		BaselinePackID:        record.BaselinePackID,
		CandidatePackID:       record.CandidatePackID,
		EligibilityState:      record.EligibilityState,
		RequiredByPolicy:      record.RequiredByPolicy,
		OwnerApprovalRequired: record.OwnerApprovalRequired,
		RequirementState:      string(record.State),
		Reason:                record.Reason,
		CreatedAt:             formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:             record.CreatedBy,
	}
}

func loadOperatorHotUpdateCanaryEvidenceStatuses(root string) ([]OperatorHotUpdateCanaryEvidenceStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreHotUpdateCanaryEvidenceDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	evidence := make([]OperatorHotUpdateCanaryEvidenceStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorHotUpdateCanaryEvidenceStatus(root, filepath.Join(StoreHotUpdateCanaryEvidenceDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		evidence = append(evidence, status)
	}
	return evidence, true, invalid, nil
}

func loadOperatorHotUpdateCanaryEvidenceStatus(root, path string) OperatorHotUpdateCanaryEvidenceStatus {
	status := OperatorHotUpdateCanaryEvidenceStatus{
		State:            "invalid",
		CanaryEvidenceID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record HotUpdateCanaryEvidenceRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeHotUpdateCanaryEvidenceRecord(record)
	status = operatorHotUpdateCanaryEvidenceStatusFromRecord(record)
	if err := ValidateHotUpdateCanaryEvidenceRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateHotUpdateCanaryEvidenceLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorHotUpdateCanaryEvidenceStatusFromRecord(record HotUpdateCanaryEvidenceRecord) OperatorHotUpdateCanaryEvidenceStatus {
	return OperatorHotUpdateCanaryEvidenceStatus{
		CanaryEvidenceID:    record.CanaryEvidenceID,
		CanaryRequirementID: record.CanaryRequirementID,
		ResultID:            record.ResultID,
		RunID:               record.RunID,
		CandidateID:         record.CandidateID,
		EvalSuiteID:         record.EvalSuiteID,
		PromotionPolicyID:   record.PromotionPolicyID,
		BaselinePackID:      record.BaselinePackID,
		CandidatePackID:     record.CandidatePackID,
		EvidenceState:       string(record.EvidenceState),
		Passed:              record.Passed,
		Reason:              record.Reason,
		ObservedAt:          formatOperatorStatusTime(record.ObservedAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorHotUpdateCanarySatisfactionStatuses(root string) ([]OperatorHotUpdateCanarySatisfactionStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreHotUpdateCanaryRequirementsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	statuses := make([]OperatorHotUpdateCanarySatisfactionStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		requirementStatus, extraInvalid, err := loadOperatorHotUpdateCanarySatisfactionStatus(root, filepath.Join(StoreHotUpdateCanaryRequirementsDir(root), name))
		if err != nil {
			return nil, true, false, err
		}
		if requirementStatus.State == "invalid" {
			invalid = true
		}
		statuses = append(statuses, requirementStatus)
		for _, status := range extraInvalid {
			invalid = true
			statuses = append(statuses, status)
		}
	}
	sort.SliceStable(statuses, func(i, j int) bool {
		if statuses[i].CanaryRequirementID != statuses[j].CanaryRequirementID {
			return statuses[i].CanaryRequirementID < statuses[j].CanaryRequirementID
		}
		return statuses[i].SelectedCanaryEvidenceID < statuses[j].SelectedCanaryEvidenceID
	})
	return statuses, true, invalid, nil
}

func loadOperatorHotUpdateCanarySatisfactionStatus(root, path string) (OperatorHotUpdateCanarySatisfactionStatus, []OperatorHotUpdateCanarySatisfactionStatus, error) {
	status := OperatorHotUpdateCanarySatisfactionStatus{
		State:               "invalid",
		CanaryRequirementID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		SatisfactionState:   string(HotUpdateCanarySatisfactionStateInvalid),
	}

	var requirement HotUpdateCanaryRequirementRecord
	if err := LoadStoreJSON(path, &requirement); err != nil {
		status.Error = err.Error()
		return status, nil, nil
	}
	requirement = NormalizeHotUpdateCanaryRequirementRecord(requirement)
	if requirement.CanaryRequirementID != "" {
		status.CanaryRequirementID = requirement.CanaryRequirementID
	}
	status = operatorHotUpdateCanarySatisfactionStatusFromRequirement(requirement)
	if err := ValidateHotUpdateCanaryRequirementRecord(requirement); err != nil {
		status.State = "invalid"
		status.SatisfactionState = string(HotUpdateCanarySatisfactionStateInvalid)
		status.Error = err.Error()
		return status, nil, nil
	}
	if err := validateHotUpdateCanaryRequirementLinkage(root, requirement); err != nil {
		status.State = "invalid"
		status.SatisfactionState = string(HotUpdateCanarySatisfactionStateInvalid)
		status.Error = err.Error()
		return status, nil, nil
	}

	assessment, err := assessHotUpdateCanarySatisfactionForRequirement(root, requirement)
	if err != nil {
		return OperatorHotUpdateCanarySatisfactionStatus{}, nil, err
	}
	status = operatorHotUpdateCanarySatisfactionStatusFromAssessment(assessment)

	valid, invalidEvidence, err := loadHotUpdateCanarySatisfactionEvidenceCandidates(root, requirement)
	if err != nil {
		return OperatorHotUpdateCanarySatisfactionStatus{}, nil, err
	}
	if len(valid) == 0 {
		return status, nil, nil
	}
	invalidStatuses := make([]OperatorHotUpdateCanarySatisfactionStatus, 0, len(invalidEvidence))
	for _, candidate := range invalidEvidence {
		invalidStatuses = append(invalidStatuses, operatorHotUpdateCanarySatisfactionStatusFromAssessment(hotUpdateCanaryInvalidSatisfactionAssessmentFromEvidence(requirement, candidate)))
	}
	return status, invalidStatuses, nil
}

func operatorHotUpdateCanarySatisfactionStatusFromRequirement(record HotUpdateCanaryRequirementRecord) OperatorHotUpdateCanarySatisfactionStatus {
	return OperatorHotUpdateCanarySatisfactionStatus{
		CanaryRequirementID:   record.CanaryRequirementID,
		ResultID:              record.ResultID,
		RunID:                 record.RunID,
		CandidateID:           record.CandidateID,
		EvalSuiteID:           record.EvalSuiteID,
		PromotionPolicyID:     record.PromotionPolicyID,
		BaselinePackID:        record.BaselinePackID,
		CandidatePackID:       record.CandidatePackID,
		EligibilityState:      record.EligibilityState,
		OwnerApprovalRequired: record.OwnerApprovalRequired,
	}
}

func operatorHotUpdateCanarySatisfactionStatusFromAssessment(assessment HotUpdateCanarySatisfactionAssessment) OperatorHotUpdateCanarySatisfactionStatus {
	return OperatorHotUpdateCanarySatisfactionStatus{
		State:                    assessment.State,
		CanaryRequirementID:      assessment.CanaryRequirementID,
		SelectedCanaryEvidenceID: assessment.SelectedCanaryEvidenceID,
		ResultID:                 assessment.ResultID,
		RunID:                    assessment.RunID,
		CandidateID:              assessment.CandidateID,
		EvalSuiteID:              assessment.EvalSuiteID,
		PromotionPolicyID:        assessment.PromotionPolicyID,
		BaselinePackID:           assessment.BaselinePackID,
		CandidatePackID:          assessment.CandidatePackID,
		EligibilityState:         assessment.EligibilityState,
		OwnerApprovalRequired:    assessment.OwnerApprovalRequired,
		SatisfactionState:        string(assessment.SatisfactionState),
		EvidenceState:            string(assessment.EvidenceState),
		Passed:                   assessment.Passed,
		ObservedAt:               formatOperatorStatusTime(assessment.ObservedAt),
		Reason:                   assessment.Reason,
		Error:                    assessment.Error,
	}
}

func loadOperatorHotUpdateGateStatuses(root string) ([]OperatorHotUpdateGateStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreHotUpdateGatesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	gates := make([]OperatorHotUpdateGateStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorHotUpdateGateStatus(root, filepath.Join(StoreHotUpdateGatesDir(root), name))
		if status.Error != "" {
			invalid = true
		}
		gates = append(gates, status)
	}
	return gates, true, invalid, nil
}

func loadOperatorHotUpdateGateStatus(root, path string) OperatorHotUpdateGateStatus {
	status := OperatorHotUpdateGateStatus{
		HotUpdateID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record HotUpdateGateRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeHotUpdateGateRecord(record)
	status = operatorHotUpdateGateStatusFromRecord(record)
	if err := ValidateHotUpdateGateRecord(record); err != nil {
		status.Error = err.Error()
		return status
	}
	if err := validateHotUpdateGateReadModelLinkage(root, record); err != nil {
		status.Error = err.Error()
		return status
	}
	return status
}

func operatorHotUpdateGateStatusFromRecord(record HotUpdateGateRecord) OperatorHotUpdateGateStatus {
	return OperatorHotUpdateGateStatus{
		HotUpdateID:              record.HotUpdateID,
		CandidatePackID:          record.CandidatePackID,
		PreviousActivePackID:     record.PreviousActivePackID,
		RollbackTargetPackID:     record.RollbackTargetPackID,
		TargetSurfaces:           append([]string(nil), record.TargetSurfaces...),
		SurfaceClasses:           append([]string(nil), record.SurfaceClasses...),
		ReloadMode:               string(record.ReloadMode),
		CompatibilityContractRef: record.CompatibilityContractRef,
		PreparedAt:               formatOperatorStatusTime(record.PreparedAt),
		PhaseUpdatedAt:           formatOperatorStatusTime(record.PhaseUpdatedAt),
		PhaseUpdatedBy:           record.PhaseUpdatedBy,
		State:                    string(record.State),
		Decision:                 string(record.Decision),
		FailureReason:            record.FailureReason,
	}
}

func loadOperatorHotUpdateOutcomeStatuses(root string) ([]OperatorHotUpdateOutcomeStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreHotUpdateOutcomesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	outcomes := make([]OperatorHotUpdateOutcomeStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorHotUpdateOutcomeStatus(root, filepath.Join(StoreHotUpdateOutcomesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		outcomes = append(outcomes, status)
	}
	return outcomes, true, invalid, nil
}

func loadOperatorHotUpdateOutcomeStatus(root, path string) OperatorHotUpdateOutcomeStatus {
	status := OperatorHotUpdateOutcomeStatus{
		State:     "invalid",
		OutcomeID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record HotUpdateOutcomeRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeHotUpdateOutcomeRecord(record)
	status = operatorHotUpdateOutcomeStatusFromRecord(record)
	if err := ValidateHotUpdateOutcomeRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateHotUpdateOutcomeLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorHotUpdateOutcomeStatusFromRecord(record HotUpdateOutcomeRecord) OperatorHotUpdateOutcomeStatus {
	return OperatorHotUpdateOutcomeStatus{
		OutcomeID:         record.OutcomeID,
		HotUpdateID:       record.HotUpdateID,
		CandidateID:       record.CandidateID,
		RunID:             record.RunID,
		CandidateResultID: record.CandidateResultID,
		CandidatePackID:   record.CandidatePackID,
		OutcomeKind:       string(record.OutcomeKind),
		Reason:            record.Reason,
		Notes:             record.Notes,
		OutcomeAt:         formatOperatorStatusTime(record.OutcomeAt),
		CreatedAt:         formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:         record.CreatedBy,
	}
}

func validateHotUpdateGateReadModelLinkage(root string, record HotUpdateGateRecord) error {
	if _, err := LoadRuntimePackRecord(root, record.CandidatePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate candidate_pack_id %q: %w", record.CandidatePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.PreviousActivePackID); err != nil {
		return fmt.Errorf("mission store hot-update gate previous_active_pack_id %q: %w", record.PreviousActivePackID, err)
	}
	if _, err := LoadRuntimePackRecord(root, record.RollbackTargetPackID); err != nil {
		return fmt.Errorf("mission store hot-update gate rollback_target_pack_id %q: %w", record.RollbackTargetPackID, err)
	}
	return nil
}

func loadOperatorPromotionStatuses(root string) ([]OperatorPromotionStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StorePromotionsDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	promotions := make([]OperatorPromotionStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorPromotionStatus(root, filepath.Join(StorePromotionsDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		promotions = append(promotions, status)
	}
	return promotions, true, invalid, nil
}

func loadOperatorPromotionStatus(root, path string) OperatorPromotionStatus {
	status := OperatorPromotionStatus{
		State:       "invalid",
		PromotionID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record PromotionRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizePromotionRecord(record)
	status = operatorPromotionStatusFromRecord(record)
	if err := ValidatePromotionRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validatePromotionLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorPromotionStatusFromRecord(record PromotionRecord) OperatorPromotionStatus {
	return OperatorPromotionStatus{
		PromotionID:          record.PromotionID,
		PromotedPackID:       record.PromotedPackID,
		PreviousActivePackID: record.PreviousActivePackID,
		LastKnownGoodPackID:  record.LastKnownGoodPackID,
		LastKnownGoodBasis:   record.LastKnownGoodBasis,
		HotUpdateID:          record.HotUpdateID,
		OutcomeID:            record.OutcomeID,
		CandidateID:          record.CandidateID,
		RunID:                record.RunID,
		CandidateResultID:    record.CandidateResultID,
		Reason:               record.Reason,
		Notes:                record.Notes,
		PromotedAt:           formatOperatorStatusTime(record.PromotedAt),
		CreatedAt:            formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:            record.CreatedBy,
	}
}

func loadOperatorRollbackStatuses(root string) ([]OperatorRollbackStatus, bool, bool, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, true, false, err
	}

	entries, err := os.ReadDir(StoreRollbacksDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isStoreJSONDataFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, false, false, nil
	}
	sort.Strings(names)

	rollbacks := make([]OperatorRollbackStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorRollbackStatus(root, filepath.Join(StoreRollbacksDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		rollbacks = append(rollbacks, status)
	}
	return rollbacks, true, invalid, nil
}

func loadOperatorRollbackStatus(root, path string) OperatorRollbackStatus {
	status := OperatorRollbackStatus{
		State:      "invalid",
		RollbackID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record RollbackRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeRollbackRecord(record)
	status = operatorRollbackStatusFromRecord(record)
	if err := ValidateRollbackRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateRollbackLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func loadOperatorRollbackApplyStatuses(root string) ([]OperatorRollbackApplyStatus, bool, bool, error) {
	entries, err := os.ReadDir(StoreRollbackAppliesDir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, false, nil
		}
		return nil, true, false, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	applies := make([]OperatorRollbackApplyStatus, 0, len(names))
	invalid := false
	for _, name := range names {
		status := loadOperatorRollbackApplyStatus(root, filepath.Join(StoreRollbackAppliesDir(root), name))
		if status.State == "invalid" {
			invalid = true
		}
		applies = append(applies, status)
	}
	return applies, true, invalid, nil
}

func operatorRollbackStatusFromRecord(record RollbackRecord) OperatorRollbackStatus {
	return OperatorRollbackStatus{
		RollbackID:          record.RollbackID,
		PromotionID:         record.PromotionID,
		HotUpdateID:         record.HotUpdateID,
		OutcomeID:           record.OutcomeID,
		FromPackID:          record.FromPackID,
		TargetPackID:        record.TargetPackID,
		LastKnownGoodPackID: record.LastKnownGoodPackID,
		Reason:              record.Reason,
		Notes:               record.Notes,
		RollbackAt:          formatOperatorStatusTime(record.RollbackAt),
		CreatedAt:           formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:           record.CreatedBy,
	}
}

func loadOperatorRollbackApplyStatus(root, path string) OperatorRollbackApplyStatus {
	status := OperatorRollbackApplyStatus{
		State:           "invalid",
		RollbackApplyID: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	var record RollbackApplyRecord
	if err := LoadStoreJSON(path, &record); err != nil {
		status.Error = err.Error()
		return status
	}

	record = NormalizeRollbackApplyRecord(record)
	status = operatorRollbackApplyStatusFromRecord(record)
	if err := ValidateRollbackApplyRecord(record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	if err := validateRollbackApplyLinkage(root, record); err != nil {
		status.State = "invalid"
		status.Error = err.Error()
		return status
	}
	status.State = "configured"
	return status
}

func operatorRollbackApplyStatusFromRecord(record RollbackApplyRecord) OperatorRollbackApplyStatus {
	return OperatorRollbackApplyStatus{
		RollbackApplyID: record.ApplyID,
		RollbackID:      record.RollbackID,
		Phase:           string(record.Phase),
		ActivationState: string(record.ActivationState),
		CreatedAt:       formatOperatorStatusTime(record.CreatedAt),
		CreatedBy:       record.CreatedBy,
	}
}

func selectOperatorStatusApprovalRequest(runtime JobRuntimeState) (ApprovalRequest, bool) {
	var fallback *ApprovalRequest
	for i := len(runtime.ApprovalRequests) - 1; i >= 0; i-- {
		request := runtime.ApprovalRequests[i]
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}
		if runtime.ActiveStepID != "" && request.StepID == runtime.ActiveStepID {
			return request, true
		}
		if fallback == nil {
			candidate := request
			fallback = &candidate
		}
	}
	if fallback == nil {
		return ApprovalRequest{}, false
	}
	return *fallback, true
}

func selectOperatorStatusRecentAudit(runtime JobRuntimeState) []OperatorRecentAuditStatus {
	if len(runtime.AuditHistory) == 0 {
		return nil
	}

	count := OperatorStatusRecentAuditLimit
	if len(runtime.AuditHistory) < count {
		count = len(runtime.AuditHistory)
	}

	recent := make([]OperatorRecentAuditStatus, 0, count)
	for i := len(runtime.AuditHistory) - 1; i >= len(runtime.AuditHistory)-count; i-- {
		event := normalizeAuditEvent(runtime.AuditHistory[i])
		recent = append(recent, OperatorRecentAuditStatus{
			EventID:     event.EventID,
			JobID:       event.JobID,
			StepID:      event.StepID,
			Action:      event.ToolName,
			ActionClass: event.ActionClass,
			Result:      event.Result,
			Allowed:     event.Allowed,
			Code:        event.Code,
			Timestamp:   event.Timestamp.UTC().Format(time.RFC3339Nano),
		})
	}

	return recent
}

func selectOperatorStatusApprovalHistory(runtime JobRuntimeState) []OperatorApprovalHistoryEntry {
	if len(runtime.ApprovalRequests) == 0 {
		return nil
	}

	history := make([]OperatorApprovalHistoryEntry, 0, minInt(len(runtime.ApprovalRequests), OperatorStatusApprovalHistoryLimit))
	for i := len(runtime.ApprovalRequests) - 1; i >= 0 && len(history) < OperatorStatusApprovalHistoryLimit; i-- {
		request := runtime.ApprovalRequests[i]
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}

		entry := OperatorApprovalHistoryEntry{
			StepID:          request.StepID,
			RequestedAction: request.RequestedAction,
			Scope:           request.Scope,
			State:           string(request.State),
			RequestedVia:    request.RequestedVia,
			GrantedVia:      request.GrantedVia,
			SessionChannel:  request.SessionChannel,
			SessionChatID:   request.SessionChatID,
			RequestedAt:     formatOperatorStatusTime(request.RequestedAt),
			ResolvedAt:      formatOperatorStatusTime(request.ResolvedAt),
			ExpiresAt:       formatOperatorStatusTime(request.ExpiresAt),
		}
		if revokedAt := findOperatorStatusApprovalRevokedAt(runtime.ApprovalGrants, request); revokedAt != nil {
			entry.RevokedAt = revokedAt
		}

		history = append(history, entry)
	}

	if len(history) == 0 {
		return nil
	}
	return history
}

func selectOperatorStatusArtifacts(runtime JobRuntimeState) []OperatorArtifactStatus {
	candidates := collectOperatorStatusArtifactCandidates(runtime)
	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.planIndex != right.planIndex {
			return left.planIndex < right.planIndex
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StepID != right.StepID {
			return left.StepID < right.StepID
		}
		return left.StepType < right.StepType
	})

	if len(candidates) > OperatorStatusArtifactLimit {
		candidates = candidates[:OperatorStatusArtifactLimit]
	}

	artifacts := make([]OperatorArtifactStatus, len(candidates))
	for i, candidate := range candidates {
		artifacts[i] = OperatorArtifactStatus{
			StepID:   candidate.StepID,
			StepType: candidate.StepType,
			Path:     candidate.Path,
			State:    candidate.State,
		}
	}
	return artifacts
}

func selectOperatorStatusFrankZohoSendProof(runtime JobRuntimeState) []OperatorFrankZohoSendProofStatus {
	if len(runtime.FrankZohoSendReceipts) == 0 {
		return nil
	}

	proof := make([]OperatorFrankZohoSendProofStatus, 0, len(runtime.FrankZohoSendReceipts))
	for _, receipt := range runtime.FrankZohoSendReceipts {
		normalized := NormalizeFrankZohoSendReceipt(receipt)
		proof = append(proof, OperatorFrankZohoSendProofStatus{
			StepID:             normalized.StepID,
			ProviderMessageID:  normalized.ProviderMessageID,
			ProviderMailID:     normalized.ProviderMailID,
			MIMEMessageID:      normalized.MIMEMessageID,
			ProviderAccountID:  normalized.ProviderAccountID,
			OriginalMessageURL: normalized.OriginalMessageURL,
		})
	}
	return proof
}

func selectOperatorStatusCampaignZohoEmailOutbounds(runtime JobRuntimeState) []OperatorCampaignZohoEmailOutboundActionStatus {
	if len(runtime.CampaignZohoEmailOutboundActions) == 0 {
		return nil
	}

	actions := make([]OperatorCampaignZohoEmailOutboundActionStatus, 0, len(runtime.CampaignZohoEmailOutboundActions))
	for _, action := range runtime.CampaignZohoEmailOutboundActions {
		normalized := NormalizeCampaignZohoEmailOutboundAction(action)
		actions = append(actions, OperatorCampaignZohoEmailOutboundActionStatus{
			ActionID:                         normalized.ActionID,
			StepID:                           normalized.StepID,
			CampaignID:                       normalized.CampaignID,
			State:                            string(normalized.State),
			Provider:                         normalized.Provider,
			ProviderAccountID:                normalized.ProviderAccountID,
			FromAddress:                      normalized.FromAddress,
			FromDisplayName:                  normalized.FromDisplayName,
			To:                               append([]string(nil), normalized.Addressing.To...),
			CC:                               append([]string(nil), normalized.Addressing.CC...),
			BCC:                              append([]string(nil), normalized.Addressing.BCC...),
			Subject:                          normalized.Subject,
			BodyFormat:                       normalized.BodyFormat,
			BodySHA256:                       normalized.BodySHA256,
			PreparedAt:                       formatOperatorStatusTime(normalized.PreparedAt),
			SentAt:                           formatOperatorStatusTime(normalized.SentAt),
			VerifiedAt:                       formatOperatorStatusTime(normalized.VerifiedAt),
			FailedAt:                         formatOperatorStatusTime(normalized.FailedAt),
			ReplyToInboundReplyID:            normalized.ReplyToInboundReplyID,
			ReplyToOutboundActionID:          normalized.ReplyToOutboundActionID,
			ProviderMessageID:                normalized.ProviderMessageID,
			ProviderMailID:                   normalized.ProviderMailID,
			MIMEMessageID:                    normalized.MIMEMessageID,
			OriginalMessageURL:               normalized.OriginalMessageURL,
			FailureHTTPStatus:                normalized.Failure.HTTPStatus,
			FailureProviderStatusCode:        normalized.Failure.ProviderStatusCode,
			FailureProviderStatusDescription: normalized.Failure.ProviderStatusDescription,
		})
	}
	return actions
}

func selectOperatorStatusFrankZohoInboundReplies(runtime JobRuntimeState) []OperatorFrankZohoInboundReplyStatus {
	if len(runtime.FrankZohoInboundReplies) == 0 {
		return nil
	}

	replies := make([]OperatorFrankZohoInboundReplyStatus, 0, len(runtime.FrankZohoInboundReplies))
	for _, reply := range runtime.FrankZohoInboundReplies {
		normalized := NormalizeFrankZohoInboundReply(reply)
		replies = append(replies, OperatorFrankZohoInboundReplyStatus{
			ReplyID:            normalized.ReplyID,
			StepID:             normalized.StepID,
			Provider:           normalized.Provider,
			ProviderAccountID:  normalized.ProviderAccountID,
			ProviderMessageID:  normalized.ProviderMessageID,
			ProviderMailID:     normalized.ProviderMailID,
			MIMEMessageID:      normalized.MIMEMessageID,
			InReplyTo:          normalized.InReplyTo,
			References:         append([]string(nil), normalized.References...),
			FromAddress:        normalized.FromAddress,
			FromDisplayName:    normalized.FromDisplayName,
			FromAddressCount:   normalized.FromAddressCount,
			Subject:            normalized.Subject,
			ReceivedAt:         formatOperatorStatusTime(normalized.ReceivedAt),
			OriginalMessageURL: normalized.OriginalMessageURL,
		})
	}
	return replies
}

func selectOperatorStatusCampaignZohoEmailReplyWorkItems(runtime JobRuntimeState) []OperatorCampaignZohoEmailReplyWorkItemStatus {
	if len(runtime.CampaignZohoEmailReplyWorkItems) == 0 {
		return nil
	}

	latestFollowUpByInbound := latestCampaignZohoEmailFollowUpActionsByInboundReply(runtime)
	items := make([]OperatorCampaignZohoEmailReplyWorkItemStatus, 0, len(runtime.CampaignZohoEmailReplyWorkItems))
	for _, item := range runtime.CampaignZohoEmailReplyWorkItems {
		normalized := NormalizeCampaignZohoEmailReplyWorkItem(item)
		latestFollowUp, hasLatestFollowUp := latestFollowUpByInbound[normalized.InboundReplyID]
		derivedState := operatorStatusReplyWorkDerivedIterationState(normalized, latestFollowUp, hasLatestFollowUp)
		latestFollowUpActionID := ""
		latestFollowUpActionState := ""
		if hasLatestFollowUp {
			latestFollowUpActionID = latestFollowUp.ActionID
			latestFollowUpActionState = string(latestFollowUp.State)
		}
		items = append(items, OperatorCampaignZohoEmailReplyWorkItemStatus{
			ReplyWorkItemID:           normalized.ReplyWorkItemID,
			InboundReplyID:            normalized.InboundReplyID,
			CampaignID:                normalized.CampaignID,
			State:                     string(normalized.State),
			DerivedIterationState:     derivedState,
			LatestFollowUpActionID:    latestFollowUpActionID,
			LatestFollowUpActionState: latestFollowUpActionState,
			DeferredUntil:             formatOperatorStatusTime(normalized.DeferredUntil),
			ClaimedFollowUpActionID:   normalized.ClaimedFollowUpActionID,
			CreatedAt:                 formatOperatorStatusTime(normalized.CreatedAt),
			UpdatedAt:                 formatOperatorStatusTime(normalized.UpdatedAt),
		})
	}
	return items
}

func latestCampaignZohoEmailFollowUpActionsByInboundReply(runtime JobRuntimeState) map[string]CampaignZohoEmailOutboundAction {
	if len(runtime.CampaignZohoEmailOutboundActions) == 0 {
		return nil
	}

	latest := make(map[string]CampaignZohoEmailOutboundAction, len(runtime.CampaignZohoEmailOutboundActions))
	for _, action := range runtime.CampaignZohoEmailOutboundActions {
		normalized := NormalizeCampaignZohoEmailOutboundAction(action)
		if normalized.ReplyToInboundReplyID == "" {
			continue
		}
		existing, ok := latest[normalized.ReplyToInboundReplyID]
		if !ok || operatorStatusFollowUpActionSortsAfter(normalized, existing) {
			latest[normalized.ReplyToInboundReplyID] = normalized
		}
	}
	return latest
}

func operatorStatusReplyWorkDerivedIterationState(item CampaignZohoEmailReplyWorkItem, latestFollowUp CampaignZohoEmailOutboundAction, hasLatestFollowUp bool) string {
	switch item.State {
	case CampaignZohoEmailReplyWorkItemStateOpen:
		if hasLatestFollowUp && latestFollowUp.State == CampaignZohoEmailOutboundActionStateFailed {
			return "reopened_after_terminal_failure"
		}
		return "open"
	case CampaignZohoEmailReplyWorkItemStateDeferred:
		return "deferred"
	case CampaignZohoEmailReplyWorkItemStateClaimed:
		if hasLatestFollowUp && latestFollowUp.State == CampaignZohoEmailOutboundActionStateFailed {
			return "blocked_terminal_failure"
		}
		return "claimed_unresolved"
	case CampaignZohoEmailReplyWorkItemStateResponded:
		return "responded"
	case CampaignZohoEmailReplyWorkItemStateIgnored:
		return "ignored"
	default:
		return string(item.State)
	}
}

func operatorStatusFollowUpActionSortsAfter(left, right CampaignZohoEmailOutboundAction) bool {
	leftAt := operatorStatusFollowUpActionTimestamp(left)
	rightAt := operatorStatusFollowUpActionTimestamp(right)
	if !leftAt.Equal(rightAt) {
		return leftAt.After(rightAt)
	}
	leftRank := operatorStatusFollowUpActionStateRank(left.State)
	rightRank := operatorStatusFollowUpActionStateRank(right.State)
	if leftRank != rightRank {
		return leftRank > rightRank
	}
	return left.ActionID > right.ActionID
}

func operatorStatusFollowUpActionTimestamp(action CampaignZohoEmailOutboundAction) time.Time {
	normalized := NormalizeCampaignZohoEmailOutboundAction(action)
	switch normalized.State {
	case CampaignZohoEmailOutboundActionStateVerified:
		if !normalized.VerifiedAt.IsZero() {
			return normalized.VerifiedAt
		}
	case CampaignZohoEmailOutboundActionStateFailed:
		if !normalized.FailedAt.IsZero() {
			return normalized.FailedAt
		}
	case CampaignZohoEmailOutboundActionStateSent:
		if !normalized.SentAt.IsZero() {
			return normalized.SentAt
		}
	}
	return normalized.PreparedAt
}

func operatorStatusFollowUpActionStateRank(state CampaignZohoEmailOutboundActionState) int {
	switch state {
	case CampaignZohoEmailOutboundActionStateVerified:
		return 4
	case CampaignZohoEmailOutboundActionStateFailed:
		return 3
	case CampaignZohoEmailOutboundActionStateSent:
		return 2
	case CampaignZohoEmailOutboundActionStatePrepared:
		return 1
	default:
		return 0
	}
}

type operatorArtifactCandidate struct {
	StepID    string
	StepType  StepType
	Path      string
	State     string
	planIndex int
}

func collectOperatorStatusArtifactCandidates(runtime JobRuntimeState) []operatorArtifactCandidate {
	if len(runtime.CompletedSteps) == 0 {
		return nil
	}

	stepByID := make(map[string]Step, len(runtime.CompletedSteps))
	stepOrderIndex := make(map[string]int, len(runtime.CompletedSteps))
	defaultPlanIndex := len(runtime.CompletedSteps) + 1
	if runtime.InspectablePlan != nil {
		for i, step := range runtime.InspectablePlan.Steps {
			stepByID[step.ID] = copyStep(step)
			stepOrderIndex[step.ID] = i
		}
		defaultPlanIndex = len(runtime.InspectablePlan.Steps) + 1
	}

	seen := make(map[string]struct{}, len(runtime.CompletedSteps))
	candidates := make([]operatorArtifactCandidate, 0, len(runtime.CompletedSteps))
	for i := len(runtime.CompletedSteps) - 1; i >= 0; i-- {
		record := runtime.CompletedSteps[i]
		step, hasStep := stepByID[record.StepID]

		stepType, path, ok := operatorStatusArtifactRecord(record, step, hasStep)
		if !ok {
			continue
		}

		key := string(stepType) + "\x1f" + record.StepID + "\x1f" + path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		state := ""
		if record.ResultingState != nil {
			state = record.ResultingState.State
		}

		planIndex := defaultPlanIndex
		if index, ok := stepOrderIndex[record.StepID]; ok {
			planIndex = index
		}
		candidates = append(candidates, operatorArtifactCandidate{
			StepID:    record.StepID,
			StepType:  stepType,
			Path:      path,
			State:     state,
			planIndex: planIndex,
		})
	}

	return candidates
}

func operatorStatusArtifactRecord(record RuntimeStepRecord, step Step, hasStep bool) (StepType, string, bool) {
	if hasStep {
		if path, ok := operatorStatusArtifactPathForStep(step); ok {
			if record.ResultingState != nil && isOperatorStatusArtifactStepType(StepType(record.ResultingState.Kind)) {
				if target := cleanedArtifactPath(record.ResultingState.Target); target != "" {
					return step.Type, target, true
				}
			}
			return step.Type, path, true
		}
	}

	if record.ResultingState == nil {
		return "", "", false
	}

	stepType := StepType(record.ResultingState.Kind)
	if !isOperatorStatusArtifactStepType(stepType) {
		return "", "", false
	}
	path := cleanedArtifactPath(record.ResultingState.Target)
	if path == "" {
		return "", "", false
	}
	return stepType, path, true
}

func operatorStatusArtifactPathForStep(step Step) (string, bool) {
	switch step.Type {
	case StepTypeStaticArtifact:
		if path := staticArtifactPath(step); path != "" {
			return path, true
		}
	case StepTypeOneShotCode:
		if path := oneShotArtifactPath(step); path != "" {
			return path, true
		}
	case StepTypeLongRunningCode:
		if path := longRunningArtifactPath(step); path != "" {
			return path, true
		}
	}
	return "", false
}

func isOperatorStatusArtifactStepType(stepType StepType) bool {
	switch stepType {
	case StepTypeStaticArtifact, StepTypeOneShotCode, StepTypeLongRunningCode:
		return true
	default:
		return false
	}
}

func buildOperatorStatusTruncation(runtime JobRuntimeState, shownApprovalHistory int, shownRecentAudit int, shownArtifacts int) *OperatorStatusTruncation {
	truncation := OperatorStatusTruncation{}

	approvalHistoryTotal := countOperatorStatusApprovalHistory(runtime)
	if approvalHistoryTotal > shownApprovalHistory {
		truncation.ApprovalHistoryOmitted = approvalHistoryTotal - shownApprovalHistory
	}
	if len(runtime.AuditHistory) > shownRecentAudit {
		truncation.RecentAuditOmitted = len(runtime.AuditHistory) - shownRecentAudit
	}
	if artifactsTotal := len(collectOperatorStatusArtifactCandidates(runtime)); artifactsTotal > shownArtifacts {
		truncation.ArtifactsOmitted = artifactsTotal - shownArtifacts
	}
	if truncation.ApprovalHistoryOmitted == 0 && truncation.RecentAuditOmitted == 0 && truncation.ArtifactsOmitted == 0 {
		return nil
	}
	return &truncation
}

func countOperatorStatusApprovalHistory(runtime JobRuntimeState) int {
	count := 0
	for _, request := range runtime.ApprovalRequests {
		if runtime.JobID != "" && request.JobID != runtime.JobID {
			continue
		}
		count++
	}
	return count
}

func findOperatorStatusApprovalRevokedAt(grants []ApprovalGrant, request ApprovalRequest) *string {
	if request.State != ApprovalStateRevoked {
		return nil
	}
	if revokedAt := formatOperatorStatusTime(legacyApprovalRequestRevokedAt(request, grants)); revokedAt != nil {
		return revokedAt
	}
	return nil
}

func formatOperatorStatusTime(at time.Time) *string {
	if at.IsZero() {
		return nil
	}
	formatted := at.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
