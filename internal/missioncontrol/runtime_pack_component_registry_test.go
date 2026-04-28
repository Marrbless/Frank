package missioncontrol

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRuntimePackComponentRecordRoundTripListReplayAndDivergentDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	_, changed, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-b", nil))
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(prompt-pack-b) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreRuntimePackComponentRecord(prompt-pack-b) changed = false, want true")
	}

	want := validRuntimePackComponentRecord(now.Add(time.Minute), RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
		record.Kind = " prompt_pack "
		record.ComponentID = " prompt-pack-a "
		record.ParentComponentID = " prompt-pack-root "
		record.ContentRef = " local-fixture://prompt-pack-a "
		record.ContentSHA256 = strings.ToUpper(record.ContentSHA256)
		record.SourceSummary = " prompt metadata "
		record.ProvenanceRef = " import:fixture-a "
		record.CreatedBy = " operator "
	})
	got, changed, err := StoreRuntimePackComponentRecord(root, want)
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(prompt-pack-a) error = %v", err)
	}
	if !changed {
		t.Fatal("StoreRuntimePackComponentRecord(prompt-pack-a) changed = false, want true")
	}

	want.RecordVersion = StoreRecordVersion
	want = NormalizeRuntimePackComponentRecord(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StoreRuntimePackComponentRecord() = %#v, want %#v", got, want)
	}

	loaded, err := LoadRuntimePackComponentRecord(root, RuntimePackComponentKindPromptPack, "prompt-pack-a")
	if err != nil {
		t.Fatalf("LoadRuntimePackComponentRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadRuntimePackComponentRecord() = %#v, want %#v", loaded, want)
	}

	records, err := ListRuntimePackComponentRecords(root, RuntimePackComponentKindPromptPack)
	if err != nil {
		t.Fatalf("ListRuntimePackComponentRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListRuntimePackComponentRecords() len = %d, want 2", len(records))
	}
	if records[0].ComponentID != "prompt-pack-a" || records[1].ComponentID != "prompt-pack-b" {
		t.Fatalf("ListRuntimePackComponentRecords() ids = [%q %q], want [prompt-pack-a prompt-pack-b]", records[0].ComponentID, records[1].ComponentID)
	}

	replayed, changed, err := StoreRuntimePackComponentRecord(root, want)
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StoreRuntimePackComponentRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StoreRuntimePackComponentRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.SourceSummary = "different prompt metadata"
	if _, _, err := StoreRuntimePackComponentRecord(root, divergent); err == nil {
		t.Fatal("StoreRuntimePackComponentRecord(divergent) error = nil, want duplicate rejection")
	} else if !strings.Contains(err.Error(), `mission store runtime pack component "prompt-pack-a" kind "prompt_pack" already exists`) {
		t.Fatalf("StoreRuntimePackComponentRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestRuntimePackComponentRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing kind",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.Kind = " "
				}))
				return err
			},
			want: "mission store runtime pack component kind is required",
		},
		{
			name: "invalid kind",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.Kind = "policy_pack"
				}))
				return err
			},
			want: `mission store runtime pack component kind "policy_pack" is invalid`,
		},
		{
			name: "missing component id",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.ComponentID = " "
				}))
				return err
			},
			want: "mission store runtime pack component component_id is required",
		},
		{
			name: "missing content ref",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.ContentRef = " "
				}))
				return err
			},
			want: "mission store runtime pack component content_ref is required",
		},
		{
			name: "invalid content sha",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.ContentSHA256 = "not-a-sha"
				}))
				return err
			},
			want: `mission store runtime pack component content_sha256 "not-a-sha" is invalid`,
		},
		{
			name: "missing provenance",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.ProvenanceRef = " "
				}))
				return err
			},
			want: "mission store runtime pack component provenance_ref is required",
		},
		{
			name: "missing created by",
			run: func() error {
				_, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, RuntimePackComponentKindPromptPack, "prompt-pack-a", func(record *RuntimePackComponentRecord) {
					record.CreatedBy = " "
				}))
				return err
			},
			want: "mission store runtime pack component created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreRuntimePackComponentRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreRuntimePackComponentRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestResolveRuntimePackComponentsRequiresAllPackRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	pack := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-with-components"
		record.PromptPackRef = "prompt-pack-a"
		record.SkillPackRef = "skill-pack-a"
		record.ManifestRef = "manifest-pack-a"
		record.ExtensionPackRef = "extension-pack-a"
	})
	pack = NormalizeRuntimePackRecord(pack)
	pack.RecordVersion = StoreRecordVersion
	storeRuntimePackComponentFixture(t, root, now, RuntimePackComponentKindPromptPack, "prompt-pack-a")
	storeRuntimePackComponentFixture(t, root, now, RuntimePackComponentKindSkillPack, "skill-pack-a")
	storeRuntimePackComponentFixture(t, root, now, RuntimePackComponentKindManifestPack, "manifest-pack-a")
	storeRuntimePackComponentFixture(t, root, now, RuntimePackComponentKindExtensionPack, "extension-pack-a")

	components, err := ResolveRuntimePackComponents(root, pack)
	if err != nil {
		t.Fatalf("ResolveRuntimePackComponents() error = %v", err)
	}
	if components.PromptPack.ComponentID != "prompt-pack-a" || components.PromptPack.Kind != RuntimePackComponentKindPromptPack {
		t.Fatalf("ResolveRuntimePackComponents().PromptPack = %#v, want prompt-pack-a prompt_pack", components.PromptPack)
	}
	if components.SkillPack.ComponentID != "skill-pack-a" || components.SkillPack.Kind != RuntimePackComponentKindSkillPack {
		t.Fatalf("ResolveRuntimePackComponents().SkillPack = %#v, want skill-pack-a skill_pack", components.SkillPack)
	}
	if components.ManifestPack.ComponentID != "manifest-pack-a" || components.ManifestPack.Kind != RuntimePackComponentKindManifestPack {
		t.Fatalf("ResolveRuntimePackComponents().ManifestPack = %#v, want manifest-pack-a manifest_pack", components.ManifestPack)
	}
	if components.ExtensionPack.ComponentID != "extension-pack-a" || components.ExtensionPack.Kind != RuntimePackComponentKindExtensionPack {
		t.Fatalf("ResolveRuntimePackComponents().ExtensionPack = %#v, want extension-pack-a extension_pack", components.ExtensionPack)
	}

	missingRoot := t.TempDir()
	storeRuntimePackComponentFixture(t, missingRoot, now, RuntimePackComponentKindPromptPack, "prompt-pack-a")
	storeRuntimePackComponentFixture(t, missingRoot, now, RuntimePackComponentKindSkillPack, "skill-pack-a")
	storeRuntimePackComponentFixture(t, missingRoot, now, RuntimePackComponentKindManifestPack, "manifest-pack-a")
	if _, err := ResolveRuntimePackComponents(missingRoot, pack); err == nil {
		t.Fatal("ResolveRuntimePackComponents(missing extension) error = nil, want missing component rejection")
	} else if !errors.Is(err, ErrRuntimePackComponentRecordNotFound) {
		t.Fatalf("ResolveRuntimePackComponents(missing extension) error = %v, want %v", err, ErrRuntimePackComponentRecordNotFound)
	} else if !strings.Contains(err.Error(), `mission store runtime pack extension_pack_ref "extension-pack-a"`) {
		t.Fatalf("ResolveRuntimePackComponents(missing extension) error = %q, want extension ref context", err.Error())
	}
}

