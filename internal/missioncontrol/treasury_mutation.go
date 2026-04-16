package missioncontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const storeTreasuryMutationLeaseDuration = time.Minute

type FirstTreasuryAcquisitionInput struct {
	TreasuryID string
	EntryID    string
	AssetCode  string
	Amount     string
	SourceRef  string
}

type ActivateFundedTreasuryInput struct {
	TreasuryID string
}

type PostBootstrapTreasuryAcquisitionInput struct {
	TreasuryID string
}

// RecordFirstTreasuryAcquisition records the first landed value for a
// bootstrap treasury by appending one acquisition ledger entry and then
// transitioning the same treasury to funded behind the mission-store writer
// lock. The treasury record is written last so visible committed state never
// shows funded without its corresponding acquisition entry.
func RecordFirstTreasuryAcquisition(root string, lease WriterLockLease, input FirstTreasuryAcquisitionInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizeFirstTreasuryAcquisitionInput(input)
	lock, err := acquireTreasuryMutationWriterLock(root, lease, now)
	if err != nil {
		return err
	}

	return withLockedTreasuryMutation(root, lock, now, func() error {
		treasury, err := LoadTreasuryRecord(root, input.TreasuryID)
		if err != nil {
			return err
		}
		if treasury.State != TreasuryStateBootstrap {
			return fmt.Errorf(
				"mission store treasury %q first acquisition requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateBootstrap,
				treasury.State,
			)
		}
		if _, err := resolveTreasuryActiveContainerID(treasury); err != nil {
			return err
		}
		if err := ensureTreasuryFirstAcquisitionNotRecorded(root, treasury.TreasuryID, input.EntryID); err != nil {
			return err
		}

		entry := normalizeTreasuryLedgerEntry(TreasuryLedgerEntry{
			RecordVersion: StoreRecordVersion,
			EntryID:       input.EntryID,
			TreasuryID:    treasury.TreasuryID,
			EntryKind:     TreasuryLedgerEntryKindAcquisition,
			AssetCode:     input.AssetCode,
			Amount:        input.Amount,
			CreatedAt:     now,
			SourceRef:     input.SourceRef,
		})
		if err := ValidateTreasuryLedgerEntry(entry); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.State = TreasuryStateFunded
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitFirstTreasuryAcquisitionBatch(root, updatedTreasury, entry)
	})
}

// ActivateFundedTreasury transitions one funded treasury to active behind the
// mission-store writer lock without moving money or appending a ledger entry.
// The treasury record write is the only visible commit point, so retries fail
// closed once the state transition has landed.
func ActivateFundedTreasury(root string, lease WriterLockLease, input ActivateFundedTreasuryInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizeActivateFundedTreasuryInput(input)
	lock, err := acquireTreasuryMutationWriterLock(root, lease, now)
	if err != nil {
		return err
	}

	return withLockedTreasuryMutation(root, lock, now, func() error {
		treasury, err := LoadTreasuryRecord(root, input.TreasuryID)
		if err != nil {
			return err
		}
		if treasury.State != TreasuryStateFunded {
			return fmt.Errorf(
				"mission store treasury %q activation requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateFunded,
				treasury.State,
			)
		}
		if _, err := resolveTreasuryActiveContainerID(treasury); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.State = TreasuryStateActive
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitFundedTreasuryActivation(root, updatedTreasury)
	})
}

