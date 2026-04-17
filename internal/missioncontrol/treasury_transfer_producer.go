package missioncontrol

import "time"

type PostActiveTreasuryTransferInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasuryTransfer is the single missioncontrol-owned
// execution producer for one post-active treasury transfer. It resolves the
// committed transfer block from Step.TreasuryRef truth, requires truthful
// committed source and target container refs on the same TreasuryRecord, then
// delegates the durable movement entry append and same-record consumed linkage
// to the treasury mutation.
func ProducePostActiveTreasuryTransfer(root string, lease WriterLockLease, input PostActiveTreasuryTransferInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasuryTransferInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveTransfer(ExecutionContext{
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
	if _, err := RequireAutonomyEligibleTarget(root, resolved.TargetContainer.EligibilityTargetRef); err != nil {
		return err
	}

	return RecordPostActiveTreasuryTransfer(root, lease, PostActiveTreasuryTransferRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasuryTransferInput(input PostActiveTreasuryTransferInput) PostActiveTreasuryTransferInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
