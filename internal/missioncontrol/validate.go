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
	RejectionCodeMissingDependencyTarget       RejectionCode = "missing_dependency_target"
	RejectionCodeDependencyCycle               RejectionCode = "dependency_cycle"
	RejectionCodeMissingTerminalFinalStep      RejectionCode = "missing_terminal_final_response"
	RejectionCodeInvalidStepType               RejectionCode = "invalid_step_type"
	RejectionCodeLongRunningStartForbidden     RejectionCode = "longrun_start_forbidden"
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

	errors := make([]ValidationError, 0, len(duplicateErrors)+len(missingDependencyErrors)+len(cycleErrors)+len(terminalErrors)+len(invalidTypeErrors)+len(authorityErrors)+len(toolScopeErrors))
	errors = append(errors, duplicateErrors...)
	errors = append(errors, missingDependencyErrors...)
	errors = append(errors, cycleErrors...)
	errors = append(errors, terminalErrors...)
	errors = append(errors, invalidTypeErrors...)
	errors = append(errors, authorityErrors...)
	errors = append(errors, toolScopeErrors...)
	return errors
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
		return fmt.Errorf("Frank object ref kind %q is invalid", strings.TrimSpace(string(kind)))
	}
}

func validateFrankRegistryObjectRef(ref FrankRegistryObjectRef) error {
	ref = NormalizeFrankRegistryObjectRef(ref)
	if err := validateFrankRegistryObjectKind(ref.Kind); err != nil {
		return err
	}
	if ref.ObjectID == "" {
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
