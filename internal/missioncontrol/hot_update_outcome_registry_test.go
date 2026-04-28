package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHotUpdateOutcomeRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeHotUpdateOutcomeFixtures(t, root, now)

	second := validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-b"
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.CandidatePackID = "pack-candidate"
		record.OutcomeKind = HotUpdateOutcomeKindBlocked
		record.Reason = "policy blocked activation"
		record.Notes = "control-plane block only"
		record.CreatedBy = "system"
	})
	if err := StoreHotUpdateOutcomeRecord(root, second); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-b) error = %v", err)
	}

	want := validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = " outcome-a "
		record.HotUpdateID = " hot-update-1 "
		record.CandidateID = " candidate-1 "
		record.RunID = " run-1 "
		record.CandidateResultID = " result-1 "
		record.CandidatePackID = " pack-candidate "
		record.Reason = " operator kept staged "
		record.Notes = " read-only outcome ledger "
		record.CreatedBy = " operator "
	})
	if err := StoreHotUpdateOutcomeRecord(root, want); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(outcome-a) error = %v", err)
	}

	got, err := LoadHotUpdateOutcomeRecord(root, "outcome-a")
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.OutcomeID = "outcome-a"
	want.HotUpdateID = "hot-update-1"
	want.CandidateID = "candidate-1"
	want.RunID = "run-1"
	want.CandidateResultID = "result-1"
	want.CandidatePackID = "pack-candidate"
	want.Reason = "operator kept staged"
	want.Notes = "read-only outcome ledger"
	want.OutcomeAt = want.OutcomeAt.UTC()
	want.CreatedAt = want.CreatedAt.UTC()
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadHotUpdateOutcomeRecord() = %#v, want %#v", got, want)
	}

	records, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 2", len(records))
	}
	if records[0].OutcomeID != "outcome-a" || records[1].OutcomeID != "outcome-b" {
		t.Fatalf("ListHotUpdateOutcomeRecords() ids = [%q %q], want [outcome-a outcome-b]", records[0].OutcomeID, records[1].OutcomeID)
	}
}

