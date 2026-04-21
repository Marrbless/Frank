package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRuntimePackRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 13, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	second := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-b"
		record.Channel = "desktop_dev"
		record.SourceSummary = "second pack"
	})
	if err := StoreRuntimePackRecord(root, second); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-b) error = %v", err)
	}

	want := validRuntimePackRecord(now.Add(2*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = " pack-a "
		record.Channel = " phone "
		record.PromptPackRef = " prompt-pack-a "
		record.SkillPackRef = " skill-pack-a "
		record.ManifestRef = " manifest-a "
		record.ExtensionPackRef = " extension-pack-a "
		record.PolicyRef = " policy-a "
		record.SourceSummary = " seeded baseline "
		record.MutableSurfaces = []string{" prompts ", " skills "}
		record.ImmutableSurfaces = []string{" policy ", " treasury "}
		record.SurfaceClasses = []string{" class_1 ", " class_2 "}
		record.CompatibilityContractRef = " compat-v1 "
		record.RollbackTargetPackID = " pack-b "
	})
	if err := StoreRuntimePackRecord(root, want); err != nil {
		t.Fatalf("StoreRuntimePackRecord(pack-a) error = %v", err)
	}

	got, err := LoadRuntimePackRecord(root, "pack-a")
	if err != nil {
		t.Fatalf("LoadRuntimePackRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.PackID = "pack-a"
	want.Channel = "phone"
	want.PromptPackRef = "prompt-pack-a"
	want.SkillPackRef = "skill-pack-a"
	want.ManifestRef = "manifest-a"
	want.ExtensionPackRef = "extension-pack-a"
	want.PolicyRef = "policy-a"
	want.SourceSummary = "seeded baseline"
	want.MutableSurfaces = []string{"prompts", "skills"}
	want.ImmutableSurfaces = []string{"policy", "treasury"}
	want.SurfaceClasses = []string{"class_1", "class_2"}
	want.CompatibilityContractRef = "compat-v1"
	want.RollbackTargetPackID = "pack-b"
	want.CreatedAt = want.CreatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRuntimePackRecord() = %#v, want %#v", got, want)
	}

	records, err := ListRuntimePackRecords(root)
	if err != nil {
		t.Fatalf("ListRuntimePackRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListRuntimePackRecords() len = %d, want 2", len(records))
	}
	if records[0].PackID != "pack-a" || records[1].PackID != "pack-b" {
		t.Fatalf("ListRuntimePackRecords() ids = [%q %q], want [pack-a pack-b]", records[0].PackID, records[1].PackID)
	}
}

func TestRuntimePackRecordValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing pack id",
			run: func() error {
				return StoreRuntimePackRecord(root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
					record.PackID = "   "
				}))
			},
			want: "mission store runtime pack pack_id is required",
		},
		{
			name: "invalid pack id",
			run: func() error {
				return StoreRuntimePackRecord(root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
					record.PackID = "pack/one"
				}))
			},
			want: `mission store runtime pack pack_id "pack/one" is invalid`,
		},
		{
			name: "missing channel",
			run: func() error {
				return StoreRuntimePackRecord(root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
					record.Channel = " "
				}))
			},
			want: "mission store runtime pack channel is required",
		},
		{
			name: "missing mutable surfaces",
			run: func() error {
				return StoreRuntimePackRecord(root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
					record.MutableSurfaces = nil
				}))
			},
			want: "mission store runtime pack mutable_surfaces are required",
		},
		{
			name: "missing created at",
			run: func() error {
				return StoreRuntimePackRecord(root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
					record.CreatedAt = time.Time{}
				}))
			},
			want: "mission store runtime pack created_at is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreRuntimePackRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreRuntimePackRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestActiveRuntimePackPointerRoundTripAndResolve(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 15, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
		record.SourceSummary = "previous active"
	}))
	active := validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.RollbackTargetPackID = "pack-prev"
	})
	mustStoreRuntimePack(t, root, active)

	want := ActiveRuntimePackPointer{
		ActivePackID:         " pack-active ",
		PreviousActivePackID: " pack-prev ",
		LastKnownGoodPackID:  " pack-prev ",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            " owner ",
		UpdateRecordRef:      " manual-bootstrap ",
		ReloadGeneration:     3,
	}
	if err := StoreActiveRuntimePackPointer(root, want); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	got, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.ActivePackID = "pack-active"
	want.PreviousActivePackID = "pack-prev"
	want.LastKnownGoodPackID = "pack-prev"
	want.UpdatedAt = want.UpdatedAt.UTC()
	want.UpdatedBy = "owner"
	want.UpdateRecordRef = "manual-bootstrap"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadActiveRuntimePackPointer() = %#v, want %#v", got, want)
	}

	resolved, err := ResolveActiveRuntimePackRecord(root)
	if err != nil {
		t.Fatalf("ResolveActiveRuntimePackRecord() error = %v", err)
	}
	active.RecordVersion = StoreRecordVersion
	active = NormalizeRuntimePackRecord(active)
	if !reflect.DeepEqual(resolved, active) {
		t.Fatalf("ResolveActiveRuntimePackRecord() = %#v, want %#v", resolved, active)
	}
}

