package missioncontrol

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	RejectionCodeDuplicateStepID               RejectionCode = "duplicate_step_id"
	RejectionCodeInvalidGovernedExternalTarget RejectionCode = "invalid_governed_external_target"
	RejectionCodeInvalidFrankObjectRef         RejectionCode = "invalid_frank_object_ref"
	RejectionCodeInvalidCampaignRef            RejectionCode = "invalid_campaign_ref"
	RejectionCodeInvalidTreasuryRef            RejectionCode = "invalid_treasury_ref"
	RejectionCodeInvalidCapabilityRequirement  RejectionCode = "invalid_capability_requirement"
	RejectionCodeInvalidCapabilityProposalRef  RejectionCode = "invalid_capability_onboarding_proposal_ref"
	RejectionCodeMissingCapabilityProposal     RejectionCode = "missing_capability_onboarding_proposal"
	RejectionCodeInvalidIdentityMode           RejectionCode = "invalid_identity_mode"
	RejectionCodeInvalidExecutionPlane         RejectionCode = "invalid_execution_plane"
	RejectionCodeInvalidExecutionHost          RejectionCode = "invalid_execution_host"
	RejectionCodeInvalidMissionFamily          RejectionCode = "invalid_mission_family"
	RejectionCodeMissingDependencyTarget       RejectionCode = "missing_dependency_target"
	RejectionCodeDependencyCycle               RejectionCode = "dependency_cycle"
	RejectionCodeMissingTerminalFinalStep      RejectionCode = "missing_terminal_final_response"
	RejectionCodeInvalidStepType               RejectionCode = "invalid_step_type"
	RejectionCodeLongRunningStartForbidden     RejectionCode = "longrun_start_forbidden"
)

const (
	RejectionCodeV4ExecutionPlaneRequired           RejectionCode = "E_EXECUTION_PLANE_REQUIRED"
	RejectionCodeV4ExecutionHostRequired            RejectionCode = "E_EXECUTION_HOST_REQUIRED"
	RejectionCodeV4ImprovementWorkspaceRequired     RejectionCode = "E_IMPROVEMENT_WORKSPACE_REQUIRED"
	RejectionCodeV4HotUpdateGateRequired            RejectionCode = "E_HOT_UPDATE_GATE_REQUIRED"
	RejectionCodeV4BaselineRequired                 RejectionCode = "E_BASELINE_REQUIRED"
	RejectionCodeV4HoldoutRequired                  RejectionCode = "E_HOLDOUT_REQUIRED"
	RejectionCodeV4SmokeCheckRequired               RejectionCode = "E_SMOKE_CHECK_REQUIRED"
	RejectionCodeV4EvalImmutable                    RejectionCode = "E_EVAL_IMMUTABLE"
	RejectionCodeV4MutationScopeViolation           RejectionCode = "E_MUTATION_SCOPE_VIOLATION"
	RejectionCodeV4SurfaceClassRequired             RejectionCode = "E_SURFACE_CLASS_REQUIRED"
	RejectionCodeV4ForbiddenSurfaceChange           RejectionCode = "E_FORBIDDEN_SURFACE_CHANGE"
	RejectionCodeV4TopologyChangeDisabled           RejectionCode = "E_TOPOLOGY_CHANGE_DISABLED"
	RejectionCodeV4PromotionPolicyRequired          RejectionCode = "E_PROMOTION_POLICY_REQUIRED"
	RejectionCodeV4HotUpdatePolicyRequired          RejectionCode = "E_HOT_UPDATE_POLICY_REQUIRED"
	RejectionCodeV4CanaryRequired                   RejectionCode = "E_CANARY_REQUIRED"
	RejectionCodeV4PromotionApprovalRequired        RejectionCode = "E_PROMOTION_APPROVAL_REQUIRED"
	RejectionCodeV4HotUpdateApprovalRequired        RejectionCode = "E_HOT_UPDATE_APPROVAL_REQUIRED"
	RejectionCodeV4ActiveJobDeployLock              RejectionCode = "E_ACTIVE_JOB_DEPLOY_LOCK"
	RejectionCodeV4PackNotFound                     RejectionCode = "E_PACK_NOT_FOUND"
	RejectionCodeV4LastKnownGoodRequired            RejectionCode = "E_LAST_KNOWN_GOOD_REQUIRED"
	RejectionCodeV4CanaryFailed                     RejectionCode = "E_CANARY_FAILED"
	RejectionCodeV4SmokeCheckFailed                 RejectionCode = "E_SMOKE_CHECK_FAILED"
	RejectionCodeV4RollbackRequired                 RejectionCode = "E_ROLLBACK_REQUIRED"
	RejectionCodeV4PromotionAlreadyApplied          RejectionCode = "E_PROMOTION_ALREADY_APPLIED"
	RejectionCodeV4HotUpdateAlreadyApplied          RejectionCode = "E_HOT_UPDATE_ALREADY_APPLIED"
	RejectionCodeV4ReloadModeUnsupported            RejectionCode = "E_RELOAD_MODE_UNSUPPORTED"
	RejectionCodeV4ReloadQuiesceFailed              RejectionCode = "E_RELOAD_QUIESCE_FAILED"
	RejectionCodeV4ExtensionCompatibilityRequired   RejectionCode = "E_EXTENSION_COMPATIBILITY_REQUIRED"
	RejectionCodeV4ExtensionPermissionWidening      RejectionCode = "E_EXTENSION_PERMISSION_WIDENING"
	RejectionCodeV4RuntimeSourceMutationForbidden   RejectionCode = "E_RUNTIME_SOURCE_MUTATION_FORBIDDEN"
	RejectionCodeV4PolicyMutationForbidden          RejectionCode = "E_POLICY_MUTATION_FORBIDDEN"
	RejectionCodeV4ActivePackAdhocMutationForbidden RejectionCode = "E_ACTIVE_PACK_ADHOC_MUTATION_FORBIDDEN"
	RejectionCodeV4AutonomyEnvelopeRequired         RejectionCode = "E_AUTONOMY_ENVELOPE_REQUIRED"
	RejectionCodeV4StandingDirectiveRequired        RejectionCode = "E_STANDING_DIRECTIVE_REQUIRED"
	RejectionCodeV4AutonomyBudgetExceeded           RejectionCode = "E_AUTONOMY_BUDGET_EXCEEDED"
	RejectionCodeV4NoEligibleAutonomousAction       RejectionCode = "E_NO_ELIGIBLE_AUTONOMOUS_ACTION"
	RejectionCodeV4AutonomyPaused                   RejectionCode = "E_AUTONOMY_PAUSED"
	RejectionCodeV4ExternalActionLimitReached       RejectionCode = "E_EXTERNAL_ACTION_LIMIT_REACHED"
	RejectionCodeV4RepeatedFailurePause             RejectionCode = "E_REPEATED_FAILURE_PAUSE"
	RejectionCodeV4PackageAuthorityGrantForbidden   RejectionCode = "E_PACKAGE_AUTHORITY_GRANT_FORBIDDEN"

	RejectionCodeV4LabOnlyFamily              RejectionCode = "E_LAB_ONLY_FAMILY"
	RejectionCodeV4MissionFamilyRequired      RejectionCode = "E_MISSION_FAMILY_REQUIRED"
	RejectionCodeV4TrainRequired              RejectionCode = "E_TRAIN_REQUIRED"
	RejectionCodeV4ExecutionPlaneUnknown      RejectionCode = "E_EXECUTION_PLANE_UNKNOWN"
	RejectionCodeV4ExecutionHostUnknown       RejectionCode = "E_EXECUTION_HOST_UNKNOWN"
	RejectionCodeV4MissionFamilyUnknown       RejectionCode = "E_MISSION_FAMILY_UNKNOWN"
	RejectionCodeV4ExecutionPlaneIncompatible RejectionCode = "E_EXECUTION_PLANE_INCOMPATIBLE"
	RejectionCodeV4LivePhoneSelfEditForbidden RejectionCode = "E_LIVE_PHONE_SELF_EDIT_FORBIDDEN"
)