// RecordPostBootstrapTreasuryAcquisition records exactly one additional landed
// acquisition for an already-active treasury by consuming the committed
// treasury.post_bootstrap_acquisition block on the same TreasuryRecord. The
// ledger entry is written before the treasury update so retries fail closed if
// a partial write leaves ambiguous state.
func RecordPostBootstrapTreasuryAcquisition(root string, lease WriterLockLease, input PostBootstrapTreasuryAcquisitionInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizePostBootstrapTreasuryAcquisitionInput(input)
	lock, err := acquireTreasuryMutationWriterLock(root, lease, now)
	if err != nil {
		return err
	}

	return withLockedTreasuryMutation(root, lock, now, func() error {
		treasury, err := LoadTreasuryRecord(root, input.TreasuryID)
		if err != nil {
			return err
		}
		if treasury.State != TreasuryStateActive {
			return fmt.Errorf(
				"mission store treasury %q post-bootstrap acquisition requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateActive,
				treasury.State,
			)
		}
		if _, err := resolveTreasuryActiveContainerID(treasury); err != nil {
			return err
		}
		if treasury.PostBootstrapAcquisition == nil {
			return fmt.Errorf(
				"mission store treasury %q post-bootstrap acquisition requires committed treasury.post_bootstrap_acquisition",
				treasury.TreasuryID,
			)
		}
		if strings.TrimSpace(treasury.PostBootstrapAcquisition.ConsumedEntryID) != "" {
			return fmt.Errorf(
				"mission store treasury %q post-bootstrap acquisition already consumed by entry %q",
				treasury.TreasuryID,
				strings.TrimSpace(treasury.PostBootstrapAcquisition.ConsumedEntryID),
			)
		}

		block := *treasury.PostBootstrapAcquisition
		entryID := derivePostBootstrapTreasuryAcquisitionEntryID(treasury.TreasuryID, block)
		if err := ensurePostBootstrapTreasuryAcquisitionEntryAvailable(root, treasury.TreasuryID, entryID); err != nil {
			return err
		}

		entry := normalizeTreasuryLedgerEntry(TreasuryLedgerEntry{
			RecordVersion: StoreRecordVersion,
			EntryID:       entryID,
			TreasuryID:    treasury.TreasuryID,
			EntryKind:     TreasuryLedgerEntryKindAcquisition,
			AssetCode:     block.AssetCode,
			Amount:        block.Amount,
			CreatedAt:     now,
			SourceRef:     block.SourceRef,
		})
		if err := ValidateTreasuryLedgerEntry(entry); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.PostBootstrapAcquisition.ConsumedEntryID = entryID
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitPostBootstrapTreasuryAcquisitionBatch(root, updatedTreasury, entry)
	})
}

func normalizeFirstTreasuryAcquisitionInput(input FirstTreasuryAcquisitionInput) FirstTreasuryAcquisitionInput {
	input.TreasuryID = strings.TrimSpace(input.TreasuryID)
	input.EntryID = strings.TrimSpace(input.EntryID)
	input.AssetCode = strings.TrimSpace(input.AssetCode)
	input.Amount = strings.TrimSpace(input.Amount)
	input.SourceRef = strings.TrimSpace(input.SourceRef)
	return input
}

func normalizeActivateFundedTreasuryInput(input ActivateFundedTreasuryInput) ActivateFundedTreasuryInput {
	input.TreasuryID = strings.TrimSpace(input.TreasuryID)
	return input
}

func normalizePostBootstrapTreasuryAcquisitionInput(input PostBootstrapTreasuryAcquisitionInput) PostBootstrapTreasuryAcquisitionInput {
	input.TreasuryID = strings.TrimSpace(input.TreasuryID)
	return input
}

func acquireTreasuryMutationWriterLock(root string, lease WriterLockLease, now time.Time) (WriterLockRecord, error) {
	lock, _, err := AcquireWriterLock(root, now, storeTreasuryMutationLeaseDuration, lease)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, ErrWriterLockExpired) {
		return TakeoverWriterLock(root, now, storeTreasuryMutationLeaseDuration, lease)
	}
	return WriterLockRecord{}, err
}

func withLockedTreasuryMutation(root string, lock WriterLockRecord, now time.Time, fn func() error) error {
	guard, err := acquireStoreWriterGuard(root)
	if err != nil {
		return err
	}
	defer func() { _ = guard.release() }()

	if err := ValidateHeldWriterLock(root, lock, now); err != nil {
		return err
	}
	return fn()
}