func TestLastKnownGoodRuntimePackPointerRoundTripAndResolve(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 16, 0, 0, 0, time.UTC)
	pack := validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-lkg"
		record.SourceSummary = "verified pack"
	})
	mustStoreRuntimePack(t, root, pack)

	want := LastKnownGoodRuntimePackPointer{
		PackID:            " pack-lkg ",
		Basis:             " smoke_check ",
		VerifiedAt:        now.Add(time.Minute),
		VerifiedBy:        " owner ",
		RollbackRecordRef: " bootstrap ",
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, want); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	got, err := LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.PackID = "pack-lkg"
	want.Basis = "smoke_check"
	want.VerifiedAt = want.VerifiedAt.UTC()
	want.VerifiedBy = "owner"
	want.RollbackRecordRef = "bootstrap"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() = %#v, want %#v", got, want)
	}

	resolved, err := ResolveLastKnownGoodRuntimePackRecord(root)
	if err != nil {
		t.Fatalf("ResolveLastKnownGoodRuntimePackRecord() error = %v", err)
	}
	pack.RecordVersion = StoreRecordVersion
	pack = NormalizeRuntimePackRecord(pack)
	if !reflect.DeepEqual(resolved, pack) {
		t.Fatalf("ResolveLastKnownGoodRuntimePackRecord() = %#v, want %#v", resolved, pack)
	}
}

func TestRuntimePackPointerReplayIsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 17, 0, 0, 0, time.UTC)
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-prev"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-active"
		record.RollbackTargetPackID = "pack-prev"
	}))

	pointer := ActiveRuntimePackPointer{
		ActivePackID:         "pack-active",
		PreviousActivePackID: "pack-prev",
		LastKnownGoodPackID:  "pack-prev",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "system",
		UpdateRecordRef:      "replay-safe",
		ReloadGeneration:     1,
	}
	if err := StoreActiveRuntimePackPointer(root, pointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreActiveRuntimePackPointer(root, pointer); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}

	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("active pointer file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}

func TestRuntimePackPointersRejectMissingRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 21, 18, 0, 0, 0, time.UTC)

	err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:    "missing-pack",
		UpdatedAt:       now,
		UpdatedBy:       "system",
		UpdateRecordRef: "bootstrap",
	})
	if err == nil {
		t.Fatal("StoreActiveRuntimePackPointer() error = nil, want missing pack rejection")
	}
	if !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %q, want missing pack rejection", err.Error())
	}

	err = StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "missing-pack",
		Basis:             "bootstrap",
		VerifiedAt:        now,
		VerifiedBy:        "system",
		RollbackRecordRef: "bootstrap",
	})
	if err == nil {
		t.Fatal("StoreLastKnownGoodRuntimePackPointer() error = nil, want missing pack rejection")
	}
	if !strings.Contains(err.Error(), ErrRuntimePackRecordNotFound.Error()) {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %q, want missing pack rejection", err.Error())
	}
}

func TestLoadRuntimePackPointersNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if _, err := LoadActiveRuntimePackPointer(root); !errors.Is(err, ErrActiveRuntimePackPointerNotFound) {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v, want %v", err, ErrActiveRuntimePackPointerNotFound)
	}
	if _, err := LoadLastKnownGoodRuntimePackPointer(root); !errors.Is(err, ErrLastKnownGoodRuntimePackPointerNotFound) {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() error = %v, want %v", err, ErrLastKnownGoodRuntimePackPointerNotFound)
	}
	if _, err := LoadRuntimePackRecord(root, "missing-pack"); !errors.Is(err, ErrRuntimePackRecordNotFound) {
		t.Fatalf("LoadRuntimePackRecord() error = %v, want %v", err, ErrRuntimePackRecordNotFound)
	}
}

func validRuntimePackRecord(now time.Time, mutate func(*RuntimePackRecord)) RuntimePackRecord {
	record := RuntimePackRecord{
		PackID:                   "pack-root",
		ParentPackID:             "",
		CreatedAt:                now,
		Channel:                  "phone",
		PromptPackRef:            "prompt-pack-root",
		SkillPackRef:             "skill-pack-root",
		ManifestRef:              "manifest-root",
		ExtensionPackRef:         "extension-pack-root",
		PolicyRef:                "policy-root",
		SourceSummary:            "seeded active pack",
		MutableSurfaces:          []string{"prompts", "skills"},
		ImmutableSurfaces:        []string{"policy", "authority"},
		SurfaceClasses:           []string{"class_1"},
		CompatibilityContractRef: "compat-v1",
		RollbackTargetPackID:     "",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func mustStoreRuntimePack(t *testing.T, root string, record RuntimePackRecord) {
	t.Helper()

	if err := StoreRuntimePackRecord(root, record); err != nil {
		t.Fatalf("StoreRuntimePackRecord(%s) error = %v", record.PackID, err)
	}
}
