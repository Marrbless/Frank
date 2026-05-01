package modelsetup

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/local/picobot/internal/config"
)

type Detector struct {
	LookPath    func(string) (string, error)
	Stat        func(string) (os.FileInfo, error)
	ReadFile    func(string) ([]byte, error)
	UserHomeDir func() (string, error)
	Getenv      func(string) string
	GOOS        string
	GOARCH      string
}

func DefaultDetector() Detector {
	return Detector{
		LookPath:    exec.LookPath,
		Stat:        os.Stat,
		ReadFile:    os.ReadFile,
		UserHomeDir: os.UserHomeDir,
		Getenv:      os.Getenv,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
	}
}

func DetectEnvironment(configPath string, detector Detector) EnvSnapshot {
	detector = detector.withDefaults()
	env := MinimalUnknownEnvSnapshot(configPath)
	env.OS = detector.GOOS
	env.Arch = detector.GOARCH
	env.Platform = platformName(detector)
	env.Termux = detectTermux(detector)
	env.TermuxBoot = detectTermuxBoot(detector, env.Termux)
	env.Tmux = detectCommand(detector, "tmux")
	env.Ollama = detectCommand(detector, "ollama")
	env.LlamaCPP = detectLlamaCPP(detector)
	env.ExistingBootScripts = detectBootScripts(detector)
	applyConfigFacts(&env, detector)
	return normalizedEnvSnapshot(env)
}

func (d Detector) withDefaults() Detector {
	if d.LookPath == nil {
		d.LookPath = exec.LookPath
	}
	if d.Stat == nil {
		d.Stat = os.Stat
	}
	if d.ReadFile == nil {
		d.ReadFile = os.ReadFile
	}
	if d.UserHomeDir == nil {
		d.UserHomeDir = os.UserHomeDir
	}
	if d.Getenv == nil {
		d.Getenv = os.Getenv
	}
	if d.GOOS == "" {
		d.GOOS = runtime.GOOS
	}
	if d.GOARCH == "" {
		d.GOARCH = runtime.GOARCH
	}
	return d
}

func platformName(detector Detector) string {
	if detectTermux(detector) == StatePresent {
		return "android_termux_" + detector.GOARCH
	}
	if detector.GOOS == "" || detector.GOARCH == "" {
		return "unknown"
	}
	return detector.GOOS + "_" + detector.GOARCH
}

func detectTermux(detector Detector) State {
	prefix := detector.Getenv("PREFIX")
	home := detector.Getenv("HOME")
	if strings.Contains(prefix, "com.termux") || strings.Contains(home, "com.termux") {
		return StatePresent
	}
	if detector.GOOS == "android" {
		return StateUnknown
	}
	return StateUnsupported
}

func detectTermuxBoot(detector Detector, termux State) State {
	if termux == StateUnsupported {
		return StateUnsupported
	}
	home, err := detector.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return StateUnknown
	}
	state := pathState(detector, filepath.Join(home, ".termux", "boot"))
	if state == StateMissing && termux == StateUnknown {
		return StateUnknown
	}
	return state
}

func detectCommand(detector Detector, name string) State {
	_, err := detector.LookPath(name)
	if err == nil {
		return StatePresent
	}
	if errors.Is(err, exec.ErrNotFound) {
		return StateMissing
	}
	return StateUnknown
}

func detectLlamaCPP(detector Detector) State {
	names := []string{"llama-server", "llamacpp-server"}
	var found int
	var unknown bool
	for _, name := range names {
		state := detectCommand(detector, name)
		switch state {
		case StatePresent:
			found++
		case StateUnknown:
			unknown = true
		}
	}
	if found > 1 {
		return StateAmbiguous
	}
	if found == 1 {
		return StatePresent
	}
	if unknown {
		return StateUnknown
	}
	return StateMissing
}

func detectBootScripts(detector Detector) []string {
	home, err := detector.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".termux", "boot", "frank-model-runtime"),
		filepath.Join(home, ".termux", "boot", "frank-gateway"),
		filepath.Join(home, ".termux", "boot", "start-frank"),
	}
	var found []string
	for _, candidate := range candidates {
		if pathState(detector, candidate) == StatePresent {
			found = append(found, candidate)
		}
	}
	return found
}

func pathState(detector Detector, path string) State {
	_, err := detector.Stat(path)
	if err == nil {
		return StatePresent
	}
	if errors.Is(err, os.ErrNotExist) {
		return StateMissing
	}
	return StateUnknown
}

func applyConfigFacts(env *EnvSnapshot, detector Detector) {
	if strings.TrimSpace(env.ConfigPath) == "" {
		return
	}
	data, err := detector.ReadFile(env.ConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		env.UnsafeStates = append(env.UnsafeStates, "config_read_unknown")
		return
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		env.UnsafeStates = append(env.UnsafeStates, "config_parse_failed")
		return
	}
	if cfg.Providers.OpenAI != nil {
		env.ExistingProviders = append(env.ExistingProviders, config.LegacyProviderRef)
	}
	for ref := range cfg.Providers.Named {
		env.ExistingProviders = append(env.ExistingProviders, ref)
	}
	for ref := range cfg.Models {
		env.ExistingModels = append(env.ExistingModels, ref)
	}
	for ref := range cfg.ModelAliases {
		env.ExistingAliases = append(env.ExistingAliases, ref)
	}
	for ref := range cfg.LocalRuntimes {
		env.ExistingLocalRuntimes = append(env.ExistingLocalRuntimes, ref)
	}
}
