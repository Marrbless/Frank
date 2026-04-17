package tools

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func mustTaskStateReadoutJSON[T any](t *testing.T, readout string) T {
	t.Helper()

	var decoded T
	if err := json.Unmarshal([]byte(readout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return decoded
}

func assertTaskStateJSONObjectKeys(t *testing.T, value any, want ...string) {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON value = %#v, want object", value)
	}

	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)

	wantKeys := append([]string(nil), want...)
	sort.Strings(wantKeys)

	if len(got) != len(wantKeys) {
		t.Fatalf("JSON keys = %#v, want %#v", got, wantKeys)
	}
	for i := range got {
		if got[i] != wantKeys[i] {
			t.Fatalf("JSON keys = %#v, want %#v", got, wantKeys)
		}
	}
}

func mustTaskStateJSONArray(t *testing.T, value any, label string) []any {
	t.Helper()

	array, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", label, value)
	}
	return array
}

func mustTaskStateJSONObject(t *testing.T, readout string) map[string]any {
	t.Helper()

	return mustTaskStateReadoutJSON[map[string]any](t, readout)
}

func assertTaskStateResolvedCampaignPreflightJSONEnvelope(t *testing.T, value any) {
	t.Helper()

	preflight, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("campaign_preflight = %#v, want object", value)
	}
	assertTaskStateJSONObjectKeys(t, preflight, "accounts", "campaign", "containers", "identities")

	campaign, ok := preflight["campaign"].(map[string]any)
	if !ok {
		t.Fatalf("campaign_preflight.campaign = %#v, want object", preflight["campaign"])
	}
	wantCampaignKeys := []string{"campaign_id", "campaign_kind", "compliance_checks", "created_at", "display_name", "failure_threshold", "frank_object_refs", "governed_external_targets", "identity_mode", "objective", "record_version", "state", "stop_conditions", "updated_at"}
	if _, ok := campaign["zoho_email_addressing"]; ok {
		wantCampaignKeys = append(wantCampaignKeys, "zoho_email_addressing")
	}
	assertTaskStateJSONObjectKeys(t, campaign, wantCampaignKeys...)
	assertTaskStateJSONObjectKeys(t, campaign["failure_threshold"], "limit", "metric")
	if value, ok := campaign["zoho_email_addressing"]; ok {
		addressing, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("campaign_preflight.campaign.zoho_email_addressing = %#v, want object", value)
		}
		assertTaskStateJSONObjectKeys(t, addressing, "to", "cc", "bcc")
	}

	governedTargets := mustTaskStateJSONArray(t, campaign["governed_external_targets"], "campaign_preflight.campaign.governed_external_targets")
	if len(governedTargets) != 1 {
		t.Fatalf("campaign_preflight.campaign.governed_external_targets len = %d, want 1", len(governedTargets))
	}
	assertTaskStateJSONObjectKeys(t, governedTargets[0], "kind", "registry_id")

	objectRefs := mustTaskStateJSONArray(t, campaign["frank_object_refs"], "campaign_preflight.campaign.frank_object_refs")
	if len(objectRefs) != 3 {
		t.Fatalf("campaign_preflight.campaign.frank_object_refs len = %d, want 3", len(objectRefs))
	}
	for _, value := range objectRefs {
		assertTaskStateJSONObjectKeys(t, value, "kind", "object_id")
	}

	identities := mustTaskStateJSONArray(t, preflight["identities"], "campaign_preflight.identities")
	if len(identities) != 1 {
		t.Fatalf("campaign_preflight.identities len = %d, want 1", len(identities))
	}
	assertTaskStateJSONObjectKeys(t, identities[0], "created_at", "display_name", "eligibility_target_ref", "identity_id", "identity_kind", "identity_mode", "provider_or_platform_id", "record_version", "state", "updated_at")

	accounts := mustTaskStateJSONArray(t, preflight["accounts"], "campaign_preflight.accounts")
	if len(accounts) != 1 {
		t.Fatalf("campaign_preflight.accounts len = %d, want 1", len(accounts))
	}
	assertTaskStateJSONObjectKeys(t, accounts[0], "account_id", "account_kind", "control_model", "created_at", "eligibility_target_ref", "identity_id", "label", "provider_or_platform_id", "record_version", "recovery_model", "state", "updated_at")

	containers := mustTaskStateJSONArray(t, preflight["containers"], "campaign_preflight.containers")
	if len(containers) != 1 {
		t.Fatalf("campaign_preflight.containers len = %d, want 1", len(containers))
	}
	assertTaskStateJSONObjectKeys(t, containers[0], "container_class_id", "container_id", "container_kind", "created_at", "eligibility_target_ref", "label", "record_version", "state", "updated_at")
}

