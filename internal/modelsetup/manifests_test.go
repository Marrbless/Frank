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

func TestBuiltinManifestsContainApprovedPhoneLlamaCPPArtifacts(t *testing.T) {
	registry := BuiltinManifests()
	runtime, ok := registry.Lookup(ManifestLlamaCPPAndroidARM64B8994)
	if !ok {
		t.Fatalf("runtime manifest %q missing", ManifestLlamaCPPAndroidARM64B8994)
	}
	model, ok := registry.Lookup(ManifestQwen25TinyGGUF)
	if !ok {
		t.Fatalf("model manifest %q missing", ManifestQwen25TinyGGUF)
	}
	for _, manifest := range []ArtifactManifest{runtime, model} {
		if err := ValidateManifest(manifest); err != nil {
			t.Fatalf("ValidateManifest(%q) error = %v", manifest.ID, err)
		}
	}
	if runtime.SourceURL != "https://github.com/ggml-org/llama.cpp/releases/download/b8994/llama-b8994-bin-android-arm64.tar.gz" {
		t.Fatalf("runtime URL = %q", runtime.SourceURL)
	}
	if runtime.ChecksumSHA256 != "bf0445968910d36ef85cc273501601189db3ed052a57c0393ba56bd346ab7d54" {
		t.Fatalf("runtime checksum = %q", runtime.ChecksumSHA256)
	}
	if model.SourceURL != "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/df5bf01389a39c743ab467d734bf501681e041c5/qwen2.5-0.5b-instruct-q4_k_m.gguf" {
		t.Fatalf("model URL = %q", model.SourceURL)
	}
	if model.ChecksumSHA256 != "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db" {
		t.Fatalf("model checksum = %q", model.ChecksumSHA256)
	}
}

func TestValidateManifestBlocksUnsafeApprovalInputs(t *testing.T) {
	data := []byte("data")
	tests := []struct {
		name   string
		mutate func(*ArtifactManifest)
		want   string
	}{
		{
			name: "missing checksum",
			mutate: func(manifest *ArtifactManifest) {
				manifest.ChecksumSHA256 = ""
			},
			want: "checksum is required",
		},
		{
			name: "missing size",
			mutate: func(manifest *ArtifactManifest) {
				manifest.SizeBytes = 0
			},
			want: "size is required",
		},
		{
			name: "mutable latest url",
			mutate: func(manifest *ArtifactManifest) {
				manifest.SourceURL = "https://github.com/example/project/releases/latest/download/model.gguf"
			},
			want: "not immutable",
		},
		{
			name: "mutable branch resolver",
			mutate: func(manifest *ArtifactManifest) {
				manifest.SourceURL = "https://huggingface.co/org/model/resolve/main/model.gguf"
			},
			want: "not immutable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validFakeManifest(data)
			tt.mutate(&manifest)
			err := ValidateManifest(manifest)
			if err == nil {
				t.Fatal("ValidateManifest() error = nil, want safety failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			step := PlanManifestDownloadStep(manifest.ID, ManifestRegistry{manifest.ID: manifest}, filepath.Join(t.TempDir(), "artifact"))
			if step.Status != PlanStatusBlocked {
				t.Fatalf("unsafe manifest plan status = %q, want blocked", step.Status)
			}
		})
	}
}

func TestManualRequiredManifestCannotBecomeApproved(t *testing.T) {
	manifest := ManualRequiredOllamaTermuxManifest()
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("ValidateManifest(manual_required) error = nil, want rejection")
	}
	step := PlanManifestDownloadStep(manifest.ID, ManifestRegistry{manifest.ID: manifest}, filepath.Join(t.TempDir(), "ollama"))
	if step.Status != PlanStatusBlocked {
		t.Fatalf("manual_required manifest step status = %q, want blocked", step.Status)
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
