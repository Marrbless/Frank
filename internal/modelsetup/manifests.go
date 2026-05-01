package modelsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ManifestLlamaCPPAndroidARM64B8994 = "llamacpp-android-arm64-b8994-cpu"
	ManifestQwen25TinyGGUF            = "qwen2.5-0.5b-instruct-q4_k_m-gguf-qwen-df5bf013"
	ManifestOllamaTermuxManual        = "ollama-termux-arm64-manual-required"

	LlamaCPPRuntimeArchivePath = "$HOME/.local/frank/artifacts/llama-b8994-bin-android-arm64.tar.gz"
	LlamaCPPRuntimeInstallDir  = "$HOME/.local/frank/llama.cpp/b8994"
	Qwen25TinyGGUFPath         = "$HOME/.local/frank/models/qwen2.5-0.5b-instruct-q4_k_m.gguf"
)

type ArtifactManifest struct {
	ID             string
	SourceURL      string
	Version        string
	ChecksumSHA256 string
	SizeBytes      int64
	LicenseNotes   string
	Platform       string
	Arch           string
	InstallCommand []string
	SafetyNotes    []string
}

type ManifestRegistry map[string]ArtifactManifest

type Downloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
}

type HTTPDownloader struct {
	Client *http.Client
}

func BuiltinManifests() ManifestRegistry {
	return ManifestRegistry{
		ManifestLlamaCPPAndroidARM64B8994: {
			ID:             ManifestLlamaCPPAndroidARM64B8994,
			SourceURL:      "https://github.com/ggml-org/llama.cpp/releases/download/b8994/llama-b8994-bin-android-arm64.tar.gz",
			Version:        "b8994",
			ChecksumSHA256: "bf0445968910d36ef85cc273501601189db3ed052a57c0393ba56bd346ab7d54",
			SizeBytes:      61000000,
			LicenseNotes:   "llama.cpp release artifact. Frank should preserve bundled license files during install.",
			Platform:       "Android/Termux",
			Arch:           "arm64",
			InstallCommand: []string{
				`mkdir -p "$HOME/.local/frank/artifacts" "$HOME/.local/frank/llama.cpp/b8994"`,
				`curl -fL -o "$HOME/.local/frank/artifacts/llama-b8994-bin-android-arm64.tar.gz" "https://github.com/ggml-org/llama.cpp/releases/download/b8994/llama-b8994-bin-android-arm64.tar.gz"`,
				`echo 'bf0445968910d36ef85cc273501601189db3ed052a57c0393ba56bd346ab7d54  '"$HOME"'/.local/frank/artifacts/llama-b8994-bin-android-arm64.tar.gz' | sha256sum -c -`,
				`tar -xzf "$HOME/.local/frank/artifacts/llama-b8994-bin-android-arm64.tar.gz" -C "$HOME/.local/frank/llama.cpp/b8994"`,
			},
			SafetyNotes: []string{
				"CPU-only phone-local runtime.",
				"Bind only to 127.0.0.1.",
				"Do not create Termux:Boot scripts without explicit approval.",
				"Do not add --mission-resume-approved automatically.",
				"Installer must locate llama-server or llama-cli after unpack instead of assuming archive layout.",
			},
		},
		ManifestQwen25TinyGGUF: {
			ID:             ManifestQwen25TinyGGUF,
			SourceURL:      "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/df5bf01389a39c743ab467d734bf501681e041c5/qwen2.5-0.5b-instruct-q4_k_m.gguf",
			Version:        "df5bf01389a39c743ab467d734bf501681e041c5",
			ChecksumSHA256: "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db",
			SizeBytes:      491000000,
			LicenseNotes:   "Apache-2.0 model license.",
			Platform:       "GGUF model",
			Arch:           "qwen2",
			InstallCommand: []string{
				`mkdir -p "$HOME/.local/frank/models"`,
				`curl -fL -o "$HOME/.local/frank/models/qwen2.5-0.5b-instruct-q4_k_m.gguf" "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/df5bf01389a39c743ab467d734bf501681e041c5/qwen2.5-0.5b-instruct-q4_k_m.gguf"`,
				`echo '74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db  '"$HOME"'/.local/frank/models/qwen2.5-0.5b-instruct-q4_k_m.gguf' | sha256sum -c -`,
			},
			SafetyNotes: []string{
				"Recommended phone context default: 2048 tokens.",
				"Recommended phone max output default: 512 tokens.",
				"Allow manual bump to 4096 context only after local readiness check.",
				"supportsTools=false.",
				"authorityTier=low.",
			},
		},
	}
}

