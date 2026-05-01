package modelsetup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/local/picobot/internal/config"
)

func TestDetectEnvironmentReturnsTypedStatesAndConfigFacts(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := config.DefaultConfig()
	cfg.Providers.Named = map[string]config.ProviderConfig{
		"ollama_phone": {Type: config.ProviderTypeOpenAICompatible, APIKey: "ollama", APIBase: "http://127.0.0.1:11434/v1"},
	}
	cfg.Models = map[string]config.ModelProfileConfig{
		"local_fast": {Provider: "ollama_phone", ProviderModel: "qwen3:1.7b"},
	}
	cfg.ModelAliases = map[string]string{"phone": "local_fast"}
	cfg.LocalRuntimes = map[string]config.LocalRuntimeConfig{"ollama_phone": {Provider: "ollama_phone"}}
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, ".termux", "boot"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	bootScript := filepath.Join(home, ".termux", "boot", "frank-model-runtime")
	if err := os.WriteFile(bootScript, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	detector := fakeDetector(map[string]State{
		"tmux":            StatePresent,
		"ollama":          StateMissing,
		"llama-server":    StatePresent,
		"llamacpp-server": StateMissing,
	}, home)
	detector.GOOS = "android"
	detector.GOARCH = "arm64"
	detector.Getenv = func(key string) string {
		switch key {
		case "PREFIX":
			return "/data/data/com.termux/files/usr"
		case "HOME":
			return home
		default:
			return ""
		}
	}

	env := DetectEnvironment(cfgPath, detector)
	if env.Platform != "android_termux_arm64" {
		t.Fatalf("Platform = %q, want android_termux_arm64", env.Platform)
	}
	if env.Termux != StatePresent || env.TermuxBoot != StatePresent || env.Tmux != StatePresent || env.Ollama != StateMissing || env.LlamaCPP != StatePresent {
		t.Fatalf("states = termux:%s boot:%s tmux:%s ollama:%s llama:%s", env.Termux, env.TermuxBoot, env.Tmux, env.Ollama, env.LlamaCPP)
	}
	for _, want := range []string{"openai", "ollama_phone"} {
		if !containsStringExact(env.ExistingProviders, want) {
			t.Fatalf("ExistingProviders = %#v, want %q", env.ExistingProviders, want)
		}
	}
	if !containsStringExact(env.ExistingModels, "local_fast") || !containsStringExact(env.ExistingAliases, "phone") || !containsStringExact(env.ExistingLocalRuntimes, "ollama_phone") {
		t.Fatalf("config facts missing: %#v %#v %#v", env.ExistingModels, env.ExistingAliases, env.ExistingLocalRuntimes)
	}
	if !containsStringExact(env.ExistingBootScripts, bootScript) {
		t.Fatalf("ExistingBootScripts = %#v, want %q", env.ExistingBootScripts, bootScript)
	}
}

func TestDetectEnvironmentSupportsUnknownUnsupportedAndAmbiguousStates(t *testing.T) {
	detector := fakeDetector(map[string]State{
		"tmux":            StateUnknown,
		"ollama":          StateMissing,
		"llama-server":    StatePresent,
		"llamacpp-server": StatePresent,
	}, t.TempDir())
	detector.GOOS = "linux"
	detector.GOARCH = "amd64"

	env := DetectEnvironment(filepath.Join(t.TempDir(), "missing.json"), detector)
	if env.Termux != StateUnsupported {
		t.Fatalf("Termux = %q, want unsupported", env.Termux)
	}
	if env.Tmux != StateUnknown {
		t.Fatalf("Tmux = %q, want unknown", env.Tmux)
	}
	if env.LlamaCPP != StateAmbiguous {
		t.Fatalf("LlamaCPP = %q, want ambiguous", env.LlamaCPP)
	}
}

func fakeDetector(commands map[string]State, home string) Detector {
	return Detector{
		LookPath: func(name string) (string, error) {
			switch commands[name] {
			case StatePresent:
				return filepath.Join("/usr/bin", name), nil
			case StateUnknown:
				return "", errors.New("lookup failed")
			default:
				return "", exec.ErrNotFound
			}
		},
		Stat:        os.Stat,
		ReadFile:    os.ReadFile,
		UserHomeDir: func() (string, error) { return home, nil },
		Getenv: func(key string) string {
			if key == "HOME" {
				return home
			}
			return ""
		},
		GOOS:   "linux",
		GOARCH: "amd64",
	}
}

func containsStringExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
