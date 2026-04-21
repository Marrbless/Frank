package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/missioncontrol"
)

func assertMainJSONObjectKeys(t *testing.T, object map[string]any, want ...string) {
	t.Helper()

	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)

	wantKeys := append([]string(nil), want...)
	sort.Strings(wantKeys)

	if !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("JSON keys = %#v, want %#v", got, wantKeys)
	}
}

func TestPromptSecretFallsBackToReaderWhenNotTerminal(t *testing.T) {
	previousIsTerminal := promptSecretIsTerminal
	previousReadPassword := promptSecretReadPassword
	promptSecretIsTerminal = func(fd int) bool { return false }
	promptSecretReadPassword = func(fd int) ([]byte, error) {
		t.Fatal("promptSecretReadPassword should not be called when stdin is not a terminal")
		return nil, nil
	}
	defer func() {
		promptSecretIsTerminal = previousIsTerminal
		promptSecretReadPassword = previousReadPassword
	}()

	reader := bufio.NewReader(strings.NewReader(" visible-secret \n"))
	if got, want := promptSecret(reader, "Bot token: "), "visible-secret"; got != want {
		t.Fatalf("promptSecret() = %q, want %q", got, want)
	}
}

func TestPromptSecretUsesHiddenInputWhenTerminalAvailable(t *testing.T) {
	previousIsTerminal := promptSecretIsTerminal
	previousReadPassword := promptSecretReadPassword
	promptSecretIsTerminal = func(fd int) bool { return true }
	promptSecretReadPassword = func(fd int) ([]byte, error) {
		return []byte(" hidden-secret \n"), nil
	}
	defer func() {
		promptSecretIsTerminal = previousIsTerminal
		promptSecretReadPassword = previousReadPassword
	}()

	reader := bufio.NewReader(strings.NewReader("should-not-be-used\n"))
	if got, want := promptSecret(reader, "Bot token: "), "hidden-secret"; got != want {
		t.Fatalf("promptSecret() = %q, want %q", got, want)
	}
}

func TestAgentCLI_ModelFlag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if _, _, err := config.Onboard(); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	cfgPath, _, _ := config.ResolveDefaultPaths()
	cfg2, _ := config.LoadConfig()
	cfg2.Providers.OpenAI = nil
	_ = config.SaveConfig(cfg2, cfgPath)

	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"agent", "--model", "stub-model", "-m", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(stub) Echo") {
		t.Fatalf("expected stub echo output, got: %q", out)
	}
}

func TestMissionPackageLogsCommandReturnsStableSummary(t *testing.T) {
	root := t.TempDir()
	openedAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, openedAt); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("gateway line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "package-logs", "--mission-store-root", root, "--reason", "manual"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPackageLogsSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "packaged" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "packaged")
	}
	if summary.Reason != missioncontrol.LogPackageReasonManual {
		t.Fatalf("summary.Reason = %q, want %q", summary.Reason, missioncontrol.LogPackageReasonManual)
	}
	if summary.PackageID == "" {
		t.Fatal("summary.PackageID = empty, want package ID")
	}
	if summary.LogRelPath != filepath.ToSlash(filepath.Join("log_packages", summary.PackageID, "gateway.log")) {
		t.Fatalf("summary.LogRelPath = %q, want gateway log relpath", summary.LogRelPath)
	}
	if summary.CurrentLogRelPath != filepath.ToSlash(filepath.Join("logs", "current.log")) {
		t.Fatalf("summary.CurrentLogRelPath = %q, want %q", summary.CurrentLogRelPath, filepath.ToSlash(filepath.Join("logs", "current.log")))
	}
	if summary.CurrentMetaRelPath != filepath.ToSlash(filepath.Join("logs", "current.meta.json")) {
		t.Fatalf("summary.CurrentMetaRelPath = %q, want %q", summary.CurrentMetaRelPath, filepath.ToSlash(filepath.Join("logs", "current.meta.json")))
	}
	if summary.ByteCount == 0 {
		t.Fatal("summary.ByteCount = 0, want packaged byte count")
	}
}

