package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOldChannelSchemaDefaultsOpenModesClosed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".picobot", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	oldConfig := []byte(`{
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
}`)
	if err := os.WriteFile(cfgPath, oldConfig, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.Channels.Telegram.Enabled || cfg.Channels.Telegram.OpenMode {
		t.Fatalf("telegram enabled/openMode = %t/%t, want true/false", cfg.Channels.Telegram.Enabled, cfg.Channels.Telegram.OpenMode)
	}
	if !cfg.Channels.Discord.Enabled || cfg.Channels.Discord.OpenMode {
		t.Fatalf("discord enabled/openMode = %t/%t, want true/false", cfg.Channels.Discord.Enabled, cfg.Channels.Discord.OpenMode)
	}
	if !cfg.Channels.Slack.Enabled || cfg.Channels.Slack.OpenUserMode || cfg.Channels.Slack.OpenChannelMode {
		t.Fatalf("slack enabled/open modes = %t/%t/%t, want true/false/false", cfg.Channels.Slack.Enabled, cfg.Channels.Slack.OpenUserMode, cfg.Channels.Slack.OpenChannelMode)
	}
	if !cfg.Channels.WhatsApp.Enabled || cfg.Channels.WhatsApp.OpenMode {
		t.Fatalf("whatsapp enabled/openMode = %t/%t, want true/false", cfg.Channels.WhatsApp.Enabled, cfg.Channels.WhatsApp.OpenMode)
	}
}
