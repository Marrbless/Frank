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

func TestRecertifyLastKnownGoodFromPromotionRecertifiesPromotedPack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	promotion, outcome, gate := storeHotUpdatePromotionForLastKnownGoodRecertFixture(t, root, now)
	before := snapshotLastKnownGoodRecertificationSideEffects(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID)
	verifiedAt := now.Add(30 * time.Minute)

	got, changed, err := RecertifyLastKnownGoodFromPromotion(root, promotion.PromotionID, " operator ", verifiedAt)
	if err != nil {
		t.Fatalf("RecertifyLastKnownGoodFromPromotion() error = %v", err)
	}
	if !changed {
		t.Fatal("RecertifyLastKnownGoodFromPromotion() changed = false, want true")
	}

	want := LastKnownGoodRuntimePackPointer{
		RecordVersion:     StoreRecordVersion,
		PackID:            "pack-candidate",
		Basis:             "hot_update_promotion:" + promotion.PromotionID,
		VerifiedAt:        verifiedAt,
		VerifiedBy:        "operator",
		RollbackRecordRef: "hot_update_promotion:" + promotion.PromotionID,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RecertifyLastKnownGoodFromPromotion() = %#v, want %#v", got, want)
	}
	stored, err := LoadLastKnownGoodRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() error = %v", err)
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("LoadLastKnownGoodRuntimePackPointer() = %#v, want %#v", stored, want)
	}
	afterBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer after) error = %v", err)
	}
	if string(afterBytes) == string(before.lastKnownGoodBytes) {
		t.Fatal("last-known-good pointer bytes unchanged after first recertification")
	}
	assertLastKnownGoodRecertificationSideEffectsUnchangedExceptLastKnownGood(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID, before)
}

func TestRecertifyLastKnownGoodFromPromotionRejectsInvalidAuthorityChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, root string, now time.Time) (promotionID string, want string)
	}{
		{
			name: "missing promotion",
			setup: func(t *testing.T, root string, now time.Time) (string, string) {
				return "missing-promotion", ErrPromotionRecordNotFound.Error()
			},
		},
		{
			name: "promotion without outcome id",
			setup: func(t *testing.T, root string, now time.Time) (string, string) {
				outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
				record := PromotionRecord{
					PromotionID:          "promotion-no-outcome",
					PromotedPackID:       "pack-candidate",
					PreviousActivePackID: "pack-base",
					HotUpdateID:          "hot-update-1",
					Reason:               "legacy promotion",
					PromotedAt:           outcome.OutcomeAt,
					CreatedAt:            now.Add(12 * time.Minute),
					CreatedBy:            "operator",
				}
				if err := StorePromotionRecord(root, record); err != nil {
					t.Fatalf("StorePromotionRecord() error = %v", err)
				}
				return record.PromotionID, `promotion "promotion-no-outcome" outcome_id is required`
			},
		},
		{
			name: "linked outcome missing",
			setup: func(t *testing.T, root string, now time.Time) (string, string) {
				storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
				record := validPromotionRecord(now.Add(12*time.Minute), func(record *PromotionRecord) {
					record.PromotionID = "promotion-missing-outcome"
					record.HotUpdateID = "hot-update-1"
					record.OutcomeID = "missing-outcome"
					record.CandidateID = ""
					record.RunID = ""
					record.CandidateResultID = ""
					record.LastKnownGoodPackID = ""
					record.LastKnownGoodBasis = ""
				})
				writeRawPromotionRecord(t, root, record)
				return record.PromotionID, ErrHotUpdateOutcomeRecordNotFound.Error()
			},
		},
		{
			name: "linked outcome not hot updated",
			setup: func(t *testing.T, root string, now time.Time) (string, string) {
				outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeKind = HotUpdateOutcomeKindFailed
					record.Reason = "reload failed"
				}, nil)
				record := PromotionRecord{
					PromotionID:          "promotion-failed-outcome",
					PromotedPackID:       "pack-candidate",
					PreviousActivePackID: "pack-base",
					HotUpdateID:          "hot-update-1",
					OutcomeID:            outcome.OutcomeID,
					Reason:               "legacy promotion",
					PromotedAt:           outcome.OutcomeAt,
					CreatedAt:            now.Add(12 * time.Minute),
					CreatedBy:            "operator",
				}
				if err := StorePromotionRecord(root, record); err != nil {
					t.Fatalf("StorePromotionRecord() error = %v", err)
				}
				return record.PromotionID, `outcome_kind "failed" does not permit last-known-good recertification`
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC)
			promotionID, want := tc.setup(t, root, now)

			_, changed, err := RecertifyLastKnownGoodFromPromotion(root, promotionID, "operator", now.Add(30*time.Minute))
			if err == nil {
				t.Fatal("RecertifyLastKnownGoodFromPromotion() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("RecertifyLastKnownGoodFromPromotion() changed = true, want false")
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("RecertifyLastKnownGoodFromPromotion() error = %q, want substring %q", err.Error(), want)
			}
		})
	}
}