func ensureTreasuryFirstAcquisitionNotRecorded(root, treasuryID, entryID string) error {
	if _, err := os.Stat(StoreTreasuryLedgerEntryPath(root, treasuryID, entryID)); err == nil {
		return fmt.Errorf("mission store treasury first acquisition entry %q already exists", entryID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	entries, err := ListTreasuryLedgerEntries(root, treasuryID)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.EntryKind == TreasuryLedgerEntryKindAcquisition {
			return fmt.Errorf(
				"mission store treasury %q already has recorded acquisition ledger entry %q",
				treasuryID,
				entry.EntryID,
			)
		}
	}
	return nil
}

func ensurePostBootstrapTreasuryAcquisitionEntryAvailable(root, treasuryID, entryID string) error {
	if _, err := os.Stat(StoreTreasuryLedgerEntryPath(root, treasuryID, entryID)); err == nil {
		return fmt.Errorf(
			"mission store treasury %q post-bootstrap acquisition derived entry %q already exists without committed consumed_entry_id",
			treasuryID,
			entryID,
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func commitFirstTreasuryAcquisitionBatch(root string, treasury TreasuryRecord, entry TreasuryLedgerEntry) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateTreasuryRecord(treasury); err != nil {
		return err
	}
	if err := ValidateTreasuryContainerLinks(root, treasury.ContainerRefs); err != nil {
		return err
	}
	if err := ValidateTreasuryLedgerEntry(entry); err != nil {
		return err
	}
	if entry.TreasuryID != treasury.TreasuryID {
		return fmt.Errorf(
			"mission store treasury first acquisition entry treasury_id %q does not match treasury %q",
			entry.TreasuryID,
			treasury.TreasuryID,
		)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindAcquisition {
		return fmt.Errorf(
			"mission store treasury first acquisition entry_kind %q is invalid",
			entry.EntryKind,
		)
	}

	// Write the ledger entry before the treasury state transition so the final
	// treasury write acts as the visible commit point. Any interrupted retry then
	// fails closed on the existing acquisition entry rather than appending a
	// second bootstrap acquisition.
	if err := storeBatchWriteRecord(StoreTreasuryLedgerEntryPath(root, entry.TreasuryID, entry.EntryID), entry); err != nil {
		return err
	}
	return storeBatchWriteRecord(StoreTreasuryPath(root, treasury.TreasuryID), treasury)
}

func commitPostBootstrapTreasuryAcquisitionBatch(root string, treasury TreasuryRecord, entry TreasuryLedgerEntry) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateTreasuryRecord(treasury); err != nil {
		return err
	}
	if err := ValidateTreasuryContainerLinks(root, treasury.ContainerRefs); err != nil {
		return err
	}
	if err := ValidateTreasuryLedgerEntry(entry); err != nil {
		return err
	}
	if entry.TreasuryID != treasury.TreasuryID {
		return fmt.Errorf(
			"mission store treasury post-bootstrap acquisition entry treasury_id %q does not match treasury %q",
			entry.TreasuryID,
			treasury.TreasuryID,
		)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindAcquisition {
		return fmt.Errorf(
			"mission store treasury post-bootstrap acquisition entry_kind %q is invalid",
			entry.EntryKind,
		)
	}

	if err := storeBatchWriteRecord(StoreTreasuryLedgerEntryPath(root, entry.TreasuryID, entry.EntryID), entry); err != nil {
		return err
	}
	return storeBatchWriteRecord(StoreTreasuryPath(root, treasury.TreasuryID), treasury)
}

func commitFundedTreasuryActivation(root string, treasury TreasuryRecord) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if err := ValidateTreasuryRecord(treasury); err != nil {
		return err
	}
	if err := ValidateTreasuryContainerLinks(root, treasury.ContainerRefs); err != nil {
		return err
	}
	if treasury.State != TreasuryStateActive {
		return fmt.Errorf(
			"mission store treasury activation target state %q is invalid",
			treasury.State,
		)
	}
	return storeBatchWriteRecord(StoreTreasuryPath(root, treasury.TreasuryID), treasury)
}

func derivePostBootstrapTreasuryAcquisitionEntryID(treasuryID string, block TreasuryPostBootstrapAcquisition) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(treasuryID),
		strings.TrimSpace(block.AssetCode),
		strings.TrimSpace(block.Amount),
		strings.TrimSpace(block.SourceRef),
		strings.TrimSpace(block.EvidenceLocator),
		block.ConfirmedAt.UTC().Format(time.RFC3339Nano),
	}, "\x1f")))
	return "entry-post-" + hex.EncodeToString(sum[:16])
}
