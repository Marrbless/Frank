package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPromotionRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storePromotionFixtures(t, root, now)

	second := validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-b"
		record.OutcomeID = ""
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.LastKnownGoodPackID = ""
		record.LastKnownGoodBasis = ""
		record.Reason = "manual promotion checkpoint"
		record.Notes = "operator record only"
		record.CreatedBy = "system"
	})
	if err := StorePromotionRecord(root, second); err != nil {
		t.Fatalf("StorePromotionRecord(promotion-b) error = %v", err)
	}

	want := validPromotionRecord(now.Add(11*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = " promotion-a "
		record.PromotedPackID = " pack-candidate "
		record.PreviousActivePackID = " pack-base "
		record.LastKnownGoodPackID = " pack-base "
		record.LastKnownGoodBasis = " holdout_pass "
		record.HotUpdateID = " hot-update-1 "
		record.OutcomeID = " outcome-1 "
		record.CandidateID = " candidate-1 "
		record.RunID = " run-1 "
		record.CandidateResultID = " result-1 "
		record.Reason = " operator promotion "
		record.Notes = " durable promotion ledger "
		record.CreatedBy = " operator "
	})
	if err := StorePromotionRecord(root, want); err != nil {
		t.Fatalf("StorePromotionRecord(promotion-a) error = %v", err)
	}

	got, err := LoadPromotionRecord(root, "promotion-a")
	if err != nil {
		t.Fatalf("LoadPromotionRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.PromotionID = "promotion-a"
	want.PromotedPackID = "pack-candidate"
	want.PreviousActivePackID = "pack-base"
	want.LastKnownGoodPackID = "pack-base"
	want.LastKnownGoodBasis = "holdout_pass"
	want.HotUpdateID = "hot-update-1"
	want.OutcomeID = "outcome-1"
	want.CandidateID = "candidate-1"
	want.RunID = "run-1"
	want.CandidateResultID = "result-1"
	want.Reason = "operator promotion"
	want.Notes = "durable promotion ledger"
	want.PromotedAt = want.PromotedAt.UTC()
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadPromotionRecord() = %#v, want %#v", got, want)
	}

	records, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListPromotionRecords() len = %d, want 2", len(records))
	}
	if records[0].PromotionID != "promotion-a" || records[1].PromotionID != "promotion-b" {
		t.Fatalf("ListPromotionRecords() ids = [%q %q], want [promotion-a promotion-b]", records[0].PromotionID, records[1].PromotionID)
	}
}

