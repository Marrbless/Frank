package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestValidateConfigForStartup_DefaultConfigPasses(t *testing.T) {
	cfg := config.DefaultConfig()
	if errs := validateConfigForStartup(cfg); len(errs) != 0 {
		t.Fatalf("validateConfigForStartup() errors = %v, want none", errs)
	}
}

func TestValidateConfigForStartup_EnabledChannelsFailClosed(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Slack.Enabled = true
	cfg.Channels.WhatsApp.Enabled = true

	errs := validateConfigForStartup(cfg)
	if len(errs) != 5 {
		t.Fatalf("validateConfigForStartup() error count = %d, want 5: %v", len(errs), errs)
	}
	got := joinErrors(errs)
	for _, want := range []string{"telegram allowlist is empty", "discord allowlist is empty", "slack users allowlist is empty", "slack channels allowlist is empty", "whatsapp allowlist is empty"} {
		if !strings.Contains(got, want) {
			t.Fatalf("validateConfigForStartup() errors = %q, want %q", got, want)
		}
	}
}

func TestValidateConfigForStartup_ExplicitOpenModesPass(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.OpenMode = true
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.OpenMode = true
	cfg.Channels.Slack.Enabled = true
	cfg.Channels.Slack.OpenUserMode = true
	cfg.Channels.Slack.OpenChannelMode = true
	cfg.Channels.WhatsApp.Enabled = true
	cfg.Channels.WhatsApp.OpenMode = true

	if errs := validateConfigForStartup(cfg); len(errs) != 0 {
		t.Fatalf("validateConfigForStartup() errors = %v, want none", errs)
	}
}

func TestValidateConfigForStartup_OldChannelSchemaFailsClosed(t *testing.T) {
	var cfg config.Config
	if err := json.Unmarshal([]byte(`{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "telegram-token",
      "allowFrom": []
    },
    "discord": {
      "enabled": true,
      "token": "discord-token",
      "allowFrom": []
    },
    "slack": {
      "enabled": true,
      "appToken": "xapp-token",
      "botToken": "xoxb-token",
      "allowUsers": [],
      "allowChannels": []
    },
    "whatsapp": {
      "enabled": true,
      "dbPath": "~/.picobot/whatsapp.db",
      "allowFrom": []
    }
  }
}`), &cfg); err != nil {
		t.Fatalf("Unmarshal old config: %v", err)
	}

	errs := validateConfigForStartup(cfg)
	if len(errs) != 5 {
		t.Fatalf("validateConfigForStartup() error count = %d, want 5: %v", len(errs), errs)
	}
	got := joinErrors(errs)
	for _, want := range []string{"telegram allowlist is empty", "discord allowlist is empty", "slack users allowlist is empty", "slack channels allowlist is empty", "whatsapp allowlist is empty"} {
		if !strings.Contains(got, want) {
			t.Fatalf("validateConfigForStartup() errors = %q, want %q", got, want)
		}
	}
}

func TestConfigValidateCommandReportsAllowlistErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "validate"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("config validate error = nil, want validation failure")
	}
	if !strings.Contains(stderr.String(), "telegram allowlist is empty") {
		t.Fatalf("stderr = %q, want telegram allowlist error", stderr.String())
	}
}

func TestConfigValidateCommandWarnsAboutPermissiveSecretConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not reliable on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "telegram-token"
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if err := os.Chmod(cfgPath, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "validate"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("config validate error = %v, want nil", err)
	}
	if !strings.Contains(stderr.String(), "contains secrets") || !strings.Contains(stderr.String(), "0644") {
		t.Fatalf("stderr = %q, want config permission warning", stderr.String())
	}
}

func joinErrors(errs []error) string {
	var parts []string
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "\n")
}