func TestResolveActiveRuntimePackComponentsUsesCommittedActivePointerOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 13, 0, 0, 0, time.UTC)
	active := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.PromptPackRef = "prompt-pack-active"
		record.SkillPackRef = "skill-pack-active"
		record.ManifestRef = "manifest-pack-active"
		record.ExtensionPackRef = "extension-pack-active"
	})
	candidate := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.PromptPackRef = "prompt-pack-candidate"
		record.SkillPackRef = "skill-pack-candidate"
		record.ManifestRef = "manifest-pack-candidate"
		record.ExtensionPackRef = "extension-pack-candidate"
		record.RollbackTargetPackID = "pack-active"
	})
	mustStoreRuntimePack(t, root, active)
	mustStoreRuntimePack(t, root, candidate)
	mustStoreRuntimePackComponentFixtureAllowReplay(t, root, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RuntimePackComponentKindPromptPack, "prompt-pack-active")
	mustStoreRuntimePackComponentFixtureAllowReplay(t, root, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RuntimePackComponentKindSkillPack, "skill-pack-active")
	mustStoreRuntimePackComponentFixtureAllowReplay(t, root, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RuntimePackComponentKindManifestPack, "manifest-pack-active")
	mustStoreRuntimePackComponentFixtureAllowReplay(t, root, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RuntimePackComponentKindExtensionPack, "extension-pack-active")
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-active",
		PreviousActivePackID: "pack-candidate",
		LastKnownGoodPackID:  "pack-active",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "hot_update:test",
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	gotPack, components, err := ResolveActiveRuntimePackComponents(root)
	if err != nil {
		t.Fatalf("ResolveActiveRuntimePackComponents() error = %v", err)
	}
	if gotPack.PackID != "pack-active" {
		t.Fatalf("ResolveActiveRuntimePackComponents() pack_id = %q, want pack-active", gotPack.PackID)
	}
	if components.PromptPack.ComponentID != "prompt-pack-active" {
		t.Fatalf("ResolveActiveRuntimePackComponents() prompt component = %q, want prompt-pack-active", components.PromptPack.ComponentID)
	}
}

