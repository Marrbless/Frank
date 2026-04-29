package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/local/picobot/internal/missioncontrol"
)

const (
	SkillsDirName = "skills"
	SkillFileName = "SKILL.md"

	SkipReasonDuplicateSelection       = "duplicate_selected_skill"
	SkipReasonInvalidSkillID           = "invalid_skill_id"
	SkipReasonSkillNotFound            = "skill_not_found"
	SkipReasonIDMismatch               = "skill_id_must_match_directory"
	SkipReasonMissingDescription       = "missing_description"
	SkipReasonMissingActivationScopes  = "missing_allowed_activation_scopes"
	SkipReasonScopeNotAllowed          = "activation_scope_not_allowed"
	SkipReasonToolActionEffectsBlocked = "tool_or_action_effects_not_supported"
	SkipReasonUnreadable               = "unreadable_skill"
	SkipReasonInvalidFrontmatter       = "invalid_frontmatter"
)

// Skill represents a governed local skill with parsed metadata and body content.
type Skill struct {
	ID                      string
	Name                    string
	Version                 string
	Description             string
	AllowedActivationScopes []string
	PromptOnly              bool
	CanAffectToolsOrActions bool
	ContentHash             string
	Content                 string
	Path                    string
}

type LoadReport struct {
	Root     string
	Scope    string
	Selected []string
	Active   []Skill
	Skipped  []missioncontrol.GovernedSkillSkippedStatus
}

// Loader handles loading governed skills from the workspace skills directory.
type Loader struct {
	workspacePath string
}

func NewLoader(workspacePath string) *Loader {
	return &Loader{workspacePath: workspacePath}
}

func (l *Loader) SkillsRoot() string {
	return filepath.Join(l.workspacePath, SkillsDirName)
}

// LoadAll loads all valid prompt-only skills in deterministic directory order.
// It is retained for management/readback tests; prompt construction should use
// LoadSelected so unselected skills never enter runtime behavior.
func (l *Loader) LoadAll() ([]Skill, error) {
	entries, err := os.ReadDir(l.SkillsRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return []Skill{}, nil
		}
		return nil, err
	}

	var loaded []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skill, skipped := l.loadSkillByID(entry.Name(), "")
		if skipped != nil {
			continue
		}
		loaded = append(loaded, skill)
	}
	return loaded, nil
}

func (l *Loader) LoadByName(name string) (Skill, error) {
	skill, skipped := l.loadSkillByID(name, "")
	if skipped != nil {
		return Skill{}, fmt.Errorf("%s", skipped.Reason)
	}
	return skill, nil
}

func (l *Loader) LoadSelected(selected []string, scope string) (LoadReport, error) {
	report := LoadReport{
		Root:     l.SkillsRoot(),
		Scope:    strings.TrimSpace(scope),
		Selected: normalizeSelectedWithDuplicates(selected),
	}
	if len(report.Selected) == 0 {
		return report, nil
	}

	seen := make(map[string]struct{}, len(report.Selected))
	for _, id := range report.Selected {
		if _, ok := seen[id]; ok {
			report.Skipped = append(report.Skipped, missioncontrol.GovernedSkillSkippedStatus{
				ID:     id,
				Path:   filepath.Join(report.Root, id, SkillFileName),
				Reason: SkipReasonDuplicateSelection,
			})
			continue
		}
		seen[id] = struct{}{}

		skill, skipped := l.loadSkillByID(id, report.Scope)
		if skipped != nil {
			report.Skipped = append(report.Skipped, *skipped)
			continue
		}
		report.Active = append(report.Active, skill)
	}
	return report, nil
}

