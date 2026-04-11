package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

var treasuryActivationPolicyMutation = func(root string, lease WriterLockLease, input ActivateFundedTreasuryInput, now time.Time) error {
	return ActivateFundedTreasury(root, lease, input, now)
}

type DefaultTreasuryActivationPolicyInput struct {
	TreasuryRef TreasuryRef
}

// ApplyDefaultTreasuryActivationPolicy is the narrow policy-owned caller for
// treasury activation. It reuses the existing treasury read model and current
// autonomy-eligibility policy, then delegates the state transition to the
// landed ActivateFundedTreasury mutation without widening into transaction
// execution or introducing new treasury truth.
func ApplyDefaultTreasuryActivationPolicy(root string, lease WriterLockLease, input DefaultTreasuryActivationPolicyInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizeDefaultTreasuryActivationPolicyInput(input)
	treasury, err := ResolveTreasuryRef(root, input.TreasuryRef)
	if err != nil {
		return err
	}
	if treasury.State != TreasuryStateFunded {
		return fmt.Errorf(
			"mission store treasury %q default activation policy requires state %q, got %q",
			treasury.TreasuryID,
			TreasuryStateFunded,
			treasury.State,
		)
	}
	if err := requireDefaultTreasuryActivationPolicy(root, treasury); err != nil {
		return err
	}

	return treasuryActivationPolicyMutation(root, lease, ActivateFundedTreasuryInput{
		TreasuryID: treasury.TreasuryID,
	}, now)
}

func normalizeDefaultTreasuryActivationPolicyInput(input DefaultTreasuryActivationPolicyInput) DefaultTreasuryActivationPolicyInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}

func requireDefaultTreasuryActivationPolicy(root string, treasury TreasuryRecord) error {
	view := treasury.AsObjectView()
	if view.State != TreasuryStateFunded {
		return fmt.Errorf(
			"mission store treasury %q default activation policy requires state %q, got %q",
			treasury.TreasuryID,
			TreasuryStateFunded,
			view.State,
		)
	}

	activePermitted, _ := DefaultTreasuryTransactionPolicy(TreasuryStateActive)
	if len(activePermitted) == 0 {
		return fmt.Errorf("mission store treasury %q default activation policy does not permit activation", treasury.TreasuryID)
	}

	container, err := resolveDefaultTreasuryActivationPolicyContainer(root, treasury, view)
	if err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, container.EligibilityTargetRef); err != nil {
		return err
	}
	return nil
}

func resolveDefaultTreasuryActivationPolicyContainer(root string, treasury TreasuryRecord, view TreasuryObjectView) (FrankContainerRecord, error) {
	activeContainerID := strings.TrimSpace(view.ActiveContainerID)
	if activeContainerID == "" {
		return FrankContainerRecord{}, fmt.Errorf(
			"mission store treasury %q default activation policy requires exactly one active_container_id derivable from container_refs",
			treasury.TreasuryID,
		)
	}

	resolvedRefs, err := ResolveFrankRegistryObjectRefs(root, treasury.ContainerRefs)
	if err != nil {
		return FrankContainerRecord{}, err
	}
	for _, resolved := range resolvedRefs {
		if strings.TrimSpace(resolved.Ref.ObjectID) != activeContainerID {
			continue
		}
		if resolved.Container == nil {
			return FrankContainerRecord{}, fmt.Errorf(
				"resolve treasury container ref kind %q object_id %q: expected Frank container record",
				resolved.Ref.Kind,
				resolved.Ref.ObjectID,
			)
		}
		return *resolved.Container, nil
	}

	return FrankContainerRecord{}, fmt.Errorf(
		"mission store treasury %q default activation policy active_container_id %q is missing from container_refs",
		treasury.TreasuryID,
		activeContainerID,
	)
}
