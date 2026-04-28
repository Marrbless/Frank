package missioncontrol

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRuntimeExtensionPackRecordRoundTripListReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	_, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now, "extension-pack-b", nil))
	if err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(extension-pack-b) error = %v", err)
	}

	want := validRuntimeExtensionPackRecord(now.Add(time.Minute), " extension-pack-a ", func(record *RuntimeExtensionPackRecord) {
		record.ParentExtensionPackID = " extension-pack-root "
		record.Extensions = []string{" local-router ", " "}
		record.DeclaredTools = []RuntimeExtensionToolDeclaration{{
			ToolName:             " inspect_local_state ",
			PermissionRefs:       []string{" local_read ", " "},
			CompatibilitySummary: " no api changes ",
		}}
		record.DeclaredEvents = []string{" wake_cycle ", " "}
		record.DeclaredPermissions = []string{" local_read ", " "}
		record.ExternalSideEffects = []string{" "}
		record.CompatibilityContractRef = " compat-v1 "
		record.ChangeSummary = " local extension update "
		record.CreatedBy = " operator "
	})
	got, changed, err := StoreRuntimeExtensionPackRecord(root, want)
	if err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(extension-pack-a) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreRuntimeExtensionPackRecord(extension-pack-a) changed = false, want true")
	}

	want.RecordVersion = StoreRecordVersion
	want = NormalizeRuntimeExtensionPackRecord(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StoreRuntimeExtensionPackRecord() = %#v, want %#v", got, want)
	}
	loaded, err := LoadRuntimeExtensionPackRecord(root, "extension-pack-a")
	if err != nil {
		t.Fatalf("LoadRuntimeExtensionPackRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadRuntimeExtensionPackRecord() = %#v, want %#v", loaded, want)
	}
	records, err := ListRuntimeExtensionPackRecords(root)
	if err != nil {
		t.Fatalf("ListRuntimeExtensionPackRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].ExtensionPackID != "extension-pack-a" || records[1].ExtensionPackID != "extension-pack-b" {
		t.Fatalf("ListRuntimeExtensionPackRecords() = %#v, want extension-pack-a then extension-pack-b", records)
	}

	replayed, changed, err := StoreRuntimeExtensionPackRecord(root, want)
	if err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreRuntimeExtensionPackRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StoreRuntimeExtensionPackRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.ChangeSummary = "different extension change"
	if _, _, err := StoreRuntimeExtensionPackRecord(root, divergent); err == nil {
		t.Fatal("StoreRuntimeExtensionPackRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store runtime extension pack "extension-pack-a" already exists`) {
		t.Fatalf("StoreRuntimeExtensionPackRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestRuntimeExtensionPackRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*RuntimeExtensionPackRecord)
		want string
	}{
		{name: "missing id", edit: func(record *RuntimeExtensionPackRecord) { record.ExtensionPackID = " " }, want: "extension_pack_id is required"},
		{name: "missing extensions", edit: func(record *RuntimeExtensionPackRecord) { record.Extensions = nil }, want: "extensions are required"},
		{name: "missing tools", edit: func(record *RuntimeExtensionPackRecord) { record.DeclaredTools = nil }, want: "declared_tools are required"},
		{name: "missing events", edit: func(record *RuntimeExtensionPackRecord) { record.DeclaredEvents = nil }, want: "declared_events are required"},
		{name: "missing permissions", edit: func(record *RuntimeExtensionPackRecord) { record.DeclaredPermissions = nil }, want: "declared_permissions are required"},
		{name: "policy permission", edit: func(record *RuntimeExtensionPackRecord) {
			record.DeclaredPermissions = []string{"local_read", "approval_syntax.write"}
		}, want: string(RejectionCodeV4PolicyMutationForbidden)},
		{name: "policy tool permission", edit: func(record *RuntimeExtensionPackRecord) {
			record.DeclaredTools[0].PermissionRefs = []string{"autonomy_predicate_write"}
		}, want: string(RejectionCodeV4PolicyMutationForbidden)},
		{name: "missing compatibility", edit: func(record *RuntimeExtensionPackRecord) { record.CompatibilityContractRef = " " }, want: "compatibility_contract_ref is required"},
		{name: "not hot reloadable", edit: func(record *RuntimeExtensionPackRecord) { record.HotReloadable = false }, want: "hot_reloadable must be true"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(now, "extension-pack-"+strings.ReplaceAll(tc.name, " ", "-"), tc.edit))
			if err == nil {
				t.Fatal("StoreRuntimeExtensionPackRecord() error = nil, want fail-closed validation")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreRuntimeExtensionPackRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func validRuntimeExtensionPackRecord(now time.Time, extensionPackID string, edit func(*RuntimeExtensionPackRecord)) RuntimeExtensionPackRecord {
	record := RuntimeExtensionPackRecord{
		ExtensionPackID: extensionPackID,
		Extensions:      []string{"local-router"},
		DeclaredTools: []RuntimeExtensionToolDeclaration{{
			ToolName:             "inspect_local_state",
			PermissionRefs:       []string{"local_read"},
			CompatibilitySummary: "compatible local read tool",
		}},
		DeclaredEvents:           []string{"wake_cycle"},
		DeclaredPermissions:      []string{"local_read"},
		CompatibilityContractRef: "compat-v1",
		HotReloadable:            true,
		ChangeSummary:            "extension fixture",
		CreatedAt:                now,
		CreatedBy:                "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}
