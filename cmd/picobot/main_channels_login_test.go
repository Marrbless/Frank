package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestPromptAllowlistOrOpenRequiresExplicitOpen(t *testing.T) {
	allow, open, ok := promptAllowlistOrOpen(
		bufio.NewReader(strings.NewReader("\n\n")),
		"allow: ",
		"open: ",
	)
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if len(allow) != 0 || open {
		t.Fatalf("allow=%v open=%t, want empty/false", allow, open)
	}
}

func TestPromptAllowlistOrOpenAcceptsAllowlist(t *testing.T) {
	allow, open, ok := promptAllowlistOrOpen(
		bufio.NewReader(strings.NewReader("123, 456\n")),
		"allow: ",
		"open: ",
	)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if open {
		t.Fatalf("open = true, want false")
	}
	if len(allow) != 2 || allow[0] != "123" || allow[1] != "456" {
		t.Fatalf("allow = %v, want [123 456]", allow)
	}
}

func TestPromptAllowlistOrOpenAcceptsExplicitOpen(t *testing.T) {
	allow, open, ok := promptAllowlistOrOpen(
		bufio.NewReader(strings.NewReader("\nOPEN\n")),
		"allow: ",
		"open: ",
	)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if len(allow) != 0 || !open {
		t.Fatalf("allow=%v open=%t, want empty/true", allow, open)
	}
}

func TestRequireAllowlistOrOpen(t *testing.T) {
	if err := requireAllowlistOrOpen("telegram", []string{"123"}, false); err != nil {
		t.Fatalf("allowlist case error = %v", err)
	}
	if err := requireAllowlistOrOpen("telegram", nil, true); err != nil {
		t.Fatalf("open mode case error = %v", err)
	}
	if err := requireAllowlistOrOpen("telegram", nil, false); err == nil {
		t.Fatalf("empty allowlist without open mode error = nil, want error")
	}
}

func TestSetupTelegramInteractiveAbortsWithoutAllowlistOrOpen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("token\n\n\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupTelegramInteractive(reader, cfg, cfgPath)
	})

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exists", err)
	}
	assertTranscriptContains(t, stdout, "Telegram Setup", "Allowed user IDs")
	assertTranscriptContains(t, stderr, "requires allowed user IDs or explicit OPEN acknowledgement")
}

func TestSetupTelegramInteractiveRecordsExplicitOpenMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("token\n\nOPEN\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupTelegramInteractive(reader, cfg, cfgPath)
	})

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if !saved.Channels.Telegram.Enabled {
		t.Fatalf("telegram enabled = false, want true")
	}
	if !saved.Channels.Telegram.OpenMode {
		t.Fatalf("telegram open mode = false, want true")
	}
	if len(saved.Channels.Telegram.AllowFrom) != 0 {
		t.Fatalf("telegram allowFrom = %v, want empty", saved.Channels.Telegram.AllowFrom)
	}
	assertTranscriptContains(t, stdout, "Telegram Setup", "Telegram configured")
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestSetupDiscordInteractiveRecordsAllowlistTranscript(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("discord-token\n111, 222\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupDiscordInteractive(reader, cfg, cfgPath)
	})

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if !saved.Channels.Discord.Enabled {
		t.Fatalf("discord enabled = false, want true")
	}
	if saved.Channels.Discord.OpenMode {
		t.Fatalf("discord open mode = true, want false")
	}
	if got := strings.Join(saved.Channels.Discord.AllowFrom, ","); got != "111,222" {
		t.Fatalf("discord allowFrom = %v, want [111 222]", saved.Channels.Discord.AllowFrom)
	}
	assertTranscriptContains(t, stdout, "Discord Setup", "Discord configured")
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestSetupDiscordInteractiveAbortTranscript(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("discord-token\n\n\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupDiscordInteractive(reader, cfg, cfgPath)
	})

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exists", err)
	}
	assertTranscriptContains(t, stdout, "Discord Setup", "Allowed user IDs")
	assertTranscriptContains(t, stderr, "requires allowed user IDs or explicit OPEN acknowledgement")
}

