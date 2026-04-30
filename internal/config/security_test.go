package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigFilePermissionWarningDetectsSecretWorldReadableConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not reliable on windows")
	}
	cfg := DefaultConfig()
	cfg.Channels.Telegram.Token = "telegram-token"
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(cfg, path); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	warning, err := ConfigFilePermissionWarning(path, cfg)
	if err != nil {
		t.Fatalf("ConfigFilePermissionWarning() error = %v", err)
	}
	if !strings.Contains(warning, "contains secrets") || !strings.Contains(warning, "0644") {
		t.Fatalf("warning = %q, want secret permission warning with mode", warning)
	}
}

func TestConfigFilePermissionWarningIgnoresDefaultPlaceholder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not reliable on windows")
	}
	cfg := DefaultConfig()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(cfg, path); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	warning, err := ConfigFilePermissionWarning(path, cfg)
	if err != nil {
		t.Fatalf("ConfigFilePermissionWarning() error = %v", err)
	}
	if warning != "" {
		t.Fatalf("warning = %q, want empty", warning)
	}
}