func (r LoadReport) Status() missioncontrol.GovernedSkillSelectionStatus {
	status := missioncontrol.GovernedSkillSelectionStatus{
		Root:     r.Root,
		Scope:    r.Scope,
		Selected: append([]string(nil), r.Selected...),
		Skipped:  append([]missioncontrol.GovernedSkillSkippedStatus(nil), r.Skipped...),
	}
	status.Active = make([]missioncontrol.GovernedSkillStatus, 0, len(r.Active))
	for _, skill := range r.Active {
		status.Active = append(status.Active, missioncontrol.GovernedSkillStatus{
			ID:                      skill.ID,
			Version:                 skill.Version,
			ContentHash:             skill.ContentHash,
			Description:             skill.Description,
			AllowedActivationScopes: append([]string(nil), skill.AllowedActivationScopes...),
			PromptOnly:              skill.PromptOnly,
			CanAffectToolsOrActions: skill.CanAffectToolsOrActions,
			Path:                    skill.Path,
		})
	}
	return status
}

func (l *Loader) loadSkillByID(id string, scope string) (Skill, *missioncontrol.GovernedSkillSkippedStatus) {
	id = strings.TrimSpace(id)
	path := filepath.Join(l.SkillsRoot(), id, SkillFileName)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, `\`) || id == "." || id == ".." {
		return Skill{}, &missioncontrol.GovernedSkillSkippedStatus{ID: id, Path: path, Reason: SkipReasonInvalidSkillID}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		reason := SkipReasonUnreadable
		if os.IsNotExist(err) {
			reason = SkipReasonSkillNotFound
		}
		return Skill{}, &missioncontrol.GovernedSkillSkippedStatus{ID: id, Path: path, Reason: reason}
	}

	skill, reason := parseSkill(path, id, content)
	if reason != "" {
		return Skill{}, &missioncontrol.GovernedSkillSkippedStatus{ID: id, Path: path, Reason: reason}
	}
	if scope != "" && !containsString(skill.AllowedActivationScopes, scope) {
		return Skill{}, &missioncontrol.GovernedSkillSkippedStatus{ID: id, Path: path, Reason: SkipReasonScopeNotAllowed}
	}
	return skill, nil
}

func parseSkill(path string, dirID string, content []byte) (Skill, string) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return Skill{}, SkipReasonInvalidFrontmatter
	}

	meta := make(map[string]string)
	contentStart := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			contentStart = i + 1
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			meta[key] = trimYAMLScalar(value)
		}
	}
	if contentStart < 0 {
		return Skill{}, SkipReasonInvalidFrontmatter
	}

	id := firstNonEmpty(meta["id"], meta["ref"], meta["name"])
	if id == "" {
		return Skill{}, SkipReasonInvalidSkillID
	}
	if id != dirID {
		return Skill{}, SkipReasonIDMismatch
	}
	description := strings.TrimSpace(meta["description"])
	if description == "" {
		return Skill{}, SkipReasonMissingDescription
	}
	scopes := parseListValue(meta["allowed_activation_scopes"])
	if len(scopes) == 0 {
		return Skill{}, SkipReasonMissingActivationScopes
	}

	promptOnly := parseBool(meta["prompt_only"])
	canAffect := parseBool(firstNonEmpty(meta["can_affect_tools_or_actions"], meta["can_affect_tools_actions"], meta["affects_tools_actions"]))
	if !promptOnly || canAffect {
		return Skill{}, SkipReasonToolActionEffectsBlocked
	}

	hash := sha256.Sum256(content)
	body := strings.TrimSpace(strings.Join(lines[contentStart:], "\n"))
	return Skill{
		ID:                      id,
		Name:                    id,
		Version:                 strings.TrimSpace(meta["version"]),
		Description:             description,
		AllowedActivationScopes: scopes,
		PromptOnly:              promptOnly,
		CanAffectToolsOrActions: canAffect,
		ContentHash:             "sha256:" + hex.EncodeToString(hash[:]),
		Content:                 body,
		Path:                    path,
	}, ""
}

func normalizeSelectedWithDuplicates(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func parseListValue(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := trimYAMLScalar(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}

func trimYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "1":
		return true
	default:
		return false
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
