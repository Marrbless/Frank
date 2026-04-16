package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

type FirstValueTreasuryBootstrapInput struct {
	TreasuryRef TreasuryRef
	EntryID     string
}

// ProduceFirstValueTreasuryBootstrap is the single missioncontrol-owned
// execution producer for first-value treasury bootstrap after the committed
// first acquisition has already landed. It fails closed unless the treasury is
// funded and exactly one committed acquisition entry matches the supplied
// entry_id, then delegates the funded-to-active transition to the existing
// treasury activation producer without widening treasury truth.
func ProduceFirstValueTreasuryBootstrap(root string, lease WriterLockLease, input FirstValueTreasuryBootstrapInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizeFirstValueTreasuryBootstrapInput(input)
	treasury, err := ResolveTreasuryRef(root, input.TreasuryRef)
	if err != nil {
		return err
	}
	if treasury.State != TreasuryStateFunded {
		return fmt.Errorf(
			"mission store treasury %q first-value bootstrap producer requires state %q, got %q",
			treasury.TreasuryID,
			TreasuryStateFunded,
			treasury.State,
		)
	}
	if err := requireDefaultTreasuryActivationPolicy(root, treasury); err != nil {
		return err
	}
	if err := requireCommittedFirstValueTreasuryBootstrapEntry(root, treasury, input.EntryID); err != nil {
		return err
	}

	return ProduceFundedTreasuryActivation(root, lease, DefaultTreasuryActivationPolicyInput{
		TreasuryRef: input.TreasuryRef,
	}, now)
}

func normalizeFirstValueTreasuryBootstrapInput(input FirstValueTreasuryBootstrapInput) FirstValueTreasuryBootstrapInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	input.EntryID = strings.TrimSpace(input.EntryID)
	return input
}

func requireCommittedFirstValueTreasuryBootstrapEntry(root string, treasury TreasuryRecord, entryID string) error {
	if err := validateTreasuryEntryID(entryID, "mission store treasury first-value bootstrap producer"); err != nil {
		return err
	}

	entry, err := LoadTreasuryLedgerEntry(root, treasury.TreasuryID, entryID)
	if err != nil {
		if err == ErrTreasuryLedgerEntryNotFound {
			return fmt.Errorf(
				"mission store treasury %q first-value bootstrap producer requires committed acquisition ledger entry %q",
				treasury.TreasuryID,
				entryID,
			)
		}
		return err
	}
	if entry.EntryKind != TreasuryLedgerEntryKindAcquisition {
		return fmt.Errorf(
			"mission store treasury %q first-value bootstrap producer requires acquisition ledger entry %q, got %q",
			treasury.TreasuryID,
			entry.EntryID,
			entry.EntryKind,
		)
	}

	entries, err := ListTreasuryLedgerEntries(root, treasury.TreasuryID)
	if err != nil {
		return err
	}
	acquisitions := 0
	for _, candidate := range entries {
		if candidate.EntryKind != TreasuryLedgerEntryKindAcquisition {
			continue
		}
		acquisitions++
	}
	if acquisitions != 1 {
		return fmt.Errorf(
			"mission store treasury %q first-value bootstrap producer requires exactly one committed acquisition ledger entry, got %d",
			treasury.TreasuryID,
			acquisitions,
		)
	}
	return nil
}
