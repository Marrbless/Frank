package missioncontrol

import "fmt"

type RuntimeExtensionPermissionAssessmentState string

const (
	RuntimeExtensionPermissionAssessmentStateAllowed RuntimeExtensionPermissionAssessmentState = "allowed"
	RuntimeExtensionPermissionAssessmentStateBlocked RuntimeExtensionPermissionAssessmentState = "blocked"
)

type RuntimeExtensionPermissionBlocker struct {
	Code       RejectionCode `json:"code"`
	Reason     string        `json:"reason"`
	Permission string        `json:"permission,omitempty"`
	ToolName   string        `json:"tool_name,omitempty"`
}

type RuntimeExtensionPermissionAssessment struct {
	State                    RuntimeExtensionPermissionAssessmentState `json:"state"`
	BaselineExtensionPackID  string                                    `json:"baseline_extension_pack_id"`
	CandidateExtensionPackID string                                    `json:"candidate_extension_pack_id"`
	CompatibilityContractRef string                                    `json:"compatibility_contract_ref,omitempty"`
	WidenedPermissions       []string                                  `json:"widened_permissions,omitempty"`
	NewExternalTools         []string                                  `json:"new_external_tools,omitempty"`
	NewExternalSideEffects   []string                                  `json:"new_external_side_effects,omitempty"`
	Blockers                 []RuntimeExtensionPermissionBlocker       `json:"blockers,omitempty"`
}

func AssessRuntimeExtensionPermissionWidening(root, baselineExtensionPackID, candidateExtensionPackID string) (RuntimeExtensionPermissionAssessment, error) {
	baseline, err := LoadRuntimeExtensionPackRecord(root, baselineExtensionPackID)
	if err != nil {
		return RuntimeExtensionPermissionAssessment{}, fmt.Errorf("mission store runtime extension permission assessment baseline_extension_pack_id %q: %w", baselineExtensionPackID, err)
	}
	candidate, err := LoadRuntimeExtensionPackRecord(root, candidateExtensionPackID)
	if err != nil {
		return RuntimeExtensionPermissionAssessment{}, fmt.Errorf("mission store runtime extension permission assessment candidate_extension_pack_id %q: %w", candidateExtensionPackID, err)
	}
	return AssessRuntimeExtensionPermissionWideningRecords(baseline, candidate), nil
}

func AssessRuntimeExtensionPermissionWideningRecords(baseline, candidate RuntimeExtensionPackRecord) RuntimeExtensionPermissionAssessment {
	baseline = NormalizeRuntimeExtensionPackRecord(baseline)
	candidate = NormalizeRuntimeExtensionPackRecord(candidate)
	assessment := RuntimeExtensionPermissionAssessment{
		State:                    RuntimeExtensionPermissionAssessmentStateAllowed,
		BaselineExtensionPackID:  baseline.ExtensionPackID,
		CandidateExtensionPackID: candidate.ExtensionPackID,
		CompatibilityContractRef: candidate.CompatibilityContractRef,
	}
	addBlocker := func(blocker RuntimeExtensionPermissionBlocker) {
		assessment.Blockers = append(assessment.Blockers, blocker)
		assessment.State = RuntimeExtensionPermissionAssessmentStateBlocked
	}

	if candidate.CompatibilityContractRef != baseline.CompatibilityContractRef {
		addBlocker(RuntimeExtensionPermissionBlocker{
			Code:   RejectionCodeV4ExtensionCompatibilityRequired,
			Reason: fmt.Sprintf("candidate compatibility_contract_ref %q does not match baseline %q", candidate.CompatibilityContractRef, baseline.CompatibilityContractRef),
		})
	}
	for _, permission := range runtimeExtensionStringSetDifference(candidate.DeclaredPermissions, baseline.DeclaredPermissions) {
		assessment.WidenedPermissions = append(assessment.WidenedPermissions, permission)
		addBlocker(RuntimeExtensionPermissionBlocker{
			Code:       RejectionCodeV4ExtensionPermissionWidening,
			Reason:     fmt.Sprintf("candidate declares new permission %q", permission),
			Permission: permission,
		})
	}
	for _, sideEffect := range runtimeExtensionStringSetDifference(candidate.ExternalSideEffects, baseline.ExternalSideEffects) {
		assessment.NewExternalSideEffects = append(assessment.NewExternalSideEffects, sideEffect)
		addBlocker(RuntimeExtensionPermissionBlocker{
			Code:   RejectionCodeV4ExtensionPermissionWidening,
			Reason: fmt.Sprintf("candidate declares new external side effect %q", sideEffect),
		})
	}

	baselineExternalTools := runtimeExtensionExternalToolSet(baseline.DeclaredTools)
	for _, tool := range candidate.DeclaredTools {
		if !tool.ExternalSideEffect {
			continue
		}
		if _, ok := baselineExternalTools[tool.ToolName]; ok {
			continue
		}
		assessment.NewExternalTools = append(assessment.NewExternalTools, tool.ToolName)
		addBlocker(RuntimeExtensionPermissionBlocker{
			Code:     RejectionCodeV4ExtensionPermissionWidening,
			Reason:   fmt.Sprintf("candidate declares new external side-effect tool %q", tool.ToolName),
			ToolName: tool.ToolName,
		})
	}
	return assessment
}

func runtimeExtensionStringSetDifference(candidate, baseline []string) []string {
	baselineSet := make(map[string]struct{}, len(baseline))
	for _, value := range baseline {
		baselineSet[value] = struct{}{}
	}
	diff := make([]string, 0)
	for _, value := range candidate {
		if _, ok := baselineSet[value]; !ok {
			diff = append(diff, value)
		}
	}
	return diff
}

func runtimeExtensionExternalToolSet(tools []RuntimeExtensionToolDeclaration) map[string]struct{} {
	set := make(map[string]struct{})
	for _, tool := range tools {
		tool = NormalizeRuntimeExtensionToolDeclaration(tool)
		if tool.ExternalSideEffect && tool.ToolName != "" {
			set[tool.ToolName] = struct{}{}
		}
	}
	return set
}