func TestSetupSlackInteractiveRecordsAllowlistTranscript(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("xapp-token\nxoxb-token\nU123, U456\nC123, G456\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupSlackInteractive(reader, cfg, cfgPath)
	})

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if !saved.Channels.Slack.Enabled {
		t.Fatalf("slack enabled = false, want true")
	}
	if saved.Channels.Slack.OpenUserMode || saved.Channels.Slack.OpenChannelMode {
		t.Fatalf("slack open modes = %t/%t, want false/false", saved.Channels.Slack.OpenUserMode, saved.Channels.Slack.OpenChannelMode)
	}
	if got := strings.Join(saved.Channels.Slack.AllowUsers, ","); got != "U123,U456" {
		t.Fatalf("slack allowUsers = %v, want [U123 U456]", saved.Channels.Slack.AllowUsers)
	}
	if got := strings.Join(saved.Channels.Slack.AllowChannels, ","); got != "C123,G456" {
		t.Fatalf("slack allowChannels = %v, want [C123 G456]", saved.Channels.Slack.AllowChannels)
	}
	assertTranscriptContains(t, stdout, "Slack Setup", "Slack configured")
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestSetupSlackInteractiveAbortTranscript(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("xapp-token\nxoxb-token\nU123\n\n\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupSlackInteractive(reader, cfg, cfgPath)
	})

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exists", err)
	}
	assertTranscriptContains(t, stdout, "Slack Setup", "Allowed channel IDs")
	assertTranscriptContains(t, stderr, "requires allowed channel IDs or explicit OPEN acknowledgement")
}

func TestSetupWhatsAppInteractiveRecordsAllowlist(t *testing.T) {
	oldSetupWhatsApp := setupWhatsApp
	setupWhatsApp = func(path string) error { return nil }
	t.Cleanup(func() { setupWhatsApp = oldSetupWhatsApp })

	cfg := config.DefaultConfig()
	cfg.Channels.WhatsApp.DBPath = filepath.Join(t.TempDir(), "whatsapp.db")
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("15551234567,lid-user\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupWhatsAppInteractive(reader, cfg, cfgPath)
	})

	var saved config.Config
	readConfigFile(t, cfgPath, &saved)
	if !saved.Channels.WhatsApp.Enabled {
		t.Fatalf("whatsapp enabled = false, want true")
	}
	if saved.Channels.WhatsApp.OpenMode {
		t.Fatalf("whatsapp open mode = true, want false")
	}
	if len(saved.Channels.WhatsApp.AllowFrom) != 2 {
		t.Fatalf("whatsapp allowFrom = %v, want two entries", saved.Channels.WhatsApp.AllowFrom)
	}
	assertTranscriptContains(t, stdout, "WhatsApp Setup", "WhatsApp setup complete")
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestSetupWhatsAppInteractiveAbortTranscript(t *testing.T) {
	oldSetupWhatsApp := setupWhatsApp
	setupCalled := false
	setupWhatsApp = func(path string) error {
		setupCalled = true
		return nil
	}
	t.Cleanup(func() { setupWhatsApp = oldSetupWhatsApp })

	cfg := config.DefaultConfig()
	cfg.Channels.WhatsApp.DBPath = filepath.Join(t.TempDir(), "whatsapp.db")
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	reader := bufio.NewReader(strings.NewReader("\n\n"))

	stdout, stderr := captureSetupTranscript(t, func() {
		setupWhatsAppInteractive(reader, cfg, cfgPath)
	})

	if setupCalled {
		t.Fatal("setupWhatsApp called, want abort before setup")
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("config path stat error = %v, want not exists", err)
	}
	assertTranscriptContains(t, stdout, "WhatsApp Setup", "Allowed WhatsApp sender IDs")
	assertTranscriptContains(t, stderr, "requires allowed sender IDs or explicit OPEN acknowledgement")
}

func readConfigFile(t *testing.T, path string, dst *config.Config) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
}

func captureSetupTranscript(t *testing.T, run func()) (string, string) {
	t.Helper()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	stdoutCh := make(chan string, 1)
	stderrCh := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(stdoutReader)
		stdoutCh <- string(data)
	}()
	go func() {
		data, _ := io.ReadAll(stderrReader)
		stderrCh <- string(data)
	}()

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	run()
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	stdout := <-stdoutCh
	stderr := <-stderrCh
	_ = stdoutReader.Close()
	_ = stderrReader.Close()
	return stdout, stderr
}

func assertTranscriptContains(t *testing.T, transcript string, wantParts ...string) {
	t.Helper()
	for _, want := range wantParts {
		if !strings.Contains(transcript, want) {
			t.Fatalf("transcript = %q, want substring %q", transcript, want)
		}
	}
}
