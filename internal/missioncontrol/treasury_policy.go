package missioncontrol

import (
	"fmt"
	"strings"
	"time"
)

var treasuryActivationPolicyMutation = func(root string, lease WriterLockLease, input ActivateFundedTreasuryInput, now time.Time) error {
	return ActivateFundedTreasury(root, lease, input, now)
}

var treasuryBootstrapPolicyMutation = func(root string, lease WriterLockLease, input FirstTreasuryAcquisitionInput, now time.Time) error {
	return RecordFirstTreasuryAcquisition(root, lease, input, now)
}

type DefaultTreasuryBootstrapPolicyInput struct {
	TreasuryRef TreasuryRef
	EntryID     string
	AssetCode   string
	Amount      string
	SourceRef   string
}

type DefaultTreasuryActivationPolicyInput struct {
	TreasuryRef TreasuryRef
}

// ApplyDefaultTreasuryBootstrapPolicy is the narrow policy-owned caller for
// first-value treasury bootstrap. It reuses the existing treasury read model
// and current autonomy-eligibility policy, then delegates the durable ledger
// append and state transition to the landed RecordFirstTreasuryAcquisition
// mutation without widening into post-activation transaction execution or
// introducing new treasury truth.
func ApplyDefaultTreasuryBootstrapPolicy(root string, lease WriterLockLease, input DefaultTreasuryBootstrapPolicyInput, now time.Time) error {
	if err := ValidateStoreRoot(root); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	input = normalizeDefaultTreasuryBootstrapPolicyInput(input)
	treasury, err := ResolveTreasuryRef(root, input.TreasuryRef)
	if err != nil {
		return err
	}
	if treasury.State != TreasuryStateBootstrap {
		return fmt.Errorf(
			"mission store treasury %q default bootstrap policy requires state %q, got %q",
			treasury.TreasuryID,
			TreasuryStateBootstrap,
			treasury.State,
		)
	}
	if err := requireDefaultTreasuryBootstrapPolicy(root, treasury); err != nil {
		return err
	}

	return treasuryBootstrapPolicyMutation(root, lease, FirstTreasuryAcquisitionInput{
		TreasuryID: treasury.TreasuryID,
		EntryID:    input.EntryID,
		AssetCode:  input.AssetCode,
		Amount:     input.Amount,
		SourceRef:  input.SourceRef,
	}, now)
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

func normalizeDefaultTreasuryBootstrapPolicyInput(input DefaultTreasuryBootstrapPolicyInput) DefaultTreasuryBootstrapPolicyInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	input.EntryID = strings.TrimSpace(input.EntryID)
	input.AssetCode = strings.TrimSpace(input.AssetCode)
	input.Amount = strings.TrimSpace(input.Amount)
	input.SourceRef = strings.TrimSpace(input.SourceRef)
	return input
}

func normalizeDefaultTreasuryActivationPolicyInput(input DefaultTreasuryActivationPolicyInput) DefaultTreasuryActivationPolicyInput {
	input.TreasuryRef = NormalizeTreasuryRef(input.TreasuryRef)
	return input
}

func requireDefaultTreasuryBootstrapPolicy(root string, treasury TreasuryRecord) error {
	view := treasury.AsObjectView()
	if view.State != TreasuryStateBootstrap {
		return fmt.Errorf(
			"mission store treasury %q default bootstrap policy requires state %q, got %q",
			treasury.TreasuryID,
			TreasuryStateBootstrap,
			view.State,
		)
	}

	container, err := resolveDefaultTreasuryPolicyContainer(root, treasury, view, "default bootstrap policy")
	if err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, container.EligibilityTargetRef); err != nil {
		return err
	}
	return nil
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

	container, err := resolveDefaultTreasuryPolicyContainer(root, treasury, view, "default activation policy")
	if err != nil {
		return err
	}
	if _, err := RequireAutonomyEligibleTarget(root, container.EligibilityTargetRef); err != nil {
		return err
	}
	return nil
}

func resolveDefaultTreasuryPolicyContainer(root string, treasury TreasuryRecord, view TreasuryObjectView, policyName string) (FrankContainerRecord, error) {
	activeContainerID := strings.TrimSpace(view.ActiveContainerID)
	if activeContainerID == "" {
		return FrankContainerRecord{}, fmt.Errorf(
			"mission store treasury %q %s requires exactly one active_container_id derivable from container_refs",
			treasury.TreasuryID,
			strings.TrimSpace(policyName),
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
		"mission store treasury %q %s active_container_id %q is missing from container_refs",
		treasury.TreasuryID,
		strings.TrimSpace(policyName),
		activeContainerID,
	)
}
