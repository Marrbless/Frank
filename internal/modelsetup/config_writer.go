package modelsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/local/picobot/internal/config"
)

type ConfigWriteOptions struct {
	ApproveReplacements bool
	Force               bool
	Now                 time.Time
}

type ConfigWriteResult struct {
	Status     PlanStatus
	ConfigPath string
	BackupPath string
	Changed    bool
	Messages   []string
}

func ApplyConfigPlan(plan Plan, opts ConfigWriteOptions) (ConfigWriteResult, error) {
	if plan.ConfigPatch == nil {
		return ConfigWriteResult{Status: PlanStatusSkipped}, fmt.Errorf("plan config patch is required")
	}
	path := strings.TrimSpace(plan.Environment.ConfigPath)
	if path == "" {
		return ConfigWriteResult{Status: PlanStatusBlocked}, fmt.Errorf("config path is required")
	}
	if plan.Status == PlanStatusBlocked {
		return ConfigWriteResult{Status: PlanStatusBlocked, ConfigPath: path}, fmt.Errorf("cannot write blocked plan")
	}
	cfg, existed, existingBytes, err := readConfigForPatch(path)
	if err != nil {
		return ConfigWriteResult{Status: PlanStatusFailed, ConfigPath: path}, err
	}
	patched, changed, err := applyPatchToConfig(cfg, *plan.ConfigPatch, opts)
	if err != nil {
		return ConfigWriteResult{Status: PlanStatusBlocked, ConfigPath: path}, err
	}
	if _, err := config.BuildModelRegistry(patched); err != nil {
		return ConfigWriteResult{Status: PlanStatusBlocked, ConfigPath: path}, fmt.Errorf("generated config failed V5 registry validation: %w", err)
	}
	if !changed {
		return ConfigWriteResult{Status: PlanStatusAlreadyPresent, ConfigPath: path, Changed: false}, nil
	}
	backupPath := ""
	if existed {
		backupPath, err = writeConfigBackup(path, existingBytes, opts.Now)
		if err != nil {
			return ConfigWriteResult{Status: PlanStatusFailed, ConfigPath: path}, err
		}
	}
	if err := writeConfigAtomic(path, patched); err != nil {
		return ConfigWriteResult{Status: PlanStatusFailed, ConfigPath: path, BackupPath: backupPath}, err
	}
	written, err := loadConfigFromPath(path)
	if err != nil {
		restoreBackup(path, backupPath)
		return ConfigWriteResult{Status: PlanStatusRolledBack, ConfigPath: path, BackupPath: backupPath}, fmt.Errorf("written config reload failed: %w", err)
	}
	if _, err := config.BuildModelRegistry(written); err != nil {
		restoreBackup(path, backupPath)
		return ConfigWriteResult{Status: PlanStatusRolledBack, ConfigPath: path, BackupPath: backupPath}, fmt.Errorf("written config failed V5 registry validation: %w", err)
	}
	return ConfigWriteResult{
		Status:     PlanStatusChanged,
		ConfigPath: path,
		BackupPath: backupPath,
		Changed:    true,
		Messages:   []string{"config write validated through V5 registry"},
	}, nil
}

func readConfigForPatch(path string) (config.Config, bool, []byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return config.DefaultConfig(), false, nil, nil
	}
	if err != nil {
		return config.Config{}, false, nil, err
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, true, data, err
	}
	return cfg, true, data, nil
}