func TestPromotionReplayIsIdempotentAndImmutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC)

	storePromotionFixtures(t, root, now)

	record := validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-replay"
		record.Reason = "exact replay"
		record.Notes = "same bytes expected"
	})
	if err := StorePromotionRecord(root, record); err != nil {
		t.Fatalf("StorePromotionRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StorePromotionPath(root, record.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StorePromotionRecord(root, record); err != nil {
		t.Fatalf("StorePromotionRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StorePromotionPath(root, record.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("promotion file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(11*time.Minute), func(changed *PromotionRecord) {
		changed.PromotionID = "promotion-replay"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want immutable duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store promotion "promotion-replay" already exists`) {
		t.Fatalf("StorePromotionRecord() error = %q, want immutable duplicate rejection", err.Error())
	}
}

func TestPromotionValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing promotion id",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.PromotionID = " "
				}))
			},
			want: "promotion ref promotion_id is required",
		},
		{
			name: "missing reason",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.Reason = " "
				}))
			},
			want: "mission store promotion reason is required",
		},
		{
			name: "missing promoted at",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.PromotedAt = time.Time{}
				}))
			},
			want: "mission store promotion promoted_at is required",
		},
		{
			name: "basis without last known good pack",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.LastKnownGoodPackID = ""
					record.LastKnownGoodBasis = "holdout_pass"
				}))
			},
			want: "mission store promotion last_known_good_pack_id is required with last_known_good_basis",
		},
		{
			name: "missing created by",
			run: func() error {
				return StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
					record.CreatedBy = " "
				}))
			},
			want: "mission store promotion created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StorePromotionRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StorePromotionRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestPromotionRejectsMissingAndMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 27, 13, 0, 0, 0, time.UTC)

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
		record.SourceSummary = "baseline pack"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.SourceSummary = "candidate pack"
	}))

	err := StorePromotionRecord(root, validPromotionRecord(now, func(record *PromotionRecord) {
		record.HotUpdateID = "missing-gate"
		record.OutcomeID = ""
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.LastKnownGoodPackID = ""
		record.LastKnownGoodBasis = ""
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StorePromotionRecord() error = %q, want missing gate rejection", err.Error())
	}

	storePromotionFixtures(t, root, now.Add(time.Minute))

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(10*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-mismatch"
		record.PromotedPackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want promoted pack linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `promoted_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`) {
		t.Fatalf("StorePromotionRecord() error = %q, want promoted pack linkage mismatch rejection", err.Error())
	}

	err = StorePromotionRecord(root, validPromotionRecord(now.Add(11*time.Minute), func(record *PromotionRecord) {
		record.PromotionID = "promotion-missing-outcome"
		record.OutcomeID = "missing-outcome"
	}))
	if err == nil {
		t.Fatal("StorePromotionRecord() error = nil, want missing outcome rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateOutcomeRecordNotFound.Error()) {
		t.Fatalf("StorePromotionRecord() error = %q, want missing outcome rejection", err.Error())
	}
}

func TestLoadPromotionRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadPromotionRecord(root, "missing-promotion"); !errors.Is(err, ErrPromotionRecordNotFound) {
		t.Fatalf("LoadPromotionRecord() error = %v, want %v", err, ErrPromotionRecordNotFound)
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeCreatesPromotionAndPreservesState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 9, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	outcome, gate := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
	before := snapshotHotUpdatePromotionSideEffects(t, root, outcome.OutcomeID, gate.HotUpdateID)

	createdAt := now.Add(12 * time.Minute)
	got, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, " "+outcome.OutcomeID+" ", " operator ", createdAt)
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = false, want true")
	}

	want := PromotionRecord{
		RecordVersion:        StoreRecordVersion,
		PromotionID:          "hot-update-promotion-hot-update-1",
		PromotedPackID:       "pack-candidate",
		PreviousActivePackID: "pack-base",
		HotUpdateID:          "hot-update-1",
		OutcomeID:            "hot-update-outcome-hot-update-1",
		Reason:               "hot update outcome promoted",
		PromotedAt:           outcome.OutcomeAt.UTC(),
		CreatedAt:            createdAt.UTC(),
		CreatedBy:            "operator",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() = %#v, want %#v", got, want)
	}
	if got.CanaryRef != "" || got.ApprovalRef != "" {
		t.Fatalf("audit refs = %q/%q, want empty for non-canary promotion", got.CanaryRef, got.ApprovalRef)
	}

	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 1 {
		t.Fatalf("ListPromotionRecords() len = %d, want 1", len(promotions))
	}
	assertHotUpdatePromotionSideEffectsUnchanged(t, root, outcome.OutcomeID, gate.HotUpdateID, before)
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomePropagatesCanaryAuditLineage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		owner bool
	}{
		{name: "no owner"},
		{name: "owner approved", owner: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 4, 9, 30, 0, 0, time.UTC)
			fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", tc.owner)
			outcome, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
			}
			if !changed {
				t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
			}
			before := snapshotHotUpdatePromotionSideEffects(t, root, outcome.OutcomeID, fixture.gate.HotUpdateID)
			sourceBefore := snapshotCanaryAuditSourceRecords(t, root, fixture)

			got, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(32*time.Minute))
			if err != nil {
				t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v", err)
			}
			if !changed {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = false, want true")
			}
			if got.CanaryRef != outcome.CanaryRef || got.CanaryRef != fixture.authority.CanarySatisfactionAuthorityID {
				t.Fatalf("CanaryRef = %q, want outcome/gate canary ref %q", got.CanaryRef, fixture.authority.CanarySatisfactionAuthorityID)
			}
			if tc.owner {
				if got.ApprovalRef != outcome.ApprovalRef || got.ApprovalRef != fixture.decision.OwnerApprovalDecisionID {
					t.Fatalf("ApprovalRef = %q, want outcome/gate approval ref %q", got.ApprovalRef, fixture.decision.OwnerApprovalDecisionID)
				}
			} else if got.ApprovalRef != "" {
				t.Fatalf("ApprovalRef = %q, want empty", got.ApprovalRef)
			}

			assertHotUpdatePromotionSideEffectsUnchanged(t, root, outcome.OutcomeID, fixture.gate.HotUpdateID, before)
			assertCanaryAuditSourceRecordsUnchanged(t, root, fixture, sourceBefore)
			assertNoHotUpdatePromotionForbiddenRecords(t, root)
		})
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeDoesNotReauthorizeCanaryAfterTerminalOutcome(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 9, 45, 0, 0, time.UTC)
	fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", true)
	outcome, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}

	staleDecision := fixture.decision
	staleDecision.Decision = HotUpdateOwnerApprovalDecisionRejected
	staleDecision.Reason = "drift after terminal outcome"
	if err := WriteStoreJSONAtomic(StoreHotUpdateOwnerApprovalDecisionPath(root, staleDecision.OwnerApprovalDecisionID), staleDecision); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(stale owner approval decision) error = %v", err)
	}

	got, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(32*time.Minute))
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = false, want true")
	}
	if got.CanaryRef != outcome.CanaryRef || got.ApprovalRef != outcome.ApprovalRef {
		t.Fatalf("promotion refs = %q/%q, want outcome refs %q/%q", got.CanaryRef, got.ApprovalRef, outcome.CanaryRef, outcome.ApprovalRef)
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeRejectsMissingOutcome(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)

	_, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, "missing-outcome", "operator", now)
	if !errors.Is(err, ErrHotUpdateOutcomeRecordNotFound) {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v, want %v", err, ErrHotUpdateOutcomeRecordNotFound)
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
	}
	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeRejectsNonHotUpdatedOutcomes(t *testing.T) {
	t.Parallel()

	kinds := []HotUpdateOutcomeKind{
		HotUpdateOutcomeKindFailed,
		HotUpdateOutcomeKindKeptStaged,
		HotUpdateOutcomeKindDiscarded,
		HotUpdateOutcomeKindBlocked,
		HotUpdateOutcomeKindApprovalRequired,
		HotUpdateOutcomeKindColdRestartNeeded,
		HotUpdateOutcomeKindCanaryApplied,
		HotUpdateOutcomeKindPromoted,
		HotUpdateOutcomeKindRolledBack,
		HotUpdateOutcomeKindAborted,
	}

	for _, kind := range kinds {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC)
			outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, func(record *HotUpdateOutcomeRecord) {
				record.OutcomeKind = kind
				record.Reason = "non-successful outcome"
			}, nil)

			_, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(12*time.Minute))
			if err == nil {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() error = nil, want non-hot_updated rejection")
			}
			if changed {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
			}
			if !strings.Contains(err.Error(), "does not permit promotion creation") {
				t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %q, want non-hot_updated context", err.Error())
			}
			assertPromotionRecordCount(t, root, 0)
		})
	}

	t.Run("unknown future kind", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		now := time.Date(2026, 5, 4, 11, 30, 0, 0, time.UTC)
		outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
		outcome.OutcomeKind = HotUpdateOutcomeKind("future_kind")
		writeRawHotUpdateOutcomeRecord(t, root, outcome)

		_, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(12*time.Minute))
		if err == nil {
			t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() error = nil, want invalid kind rejection")
		}
		if changed {
			t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
		}
		if !strings.Contains(err.Error(), `outcome_kind "future_kind" is invalid`) {
			t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %q, want invalid kind context", err.Error())
		}
		assertPromotionRecordCount(t, root, 0)
	})
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeRejectsInvalidOutcomeLinkage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, root string, now time.Time) string
		want  string
	}{
		{
			name: "empty candidate pack",
			setup: func(t *testing.T, root string, now time.Time) string {
				outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, func(record *HotUpdateOutcomeRecord) {
					record.CandidatePackID = ""
				}, nil)
				return outcome.OutcomeID
			},
			want: `candidate_pack_id is required for promotion creation`,
		},
		{
			name: "candidate pack mismatch with gate",
			setup: func(t *testing.T, root string, now time.Time) string {
				outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
				mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(13*time.Minute), func(record *RuntimePackRecord) {
					record.PackID = "pack-other"
					record.ParentPackID = "pack-base"
					record.RollbackTargetPackID = "pack-base"
				}))
				outcome.CandidatePackID = "pack-other"
				writeRawHotUpdateOutcomeRecord(t, root, outcome)
				return outcome.OutcomeID
			},
			want: `candidate_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`,
		},
		{
			name: "gate missing previous active pack",
			setup: func(t *testing.T, root string, now time.Time) string {
				outcome, gate := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
				gate.PreviousActivePackID = "pack-missing"
				writeRawHotUpdateGateRecord(t, root, gate)
				return outcome.OutcomeID
			},
			want: `previous_active_pack_id "pack-missing"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
			outcomeID := tc.setup(t, root, now)

			_, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcomeID, "operator", now.Add(20*time.Minute))
			if err == nil {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() error = nil, want fail-closed rejection")
			}
			if changed {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %q, want substring %q", err.Error(), tc.want)
			}
			assertPromotionRecordCount(t, root, 0)
		})
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeRejectsOutcomeGateLineageMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*HotUpdateOutcomeRecord)
		want   string
	}{
		{
			name: "canary ref",
			mutate: func(record *HotUpdateOutcomeRecord) {
				record.CanaryRef = "hot-update-canary-satisfaction-authority-other"
			},
			want: "canary_ref",
		},
		{
			name: "approval ref",
			mutate: func(record *HotUpdateOutcomeRecord) {
				record.ApprovalRef = HotUpdateOwnerApprovalDecisionIDFromRequest("hot-update-owner-approval-request-other")
			},
			want: "approval_ref",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 4, 12, 30, 0, 0, time.UTC)
			fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", true)
			outcome, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
			}
			if !changed {
				t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
			}
			tc.mutate(&outcome)
			writeRawHotUpdateOutcomeRecord(t, root, outcome)

			_, changed, err = CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(32*time.Minute))
			if err == nil {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() error = nil, want lineage mismatch rejection")
			}
			if changed {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %q, want substring %q", err.Error(), tc.want)
			}
			assertPromotionRecordCount(t, root, 0)
		})
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeReplayIsIdempotentAndDivergentDuplicateFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC)
	outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
	createdAt := now.Add(12 * time.Minute)

	first, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(first) error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StorePromotionPath(root, first.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(first promotion) error = %v", err)
	}

	second, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StorePromotionPath(root, first.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(second promotion) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("promotion file changed on exact replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	_, changed, err = CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt.Add(time.Minute))
	if err == nil {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent replay) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent replay) changed = true, want false")
	}
	if !strings.Contains(err.Error(), `mission store promotion "hot-update-promotion-hot-update-1" already exists`) {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent replay) error = %q, want deterministic duplicate context", err.Error())
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeDivergentDuplicateLineageFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 13, 30, 0, 0, time.UTC)
	fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", true)
	outcome, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}
	createdAt := now.Add(32 * time.Minute)

	first, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(first) error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StorePromotionPath(root, first.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(first promotion) error = %v", err)
	}

	second, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(replay) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StorePromotionPath(root, first.PromotionID))
	if err != nil {
		t.Fatalf("ReadFile(second promotion) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatalf("promotion file changed on exact replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	divergent := first
	divergent.CanaryRef = "hot-update-canary-satisfaction-authority-other"
	writeRawPromotionRecord(t, root, divergent)
	_, changed, err = CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent canary_ref) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent canary_ref) changed = true, want false")
	}
	if !strings.Contains(err.Error(), "canary_ref") {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent canary_ref) error = %q, want canary_ref context", err.Error())
	}

	if err := WriteStoreJSONAtomic(StorePromotionPath(root, first.PromotionID), first); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(restore promotion) error = %v", err)
	}
	divergent = first
	divergent.ApprovalRef = HotUpdateOwnerApprovalDecisionIDFromRequest("hot-update-owner-approval-request-other")
	writeRawPromotionRecord(t, root, divergent)
	_, changed, err = CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent approval_ref) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent approval_ref) changed = true, want false")
	}
	if !strings.Contains(err.Error(), "approval_ref") {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome(divergent approval_ref) error = %q, want approval_ref context", err.Error())
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeRejectsExistingDifferentPromotionForSameOutcomeOrHotUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*PromotionRecord)
		want   string
	}{
		{
			name: "same outcome different promotion id",
			mutate: func(record *PromotionRecord) {
				record.PromotionID = "legacy-promotion"
			},
			want: `outcome_id "hot-update-outcome-hot-update-1" already exists as "legacy-promotion"`,
		},
		{
			name: "same hot update different promotion id",
			mutate: func(record *PromotionRecord) {
				record.PromotionID = "legacy-promotion"
				record.OutcomeID = ""
			},
			want: `hot_update_id "hot-update-1" already exists as "legacy-promotion"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
			outcome, _ := storeSuccessfulHotUpdateOutcomePromotionFixture(t, root, now, nil, nil)
			legacy := PromotionRecord{
				PromotionID:          "legacy-promotion",
				PromotedPackID:       "pack-candidate",
				PreviousActivePackID: "pack-base",
				HotUpdateID:          "hot-update-1",
				OutcomeID:            outcome.OutcomeID,
				Reason:               "legacy promotion",
				PromotedAt:           outcome.OutcomeAt,
				CreatedAt:            now.Add(13 * time.Minute),
				CreatedBy:            "operator",
			}
			tc.mutate(&legacy)
			if err := StorePromotionRecord(root, legacy); err != nil {
				t.Fatalf("StorePromotionRecord(legacy) error = %v", err)
			}

			_, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(14*time.Minute))
			if err == nil {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() error = nil, want duplicate linkage rejection")
			}
			if changed {
				t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = true, want false")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %q, want substring %q", err.Error(), tc.want)
			}
			assertPromotionRecordCount(t, root, 1)
		})
	}
}

func TestCreatePromotionFromSuccessfulHotUpdateOutcomeCopiesOptionalRefsWhenPresent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	storeHotUpdateOutcomeFixtures(t, root, now)

	outcome := validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "hot-update-outcome-hot-update-1"
		record.HotUpdateID = "hot-update-1"
		record.CandidateID = "candidate-1"
		record.RunID = "run-1"
		record.CandidateResultID = "result-1"
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindHotUpdated
		record.Reason = "hot update reload/apply succeeded"
		record.CreatedBy = "operator"
	})
	if err := StoreHotUpdateOutcomeRecord(root, outcome); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}

	got, changed, err := CreatePromotionFromSuccessfulHotUpdateOutcome(root, outcome.OutcomeID, "operator", now.Add(9*time.Minute))
	if err != nil {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() error = %v", err)
	}
	if !changed {
		t.Fatal("CreatePromotionFromSuccessfulHotUpdateOutcome() changed = false, want true")
	}
	if got.CandidateID != "candidate-1" {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome().CandidateID = %q, want candidate-1", got.CandidateID)
	}
	if got.RunID != "run-1" {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome().RunID = %q, want run-1", got.RunID)
	}
	if got.CandidateResultID != "result-1" {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome().CandidateResultID = %q, want result-1", got.CandidateResultID)
	}
	if got.LastKnownGoodPackID != "" || got.LastKnownGoodBasis != "" {
		t.Fatalf("CreatePromotionFromSuccessfulHotUpdateOutcome() LKG fields = (%q, %q), want empty", got.LastKnownGoodPackID, got.LastKnownGoodBasis)
	}
}

func validPromotionRecord(now time.Time, mutate func(*PromotionRecord)) PromotionRecord {
	record := PromotionRecord{
		PromotionID:          "promotion-root",
		PromotedPackID:       "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		LastKnownGoodBasis:   "holdout_pass",
		HotUpdateID:          "hot-update-1",
		OutcomeID:            "outcome-1",
		CandidateID:          "candidate-1",
		RunID:                "run-1",
		CandidateResultID:    "result-1",
		Reason:               "promotion approved",
		Notes:                "persisted without applying control-plane changes",
		PromotedAt:           now,
		CreatedAt:            now.Add(time.Minute),
		CreatedBy:            "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storePromotionFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeHotUpdateOutcomeFixtures(t, root, now)
	if err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-1"
		record.HotUpdateID = "hot-update-1"
		record.CandidateID = "candidate-1"
		record.RunID = "run-1"
		record.CandidateResultID = "result-1"
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindPromoted
		record.Reason = "promotion candidate selected"
		record.Notes = "recorded before promotion ledger"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}
}

func storeSuccessfulHotUpdateOutcomePromotionFixture(t *testing.T, root string, now time.Time, mutateOutcome func(*HotUpdateOutcomeRecord), mutateGate func(*HotUpdateGateRecord)) (HotUpdateOutcomeRecord, HotUpdateGateRecord) {
	t.Helper()

	gate := storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-1", HotUpdateGateStateReloadApplySucceeded, "")
	if mutateGate != nil {
		mutateGate(&gate)
		if err := StoreHotUpdateGateRecord(root, gate); err != nil {
			t.Fatalf("StoreHotUpdateGateRecord(mutated gate) error = %v", err)
		}
	}

	outcome := validHotUpdateOutcomeRecord(now.Add(11*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "hot-update-outcome-hot-update-1"
		record.HotUpdateID = gate.HotUpdateID
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindHotUpdated
		record.Reason = "hot update reload/apply succeeded"
		record.Notes = ""
		record.OutcomeAt = gate.PhaseUpdatedAt
		record.CreatedBy = "operator"
	})
	if mutateOutcome != nil {
		mutateOutcome(&outcome)
	}
	if err := StoreHotUpdateOutcomeRecord(root, outcome); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %v", err)
	}
	storedOutcome, err := LoadHotUpdateOutcomeRecord(root, outcome.OutcomeID)
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}
	storedGate, err := LoadHotUpdateGateRecord(root, gate.HotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	return storedOutcome, storedGate
}

type hotUpdatePromotionSideEffectSnapshot struct {
	activePointerBytes []byte
	lastKnownGoodBytes []byte
	gateBytes          []byte
	outcomeBytes       []byte
	reloadGeneration   uint64
	outcomeRecords     int
	gateRecords        int
}

func snapshotHotUpdatePromotionSideEffects(t *testing.T, root, outcomeID, hotUpdateID string) hotUpdatePromotionSideEffectSnapshot {
	t.Helper()

	activePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer) error = %v", err)
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	lastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer) error = %v", err)
	}
	gateBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate) error = %v", err)
	}
	outcomeBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, outcomeID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update outcome) error = %v", err)
	}
	outcomeRecords, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	gateRecords, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}

	return hotUpdatePromotionSideEffectSnapshot{
		activePointerBytes: activePointerBytes,
		lastKnownGoodBytes: lastKnownGoodBytes,
		gateBytes:          gateBytes,
		outcomeBytes:       outcomeBytes,
		reloadGeneration:   activePointer.ReloadGeneration,
		outcomeRecords:     len(outcomeRecords),
		gateRecords:        len(gateRecords),
	}
}

func assertHotUpdatePromotionSideEffectsUnchanged(t *testing.T, root, outcomeID, hotUpdateID string, before hotUpdatePromotionSideEffectSnapshot) {
	t.Helper()

	after := snapshotHotUpdatePromotionSideEffects(t, root, outcomeID, hotUpdateID)
	if string(after.activePointerBytes) != string(before.activePointerBytes) {
		t.Fatalf("active runtime-pack pointer changed\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(after.activePointerBytes))
	}
	if after.reloadGeneration != before.reloadGeneration {
		t.Fatalf("reload_generation = %d, want %d", after.reloadGeneration, before.reloadGeneration)
	}
	if string(after.lastKnownGoodBytes) != string(before.lastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(after.lastKnownGoodBytes))
	}
	if string(after.gateBytes) != string(before.gateBytes) {
		t.Fatalf("hot-update gate changed\nbefore:\n%s\nafter:\n%s", string(before.gateBytes), string(after.gateBytes))
	}
	if string(after.outcomeBytes) != string(before.outcomeBytes) {
		t.Fatalf("hot-update outcome changed\nbefore:\n%s\nafter:\n%s", string(before.outcomeBytes), string(after.outcomeBytes))
	}
	if after.outcomeRecords != before.outcomeRecords {
		t.Fatalf("hot-update outcome record count = %d, want %d", after.outcomeRecords, before.outcomeRecords)
	}
	if after.gateRecords != before.gateRecords {
		t.Fatalf("hot-update gate record count = %d, want %d", after.gateRecords, before.gateRecords)
	}
}

func writeRawHotUpdateOutcomeRecord(t *testing.T, root string, record HotUpdateOutcomeRecord) {
	t.Helper()

	record = NormalizeHotUpdateOutcomeRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := WriteStoreJSONAtomic(StoreHotUpdateOutcomePath(root, record.OutcomeID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(hot-update outcome) error = %v", err)
	}
}

func writeRawHotUpdateGateRecord(t *testing.T, root string, record HotUpdateGateRecord) {
	t.Helper()

	record = NormalizeHotUpdateGateRecord(record)
	record.RecordVersion = normalizeRecordVersion(record.RecordVersion)
	if err := WriteStoreJSONAtomic(StoreHotUpdateGatePath(root, record.HotUpdateID), record); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(hot-update gate) error = %v", err)
	}
}

func assertNoHotUpdatePromotionForbiddenRecords(t *testing.T, root string) {
	t.Helper()

	rollbacks, err := ListRollbackRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackRecords() error = %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("ListRollbackRecords() len = %d, want 0", len(rollbacks))
	}
	applies, err := ListRollbackApplyRecords(root)
	if err != nil {
		t.Fatalf("ListRollbackApplyRecords() error = %v", err)
	}
	if len(applies) != 0 {
		t.Fatalf("ListRollbackApplyRecords() len = %d, want 0", len(applies))
	}
	decisions, err := ListCandidatePromotionDecisionRecords(root)
	if err != nil {
		t.Fatalf("ListCandidatePromotionDecisionRecords() error = %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("ListCandidatePromotionDecisionRecords() len = %d, want 0", len(decisions))
	}
}

func assertPromotionRecordCount(t *testing.T, root string, want int) {
	t.Helper()

	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != want {
		t.Fatalf("ListPromotionRecords() len = %d, want %d", len(promotions), want)
	}
}