func ValidatePlan(job Job) []ValidationError {
	steps := job.Plan.Steps
	if len(steps) == 0 {
		return []ValidationError{
			{
				Code:    RejectionCodeMissingTerminalFinalStep,
				Message: "plan must include a terminal final_response step",
			},
		}
	}

	executionMetadataErrors := validateV4ExecutionMetadata(job)

	firstIndexByID := make(map[string]int, len(steps))
	duplicateSeen := make(map[string]bool, len(steps))
	duplicateErrors := make([]ValidationError, 0)
	uniqueSteps := make(map[string]Step, len(steps))
	stepOrder := make([]string, 0, len(steps))

	for i, step := range steps {
		if firstIndex, seen := firstIndexByID[step.ID]; seen {
			if !duplicateSeen[step.ID] {
				duplicateSeen[step.ID] = true
				duplicateErrors = append(duplicateErrors, ValidationError{
					Code:    RejectionCodeDuplicateStepID,
					StepID:  step.ID,
					Message: "duplicate step ID also used at index " + strconv.Itoa(firstIndex),
				})
			}
			continue
		}

		firstIndexByID[step.ID] = i
		uniqueSteps[step.ID] = step
		stepOrder = append(stepOrder, step.ID)
	}

	allowedTools := make(map[string]struct{}, len(job.AllowedTools))
	for _, tool := range job.AllowedTools {
		allowedTools[tool] = struct{}{}
	}

	dependentsByID := make(map[string]int, len(uniqueSteps))
	missingDependencyErrors := make([]ValidationError, 0)
	invalidTypeErrors := make([]ValidationError, 0)
	authorityErrors := make([]ValidationError, 0)
	toolScopeErrors := make([]ValidationError, 0)

	maxAuthority, maxAuthorityOK := authorityRank(job.MaxAuthority)
	storeRoot := strings.TrimSpace(job.MissionStoreRoot)

	for _, step := range steps {
		if !isValidStepType(step.Type) {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "step type must be one of discussion, static_artifact, one_shot_code, long_running_code, system_action, wait_user, final_response",
			})
		}
		if isV2OnlyStepType(step.Type) && job.SpecVersion != JobSpecVersionV2 {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: `step type "` + string(step.Type) + `" requires job spec_version frank_v2`,
			})
		}
		if step.Type == StepTypeWaitUser && !isValidWaitUserSubtype(step.Subtype) {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "wait_user step requires blocker, authorization, or definition subtype",
			})
		}
		if job.SpecVersion == JobSpecVersionV2 && step.Type == StepTypeStaticArtifact && staticArtifactPath(step) == "" {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "static_artifact step requires explicit static_artifact_path metadata for frank_v2",
			})
		}
		if job.SpecVersion == JobSpecVersionV2 && step.Type == StepTypeStaticArtifact && staticArtifactFormat(step) == "" {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "static_artifact step requires explicit static_artifact_format metadata for frank_v2",
			})
		}
		if job.SpecVersion == JobSpecVersionV2 && step.Type == StepTypeOneShotCode && oneShotArtifactPath(step) == "" {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "one_shot_code step requires explicit one_shot_artifact_path metadata for frank_v2",
			})
		}
		if step.Type == StepTypeLongRunningCode && hasLongRunningStartIntent(step) {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeLongRunningStartForbidden,
				StepID:  step.ID,
				Message: "long_running_code must not start a process; move start/stop semantics to system_action",
			})
		}
		if step.Type == StepTypeLongRunningCode && !hasLongRunningStartupCommand(step) {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "long_running_code step requires explicit long_running_startup_command metadata",
			})
		}
		if step.Type == StepTypeLongRunningCode && longRunningArtifactPath(step) == "" {
			invalidTypeErrors = append(invalidTypeErrors, ValidationError{
				Code:    RejectionCodeInvalidStepType,
				StepID:  step.ID,
				Message: "long_running_code step requires explicit long_running_artifact_path metadata",
			})
		}
		if step.Type == StepTypeSystemAction {
			invalidTypeErrors = append(invalidTypeErrors, validateSystemActionStep(job, step)...)
		}
		invalidTypeErrors = append(invalidTypeErrors, validateIdentityModeDeclaration(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateGovernedExternalTargets(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateFrankObjectRefs(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateCampaignRefDeclaration(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateTreasuryRefDeclaration(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateCapabilityRequirementDeclaration(step)...)
		invalidTypeErrors = append(invalidTypeErrors, validateCapabilityOnboardingProposalDeclaration(step, storeRoot)...)

		if step.RequiredAuthority != "" {
			requiredAuthority, requiredAuthorityOK := authorityRank(step.RequiredAuthority)
			if !requiredAuthorityOK || !maxAuthorityOK || requiredAuthority > maxAuthority {
				authorityErrors = append(authorityErrors, ValidationError{
					Code:    RejectionCodeAuthorityExceeded,
					StepID:  step.ID,
					Message: "step required authority exceeds job max authority",
				})
			}
		}

		for _, tool := range step.AllowedTools {
			if _, ok := allowedTools[tool]; ok {
				continue
			}

			toolScopeErrors = append(toolScopeErrors, ValidationError{
				Code:    RejectionCodeToolNotAllowed,
				StepID:  step.ID,
				Message: "step tool is not allowed by job tool scope: " + tool,
			})
		}

		for _, dependencyID := range step.DependsOn {
			if _, ok := uniqueSteps[dependencyID]; !ok {
				missingDependencyErrors = append(missingDependencyErrors, ValidationError{
					Code:    RejectionCodeMissingDependencyTarget,
					StepID:  step.ID,
					Message: "missing dependency target: " + dependencyID,
				})
				continue
			}

			dependentsByID[dependencyID]++
		}
	}

	cycleErrors := findCycleErrors(stepOrder, uniqueSteps)

	hasTerminalFinalResponse := false
	for _, step := range steps {
		if step.Type == StepTypeFinalResponse && dependentsByID[step.ID] == 0 {
			hasTerminalFinalResponse = true
			break
		}
	}

	terminalErrors := make([]ValidationError, 0, 1)
	if !hasTerminalFinalResponse {
		terminalErrors = append(terminalErrors, ValidationError{
			Code:    RejectionCodeMissingTerminalFinalStep,
			Message: "plan must include a terminal final_response step",
		})
	}

	errors := make([]ValidationError, 0, len(executionMetadataErrors)+len(duplicateErrors)+len(missingDependencyErrors)+len(cycleErrors)+len(terminalErrors)+len(invalidTypeErrors)+len(authorityErrors)+len(toolScopeErrors))
	errors = append(errors, executionMetadataErrors...)
	errors = append(errors, duplicateErrors...)
	errors = append(errors, missingDependencyErrors...)
	errors = append(errors, cycleErrors...)
	errors = append(errors, terminalErrors...)
	errors = append(errors, invalidTypeErrors...)
	errors = append(errors, authorityErrors...)
	errors = append(errors, toolScopeErrors...)
	return errors
}

func validateV4ExecutionMetadata(job Job) []ValidationError {
	if strings.TrimSpace(job.SpecVersion) != JobSpecVersionV4 {
		return nil
	}

	plane := strings.TrimSpace(job.ExecutionPlane)
	host := strings.TrimSpace(job.ExecutionHost)
	family := strings.TrimSpace(job.MissionFamily)
	promotionPolicyID := strings.TrimSpace(job.PromotionPolicyID)
	errors := make([]ValidationError, 0, 3)

	if plane == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ExecutionPlaneRequired,
			Message: "frank_v4 job requires execution_plane",
		})
	} else if !isKnownExecutionPlane(plane) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ExecutionPlaneUnknown,
			Message: "execution_plane must be live_runtime, improvement_workspace, or hot_update_gate",
		})
	}

	if host == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ExecutionHostRequired,
			Message: "frank_v4 job requires execution_host",
		})
	} else if !isKnownExecutionHost(host) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ExecutionHostUnknown,
			Message: "execution_host must be phone, desktop, workspace, desktop_dev, or remote_provider",
		})
	}

	if family == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4MissionFamilyRequired,
			Message: "frank_v4 job requires mission_family",
		})
	} else if !isKnownMissionFamily(family) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4MissionFamilyUnknown,
			Message: "mission_family is not recognized for frank_v4",
		})
	} else if plane != "" && isKnownExecutionPlane(plane) {
		if want, ok := requiredExecutionPlaneForMissionFamily(family); ok && plane != want {
			errors = append(errors, ValidationError{
				Code:    v4ExecutionPlaneIncompatibilityCodeForFamily(family),
				Message: fmt.Sprintf("mission_family %q requires execution_plane %q", family, want),
			})
		}
	}
	if isImprovementMissionFamily(family) && plane == ExecutionPlaneImprovementWorkspace && host != "" && isKnownExecutionHost(host) && !isImprovementWorkspaceExecutionHost(host) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ImprovementWorkspaceRequired,
			Message: fmt.Sprintf("mission_family %q requires execution_host phone, desktop_dev, or workspace when execution_plane is improvement_workspace", family),
		})
	}
	if isImprovementMissionFamily(family) {
		errors = append(errors, validateV4PromotionPolicyReference(promotionPolicyID, job.MissionStoreRoot)...)
		errors = append(errors, validateV4EvidenceReferences(job)...)
	}
	errors = append(errors, validateV4ImprovementTargetSurfaces(job, plane, host, family)...)

	return errors
}

