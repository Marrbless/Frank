package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCommandGeneratesSupportedShells(t *testing.T) {
	cases := []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: "__start_picobot"},
		{shell: "zsh", want: "#compdef picobot"},
		{shell: "fish", want: "complete -c picobot"},
		{shell: "powershell", want: "Register-ArgumentCompleter"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.shell, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs([]string{"completion", tc.shell})
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("completion %s error = %v", tc.shell, err)
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("completion %s output missing %q", tc.shell, tc.want)
			}
		})
	}
}

func TestCompletionCommandRejectsUnsupportedShell(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"completion", "tcsh"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("completion tcsh error = nil, want unsupported shell error")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("completion tcsh error = %q, want unsupported shell", err.Error())
	}
}
