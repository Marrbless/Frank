package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	ImportSkipReasonNoSkillFile          = "missing_skill_file"
	ImportSkipReasonInvalidFrontmatter   = "invalid_frontmatter"
	ImportSkipReasonInvalidSkillID       = "invalid_skill_id"
	ImportSkipReasonMissingDescription   = "missing_description"
	ImportSkipReasonToolEffectsBlocked   = "tool_or_action_effects_not_supported"
	ImportSkipReasonDuplicateCandidate   = "duplicate_candidate"
	ImportSkipReasonDestinationExists    = "destination_exists"
	ImportSkipReasonReadFailed           = "read_failed"
	ImportSkipReasonWriteFailed          = "write_failed"
	ImportSkipReasonNoDetectedSkillRoots = "no_detected_skill_roots"
)

type ImportOptions struct {
	WorkspacePath string
	Sources       []string
	Overwrite     bool
}

type ImportReport struct {
	Workspace string               `json:"workspace"`
	Sources   []string             `json:"sources,omitempty"`
	Imported  []ImportedSkill      `json:"imported,omitempty"`
	Skipped   []SkippedSkillImport `json:"skipped,omitempty"`
}

type ImportedSkill struct {
	ID              string `json:"id"`
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	ContentHash     string `json:"content_hash"`
	Description     string `json:"description"`
}