func validateV4PromotionPolicyReference(promotionPolicyID, storeRoot string) []ValidationError {
	promotionPolicyID = strings.TrimSpace(promotionPolicyID)
	if promotionPolicyID == "" {
		return []ValidationError{
			{
				Code:    RejectionCodeV4PromotionPolicyRequired,
				Message: "improvement-family job requires promotion_policy_id",
			},
		}
	}
	if err := ValidatePromotionPolicyRef(PromotionPolicyRef{PromotionPolicyID: promotionPolicyID}); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeV4PromotionPolicyRequired,
				Message: err.Error(),
			},
		}
	}

	storeRoot = strings.TrimSpace(storeRoot)
	if storeRoot == "" {
		return nil
	}
	if _, err := LoadPromotionPolicyRecord(storeRoot, promotionPolicyID); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeV4PromotionPolicyRequired,
				Message: fmt.Sprintf("promotion_policy_id %q is not registered", promotionPolicyID),
			},
		}
	}
	return nil
}

func validateV4EvidenceReferences(job Job) []ValidationError {
	baselineRef := strings.TrimSpace(job.BaselineRef)
	trainRef := strings.TrimSpace(job.TrainRef)
	holdoutRef := strings.TrimSpace(job.HoldoutRef)
	errors := make([]ValidationError, 0, 4)

	if baselineRef == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4BaselineRequired,
			Message: "improvement-family job requires baseline_ref",
		})
	}
	if trainRef == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4TrainRequired,
			Message: "improvement-family job requires train_ref",
		})
	}
	if holdoutRef == "" {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4HoldoutRequired,
			Message: "improvement-family job requires holdout_ref",
		})
	}
	if trainRef != "" && holdoutRef != "" && trainRef == holdoutRef {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4MutationScopeViolation,
			Message: "train_ref and holdout_ref must be distinct",
		})
	}

	return errors
}

