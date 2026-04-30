package docscheck

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

var currentDocLinkFiles = []string{
	"AGENTS.md",
	"START_HERE_OPERATOR.md",
	"CONTEXT.md",
	"README.md",
	"docs/CANONICAL_RUNTIME_TRUTH.md",
	"docs/HOW_TO_START.md",
	"docs/CONFIG.md",
	"docs/DEVELOPMENT.md",
	"docs/RELEASE_CHECKLIST.md",
	"docs/ANDROID_PHONE_DEPLOYMENT.md",
	"docs/HOT_UPDATE_OPERATOR_RUNBOOK.md",
	"docs/maintenance/CURRENT.md",
	"docs/maintenance/TOTAL_REPO_10X_ASSESSMENT.md",
	"docs/maintenance/WRITE_BOUNDARIES.md",
	"docs/maintenance/TOOL_PERMISSION_MANIFEST.md",
	"docs/maintenance/PII_DURABLE_FIELD_INVENTORY.md",
	"docs/maintenance/SPEC_TO_CODE_ROUTE_TABLE.md",
	"docs/maintenance/AUTONOMOUS_IMPROVEMENT_MATRIX.md",
	"docs/maintenance/AUTONOMOUS_MATRIX_FRESHNESS.md",
	"docs/maintenance/AUTONOMOUS_ISSUE_DRAFTS.md",
	"docs/maintenance/AUTONOMOUS_IMPLEMENTATION_RECEIPT_TEMPLATE.md",
	"docs/maintenance/AUTONOMOUS_REVIEW_BRIEF_TEMPLATE.md",
	"docs/maintenance/AUTONOMOUS_STOP_CONDITIONS.md",
}

type markdownLink struct {
	line   int
	target string
}

func TestCurrentDocsRelativeLinksResolve(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range currentDocLinkFiles {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(root, filepath.FromSlash(rel))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read current doc: %v", err)
			}
			for _, link := range markdownLinks(string(data)) {
				targetPath, fragment, ok := localTarget(link.target)
				if !ok {
					continue
				}
				if targetPath == "" {
					targetPath = rel
				}
				resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(targetPath)))
				if _, err := os.Stat(resolved); err != nil {
					t.Errorf("line %d: link target %q resolves to missing path %q: %v", link.line, link.target, resolved, err)
					continue
				}
				if fragment != "" && strings.EqualFold(filepath.Ext(resolved), ".md") {
					assertMarkdownAnchorExists(t, resolved, fragment, link)
				}
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func markdownLinks(content string) []markdownLink {
	lines := strings.Split(content, "\n")
	links := make([]markdownLink, 0)
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, target := range markdownLinkTargetsOnLine(line) {
			links = append(links, markdownLink{line: i + 1, target: target})
		}
	}
	return links
}

func markdownLinkTargetsOnLine(line string) []string {
	targets := make([]string, 0)
	start := 0
	for {
		idx := strings.Index(line[start:], "](")
		if idx < 0 {
			return targets
		}
		targetStart := start + idx + len("](")
		targetEnd := strings.IndexByte(line[targetStart:], ')')
		if targetEnd < 0 {
			return targets
		}
		raw := strings.TrimSpace(line[targetStart : targetStart+targetEnd])
		if raw != "" {
			targets = append(targets, stripOptionalMarkdownTitle(raw))
		}
		start = targetStart + targetEnd + 1
	}
}

func stripOptionalMarkdownTitle(target string) string {
	if strings.HasPrefix(target, "<") {
		if end := strings.IndexByte(target, '>'); end >= 0 {
			return strings.Trim(target[:end+1], "<>")
		}
	}
	for i, r := range target {
		if unicode.IsSpace(r) {
			return target[:i]
		}
	}
	return target
}

func localTarget(target string) (path string, fragment string, ok bool) {
	target = strings.TrimSpace(target)
	if target == "" || strings.HasPrefix(target, "#") {
		return "", strings.TrimPrefix(target, "#"), true
	}
	if strings.HasPrefix(target, "//") || hasScheme(target) {
		return "", "", false
	}
	if strings.HasPrefix(target, "/") {
		return "", "", false
	}
	if idx := strings.IndexByte(target, '#'); idx >= 0 {
		path = target[:idx]
		fragment = target[idx+1:]
	} else {
		path = target
	}
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	if decoded, err := url.QueryUnescape(fragment); err == nil {
		fragment = decoded
	}
	return path, fragment, true
}

func hasScheme(target string) bool {
	for i, r := range target {
		switch {
		case r == ':':
			return i > 0
		case r == '/' || r == '#' || r == '?':
			return false
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '-' || r == '.':
			continue
		default:
			return false
		}
	}
	return false
}

func assertMarkdownAnchorExists(t *testing.T, markdownPath string, fragment string, link markdownLink) {
	t.Helper()
	anchors, err := markdownAnchors(markdownPath)
	if err != nil {
		t.Errorf("line %d: read anchor target %q: %v", link.line, markdownPath, err)
		return
	}
	if _, ok := anchors[slugifyHeading(fragment)]; ok {
		return
	}
	t.Errorf("line %d: link target %q has missing anchor %q in %q", link.line, link.target, fragment, markdownPath)
}

func markdownAnchors(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	anchors := make(map[string]struct{})
	counts := make(map[string]int)
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		title := strings.TrimLeft(trimmed, "#")
		if len(title) == len(trimmed) || !strings.HasPrefix(title, " ") {
			continue
		}
		slug := slugifyHeading(strings.TrimSpace(title))
		if slug == "" {
			continue
		}
		count := counts[slug]
		counts[slug] = count + 1
		if count > 0 {
			slug = slug + "-" + strconv.Itoa(count)
		}
		anchors[slug] = struct{}{}
	}
	return anchors, nil
}

func slugifyHeading(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	var b strings.Builder
	prevDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