func TestLoadCommittedActiveRuntimePackForRestartRequiresCommittedActiveComponents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	active := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.PromptPackRef = "prompt-pack-active"
		record.SkillPackRef = "skill-pack-active"
		record.ManifestRef = "manifest-pack-active"
		record.ExtensionPackRef = "extension-pack-active"
	})
	candidate := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.PromptPackRef = "prompt-pack-candidate"
		record.SkillPackRef = "skill-pack-candidate"
		record.ManifestRef = "manifest-pack-candidate"
		record.ExtensionPackRef = "extension-pack-candidate"
		record.RollbackTargetPackID = "pack-active"
	})
	if err := StoreRuntimePackRecord(root, active); err != nil {
		t.Fatalf("StoreRuntimePackRecord(active) error = %v", err)
	}
	if err := StoreRuntimePackRecord(root, candidate); err != nil {
		t.Fatalf("StoreRuntimePackRecord(candidate) error = %v", err)
	}
	mustStoreRuntimePackComponentRefs(t, root, candidate)
	storeRuntimePackComponentFixture(t, root, now.Add(2*time.Minute), RuntimePackComponentKindPromptPack, "prompt-pack-active")
	storeRuntimePackComponentFixture(t, root, now.Add(2*time.Minute), RuntimePackComponentKindSkillPack, "skill-pack-active")
	storeRuntimePackComponentFixture(t, root, now.Add(2*time.Minute), RuntimePackComponentKindManifestPack, "manifest-pack-active")
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-active",
		PreviousActivePackID: "pack-candidate",
		LastKnownGoodPackID:  "pack-active",
		UpdatedAt:            now.Add(3 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "restart-read:fixture",
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	if _, err := LoadCommittedActiveRuntimePackForRestart(root); err == nil {
		t.Fatal("LoadCommittedActiveRuntimePackForRestart() error = nil, want missing committed active component rejection")
	} else if !errors.Is(err, ErrRuntimePackComponentRecordNotFound) {
		t.Fatalf("LoadCommittedActiveRuntimePackForRestart() error = %v, want %v", err, ErrRuntimePackComponentRecordNotFound)
	} else if !strings.Contains(err.Error(), `mission store runtime pack extension_pack_ref "extension-pack-active"`) {
		t.Fatalf("LoadCommittedActiveRuntimePackForRestart() error = %q, want active extension ref context", err.Error())
	}
}

