package docscheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type envExampleSpec struct {
	rel      string
	required []string
}

type envAssignment struct {
	line       int
	key        string
	hasComment bool
}

func TestEnvExamplesHaveRequiredKeysAndComments(t *testing.T) {
	root := repoRoot(t)
	specs := []envExampleSpec{
		{
			rel: "configs/desktop.env.example",
			required: []string{
				"PBOT_ROLE",
				"PBOT_ALLOW_EXEC",
				"PBOT_ALLOW_NETWORK",
				"PBOT_APPROVAL_MODE",
			},
		},
		{
			rel: "configs/phone.env.example",
			required: []string{
				"PBOT_ROLE",
				"PBOT_ALLOW_EXEC",
				"PBOT_ALLOW_NETWORK",
				"PBOT_APPROVAL_MODE",
			},
		},
		{
			rel: "docker/.env.example",
			required: []string{
				"OPENAI_API_KEY",
				"OPENAI_API_BASE",
				"PICOBOT_MODEL",
				"PICOBOT_MAX_TOKENS",
				"PICOBOT_MAX_TOOL_ITERATIONS",
				"PICOBOT_ENABLE_TOOL_ACTIVITY_INDICATOR",
				"TELEGRAM_BOT_TOKEN",
				"TELEGRAM_ALLOW_FROM",
				"DISCORD_BOT_TOKEN",
				"DISCORD_ALLOW_FROM",
				"SLACK_APP_TOKEN",
				"SLACK_BOT_TOKEN",
				"SLACK_ALLOW_USERS",
				"SLACK_ALLOW_CHANNELS",
			},
		},
	}

	for _, spec := range specs {
		spec := spec
		t.Run(spec.rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(spec.rel)))
			if err != nil {
				t.Fatalf("read env example: %v", err)
			}
			assignments, err := parseEnvExampleAssignments(string(data))
			if err != nil {
				t.Fatalf("parse env example: %v", err)
			}
			assertEnvExampleRequiredKeys(t, spec, assignments)
			for _, assignment := range assignments {
				if !assignment.hasComment {
					t.Errorf("line %d: %s must have a directly preceding comment", assignment.line, assignment.key)
				}
			}
		})
	}
}

func parseEnvExampleAssignments(content string) ([]envAssignment, error) {
	lines := strings.Split(content, "\n")
	assignments := make([]envAssignment, 0)
	hasComment := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			hasComment = false
			continue
		case strings.HasPrefix(trimmed, "#"):
			hasComment = true
			continue
		}

		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, errf("line %d: env example line must be a KEY=value assignment or comment", i+1)
		}
		if err := validateEnvExampleKey(key); err != nil {
			return nil, errf("line %d: %v", i+1, err)
		}
		assignments = append(assignments, envAssignment{
			line:       i + 1,
			key:        key,
			hasComment: hasComment,
		})
		hasComment = false
	}
	return assignments, nil
}

func validateEnvExampleKey(key string) error {
	if key == "" {
		return errf("empty env key")
	}
	for _, r := range key {
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' {
			continue
		}
		return errf("invalid env key %q", key)
	}
	return nil
}

func assertEnvExampleRequiredKeys(t *testing.T, spec envExampleSpec, assignments []envAssignment) {
	t.Helper()

	seen := make(map[string]envAssignment, len(assignments))
	for _, assignment := range assignments {
		if previous, ok := seen[assignment.key]; ok {
			t.Errorf("%s:%d duplicates %s first declared on line %d", spec.rel, assignment.line, assignment.key, previous.line)
		}
		seen[assignment.key] = assignment
	}
	for _, key := range spec.required {
		if _, ok := seen[key]; !ok {
			t.Errorf("%s is missing required key %s", spec.rel, key)
		}
	}
}
