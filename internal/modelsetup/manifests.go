package modelsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func BuiltinManifests() ManifestRegistry {
	return ManifestRegistry{}
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
