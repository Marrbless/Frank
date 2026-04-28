package missioncontrol

import (
	"errors"
	"fmt"
	"strings"
)

type RuntimePackComponentAdmissionState string

const (
	RuntimePackComponentAdmissionStateAdmitted RuntimePackComponentAdmissionState = "admitted"
	RuntimePackComponentAdmissionStateBlocked  RuntimePackComponentAdmissionState = "blocked"
)

type RuntimePackComponentAdmissionStatus struct {
	State            RuntimePackComponentAdmissionState `json:"state"`
	RejectionCode    RejectionCode                      `json:"rejection_code,omitempty"`
	Reason           string                             `json:"reason,omitempty"`
	HotUpdateID      string                             `json:"hot_update_id"`
	CandidatePackID  string                             `json:"candidate_pack_id"`
	Kind             RuntimePackComponentKind           `json:"kind"`
	ComponentID      string                             `json:"component_id"`
	SurfaceClass     string                             `json:"surface_class,omitempty"`
	DeclaredSurfaces []string                           `json:"declared_surfaces,omitempty"`
}

func AssessRuntimePackComponentAdmissionForHotUpdate(root, hotUpdateID string) ([]RuntimePackComponentAdmissionStatus, error) {
	if err := ValidateStoreRoot(root); err != nil {
		return nil, err
	}
	ref := NormalizeHotUpdateGateRef(HotUpdateGateRef{HotUpdateID: hotUpdateID})
	if err := ValidateHotUpdateGateRef(ref); err != nil {
		return nil, err
	}
	gate, err := LoadHotUpdateGateRecord(root, ref.HotUpdateID)
	if err != nil {
		if errors.Is(err, ErrHotUpdateGateRecordNotFound) {
			return nil, fmt.Errorf("mission store runtime pack component admission hot_update_id %q: %w", ref.HotUpdateID, err)
		}
		return nil, err
	}
	candidate, err := LoadRuntimePackRecord(root, gate.CandidatePackID)
	if err != nil {
		return nil, fmt.Errorf("mission store runtime pack component admission candidate_pack_id %q: %w", gate.CandidatePackID, err)
	}
	components, err := ResolveRuntimePackComponents(root, candidate)
	if err != nil {
		return nil, err
	}
	return assessRuntimePackComponentAdmission(gate, candidate, components), nil
}

func RequireRuntimePackComponentAdmissionForHotUpdate(root, hotUpdateID string) error {
	statuses, err := AssessRuntimePackComponentAdmissionForHotUpdate(root, hotUpdateID)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		if status.State == RuntimePackComponentAdmissionStateBlocked {
			return fmt.Errorf("mission store runtime pack component admission hot_update_id %q component %s/%s blocked: %s: %s", status.HotUpdateID, status.Kind, status.ComponentID, status.RejectionCode, status.Reason)
		}
	}
	return nil
}

func assessRuntimePackComponentAdmission(gate HotUpdateGateRecord, candidate RuntimePackRecord, components ResolvedRuntimePackComponents) []RuntimePackComponentAdmissionStatus {
	candidate = NormalizeRuntimePackRecord(candidate)
	gate = NormalizeHotUpdateGateRecord(gate)
	return []RuntimePackComponentAdmissionStatus{
		assessSingleRuntimePackComponentAdmission(gate, candidate, components.PromptPack),
		assessSingleRuntimePackComponentAdmission(gate, candidate, components.SkillPack),
		assessSingleRuntimePackComponentAdmission(gate, candidate, components.ManifestPack),
		assessSingleRuntimePackComponentAdmission(gate, candidate, components.ExtensionPack),
	}
}

func assessSingleRuntimePackComponentAdmission(gate HotUpdateGateRecord, candidate RuntimePackRecord, component RuntimePackComponentRecord) RuntimePackComponentAdmissionStatus {
	component = NormalizeRuntimePackComponentRecord(component)
	status := RuntimePackComponentAdmissionStatus{
		State:            RuntimePackComponentAdmissionStateAdmitted,
		HotUpdateID:      gate.HotUpdateID,
		CandidatePackID:  candidate.PackID,
		Kind:             component.Kind,
		ComponentID:      component.ComponentID,
		SurfaceClass:     component.SurfaceClass,
		DeclaredSurfaces: append([]string(nil), component.DeclaredSurfaces...),
	}
	block := func(code RejectionCode, reason string) RuntimePackComponentAdmissionStatus {
		status.State = RuntimePackComponentAdmissionStateBlocked
		status.RejectionCode = code
		status.Reason = reason
		return status
	}

	if !component.HotReloadable {
		return block(RejectionCodeV4ReloadModeUnsupported, "component is not hot_reloadable")
	}
	if component.SurfaceClass == "" {
		return block(RejectionCodeV4SurfaceClassRequired, "component surface_class is required")
	}
	if !runtimePackStringSetContains(candidate.SurfaceClasses, component.SurfaceClass) {
		return block(RejectionCodeV4SurfaceClassRequired, fmt.Sprintf("component surface_class %q is not declared by candidate pack", component.SurfaceClass))
	}
	if !runtimePackStringSetContains(gate.SurfaceClasses, component.SurfaceClass) {
		return block(RejectionCodeV4SurfaceClassRequired, fmt.Sprintf("component surface_class %q is not declared by hot-update gate", component.SurfaceClass))
	}
	if len(component.DeclaredSurfaces) == 0 {
		return block(RejectionCodeV4MutationScopeViolation, "component declared_surfaces are required")
	}
	for _, surface := range component.DeclaredSurfaces {
		if runtimePackStringSetContains(candidate.ImmutableSurfaces, surface) {
			return block(RejectionCodeV4ForbiddenSurfaceChange, fmt.Sprintf("component declared surface %q is immutable for candidate pack", surface))
		}
		if !runtimePackStringSetContains(candidate.MutableSurfaces, surface) {
			return block(RejectionCodeV4MutationScopeViolation, fmt.Sprintf("component declared surface %q is not mutable for candidate pack", surface))
		}
		if !runtimePackStringSetContains(gate.TargetSurfaces, surface) {
			return block(RejectionCodeV4MutationScopeViolation, fmt.Sprintf("component declared surface %q is not targeted by hot-update gate", surface))
		}
	}
	return status
}

func runtimePackStringSetContains(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}