func TestMissionPackageLogsCommandPrunesExpiredPackagesAfterSuccessfulPackaging(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	oldPackageID := "20251230T120000.000000000Z-manual"
	if err := writeCommandLogPackageForTest(root, oldPackageID, now.AddDate(0, 0, -91)); err != nil {
		t.Fatalf("writeCommandLogPackageForTest() error = %v", err)
	}
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("gateway line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "package-logs", "--mission-store-root", root, "--reason", "manual"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertPathNotExists(t, missioncontrol.StoreLogPackageDir(root, oldPackageID))
}

func TestMissionPruneStoreCommandReturnsStableSummary(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	packageID := "20251231T120000.000000000Z-manual"
	if err := os.MkdirAll(missioncontrol.StoreLogPackageDir(root, packageID), 0o755); err != nil {
		t.Fatalf("MkdirAll(package dir) error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreLogPackageGatewayLogPath(root, packageID), []byte("gateway\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(gateway.log) error = %v", err)
	}
	if err := missioncontrol.StoreLogPackageManifestRecord(root, missioncontrol.LogPackageManifest{
		RecordVersion:   missioncontrol.StoreRecordVersion,
		PackageID:       packageID,
		Reason:          missioncontrol.LogPackageReasonManual,
		CreatedAt:       now.AddDate(0, 0, -91),
		SegmentOpenedAt: now.AddDate(0, 0, -91).Add(-time.Hour),
		SegmentClosedAt: now.AddDate(0, 0, -91),
		LogRelPath:      filepath.ToSlash(filepath.Join("log_packages", packageID, "gateway.log")),
		ByteCount:       int64(len("gateway\n")),
	}); err != nil {
		t.Fatalf("StoreLogPackageManifestRecord() error = %v", err)
	}

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "prune-store", "--mission-store-root", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPruneStoreSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "pruned" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "pruned")
	}
	if summary.StoreRoot != root {
		t.Fatalf("summary.StoreRoot = %q, want %q", summary.StoreRoot, root)
	}
	if summary.PrunedPackageDirs != 1 {
		t.Fatalf("summary.PrunedPackageDirs = %d, want 1", summary.PrunedPackageDirs)
	}
	if summary.PrunedAuditFiles != 0 || summary.PrunedApprovalRequestFiles != 0 || summary.PrunedApprovalGrantFiles != 0 || summary.PrunedArtifactFiles != 0 || summary.SkippedNonterminalJobTrees != 0 {
		t.Fatalf("summary = %#v, want only packaged dir count", summary)
	}
}

func TestConfigureGatewayMissionStoreLoggingPrunesExpiredPackagesAfterStartupPackaging(t *testing.T) {
	originalGatewayLogNow := gatewayLogNow
	t.Cleanup(func() { gatewayLogNow = originalGatewayLogNow })

	now := time.Date(2026, 4, 6, 0, 1, 0, 0, time.UTC)
	gatewayLogNow = func() time.Time { return now }

	root := t.TempDir()
	oldPackageID := "20251230T120000.000000000Z-reboot"
	if err := writeCommandLogPackageForTest(root, oldPackageID, now.AddDate(0, 0, -91)); err != nil {
		t.Fatalf("writeCommandLogPackageForTest() error = %v", err)
	}
	if _, err := missioncontrol.EnsureCurrentLogSegment(root, now.Add(-time.Hour)); err != nil {
		t.Fatalf("EnsureCurrentLogSegment() error = %v", err)
	}
	if err := os.WriteFile(missioncontrol.StoreCurrentLogPath(root), []byte("startup line\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(current.log) error = %v", err)
	}

	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	if err := cmd.Flags().Set("mission-store-root", root); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != root {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want %q", storeRoot, root)
	}
	if lease.LeaseHolderID == "" {
		t.Fatal("configureGatewayMissionStoreLogging().lease.LeaseHolderID = empty, want gateway lease holder")
	}
	assertPathNotExists(t, missioncontrol.StoreLogPackageDir(root, oldPackageID))
}