func TestRecertifyLastKnownGoodFromPromotionRejectsPointerGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, root string, now time.Time, promotion PromotionRecord)
		want  string
	}{
		{
			name: "active pointer missing",
			setup: func(t *testing.T, root string, now time.Time, promotion PromotionRecord) {
				if err := os.Remove(StoreActiveRuntimePackPointerPath(root)); err != nil {
					t.Fatalf("Remove(active pointer) error = %v", err)
				}
			},
			want: ErrActiveRuntimePackPointerNotFound.Error(),
		},
		{
			name: "active pointer mismatch",
			setup: func(t *testing.T, root string, now time.Time, promotion PromotionRecord) {
				if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
					ActivePackID:         promotion.PreviousActivePackID,
					PreviousActivePackID: promotion.PreviousActivePackID,
					LastKnownGoodPackID:  promotion.PreviousActivePackID,
					UpdatedAt:            now.Add(20 * time.Minute),
					UpdatedBy:            "operator",
					UpdateRecordRef:      "manual-active-mismatch",
					ReloadGeneration:     7,
				}); err != nil {
					t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
				}
			},
			want: `active_pack_id "pack-base" does not match promotion promoted_pack_id "pack-candidate"`,
		},
		{
			name: "current lkg missing",
			setup: func(t *testing.T, root string, now time.Time, promotion PromotionRecord) {
				if err := os.Remove(StoreLastKnownGoodRuntimePackPointerPath(root)); err != nil {
					t.Fatalf("Remove(last-known-good pointer) error = %v", err)
				}
			},
			want: ErrLastKnownGoodRuntimePackPointerNotFound.Error(),
		},
		{
			name: "current lkg not previous or promoted pack",
			setup: func(t *testing.T, root string, now time.Time, promotion PromotionRecord) {
				mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(21*time.Minute), func(record *RuntimePackRecord) {
					record.PackID = "pack-other"
					record.ParentPackID = "pack-base"
					record.RollbackTargetPackID = "pack-base"
				}))
				if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
					PackID:            "pack-other",
					Basis:             "external_basis",
					VerifiedAt:        now.Add(22 * time.Minute),
					VerifiedBy:        "operator",
					RollbackRecordRef: "external",
				}); err != nil {
					t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
				}
			},
			want: `pack_id "pack-other" does not match promotion previous_active_pack_id "pack-base"`,
		},
		{
			name: "current lkg promoted pack with divergent verification",
			setup: func(t *testing.T, root string, now time.Time, promotion PromotionRecord) {
				if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
					PackID:            promotion.PromotedPackID,
					Basis:             "manual_hot_update_promotion",
					VerifiedAt:        now.Add(22 * time.Minute),
					VerifiedBy:        "operator",
					RollbackRecordRef: "manual",
				}); err != nil {
					t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
				}
			},
			want: `already points to promoted pack but differs from deterministic recertification`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
			promotion, outcome, gate := storeHotUpdatePromotionForLastKnownGoodRecertFixture(t, root, now)
			tc.setup(t, root, now, promotion)
			before := snapshotLastKnownGoodRecertificationSideEffectsAllowMissingPointers(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID)

			_, changed, err := RecertifyLastKnownGoodFromPromotion(root, promotion.PromotionID, "operator", now.Add(30*time.Minute))
			if err == nil {
				t.Fatal("RecertifyLastKnownGoodFromPromotion() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("RecertifyLastKnownGoodFromPromotion() changed = true, want false")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("RecertifyLastKnownGoodFromPromotion() error = %q, want substring %q", err.Error(), tc.want)
			}
			assertLastKnownGoodRecertificationSideEffectsFullyUnchangedAllowMissingPointers(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID, before)
		})
	}
}

