package missioncontrol

import "strings"

const SkillActivationScopeMissionStepPrompt = "mission_step_prompt"

type GovernedSkillStatus struct {
	ID                      string   `json:"id"`
	Version                 string   `json:"version,omitempty"`
	ContentHash             string   `json:"content_hash,omitempty"`
	Description             string   `json:"description,omitempty"`
	AllowedActivationScopes []string `json:"allowed_activation_scopes,omitempty"`
	PromptOnly              bool     `json:"prompt_only"`
	CanAffectToolsOrActions bool     `json:"can_affect_tools_or_actions,omitempty"`
	Path                    string   `json:"path,omitempty"`
}

type GovernedSkillSkippedStatus struct {
	ID     string `json:"id,omitempty"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type GovernedSkillSelectionStatus struct {
	Root     string                       `json:"root,omitempty"`
	Scope    string                       `json:"scope,omitempty"`
	Selected []string                     `json:"selected,omitempty"`
	Active   []GovernedSkillStatus        `json:"active,omitempty"`
	Skipped  []GovernedSkillSkippedStatus `json:"skipped,omitempty"`
}

func EffectiveSelectedSkills(job *Job, step *Step) []string {
	var selected []string
	if job != nil {
		selected = append(selected, job.SelectedSkills...)
	}
	if step != nil {
		selected = append(selected, step.SelectedSkills...)
	}
	return NormalizeSelectedSkills(selected)
}

func NormalizeSelectedSkills(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func WithGovernedSkillSelectionStatus(summary OperatorStatusSummary, status GovernedSkillSelectionStatus) OperatorStatusSummary {
	if len(status.Selected) == 0 && len(status.Active) == 0 && len(status.Skipped) == 0 {
		return summary
	}
	summary.Skills = &status
	return summary
}