func assertTaskStateResolvedTreasuryPreflightJSONEnvelope(t *testing.T, value any) {
	t.Helper()

	preflight, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight = %#v, want object", value)
	}
	assertTaskStateJSONObjectKeys(t, preflight, "containers", "treasury")

	treasury, ok := preflight["treasury"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.treasury = %#v, want object", preflight["treasury"])
	}
	treasuryKeys := []string{"container_refs", "created_at", "display_name", "record_version", "state", "treasury_id", "updated_at", "zero_seed_policy"}
	if _, ok := treasury["bootstrap_acquisition"]; ok {
		treasuryKeys = append(treasuryKeys, "bootstrap_acquisition")
	}
	if _, ok := treasury["post_bootstrap_acquisition"]; ok {
		treasuryKeys = append(treasuryKeys, "post_bootstrap_acquisition")
	}
	if _, ok := treasury["post_active_suspend"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_suspend")
	}
	if _, ok := treasury["post_active_allocate"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_allocate")
	}
	if _, ok := treasury["post_active_reinvest"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_reinvest")
	}
	if _, ok := treasury["post_active_spend"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_spend")
	}
	if _, ok := treasury["post_active_transfer"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_transfer")
	}
	if _, ok := treasury["post_active_save"]; ok {
		treasuryKeys = append(treasuryKeys, "post_active_save")
	}
	assertTaskStateJSONObjectKeys(t, treasury, treasuryKeys...)
	if bootstrap, ok := treasury["bootstrap_acquisition"]; ok {
		bootstrapObject, ok := bootstrap.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.bootstrap_acquisition = %#v, want object", bootstrap)
		}
		assertTaskStateJSONObjectKeys(t, bootstrapObject, "amount", "asset_code", "confirmed_at", "entry_id", "evidence_locator", "source_ref")
	}
	if postBootstrap, ok := treasury["post_bootstrap_acquisition"]; ok {
		postBootstrapObject, ok := postBootstrap.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_bootstrap_acquisition = %#v, want object", postBootstrap)
		}
		postBootstrapKeys := []string{"amount", "asset_code", "confirmed_at", "evidence_locator", "source_ref"}
		if _, ok := postBootstrapObject["consumed_entry_id"]; ok {
			postBootstrapKeys = append(postBootstrapKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postBootstrapObject, postBootstrapKeys...)
	}
	if postActiveSuspend, ok := treasury["post_active_suspend"]; ok {
		postActiveSuspendObject, ok := postActiveSuspend.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_suspend = %#v, want object", postActiveSuspend)
		}
		postActiveSuspendKeys := []string{"reason", "source_ref"}
		if _, ok := postActiveSuspendObject["consumed_transition_id"]; ok {
			postActiveSuspendKeys = append(postActiveSuspendKeys, "consumed_transition_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveSuspendObject, postActiveSuspendKeys...)
	}
	if postActiveAllocate, ok := treasury["post_active_allocate"]; ok {
		postActiveAllocateObject, ok := postActiveAllocate.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_allocate = %#v, want object", postActiveAllocate)
		}
		postActiveAllocateKeys := []string{"allocation_target_ref", "amount", "asset_code", "source_container_ref", "source_ref"}
		if _, ok := postActiveAllocateObject["consumed_entry_id"]; ok {
			postActiveAllocateKeys = append(postActiveAllocateKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveAllocateObject, postActiveAllocateKeys...)
		assertTaskStateJSONObjectKeys(t, postActiveAllocateObject["source_container_ref"], "kind", "object_id")
	}
	if postActiveReinvest, ok := treasury["post_active_reinvest"]; ok {
		postActiveReinvestObject, ok := postActiveReinvest.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_reinvest = %#v, want object", postActiveReinvest)
		}
		postActiveReinvestKeys := []string{"confirmed_at", "evidence_locator", "source_amount", "source_asset_code", "source_container_ref", "source_ref", "target_amount", "target_asset_code", "target_container_ref"}
		if _, ok := postActiveReinvestObject["consumed_entry_id"]; ok {
			postActiveReinvestKeys = append(postActiveReinvestKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveReinvestObject, postActiveReinvestKeys...)
		assertTaskStateJSONObjectKeys(t, postActiveReinvestObject["source_container_ref"], "kind", "object_id")
		assertTaskStateJSONObjectKeys(t, postActiveReinvestObject["target_container_ref"], "kind", "object_id")
	}
	if postActiveSpend, ok := treasury["post_active_spend"]; ok {
		postActiveSpendObject, ok := postActiveSpend.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_spend = %#v, want object", postActiveSpend)
		}
		postActiveSpendKeys := []string{"amount", "asset_code", "source_container_ref", "source_ref", "target_ref"}
		if _, ok := postActiveSpendObject["evidence_locator"]; ok {
			postActiveSpendKeys = append(postActiveSpendKeys, "evidence_locator")
		}
		if _, ok := postActiveSpendObject["consumed_entry_id"]; ok {
			postActiveSpendKeys = append(postActiveSpendKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveSpendObject, postActiveSpendKeys...)
		assertTaskStateJSONObjectKeys(t, postActiveSpendObject["source_container_ref"], "kind", "object_id")
	}
	if postActiveTransfer, ok := treasury["post_active_transfer"]; ok {
		postActiveTransferObject, ok := postActiveTransfer.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_transfer = %#v, want object", postActiveTransfer)
		}
		postActiveTransferKeys := []string{"amount", "asset_code", "source_container_ref", "source_ref", "target_container_ref"}
		if _, ok := postActiveTransferObject["evidence_locator"]; ok {
			postActiveTransferKeys = append(postActiveTransferKeys, "evidence_locator")
		}
		if _, ok := postActiveTransferObject["consumed_entry_id"]; ok {
			postActiveTransferKeys = append(postActiveTransferKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveTransferObject, postActiveTransferKeys...)
		assertTaskStateJSONObjectKeys(t, postActiveTransferObject["source_container_ref"], "kind", "object_id")
		assertTaskStateJSONObjectKeys(t, postActiveTransferObject["target_container_ref"], "kind", "object_id")
	}
	if postActiveSave, ok := treasury["post_active_save"]; ok {
		postActiveSaveObject, ok := postActiveSave.(map[string]any)
		if !ok {
			t.Fatalf("treasury_preflight.treasury.post_active_save = %#v, want object", postActiveSave)
		}
		postActiveSaveKeys := []string{"amount", "asset_code", "source_ref", "target_container_id"}
		if _, ok := postActiveSaveObject["evidence_locator"]; ok {
			postActiveSaveKeys = append(postActiveSaveKeys, "evidence_locator")
		}
		if _, ok := postActiveSaveObject["consumed_entry_id"]; ok {
			postActiveSaveKeys = append(postActiveSaveKeys, "consumed_entry_id")
		}
		assertTaskStateJSONObjectKeys(t, postActiveSaveObject, postActiveSaveKeys...)
	}

	containerRefs := mustTaskStateJSONArray(t, treasury["container_refs"], "treasury_preflight.treasury.container_refs")
	if len(containerRefs) != 1 {
		t.Fatalf("treasury_preflight.treasury.container_refs len = %d, want 1", len(containerRefs))
	}
	assertTaskStateJSONObjectKeys(t, containerRefs[0], "kind", "object_id")

	containers := mustTaskStateJSONArray(t, preflight["containers"], "treasury_preflight.containers")
	if len(containers) != 1 {
		t.Fatalf("treasury_preflight.containers len = %d, want 1", len(containers))
	}
	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0] = %#v, want object", containers[0])
	}
	assertTaskStateJSONObjectKeys(t, container, "container_class_id", "container_id", "container_kind", "created_at", "eligibility_target_ref", "label", "record_version", "state", "updated_at")

	eligibility, ok := container["eligibility_target_ref"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0].eligibility_target_ref = %#v, want object", container["eligibility_target_ref"])
	}
	assertTaskStateJSONObjectKeys(t, eligibility, "kind", "registry_id")
}

