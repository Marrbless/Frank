package modelsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDownloader struct {
	data []byte
	urls []string
}

func (d *fakeDownloader) Download(ctx context.Context, url string) ([]byte, error) {
	d.urls = append(d.urls, url)
	return append([]byte(nil), d.data...), nil
}

func TestMissingBuiltinManifestProducesManualRequiredStep(t *testing.T) {
	step := PlanManifestDownloadStep("missing-llamacpp", BuiltinManifests(), filepath.Join(t.TempDir(), "artifact"))
	if step.Status != PlanStatusManualRequired {
		t.Fatalf("step status = %q, want manual_required", step.Status)
	}
	if !step.RequiresManifest {
		t.Fatal("RequiresManifest = false, want true")
	}
	if !containsString(step.ManualInstructions, "No checked-in manifest exists") {
		t.Fatalf("ManualInstructions = %#v, want missing manifest guidance", step.ManualInstructions)
	}
}

func TestDownloadAndVerifyManifestWritesFakeArtifact(t *testing.T) {
	data := []byte("fake artifact")
	manifest := validFakeManifest(data)
	dest := filepath.Join(t.TempDir(), "artifact.bin")
	downloader := &fakeDownloader{data: data}
	if err := DownloadAndVerifyManifest(context.Background(), manifest, downloader, dest); err != nil {
		t.Fatalf("DownloadAndVerifyManifest() error = %v", err)
	}
	if len(downloader.urls) != 1 || downloader.urls[0] != manifest.SourceURL {
		t.Fatalf("download urls = %#v, want %q", downloader.urls, manifest.SourceURL)
	}
	written, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile(dest) error = %v", err)
	}
	if string(written) != string(data) {
		t.Fatalf("written data = %q, want %q", string(written), string(data))
	}
}

func TestDownloadAndVerifyManifestRejectsChecksumMismatch(t *testing.T) {
	data := []byte("fake artifact")
	manifest := validFakeManifest(data)
	manifest.ChecksumSHA256 = strings.Repeat("0", 64)
	dest := filepath.Join(t.TempDir(), "artifact.bin")
	err := DownloadAndVerifyManifest(context.Background(), manifest, &fakeDownloader{data: data}, dest)
	if err == nil {
		t.Fatal("DownloadAndVerifyManifest() error = nil, want checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v, want checksum mismatch", err)
	}
	if fileExists(dest) {
		t.Fatalf("destination exists after checksum mismatch")
	}
}

func TestValidateManifestRequiresAllSafetyFields(t *testing.T) {
	manifest := validFakeManifest([]byte("data"))
	manifest.ChecksumSHA256 = ""
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("ValidateManifest() error = nil, want checksum required")
	}
}

func validFakeManifest(data []byte) ArtifactManifest {
	sum := sha256.Sum256(data)
	return ArtifactManifest{
		ID:             "fake-llamacpp-model",
		SourceURL:      "https://example.invalid/fake.bin",
		Version:        "immutable-test-release",
		ChecksumSHA256: hex.EncodeToString(sum[:]),
		SizeBytes:      int64(len(data)),
		LicenseNotes:   "test fixture only",
		Platform:       "linux",
		Arch:           "amd64",
		InstallCommand: []string{"install", "fake.bin"},
		SafetyNotes:    []string{"test fixture"},
	}
}
