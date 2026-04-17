package missioncontrol

import "time"

type PostSuspendTreasuryResumeInput struct {
	TreasuryRef TreasuryRef
}

// ProducePostSuspendTreasuryResume is the single missioncontrol-owned
// execution producer for one post-suspend treasury resume. It resolves the
// committed resume block from Step.TreasuryRef truth, then delegates the
// durable suspended->active state transition and same-record consumed linkage
// to the treasury mutation without introducing a second lifecycle registry.
func ProducePostSuspendTreasuryResume(root string, lease WriterLockLease, input PostSuspendTreasuryResumeInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}

	input = normalizePostSuspendTreasuryResumeInput(input)
	resolved, err := ResolveExecutionContextTreasuryPostSuspendResume(ExecutionContext{
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

	return RecordPostSuspendTreasuryResume(root, lease, PostSuspendTreasuryResumeRecordInput{
		TreasuryID: resolved.Treasury.TreasuryID,
	}, now)
}

func normalizePostSuspendTreasuryResumeInput(input PostSuspendTreasuryResumeInput) PostSuspendTreasuryResumeInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}
