package missioncontrol

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func assertWriterEpochCoherent(t *testing.T, root string) {
	t.Helper()

	if err := ValidateWriterEpochCoherence(root); err != nil {
		t.Fatalf("ValidateWriterEpochCoherence() error = %v", err)
	}
}

func assertManifestLastRecoveredAt(t *testing.T, root string, want time.Time) {
	t.Helper()

	manifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if !manifest.LastRecoveredAt.Equal(want.UTC()) {
		t.Fatalf("LoadStoreManifest().LastRecoveredAt = %s, want %s", manifest.LastRecoveredAt, want.UTC())
	}
}

func withStoreWriterGuardTimingForTest(t *testing.T, maxWait time.Duration, staleAfter time.Duration) {
	t.Helper()

	originalMaxWait := storeWriterGuardMaxWait
	originalStaleAfter := storeWriterGuardStaleAfter
	originalSleep := storeWriterGuardSleep
	t.Cleanup(func() {
		storeWriterGuardMaxWait = originalMaxWait
		storeWriterGuardStaleAfter = originalStaleAfter
		storeWriterGuardSleep = originalSleep
	})

	storeWriterGuardMaxWait = maxWait
	storeWriterGuardStaleAfter = staleAfter
	storeWriterGuardSleep = func(time.Duration) {}
}

func TestAcquireWriterLockEnforcesSingleWriter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	first, tookOver, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock(first) error = %v", err)
	}
	if tookOver {
		t.Fatal("AcquireWriterLock(first) tookOver = true, want false")
	}

	_, _, err = AcquireWriterLock(root, now.Add(30*time.Second), time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if !errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("AcquireWriterLock(second) error = %v, want %v", err, ErrWriterLockHeld)
	}

	loaded, err := LoadWriterLock(root)
	if err != nil {
		t.Fatalf("LoadWriterLock() error = %v", err)
	}
	if loaded.LeaseHolderID != first.LeaseHolderID {
		t.Fatalf("LoadWriterLock().LeaseHolderID = %q, want %q", loaded.LeaseHolderID, first.LeaseHolderID)
	}

	manifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if manifest.CurrentWriterEpoch != loaded.WriterEpoch {
		t.Fatalf("LoadStoreManifest().CurrentWriterEpoch = %d, want %d", manifest.CurrentWriterEpoch, loaded.WriterEpoch)
	}
	assertWriterEpochCoherent(t, root)
}

func TestAcquireWriterLockConcurrentCallersYieldExactlyOneWriter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	const workers = 2
	var wg sync.WaitGroup
	wg.Add(workers)
	start := make(chan struct{})

	type result struct {
		record WriterLockRecord
		err    error
	}
	results := make(chan result, workers)
	for i := range workers {
		holderID := "holder-1"
		if i == 1 {
			holderID = "holder-2"
		}
		go func(holderID string) {
			defer wg.Done()
			<-start
			record, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: holderID})
			results <- result{record: record, err: err}
		}(holderID)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	heldErrors := 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
		case errors.Is(result.err, ErrWriterLockHeld):
			heldErrors++
		default:
			t.Fatalf("AcquireWriterLock(concurrent) error = %v, want nil or %v", result.err, ErrWriterLockHeld)
		}
	}

	if successes != 1 {
		t.Fatalf("AcquireWriterLock(concurrent) successes = %d, want 1", successes)
	}
	if heldErrors != 1 {
		t.Fatalf("AcquireWriterLock(concurrent) heldErrors = %d, want 1", heldErrors)
	}
}

func TestAcquireWriterLockUnreadableLockFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if _, err := InitStoreManifest(root, now); err != nil {
		t.Fatalf("InitStoreManifest() error = %v", err)
	}
	if err := os.WriteFile(StoreWriterLockPath(root), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile(writer.lock) error = %v", err)
	}

	_, _, err := AcquireWriterLock(root, now.Add(time.Second), time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if err == nil {
		t.Fatal("AcquireWriterLock() error = nil, want unreadable lock failure")
	}
	if errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("AcquireWriterLock() error = %v, want parse/validation failure", err)
	}
}

