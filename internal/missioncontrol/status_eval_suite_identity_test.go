package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestLoadOperatorEvalSuiteIdentityStatusConfigured(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 19, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now.Add(5*time.Minute), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-2"
		record.RubricRef = "rubric://holdout-v2"
		record.RubricSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		record.TrainCorpusRef = "corpus://train-v2"
		record.TrainCorpusSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		record.HoldoutCorpusRef = "corpus://holdout-v2"
		record.HoldoutCorpusSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		record.EvaluatorRef = "evaluator://frozen-v2"
		record.EvaluatorSHA256 = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
		record.FrozenContentRef = "freeze/eval-suite-v2"
		record.NegativeCaseCount = 4
		record.BoundaryCaseCount = 2
		record.FrozenForRun = true
		record.CandidateID = "candidate-1"
		record.BaselinePackID = "pack-base"
		record.CandidatePackID = "pack-candidate"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreEvalSuiteRecord() error = %v", err)
	}

	got := LoadOperatorEvalSuiteIdentityStatus(root)
	if got.State != "configured" {
		t.Fatalf("State = %q, want configured", got.State)
	}
	if len(got.Suites) != 2 {
		t.Fatalf("Suites len = %d, want 2", len(got.Suites))
	}
	suite := got.Suites[1]
	if suite.State != "configured" {
		t.Fatalf("Suites[1].State = %q, want configured", suite.State)
	}
	if suite.EvalSuiteID != "eval-suite-2" {
		t.Fatalf("Suites[1].EvalSuiteID = %q, want eval-suite-2", suite.EvalSuiteID)
	}
	if suite.CandidateID != "candidate-1" {
		t.Fatalf("Suites[1].CandidateID = %q, want candidate-1", suite.CandidateID)
	}
	if suite.BaselinePackID != "pack-base" {
		t.Fatalf("Suites[1].BaselinePackID = %q, want pack-base", suite.BaselinePackID)
	}
	if suite.CandidatePackID != "pack-candidate" {
		t.Fatalf("Suites[1].CandidatePackID = %q, want pack-candidate", suite.CandidatePackID)
	}
	if suite.RubricRef != "rubric://holdout-v2" {
		t.Fatalf("Suites[1].RubricRef = %q, want rubric://holdout-v2", suite.RubricRef)
	}
	if suite.TrainCorpusRef != "corpus://train-v2" {
		t.Fatalf("Suites[1].TrainCorpusRef = %q, want corpus://train-v2", suite.TrainCorpusRef)
	}
	if suite.HoldoutCorpusRef != "corpus://holdout-v2" {
		t.Fatalf("Suites[1].HoldoutCorpusRef = %q, want corpus://holdout-v2", suite.HoldoutCorpusRef)
	}
	if suite.EvaluatorRef != "evaluator://frozen-v2" {
		t.Fatalf("Suites[1].EvaluatorRef = %q, want evaluator://frozen-v2", suite.EvaluatorRef)
	}
	if suite.RubricSHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("Suites[1].RubricSHA256 = %q, want rubric content identity", suite.RubricSHA256)
	}
	if suite.TrainCorpusSHA256 != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("Suites[1].TrainCorpusSHA256 = %q, want train corpus content identity", suite.TrainCorpusSHA256)
	}
	if suite.HoldoutCorpusSHA256 != "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("Suites[1].HoldoutCorpusSHA256 = %q, want holdout corpus content identity", suite.HoldoutCorpusSHA256)
	}
	if suite.EvaluatorSHA256 != "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" {
		t.Fatalf("Suites[1].EvaluatorSHA256 = %q, want evaluator content identity", suite.EvaluatorSHA256)
	}
	if suite.FrozenContentRef != "freeze/eval-suite-v2" {
		t.Fatalf("Suites[1].FrozenContentRef = %q, want freeze/eval-suite-v2", suite.FrozenContentRef)
	}
	if !suite.FrozenForRun {
		t.Fatal("Suites[1].FrozenForRun = false, want true")
	}
	if suite.CreatedAt == nil || *suite.CreatedAt != "2026-04-25T19:05:00Z" {
		t.Fatalf("Suites[1].CreatedAt = %#v, want 2026-04-25T19:05:00Z", suite.CreatedAt)
	}
	if suite.CreatedBy != "operator" {
		t.Fatalf("Suites[1].CreatedBy = %q, want operator", suite.CreatedBy)
	}
	if suite.Error != "" {
		t.Fatalf("Suites[1].Error = %q, want empty", suite.Error)
	}
}