func TestStoreRuntimePackComponentRecordRejectsAdhocActivePackRef(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 7, 15, 0, 0, 0, time.UTC)
	active := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.PromptPackRef = "prompt-pack-active"
	})
	if err := StoreRuntimePackRecord(root, active); err != nil {
		t.Fatalf("StoreRuntimePackRecord(active) error = %v", err)
	}
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:        "pack-active",
		LastKnownGoodPackID: "pack-active",
		UpdatedAt:           now.Add(time.Minute),
		UpdatedBy:           "operator",
		UpdateRecordRef:     "hot_update:test",
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	activeComponent := validRuntimePackComponentRecord(now.Add(2*time.Minute), RuntimePackComponentKindPromptPack, "prompt-pack-active", nil)
	if _, changed, err := StoreRuntimePackComponentRecord(root, activeComponent); err == nil {
		t.Fatal("StoreRuntimePackComponentRecord(active component) error = nil, want ad hoc mutation rejection")
	} else if changed {
		t.Fatal("StoreRuntimePackComponentRecord(active component) changed = true, want false")
	} else if !strings.Contains(err.Error(), string(RejectionCodeV4ActivePackAdhocMutationForbidden)) {
		t.Fatalf("StoreRuntimePackComponentRecord(active component) error = %q, want %s", err.Error(), RejectionCodeV4ActivePackAdhocMutationForbidden)
	}
	if _, err := LoadRuntimePackComponentRecord(root, RuntimePackComponentKindPromptPack, "prompt-pack-active"); !errors.Is(err, ErrRuntimePackComponentRecordNotFound) {
		t.Fatalf("LoadRuntimePackComponentRecord(active component) error = %v, want not found after rejection", err)
	}

	inactiveComponent := validRuntimePackComponentRecord(now.Add(3*time.Minute), RuntimePackComponentKindPromptPack, "prompt-pack-candidate", nil)
	stored, changed, err := StoreRuntimePackComponentRecord(root, inactiveComponent)
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(inactive component) error = %v", err)
	}
	if !changed || stored.ComponentID != "prompt-pack-candidate" {
		t.Fatalf("StoreRuntimePackComponentRecord(inactive component) = %#v changed=%v, want stored candidate component", stored, changed)
	}
}

func validRuntimePackComponentRecord(now time.Time, kind RuntimePackComponentKind, componentID string, mutate func(*RuntimePackComponentRecord)) RuntimePackComponentRecord {
	record := RuntimePackComponentRecord{
		Kind:          kind,
		ComponentID:   componentID,
		ContentRef:    "local-fixture://" + componentID,
		ContentSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		SourceSummary: string(kind) + " metadata fixture",
		ProvenanceRef: "fixture:" + componentID,
		CreatedAt:     now,
		CreatedBy:     "operator",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeRuntimePackComponentFixture(t *testing.T, root string, now time.Time, kind RuntimePackComponentKind, componentID string) RuntimePackComponentRecord {
	t.Helper()

	record, changed, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, kind, componentID, nil))
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(%s/%s) error = %v", kind, componentID, err)
	}
	if !changed {
		t.Fatalf("StoreRuntimePackComponentRecord(%s/%s) changed = false, want true", kind, componentID)
	}
	return record
}

func mustStoreRuntimePackComponentRefs(t *testing.T, root string, record RuntimePackRecord) {
	t.Helper()

	componentCreatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, ref := range RuntimePackComponentRefs(record) {
		mustStoreRuntimePackComponentFixtureAllowReplay(t, root, componentCreatedAt, ref.Kind, ref.ComponentID)
	}
}

func mustStoreRuntimePackComponentFixtureAllowReplay(t *testing.T, root string, now time.Time, kind RuntimePackComponentKind, componentID string) RuntimePackComponentRecord {
	t.Helper()

	record, _, err := StoreRuntimePackComponentRecord(root, validRuntimePackComponentRecord(now, kind, componentID, nil))
	if err != nil {
		t.Fatalf("StoreRuntimePackComponentRecord(%s/%s) error = %v", kind, componentID, err)
	}
	return record
}