func TestHotUpdateOutcomeReplayIsIdempotentAndAppendOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)

	storeHotUpdateOutcomeFixtures(t, root, now)

	record := validHotUpdateOutcomeRecord(now.Add(8*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-replay"
		record.Reason = "exact replay"
		record.Notes = "same bytes expected"
	})
	if err := StoreHotUpdateOutcomeRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, record.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreHotUpdateOutcomeRecord(root, record); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, record.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update outcome file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(changed *HotUpdateOutcomeRecord) {
		changed.OutcomeID = "outcome-replay"
		changed.Notes = "divergent replay"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want append-only duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store hot-update outcome "outcome-replay" already exists`) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want append-only duplicate rejection", err.Error())
	}
}

func TestHotUpdateOutcomeValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing outcome id",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeID = " "
				}))
			},
			want: "hot-update outcome ref outcome_id is required",
		},
		{
			name: "invalid outcome kind",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeKind = HotUpdateOutcomeKind("bad_kind")
				}))
			},
			want: `mission store hot-update outcome outcome_kind "bad_kind" is invalid`,
		},
		{
			name: "missing outcome time",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.OutcomeAt = time.Time{}
				}))
			},
			want: "mission store hot-update outcome outcome_at is required",
		},
		{
			name: "missing created by",
			run: func() error {
				return StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
					record.CreatedBy = " "
				}))
			},
			want: "mission store hot-update outcome created_by is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestHotUpdateOutcomeRejectsMissingAndMismatchedLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)

	err := StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now, func(record *HotUpdateOutcomeRecord) {
		record.HotUpdateID = "missing-gate"
		record.CandidateID = ""
		record.RunID = ""
		record.CandidateResultID = ""
		record.CandidatePackID = ""
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want missing gate rejection")
	}
	if !strings.Contains(err.Error(), ErrHotUpdateGateRecordNotFound.Error()) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want missing gate rejection", err.Error())
	}

	storeHotUpdateOutcomeFixtures(t, root, now.Add(time.Minute))

	err = StoreHotUpdateOutcomeRecord(root, validHotUpdateOutcomeRecord(now.Add(9*time.Minute), func(record *HotUpdateOutcomeRecord) {
		record.OutcomeID = "outcome-mismatch"
		record.CandidatePackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreHotUpdateOutcomeRecord() error = nil, want linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `candidate_pack_id "pack-other" does not match hot-update gate candidate_pack_id "pack-candidate"`) {
		t.Fatalf("StoreHotUpdateOutcomeRecord() error = %q, want candidate_pack_id linkage mismatch rejection", err.Error())
	}
}

func TestLoadHotUpdateOutcomeRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadHotUpdateOutcomeRecord(root, "missing-outcome"); !errors.Is(err, ErrHotUpdateOutcomeRecordNotFound) {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v, want %v", err, ErrHotUpdateOutcomeRecordNotFound)
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateSucceededCreatesHotUpdatedOutcomeAndPreservesState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.FixedZone("offset", -4*60*60))
	terminalAt := now.Add(10 * time.Minute)
	createdAt := now.Add(11 * time.Minute)

	storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-success", HotUpdateGateStateReloadApplySucceeded, "")
	before := snapshotHotUpdateOutcomeSideEffects(t, root, "hot-update-success")

	got, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-success", " outcome-writer ", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}
	want := HotUpdateOutcomeRecord{
		RecordVersion:   StoreRecordVersion,
		OutcomeID:       "hot-update-outcome-hot-update-success",
		HotUpdateID:     "hot-update-success",
		CandidatePackID: "pack-candidate",
		OutcomeKind:     HotUpdateOutcomeKindHotUpdated,
		Reason:          "hot update reload/apply succeeded",
		OutcomeAt:       terminalAt.UTC(),
		CreatedAt:       createdAt.UTC(),
		CreatedBy:       "outcome-writer",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() = %#v, want %#v", got, want)
	}
	if got.CandidateID != "" || got.RunID != "" || got.CandidateResultID != "" {
		t.Fatalf("optional candidate/run/result refs = %q/%q/%q, want empty", got.CandidateID, got.RunID, got.CandidateResultID)
	}
	if got.CanaryRef != "" || got.ApprovalRef != "" {
		t.Fatalf("audit refs = %q/%q, want empty for non-canary outcome", got.CanaryRef, got.ApprovalRef)
	}

	loaded, err := LoadHotUpdateOutcomeRecord(root, got.OutcomeID)
	if err != nil {
		t.Fatalf("LoadHotUpdateOutcomeRecord() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, got) {
		t.Fatalf("LoadHotUpdateOutcomeRecord() = %#v, want %#v", loaded, got)
	}
	assertHotUpdateOutcomeSideEffectsUnchanged(t, root, "hot-update-success", before)
}

func TestCreateHotUpdateOutcomeFromTerminalGateFailedCopiesFailureDetail(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC)
	failureReason := "operator_terminal_failure: operator requested stop after recovery review"

	storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-failed", HotUpdateGateStateReloadApplyFailed, failureReason)

	got, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-failed", "operator", now.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}
	if got.OutcomeID != "hot-update-outcome-hot-update-failed" {
		t.Fatalf("OutcomeID = %q, want deterministic outcome id", got.OutcomeID)
	}
	if got.OutcomeKind != HotUpdateOutcomeKindFailed {
		t.Fatalf("OutcomeKind = %q, want failed", got.OutcomeKind)
	}
	if got.Reason != failureReason {
		t.Fatalf("Reason = %q, want copied deterministic failure detail", got.Reason)
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGatePropagatesCanaryAuditLineage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		owner         bool
		state         HotUpdateGateState
		failureReason string
		wantKind      HotUpdateOutcomeKind
	}{
		{
			name:     "no owner success",
			wantKind: HotUpdateOutcomeKindHotUpdated,
		},
		{
			name:     "owner approved success",
			owner:    true,
			wantKind: HotUpdateOutcomeKindHotUpdated,
		},
		{
			name:          "no owner failed",
			state:         HotUpdateGateStateReloadApplyFailed,
			failureReason: "operator_terminal_failure: canary terminal failure recorded",
			wantKind:      HotUpdateOutcomeKindFailed,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)
			state := tc.state
			if state == "" {
				state = HotUpdateGateStateReloadApplySucceeded
			}
			fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, state, tc.failureReason, tc.owner)
			before := snapshotHotUpdateOutcomeSideEffects(t, root, fixture.gate.HotUpdateID)
			sourceBefore := snapshotCanaryAuditSourceRecords(t, root, fixture)

			got, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
			if err != nil {
				t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
			}
			if !changed {
				t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
			}
			if got.CanaryRef != fixture.authority.CanarySatisfactionAuthorityID {
				t.Fatalf("CanaryRef = %q, want %q", got.CanaryRef, fixture.authority.CanarySatisfactionAuthorityID)
			}
			if tc.owner {
				if got.ApprovalRef != fixture.decision.OwnerApprovalDecisionID {
					t.Fatalf("ApprovalRef = %q, want %q", got.ApprovalRef, fixture.decision.OwnerApprovalDecisionID)
				}
			} else if got.ApprovalRef != "" {
				t.Fatalf("ApprovalRef = %q, want empty", got.ApprovalRef)
			}
			if got.OutcomeKind != tc.wantKind {
				t.Fatalf("OutcomeKind = %q, want %q", got.OutcomeKind, tc.wantKind)
			}
			if got.OutcomeKind == HotUpdateOutcomeKindCanaryApplied {
				t.Fatal("OutcomeKind = canary_applied, want existing hot_updated/failed kinds")
			}

			assertHotUpdateOutcomeSideEffectsUnchanged(t, root, fixture.gate.HotUpdateID, before)
			assertCanaryAuditSourceRecordsUnchanged(t, root, fixture, sourceBefore)
			assertNoHotUpdateOutcomeForbiddenRecords(t, root)
		})
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateDoesNotReauthorizeCanaryAfterTerminalExecution(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 12, 45, 0, 0, time.UTC)
	fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", false)

	staleEvidence := fixture.evidence
	staleEvidence.EvidenceState = HotUpdateCanaryEvidenceStateFailed
	staleEvidence.Passed = false
	staleEvidence.Reason = "drift after terminal execution"
	if err := WriteStoreJSONAtomic(StoreHotUpdateCanaryEvidencePath(root, staleEvidence.CanaryEvidenceID), staleEvidence); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(stale evidence) error = %v", err)
	}

	got, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", now.Add(31*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = false, want true")
	}
	if got.CanaryRef != fixture.authority.CanarySatisfactionAuthorityID {
		t.Fatalf("CanaryRef = %q, want terminal gate lineage %q", got.CanaryRef, fixture.authority.CanarySatisfactionAuthorityID)
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateFailedRequiresFailureReason(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-empty-failure", HotUpdateGateStateReloadApplyFailed, " \t ")

	_, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-empty-failure", "operator", now.Add(11*time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() error = nil, want missing failure_reason rejection")
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = true, want false")
	}
	if !strings.Contains(err.Error(), `failure_reason is required for outcome creation`) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %q, want failure_reason context", err.Error())
	}
	records, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(records))
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateReplayIsIdempotentAndDivergentDuplicateFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 13, 0, 0, 0, time.UTC)
	createdAt := now.Add(11 * time.Minute)

	storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-replay", HotUpdateGateStateReloadApplySucceeded, "")

	first, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-replay", "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(first) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, first.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(first outcome) error = %v", err)
	}

	second, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-replay", "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(replay) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, first.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(second outcome) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("hot-update outcome file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	_, changed, err = CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-replay", "operator", createdAt.Add(time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent) error = nil, want duplicate rejection")
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent) changed = true, want false")
	}
	if !strings.Contains(err.Error(), `mission store hot-update outcome "hot-update-outcome-hot-update-replay" already exists`) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(divergent) error = %q, want duplicate context", err.Error())
	}
	records, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 1", len(records))
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateDivergentDuplicateLineageFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 13, 30, 0, 0, time.UTC)
	fixture := storeCanaryHotUpdateTerminalOutcomeFixture(t, root, now, HotUpdateGateStateReloadApplySucceeded, "", true)
	createdAt := now.Add(31 * time.Minute)

	first, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(first) error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(first) changed = false, want true")
	}
	firstBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, first.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(first outcome) error = %v", err)
	}

	second, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", createdAt)
	if err != nil {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(replay) error = %v", err)
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(replay) changed = true, want false")
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(replay) = %#v, want %#v", second, first)
	}
	secondBytes, err := os.ReadFile(StoreHotUpdateOutcomePath(root, first.OutcomeID))
	if err != nil {
		t.Fatalf("ReadFile(second outcome) error = %v", err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatalf("hot-update outcome file changed on exact replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	divergent := first
	divergent.CanaryRef = "hot-update-canary-satisfaction-authority-other"
	writeRawHotUpdateOutcomeRecord(t, root, divergent)
	_, changed, err = CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent canary_ref) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent canary_ref) changed = true, want false")
	}
	if !strings.Contains(err.Error(), "canary_ref") {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(divergent canary_ref) error = %q, want canary_ref context", err.Error())
	}

	if err := WriteStoreJSONAtomic(StoreHotUpdateOutcomePath(root, first.OutcomeID), first); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(restore outcome) error = %v", err)
	}
	divergent = first
	divergent.ApprovalRef = HotUpdateOwnerApprovalDecisionIDFromRequest("hot-update-owner-approval-request-other")
	writeRawHotUpdateOutcomeRecord(t, root, divergent)
	_, changed, err = CreateHotUpdateOutcomeFromTerminalGate(root, fixture.gate.HotUpdateID, "operator", createdAt)
	if err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent approval_ref) error = nil, want fail-closed rejection")
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate(divergent approval_ref) changed = true, want false")
	}
	if !strings.Contains(err.Error(), "approval_ref") {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate(divergent approval_ref) error = %q, want approval_ref context", err.Error())
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateRejectsExistingDifferentOutcomeForSameHotUpdate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC)

	gate := storeHotUpdateTerminalOutcomeFixture(t, root, now, "hot-update-existing", HotUpdateGateStateReloadApplySucceeded, "")
	if err := StoreHotUpdateOutcomeRecord(root, HotUpdateOutcomeRecord{
		OutcomeID:       "legacy-outcome",
		HotUpdateID:     "hot-update-existing",
		CandidatePackID: "pack-candidate",
		OutcomeKind:     HotUpdateOutcomeKindHotUpdated,
		Reason:          "hot update reload/apply succeeded",
		OutcomeAt:       gate.PhaseUpdatedAt,
		CreatedAt:       now.Add(11 * time.Minute),
		CreatedBy:       "operator",
	}); err != nil {
		t.Fatalf("StoreHotUpdateOutcomeRecord(legacy) error = %v", err)
	}

	_, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, "hot-update-existing", "operator", now.Add(11*time.Minute))
	if err == nil {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() error = nil, want existing hot_update_id duplicate rejection")
	}
	if changed {
		t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = true, want false")
	}
	if !strings.Contains(err.Error(), `hot_update_id "hot-update-existing" already exists as "legacy-outcome"`) {
		t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %q, want hot_update_id duplicate context", err.Error())
	}
	records, err := ListHotUpdateOutcomeRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].OutcomeID != "legacy-outcome" {
		t.Fatalf("ListHotUpdateOutcomeRecords() = %#v, want only legacy-outcome", records)
	}
}

func TestCreateHotUpdateOutcomeFromTerminalGateRejectsNonTerminalStates(t *testing.T) {
	t.Parallel()

	tests := []HotUpdateGateState{
		HotUpdateGateStatePrepared,
		HotUpdateGateStateValidated,
		HotUpdateGateStateStaged,
		HotUpdateGateStateReloading,
		HotUpdateGateStateReloadApplyInProgress,
		HotUpdateGateStateReloadApplyRecoveryNeeded,
	}

	for _, state := range tests {
		state := state
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			now := time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC)
			hotUpdateID := "hot-update-" + strings.ReplaceAll(string(state), "_", "-")

			storeHotUpdateTerminalOutcomeFixture(t, root, now, hotUpdateID, state, "")

			_, changed, err := CreateHotUpdateOutcomeFromTerminalGate(root, hotUpdateID, "operator", now.Add(11*time.Minute))
			if err == nil {
				t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() error = nil, want non-terminal rejection")
			}
			if changed {
				t.Fatal("CreateHotUpdateOutcomeFromTerminalGate() changed = true, want false")
			}
			if !strings.Contains(err.Error(), `does not permit outcome creation`) {
				t.Fatalf("CreateHotUpdateOutcomeFromTerminalGate() error = %q, want non-terminal context", err.Error())
			}
			records, err := ListHotUpdateOutcomeRecords(root)
			if err != nil {
				t.Fatalf("ListHotUpdateOutcomeRecords() error = %v", err)
			}
			if len(records) != 0 {
				t.Fatalf("ListHotUpdateOutcomeRecords() len = %d, want 0", len(records))
			}
		})
	}
}

func validHotUpdateOutcomeRecord(now time.Time, mutate func(*HotUpdateOutcomeRecord)) HotUpdateOutcomeRecord {
	record := HotUpdateOutcomeRecord{
		OutcomeID:         "outcome-root",
		HotUpdateID:       "hot-update-1",
		CandidateID:       "candidate-1",
		RunID:             "run-1",
		CandidateResultID: "result-1",
		CandidatePackID:   "pack-candidate",
		OutcomeKind:       HotUpdateOutcomeKindKeptStaged,
		Reason:            "candidate kept staged",
		Notes:             "recorded for later operator control",
		OutcomeAt:         now,
		CreatedAt:         now.Add(time.Minute),
		CreatedBy:         "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeHotUpdateOutcomeFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

	storeImprovementRunFixtures(t, root, now)
	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-1"
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}
	if err := StoreCandidateResultRecord(root, validCandidateResultRecord(now.Add(6*time.Minute), func(record *CandidateResultRecord) {
		record.ResultID = "result-1"
		record.RunID = "run-1"
		record.HotUpdateID = "hot-update-1"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreCandidateResultRecord() error = %v", err)
	}
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(7*time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-other"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	}))
}

func storeHotUpdateTerminalOutcomeFixture(t *testing.T, root string, now time.Time, hotUpdateID string, state HotUpdateGateState, failureReason string) HotUpdateGateRecord {
	t.Helper()

	mustStoreRuntimePack(t, root, validRuntimePackRecord(now, func(record *RuntimePackRecord) {
		record.PackID = "pack-base"
	}))
	mustStoreRuntimePack(t, root, validRuntimePackRecord(now.Add(time.Minute), func(record *RuntimePackRecord) {
		record.PackID = "pack-candidate"
		record.ParentPackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.MutableSurfaces = []string{"skills"}
		record.SurfaceClasses = []string{"class_1"}
		record.CompatibilityContractRef = "compat-v1"
	}))
	if err := StoreActiveRuntimePackPointer(root, ActiveRuntimePackPointer{
		ActivePackID:         "pack-candidate",
		PreviousActivePackID: "pack-base",
		LastKnownGoodPackID:  "pack-base",
		UpdatedAt:            now.Add(2 * time.Minute),
		UpdatedBy:            "operator",
		UpdateRecordRef:      hotUpdateGatePointerUpdateRecordRef(hotUpdateID),
		ReloadGeneration:     7,
	}); err != nil {
		t.Fatalf("StoreActiveRuntimePackPointer() error = %v", err)
	}
	if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
		PackID:            "pack-base",
		Basis:             "holdout_pass",
		VerifiedAt:        now.Add(3 * time.Minute),
		VerifiedBy:        "operator",
		RollbackRecordRef: "bootstrap",
	}); err != nil {
		t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
	}

	gate := validHotUpdateGateRecord(now.Add(4*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = hotUpdateID
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
		record.State = state
		record.FailureReason = failureReason
		record.PhaseUpdatedAt = now.Add(10 * time.Minute)
		record.PhaseUpdatedBy = "operator"
	})
	if err := StoreHotUpdateGateRecord(root, gate); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	if state == HotUpdateGateStateReloadApplySucceeded {
		if _, _, err := CreateHotUpdateSmokeCheckFromGate(root, hotUpdateID, HotUpdateSmokeCheckStatePassed, now.Add(9*time.Minute), "operator", now.Add(9*time.Minute+30*time.Second), "terminal hot-update smoke passed"); err != nil {
			t.Fatalf("CreateHotUpdateSmokeCheckFromGate() error = %v", err)
		}
	}
	stored, err := LoadHotUpdateGateRecord(root, hotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord() error = %v", err)
	}
	return stored
}

type canaryTerminalOutcomeFixture struct {
	requirement HotUpdateCanaryRequirementRecord
	evidence    HotUpdateCanaryEvidenceRecord
	authority   HotUpdateCanarySatisfactionAuthorityRecord
	request     HotUpdateOwnerApprovalRequestRecord
	decision    HotUpdateOwnerApprovalDecisionRecord
	gate        HotUpdateGateRecord
}

func storeCanaryHotUpdateTerminalOutcomeFixture(t *testing.T, root string, now time.Time, state HotUpdateGateState, failureReason string, ownerApproval bool) canaryTerminalOutcomeFixture {
	t.Helper()

	var fixture canaryTerminalOutcomeFixture
	approvalRef := ""
	if ownerApproval {
		fixture.requirement, fixture.evidence, fixture.authority, fixture.request, fixture.decision = storeOwnerApprovedCanaryGateFixture(t, root, now, HotUpdateOwnerApprovalDecisionGranted)
		approvalRef = fixture.decision.OwnerApprovalDecisionID
		if err := StoreLastKnownGoodRuntimePackPointer(root, LastKnownGoodRuntimePackPointer{
			PackID:            fixture.authority.BaselinePackID,
			Basis:             "holdout_pass",
			VerifiedAt:        now.Add(25 * time.Minute),
			VerifiedBy:        "operator",
			RollbackRecordRef: "bootstrap",
		}); err != nil {
			t.Fatalf("StoreLastKnownGoodRuntimePackPointer() error = %v", err)
		}
	} else {
		fixture.requirement, fixture.evidence, fixture.authority = storeAuthorizedCanaryGateFixture(t, root, now, true)
	}

	gate, changed, err := CreateHotUpdateGateFromCanarySatisfactionAuthority(root, fixture.authority.CanarySatisfactionAuthorityID, approvalRef, "operator", now.Add(26*time.Minute))
	if err != nil {
		t.Fatalf("CreateHotUpdateGateFromCanarySatisfactionAuthority() error = %v", err)
	}
	if !changed {
		t.Fatal("CreateHotUpdateGateFromCanarySatisfactionAuthority() changed = false, want true")
	}
	gate.State = state
	gate.FailureReason = failureReason
	gate.PhaseUpdatedAt = now.Add(30 * time.Minute)
	gate.PhaseUpdatedBy = "operator"
	writeRawHotUpdateGateRecord(t, root, gate)
	if state == HotUpdateGateStateReloadApplySucceeded {
		if _, _, err := CreateHotUpdateSmokeCheckFromGate(root, gate.HotUpdateID, HotUpdateSmokeCheckStatePassed, now.Add(29*time.Minute), "operator", now.Add(29*time.Minute+30*time.Second), "terminal canary hot-update smoke passed"); err != nil {
			t.Fatalf("CreateHotUpdateSmokeCheckFromGate() error = %v", err)
		}
	}

	stored, err := LoadHotUpdateGateRecord(root, gate.HotUpdateID)
	if err != nil {
		t.Fatalf("LoadHotUpdateGateRecord(canary terminal) error = %v", err)
	}
	fixture.gate = stored
	return fixture
}

func snapshotCanaryAuditSourceRecords(t *testing.T, root string, fixture canaryTerminalOutcomeFixture) map[string][]byte {
	t.Helper()

	paths := []string{
		StoreHotUpdateCanaryRequirementPath(root, fixture.requirement.CanaryRequirementID),
		StoreHotUpdateCanaryEvidencePath(root, fixture.evidence.CanaryEvidenceID),
		StoreHotUpdateCanarySatisfactionAuthorityPath(root, fixture.authority.CanarySatisfactionAuthorityID),
	}
	if fixture.request.OwnerApprovalRequestID != "" {
		paths = append(paths, StoreHotUpdateOwnerApprovalRequestPath(root, fixture.request.OwnerApprovalRequestID))
	}
	if fixture.decision.OwnerApprovalDecisionID != "" {
		paths = append(paths, StoreHotUpdateOwnerApprovalDecisionPath(root, fixture.decision.OwnerApprovalDecisionID))
	}

	snapshot := make(map[string][]byte, len(paths))
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshot[path] = bytes
	}
	return snapshot
}

func assertCanaryAuditSourceRecordsUnchanged(t *testing.T, root string, fixture canaryTerminalOutcomeFixture, before map[string][]byte) {
	t.Helper()

	after := snapshotCanaryAuditSourceRecords(t, root, fixture)
	if len(after) != len(before) {
		t.Fatalf("canary source snapshot len = %d, want %d", len(after), len(before))
	}
	for path, beforeBytes := range before {
		afterBytes, ok := after[path]
		if !ok {
			t.Fatalf("canary source path %s missing after operation", path)
		}
		if string(afterBytes) != string(beforeBytes) {
			t.Fatalf("canary source record changed at %s\nbefore:\n%s\nafter:\n%s", path, string(beforeBytes), string(afterBytes))
		}
	}
}

func assertNoHotUpdateOutcomeForbiddenRecords(t *testing.T, root string) {
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

type hotUpdateOutcomeSideEffectSnapshot struct {
	activePointerBytes   []byte
	lastKnownGoodBytes   []byte
	hotUpdateGateBytes   []byte
	reloadGeneration     uint64
	hotUpdateGateRecords int
}

func snapshotHotUpdateOutcomeSideEffects(t *testing.T, root string, hotUpdateID string) hotUpdateOutcomeSideEffectSnapshot {
	t.Helper()

	activePointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer) error = %v", err)
	}
	lastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer) error = %v", err)
	}
	hotUpdateGateBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate) error = %v", err)
	}
	activePointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer() error = %v", err)
	}
	gates, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords() error = %v", err)
	}
	return hotUpdateOutcomeSideEffectSnapshot{
		activePointerBytes:   activePointerBytes,
		lastKnownGoodBytes:   lastKnownGoodBytes,
		hotUpdateGateBytes:   hotUpdateGateBytes,
		reloadGeneration:     activePointer.ReloadGeneration,
		hotUpdateGateRecords: len(gates),
	}
}

func assertHotUpdateOutcomeSideEffectsUnchanged(t *testing.T, root string, hotUpdateID string, before hotUpdateOutcomeSideEffectSnapshot) {
	t.Helper()

	afterPointerBytes, err := os.ReadFile(StoreActiveRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(active pointer after) error = %v", err)
	}
	if string(before.activePointerBytes) != string(afterPointerBytes) {
		t.Fatalf("active runtime-pack pointer changed during outcome creation\nbefore:\n%s\nafter:\n%s", string(before.activePointerBytes), string(afterPointerBytes))
	}
	afterPointer, err := LoadActiveRuntimePackPointer(root)
	if err != nil {
		t.Fatalf("LoadActiveRuntimePackPointer(after) error = %v", err)
	}
	if afterPointer.ReloadGeneration != before.reloadGeneration {
		t.Fatalf("ReloadGeneration = %d, want %d", afterPointer.ReloadGeneration, before.reloadGeneration)
	}

	afterLastKnownGoodBytes, err := os.ReadFile(StoreLastKnownGoodRuntimePackPointerPath(root))
	if err != nil {
		t.Fatalf("ReadFile(last-known-good pointer after) error = %v", err)
	}
	if string(before.lastKnownGoodBytes) != string(afterLastKnownGoodBytes) {
		t.Fatalf("last-known-good pointer changed during outcome creation\nbefore:\n%s\nafter:\n%s", string(before.lastKnownGoodBytes), string(afterLastKnownGoodBytes))
	}

	afterGateBytes, err := os.ReadFile(StoreHotUpdateGatePath(root, hotUpdateID))
	if err != nil {
		t.Fatalf("ReadFile(hot-update gate after) error = %v", err)
	}
	if string(before.hotUpdateGateBytes) != string(afterGateBytes) {
		t.Fatalf("hot-update gate changed during outcome creation\nbefore:\n%s\nafter:\n%s", string(before.hotUpdateGateBytes), string(afterGateBytes))
	}
	gates, err := ListHotUpdateGateRecords(root)
	if err != nil {
		t.Fatalf("ListHotUpdateGateRecords(after) error = %v", err)
	}
	if len(gates) != before.hotUpdateGateRecords {
		t.Fatalf("ListHotUpdateGateRecords(after) len = %d, want %d", len(gates), before.hotUpdateGateRecords)
	}

	promotions, err := ListPromotionRecords(root)
	if err != nil {
		t.Fatalf("ListPromotionRecords() error = %v", err)
	}
	if len(promotions) != 0 {
		t.Fatalf("ListPromotionRecords() len = %d, want 0", len(promotions))
	}
}
