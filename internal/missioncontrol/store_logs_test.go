package missioncontrol

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPackageCurrentLogSegmentPackagesNonEmptySegmentAndReopensFreshCurrentLog(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	packagedAt := openedAt.Add(2 * time.Hour)
	logData := []byte("frank gateway line 1\nfrank gateway line 2\n")

	meta, err := EnsureCurrentLogSegment(root, openedAt)
	if err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if !meta.OpenedAt.Equal(openedAt) {
		t.Fatalf("EnsureCurrentLogSegment().OpenedAt = %s, want %s", meta.OpenedAt, openedAt)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), logData, 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	currentInfo, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log before package) error = %v", err)
	}

	result, err := PackageCurrentLogSegment(root, LogPackageReasonManual, WriterLockLease{LeaseHolderID: "packager-1"}, packagedAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegment() error = %v", err)
	}
	if result.NoOp {
		t.Fatalf("PackageCurrentLogSegment().NoOp = true, want false")
	}
	if result.PackageID == "" {
		t.Fatal("PackageCurrentLogSegment().PackageID = empty, want generated ID")
	}
	if result.ByteCount != int64(len(logData)) {
		t.Fatalf("PackageCurrentLogSegment().ByteCount = %d, want %d", result.ByteCount, len(logData))
	}
	if result.LogRelPath != storeGatewayLogRelPath(result.PackageID) {
		t.Fatalf("PackageCurrentLogSegment().LogRelPath = %q, want %q", result.LogRelPath, storeGatewayLogRelPath(result.PackageID))
	}

	manifest, err := LoadLogPackageManifest(root, result.PackageID)
	if err != nil {
		t.Fatalf("LoadLogPackageManifest() error = %v", err)
	}
	if !manifest.SegmentOpenedAt.Equal(openedAt) {
		t.Fatalf("manifest.SegmentOpenedAt = %s, want %s", manifest.SegmentOpenedAt, openedAt)
	}
	if !manifest.SegmentClosedAt.Equal(packagedAt) {
		t.Fatalf("manifest.SegmentClosedAt = %s, want %s", manifest.SegmentClosedAt, packagedAt)
	}
	if manifest.ByteCount != int64(len(logData)) {
		t.Fatalf("manifest.ByteCount = %d, want %d", manifest.ByteCount, len(logData))
	}

	packagedInfo, err := os.Stat(StoreLogPackageGatewayLogPath(root, result.PackageID))
	if err != nil {
		t.Fatalf("Stat(gateway.log) error = %v", err)
	}
	if !os.SameFile(currentInfo, packagedInfo) {
		t.Fatal("gateway.log does not preserve the original current.log file identity")
	}

	packagedData, err := os.ReadFile(StoreLogPackageGatewayLogPath(root, result.PackageID))
	if err != nil {
		t.Fatalf("ReadFile(gateway.log) error = %v", err)
	}
	if string(packagedData) != string(logData) {
		t.Fatalf("ReadFile(gateway.log) = %q, want %q", string(packagedData), string(logData))
	}

	reopenedInfo, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log after package) error = %v", err)
	}
	if os.SameFile(currentInfo, reopenedInfo) {
		t.Fatal("current.log after package reuses the packaged file, want fresh empty segment")
	}
	if reopenedInfo.Size() != 0 {
		t.Fatalf("current.log size after package = %d, want 0", reopenedInfo.Size())
	}

	reopenedMeta, err := LoadCurrentLogSegmentMeta(root)
	if err != nil {
		t.Fatalf("LoadCurrentLogSegmentMeta() error = %v", err)
	}
	if !reopenedMeta.OpenedAt.Equal(packagedAt) {
		t.Fatalf("LoadCurrentLogSegmentMeta().OpenedAt = %s, want %s", reopenedMeta.OpenedAt, packagedAt)
	}
}

func TestPackageCurrentLogSegmentNoOpWhenCurrentSegmentAbsent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	result, err := PackageCurrentLogSegment(root, LogPackageReasonManual, WriterLockLease{LeaseHolderID: "packager-1"}, now)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegment() error = %v", err)
	}
	if !result.NoOp || result.NoOpCause != "absent" {
		t.Fatalf("PackageCurrentLogSegment() = %#v, want no-op cause absent", result)
	}
	if _, err := os.Stat(StoreCurrentLogPath(root)); err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if _, err := os.Stat(StoreCurrentLogMetaPath(root)); err != nil {
		t.Fatalf("Stat(current.meta.json) error = %v", err)
	}
	entries, err := os.ReadDir(StoreLogPackagesDir(root))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadDir(log_packages) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("log_packages entries = %d, want 0", len(entries))
	}
}

