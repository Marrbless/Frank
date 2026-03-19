package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecTool_DisallowsPythonInline(t *testing.T) {
	tool := NewExecToolWithWorkspaceAndState(5, t.TempDir(), NewTaskState())

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"cmd": []interface{}{"python3", "-c", "print('hi')"},
	})
	if err == nil {
		t.Fatal("expected python -c to be disallowed")
	}
	if !strings.Contains(err.Error(), "python -c is disallowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecTool_AllowsPythonScriptDirectly(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hello.py")
	if err := os.WriteFile(script, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	tool := NewExecToolWithWorkspaceAndState(5, dir, NewTaskState())
	out, err := tool.Execute(context.Background(), map[string]interface{}{
		"cmd": []interface{}{"python3", "hello.py"},
	})
	if err != nil {
		t.Fatalf("expected direct python script execution to work, got: %v", err)
	}
	if out != "ok" {
		t.Fatalf("unexpected output: %q", out)
	}
}
