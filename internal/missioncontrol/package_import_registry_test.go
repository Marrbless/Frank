package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPackageImportRecordRoundTripReplayAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 12, 9, 0, 0, 0, time.UTC)
	storePackageImportCandidateFixture(t, root, now)

	want := validPackageImportRecord(now.Add(3*time.Minute), func(record *PackageImportRecord) {
		record.ImportID = " donor-import-1 "
		record.ContentSHA256 = strings.ToUpper(record.ContentSHA256)
		record.ContentKinds = []RuntimePackComponentKind{" skill_pack ", RuntimePackComponentKindPromptPack, RuntimePackComponentKindSkillPack}
		record.DeclaredSurfaces = []string{" skills ", " prompts "}
		record.CreatedBy = " operator "
	})
	got, changed, err := StorePackageImportRecord(root, want)
	if err != nil {
		t.Fatalf("StorePackageImportRecord(first) error = %v", err)
	}
	if !changed {
		t.Fatal("StorePackageImportRecord(first) changed = false, want true")
	}
	want.RecordVersion = StoreRecordVersion
	want.ImportID = "donor-import-1"
	want.ContentSHA256 = strings.ToLower(want.ContentSHA256)
	want.ContentKinds = []RuntimePackComponentKind{RuntimePackComponentKindSkillPack, RuntimePackComponentKindPromptPack}
	want.DeclaredSurfaces = []string{"skills", "prompts"}
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StorePackageImportRecord(first) = %#v, want %#v", got, want)
	}

	loaded, err := LoadPackageImportRecord(root, " donor-import-1 ")
	if err != nil {
		t.Fatalf("LoadPackageImportRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("LoadPackageImportRecord() = %#v, want %#v", loaded, want)
	}
	replayed, changed, err := StorePackageImportRecord(root, want)
	if err != nil {
		t.Fatalf("StorePackageImportRecord(replay) error = %v", err)
	}
	if changed {
		t.Fatal("StorePackageImportRecord(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("StorePackageImportRecord(replay) = %#v, want %#v", replayed, want)
	}

	divergent := want
	divergent.SourceSummary = "different donor content"
	if _, _, err := StorePackageImportRecord(root, divergent); err == nil {
		t.Fatal("StorePackageImportRecord(divergent) error = nil, want duplicate rejection")
	}

	records, err := ListPackageImportRecords(root)
	if err != nil {
		t.Fatalf("ListPackageImportRecords() error = %v", err)
	}
	if len(records) != 1 || !reflect.DeepEqual(records[0], want) {
		t.Fatalf("ListPackageImportRecords() = %#v, want one stored import", records)
	}
}

func TestPackageImportRecordRejectsInvalidShapeAndAuthorityGrants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		edit func(*PackageImportRecord)
		want string
	}{
		{
			name: "missing source package ref",
			edit: func(record *PackageImportRecord) {
				record.SourcePackageRef = " "
			},
			want: "source_package_ref is required",
		},
		{
			name: "missing content kind",
			edit: func(record *PackageImportRecord) {
				record.ContentKinds = nil
			},
			want: "content_kinds are required",
		},
		{
			name: "non candidate activation state",
			edit: func(record *PackageImportRecord) {
				record.ActivationState = PackageImportActivationState("active")
			},
			want: `activation_state must be "candidate_only"`,
		},
		{
			name: "authority grant",
			edit: func(record *PackageImportRecord) {
				record.DeclaredAuthorityGrants = []PackageAuthorityGrantDeclaration{
					{AuthorityKind: "provider", TargetRef: "provider/mail", Reason: "package declares provider authority"},
					{AuthorityKind: "spending", TargetRef: "treasury/default"},
				}
			},
			want: string(RejectionCodeV4PackageAuthorityGrantForbidden),
		},
		{
			name: "policy surface declaration",
			edit: func(record *PackageImportRecord) {
				record.DeclaredSurfaces = []string{"skills", "treasury_policy"}
			},
			want: string(RejectionCodeV4PolicyMutationForbidden),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
			storePackageImportCandidateFixture(t, root, now)
			record := validPackageImportRecord(now.Add(3*time.Minute), tt.edit)
			_, changed, err := StorePackageImportRecord(root, record)
			if err == nil {
				t.Fatal("StorePackageImportRecord() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("StorePackageImportRecord() changed = true, want false")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("StorePackageImportRecord() error = %q, want substring %q", err.Error(), tt.want)
			}
			if strings.Contains(tt.want, string(RejectionCodeV4PackageAuthorityGrantForbidden)) {
				for _, want := range []string{"provider", "spending"} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("StorePackageImportRecord() error = %q, want authority kind %q", err.Error(), want)
					}
				}
			}
			if strings.Contains(tt.want, string(RejectionCodeV4PolicyMutationForbidden)) && !strings.Contains(err.Error(), "treasury_policy") {
				t.Fatalf("StorePackageImportRecord() error = %q, want frozen policy declaration", err.Error())
			}
			if _, err := LoadPackageImportRecord(root, "donor-import-1"); !errors.Is(err, ErrPackageImportRecordNotFound) {
				t.Fatalf("LoadPackageImportRecord() error = %v, want not found after rejected import", err)
			}
		})
	}
}

