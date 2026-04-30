package docscheck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const shellSyntaxSnippetTag = "picobot-check:shell-syntax"

type shellSnippet struct {
	file string
	line int
	body string
	info string
}

func TestTaggedShellSnippetsSyntax(t *testing.T) {
	root := repoRoot(t)
	total := 0
	for _, rel := range currentDocLinkFiles {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read current doc %q: %v", rel, err)
		}
		snippets, err := taggedShellSnippets(rel, string(data))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		for _, snippet := range snippets {
			total++
			cmd := exec.Command("sh", "-n")
			cmd.Dir = root
			cmd.Stdin = strings.NewReader(snippet.body)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s:%d shell syntax failed: %v\n%s", snippet.file, snippet.line, err, string(output))
			}
		}
	}
	if total == 0 {
		t.Fatalf("no shell snippets tagged with %s", shellSyntaxSnippetTag)
	}
}

func taggedShellSnippets(file string, content string) ([]shellSnippet, error) {
	lines := strings.Split(content, "\n")
	var snippets []shellSnippet
	var current *shellSnippet
	var body strings.Builder
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if current == nil {
			if !strings.HasPrefix(trimmed, "```") {
				continue
			}
			info := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			if !strings.Contains(info, shellSyntaxSnippetTag) {
				continue
			}
			if !isShellFenceInfo(info) {
				return snippets, errf("line %d: %s tag requires a sh, shell, or bash fence", i+1, shellSyntaxSnippetTag)
			}
			current = &shellSnippet{file: file, line: i + 1, info: info}
			body.Reset()
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			current.body = body.String()
			snippets = append(snippets, *current)
			current = nil
			body.Reset()
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	if current != nil {
		return snippets, errf("line %d: unterminated tagged shell fence", current.line)
	}
	return snippets, nil
}

func isShellFenceInfo(info string) bool {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "sh", "shell", "bash":
		return true
	default:
		return false
	}
}

type docsCheckError string

func (e docsCheckError) Error() string {
	return string(e)
}

func errf(format string, args ...any) error {
	return docsCheckError(strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf(format, args...), "\n", " ")))
}
