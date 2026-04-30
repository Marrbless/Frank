package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigReferenceJSONExamplesDecodeStrictly(t *testing.T) {
	docsPath := repoRootPath(t, "docs", "CONFIG.md")
	examples := extractMarkdownFences(t, docsPath, "json")
	if len(examples) == 0 {
		t.Fatalf("no JSON examples found in %s", docsPath)
	}

	for _, example := range examples {
		t.Run(fmt.Sprintf("line_%d", example.line), func(t *testing.T) {
			var cfg Config
			decoder := json.NewDecoder(strings.NewReader(example.body))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&cfg); err != nil {
				t.Fatalf("%s:%d JSON example does not decode as Config: %v\n%s", docsPath, example.line, err, example.body)
			}
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				t.Fatalf("%s:%d JSON example has trailing JSON values: %v\n%s", docsPath, example.line, err, example.body)
			}
		})
	}
}

type markdownFence struct {
	line int
	body string
}

func extractMarkdownFences(t *testing.T, path, language string) []markdownFence {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var examples []markdownFence
	var body strings.Builder
	var startLine int
	inFence := false
	capturing := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		if strings.HasPrefix(line, "```") {
			info := strings.TrimSpace(strings.TrimPrefix(line, "```"))
			if !inFence {
				inFence = true
				capturing = info == language
				startLine = lineNumber
				body.Reset()
				continue
			}
			if capturing {
				examples = append(examples, markdownFence{line: startLine, body: body.String()})
			}
			inFence = false
			capturing = false
			continue
		}
		if capturing {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %q: %v", path, err)
	}
	if inFence {
		t.Fatalf("%s has unclosed fenced code block starting at line %d", path, startLine)
	}
	return examples
}

func repoRootPath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(append([]string{root}, parts...)...)
}
