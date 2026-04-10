package tools

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

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
	assertTaskStateJSONObjectKeys(t, treasury, "container_refs", "created_at", "display_name", "record_version", "state", "treasury_id", "updated_at", "zero_seed_policy")

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

func assertTaskStateReadoutAdapterBoundary(t *testing.T, readout string, allowTreasuryPreflight bool) {
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
