package tools

import (
	"context"
	"strings"
	"testing"
)

func TestExecTool_DisallowsShellInterpreter(t *testing.T) {
	tool := NewExecToolWithWorkspaceAndState(5, t.TempDir(), NewTaskState())

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"cmd": []interface{}{"sh", "-lc", "echo hi"},
	})
	if err == nil {
		t.Fatal("expected shell interpreter to be disallowed")
	}
	if !strings.Contains(err.Error(), "shell interpreters are disallowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecTool_StillAllowsFrankHelper(t *testing.T) {
	tool := NewExecToolWithWorkspaceAndState(5, t.TempDir(), NewTaskState())

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"cmd": []interface{}{"frank_new_project", "hello"},
	})
	if err != nil {
		t.Fatalf("expected frank_new_project to work, got: %v", err)
	}
}
