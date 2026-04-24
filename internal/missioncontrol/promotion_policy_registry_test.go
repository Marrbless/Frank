package missioncontrol

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPromotionPolicyRecordStoreLoadListAndRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 15, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	second := validPromotionPolicyRecord(now.Add(time.Minute), func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-b"
		record.AllowedSurfaceClasses = []string{"skill"}
		record.ForbiddenSurfaceChanges = []string{"runtime_source"}
	})
	if err := StorePromotionPolicyRecord(root, second); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(promotion-policy-b) error = %v", err)
	}

	want := validPromotionPolicyRecord(now, func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = " promotion-policy-a "
		record.AllowedSurfaceClasses = []string{" skill ", " prompt_pack ", "skill"}
		record.EpsilonRule = " epsilon <= 0.01 "
		record.RegressionRule = " no holdout regression "
		record.CompatibilityRule = " compatibility contract required "
		record.ResourceRule = " canary resource ceiling "
		record.MaxCanaryDuration = " 30m "
		record.ForbiddenSurfaceChanges = []string{" runtime_source ", "policy", "", "policy"}
		record.CreatedBy = " operator "
		record.Notes = " frozen skeleton policy "
	})
	if err := StorePromotionPolicyRecord(root, want); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(promotion-policy-a) error = %v", err)
	}

	got, err := LoadPromotionPolicyRecord(root, "promotion-policy-a")
	if err != nil {
		t.Fatalf("LoadPromotionPolicyRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.PromotionPolicyID = "promotion-policy-a"
	want.AllowedSurfaceClasses = []string{"prompt_pack", "skill"}
	want.EpsilonRule = "epsilon <= 0.01"
	want.RegressionRule = "no holdout regression"
	want.CompatibilityRule = "compatibility contract required"
	want.ResourceRule = "canary resource ceiling"
	want.MaxCanaryDuration = "30m"
	want.ForbiddenSurfaceChanges = []string{"policy", "runtime_source"}
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	want.Notes = "frozen skeleton policy"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadPromotionPolicyRecord() = %#v, want %#v", got, want)
	}

	records, err := ListPromotionPolicyRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionPolicyRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListPromotionPolicyRecords() len = %d, want 2", len(records))
	}
	if records[0].PromotionPolicyID != "promotion-policy-a" || records[1].PromotionPolicyID != "promotion-policy-b" {
		t.Fatalf("ListPromotionPolicyRecords() ids = [%q %q], want [promotion-policy-a promotion-policy-b]", records[0].PromotionPolicyID, records[1].PromotionPolicyID)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded PromotionPolicyRecord
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, got) {
		t.Fatalf("JSON round-trip = %#v, want %#v", decoded, got)
	}
}

