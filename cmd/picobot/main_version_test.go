package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommandBuiltBinaryDefaultMetadata(t *testing.T) {
	binary := buildPicobotVersionTestBinary(t, nil)

	output := runPicobotVersionTestBinary(t, binary)
	want := "\U0001f916 picobot v" + version + "\ncommit: unknown\ndate: unknown\n"
	if output != want {
		t.Fatalf("version output = %q, want %q", output, want)
	}
}

func TestVersionCommandBuiltBinaryLdflagsMetadata(t *testing.T) {
	binary := buildPicobotVersionTestBinary(t, []string{
		"-X", "main.version=9.8.7-test",
		"-X", "main.buildCommit=abc1234",
		"-X", "main.buildDate=2026-04-30T18:13:56Z",
	})

	output := runPicobotVersionTestBinary(t, binary)
	want := "\U0001f916 picobot v9.8.7-test\ncommit: abc1234\ndate: 2026-04-30T18:13:56Z\n"
	if output != want {
		t.Fatalf("version output = %q, want %q", output, want)
	}
}

func buildPicobotVersionTestBinary(t *testing.T, ldflagParts []string) string {
	t.Helper()

	binaryName := "picobot-version-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	args := []string{"build", "-buildvcs=false"}
	if len(ldflagParts) > 0 {
		args = append(args, "-ldflags", strings.Join(ldflagParts, " "))
	}
	args = append(args, "-o", binaryPath, "./cmd/picobot")

	cmd := exec.Command("go", args...)
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %v failed: %v\n%s", args, err, output)
	}
	return binaryPath
}

func runPicobotVersionTestBinary(t *testing.T, binaryPath string) string {
	t.Helper()

	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s version failed: %v\n%s", binaryPath, err, output)
	}
	return string(output)
}
