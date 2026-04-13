package missioncontrol

import "time"

// ProduceFirstValueTreasuryBootstrap is the single missioncontrol-owned
// execution producer for first-value treasury bootstrap. It accepts the
// already-landed default bootstrap policy input surface and delegates to the
// default policy without widening into post-funding transaction execution or
// inventing any new treasury identity, eligibility, or object truth.
func ProduceFirstValueTreasuryBootstrap(root string, lease WriterLockLease, input DefaultTreasuryBootstrapPolicyInput, now time.Time) error {
	return ApplyDefaultTreasuryBootstrapPolicy(root, lease, input, now)
}
