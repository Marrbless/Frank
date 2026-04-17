package missioncontrol

import "time"

type PostActiveTreasurySuspendInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostActiveTreasurySuspend is the single missioncontrol-owned
// execution producer for one post-active treasury suspend. It resolves the
// committed suspend block from Step.TreasuryRef truth, then delegates the
// durable active->suspended state transition and same-record consumed linkage
// to the treasury mutation without introducing a second lifecycle registry.
func ProducePostActiveTreasurySuspend(root string, lease WriterLockLease, input PostActiveTreasurySuspendInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostActiveTreasurySuspendInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostActiveSuspend(ExecutionContext{
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

	return RecordPostActiveTreasurySuspend(root, lease, PostActiveTreasurySuspendRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostActiveTreasurySuspendInput(input PostActiveTreasurySuspendInput) PostActiveTreasurySuspendInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