func TestRecertifyLastKnownGoodFromPromotionReplayIsIdempotentAndByteStable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 5, 13, 0, 0, 0, time.UTC)
	promotion, outcome, gate := storeHotUpdatePromotionForLastKnownGoodRecertFixture(t, root, now)
	verifiedAt := now.Add(30 * time.Minute)
	if _, changed, err := RecertifyLastKnownGoodFromPromotion(root, promotion.PromotionID, "operator", verifiedAt); err != nil {
		t.Fatalf("RecertifyLastKnownGoodFromPromotion(first) error = %v", err)
	} else if !changed {
		t.Fatal("RecertifyLastKnownGoodFromPromotion(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(first last-known-good pointer) error = %v", err)
	}
	beforeReplay := snapshotLastKnownGoodRecertificationSideEffects(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID)

	got, changed, err := RecertifyLastKnownGoodFromPromotion(root, promotion.PromotionID, " operator ", verifiedAt)
	if err != nil {
		t.Fatalf("RecertifyLastKnownGoodFromPromotion(replay) error = %v", err)
	}
	if changed {
		t.Fatal("RecertifyLastKnownGoodFromPromotion(replay) changed = true, want false")
	}
	secondBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(second last-known-good pointer) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatalf("last-known-good pointer changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
	if got.PackID != promotion.PromotedPackID {
		t.Fatalf("RecertifyLastKnownGoodFromPromotion(replay).PackID = %q, want %q", got.PackID, promotion.PromotedPackID)
	}
	assertLastKnownGoodRecertificationSideEffectsFullyUnchanged(t, root, promotion.PromotionID, outcome.OutcomeID, gate.HotUpdateID, beforeReplay)
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
	mustStoreRuntimePackComponentRefs(t, root, record)
	mustStoreRuntimeExtensionPackRef(t, root, record.ExtensionPackRef)
}

func mustStoreRuntimeExtensionPackRef(t *testing.T, root string, extensionPackID string) {
	t.Helper()

	if _, err := LoadRuntimeExtensionPackRecord(root, extensionPackID); err == nil {
		return
	} else if !errors.Is(err, ErrRuntimeExtensionPackRecordNotFound) {
		t.Fatalf("LoadRuntimeExtensionPackRecord(%s) error = %v", extensionPackID, err)
	}
	_, _, err := StoreRuntimeExtensionPackRecord(root, validRuntimeExtensionPackRecord(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), extensionPackID, nil))
	if err != nil {
		t.Fatalf("StoreRuntimeExtensionPackRecord(%s) error = %v", extensionPackID, err)
	}
}

func storeHotUpdatePromotionForLastKnownGoodRecertFixture(t *testing.T, root string, now time.Time) (PromotionRecord, HotUpdateOutcomeRecord, HotUpdateGateRecord) {
	t.Helper()

	outcome, gate := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
	promotion, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(12*time.Minute))
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = false, want true")
	}
	return promotion, outcome, gate
}

type lastKnownGoodRecertificationSideEffectSnapshot struct {
	activePointerBytes []byte
	lastKnownGoodBytes []byte
	promotionBytes     []byte
	outcomeBytes       []byte
	gateBytes          []byte
	reloadGeneration   uint64
	promotionRecords   int
	outcomeRecords     int
}

func snapshotLastKnownGoodRecertificationSideEffects(t *testing.T, root, promotionID, outcomeID, hotUpdateID string) lastKnownGoodRecertificationSideEffectSnapshot {
	t.Helper()

	return snapshotLastKnownGoodRecertificationSideEffectsWithOptions(t, root, promotionID, outcomeID, hotUpdateID, false)
}

func snapshotLastKnownGoodRecertificationSideEffectsAllowMissingPointers(t *testing.T, root, promotionID, outcomeID, hotUpdateID string) lastKnownGoodRecertificationSideEffectSnapshot {
	t.Helper()

	return snapshotLastKnownGoodRecertificationSideEffectsWithOptions(t, root, promotionID, outcomeID, hotUpdateID, true)
}