func validateV4ImprovementTargetSurfaces(job Job, plane, host, family string) []ValidationError {
	if !isImprovementMissionFamily(family) || plane != ExecutionPlaneImprovementWorkspace || !isKnownExecutionHost(host) || !isImprovementWorkspaceExecutionHost(host) {
		return nil
	}

	errors := make([]ValidationError, 0, 4)
	if len(job.TargetSurfaces) == 0 && len(job.MutableSurfaces) == 0 {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4MutationScopeViolation,
			Message: "improvement-family job requires at least one target_surfaces or mutable_surfaces entry",
		})
	}
	if len(job.ImmutableSurfaces) == 0 {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4ForbiddenSurfaceChange,
			Message: "improvement-family job requires immutable_surfaces",
		})
	}

	errors = append(errors, validateV4JobSurfaceRefs("target_surfaces", job.TargetSurfaces, RejectionCodeV4MutationScopeViolation)...)
	errors = append(errors, validateV4JobSurfaceRefs("mutable_surfaces", job.MutableSurfaces, RejectionCodeV4MutationScopeViolation)...)
	errors = append(errors, validateV4JobSurfaceRefs("immutable_surfaces", job.ImmutableSurfaces, RejectionCodeV4ForbiddenSurfaceChange)...)

	if family == MissionFamilyImproveTopology && !hasV4JobSurfaceClass(job.TargetSurfaces, job.MutableSurfaces, JobSurfaceClassSkillTopology) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4MutationScopeViolation,
			Message: fmt.Sprintf("mission_family %q requires target surface class %q", family, JobSurfaceClassSkillTopology),
		})
	}
	if family == MissionFamilyImproveTopology && !job.TopologyModeEnabled {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4TopologyChangeDisabled,
			Message: fmt.Sprintf("mission_family %q requires topology_mode_enabled=true", family),
		})
	}
	if family == MissionFamilyProposeSourcePatch {
		errors = append(errors, validateV4SourcePatchArtifactOnlySurfaces(job.TargetSurfaces, "target_surfaces")...)
		errors = append(errors, validateV4SourcePatchArtifactOnlySurfaces(job.MutableSurfaces, "mutable_surfaces")...)
	}
	if family == MissionFamilyProposeSourcePatch && !hasV4JobSurfaceClass(job.TargetSurfaces, job.MutableSurfaces, JobSurfaceClassSourcePatchArtifact) {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4RuntimeSourceMutationForbidden,
			Message: fmt.Sprintf("mission_family %q requires target surface class %q", family, JobSurfaceClassSourcePatchArtifact),
		})
	}

	return errors
}

