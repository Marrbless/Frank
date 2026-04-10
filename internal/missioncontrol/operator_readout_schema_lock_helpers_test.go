package missioncontrol

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

var operatorReadoutAdapterOnlyFields = []string{
	"\"audience_class_or_target\"",
	"\"message_family_or_participation_style\"",
	"\"cadence\"",
	"\"escalation_rules\"",
	"\"budget\":",
	"\"active_container_id\"",
	"\"custody_model\"",
	"\"permitted_transaction_classes\"",
	"\"forbidden_transaction_classes\"",
	"\"ledger_ref\"",
	"\"direction\":\"internal\"",
	"\"status\":\"recorded\"",
}

func assertJSONObjectKeys(t *testing.T, value any, want ...string) {
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

func mustJSONArray(t *testing.T, value any, label string) []any {
	t.Helper()

	array, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", label, value)
	}
	return array
}

func mustOperatorReadoutJSONObject(t *testing.T, readout string) map[string]any {
	t.Helper()

	var object map[string]any
	if err := json.Unmarshal([]byte(readout), &object); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return object
}

func assertOperatorReadoutAdapterBoundary(t *testing.T, readout, label string, allowTreasuryPreflight bool) {
	t.Helper()

	if !allowTreasuryPreflight && strings.Contains(readout, "\"treasury_preflight\"") {
		t.Fatalf("%s unexpectedly contains %s: %s", label, "\"treasury_preflight\"", readout)
	}

	for _, key := range operatorReadoutAdapterOnlyFields {
		if strings.Contains(readout, key) {
			t.Fatalf("%s unexpectedly contains %s: %s", label, key, readout)
		}
	}
}

func assertResolvedTreasuryPreflightJSONEnvelope(t *testing.T, value any) {
	t.Helper()

	preflight, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight = %#v, want object", value)
	}
	assertJSONObjectKeys(t, preflight, "containers", "treasury")

	treasury, ok := preflight["treasury"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.treasury = %#v, want object", preflight["treasury"])
	}
	assertJSONObjectKeys(t, treasury, "container_refs", "created_at", "display_name", "record_version", "state", "treasury_id", "updated_at", "zero_seed_policy")

	containerRefs := mustJSONArray(t, treasury["container_refs"], "treasury_preflight.treasury.container_refs")
	if len(containerRefs) != 1 {
		t.Fatalf("treasury_preflight.treasury.container_refs len = %d, want 1", len(containerRefs))
	}
	assertJSONObjectKeys(t, containerRefs[0], "kind", "object_id")

	containers := mustJSONArray(t, preflight["containers"], "treasury_preflight.containers")
	if len(containers) != 1 {
		t.Fatalf("treasury_preflight.containers len = %d, want 1", len(containers))
	}
	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0] = %#v, want object", containers[0])
	}
	assertJSONObjectKeys(t, container, "container_class_id", "container_id", "container_kind", "created_at", "eligibility_target_ref", "label", "record_version", "state", "updated_at")

	eligibility, ok := container["eligibility_target_ref"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0].eligibility_target_ref = %#v, want object", container["eligibility_target_ref"])
	}
	assertJSONObjectKeys(t, eligibility, "kind", "registry_id")
}
