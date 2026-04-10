package agent

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func assertLoopCheckInJSONObjectKeys(t *testing.T, value any, want ...string) {
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

func mustLoopCheckInJSONArray(t *testing.T, value any, label string) []any {
	t.Helper()

	array, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", label, value)
	}
	return array
}

func assertLoopCheckInResolvedTreasuryPreflightJSONEnvelope(t *testing.T, value any) {
	t.Helper()

	preflight, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight = %#v, want object", value)
	}
	assertLoopCheckInJSONObjectKeys(t, preflight, "containers", "treasury")

	treasury, ok := preflight["treasury"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.treasury = %#v, want object", preflight["treasury"])
	}
	assertLoopCheckInJSONObjectKeys(t, treasury, "container_refs", "created_at", "display_name", "record_version", "state", "treasury_id", "updated_at", "zero_seed_policy")

	containerRefs := mustLoopCheckInJSONArray(t, treasury["container_refs"], "treasury_preflight.treasury.container_refs")
	if len(containerRefs) != 1 {
		t.Fatalf("treasury_preflight.treasury.container_refs len = %d, want 1", len(containerRefs))
	}
	assertLoopCheckInJSONObjectKeys(t, containerRefs[0], "kind", "object_id")

	containers := mustLoopCheckInJSONArray(t, preflight["containers"], "treasury_preflight.containers")
	if len(containers) != 1 {
		t.Fatalf("treasury_preflight.containers len = %d, want 1", len(containers))
	}
	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0] = %#v, want object", containers[0])
	}
	assertLoopCheckInJSONObjectKeys(t, container, "container_class_id", "container_id", "container_kind", "created_at", "eligibility_target_ref", "label", "record_version", "state", "updated_at")

	eligibility, ok := container["eligibility_target_ref"].(map[string]any)
	if !ok {
		t.Fatalf("treasury_preflight.containers[0].eligibility_target_ref = %#v, want object", container["eligibility_target_ref"])
	}
	assertLoopCheckInJSONObjectKeys(t, eligibility, "kind", "registry_id")
}

func assertLoopCheckInOperatorStatusEnvelope(t *testing.T, content, prefix string, allowTreasuryPreflight bool, wantKeys ...string) map[string]any {
	t.Helper()

	if !strings.HasPrefix(content, prefix) {
		t.Fatalf("content = %q, want prefix %q", content, prefix)
	}

	payload := strings.TrimPrefix(content, prefix)
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
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
					t.Fatalf("notification summary unexpectedly contains adapter-only field %q: %s", key, payload)
				}
				walk(value)
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(decoded)

	if strings.Contains(payload, `"status": "recorded"`) {
		t.Fatalf("notification summary unexpectedly contains derived-only ledger status: %s", payload)
	}

	object, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("summary payload = %#v, want object", decoded)
	}
	assertLoopCheckInJSONObjectKeys(t, object, wantKeys...)

	if allowTreasuryPreflight {
		assertLoopCheckInResolvedTreasuryPreflightJSONEnvelope(t, object["treasury_preflight"])
	}

	return object
}
