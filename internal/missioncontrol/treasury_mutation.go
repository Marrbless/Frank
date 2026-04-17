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

type PostActiveTreasurySpendRecordInput struct {
	TreasuryID string
}

type PostActiveTreasuryTransferRecordInput struct {
	TreasuryID string
}

type PostActiveTreasurySaveRecordInput struct {
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

// RecordPostActiveTreasurySpend records exactly one post-active treasury spend
// by consuming the committed treasury.post_active_spend block on the same
// TreasuryRecord. The disposition entry is written before the treasury update
// so retries fail closed if a partial write leaves ambiguous state.
func RecordPostActiveTreasurySpend(root string, lease WriterLockLease, input PostActiveTreasurySpendRecordInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizePostActiveTreasurySpendRecordInput(input)
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
				"mission store treasury %q post-active spend requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateActive,
				treasury.State,
			)
		}
		activeContainerID, err := resolveTreasuryActiveContainerID(treasury)
		if err != nil {
			return err
		}
		if treasury.PostActiveSpend == nil {
			return fmt.Errorf(
				"mission store treasury %q post-active spend requires committed treasury.post_active_spend",
				treasury.TreasuryID,
			)
		}
		if strings.TrimSpace(treasury.PostActiveSpend.ConsumedEntryID) != "" {
			return fmt.Errorf(
				"mission store treasury %q post-active spend already consumed by entry %q",
				treasury.TreasuryID,
				strings.TrimSpace(treasury.PostActiveSpend.ConsumedEntryID),
			)
		}

		block := *treasury.PostActiveSpend
		sourceRef := NormalizeFrankRegistryObjectRef(block.SourceContainerRef)
		if sourceRef.ObjectID != activeContainerID {
			return fmt.Errorf(
				"mission store treasury %q post-active spend source_container_ref object_id %q must match active treasury container %q",
				treasury.TreasuryID,
				sourceRef.ObjectID,
				activeContainerID,
			)
		}
		if _, err := ResolveFrankRegistryObjectRef(root, sourceRef); err != nil {
			return fmt.Errorf(
				"mission store treasury %q post-active spend source_container_ref: %w",
				treasury.TreasuryID,
				err,
			)
		}

		entryID := derivePostActiveTreasurySpendEntryID(treasury.TreasuryID, block)
		if err := ensurePostActiveTreasurySpendEntryAvailable(root, treasury.TreasuryID, entryID); err != nil {
			return err
		}

		entry := normalizeTreasuryLedgerEntry(TreasuryLedgerEntry{
			RecordVersion: StoreRecordVersion,
			EntryID:       entryID,
			TreasuryID:    treasury.TreasuryID,
			EntryKind:     TreasuryLedgerEntryKindDisposition,
			AssetCode:     block.AssetCode,
			Amount:        block.Amount,
			CreatedAt:     now,
			SourceRef:     block.SourceRef,
		})
		if err := ValidateTreasuryLedgerEntry(entry); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.PostActiveSpend.ConsumedEntryID = entryID
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitPostActiveTreasurySpendBatch(root, updatedTreasury, entry)
	})
}

// RecordPostActiveTreasuryTransfer records exactly one post-active treasury
// transfer by consuming the committed treasury.post_active_transfer block on
// the same TreasuryRecord. The movement entry is written before the treasury
// update so retries fail closed if a partial write leaves ambiguous state.
func RecordPostActiveTreasuryTransfer(root string, lease WriterLockLease, input PostActiveTreasuryTransferRecordInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizePostActiveTreasuryTransferRecordInput(input)
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
				"mission store treasury %q post-active transfer requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateActive,
				treasury.State,
			)
		}
		activeContainerID, err := resolveTreasuryActiveContainerID(treasury)
		if err != nil {
			return err
		}
		if treasury.PostActiveTransfer == nil {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer requires committed treasury.post_active_transfer",
				treasury.TreasuryID,
			)
		}
		if strings.TrimSpace(treasury.PostActiveTransfer.ConsumedEntryID) != "" {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer already consumed by entry %q",
				treasury.TreasuryID,
				strings.TrimSpace(treasury.PostActiveTransfer.ConsumedEntryID),
			)
		}

		block := *treasury.PostActiveTransfer
		sourceRef := NormalizeFrankRegistryObjectRef(block.SourceContainerRef)
		targetRef := NormalizeFrankRegistryObjectRef(block.TargetContainerRef)
		if sourceRef.ObjectID != activeContainerID {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer source_container_ref object_id %q must match active treasury container %q",
				treasury.TreasuryID,
				sourceRef.ObjectID,
				activeContainerID,
			)
		}
		if sourceRef.ObjectID == targetRef.ObjectID {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer requires distinct source and target containers %q",
				treasury.TreasuryID,
				sourceRef.ObjectID,
			)
		}
		if _, err := ResolveFrankRegistryObjectRef(root, sourceRef); err != nil {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer source_container_ref: %w",
				treasury.TreasuryID,
				err,
			)
		}
		if _, err := ResolveFrankRegistryObjectRef(root, targetRef); err != nil {
			return fmt.Errorf(
				"mission store treasury %q post-active transfer target_container_ref: %w",
				treasury.TreasuryID,
				err,
			)
		}

		entryID := derivePostActiveTreasuryTransferEntryID(treasury.TreasuryID, block)
		if err := ensurePostActiveTreasuryTransferEntryAvailable(root, treasury.TreasuryID, entryID); err != nil {
			return err
		}

		entry := normalizeTreasuryLedgerEntry(TreasuryLedgerEntry{
			RecordVersion: StoreRecordVersion,
			EntryID:       entryID,
			TreasuryID:    treasury.TreasuryID,
			EntryKind:     TreasuryLedgerEntryKindMovement,
			AssetCode:     block.AssetCode,
			Amount:        block.Amount,
			CreatedAt:     now,
			SourceRef:     block.SourceRef,
		})
		if err := ValidateTreasuryLedgerEntry(entry); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.PostActiveTransfer.ConsumedEntryID = entryID
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitPostActiveTreasuryTransferBatch(root, updatedTreasury, entry)
	})
}

