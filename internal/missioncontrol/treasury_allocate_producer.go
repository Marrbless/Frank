package missioncontrol

import (
	"fmt"
	"time"
)

type PostActiveTreasuryAllocateInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasuryAllocate is the single missioncontrol-owned
// execution producer for one post-active treasury allocate. It resolves the
// committed allocate block from Step.TreasuryRef truth, verifies that active
// treasury policy permits allocate transactions, then delegates the durable
// internal movement entry append and same-record consumed linkage to the
// treasury mutation.
func ProducePostActiveTreasuryAllocate(root string, lease WriterLockLease, input PostActiveTreasuryAllocateInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasuryAllocateInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveAllocate(ExecutionContext{
		MissionStoreRoot: root,
		Step: &Step{
			TreasuryRef: &input.TreasuryRef,
		},
	})
	if err != nil {
		return err
	}
	if resolved == nil {
		return nil
	}

	permitted, _ := DefaultTreasuryTransactionPolicy(resolved.Treasury.State)
	if !containsTreasuryTransactionClass(permitted, TreasuryTransactionClassAllocate) {
		return fmt.Errorf(
			"mission store treasury %q post-active allocate producer requires %q to be permitted in state %q",
			resolved.Treasury.TreasuryID,
			TreasuryTransactionClassAllocate,
			resolved.Treasury.State,
		)
	}

	return RecordPostActiveTreasuryAllocate(root, lease, PostActiveTreasuryAllocateRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasuryAllocateInput(input PostActiveTreasuryAllocateInput) PostActiveTreasuryAllocateInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
