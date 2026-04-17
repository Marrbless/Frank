package missioncontrol

import (
	"fmt"
	"time"
)

type PostActiveTreasurySaveInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasurySave is the single missioncontrol-owned execution
// producer for one post-active treasury save. It resolves the committed save
// block from Step.TreasuryRef truth, verifies that active treasury policy
// permits save transactions, requires a committed autonomy-compatible target
// container, then delegates the durable movement entry append and same-record
// consumed linkage to the treasury mutation.
func ProducePostActiveTreasurySave(root string, lease WriterLockLease, input PostActiveTreasurySaveInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasurySaveInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveSave(ExecutionContext{
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
	if !containsTreasuryTransactionClass(permitted, TreasuryTransactionClassSave) {
		return fmt.Errorf(
			"mission store treasury %q post-active save producer requires %q to be permitted in state %q",
			resolved.Treasury.TreasuryID,
			TreasuryTransactionClassSave,
			resolved.Treasury.State,
		)
	}
	if _, err := RequireAutonomyEligibleTarget(root, resolved.TargetContainer.EligibilityTargetRef); err != nil {
		return err
	}

	return RecordPostActiveTreasurySave(root, lease, PostActiveTreasurySaveRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasurySaveInput(input PostActiveTreasurySaveInput) PostActiveTreasurySaveInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}

func containsTreasuryTransactionClass(classes []TreasuryTransactionClass, want TreasuryTransactionClass) bool {
	for _, class := range classes {
		if class == want {
			return true
		}
	}
	return false
}
