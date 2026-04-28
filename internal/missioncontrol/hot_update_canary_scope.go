package missioncontrol

import (
	"fmt"
	"strings"
)

type HotUpdateCanaryEvidenceSource string

const (
	HotUpdateCanaryEvidenceSourceOperatorRecorded HotUpdateCanaryEvidenceSource = "operator_recorded"
	HotUpdateCanaryEvidenceSourceAutomaticTraffic HotUpdateCanaryEvidenceSource = "automatic_traffic"
)

func normalizeHotUpdateCanaryScopeRefs(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(refs))
	normalized := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		normalized = append(normalized, ref)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateHotUpdateCanaryScopeRefs(surface, fieldName string, refs []string) error {
	if len(refs) == 0 {
		return fmt.Errorf("%s %s are required", surface, fieldName)
	}
	for index, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("%s %s[%d] is required", surface, fieldName, index)
		}
	}
	return nil
}

func hotUpdateCanaryScopeContainsAll(scope, exercised []string) (string, bool) {
	allowed := make(map[string]struct{}, len(scope))
	for _, ref := range scope {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			allowed[ref] = struct{}{}
		}
	}
	for _, ref := range exercised {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := allowed[ref]; !ok {
			return ref, false
		}
	}
	return "", true
}

func isValidHotUpdateCanaryEvidenceSource(source HotUpdateCanaryEvidenceSource) bool {
	switch source {
	case HotUpdateCanaryEvidenceSourceOperatorRecorded,
		HotUpdateCanaryEvidenceSourceAutomaticTraffic:
		return true
	default:
		return false
	}
}