func TestAcquireWriterLockFreshGuardBlocksCorrectly(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	withStoreWriterGuardTimingForTest(t, 5*time.Millisecond, time.Minute)

	if err := os.WriteFile(StoreWriterGuardPath(root), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(writer.lock.guard) error = %v", err)
	}

	_, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if !errors.Is(err, ErrWriterLockHeld) {
		t.Fatalf("AcquireWriterLock() error = %v, want %v", err, ErrWriterLockHeld)
	}
}

func TestAcquireWriterLockRecoversProvablyStaleGuard(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	withStoreWriterGuardTimingForTest(t, 5*time.Millisecond, 10*time.Millisecond)

	guardPath := StoreWriterGuardPath(root)
	if err := os.WriteFile(guardPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(writer.lock.guard) error = %v", err)
	}
	oldTime := time.Now().Add(-time.Minute)
	if err := os.Chtimes(guardPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(writer.lock.guard) error = %v", err)
	}

	lock, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}
	if lock.LeaseHolderID != "holder-1" {
		t.Fatalf("AcquireWriterLock().LeaseHolderID = %q, want %q", lock.LeaseHolderID, "holder-1")
	}
	assertWriterEpochCoherent(t, root)
}

func TestTakeoverWriterLockUnreadableLockUsesManifestEpoch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          now,
		StoreState:             StoreStateReady,
		CurrentWriterEpoch:     7,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}
	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}
	if err := os.WriteFile(StoreWriterLockPath(root), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile(writer.lock) error = %v", err)
	}

	lock, err := TakeoverWriterLock(root, now.Add(2*time.Minute), time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if err != nil {
		t.Fatalf("TakeoverWriterLock() error = %v", err)
	}
	if lock.WriterEpoch != 8 {
		t.Fatalf("TakeoverWriterLock().WriterEpoch = %d, want 8", lock.WriterEpoch)
	}

	loadedManifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if loadedManifest.CurrentWriterEpoch != lock.WriterEpoch {
		t.Fatalf("LoadStoreManifest().CurrentWriterEpoch = %d, want %d", loadedManifest.CurrentWriterEpoch, lock.WriterEpoch)
	}
	if !loadedManifest.LastRecoveredAt.Equal(now.Add(2 * time.Minute).UTC()) {
		t.Fatalf("LoadStoreManifest().LastRecoveredAt = %s, want %s", loadedManifest.LastRecoveredAt, now.Add(2*time.Minute).UTC())
	}
	assertWriterEpochCoherent(t, root)
}

func TestTakeoverWriterLockStaleLockBumpsEpochFromManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	first, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock(first) error = %v", err)
	}

	loadedManifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	loadedManifest.CurrentWriterEpoch = first.WriterEpoch + 4
	if err := StoreManifestRecord(root, loadedManifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}

	second, err := TakeoverWriterLock(root, now.Add(2*time.Minute), time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if err != nil {
		t.Fatalf("TakeoverWriterLock() error = %v", err)
	}
	if second.WriterEpoch != loadedManifest.CurrentWriterEpoch+1 {
		t.Fatalf("TakeoverWriterLock().WriterEpoch = %d, want %d", second.WriterEpoch, loadedManifest.CurrentWriterEpoch+1)
	}

	finalManifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if finalManifest.CurrentWriterEpoch != second.WriterEpoch {
		t.Fatalf("LoadStoreManifest().CurrentWriterEpoch = %d, want %d", finalManifest.CurrentWriterEpoch, second.WriterEpoch)
	}
	if !finalManifest.LastRecoveredAt.Equal(now.Add(2 * time.Minute).UTC()) {
		t.Fatalf("LoadStoreManifest().LastRecoveredAt = %s, want %s", finalManifest.LastRecoveredAt, now.Add(2*time.Minute).UTC())
	}
	assertWriterEpochCoherent(t, root)
}