func ManualRequiredOllamaTermuxManifest() ArtifactManifest {
	return ArtifactManifest{
		ID:             ManifestOllamaTermuxManual,
		SourceURL:      "manual_required",
		Version:        "manual_required",
		ChecksumSHA256: "manual_required",
		SizeBytes:      0,
		LicenseNotes:   "Termux package identity must be verified locally before approval.",
		Platform:       "Android/Termux",
		Arch:           "arm64",
		InstallCommand: []string{
			"pkg update",
			"pkg upgrade -y",
			"apt-cache policy ollama",
			"pkg install ollama",
		},
		SafetyNotes: []string{
			"Must stay manual_required until exact package version/source/checksum are verified.",
			"Do not approve mutable latest/source installs.",
			"Do not bind LAN.",
			"Do not create boot scripts automatically.",
			"Prefer phone-llamacpp-tiny for first approved one-command path.",
		},
	}
}

func (r ManifestRegistry) Lookup(id string) (ArtifactManifest, bool) {
	manifest, ok := r[strings.TrimSpace(id)]
	return manifest, ok
}

func ValidateManifest(manifest ArtifactManifest) error {
	if strings.TrimSpace(manifest.ID) == "" {
		return fmt.Errorf("manifest id is required")
	}
	if strings.TrimSpace(manifest.SourceURL) == "" {
		return fmt.Errorf("manifest %q source URL is required", manifest.ID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("manifest %q version or immutable release identifier is required", manifest.ID)
	}
	if strings.TrimSpace(manifest.ChecksumSHA256) == "" {
		return fmt.Errorf("manifest %q checksum is required", manifest.ID)
	}
	if strings.TrimSpace(manifest.SourceURL) == "manual_required" ||
		strings.TrimSpace(manifest.Version) == "manual_required" ||
		strings.TrimSpace(manifest.ChecksumSHA256) == "manual_required" {
		return fmt.Errorf("manifest %q is manual_required and cannot be automatically approved", manifest.ID)
	}
	if err := validateImmutableManifestURL(manifest.SourceURL); err != nil {
		return fmt.Errorf("manifest %q source URL is not immutable: %w", manifest.ID, err)
	}
	if err := validateSHA256(manifest.ChecksumSHA256); err != nil {
		return fmt.Errorf("manifest %q checksum is invalid: %w", manifest.ID, err)
	}
	if manifest.SizeBytes <= 0 {
		return fmt.Errorf("manifest %q size is required", manifest.ID)
	}
	if strings.TrimSpace(manifest.LicenseNotes) == "" {
		return fmt.Errorf("manifest %q license notes are required", manifest.ID)
	}
	if strings.TrimSpace(manifest.Platform) == "" {
		return fmt.Errorf("manifest %q platform is required", manifest.ID)
	}
	if strings.TrimSpace(manifest.Arch) == "" {
		return fmt.Errorf("manifest %q architecture is required", manifest.ID)
	}
	if len(manifest.InstallCommand) == 0 {
		return fmt.Errorf("manifest %q expected unpack/install command is required", manifest.ID)
	}
	if len(manifest.SafetyNotes) == 0 {
		return fmt.Errorf("manifest %q safety notes are required", manifest.ID)
	}
	return nil
}

func PlanManifestDownloadStep(id string, registry ManifestRegistry, destination string) PlanStep {
	manifest, ok := registry.Lookup(id)
	if !ok {
		return PlanStep{
			ID:                     "manifest-download-" + strings.TrimSpace(id),
			Summary:                "Manifest-gated download is unavailable",
			SideEffect:             SideEffectDownload,
			ApprovalRequired:       true,
			IdempotencyKey:         "manifest:" + strings.TrimSpace(id),
			RollbackCleanup:        "none",
			Status:                 PlanStatusManualRequired,
			ManualInstructions:     []string{"No checked-in manifest exists; follow manual setup instructions instead of automatic download."},
			RequiresManifest:       true,
			PreserveWhenTruncating: true,
		}
	}
	status := PlanStatusPlanned
	manual := []string(nil)
	if err := ValidateManifest(manifest); err != nil {
		status = PlanStatusBlocked
		manual = []string{err.Error()}
	}
	return PlanStep{
		ID:                     "manifest-download-" + manifest.ID,
		Summary:                "Download manifest-gated artifact",
		SideEffect:             SideEffectDownload,
		NetworkURL:             manifest.SourceURL,
		ExpectedDownloadSize:   fmt.Sprintf("%d", manifest.SizeBytes),
		ExpectedDiskImpact:     fmt.Sprintf("%d", manifest.SizeBytes),
		FilesToWrite:           []string{destination},
		ApprovalRequired:       true,
		ApprovalReason:         "downloads require explicit approval and checksum verification",
		IdempotencyKey:         "manifest:" + manifest.ID + ":" + manifest.ChecksumSHA256,
		AlreadyPresentRule:     "destination exists and checksum matches manifest",
		RollbackCleanup:        "remove partial downloaded artifact on checksum or write failure",
		RedactionPolicy:        "manifest downloads contain no secrets",
		Status:                 status,
		ManualInstructions:     manual,
		RequiresManifest:       true,
		ManifestID:             manifest.ID,
		ChecksumSHA256:         manifest.ChecksumSHA256,
		PreserveWhenTruncating: status != PlanStatusPlanned,
	}
}

func DownloadAndVerifyManifest(ctx context.Context, manifest ArtifactManifest, downloader Downloader, destination string) error {
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	if downloader == nil {
		return fmt.Errorf("downloader is required")
	}
	data, err := downloader.Download(ctx, manifest.SourceURL)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	want := strings.ToLower(strings.TrimSpace(manifest.ChecksumSHA256))
	if got != want {
		return fmt.Errorf("checksum mismatch for manifest %q", manifest.ID)
	}
	if int64(len(data)) != manifest.SizeBytes {
		return fmt.Errorf("size mismatch for manifest %q", manifest.ID)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".manifest-download-*.tmp")
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
	return os.Rename(tmpPath, destination)
}

func (d HTTPDownloader) Download(ctx context.Context, sourceURL string) ([]byte, error) {
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed with HTTP status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func ManifestDestinationReady(manifest ArtifactManifest, destination string) (bool, error) {
	if err := ValidateManifest(manifest); err != nil {
		return false, err
	}
	data, err := os.ReadFile(destination)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.ToLower(strings.TrimSpace(manifest.ChecksumSHA256)) {
		return false, nil
	}
	return int64(len(data)) == manifest.SizeBytes, nil
}

func ExpandHomePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "$HOME" || strings.HasPrefix(path, "$HOME/") {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", fmt.Errorf("HOME is required to expand %q", path)
		}
		if path == "$HOME" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "$HOME/")), nil
	}
	return path, nil
}