func TestPackageCurrentLogSegmentNoOpWhenCurrentSegmentEmpty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	if _, err := EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}

	result, err := PackageCurrentLogSegment(root, LogPackageReasonManual, WriterLockLease{LeaseHolderID: "packager-1"}, now)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegment() error = %v", err)
	}
	if !result.NoOp || result.NoOpCause != "empty" {
		t.Fatalf("PackageCurrentLogSegment() = %#v, want no-op cause empty", result)
	}
	if _, err := os.Stat(StoreCurrentLogPath(root)); err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if _, err := os.Stat(StoreCurrentLogMetaPath(root)); err != nil {
		t.Fatalf("Stat(current.meta.json) error = %v", err)
	}
}

func TestLogPackageManifestRoundTripsAndRejectsMalformedData(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	manifest := LogPackageManifest{
		RecordVersion:   StoreRecordVersion,
		PackageID:       "20260405T100000.000000000Z-manual",
		Reason:          LogPackageReasonManual,
		CreatedAt:       now,
		SegmentOpenedAt: now.Add(-time.Hour),
		SegmentClosedAt: now,
		LogRelPath:      storeGatewayLogRelPath("20260405T100000.000000000Z-manual"),
		ByteCount:       42,
	}

	if err := ensureStoreChildDir(root, StoreLogPackagesDir(root)); err != nil {
		t.Fatalf("ensureStoreChildDir(log_packages) error = %v", err)
	}
	if err := ensureStoreChildDir(StoreLogPackagesDir(root), StoreLogPackageDir(root, manifest.PackageID)); err != nil {
		t.Fatalf("ensureStoreChildDir(packageDir) error = %v", err)
	}
	if err := StoreLogPackageManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreLogPackageManifestRecord() error = %v", err)
	}

	loaded, err := LoadLogPackageManifest(root, manifest.PackageID)
	if err != nil {
		t.Fatalf("LoadLogPackageManifest() error = %v", err)
	}
	if loaded != manifest {
		t.Fatalf("LoadLogPackageManifest() = %#v, want %#v", loaded, manifest)
	}

	if err := ensureStoreChildDir(StoreLogPackagesDir(root), StoreLogPackageDir(root, "bad")); err != nil {
		t.Fatalf("ensureStoreChildDir(bad packageDir) error = %v", err)
	}
	if err := os.WriteFile(StoreLogPackageManifestPath(root, "bad"), []byte("{\"record_version\":0}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(bad manifest) error = %v", err)
	}
	if _, err := LoadLogPackageManifest(root, "bad"); err == nil {
		t.Fatal("LoadLogPackageManifest(malformed) error = nil, want validation failure")
	}
}

func TestPackageCurrentLogSegmentRequiresWriterLockAndFailsClosedWhenLiveLockHeld(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("busy\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}
	if _, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"}); err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}

	_, err := PackageCurrentLogSegment(root, LogPackageReasonManual, WriterLockLease{LeaseHolderID: "holder-2"}, now.Add(10*time.Second))
	if !errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("PackageCurrentLogSegment() error = %v, want %v", err, ErrWriterLockHeld)
	}
}

func TestPackageCurrentLogSegmentUsesRenameSemantics(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	logData := []byte("rename check\n")
	if _, err := EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), logData, 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}
	beforeInfo, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log before package) error = %v", err)
	}

	result, err := PackageCurrentLogSegment(root, LogPackageReasonManual, WriterLockLease{LeaseHolderID: "packager-1"}, now)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegment() error = %v", err)
	}

	gatewayInfo, err := os.Stat(StoreLogPackageGatewayLogPath(root, result.PackageID))
	if err != nil {
		t.Fatalf("Stat(gateway.log) error = %v", err)
	}
	if !os.SameFile(beforeInfo, gatewayInfo) {
		t.Fatal("gateway.log does not share identity with the pre-package current.log file")
	}

	afterData, err := os.ReadFile(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("ReadFile(current.log after package) error = %v", err)
	}
	if len(afterData) != 0 {
		t.Fatalf("current.log after package = %q, want empty fresh segment", string(afterData))
	}
}