func validateV4SourcePatchArtifactOnlySurfaces(refs []JobSurfaceRef, field string) []ValidationError {
	errors := make([]ValidationError, 0)
	for i, raw := range refs {
		ref := NormalizeJobSurfaceRef(raw)
		if ref.Class == "" || !isKnownJobSurfaceClass(ref.Class) || ref.Class == JobSurfaceClassSourcePatchArtifact {
			continue
		}
		errors = append(errors, ValidationError{
			Code:    RejectionCodeV4RuntimeSourceMutationForbidden,
			Message: fmt.Sprintf("mission_family %q requires %s[%d].class %q to be %q", MissionFamilyProposeSourcePatch, field, i, ref.Class, JobSurfaceClassSourcePatchArtifact),
		})
	}
	return errors
}

func validateV4JobSurfaceRefs(field string, refs []JobSurfaceRef, duplicateCode RejectionCode) []ValidationError {
	errors := make([]ValidationError, 0)
	seen := make(map[string]int, len(refs))
	for i, raw := range refs {
		ref := NormalizeJobSurfaceRef(raw)
		if ref.Class == "" {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeV4SurfaceClassRequired,
				Message: fmt.Sprintf("%s[%d].class is required", field, i),
			})
			continue
		}
		if !isKnownJobSurfaceClass(ref.Class) {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeV4SurfaceClassRequired,
				Message: fmt.Sprintf("%s[%d].class %q is not recognized", field, i, ref.Class),
			})
		}
		if ref.Ref == "" {
			errors = append(errors, ValidationError{
				Code:    duplicateCode,
				Message: fmt.Sprintf("%s[%d].ref is required", field, i),
			})
			continue
		}
		key := ref.Class + "\x00" + ref.Ref
		if firstIndex, ok := seen[key]; ok {
			errors = append(errors, ValidationError{
				Code:    duplicateCode,
				Message: fmt.Sprintf("%s contains duplicate surface ref %q first declared at index %d", field, ref.Ref, firstIndex),
			})
			continue
		}
		seen[key] = i
	}
	return errors
}

func hasV4JobSurfaceClass(targetSurfaces, mutableSurfaces []JobSurfaceRef, class string) bool {
	for _, ref := range targetSurfaces {
		if NormalizeJobSurfaceRef(ref).Class == class {
			return true
		}
	}
	for _, ref := range mutableSurfaces {
		if NormalizeJobSurfaceRef(ref).Class == class {
			return true
		}
	}
	return false
}

func v4ExecutionPlaneIncompatibilityCodeForFamily(family string) RejectionCode {
	if want, ok := requiredExecutionPlaneForMissionFamily(family); ok && want == ExecutionPlaneImprovementWorkspace {
		return RejectionCodeV4LabOnlyFamily
	}
	return RejectionCodeV4ExecutionPlaneIncompatible
}

func isKnownExecutionPlane(plane string) bool {
	switch plane {
	case ExecutionPlaneLiveRuntime, ExecutionPlaneImprovementWorkspace, ExecutionPlaneHotUpdateGate:
		return true
	default:
		return false
	}
}

func isKnownExecutionHost(host string) bool {
	switch host {
	case ExecutionHostPhone, ExecutionHostDesktop, ExecutionHostWorkspace, ExecutionHostDesktopDev, ExecutionHostRemoteProvider:
		return true
	default:
		return false
	}
}

func isKnownJobSurfaceClass(class string) bool {
	switch class {
	case JobSurfaceClassPromptPack,
		JobSurfaceClassSkill,
		JobSurfaceClassManifestEntry,
		JobSurfaceClassSkillTopology,
		JobSurfaceClassSourcePatchArtifact:
		return true
	default:
		return false
	}
}

func isKnownMissionFamily(family string) bool {
	if _, ok := requiredExecutionPlaneForMissionFamily(family); ok {
		return true
	}
	return false
}

