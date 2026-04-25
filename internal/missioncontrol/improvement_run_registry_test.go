package missioncontrol

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestImprovementRunRecordRoundTripAndList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 14, 0, 0, 0, time.FixedZone("offset", -4*60*60))

	storeImprovementRunFixtures(t, root, now)

	second := validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-b"
		record.Objective = "second improvement run"
		record.HotUpdateID = ""
		record.State = ImprovementRunStateQueued
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	})
	if err := StoreImprovementRunRecord(root, second); err != nil {
		t.Fatalf("StoreImprovementRunRecord(run-b) error = %v", err)
	}

	want := validImprovementRunRecord(now.Add(6*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = " run-a "
		record.Objective = " improve prompt pack "
		record.ExecutionPlane = " improvement_workspace "
		record.ExecutionHost = " phone "
		record.MissionFamily = " improve_prompt_pack "
		record.TargetType = " prompt_pack "
		record.TargetRef = " prompt-pack://primary "
		record.SurfaceClass = " hot_reloadable "
		record.CandidateID = " candidate-1 "
		record.EvalSuiteID = " eval-suite-1 "
		record.BaselinePackID = " pack-base "
		record.CandidatePackID = " pack-candidate "
		record.HotUpdateID = " hot-update-1 "
		record.State = ImprovementRunStateCandidateReady
		record.Decision = ImprovementRunDecisionKeep
		record.StopReason = " holdout ready "
		record.CreatedBy = " operator "
	})
	if err := StoreImprovementRunRecord(root, want); err != nil {
		t.Fatalf("StoreImprovementRunRecord(run-a) error = %v", err)
	}

	got, err := LoadImprovementRunRecord(root, "run-a")
	if err != nil {
		t.Fatalf("LoadImprovementRunRecord() error = %v", err)
	}

	want.RecordVersion = StoreRecordVersion
	want.RunID = "run-a"
	want.Objective = "improve prompt pack"
	want.ExecutionPlane = "improvement_workspace"
	want.ExecutionHost = "phone"
	want.MissionFamily = "improve_prompt_pack"
	want.TargetType = "prompt_pack"
	want.TargetRef = "prompt-pack://primary"
	want.SurfaceClass = "hot_reloadable"
	want.CandidateID = "candidate-1"
	want.EvalSuiteID = "eval-suite-1"
	want.BaselinePackID = "pack-base"
	want.CandidatePackID = "pack-candidate"
	want.HotUpdateID = "hot-update-1"
	want.CreatedAt = want.CreatedAt.UTC()
	want.CompletedAt = want.CompletedAt.UTC()
	want.StopReason = "holdout ready"
	want.CreatedBy = "operator"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadImprovementRunRecord() = %#v, want %#v", got, want)
	}

	records, err := ListImprovementRunRecords(root)
	if err != nil {
		t.Fatalf("ListImprovementRunRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListImprovementRunRecords() len = %d, want 2", len(records))
	}
	if records[0].RunID != "run-a" || records[1].RunID != "run-b" {
		t.Fatalf("ListImprovementRunRecords() ids = [%q %q], want [run-a run-b]", records[0].RunID, records[1].RunID)
	}
}

func TestImprovementRunReplayIsIdempotentAndAppendOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 15, 0, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)

	record := validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-replay"
		record.State = ImprovementRunStateQueued
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
		record.HotUpdateID = ""
	})
	if err := StoreImprovementRunRecord(root, record); err != nil {
		t.Fatalf("StoreImprovementRunRecord(first) error = %v", err)
	}
	firstBytes, err := os.ReadFile(StoreImprovementRunPath(root, record.RunID))
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}

	if err := StoreImprovementRunRecord(root, record); err != nil {
		t.Fatalf("StoreImprovementRunRecord(replay) error = %v", err)
	}
	secondBytes, err := os.ReadFile(StoreImprovementRunPath(root, record.RunID))
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("improvement run file changed on idempotent replay\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}

	err = StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(6*time.Minute), func(changed *ImprovementRunRecord) {
		changed.RunID = "run-replay"
		changed.Objective = "different objective"
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want append-only duplicate rejection")
	}
	if !strings.Contains(err.Error(), `mission store improvement run "run-replay" already exists`) {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want append-only duplicate rejection", err.Error())
	}
}

func TestImprovementRunValidationFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "missing run id",
			run: func() error {
				return StoreImprovementRunRecord(root, validImprovementRunRecord(now, func(record *ImprovementRunRecord) {
					record.RunID = " "
				}))
			},
			want: "improvement run ref run_id is required",
		},
		{
			name: "missing objective",
			run: func() error {
				return StoreImprovementRunRecord(root, validImprovementRunRecord(now, func(record *ImprovementRunRecord) {
					record.Objective = " "
				}))
			},
			want: "mission store improvement run objective is required",
		},
		{
			name: "invalid state",
			run: func() error {
				return StoreImprovementRunRecord(root, validImprovementRunRecord(now, func(record *ImprovementRunRecord) {
					record.State = ImprovementRunState("bad_state")
				}))
			},
			want: `mission store improvement run state "bad_state" is invalid`,
		},
		{
			name: "completed without decision",
			run: func() error {
				return StoreImprovementRunRecord(root, validImprovementRunRecord(now, func(record *ImprovementRunRecord) {
					record.Decision = ""
					record.CompletedAt = now.Add(time.Minute)
				}))
			},
			want: "mission store improvement run completed_at requires decision",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if err == nil {
				t.Fatal("StoreImprovementRunRecord() error = nil, want fail-closed rejection")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("StoreImprovementRunRecord() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestImprovementRunRejectsMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 17, 0, 0, 0, time.UTC)

	err := StoreImprovementRunRecord(root, validImprovementRunRecord(now, func(record *ImprovementRunRecord) {
		record.CandidateID = "missing-candidate"
		record.EvalSuiteID = "missing-eval"
		record.BaselinePackID = "missing-base"
		record.CandidatePackID = "missing-pack"
		record.HotUpdateID = "missing-gate"
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want missing ref rejection")
	}
	if !strings.Contains(err.Error(), ErrImprovementCandidateRecordNotFound.Error()) && !strings.Contains(err.Error(), ErrEvalSuiteRecordNotFound.Error()) {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want missing ref rejection", err.Error())
	}

	storeImprovementRunFixtures(t, root, now.Add(time.Minute))

	err = StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(6*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-mismatch"
		record.BaselinePackID = "pack-other"
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want candidate linkage mismatch rejection")
	}
	if !strings.Contains(err.Error(), `baseline_pack_id "pack-other" does not match candidate baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want candidate linkage mismatch rejection", err.Error())
	}
}

func TestImprovementRunRequiresExistingFrozenEvalSuite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 17, 30, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)

	err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-missing-eval-suite"
		record.EvalSuiteID = "missing-eval-suite"
		record.HotUpdateID = ""
		record.State = ImprovementRunStateQueued
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want missing eval-suite rejection")
	}
	if !strings.Contains(err.Error(), ErrEvalSuiteRecordNotFound.Error()) {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want missing eval-suite rejection", err.Error())
	}

	unfrozen := validEvalSuiteRecord(now.Add(6*time.Minute), func(record *EvalSuiteRecord) {
		record.RecordVersion = StoreRecordVersion
		record.EvalSuiteID = "eval-suite-unfrozen"
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.FrozenForRun = false
	})
	if err := WriteStoreJSONAtomic(StoreEvalSuitePath(root, unfrozen.EvalSuiteID), unfrozen); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(unfrozen eval-suite) error = %v", err)
	}

	err = StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(7*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-unfrozen-eval-suite"
		record.EvalSuiteID = "eval-suite-unfrozen"
		record.HotUpdateID = ""
		record.State = ImprovementRunStateQueued
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want unfrozen eval-suite rejection")
	}
	if !strings.Contains(err.Error(), "mission store eval-suite frozen_for_run must be true") {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want unfrozen eval-suite rejection", err.Error())
	}
}

func TestImprovementRunRequiresEvalSuitePackLinkage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 17, 45, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)

	if err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now.Add(5*time.Minute), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-pack-mismatch"
		record.CandidateID = ""
		record.BaselinePackID = "pack-candidate"
		record.CandidatePackID = ""
	})); err != nil {
		t.Fatalf("StoreEvalSuiteRecord(mismatch fixture) error = %v", err)
	}

	err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(6*time.Minute), func(record *ImprovementRunRecord) {
		record.RunID = "run-eval-suite-pack-mismatch"
		record.EvalSuiteID = "eval-suite-pack-mismatch"
		record.HotUpdateID = ""
		record.State = ImprovementRunStateQueued
		record.Decision = ""
		record.CompletedAt = time.Time{}
		record.StopReason = ""
	}))
	if err == nil {
		t.Fatal("StoreImprovementRunRecord() error = nil, want eval-suite pack linkage rejection")
	}
	if !strings.Contains(err.Error(), `eval_suite_id "eval-suite-pack-mismatch" baseline_pack_id "pack-candidate" does not match run baseline_pack_id "pack-base"`) {
		t.Fatalf("StoreImprovementRunRecord() error = %q, want eval-suite pack linkage rejection", err.Error())
	}
}

func TestImprovementRunStoreDoesNotMutateLinkedRecordsOrRuntimePointers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 23, 18, 0, 0, 0, time.UTC)

	storeImprovementRunFixtures(t, root, now)

	snapshots := map[string][]byte{}
	for _, path := range []string{
		StoreRuntimePackPath(root, "pack-base"),
		StoreRuntimePackPath(root, "pack-candidate"),
		StoreImprovementCandidatePath(root, "candidate-1"),
		StoreEvalSuitePath(root, "eval-suite-1"),
		StoreHotUpdateGatePath(root, "hot-update-1"),
	} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		snapshots[path] = bytes
	}

	if err := StoreImprovementRunRecord(root, validImprovementRunRecord(now.Add(5*time.Minute), nil)); err != nil {
		t.Fatalf("StoreImprovementRunRecord() error = %v", err)
	}

	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) after store error = %v", path, err)
		}
		if string(after) != string(before) {
			t.Fatalf("linked record %s changed after improvement-run store", path)
		}
	}

	absentPaths := []string{
		StoreCandidateResultsDir(root),
		StoreHotUpdateOutcomesDir(root),
		StorePromotionsDir(root),
		StoreRollbacksDir(root),
		StoreRollbackAppliesDir(root),
		StoreActiveRuntimePackPointerPath(root),
		StoreLastKnownGoodRuntimePackPointerPath(root),
	}
	for _, path := range absentPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path %s exists or errored after improvement-run store: %v", path, err)
		}
	}
}

func TestLoadImprovementRunRecordNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadImprovementRunRecord(root, "missing-run"); !errors.Is(err, ErrImprovementRunRecordNotFound) {
		t.Fatalf("LoadImprovementRunRecord() error = %v, want %v", err, ErrImprovementRunRecordNotFound)
	}
}

func validImprovementRunRecord(now time.Time, mutate func(*ImprovementRunRecord)) ImprovementRunRecord {
	record := ImprovementRunRecord{
		RunID:           "run-root",
		Objective:       "improve runtime pack",
		ExecutionPlane:  "improvement_workspace",
		ExecutionHost:   "phone",
		MissionFamily:   "evaluate_candidate",
		TargetType:      "prompt_pack",
		TargetRef:       "prompt-pack://default",
		SurfaceClass:    "hot_reloadable",
		CandidateID:     "candidate-1",
		EvalSuiteID:     "eval-suite-1",
		BaselinePackID:  "pack-base",
		CandidatePackID: "pack-candidate",
		HotUpdateID:     "hot-update-1",
		State:           ImprovementRunStateCandidateReady,
		Decision:        ImprovementRunDecisionKeep,
		CreatedAt:       now,
		CompletedAt:     now.Add(time.Minute),
		StopReason:      "ready for next control surface",
		CreatedBy:       "system",
	}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func storeImprovementRunFixtures(t *testing.T, root string, now time.Time) {
	t.Helper()

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
	if err := StoreHotUpdateGateRecord(root, validHotUpdateGateRecord(now.Add(2*time.Minute), func(record *HotUpdateGateRecord) {
		record.HotUpdateID = "hot-update-1"
		record.CandidatePackID = "pack-candidate"
		record.PreviousActivePackID = "pack-base"
		record.RollbackTargetPackID = "pack-base"
	})); err != nil {
		t.Fatalf("StoreHotUpdateGateRecord() error = %v", err)
	}
	if err := StoreImprovementCandidateRecord(root, validImprovementCandidateRecord(now.Add(3*time.Minute), func(record *ImprovementCandidateRecord) {
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.HotUpdateID = "hot-update-1"
		record.SourceSummary = "candidate linkage"
		record.SourceWorkspaceRef = ""
	})); err != nil {
		t.Fatalf("StoreImprovementCandidateRecord() error = %v", err)
	}
	if err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now.Add(4*time.Minute), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-1"
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.CreatedBy = "system"
	})); err != nil {
		t.Fatalf("StoreEvalSuiteRecord() error = %v", err)
	}
}