func TestPackageCurrentLogSegmentLeavesCurrentLogAndMetaPresentAfterSuccessfulPackaging(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	if _, err := PackageCurrentLogSegment(root, LogPackageReasonDaily, WriterLockLease{LeaseHolderID: "packager-1"}, now); err != nil {
		t.Fatalf("PackageCurrentLogSegment() error = %v", err)
	}
	if _, err := os.Stat(StoreCurrentLogPath(root)); err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if _, err := os.Stat(StoreCurrentLogMetaPath(root)); err != nil {
		t.Fatalf("Stat(current.meta.json) error = %v", err)
	}
}

func TestPackageCurrentLogSegmentOnGatewayStartupPackagesPreExistingNonEmptySegmentOnceWithReasonReboot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 23, 55, 0, 0, time.UTC)
	startupAt := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("pre-reboot line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	result, err := PackageCurrentLogSegmentOnGatewayStartup(root, WriterLockLease{LeaseHolderID: "gateway-1"}, startupAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() error = %v", err)
	}
	if result.NoOp {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() = %#v, want packaged result", result)
	}

	manifest, err := LoadLogPackageManifest(root, result.PackageID)
	if err != nil {
		t.Fatalf("LoadLogPackageManifest() error = %v", err)
	}
	if manifest.Reason != LogPackageReasonReboot {
		t.Fatalf("manifest.Reason = %q, want %q", manifest.Reason, LogPackageReasonReboot)
	}
	if _, err := os.Stat(StoreCurrentLogPath(root)); err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	info, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("current.log size after startup packaging = %d, want 0", info.Size())
	}
	meta, err := LoadCurrentLogSegmentMeta(root)
	if err != nil {
		t.Fatalf("LoadCurrentLogSegmentMeta() error = %v", err)
	}
	if !meta.OpenedAt.Equal(startupAt) {
		t.Fatalf("LoadCurrentLogSegmentMeta().OpenedAt = %s, want %s", meta.OpenedAt, startupAt)
	}
}

func TestPackageCurrentLogSegmentOnGatewayStartupNoOpWhenSegmentAbsent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)

	result, err := PackageCurrentLogSegmentOnGatewayStartup(root, WriterLockLease{LeaseHolderID: "gateway-1"}, now)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() error = %v", err)
	}
	if !result.NoOp || result.NoOpCause != "absent" {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() = %#v, want no-op cause absent", result)
	}
	if _, err := os.Stat(StoreCurrentLogPath(root)); err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	meta, err := LoadCurrentLogSegmentMeta(root)
	if err != nil {
		t.Fatalf("LoadCurrentLogSegmentMeta() error = %v", err)
	}
	if !meta.OpenedAt.Equal(now) {
		t.Fatalf("LoadCurrentLogSegmentMeta().OpenedAt = %s, want %s", meta.OpenedAt, now)
	}
}

func TestPackageCurrentLogSegmentOnGatewayStartupNoOpWhenSegmentEmpty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 23, 55, 0, 0, time.UTC)
	startupAt := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}

	result, err := PackageCurrentLogSegmentOnGatewayStartup(root, WriterLockLease{LeaseHolderID: "gateway-1"}, startupAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() error = %v", err)
	}
	if !result.NoOp || result.NoOpCause != "empty" {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() = %#v, want no-op cause empty", result)
	}
	info, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("current.log size = %d, want 0", info.Size())
	}
	meta, err := LoadCurrentLogSegmentMeta(root)
	if err != nil {
		t.Fatalf("LoadCurrentLogSegmentMeta() error = %v", err)
	}
	if !meta.OpenedAt.Equal(startupAt) {
		t.Fatalf("LoadCurrentLogSegmentMeta().OpenedAt = %s, want %s", meta.OpenedAt, startupAt)
	}
}