// RecordPostActiveTreasurySave records exactly one post-active treasury save by
// consuming the committed treasury.post_active_save block on the same
// TreasuryRecord. The movement entry is written before the treasury update so
// retries fail closed if a partial write leaves ambiguous state.
func RecordPostActiveTreasurySave(root string, lease WriterLockLease, input PostActiveTreasurySaveRecordInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizePostActiveTreasurySaveRecordInput(input)
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
				"mission store treasury %q post-active save requires state %q, got %q",
				treasury.TreasuryID,
				TreasuryStateActive,
				treasury.State,
			)
		}
		activeContainerID, err := resolveTreasuryActiveContainerID(treasury)
		if err != nil {
			return err
		}
		if treasury.PostActiveSave == nil {
			return fmt.Errorf(
				"mission store treasury %q post-active save requires committed treasury.post_active_save",
				treasury.TreasuryID,
			)
		}
		if strings.TrimSpace(treasury.PostActiveSave.ConsumedEntryID) != "" {
			return fmt.Errorf(
				"mission store treasury %q post-active save already consumed by entry %q",
				treasury.TreasuryID,
				strings.TrimSpace(treasury.PostActiveSave.ConsumedEntryID),
			)
		}

		block := *treasury.PostActiveSave
		if block.TargetContainerID == activeContainerID {
			return fmt.Errorf(
				"mission store treasury %q post-active save target_container_id %q must differ from active container %q",
				treasury.TreasuryID,
				block.TargetContainerID,
				activeContainerID,
			)
		}
		if _, err := LoadFrankContainerRecord(root, block.TargetContainerID); err != nil {
			return fmt.Errorf(
				"mission store treasury %q post-active save target_container_id %q: %w",
				treasury.TreasuryID,
				block.TargetContainerID,
				err,
			)
		}

		entryID := derivePostActiveTreasurySaveEntryID(treasury.TreasuryID, block)
		if err := ensurePostActiveTreasurySaveEntryAvailable(root, treasury.TreasuryID, entryID); err != nil {
			return err
		}

		entry := normalizeTreasuryLedgerEntry(TreasuryLedgerEntry{
			RecordVersion: StoreRecordVersion,
			EntryID:       entryID,
			TreasuryID:    treasury.TreasuryID,
			EntryKind:     TreasuryLedgerEntryKindMovement,
			AssetCode:     block.AssetCode,
			Amount:        block.Amount,
			CreatedAt:     now,
			SourceRef:     block.SourceRef,
		})
		if err := ValidateTreasuryLedgerEntry(entry); err != nil {
			return err
		}

		updatedTreasury := normalizeTreasuryRecord(treasury)
		updatedTreasury.PostActiveSave.ConsumedEntryID = entryID
		updatedTreasury.UpdatedAt = now
		if err := ValidateTreasuryRecord(updatedTreasury); err != nil {
			return err
		}
		if err := ValidateTreasuryContainerLinks(root, updatedTreasury.ContainerRefs); err != nil {
			return err
		}

		return commitPostActiveTreasurySaveBatch(root, updatedTreasury, entry)
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

func normalizePostActiveTreasurySpendRecordInput(input PostActiveTreasurySpendRecordInput) PostActiveTreasurySpendRecordInput {
	input.TreasuryID = strings.TrimSpace(input.TreasuryID)
	return input
}

func normalizePostActiveTreasuryTransferRecordInput(input PostActiveTreasuryTransferRecordInput) PostActiveTreasuryTransferRecordInput {
	input.TreasuryID = strings.TrimSpace(input.TreasuryID)
	return input
}