func snapshotLastKnownGoodRecertificationSideEffectsWithOptions(t *testing.T, root, promotionID, outcomeID, hotUpdateID string, allowMissingPointers bool) lastKnownGoodRecertificationSideEffectSnapshot {
	t.Helper()

	activePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil && !allowMissingPointers {
		t.Fatalf("ReadFile(active pointer) error = %v", err)
	}
	var reloadGeneration uint64
	if err == nil {
		activePointer, err := LoadActiveRuntimePackPointer(root)
		if err != nil {
			t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
		}
		reloadGeneration = activePointer.ReloadGeneration
	}
	lastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil && !allowMissingPointers {
		t.Fatalf("ReadFile(last-known-good pointer) error = %v", err)
	}
	promotionBytes, err := os.ReadFile(StorePromotionPath(root, promotionID))
	if err != nil {
		t.Fatalf("ReadFile(promotion) error = %v", err)
	}
	outcomeBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, outcomeID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update outcome) error = %v", err)
	}
	gateBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate) error = %v", err)
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	outcomes, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	return lastKnownGoodRecertificationSideEffectSnapshot{
		activePointerBytes: activePointerBytes,
		lastKnownGoodBytes: lastKnownGoodBytes,
		promotionBytes:     promotionBytes,
		outcomeBytes:       outcomeBytes,
		gateBytes:          gateBytes,
		reloadGeneration:   reloadGeneration,
		promotionRecords:   len(promotions),
		outcomeRecords:     len(outcomes),
	}
}

func assertLastKnownGoodRecertificationSideEffectsUnchangedExceptLastKnownGood(t *testing.T, root, promotionID, outcomeID, hotUpdateID string, before lastKnownGoodRecertificationSideEffectSnapshot) {
	t.Helper()

	after := snapshotLastKnownGoodRecertificationSideEffects(t, root, promotionID, outcomeID, hotUpdateID)
	assertLastKnownGoodRecertificationStableFields(t, before, after)
}

func assertLastKnownGoodRecertificationSideEffectsFullyUnchanged(t *testing.T, root, promotionID, outcomeID, hotUpdateID string, before lastKnownGoodRecertificationSideEffectSnapshot) {
	t.Helper()

	after := snapshotLastKnownGoodRecertificationSideEffects(t, root, promotionID, outcomeID, hotUpdateID)
	assertLastKnownGoodRecertificationStableFields(t, before, after)
	if string(after.lastKnownGoodBytes) != string(before.lastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(after.lastKnownGoodBytes))
	}
}

func assertLastKnownGoodRecertificationSideEffectsFullyUnchangedAllowMissingPointers(t *testing.T, root, promotionID, outcomeID, hotUpdateID string, before lastKnownGoodRecertificationSideEffectSnapshot) {
	t.Helper()

	after := snapshotLastKnownGoodRecertificationSideEffectsAllowMissingPointers(t, root, promotionID, outcomeID, hotUpdateID)
	assertLastKnownGoodRecertificationStableFields(t, before, after)
	if string(after.lastKnownGoodBytes) != string(before.lastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(after.lastKnownGoodBytes))
	}
}

func assertLastKnownGoodRecertificationStableFields(t *testing.T, before, after lastKnownGoodRecertificationSideEffectSnapshot) {
	t.Helper()

	if string(after.activePointerBytes) != string(before.activePointerBytes) {
		t.Fatalf("active runtime-pack pointer changed\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(after.activePointerBytes))
	}
	if after.reloadGeneration != before.reloadGeneration {
		t.Fatalf("reload_generation = %d, want %d", after.reloadGeneration, before.reloadGeneration)
	}
	if string(after.promotionBytes) != string(before.promotionBytes) {
		t.Fatalf("promotion record changed\nbefore:\n%s\nafter:\n%s", string(before.promotionBytes), string(after.promotionBytes))
	}
	if string(after.outcomeBytes) != string(before.outcomeBytes) {
		t.Fatalf("hot-update outcome changed\nbefore:\n%s\nafter:\n%s", string(before.outcomeBytes), string(after.outcomeBytes))
	}
	if string(after.gateBytes) != string(before.gateBytes) {
		t.Fatalf("hot-update gate changed\nbefore:\n%s\nafter:\n%s", string(before.gateBytes), string(after.gateBytes))
	}
	if after.promotionRecords != before.promotionRecords {
		t.Fatalf("promotion record count = %d, want %d", after.promotionRecords, before.promotionRecords)
	}
	if after.outcomeRecords != before.outcomeRecords {
		t.Fatalf("hot-update outcome record count = %d, want %d", after.outcomeRecords, before.outcomeRecords)
	}
}

func writeRawPromotionRecord(t *testing.T, root string, record PromotionRecord) {
	t.Helper()

	record = NormalizePromotionRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := WriteStoreJSONAtomic(StorePromotionPath(root, record.PromotionID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(promotion) error = %v", err)
	}
}
