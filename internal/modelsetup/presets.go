package modelsetup

import (
	"fmt"
	"sort"

	"github.com/local/picobot/internal/config"
)

const (
	PresetPhoneOllamaTiny      = "phone-ollama-tiny"
	PresetPhoneLlamaCPPTiny    = "phone-llamacpp-tiny"
	PresetDesktopOllamaLocal   = "desktop-ollama-local"
	PresetDesktopLlamaCPPLocal = "desktop-llamacpp-local"
	PresetCloudOpenRouter      = "cloud-openrouter"
	PresetCloudOpenAI          = "cloud-openai"
	PresetMixedLocalCloudSafe  = "mixed-local-cloud-safe"
	PresetLANLlamaCPPLocal     = "lan-llamacpp-local"
)

func Catalog() []Preset {
	localTinyCaps := config.ModelCapabilities{
		Local:           true,
		Offline:         true,
		SupportsTools:   false,
		ContextTokens:   2048,
		MaxOutputTokens: 512,
		AuthorityTier:   config.ModelAuthorityLow,
		CostTier:        config.ModelCostFree,
		LatencyTier:     config.ModelLatencySlow,
	}
	cloudCaps := config.ModelCapabilities{
		Local:           false,
		Offline:         false,
		SupportsTools:   true,
		ContextTokens:   1000000,
		MaxOutputTokens: 8192,
		AuthorityTier:   config.ModelAuthorityHigh,
		CostTier:        config.ModelCostStandard,
		LatencyTier:     config.ModelLatencyNormal,
	}
	localTemp := 0.3
	cloudTemp := 0.5
	return []Preset{
		{
			Name:               PresetPhoneOllamaTiny,
			DisplayName:        "Phone Ollama tiny local model",
			DefaultSafe:        true,
			RuntimeKind:        RuntimeKindOllama,
			ProviderRef:        "ollama_phone",
			ModelRef:           "local_fast",
			ProviderModel:      "qwen3:1.7b",
			ModelSource:        "ollama",
			ExpectedDiskImpact: "unknown_until_manifest_or_pull",
			ExpectedRAMRange:   "phone_tiny",
			Capabilities:       localTinyCaps,
			Request: config.ModelRequestConfig{
				MaxTokens:   512,
				Temperature: &localTemp,
				TimeoutS:    300,
			},
			BaseURL:           "http://127.0.0.1:11434/v1",
			HealthURL:         "http://127.0.0.1:11434/api/tags",
			BindAddress:       "127.0.0.1",
			Port:              11434,
			BootSupported:     true,
			DownloadsRequired: true,
			SafetyNotes: []string{
				"local model defaults to low authority and no tools",
				"automatic install/pull requires a later approved package path or manifest",
			},
		},
		{
			Name:                  PresetPhoneLlamaCPPTiny,
			DisplayName:           "Phone llama.cpp tiny local model",
			DefaultSafe:           true,
			RuntimeKind:           RuntimeKindLlamaCPP,
			ProviderRef:           "llamacpp_phone",
			ModelRef:              "local_fast",
			ProviderModel:         "qwen2.5-0.5b-instruct-q4_k_m",
			ModelSource:           "manifest_or_existing_gguf",
			ManifestID:            ManifestLlamaCPPAndroidARM64B8994,
			ModelManifestID:       ManifestQwen25TinyGGUF,
			ExpectedDiskImpact:    "552000000",
			ExpectedRAMRange:      "phone_tiny",
			Capabilities:          localTinyCaps,
			Request:               config.ModelRequestConfig{MaxTokens: 512, Temperature: &localTemp, TimeoutS: 300},
			BaseURL:               "http://127.0.0.1:8080/v1",
			HealthURL:             "http://127.0.0.1:8080/health",
			BindAddress:           "127.0.0.1",
			Port:                  8080,
			BootSupported:         true,
			SupportsRegisterLocal: true,
			SafetyNotes: []string{
				"register-existing is the first safe llama.cpp path",
				"automatic binary/model downloads use checked-in immutable manifests with SHA256 verification",
				"runtime binds only to 127.0.0.1",
			},
		},
		{
			Name:               PresetDesktopOllamaLocal,
			DisplayName:        "Desktop Ollama local model",
			DefaultSafe:        true,
			RuntimeKind:        RuntimeKindOllama,
			ProviderRef:        "ollama_local",
			ModelRef:           "local_fast",
			ProviderModel:      "qwen3:1.7b",
			ModelSource:        "ollama",
			ExpectedDiskImpact: "unknown_until_manifest_or_pull",
			ExpectedRAMRange:   "desktop_local",
			Capabilities:       localTinyCaps,
			Request:            config.ModelRequestConfig{MaxTokens: 512, Temperature: &localTemp, TimeoutS: 300},
			BaseURL:            "http://127.0.0.1:11434/v1",
			HealthURL:          "http://127.0.0.1:11434/api/tags",
			BindAddress:        "127.0.0.1",
			Port:               11434,
			DownloadsRequired:  true,
			SafetyNotes:        []string{"cloud fallback remains disabled by default"},
		},
		{
			Name:                  PresetDesktopLlamaCPPLocal,
			DisplayName:           "Desktop llama.cpp local model",
			DefaultSafe:           true,
			RuntimeKind:           RuntimeKindLlamaCPP,
			ProviderRef:           "llamacpp_local",
			ModelRef:              "local_fast",
			ProviderModel:         "qwen3-1.7b-q8_0",
			ModelSource:           "existing_gguf",
			ExpectedDiskImpact:    "operator_registered_model_file",
			ExpectedRAMRange:      "desktop_local",
			Capabilities:          localTinyCaps,
			Request:               config.ModelRequestConfig{MaxTokens: 512, Temperature: &localTemp, TimeoutS: 300},
			BaseURL:               "http://127.0.0.1:8080/v1",
			HealthURL:             "http://127.0.0.1:8080/health",
			BindAddress:           "127.0.0.1",
			Port:                  8080,
			SupportsRegisterLocal: true,
			SafetyNotes:           []string{"register-existing is the first safe llama.cpp path"},
		},
		{
			Name:              PresetCloudOpenRouter,
			DisplayName:       "OpenRouter cloud profile stubs",
			DefaultSafe:       true,
			RuntimeKind:       RuntimeKindCloud,
			ProviderRef:       "openrouter",
			ModelRef:          "cloud_reasoning",
			ProviderModel:     "openai/gpt-5.4-mini",
			ModelSource:       "cloud_stub",
			Capabilities:      cloudCaps,
			Request:           config.ModelRequestConfig{MaxTokens: 8192, Temperature: &cloudTemp, TimeoutS: 120},
			BaseURL:           "https://openrouter.ai/api/v1",
			CloudKeysRequired: true,
			SafetyNotes:       []string{"creates provider/model stubs only; key values are not collected"},
		},
		{
			Name:              PresetCloudOpenAI,
			DisplayName:       "OpenAI cloud profile stubs",
			DefaultSafe:       true,
			RuntimeKind:       RuntimeKindCloud,
			ProviderRef:       "openai",
			ModelRef:          "cloud_reasoning",
			ProviderModel:     "gpt-5.4-mini",
			ModelSource:       "cloud_stub",
			Capabilities:      cloudCaps,
			Request:           config.ModelRequestConfig{MaxTokens: 8192, Temperature: &cloudTemp, TimeoutS: 120},
			BaseURL:           "https://api.openai.com/v1",
			CloudKeysRequired: true,
			SafetyNotes:       []string{"creates provider/model stubs only; key values are not collected"},
		},
		{
			Name:               PresetMixedLocalCloudSafe,
			DisplayName:        "Mixed local plus cloud-safe stubs",
			DefaultSafe:        true,
			RuntimeKind:        RuntimeKindOllama,
			ProviderRef:        "ollama_phone",
			ModelRef:           "local_fast",
			ProviderModel:      "qwen3:1.7b",
			ModelSource:        "ollama",
			ExpectedDiskImpact: "unknown_until_manifest_or_pull",
			ExpectedRAMRange:   "phone_tiny",
			Capabilities:       localTinyCaps,
			Request:            config.ModelRequestConfig{MaxTokens: 512, Temperature: &localTemp, TimeoutS: 300},
			BaseURL:            "http://127.0.0.1:11434/v1",
			HealthURL:          "http://127.0.0.1:11434/api/tags",
			BindAddress:        "127.0.0.1",
			Port:               11434,
			BootSupported:      true,
			DownloadsRequired:  true,
			SafetyNotes: []string{
				"adds local profile and cloud stubs",
				"cloud fallback from local remains disabled unless explicitly approved",
			},
		},
		{
			Name:                  PresetLANLlamaCPPLocal,
			DisplayName:           "LAN llama.cpp local model",
			DefaultSafe:           false,
			ExplicitlyGated:       true,
			RuntimeKind:           RuntimeKindLlamaCPP,
			ProviderRef:           "llamacpp_lan",
			ModelRef:              "local_lan",
			ProviderModel:         "qwen3-1.7b-q8_0",
			ModelSource:           "existing_gguf",
			ExpectedDiskImpact:    "operator_registered_model_file",
			ExpectedRAMRange:      "lan_local",
			Capabilities:          localTinyCaps,
			Request:               config.ModelRequestConfig{MaxTokens: 512, Temperature: &localTemp, TimeoutS: 300},
			BaseURL:               "http://127.0.0.1:8080/v1",
			HealthURL:             "http://127.0.0.1:8080/health",
			BindAddress:           "127.0.0.1",
			Port:                  8080,
			SupportsRegisterLocal: true,
			SafetyNotes: []string{
				"LAN binding is blocked unless separately approved",
				"default generated runtime still binds to 127.0.0.1",
			},
		},
	}
}