func assertTaskStateReadoutAdapterBoundary(t *testing.T, readout string, allowCampaignPreflight bool, allowTreasuryPreflight bool) {
	t.Helper()

	var payload any
	if err := json.Unmarshal([]byte(readout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	forbiddenKeys := map[string]struct{}{
		"audience_class_or_target":              {},
		"message_family_or_participation_style": {},
		"cadence":                               {},
		"escalation_rules":                      {},
		"budget":                                {},
		"active_container_id":                   {},
		"custody_model":                         {},
		"permitted_transaction_classes":         {},
		"forbidden_transaction_classes":         {},
		"ledger_ref":                            {},
		"direction":                             {},
	}
	if !allowTreasuryPreflight {
		forbiddenKeys["treasury_preflight"] = struct{}{}
	}
	if !allowCampaignPreflight {
		forbiddenKeys["campaign_preflight"] = struct{}{}
	}

	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			for key, value := range typed {
				if _, ok := forbiddenKeys[key]; ok {
					t.Fatalf("readout unexpectedly contains adapter-only field %q: %s", key, readout)
				}
				walk(value)
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(payload)

	if strings.Contains(readout, `"status": "recorded"`) {
		t.Fatalf("readout unexpectedly contains derived-only ledger status: %s", readout)
	}
}
