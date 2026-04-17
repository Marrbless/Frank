package missioncontrol

import (
	"fmt"
	"time"
)

type PostActiveTreasuryReinvestInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasuryReinvest is the single missioncontrol-owned
// execution producer for one post-active treasury reinvest. It resolves the
// committed reinvest block from Step.TreasuryRef truth, verifies that active
// treasury policy permits reinvest transactions, requires a committed
// autonomy-compatible target container, then delegates the durable paired
// ledger append and same-record consumed linkage to the treasury mutation.
func ProducePostActiveTreasuryReinvest(root string, lease WriterLockLease, input PostActiveTreasuryReinvestInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasuryReinvestInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveReinvest(ExecutionContext{
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
	if !containsTreasuryTransactionClass(permitted, TreasuryTransactionClassReinvest) {
		return fmt.Errorf(
			"mission store treasury %q post-active reinvest producer requires %q to be permitted in state %q",
			resolved.Treasury.TreasuryID,
			TreasuryTransactionClassReinvest,
			resolved.Treasury.State,
		)
	}
	if _, err := RequireAutonomyEligibleTarget(root, resolved.TargetContainer.EligibilityTargetRef); err != nil {
		return err
	}

	return RecordPostActiveTreasuryReinvest(root, lease, PostActiveTreasuryReinvestRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasuryReinvestInput(input PostActiveTreasuryReinvestInput) PostActiveTreasuryReinvestInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
