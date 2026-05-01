package modelsetup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateTermuxBootScriptsAreDeterministicAndSyntaxValid(t *testing.T) {
	runtime, err := GenerateTermuxModelRuntimeScript("$HOME/Frank", "frank-model", "ollama serve")
	if err != nil {
		t.Fatalf("GenerateTermuxModelRuntimeScript() error = %v", err)
	}
	runtimeAgain, err := GenerateTermuxModelRuntimeScript("$HOME/Frank", "frank-model", "ollama serve")
	if err != nil {
		t.Fatalf("GenerateTermuxModelRuntimeScript() second error = %v", err)
	}
	if runtime != runtimeAgain {
		t.Fatalf("runtime script is not deterministic")
	}
	for _, want := range []string{"#!/data/data/com.termux/files/usr/bin/sh", "tmux has-session", "ollama serve"} {
		if !strings.Contains(runtime, want) {
			t.Fatalf("runtime script = %q, want %q", runtime, want)
		}
	}
	gateway, err := GenerateTermuxGatewayScript("$HOME/Frank", "frank", "./picobot gateway")
	if err != nil {
		t.Fatalf("GenerateTermuxGatewayScript() error = %v", err)
	}
	assertShellSyntax(t, runtime)
	assertShellSyntax(t, gateway)
}

func TestGenerateTermuxScriptsRejectUnsafeDefaults(t *testing.T) {
	if _, err := GenerateTermuxModelRuntimeScript("$HOME/Frank", "frank-model", "llama-server --host 0.0.0.0"); err == nil {
		t.Fatal("GenerateTermuxModelRuntimeScript() error = nil, want unsafe bind rejection")
	}
	if _, err := GenerateTermuxGatewayScript("$HOME/Frank", "frank", "./picobot gateway --mission-resume-approved"); err == nil {
		t.Fatal("GenerateTermuxGatewayScript() error = nil, want mission resume approval rejection")
	}
}

func TestWriteBootScriptPreservesExistingUnlessForced(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".termux", "boot", "frank-model-runtime")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("existing\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}
	result, err := WriteBootScript(path, "new\n", BootScriptWriteOptions{})
	if err == nil {
		t.Fatal("WriteBootScript() error = nil, want blocked overwrite")
	}
	if result.Status != PlanStatusBlocked {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "existing\n" {
		t.Fatalf("existing script was overwritten: %q", string(data))
	}

	result, err = WriteBootScript(path, "new\n", BootScriptWriteOptions{Force: true, Now: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("WriteBootScript(force) error = %v", err)
	}
	if result.Status != PlanStatusChanged || result.BackupPath == "" {
		t.Fatalf("result = %#v, want changed with backup", result)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("ReadFile(backup) error = %v", err)
	}
	if string(backup) != "existing\n" {
		t.Fatalf("backup = %q, want existing", string(backup))
	}
}

func TestWriteBootScriptAlreadyPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".termux", "boot", "frank-model-runtime")
	script := "#!/data/data/com.termux/files/usr/bin/sh\n"
	result, err := WriteBootScript(path, script, BootScriptWriteOptions{})
	if err != nil {
		t.Fatalf("WriteBootScript(first) error = %v", err)
	}
	if result.Status != PlanStatusChanged {
		t.Fatalf("first status = %q, want changed", result.Status)
	}
	result, err = WriteBootScript(path, script, BootScriptWriteOptions{})
	if err != nil {
		t.Fatalf("WriteBootScript(second) error = %v", err)
	}
	if result.Status != PlanStatusAlreadyPresent {
		t.Fatalf("second status = %q, want already_present", result.Status)
	}
}

func assertShellSyntax(t *testing.T, script string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available for syntax check")
	}
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile(script) error = %v", err)
	}
	if out, err := exec.Command(sh, "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("sh -n failed: %v\n%s", err, string(out))
	}
}
