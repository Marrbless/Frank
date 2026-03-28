package missioncontrol

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeFinalResponseAppendsDeterministicMissionSummary(t *testing.T) {
	t.Parallel()

	job := testV2Job([]Step{
		{ID: "alpha", Type: StepTypeStaticArtifact, StaticArtifactPath: "alpha.txt", StaticArtifactFormat: "text"},
		{ID: "beta", Type: StepTypeStaticArtifact, StaticArtifactPath: "beta.txt", StaticArtifactFormat: "txt"},
		{ID: "gamma", Type: StepTypeOneShotCode, OneShotArtifactPath: "gamma.txt"},
		{ID: "delta", Type: StepTypeLongRunningCode, LongRunningStartupCommand: []string{"./delta"}, LongRunningArtifactPath: "delta.txt"},
		{ID: "epsilon", Type: StepTypeStaticArtifact, StaticArtifactPath: "epsilon.txt", StaticArtifactFormat: "text"},
		{ID: "zeta", Type: StepTypeOneShotCode, OneShotArtifactPath: "zeta.txt"},
		{ID: "review", Type: StepTypeDiscussion, Subtype: StepSubtypeBlocker},
		{ID: "hold", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "followup", Type: StepTypeDiscussion, Subtype: StepSubtypeDefinition},
		{ID: "verify", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "publish", Type: StepTypeDiscussion, Subtype: StepSubtypeDefinition},
		{ID: "cleanup", Type: StepTypeWaitUser, Subtype: StepSubtypeDefinition},
		{ID: "final", Type: StepTypeFinalResponse, DependsOn: []string{"cleanup"}},
	})

	ec := testStepValidationExecutionContextForJob(job, "final", JobStateRunning)
	ec.Runtime.CompletedSteps = []RuntimeStepRecord{
		{StepID: "zeta", At: time.Date(2026, 3, 27, 12, 6, 0, 0, time.UTC)},
		{StepID: "beta", At: time.Date(2026, 3, 27, 12, 2, 0, 0, time.UTC), ResultingState: &RuntimeResultingStateRecord{Kind: string(StepTypeStaticArtifact), Target: "beta.txt", State: "already_present"}},
		{StepID: "epsilon", At: time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC)},
		{StepID: "delta", At: time.Date(2026, 3, 27, 12, 4, 0, 0, time.UTC)},
		{StepID: "gamma", At: time.Date(2026, 3, 27, 12, 3, 0, 0, time.UTC)},
		{StepID: "alpha", At: time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)},
	}
	ec.Runtime.FailedSteps = []RuntimeStepRecord{
		{StepID: "review", At: time.Date(2026, 3, 27, 12, 7, 0, 0, time.UTC)},
	}

	got := NormalizeFinalResponse(ec, "Here is the final answer with the requested outputs.")
	want := "Here is the final answer with the requested outputs.\n\nMission summary:\n" +
		"Artifacts: alpha.txt; beta.txt (already present); gamma.txt; delta.txt; epsilon.txt; +1 more omitted\n" +
		"Pending/blocked steps: review (blocked); hold; followup; verify; publish; +1 more omitted"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}

func TestNormalizeFinalResponseTruncatesRawBodyWithOmissionMarker(t *testing.T) {
	t.Parallel()

	ec := testStepValidationExecutionContext(Step{
		ID:   "final",
		Type: StepTypeFinalResponse,
	}, JobStateRunning)

	raw := strings.Repeat("a", finalResponseBodyRuneLimit+7)
	got := NormalizeFinalResponse(ec, raw)
	want := strings.Repeat("a", finalResponseBodyRuneLimit) +
		"\n\n[final response body truncated; 7 characters omitted]"
	if got != want {
		t.Fatalf("NormalizeFinalResponse() = %q, want %q", got, want)
	}
}