func isImprovementMissionFamily(family string) bool {
	switch family {
	case MissionFamilyImprovePromptpack,
		MissionFamilyImproveSkills,
		MissionFamilyImproveRoutingManifest,
		MissionFamilyImproveRuntimeExtension,
		MissionFamilyEvaluateCandidate,
		MissionFamilyPromoteCandidate,
		MissionFamilyRollbackCandidate,
		MissionFamilyImproveTopology,
		MissionFamilyProposeSourcePatch:
		return true
	default:
		return false
	}
}

func isImprovementWorkspaceExecutionHost(host string) bool {
	switch host {
	case ExecutionHostPhone, ExecutionHostDesktopDev, ExecutionHostWorkspace:
		return true
	default:
		return false
	}
}

func requiredExecutionPlaneForMissionFamily(family string) (string, bool) {
	switch family {
	case MissionFamilyBuild,
		MissionFamilyResearch,
		MissionFamilyMonitor,
		MissionFamilyOperate,
		MissionFamilyMaintenance,
		MissionFamilyOutreach,
		MissionFamilyCommunityDiscovery,
		MissionFamilyOpportunityScan,
		MissionFamilyBootstrapRevenue,
		MissionFamilyBootstrapIdentityAndAccounts,
		MissionFamilyContinuousAutonomyTick,
		MissionFamilyStandingDirectiveReview,
		MissionFamilyAutonomousMissionProposal,
		MissionFamilyAutonomyBudgetReport,
		MissionFamilyAutonomyPause,
		MissionFamilyAutonomyResume:
		return ExecutionPlaneLiveRuntime, true
	case MissionFamilyImprovePromptpack,
		MissionFamilyImproveSkills,
		MissionFamilyImproveRoutingManifest,
		MissionFamilyImproveRuntimeExtension,
		MissionFamilyEvaluateCandidate,
		MissionFamilyPromoteCandidate,
		MissionFamilyRollbackCandidate,
		MissionFamilyImproveTopology,
		MissionFamilyProposeSourcePatch:
		return ExecutionPlaneImprovementWorkspace, true
	case MissionFamilyPrepareHotUpdate,
		MissionFamilyValidateHotUpdate,
		MissionFamilyStageHotUpdate,
		MissionFamilyApplyHotUpdate,
		MissionFamilySmokeTestHotUpdate,
		MissionFamilyCanaryHotUpdate,
		MissionFamilyCommitHotUpdate,
		MissionFamilyRollbackHotUpdate:
		return ExecutionPlaneHotUpdateGate, true
	default:
		return "", false
	}
}

func validateCapabilityRequirementDeclaration(step Step) []ValidationError {
	errors := make([]ValidationError, 0, 2)
	if err := validateCapabilityRequirementStrings(step.RequiredCapabilities, "required_capabilities"); err != nil {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidCapabilityRequirement,
			StepID:  step.ID,
			Message: err.Error(),
		})
	}
	if err := validateCapabilityRequirementStrings(step.RequiredDataDomains, "required_data_domains"); err != nil {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeInvalidCapabilityRequirement,
			StepID:  step.ID,
			Message: err.Error(),
		})
	}
	return errors
}

func validateCapabilityRequirementStrings(values []string, field string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s entries must be non-empty", field)
		}
	}
	return nil
}

func validateCapabilityOnboardingProposalDeclaration(step Step, storeRoot string) []ValidationError {
	requiredCapabilities := NormalizeStepRequiredCapabilities(step.RequiredCapabilities)
	requiredDataDomains := NormalizeStepRequiredDataDomains(step.RequiredDataDomains)
	requiresProposal := len(requiredCapabilities) > 0 || len(requiredDataDomains) > 0

	if step.CapabilityOnboardingProposalRef == nil {
		if !requiresProposal {
			return nil
		}
		return []ValidationError{
			{
				Code:    RejectionCodeMissingCapabilityProposal,
				StepID:  step.ID,
				Message: "capability onboarding proposal ref is required when step declares required capabilities or data domains",
			},
		}
	}

	normalizedRef := NormalizeCapabilityOnboardingProposalRef(*step.CapabilityOnboardingProposalRef)
	if err := ValidateCapabilityOnboardingProposalRef(normalizedRef); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidCapabilityProposalRef,
				StepID:  step.ID,
				Message: err.Error(),
			},
		}
	}
	if strings.TrimSpace(storeRoot) == "" {
		return []ValidationError{
			{
				Code:    RejectionCodeMissingCapabilityProposal,
				StepID:  step.ID,
				Message: "mission store root is required to resolve capability onboarding proposal refs",
			},
		}
	}

	record, err := ResolveCapabilityOnboardingProposalRef(storeRoot, normalizedRef)
	if err != nil {
		if errors.Is(err, ErrCapabilityOnboardingProposalRecordNotFound) {
			return []ValidationError{
				{
					Code:    RejectionCodeMissingCapabilityProposal,
					StepID:  step.ID,
					Message: fmt.Sprintf("capability onboarding proposal %q not found", normalizedRef.ProposalID),
				},
			}
		}
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidCapabilityProposalRef,
				StepID:  step.ID,
				Message: err.Error(),
			},
		}
	}
	if !CapabilityOnboardingProposalStateValidForPlan(record.State) {
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidCapabilityProposalRef,
				StepID:  step.ID,
				Message: fmt.Sprintf("capability onboarding proposal %q state %q is not valid for plan validation", record.ProposalID, record.State),
			},
		}
	}
	return nil
}