func PresetByName(name string) (Preset, bool) {
	for _, preset := range Catalog() {
		if preset.Name == name {
			return preset, true
		}
	}
	return Preset{}, false
}

func DefaultSafePresetNames() []string {
	var names []string
	for _, preset := range Catalog() {
		if preset.DefaultSafe {
			names = append(names, preset.Name)
		}
	}
	sort.Strings(names)
	return names
}

func GatedPresetNames() []string {
	var names []string
	for _, preset := range Catalog() {
		if preset.ExplicitlyGated {
			names = append(names, preset.Name)
		}
	}
	sort.Strings(names)
	return names
}

func ValidateCatalog() error {
	seen := make(map[string]bool)
	for _, preset := range Catalog() {
		if preset.Name == "" {
			return fmt.Errorf("preset name is required")
		}
		if seen[preset.Name] {
			return fmt.Errorf("duplicate preset %q", preset.Name)
		}
		seen[preset.Name] = true
		if _, err := config.NormalizeProviderRef(preset.ProviderRef); err != nil {
			return fmt.Errorf("preset %q provider_ref: %w", preset.Name, err)
		}
		if _, err := config.NormalizeModelRef(preset.ModelRef); err != nil {
			return fmt.Errorf("preset %q model_ref: %w", preset.Name, err)
		}
		if preset.RuntimeKind != RuntimeKindCloud && preset.BindAddress != "127.0.0.1" {
			return fmt.Errorf("preset %q default bind address %q is not localhost", preset.Name, preset.BindAddress)
		}
		if preset.RuntimeKind != RuntimeKindCloud {
			if preset.Capabilities.SupportsTools {
				return fmt.Errorf("preset %q local model supports tools by default", preset.Name)
			}
			if preset.Capabilities.AuthorityTier != config.ModelAuthorityLow {
				return fmt.Errorf("preset %q local authority tier = %q, want low", preset.Name, preset.Capabilities.AuthorityTier)
			}
			if preset.CloudFallbackDefault {
				return fmt.Errorf("preset %q enables cloud fallback by default", preset.Name)
			}
		}
	}
	required := []string{
		PresetPhoneOllamaTiny,
		PresetPhoneLlamaCPPTiny,
		PresetDesktopOllamaLocal,
		PresetDesktopLlamaCPPLocal,
		PresetCloudOpenRouter,
		PresetCloudOpenAI,
		PresetMixedLocalCloudSafe,
	}
	for _, name := range required {
		preset, ok := PresetByName(name)
		if !ok {
			return fmt.Errorf("required preset %q is missing", name)
		}
		if !preset.DefaultSafe {
			return fmt.Errorf("required preset %q is not default-safe", name)
		}
	}
	lan, ok := PresetByName(PresetLANLlamaCPPLocal)
	if !ok {
		return fmt.Errorf("optional gated preset %q is missing", PresetLANLlamaCPPLocal)
	}
	if !lan.ExplicitlyGated || lan.DefaultSafe {
		return fmt.Errorf("preset %q must be explicitly gated and not default-safe", PresetLANLlamaCPPLocal)
	}
	return nil
}