func TestMissionPruneStoreCommandReturnsStableNoOpSummary(t *testing.T) {
	root := t.TempDir()

	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mission", "prune-store", "--mission-store-root", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary missionPruneStoreSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if summary.Action != "noop" {
		t.Fatalf("summary.Action = %q, want %q", summary.Action, "noop")
	}
	if summary.StoreRoot != root {
		t.Fatalf("summary.StoreRoot = %q, want %q", summary.StoreRoot, root)
	}
	if summary.PrunedPackageDirs != 0 ||
		summary.PrunedAuditFiles != 0 ||
		summary.PrunedApprovalRequestFiles != 0 ||
		summary.PrunedApprovalGrantFiles != 0 ||
		summary.PrunedArtifactFiles != 0 ||
		summary.SkippedNonterminalJobTrees != 0 {
		t.Fatalf("summary = %#v, want zero-count no-op", summary)
	}
}

func TestConfigureGatewayMissionStoreLoggingRoutesStdlibLoggerIntoActiveSegment(t *testing.T) {
	root := t.TempDir()
	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	if err := cmd.Flags().Set("mission-store-root", root); err != nil {
		t.Fatalf("Flags().Set(mission-store-root) error = %v", err)
	}

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != root {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want %q", storeRoot, root)
	}
	if lease.LeaseHolderID == "" {
		t.Fatal("configureGatewayMissionStoreLogging().lease.LeaseHolderID = empty, want gateway lease holder")
	}

	log.Printf("gateway logger line")

	data, err := os.ReadFile(missioncontrol.StoreCurrentLogPath(root))
	if err != nil {
		t.Fatalf("ReadFile(current.log) error = %v", err)
	}
	if !strings.Contains(string(data), "gateway logger line") {
		t.Fatalf("ReadFile(current.log) = %q, want logger line", string(data))
	}
}

func writeCommandLogPackageForTest(root string, packageID string, createdAt time.Time) error {
	if err := os.MkdirAll(missioncontrol.StoreLogPackageDir(root, packageID), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(missioncontrol.StoreLogPackageGatewayLogPath(root, packageID), []byte("gateway\n"), 0o644); err != nil {
		return err
	}
	return missioncontrol.StoreLogPackageManifestRecord(root, missioncontrol.LogPackageManifest{
		RecordVersion:   missioncontrol.StoreRecordVersion,
		PackageID:       packageID,
		Reason:          missioncontrol.LogPackageReasonManual,
		CreatedAt:       createdAt,
		SegmentOpenedAt: createdAt.Add(-time.Hour),
		SegmentClosedAt: createdAt,
		LogRelPath:      filepath.ToSlash(filepath.Join("log_packages", packageID, "gateway.log")),
		ByteCount:       int64(len("gateway\n")),
	})
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return
	} else if err != nil {
		t.Fatalf("Stat(%q) error = %v, want os.ErrNotExist", path, err)
	}
	t.Fatalf("Stat(%q) error = nil, want os.ErrNotExist", path)
}

func TestConfigureGatewayMissionStoreLoggingWithoutStoreRootPreservesExistingLoggerBehavior(t *testing.T) {
	logBuf, restoreStandardLogger := captureStandardLogger(t)
	defer restoreStandardLogger()

	cmd := &cobra.Command{Use: "gateway"}
	addMissionBootstrapFlags(cmd)
	previousWriter := log.Writer()

	storeRoot, lease, restore, err := configureGatewayMissionStoreLogging(cmd)
	if err != nil {
		t.Fatalf("configureGatewayMissionStoreLogging() error = %v", err)
	}
	defer restore()

	if storeRoot != "" {
		t.Fatalf("configureGatewayMissionStoreLogging().storeRoot = %q, want empty", storeRoot)
	}
	if lease.LeaseHolderID != "" {
		t.Fatalf("configureGatewayMissionStoreLogging().lease = %#v, want zero lease", lease)
	}
	if log.Writer() != previousWriter {
		t.Fatal("configureGatewayMissionStoreLogging() changed logger output without a store root")
	}

	log.Printf("fallback logger line")
	if !strings.Contains(logBuf.String(), "fallback logger line") {
		t.Fatalf("log output = %q, want fallback logger line", logBuf.String())
	}
}

func TestRemoveMissionStatusSnapshotRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := removeMissionStatusSnapshot(path); err != nil {
		t.Fatalf("removeMissionStatusSnapshot() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exists", err)
	}
}