func loadConfigFromPath(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, err
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func applyPatchToConfig(cfg config.Config, patch ConfigPatch, opts ConfigWriteOptions) (config.Config, bool, error) {
	patched := cloneConfig(cfg)
	if patched.Providers.Named == nil {
		patched.Providers.Named = make(map[string]config.ProviderConfig)
	}
	if patched.Models == nil {
		patched.Models = make(map[string]config.ModelProfileConfig)
	}
	if patched.ModelAliases == nil {
		patched.ModelAliases = make(map[string]string)
	}
	if patched.ModelRouting.Fallbacks == nil {
		patched.ModelRouting.Fallbacks = make(map[string][]string)
	}
	if patched.LocalRuntimes == nil {
		patched.LocalRuntimes = make(map[string]config.LocalRuntimeConfig)
	}
	providerRef, err := config.NormalizeProviderRef(patch.ProviderRef)
	if err != nil {
		return config.Config{}, false, err
	}
	modelRef, err := config.NormalizeModelRef(patch.ModelRef)
	if err != nil {
		return config.Config{}, false, err
	}
	providerKey, existingProvider, providerExists, err := findProviderByNormalizedRef(patched, providerRef)
	if err != nil {
		return config.Config{}, false, err
	}
	if providerExists && !providerEquivalent(existingProvider, patch.ProviderConfig) && !opts.ApproveReplacements && !opts.Force {
		return config.Config{}, false, fmt.Errorf("provider_ref %q already exists with different config; replacement requires approval", providerRef)
	}
	if providerRef == config.LegacyProviderRef {
		provider := patch.ProviderConfig
		patched.Providers.OpenAI = &provider
	} else {
		delete(patched.Providers.Named, providerKey)
		patched.Providers.Named[providerRef] = patch.ProviderConfig
	}

	modelKey, existingModel, modelExists, err := findModelByNormalizedRef(patched.Models, modelRef)
	if err != nil {
		return config.Config{}, false, err
	}
	if modelExists && !reflect.DeepEqual(existingModel, patch.ModelConfig) && !opts.ApproveReplacements && !opts.Force {
		return config.Config{}, false, fmt.Errorf("model_ref %q already exists with different config; replacement requires approval", modelRef)
	}
	delete(patched.Models, modelKey)
	patched.Models[modelRef] = patch.ModelConfig

	for alias, target := range patch.AliasRefs {
		normalizedAlias, err := config.NormalizeModelRef(alias)
		if err != nil {
			return config.Config{}, false, err
		}
		normalizedTarget, err := config.NormalizeModelRef(target)
		if err != nil {
			return config.Config{}, false, err
		}
		aliasKey, existingTarget, aliasExists, err := findAliasByNormalizedRef(patched.ModelAliases, normalizedAlias)
		if err != nil {
			return config.Config{}, false, err
		}
		if aliasExists && existingTarget != normalizedTarget && !opts.ApproveReplacements && !opts.Force {
			return config.Config{}, false, fmt.Errorf("alias %q already targets %q; replacement requires approval", normalizedAlias, existingTarget)
		}
		delete(patched.ModelAliases, aliasKey)
		patched.ModelAliases[normalizedAlias] = normalizedTarget
	}

	if patch.RuntimeRef != "" {
		runtimeRef, err := config.NormalizeProviderRef(patch.RuntimeRef)
		if err != nil {
			return config.Config{}, false, err
		}
		runtimeKey, existingRuntime, runtimeExists, err := findRuntimeByNormalizedRef(patched.LocalRuntimes, runtimeRef)
		if err != nil {
			return config.Config{}, false, err
		}
		if runtimeExists && !reflect.DeepEqual(existingRuntime, patch.RuntimeConfig) && !opts.ApproveReplacements && !opts.Force {
			return config.Config{}, false, fmt.Errorf("local runtime %q already exists with different config; replacement requires approval", runtimeRef)
		}
		delete(patched.LocalRuntimes, runtimeKey)
		patched.LocalRuntimes[runtimeRef] = patch.RuntimeConfig
	}

	patched.Agents.Defaults.Model = patch.DefaultModelRef
	patched.ModelRouting = patch.RoutingConfig
	return patched, !configsEquivalent(cfg, patched), nil
}

func findProviderByNormalizedRef(cfg config.Config, ref string) (string, config.ProviderConfig, bool, error) {
	var matches []string
	providers := make(map[string]config.ProviderConfig)
	if cfg.Providers.OpenAI != nil {
		providers[config.LegacyProviderRef] = *cfg.Providers.OpenAI
	}
	for key, provider := range cfg.Providers.Named {
		providers[key] = provider
	}
	for key, provider := range providers {
		normalized, err := config.NormalizeProviderRef(key)
		if err != nil {
			return "", config.ProviderConfig{}, false, err
		}
		if normalized == ref {
			matches = append(matches, key)
			if len(matches) == 1 {
				return key, provider, true, nil
			}
		}
	}
	return "", config.ProviderConfig{}, false, nil
}

func findModelByNormalizedRef(models map[string]config.ModelProfileConfig, ref string) (string, config.ModelProfileConfig, bool, error) {
	for key, model := range models {
		normalized, err := config.NormalizeModelRef(key)
		if err != nil {
			return "", config.ModelProfileConfig{}, false, err
		}
		if normalized == ref {
			return key, model, true, nil
		}
	}
	return "", config.ModelProfileConfig{}, false, nil
}

func findAliasByNormalizedRef(aliases map[string]string, ref string) (string, string, bool, error) {
	for key, target := range aliases {
		normalized, err := config.NormalizeModelRef(key)
		if err != nil {
			return "", "", false, err
		}
		if normalized == ref {
			normalizedTarget, err := config.NormalizeModelRef(target)
			if err != nil {
				return "", "", false, err
			}
			return key, normalizedTarget, true, nil
		}
	}
	return "", "", false, nil
}

func findRuntimeByNormalizedRef(runtimes map[string]config.LocalRuntimeConfig, ref string) (string, config.LocalRuntimeConfig, bool, error) {
	for key, runtime := range runtimes {
		normalized, err := config.NormalizeProviderRef(key)
		if err != nil {
			return "", config.LocalRuntimeConfig{}, false, err
		}
		if normalized == ref {
			return key, runtime, true, nil
		}
	}
	return "", config.LocalRuntimeConfig{}, false, nil
}

func providerEquivalent(left, right config.ProviderConfig) bool {
	left.Type = firstNonEmpty(left.Type, config.ProviderTypeOpenAICompatible)
	right.Type = firstNonEmpty(right.Type, config.ProviderTypeOpenAICompatible)
	return reflect.DeepEqual(left, right)
}

func configsEquivalent(left, right config.Config) bool {
	leftBytes, _ := json.Marshal(left)
	rightBytes, _ := json.Marshal(right)
	return bytes.Equal(leftBytes, rightBytes)
}

func cloneConfig(cfg config.Config) config.Config {
	data, _ := json.Marshal(cfg)
	var cloned config.Config
	_ = json.Unmarshal(data, &cloned)
	return cloned
}

func writeConfigBackup(path string, data []byte, now time.Time) (string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	backupPath := fmt.Sprintf("%s.v6-backup-%s", path, now.UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", err
	}
	return backupPath, nil
}

func writeConfigAtomic(path string, cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-v6-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func restoreBackup(path, backupPath string) {
	if backupPath == "" {
		return
	}
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}