func TestPackageCurrentLogSegmentOnUTCDayRolloverPackagesOnceWithReasonDailyAndOpensNewCurrentSegment(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 23, 55, 0, 0, time.UTC)
	rolloverAt := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("day one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	result, err := PackageCurrentLogSegmentOnUTCDayRollover(root, WriterLockLease{LeaseHolderID: "gateway-1"}, rolloverAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover() error = %v", err)
	}
	if result.NoOp {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover() = %#v, want packaged result", result)
	}
	manifest, err := LoadLogPackageManifest(root, result.PackageID)
	if err != nil {
		t.Fatalf("LoadLogPackageManifest() error = %v", err)
	}
	if manifest.Reason != LogPackageReasonDaily {
		t.Fatalf("manifest.Reason = %q, want %q", manifest.Reason, LogPackageReasonDaily)
	}
	info, err := os.Stat(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("Stat(current.log) error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("current.log size after rollover packaging = %d, want 0", info.Size())
	}
	meta, err := LoadCurrentLogSegmentMeta(root)
	if err != nil {
		t.Fatalf("LoadCurrentLogSegmentMeta() error = %v", err)
	}
	if !meta.OpenedAt.Equal(rolloverAt) {
		t.Fatalf("LoadCurrentLogSegmentMeta().OpenedAt = %s, want %s", meta.OpenedAt, rolloverAt)
	}
}

func TestPackageCurrentLogSegmentOnUTCDayRolloverRepeatedChecksSameDayDoNotPackageAgain(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 23, 55, 0, 0, time.UTC)
	firstCheckAt := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	secondCheckAt := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("day one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	first, err := PackageCurrentLogSegmentOnUTCDayRollover(root, WriterLockLease{LeaseHolderID: "gateway-1"}, firstCheckAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover(first) error = %v", err)
	}
	if first.NoOp {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover(first) = %#v, want packaged result", first)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("day two\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log day two) error = %v", err)
	}

	second, err := PackageCurrentLogSegmentOnUTCDayRollover(root, WriterLockLease{LeaseHolderID: "gateway-1"}, secondCheckAt)
	if err != nil {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover(second) error = %v", err)
	}
	if !second.NoOp || second.NoOpCause != "same_day" {
		t.Fatalf("PackageCurrentLogSegmentOnUTCDayRollover(second) = %#v, want no-op cause same_day", second)
	}
	entries, err := os.ReadDir(StoreLogPackagesDir(root))
	if err != nil {
		t.Fatalf("ReadDir(log_packages) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("log_packages entries = %d, want 1", len(entries))
	}
	currentData, err := os.ReadFile(StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("ReadFile(current.log) error = %v", err)
	}
	if string(currentData) != "day two\n" {
		t.Fatalf("ReadFile(current.log) = %q, want %q", string(currentData), "day two\n")
	}
}

func TestAutomaticLogPackagingFailsClosedWhenLiveLockHeld(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	if _, err := EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(StoreCurrentLogPath(root), []byte("locked\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}
	if _, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"}); err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}

	_, err := PackageCurrentLogSegmentOnGatewayStartup(root, WriterLockLease{LeaseHolderID: "holder-2"}, now)
	if !errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("PackageCurrentLogSegmentOnGatewayStartup() error = %v, want %v", err, ErrWriterLockHeld)
	}
}

func TestLoadLogPackageManifestRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	packageID := "20260405T100000.000000000Z-manual"
	if err := ensureStoreChildDir(root, StoreLogPackagesDir(root)); err != nil {
		t.Fatalf("ensureStoreChildDir(log_packages) error = %v", err)
	}
	packageDir := StoreLogPackageDir(root, packageID)
	if err := ensureStoreChildDir(StoreLogPackagesDir(root), packageDir); err != nil {
		t.Fatalf("ensureStoreChildDir(packageDir) error = %v", err)
	}

	data := []byte("{\n  \"record_version\": 1,\n  \"package_id\": \"20260405T100000.000000000Z-manual\",\n  \"reason\": \"manual\",\n  \"created_at\": \"2026-04-05T10:00:00Z\",\n  \"segment_opened_at\": \"2026-04-05T09:00:00Z\",\n  \"segment_closed_at\": \"2026-04-05T10:00:00Z\",\n  \"log_relpath\": \"log_packages/20260405T100000.000000000Z-manual/gateway.log\",\n  \"byte_count\": 1,\n  \"unexpected\": true\n}\n")
	if err := os.WriteFile(filepath.Join(packageDir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(manifest.json) error = %v", err)
	}

	if _, err := LoadLogPackageManifest(root, packageID); err == nil {
		t.Fatal("LoadLogPackageManifest() error = nil, want unknown-field failure")
	}
}
