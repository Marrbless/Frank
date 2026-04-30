package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestChannelsAllowlistAddListRemoveTelegram(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.AllowFrom = []string{"111"}
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	stdout, stderr, err := executePicobotCommand("channels", "allowlist", "add", "telegram", "222", "111")
	if err != nil {
		t.Fatalf("add command error = %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "telegram allowlist now has 2 entries") || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q, want add confirmation without stderr", stdout, stderr)
	}

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if got := strings.Join(saved.Channels.Telegram.AllowFrom, ","); got != "111,222" {
		t.Fatalf("telegram allowFrom = %v, want [111 222]", saved.Channels.Telegram.AllowFrom)
	}

	stdout, stderr, err = executePicobotCommand("channels", "allowlist", "list", "telegram")
	if err != nil {
		t.Fatalf("list command error = %v stderr=%q", err, stderr)
	}
	if stdout != "111\n222\n" || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q, want listed IDs without stderr", stdout, stderr)
	}

	stdout, stderr, err = executePicobotCommand("channels", "allowlist", "remove", "telegram", "111", "missing")
	if err != nil {
		t.Fatalf("remove command error = %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stdout, "telegram allowlist now has 1 entry") || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q, want remove confirmation without stderr", stdout, stderr)
	}
	readConfigFile(t, cfgPath, &saved)
	if got := strings.Join(saved.Channels.Telegram.AllowFrom, ","); got != "222" {
		t.Fatalf("telegram allowFrom = %v, want [222]", saved.Channels.Telegram.AllowFrom)
	}
}

func TestChannelsAllowlistSupportsSlackScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(config.DefaultConfig(), cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if _, stderr, err := executePicobotCommand("channels", "allowlist", "add", "slack-users", "U1"); err != nil {
		t.Fatalf("add slack-users error = %v stderr=%q", err, stderr)
	}
	if _, stderr, err := executePicobotCommand("channels", "allowlist", "add", "slack:channels", "C1"); err != nil {
		t.Fatalf("add slack:channels error = %v stderr=%q", err, stderr)
	}

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if got := strings.Join(saved.Channels.Slack.AllowUsers, ","); got != "U1" {
		t.Fatalf("slack allowUsers = %v, want [U1]", saved.Channels.Slack.AllowUsers)
	}
	if got := strings.Join(saved.Channels.Slack.AllowChannels, ","); got != "C1" {
		t.Fatalf("slack allowChannels = %v, want [C1]", saved.Channels.Slack.AllowChannels)
	}
}

func TestChannelsAllowlistWarnsWhenOpenModeIsEnabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.OpenMode = true
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	_, stderr, err := executePicobotCommand("channels", "allowlist", "add", "telegram", "111")
	if err != nil {
		t.Fatalf("add command error = %v stderr=%q", err, stderr)
	}
	if !strings.Contains(stderr, "open mode is enabled") {
		t.Fatalf("stderr = %q, want open-mode warning", stderr)
	}
}

func TestChannelsAllowlistRejectsUnknownScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(config.DefaultConfig(), cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	_, _, err := executePicobotCommand("channels", "allowlist", "add", "unknown", "123")
	if err == nil || !strings.Contains(err.Error(), "unknown allowlist scope") {
		t.Fatalf("error = %v, want unknown scope", err)
	}
}

func executePicobotCommand(args ...string) (string, string, error) {
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