func validateIdentityModeDeclaration(step Step) []ValidationError {
	if err := validateIdentityMode(step.IdentityMode); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidIdentityMode,
				StepID:  step.ID,
				Message: err.Error(),
			},
		}
	}
	return nil
}

func validateGovernedExternalTargets(step Step) []ValidationError {
	if len(step.GovernedExternalTargets) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(step.GovernedExternalTargets))
	errors := make([]ValidationError, 0)
	for _, target := range step.GovernedExternalTargets {
		if err := validateAutonomyEligibilityTargetRef(target); err != nil {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeInvalidGovernedExternalTarget,
				StepID:  step.ID,
				Message: "governed external target is invalid: " + err.Error(),
			})
			continue
		}

		key := normalizedGovernedExternalTargetKey(target)
		if _, ok := seen[key]; ok {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeInvalidGovernedExternalTarget,
				StepID:  step.ID,
				Message: fmt.Sprintf("duplicate governed external target kind %q registry_id %q", target.Kind, strings.TrimSpace(target.RegistryID)),
			})
			continue
		}
		seen[key] = struct{}{}
	}

	return errors
}

func normalizedGovernedExternalTargetKey(target AutonomyEligibilityTargetRef) string {
	return string(target.Kind) + "\x1f" + strings.TrimSpace(target.RegistryID)
}

func validateFrankRegistryObjectKind(kind FrankRegistryObjectKind) error {
	switch NormalizeFrankRegistryObjectKind(kind) {
	case FrankRegistryObjectKindIdentity, FrankRegistryObjectKindAccount, FrankRegistryObjectKindContainer:
		return nil
	default:
		//nolint:staticcheck // Preserve established operator/test error text.
		return fmt.Errorf("Frank object ref kind %q is invalid", strings.TrimSpace(string(kind)))
	}
}

func validateFrankRegistryObjectRef(ref FrankRegistryObjectRef) error {
	ref = NormalizeFrankRegistryObjectRef(ref)
	if err := validateFrankRegistryObjectKind(ref.Kind); err != nil {
		return err
	}
	if ref.ObjectID == "" {
		//nolint:staticcheck // Preserve established operator/test error text.
		return fmt.Errorf("Frank object ref object_id is required")
	}
	return nil
}

func validateFrankObjectRefs(step Step) []ValidationError {
	if len(step.FrankObjectRefs) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(step.FrankObjectRefs))
	errors := make([]ValidationError, 0)
	for _, ref := range step.FrankObjectRefs {
		normalized := NormalizeFrankRegistryObjectRef(ref)
		if err := validateFrankRegistryObjectRef(normalized); err != nil {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeInvalidFrankObjectRef,
				StepID:  step.ID,
				Message: "Frank object ref is invalid: " + err.Error(),
			})
			continue
		}

		key := normalizedFrankRegistryObjectRefKey(normalized)
		if _, ok := seen[key]; ok {
			errors = append(errors, ValidationError{
				Code:    RejectionCodeInvalidFrankObjectRef,
				StepID:  step.ID,
				Message: fmt.Sprintf("duplicate Frank object ref kind %q object_id %q", normalized.Kind, normalized.ObjectID),
			})
			continue
		}
		seen[key] = struct{}{}
	}

	return errors
}

func normalizedFrankRegistryObjectRefKey(ref FrankRegistryObjectRef) string {
	normalized := NormalizeFrankRegistryObjectRef(ref)
	return string(normalized.Kind) + "\x1f" + normalized.ObjectID
}

func validateCampaignRefDeclaration(step Step) []ValidationError {
	if step.CampaignRef == nil {
		return nil
	}

	if err := ValidateCampaignRef(*step.CampaignRef); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidCampaignRef,
				StepID:  step.ID,
				Message: "campaign ref is invalid: " + err.Error(),
			},
		}
	}

	return nil
}

func validateTreasuryRefDeclaration(step Step) []ValidationError {
	if step.TreasuryRef == nil {
		return nil
	}

	if err := ValidateTreasuryRef(*step.TreasuryRef); err != nil {
		return []ValidationError{
			{
				Code:    RejectionCodeInvalidTreasuryRef,
				StepID:  step.ID,
				Message: "treasury ref is invalid: " + err.Error(),
			},
		}
	}

	return nil
}

func isValidStepType(stepType StepType) bool {
	switch stepType {
	case StepTypeDiscussion, StepTypeStaticArtifact, StepTypeOneShotCode, StepTypeLongRunningCode, StepTypeSystemAction, StepTypeWaitUser, StepTypeFinalResponse:
		return true
	default:
		return false
	}
}

func isV2OnlyStepType(stepType StepType) bool {
	switch stepType {
	case StepTypeLongRunningCode, StepTypeSystemAction, StepTypeWaitUser:
		return true
	default:
		return false
	}
}

func isValidWaitUserSubtype(subtype StepSubtype) bool {
	switch subtype {
	case StepSubtypeBlocker, StepSubtypeAuthorization, StepSubtypeDefinition:
		return true
	default:
		return false
	}
}