func TestTakeoverWriterLockStaleIncoherentLockUsesMaxEpochPlusOne(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          now,
		StoreState:             StoreStateReady,
		CurrentWriterEpoch:     3,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}
	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}

	lock := WriterLockRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    7,
		LeaseHolderID:  "holder-1",
		StartedAt:      now.Add(-2 * time.Minute),
		RenewedAt:      now.Add(-2 * time.Minute),
		LeaseExpiresAt: now.Add(-time.Minute),
	}
	if err := StoreWriterLockRecord(root, lock); err != nil {
		t.Fatalf("StoreWriterLockRecord() error = %v", err)
	}

	recoveredAt := now.Add(3 * time.Minute)
	second, err := TakeoverWriterLock(root, recoveredAt, time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if err != nil {
		t.Fatalf("TakeoverWriterLock() error = %v", err)
	}
	if second.WriterEpoch != 8 {
		t.Fatalf("TakeoverWriterLock().WriterEpoch = %d, want 8", second.WriterEpoch)
	}
	assertWriterEpochCoherent(t, root)
	assertManifestLastRecoveredAt(t, root, recoveredAt)
}

func TestAcquireWriterLockLiveIncoherentLockFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          now,
		StoreState:             StoreStateReady,
		CurrentWriterEpoch:     2,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}
	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}

	lock := WriterLockRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    3,
		LeaseHolderID:  "holder-1",
		StartedAt:      now,
		RenewedAt:      now,
		LeaseExpiresAt: now.Add(time.Minute),
	}
	if err := StoreWriterLockRecord(root, lock); err != nil {
		t.Fatalf("StoreWriterLockRecord() error = %v", err)
	}

	_, _, err := AcquireWriterLock(root, now.Add(10*time.Second), time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if !errors.Is(err, ErrWriterEpochIncoherent) {
		t.Fatalf("AcquireWriterLock() error = %v, want %v", err, ErrWriterEpochIncoherent)
	}
}

func TestAcquireWriterLockSameHolderAfterExpiryFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	lock, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock(initial) error = %v", err)
	}

	_, _, err = AcquireWriterLock(root, now.Add(2*time.Minute), time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if !errors.Is(err, ErrWriterLockExpired) {
		t.Fatalf("AcquireWriterLock(expired same-holder) error = %v, want %v", err, ErrWriterLockExpired)
	}

	loaded, err := LoadWriterLock(root)
	if err != nil {
		t.Fatalf("LoadWriterLock() error = %v", err)
	}
	if loaded.WriterEpoch != lock.WriterEpoch {
		t.Fatalf("LoadWriterLock().WriterEpoch = %d, want unchanged %d", loaded.WriterEpoch, lock.WriterEpoch)
	}
}

func TestRenewWriterLockLiveIncoherentLockFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	lock := WriterLockRecord{
		RecordVersion:  StoreRecordVersion,
		WriterEpoch:    4,
		LeaseHolderID:  "holder-1",
		StartedAt:      now,
		RenewedAt:      now,
		LeaseExpiresAt: now.Add(time.Minute),
	}
	manifest := StoreManifest{
		RecordVersion:          StoreRecordVersion,
		SchemaVersion:          StoreSchemaVersion,
		StoreID:                "store-1",
		InitializedAt:          now,
		StoreState:             StoreStateReady,
		CurrentWriterEpoch:     3,
		RetentionPolicyVersion: StoreRetentionVersionV1,
	}
	if err := StoreManifestRecord(root, manifest); err != nil {
		t.Fatalf("StoreManifestRecord() error = %v", err)
	}
	if err := StoreWriterLockRecord(root, lock); err != nil {
		t.Fatalf("StoreWriterLockRecord() error = %v", err)
	}

	_, err := RenewWriterLock(root, lock, now.Add(10*time.Second), time.Minute)
	if !errors.Is(err, ErrWriterEpochIncoherent) {
		t.Fatalf("RenewWriterLock() error = %v, want %v", err, ErrWriterEpochIncoherent)
	}
}

