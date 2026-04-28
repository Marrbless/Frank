package missioncontrol

import (
	"fmt"
	"strings"
)

type frozenPolicySurfaceDeclaration struct {
	Declaration   string
	PolicySurface string
}

var frozenPolicySurfaceRoots = []string{
	"authority",
	"approval",
	"autonomy",
	"treasury",
	"campaign",
}

var frozenPolicySurfaceCompanions = map[string]struct{}{
	"authority":    {},
	"approval":     {},
	"autonomy":     {},
	"campaign":     {},
	"condition":    {},
	"conditions":   {},
	"control":      {},
	"eligibility":  {},
	"gate":         {},
	"gates":        {},
	"guardrail":    {},
	"guardrails":   {},
	"identity":     {},
	"lifecycle":    {},
	"owner":        {},
	"policies":     {},
	"policy":       {},
	"predicate":    {},
	"requirement":  {},
	"requirements": {},
	"rule":         {},
	"rules":        {},
	"stop":         {},
	"syntax":       {},
	"tier":         {},
	"tiers":        {},
	"treasury":     {},
}

func findFrozenPolicySurfaceDeclaration(declarations []string) (frozenPolicySurfaceDeclaration, bool) {
	for _, declaration := range normalizeRuntimePackStrings(declarations) {
		if policySurface, ok := frozenPolicySurfaceRootForDeclaration(declaration); ok {
			return frozenPolicySurfaceDeclaration{
				Declaration:   declaration,
				PolicySurface: policySurface,
			}, true
		}
	}
	return frozenPolicySurfaceDeclaration{}, false
}

func frozenPolicySurfaceRootForDeclaration(declaration string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(declaration))
	if normalized == "" {
		return "", false
	}
	for _, root := range frozenPolicySurfaceRoots {
		if normalized == root {
			return root, true
		}
	}

	tokens := policySurfaceDeclarationTokens(normalized)
	if len(tokens) == 0 {
		return "", false
	}
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		tokenSet[token] = struct{}{}
	}
	for _, root := range frozenPolicySurfaceRoots {
		if _, ok := tokenSet[root]; !ok {
			continue
		}
		if len(tokenSet) == 1 {
			return root, true
		}
		for companion := range frozenPolicySurfaceCompanions {
			if companion == root {
				continue
			}
			if _, ok := tokenSet[companion]; ok {
				return root, true
			}
		}
	}
	return "", false
}

func policySurfaceDeclarationTokens(declaration string) []string {
	return strings.FieldsFunc(declaration, func(r rune) bool {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= '0' && r <= '9' {
			return false
		}
		return true
	})
}

func policySurfaceMutationBlockerSummary(declarationKind string, declaration frozenPolicySurfaceDeclaration) string {
	return fmt.Sprintf("%s: %s %q targets frozen policy surface %q", RejectionCodeV4PolicyMutationForbidden, declarationKind, declaration.Declaration, declaration.PolicySurface)
}
