package missioncontrol

import (
	"fmt"
	"time"
)

type PostActiveTreasurySpendInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasurySpend is the single missioncontrol-owned execution
// producer for one post-active treasury spend. It resolves the committed spend
// block from Step.TreasuryRef truth, verifies that active treasury policy
// permits spend transactions, then delegates the durable disposition entry
// append and same-record consumed linkage to the treasury mutation.
func ProducePostActiveTreasurySpend(root string, lease WriterLockLease, input PostActiveTreasurySpendInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasurySpendInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveSpend(ExecutionContext{
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
	if !containsTreasuryTransactionClass(permitted, TreasuryTransactionClassSpend) {
		return fmt.Errorf(
			"mission store treasury %q post-active spend producer requires %q to be permitted in state %q",
			resolved.Treasury.TreasuryID,
			TreasuryTransactionClassSpend,
			resolved.Treasury.State,
		)
	}

	return RecordPostActiveTreasurySpend(root, lease, PostActiveTreasurySpendRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasurySpendInput(input PostActiveTreasurySpendInput) PostActiveTreasurySpendInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