func hasLongRunningStartIntent(step Step) bool {
	for _, criterion := range step.SuccessCriteria {
		normalized := normalizeIntentText(criterion)
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "start the service") ||
			strings.Contains(normalized, "start the server") ||
			strings.Contains(normalized, "start the daemon") ||
			strings.Contains(normalized, "launch the service") ||
			strings.Contains(normalized, "launch the server") ||
			strings.Contains(normalized, "run the service") ||
			strings.Contains(normalized, "run the server") ||
			strings.Contains(normalized, "service running") ||
			strings.Contains(normalized, "server running") ||
			strings.Contains(normalized, "daemon running") {
			return true
		}
	}
	return false
}

func hasLongRunningStartupCommand(step Step) bool {
	return len(longRunningStartupCommand(step)) > 0
}

func longRunningStartupCommand(step Step) []string {
	if len(step.LongRunningStartupCommand) == 0 {
		return nil
	}
	cmd := make([]string, 0, len(step.LongRunningStartupCommand))
	for _, arg := range step.LongRunningStartupCommand {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			return nil
		}
		cmd = append(cmd, trimmed)
	}
	return cmd
}

func longRunningArtifactPath(step Step) string {
	return cleanedArtifactPath(step.LongRunningArtifactPath)
}

func staticArtifactPath(step Step) string {
	return cleanedArtifactPath(step.StaticArtifactPath)
}

func staticArtifactFormat(step Step) string {
	switch strings.ToLower(strings.TrimSpace(step.StaticArtifactFormat)) {
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "markdown", "md":
		return "markdown"
	case "text", "txt":
		return "text"
	default:
		return ""
	}
}

func oneShotArtifactPath(step Step) string {
	return cleanedArtifactPath(step.OneShotArtifactPath)
}

func normalizeIntentText(input string) string {
	return strings.Join(strings.Fields(strings.ToLower(input)), " ")
}

func authorityRank(tier AuthorityTier) (int, bool) {
	switch tier {
	case AuthorityTierLow:
		return 0, true
	case AuthorityTierMedium:
		return 1, true
	case AuthorityTierHigh:
		return 2, true
	default:
		return 0, false
	}
}

func findCycleErrors(stepOrder []string, stepsByID map[string]Step) []ValidationError {
	adjacency := make(map[string][]string, len(stepsByID))
	for _, stepID := range stepOrder {
		step := stepsByID[stepID]
		for _, dependencyID := range step.DependsOn {
			if _, ok := stepsByID[dependencyID]; ok {
				adjacency[stepID] = append(adjacency[stepID], dependencyID)
			}
		}
	}

	indexByID := make(map[string]int, len(stepsByID))
	lowLinkByID := make(map[string]int, len(stepsByID))
	onStack := make(map[string]bool, len(stepsByID))
	stack := make([]string, 0, len(stepsByID))
	index := 0
	components := make([][]string, 0)

	var strongConnect func(string)
	strongConnect = func(stepID string) {
		indexByID[stepID] = index
		lowLinkByID[stepID] = index
		index++
		stack = append(stack, stepID)
		onStack[stepID] = true

		for _, dependencyID := range adjacency[stepID] {
			if _, visited := indexByID[dependencyID]; !visited {
				strongConnect(dependencyID)
				if lowLinkByID[dependencyID] < lowLinkByID[stepID] {
					lowLinkByID[stepID] = lowLinkByID[dependencyID]
				}
				continue
			}

			if onStack[dependencyID] && indexByID[dependencyID] < lowLinkByID[stepID] {
				lowLinkByID[stepID] = indexByID[dependencyID]
			}
		}

		if lowLinkByID[stepID] != indexByID[stepID] {
			return
		}

		component := make([]string, 0, 1)
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == stepID {
				break
			}
		}
		components = append(components, component)
	}

	for _, stepID := range stepOrder {
		if _, visited := indexByID[stepID]; !visited {
			strongConnect(stepID)
		}
	}

	orderByID := make(map[string]int, len(stepOrder))
	for i, stepID := range stepOrder {
		orderByID[stepID] = i
	}

	type cycleComponent struct {
		firstIndex int
		stepID     string
	}

	cycles := make([]cycleComponent, 0)
	for _, component := range components {
		if len(component) == 1 {
			stepID := component[0]
			if !hasSelfDependency(stepsByID[stepID], stepID) {
				continue
			}
		}

		firstStepID := component[0]
		firstIndex := orderByID[firstStepID]
		for _, stepID := range component[1:] {
			if orderByID[stepID] < firstIndex {
				firstStepID = stepID
				firstIndex = orderByID[stepID]
			}
		}
		cycles = append(cycles, cycleComponent{
			firstIndex: firstIndex,
			stepID:     firstStepID,
		})
	}

	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].firstIndex < cycles[j].firstIndex
	})

	errors := make([]ValidationError, 0, len(cycles))
	for _, cycle := range cycles {
		errors = append(errors, ValidationError{
			Code:    RejectionCodeDependencyCycle,
			StepID:  cycle.stepID,
			Message: "dependency cycle detected",
		})
	}
	return errors
}

func hasSelfDependency(step Step, stepID string) bool {
	for _, dependencyID := range step.DependsOn {
		if dependencyID == stepID {
			return true
		}
	}
	return false
}