type SkippedSkillImport struct {
	ID         string `json:"id,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Reason     string `json:"reason"`
}

type importCandidate struct {
	ID          string
	Description string
	Body        string
	SourcePath  string
}

func AutoDetectImportSources(workspacePath, cwd, home string) []string {
	workspacePath = strings.TrimSpace(workspacePath)
	cwd = strings.TrimSpace(cwd)
	home = strings.TrimSpace(home)

	candidates := []string{}
	for _, root := range []string{workspacePath, cwd} {
		if root == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(root, ".agents", "skills"),
			filepath.Join(root, ".codex", "skills"),
		)
	}
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".agents", "skills"),
			filepath.Join(home, ".codex", "skills"),
		)
	}

	return existingUniqueDirs(candidates)
}

func ImportGovernedSkills(opts ImportOptions) (ImportReport, error) {
	workspace := strings.TrimSpace(opts.WorkspacePath)
	if workspace == "" {
		return ImportReport{}, fmt.Errorf("workspace path is required")
	}

	sources := existingUniqueDirs(opts.Sources)
	report := ImportReport{
		Workspace: workspace,
		Sources:   append([]string(nil), sources...),
	}
	if len(sources) == 0 {
		report.Skipped = append(report.Skipped, SkippedSkillImport{Reason: ImportSkipReasonNoDetectedSkillRoots})
		return report, nil
	}

	candidates, skipped := discoverImportCandidates(sources)
	report.Skipped = append(report.Skipped, skipped...)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ID != candidates[j].ID {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].SourcePath < candidates[j].SourcePath
	})

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate.ID]; ok {
			report.Skipped = append(report.Skipped, SkippedSkillImport{
				ID:         candidate.ID,
				SourcePath: candidate.SourcePath,
				Reason:     ImportSkipReasonDuplicateCandidate,
			})
			continue
		}
		seen[candidate.ID] = struct{}{}

		imported, skip := writeImportedSkill(workspace, candidate, opts.Overwrite)
		if skip != nil {
			report.Skipped = append(report.Skipped, *skip)
			continue
		}
		report.Imported = append(report.Imported, imported)
	}

	return report, nil
}

func discoverImportCandidates(sources []string) ([]importCandidate, []SkippedSkillImport) {
	var candidates []importCandidate
	var skipped []SkippedSkillImport

	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}

		if _, err := os.Stat(filepath.Join(source, SkillFileName)); err == nil {
			candidate, skip := readImportCandidate(source)
			if skip != nil {
				skipped = append(skipped, *skip)
				continue
			}
			candidates = append(candidates, candidate)
			continue
		}

		entries, err := os.ReadDir(source)
		if err != nil {
			skipped = append(skipped, SkippedSkillImport{SourcePath: source, Reason: ImportSkipReasonReadFailed})
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		found := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(source, entry.Name())
			if _, err := os.Stat(filepath.Join(dir, SkillFileName)); err != nil {
				continue
			}
			found = true
			candidate, skip := readImportCandidate(dir)
			if skip != nil {
				skipped = append(skipped, *skip)
				continue
			}
			candidates = append(candidates, candidate)
		}
		if !found {
			skipped = append(skipped, SkippedSkillImport{SourcePath: source, Reason: ImportSkipReasonNoSkillFile})
		}
	}

	return candidates, skipped
}

func readImportCandidate(skillDir string) (importCandidate, *SkippedSkillImport) {
	path := filepath.Join(skillDir, SkillFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		return importCandidate{}, &SkippedSkillImport{SourcePath: path, Reason: ImportSkipReasonReadFailed}
	}

	meta, body, ok := parseImportFrontmatter(content)
	if !ok {
		return importCandidate{}, &SkippedSkillImport{SourcePath: path, Reason: ImportSkipReasonInvalidFrontmatter}
	}

	rawID := firstNonEmpty(meta["id"], meta["ref"], meta["name"], filepath.Base(skillDir))
	id := normalizeImportedSkillID(rawID)
	if id == "" {
		return importCandidate{}, &SkippedSkillImport{SourcePath: path, Reason: ImportSkipReasonInvalidSkillID}
	}
	description := strings.TrimSpace(meta["description"])
	if description == "" {
		return importCandidate{}, &SkippedSkillImport{ID: id, SourcePath: path, Reason: ImportSkipReasonMissingDescription}
	}

	promptOnlyValue := strings.TrimSpace(meta["prompt_only"])
	promptOnly := promptOnlyValue == "" || parseBool(promptOnlyValue)
	canAffectToolsOrActions := parseBool(firstNonEmpty(meta["can_affect_tools_or_actions"], meta["can_affect_tools_actions"], meta["affects_tools_actions"]))
	if !promptOnly || canAffectToolsOrActions {
		return importCandidate{}, &SkippedSkillImport{ID: id, SourcePath: path, Reason: ImportSkipReasonToolEffectsBlocked}
	}

	return importCandidate{
		ID:          id,
		Description: description,
		Body:        strings.TrimSpace(body),
		SourcePath:  path,
	}, nil
}

func parseImportFrontmatter(content []byte) (map[string]string, string, bool) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, "", false
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
		if key == "" {
			continue
		}
		meta[key] = trimYAMLScalar(strings.TrimSpace(parts[1]))
	}
	if contentStart < 0 {
		return nil, "", false
	}
	return meta, strings.Join(lines[contentStart:], "\n"), true
}

func normalizeImportedSkillID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '_' || r == '.':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || unicode.IsSpace(r):
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	id := strings.Trim(builder.String(), "-")
	if id == "." || id == ".." || strings.Contains(id, "/") || strings.Contains(id, `\`) {
		return ""
	}
	return id
}

func writeImportedSkill(workspace string, candidate importCandidate, overwrite bool) (ImportedSkill, *SkippedSkillImport) {
	destDir := filepath.Join(workspace, SkillsDirName, candidate.ID)
	destPath := filepath.Join(destDir, SkillFileName)
	if _, err := os.Stat(destPath); err == nil && !overwrite {
		return ImportedSkill{}, &SkippedSkillImport{ID: candidate.ID, SourcePath: candidate.SourcePath, Reason: ImportSkipReasonDestinationExists}
	} else if err != nil && !os.IsNotExist(err) {
		return ImportedSkill{}, &SkippedSkillImport{ID: candidate.ID, SourcePath: candidate.SourcePath, Reason: ImportSkipReasonWriteFailed}
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ImportedSkill{}, &SkippedSkillImport{ID: candidate.ID, SourcePath: candidate.SourcePath, Reason: ImportSkipReasonWriteFailed}
	}

	sourceHash := sha256.Sum256([]byte(candidate.Body))
	content := buildGovernedImportedSkillContent(candidate, hex.EncodeToString(sourceHash[:]))
	if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
		return ImportedSkill{}, &SkippedSkillImport{ID: candidate.ID, SourcePath: candidate.SourcePath, Reason: ImportSkipReasonWriteFailed}
	}

	contentHash := sha256.Sum256([]byte(content))
	return ImportedSkill{
		ID:              candidate.ID,
		SourcePath:      candidate.SourcePath,
		DestinationPath: destPath,
		ContentHash:     "sha256:" + hex.EncodeToString(contentHash[:]),
		Description:     candidate.Description,
	}, nil
}

func buildGovernedImportedSkillContent(candidate importCandidate, sourceHash string) string {
	return fmt.Sprintf(`---
id: %s
version: imported-sha256-%s
description: %s
allowed_activation_scopes: mission_step_prompt
prompt_only: true
can_affect_tools_or_actions: false
---

%s
`, candidate.ID, sourceHash[:12], quoteYAMLString(candidate.Description), candidate.Body)
}

func quoteYAMLString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func existingUniqueDirs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned := filepath.Clean(value)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		info, err := os.Stat(cleaned)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}
