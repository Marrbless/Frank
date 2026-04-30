package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"
)

const defaultOpenAIAPIKeyPlaceholder = "REPLACE_WITH_REAL_API_KEY"

// ConfigContainsSecrets reports whether cfg has plaintext credentials worth protecting on disk.
func ConfigContainsSecrets(cfg Config) bool {
	if hasSecret(cfg.Channels.Telegram.Token) ||
		hasSecret(cfg.Channels.Discord.Token) ||
		hasSecret(cfg.Channels.Slack.AppToken) ||
		hasSecret(cfg.Channels.Slack.BotToken) {
		return true
	}
	if cfg.Providers.OpenAI != nil && hasNonPlaceholderSecret(cfg.Providers.OpenAI.APIKey, defaultOpenAIAPIKeyPlaceholder) {
		return true
	}
	for _, server := range cfg.MCPServers {
		for key, value := range server.Headers {
			if hasSecret(value) && headerNameLooksSecret(key) {
				return true
			}
		}
	}
	return false
}

// ConfigFilePermissionWarning returns a warning when a secret-bearing config is readable by group or other users.
func ConfigFilePermissionWarning(path string, cfg Config) (string, error) {
	if runtime.GOOS == "windows" || !ConfigContainsSecrets(cfg) {
		return "", nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("config path %q is a directory", path)
	}
	mode := info.Mode().Perm()
	if mode&0o077 == 0 {
		return "", nil
	}
	return fmt.Sprintf("config file %s contains secrets and has permissive mode %04o; run chmod 600 %s", path, mode, path), nil
}

func hasSecret(value string) bool {
	return strings.TrimSpace(value) != ""
}

func hasNonPlaceholderSecret(value string, placeholders ...string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	for _, placeholder := range placeholders {
		if trimmed == placeholder {
			return false
		}
	}
	return true
}

func headerNameLooksSecret(name string) bool {
	normalized := strings.ToLower(name)
	return strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "api-key") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}
