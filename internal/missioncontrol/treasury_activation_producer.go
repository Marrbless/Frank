package missioncontrol

import "time"

// ProduceFundedTreasuryActivation is the single missioncontrol-owned execution
// producer for funded-to-active treasury activation. It accepts the already-
// landed default activation policy input surface and delegates to the default
// policy without widening into transaction execution or inventing any new
// treasury identity, eligibility, or object truth.
func ProduceFundedTreasuryActivation(root string, lease WriterLockLease, input DefaultTreasuryActivationPolicyInput, now time.Time) error {
	return ApplyDefaultTreasuryActivationPolicy(root, lease, input, now)
}