func validateSHA256(value string) error {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 64 {
		return fmt.Errorf("expected 64 hex characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return err
	}
	return nil
}

func validateImmutableManifestURL(raw string) error {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("scheme %q is not https", parsed.Scheme)
	}
	lower := strings.ToLower(parsed.EscapedPath())
	if strings.Contains(lower, "/latest") || strings.Contains(lower, "/main/") || strings.Contains(lower, "/master/") || strings.Contains(lower, "/resolve/latest/") {
		return fmt.Errorf("mutable path segment is not allowed")
	}
	if strings.Contains(strings.ToLower(parsed.Host), "huggingface.co") && strings.Contains(lower, "/resolve/") {
		parts := strings.Split(lower, "/resolve/")
		if len(parts) != 2 {
			return fmt.Errorf("huggingface resolve URL is malformed")
		}
		ref := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")[0]
		if len(ref) != 40 || validateHex(ref) != nil {
			return fmt.Errorf("huggingface resolve ref must be an immutable commit SHA")
		}
	}
	if strings.Contains(strings.ToLower(parsed.Host), "github.com") && strings.Contains(lower, "/releases/download/") {
		ref := strings.Split(strings.Split(lower, "/releases/download/")[1], "/")[0]
		if ref == "" || ref == "latest" {
			return fmt.Errorf("github release ref must be immutable")
		}
	}
	return nil
}

func validateHex(value string) error {
	_, err := hex.DecodeString(value)
	return err
}