func TestPromotionPolicyReplayIsIdempotentAndDivergentDuplicateFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 16, 0, 0, 0, time.UTC)
	record := validPromotionPolicyRecord(now, func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-replay"
	})

	if err := StorePromotionPolicyRecord(root, record); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StorePromotionPolicyPath(root, record.PromotionPolicyID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StorePromotionPolicyRecord(root, record); err != nil {
		t.Fatalf("StorePromotionPolicyRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StorePromotionPolicyPath(root, record.PromotionPolicyID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("promotion policy file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now, func(record *PromotionPolicyRecord) {
		record.PromotionPolicyID = "promotion-policy-replay"
		record.EpsilonRule = "epsilon <= 0.02"
	}))
	if err == nil {
		t.Fatal("StorePromotionPolicyRecord(divergent) error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store promotion policy "promotion-policy-replay" already exists`) {
		t.Fatalf("StorePromotionPolicyRecord(divergent) error = %q, want duplicate context", err.Error())
	}
}

func TestPromotionPolicyValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		edit func(*PromotionPolicyRecord)
		want string
	}{
		{
			name: "missing promotion policy id",
			edit: func(record *PromotionPolicyRecord) {
				record.PromotionPolicyID = " "
			},
			want: "promotion policy ref promotion_policy_id is required",
		},
		{
			name: "missing allowed surface classes",
			edit: func(record *PromotionPolicyRecord) {
				record.AllowedSurfaceClasses = nil
			},
			want: "mission store promotion policy allowed_surface_classes are required",
		},
		{
			name: "missing epsilon rule",
			edit: func(record *PromotionPolicyRecord) {
				record.EpsilonRule = " "
			},
			want: "mission store promotion policy epsilon_rule is required",
		},
		{
			name: "missing regression rule",
			edit: func(record *PromotionPolicyRecord) {
				record.RegressionRule = " "
			},
			want: "mission store promotion policy regression_rule is required",
		},
		{
			name: "missing compatibility rule",
			edit: func(record *PromotionPolicyRecord) {
				record.CompatibilityRule = " "
			},
			want: "mission store promotion policy compatibility_rule is required",
		},
		{
			name: "missing resource rule",
			edit: func(record *PromotionPolicyRecord) {
				record.ResourceRule = " "
			},
			want: "mission store promotion policy resource_rule is required",
		},
		{
			name: "invalid max canary duration",
			edit: func(record *PromotionPolicyRecord) {
				record.MaxCanaryDuration = "not-a-duration"
			},
			want: `mission store promotion policy max_canary_duration "not-a-duration" is invalid`,
		},
		{
			name: "negative max canary duration",
			edit: func(record *PromotionPolicyRecord) {
				record.MaxCanaryDuration = "-1s"
			},
			want: "mission store promotion policy max_canary_duration must be positive",
		},
		{
			name: "missing forbidden surface changes",
			edit: func(record *PromotionPolicyRecord) {
				record.ForbiddenSurfaceChanges = nil
			},
			want: "mission store promotion policy forbidden_surface_changes are required",
		},
		{
			name: "missing created at",
			edit: func(record *PromotionPolicyRecord) {
				record.CreatedAt = time.Time{}
			},
			want: "mission store promotion policy created_at is required",
		},
		{
			name: "missing created by",
			edit: func(record *PromotionPolicyRecord) {
				record.CreatedBy = " "
			},
			want: "mission store promotion policy created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now, tc.edit))
			if err == nil {
				t.Fatal("StorePromotionPolicyRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StorePromotionPolicyRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestPromotionPolicyStoreDoesNotMutateRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 29, 18, 0, 0, 0, time.UTC)
	if err := StorePromotionPolicyRecord(root, validPromotionPolicyRecord(now)); err != nil {
		t.Fatalf("StorePromotionPolicyRecord() error = %v", err)
	}

	absentPaths := []string{
		StoreRuntimePacksDir(root),
		StoreImprovementCandidatesDir(root),
		StoreEvalSuitesDir(root),
		StoreImprovementRunsDir(root),
		StoreCandidateResultsDir(root),
		StoreHotUpdateGatesDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after promotion policy store: %v", path, err)
		}
	}
}

func validPromotionPolicyRecord(at time.Time, edits ...func(*PromotionPolicyRecord)) PromotionPolicyRecord {
	record := PromotionPolicyRecord{
		PromotionPolicyID:         "promotion-policy-1",
		RequiresHoldoutPass:       true,
		RequiresCanary:            true,
		RequiresOwnerApproval:     true,
		AllowsAutonomousHotUpdate: false,
		AllowedSurfaceClasses:     []string{"prompt_pack", "skill", "source_patch_artifact"},
		EpsilonRule:               "epsilon <= 0.01",
		RegressionRule:            "holdout_regression <= 0",
		CompatibilityRule:         "compatibility_contract_passed",
		ResourceRule:              "canary_budget_within_limit",
		MaxCanaryDuration:         "15m",
		ForbiddenSurfaceChanges:   []string{"policy", "runtime_source"},
		CreatedAt:                 at,
		CreatedBy:                 "operator",
		Notes:                     "registry skeleton fixture",
	}
	for _, edit := range edits {
		edit(&record)
	}
	return record
}
