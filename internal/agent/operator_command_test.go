package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStaticOperatorCommandHotUpdateHelpAliases(t *testing.T) {
	t.Parallel()

	for _, input := range append(staticHotUpdateHelpAliases(), " help v4 ") {
		got, ok := parseStaticOperatorCommand(input)
		if !ok {
			t.Fatalf("parseStaticOperatorCommand(%q) ok = false, want true", input)
		}
		if got.Kind != operatorCommandHotUpdateHelp {
			t.Fatalf("parseStaticOperatorCommand(%q).Kind = %q, want %q", input, got.Kind, operatorCommandHotUpdateHelp)
		}
	}
}

func TestHotUpdateHelpTextListsStaticHelpAliases(t *testing.T) {
	t.Parallel()

	help := hotUpdateOperatorHelpText()
	for _, alias := range staticHotUpdateHelpAliases() {
		if !strings.Contains(help, alias) {
			t.Fatalf("hotUpdateOperatorHelpText() missing alias %q\nhelp:\n%s", alias, help)
		}
	}
}

func TestHowToStartDocumentsStaticOperatorCommands(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "docs", "HOW_TO_START.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	doc := string(content)
	for _, snippet := range []string{
		"STATUS <job_id>",
		"PAUSE <job_id>",
		"RESUME <job_id>",
		"ABORT <job_id>",
		"APPROVE <job_id> <step_id>",
		"DENY <job_id> <step_id>",
		"SET_STEP <job_id> <step_id>",
	} {
		if !strings.Contains(doc, snippet) {
			t.Fatalf("HOW_TO_START.md missing operator command snippet %q", snippet)
		}
	}
	for _, alias := range staticHotUpdateHelpAliases() {
		if !strings.Contains(doc, alias) {
			t.Fatalf("HOW_TO_START.md missing hot-update help alias %q", alias)
		}
	}
}

func TestParseStaticOperatorCommandStatus(t *testing.T) {
	t.Parallel()

	got, ok := parseStaticOperatorCommand(" STATUS job-1 ")
	if !ok {
		t.Fatalf("parseStaticOperatorCommand(STATUS) ok = false, want true")
	}
	if got.Kind != operatorCommandStatus {
		t.Fatalf("Kind = %q, want %q", got.Kind, operatorCommandStatus)
	}
	if got.JobID != "job-1" {
		t.Fatalf("JobID = %q, want %q", got.JobID, "job-1")
	}
}

func TestParseStaticOperatorCommandRuntimeActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		kind  operatorCommandKind
		jobID string
	}{
		{"PAUSE job-1", operatorCommandPause, "job-1"},
		{"resume job-2", operatorCommandResume, "job-2"},
		{" Abort job-3 ", operatorCommandAbort, "job-3"},
	}
	for _, tt := range tests {
		got, ok := parseStaticOperatorCommand(tt.input)
		if !ok {
			t.Fatalf("parseStaticOperatorCommand(%q) ok = false, want true", tt.input)
		}
		if got.Kind != tt.kind || got.JobID != tt.jobID {
			t.Fatalf("parseStaticOperatorCommand(%q) = %#v, want kind %q job %q", tt.input, got, tt.kind, tt.jobID)
		}
	}
}

func TestParseStaticOperatorCommandApprovalDecisions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		kind   operatorCommandKind
		jobID  string
		stepID string
	}{
		{"APPROVE job-1 step-1", operatorCommandApprove, "job-1", "step-1"},
		{" deny job-2 step-2 ", operatorCommandDeny, "job-2", "step-2"},
	}
	for _, tt := range tests {
		got, ok := parseStaticOperatorCommand(tt.input)
		if !ok {
			t.Fatalf("parseStaticOperatorCommand(%q) ok = false, want true", tt.input)
		}
		if got.Kind != tt.kind || got.JobID != tt.jobID || got.StepID != tt.stepID {
			t.Fatalf("parseStaticOperatorCommand(%q) = %#v, want kind %q job %q step %q", tt.input, got, tt.kind, tt.jobID, tt.stepID)
		}
	}
}

func TestParseStaticOperatorCommandSetStep(t *testing.T) {
	t.Parallel()

	got, ok := parseStaticOperatorCommand(" set_step job-1 final ")
	if !ok {
		t.Fatal("parseStaticOperatorCommand(SET_STEP) ok = false, want true")
	}
	if got.Kind != operatorCommandSetStep || got.JobID != "job-1" || got.StepID != "final" {
		t.Fatalf("parseStaticOperatorCommand(SET_STEP) = %#v, want set-step job-1 final", got)
	}
}

func TestParseStaticOperatorCommandIgnoresMalformedRuntimeActions(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"STATUS", "PAUSE", "RESUME", "ABORT", "APPROVE job-1", "DENY job-1", "SET_STEP job-1"} {
		if got, ok := parseStaticOperatorCommand(input); ok {
			t.Fatalf("parseStaticOperatorCommand(%q) = %#v, true; want false", input, got)
		}
	}
}