func TestPackageImportRecordRequiresCandidateOnlyLinkage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 12, 11, 0, 0, 0, time.UTC)
	storePackageImportCandidateFixture(t, root, now)

	missingCandidate := validPackageImportRecord(now.Add(3*time.Minute), func(record *PackageImportRecord) {
		record.ImportID = "missing-candidate-import"
		record.CandidateID = "candidate-missing"
	})
	if _, _, err := StorePackageImportRecord(root, missingCandidate); err == nil || !strings.Contains(err.Error(), ErrImprovementCandidateRecordNotFound.Error()) {
		t.Fatalf("StorePackageImportRecord(missing candidate) error = %v, want missing candidate", err)
	}

	mismatchedPack := validPackageImportRecord(now.Add(4*time.Minute), func(record *PackageImportRecord) {
		record.ImportID = "mismatched-pack-import"
		record.CandidatePackID = "pack-base"
	})
	if _, _, err := StorePackageImportRecord(root, mismatchedPack); err == nil || !strings.Contains(err.Error(), "does not match candidate") {
		t.Fatalf("StorePackageImportRecord(mismatched pack) error = %v, want candidate pack mismatch", err)
	}
}

func TestPackageImportRecordRejectsActivePackCandidateImport(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	storePackageImportCandidateFixture(t, root, now)
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(3 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      "hot_update:test",
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}

	record := validPackageImportRecord(now.Add(4*time.Minute), nil)
	if _, changed, err := StorePackageImportRecord(root, record); err == nil {
		t.Fatal("StorePackageImportRecord(active candidate pack) error = nil, want ad hoc active-pack rejection")
	} else if changed {
		t.Fatal("StorePackageImportRecord(active candidate pack) changed = true, want false")
	} else if !strings.Contains(err.Error(), string(RejectionCodeV4ActivePackAdhocMutationForbidden)) {
		t.Fatalf("StorePackageImportRecord(active candidate pack) error = %q, want %s", err.Error(), RejectionCodeV4ActivePackAdhocMutationForbidden)
	}
	if _, err := LoadPackageImportRecord(root, record.ImportID); !errors.Is(err, ErrPackageImportRecordNotFound) {
		t.Fatalf("LoadPackageImportRecord() error = %v, want not found after rejected active import", err)
	}
}

func TestLoadPackageImportRecordRejectsMalformedStoredRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := StorePackageImportPath(root, "malformed-import")
	if err := os.MkdirAll(StorePackageImportsDir(root), 0o755); err != nil {
		t.Fatalf("MkdirAll(package imports) error = %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(malformed import) error = %v", err)
	}
	if _, err := LoadPackageImportRecord(root, "malformed-import"); err == nil {
		t.Fatal("LoadPackageImportRecord() error = nil, want malformed JSON rejection")
	}
}

func storePackageImportCandidateFixture(t *testing.T, root string, now time.Time) {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.MutableSurfaces = []string{"prompts", "skills"}
	}))
	if err := StoreImprovementCandidateRecord(root, ImprovementCandidateRecord{
		CandidateID:         "candidate-1",
		BaselinePackID:      "pack-base",
		CandidatePackID:     "pack-candidate",
		SourceWorkspaceRef:  "workspace/runs/package-import",
		SourceSummary:       "candidate package import fixture",
		ValidationBasisRefs: []string{"eval-suite-1"},
		CreatedAt:           now.Add(2 * time.Minute),
		CreatedBy:           "operator",
	}); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}
}

func validPackageImportRecord(now time.Time, edit func(*PackageImportRecord)) PackageImportRecord {
	record := PackageImportRecord{
		ImportID:         "donor-import-1",
		CandidateID:      "candidate-1",
		CandidatePackID:  "pack-candidate",
		SourcePackageRef: "pi-donor://skills/local-routing@v1",
		SourceProject:    "pi-donor",
		SourceVersion:    "v1",
		ContentRef:       "local-fixture://imports/pi-donor/local-routing-v1",
		ContentSHA256:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ContentKinds: []RuntimePackComponentKind{
			RuntimePackComponentKindSkillPack,
			RuntimePackComponentKindPromptPack,
		},
		SurfaceClass:     "class_1",
		DeclaredSurfaces: []string{"skills", "prompts"},
		ActivationState:  PackageImportActivationStateCandidateOnly,
		ProvenanceRef:    "fixture:pi-donor/local-routing-v1",
		SourceSummary:    "fixture donor package imported as candidate content",
		CreatedAt:        now,
		CreatedBy:        "operator",
	}
	if edit != nil {
		edit(&record)
	}
	return record
}