func TestRenewWriterLockSameHolderAfterExpiryFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	lock, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock(initial) error = %v", err)
	}

	_, err = RenewWriterLock(root, lock, now.Add(2*time.Minute), time.Minute)
	if !errors.Is(err, ErrWriterLockExpired) {
		t.Fatalf("RenewWriterLock(expired same-holder) error = %v, want %v", err, ErrWriterLockExpired)
	}
}

func TestRenewWriterLockExtendsLeaseForCurrentHolderWithoutChangingEpoch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	lock, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock() error = %v", err)
	}

	renewed, err := RenewWriterLock(root, lock, now.Add(30*time.Second), 2*time.Minute)
	if err != nil {
		t.Fatalf("RenewWriterLock() error = %v", err)
	}
	if !renewed.LeaseExpiresAt.After(lock.LeaseExpiresAt) {
		t.Fatalf("LeaseExpiresAt = %s, want after %s", renewed.LeaseExpiresAt, lock.LeaseExpiresAt)
	}
	if renewed.WriterEpoch != lock.WriterEpoch {
		t.Fatalf("WriterEpoch = %d, want unchanged %d", renewed.WriterEpoch, lock.WriterEpoch)
	}

	manifest, err := LoadStoreManifest(root)
	if err != nil {
		t.Fatalf("LoadStoreManifest() error = %v", err)
	}
	if manifest.CurrentWriterEpoch != lock.WriterEpoch {
		t.Fatalf("LoadStoreManifest().CurrentWriterEpoch = %d, want %d", manifest.CurrentWriterEpoch, lock.WriterEpoch)
	}
	assertWriterEpochCoherent(t, root)
}

func TestTakeoverWriterLockAfterExpiryStillSucceedsAndRestoresCoherence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	first, _, err := AcquireWriterLock(root, now, time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err != nil {
		t.Fatalf("AcquireWriterLock(initial) error = %v", err)
	}

	takenOverAt := now.Add(2 * time.Minute)
	second, err := TakeoverWriterLock(root, takenOverAt, time.Minute, WriterLockLease{LeaseHolderID: "holder-2"})
	if err != nil {
		t.Fatalf("TakeoverWriterLock() error = %v", err)
	}
	if second.WriterEpoch != first.WriterEpoch+1 {
		t.Fatalf("TakeoverWriterLock().WriterEpoch = %d, want %d", second.WriterEpoch, first.WriterEpoch+1)
	}
	assertWriterEpochCoherent(t, root)
	assertManifestLastRecoveredAt(t, root, takenOverAt)
}

func TestAcquireWriterLockManifestWriteFailureAfterLockWriteFailsClosed(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)

	if _, err := InitStoreManifest(root, now); err != nil {
		t.Fatalf("InitStoreManifest() error = %v", err)
	}

	originalWriteManifest := storeManifestWriteJSONAtomic
	t.Cleanup(func() { storeManifestWriteJSONAtomic = originalWriteManifest })
	storeManifestWriteJSONAtomic = func(path string, value any) error {
		return fmt.Errorf("forced manifest write failure for %s", path)
	}

	_, _, err := AcquireWriterLock(root, now.Add(time.Second), time.Minute, WriterLockLease{LeaseHolderID: "holder-1"})
	if err == nil {
		t.Fatal("AcquireWriterLock() error = nil, want manifest write failure")
	}

	coherenceErr := ValidateWriterEpochCoherence(root)
	if coherenceErr == nil {
		t.Fatal("ValidateWriterEpochCoherence() error = nil, want mismatch after failed manifest update")
	}
}