func normalizePostActiveTreasurySaveRecordInput(input PostActiveTreasurySaveRecordInput) PostActiveTreasurySaveRecordInput {
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

func ensurePostActiveTreasurySaveEntryAvailable(root, treasuryID, entryID string) error {
	if _, err := os.Stat(StoreTreasuryLedgerEntryPath(root, treasuryID, entryID)); err == nil {
		return fmt.Errorf(
			"mission store treasury %q post-active save derived entry %q already exists without committed consumed_entry_id",
			treasuryID,
			entryID,
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ensurePostActiveTreasurySpendEntryAvailable(root, treasuryID, entryID string) error {
	if _, err := os.Stat(StoreTreasuryLedgerEntryPath(root, treasuryID, entryID)); err == nil {
		return fmt.Errorf(
			"mission store treasury %q post-active spend derived entry %q already exists without committed consumed_entry_id",
			treasuryID,
			entryID,
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ensurePostActiveTreasuryTransferEntryAvailable(root, treasuryID, entryID string) error {
	if _, err := os.Stat(StoreTreasuryLedgerEntryPath(root, treasuryID, entryID)); err == nil {
		return fmt.Errorf(
			"mission store treasury %q post-active transfer derived entry %q already exists without committed consumed_entry_id",
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

func commitPostActiveTreasurySaveBatch(root string, treasury TreasuryRecord, entry TreasuryLedgerEntry) error {
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
			"mission store treasury post-active save entry treasury_id %q does not match treasury %q",
			entry.TreasuryID,
			treasury.TreasuryID,
		)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindMovement {
		return fmt.Errorf(
			"mission store treasury post-active save entry_kind %q is invalid",
			entry.EntryKind,
		)
	}

	if err := storeBatchWriteRecord(StoreTreasuryLedgerEntryPath(root, entry.TreasuryID, entry.EntryID), entry); err != nil {
		return err
	}
	return storeBatchWriteRecord(StoreTreasuryPath(root, treasury.TreasuryID), treasury)
}

func commitPostActiveTreasurySpendBatch(root string, treasury TreasuryRecord, entry TreasuryLedgerEntry) error {
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
			"mission store treasury post-active spend entry treasury_id %q does not match treasury %q",
			entry.TreasuryID,
			treasury.TreasuryID,
		)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindDisposition {
		return fmt.Errorf(
			"mission store treasury post-active spend entry_kind %q is invalid",
			entry.EntryKind,
		)
	}

	if err := storeBatchWriteRecord(StoreTreasuryLedgerEntryPath(root, entry.TreasuryID, entry.EntryID), entry); err != nil {
		return err
	}
	return storeBatchWriteRecord(StoreTreasuryPath(root, treasury.TreasuryID), treasury)
}

func commitPostActiveTreasuryTransferBatch(root string, treasury TreasuryRecord, entry TreasuryLedgerEntry) error {
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
			"mission store treasury post-active transfer entry treasury_id %q does not match treasury %q",
			entry.TreasuryID,
			treasury.TreasuryID,
		)
	}
	if entry.EntryKind != TreasuryLedgerEntryKindMovement {
		return fmt.Errorf(
			"mission store treasury post-active transfer entry_kind %q is invalid",
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

func derivePostActiveTreasurySaveEntryID(treasuryID string, block TreasuryPostActiveSave) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(treasuryID),
		strings.TrimSpace(block.AssetCode),
		strings.TrimSpace(block.Amount),
		strings.TrimSpace(block.TargetContainerID),
		strings.TrimSpace(block.SourceRef),
		strings.TrimSpace(block.EvidenceLocator),
	}, "\x1f")))
	return "entry-save-" + hex.EncodeToString(sum[:16])
}

func derivePostActiveTreasurySpendEntryID(treasuryID string, block TreasuryPostActiveSpend) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(treasuryID),
		normalizedFrankRegistryObjectRefKey(NormalizeFrankRegistryObjectRef(block.SourceContainerRef)),
		strings.TrimSpace(block.TargetRef),
		strings.TrimSpace(block.AssetCode),
		strings.TrimSpace(block.Amount),
		strings.TrimSpace(block.SourceRef),
		strings.TrimSpace(block.EvidenceLocator),
	}, "\x1f")))
	return "entry-spend-" + hex.EncodeToString(sum[:16])
}

func derivePostActiveTreasuryTransferEntryID(treasuryID string, block TreasuryPostActiveTransfer) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(treasuryID),
		normalizedFrankRegistryObjectRefKey(NormalizeFrankRegistryObjectRef(block.SourceContainerRef)),
		normalizedFrankRegistryObjectRefKey(NormalizeFrankRegistryObjectRef(block.TargetContainerRef)),
		strings.TrimSpace(block.AssetCode),
		strings.TrimSpace(block.Amount),
		strings.TrimSpace(block.SourceRef),
		strings.TrimSpace(block.EvidenceLocator),
	}, "\x1f")))
	return "entry-transfer-" + hex.EncodeToString(sum[:16])
}