func TestLoadOperatorEvalSuiteIdentityStatusNotConfigured(t *testing.T) {
	t.Parallel()

	got := LoadOperatorEvalSuiteIdentityStatus(t.TempDir())
	if got.State != "not_configured" {
		t.Fatalf("State = %q, want not_configured", got.State)
	}
	if len(got.Suites) != 0 {
		t.Fatalf("Suites len = %d, want 0", len(got.Suites))
	}
}

func TestLoadOperatorEvalSuiteIdentityStatusInvalidMissingLinkedRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, 4, 25, 20, 0, 0, 0, time.UTC)
	storeImprovementRunFixtures(t, root, now)
	if err := WriteStoreJSONAtomic(StoreEvalSuitePath(root, "eval-suite-bad"), validEvalSuiteRecord(now.Add(5*time.Minute), func(record *EvalSuiteRecord) {
		record.RecordVersion = StoreRecordVersion
		record.EvalSuiteID = "eval-suite-bad"
		record.CandidatePackID = "pack-missing"
	})); err != nil {
		t.Fatalf("WriteStoreJSONAtomic(eval-suite-bad) error = %v", err)
	}

	got := LoadOperatorEvalSuiteIdentityStatus(root)
	if got.State != "invalid" {
		t.Fatalf("State = %q, want invalid", got.State)
	}
	if len(got.Suites) != 2 {
		t.Fatalf("Suites len = %d, want 2", len(got.Suites))
	}
	suite := got.Suites[1]
	if suite.State != "invalid" {
		t.Fatalf("Suites[1].State = %q, want invalid", suite.State)
	}
	if suite.EvalSuiteID != "eval-suite-bad" {
		t.Fatalf("Suites[1].EvalSuiteID = %q, want eval-suite-bad", suite.EvalSuiteID)
	}
	if suite.CandidatePackID != "pack-missing" {
		t.Fatalf("Suites[1].CandidatePackID = %q, want pack-missing", suite.CandidatePackID)
	}
	if !strings.Contains(suite.Error, `candidate_pack_id "pack-missing"`) {
		t.Fatalf("Suites[1].Error = %q, want missing candidate_pack_id context", suite.Error)
	}
}

func TestBuildCommittedMissionStatusSnapshotIncludesEvalSuiteIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := testLeaseSafeNow()
	job := testProjectedRuntimeJob()
	control, err := BuildRuntimeControlContext(job, "build")
	if err != nil {
		t.Fatalf("BuildRuntimeControlContext() error = %v", err)
	}
	runtime := JobRuntimeState{
		JobID:        job.ID,
		State:        JobStateRunning,
		ActiveStepID: "build",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-time.Minute),
		StartedAt:    now.Add(-2 * time.Minute),
		ActiveStepAt: now.Add(-90 * time.Second),
	}
	if err := PersistProjectedRuntimeState(root, WriterLockLease{LeaseHolderID: "holder-1"}, &job, runtime, &control, now); err != nil {
		t.Fatalf("PersistProjectedRuntimeState() error = %v", err)
	}

	storeImprovementRunFixtures(t, root, now.Add(-10*time.Minute))
	if err := StoreEvalSuiteRecord(root, validEvalSuiteRecord(now.Add(-30*time.Second), func(record *EvalSuiteRecord) {
		record.EvalSuiteID = "eval-suite-2"
		record.CreatedBy = "operator"
	})); err != nil {
		t.Fatalf("StoreEvalSuiteRecord() error = %v", err)
	}

	snapshot, err := BuildCommittedMissionStatusSnapshot(root, job.ID, MissionStatusSnapshotOptions{
		MissionFile: "mission.json",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("BuildCommittedMissionStatusSnapshot() error = %v", err)
	}
	if snapshot.RuntimeSummary == nil || snapshot.RuntimeSummary.EvalSuiteIdentity == nil {
		t.Fatalf("RuntimeSummary.EvalSuiteIdentity = %#v, want populated eval suite identity", snapshot.RuntimeSummary)
	}
	if snapshot.RuntimeSummary.EvalSuiteIdentity.State != "configured" {
		t.Fatalf("RuntimeSummary.EvalSuiteIdentity.State = %q, want configured", snapshot.RuntimeSummary.EvalSuiteIdentity.State)
	}
	if len(snapshot.RuntimeSummary.EvalSuiteIdentity.Suites) != 2 {
		t.Fatalf("RuntimeSummary.EvalSuiteIdentity.Suites len = %d, want 2", len(snapshot.RuntimeSummary.EvalSuiteIdentity.Suites))
	}
	if snapshot.RuntimeSummary.EvalSuiteIdentity.Suites[1].EvalSuiteID != "eval-suite-2" {
		t.Fatalf("RuntimeSummary.EvalSuiteIdentity.Suites[1].EvalSuiteID = %q, want eval-suite-2", snapshot.RuntimeSummary.EvalSuiteIdentity.Suites[1].EvalSuiteID)
	}
}
